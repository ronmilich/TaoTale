import { Title } from "@solidjs/meta";
import { For, Show, createEffect, createMemo, createSignal, onSettled } from "solid-js";
import {
  draftAudioURLs,
  type Item,
  type ItemVocabulary,
  type VocabularyAnnotation,
} from "../../api/client";
import controls from "../../components/Controls.module.css";
import type { Language, VocabularyDisplay } from "../../preferences/storage";
import { annotateRuns, sameAnnotations, verifiedAnnotations } from "./annotations";
import styles from "./ReaderArticle.module.css";
import ReaderActions from "./ReaderActions";
import ReaderPlayer, { type Recording, type Seek } from "./ReaderPlayer";
import ReaderVocabulary from "./ReaderVocabulary";
import { playbackResumeKey } from "./playback";
import { readTranscriptSelection, type TranscriptSelection } from "./selection";
import { transcriptRuns } from "./transcript";

export default function ReaderArticle(props: {
  title: string;
  description?: string | null;
  language: Language;
  source: string;
  item?: Item;
  retry?: () => void;
  loading?: boolean;
}) {
  const [mediaFailed, setMediaFailed] = createSignal(false);
  const [active, setActive] = createSignal(-1);
  const [seek, setSeek] = createSignal<Seek>();
  const [selection, setSelection] = createSignal<TranscriptSelection>();
  const [hasSelection, setHasSelection] = createSignal(false);
  const [selecting, setSelecting] = createSignal(false);
  const [vocabulary, setVocabulary] = createSignal<{
    view?: ItemVocabulary;
    display: VocabularyDisplay;
  }>();
  let transcript!: HTMLDivElement;
  const runs = createMemo(() => transcriptRuns(props.source, props.item?.transcript));
  const textRuns = createMemo(() => runs() ?? [{ text: props.source }]);
  const view = createMemo(() => vocabulary()?.view);
  const proposed = createMemo(() => verifiedAnnotations(props.source, view()));
  const [annotations, setAnnotations] = createSignal<VocabularyAnnotation[]>([], {
    equals: sameAnnotations,
  });
  const parts = createMemo(() => annotateRuns(textRuns(), annotations()));
  const decorated = createMemo(() => {
    const current = vocabulary();
    const statuses = new Map<string, "LEARNING" | "ACQUIRED">();
    if (current?.view?.sourceText === props.source) {
      for (const { id, status } of current.view.items) {
        if (current.display === "both" || status.toLowerCase() === current.display) {
          statuses.set(id, status);
        }
      }
    }
    return statuses;
  });

  function transcriptSelection(selection: Selection | null) {
    return Boolean(
      transcript &&
      selection &&
      !selection.isCollapsed &&
      selection.rangeCount &&
      selection.getRangeAt(0).intersectsNode(transcript),
    );
  }
  createEffect(
    () => [proposed(), hasSelection() || selecting()] as const,
    ([next, held]) => {
      if (!held && !transcriptSelection(window.getSelection())) setAnnotations(next);
    },
  );
  const recording = createMemo<Recording | undefined>(() => {
    const item = props.item;
    if (item?.audioStatus !== "READY" || !item.audioAssetId || !item.transcript || !runs()) return;
    return {
      ...draftAudioURLs(item.id, item.audioAssetId),
      transcript: item.transcript,
      resumeKey: playbackResumeKey(item.id, item.audioAssetId),
    };
  });
  let origin: { x: number; y: number } | undefined;
  let pointerSegment: HTMLElement | undefined;
  let dragged = false;
  let pendingClick: number | undefined;
  function cancelClick() {
    window.clearTimeout(pendingClick);
    pendingClick = undefined;
  }
  function releasePointer() {
    if (pointerSegment && seek()) pointerSegment.tabIndex = 0;
    pointerSegment = undefined;
    origin = undefined;
    setSelecting(false);
  }
  onSettled(() => {
    const listeners = new AbortController();
    const options = { signal: listeners.signal };
    function cancel() {
      cancelClick();
      dragged = true;
      releasePointer();
    }
    window.addEventListener("pointerup", releasePointer, options);
    window.addEventListener("pointercancel", cancel, options);
    window.addEventListener("blur", cancel, options);
    if (props.item) {
      document.addEventListener(
        "selectionchange",
        () => {
          const native = window.getSelection();
          setSelection(readTranscriptSelection(transcript, props.source, native));
          setHasSelection(transcriptSelection(native));
        },
        options,
      );
    }
    return () => {
      cancelClick();
      listeners.abort();
    };
  });
  function seekSegment(index: number) {
    const segment = recording()?.transcript.segments[index];
    if (segment && !mediaFailed() && !window.getSelection()?.toString()) seek()?.(segment.startMs);
  }

  return (
    <article class={[styles.shell, { [styles.withPlayer]: Boolean(props.item) }]}>
      <Title>{props.title} · TaoTale</Title>
      <a class={styles.back} href={`/library?languageCode=${props.language}`}>
        Back to Library
      </a>
      <div class={styles.paper}>
        <header class={styles.intro}>
          <h1>{props.title}</h1>
          <Show when={props.description}>
            <p>{props.description}</p>
          </Show>
          <Show when={props.item}>
            {(item) => (
              <>
                <ReaderActions item={item()} download={seek() && recording()?.download} />
                <ReaderVocabulary
                  item={item()}
                  selection={selection()}
                  highlights={(view, display) => setVocabulary({ view, display })}
                />
              </>
            )}
          </Show>
        </header>
        <div
          ref={transcript}
          class={styles.transcript}
          lang={props.language}
          role="region"
          aria-label="Transcript"
          onPointerDown={(event) => {
            cancelClick();
            releasePointer();
            setSelecting(true);
            origin = { x: event.clientX, y: event.clientY };
            // Chromium starts a focus gesture on tabbable spans, suppressing native text selection.
            pointerSegment =
              (event.target as Element).closest<HTMLElement>("[data-segment-index][tabindex]") ??
              undefined;
            pointerSegment?.removeAttribute("tabindex");
            dragged =
              !event.isPrimary ||
              event.button !== 0 ||
              event.shiftKey ||
              Boolean(window.getSelection()?.toString());
          }}
          onPointerMove={(event) => {
            if (origin && Math.hypot(event.clientX - origin.x, event.clientY - origin.y) > 4)
              dragged = true;
          }}
          onPointerLeave={() => {
            if (origin) dragged = true;
          }}
        >
          <For each={textRuns()}>
            {(run, i) => {
              const text = (
                <For each={parts()[i()]}>
                  {(part) =>
                    part.vocabularyId ? (
                      <span
                        data-vocabulary-id={part.vocabularyId}
                        data-vocabulary-status={decorated().get(part.vocabularyId)}
                      >
                        {part.text}
                      </span>
                    ) : (
                      part.text
                    )
                  }
                </For>
              );
              return run.segmentIndex === undefined ? (
                text
              ) : (
                <span
                  data-segment-index={run.segmentIndex}
                  data-active={active() === run.segmentIndex ? "true" : undefined}
                  role={seek() ? "button" : undefined}
                  tabindex={seek() ? 0 : undefined}
                  onClick={(event) => {
                    if (event.shiftKey || event.detail > 1) return;
                    if (event.detail === 0) seekSegment(run.segmentIndex!);
                    else if (!dragged) {
                      // Give a second press time to begin word selection within the seek budget.
                      pendingClick = window.setTimeout(() => seekSegment(run.segmentIndex!), 90);
                    }
                  }}
                  onKeyDown={(event) => {
                    if (seek() && (event.key === "Enter" || event.key === " ")) {
                      event.preventDefault();
                      seekSegment(run.segmentIndex!);
                    }
                  }}
                >
                  {text}
                </span>
              );
            }}
          </For>
        </div>
      </div>
      <Show when={props.item}>
        <section class={styles.dock} aria-label="Article audio">
          <div class={styles.player}>
            <Show
              when={!mediaFailed() && recording()}
              keyed
              fallback={
                <>
                  <p role="status">Audio unavailable. You can keep reading.</p>
                  <button
                    type="button"
                    class={controls.secondary}
                    disabled={props.loading}
                    onClick={() => props.retry?.()}
                  >
                    {props.loading ? "Retrying…" : "Retry"}
                  </button>
                </>
              }
            >
              {(asset) => (
                <ReaderPlayer
                  recording={asset}
                  onSegment={setActive}
                  onSeekable={(action) => setSeek(() => action)}
                  onError={() => setMediaFailed(true)}
                />
              )}
            </Show>
          </div>
        </section>
      </Show>
    </article>
  );
}
