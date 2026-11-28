import { Title } from "@solidjs/meta";
import { useBeforeLeave, useNavigate, type BeforeLeaveEventArgs } from "@solidjs/router";
import { Show, createEffect, createMemo, createSignal, onCleanup } from "solid-js";
import {
  ApiError,
  discardRevision,
  getItem,
  getRevision,
  openRevision,
  patchRevision,
  publishRevision,
  type WorkingRevision,
} from "../../api/client";
import controls from "../../components/Controls.module.css";
import { clearPlaybackResume } from "../reader/playback";
import AudioPanel from "./AudioPanel";
import styles from "./DraftEditor.module.css";
import { textLimits } from "./draft";
import { useDraftAudio } from "./useDraftAudio";

type Speech = Pick<WorkingRevision, "sourceText" | "voiceId" | "speakingRate">;
const speechFields = ["sourceText", "voiceId", "speakingRate"] as const;

function speech(revision: Speech): Speech {
  return {
    sourceText: revision.sourceText,
    voiceId: revision.voiceId,
    speakingRate: revision.speakingRate,
  };
}

export default function RevisionEditor(props: { itemId: string }) {
  const navigate = useNavigate();
  const readerURL = () => `/items/${encodeURIComponent(props.itemId)}`;
  const [revision, setRevision] = createSignal<WorkingRevision | null>(null);
  const [values, setValues] = createSignal<Speech>(
    speech({ sourceText: "", voiceId: null, speakingRate: null }),
  );
  const [loadAttempt, setLoadAttempt] = createSignal(0);
  const [busy, setBusy] = createSignal(false);
  const [loadError, setLoadError] = createSignal("");
  const [message, setMessage] = createSignal("");
  const [saved, setSaved] = createSignal(false);
  const [needsReview, setNeedsReview] = createSignal(false);
  const [latest, setLatest] = createSignal<WorkingRevision>();
  const [uncertainAsset, setUncertainAsset] = createSignal<string>();
  const [leaveRequest, setLeaveRequest] = createSignal<BeforeLeaveEventArgs>();
  const [discarding, setDiscarding] = createSignal(false);
  let leaveDialog!: HTMLDialogElement;
  let discardDialog!: HTMLDialogElement;
  let leaving = false;
  let mounted = true;
  onCleanup(() => {
    mounted = false;
  });

  const dirty = createMemo(() => {
    const baseline = revision();
    return Boolean(baseline && speechFields.some((field) => values()[field] !== baseline[field]));
  });
  const sourceError = createMemo(() => {
    const source = values().sourceText;
    if (source.length > textLimits.sourceText) return "Use at most 1500 UTF-16 units.";
    if (source.includes("\0") || !source.isWellFormed()) return "Remove unsupported characters.";
    return "";
  });
  const blocked = () => busy() || needsReview() || Boolean(uncertainAsset());
  const audio = useDraftAudio({
    enabled: () => Boolean(revision()),
    item: revision,
    values: () => ({ ...values(), languageCode: revision()?.languageCode ?? "" }),
    synthesisChanged: dirty,
    editingBlocked: () => blocked() || Boolean(sourceError()),
    setSpeech(voiceId, speakingRate) {
      setValues((current) => ({ ...current, voiceId, speakingRate }));
      setSaved(false);
    },
    review: () => setNeedsReview(true),
  });
  const canPublish = () =>
    !blocked() &&
    !dirty() &&
    !audio.submitting() &&
    !audio.needsRecovery() &&
    !audio.active() &&
    Boolean(audio.assetId());

  function confirm(rev: WorkingRevision, edits = speech(rev)) {
    setRevision(rev);
    setValues(edits);
    setNeedsReview(false);
    setLatest(undefined);
    setMessage("");
  }

  createEffect(loadAttempt, () => {
    const controller = new AbortController();
    setLoadError("");
    openRevision(props.itemId, controller.signal).then(
      (rev) => {
        if (!controller.signal.aborted) confirm(rev);
      },
      (error) => {
        if (!controller.signal.aborted)
          setLoadError(
            error instanceof ApiError ? error.message : "Could not open the working revision.",
          );
      },
    );
    return () => controller.abort();
  });

  function mutationError(error: unknown, fallback: string) {
    if (error instanceof ApiError) {
      if (error.code === "VERSION_CONFLICT") setNeedsReview(true);
      if (["AUDIO_UNAVAILABLE", "AUDIO_NOT_READY"].includes(error.code)) audio.checkStatus();
      setMessage(error.message);
    } else {
      setMessage(fallback);
    }
  }

  async function save() {
    const rev = revision();
    if (!rev || blocked() || audio.submitting() || sourceError()) return false;
    if (!dirty()) return true;
    setBusy(true);
    setMessage("");
    try {
      confirm(
        await patchRevision(rev.itemId, rev.id, {
          expectedVersion: rev.version,
          ...speech(values()),
        }),
      );
      setSaved(true);
      return true;
    } catch (error) {
      mutationError(error, "Could not confirm the revision was saved. Your edits are still here.");
      return false;
    } finally {
      setBusy(false);
    }
  }

  async function review() {
    const rev = revision();
    if (!rev || busy()) return;
    setBusy(true);
    try {
      setLatest(await getRevision(rev.itemId, rev.id, AbortSignal.timeout(15_000)));
      setMessage("");
    } catch (error) {
      mutationError(error, "Could not load the latest revision.");
    } finally {
      setBusy(false);
    }
  }

  function applyReview(keepEdits: boolean) {
    const current = latest();
    const old = revision();
    if (!current || !old) return;
    const original = values();
    audio.reset();
    confirm(current, {
      sourceText:
        keepEdits && original.sourceText !== old.sourceText
          ? original.sourceText
          : current.sourceText,
      voiceId: keepEdits && original.voiceId !== old.voiceId ? original.voiceId : current.voiceId,
      speakingRate:
        keepEdits && original.speakingRate !== old.speakingRate
          ? original.speakingRate
          : current.speakingRate,
    });
  }

  function finish(url: string) {
    if (!mounted) return;
    leaving = true;
    navigate(url, { replace: true });
  }

  async function publish() {
    const rev = revision();
    const recovery = uncertainAsset();
    const asset = recovery ?? audio.assetId();
    if (!rev || !asset || busy() || (!recovery && !canPublish())) return;
    setBusy(true);
    setMessage("");
    try {
      if (
        !recovery ||
        (await getItem(rev.itemId, AbortSignal.timeout(15_000))).audioAssetId !== asset
      ) {
        await publishRevision(rev.itemId, rev.id, {
          expectedVersion: rev.version,
          expectedItemVersion: rev.itemVersion,
          audioAssetId: asset,
        });
      }
      clearPlaybackResume(rev.itemId, rev.baseAudioAssetId);
      finish(readerURL());
    } catch (error) {
      if (
        error instanceof ApiError &&
        ((error.status >= 400 && error.status < 500) || error.code === "AUDIO_UNAVAILABLE")
      ) {
        setUncertainAsset(undefined);
        mutationError(error, "");
      } else {
        setUncertainAsset(asset);
        setMessage("The server did not confirm publication. Check the original request.");
      }
    } finally {
      setBusy(false);
    }
  }

  async function discard() {
    const rev = revision();
    if (!rev || blocked() || audio.submitting()) return;
    setDiscarding(false);
    setBusy(true);
    try {
      await discardRevision(rev.itemId, rev.id, rev.version);
      finish(`${readerURL()}/edit`);
    } catch (error) {
      mutationError(error, "Could not confirm the revision was discarded.");
    } finally {
      setBusy(false);
    }
  }

  useBeforeLeave((event) => {
    if (leaving || event.defaultPrevented || (!dirty() && !busy())) return;
    event.preventDefault();
    if (!busy() && !leaveRequest()) setLeaveRequest(event);
  });
  createEffect(
    () => dirty() || busy(),
    (warn) => {
      if (!warn) return;
      const handler = (event: BeforeUnloadEvent) => {
        event.preventDefault();
        event.returnValue = true;
      };
      window.addEventListener("beforeunload", handler);
      return () => window.removeEventListener("beforeunload", handler);
    },
  );
  createEffect(leaveRequest, (request) => {
    if (request && !leaveDialog.open) leaveDialog.showModal();
    if (!request && leaveDialog.open) leaveDialog.close();
  });
  createEffect(discarding, (requested) => {
    if (requested && !discardDialog.open) discardDialog.showModal();
    if (!requested && discardDialog.open) discardDialog.close();
  });

  async function saveAndLeave() {
    const request = leaveRequest();
    if (!request || busy()) return;
    const confirmed = await save();
    setLeaveRequest(undefined);
    if (confirmed) request.retry(true);
  }

  return (
    <section class={styles.editor}>
      <Title>Edit Text &amp; Voice · TaoTale</Title>
      <header class={styles.header}>
        <h1>Edit Text &amp; Voice</h1>
        <div class={styles.actions}>
          <a class={controls.secondary} href={`${readerURL()}/edit`}>
            Back to Edit Article
          </a>
          <a class={controls.secondary} href={readerURL()}>
            Back to Reader
          </a>
        </div>
      </header>
      <Show
        when={revision()}
        fallback={
          <div class={styles.message}>
            <Show when={loadError()} fallback={<p role="status">Opening working revision…</p>}>
              <p role="alert">{loadError()}</p>
              <button
                type="button"
                class={controls.secondary}
                onClick={() => setLoadAttempt((n) => n + 1)}
              >
                Retry
              </button>
            </Show>
          </div>
        }
      >
        <form
          class={styles.form}
          onSubmit={(event) => {
            event.preventDefault();
            void save();
          }}
        >
          <section class={styles.paper}>
            <p class={styles.hint}>
              Readers hear the published recording until you publish this revision.
            </p>
            <div class={styles.field}>
              <label for="revision-source">Source text</label>
              <textarea
                id="revision-source"
                class={styles.source}
                value={values().sourceText}
                disabled={busy() || Boolean(uncertainAsset())}
                aria-invalid={sourceError() ? "true" : undefined}
                aria-describedby="revision-source-length"
                onInput={(event) => {
                  const sourceText = event.currentTarget.value;
                  setValues((current) => ({ ...current, sourceText }));
                  setSaved(false);
                }}
              />
              <p id="revision-source-length" class={styles.count}>
                {values().sourceText.length} / {textLimits.sourceText}
              </p>
              <Show when={sourceError()}>
                <p class={styles.fieldError} role="alert">
                  {sourceError()}
                </p>
              </Show>
            </div>
          </section>
          <AudioPanel audio={audio} saved changed={dirty()} disabled={blocked()} />
          <Show when={message()}>
            <p class={styles.alert} role="alert">
              {message()}
            </p>
          </Show>
          <Show when={needsReview()}>
            <section class={styles.review}>
              <h2>Review the latest revision</h2>
              <Show
                when={latest()}
                fallback={
                  <button
                    type="button"
                    class={controls.secondary}
                    disabled={busy()}
                    onClick={() => void review()}
                  >
                    Load latest version
                  </button>
                }
              >
                {(current) => (
                  <>
                    <p>
                      <strong>Latest saved text</strong>
                    </p>
                    <p>{current().sourceText}</p>
                    <p>
                      Voice {current().voiceId ?? "Default"} · Speaking rate{" "}
                      {current().speakingRate ?? 1}×
                    </p>
                    <div class={styles.actions}>
                      <button
                        type="button"
                        class={controls.secondary}
                        onClick={() => applyReview(false)}
                      >
                        Use Latest Version
                      </button>
                      <button
                        type="button"
                        class={controls.primary}
                        onClick={() => applyReview(true)}
                      >
                        Keep My Edits
                      </button>
                    </div>
                  </>
                )}
              </Show>
            </section>
          </Show>
          <footer class={styles.footer}>
            <p role="status" class={styles.saveStatus}>
              {saved() && !dirty() ? "Revision saved" : ""}
            </p>
            <div class={styles.footerActions}>
              <button
                type="button"
                class={controls.secondary}
                disabled={blocked() || audio.submitting()}
                onClick={() => setDiscarding(true)}
              >
                Discard Revision
              </button>
              <button
                type="submit"
                class={controls.secondary}
                disabled={blocked() || audio.submitting() || !dirty() || Boolean(sourceError())}
              >
                Save Revision
              </button>
              <button
                type="button"
                class={controls.primary}
                disabled={uncertainAsset() ? busy() : !canPublish()}
                onClick={() => void publish()}
              >
                {uncertainAsset() ? "Check Publication" : "Publish Revision"}
              </button>
            </div>
          </footer>
        </form>
      </Show>
      <dialog
        ref={leaveDialog}
        class={styles.dialog}
        aria-labelledby="revision-leave"
        onCancel={() => setLeaveRequest(undefined)}
      >
        <h2 id="revision-leave">Leave with unsaved changes?</h2>
        <p>You can save the working revision before leaving.</p>
        <div class={styles.actions}>
          <button
            type="button"
            class={controls.secondary}
            autofocus
            disabled={busy()}
            onClick={() => setLeaveRequest(undefined)}
          >
            Stay
          </button>
          <button
            type="button"
            class={controls.secondary}
            disabled={busy()}
            onClick={() => {
              const request = leaveRequest();
              setLeaveRequest(undefined);
              request?.retry(true);
            }}
          >
            Leave Without Saving
          </button>
          <button
            type="button"
            class={controls.primary}
            disabled={blocked() || audio.submitting() || Boolean(sourceError())}
            onClick={() => void saveAndLeave()}
          >
            Save Revision &amp; Leave
          </button>
        </div>
      </dialog>
      <dialog
        ref={discardDialog}
        class={styles.dialog}
        aria-labelledby="revision-discard"
        onCancel={() => setDiscarding(false)}
      >
        <h2 id="revision-discard">Discard this revision?</h2>
        <p>
          This deletes the working text and candidate audio. The published recording stays
          available.
        </p>
        <div class={styles.actions}>
          <button
            type="button"
            class={controls.secondary}
            autofocus
            onClick={() => setDiscarding(false)}
          >
            Keep Revision
          </button>
          <button type="button" class={controls.primary} onClick={() => void discard()}>
            Discard Revision
          </button>
        </div>
      </dialog>
    </section>
  );
}
