import { For, Show } from "solid-js";
import AudioPlayer from "../../components/AudioPlayer";
import controls from "../../components/Controls.module.css";
import type { SpeechRate } from "../../preferences/storage";
import styles from "./DraftEditor.module.css";
import { audioStatusLabel, type useDraftAudio } from "./useDraftAudio";

export default function AudioPanel(props: {
  audio: ReturnType<typeof useDraftAudio>;
  saved: boolean;
  changed: boolean;
  disabled: boolean;
}) {
  const selectedVoice = () => props.audio.selection()?.voice.voiceId ?? "";
  const selectedRate = () => props.audio.selection()?.rate;
  const unavailableVoice = () => props.audio.voices()?.length && !selectedVoice();

  return (
    <section class={styles.paper} aria-labelledby="draft-audio">
      <h2 id="draft-audio">Audio</h2>
      <div class={styles.speechFields}>
        <div class={styles.field}>
          <label for="draft-voiceId">Voice</label>
          <select
            id="draft-voiceId"
            value={selectedVoice()}
            disabled={props.disabled || !props.audio.voices()?.length}
            onChange={(event) =>
              props.audio.changeSpeech(event.currentTarget.value, selectedRate() ?? 1)
            }
          >
            <Show when={!selectedVoice()}>
              <option value="">Choose a voice</option>
            </Show>
            <For each={props.audio.voices()}>
              {(voice) => (
                <option value={voice.voiceId} selected={voice.voiceId === selectedVoice()}>
                  {voice.displayName}
                </option>
              )}
            </For>
          </select>
        </div>
        <div class={styles.field}>
          <label for="draft-speakingRate">Speaking rate</label>
          <select
            id="draft-speakingRate"
            value={String(selectedRate() ?? 1)}
            disabled={props.disabled || selectedRate() === undefined}
            onChange={(event) =>
              props.audio.changeSpeech(
                selectedVoice(),
                Number(event.currentTarget.value) as SpeechRate,
              )
            }
          >
            <For each={props.audio.selection()?.rates ?? [1]}>
              {(rate) => (
                <option value={String(rate)} selected={rate === selectedRate()}>
                  {rate}×
                </option>
              )}
            </For>
          </select>
        </div>
      </div>
      <Show when={props.audio.voiceError()}>
        <div class={styles.audioMessage}>
          <p role="status">{props.audio.voiceError()}</p>
          <button
            type="button"
            class={controls.secondary}
            onClick={() => props.audio.retryVoices()}
          >
            Retry voices
          </button>
        </div>
      </Show>
      <Show when={props.audio.voices()?.length === 0}>
        <p class={styles.hint}>No voices are available for this language.</p>
      </Show>
      <Show when={unavailableVoice()}>
        <p class={styles.hint}>Choose an available voice to replace the saved voice.</p>
      </Show>
      <div class={styles.audioActions}>
        <button
          type="button"
          class={controls.primary}
          disabled={!props.audio.canGenerate()}
          onClick={() => props.audio.generate()}
        >
          {props.saved ? "Generate Audio" : "Generate Preview"}
        </button>
        <p role="status" class={styles.audioStatus}>
          {props.audio.submitting() ? "Submitting…" : audioStatusLabel[props.audio.status()]}
        </p>
      </div>
      <Show when={!props.saved}>
        <p class={styles.hint}>Temporary previews expire after 24 hours.</p>
      </Show>
      <Show when={props.saved && props.changed}>
        <p class={styles.hint}>Save your text, voice, or rate changes before generating audio.</p>
      </Show>
      <Show when={props.audio.errorCode() === "GENERATION_INTERRUPTED"}>
        <p role="status">
          The backend restarted during generation. You can start a new generation when ready.
        </p>
      </Show>
      <Show
        when={
          props.audio.status() === "FAILED" && props.audio.errorCode() !== "GENERATION_INTERRUPTED"
        }
      >
        <p class={styles.hint}>Audio generation failed. Check the speech provider and try again.</p>
      </Show>
      <Show when={props.audio.actionError()}>
        <p role="alert" class={styles.fieldError}>
          {props.audio.actionError()}
        </p>
      </Show>
      <Show when={props.audio.needsRecovery() && !props.audio.submitting()}>
        <button
          type="button"
          class={`${controls.secondary} ${styles.fieldAction}`}
          disabled={props.disabled}
          onClick={() => props.audio.recover()}
        >
          Recover generation
        </button>
      </Show>
      <Show when={props.audio.pollError()}>
        <div class={styles.audioMessage}>
          <p role="status">{props.audio.pollError()}</p>
          <button
            type="button"
            class={controls.secondary}
            onClick={() => props.audio.checkStatus()}
          >
            Check audio status
          </button>
        </div>
      </Show>
      <Show when={props.audio.media()?.src} keyed>
        {(src) => <AudioPlayer src={src} download={props.audio.media()!.download} />}
      </Show>
    </section>
  );
}
