# 12 — TaoTale quality, testing and MVP acceptance

**Status:** Target release gates, revised **25 November 2026**. No new Go/Solid TaoTale tests or benchmarks are claimed to have run. Historic Mandarin Reader test counts are not TaoTale results. **Authority:** [01 — Product](01-taotale-overview-and-features.md), [03 — Client](03-taotale-client.md), [04 — Backend](04-taotale-backend.md), [05 — DB](05-taotale-database.md), [06 — Audio](06-taotale-segmentation-and-audio.md), [08 — Iris](08-taotale-design-system.md), [09 — UI states](09-taotale-ui-states.md).

## 1. Release scope

MVP: local persistent Docker PostgreSQL and files, a seeded owner without login, **`zh-CN` and `ko-KR`**, Library/Favorites, Create/Edit/Drafts/Revisions, manual Azure TTS with verified published audio, Reader synchronized highlighting + text-only fallback, manual Vocabulary/Topics, Settings and paired backup/restore guidance. Desktop and tablet browser layouts, plain native controls and Iris light/dark designs are in scope.

Not required: Dashboard, Vocabulary Practice, AI workspace, Islands, Japanese, phone layouts, login/subscription/billing, separate UI library or component SDK, formal accessibility certification and advanced keyboard/overlay behavior. **Manual Generate Audio is still required.**

## 2. Release gates

Numeric values below are provisional inherited starting budgets, not measurements. Confirm reference machine and browsers before calling a release pass.

| ID | Requirement / starting target |
|---|---|
| NFR-01 | Metadata API `GET /items`, item details and vocabulary list/stats **p95 ≤400ms** locally excluding external audio/provider |
| NFR-02 | First usable Library/Vocabulary **≤2.5s**; Reader original text **≤2.0s** excluding MP3 download; useful loading state |
| NFR-03 | Reader highlight agrees with native audio clock at **0.75×/1×/1.25×/1.5×**; seek/display reaction target **≤150ms**; both languages tested |
| NFR-04 | Frequent audio time changes do **not recreate** the transcript or unrelated app views; typing remains responsive on long text |
| NFR-05 | All published active assets are playable and exactly matched/aligned; malformed candidates never become READY/published; legacy recordings remain valid |
| NFR-06 | Restart/interruption recovers job records without repeating uncertain charged Azure requests; preserve active published pair |
| NFR-07 | Explicit save/discard/stay behavior, native tab-close warning, no autosave or false Saved notice |
| NFR-08 | Fresh-volume paired PostgreSQL/media restore recovers every active asset, notes, Favorites, topics, language-specific words and timing |
| NFR-09 | No secrets/private text in logs, frontend bundle or errors; local HTTP/DB bind to loopback; block cross-origin/Host/path abuse |
| NFR-10 | Cover **≤1,000,000 bytes** and valid signature/dimensions, Unicode/source and voice/locale limits validated; no silent truncation |
| NFR-11 | **Basic usability and visual quality:** regular labeled native controls, visible selection/pressed/error states, readable contrast, no clipped text or hidden actions. **No formal WCAG/ARIA audit, focus-trap/roving-tabindex suite or complex component keyboard mechanics** |
| NFR-12 | Iris light/dark layouts checked at **1280/1440px desktop** and a representative **tablet (~768–1024px)**; no layout-blocking overflow, transcript/player collision or lost edits. Phone QA deferred |
| NFR-13 | Same paid generation key/intent never knowingly calls Azure twice; failed transaction/revision keeps prior asset |
| NFR-14 | Domain unit/integration tests green; starting goal **≥80% line / ≥75% branch** of testable core domain code. No UI-library-package release or package-consumer test. Static checks for Go and frontend |
| NFR-15 | Empty/populated local PostgreSQL migrations/seed deterministic; preserve seeded IDs and user data; no Prisma runtime |
| NFR-16 | One active generation per target, bounded worker concurrency (initial cap 2), honest 429/cost/error states and safe cleanup |

## 3. Core fixtures

| Fixture | Coverage |
|---|---|
| F1-CN | Legacy Chinese v1 timing/segmentation plus newly versioned Chinese policy |
| F1-KO | Korean Hangul, punctuation, spacing, particles, mixed Latin/digits, paragraphs and adopted versioned policy |
| F2 | CR, LF, CRLF, blank lines, whitespace, punctuation, SSML XML-sensitive text, emoji; preserve exact original |
| F3 | Go UTF-8 ↔ JS UTF-16 index map, supplementary characters, combining marks and fingerprints |
| F4 | Azure event invalid order/overlap/gaps, invalid timestamps and mismatch with measured duration |
| F5 | Valid MP3, missing/truncated audio, stale fingerprint, invalid/missing released asset |
| F6 | Exact encoded cover threshold **1,000,000 accepted by size / 1,000,001 rejected**, actual MIME/dimensions, old oversize cover |
| F7 | Representative **1,000 published + 100 drafts, 10,000 words, 100 Topics** split between languages |
| F8 | Same spelling in two languages, invalid cross-language links, correct voice options, legacy backfill and `native` difficulty review |
| F9 | **App-local Iris UI:** light/dark CSS tokens, basic Button/Field/Card styles, visually consistent pages, desktop/tablet screenshots, native controls, theme switch with audio preserved; **no UI-package integration** |

**Automated provider tests:** mock returns deterministic audio/timing/errors without paid Azure calls. Separate explicit real Azure smoke must validate Chinese and Korean SDK timing, MP3 duration and audible output. Never invent timing when actual provider integration fails.

## 4. Test layers

### 4.1 Go unit/service

Test Chinese/Korean segmentation, source-index conversion, legacy fingerprints, boundary/event mapping, timing and duration validation; versions and publication; duplicate generation/idempotency; Topics/Favorites/vocabulary invariants; media cleanup/recovery and sanitized failures.

### 4.2 PostgreSQL / Gin integration

Use disposable Docker PostgreSQL with actual migrations and generated sqlc queries. Test owner/locale constraints, paginated filters and stable order, item lifecycle/revisions, media Range, upload validation, vocabulary references, Topic CRUD, concurrent favorites and stale versions, migration/recovery and typed HTTP errors. Validate responses against the final TaoTale OpenAPI contract, not the historical Mandarin Reader YAML.

### 4.3 Solid frontend and visual tests

- Unit/component: page filters/URLs, form dirty/validation logic, Favorite set value, voice list, Reader segment lookup, annotation-source preservation, audio resume, panel/theme changes and error states.
- End-to-end: Go + PostgreSQL + filesystem; Library → Draft → Generate → Publish → Reader → Vocabulary, plus revision safety and backup/restart flows.
- Visual: compare representative screenshots of **all five MVP views** in light and dark at desktop/tablet widths; inspect typography, Iris color consistency, card spacing, text clipping, selected states, forms and empty/error states.
- Basic interaction: check click/touch and normal keyboard use of **native buttons/links/forms**, but **do not introduce specialized component-behavior JS or a full accessibility-testing project**. If a polished modal is needed, prefer native `<dialog>`.
- No separate npm UI-component package, ESM/CSS publication, peer-version isolation, Tooltip/Menu/Popover focus suite, overlay portal testing or external theme override compatibility tests.

## 5. Critical end-to-end user journeys

1. Fresh local start with Docker PostgreSQL, Go and Solid; stable owner; Library available **without login**; restart preserves data.
2. Chinese and Korean text → saved draft → real valid TTS → Publish → synced Reader → audio download.
3. Temporary preview may be claimed or expire; no hidden draft save; same generation key never repeats billable work.
4. Draft/source edit, published metadata edit and working revision; failed/discarded candidate keeps original Reader safe.
5. Favorite draft/published, filter, restart and remove from Favorites without modifying content version, newest order or audio.
6. Difficulty only Beginner/Intermediate/Advanced/Unspecified, independent of audio; legacy `native` is reviewed.
7. Cover validation at 1MB and 1MB+1, preserving existing/legacy cover on invalid replacement.
8. Reader selection creates/links same-language vocabulary; reversible Learning/Acquired; correct Topic and references; deletion preserves words.
9. Bad audio/timing leaves original text readable; vocabulary still possible, no misleading timed seeking.
10. Dirty navigation and failures preserve input; no false confirmation; destructive actions require confirmation.
11. Paired database/media dump restored into fresh local environment; compare MP3 hashes, timing, Favorites and both languages.
12. **App-owned Iris UI** looks coherent in light/dark on desktop/tablet, forms use simple native controls, no external UI package is installed, and theme/dialog/panel changes do not restart Reader playback.

## 6. Evidence, reporting and remaining gates

Capture OS, hardware, app commit, Go/SQL/TypeScript versions, schema revision, browser versions, test counts, real-provider Chinese/Korean evidence, performance samples, screenshot review, local security check and paired restore results. No historical test gets credit as a current pass. Routine CI: Go format/vet/tests, sqlc/migration checks, OpenAPI/types, Solid typecheck/build/component tests, browser core smoke and dependency/secret checks. **Remove UI-package build/publish/consumer/ARIA audit steps from CI.**

The [decision register](02-taotale-decision-register.md#6-implementation-decision-register) records open choices, and the [TaoTale roadmap](taotale-roadmap.md) identifies the milestone that verifies each one.
