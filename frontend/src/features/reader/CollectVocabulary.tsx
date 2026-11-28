import { Show, createEffect, createSignal, onSettled } from "solid-js";
import {
  ApiError,
  createVocabulary,
  linkVocabularyReference,
  lookupVocabulary,
  type Item,
  type VocabularyEntry,
} from "../../api/client";
import controls from "../../components/Controls.module.css";
import type { useTopics } from "../topics/topics";
import WordFields from "../vocabulary/WordFields";
import { WordStatus } from "../vocabulary/WordBadges";
import styles from "../vocabulary/Vocabulary.module.css";
import { wordError, wordValues, type WordValues } from "../vocabulary/words";

export default function CollectVocabulary(props: {
  item: Item;
  word: string;
  topics: ReturnType<typeof useTopics>;
  close: () => void;
  saved: (word: string) => void;
}) {
  let dialog!: HTMLDialogElement;
  const [values, setValues] = createSignal<WordValues>(() => ({
    ...wordValues(),
    word: props.word,
  }));
  const [entry, setEntry] = createSignal<VocabularyEntry | null>();
  const [lookupFailed, setLookupFailed] = createSignal(false);
  const [attempt, setAttempt] = createSignal(0);
  const [busy, setBusy] = createSignal(false);
  const [error, setError] = createSignal("");
  onSettled(() => {
    dialog.showModal();
    return () => {
      if (dialog.open) dialog.close();
    };
  });
  createEffect(attempt, () => {
    const controller = new AbortController();
    setEntry(undefined);
    setLookupFailed(false);
    lookupVocabulary(
      props.item.languageCode,
      props.word,
      AbortSignal.any([controller.signal, AbortSignal.timeout(15_000)]),
    ).then(
      ({ entry }) => {
        if (!controller.signal.aborted) setEntry(entry);
      },
      () => {
        if (!controller.signal.aborted) setLookupFailed(true);
      },
    );
    return () => controller.abort();
  });

  async function save() {
    const existing = entry();
    if (busy() || existing === undefined) return;
    const input = values();
    const invalid = existing ? "" : wordError(input);
    setError(invalid);
    if (invalid) return;
    setBusy(true);
    try {
      if (existing) {
        await linkVocabularyReference(existing.id, props.item.id);
        props.saved(existing.word);
      } else {
        const saved = await createVocabulary({
          ...input,
          itemId: props.item.id,
          languageCode: props.item.languageCode,
          reading: input.reading || null,
          readingSystem: input.readingSystem || null,
          meaning: input.meaning || null,
          note: input.note || null,
        });
        props.saved(saved.word);
      }
    } catch (cause) {
      if (!existing && cause instanceof ApiError && cause.code === "DUPLICATE_VOCABULARY") {
        setAttempt((number) => number + 1);
      } else {
        setError(
          existing
            ? "Could not link the word. Try again."
            : "Could not save the word. Your edits are still here.",
        );
      }
    } finally {
      setBusy(false);
    }
  }

  return (
    <dialog
      ref={dialog}
      class={styles.dialog}
      aria-labelledby="reader-collect-title"
      onCancel={(event) => {
        event.preventDefault();
        if (!busy()) props.close();
      }}
    >
      <h2 id="reader-collect-title">Add to Vocabulary</h2>
      <form
        class={styles.form}
        novalidate
        onSubmit={(event) => {
          event.preventDefault();
          void save();
        }}
      >
        <Show
          when={entry() !== undefined}
          fallback={
            <Show when={lookupFailed()} fallback={<p role="status">Checking vocabulary…</p>}>
              <p role="alert" class={styles.error}>
                Could not check vocabulary.
              </p>
              <button
                type="button"
                class={controls.secondary}
                onClick={() => setAttempt((number) => number + 1)}
              >
                Retry lookup
              </button>
            </Show>
          }
        >
          <Show
            when={entry()}
            fallback={
              <WordFields
                language={props.item.languageCode}
                values={values()}
                topics={props.topics}
                readonlyWord
                disabled={busy()}
                change={(key, value) => setValues((current) => ({ ...current, [key]: value }))}
              />
            }
          >
            {(existing) => (
              <div>
                <p>
                  <strong>Already in Vocabulary</strong>
                </p>
                <p class={styles.word} lang={props.item.languageCode}>
                  {existing().word}
                </p>
                <Show when={existing().reading}>
                  <p>{existing().reading}</p>
                </Show>
                <Show when={existing().meaning}>
                  <p>{existing().meaning}</p>
                </Show>
                <WordStatus status={existing().status} />
              </div>
            )}
          </Show>
        </Show>
        <Show when={error()}>
          <p role="alert" class={styles.error}>
            {error()}
          </p>
        </Show>
        <div class={styles.controls}>
          <button
            type="button"
            class={controls.secondary}
            disabled={busy()}
            onClick={() => props.close()}
          >
            Cancel
          </button>
          <Show when={entry() !== undefined}>
            <button type="submit" class={controls.primary} disabled={busy()}>
              {busy() ? "Saving…" : entry() ? "Link to Article" : "Save Word"}
            </button>
          </Show>
        </div>
      </form>
    </dialog>
  );
}
