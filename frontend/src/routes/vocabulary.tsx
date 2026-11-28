import { Title } from "@solidjs/meta";
import { useLocation, useNavigate } from "@solidjs/router";
import { Show, createEffect, createMemo, createSignal, onCleanup } from "solid-js";
import {
  bulkVocabulary,
  getVocabularyStats,
  listVocabulary,
  type VocabularyBulk,
  type VocabularyPage,
  type VocabularyStats,
} from "../api/client";
import controls from "../components/Controls.module.css";
import { readPage, searchLimit, single } from "../features/listQuery";
import { useTopics } from "../features/topics/topics";
import { vocabularyFilters, vocabularyURL } from "../features/vocabulary/query";
import VocabularyFilters from "../features/vocabulary/VocabularyFilters";
import VocabularyTable from "../features/vocabulary/VocabularyTable";
import VocabularyWordEditor from "../features/vocabulary/VocabularyWordEditor";
import styles from "../features/vocabulary/Vocabulary.module.css";
import { statusLabel, useVocabularyChanges } from "../features/vocabulary/words";
import { usePreferences } from "../preferences/Preferences";
import type { Language } from "../preferences/storage";

export default function Vocabulary() {
  const location = useLocation();
  const navigate = useNavigate();
  const preferences = usePreferences();
  const topics = useTopics();
  const query = createMemo(() => new URLSearchParams(location.search));
  const language = createMemo(() => {
    const requested = single(query(), "languageCode");
    const available = preferences.languages();
    return (
      available?.find(({ code }) => code === requested)?.code ??
      available?.find(({ code }) => code === preferences.language())?.code
    );
  });
  const languageName = () =>
    preferences.languages()?.find(({ code }) => code === language())?.displayName;
  const filterKey = createMemo(() => vocabularyFilters(query()).toString());
  const filters = createMemo(() => new URLSearchParams(filterKey()));
  const page = createMemo(() => readPage(query()));
  const address = createMemo(() => {
    const code = language();
    return code ? vocabularyURL(code, filters(), page()) : "";
  });
  const [search, setSearch] = createSignal("");
  let searchTimer: number | undefined;
  const cancelSearch = () => window.clearTimeout(searchTimer);
  onCleanup(cancelSearch);

  const [refresh, setRefresh] = createSignal(0);
  const [detailRefresh, setDetailRefresh] = createSignal(0);
  const [lastPage, setLastPage] = createSignal<{ address: string; result: VocabularyPage }>();
  const [lastStats, setLastStats] = createSignal<VocabularyStats>();
  const [pageError, setPageError] = createSignal(false);
  const [statsError, setStatsError] = createSignal(false);
  const [loading, setLoading] = createSignal(true);
  const [selection, setSelection] = createSignal<string[]>([]);
  const [detail, setDetail] = createSignal<{ language: Language; id?: string }>();
  const [mutation, setMutation] = createSignal<"editor" | "bulk">();
  const [message, setMessage] = createSignal("");
  const [error, setError] = createSignal("");
  const activeDetail = createMemo(() => (detail()?.language === language() ? detail() : undefined));
  const visible = createMemo(() => {
    const value = lastPage();
    return value?.address === address() ? value.result : undefined;
  });
  const selected = createMemo(() =>
    selection().filter((id) => visible()?.items.some((word) => word.id === id)),
  );
  const stats = createMemo(() =>
    lastStats()?.languageCode === language() ? lastStats() : undefined,
  );
  const maxPage = () =>
    Math.max(1, Math.ceil((visible()?.total ?? 0) / (visible()?.pageSize ?? 10)));
  const reload = () => setRefresh((number) => number + 1);
  const refreshDetail = () => setDetailRefresh((number) => number + 1);
  const broadcast = useVocabularyChanges((changedLanguage) => {
    if (changedLanguage === language()) {
      reload();
      refreshDetail();
    }
  });

  createEffect(
    () => location.search,
    () => {
      cancelSearch();
      setSearch(filters().get("q") ?? "");
    },
  );

  createEffect(address, () => {
    setSelection([]);
    setError("");
  });

  createEffect(language, () => {
    setMessage("");
    setDetail(undefined);
  });

  createEffect(
    () => [language(), address(), location.search] as const,
    ([code, canonical, current]) => {
      if (!code) return;
      if (code !== preferences.language()) preferences.changeLanguage(code);
      if (current && canonical !== `/vocabulary${current}`)
        navigate(canonical, { replace: true, scroll: false });
    },
  );

  createEffect(
    () => [language(), page(), filterKey(), address(), refresh()] as const,
    ([code, number, key, url]) => {
      if (!code) return;
      const controller = new AbortController();
      const requested = new URLSearchParams(key);
      setLoading(true);
      setPageError(false);
      listVocabulary(
        code,
        number,
        requested,
        AbortSignal.any([controller.signal, AbortSignal.timeout(15_000)]),
      ).then(
        (result) => {
          if (controller.signal.aborted) return;
          const last = Math.max(1, Math.ceil(result.total / result.pageSize));
          if (number > last) {
            navigate(vocabularyURL(code, requested, last), { replace: true, scroll: false });
            return;
          }
          setLastPage({ address: url, result });
          setLoading(false);
        },
        () => {
          if (controller.signal.aborted) return;
          setPageError(true);
          setLoading(false);
        },
      );
      return () => controller.abort();
    },
  );

  createEffect(
    () => [language(), refresh()] as const,
    ([code]) => {
      if (!code) return;
      const controller = new AbortController();
      setStatsError(false);
      getVocabularyStats(
        code,
        AbortSignal.any([controller.signal, AbortSignal.timeout(15_000)]),
      ).then(
        (result) => {
          if (!controller.signal.aborted) setLastStats(result);
        },
        () => {
          if (!controller.signal.aborted) setStatsError(true);
        },
      );
      return () => controller.abort();
    },
  );

  function changeFilters(next: URLSearchParams, replace = false) {
    cancelSearch();
    const code = language();
    const canonical = vocabularyFilters(next);
    if (code && canonical.toString() !== filterKey()) {
      navigate(vocabularyURL(code, canonical), { replace, scroll: false });
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

  function open(id?: string) {
    const code = language();
    if (code) setDetail({ language: code, id });
  }

  function changed(code: Language, text: string) {
    broadcast(code);
    if (code !== language()) return;
    setMessage(text);
    reload();
  }

  async function applyBulk(action: VocabularyBulk["action"], status?: VocabularyBulk["status"]) {
    const ids = selected();
    const code = language();
    if (!code || !ids.length || mutation() || loading()) return;
    if (
      action === "delete" &&
      !window.confirm(
        `Delete ${ids.length} selected ${ids.length === 1 ? "word" : "words"} and their article links?`,
      )
    )
      return;
    setMutation("bulk");
    setMessage("");
    setError("");
    try {
      const { affected } = await bulkVocabulary({ action, status, ids, languageCode: code });
      const label = `${affected} ${affected === 1 ? "word" : "words"}`;
      changed(code, status ? `${label} marked ${statusLabel[status]}.` : `${label} deleted.`);
      if (code !== language()) return;
      setSelection([]);
      const id = activeDetail()?.id;
      if (id && ids.includes(id)) {
        if (action === "delete") setDetail(undefined);
        else refreshDetail();
      }
    } catch {
      if (code === language())
        setError("Could not confirm the changes. The table shows the last confirmed status.");
    } finally {
      setMutation(undefined);
    }
  }

  const unavailable = () => preferences.languageError() || pageError() || statsError();
  return (
    <div class={styles.page}>
      <Title>Vocabulary · TaoTale</Title>
      <header class={styles.header}>
        <div>
          <h1>Vocabulary</h1>
          <Show when={languageName()}>
            <p>{languageName()}</p>
          </Show>
        </div>
        <button
          type="button"
          class={controls.primary}
          disabled={!language() || Boolean(mutation())}
          onClick={() => open()}
        >
          Add word
        </button>
      </header>
      <Show when={stats()}>
        {(totals) => (
          <dl class={styles.summary} aria-label="Vocabulary statistics">
            <div>
              <dt>Total words</dt>
              <dd>{totals().total}</dd>
            </div>
            <div>
              <dt>Learning</dt>
              <dd>{totals().learning}</dd>
            </div>
            <div>
              <dt>Acquired</dt>
              <dd>{totals().acquired}</dd>
            </div>
            <div>
              <dt>Referenced texts</dt>
              <dd>{totals().referencedTexts}</dd>
            </div>
          </dl>
        )}
      </Show>
      <Show when={unavailable()}>
        <div class={styles.controls}>
          <p role="alert" class={styles.error}>
            {preferences.languageError()
              ? "Could not load learning languages."
              : statsError() && !pageError()
                ? "Could not load vocabulary statistics."
                : "Could not load vocabulary."}
          </p>
          <button
            type="button"
            class={controls.secondary}
            onClick={() => {
              if (preferences.languageError()) preferences.retryLanguages();
              else reload();
            }}
          >
            Retry
          </button>
        </div>
      </Show>
      <Show when={preferences.languages()?.length === 0}>
        <p>No learning languages are enabled.</p>
      </Show>
      <Show when={activeDetail()} keyed>
        {(active) => (
          <VocabularyWordEditor
            language={active.language}
            id={active.id}
            refresh={detailRefresh()}
            topics={topics}
            disabled={mutation() === "bulk"}
            close={() => setDetail(undefined)}
            open={open}
            busyChanged={(busy) => setMutation(busy ? "editor" : undefined)}
            changed={(message) => changed(active.language, message)}
          />
        )}
      </Show>
      <Show
        when={language()}
        fallback={
          <Show when={!unavailable() && !preferences.languages()}>
            <p role="status">Loading vocabulary…</p>
          </Show>
        }
      >
        {(code) => (
          <section class={styles.list} aria-label="Browse vocabulary">
            <VocabularyFilters
              filters={filters()}
              search={search()}
              topics={topics}
              change={changeFilter}
              searchInput={searchInput}
              clear={() => {
                setSearch("");
                changeFilters(new URLSearchParams());
              }}
            />
            <Show when={message()}>
              <p role="status">{message()}</p>
            </Show>
            <Show when={error()}>
              <p role="alert" class={styles.error}>
                {error()}
              </p>
            </Show>
            <Show when={loading()}>
              <p role="status" class={styles.muted}>
                {visible() ? "Refreshing vocabulary…" : "Loading vocabulary…"}
              </p>
            </Show>
            <Show when={selected().length}>
              <div class={styles.controls}>
                <span role="status">
                  {mutation() === "bulk"
                    ? "Updating selected words…"
                    : `${selected().length} selected`}
                </span>
                <button
                  type="button"
                  class={controls.secondary}
                  disabled={loading() || Boolean(mutation())}
                  onClick={() => void applyBulk("setStatus", "LEARNING")}
                >
                  Mark selected Learning
                </button>
                <button
                  type="button"
                  class={controls.secondary}
                  disabled={loading() || Boolean(mutation())}
                  onClick={() => void applyBulk("setStatus", "ACQUIRED")}
                >
                  Mark selected Acquired
                </button>
                <button
                  type="button"
                  class={controls.secondary}
                  disabled={loading() || Boolean(mutation())}
                  onClick={() => void applyBulk("delete")}
                >
                  Delete selected
                </button>
              </div>
            </Show>
            <Show when={visible()}>
              {(result) => (
                <>
                  <Show
                    when={result().items.length}
                    fallback={
                      <p class={styles.muted}>
                        {filters().size
                          ? "No words match your filters."
                          : "No words saved in this language yet."}
                      </p>
                    }
                  >
                    <VocabularyTable
                      words={result().items}
                      topics={topics.items()}
                      selected={selected()}
                      changeSelection={setSelection}
                      disabled={loading() || Boolean(mutation())}
                      open={open}
                    />
                  </Show>
                  <Show when={maxPage() > 1}>
                    <nav class={styles.paging} aria-label="Vocabulary pages">
                      <Show when={result().page > 1}>
                        <a
                          class={controls.secondary}
                          href={vocabularyURL(code(), filters(), result().page - 1)}
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
                          href={vocabularyURL(code(), filters(), result().page + 1)}
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
        )}
      </Show>
    </div>
  );
}
