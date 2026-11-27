import { For, Show, createSignal, onSettled } from "solid-js";
import controls from "../../components/Controls.module.css";
import {
  playbackSpeeds,
  readPreference,
  savePreference,
  validatePreference,
} from "../../preferences/storage";
import { bindPlayback, formatTime, type Transcript } from "./playback";
import styles from "./ReaderArticle.module.css";

export type Recording = {
  src: string;
  download: string;
  resumeKey: string;
  transcript: Transcript;
};
export type Seek = (ms: number) => void;

export default function ReaderPlayer(props: {
  recording: Recording;
  onSegment(index: number): void;
  onSeekable(seek?: Seek): void;
  onError(): void;
}) {
  let audio!: HTMLAudioElement;
  let binding: ReturnType<typeof bindPlayback> | undefined;
  const [ready, setReady] = createSignal(false);
  const [playing, setPlaying] = createSignal(false);
  const [buffering, setBuffering] = createSignal(false);
  const [position, setPosition] = createSignal(0);
  const [speed, setSpeed] = createSignal(readPreference("playbackSpeed"));
  const duration = () => props.recording.transcript.durationMs;
  const seek = (ms: number) => binding?.seek(ms);

  onSettled(() => {
    audio.defaultPlaybackRate = audio.playbackRate = Number(speed());
    binding = bindPlayback(audio, props.recording.transcript, props.recording.resumeKey, {
      position: setPosition,
      segment: props.onSegment,
      playing: setPlaying,
      buffering: setBuffering,
      ready() {
        setReady(true);
        props.onSeekable(seek);
      },
      failed: props.onError,
    });
    return () => {
      binding?.dispose();
      binding = undefined;
      props.onSeekable();
      props.onSegment(-1);
    };
  });

  async function toggle() {
    if (!audio.paused) {
      audio.pause();
      return;
    }
    if (audio.ended) seek(0);
    try {
      await audio.play();
    } catch (error) {
      if (binding && !(error instanceof DOMException && error.name === "AbortError"))
        props.onError();
    }
  }
  return (
    <div class={styles.transport}>
      <audio
        ref={audio}
        preload="metadata"
        aria-label="Article recording"
        src={props.recording.src}
      />
      <div class={styles.seekBar}>
        <input
          type="range"
          aria-label="Playback position"
          aria-valuetext={formatTime(position())}
          min="0"
          max={duration()}
          step="any"
          value={position()}
          disabled={!ready()}
          onInput={(event) => seek(event.currentTarget.valueAsNumber)}
        />
        <span class={styles.time}>
          {formatTime(position())} / {formatTime(duration())}
        </span>
      </div>
      <div class={styles.actions}>
        <button
          type="button"
          class={controls.primary}
          disabled={!ready()}
          onClick={() => void toggle()}
        >
          {playing() ? "Pause" : "Play"}
        </button>
        <button
          type="button"
          class={controls.secondary}
          aria-label="Back 5 seconds"
          disabled={!ready()}
          onClick={() => seek(audio.currentTime * 1000 - 5000)}
        >
          −5 s
        </button>
        <button
          type="button"
          class={controls.secondary}
          aria-label="Forward 5 seconds"
          disabled={!ready()}
          onClick={() => seek(audio.currentTime * 1000 + 5000)}
        >
          +5 s
        </button>
        <select
          aria-label="Playback speed"
          value={speed()}
          onChange={(event) => {
            const value = validatePreference("playbackSpeed", event.currentTarget.value);
            audio.playbackRate = Number(value);
            setSpeed(value);
            savePreference("playbackSpeed", value);
          }}
        >
          <For each={playbackSpeeds}>
            {(rate) => (
              <option value={rate} selected={speed() === rate}>
                {rate}×
              </option>
            )}
          </For>
        </select>
        <Show when={buffering()}>
          <span role="status">Buffering…</span>
        </Show>
      </div>
    </div>
  );
}
