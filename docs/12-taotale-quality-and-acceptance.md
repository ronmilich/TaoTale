# 12 — TaoTale quality, testing and MVP acceptance

**Status:** Proposed measurable release gates and test strategy for a **new Go/SolidJS implementation** (23 November 2026). **No TaoTale implementation tests or benchmarks were run to produce this document.** Historical Mandarin Reader results are reference evidence only and **cannot** be credited as TaoTale pass results.

**Authority:** [01 — MVP scope](01-taotale-overview-and-features.md), [03 — Solid client](03-taotale-client.md), [04 — Go backend](04-backend.md), [05 — Database](05-database.md), [06 — Audio](06-taotale-segmentation-and-audio.md), [08 — UI library](08-taotale-design-system.md), [09 — States](09-taotale-ui-states.md), [10 — API](10-taotale-api.md), [11 — Operations](11-taotale-operations-and-security.md).

## 1. Scope of release certification

The **MVP must demonstrably support**: local Docker PostgreSQL and persistent assets; one seeded owner without authentication; **`zh-CN` and `ko-KR`**; Library of drafts/published items, language filtering and **Favorites**; Create/Edit, Azure TTS jobs/previews/validated publication and revisions; Reader with synchronized highlighting and fallback; manually managed language-scoped Vocabulary and shared Topics; Settings and backup/restore guidance. Desktop and **tablet-sized** browser support, keyboard/focus and zoom are in scope.

**Explicitly not required:** Dashboard, Vocabulary Practice, AI Text/Audio Generation Workspace, Language Islands Builder, Japanese, mobile phone layouts, login/logout, subscriptions or billing. **Manual Generate Audio inside the editor is required** and must not be mistaken for the deferred AI workspace.

## 2. Nonfunctional acceptance gates (adapted NFR-01–16)

The numeric values inherited from Mandarin Reader below are **provisional starting budgets**, not newly measured Go/Solid performance or guaranteed Azure latency. Final target hardware/browser and any changed thresholds must be recorded **before** the release test, not adjusted retroactively to claim a pass.

| ID | TaoTale release requirement / proposed threshold |
|---|---|
| NFR-01 | Metadata API (`GET /items`, item detail, vocabulary list/stats) **p95 ≤400 ms** on reference local dataset excluding audio and external provider time; no errors. Record Go/sqlc query timings. |
| NFR-02 | Production-like local first useful Library/Vocabulary **≤2.5 s**, Reader original text **≤2.0 s** excluding MP3 download. No blank UI while a lazy chunk loads. |
| NFR-03 | Timed Reader highlight agrees with native audio clock at **0.75×/1×/1.25×/1.5×**; event/seek response target **≤150 ms** (allow native seeked convergence). No incorrect text/audio asset pairing. Test Chinese and Korean separately. |
| NFR-04 | **Solid fine-grained updates**: playback clock must not recreate the Reader/transcript or rerender unaffected Library/Vocabulary UI on each tick; no sustained visible input delay **>100 ms** with representative long source/annotations. |
| NFR-05 | **100%** published active audio assets are decodable and exact-source-aligned; corrupt, mismatched or incompletely timed candidates cannot become READY/published. Legacy verified recordings remain playable without resegmentation. |
| NFR-06 | Backend interruption reconciles outstanding jobs without resubmitting possibly billed Azure calls; original published pair preserved. Historical restart-readiness reference **≤30 s**, measured on target machine. |
| NFR-07 | Explicit dirty save/discard/stay guards work for all editor modes and Notes; browser close/reload uses native warning only. No autosave or false Saved feedback. |
| NFR-08 | A **fresh-volume paired restore** recovers PostgreSQL records and referenced local MP3/covers, Notes, Favorites, Topics, language-specific vocabulary and original timing with zero missing active assets. |
| NFR-09 | No private content/secrets in source, bundles, logs or sanitized errors; API + DB bound locally; reject unsafe cross-origin/Host/filename/path cases. Never claim public-network security from local-only tests. |
| NFR-10 | All limits/Unicode edge cases enforced server-side and client-side: **cover ≤1,000,000 encoded bytes**, title/source/word boundaries, actual file signature/dimensions, unsafe path/SSML, invalid locale/voice/difficulty and overlong JSON. No truncation. |
| NFR-11 | Keyboard primary journeys, visible focus, semantic controls/status, focus return/trap, appropriate 4.5:1 normal text contrast and ≥44px targets where applicable. No critical automated/manual accessibility defects in supported layouts. |
| NFR-12 | Desktop **1280/1440px** and representative **tablet-sized** viewport/zoom up to **200%**: no blocking overflow, text selection/focus works, player never covers the final paragraph; **phone QA excluded**. |
| NFR-13 | Same generation idempotency key/intent never invokes Azure twice; failed media/DB transactions, timeouts, restarts and revision attempts leave active asset unchanged. UI reports unknown outcomes accurately. |
| NFR-14 | All required checks green; **≥80% line / ≥75% branch** coverage of testable business-domain code as a starting goal, plus complete critical-invariant scenarios. No skipped test counted as pass. Static/dep security checks for Go, Solid app and independent UI package. |
| NFR-15 | Brand-new and populated PostgreSQL volumes migrate/seed deterministically using versioned SQL; repeat seed preserves IDs and user edits. Mismatch blocks readiness; no Prisma runtime. |
| NFR-16 | At most one active generation/target, bounded concurrency (starting cap 2), visible 429/cost-limit states, cleanup of temporary previews/uploads without deleting active or revision-base assets. No fixed provider-generation completion SLA. |

## 3. Reference data and golden fixtures

Create deterministic fixtures with pinned IDs and expected outputs. **Keep the legacy F1–F7 corpus for parity**, but extend it for multilingual behavior:

| Fixture | Required coverage |
|---|---|
| **F1-CN** | Legacy Chinese exact text and approved v1 offsets; new registered `zh-CN` strategy separately versioned. Includes Chinese commas/full stops/Han-hyphen/quotes. |
| **F1-KO** | Korean Hangul, spaces, particles/endings, terminal punctuation, quotation marks, paragraph breaks, mixed Latin/digits and abbreviations; validate the **actual adopted Korean segmentation policy**, not Chinese fallback. |
| **F2** | CR/LF/CRLF, blank lines, leading/trailing whitespace, repeated delimiters, SSML XML-sensitive characters, emoji and mixed scripts, original source preserved exactly. |
| **F3** | UTF-16 offsets for supplementary Han and emoji, surrogate pairs, combining marks, grapheme boundaries, Go UTF-8 byte ↔ UTF-16 conversion and JSON escaping/fingerprint parity. |
| **F4** | Provider events with missing/out-of-order/overlapping/spurious spans, negative/noninteger timestamps, legitimate gaps, timing outside measured duration and unacceptable drift. |
| **F5** | Valid known-duration MP3, truncated/corrupt/unreadable/stale asset, mismatched fingerprint and generated file absent on disk; active published fallback. |
| **F6** | JPEG/PNG/WebP exact **1,000,000-byte accepted by size** and **1,000,001-byte rejected**, fake signature, decompression/dimensions, filename traversal and existing large legacy cover read. |
| **F7** | Initial scale target: **1,000 published items + 100 drafts, 10,000 vocabulary entries and 100 Topics**, split between `zh-CN`/`ko-KR`, with Favorites, references and stable timestamps. Validate filters, counts and no N+1 queries. |
| **F8** | Two-language isolation: same word string in Chinese/Korean, matching cross-language ref rejected, language-aware voice filtering, disabled language, legacy data `zh-CN` backfill, old `native` difficulty migration review. |
| **F9** | UI-package integration: published CSS/types/exports, compatible single Solid runtime, theme overrides, dialog focus/tooltip keyboard/transition reduced motion and no direct Kobalte/Tailwind import in TaoTale. |

**Mock provider** returns deterministic valid MP3 bytes and controlled aligned events; no paid Azure calls in routine CI. A separately approved **real Azure** smoke must verify actual Chinese and Korean voice events/timing, duration/MP3 decoding, SDK native dependencies and human-audible quality on the target OS. If either language cannot produce reliable timing, it is a release blocker for synchronized playable publication, not a reason to fabricate timestamps.

## 4. Automated test layers

### 4.1 Go unit and service tests

- Pure Mandarin/Korean segmentation strategies, legacy version playback, canonical fingerprint vectors (including original JS-format compatibility), SSML offset translation, UTF-16 mapping, timing validation/rounding/coalescing and measured MP3 evidence.
- Item lifecycle/state/version behavior, dirty synthesis inputs, Favorites independent update, topic default protection/contrast, per-language vocabulary identity and manual provenance, permitted optional auto-match rules and unsupported-language fallback.
- Generation admission/idempotency, one active target job, worker cancellation/crash recovery, bounded cleanup, sanitization, file leases and failed promotion with no published-asset damage.

### 4.2 Real PostgreSQL and Gin integration

- Launch **disposable local PostgreSQL** through Docker, apply actual SQL migrations and generate `sqlc` methods from checked-in queries. Test populated migration and seed twice, FK/unique/constraint behavior, advisory/row locking and rollback.
- Test Library paging/filters/Favorites stable sort; item draft/direct publish/revision lifecycle; byte-range/asset pinning; upload exact size/signature; same-language source links and cross-language rejection; bulk vocabulary operations; Topic CRUD.
- Test same-key request replay, unknown-outcome recovery, restart mid-generation, missing media, stale versions, concurrent Favorites and metadata writes, and preservation of original recording on failed publication. Inspect **actual SQL errors/mapped API responses**, not ORM mocks.
- Test OpenAPI request/response/error/header conformance against **canonical TaoTale YAML** once authored; the old Mandarin Reader YAML alone does not certify new endpoints.

### 4.3 SolidJS app and external UI package

- Solid component tests for language-scoped route/URL canonicalization, resources/stale data/errors, forms/dirty guards, Favorites set-value and filtered result removal, correct speaker/voice options, invalid input and status cache updates.
- Reader tests: active segment changes only when index changes, asset-pinned resume, non-collapsed selection suppresses seek, UTF-16 mixed-node offsets, annotations independent of invalid audio, no transcript or audio remount on playback/theme/dialog changes.
- Separate **npm UI package** tests: accessibility/ARIA/focus, Escape/outside close, keyboard Menu/Dialog/Tooltip, transition cleanup and reduced motion, override/theme behavior, built package ESM/types/CSS and peer dependency isolation. **TaoTale must not import `@kobalte/core` or Tailwind directly.**
- Browser E2E for app + actual Go/PostgreSQL/media path, not only mocked HTTP. Primary smoke Chromium, wider supported engines and tablet-oriented interactions/zoom before release. Use Playwright/axe or equally verified tools without introducing unnecessary runtime dependencies.

## 5. Critical end-to-end acceptance journeys

1. Fresh local startup: Docker DB volume + Go Gin backend + Solid frontend, seeded owner/languages/topics, Library visible **without login**; restart retains all data.
2. Chinese and Korean manual text → Save Draft → provider-backed generation → measured MP3 + exact alignment → Publish → Reader highlights, seek, download.
3. New unsaved preview claims valid draft (or expires); no hidden draft from Generate Audio; duplicate submission with same key does not repeat Azure call.
4. New draft/source edit, published metadata edit, working revision success/failure/discard; original published recording always safe when candidate fails.
5. Favorite draft/published item, filter Favorites only, restart/reopen, unstar removes result while preserving version/audio/newest order.
6. Difficulty only Beginner/Intermediate/Advanced/Unspecified; filter and edit do not regenerate audio; old `native` data migration is reviewed, never silently remapped.
7. Upload valid ≤1 MB cover; reject 1 MB+1 byte/false MIME/invalid dimensions, preserve previous image and older oversized legacy covers.
8. Reader selection creates/links vocabulary in **same language**; Learning/Acquired manual toggle, source refs, Topic assignment, delete item preserving word, cross-language link rejected.
9. Invalid MP3/alignment shows original text with disabled false seeking; independent vocabulary selection remains functional. Offline Azure failure does not prevent previously saved audio playback.
10. Test every dirty navigation choice including Back/tab unload, while generation pending and during failed save. Confirm dialogs, focus recovery and no false success.
11. Pair DB dump and media archive, restore into a **fresh** volume; compare fingerprints, audio bytes, Topics, Favorites, vocabulary statuses and both language segments.
12. Published UI library installed as a dependency; change Quiet Study theme and component overrides without library source edits; Reader maintains playback; keyboard/tablet/zoom acceptance passes.

## 6. Measurement and release evidence

Reference environment: **at least 4 logical CPUs and 8 GiB RAM** as an initial benchmark machine, production-mode local builds, PostgreSQL on Docker with durable volume, pinned Go/Gin/pgx/sqlc, Solid/UI-package/browser versions, exact git commits and fixture generation seed. Record hardware/OS, warm/cold cache, p50/p95, failures, request traces, JS/CSS sizes, input delay, Go RSS/CPU, database queries and media checks. Run tests without competing heavy workloads; do not mingle external Azure latency with metadata API timings.

**Suggested CI stages:** Go formatting/vet/static lint → Go unit/race/coverage → sqlc generation drift and SQL migrations/integration → OpenAPI lint/contract and TypeScript generation freshness → Solid typecheck/build/component tests → UI-package build/accessibility/consumer smoke → browser E2E smoke → dependency/license/secret checks. Release-only: desktop/tablet browsers, real provider smoke, restart/fault, paired restore and benchmark report. Exact commands depend on the chosen repository/tool versions.

**Release report:** commit, migration revision, OpenAPI spec hash, npm UI-package version, actual test pass/fail/skip counts, coverage, lint/audit findings, timing and screenshots/traces, multi-language real-TTS evidence, backup/restore checksums and known limitations. **No unexecuted test or historical report should be recorded as passing.**

## 7. Preserved historical evidence and open gates

The old Mandarin Reader ledger (November 2026) reported scoped tests and one verified Azure example; these are useful regression references, **not proof of a Go/SolidJS/Korean port**. The old dependency/security advisories and framework versions also require fresh assessment for today's dependencies, not automatic inheritance.

**Open before freezing release criteria:** precise accepted Korean segmenter and word matcher; Azure Go SDK platform and boundary fidelity; MP3 decoder/probe; final source length and chunking policy; exact supported tablet/zoom/browser matrix; initial numeric performance budgets confirmed on target PC; binary backup implementation/validation; compiled npm UI package and published version contract.
