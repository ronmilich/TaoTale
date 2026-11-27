type Run = { text: string; segmentIndex?: number };

/** Reconstruct the authored string using the stored UTF-16 boundaries. */
export function transcriptRuns(source: string, transcript: unknown): Run[] | undefined {
  if (!source.isWellFormed() || !transcript || typeof transcript !== "object") return;
  const { durationMs, segments } = transcript as { durationMs: number; segments: unknown };
  if (
    !Number.isSafeInteger(durationMs) ||
    durationMs <= 0 ||
    !Array.isArray(segments) ||
    !segments.length
  )
    return;

  const runs: Run[] = [];
  let offset = 0;
  let time = 0;
  for (let index = 0; index < segments.length; index++) {
    const segment = segments[index];
    if (!segment || typeof segment !== "object") return;
    const { textStart, textEnd, text, startMs, endMs } = segment;
    if (
      segment.index !== index ||
      !Number.isSafeInteger(textStart) ||
      !Number.isSafeInteger(textEnd) ||
      !Number.isSafeInteger(startMs) ||
      !Number.isSafeInteger(endMs) ||
      textStart < offset ||
      textEnd <= textStart ||
      textEnd > source.length ||
      startMs < time ||
      endMs <= startMs ||
      endMs > durationMs
    )
      return;
    const gap = source.slice(offset, textStart);
    const exact = source.slice(textStart, textEnd);
    // A split surrogate cannot be part of a valid run. Gaps can contain
    // punctuation or whitespace but must not drop speech-bearing characters.
    if (!exact.isWellFormed() || exact !== text || /[\p{L}\p{N}]/u.test(gap)) return;
    if (gap) runs.push({ text: gap });
    runs.push({ text: exact, segmentIndex: index });
    offset = textEnd;
    time = endMs;
  }
  const tail = source.slice(offset);
  if (/[\p{L}\p{N}]/u.test(tail)) return;
  if (tail) runs.push({ text: tail });
  return runs;
}
