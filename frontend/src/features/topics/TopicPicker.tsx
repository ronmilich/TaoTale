import { Show, createSignal } from "solid-js";
import controls from "../../components/Controls.module.css";
import TopicChoices from "./TopicChoices";
import TopicEditor from "./TopicEditor";
import styles from "./Topics.module.css";
import { useTopics } from "./topics";

export default function TopicPicker(props: {
  selected: readonly string[];
  change: (ids: string[]) => void;
}) {
  const topics = useTopics();
  const [creating, setCreating] = createSignal(false);
  return (
    <fieldset class={styles.picker}>
      <legend>Topics</legend>
      <Show when={topics.failed()}>
        <p role="alert">Could not load topics.</p>
        <button type="button" onClick={() => void topics.reload()}>
          Retry topics
        </button>
      </Show>
      <Show when={!topics.items() && !topics.failed()}>
        <p role="status">Loading topics…</p>
      </Show>
      <TopicChoices
        topics={topics.items()}
        selected={props.selected}
        change={props.change}
        limit={10}
      />
      <p>{props.selected.length} of 10 selected</p>
      <Show
        when={creating()}
        fallback={
          <button
            type="button"
            class={controls.secondary}
            disabled={!topics.items() || props.selected.length >= 10}
            onClick={() => setCreating(true)}
          >
            Create topic
          </button>
        }
      >
        <TopicEditor
          cancel={() => setCreating(false)}
          saved={(topic) => {
            topics.save(topic);
            if (props.selected.length < 10) props.change([...props.selected, topic.id]);
            setCreating(false);
          }}
        />
      </Show>
    </fieldset>
  );
}
