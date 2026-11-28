import { For, Show, createUniqueId } from "solid-js";
import controls from "../../components/Controls.module.css";
import TopicChoices from "../topics/TopicChoices";
import type { useTopics } from "../topics/topics";
import { sorts } from "./query";
import styles from "./Vocabulary.module.css";
import { statusLabel, statuses, topicLimit } from "./words";

export default function VocabularyFilters(props: {
  filters: URLSearchParams;
  search: string;
  searchInput: (value: string) => string;
  change: (key: string, value: string) => void;
  clear: () => void;
  topics: ReturnType<typeof useTopics>;
}) {
  const id = createUniqueId();
  const selectedTopics = () => props.filters.get("topics")?.split(",") ?? [];
  return (
    <>
      <div class={styles.filters}>
        <div class={styles.field}>
          <label for={`${id}-search`}>Search vocabulary</label>
          <input
            id={`${id}-search`}
            type="search"
            value={props.search}
            placeholder="Word, reading, or meaning"
            onInput={(event) => {
              event.currentTarget.value = props.searchInput(event.currentTarget.value);
            }}
          />
        </div>
        <div class={styles.field}>
          <label for={`${id}-status`}>Status</label>
          <select
            id={`${id}-status`}
            value={props.filters.get("status") ?? ""}
            onChange={(event) => props.change("status", event.currentTarget.value)}
          >
            <option value="" selected={!props.filters.has("status")}>
              All statuses
            </option>
            <For each={statuses}>
              {(status) => (
                <option value={status} selected={props.filters.get("status") === status}>
                  {statusLabel[status]}
                </option>
              )}
            </For>
          </select>
        </div>
        <div class={styles.field}>
          <label for={`${id}-sort`}>Sort by</label>
          <select
            id={`${id}-sort`}
            value={props.filters.get("sort") ?? ""}
            onChange={(event) => props.change("sort", event.currentTarget.value)}
          >
            <For each={sorts}>
              {(option) => (
                <option
                  value={option.value}
                  selected={(props.filters.get("sort") ?? "") === option.value}
                >
                  {option.label}
                </option>
              )}
            </For>
          </select>
        </div>
      </div>
      <div class={styles.controls}>
        <details class={styles.topicsFilter}>
          <summary>
            Filter by Topics{selectedTopics().length ? ` (${selectedTopics().length})` : ""}
          </summary>
          <Show when={props.topics.failed()}>
            <p role="alert" class={styles.error}>
              Could not load Topics.
            </p>
            <button
              type="button"
              class={controls.secondary}
              onClick={() => void props.topics.reload()}
            >
              Retry topics
            </button>
          </Show>
          <TopicChoices
            topics={props.topics.items()}
            selected={selectedTopics()}
            limit={topicLimit}
            change={(ids) => props.change("topics", ids.join(","))}
          />
        </details>
        <Show when={props.filters.size || props.search}>
          <button type="button" class={controls.secondary} onClick={() => props.clear()}>
            Clear filters
          </button>
        </Show>
      </div>
    </>
  );
}
