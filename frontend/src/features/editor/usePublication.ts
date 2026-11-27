import { createMemo, createSignal } from "solid-js";
import {
  ApiError,
  getItem,
  patchItem,
  publishDraft,
  publishPreview,
  type DirectPublication,
  type Item,
} from "../../api/client";
import { draftPatch, type DraftFields } from "./draft";
import type { useDraftAudio } from "./useDraftAudio";

type Intent =
  | { kind: "saved"; id: string; version: number; asset: string }
  | { kind: "direct"; key: string; body: DirectPublication };

export function usePublication(options: {
  audio: ReturnType<typeof useDraftAudio>;
  item: () => Item | null;
  values: () => DraftFields;
  baseline: () => DraftFields;
  dirty: () => boolean;
  synthesisChanged: () => boolean;
  invalid: () => boolean;
  saving: () => boolean;
  conflict: () => boolean;
  saved: (item: Item) => void;
  review: () => void;
  openReader: (id: string) => void;
}) {
  const audio = options.audio;
  const [pending, setPending] = createSignal(false);
  const [uncertain, setUncertain] = createSignal<Intent>();
  const [message, setMessage] = createSignal("");
  const locked = () => pending() || Boolean(uncertain());
  const reason = createMemo(() => {
    if (pending()) return "Publishing…";
    if (uncertain()) return "Recover the previous publication to confirm its outcome.";
    if (options.saving()) return "Waiting for your draft to finish saving.";
    if (options.conflict()) return "Review the latest draft before publishing.";
    if (!options.values().title.trim() || !options.values().sourceText.trim())
      return "Add a title and source text before publishing.";
    if (options.invalid()) return "Correct the highlighted fields before publishing.";
    if (options.item() && options.synthesisChanged())
      return "Save your text, voice, and rate changes before publishing.";
    if (audio.needsRecovery() || audio.submitting())
      return "Confirm the audio request before publishing.";
    if (audio.active()) return "Waiting for audio generation to finish.";
    if (!audio.assetId() || (!options.item() && !audio.claimablePreview()))
      return "Generate matching audio before publishing.";
    return "";
  });

  async function submit() {
    const previous = uncertain();
    if (pending() || (!previous && reason())) return;
    setPending(true);
    setMessage("");
    let intent = previous;
    try {
      if (!intent) {
        let item = options.item();
        const values = options.values();
        const asset = audio.assetId();
        if (!asset) return;
        if (item) {
          if (options.dirty()) {
            item = await patchItem(item.id, draftPatch(values, options.baseline(), item.version));
            options.saved(item);
          }
          intent = { kind: "saved", id: item.id, version: item.version, asset };
        } else {
          const speech = audio.inputs();
          const previewGenerationId = audio.claimablePreview();
          if (!speech || !previewGenerationId) return;
          intent = {
            kind: "direct",
            key: crypto.randomUUID(),
            body: {
              ...speech,
              previewGenerationId,
              title: values.title,
              description: values.description || null,
              notes: values.notes || null,
              difficulty: values.difficulty || null,
              topicIds: values.topicIds,
            },
          };
        }
      }
      if (previous?.kind === "saved") {
        const latest = await getItem(previous.id, AbortSignal.timeout(15_000));
        if (latest.status === "published") {
          options.openReader(latest.id);
          return;
        }
      }
      const result =
        intent.kind === "saved"
          ? await publishDraft(intent.id, intent.version, intent.asset)
          : await publishPreview(intent.body, intent.key);
      options.openReader(result.itemId);
    } catch (error) {
      if (
        error instanceof ApiError &&
        ((error.status >= 400 && error.status < 500) || error.code === "AUDIO_UNAVAILABLE")
      ) {
        setUncertain(undefined);
        if (error.code === "VERSION_CONFLICT") options.review();
        if (error.code === "PREVIEW_NOT_FOUND") audio.releasePreview();
        if (["AUDIO_NOT_READY", "AUDIO_UNAVAILABLE", "PREVIEW_MISMATCH"].includes(error.code))
          audio.checkStatus();
        setMessage(error.message);
      } else if (intent) {
        setUncertain(intent);
        setMessage(
          "The server did not confirm publication. Recover it to check the original request.",
        );
      } else {
        setMessage("Could not confirm the metadata save. Your edits are still here.");
      }
    } finally {
      setPending(false);
    }
  }

  return {
    reason,
    locked,
    pending,
    uncertain,
    message,
    submit,
    clearMessage: () => setMessage(""),
  };
}
