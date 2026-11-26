import type { RouteProps } from "@solidjs/router";
import { createMemo, Show } from "solid-js";
import Page, { MissingPage } from "../../../components/Page";
import styles from "../../../components/Page.module.css";
import { samples } from "../../../features/library/samples";

export default function Reader(props: RouteProps<"/items/:id">) {
  const article = createMemo(() =>
    Object.values(samples).find((entry) => entry.id === props.params.id),
  );
  return (
    <Show when={article()} fallback={<MissingPage />}>
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
