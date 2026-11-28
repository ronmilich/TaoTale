import { For, Show, createEffect } from "solid-js";
import type { Topic, VocabularyEntry } from "../../api/client";
import styles from "./Vocabulary.module.css";
import { WordStatus, WordTopics } from "./WordBadges";

export default function VocabularyTable(props: {
  words: VocabularyEntry[];
  topics?: Topic[];
  selected: string[];
  disabled: boolean;
  changeSelection: (ids: string[]) => void;
  open: (id: string) => void;
}) {
  let selectAll!: HTMLInputElement;
  const all = () => props.selected.length === props.words.length && props.words.length > 0;
  createEffect(
    () => props.selected.length > 0 && !all(),
    (some) => {
      selectAll.indeterminate = some;
    },
  );
  return (
    <table class={styles.table} aria-label="Vocabulary entries">
      <thead>
        <tr>
          <th scope="col">
            <input
              ref={selectAll}
              type="checkbox"
              aria-label="Select all words on this page"
              disabled={props.disabled}
              checked={all()}
              onChange={(event) =>
                props.changeSelection(
                  event.currentTarget.checked ? props.words.map((word) => word.id) : [],
                )
              }
            />
          </th>
          <th scope="col">Word</th>
          <th scope="col">Reading</th>
          <th scope="col">Meaning</th>
          <th scope="col">Status</th>
          <th scope="col">Topics</th>
        </tr>
      </thead>
      <tbody>
        <For each={props.words} keyed={(word) => word.id}>
          {(word) => {
            const checked = () => props.selected.includes(word().id);
            return (
              <tr data-selected={checked() ? "true" : undefined}>
                <td>
                  <input
                    type="checkbox"
                    aria-label={`Select ${word().word}`}
                    checked={checked()}
                    disabled={props.disabled}
                    onChange={(event) =>
                      props.changeSelection(
                        event.currentTarget.checked
                          ? [...props.selected, word().id]
                          : props.selected.filter((id) => id !== word().id),
                      )
                    }
                  />
                </td>
                <th scope="row">
                  <button
                    type="button"
                    class={[styles.wordLink, styles.word]}
                    lang={word().languageCode}
                    disabled={props.disabled}
                    onClick={() => props.open(word().id)}
                  >
                    {word().word}
                  </button>
                </th>
                <td>
                  <Show when={word().reading} fallback="—">
                    {word().reading}
                  </Show>
                </td>
                <td>
                  <Show when={word().meaning} fallback="—">
                    {word().meaning}
                  </Show>
                </td>
                <td>
                  <WordStatus status={word().status} />
                </td>
                <td>
                  <WordTopics ids={word().topicIds} topics={props.topics} />
                </td>
              </tr>
            );
          }}
        </For>
      </tbody>
    </table>
  );
}
