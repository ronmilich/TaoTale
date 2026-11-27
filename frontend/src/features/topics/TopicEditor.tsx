import { For, Show, createSignal, createUniqueId } from "solid-js";
import { ApiError, createTopic, patchTopic, type Topic } from "../../api/client";
import controls from "../../components/Controls.module.css";
import TopicChip from "./TopicChip";
import styles from "./Topics.module.css";
import { palette, topicPresets, type TopicPreset } from "./topics";

const presets = Object.keys(topicPresets) as TopicPreset[];

export default function TopicEditor(props: {
  topic?: Topic;
  saved: (topic: Topic) => void;
  cancel: () => void;
}) {
  const id = createUniqueId();
  const [name, setName] = createSignal(() => props.topic?.name ?? "");
  const [color, setColor] = createSignal<TopicPreset | undefined>(() =>
    props.topic
      ? presets.find((preset) => {
          const existing = props.topic?.colors;
          const [background, text, darkBackground, darkText] = topicPresets[preset];
          return (
            existing?.light.background === background &&
            existing.light.text === text &&
            existing.dark.background === darkBackground &&
            existing.dark.text === darkText
          );
        })
      : "iris",
  );
  const [saving, setSaving] = createSignal(false);
  const [message, setMessage] = createSignal("");

  async function submit() {
    if (saving()) return;
    const title = name().trim();
    if (!title || [...title].length > 40) {
      setMessage("Enter a topic name of 1 to 40 characters.");
      return;
    }
    setSaving(true);
    setMessage("");
    try {
      const topic = props.topic;
      const preset = color();
      const result = topic
        ? await patchTopic(topic.id, { name: title, colorPreset: preset })
        : await createTopic({ name: title, colorPreset: preset ?? "iris" });
      props.saved(result);
    } catch (error) {
      setMessage(
        error instanceof ApiError && error.code === "DUPLICATE_TOPIC"
          ? "A topic already has that name."
          : "Could not save the topic. Try again.",
      );
    } finally {
      setSaving(false);
    }
  }

  return (
    <fieldset class={styles.editor} disabled={saving()}>
      <legend>{props.topic ? "Edit topic" : "New topic"}</legend>
      <label for={id}>Topic name</label>
      <input
        id={id}
        value={name()}
        onInput={(event) => setName(event.currentTarget.value)}
        onKeyDown={(event) => {
          if (event.key === "Enter") {
            event.preventDefault();
            void submit();
          }
        }}
      />
      <fieldset class={styles.swatches}>
        <legend>Topic color</legend>
        <For each={presets}>
          {(preset) => (
            <label>
              <input
                type="radio"
                name={id}
                checked={color() === preset}
                onChange={() => setColor(preset)}
              />
              <TopicChip
                name={preset[0].toUpperCase() + preset.slice(1)}
                colors={palette(topicPresets[preset])}
              />
            </label>
          )}
        </For>
      </fieldset>
      <Show when={message()}>{(error) => <p role="alert">{error()}</p>}</Show>
      <div class={styles.buttons}>
        <button type="button" class={controls.primary} onClick={() => void submit()}>
          {saving() ? "Saving topic…" : "Save topic"}
        </button>
        <button type="button" class={controls.secondary} onClick={() => props.cancel()}>
          Cancel
        </button>
      </div>
    </fieldset>
  );
}
