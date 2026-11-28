import { For, Show, createEffect, createMemo, createSignal } from "solid-js";
import { listVoices, type LearningLanguage, type Voice } from "../../api/client";
import controls from "../../components/Controls.module.css";
import { usePreferences } from "../../preferences/Preferences";
import { speechRates, type SpeechRate } from "../../preferences/storage";
import styles from "./Settings.module.css";

export default function SpeechDefaults(props: { language: LearningLanguage }) {
  const preferences = usePreferences();
  const [voices, setVoices] = createSignal<Voice[]>();
  const [failed, setFailed] = createSignal(false);
  const [attempt, setAttempt] = createSignal(0);
  const selection = createMemo(() => {
    const saved = preferences.speech(props.language.code);
    const catalog = voices();
    const voice = catalog?.find((entry) => entry.voiceId === saved?.voiceId) ?? catalog?.[0];
    const rates = speechRates.filter((rate) => voice?.speakingRates.includes(rate));
    const rate =
      saved && rates.includes(saved.speakingRate)
        ? saved.speakingRate
        : rates.includes(1)
          ? 1
          : rates[0];
    return { voice, rates, rate };
  });
  createEffect(
    () => [props.language.code, attempt()] as const,
    ([code]) => {
      const request = new AbortController();
      setFailed(false);
      setVoices(undefined);
      listVoices(code, AbortSignal.any([request.signal, AbortSignal.timeout(10_000)])).then(
        ({ items }) => {
          if (!request.signal.aborted)
            setVoices(items.filter((item) => item.languageCode === code));
        },
        () => {
          if (!request.signal.aborted) setFailed(true);
        },
      );
      return () => request.abort();
    },
  );

  function chooseVoice(id: string) {
    const voice = voices()?.find((entry) => entry.voiceId === id);
    if (!voice) return;
    const current = selection().rate;
    const rates = speechRates.filter((rate) => voice.speakingRates.includes(rate));
    const speakingRate =
      current !== undefined && rates.includes(current) ? current : rates.includes(1) ? 1 : rates[0];
    if (speakingRate !== undefined)
      preferences.changeSpeech(props.language.code, { voiceId: id, speakingRate });
  }

  return (
    <fieldset class={styles.speech}>
      <legend>{props.language.displayName}</legend>
      <Show
        when={voices()?.length}
        fallback={
          <Show
            when={failed()}
            fallback={<p role="status">{voices() ? "No voices available." : "Loading voices…"}</p>}
          >
            <p role="alert">Speech voices are unavailable.</p>
            <button
              type="button"
              class={controls.secondary}
              onClick={() => setAttempt((value) => value + 1)}
            >
              Retry voices
            </button>
          </Show>
        }
      >
        <div class={styles.fields}>
          <label>
            Default voice
            <select
              value={selection().voice?.voiceId ?? ""}
              onChange={(event) => chooseVoice(event.currentTarget.value)}
            >
              <For each={voices()}>
                {(voice) => (
                  <option
                    value={voice.voiceId}
                    selected={selection().voice?.voiceId === voice.voiceId}
                  >
                    {voice.displayName}
                  </option>
                )}
              </For>
            </select>
          </label>
          <label>
            Default speaking rate
            <select
              value={String(selection().rate ?? "")}
              onChange={(event) => {
                const { voice, rates } = selection();
                const rate = Number(event.currentTarget.value) as SpeechRate;
                if (voice && rates.includes(rate))
                  preferences.changeSpeech(props.language.code, {
                    voiceId: voice.voiceId,
                    speakingRate: rate,
                  });
              }}
            >
              <For each={selection().rates}>
                {(rate) => (
                  <option value={rate} selected={selection().rate === rate}>
                    {rate}×
                  </option>
                )}
              </For>
            </select>
          </label>
        </div>
      </Show>
    </fieldset>
  );
}
