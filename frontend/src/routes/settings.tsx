import { For, Show } from "solid-js";
import Page from "../components/Page";
import controls from "../components/Controls.module.css";
import LocalData from "../features/settings/LocalData";
import SpeechDefaults from "../features/settings/SpeechDefaults";
import SystemStatus from "../features/settings/SystemStatus";
import styles from "../features/settings/Settings.module.css";
import { usePreferences } from "../preferences/Preferences";
import { playbackSpeeds, validatePreference, vocabularyDisplays } from "../preferences/storage";

export default function Settings() {
  const preferences = usePreferences();
  return (
    <Page title="Settings">
      <div class={styles.stack}>
        <section class={styles.panel} aria-labelledby="general-settings-title">
          <h2 id="general-settings-title">Preferences</h2>
          <div class={styles.fields}>
            <label>
              Appearance
              <select
                value={preferences.theme()}
                onChange={(event) =>
                  preferences.changeTheme(validatePreference("theme", event.currentTarget.value))
                }
              >
                <option value="system">System</option>
                <option value="light">Light</option>
                <option value="dark">Dark</option>
              </select>
            </label>
            <label>
              Active language
              <select
                disabled={!preferences.languages()?.length}
                value={preferences.language()}
                onChange={(event) =>
                  preferences.changeLanguage(
                    validatePreference("language", event.currentTarget.value),
                  )
                }
              >
                <Show
                  when={preferences.languages()?.length}
                  fallback={
                    <option value={preferences.language()}>
                      {preferences.language() === "zh-CN" ? "Mandarin Chinese" : "Korean"}
                    </option>
                  }
                >
                  <For each={preferences.languages()}>
                    {(language) => (
                      <option
                        value={language.code}
                        selected={language.code === preferences.language()}
                      >
                        {language.displayName}
                      </option>
                    )}
                  </For>
                </Show>
              </select>
            </label>
            <label>
              Default Reader speed
              <select
                value={preferences.playbackSpeed()}
                onChange={(event) =>
                  preferences.changePlaybackSpeed(
                    validatePreference("playbackSpeed", event.currentTarget.value),
                  )
                }
              >
                <For each={playbackSpeeds}>
                  {(speed) => (
                    <option value={speed} selected={speed === preferences.playbackSpeed()}>
                      {speed}×
                    </option>
                  )}
                </For>
              </select>
            </label>
            <label>
              Reader vocabulary display
              <select
                value={preferences.reader().vocabularyDisplay}
                onChange={(event) => {
                  const display = vocabularyDisplays.find(
                    (option) => option === event.currentTarget.value,
                  );
                  if (display) preferences.changeReader({ vocabularyDisplay: display });
                }}
              >
                <option value="both">Both</option>
                <option value="learning">Learning</option>
                <option value="acquired">Acquired</option>
              </select>
            </label>
          </div>
          <label class={styles.checkbox}>
            <input
              type="checkbox"
              checked={preferences.reader().showVocabularyDetails}
              onChange={(event) =>
                preferences.changeReader({ showVocabularyDetails: event.currentTarget.checked })
              }
            />
            Open vocabulary details by default in the Reader
          </label>
        </section>
        <section class={styles.panel} aria-labelledby="speech-settings-title">
          <h2 id="speech-settings-title">Speech defaults</h2>
          <p class={styles.hint}>Defaults apply when you create an article in that language.</p>
          <Show when={preferences.languageError()}>
            <p role="alert">Could not load learning languages.</p>
            <button type="button" class={controls.secondary} onClick={preferences.retryLanguages}>
              Retry languages
            </button>
          </Show>
          <Show when={!preferences.languages() && !preferences.languageError()}>
            <p role="status">Loading languages…</p>
          </Show>
          <Show when={preferences.languages()?.length === 0}>
            <p>No learning languages are enabled.</p>
          </Show>
          <For each={preferences.languages()}>
            {(language) => <SpeechDefaults language={language} />}
          </For>
        </section>
        <section class={styles.panel} aria-labelledby="system-status-title">
          <SystemStatus />
        </section>
        <section class={styles.panel} aria-labelledby="local-data-title">
          <LocalData />
        </section>
      </div>
    </Page>
  );
}
