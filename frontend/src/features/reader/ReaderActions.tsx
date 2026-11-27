import { useBeforeLeave, type BeforeLeaveEventArgs } from "@solidjs/router";
import { Show, createEffect, createSignal } from "solid-js";
import { ApiError, getItem, patchItem, setItemFavorite, type Item } from "../../api/client";
import controls from "../../components/Controls.module.css";
import styles from "./ReaderArticle.module.css";

export default function ReaderActions(props: { item: Item; download?: string }) {
  const [favorite, setFavorite] = createSignal<boolean>();
  const [favoriting, setFavoriting] = createSignal(false);
  const [favoriteError, setFavoriteError] = createSignal(false);
  const isFavorite = () => favorite() ?? props.item.isFavorite;

  async function toggleFavorite() {
    if (favoriting()) return;
    setFavoriting(true);
    setFavoriteError(false);
    try {
      const result = await setItemFavorite(props.item.id, !isFavorite());
      setFavorite(result.isFavorite);
    } catch {
      setFavoriteError(true);
    } finally {
      setFavoriting(false);
    }
  }

  return (
    <>
      <div class={styles.readerActions}>
        <a class={controls.secondary} href={`/items/${encodeURIComponent(props.item.id)}/edit`}>
          Edit article
        </a>
        <button
          type="button"
          class={controls.secondary}
          aria-pressed={isFavorite() ? "true" : "false"}
          disabled={favoriting()}
          onClick={() => void toggleFavorite()}
        >
          {isFavorite() ? "Remove from Favorites" : "Favorite"}
        </button>
        <Show when={props.download}>
          {(href) => (
            <a class={controls.secondary} href={href()} download>
              Download Audio
            </a>
          )}
        </Show>
        <ReaderNotes item={props.item} />
      </div>
      <Show when={favoriting()}>
        <p role="status">Updating favorite…</p>
      </Show>
      <Show when={favorite() !== undefined && !favoriting() && !favoriteError()}>
        <p role="status">{isFavorite() ? "Added to Favorites." : "Removed from Favorites."}</p>
      </Show>
      <Show when={favoriteError()}>
        <p role="alert">Could not update favorite. Try again.</p>
      </Show>
    </>
  );
}

function ReaderNotes(props: { item: Item }) {
  const [visible, setVisible] = createSignal(false);
  const [saved, setSaved] = createSignal<{
    notes: string | null;
    version: number;
  }>();
  const current = () => saved() ?? props.item;
  const [draft, setDraft] = createSignal<string>();
  const [saving, setSaving] = createSignal(false);
  const [error, setError] = createSignal("");
  const [conflict, setConflict] = createSignal(false);
  const [confirmation, setConfirmation] = createSignal("");
  const [leave, setLeave] = createSignal<BeforeLeaveEventArgs>();
  let dialog!: HTMLDialogElement;
  const dirty = () => draft() !== undefined && draft() !== (current().notes ?? "");

  function cancelEdit() {
    setDraft(undefined);
    setError("");
    setConflict(false);
    setConfirmation("");
  }

  async function saveNotes() {
    if (saving() || conflict() || !dirty()) return false;
    const notes = draft()!;
    if ([...notes].length > 5000 || notes.includes("\0")) {
      setError("Notes must be plain text of up to 5,000 characters.");
      return false;
    }
    setSaving(true);
    setError("");
    try {
      const item = await patchItem(
        props.item.id,
        { expectedVersion: current().version, notes },
        AbortSignal.timeout(15_000),
      );
      setSaved({ notes: item.notes, version: item.version });
      setDraft(undefined);
      setConfirmation("Notes saved.");
      return true;
    } catch (cause) {
      if (cause instanceof ApiError && cause.code === "VERSION_CONFLICT") {
        setConflict(true);
        setError("This article changed elsewhere. Load the latest notes before editing again.");
      } else {
        setError("Could not save notes. Your edits are still here.");
      }
      return false;
    } finally {
      setSaving(false);
    }
  }

  async function loadLatest() {
    if (saving()) return;
    setSaving(true);
    try {
      const item = await getItem(props.item.id, AbortSignal.timeout(15_000));
      setSaved({ notes: item.notes, version: item.version });
      cancelEdit();
    } catch {
      setError("Could not load the latest notes. Your edits are still here.");
    } finally {
      setSaving(false);
    }
  }

  useBeforeLeave((event) => {
    if (event.defaultPrevented || (!dirty() && !saving())) return;
    event.preventDefault();
    if (!leave() && !saving()) setLeave(event);
  });

  createEffect(dirty, (hasEdits) => {
    if (!hasEdits) return;
    function warn(event: BeforeUnloadEvent) {
      event.preventDefault();
      event.returnValue = true;
    }
    window.addEventListener("beforeunload", warn);
    return () => window.removeEventListener("beforeunload", warn);
  });

  createEffect(leave, (request) => {
    if (request && !dialog.open) dialog.showModal();
    if (!request && dialog.open) dialog.close();
  });

  async function saveAndLeave() {
    const request = leave();
    if (!request || saving()) return;
    const success = await saveNotes();
    setLeave(undefined);
    if (success) request.retry(true);
  }

  return (
    <>
      <button
        type="button"
        class={controls.secondary}
        aria-expanded={visible() ? "true" : "false"}
        aria-controls="reader-notes"
        disabled={dirty() || saving()}
        onClick={() => setVisible((value) => !value)}
      >
        Notes
      </button>
      <section
        id="reader-notes"
        class={styles.notes}
        aria-label="Article notes"
        hidden={!visible()}
      >
        <Show
          when={draft() !== undefined}
          fallback={
            <>
              <p class={styles.noteText}>{current().notes || "No notes yet."}</p>
              <button
                type="button"
                class={controls.secondary}
                onClick={() => {
                  setConfirmation("");
                  setDraft(current().notes ?? "");
                }}
              >
                Edit Notes
              </button>
            </>
          }
        >
          <form
            onSubmit={(event) => {
              event.preventDefault();
              void saveNotes();
            }}
          >
            <label for="reader-notes-input">Notes</label>
            <textarea
              id="reader-notes-input"
              rows={4}
              value={draft() ?? ""}
              readonly={saving()}
              onInput={(event) => setDraft(event.currentTarget.value)}
            />
            <div class={styles.actions}>
              <button
                type="submit"
                class={controls.primary}
                disabled={!dirty() || saving() || conflict()}
              >
                {saving() ? "Working…" : "Save Notes"}
              </button>
              <button
                type="button"
                class={controls.secondary}
                disabled={saving()}
                onClick={cancelEdit}
              >
                Cancel
              </button>
              <Show when={conflict()}>
                <button
                  type="button"
                  class={controls.secondary}
                  disabled={saving()}
                  onClick={() => void loadLatest()}
                >
                  Discard Edits and Load Latest
                </button>
              </Show>
            </div>
          </form>
        </Show>
        <Show when={error()}>{(message) => <p role="alert">{message()}</p>}</Show>
        <Show when={confirmation()}>{(message) => <p role="status">{message()}</p>}</Show>
      </section>
      <dialog
        ref={dialog}
        class={styles.notesDialog}
        aria-labelledby="reader-leave-title"
        onCancel={(event) => {
          event.preventDefault();
          if (!saving()) setLeave(undefined);
        }}
      >
        <h2 id="reader-leave-title">Leave with unsaved notes?</h2>
        <div class={styles.actions}>
          <button
            type="button"
            class={controls.secondary}
            autofocus
            disabled={saving()}
            onClick={() => setLeave(undefined)}
          >
            Stay
          </button>
          <button
            type="button"
            class={controls.secondary}
            disabled={saving()}
            onClick={() => leave()?.retry(true)}
          >
            Discard &amp; Leave
          </button>
          <button
            type="button"
            class={controls.primary}
            disabled={saving() || conflict()}
            onClick={() => void saveAndLeave()}
          >
            {saving() ? "Saving Notes…" : "Save Notes & Leave"}
          </button>
        </div>
      </dialog>
    </>
  );
}
