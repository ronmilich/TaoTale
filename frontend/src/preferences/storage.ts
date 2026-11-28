import type { LearningLanguage } from "../api/client";

export type Language = LearningLanguage["code"];
export const speechRates = [0.75, 1, 1.25] as const;
export type SpeechRate = (typeof speechRates)[number];
type SpeechPreference = { voiceId: string; speakingRate: SpeechRate };
export const themes = ["system", "light", "dark"] as const;
export type Theme = (typeof themes)[number];
export const playbackSpeeds = ["0.75", "1", "1.25", "1.5"] as const;
export const vocabularyDisplays = ["both", "learning", "acquired"] as const;
export type VocabularyDisplay = (typeof vocabularyDisplays)[number];
type Preferences = {
  theme: Theme;
  language: Language;
  playbackSpeed: (typeof playbackSpeeds)[number];
};

const prefix = "taotale.preferences.";
const defaults: Preferences = { theme: "system", language: "zh-CN", playbackSpeed: "1" };
const choices: { [K in keyof Preferences]: ReadonlyArray<Preferences[K]> } = {
  theme: themes,
  language: ["zh-CN", "ko-KR"],
  playbackSpeed: playbackSpeeds,
};

export function validatePreference<K extends keyof Preferences>(
  key: K,
  value: string | null,
): Preferences[K] {
  return choices[key].find((choice) => choice === value) ?? defaults[key];
}

export function readPreference<K extends keyof Preferences>(key: K): Preferences[K] {
  try {
    return validatePreference(key, window.localStorage.getItem(prefix + key));
  } catch {
    return defaults[key];
  }
}

export function savePreference<K extends keyof Preferences>(key: K, value: Preferences[K]) {
  try {
    window.localStorage.setItem(prefix + key, value);
  } catch {
    // The in-memory setting still works when storage is unavailable.
  }
}

export function readSpeechPreference(language: Language): SpeechPreference | undefined {
  try {
    const value = JSON.parse(window.localStorage.getItem(prefix + "speech." + language) ?? "null");
    if (value && typeof value.voiceId === "string" && speechRates.includes(value.speakingRate)) {
      return { voiceId: value.voiceId, speakingRate: value.speakingRate };
    }
  } catch {
    // A malformed or inaccessible preference falls back to the voice catalog.
  }
}

export function saveSpeechPreference(language: Language, value: SpeechPreference) {
  try {
    window.localStorage.setItem(prefix + "speech." + language, JSON.stringify(value));
  } catch {
    // The draft still has the selected values.
  }
}

export function readVocabularyDisplay(): VocabularyDisplay {
  try {
    const value = JSON.parse(window.localStorage.getItem(prefix + "reader") ?? "null");
    return vocabularyDisplays.find((option) => option === value?.vocabularyDisplay) ?? "both";
  } catch {
    return "both";
  }
}

export function saveVocabularyDisplay(value: VocabularyDisplay) {
  try {
    window.localStorage.setItem(prefix + "reader", JSON.stringify({ vocabularyDisplay: value }));
  } catch {
    // The panel still uses the selected display in memory.
  }
}

export const darkMedia = "(prefers-color-scheme: dark)";

export function applyTheme(mode: Theme) {
  const dark = mode === "dark" || (mode === "system" && window.matchMedia?.(darkMedia).matches);
  document.documentElement.dataset.theme = dark ? "dark" : "light";
}

export const themeInitializer = `(() => {
  let theme;
  try { theme = localStorage.getItem("${prefix}theme"); } catch {}
  const dark = theme === "dark" ||
    (theme !== "light" && window.matchMedia?.("${darkMedia}").matches);
  document.documentElement.dataset.theme = dark ? "dark" : "light";
})();`;
