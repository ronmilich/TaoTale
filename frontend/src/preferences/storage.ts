import type { LearningLanguage } from "../api/client";

export type Language = LearningLanguage["code"];
export const themes = ["system", "light", "dark"] as const;
export type Theme = (typeof themes)[number];
type Preferences = { theme: Theme; language: Language };

const prefix = "taotale.preferences.";
const defaults: Preferences = { theme: "system", language: "zh-CN" };
const choices: { [K in keyof Preferences]: ReadonlyArray<Preferences[K]> } = {
  theme: themes,
  language: ["zh-CN", "ko-KR"],
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
