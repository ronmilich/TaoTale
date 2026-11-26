import { Title } from "@solidjs/meta";
import type { ParentProps } from "solid-js";
import styles from "./Page.module.css";

export default function Page(props: ParentProps<{ title: string; description?: string }>) {
  return (
    <section class={styles.page}>
      <Title>{props.title} · TaoTale</Title>
      <h1>{props.title}</h1>
      <p class={styles.description}>{props.description}</p>
      <div class={styles.content}>{props.children}</div>
    </section>
  );
}

export function MissingPage() {
  return (
    <Page title="Page not found" description="This page is unavailable.">
      <a href="/library">Back to Library</a>
    </Page>
  );
}
