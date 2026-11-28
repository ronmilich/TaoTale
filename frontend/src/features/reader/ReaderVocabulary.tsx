import { For, Show, createEffect, createMemo, createSignal, onSettled } from "solid-js";
import {
  getItemVocabulary,
  patchVocabulary,
  syncItemVocabulary,
  type Item,
  type ItemVocabulary,
  type ItemVocabularyEntry,
} from "../../api/client";
import controls from "../../components/Controls.module.css";
import { usePreferences } from "../../preferences/Preferences";
import { vocabularyDisplays, type VocabularyDisplay } from "../../preferences/storage";
import { useTopics } from "../topics/topics";
import { WordStatus, WordTopics } from "../vocabulary/WordBadges";
import styles from "../vocabulary/Vocabulary.module.css";
import { otherStatus, statusLabel, useVocabularyChanges } from "../vocabulary/words";
import CollectVocabulary from "./CollectVocabulary";
import readerStyles from "./ReaderArticle.module.css";
import { selectionError, type TranscriptSelection } from "./selection";

export default function ReaderVocabulary(props: {
  item: Item;
  selection?: TranscriptSelection;
  highlights: (view: ItemVocabulary | undefined, display: VocabularyDisplay) => void;
}) {
  const topics = useTopics();
  const preferences = usePreferences();
  const [collection, setCollection] = createSignal<TranscriptSelection>();
  const [confirmation, setConfirmation] = createSignal("");
  const [open, setOpen] = createSignal(false);
  onSettled(() => {
    setOpen(preferences.reader().showVocabularyDetails);
  });
  const [refresh, setRefresh] = createSignal(0);
  const [view, setView] = createSignal<ItemVocabulary>();
  const [loading, setLoading] = createSignal(false);
  const [failed, setFailed] = createSignal(false);
  const [pending, setPending] = createSignal<string>();
  const [syncing, setSyncing] = createSignal(false);
  const [statusError, setStatusError] = createSignal("");
  const display = () => preferences.reader().vocabularyDisplay;
  const broadcast = useVocabularyChanges((language) => {
    if (language === props.item.languageCode) setRefresh((number) => number + 1);
  });
  const invalid = () =>
    props.selection ? selectionError(props.selection.text, props.item.languageCode) : "";
  const supportsMatching = () => props.item.automaticVocabularyMatching === true;
  const shouldLoad = createMemo(() => open() || supportsMatching());
  const rows = createMemo(() =>
    view()?.items.filter(
      (entry) => entry.reference.manualLinked || (supportsMatching() && entry.reference.autoLinked),
    ),
  );
  const visible = createMemo(() => {
    const filter = display();
    return rows()?.filter(({ status }) => filter === "both" || status.toLowerCase() === filter);
  });

  createEffect(
    () => [shouldLoad(), props.item.id, refresh()] as const,
    ([enabled, itemId]) => {
      if (!enabled) return;
      const controller = new AbortController();
      setLoading(true);
      setFailed(false);
      getItemVocabulary(
        itemId,
        AbortSignal.any([controller.signal, AbortSignal.timeout(15_000)]),
      ).then(
        (projection) => {
          if (controller.signal.aborted) return;
          setView(projection);
          setLoading(false);
        },
        () => {
          if (controller.signal.aborted) return;
          setFailed(true);
          setLoading(false);
        },
      );
      return () => controller.abort();
    },
  );

  createEffect(
    () => [view(), display()] as const,
    ([projection, filter]) => {
      props.highlights(projection, filter);
    },
  );

  function changed() {
    setRefresh((number) => number + 1);
    broadcast(props.item.languageCode);
  }

  async function toggleStatus(entry: ItemVocabularyEntry) {
    if (pending()) return;
    setPending(entry.id);
    setStatusError("");
    setConfirmation("");
    try {
      const saved = await patchVocabulary(entry.id, { status: otherStatus(entry.status) });
      setView(
        (current) =>
          current && {
            ...current,
            items: current.items.map((row) => (row.id === entry.id ? { ...row, ...saved } : row)),
          },
      );
      setConfirmation(`${saved.word} is now ${statusLabel[saved.status]}.`);
      changed();
    } catch {
      setStatusError(
        `Could not confirm the status change. Last confirmed status is ${statusLabel[entry.status]}.`,
      );
    } finally {
      setPending(undefined);
    }
  }

  async function sync() {
    if (syncing() || pending()) return;
    setSyncing(true);
    setStatusError("");
    setConfirmation("");
    try {
      const { matched } = await syncItemVocabulary(props.item.id);
      setConfirmation(
        matched
          ? `${matched} vocabulary ${matched === 1 ? "match" : "matches"} in this article.`
          : "No vocabulary matches in this article.",
      );
      changed();
    } catch {
      setStatusError("Could not confirm vocabulary sync. Try again.");
    } finally {
      setSyncing(false);
    }
  }

  return (
    <>
      <div class={[readerStyles.readerActions, styles.collection]}>
        <button
          type="button"
          class={controls.primary}
          disabled={!props.selection || Boolean(invalid())}
          onPointerDown={(event) => {
            if (event.button === 0 && props.selection) event.preventDefault();
          }}
          onClick={() => {
            if (!props.selection || invalid()) return;
            setConfirmation("");
            setCollection(props.selection);
          }}
        >
          Add to Vocabulary
        </button>
        <Show when={invalid()}>
          <p role="status">{invalid()}</p>
        </Show>
      </div>
      <Show when={confirmation()}>
        <p role="status">{confirmation()}</p>
      </Show>
      <Show when={supportsMatching() && view()?.annotations?.length}>
        <div
          class={readerStyles.vocabularyLegend}
          role="group"
          aria-label="Vocabulary highlight legend"
        >
          <span data-vocabulary-status="LEARNING">Learning</span>
          <span data-vocabulary-status="ACQUIRED">Acquired</span>
        </div>
      </Show>
      <details
        class={styles.panel}
        open={open()}
        onToggle={(event) => setOpen(event.currentTarget.open)}
      >
        <summary>Article vocabulary</summary>
        <div class={styles.header}>
          <div class={styles.entryActions}>
            <label for="reader-vocabulary-display">Show</label>
            <select
              id="reader-vocabulary-display"
              value={display()}
              onChange={(event) => {
                const selected = vocabularyDisplays.find(
                  (value) => value === event.currentTarget.value,
                );
                if (selected) {
                  preferences.changeReader({ vocabularyDisplay: selected });
                }
              }}
            >
              <option value="both" selected={display() === "both"}>
                Both
              </option>
              <option value="learning" selected={display() === "learning"}>
                Learning
              </option>
              <option value="acquired" selected={display() === "acquired"}>
                Acquired
              </option>
            </select>
          </div>
          <div class={styles.entryActions}>
            <Show when={supportsMatching()}>
              <button
                type="button"
                class={controls.secondary}
                disabled={syncing() || Boolean(pending())}
                onClick={() => void sync()}
              >
                {syncing() ? "Syncing…" : "Sync vocabulary"}
              </button>
            </Show>
            <a href={`/vocabulary?languageCode=${props.item.languageCode}`}>Open Vocabulary</a>
          </div>
        </div>
        <Show when={open()}>
          <Show when={loading() && !rows()}>
            <p role="status">Loading linked words…</p>
          </Show>
          <Show when={failed()}>
            <div class={styles.entryActions}>
              <p role="alert" class={styles.error}>
                Could not load linked words.
              </p>
              <button
                type="button"
                class={controls.secondary}
                onClick={() => setRefresh((number) => number + 1)}
              >
                Retry words
              </button>
            </div>
          </Show>
          <Show when={topics.failed()}>
            <div class={styles.entryActions}>
              <p role="alert" class={styles.error}>
                Could not load Topics.
              </p>
              <button type="button" class={controls.secondary} onClick={() => void topics.reload()}>
                Retry topics
              </button>
            </div>
          </Show>
          <Show when={statusError()}>
            <p role="alert" class={styles.error}>
              {statusError()}
            </p>
          </Show>
          <Show when={!visible()?.length && rows() && !loading()}>
            <p class={styles.muted}>
              {rows()?.length
                ? "No linked words match this display."
                : "No words linked to this article yet."}
            </p>
          </Show>
          <ul class={styles.panelRows} aria-label="Linked words">
            <For each={visible()} keyed={(entry) => entry.id}>
              {(entry) => (
                <li>
                  <div class={styles.entryActions}>
                    <strong class={styles.word} lang={props.item.languageCode}>
                      {entry().word}
                    </strong>
                    <WordStatus status={entry().status} />
                    <Show when={entry().reference.autoLinked && !entry().reference.manualLinked}>
                      <small class={styles.muted}>Matched in article</small>
                    </Show>
                  </div>
                  <Show when={entry().reading}>
                    <div>
                      {entry().reading}
                      {entry().readingSystem ? ` · ${entry().readingSystem}` : ""}
                    </div>
                  </Show>
                  <Show when={entry().meaning}>
                    <div>{entry().meaning}</div>
                  </Show>
                  <WordTopics ids={entry().topicIds} topics={topics.items()} />
                  <div>
                    <button
                      type="button"
                      class={controls.secondary}
                      disabled={syncing() || Boolean(pending())}
                      aria-busy={pending() === entry().id ? "true" : undefined}
                      aria-label={`Mark ${entry().word} as ${statusLabel[otherStatus(entry().status)]}`}
                      onClick={() => void toggleStatus(entry())}
                    >
                      {pending() === entry().id
                        ? "Saving…"
                        : `Mark ${statusLabel[otherStatus(entry().status)]}`}
                    </button>
                  </div>
                </li>
              )}
            </For>
          </ul>
        </Show>
      </details>
      <Show when={collection()} keyed>
        {(snapshot) => (
          <CollectVocabulary
            item={props.item}
            word={snapshot.text}
            topics={topics}
            close={() => setCollection(undefined)}
            saved={(word) => {
              setCollection(undefined);
              setConfirmation(`“${word}” linked to this article.`);
              changed();
            }}
          />
        )}
      </Show>
    </>
  );
}
