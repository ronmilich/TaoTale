import { For, Show, createUniqueId } from "solid-js";
import TopicChoices from "../topics/TopicChoices";
import type { useTopics } from "../topics/topics";
import { metadata, topicLimit, type WordValues } from "./words";
import styles from "./Vocabulary.module.css";

export default function WordFields(props: {
  language: string;
  values: WordValues;
  topics: ReturnType<typeof useTopics>;
  readonlyWord?: boolean;
  disabled: boolean;
  change: <K extends keyof WordValues>(key: K, value: WordValues[K]) => void;
}) {
  const id = createUniqueId();

  return (
    <fieldset class={styles.fields} disabled={props.disabled}>
      <div class={styles.field}>
        <label for={`${id}-word`}>Word or phrase</label>
        <textarea
          id={`${id}-word`}
          lang={props.language}
          rows={Math.min(4, props.values.word.split("\n").length)}
          value={props.values.word}
          readonly={props.readonlyWord}
          required
          onInput={(event) => props.change("word", event.currentTarget.value)}
        />
      </div>
      <div class={styles.fieldGrid}>
        <For each={metadata}>
          {({ name, label }) => (
            <div class={styles.field}>
              <label for={`${id}-${name}`}>{label}</label>
              <Show
                when={name === "note" || name === "meaning"}
                fallback={
                  <input
                    id={`${id}-${name}`}
                    value={props.values[name]}
                    onInput={(event) => props.change(name, event.currentTarget.value)}
                  />
                }
              >
                <textarea
                  id={`${id}-${name}`}
                  rows={2}
                  value={props.values[name]}
                  onInput={(event) => props.change(name, event.currentTarget.value)}
                />
              </Show>
            </div>
          )}
        </For>
      </div>
      <fieldset class={styles.topicFields}>
        <legend>Topics</legend>
        <Show when={props.topics.failed()}>
          <p role="alert">Could not load Topics.</p>
          <button type="button" onClick={() => void props.topics.reload()}>
            Retry topics
          </button>
        </Show>
        <Show when={!props.topics.items() && !props.topics.failed()}>
          <p role="status">Loading Topics…</p>
        </Show>
        <TopicChoices
          topics={props.topics.items()}
          selected={props.values.topicIds}
          limit={topicLimit}
          change={(ids) => props.change("topicIds", ids)}
        />
      </fieldset>
    </fieldset>
  );
}
