import {
  createContext,
  createEffect,
  createMemo,
  createSignal,
  onSettled,
  useContext,
  type Accessor,
  type ParentProps,
} from "solid-js";
import { apiGet, type LearningLanguage } from "../api/client";
import {
  applyTheme,
  darkMedia,
  readPreference,
  savePreference,
  type Language,
  type Theme,
} from "./storage";

interface LocalPreferences {
  theme: Accessor<Theme>;
  changeTheme: (value: Theme) => void;
  language: Accessor<Language>;
  changeLanguage: (value: Language) => void;
  languages: Accessor<LearningLanguage[] | undefined>;
  languageError: Accessor<boolean>;
  retryLanguages: () => void;
}

const Preferences = createContext<LocalPreferences>();

export function PreferenceProvider(props: ParentProps) {
  const [theme, setTheme] = createSignal(readPreference("theme"));
  const [preferredLanguage, setPreferredLanguage] = createSignal(readPreference("language"));
  const [languages, setLanguages] = createSignal<LearningLanguage[]>();
  const [languageError, setLanguageError] = createSignal(false);
  const [languageAttempt, setLanguageAttempt] = createSignal(0);
  const language = createMemo(() => {
    const preferred = preferredLanguage();
    const available = languages();
    return (
      available?.find((entry) => entry.code === preferred)?.code ??
      available?.[0]?.code ??
      preferred
    );
  });

  createEffect(theme, (mode) => applyTheme(mode));
  createEffect(languageAttempt, () => {
    const controller = new AbortController();
    apiGet("/languages", controller.signal).then(
      (result) => {
        if (!controller.signal.aborted) setLanguages(result.items);
      },
      () => {
        if (!controller.signal.aborted) setLanguageError(true);
      },
    );
    return () => controller.abort();
  });
  onSettled(() => {
    const media = window.matchMedia?.(darkMedia);
    const onSystemChange = () => applyTheme(theme());
    media?.addEventListener("change", onSystemChange);
    return () => {
      media?.removeEventListener("change", onSystemChange);
    };
  });

  const value: LocalPreferences = {
    theme,
    changeTheme(mode) {
      setTheme(mode);
      savePreference("theme", mode);
    },
    language,
    changeLanguage(code) {
      setPreferredLanguage(code);
      savePreference("language", code);
    },
    languages,
    languageError,
    retryLanguages() {
      setLanguageError(false);
      setLanguageAttempt((attempt) => attempt + 1);
    },
  };
  return <Preferences value={value}>{props.children}</Preferences>;
}

export function usePreferences() {
  const preferences = useContext(Preferences);
  if (!preferences) throw new Error("Preferences are unavailable outside the application shell.");
  return preferences;
}
