# 10 — TaoTale API contract and compatibility

**Status:** **Target operation inventory**, revised 23 November 2026; **not** a substitute for a machine-readable OpenAPI 3.1 file. The **new TaoTale OpenAPI contract must be authored and validated** before implementing Gin handlers and generating SolidJS client types. Where route spellings differ from Mandarin Reader, the paths below are **proposed target paths**, not a claim that an existing YAML file already implements them.

**Authority:** [01 — Overview](01-taotale-overview-and-features.md), [04 — Go backend](04-taotale-backend.md), [05 — PostgreSQL/sqlc schema](05-taotale-database.md), [06 — Alignment](06-taotale-segmentation-and-audio.md) and [09 — UI states](09-taotale-ui-states.md).

## 1. Architecture and security boundary

- **REST JSON** under `/api/v1`; **Go + Gin** handles transport, service layer enforces invariants, **sqlc + pgx/v5** executes reviewed PostgreSQL queries. OpenAPI 3.1 is design-first; generate client TypeScript and validate implementation against one canonical contract. No runtime Prisma/NestJS/Swagger decorator assumptions.
- **Single internal User**, resolved by backend. There is no authentication, registration, profile, session, billing or subscription API in MVP. Ignore/reject arbitrary client-provided `userId`; do not model an unauthenticated network API as safe for public exposure.
- Same-origin on **loopback** by default; no browser secret keys, DB connection credentials or filesystem paths. Permit expected local development proxy origin only by explicit configuration. CSRF/cross-origin write restrictions remain relevant even without login.
- JSON UTF-8, camelCase externally; UUIDs for resource IDs, BCP-47 learning-language code (`zh-CN`, `ko-KR`), SQL snake_case internally. Use documented max JSON body (legacy baseline 100 KB), strict validation, explicit `null` clears nullable PATCH fields, omission leaves unchanged. Length checks retain Unicode code-point vs UTF-16 distinctions.
- `X-Request-Id` and sanitized typed error shape:

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Check the highlighted fields.",
    "details": { "fields": [{ "path": "title", "code": "required", "message": "A title is required." }] },
    "requestId": "example-request-id"
  }
}
```

- Do not leak stack traces, SQL, private text, absolute media keys or Azure credentials. API JSON should not be cached by shared intermediaries; media is private, immutable asset bytes support safe byte ranges and cache behavior.

## 2. Language ownership and lifecycle invariants

1. Every Library item has exactly **one language**; new drafts select a supported language, and a published item's language does not change in place. Reader always uses the item language, not a global selector.
2. Vocabulary identity and lookup include `(userId, languageCode, normalizedWordKey)`. Word→item reference operations reject mismatched languages. Topics are user-global and carry no language code.
3. Only manually selected **`beginner`**, **`intermediate`**, **`advanced`** or `null` difficulty is valid; `native` must be rejected for new writes and dealt with during migration.
4. **`isFavorite` is a Boolean on the item**, valid for drafts and published items. An explicit set-value write must not bump the content `version`, `updatedAt` or invalidate audio, and it must not be overwritten by later stale full-row item updates.
5. A published item points to an **immutable validated audio asset** containing original text identity, language, provider/voice/rate, segmentation strategy ID/version, fingerprint format/version and timed segments. Metadata updates leave the active pair unchanged.
6. Working revisions change text/voice/rate only through a saved candidate and **matching READY** generation. Stale versions return `409 VERSION_CONFLICT`; failed candidates cannot replace published audio.
7. Generation jobs persist immutable intent before a `202`, limit concurrent paid calls and recover uncertain requests with the same idempotency key/body. No automatic paid retries.
8. **No AI generation, practice, islands, dashboard, auth or payment API in MVP.** Azure speech synthesis is in scope via the Editor.

## 3. MVP operation inventory

All paths below omit the `/api/v1` prefix. HTTP methods and response DTOs must be locked in the future OpenAPI file. Existing Mandarin Reader `/tags` and item paths are historical; **`/topics` is the proposed TaoTale naming** (alias/compatibility mapping is an explicit migration decision).

| Method | Proposed path | Intent / key contract |
|---|---|---|
| GET | `/languages` | Enabled supported-language catalog; do not expose unsupported Japanese as usable. |
| GET | `/items` | Mixed draft/published, filtered by **required `languageCode`**, title, topics, difficulty, favorite, status; stable sorting/paging + total. |
| POST | `/items` | Save draft; incomplete title/text allowed, optional owned preview claim. |
| POST | `/items/publish` | Direct publish from complete item + exact validated READY preview; idempotency key required. |
| GET | `/items/{itemId}` | Detail, draft audio status or published validated active asset; include language/favorite/difficulty. |
| PATCH | `/items/{itemId}` | Versioned draft or published-metadata edit; published synthesis inputs are immutable here. |
| DELETE | `/items/{itemId}` | Delete item and source joins, preserve vocabulary entries, release unused files safely. |
| PUT | `/items/{itemId}/favorite` | **Set** `isFavorite: boolean`; idempotent, independent of item content version, returns confirmed flag. |
| POST | `/items/{itemId}/publish` | Publish saved draft with expected version and matching READY generation. |
| GET | `/items/{itemId}/vocabulary` | Reader projection of manually linked and verified automatically matched words (if enabled). |
| POST | `/items/{itemId}/vocabulary/sync` | **Conditional**: available only for a language with a verified matching strategy; otherwise typed unsupported response. |
| POST | `/items/{itemId}/audio/generate` | Saved-draft generation: snapshot saved synthesis inputs, return `202` + generation ID. |
| GET | `/items/{itemId}/audio/generations/{generationId}` | Generation status and matching candidate identity; no false READY. |
| GET | `/items/{itemId}/audio` | Current active or eligible matching candidate stream, asset-pinned when mutable route. |
| GET | `/items/{itemId}/audio/download` | Verified MP3 download, safe filename and correct Range behavior. |
| POST | `/items/{itemId}/revisions` | Open or continue one working revision; preserve active recording. |
| GET/PATCH/DELETE | `/items/{itemId}/revisions/{revisionId}` | Read, save versioned, or explicitly discard candidate. |
| POST | `/items/{itemId}/revisions/{revisionId}/audio/generate` | Generate audio for saved revision candidate. |
| GET | `/items/{itemId}/revisions/{revisionId}/audio` | Candidate stream only (never silently fall back to published bytes). |
| GET | `/items/{itemId}/revisions/{revisionId}/audio/download` | Candidate download when matching READY. |
| POST | `/items/{itemId}/revisions/{revisionId}/publish` | Validate candidate + base asset and atomically swap active pair. |
| GET | `/tts/voices?languageCode=...` | Configured, available voices and supported rates for **one supported language**. |
| POST | `/tts/previews` | Unsaved temporary audio generation, immutable text/language/voice/rate snapshot; `202`. |
| GET | `/tts/previews/{generationId}` | Temporary generation status until claimed/expired. |
| GET | `/tts/previews/{generationId}/audio` | Unclaimed READY preview MP3, expires by policy. |
| GET | `/tts/previews/{generationId}/audio/download` | Download unclaimed verified preview. |
| POST | `/storage/images` | Upload new cover **≤1,000,000 bytes** JPEG/PNG/WebP, valid dimensions; staged reference ID. |
| GET | `/storage/images/{imageId}` | Safe image stream, ETag/304 as applicable; legacy oversized cover still readable. |
| GET/POST | `/topics` | Shared one-user Topic catalog and create; not language-filtered. |
| PATCH/DELETE | `/topics/{topicId}` | Rename/recolor or delete custom Topic; protect default ones. |
| GET/POST | `/vocabulary` | **Required `languageCode` for list and creation**; server pagination and duplicate checks. |
| GET | `/vocabulary/lookup?languageCode=...&word=...` | Language-aware normalized exact duplicate lookup. |
| GET | `/vocabulary/stats?languageCode=...` | Per-language total/Learning/Acquired/referenced-text counts. |
| GET/PATCH/DELETE | `/vocabulary/{vocabularyId}` | Details, metadata/status edit or delete; language comes from existing row. |
| POST | `/vocabulary/{vocabularyId}/references` | Explicitly link one same-language item; no implicit metadata overwrite. |
| DELETE | `/vocabulary/{vocabularyId}/references/{itemId}` | Explicit unlink; optional verified matcher may rediscover on later sync. |
| POST | `/vocabulary/bulk` | Atomic same-owner status changes or deletes; explicit IDs and bounded body. |
| GET | `/health/live` | Process liveness. |
| GET | `/health/ready` | Live PostgreSQL/schema/media/owner/recovery state, sanitized failure. |

**Settings:** current theme, active language, playback and per-language voice preferences remain **validated client-local values**. Do not invent a required `/settings` write API or redundant preference table. Expose only the backend configuration/status needed to explain whether Azure generation is available, without returning secrets.

## 4. Query, shape and pagination requirements

- `GET /items` uses explicit `languageCode`, **literal case-insensitive title search** (escape SQL wildcard meaning), `status=draft|published|all`, optional difficulty, topics (OR), `favoritesOnly`, `sort=newest|oldest|title_asc|title_desc`, positive `page`/bounded `pageSize`. Preserve old initial **8 cards/page** unless changed by 01/03; cap pages server-side and use stable `id` ties. `publishedAt`, not audio availability, defines published status.
- `GET /vocabulary` and stats require an explicit language filter; search in display word, reading and meaning, optional status/topic filters, stable sorts, **10 rows/page** default. Avoid per-row API fan-out. Do not silently merge Chinese and Korean word statistics.
- Item response exposes `languageCode`, `isFavorite`, `difficulty`, stable `version`, draft/published state, audio status and immutable asset ID; generated segment ranges are UTF-16 half-open offsets and milliseconds. Source text is preserved byte-for-byte where applicable, not normalized for display or hashing.
- `AudioGeneration`: persistent `PENDING`, `PROCESSING`, `READY`, `FAILED`. `OUTDATED` and `NOT_GENERATED` are **computed UI/target states**. `FAILED` plus `GENERATION_INTERRUPTED` models interrupted runs. `READY` returned to a playable UI requires a current **matching validated asset**, not merely a historical status row.
- Topic DTOs include stable default/system identity and four accessible persisted colors; foreign/unowned references are rejected without leaking unrelated data. Vocabulary includes optional language-generic `reading`/`readingSystem` rather than mandatory `pinyin`.
- Pure metadata writes, including Favorite changes, do not require regeneration; old media URLs are pinned to expected `audioAssetId` to prevent mixed byte-range playback after revision.

## 5. Mutations, idempotency and errors

- **Idempotency-Key** (UUID) required on new TTS generation and direct publication operations. Server stores operation+canonical intent hash and a durable replay outcome. Same key+same body resolves the original attempt; same key+different body is `409 IDEMPOTENCY_KEY_REUSED`. Once a provider may have been charged, never auto-issue a new call to repair an uncertain result.
- Item/revision expected-version conditional SQL update returns `409 VERSION_CONFLICT` + current version on stale writes. Favorites are a **separate set-value operation** not a whole-item save; avoid lost updates even across multiple browser tabs.
- Use `400 VALIDATION_ERROR`, `404 *_NOT_FOUND`, `409 INVALID_ITEM_STATE` / `AUDIO_NOT_READY` / conflicts, `413 PAYLOAD_TOO_LARGE`, `415 UNSUPPORTED_MEDIA_TYPE`, `416 INVALID_RANGE`, `429 TTS_RATE_LIMITED`, `503 TTS_UNAVAILABLE` / `SERVICE_NOT_READY`, and sanitized `500 INTERNAL_ERROR` as applicable. The final enum and mappings belong in OpenAPI, not multiple hand-maintained implementations.
- **Cover upload boundary:** **1,000,000 encoded bytes inclusive** plus actual type and dimensions. Reject 1,000,001-byte submissions before replacing the old cover; preserve any migrated >1 MB asset on read. Do not truncate/compress silently.
- No automatic write queue while disconnected; clients can retry safe reads. Mutation recovery explicitly distinguishes a confirmed failed job (new deliberate attempt/key) from a timeout with unknown result (same key/body first).

## 6. Media and segmentation interoperability

- Full audio response `200` or valid single byte-range `206`, unsatisfiable valid range `416`, malformed/multiple ranges handled according to the adopted media contract. `Content-Type: audio/mpeg`, `Accept-Ranges`, correct `Content-Length`/`Content-Range`, private cache behavior. Serve via item/asset ID; never accept arbitrary filesystem storage keys.
- The Go Azure adapter translates **UTF-8/Go indexes** to the chosen persisted **UTF-16** source offsets, validates SSML mapping/word-boundary events and measured decoded MP3 duration. Each asset pins `languageCode`, segmentation strategy ID/version and fingerprint format. Old published Mandarin assets remain readable under a documented compatibility path; do not silently resegment old audio.
- No fabricated duration/segment timing. On invalid alignment, Reader can return original text and independent vocabulary data but cannot claim accurate seek targets.

## 7. OpenAPI and migration workflow

1. Inventory the historical [Mandarin Reader OpenAPI 3.1 YAML](https://github.com/ronmilich/mandarin-reader/blob/dc71c1b1820345023585be95e72674bf1b3a4a00/docs/api/openapi.yaml). It had 52 historical method/path pairs; **that is not TaoTale's final endpoint count**.
2. Create TaoTale's canonical YAML: language additions, favorites, three difficulty values, language-specific vocabulary and TTS, proposed `/topics` rename; remove or mark deferred all `/ai/*` and other non-MVP routes. Decide if legacy `/tags` or older filter syntax needs an alias; do not promise compatibility without tests.
3. Generate SolidJS TypeScript types, validate Gin handlers and examples against the YAML, and make the contract part of CI. Verify nullability, UTF-16 offsets, error and media headers, language and version invariants with real PostgreSQL integration tests.
4. Data and API migration are independent: **preserve existing UUIDs/audio bytes** even if URLs and DTOs change. Import against a copy and compare published asset identity and vocabulary source references.

**Open decisions:** exact OpenAPI code generator/runtime validation strategy, final `/topics` vs `/tags` path, source length/chunking adjustments and whether a verified Chinese auto-sync operation is MVP-enabled. Those decisions require a contract revision, not ad-hoc handler differences.
