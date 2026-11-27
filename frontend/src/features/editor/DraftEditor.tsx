import {
  useBeforeLeave,
  useLocation,
  useNavigate,
  type BeforeLeaveEventArgs,
} from "@solidjs/router";
import { Title } from "@solidjs/meta";
import {
  For,
  Show,
  createEffect,
  createMemo,
  createSignal,
  createStore,
  onCleanup,
  until,
} from "solid-js";
import { ApiError, createItem, getItem, patchItem, type Item } from "../../api/client";
import controls from "../../components/Controls.module.css";
import { usePreferences } from "../../preferences/Preferences";
import AudioPanel from "./AudioPanel";
import {
  difficulties,
  draftFields,
  draftFieldChanged,
  draftPatch,
  editableFields,
  emptyDraft,
  rebaseDraft,
  textFields,
  textLength,
  textLimits,
  type DraftFields,
} from "./draft";
import { DraftForm } from "./DraftForm";
import styles from "./DraftEditor.module.css";
import { useDraftAudio } from "./useDraftAudio";
import { usePublication } from "./usePublication";

const invalidAddress = Symbol("invalid draft address");
const draftUUID = /^[0-9a-f]{8}(?:-[0-9a-f]{4}){3}-[0-9a-f]{12}$/i;

function draftIdFrom(search: string) {
  const ids = new URLSearchParams(search).getAll("draftId");
  if (ids.length === 0) return null;
  return ids.length === 1 && draftUUID.test(ids[0]) ? ids[0].toLowerCase() : invalidAddress;
}

type LoadState =
  | { phase: "loading" | "ready" }
  | { phase: "error"; message: string; retry: boolean };
type Conflict =
  | { phase: "unreviewed" | "loading" }
  | { phase: "error"; message: string }
  | { phase: "ready"; latest: Item };

function draftReadError(error: unknown) {
  if (error instanceof ApiError && error.code === "ITEM_NOT_FOUND")
    return "This draft was not found.";
  return "Could not load this draft. Try again.";
}

export default function DraftEditor() {
  const location = useLocation();
  const navigate = useNavigate();
  const preferences = usePreferences();
  const requestedId = createMemo(() => draftIdFrom(location.search));
  const [draft, setDraft] = createStore({
    item: null as Item | null,
    baseline: emptyDraft(""),
    values: emptyDraft(""),
  });
  const [load, setLoad] = createSignal<LoadState>({ phase: "loading" });
  const [loadAttempt, setLoadAttempt] = createSignal(0);
  const [saving, setSaving] = createSignal(false);
  const [showValidation, setShowValidation] = createSignal(false);
  const [saveError, setSaveError] = createSignal("");
  const [conflict, setConflict] = createSignal<Conflict | null>(null);
  const [leaveRequest, setLeaveRequest] = createSignal<BeforeLeaveEventArgs>();
  let loadedId: string | null | undefined;
  let canonicalPath: string | undefined;
  let reviewRequest: AbortController | undefined;
  let leaveDialog!: HTMLDialogElement;

  const synthesisInputsChanged = createMemo(
    () =>
      draft.values.languageCode !== draft.baseline.languageCode ||
      draft.values.sourceText !== draft.baseline.sourceText ||
      draft.values.voiceId !== draft.baseline.voiceId ||
      draft.values.speakingRate !== draft.baseline.speakingRate,
  );
  const dirty = createMemo(
    () =>
      synthesisInputsChanged() ||
      editableFields.some(({ id }) => draftFieldChanged(id, draft.values, draft.baseline)),
  );
  const errors = createMemo(() => {
    const result: Partial<Record<keyof DraftFields, string>> = {};
    if (
      !draft.item &&
      !preferences.languages()?.some((entry) => entry.code === draft.values.languageCode)
    ) {
      result.languageCode = preferences.languageError()
        ? "Item languages are unavailable."
        : "Choose an enabled language.";
    }
    for (const name of textFields) {
      const value = draft.values[name];
      if (textLength(name, value) > textLimits[name]) {
        result[name] =
          name === "sourceText"
            ? "Use at most 1500 UTF-16 units."
            : `Use at most ${textLimits[name]} characters.`;
      } else if (value.includes("\0") || !value.isWellFormed()) {
        result[name] = "Remove unsupported characters.";
      }
    }
    return result;
  });
  const shownErrors = createMemo(() => ({
    ...errors(),
    languageCode: showValidation() ? errors().languageCode : undefined,
  }));
  const audio = useDraftAudio({
    item: () => draft.item,
    values: () => draft.values,
    synthesisChanged: synthesisInputsChanged,
    editingBlocked: (): boolean => saving() || Boolean(conflict()) || publication.locked(),
    setSpeech(voiceId, speakingRate) {
      setDraft((state) => {
        state.values.voiceId = voiceId;
        state.values.speakingRate = speakingRate;
      });
    },
  });
  const publication = usePublication({
    audio,
    item: () => draft.item,
    values: () => draft.values,
    baseline: () => draft.baseline,
    dirty,
    synthesisChanged: synthesisInputsChanged,
    invalid: () => Object.keys(errors()).length > 0,
    saving,
    conflict: () => Boolean(conflict()),
    saved: confirmDraft,
    review: () => setConflict({ phase: "unreviewed" }),
    openReader(id) {
      canonicalPath = `/items/${encodeURIComponent(id)}`;
      navigate(canonicalPath, { replace: true });
      canonicalPath = undefined;
    },
  });

  function setField<K extends keyof DraftFields>(name: K, value: DraftFields[K]) {
    setDraft((state) => {
      state.values[name] = value;
      if (name === "languageCode") {
        state.values.voiceId = null;
        state.values.speakingRate = null;
      }
    });
  }

  function confirmDraft(item: Item | null, values?: DraftFields) {
    const baseline = item ? draftFields(item) : emptyDraft(preferences.language());
    setDraft((state) => {
      state.item = item;
      state.baseline = baseline;
      state.values = values ?? { ...baseline };
    });
    setShowValidation(false);
    setSaveError("");
    setConflict(null);
  }

  createEffect(
    () => [requestedId(), loadAttempt()] as const,
    ([id]) => {
      if (id === loadedId) {
        setLoad({ phase: "ready" });
        return;
      }
      reviewRequest?.abort();
      audio.reset();
      publication.clearMessage();
      if (id === null) {
        loadedId = null;
        confirmDraft(null);
        setLoad({ phase: "ready" });
        return;
      }
      if (id === invalidAddress) {
        setLoad({ phase: "error", message: "This draft link is invalid.", retry: false });
        return;
      }

      const controller = new AbortController();
      setLoad({ phase: "loading" });
      getItem(id, controller.signal).then(
        (item) => {
          if (controller.signal.aborted) return;
          if (item.status !== "draft") {
            setLoad({
              phase: "error",
              message: "This article is already published.",
              retry: false,
            });
            return;
          }
          loadedId = id;
          confirmDraft(item);
          setLoad({ phase: "ready" });
        },
        (error) => {
          if (!controller.signal.aborted) {
            setLoad({ phase: "error", message: draftReadError(error), retry: true });
          }
        },
      );
      return () => controller.abort();
    },
  );
  onCleanup(() => reviewRequest?.abort());

  async function saveDraft() {
    if (saving() || conflict() || audio.submitting() || publication.locked()) return false;
    const invalidField = Object.keys(errors())[0];
    if (invalidField) {
      setShowValidation(true);
      document.getElementById(`draft-${invalidField}`)?.focus();
      return false;
    }
    const item = draft.item;
    if (item && !dirty()) return true;
    const values = { ...draft.values };
    const languageCode = values.languageCode;
    if (languageCode === "") return false;
    const speech = audio.inputs();
    const previewGenerationId = item ? undefined : audio.claimablePreview();
    setSaveError("");
    setSaving(true);

    let saved: Item;
    try {
      saved = item
        ? await patchItem(item.id, draftPatch(values, draft.baseline, item.version))
        : await createItem({
            languageCode,
            title: values.title || null,
            sourceText: values.sourceText,
            description: values.description || null,
            notes: values.notes || null,
            difficulty: values.difficulty || null,
            topicIds: values.topicIds,
            voiceId: speech?.voiceId ?? values.voiceId,
            speakingRate: speech?.speakingRate ?? values.speakingRate,
            previewGenerationId,
          });
    } catch (error) {
      if (error instanceof ApiError && error.code === "VERSION_CONFLICT") {
        setConflict({ phase: "unreviewed" });
      } else if (
        previewGenerationId &&
        error instanceof ApiError &&
        ["PREVIEW_NOT_FOUND", "PREVIEW_MISMATCH", "AUDIO_UNAVAILABLE"].includes(error.code)
      ) {
        audio.releasePreview();
        setSaveError("This preview is unavailable. Choose Save Draft again to save your text.");
      } else {
        setSaveError(
          error instanceof ApiError
            ? error.message
            : "Could not confirm the draft was saved. Your edits are still here.",
        );
      }
      setSaving(false);
      return false;
    }

    loadedId = saved.id;
    confirmDraft(saved);
    if (!item) {
      audio.reset();
      canonicalPath = `/create?draftId=${encodeURIComponent(saved.id)}`;
      navigate(canonicalPath, { replace: true, scroll: false });
      canonicalPath = undefined;
      await until(() => requestedId() === saved.id);
    }
    setSaving(false);
    return true;
  }

  async function reviewLatest() {
    const item = draft.item;
    if (!item || conflict()?.phase === "loading") return;
    const controller = new AbortController();
    reviewRequest?.abort();
    reviewRequest = controller;
    setConflict({ phase: "loading" });
    try {
      const latest = await getItem(item.id, controller.signal);
      if (controller.signal.aborted) return;
      if (latest.status !== "draft") {
        setConflict({ phase: "error", message: "This article is already published." });
        return;
      }
      setConflict({ phase: "ready", latest });
    } catch (error) {
      if (!controller.signal.aborted) {
        setConflict({ phase: "error", message: draftReadError(error) });
      }
    }
  }

  function applyReview(keepEdits: boolean) {
    const review = conflict();
    if (review?.phase !== "ready") return;
    const values = keepEdits
      ? rebaseDraft(draft.values, draft.baseline, draftFields(review.latest))
      : undefined;
    confirmDraft(review.latest, values);
  }

  useBeforeLeave((event) => {
    if (event.defaultPrevented || (!dirty() && !saving() && !publication.locked())) return;
    if (typeof event.to === "string") {
      if (event.to === canonicalPath) return;
      const destination = new URL(event.to, "http://editor.local");
      if (destination.pathname === "/create" && draftIdFrom(destination.search) === requestedId()) {
        return;
      }
    }
    event.preventDefault();
    if (!leaveRequest() && !saving() && !publication.locked()) setLeaveRequest(event);
  });

  createEffect(
    () => dirty() || publication.locked(),
    (hasEdits) => {
      if (!hasEdits) return;
      function warn(event: BeforeUnloadEvent) {
        event.preventDefault();
        event.returnValue = true;
      }
      window.addEventListener("beforeunload", warn);
      return () => window.removeEventListener("beforeunload", warn);
    },
  );

  createEffect(leaveRequest, (request) => {
    if (request && !leaveDialog.open) leaveDialog.showModal();
    if (!request && leaveDialog.open) leaveDialog.close();
  });

  async function saveAndLeave() {
    const request = leaveRequest();
    if (!request || saving()) return;
    const saved = await saveDraft();
    setLeaveRequest(undefined);
    if (saved) request.retry(true);
  }

  function discardAndLeave() {
    const request = leaveRequest();
    if (!request || saving()) return;
    confirmDraft(draft.item);
    setLeaveRequest(undefined);
    request.retry(true);
  }

  const heading = () => (requestedId() === null ? "New article" : "Edit draft");
  const canReview = () => conflict()?.phase === "ready";
  const reviewValues = createMemo(() => {
    const review = conflict();
    return review?.phase === "ready" ? draftFields(review.latest) : undefined;
  });
  const changedFields = createMemo(() => {
    const latest = reviewValues();
    return latest
      ? editableFields.filter(({ id }) => draftFieldChanged(id, latest, draft.baseline))
      : [];
  });

  return (
    <section class={styles.editor}>
      <Title>{heading()} · TaoTale</Title>
      <header class={styles.header}>
        <h1>{heading()}</h1>
        <a href="/library" class={controls.secondary}>
          Back to Library
        </a>
      </header>
      <Show when={load().phase === "loading"}>
        <p role="status" class={styles.message}>
          Loading draft…
        </p>
      </Show>
      <Show
        when={(() => {
          const state = load();
          return state.phase === "error" ? state : undefined;
        })()}
      >
        {(failed) => (
          <div class={styles.message}>
            <p role="alert">{failed().message}</p>
            <Show when={failed().retry}>
              <button
                type="button"
                class={controls.secondary}
                onClick={() => setLoadAttempt((n) => n + 1)}
              >
                Retry
              </button>
            </Show>
          </div>
        )}
      </Show>
      <Show when={load().phase === "ready"}>
        <form
          class={styles.form}
          novalidate
          onSubmit={(event) => {
            event.preventDefault();
            void saveDraft();
          }}
        >
          <DraftForm
            values={draft.values}
            errors={shownErrors()}
            languages={preferences.languages()}
            languageError={preferences.languageError()}
            retryLanguages={preferences.retryLanguages}
            saved={Boolean(draft.item)}
            saving={saving() || publication.locked()}
            audio={
              <AudioPanel
                audio={audio}
                saved={Boolean(draft.item)}
                changed={synthesisInputsChanged()}
                disabled={saving() || Boolean(conflict()) || publication.locked()}
              />
            }
            update={setField}
          />
          <Show when={saveError()}>
            <p role="alert" class={styles.alert}>
              {saveError()}
            </p>
          </Show>
          <Show when={publication.message()}>
            <p role="alert" class={styles.alert}>
              {publication.message()}
            </p>
          </Show>
          <Show when={conflict()}>
            <section class={styles.review} aria-labelledby="draft-review">
              <h2 id="draft-review">Review the latest draft</h2>
              <p role="alert">This draft changed since you opened it. Your edits are still here.</p>
              <Show
                when={canReview()}
                fallback={
                  <div class={styles.reviewActions}>
                    <Show
                      when={(() => {
                        const review = conflict();
                        return review?.phase === "error" ? review.message : undefined;
                      })()}
                    >
                      {(message) => <p role="alert">{message()}</p>}
                    </Show>
                    <button
                      type="button"
                      class={controls.secondary}
                      disabled={conflict()?.phase === "loading"}
                      onClick={() => void reviewLatest()}
                    >
                      {conflict()?.phase === "loading"
                        ? "Loading latest version…"
                        : "Load latest version"}
                    </button>
                  </div>
                }
              >
                <Show
                  when={changedFields().length}
                  fallback={<p>The editable fields match the version you opened.</p>}
                >
                  <For each={changedFields()}>
                    {({ id, label }) => (
                      <div class={styles.comparison}>
                        <h3>{label}</h3>
                        <dl>
                          <div>
                            <dt>Latest saved</dt>
                            <dd>{fieldText(id, reviewValues()?.[id])}</dd>
                          </div>
                          <div>
                            <dt>Your draft</dt>
                            <dd>{fieldText(id, draft.values[id])}</dd>
                          </div>
                        </dl>
                      </div>
                    )}
                  </For>
                </Show>
                <p>
                  Keeping your edits applies them to the latest version. Review and save when ready.
                </p>
                <div class={styles.actions}>
                  <button
                    type="button"
                    class={controls.secondary}
                    onClick={() => applyReview(false)}
                  >
                    Use latest version
                  </button>
                  <button type="button" class={controls.primary} onClick={() => applyReview(true)}>
                    Keep my edits
                  </button>
                </div>
              </Show>
            </section>
          </Show>
          <footer class={styles.footer}>
            <div aria-live="polite" class={styles.saveStatus}>
              {saving()
                ? "Saving draft…"
                : dirty()
                  ? "Unsaved changes"
                  : draft.item
                    ? "Draft saved"
                    : ""}
              <Show when={synthesisInputsChanged() && draft.item?.audioStatus === "READY"}>
                <p>Saving these text changes will make the current audio outdated.</p>
              </Show>
            </div>
            <div class={styles.footerActions}>
              <Show when={publication.reason()}>
                <p id="publish-reason" class={styles.hint}>
                  {publication.reason()}
                </p>
              </Show>
              <button
                type="button"
                class={controls.secondary}
                disabled={
                  publication.pending() ||
                  (Boolean(publication.reason()) && !publication.uncertain())
                }
                aria-describedby={publication.reason() ? "publish-reason" : undefined}
                onClick={() => void publication.submit()}
              >
                {publication.uncertain() ? "Recover publication" : "Publish to Library"}
              </button>
              <button
                type="submit"
                class={controls.primary}
                disabled={
                  saving() ||
                  publication.locked() ||
                  audio.submitting() ||
                  Boolean(conflict()) ||
                  (Boolean(draft.item) && !dirty())
                }
              >
                Save Draft
              </button>
            </div>
          </footer>
        </form>
      </Show>
      <dialog
        class={styles.dialog}
        ref={leaveDialog}
        aria-labelledby="draft-leave-heading"
        aria-describedby="draft-leave-description"
        onCancel={(event) => {
          event.preventDefault();
          if (!saving()) setLeaveRequest(undefined);
        }}
      >
        <h2 id="draft-leave-heading">Leave this draft?</h2>
        <p id="draft-leave-description">
          {saving() ? "Saving draft…" : "Your changes have not been saved."}
        </p>
        <div class={styles.actions}>
          <button
            type="button"
            class={controls.secondary}
            autofocus
            disabled={saving()}
            onClick={() => setLeaveRequest(undefined)}
          >
            Stay
          </button>
          <button
            type="button"
            class={controls.secondary}
            disabled={saving()}
            onClick={discardAndLeave}
          >
            Leave Without Saving
          </button>
          <button
            type="button"
            class={controls.primary}
            disabled={saving() || audio.submitting() || Boolean(conflict())}
            onClick={() => void saveAndLeave()}
          >
            Save Draft &amp; Leave
          </button>
        </div>
      </dialog>
    </section>
  );
}

function fieldText(
  name: (typeof editableFields)[number]["id"],
  value: DraftFields[typeof name] | undefined,
) {
  if (name === "difficulty") return difficulties.find((entry) => entry.value === value)?.label;
  if (name === "speakingRate") return value ? `${value}×` : "Default";
  if (name === "voiceId") return value ?? "Default";
  if (Array.isArray(value)) return `${value.length} selected`;
  return value || "Empty";
}
