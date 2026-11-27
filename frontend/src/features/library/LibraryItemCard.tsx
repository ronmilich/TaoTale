import { For, Show, createMemo, createSignal } from "solid-js";
import {
  ApiError,
  coverURL,
  deleteItem,
  downloadItemAudio,
  setItemFavorite,
  type Item,
  type Topic,
} from "../../api/client";
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
  deleted: (id: string) => void;
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
  const [deleting, setDeleting] = createSignal(false);
  const [deleteError, setDeleteError] = createSignal(false);
  const [downloading, setDownloading] = createSignal(false);
  const [downloadError, setDownloadError] = createSignal("");
  const recording = () =>
    !draft() && props.item.audioStatus === "READY" ? props.item.audioAssetId : null;

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

  async function remove() {
    if (
      deleting() ||
      !window.confirm(
        `Delete “${title()}”? The article and any recording will be removed. Linked vocabulary entries will remain.`,
      )
    )
      return;
    const id = props.item.id;
    setDeleteError(false);
    setDeleting(true);
    try {
      await deleteItem(id);
      props.deleted(id);
    } catch (cause) {
      if (cause instanceof ApiError && cause.code === "ITEM_NOT_FOUND") props.deleted(id);
      else setDeleteError(true);
    } finally {
      setDeleting(false);
    }
  }

  async function download() {
    const asset = recording();
    if (!asset || downloading() || deleting()) return;
    const id = props.item.id;
    setDownloadError("");
    setDownloading(true);
    try {
      const audio = await downloadItemAudio(id, asset);
      const url = URL.createObjectURL(audio);
      const link = document.createElement("a");
      link.href = url;
      link.download = `taotale-${id}.mp3`;
      link.click();
      window.setTimeout(() => URL.revokeObjectURL(url), 1_000);
    } catch (cause) {
      const unavailable =
        cause instanceof ApiError &&
        ["AUDIO_UNAVAILABLE", "AUDIO_NOT_FOUND", "ITEM_NOT_FOUND"].includes(cause.code);
      setDownloadError(
        unavailable ? "Audio unavailable." : "Could not download the audio. Try again.",
      );
    } finally {
      setDownloading(false);
    }
  }

  return (
    <article class={styles.card}>
      <div class={styles.cover} aria-hidden="true">
        <Show when={props.item.coverImageId} keyed>
          {(id) => (
            <img
              src={coverURL(id)}
              alt=""
              loading="lazy"
              decoding="async"
              onError={(event) => (event.currentTarget.hidden = true)}
            />
          )}
        </Show>
      </div>
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
            disabled={pending() || deleting()}
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
          <Show when={!draft() && !recording()}>
            <p>Audio unavailable</p>
          </Show>
          <div class={styles.cardActions}>
            <Show when={!draft()}>
              <a class={controls.secondary} href={`${href()}/edit`}>
                Edit article
              </a>
            </Show>
            <Show when={recording()}>
              <button
                type="button"
                class={controls.secondary}
                disabled={downloading() || deleting()}
                onClick={() => void download()}
              >
                {downloading() ? "Downloading…" : "Download Audio"}
              </button>
            </Show>
            <button
              type="button"
              class={[controls.secondary, styles.delete]}
              aria-label={`Delete ${title()}`}
              disabled={deleting()}
              onClick={() => void remove()}
            >
              {deleting() ? "Deleting…" : "Delete"}
            </button>
          </div>
          <Show when={downloadError()}>
            <p class={styles.actionError} role="alert">
              {downloadError()}
            </p>
          </Show>
          <Show when={deleteError()}>
            <p class={styles.actionError} role="alert">
              Could not delete this article. Try again.
            </p>
          </Show>
        </div>
      </div>
    </article>
  );
}
