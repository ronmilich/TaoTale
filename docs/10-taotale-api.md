# 10 — TaoTale API contract and compatibility

The canonical OpenAPI 3.1 contract covers health, the enabled learning languages and the Topic list. The inventory below describes subsequent MVP operations. [Canonical OpenAPI](../backend/api/openapi.yaml)

**Authority:** [01 — Overview](01-taotale-overview-and-features.md), [04 — Go backend](04-taotale-backend.md), [05 — PostgreSQL/sqlc schema](05-taotale-database.md), [06 — Alignment](06-taotale-segmentation-and-audio.md) and [09 — UI states](09-taotale-ui-states.md).

## 1. Architecture and security boundary

- **REST JSON** under `/api/v1`; **Go + Gin** handles transport, service layer enforces invariants, **sqlc + pgx/v5** executes reviewed PostgreSQL queries. OpenAPI 3.1 is design-first; generate client TypeScript and validate implementation against one canonical contract. No runtime Prisma/NestJS/Swagger decorator assumptions.
- **Single internal User**, resolved by backend. There is no authentication, registration, profile, session, billing or subscription API in MVP. Ignore/reject arbitrary client-provided `userId`; do not model an unauthenticated network API as safe for public exposure.
- Same-origin on **loopback** by default; no browser secret keys, DB connection credentials or filesystem paths. Permit expected local development proxy origin only by explicit configuration. CSRF/cross-origin write restrictions remain relevant even without login.
- JSON UTF-8 uses camelCase externally, UUIDs for resource IDs, BCP-47 learning-language codes (`zh-CN`, `ko-KR`) and SQL snake_case internally. JSON requests are capped at 100 KiB (102,400 bytes). Handlers reject unknown fields and malformed Unicode. An explicit `null` clears a nullable PATCH field and omission preserves its value. Length checks retain Unicode code-point vs UTF-16 distinctions.
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

Paths below omit the `/api/v1` prefix. Update the canonical OpenAPI file to define each new method, path, request and response before implementing the handler. OPEN-14 records the decision to use `/topics`.

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
- Topic DTOs include stable default/system identity and four persisted light/dark Topic color values; foreign/unowned references are rejected without leaking unrelated data. Vocabulary includes optional language-generic `reading`/`readingSystem` rather than mandatory `pinyin`.
- Pure metadata writes, including Favorite changes, do not require regeneration; old media URLs are pinned to expected `audioAssetId` to prevent mixed byte-range playback after revision.

## 5. Mutations, idempotency and errors

The draft API accepts omitted fields as defaults at creation. A PATCH carries `expectedVersion`. Omission preserves a field, `null` clears nullable metadata, an empty string is stored as typed, an empty `topicIds` clears associations, and `false` explicitly clears the favorite flag. Page size defaults to 8 and allows up to 50. The current sort is creation timestamp descending, with descending item ID for ties.

Topics use `/topics` across all learning languages. Writes accept `colorPreset` and the server resolves the stored colors. New Topics default to Iris. Names are unique after Unicode whitespace trimming and case folding. `DUPLICATE_TOPIC` includes `error.details.existingId`. The `usageCount` counts linked items across languages. Default Topics can be renamed and recolored while their system keys and deletion protection persist.

Saved-draft generation accepts an empty body or an empty JSON object. The server resolves an unset voice to the configured provider’s default for that language and an unset rate to 1. A new admission snapshots these values and the exact saved source. Its `202` receipt contains `generationId`. Retrying the same target and key returns the original receipt for 24 hours, including after draft edits or provider unavailability. After that window `409 IDEMPOTENCY_KEY_EXPIRED` reserves the old key permanently. Reusing a key across draft IDs returns `409 IDEMPOTENCY_KEY_REUSED`.

A different key on an active draft returns `409 GENERATION_IN_PROGRESS`. The owner’s active and rolling hourly limits return `429 TTS_RATE_LIMITED`. Deleting a draft preserves its receipt and hourly count. Polling exposes the stored `jobStatus` alongside the candidate’s `status`. A superseded generation or changed synthesis inputs produce `OUTDATED`. A missing or released asset produces `FAILED` with `AUDIO_UNAVAILABLE`. Only an eligible `READY` candidate includes a playable `audioAssetId`.

Temporary previews require exact `sourceText`, `languageCode`, `voiceId`, `speakingRate` and an idempotency key. They use the same worker and limits as saved-draft generations. Status includes `expiresAt`, 24 hours after admission. Pass `previewGenerationId` on draft creation or a versioned draft update to claim a READY preview whose synthesis inputs exactly match the saved draft. Save and claim commit in one transaction. Subsequent preview reads and claims return `404 PREVIEW_NOT_FOUND` after claim or expiry.

- **Idempotency-Key** (UUID) required on new TTS generation and direct publication operations. Server stores operation+canonical intent hash and a durable replay outcome. Same key+same body resolves the original attempt; same key+different body is `409 IDEMPOTENCY_KEY_REUSED`. Once a provider may have been charged, never auto-issue a new call to repair an uncertain result.
- Item/revision expected-version conditional SQL update returns `409 VERSION_CONFLICT` + current version on stale writes. Favorites are a **separate set-value operation** not a whole-item save; avoid lost updates even across multiple browser tabs.
- Use `400 VALIDATION_ERROR`, `404 *_NOT_FOUND`, `409 INVALID_ITEM_STATE` / `AUDIO_NOT_READY` / conflicts, `413 PAYLOAD_TOO_LARGE`, `415 UNSUPPORTED_MEDIA_TYPE`, `416 INVALID_RANGE`, `429 TTS_RATE_LIMITED`, `503 TTS_UNAVAILABLE` / `SERVICE_NOT_READY`, and sanitized `500 INTERNAL_ERROR` as applicable. The final enum and mappings belong in OpenAPI, not multiple hand-maintained implementations.
- **Cover upload boundary:** **1,000,000 encoded bytes inclusive** plus actual type and dimensions. Reject 1,000,001-byte submissions before replacing the old cover; preserve any migrated >1 MB asset on read. Do not truncate/compress silently.
- No automatic write queue while disconnected; clients can retry safe reads. Mutation recovery explicitly distinguishes a confirmed failed job (new deliberate attempt/key) from a timeout with unknown result (same key/body first).

## 6. Media and segmentation interoperability

- Item playback and download require `?audioAssetId=<uuid>` from the current item or candidate. Stale pins return `404 AUDIO_NOT_FOUND`. Preview media URLs use the generation ID and stop working after claim or expiry. Audio returns `200` or a single satisfiable byte range with `206`, `Content-Type: audio/mpeg`, `Accept-Ranges: bytes` and the correct lengths. Malformed, multiple or unsatisfiable ranges return `416 INVALID_RANGE` with `Content-Range: bytes */<size>`. Item media is private and must revalidate. Preview media is private and must not be stored. Download filenames are generated from resource IDs.
- The Go Azure adapter translates **UTF-8/Go indexes** to the chosen persisted **UTF-16** source offsets, validates SSML mapping/word-boundary events and measured decoded MP3 duration. Each asset pins `languageCode`, segmentation strategy ID/version and fingerprint format. Old published Mandarin assets remain readable under a documented compatibility path; do not silently resegment old audio.
- No fabricated duration/segment timing. On invalid alignment, Reader can return original text and independent vocabulary data but cannot claim accurate seek targets.

## 7. OpenAPI and migration workflow

1. Edit `backend/api/openapi.yaml` before changing a Gin handler. Include response status codes, request IDs and the shared error envelope where applicable.
2. From `frontend/`, run `pnpm api:generate` and commit `src/api/schema.d.ts` with the contract update. Run `pnpm api:check` to lint the YAML with Redocly and check generated declaration freshness. openapi-typescript uses its TypeScript 5 peer in isolation from the application's TypeScript 7 compiler.
3. Run `go test ./internal/httpapi -run TestOpenAPIConformance` from `backend/`. The tests compare Gin method and path registrations with OpenAPI and validate HTTP statuses, headers and bodies using kin-openapi. Add representative success and failure fixtures as operations grow.
4. Preserve existing UUIDs and audio bytes during data import. Rehearse against a copy and compare published asset identity and vocabulary source references even when the HTTP paths or DTOs change.
