import { Title } from "@solidjs/meta";
import { useLocation, useNavigate } from "@solidjs/router";
import { For, Show, createEffect, createMemo, createSignal } from "solid-js";
import { listItems, type ItemPage, type LearningLanguage } from "../api/client";
import controls from "../components/Controls.module.css";
import LibraryItemCard from "../features/library/LibraryItemCard";
import styles from "../features/library/Library.module.css";
import { usePreferences } from "../preferences/Preferences";

type Language = LearningLanguage["code"];

function pageURL(language: Language, page = 1) {
  return `/library?languageCode=${language}&page=${page}`;
}

export default function Library() {
  const location = useLocation();
  const navigate = useNavigate();
  const preferences = usePreferences();
  const query = createMemo(() => new URLSearchParams(location.search));
  const language = createMemo(() => {
    const available = preferences.languages();
    const requested = query().get("languageCode");
    return (
      available?.find((entry) => entry.code === requested)?.code ??
      available?.find((entry) => entry.code === preferences.language())?.code
    );
  });
  const page = createMemo(() => {
    const values = query().getAll("page");
    if (values.length !== 1 || !/^\d+$/.test(values[0])) return 1;
    return Math.max(1, Math.min(Number(values[0]), 2147483647));
  });
  const [lastPage, setLastPage] = createSignal<{ language: Language; data: ItemPage }>();
  const [refresh, setRefresh] = createSignal(0);
  const [loading, setLoading] = createSignal(true);
  const [failed, setFailed] = createSignal(false);
  const visible = createMemo(() => {
    const previous = lastPage();
    return previous?.language === language() ? previous?.data : undefined;
  });
  const maxPage = () =>
    Math.max(1, Math.ceil((visible()?.total ?? 0) / (visible()?.pageSize ?? 8)));

  createEffect(
    () => [language(), page(), refresh()] as const,
    ([code, number]) => {
      if (!code) return;
      if (code !== preferences.language()) preferences.changeLanguage(code);
      const controller = new AbortController();
      setLoading(true);
      setFailed(false);
      listItems(code, number, controller.signal).then(
        (data) => {
          if (controller.signal.aborted) return;
          const last = Math.max(1, Math.ceil(data.total / data.pageSize));
          if (number > last) {
            navigate(pageURL(code, last), { replace: true, scroll: false });
            return;
          }
          setLastPage({ language: code, data });
          setLoading(false);
        },
        () => {
          if (controller.signal.aborted) return;
          setFailed(true);
          setLoading(false);
        },
      );
      return () => controller.abort();
    },
  );

  createEffect(
    () => [language(), query().getAll("page"), page()] as const,
    ([code, values, number]) => {
      if (code && values.length && (values.length !== 1 || values[0] !== String(number))) {
        navigate(pageURL(code, number), { replace: true, scroll: false });
      }
    },
  );

  return (
    <>
      <Title>Library · TaoTale</Title>
      <section class={styles.intro}>
        <div>
          <h1>Your reading library</h1>
          <p>Keep articles to read and revisit in your learning language.</p>
        </div>
        <a class={controls.primary} href="/create">
          <span aria-hidden="true">＋</span> New article
        </a>
      </section>
      <section class={styles.library} aria-label="Saved articles">
        <Show when={preferences.languageError() || failed()}>
          <div class={styles.state}>
            <p role="alert">Could not load the Library. Your saved articles are still there.</p>
            <button
              type="button"
              class={controls.secondary}
              onClick={() => {
                if (preferences.languageError()) preferences.retryLanguages();
                else setRefresh((value) => value + 1);
              }}
            >
              Retry
            </button>
          </div>
        </Show>
        <Show when={preferences.languages()?.length === 0}>
          <p class={styles.state}>No learning languages are enabled.</p>
        </Show>
        <Show
          when={visible()}
          fallback={
            <Show
              when={
                loading() && !preferences.languageError() && preferences.languages()?.length !== 0
              }
            >
              <p role="status" class={styles.updating}>
                Loading articles…
              </p>
              <div class={styles.grid} aria-hidden="true">
                <For each={[1, 2, 3, 4]}>
                  {() => (
                    <div class={styles.skeleton}>
                      <div class={styles.cover} />
                    </div>
                  )}
                </For>
              </div>
            </Show>
          }
        >
          {(result) => (
            <>
              <Show when={loading()}>
                <p role="status" class={styles.updating}>
                  Refreshing library…
                </p>
              </Show>
              <Show
                when={result().items.length}
                fallback={
                  <Show when={!loading()}>
                    <div class={styles.state}>
                      <h2>No saved articles in this language</h2>
                      <a class={controls.primary} href="/create">
                        Create New Item
                      </a>
                    </div>
                  </Show>
                }
              >
                <div class={styles.grid} aria-busy={loading() ? "true" : undefined}>
                  <For each={result().items}>{(item) => <LibraryItemCard item={item} />}</For>
                </div>
              </Show>
              <Show when={maxPage() > 1}>
                <nav class={styles.paging} aria-label="Library pages">
                  <Show when={result().page > 1}>
                    <a class={controls.secondary} href={pageURL(language()!, result().page - 1)}>
                      Previous
                    </a>
                  </Show>
                  <span>
                    Page {result().page} of {maxPage()}
                  </span>
                  <Show when={result().page < maxPage()}>
                    <a class={controls.secondary} href={pageURL(language()!, result().page + 1)}>
                      Next
                    </a>
                  </Show>
                </nav>
              </Show>
            </>
          )}
        </Show>
      </section>
    </>
  );
}
