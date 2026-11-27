import { Show, createSignal } from "solid-js";
import controls from "./Controls.module.css";
import styles from "./AudioPlayer.module.css";

export default function AudioPlayer(props: { src: string; download: string }) {
  const [failed, setFailed] = createSignal(false);
  return (
    <div class={styles.player}>
      <audio
        controls
        preload="metadata"
        aria-label="Audio preview"
        src={props.src}
        onError={() => setFailed(true)}
      />
      <Show
        when={failed()}
        fallback={
          <a class={controls.secondary} href={props.download} download>
            Download Audio
          </a>
        }
      >
        <p role="status">Audio unavailable.</p>
      </Show>
    </div>
  );
}
