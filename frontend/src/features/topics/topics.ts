import { createSignal, onSettled } from "solid-js";
import { apiGet, type Topic } from "../../api/client";

export const topicPresets = {
  iris: ["#ECE7FA", "#554095", "#3E345C", "#E4D8FF"],
  periwinkle: ["#E7EAFE", "#444D9E", "#31385C", "#DCE0FF"],
  orchid: ["#F6E6F3", "#813E7B", "#513450", "#F4C8EC"],
  sage: ["#E1F0E9", "#286449", "#294C3F", "#B9E9D2"],
  amber: ["#FFF0DA", "#835921", "#54422B", "#F8DEB1"],
  slate: ["#EDECF3", "#59566B", "#3A3749", "#DEDAED"],
} as const;

export type TopicPreset = keyof typeof topicPresets;

export function palette([
  background,
  text,
  darkBackground,
  darkText,
]: readonly string[]): Topic["colors"] {
  return { light: { background, text }, dark: { background: darkBackground, text: darkText } };
}

export function useTopics() {
  const [items, setItems] = createSignal<Topic[]>();
  const [failed, setFailed] = createSignal(false);
  let request: AbortController | undefined;
  async function reload() {
    request?.abort();
    const controller = new AbortController();
    request = controller;
    setFailed(false);
    try {
      const result = await apiGet(
        "/topics",
        AbortSignal.any([controller.signal, AbortSignal.timeout(15_000)]),
      );
      if (!controller.signal.aborted) setItems(result.items);
    } catch {
      if (!controller.signal.aborted) setFailed(true);
    }
  }
  onSettled(() => {
    void reload();
    return () => request?.abort();
  });
  function save(topic: Topic) {
    setItems((current) =>
      [...(current ?? []).filter((entry) => entry.id !== topic.id), topic].sort((a, b) =>
        a.name.localeCompare(b.name),
      ),
    );
  }
  function remove(id: string) {
    setItems((current) => current?.filter((topic) => topic.id !== id));
  }
  return { items, failed, reload, save, remove };
}
