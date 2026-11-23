# 04 — TaoTale backend architecture

**Status:** Target MVP architecture — approved stack; implementation details and exact versions require verification.

**Authority:** [`01-taotale-overview-and-features.md`](01-taotale-overview-and-features.md) governs product scope; [`02-taotale-decision-register.md`](02-taotale-decision-register.md) records decisions; [`03-taotale-client.md`](03-taotale-client.md) describes the SolidJS client; [`05-taotale-database.md`](05-taotale-database.md) defines the logical persistence model. This document supersedes the **NestJS/Node/Prisma architecture** of Mandarin Reader. The historical behavior is a reference, not permission to reintroduce obsolete features.

## 1. Selected stack and deployment

| Concern | TaoTale decision | Consequence |
|---|---|---|
| Language | **Go** | Strictly typed services, explicit concurrency and cancellation, performance measured rather than assumed. |
| HTTP | **Gin** (`github.com/gin-gonic/gin`) | Thin handlers, explicit middleware, JSON API and static SPA hosting where appropriate. |
| Database | **PostgreSQL** running locally in **Docker Compose** | Persist data in a named volume/bind mount; no managed database. |
| Driver/pool | **pgx/v5** and `pgxpool` | Native PostgreSQL access, context-aware queries and pooled connections. |
| SQL access | **sqlc** | SQL files are authoritative for queries; generate type-safe Go code. **No Prisma, GORM or other ORM.** |
| Migrations | Ordered, reviewed **SQL migrations** with a lightweight runner | Migration tool selection remains open; sqlc is **not** a migration runner. |
| Asset storage | Local filesystem outside source/build and ephemeral container layers | Immutable MP3 and cover files, temporary previews, safe generated keys. |
| TTS | Existing **Azure Speech TTS**, behind a Go adapter | Preserve audio/word-boundary behavior; do not couple domain services to SDK types. |
| AI | Previously chosen AI provider behind an adapter **when its feature is implemented** | **No AI generation API/workspace, account connection or startup dependency in MVP.** |
| API contract | REST JSON under `/api/v1`, design-first OpenAPI | Generate TypeScript client types for SolidJS; test response parity. |
| Architecture | **One modular Go monolith, one backend process, one PostgreSQL database** | No Redis, microservices, message broker, auth service, billing or hosted infrastructure. |

**Locality:** one internally resolved `User` row, no login/logout/onboarding. Serve HTTP on `127.0.0.1` by default. If the Go process runs on the host, publish PostgreSQL to **host loopback only**; if Go also runs in Compose, connect to the DB service on the Compose network without exposing PostgreSQL. Persist asset paths via a host data directory (bind-mount when containerized). Desktop and tablet-sized **layouts** are MVP; LAN access from a different device is **not** assumed or approved.

**Scope:** manual Library, Favorites, Create/Edit, drafts, revisions, TTS jobs, published Reader, Vocabulary, Topics and Settings/local backup. **Post-MVP:** Dashboard, Vocabulary Practice, AI Text/Audio Generation Workspace, Language Islands, phone support and commercial account/payment functionality.

## 2. Architectural boundaries

**Handlers → application services → sqlc-generated queries and explicit adapters.** Keep infrastructure choices outside the domain model.

| Module | Responsibility |
|---|---|
| `library` | Language-scoped items, drafts, metadata, Favorites, filters, pagination, publish and working revisions. |
| `generation` | Persistent job admission, idempotency, worker, recovery, source snapshots, status and provider error mapping. |
| `tts` | Provider-neutral `TTSProvider`, Azure implementation, compatible voice catalogs, rate validation. |
| `segmentation` | Pure, **language-specific** segmentation, source offsets, algorithm/version registry and timing validation. |
| `vocabulary` | Language-scoped words, optional readings, manual status, source links, statistics and conditional, language-tested matching. |
| `topics` | Owner-global topic catalog, accessible colors and cross-language joins. |
| `media` | Local cover/audio storage, safe streams and range requests, leases/cleanup. |
| `system/settings` | Installation configuration, provider readiness and backup metadata; small UI preferences (including per-language voice/rate defaults) remain client-local for MVP per 03. |
| `system` | Config, single-user seed/lookup, migrations, health, backup/restore operations and structured logging. |

Rules:

- Gin handlers parse, validate, choose response codes and invoke services. Do not put business transactions, Azure calls or filesystem path construction in handlers.
- Services take `context.Context`, not `*gin.Context`. Keep segmentation, fingerprinting, normalization and validation as deterministic Go functions without DB/network dependencies.
- Services call **sqlc queries** (and `pgx` transactions) directly through small, explicit dependencies. Do not recreate Prisma models or introduce a generic repository framework simply to wrap generated methods.
- Item publication coordinates generation/asset validation and optional supported-language vocabulary reconciliation **inside the publication transaction**; synthesis occurs **outside** SQL transactions.
- Dependencies must be acyclic. The domain owns interfaces (`TTSProvider`, file operations), while adapters implement them. No general-purpose event bus is needed.

### Suggested Go layout (illustrative, not a second architecture framework)

```text
backend/
  cmd/taotale/main.go
  internal/
    app/                         # composition, config, boot/shutdown
    httpapi/                     # Gin handlers, routes, middleware, DTOs
    library/                     # item lifecycle + Favorites
    generation/                  # durable jobs + worker
    tts/                         # provider contract and Azure adapter
    segmentation/
      registry.go
      zhcn/                      # Chinese rules + golden tests
      kokr/                      # Korean rules + golden tests
    vocabulary/
    topics/
    media/
    system/                      # installation/config/status, not browser preferences
    persistence/
      queries/                   # sqlc generated Go; do not hand-edit
      db.go                      # pgxpool/transactions/composition
  db/
    migrations/                 # ordered SQL; tracked checksums
    queries/                    # named .sql queries for sqlc
    sqlc.yaml
  api/openapi.yaml
  tests/                         # integration/contract/restore tests
```

Do not introduce a Japanese strategy package until `ja-JP` support is planned; keep the strategy interface ready for one. Keep the codebase small: a package/module earns its existence by isolating a real responsibility, not by mimicking NestJS modules.

## 3. PostgreSQL, sqlc and transactional data access

- Start **PostgreSQL via Docker Compose** with a persistent volume, health check and an explicit non-default application credential. **No runtime Prisma schema or Prisma migration history as the source of truth.**
- SQL migrations are versioned, reviewed, forward-moving files. On startup/deployment, apply migrations with a separately selected migration runner and then seed the single User, language catalog and default Topics idempotently. Do **not** call `sqlc generate` to migrate a live database.
- Keep SQL in named query files (`-- name: ... :one/:many/:exec`) and generate Go methods/models with `sqlc` using `sql_package: "pgx/v5"`. Generated code is checked in or regenerated in a reproducible build; CI rejects drift from SQL/query inputs.
- Use `pgxpool` (small, configurable pool for the single-user application). Use `pgx.Tx` plus `sqlc` queries bound to the transaction (`WithTx`/`New(tx)` depending on generated API) for item mutation, generation admission, publication, vocabulary linking and idempotency.
- **Explicit SQL first:** add joins, indexes and batch/set-based queries for paginated Library and vocabulary views. Do not perform one query per Library card, topic chip or vocabulary reference. Add indexes based on measured `EXPLAIN (ANALYZE, BUFFERS)` results for realistic local data.
- Preserve optimistic versions for `LibraryItem` and `WorkingRevision`: `UPDATE ... WHERE id=$id AND version=$expected RETURNING ...`. A missed update is a `409 VERSION_CONFLICT`, not a silent overwrite.
- Use short DB transactions and row locks / unique constraints as correctness mechanisms. Retain advisory locks only where they simplify real cross-entity invariants; avoid unnecessarily serializing unrelated reads. One-process operation is a deployment assumption, **not** an excuse to skip transactional race safety.
- Treat `sqlc` output as database-layer types, not JSON DTOs. Map nullable values, timestamps, UUIDs, enums and JSONB explicitly at the transport boundary; never expose internal storage keys.

See `05-taotale-database.md` for the language, ownership and referential constraints that SQL must enforce.

## 4. HTTP contract, validation and local security

- Keep `/api/v1` and **OpenAPI 3.1** as the design-first source. JSON uses camelCase; SQL uses snake_case. Default JSON request size ceiling **100 KB** until a revised source-length policy is approved; uploads use a separate bounded streaming path.
- Use Gin `ShouldBind...` or a small controlled JSON decoder to validate DTOs and return one consistent error envelope (`code`, `message`, optional `fields`, `requestId`). **Gin's default JSON binder does not itself guarantee rejection of unknown properties**; implement/verify strict unknown-field handling deliberately. Reject malformed UTF-8/invalid Unicode escape sequences rather than silently altering exact source content. Avoid Go `omitempty`/zero-value confusion: PATCH must distinguish omitted, explicit `null`, empty string and `false` (especially `isFavorite`).
- IDs are UUIDs. The owner is always resolved server-side from the single local User; never accept an arbitrary `userId` in a request as access authority. If a record does not belong to that owner, return the same not-found behavior as an unknown ID.
- Filter Library and Vocabulary by required learning language (`zh-CN` or `ko-KR`); reject unsupported locale or cross-language joins. For publication, enforce exact source, language, voice, rate and segmentation identity, not merely `status=READY`.
- Add request ID, recoverer, request size/timeouts, structured **metadata-only** logging and typed server errors. Do not log complete article text, TTS payloads, speech keys, internal paths or database credentials.
- Serve UI/API from a same-origin, loopback-bound HTTP listener. Validate Host/Origin for state-changing requests and mitigate cross-site localhost requests and DNS rebinding; loopback is not authentication against arbitrary malicious pages. No general login/session layer is needed for the selected personal deployment.
- Return private/cache-conscious media responses, allow authorized/safe byte-range playback (`206` or `416`) and use **asset-pinned URLs**. Never let an outdated recording URL resolve to replacement bytes mid-seek.
- Routes should match the client MVP (`/library`, `/create`, `/items/:id`, `/vocabulary`, `/settings`); the API path list is maintained in OpenAPI and should **not** expose AI/practice/island endpoints as shipped MVP functionality.

### Idempotency for paid calls and uncertain outcomes

- Persist a client-generated UUID key and canonical request-intent hash before accepting a TTS job. Scope replay by the internal User and operation; same key/same intent returns the same generation, same key/different intent returns `409 IDEMPOTENCY_KEY_REUSED`.
- Use the original key/body to recover an unknown response. Never automatically submit a second billable Azure request because a timeout, browser reload or process restart obscured the first outcome.
- A confirmed failed job may be **explicitly** retried using a **new** generation ID/key. Ordinary GETs may retry, writes and paid operations do not retry automatically.
- Keep historical 24-hour replay/preview retention as an initial policy (not an SLA). If replay rows expire while a generation remains live, preserve a tombstone/conflict check so an old key cannot trigger a duplicate paid call.

## 5. Core Library item lifecycle

1. **Create draft:** accept incomplete title/source and a required learning language. Save explicitly; metadata, topics and Favorite are independent of generation. Optional temporary preview can be claimed if valid and still available.
2. **Generate audio:** for a saved draft/revision, synthesize a persisted immutable snapshot; unsaved synthesis-affecting changes block generation until saved. Unsaved new-item previews use an expiring `TEMP_PREVIEW` target only when offered by the UI.
3. **Publish draft/direct create:** require nonblank title/source and the **latest matching validated READY candidate**. Persist source + active asset + published timestamp atomically, and perform only language-approved vocabulary synchronization. Do not silently create a draft as a side effect of previewing.
4. **Metadata edit:** title, description, notes, difficulty (`beginner`/`intermediate`/`advanced`/`null`), Topics, cover and `isFavorite` do **not** invalidate active audio or advance its fingerprint. Favorite toggling does **not** change `updatedAt`, default newest ordering or the synthesis version; use a targeted update.
5. **Working revision:** one saved candidate per published item; independently versioned. Reader keeps the original source/asset while candidate text/voice/rate is edited and synthesized. Publishing verifies base asset and version then **atomically swaps active source, voice, rate, asset and timing identity**; failure leaves original playable.
6. **Delete:** remove joins and item state transactionally; preserve vocabulary entries, release orphaned media safely **after** commit. Do not delete shared Topics or referenced audio assets.

Both drafts and published items may be Favorites. Item language is fixed once created in the MVP (changing language means a new item), particularly after publication and vocabulary association. Published state is determined consistently using `published_at` and matching domain constraints; an incomplete or corrupt asset is never presented as valid Reader media.

## 6. Language-aware segmentation and synchronized TTS

### Registry and offset contract

```go
// Shape only; implementation details remain package-private.
type Segmenter interface {
    ID() string                  // e.g. "zh-cn-phrase", "ko-kr-phrase"
    Version() string             // immutable semantic strategy version
    Segment(source string) ([]Segment, error)
}

type Segment struct {
    TextStartUTF16 int // offsets in the exact input string, [start, end)
    TextEndUTF16   int
    Text          string
}
```

- **MVP strategies:** `zh-CN` and `ko-KR` are separately implemented and golden-tested. Chinese rules account for punctuation and writing without interword spaces; Korean rules account for Hangul spacing, endings, punctuation and mixed scripts. **Japanese is post-MVP and requires its own strategy.** Do not reuse Chinese segmentation implicitly.
- **Critical Go pitfall:** Go string indexes are **UTF-8 byte offsets**, not JavaScript/Azure-facing UTF-16 offsets. Build and test a reversible byte/rune/UTF-16 index map; handle surrogate pairs (emoji), combining marks, Hangul syllables, CJK punctuation and paragraphs. Never split an invalid surrogate boundary or normalize exact source text before aligning it.
- Preserve exact text and all whitespace/untimed gaps. Store segments as ordered nonoverlapping **UTF-16 half-open ranges** plus measured millisecond timing; validate `source[range] == segment.Text` using the explicit offset converter, not native Go slice indexing.
- Persist strategy ID + version in the job and asset, and include **language, provider, voice, rate, exact source, strategy ID/version, fingerprint schema version** in the new generation fingerprint. Do not silently re-fingerprint or resegment already published legacy Mandarin audio; retain a legacy validator/fingerprint format for old assets.
- Segmentation defines which text runs can be highlighted; it **does not** invent TTS word timestamps. If provider boundaries cannot be safely mapped/coalesced to those segments or audio duration cannot be measured reliably, mark generation `FAILED` with an alignment error rather than emitting misleading `READY` timing.

### Azure adapter and the Go feasibility gate

`TTSProvider` must offer `ListVoices(ctx, languageCode)` and `Synthesize(ctx, immutableRequest)` returning MP3 output, provider timing/boundary events and typed errors. Voice catalog entries have stable provider/voice identity and language compatibility. Azure credential/region configuration lives only on the backend.

- The **Azure Speech SDK has Go support and documents synthesis `WordBoundary` events**. Before replacing the old Node adapter, build a short **real Azure integration spike** on the target OS: generate MP3 and capture source offsets/timing for both `zh-CN` and `ko-KR`, including SSML escaping and newlines. Verify SDK runtime/CGO/platform dependencies, cancellation and concurrent invocation behavior. Official reference: [Microsoft Speech SDK synthesis events](https://learn.microsoft.com/azure/ai-services/speech-service/how-to-speech-synthesis).
- The TTS REST API can synthesize speech, but **audio-only REST is not automatically an equivalent replacement** for SDK word-boundary/alignment events. Do not switch transport until the alignment requirement is proven. Use a mock/fake provider in tests; real Azure requests are explicit and are not part of ordinary CI.
- Validate actual persisted MP3 content, decoded/measured playback duration and output boundaries; the Go standard library does **not** decode MP3. Choose and verify a decoder/probe implementation separately, and test provider-reported vs measured length before declaring READY.
- Current initial bounds from Mandarin Reader: source `≤ 1,500 UTF-16 code units`, synthesis rates `0.75/1/1.25` if supported per voice, configurable concurrent jobs (initially 2), and no implicit chunking. Reassess on real Chinese/Korean samples before raising limits; never truncate source.
- Use explicit error categories (`TTS_UNAVAILABLE`, `TTS_PROVIDER_ERROR`, `ALIGNMENT_UNRELIABLE`, `STORAGE_ERROR`, `GENERATION_INTERRUPTED`), not raw SDK error strings in responses.

## 7. Persistent generation worker and recovery

- Store a generation row as **PENDING** with immutable request snapshot and idempotency record **before** returning `202`. The Go worker takes bounded capacity, atomically claims an eligible row, sets **PROCESSING**, records heartbeat/start, and calls Azure **outside DB transactions**.
- A successful job writes to a staging file, validates MP3, measures duration, validates source/timing alignment and promotes an immutable candidate file; then it transactionally creates `AudioAsset` and marks the job **READY**. On failure it records **FAILED** with a typed code and safely cleans staged bytes.
- Persisted statuses are `PENDING`, `PROCESSING`, `READY`, `FAILED`. **OUTDATED** is a computed target-vs-latest-fingerprint condition, not a separate job status. Restart-interrupted work is `FAILED` + `GENERATION_INTERRUPTED` (or a documented unresolved state if safe determination is impossible); **never** auto-repeat a potentially billed request after a crash.
- Startup reconciles stale PENDING/PROCESSING rows and orphan files before declaring ready; distinguish jobs definitely never submitted from those that might have reached Azure. Single worker process is the deployment assumption; readiness, DB transactions and unique constraints still prevent duplicate admission.
- Background cleanup uses bounded batches and checks active item assets, revision bases, latest draft/revision candidates, unexpired previews and in-use file leases before release. Separate **DB references and filesystem cleanup**: not one atomic transaction. Avoid unbounded directory scans or a DB query per media file.
- Configurable worker limits protect accidental Azure costs; they are **not** SaaS entitlements or subscription quotas.

## 8. Local media and cover images

| Root-relative path | Purpose |
|---|---|
| `staging/{random}.part` | Untrusted/in-progress bytes; never served. |
| `temporary/audio/{generationId}.mp3` | Unclaimed, expiring preview. |
| `audio/{audioAssetId}.mp3` | Immutable attached candidate/published recording. |
| `temporary/images/{imageId}.{ext}` | Unattached, expiring cover. |
| `images/{imageId}.{ext}` | Attached item cover. |

- `MEDIA_ROOT` is absolute and stable. Generate path keys server-side, reject `..`, traversal and symlinks/junction escape; safely open/stream only known asset IDs. Do not trust user filenames or expose absolute paths in API responses.
- **New/replacement cover size must be ≤ 1,000,000 bytes**, measured from encoded bytes. Verify actual JPEG/PNG/WebP signature and decoded dimensions (inherited upper bound 4096×4096 unless 01 changes it). At exactly 1,000,000 bytes size alone is valid; 1,000,001 is rejected. Existing larger legacy covers remain readable until reviewed.
- Bound upload bodies/streaming, validate before promotion, and never replace an old cover until save succeeds. Use atomic rename on same filesystem, explicit copy/lease when needed, and after-commit cleanup; recover from disk-full/corrupt files.
- Audio endpoints support byte-range responses, safe download filenames and asset-pinned URLs. No cloud storage, public object URLs or image proxy service in MVP.

## 9. Language-scoped vocabulary and topic behavior

- Vocabulary identity: **`(user_id, language_code, normalized_word_key)`**. A Mandarin word and a Korean word with identical display spelling are separate. Preserve spelling, optional **`reading` + `reading_system`** (rather than requiring pinyin), meaning/note, manual Learning/Acquired status and source references.
- Reader collection explicitly links a selected phrase to an existing word without overwriting its status/metadata or creates word + reference atomically. Enforce owner/language equality between a word and a referenced item. Manual links are the correctness baseline.
- Historical Chinese exact-match annotation and automatic reconciliation may be retained **only after explicit language-specific tests**; do not run the Chinese substring matcher on Korean (particles and inflections can create false associations). No automatic morphology or acquired-status inference. Publication must not fail merely because an **unsupported optional auto-matcher** is absent.
- Topics remain **user-global**, with no `language_code`; reused by items and words in both languages. Topic deletion affects joins only; default protection and palette contrast remain as defined in 01.
- Library queries always filter by explicit language and can combine Favorites/status/difficulty/topics/title search, stable sorting and pagination. Return related metadata in batches. Vocabulary searches and global counts are scoped to active language.

## 10. Startup, operations, quality and rollout

1. Validate typed configuration and safe local bind/media root; connect to the Dockerized PostgreSQL service and verify schema migration version.
2. Seed/resolve exactly one stable local user, the `zh-CN`/`ko-KR` supported-language catalog and default Topics **idempotently**.
3. Reconcile interrupted jobs and storage references; start bounded worker/cleanup; expose `/health/live` and `/health/ready` (DB, schema and storage availability). Missing Azure credentials **disable generation but do not block reading existing content**.
4. Serve same-origin SolidJS build and `/api/v1` with graceful shutdown. No auth, payment, AI or post-MVP route is an MVP startup prerequisite.
5. Test unit-level segmentation/UTF-16 mapping, state transitions, fingerprints, normalization, validation and file safety; test Gin API with mocks; test actual SQL migrations and sqlc queries against disposable PostgreSQL, including race/rollback cases; test Reader→API contract with SolidJS.
6. Verify restart and full **`pg_dump` plus local asset backup/restore** (and required config) in a fresh Docker volume before calling the local data recoverable. Never use `docker compose down -v` as an ordinary shutdown.
7. Benchmark actual Library pagination, vocabulary filtering, concurrent requests, segmentation and generation worker overhead on representative content. Go/Gin/sqlc are selected for lean execution and control, **not a guarantee that network TTS or filesystem I/O will be faster**.

### Migration from Mandarin Reader

1. Inventory existing Prisma schema, migration SQL and on-disk files. Design equivalent PostgreSQL DDL and **sqlc queries**, not a mechanical translation of ORM objects.
2. Preserve the original user UUID, item/word/topic/asset IDs and timestamps wherever possible; add language scope (`zh-CN` on historical data), item `is_favorite=false` and optional generic reading fields. Preserve legacy audio fingerprint/segmentation identity.
3. Review legacy difficulty values; **do not automatically map `native` → `advanced`**. Preserve legacy oversize covers and exact text. Backfill cross-language-safe constraints and join references only after validating data.
4. Run checks comparing counts, checksums, published recordings, word links, metadata, previews and Reader offsets; test a nonempty backup/restore. Roll out Go backend against a **copy**, not the only original local database.

## 11. Open implementation choices (not reversals of approved architecture)

| ID | Still to decide / verify |
|---|---|
| BE-OPEN-01 | Exact Go, Gin, pgx, sqlc, PostgreSQL versions, and which lightweight SQL migration runner. |
| BE-OPEN-02 | Host-run Go service vs Compose-run Go service; platform-specific install/service lifecycle. |
| BE-OPEN-03 | Azure Go SDK integration and word-boundary quality/platform runtime for Chinese + Korean. |
| BE-OPEN-04 | Exact segmentation rules/versions and Go UTF-16 ↔ SSML/provider-offset mapping. |
| BE-OPEN-05 | MP3 duration decoder/probe choice; validated timing precision and audio chunking policy. |
| BE-OPEN-06 | Whether optional Chinese automatic vocab matching passes MVP tests; Korean fallback remains manual. |
| BE-OPEN-07 | Local backup/restore packaging and whether a one-click UI follows the first stable release. |

**Maintenance:** When updating `02-taotale-decision-register.md`, record **Go + Gin + pgx/v5 + sqlc** as the selected backend stack and mark Node/NestJS/Prisma as superseded. Update OpenAPI and `05-taotale-database.md` together whenever DB/API invariants change. No AI implementation is implied by preserving its adapter design.
