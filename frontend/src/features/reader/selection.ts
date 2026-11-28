import type { Language } from "../../preferences/storage";
import { wordLimit } from "../vocabulary/words";

export type TranscriptSelection = { start: number; end: number; text: string };

export function readTranscriptSelection(
  transcript: HTMLElement,
  source: string,
  selection: Selection | null,
): TranscriptSelection | undefined {
  if (!selection || selection.rangeCount !== 1 || selection.isCollapsed) return;
  const range = selection.getRangeAt(0);
  if (!transcript.contains(range.startContainer) || !transcript.contains(range.endContainer))
    return;

  // Range.toString counts text across every span and gap in UTF-16, like source.slice.
  const prefix = range.cloneRange();
  prefix.setStart(transcript, 0);
  prefix.setEnd(range.startContainer, range.startOffset);
  const offset = prefix.toString().length;
  const raw = range.toString();
  if (source.slice(offset, offset + raw.length) !== raw) return;
  const text = raw.trim();
  if (!text) return;
  const start = offset + raw.length - raw.trimStart().length;
  return { start, end: start + text.length, text };
}

export function selectionError(text: string, language: Language) {
  if (!text.isWellFormed() || text.includes("\0")) return "Select complete characters.";
  if ([...text].length > wordLimit) return "Select up to 120 characters.";
  if (language === "zh-CN" && /[\r\n\u2028\u2029，。！？；]/u.test(text))
    return "Select a Mandarin word or phrase within one line and without sentence punctuation.";
  return "";
}
