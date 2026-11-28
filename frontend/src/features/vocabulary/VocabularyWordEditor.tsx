import { For, Show, createEffect, createMemo, createSignal, onSettled } from "solid-js";
import {
  ApiError,
  createVocabulary,
  deleteVocabulary,
  getVocabulary,
  patchVocabulary,
  type VocabularyDetail,
} from "../../api/client";
import controls from "../../components/Controls.module.css";
import type { Language } from "../../preferences/storage";
import type { useTopics } from "../topics/topics";
import styles from "./Vocabulary.module.css";
import { WordStatus } from "./WordBadges";
import WordFields from "./WordFields";
import { otherStatus, statusLabel, wordError, wordPatch, wordValues } from "./words";

type Mutation = "save" | "status" | "delete";

export default function VocabularyWordEditor(props: {
  language: Language;
  id?: string;
  refresh: number;
  disabled: boolean;
  topics: ReturnType<typeof useTopics>;
  close: () => void;
  open: (id: string) => void;
  busyChanged: (busy: boolean) => void;
  changed: (message: string) => void;
}) {
  const [entry, setEntry] = createSignal<VocabularyDetail>();
  const [values, setValues] = createSignal(wordValues());
  const [baseline, setBaseline] = createSignal(wordValues());
  const [loading, setLoading] = createSignal(false);
  const [loadError, setLoadError] = createSignal("");
  const [missing, setMissing] = createSignal(false);
  const [attempt, setAttempt] = createSignal(0);
  const [pending, setPending] = createSignal<Mutation>();
  const [error, setError] = createSignal("");
  const [duplicate, setDuplicate] = createSignal<string>();
  const patch = createMemo(() => wordPatch(values(), baseline()));
  const dirty = () => Object.keys(patch()).length > 0;
  const disabled = () => props.disabled || Boolean(pending()) || missing();
  let readRequest: AbortController | undefined;
  let heading!: HTMLHeadingElement;
  let mounted = true;
  onSettled(() => {
    heading.focus();
    return () => {
      mounted = false;
    };
  });

  function confirm(saved: VocabularyDetail) {
    const fields = wordValues(saved);
    setEntry(saved);
    setValues(fields);
    setBaseline(fields);
  }

  createEffect(
    () => [props.id, props.refresh, attempt()] as const,
    ([id]) => {
      if (!id) return;
      const controller = new AbortController();
      readRequest = controller;
      setLoading(true);
      setLoadError("");
      getVocabulary(id, AbortSignal.any([controller.signal, AbortSignal.timeout(15_000)])).then(
        (word) => {
          if (controller.signal.aborted) return;
          setMissing(false);
          if (dirty()) setEntry(word);
          else confirm(word);
          setLoading(false);
        },
        (cause) => {
          if (controller.signal.aborted) return;
          const gone = cause instanceof ApiError && cause.status === 404;
          setMissing(gone);
          setLoadError(
            gone ? "This entry is no longer available." : "Could not load the vocabulary entry.",
          );
          setLoading(false);
        },
      );
      return () => controller.abort();
    },
  );

  async function mutate<T>(kind: Mutation, work: () => Promise<T>) {
    if (disabled()) return;
    readRequest?.abort();
    setLoading(false);
    setPending(kind);
    props.busyChanged(true);
    setDuplicate(undefined);
    setError("");
    try {
      return await work();
    } catch (cause) {
      if (cause instanceof ApiError && cause.code === "DUPLICATE_VOCABULARY") {
        setDuplicate(
          typeof cause.details?.existingId === "string" ? cause.details.existingId : undefined,
        );
        setError("This word is already in this language’s vocabulary.");
      } else if (kind === "status") {
        const status = entry()?.status;
        setError(
          status
            ? `Could not confirm the status change. Last confirmed status is ${statusLabel[status]}.`
            : "Could not confirm the status change.",
        );
      } else {
        setError(
          kind === "save"
            ? "Could not save the word. Your edits are still here."
            : "Could not delete the word.",
        );
      }
    } finally {
      setPending(undefined);
      props.busyChanged(false);
    }
  }

  async function save() {
    const input = values();
    const invalid = wordError(input);
    if (invalid) {
      setError(invalid);
      return;
    }
    const current = entry();
    if (current && !dirty()) return;
    const changes = patch();
    const languageCode = props.language;
    const saved = await mutate("save", () =>
      current
        ? patchVocabulary(current.id, changes)
        : createVocabulary({ ...changes, languageCode, word: input.word.trim() }),
    );
    if (!saved) return;
    if (mounted) confirm(saved);
    props.changed(`“${saved.word}” saved.`);
    if (!current && mounted) props.open(saved.id);
  }

  async function toggleStatus(current: VocabularyDetail) {
    const saved = await mutate("status", () =>
      patchVocabulary(current.id, { status: otherStatus(current.status) }),
    );
    if (!saved) return;
    setEntry(saved);
    props.changed(`“${saved.word}” is now ${statusLabel[saved.status]}.`);
  }

  async function remove(current: VocabularyDetail) {
    if (disabled() || !window.confirm(`Delete “${current.word}” and its article links?`)) return;
    const removed = await mutate("delete", async () => {
      await deleteVocabulary(current.id);
      return true;
    });
    if (!removed) return;
    props.changed(`“${current.word}” deleted.`);
    if (mounted) props.close();
  }

  return (
    <section class={styles.detail} aria-labelledby="vocabulary-details">
      <div class={styles.header}>
        <h2 id="vocabulary-details" ref={heading} tabindex={-1}>
          {props.id ? "Word details" : "Add word"}
        </h2>
        <button
          type="button"
          class={controls.secondary}
          disabled={Boolean(pending())}
          onClick={() => props.close()}
        >
          Close details
        </button>
      </div>
      <Show when={loading() && !entry()}>
        <p role="status">Loading word…</p>
      </Show>
      <Show when={loadError()}>
        <div class={styles.controls}>
          <p role="alert" class={styles.error}>
            {loadError()}
          </p>
          <button
            type="button"
            class={controls.secondary}
            onClick={() => setAttempt((number) => number + 1)}
          >
            Retry word
          </button>
        </div>
      </Show>
      <Show when={entry()}>
        {(current) => (
          <div class={styles.entryActions}>
            <WordStatus status={current().status} />
            <button
              type="button"
              class={controls.secondary}
              disabled={disabled()}
              onClick={() => void toggleStatus(current())}
            >
              {pending() === "status"
                ? "Saving status…"
                : `Mark as ${statusLabel[otherStatus(current().status)]}`}
            </button>
          </div>
        )}
      </Show>
      <Show when={!props.id || entry()}>
        <form
          class={styles.form}
          novalidate
          onSubmit={(event) => {
            event.preventDefault();
            void save();
          }}
        >
          <WordFields
            language={props.language}
            values={values()}
            topics={props.topics}
            disabled={disabled()}
            change={(key, value) => setValues((current) => ({ ...current, [key]: value }))}
          />
          <div class={styles.controls}>
            <button
              type="submit"
              class={controls.primary}
              disabled={disabled() || Boolean(props.id && !dirty())}
            >
              {pending() === "save" ? "Saving word…" : "Save Word"}
            </button>
            <Show when={entry()}>
              {(current) => (
                <button
                  type="button"
                  class={controls.secondary}
                  disabled={disabled()}
                  onClick={() => void remove(current())}
                >
                  {pending() === "delete" ? "Deleting word…" : "Delete Word"}
                </button>
              )}
            </Show>
          </div>
        </form>
      </Show>
      <Show when={error()}>
        <p role="alert" class={styles.error}>
          {error()}
        </p>
      </Show>
      <Show when={duplicate()}>
        {(id) => (
          <button type="button" class={controls.secondary} onClick={() => props.open(id())}>
            Open existing entry
          </button>
        )}
      </Show>
      <Show when={entry()}>
        {(current) => (
          <section class={styles.references} aria-label="Article references">
            <h2>Article references</h2>
            <Show
              when={current().references.length}
              fallback={<p class={styles.muted}>No articles linked to this word.</p>}
            >
              <ul>
                <For each={current().references}>
                  {(reference) => (
                    <li>
                      <a href={`/items/${encodeURIComponent(reference.itemId)}`}>
                        {reference.title || "Untitled article"}
                      </a>
                    </li>
                  )}
                </For>
              </ul>
            </Show>
          </section>
        )}
      </Show>
    </section>
  );
}
