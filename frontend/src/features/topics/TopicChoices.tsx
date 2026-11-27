import { For } from "solid-js";
import type { Topic } from "../../api/client";
import TopicChip from "./TopicChip";
import styles from "./Topics.module.css";

export default function TopicChoices(props: {
  topics?: Topic[];
  selected: readonly string[];
  limit?: number;
  change: (ids: string[]) => void;
}) {
  return (
    <div class={styles.choices}>
      <For each={props.topics}>
        {(topic) => {
          const checked = () => props.selected.includes(topic.id);
          return (
            <label>
              <input
                type="checkbox"
                checked={checked()}
                disabled={
                  !checked() && props.limit !== undefined && props.selected.length >= props.limit
                }
                onChange={(event) =>
                  props.change(
                    event.currentTarget.checked
                      ? [...props.selected, topic.id]
                      : props.selected.filter((id) => id !== topic.id),
                  )
                }
              />
              <TopicChip name={topic.name} colors={topic.colors} />
            </label>
          );
        }}
      </For>
    </div>
  );
}
