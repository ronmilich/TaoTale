import { onSettled } from "solid-js";
import type { VocabularyEntry, VocabularyPatch, VocabularyStatus } from "../../api/client";
import type { Language } from "../../preferences/storage";

export const wordLimit = 120;
export const topicLimit = 10;
export const statuses = ["LEARNING", "ACQUIRED"] as const satisfies readonly VocabularyStatus[];
export const statusLabel: Record<VocabularyStatus, string> = {
  LEARNING: "Learning",
  ACQUIRED: "Acquired",
};
export const otherStatus = (status: VocabularyStatus): VocabularyStatus =>
  status === "LEARNING" ? "ACQUIRED" : "LEARNING";

export const metadata = [
  { name: "reading", label: "Reading", limit: 160 },
  { name: "readingSystem", label: "Reading system", limit: 40 },
  { name: "meaning", label: "Meaning", limit: 300 },
  { name: "note", label: "Note", limit: 5000 },
] as const;

type TextField = "word" | (typeof metadata)[number]["name"];
export type WordValues = Record<TextField, string> & { topicIds: string[] };

export function wordValues(entry?: VocabularyEntry): WordValues {
  return {
    word: entry?.word ?? "",
    reading: entry?.reading ?? "",
    readingSystem: entry?.readingSystem ?? "",
    meaning: entry?.meaning ?? "",
    note: entry?.note ?? "",
    topicIds: entry?.topicIds ?? [],
  };
}

export function wordError(values: WordValues) {
  const word = values.word.trim();
  if (!word || [...word].length > wordLimit)
    return "Enter a word or phrase of up to 120 characters.";
  for (const { name, label, limit } of [
    { name: "word", label: "Word", limit: wordLimit },
    ...metadata,
  ] as const) {
    const value = name === "word" ? word : values[name];
    if (!value.isWellFormed() || value.includes("\0"))
      return `${label} contains unsupported characters.`;
    if ([...value].length > limit)
      return `${label} can contain up to ${limit.toLocaleString("en")} characters.`;
  }
  return "";
}

export function wordPatch(values: WordValues, baseline: WordValues): VocabularyPatch {
  const patch: VocabularyPatch = {};
  if (values.word.trim() !== baseline.word) patch.word = values.word.trim();
  for (const { name } of metadata) {
    if (values[name] !== baseline[name]) patch[name] = values[name] || null;
  }
  if (
    values.topicIds.length !== baseline.topicIds.length ||
    values.topicIds.some((id) => !baseline.topicIds.includes(id))
  ) {
    patch.topicIds = values.topicIds;
  }
  return patch;
}

// Other open views read their own projections after a confirmed write.
export function useVocabularyChanges(receive: (language: Language) => void) {
  let channel: BroadcastChannel | undefined;
  onSettled(() => {
    if (typeof BroadcastChannel === "undefined") return;
    channel = new BroadcastChannel("taotale.vocabulary");
    channel.onmessage = ({ data }) => {
      if (data === "zh-CN" || data === "ko-KR") receive(data);
    };
    return () => {
      channel?.close();
      channel = undefined;
    };
  });
  return (language: Language) => channel?.postMessage(language);
}
