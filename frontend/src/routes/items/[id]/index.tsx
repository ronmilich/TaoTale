import type { RouteProps } from "@solidjs/router";
import { createEffect, createMemo, createSignal, Show } from "solid-js";
import { ApiError, getItem, type Item } from "../../../api/client";
import Page, { MissingPage } from "../../../components/Page";
import { samples } from "../../../features/library/samples";
import ReaderArticle from "../../../features/reader/ReaderArticle";

export default function Reader(props: RouteProps<"/items/:id">) {
  const article = createMemo(() =>
    Object.values(samples).find((entry) => entry.id === props.params.id),
  );
  const [saved, setSaved] = createSignal<{ routeId: string; item: Item }>();
  const current = createMemo(() => {
    const loaded = saved();
    return loaded?.routeId === props.params.id ? loaded.item : undefined;
  });
  const [attempt, setAttempt] = createSignal(0);
  const [status, setStatus] = createSignal<"loading" | "missing" | "failed" | "ready">("loading");
  createEffect(
    () => [props.params.id, article(), attempt()] as const,
    ([id, sample]) => {
      if (sample) return;
      const controller = new AbortController();
      setStatus("loading");
      getItem(id, controller.signal).then(
        (item) => {
          if (controller.signal.aborted) return;
          setSaved({ routeId: id, item });
          setStatus("ready");
        },
        (error) => {
          if (controller.signal.aborted) return;
          if (error instanceof ApiError && error.status === 404) {
            setSaved(undefined);
            setStatus("missing");
          } else {
            setStatus("failed");
          }
        },
      );
      return () => controller.abort();
    },
  );
  return (
    <Show
      when={article()}
      fallback={
        <>
          <Show when={!current()}>
            <Show when={status() === "loading" || status() === "ready"}>
              <p role="status">Loading article…</p>
            </Show>
            <Show when={status() === "missing"}>
              <MissingPage />
            </Show>
            <Show when={status() === "failed"}>
              <Page title="Article unavailable">
                <button type="button" onClick={() => setAttempt((value) => value + 1)}>
                  Retry
                </button>
              </Page>
            </Show>
          </Show>
          <Show when={current()} keyed>
            {(item) => (
              <Show
                when={item.status === "published"}
                fallback={
                  <Page
                    title={item.title?.trim() || "Untitled draft"}
                    description="This article is a draft."
                  >
                    <a href={`/create?draftId=${item.id}`}>Continue Editing</a>
                  </Page>
                }
              >
                <ReaderArticle
                  title={item.title?.trim() || "Untitled article"}
                  description={item.description}
                  language={item.languageCode}
                  source={item.sourceText}
                  item={item}
                  loading={status() === "loading"}
                  retry={() => setAttempt((value) => value + 1)}
                />
              </Show>
            )}
          </Show>
        </>
      }
    >
      {(sample) => (
        <ReaderArticle title={sample().title} language={sample().language} source={sample().text} />
      )}
    </Show>
  );
}
