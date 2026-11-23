# 13 — TaoTale rebuild plan, adopted decisions and remaining choices

**Status:** Implementation roadmap and **decision-boundary document**, revised 23 November 2026. This **replaces** the old comparative NestJS/Go/Rust and React/Solid review: the major choices have been made. Remaining questions are implementation details or future feature decisions, **not** an invitation to silently re-open the agreed technology or MVP scope.

**Normative authorities:** [01 — Product](01-taotale-overview-and-features.md), [02 — Decision register](02-taotale-decision-register.md), [03 — Client](03-taotale-client.md), [04 — Backend](04-taotale-backend.md), [05 — Database](05-taotale-database.md), [06 — Alignment](06-taotale-segmentation-and-audio.md), [08 — Design system](08-taotale-design-system.md).

## 1. Locked project direction

| Decision | Adopted TaoTale approach | Superseded historical assumption |
|---|---|---|
| Product and audience | **Personal, locally hosted, one persistent User** | Public/commercial multiuser SaaS, onboarding, account flows, subscriptions and billing. |
| Learning languages | **Simplified Mandarin `zh-CN` + Korean `ko-KR`** initially, English app interface | Mandarin-only schema and assumptions; unapproved automatic support for Japanese/other languages. |
| Client | **SolidJS + TypeScript + Vite + `@solidjs/router`** | React/MUI/Zustand/React Hook Form/TanStack Query/React Router defaults. |
| UI library | TaoTale **consumes independently versioned npm-published SolidJS-only component package** | Building all generic components inside TaoTale; framework-agnostic component API. |
| UI behavior/styling | npm package owns generic accessibility/overlays/transitions/theming hooks, implemented in-house (TT-032); TaoTale owns Quiet Study and domain UI | Direct app use of **Kobalte** or **Tailwind**, and Kobalte inside the package. |
| Backend | **Go modular monolith + Gin** with adapters and persistent in-process jobs | NestJS/Node as selected runtime, Go/Rust open evaluation. |
| Database | **PostgreSQL locally in Docker**, `pgx/v5` + **sqlc** + reviewed SQL migrations | Prisma ORM/migrations and managed SaaS databases. |
| Media | Local files + metadata/segments in PostgreSQL; paired recoverable backups | Hosted object storage and assuming DB-only backups suffice. |
| Speech | **Azure TTS behind a provider adapter**; validate provider word boundaries and MP3 timing for each supported language | Assuming Node Azure timings/UTF-16 behavior transparently transfer to Go. |
| Segmentation | **Language-specific, ID+version-pinned Chinese and Korean strategies** | Applying one Chinese punctuation algorithm to all languages. |
| Vocabulary / Topics | Vocabulary **user+language**; Topics **user-global** | One user-wide spelling namespace or Chinese-specific `pinyin` as universal metadata. |
| Favorites | `library_items.is_favorite` on draft/published item | No Favorites, or separate collection table without a demonstrated need. |
| Difficulty | Optional **Beginner / Intermediate / Advanced** | Native or a mandatory HSK/TOPIK proficiency mapping. |
| Covers | New/replacement encoded image **≤1,000,000 bytes** | Old 5 MiB limit; silently deleting oversized legacy covers. |
| Supported displays | **Desktop and tablet-sized** web layouts, keyboard/zoom | Phone-specific layouts and cross-device deployment assumed in MVP. |
| MVP modules | Library, Create/Edit, Reader, Vocabulary, Topics, Settings, Favorites, **manual Azure audio synthesis** | Dashboard, Vocabulary Practice, AI Text/Audio Workspace or Language Islands as MVP. |

**Performance objective:** Lean Go backend, explicit SQL and Solid fine-grained updates are adopted as architectural choices, **not a claim of measured superiority**. Measure real latency, CPU, RSS, bundle size and browser responsiveness. Azure provider latency and disk I/O can dominate total time.

## 2. Non-negotiable data and interaction invariants

1. **One stable internal User** resolved by the Go service. Do not accept arbitrary `userId` from the client; no authentication screens in MVP, and no unauthenticated LAN/public exposure.
2. Original source **must not be trimmed/normalized** for storage, hashing, timing or display. All source offsets in persisted segments/API/Reader are documented **UTF-16 half-open** ranges; Go byte/rune offsets require explicit conversion.
3. Each saved Library item has one language. Publication requires **exactly matching READY, decoded and validated** immutable audio with language/provider/voice/rate/strategy ID+version/fingerprint identity. Preserve old published alignment under its original version.
4. A draft can be incomplete, previews are temporary, and a working revision never mutates the active pair until publish. Failures, crashes or stale writes do not corrupt the previous published Reader.
5. Metadata fields (Favorite, title, difficulty, Topics, description, Notes, cover) do not invalidate audio. Favorite is an **idempotent set-value** write that does not change `updatedAt`/content version or default newest order.
6. Vocabulary uniqueness is **(user, language, normalized key)**; same-language references only. Manual collection/status remains baseline for both languages. Chinese automatic matching is optional pending verification; **Korean morphological matching is not implied**.
7. Topics are owner-global, default Topics protected. Covers have a **1,000,000-byte inclusive** new/replacement limit; existing larger legacy images must survive migration.
8. Client imports the **public npm UI package**, not Kobalte or Tailwind and not unpublished internals. Generic component accessibility and transitions belong to that package; Reader synchronization stays app-specific.
9. Paid Azure requests are **explicit**, durable and idempotent; no automatic repeat on timeout or restart. A local-only app can still send text to Azure when the user selects Generate Audio.
10. Docker database and on-disk media must be backed up as a **consistent pair** and restore-tested on a fresh instance.

## 3. Implementation sequence — vertical slices with evidence

### Stage 0 — Inventory and protect source data

- Identify original Mandarin Reader schema/migrations/asset directories and existing UI behavior. Backup the actual nonempty PostgreSQL database and all MP3/cover media before modifying the original. Inventory UUIDs, source texts, legacy `native` values, oversize covers, existing pinyin, timing versions, replay rows and manual/auto vocabulary references.
- Decide whether this is an **import** to a fresh TaoTale schema or an in-place SQL upgrade; prefer a **disposable copy** for rehearsal. Preserve all old audio bytes and source offsets, even when the new algorithm differs.
- Freeze old golden vectors/fixtures and baseline measured performance to compare with the Go/Solid rewrite. No old test result is credited as a new implementation pass.

### Stage 1 — Runnable foundations and UI-package boundary

- Create Go/Gin module, structured configuration/error handling, SQL migration runner, Docker PostgreSQL volume, `pgxpool`, sqlc input/config/codegen; seed stable User and `zh-CN`/`ko-KR` languages/Topics.
- Create SolidJS/Vite/Router shell with `/library` landing, typed `fetch`, URL-backed filters, small local preferences and no account/AI/phone routes. Build the standalone UI npm package **separately** and install a real version/build in TaoTale.
- Verify published-package ESM/types/CSS, single Solid runtime, theme customization, Dialog/Tooltip/Menu keyboard/focus/transition behavior and **no direct app Kobalte/Tailwind imports**. Bootstrap only the component set required for the first real TaoTale view; do not wait for a huge speculative catalog.

### Stage 2 — Highest-risk cross-language audio feasibility gate

- Prototype **real Azure Go Speech SDK** synthesis on the target OS for Mandarin and Korean. Capture provider boundary events, SSML escape mappings, MP3 bytes, decoded measured duration, cancellation, platform native dependencies and concurrent calls.
- Implement **two distinct versioned segmentation strategies** and UTF-8/UTF-16 conversion with golden vectors. Validate exact text/timing and reject failed alignment. **Do not proceed by fabricating timestamps for Korean** if provider behavior fails the gate.
- Preserve a compatibility reader for legacy Chinese segmentation/fingerprint format and demonstrate old published asset playback unchanged.

### Stage 3 — Manual content and Reader end-to-end

- Implement item Draft, image upload cap/signatures, Topics, voice catalog, preview/saved generation jobs, idempotency, persistent worker, immutable media and SQL transactions. Deliver **Create → Generate → Publish → Reader → Download** with real persistence, not UI-only mocks.
- Add version conflicts, working revisions and metadata edits; guarantee old published pair remains playable after failed/aborted candidates. Add safe byte-range media streaming, source-accurate Reader timed segments and fallback text.
- Add Library filters/sorting/pagination, **Favorites** and correct handling of newest order/filtered result removal. Deliver browser reload/restart, dirty guard and upload failure states before breadth.

### Stage 4 — Vocabulary, Settings, safety and performance

- Deliver per-language manual vocabulary, normalized uniqueness, references, reversible Learning/Acquired status, per-language list/stats, shared Topic catalog, Reader selection and popups. Add **only verified** per-language auto-matching; do not block Korean publication on unavailable auto-sync.
- Implement Settings with client-local language/voice/rate/theme/playback/popup prefs and **truthful** local-data/backup guidance. Implement bounded cleanup, health checks, read-only integrity and safe startup/job recovery.
- Benchmark Go metadata API, SQL queries, Solid UI and Reader updates with representative dataset; run accessibility, desktop/tablet, fault injection and paired PostgreSQL+media restore. No paid calls in ordinary CI.

### Stage 5 — MVP acceptance and cutover

- Lock/update **OpenAPI 3.1** target contract, generate typed client, verify Gin conformance and supported error/media semantics; keep source code, SQL, contract and docs synchronized.
- Test an isolated import/upgrade from populated original records: compare UUIDs, words, Topics, timestamps, Favorite default, language backfill, old audio bytes/segments and unsupported `native` handling. Backups remain available for rollback.
- Record the release report with real test counts, browser matrix, npm UI-package version, perf metrics, real Azure Chinese+Korean validation, security checks and restored fresh instance. Do not cut over the only working copy on unverified assumptions.

## 4. Open implementation decisions (not permission to change product scope)

| ID | Decision needed | Recommended next step |
|---|---|---|
| OPEN-01 | Host-run Go vs Compose-run Go; PostgreSQL mapping/volume and target OS service lifecycle | Choose shortest reliable local setup; enforce loopback and persistence in either model. |
| OPEN-02 | Go/Gin/pgx/sqlc/SQL-migration versions and OpenAPI Go codegen/validation tooling | Pin versions only after a working minimal Gin+PostgreSQL vertical slice. |
| OPEN-03 | Azure Go SDK runtime dependencies and real Chinese/Korean word-boundary quality | Make a two-language experimental synthesis + measured MP3 + alignment fixture **before** full backend port. |
| OPEN-04 | New Mandarin/Korean segmentation boundaries/version and exact source limit/chunking | Approve golden outputs; never silently modify legacy published alignments. |
| OPEN-05 | Go MP3 decoder/probe and cross-platform audio duration evidence | Select based on output verification and packaging, not library count alone. |
| OPEN-06 | Automatic exact vocabulary matching enabled for Chinese? What safe behavior, if any, for Korean? | Begin with manual linking; enable only tested strategy, expose unsupported sync honestly. |
| OPEN-07 | Migration treatment for legacy difficulty `native` and old >1 MB covers | Report rows; preserve data; explicitly map or set null after review. Never delete old images automatically. |
| OPEN-08 | New API canonical `/topics` path vs old `/tags`, aliases and DTO compatibility | Lock in TaoTale OpenAPI before generating clients; maintain migration mapping. |
| OPEN-09 | Exact UI npm package name, licensing, CSS override API, compatible Solid peer version | Decide in **independent UI-library project**; verify installed package behavior in TaoTale. |
| OPEN-10 | Reference tablet viewport/browser matrix, target performance thresholds, OS restore commands | Specify reproducible release environment before acceptance testing. |

**Do not promote** optional `UserLanguagePreference` persistence or local backup UI into required MVP tables/screens without a decision; the current client-local preferences + tested manual restore are sufficient.

## 5. Post-MVP roadmap and optional backlog

**Explicit post-MVP:** Dashboard; Vocabulary Practice; AI Text/Audio Generation Workspace (reuse the chosen AI provider adapter, not a parallel publishing backend); Language Islands Builder; phone-specific UX; Japanese and other languages; SRS/flashcards, scoring, curriculum, automatic difficulty and acquired-status assessment, word-level dictionary/speech, cross-device sync, cloud/SaaS auth and billing.

**Historical optional proposals from Mandarin Reader (not automatically adopted):**

| ID | Future proposal / status |
|---|---|
| FEAT-01 | Reader vocabulary browser: This text vs All words; only after language/source reference definitions are clear. |
| FEAT-02 | Vocabulary CSV/JSON export; protect Unicode/multiline and spreadsheet formula injection. |
| FEAT-03 | Repeat selected segment or A–B audio region using native clock and immutable asset ID. |
| FEAT-04 | Per-recording local bookmarks, invalidated when source recording changes. |
| FEAT-05 | Reader text-size/line-spacing preferences preserving selection and highlights. |
| FEAT-06 | Dirty guard / retained edits in vocabulary editor beyond the minimum state contract. |
| FEAT-07 | Improved local diagnostic and recovery guide for disk/storage/generation failure; basic truthful error/recovery is already an MVP need, richer tooling is optional. |

A public distribution/commercial plan would require **new** authentication/network authorization, HTTPS, privacy/terms, media authorization, cost/tenant controls and operations review; it is not an implicit requirement just because the UI component library is published to npm. The library is a separately distributable product; **TaoTale remains personal/local.**

## 6. Decision governance

- The full approved record with history is [`02-taotale-decision-register.md`](02-taotale-decision-register.md). Change a locked decision only with a new dated entry describing reason, scope, rejected alternatives and affected files/tests.
- Product changes first update 01 and decision register; architecture/contract/persistence changes update 03–06/08/10, UI behaviors 09, operations 11 and tests 12 in the same change. Sources/traceability 14 is updated when authorities change.
- This plan is **not** proof that the Go/Solid migration, real Azure two-language gate, npm UI library or restore procedure has been completed. Track actual evidence separately.
