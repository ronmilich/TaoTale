import { For, Show, createSignal } from "solid-js";
import { deleteTopic, type Topic } from "../../api/client";
import controls from "../../components/Controls.module.css";
import TopicChip from "./TopicChip";
import TopicEditor from "./TopicEditor";
import styles from "./Topics.module.css";
import { useTopics } from "./topics";

export default function TopicManager(props: {
  topics: ReturnType<typeof useTopics>;
  removed: (id: string) => void;
}) {
  const [edit, setEdit] = createSignal<Topic | "new">();
  const [deleting, setDeleting] = createSignal(false);
  const [error, setError] = createSignal("");

  async function remove(topic: Topic) {
    const words = topic.vocabularyUsageCount ?? 0;
    const impact = `${topic.usageCount} ${topic.usageCount === 1 ? "item" : "items"} and ${words} ${words === 1 ? "word" : "words"}`;
    if (
      deleting() ||
      !window.confirm(`Delete "${topic.name}"? This will remove the topic from ${impact}.`)
    )
      return;
    setDeleting(true);
    setError("");
    try {
      await deleteTopic(topic.id);
      props.topics.remove(topic.id);
      props.removed(topic.id);
    } catch {
      setError("Could not delete the topic. Try again.");
    } finally {
      setDeleting(false);
    }
  }

  return (
    <details class={styles.manager}>
      <summary>Manage topics</summary>
      <Show when={props.topics.failed()}>
        <p role="alert">Could not load topics.</p>
        <button type="button" onClick={() => void props.topics.reload()}>
          Retry topics
        </button>
      </Show>
      <Show when={error()}>{(message) => <p role="alert">{message()}</p>}</Show>
      <Show
        when={edit()}
        keyed
        fallback={
          <button
            type="button"
            class={controls.secondary}
            disabled={!props.topics.items() || deleting()}
            onClick={() => setEdit("new")}
          >
            Create topic
          </button>
        }
      >
        {(topic) => (
          <TopicEditor
            topic={topic === "new" ? undefined : topic}
            cancel={() => setEdit(undefined)}
            saved={(saved) => {
              props.topics.save(saved);
              setEdit(undefined);
            }}
          />
        )}
      </Show>
      <For each={props.topics.items()}>
        {(topic) => (
          <div class={styles.row}>
            <TopicChip name={topic.name} colors={topic.colors} />
            <button
              type="button"
              class={controls.secondary}
              aria-label={`Edit topic ${topic.name}`}
              disabled={Boolean(edit()) || deleting()}
              onClick={() => setEdit(topic)}
            >
              Edit
            </button>
            <Show when={!topic.isDefault} fallback={<span>Default topic</span>}>
              <button
                type="button"
                class={controls.secondary}
                aria-label={`Delete topic ${topic.name}`}
                disabled={Boolean(edit()) || deleting()}
                onClick={() => void remove(topic)}
              >
                Delete
              </button>
            </Show>
          </div>
        )}
      </For>
    </details>
  );
}
