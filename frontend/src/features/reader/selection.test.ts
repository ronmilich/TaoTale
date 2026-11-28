import { expect, test } from "vitest";
import { readTranscriptSelection, selectionError } from "./selection";

test("measures trimmed DOM selections across timed spans and exact gaps in UTF-16", () => {
  const transcript = document.createElement("div");
  const first = document.createElement("span");
  const second = document.createElement("span");
  const gap = document.createTextNode("\r\n  ");
  first.textContent = "前言𠀀 한국어？";
  second.textContent = "다음 문장😀 ";
  transcript.append(first, gap, second);
  document.body.append(transcript);
  const source = transcript.textContent!;
  const range = document.createRange();
  range.setStart(first.firstChild!, 4);
  range.setEnd(second.firstChild!, second.textContent.length);
  const selection = window.getSelection()!;
  selection.removeAllRanges();
  selection.addRange(range);
  try {
    const text = "한국어？\r\n  다음 문장😀";
    expect(readTranscriptSelection(transcript, source, selection)).toEqual({
      start: 5,
      end: 5 + text.length,
      text,
    });
    expect(
      readTranscriptSelection(transcript, source.replace("한국어", "잘못된"), selection),
    ).toBeUndefined();
    range.setStart(document.body, 0);
    expect(readTranscriptSelection(transcript, source, selection)).toBeUndefined();
  } finally {
    selection.removeAllRanges();
    transcript.remove();
  }
});

test("applies Mandarin selection rules while allowing Korean phrases and complete supplementary characters", () => {
  expect(selectionError("词、𠀀", "zh-CN")).toBe("");
  expect(selectionError("你好。朋友", "zh-CN")).toContain("sentence punctuation");
  expect(selectionError("你好\n朋友", "zh-CN")).toContain("within one line");
  expect(selectionError("한글？\n 다음 문장", "ko-KR")).toBe("");
  expect(selectionError("𠀀".repeat(120), "ko-KR")).toBe("");
  expect(selectionError("𠀀".repeat(121), "ko-KR")).toContain("120");
  expect(selectionError("\ud840", "ko-KR")).toContain("complete characters");
});
