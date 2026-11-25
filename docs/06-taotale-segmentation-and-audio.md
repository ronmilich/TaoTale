# 06 — TaoTale: Text segmentation, audio alignment and Reader synchronization

**Status:** MVP implementation specification for `zh-CN` and `ko-KR`. Language-specific strategies, Go/Azure feasibility, and migration compatibility must be verified in implementation; sample language rules below are **proposed starting policies**, not evidence of verified provider behavior.

**Authority:** [01 — Product overview](01-taotale-overview-and-features.md) defines MVP scope; [03 — Client](03-taotale-client.md) defines the SolidJS Reader; [04 — Backend](04-taotale-backend.md) defines Go/Gin/Azure adapters; [05 — Database](05-taotale-database.md) defines persisted job, asset and segment fields. Historical reference: Mandarin Reader SYN/SEG/ALN rules and D-08/23/31–33/37/45/54/55. Where they conflict, current TaoTale decisions win.

## 1. Scope and non-negotiable invariants

The MVP flow is **manually enter/paste text → explicitly generate audio with Azure TTS → validate alignment → publish → read/listen with synchronized highlighting**. It is independent of the **post-MVP AI Text/Audio Generation Workspace**. Both the Mandarin and Korean pathways must work in the local, single-user installation. Japanese and additional languages are deferred until they have their own verified segmentation/alignment strategies.

- **Source preservation:** Store, transmit, fingerprint and render the *exact* user-entered text. Never silently trim, normalize to NFC, change line endings, translate, convert scripts, replace punctuation or collapse spaces in the source. Form validation may inspect a trimmed copy without changing the persisted source.
- **Separate concerns:** Language-specific **segmentation** chooses proposed text runs; provider **boundary normalization** maps synthesis events to the original source; **alignment validation** checks real audio and timing; the **Reader** renders a pinned published source/asset pair. These are different operations.
- **No invented precision:** Punctuation heuristics or predicted speech duration do not constitute verified timestamps. Missing/unmappable boundaries must not produce a falsely `READY` asset with fabricated timing.
- **Immutable publication:** A published item has an active matching source + audio + segments + strategy identity. New rules affect *future generations* and saved work; published legacy recordings keep their stored segments until explicitly replaced by a validated working revision.
- **Local operations:** PostgreSQL metadata and MP3s are persisted locally; Azure TTS remains an explicit external call. No AI text generation, login, cloud media storage, phone-specific Reader or word-level dictionary/TTS is required for MVP.

## 2. Shared source-offset and timed-segment contract (SYN)

**Canonical source offsets are UTF-16 code units**, matching JavaScript `String.length` and DOM `Range` text offsets. A source span is **half-open** `[textStart, textEnd)`. In Go, source strings are UTF-8; **Go byte indexes are never persisted as text offsets**. Unicode code-point counts used for metadata limits are a separate convention.

| Field / concept | Required meaning |
|---|---|
| `languageCode` | Exact item's registered learning language: MVP `zh-CN` or `ko-KR`. |
| `strategyId` / `strategyVersion` | Selected immutable segmentation policy identity; persisted with generation intent and asset. |
| `textStart` / `textEnd` | Integer UTF-16 offsets into the **exact** original source, end-exclusive; neither splits a surrogate pair. |
| `text` | Exact source substring for `[textStart,textEnd)`; no normalized rendering copy. |
| `startMs` / `endMs` | Integer milliseconds from **the beginning of the persisted MP3**; `0 <= startMs < endMs`. |
| `index` | Zero-based consecutive index, ordered by source position and audio time. |
| Untimed gap | Any source range outside timed segments: blanks, line breaks, interstitial whitespace or punctuation-only text. Render exactly; no seek target. |

Persist segments as an ordered JSONB array or equivalently validated representation described in 05, for example:

```json
[
  {"index":0,"text":"你好，","textStart":0,"textEnd":3,"startMs":80,"endMs":610},
  {"index":1,"text":"世界。","textStart":3,"textEnd":6,"startMs":680,"endMs":1230}
]
```

*The numeric times above demonstrate the shape only; they are not provider measurements.*

**Go converter requirements:** Build a deterministic mapping of UTF-8 byte positions ↔ Unicode scalar boundaries ↔ UTF-16 offsets; reject offsets inside a UTF-8 sequence or surrogate pair. Tests must cover supplementary Han/emoji, combining marks, Hangul syllables, CJK punctuation, tabs, CRLF, standalone CR and mixed script. Go input must be valid UTF-8; reject malformed requests instead of replacing invalid bytes and changing source identity. In the browser, validate stored spans against the original JavaScript string, not against HTML-escaped markup.

### 2.1 Segment representation and gaps

- A segment must be speech-bearing: after excluding whitespace and punctuation-only content, it has pronounceable source content. Gaps need not be stored as independent database rows because they are recoverable as exact source slices between segments.
- Segments are ordered and nonoverlapping by **source** and **time**. Text offsets and speech timing are separate axes; intersegment audio silence is allowed.
- A timed segment includes punctuation only when that punctuation is part of the exact chosen source run. Azure events may time the words but not punctuation; this is acceptable if the run has verified contained speech events.
- No segment includes paragraph line breaks for the initial MVP strategies. Preserve all original blank lines as untimed gaps.
- Store the strategy ID/version with the asset. **Never rerun the current segmenter on historical published audio and overwrite existing ranges.**

## 3. Language-specific segmentation registry (SEG)

Go owns a **pure**, deterministic registry, selected by `languageCode` and an **explicit version** rather than user-agent language or detected script. A documented shape is:

```go
// Illustrative interface; not required package/public API.
type Strategy interface {
    ID() string
    Version() string
    Segment(exactSource string) ([]SourceSegment, error)
}

type SourceSegment struct {
    TextStartUTF16 int
    TextEndUTF16   int
    Text           string
}
```

Suggested registry entries:

| Language | Strategy | Status |
|---|---|---|
| `zh-CN` | `zh-cn-phrase` (new versioned implementation) | **MVP**, separately tested. |
| `zh-CN` historical assets | `zh-cn-legacy-v1` (preserved semantics) | **Compatibility** for existing recordings and fingerprints. |
| `ko-KR` | `ko-kr-phrase` (independent implementation) | **MVP**, separately tested. |
| `ja-JP` | Own future strategy, not a Chinese alias | **Post-MVP**; do not ship a placeholder that accepts publication. |

**Versioning:** Fix the exact rule set and regression vectors for each strategy release. Any change that alters speech-bearing spans, delimiter consumption, paragraph handling or required alignment behavior requires a new version. A code refactor that produces the *exact same* segments need not change the semantic version, but must pass golden equivalence tests. Keep old validators/strategies registered as long as assets depend on them.

### 3.1 Simplified Mandarin (`zh-CN`)

Mandarin Reader **legacy v1** splits at `，` and `。`, and at ASCII `-` when the nearest non-whitespace characters on both sides of the same line are Han. It absorbs consecutive delimiters and immediately following closing quotes. CR and LF separate lines. The punctuation `！？；：、` stays inside its run. Preserve this behavior for recorded assets and verify it against the [legacy golden vectors](../testdata/legacy/mandarin-v1.json).

For **new TaoTale Chinese generation**, start with a versioned, deliberately simple phrase strategy: segment at validated Chinese sentence/phrase punctuation (candidates include `，。！？；`), preserve closing quotes with their preceding run, and respect paragraph boundaries. Determine punctuation-only and quotation edge cases with explicit tests **before freezing the version**. Do not automatically split on spaces as if Chinese were English; do not reuse language detection as segmentation. If the existing v1 policy is retained for new generations initially, use its legacy ID/version until a newly tested successor is adopted.

Illustrative **candidate** input: `随着科技的发展，人们的生活发生了变化。` → two timed source runs when comma/period are included. `你好。\n\n再见` → two timed runs with an exact untimed blank-paragraph gap. For hyphenated names, numbers, URLs and Latin insertions, do not blindly split on every hyphen.

### 3.2 Korean (`ko-KR`)

Korean has different spacing and punctuation behavior. Implement an **independent** `ko-kr-phrase` strategy with a documented first version, not a fork of Chinese regexes.

- Treat explicit newlines/paragraphs as structural boundaries, preserving them untouched.
- Start with sentence-ending punctuation such as `.`, `?`, `!` and common full-width equivalents, with any trailing closing quote/bracket attached consistently. The exact punctuation set is versioned and verified against real Korean text.
- **Do not assume each space or Hangul syllable is a speech segment.** Spaces may separate eojeol, but useful Reader highlighting may be phrase/sentence-level. Grammatical endings can exist without sentence punctuation; a long unpunctuated line should remain a bounded, predictable candidate unless a separately tested rule is adopted.
- Protect periods in decimal numbers and common abbreviations, ellipses and repeated punctuation; define deterministic behavior for `...`, `…`, parentheticals, mixed Latin and numbers.
- Strip nothing from the source; avoid automatic spacing fixes, morphology/lemmatization, honorific transformations or Korean word-boundary/dictionary claims.

Example candidate: `안녕하세요. 저는 한국어를 공부해요!` → two sentence runs; `오늘은 3.5시간 공부했어요.` must not break at the decimal point. These are **intended test behaviors**, not confirmed Azure boundary patterns.

### 3.3 Language strategy gates

A strategy is **enabled for generation/publishing** only after deterministic unit vectors and real Azure integration samples show acceptable source-to-audio mapping. Do not silently fall back from an unsupported language to Chinese. The voice must be compatible with the selected learning language; reject incompatible voice/lang combinations before charging Azure.

## 4. Synthesis fingerprint and exact identity

The generation intent and immutable output identity must cover at least:

```text
fingerprintSchemaVersion
languageCode
providerId
voiceId
speakingRate
strategyId
strategyVersion
exactSourceText
```

**New TaoTale fingerprints:** define one canonical serialization (fixed field order, explicit string escaping, stable numeric format, UTF-8 encoding and version identifier) and SHA-256 it. The value is computed on the backend; clients should not synthesize their own fingerprint as a source of truth. Persist the schema version, and test byte-for-byte Go golden vectors including `1` versus `1.0`, non-ASCII text, `<`, `&`, emoji and CRLF. Do not hash item metadata such as title, favorite, difficulty, cover, Topics or notes.

**Legacy compatibility:** Historical Mandarin Reader fingerprint schema is SHA-256 over canonical JavaScript JSON fields in order `segmentationVersion`, `speakingRate`, `text`, `voiceId`. Keep a **legacy verification branch** using golden vectors and stored version semantics. Do not recompute old hashes with new `languageCode`, provider or strategy fields; do not regenerate old MP3s merely to change hashing format.

A candidate becomes `OUTDATED` (computed, not a persisted job status) if its exact source, language, provider/voice/rate or strategy identity no longer matches the **current saved candidate intent**. A previously published source/audio pair remains valid with its **pinned historical** identity. Upgrading segmentation code alone does not make the active Reader invalid.

## 5. Azure TTS boundary normalization (ALN)

The existing **Azure Speech** provider is retained behind a **Go adapter** (`TTSProvider`). The adapter must normalize:

- Voice catalog and language compatibility; provider identity, voice ID and speaking-rate settings.
- MP3 bytes (or a verified completed output file), provider output metadata and source word-boundary events.
- Boundary source offsets/ranges, text where available, 100-nanosecond audio ticks where reported, cancellation/errors, and provider-specific limitations.

**Feasibility gate before switching from the old Node implementation:** On the actual target OS/runtime, prove that Azure's Go SDK can capture word-boundary events and MP3 output for **both** `zh-CN` and `ko-KR`. Check native library/CGO/runtime dependencies, event ordering, callback lifetime and cancellation. Go SDK API availability does **not** by itself prove event fidelity for all voices/languages. Reference: [Azure speech synthesis](https://learn.microsoft.com/azure/ai-services/speech-service/how-to-speech-synthesis) and [Go SDK event API](https://pkg.go.dev/github.com/Microsoft/cognitive-services-speech-sdk-go/speech).

**Exact mapping process:**

1. Keep one immutable source snapshot. Generate SSML using correct XML escaping and track how the provider indexes escaped text versus actual spoken source; account for wrappers and offsets. Never infer source ranges by string-searching repeated spoken words.
2. Convert each provider boundary into a verified **original-source UTF-16 span**, using the provider's actual documented offset units and a tested mapping. If returned text is available, check against the recovered source span. Reject invalid spans, surrogate splitting, out-of-order offsets or unexplained provider differences.
3. Validate source/tick order and genuine overlap in raw provider units **before** rounding. Associate word events with exactly one source segment; reject events spanning incompatible segment/gap boundaries.
4. For each segment, coalesce its verified spoken events: start = first actual spoken event start, end = last actual spoken event end. Then convert audio ticks to ms (`floor(startTicks/10000)`, `ceil(endTicks/10000)`). This avoids rounding adjacent events individually into artificial overlap.
5. Punctuation-only boundaries cannot confer timing to a gap; pauses are allowed. Never borrow timing from neighboring segments or assign a duration based only on characters.

**Single-output MVP:** Retain the initial `≤1,500 UTF-16 code units` text cap while validating provider limits for Korean and Mandarin, and a single completed MP3 output. If chunking later becomes necessary, real audio must be stitched into one valid MP3 and timestamps shifted by **measured** preceding output durations; concatenating timestamp offsets without valid audio stitching is insufficient. Existing Azure SSML/request guards can inform the implementation but must be rechecked with the Go SDK.

## 6. Validation gate before `READY` and publication (ALN)

The Go generation worker may persist an immutable `AudioAsset` and set a job to `READY` **only after all checks pass**:

1. **Source/intent:** supported language, compatible provider/voice/rate, exact source snapshot and matching strategy/fingerprint schema.
2. **Source segmentation:** every speech-bearing candidate has an exact valid UTF-16 slice; segments are ordered, positive-length, nonoverlapping and do not split surrogate pairs. Gaps reconstruct original text exactly.
3. **Provider events:** all boundaries used are finite, valid, correctly mapped, monotonic and contained within the intended speech run; no unaccounted event covers speech across a gap.
4. **Coverage:** every speech-bearing candidate has verified timing. Untimed blank/punctuation-only gaps are permitted. **Partial alignment does not count as full synchronization.**
5. **Audio file:** persisted MP3 is complete and decodable; measure **actual positive decoded duration**, rather than trusting provider metadata alone. Choose and verify a Go-compatible decoder/probe because the Go standard library does not decode MP3.
6. **Time ranges:** `0 <= startMs < endMs`, ordered and nonoverlapping, within the decoded audio length. Preserve the legacy maximum 50 ms end-overrun clamp **only for legacy-compatible validation** or after proving an identical new policy; larger unverified drift fails.
7. **Final identity:** asset bytes, duration, segments, exact source and fingerprint are committed and stored consistently; `READY` is not set merely on receiving provider success.

At draft/direct/revision publication, verify the persisted candidate **again** (identity, asset reference, alignment, expected version and current saved inputs). Reject any mismatch; **do not repair/resegment in the publish transaction**. Publication failure must leave the old published recording playable and must not silently trigger another Azure synthesis.

Persisted job states remain `PENDING → PROCESSING → READY` or `FAILED`; interrupted uncertain jobs become `FAILED` plus a documented failure code, and **must not be automatically re-submitted as paid calls**. `OUTDATED` is a computed compatibility state. Idempotency, short PostgreSQL transactions, staging files, atomic renames and orphan cleanup are owned by [04](04-taotale-backend.md) and [05](05-taotale-database.md).

## 7. Reader synchronization (SolidJS)

The SolidJS client receives the published exact source, **asset-pinned URL**, stored segments and measured duration; it **never recalculates segments from current rules**. For valid alignment:

- Reconstruct a sequence of alternating timed runs and exact untimed source gaps. Validate offsets against the JS string; preserve `white-space: pre-wrap`, blank paragraphs, native selection and `textContent`.
- Use a native `HTMLAudioElement`. The audio element's playback clock is authoritative; don't compute speech position from `setInterval` or speaking-rate multiplication.
- Identify the currently active segment by a binary search over times, considering `[startMs,endMs)` rather than highlighting an earlier segment through an unrelated long silent gap. Before the first segment, after a genuine gap and after playback ends, show no active segment. Pausing preserves the last appropriate visual state; `seeked` reconciles it.
- Sample the clock on `requestAnimationFrame` **while playing**, and handle `play`, `pause`, `seeked`, `timeupdate`, `ratechange` and `ended` as needed. Update Solid signals for the active segment **only when the index changes**. Isolate a fast-changing progress indicator from transcript and Library reactivity; clean up listeners and animation frames on disposal.
- Clicking or keyboard-activating a valid timed segment seeks to its `startMs`; preserve playing/paused state. Never seek when the user is selecting text or dragging. Fixed ±5 s controls, playback speeds and MP3 download follow 01/03.
- Persist playback position locally per `(itemId, audioAssetId)` approximately every 5 s/on pause; no autoplay on restore; replacement asset resets prior resume. Playback-rate changes affect the native clock, not stored timing.

**Graceful fallback:** If metadata/ranges are corrupt or a media file is missing, retain the original text in readable form, disable invalid timed seek/highlight/download controls and display retry/error state. This must **not** disable manual selection or independent vocabulary UI. Do not hide text merely because audio cannot load.

## 8. Vocabulary selection and annotations — independent of speech segmentation

- Selection is measured from native DOM `Range` endpoints relative to the transcript root in **UTF-16**, including spans that cross timed/gap DOM runs. Obtain the exact selected substring, trim outer whitespace only for the proposed word/phrase, validate against the language-independent base limits and preserve the original article unchanged.
- Use the **Library item's language** for lookup/link/create; never infer language from selected characters. A Chinese vocabulary entry cannot be linked to a Korean article. Existing entries are linked idempotently without overwriting status, meaning or reading.
- Speech-segmentation strategies are **not vocabulary tokenizers**. The old exact, case-sensitive substring/longest-first visual matching policy may be enabled only for a language where it has been deliberately tested. For Korean, substring occurrence can be misleading because particles/inflections may change surface forms. **Manual references and manual collection remain authoritative.**
- If language-specific automatic matching is disabled or incomplete, do not perform fake lemmatization or block publication; saved manual references, direct selection and vocabulary detail remain usable. Later automatic matching is an independently versioned capability, not an implicit side effect of segmentation.
- When tested visual annotation is enabled, calculate occurrence ranges when source/spelling/reference identity changes, keep status (`Learning`/`Acquired`) decoration separate, and resolve visual overlaps deterministically (historical longest span, then earliest start, then stable ID). Match references in the **same language only**. Render popups outside transcript DOM; ensure selection/audio seek targets remain intact.
- Any explicitly enabled publication vocabulary reconciliation uses its **own** tested per-language matcher and preserves manual/auto provenance as documented in 05. No global background scans, automatic definitions, Korean morphology or word-level audio in MVP.

## 9. Tests, migration and release evidence

**Deterministic Go golden vectors (per strategy/version):** assert exact source bytes, UTF-16 ranges, text values, gaps and reconstruction for `zh-CN`/`ko-KR`; include blank lines, CR/LF/CRLF, curly/straight quotes, Chinese punctuation, Korean sentence endings, ellipses, URLs, date/decimals, ASCII punctuation, brackets, tabs, Hangul+Han+Latin, emoji and supplementary-plane characters. Pin fixtures so a new strategy cannot silently mutate old alignments.

**Provider adapter tests:** fake Azure event streams with text offset shifts from SSML/XML escapes (`&`, `<`, `>`), repeated words, sparse events, event overlap, boundary crossing, zero/negative durations, truncated MP3, duration mismatch, cancellation and unknown outcome. In a **separate, explicit and billable manual integration test**, verify real Go Azure MP3+boundaries in both languages and compare to the old integration on representative passages. Do not run real Azure calls automatically in CI.

**Reader/browser tests:** exercise SolidJS text reconstruction and DOM selection, keyboard seek, play/pause/gaps/end/resume, rapid seeking and playback-rate changes, article vocabulary status/popup refresh without audio remount, invalid alignment fallback and source containing emoji. Verify no per-frame full-transcript rerender and no position drift after revision.

**Publication and migration tests:** preserve legacy `zh-CN` strategy/fingerprint and old MP3/timings byte-for-byte; ensure new assets have new strategy and fingerprint schema identifiers; source or strategy changes mark a **candidate** outdated but never destroy active media. Test crash/restart, duplicated idempotency key, media cleanup, broken files and database+assets restore. Existing covers above 1 MB remain readable; new/replacement covers follow 01/05 limits (not a segmentation responsibility).

**MVP release gate:** Demonstrate for **both Mandarin and Korean**: manual create → explicit Azure generation → validated `READY` → publication → accurate timed Reader playback → manual vocabulary collection → restart and reopen. Where live Azure cannot supply sufficiently reliable events, the generation must fail explicitly rather than claim synchronized audio success; **do not silently remove the language from the promised MVP**. Record the blocker and choose a verified provider/alignment approach as a separate design decision.

## 10. Deferred capabilities and open implementation choices

**Not MVP:** Japanese/other language strategies; phoneme, syllable or word-level synced karaoke; user-supplied audio alignment; speech recognition/pronunciation scoring; AI-created articles; mobile Reader optimization; automatic dictionary/lemmatization; chunked long-form audio and background Library-wide vocabulary scans.

**Implementation decisions to close with tests (not silently approved here):** exact Mandarin successor punctuation policy, Korean v1 boundaries and abbreviation exceptions, new canonical fingerprint serializer/golden vectors, Go MP3 duration decoder, exact Azure SDK event offset semantics on target platform, event-level tolerance policy, and a documented resolution for the old 50 ms validation clamp. Keep them **measurable technical questions**, not new user-facing MVP features.
