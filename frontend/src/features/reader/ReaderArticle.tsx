import { Title } from "@solidjs/meta";
import { For, Show, createMemo, createSignal } from "solid-js";
import { draftAudioURLs, type Item } from "../../api/client";
import controls from "../../components/Controls.module.css";
import type { Language } from "../../preferences/storage";
import styles from "./ReaderArticle.module.css";
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
  const runs = createMemo(() => transcriptRuns(props.source, props.item?.transcript));
  const recording = createMemo(() => {
    const item = props.item;
    return item?.audioStatus === "READY" && item.audioAssetId && runs()
      ? draftAudioURLs(item.id, item.audioAssetId).src
      : undefined;
  });

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
        </header>
        <div class={styles.transcript} lang={props.language} role="region" aria-label="Transcript">
          <Show when={runs()} fallback={props.source}>
            {(parts) => (
              <For each={parts()}>
                {(run) =>
                  run.segmentIndex === undefined ? (
                    run.text
                  ) : (
                    <span data-segment-index={run.segmentIndex}>{run.text}</span>
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
              {(src) => (
                <audio
                  controls
                  preload="metadata"
                  aria-label="Article recording"
                  src={src}
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
