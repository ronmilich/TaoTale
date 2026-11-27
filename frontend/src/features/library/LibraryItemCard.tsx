import { Show, createMemo } from "solid-js";
import type { Item } from "../../api/client";
import controls from "../../components/Controls.module.css";
import styles from "./Library.module.css";

const difficultyLabels = {
  beginner: "Beginner",
  intermediate: "Intermediate",
  advanced: "Advanced",
};
const dateFormat = new Intl.DateTimeFormat(undefined, { dateStyle: "medium" });

export default function LibraryItemCard(props: { item: Item }) {
  const draft = () => props.item.status === "draft";
  const title = () => props.item.title?.trim() || (draft() ? "Untitled draft" : "Untitled article");
  const href = () => (draft() ? `/create?draftId=${props.item.id}` : `/items/${props.item.id}`);
  const difficulty = () =>
    props.item.difficulty ? difficultyLabels[props.item.difficulty] : "Unspecified";
  const date = createMemo(() =>
    draft() ? props.item.updatedAt : (props.item.publishedAt ?? props.item.createdAt),
  );

  return (
    <article class={styles.card}>
      <div class={styles.cover} aria-hidden="true" />
      <div class={styles.details}>
        <div class={styles.tags}>
          <span class={draft() ? styles.draftBadge : styles.publishedBadge}>
            {draft() ? "Draft" : "Published"}
          </span>
          <span>{difficulty()}</span>
        </div>
        <h2 class={styles.cardTitle}>{title()}</h2>
        <Show when={props.item.description?.trim()}>
          <p class={styles.excerpt}>{props.item.description}</p>
        </Show>
        <div class={styles.cardFooter}>
          <p>
            {draft() ? "Updated" : "Published"}{" "}
            <time datetime={date()}>{dateFormat.format(new Date(date()))}</time>
          </p>
          <a class={controls.secondary} href={href()}>
            {draft() ? "Continue Editing" : "Open in Reader"} <span aria-hidden="true">→</span>
          </a>
        </div>
      </div>
    </article>
  );
}
