// Offline verification of recorded Go SDK events. Takes one or more events.json files.
import assert from 'node:assert/strict';
import { readFileSync, statSync } from 'node:fs';
import { dirname, join } from 'node:path';

const entities = { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&apos;' };

function verify(e, file) {
  assert.equal(e.Requests, 1);
  assert.equal(e.Failure, '');
  if (e.Mode !== 'sample') {
    assert.equal(e.StopTrigger, e.Mode === 'cancel' ? 'synthesis-started' : 'deadline');
    assert.equal(e.Reason, 'Canceled');
    assert.equal(e.CancelReason, 'CancelledByUser');
    assert.equal(e.CancelCode, 'NoError');
    return { mode: e.Mode, reason: e.CancelReason, events: e.Events?.length ?? 0 };
  }
  assert.equal(e.Reason, 'SynthesizingAudioCompleted');
  assert(e.DecodedSamples24kHz > 0);
  const mp3Bytes = statSync(join(dirname(file), 'audio.mp3')).size;
  assert(mp3Bytes > 0);

  // The observed SDK offsets count Unicode code points in the full escaped
  // SSML. Only whole source-code-point boundaries have a UTF-16 mapping.
  let cursor = [...e.Prefix].length;
  let original = 0;
  const offsets = new Map([[cursor, original]]);
  let encoded = '';
  for (const ch of e.Source) {
    const escaped = entities[ch] ?? ch;
    encoded += escaped;
    cursor += [...escaped].length;
    original += ch.length;
    offsets.set(cursor, original);
  }
  assert.equal(e.SSML, e.Prefix + encoded + '</prosody></voice></speak>');
  const ssml = [...e.SSML];
  const covered = new Set();
  let previousEnd = 0, previousTick = 0, wordEndTick = 0;
  for (const b of e.Events) {
    const end = b.TextOffset + b.WordLength;
    const start16 = offsets.get(b.TextOffset), end16 = offsets.get(end);
    assert(Number.isInteger(start16) && Number.isInteger(end16) && end16 > start16);
    assert.equal(ssml.slice(b.TextOffset, end).join(''), b.Text);
    assert(start16 >= previousEnd && b.AudioTicks >= previousTick);
    assert(b.DurationTicks >= 0);
    const endTick = b.AudioTicks + b.DurationTicks;
    assert(endTick * 24000 <= e.DecodedSamples24kHz * 10000000);
    previousEnd = end16;
    previousTick = b.AudioTicks;
    if (b.Type === 0) {
      assert(b.DurationTicks > 0 && b.AudioTicks >= wordEndTick, 'overlapping or empty word timing');
      wordEndTick = endTick;
      for (let i = start16; i < end16; i++) covered.add(i);
    }
  }
  for (const match of e.Source.matchAll(/[\p{L}\p{N}]/gu)) assert(covered.has(match.index), 'missing speech');
  return { locale: e.Locale, rate: e.Rate, events: e.Events.length,
    words: e.Events.filter(b => b.Type === 0).length, mp3Bytes, decodedMs: e.DecodedSamples24kHz / 24 };
}

const files = process.argv.slice(2);
assert(files.length, 'supply recorded events.json paths');
for (const file of files) {
  try {
    console.log(JSON.stringify(verify(JSON.parse(readFileSync(file, 'utf8')), file)));
  } catch (error) {
    console.error(file, error.message);
    process.exitCode = 1;
  }
}
