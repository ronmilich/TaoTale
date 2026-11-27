import type { Item, ItemPatch, LearningLanguage } from "../../api/client";

export type DraftFields = {
  languageCode: LearningLanguage["code"] | "";
  title: string;
  sourceText: string;
  description: string;
  notes: string;
  difficulty: NonNullable<Item["difficulty"]> | "";
  voiceId: string | null;
  speakingRate: Item["speakingRate"];
};

export const textLimits = { title: 120, sourceText: 1500, description: 1000, notes: 5000 };
export type TextField = keyof typeof textLimits;
export const textFields = Object.keys(textLimits) as TextField[];
export const difficulties = [
  { value: "", label: "Unspecified" },
  { value: "beginner", label: "Beginner" },
  { value: "intermediate", label: "Intermediate" },
  { value: "advanced", label: "Advanced" },
] as const;
export const editableFields = [
  { id: "title", label: "Title" },
  { id: "sourceText", label: "Source text" },
  { id: "difficulty", label: "Difficulty" },
  { id: "description", label: "Description" },
  { id: "notes", label: "Notes" },
  { id: "voiceId", label: "Voice" },
  { id: "speakingRate", label: "Speaking rate" },
] as const;

export function emptyDraft(languageCode: DraftFields["languageCode"]): DraftFields {
  return {
    languageCode,
    title: "",
    sourceText: "",
    description: "",
    notes: "",
    difficulty: "",
    voiceId: null,
    speakingRate: null,
  };
}

export function draftFields(item: Item): DraftFields {
  return {
    languageCode: item.languageCode,
    title: item.title ?? "",
    sourceText: item.sourceText,
    description: item.description ?? "",
    notes: item.notes ?? "",
    difficulty: item.difficulty ?? "",
    voiceId: item.voiceId,
    speakingRate: item.speakingRate,
  };
}

export function draftPatch(values: DraftFields, baseline: DraftFields, version: number): ItemPatch {
  const patch: ItemPatch = { expectedVersion: version };
  if (values.title !== baseline.title) patch.title = values.title || null;
  if (values.sourceText !== baseline.sourceText) patch.sourceText = values.sourceText;
  if (values.description !== baseline.description) patch.description = values.description || null;
  if (values.notes !== baseline.notes) patch.notes = values.notes || null;
  if (values.difficulty !== baseline.difficulty) patch.difficulty = values.difficulty || null;
  if (values.voiceId !== baseline.voiceId) patch.voiceId = values.voiceId;
  if (values.speakingRate !== baseline.speakingRate) patch.speakingRate = values.speakingRate;
  return patch;
}

export function textLength(field: TextField, value: string) {
  return field === "sourceText" ? value.length : Array.from(value).length;
}

export function rebaseDraft(mine: DraftFields, previous: DraftFields, latest: DraftFields) {
  const result = { ...latest };
  for (const { id } of editableFields) {
    if (mine[id] !== previous[id]) Object.assign(result, { [id]: mine[id] });
  }
  return result;
}

export function displayedText(text: string) {
  return text.replace(/\r\n?/g, "\n");
}

// Native textareas display CR and CRLF as LF. Translate their offsets back to
// the exact string so edits and pastes can leave untouched line endings intact.
export function replaceTextareaRange(text: string, from: number, to: number, inserted: string) {
  function originalOffset(offset: number) {
    let index = 0;
    for (let i = 0; i < offset; i++, index++) {
      if (text[index] === "\r" && text[index + 1] === "\n") index++;
    }
    return index;
  }
  return text.slice(0, originalOffset(from)) + inserted + text.slice(originalOffset(to));
}

export function applyTextareaEdit(previous: string, next: string) {
  if (!previous.includes("\r")) return next;
  const before = displayedText(previous);
  let start = 0;
  let beforeEnd = before.length;
  let nextEnd = next.length;
  while (start < beforeEnd && start < nextEnd && before[start] === next[start]) start++;
  while (beforeEnd > start && nextEnd > start && before[beforeEnd - 1] === next[nextEnd - 1]) {
    beforeEnd--;
    nextEnd--;
  }
  return replaceTextareaRange(previous, start, beforeEnd, next.slice(start, nextEnd));
}
