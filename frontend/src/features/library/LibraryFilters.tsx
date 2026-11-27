import { For, createUniqueId } from "solid-js";
import type { Topic } from "../../api/client";
import TopicChoices from "../topics/TopicChoices";
import styles from "./Library.module.css";

const selectFilters = [
  {
    name: "sort",
    label: "Sort",
    options: [
      ["", "Newest"],
      ["oldest", "Oldest"],
      ["title_asc", "Title A–Z"],
      ["title_desc", "Title Z–A"],
    ],
  },
  {
    name: "difficulty",
    label: "Difficulty",
    options: [
      ["", "All difficulties"],
      ["unspecified", "Unspecified"],
      ["beginner", "Beginner"],
      ["intermediate", "Intermediate"],
      ["advanced", "Advanced"],
    ],
  },
  {
    name: "status",
    label: "Status",
    options: [
      ["", "All statuses"],
      ["draft", "Draft"],
      ["published", "Published"],
    ],
  },
] as const;

export default function LibraryFilters(props: {
  search: string;
  filters: URLSearchParams;
  topics?: Topic[];
  searchInput: (value: string) => string;
  change: (key: string, value: string) => void;
}) {
  const id = createUniqueId();
  return (
    <section class={styles.filters} aria-label="Library filters">
      <div class={styles.filterRow}>
        <label>
          Search titles
          <input
            type="search"
            value={props.search}
            onInput={(event) => {
              const input = event.currentTarget;
              const bounded = props.searchInput(input.value);
              if (bounded !== input.value) input.value = bounded;
            }}
          />
        </label>
        <For each={selectFilters}>
          {({ name, label, options }) => (
            <div class={styles.filterField}>
              <label for={`${id}-${name}`}>{label}</label>
              <select
                id={`${id}-${name}`}
                onChange={(event) => props.change(name, event.currentTarget.value)}
              >
                <For each={options}>
                  {([value, text]) => (
                    <option value={value} selected={(props.filters.get(name) ?? "") === value}>
                      {text}
                    </option>
                  )}
                </For>
              </select>
            </div>
          )}
        </For>
        <label class={styles.favoriteFilter}>
          <input
            type="checkbox"
            checked={props.filters.has("favoritesOnly")}
            onChange={(event) =>
              props.change("favoritesOnly", event.currentTarget.checked ? "true" : "")
            }
          />
          Favorites only
        </label>
      </div>
      <details>
        <summary>
          Topics
          {props.filters.has("topics") && ` (${props.filters.get("topics")?.split(",").length})`}
        </summary>
        <TopicChoices
          topics={props.topics}
          selected={props.filters.get("topics")?.split(",") ?? []}
          change={(ids) => props.change("topics", ids.join(","))}
        />
      </details>
    </section>
  );
}
