import { createEffect, createMemo, createSignal, onCleanup, type Accessor } from "solid-js";
import {
  ApiError,
  draftAudioURLs,
  generateDraft,
  generatePreview,
  getDraftGeneration,
  getPreview,
  listVoices,
  previewAudioURLs,
  type Item,
  type PreviewInput,
  type Voice,
} from "../../api/client";
import {
  readSpeechPreference,
  saveSpeechPreference,
  speechRates,
  type SpeechRate,
} from "../../preferences/storage";
import type { DraftFields } from "./draft";

type Target =
  | { kind: "draft"; itemId: string; version: number }
  | { kind: "preview"; inputs: PreviewInput };
type Attempt = { key: string; target: Target };
type Job = Target & { id: string };
type AudioStatus = Item["audioStatus"] | "EXPIRED";
type Observation = {
  job: Job;
  status: AudioStatus;
  assetId?: string | null;
  error?: string | null;
  expiresAt?: string;
  active?: boolean;
};

export const audioStatusLabel: Record<AudioStatus, string> = {
  NOT_GENERATED: "Not generated",
  PENDING: "Pending",
  PROCESSING: "Processing",
  READY: "Ready",
  FAILED: "Failed",
  OUTDATED: "Outdated",
  EXPIRED: "Preview expired or claimed",
};

const pollInterval = 1000;
const pollLimit = 120_000;

function matches(left: PreviewInput | undefined, right: PreviewInput) {
  return (
    left?.languageCode === right.languageCode &&
    left.voiceId === right.voiceId &&
    left.speakingRate === right.speakingRate &&
    left.sourceText === right.sourceText
  );
}

export function useDraftAudio(options: {
  item: Accessor<Item | null>;
  values: Accessor<DraftFields>;
  synthesisChanged: Accessor<boolean>;
  editingBlocked: Accessor<boolean>;
  setSpeech: (voiceId: string | null, speakingRate: SpeechRate | null) => void;
}) {
  const [catalog, setCatalog] = createSignal<{
    language: PreviewInput["languageCode"];
    items: Voice[];
  }>();
  const [voiceError, setVoiceError] = createSignal("");
  const [voiceAttempt, setVoiceAttempt] = createSignal(0);
  const language = () => options.values().languageCode;
  const voices = createMemo(() => {
    const available = catalog();
    return available?.language === language() ? available.items : undefined;
  });
  const selection = createMemo(() => {
    const values = options.values();
    const available = voices();
    if (!values.languageCode || !available?.length) return;
    const preferred = options.item() ? undefined : readSpeechPreference(values.languageCode);
    const voice = values.voiceId
      ? available.find((entry) => entry.voiceId === values.voiceId)
      : (available.find((entry) => entry.voiceId === preferred?.voiceId) ?? available[0]);
    if (!voice) return;
    const rates = speechRates.filter((rate) => voice.speakingRates.includes(rate));
    const desired = values.speakingRate ?? preferred?.speakingRate ?? 1;
    const rate = rates.includes(desired) ? desired : rates.includes(1) ? 1 : rates[0];
    return { voice, rate, rates };
  });
  const inputs = createMemo<PreviewInput | undefined>(() => {
    const chosen = selection();
    const code = language();
    if (!chosen || !code || chosen.rate === undefined) return;
    return {
      languageCode: code,
      sourceText: options.values().sourceText,
      voiceId: chosen.voice.voiceId,
      speakingRate: chosen.rate,
    };
  });

  createEffect(
    () => [language(), voiceAttempt()] as const,
    ([code]) => {
      setVoiceError("");
      if (!code) return;
      const controller = new AbortController();
      listVoices(code, controller.signal).then(
        ({ items }) => {
          if (controller.signal.aborted) return;
          setCatalog({
            language: code,
            items: items.filter((voice) => voice.languageCode === code),
          });
        },
        (error) => {
          if (controller.signal.aborted) return;
          setCatalog(undefined);
          setVoiceError(
            error instanceof ApiError && error.code === "TTS_UNAVAILABLE"
              ? "Speech synthesis is unavailable. Check the provider configuration and try again."
              : "Could not load speech voices.",
          );
        },
      );
      return () => controller.abort();
    },
  );

  function changeSpeech(voiceId: string, rate: SpeechRate) {
    const code = language();
    const voice = voices()?.find((entry) => entry.voiceId === voiceId);
    if (!code || !voice) return;
    const speakingRate = voice.speakingRates.includes(rate)
      ? rate
      : speechRates.find((entry) => voice.speakingRates.includes(entry));
    if (speakingRate === undefined) return;
    const item = options.item();
    options.setSpeech(
      item?.voiceId === null && voiceId === voices()?.[0]?.voiceId ? null : voiceId,
      item?.speakingRate === null && speakingRate === 1 ? null : speakingRate,
    );
    saveSpeechPreference(code, { voiceId, speakingRate });
  }

  const [accepted, setAccepted] = createSignal<Job>();
  const [attempt, setAttempt] = createSignal<Attempt>();
  const [submitting, setSubmitting] = createSignal(false);
  const [actionError, setActionError] = createSignal("");
  const [pollError, setPollError] = createSignal("");
  const [pollAgain, setPollAgain] = createSignal(0);
  const [observation, setObservation] = createSignal<Observation>();
  let request: AbortController | undefined;

  const job = createMemo<Job | undefined>(() => {
    const item = options.item();
    const local = accepted();
    if (!item) return local?.kind === "preview" ? local : undefined;
    if (local?.kind === "draft" && local.itemId === item.id && local.version === item.version)
      return local;
    return item.audioGenerationId
      ? { kind: "draft", itemId: item.id, version: item.version, id: item.audioGenerationId }
      : undefined;
  });
  const observed = createMemo(() => {
    const latest = observation();
    return latest?.job === job() ? latest : undefined;
  });
  const fallbackStatus = createMemo(() => {
    const item = options.item();
    const current = job();
    if (item && item.audioGenerationId === current?.id) return item.audioStatus;
    return current ? "PENDING" : "NOT_GENERATED";
  });
  const status = createMemo<AudioStatus>(() => {
    const latest = observed()?.status ?? fallbackStatus();
    const current = job();
    if (
      latest === "READY" &&
      (current?.kind === "preview"
        ? !matches(inputs(), current.inputs)
        : options.synthesisChanged())
    )
      return "OUTDATED";
    return latest;
  });
  const active = createMemo(() => {
    const latest = observed();
    return latest
      ? Boolean(latest.active)
      : fallbackStatus() === "PENDING" || fallbackStatus() === "PROCESSING";
  });
  const assetId = createMemo(() => {
    if (status() !== "READY") return;
    const current = job();
    const item = options.item();
    return (
      observed()?.assetId ?? (current?.id === item?.audioGenerationId ? item?.audioAssetId : null)
    );
  });
  const media = createMemo(() => {
    const asset = assetId();
    const current = job();
    if (!asset || !current) return;
    return current.kind === "preview"
      ? previewAudioURLs(current.id)
      : draftAudioURLs(current.itemId, asset);
  });

  createEffect(
    () => [job(), pollAgain()] as const,
    ([current]) => {
      setPollError("");
      if (!current) return;
      const controller = new AbortController();
      let timer: ReturnType<typeof setTimeout> | undefined;
      const stop = setTimeout(() => {
        controller.abort();
        if (timer) clearTimeout(timer);
        setPollError("Status checking paused. Check audio status to continue.");
      }, pollLimit);

      async function poll() {
        if (controller.signal.aborted || !current) return;
        try {
          const result =
            current.kind === "draft"
              ? await getDraftGeneration(current.itemId, current.id, controller.signal)
              : await getPreview(current.id, controller.signal);
          if (controller.signal.aborted) return;
          const lifecycle = "jobStatus" in result ? result.jobStatus : result.status;
          const expiresAt = "expiresAt" in result ? result.expiresAt : undefined;
          const isActive = lifecycle === "PENDING" || lifecycle === "PROCESSING";
          setObservation({
            job: current,
            status: result.status,
            assetId: result.audioAssetId,
            error: result.errorCode,
            expiresAt,
            active: isActive,
          });
          setPollError("");
          if (isActive) {
            timer = setTimeout(() => void poll(), pollInterval);
          } else {
            clearTimeout(stop);
            if (expiresAt)
              timer = setTimeout(
                () => {
                  setObservation({ job: current, status: "EXPIRED" });
                },
                Math.max(0, Date.parse(expiresAt) - Date.now()),
              );
          }
        } catch (error) {
          if (controller.signal.aborted) return;
          if (error instanceof ApiError && error.status < 500) {
            clearTimeout(stop);
            setObservation({
              job: current,
              status: error.code === "PREVIEW_NOT_FOUND" ? "EXPIRED" : "FAILED",
            });
            setPollError("");
          } else {
            setPollError("Connection lost. Checking audio status…");
            timer = setTimeout(() => void poll(), pollInterval);
          }
        }
      }
      void poll();
      return () => {
        clearTimeout(stop);
        if (timer) clearTimeout(timer);
        controller.abort();
      };
    },
  );

  const canGenerate = createMemo(() => {
    const source = options.values().sourceText;
    return (
      Boolean(inputs()) &&
      /[\p{L}\p{N}]/u.test(source) &&
      source.length <= 1500 &&
      source.isWellFormed() &&
      !(options.item() && options.synthesisChanged()) &&
      !options.editingBlocked() &&
      !submitting() &&
      !active() &&
      !attempt()
    );
  });

  async function submit(intent: Attempt) {
    if (request) return;
    const controller = new AbortController();
    request = controller;
    const signal = AbortSignal.any([controller.signal, AbortSignal.timeout(15_000)]);
    setAttempt(intent);
    setSubmitting(true);
    setActionError("");
    try {
      const target = intent.target;
      const receipt =
        target.kind === "draft"
          ? await generateDraft(target.itemId, target.version, intent.key, signal)
          : await generatePreview(target.inputs, intent.key, signal);
      if (controller.signal.aborted) return;
      setAccepted({ ...target, id: receipt.generationId });
      setAttempt(undefined);
    } catch (error) {
      if (controller.signal.aborted) return;
      if (error instanceof ApiError && (error.status < 500 || error.code === "TTS_UNAVAILABLE")) {
        setAttempt(undefined);
        setActionError(
          error.code === "VERSION_CONFLICT"
            ? "The saved draft changed. Reopen it before generating audio."
            : error.message,
        );
      } else {
        setActionError(
          "Could not confirm the generation request. Recover it to check the original outcome.",
        );
      }
    } finally {
      if (request === controller) {
        request = undefined;
        setSubmitting(false);
      }
    }
  }

  function generate() {
    const currentInputs = inputs();
    if (!canGenerate() || !currentInputs) return;
    const item = options.item();
    const target: Target = item
      ? { kind: "draft", itemId: item.id, version: item.version }
      : { kind: "preview", inputs: currentInputs };
    void submit({ target, key: crypto.randomUUID() });
  }

  function recover() {
    const previous = attempt();
    if (previous && !options.editingBlocked()) void submit(previous);
  }

  function reset() {
    request?.abort();
    request = undefined;
    setAttempt(undefined);
    setAccepted(undefined);
    setSubmitting(false);
    setActionError("");
  }
  onCleanup(() => {
    request?.abort();
    request = undefined;
  });

  function claimablePreview() {
    const current = job();
    const view = observed();
    return current?.kind === "preview" &&
      status() === "READY" &&
      view?.assetId &&
      view.expiresAt &&
      Date.parse(view.expiresAt) > Date.now()
      ? current.id
      : undefined;
  }

  function releasePreview() {
    const current = job();
    if (current?.kind === "preview") setObservation({ job: current, status: "EXPIRED" });
  }

  return {
    voices,
    voiceError,
    selection,
    inputs,
    retryVoices: () => setVoiceAttempt((count) => count + 1),
    changeSpeech,
    status,
    assetId,
    media,
    submitting,
    actionError,
    pollError,
    active,
    errorCode: () => observed()?.error,
    checkStatus: () => setPollAgain((count) => count + 1),
    needsRecovery: () => Boolean(attempt()),
    canGenerate,
    generate,
    recover,
    reset,
    claimablePreview,
    releasePreview,
  };
}
