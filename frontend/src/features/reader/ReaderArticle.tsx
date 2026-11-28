import { Title } from "@solidjs/meta";
import { For, Show, createMemo, createSignal, onSettled } from "solid-js";
import { draftAudioURLs, type Item } from "../../api/client";
import controls from "../../components/Controls.module.css";
import type { Language } from "../../preferences/storage";
import styles from "./ReaderArticle.module.css";
import ReaderActions from "./ReaderActions";
import ReaderPlayer, { type Recording, type Seek } from "./ReaderPlayer";
import { playbackResumeKey } from "./playback";
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
  const runs = createMemo(() => transcriptRuns(props.source, props.item?.transcript));
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
            {(item) => <ReaderActions item={item()} download={seek() && recording()?.download} />}
          </Show>
        </header>
        <div
          class={styles.transcript}
          lang={props.language}
          role="region"
          aria-label="Transcript"
          onPointerDown={(event) => {
            cancelClick();
            releasePointer();
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
          <Show when={runs()} fallback={props.source}>
            {(parts) => (
              <For each={parts()}>
                {(run) =>
                  run.segmentIndex === undefined ? (
                    run.text
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
                          pendingClick = window.setTimeout(
                            () => seekSegment(run.segmentIndex!),
                            90,
                          );
                        }
                      }}
                      onKeyDown={(event) => {
                        if (seek() && (event.key === "Enter" || event.key === " ")) {
                          event.preventDefault();
                          seekSegment(run.segmentIndex!);
                        }
                      }}
                    >
                      {run.text}
                    </span>
                  )
                }
              </For>
            )}
          </Show>
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
