import type { RouteProps } from "@solidjs/router";
import { createEffect, createMemo, createSignal, Show } from "solid-js";
import { ApiError, getItem, type Item } from "../../../api/client";
import Page, { MissingPage } from "../../../components/Page";
import styles from "../../../components/Page.module.css";
import { samples } from "../../../features/library/samples";

export default function Reader(props: RouteProps<"/items/:id">) {
  const article = createMemo(() =>
    Object.values(samples).find((entry) => entry.id === props.params.id),
  );
  const [saved, setSaved] = createSignal<Item>();
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
          setSaved(item);
          setStatus("ready");
        },
        (error) => {
          if (controller.signal.aborted) return;
          setStatus(error instanceof ApiError && error.status === 404 ? "missing" : "failed");
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
          <Show when={status() === "loading"}>
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
          <Show when={status() === "ready" && saved()}>
            {(item) => (
              <Show
                when={item().status === "published"}
                fallback={
                  <Page
                    title={item().title?.trim() || "Untitled draft"}
                    description="This article is a draft."
                  >
                    <a href={`/create?draftId=${item().id}`}>Continue Editing</a>
                  </Page>
                }
              >
                <Page
                  title={item().title?.trim() || "Untitled article"}
                  description={item().description ?? undefined}
                >
                  <p lang={item().languageCode} class={`${styles.panel} ${styles.reader}`}>
                    {item().sourceText}
                  </p>
                </Page>
              </Show>
            )}
          </Show>
        </>
      }
    >
      {(sample) => (
        <Page title={sample().title} description="Sample article">
          <p lang={sample().language} class={`${styles.panel} ${styles.reader}`}>
            {sample().text}
          </p>
        </Page>
      )}
    </Show>
  );
}
