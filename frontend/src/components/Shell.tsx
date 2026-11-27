import { useLocation, useNavigate } from "@solidjs/router";
import { For, Show, type ParentProps } from "solid-js";
import { usePreferences } from "../preferences/Preferences";
import { validatePreference } from "../preferences/storage";
import styles from "./Shell.module.css";

const navigation = [
  { label: "Library", href: "/library" },
  { label: "Vocabulary", href: "/vocabulary" },
  { label: "Settings", href: "/settings" },
];

export default function Shell(props: ParentProps) {
  const location = useLocation();
  const navigate = useNavigate();
  const preferences = usePreferences();
  const current = (href: string) =>
    location.pathname === href ||
    (href === "/library" &&
      (/^\/items(?:\/|$)/.test(location.pathname) || ["/", "/create"].includes(location.pathname)));

  return (
    <>
      <a class={styles.skip} href="#main">
        Skip to content
      </a>
      <Show when={!/^\/items\/[^/]+\/?$/i.test(location.pathname)}>
        <header class={styles.header}>
          <div class={styles.bar}>
            <a href="/library" class={styles.brand}>
              <span class={styles.mark} aria-hidden="true">
                ✿
              </span>
              TaoTale
            </a>
            <nav class={styles.nav} aria-label="Main navigation">
              <For each={navigation}>
                {(item) => (
                  <a href={item.href} aria-current={current(item.href) ? "page" : undefined}>
                    {item.label}
                  </a>
                )}
              </For>
            </nav>
            <div class={styles.language}>
              <label for="active-language">Reading language</label>
              <select
                id="active-language"
                disabled={!preferences.languages()?.length}
                value={preferences.language()}
                onChange={(event) => {
                  const language = validatePreference("language", event.currentTarget.value);
                  preferences.changeLanguage(language);
                  if (location.pathname === "/library")
                    navigate(`/library?languageCode=${language}&page=1`);
                }}
              >
                <Show
                  when={preferences.languages()?.length}
                  fallback={
                    <option value={preferences.language()}>
                      {preferences.language() === "ko-KR" ? "Korean" : "Mandarin Chinese"}
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
              <Show when={preferences.languageError() || preferences.languages()?.length === 0}>
                <small role="status">
                  {preferences.languageError() ? "Languages unavailable" : "No enabled languages"}
                </small>
              </Show>
            </div>
          </div>
        </header>
      </Show>
      <main id="main" class={styles.main} tabindex={-1}>
        {props.children}
      </main>
    </>
  );
}
