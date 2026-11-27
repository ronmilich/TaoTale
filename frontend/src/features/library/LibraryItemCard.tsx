import { For, Show, createMemo, createSignal } from "solid-js";
import { setItemFavorite, type Item, type Topic } from "../../api/client";
import controls from "../../components/Controls.module.css";
import TopicChip from "../topics/TopicChip";
import styles from "./Library.module.css";

const difficultyLabels = {
  beginner: "Beginner",
  intermediate: "Intermediate",
  advanced: "Advanced",
};
const dateFormat = new Intl.DateTimeFormat(undefined, { dateStyle: "medium" });

export default function LibraryItemCard(props: {
  item: Item;
  topics?: Topic[];
  favoriteChanged: () => void;
}) {
  const draft = () => props.item.status === "draft";
  const title = () => props.item.title?.trim() || (draft() ? "Untitled draft" : "Untitled article");
  const href = () => (draft() ? `/create?draftId=${props.item.id}` : `/items/${props.item.id}`);
  const difficulty = () =>
    props.item.difficulty ? difficultyLabels[props.item.difficulty] : "Unspecified";
  const date = createMemo(() =>
    draft() ? props.item.updatedAt : (props.item.publishedAt ?? props.item.createdAt),
  );
  const tags = createMemo(
    () => props.topics?.filter((topic) => props.item.topicIds.includes(topic.id)) ?? [],
  );
  const [confirmed, setConfirmed] = createSignal<boolean>();
  const favorite = () => confirmed() ?? props.item.isFavorite;
  const [pending, setPending] = createSignal(false);
  const [favoriteError, setFavoriteError] = createSignal(false);

  async function toggleFavorite() {
    if (pending()) return;
    setFavoriteError(false);
    setPending(true);
    try {
      const result = await setItemFavorite(props.item.id, !favorite());
      setConfirmed(result.isFavorite);
      props.favoriteChanged();
    } catch {
      setFavoriteError(true);
    } finally {
      setPending(false);
    }
  }

  return (
    <article class={styles.card}>
      <div class={styles.cover} aria-hidden="true" />
      <div class={styles.details}>
        <div class={styles.tags}>
          <span class={draft() ? styles.draftBadge : styles.publishedBadge}>
            {draft() ? "Draft" : "Published"}
          </span>
          <span>{difficulty()}</span>
          <button
            type="button"
            class={[controls.secondary, styles.star]}
            aria-label={favorite() ? "Remove from Favorites" : "Favorite"}
            aria-pressed={favorite() ? "true" : "false"}
            disabled={pending()}
            onClick={() => void toggleFavorite()}
          >
            <span aria-hidden="true">{favorite() ? "★" : "☆"}</span>
          </button>
        </div>
        <h2 class={styles.cardTitle}>{title()}</h2>
        <Show when={tags().length}>
          <div class={styles.tags}>
            <For each={tags().slice(0, 2)}>
              {(topic) => <TopicChip name={topic.name} colors={topic.colors} />}
            </For>
            <Show when={tags().length > 2}>
              <span aria-label={`${tags().length - 2} more topics`}>+{tags().length - 2}</span>
            </Show>
          </div>
        </Show>
        <Show when={pending()}>
          <p role="status">Updating favorite…</p>
        </Show>
        <Show when={favoriteError()}>
          <p role="alert">Could not update favorite. Try again.</p>
        </Show>
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
