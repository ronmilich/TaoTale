import { For } from "solid-js";
import type { Topic, VocabularyStatus } from "../../api/client";
import TopicChip from "../topics/TopicChip";
import { statusLabel } from "./words";
import styles from "./Vocabulary.module.css";

export function WordStatus(props: { status: VocabularyStatus }) {
  return (
    <span class={styles.status} data-status={props.status}>
      {statusLabel[props.status]}
    </span>
  );
}

export function WordTopics(props: { ids: readonly string[]; topics?: Topic[] }) {
  return (
    <div class={styles.chips}>
      <For each={props.topics?.filter((topic) => props.ids.includes(topic.id))}>
        {(topic) => <TopicChip name={topic.name} colors={topic.colors} />}
      </For>
    </div>
  );
}
