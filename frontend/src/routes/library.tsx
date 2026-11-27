import { Title } from "@solidjs/meta";
import { useLocation, useNavigate } from "@solidjs/router";
import { For, Show, createEffect, createMemo, createSignal, onCleanup } from "solid-js";
import { listItems, type ItemPage, type LearningLanguage } from "../api/client";
import controls from "../components/Controls.module.css";
import LibraryItemCard from "../features/library/LibraryItemCard";
import LibraryFilters from "../features/library/LibraryFilters";
import { readFilters, readPage, searchLimit } from "../features/library/filters";
import styles from "../features/library/Library.module.css";
import TopicManager from "../features/topics/TopicManager";
import { useTopics } from "../features/topics/topics";
import { usePreferences } from "../preferences/Preferences";

type Language = LearningLanguage["code"];

function pageURL(language: Language, filters: URLSearchParams, page = 1) {
  const query = new URLSearchParams([
    ["languageCode", language],
    ...filters,
    ["page", String(page)],
  ]);
  return `/library?${query}`;
}

export default function Library() {
  const location = useLocation();
  const navigate = useNavigate();
  const preferences = usePreferences();
  const topics = useTopics();
  const query = createMemo(() => new URLSearchParams(location.search));
  const language = createMemo(() => {
    const available = preferences.languages();
    const values = query().getAll("languageCode");
    const requested = values.length === 1 ? values[0] : undefined;
    return (
      available?.find((entry) => entry.code === requested)?.code ??
      available?.find((entry) => entry.code === preferences.language())?.code
    );
  });
  const filterKey = createMemo(() => readFilters(query()).toString());
  const filters = createMemo(() => new URLSearchParams(filterKey()));
  const page = createMemo(() => readPage(query()));
  const [search, setSearch] = createSignal("");
  let searchTimer: number | undefined;
  const cancelSearch = () => window.clearTimeout(searchTimer);
  onCleanup(cancelSearch);
  createEffect(
    () => location.search,
    () => {
      cancelSearch();
      setSearch(filters().get("q") ?? "");
    },
  );

  function changeFilters(next: URLSearchParams, replace = false) {
    cancelSearch();
    const code = language();
    const canonical = readFilters(next);
    if (code && canonical.toString() !== filterKey()) {
      navigate(pageURL(code, canonical), { replace, scroll: false });
    }
  }

  function changeFilter(key: string, value: string) {
    const next = new URLSearchParams(filters());
    next.set("q", search());
    next.set(key, value);
    changeFilters(next);
  }

  function searchInput(value: string) {
    const bounded = [...value].slice(0, searchLimit).join("");
    setSearch(bounded);
    cancelSearch();
    searchTimer = window.setTimeout(() => {
      const next = new URLSearchParams(filters());
      next.set("q", bounded);
      changeFilters(next, true);
    }, 300);
    return bounded;
  }
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
    () => [language(), page(), filterKey(), refresh()] as const,
    ([code, number, key]) => {
      if (!code) return;
      if (code !== preferences.language()) preferences.changeLanguage(code);
      const controller = new AbortController();
      setLoading(true);
      setFailed(false);
      const requested = new URLSearchParams(key);
      listItems(code, number, controller.signal, requested).then(
        (data) => {
          if (controller.signal.aborted) return;
          const last = Math.max(1, Math.ceil(data.total / data.pageSize));
          if (number > last) {
            navigate(pageURL(code, requested, last), { replace: true, scroll: false });
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
    () => [language(), location.search, filterKey(), page()] as const,
    ([code, search, key, number]) => {
      if (!code || !search) return;
      const canonical = pageURL(code, new URLSearchParams(key), number);
      if (canonical !== `/library${search}`) {
        navigate(canonical, { replace: true, scroll: false });
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
        <LibraryFilters
          search={search()}
          filters={filters()}
          topics={topics.items()}
          searchInput={searchInput}
          change={changeFilter}
        />
        <TopicManager
          topics={topics}
          removed={(id) => {
            const selected = filters().get("topics")?.split(",") ?? [];
            if (selected.includes(id))
              changeFilter("topics", selected.filter((topic) => topic !== id).join(","));
            else setRefresh((value) => value + 1);
          }}
        />
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
                      <Show
                        when={
                          filters().has("q") ||
                          filters().has("status") ||
                          filters().has("difficulty") ||
                          filters().has("favoritesOnly") ||
                          filters().has("topics")
                        }
                        fallback={
                          <>
                            <h2>No saved articles in this language</h2>
                            <a class={controls.primary} href="/create">
                              Create New Item
                            </a>
                          </>
                        }
                      >
                        <h2>No items match your filters</h2>
                        <button
                          type="button"
                          class={controls.secondary}
                          onClick={() => changeFilters(new URLSearchParams())}
                        >
                          Clear filters
                        </button>
                      </Show>
                    </div>
                  </Show>
                }
              >
                <div class={styles.grid} aria-busy={loading() ? "true" : undefined}>
                  <For each={result().items}>
                    {(item) => (
                      <LibraryItemCard
                        item={item}
                        topics={topics.items()}
                        favoriteChanged={() => {
                          if (filters().has("favoritesOnly")) setRefresh((value) => value + 1);
                        }}
                      />
                    )}
                  </For>
                </div>
              </Show>
              <Show when={maxPage() > 1}>
                <nav class={styles.paging} aria-label="Library pages">
                  <Show when={result().page > 1}>
                    <a
                      class={controls.secondary}
                      href={pageURL(language()!, filters(), result().page - 1)}
                    >
                      Previous
                    </a>
                  </Show>
                  <span>
                    Page {result().page} of {maxPage()}
                  </span>
                  <Show when={result().page < maxPage()}>
                    <a
                      class={controls.secondary}
                      href={pageURL(language()!, filters(), result().page + 1)}
                    >
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
