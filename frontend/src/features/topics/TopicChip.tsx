import type { Topic } from "../../api/client";
import styles from "./Topics.module.css";

export default function TopicChip(props: { name: string; colors: Topic["colors"] }) {
  return (
    <span
      class={styles.chip}
      style={{
        "--topic-light-background": props.colors.light.background,
        "--topic-light-text": props.colors.light.text,
        "--topic-dark-background": props.colors.dark.background,
        "--topic-dark-text": props.colors.dark.text,
      }}
    >
      {props.name}
    </span>
  );
}
