import type { ItemVocabulary, VocabularyAnnotation } from "../../api/client";
import type { Run } from "./transcript";

export function verifiedAnnotations(source: string, view?: ItemVocabulary): VocabularyAnnotation[] {
  if (view?.sourceText !== source) return [];
  const words = new Map(view.items.map(({ id, word }) => [id, word]));
  const annotations = view.annotations ?? [];
  let offset = 0;
  for (const { start, end, vocabularyId } of annotations) {
    if (
      !Number.isSafeInteger(start) ||
      !Number.isSafeInteger(end) ||
      start < offset ||
      end <= start ||
      end > source.length ||
      source.slice(start, end) !== words.get(vocabularyId)
    )
      return [];
    offset = end;
  }
  return annotations;
}

export function sameAnnotations(left: VocabularyAnnotation[], right: VocabularyAnnotation[]) {
  return (
    left.length === right.length &&
    left.every(
      ({ start, end, vocabularyId }, i) =>
        start === right[i].start && end === right[i].end && vocabularyId === right[i].vocabularyId,
    )
  );
}

export type TextPart = { text: string; vocabularyId?: string };

// Split only the text within each existing seek target or gap. A word may span
// multiple runs, and their concatenation must still be exactly the source.
export function annotateRuns(runs: Run[], annotations: VocabularyAnnotation[]): TextPart[][] {
  let offset = 0;
  let i = 0;
  return runs.map(({ text }) => {
    const end = offset + text.length;
    const parts: TextPart[] = [];
    let cursor = offset;
    while (i < annotations.length && annotations[i].start < end) {
      const match = annotations[i];
      const from = Math.max(offset, match.start);
      const to = Math.min(end, match.end);
      if (cursor < from) parts.push({ text: text.slice(cursor - offset, from - offset) });
      parts.push({
        text: text.slice(from - offset, to - offset),
        vocabularyId: match.vocabularyId,
      });
      cursor = to;
      if (match.end > end) break;
      i++;
    }
    if (cursor < end) parts.push({ text: text.slice(cursor - offset) });
    offset = end;
    return parts;
  });
}
