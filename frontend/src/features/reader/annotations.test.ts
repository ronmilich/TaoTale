import { expect, test } from "vitest";
import type { ItemVocabulary } from "../../api/client";
import { annotateRuns, verifiedAnnotations } from "./annotations";

test("preserves exact UTF-16 matches across timed runs, gaps, and the text-only fallback", () => {
  const source = "前𠀀学。\r\n  语言后";
  const range = { vocabularyId: "word", start: 1, end: source.length - 1 };
  const view: ItemVocabulary = {
    sourceText: source,
    annotations: [range],
    items: [
      {
        id: range.vocabularyId,
        word: "𠀀学。\r\n  语言",
        languageCode: "zh-CN",
        status: "LEARNING",
        reading: null,
        readingSystem: null,
        meaning: null,
        note: null,
        topicIds: [],
        createdAt: "",
        updatedAt: "",
        reference: { itemId: "article", addedAt: "", manualLinked: true, autoLinked: false },
      },
    ],
  };
  const annotations = verifiedAnnotations(source, view);
  expect(annotations).toEqual([range]);
  const timed = annotateRuns(
    [
      { text: "前𠀀学。", segmentIndex: 0 },
      { text: "\r\n  " },
      { text: "语言后", segmentIndex: 1 },
    ],
    annotations,
  ).flat();
  expect(timed.map((entry) => entry.text).join("")).toBe(source);
  expect(timed.filter((entry) => entry.vocabularyId).map((entry) => entry.text)).toEqual([
    "𠀀学。",
    "\r\n  ",
    "语言",
  ]);
  expect(annotateRuns([{ text: source }], annotations).flat()).toEqual([
    { text: "前" },
    { text: view.items[0].word, vocabularyId: "word" },
    { text: "后" },
  ]);
  expect(verifiedAnnotations(source.replace("前", "先"), view)).toEqual([]);
  expect(verifiedAnnotations(source, { ...view, annotations: [{ ...range, start: 2 }] })).toEqual(
    [],
  );
  expect(verifiedAnnotations(source, { ...view, annotations: [range, range] })).toEqual([]);
});
