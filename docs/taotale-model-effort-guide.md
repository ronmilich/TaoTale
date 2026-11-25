# TaoTale — AI model and reasoning effort guide

**Source:** [Canonical implementation roadmap](taotale-roadmap.md), revised 25 November 2026 (app-local Iris UI). All 61 milestone IDs and titles are retained.

**Purpose:** Choose the model and reasoning effort for each milestone when assigning implementation work to an AI coding agent. This is an advisory lookup table, not a second roadmap or progress tracker. The canonical roadmap owns dependencies, scope, exit criteria and status.

## 1. Model and effort policy

Use the exact model names below in the Codex model picker, where available. The API identifiers are included to remove ambiguity; Codex availability and supported effort settings depend on your account and client.

| Exact model | API model ID | Effort | When to use |
|---|---|---|---|
| GPT-6.2 Sol | `gpt-6.2-sol` | Medium (`medium`) | Bounded implementation, ordinary CRUD, native controls and established contracts. |
| GPT-6.2 Sol | `gpt-6.2-sol` | High (`high`) | Cross-module logic, meaningful edge cases, architecture, visual direction or integration testing. |
| GPT-6.2 Astra | `gpt-6.2-astra` | Extra High (`xhigh`) | High-consequence invariants, uncertain technical feasibility, concurrency, recovery, data migration or security. |

**Default for TaoTale:** GPT-6.2 Sol. Reserve GPT-6.2 Astra for the critical milestones listed below. If a Sol task exposes a difficult correctness problem, escalate to Astra High before considering Extra High. If Astra is unavailable, use Sol Extra High where supported and arrange an additional independent review for high-risk changes.

These are recommended starting settings, not measured minimum requirements, token budgets, price estimates or guaranteed model availability. Extra High is justified by correctness risk, not simply task size. Use Sol Medium for routine follow-up edits after a high-risk design is proven. The existing milestone-level effort recommendations are retained.

**Official model references (checked 25 November 2026):** [OpenAI model selection](https://developers.openai.com/api/docs/guides/model-selection), [GPT-6.2 Astra and Sol announcement](https://openai.com/index/introducing-gpt-6-2-astra-sol/), [reasoning effort guide](https://developers.openai.com/api/docs/guides/reasoning), [model release notes](https://help.openai.com/en/articles/9624314-model-release-notes).

## 2. MVP milestones (TAO-01–51)

### Phase 0 — Preparation and de-risking

| ID | Task | Recommended model | Effort | Reason |
|---|---|---|---|---|
| TAO-01 | Legacy inventory and migration strategy | GPT-6.2 Sol | High | Legacy inventory, migration strategy and backup verification; irreversible data-loss risk. |
| TAO-02 | Specification reconciliation | GPT-6.2 Sol | High | Reconcile cross-document decisions, references and conflicting open IDs. |
| TAO-03 | Azure Go SDK feasibility spike | GPT-6.2 Astra | Extra High | Uncertain Azure Go SDK behavior, actual bilingual offsets, cancellation and MP3 evidence. |

### Phase 1 — Foundations

| ID | Task | Recommended model | Effort | Reason |
|---|---|---|---|---|
| TAO-04 | Repository and tooling foundation | GPT-6.2 Sol | Medium | Routine repository layout, pinned tooling, CI scaffold and dependency checks. |
| TAO-05 | Local Docker PostgreSQL and media root | GPT-6.2 Sol | High | Persistent volumes, safe media paths and loopback-only deployment. |
| TAO-06 | Go/Gin backend skeleton | GPT-6.2 Sol | High | Security-sensitive HTTP middleware, strict decoding and lifecycle management. |
| TAO-07 | SQL migrations, sqlc and seeds | GPT-6.2 Sol | High | SQL constraints, migrations, generated queries and deterministic seed behavior. |
| TAO-08 | OpenAPI contract pipeline | GPT-6.2 Sol | High | Canonical OpenAPI design, generated types and contract conformance. |
| TAO-09 | SolidJS client scaffold | GPT-6.2 Sol | Medium | Established Vite/Solid/Router setup, typed fetch and validated local preferences. |
| TAO-10 | App shell, routing and Iris theme | GPT-6.2 Sol | High | Set the visual direction: polished Iris tokens, native controls, layout and theme stability. |

### Phase 2 — Text and audio core

| ID | Task | Recommended model | Effort | Reason |
|---|---|---|---|---|
| TAO-11 | Unicode offsets and fingerprints | GPT-6.2 Astra | Extra High | UTF-8/UTF-16 boundary correctness and byte-exact legacy fingerprint compatibility. |
| TAO-12 | Segmentation registry and Mandarin strategies | GPT-6.2 Sol | High | Versioned Mandarin rules and exact reconstruction using legacy/new golden vectors. |
| TAO-13 | Korean segmentation strategy | GPT-6.2 Sol | High | Independent Korean punctuation rules, mixed scripts and edge-case fixtures. |
| TAO-14 | TTS provider adapter, voice catalog and mock provider | GPT-6.2 Sol | High | Provider adapter, cancellation, voice constraints and deterministic mock integration. |
| TAO-15 | MP3 measurement and alignment validation | GPT-6.2 Astra | Extra High | Real MP3 duration, SSML mapping, boundary timing and strict READY validation. |

### Phase 3 — Items and generation backend

| ID | Task | Recommended model | Effort | Reason |
|---|---|---|---|---|
| TAO-16 | Library item and draft API | GPT-6.2 Sol | High | Versioned PATCH semantics, ownership/language constraints and deterministic paging. |
| TAO-17 | Topics API | GPT-6.2 Sol | Medium | Bounded Topic CRUD, uniqueness, persisted colors and protected defaults. |
| TAO-18 | Generation admission and idempotency | GPT-6.2 Astra | Extra High | Concurrent admission, durable replay/tombstones and prevention of duplicate paid calls. |
| TAO-19 | Generation worker, READY pipeline and job recovery | GPT-6.2 Astra | Extra High | Worker claims, crash recovery and consistency between immutable files and SQL. |
| TAO-20 | Audio streaming, downloads and temporary previews | GPT-6.2 Sol | High | Byte ranges, asset-pinned URLs, preview expiry and atomic claiming. |

### Phase 4 — Create, publish and read

| ID | Task | Recommended model | Effort | Reason |
|---|---|---|---|---|
| TAO-21 | Editor: new item and drafts | GPT-6.2 Sol | High | Editor modes, saved baselines, explicit save and dirty-navigation behavior. |
| TAO-22 | Library list (basic) | GPT-6.2 Sol | Medium | Polished responsive Iris cards, pagination and loading/empty/error states. |
| TAO-23 | Editor: audio generation and preview | GPT-6.2 Sol | High | Generation state machine, polling, idempotency recovery and candidate preview. |
| TAO-24 | Publication | GPT-6.2 Astra | Extra High | Atomic publication, version checks and validation of exact matching READY audio. |
| TAO-25 | Reader page and transcript rendering | GPT-6.2 Sol | High | Exact transcript rendering, stored UTF-16 spans and text-only fallback. |
| TAO-26 | Audio player and synchronized highlighting | GPT-6.2 Astra | Extra High | Native audio clock, seek/selection conflicts, timing index and reactive performance. |
| TAO-27 | Reader actions: Favorite, Notes, Download | GPT-6.2 Sol | High | Independent Favorite writes, Notes saves and uninterrupted Reader playback. |

### Phase 5 — Library completeness

| ID | Task | Recommended model | Effort | Reason |
|---|---|---|---|---|
| TAO-28 | Topics UI | GPT-6.2 Sol | Medium | Native checkbox/topic selection, styled chips, swatches and basic CRUD UI. |
| TAO-29 | Library search, filters, sort and Favorites | GPT-6.2 Sol | High | URL/filter semantics, stable SQL paging, confirmed Favorites and race-safe updates. |
| TAO-30 | Cover upload pipeline | GPT-6.2 Sol | High | Untrusted file validation, bounded streaming, safe storage and replacement behavior. |
| TAO-31 | Cover UI | GPT-6.2 Sol | Medium | Native file input, thumbnail, inline errors and retained previous cover. |
| TAO-32 | Deletion and downloads in the Library | GPT-6.2 Sol | High | Destructive deletion, transactional joins and safe media release. |

### Phase 6 — Editing published content

| ID | Task | Recommended model | Effort | Reason |
|---|---|---|---|---|
| TAO-33 | Published metadata edit | GPT-6.2 Sol | High | Versioned metadata writes that preserve immutable recordings and synthesis identity. |
| TAO-34 | Working revisions: backend | GPT-6.2 Astra | Extra High | Atomic revision swap, locking, base-asset/version verification and failure isolation. |
| TAO-35 | Working revisions: client | GPT-6.2 Sol | High | Revision UI and dirty guard; preserve published playback and reset asset-specific resume. |

### Phase 7 — Vocabulary

| ID | Task | Recommended model | Effort | Reason |
|---|---|---|---|---|
| TAO-36 | Vocabulary data model and core API | GPT-6.2 Sol | High | Language-scoped uniqueness, normalized keys, reference provenance and race-safe creation. |
| TAO-37 | Vocabulary list, statistics and bulk API | GPT-6.2 Sol | High | Server-side search/stats and transactional, language-safe bulk operations. |
| TAO-38 | Reader vocabulary collection | GPT-6.2 Sol | High | Native DOM selection across transcript gaps, UTF-16 ranges and atomic word linking. |
| TAO-39 | Vocabulary view | GPT-6.2 Sol | Medium | Iris table, detail panel, filters and ordinary bulk-selection UI on an established API. |
| TAO-40 | Reader vocabulary panel | GPT-6.2 Sol | Medium | Straightforward linked-word panel, status updates and display preferences. |
| TAO-41 | Verified Chinese automatic matching *(conditional)* | GPT-6.2 Sol | High | Conditional Chinese matching, deterministic overlap handling and preserved manual provenance. |

### Phase 8 — Settings, resilience and backup

| ID | Task | Recommended model | Effort | Reason |
|---|---|---|---|---|
| TAO-42 | Settings and local data | GPT-6.2 Sol | Medium | Local preferences, sanitized status and honest manual backup instructions; escalate unresolved reset/path decisions. |
| TAO-43 | Cleanup, recovery and integrity checks | GPT-6.2 Astra | Extra High | Cleanup races, file leases, orphan detection and protecting live assets. |
| TAO-44 | Paired backup and restore | GPT-6.2 Astra | Extra High | Consistent database/media backup, binary restore and verified recovery on a fresh volume. |

### Phase 9 — Legacy migration

| ID | Task | Recommended model | Effort | Reason |
|---|---|---|---|---|
| TAO-45 | Legacy import tooling | GPT-6.2 Astra | Extra High | Lossless, idempotent legacy import and owner-approved mapping of unsupported data. |
| TAO-46 | Legacy verification and rehearsal | GPT-6.2 Sol | High | Compare imported data, hashes, offsets and playback against legacy evidence. |

### Phase 10 — Hardening and MVP release

| ID | Task | Recommended model | Effort | Reason |
|---|---|---|---|---|
| TAO-47 | Visual polish and supported-device QA | GPT-6.2 Sol | High | Visual review of all five MVP screens in Iris light/dark and desktop/tablet layouts. |
| TAO-48 | Performance baselines and budgets | GPT-6.2 Sol | High | Representative workload, measurement isolation and performance regression budgets. |
| TAO-49 | Fault-injection, security and privacy verification | GPT-6.2 Astra | Extra High | Fault injection, loopback trust boundary, hostile input and privacy/security verification. |
| TAO-50 | Real Azure bilingual release smoke | GPT-6.2 Sol | High | Real Azure bilingual end-to-end evidence and human listening check; not a mock pass. |
| TAO-51 | MVP acceptance, release report and cutover | GPT-6.2 Astra | Extra High | Cross-system release evidence, unresolved blockers and reversible migration cutover. |

## 3. Post-MVP milestones (TAO-52–61)

Deferred until scope approval and completion of the MVP. Recommendations are provisional; reconsider when each task is scheduled.

### Phase 11 — Post-MVP

| ID | Task | Recommended model | Effort | Reason |
|---|---|---|---|---|
| TAO-52 | Post-MVP scope approval and ordering | GPT-6.2 Sol | High | Product trade-offs and explicit post-MVP scope decisions. |
| TAO-53 | Dashboard | GPT-6.2 Sol | Medium | Compose Dashboard from existing language-scoped Library and vocabulary data. |
| TAO-54 | Vocabulary Practice | GPT-6.2 Sol | Medium | Simple manual recall/reveal and explicit vocabulary status changes. |
| TAO-55 | AI provider re-validation spike | GPT-6.2 Sol | High | Revalidate provider authorization, credentials and actual runtime behavior. |
| TAO-56 | AI text-generation backend | GPT-6.2 Astra | Extra High | Provider request safety, structured validation, durable idempotency and paid-call controls. |
| TAO-57 | AI workspace UI (`/generate`) | GPT-6.2 Sol | High | Candidate review/editing and reuse of existing draft/TTS/publication lifecycle. |
| TAO-58 | Language Islands Builder | GPT-6.2 Sol | High | Design domain reuse and rehearsal behavior without a parallel publishing pipeline. |
| TAO-59 | Phone and small-screen support | GPT-6.2 Sol | High | Responsive redesign and real-device transcript/audio interaction review. |
| TAO-60 | Japanese (`ja-JP`) support | GPT-6.2 Astra | Extra High | New language strategy, real voice alignment, Unicode/segmentation and regression evidence. |
| TAO-61 | Optional enhancement backlog | GPT-6.2 Sol | High | Default for backlog planning; reclassify each approved feature by actual risk. |

## 4. Allocation summary

| Exact model | Effort | MVP | Post-MVP | Total |
|---|---|---:|---:|---:|
| GPT-6.2 Sol | Medium | 9 | 2 | 11 |
| GPT-6.2 Sol | High | 29 | 6 | 35 |
| GPT-6.2 Astra | Extra High | 13 | 2 | 15 |
| **All milestones** | | **51** | **10** | **61** |

Model allocation by milestone count: **Sol 46/61 (75.4%)**, **Astra 15/61 (24.6%)**. This is not a forecast of token consumption or cost.

## 5. How to use this guide

- Start a fresh coding-agent task for each milestone or a bounded slice of a large milestone. Provide its exact roadmap scope, dependencies, relevant specification files and exit criteria.
- For High/Extra High milestones, inspect the existing implementation first, establish invariants and tests, then implement in small reviewable changes. Use an independent GPT-6.2 Astra High or Extra High review for the critical paths (audio validity, concurrent paid requests, publication/revisions, deletion/cleanup, backup/restore, migration and security).
- Escalate Medium work if it reveals unclear contracts, race conditions, persistent data changes, or repeated failed attempts. Do not increase reasoning effort merely to polish CSS: inspect screenshots and refine the Iris design with targeted iterations.
- Keep ordinary controls native and app-local. Do not reintroduce UI library dependencies, release gates, custom generic behavior engines or formal accessibility certification. Product JavaScript for selection, playback, state and requests remains necessary.
- Run the required tests and record actual results in the canonical roadmap only. Do not mark a task complete merely because the model generated code. Paid Azure requests require owner approval; use mock TTS for routine CI. Rehearse migrations and restores on disposable copies.
- TAO-41 is conditional; it may close as not enabled. TAO-61 is a placeholder for individually scoped enhancements, not authorization to implement all of them.

### Reusable coding-agent instruction

```text
Implement milestone TAO-XX from docs/roadmaps/taotale-roadmap.md.
Read its scope, dependencies, exit criteria and relevant TaoTale specifications.
Inspect the current repository and confirm prerequisites before changing code.
Keep the implementation small, app-local and consistent with the Iris design.
Implement only the milestone scope; run the required tests and report actual results.
Record blockers and decisions rather than silently changing the product scope.
Update the canonical roadmap status only when the exit criteria are verified.
```

**Important:** The effort recommendations do not imply that any milestone has already started or passed. This document can live alongside `taotale-roadmap.md` in `docs/roadmaps/`; it does not replace that file or recreate the retired implementation-task checklist.
