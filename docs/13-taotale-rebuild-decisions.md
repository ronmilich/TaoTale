# 13 — TaoTale rebuild plan, adopted decisions and remaining choices

**Status:** Current MVP implementation roadmap, revised **25 November 2026**. **Primary authority:** [01 — Product](01-taotale-overview-and-features.md), [02 — Decision register](02-taotale-decision-register.md), [03 — Client](03-taotale-client.md), [04 — Go](04-taotale-backend.md), [05 — DB](05-taotale-database.md), [06 — Audio](06-taotale-segmentation-and-audio.md), [08 — Iris](08-taotale-design-system.md), [15 — Task checklist](15-taotale-implementation-tasks.md).

## 1. Locked direction

| Area | Approved decision | Explicitly not required |
|---|---|---|
| Product | Personal, local, single persisted user | Commercial SaaS/auth/billing |
| Languages | `zh-CN` and `ko-KR` initially; UI English | Japanese until capability is verified |
| Frontend | **SolidJS + TypeScript + Vite + Solid Router** | React/MUI/React Hook Form/Zustand |
| UI ownership | **All components and styles inside TaoTale frontend**, with small app-local `components/ui` and CSS Modules | Standalone UI npm library, Eroxo integration, package publishing, external component SDK |
| UI mechanics | **Native HTML controls** with little custom behavior; CSS hover/transitions | Kobalte/Radix/complex menu/tooltip/combobox/focus/ARIA/overlay engines |
| Visual direction | **Iris** brand: violet, lavender, periwinkle, orchid, quiet paper surfaces, polished light/dark screens | Old sage/mint Quiet Study palette, generic library theming engine |
| Backend | Go modular monolith, Gin, explicit services/adapters | NestJS/Node backend |
| Data | Local Docker PostgreSQL, `pgx/v5`, `sqlc`, SQL migrations | Prisma/ORM, cloud database |
| Media | Local durable MP3/covers + immutable asset metadata; paired backup | Cloud object storage or DB-only backup |
| Speech | Azure TTS Go adapter, measured MP3, validated timed boundaries | Invented duration, automatic retry of charged requests |
| Segmentation | Tested and versioned Mandarin/Korean policies, legacy audio compatibility | Chinese regex silently applied to Korean |
| Content | Drafts, saved revisions, Reader, Favorites, manual Vocabulary/Topics | Dashboard, Practice, AI workspace, Islands in MVP |
| Quality | Domain correctness and simple app visual/usability review on desktop/tablet | Formal accessibility certification or separate UI-library CI |

The chosen technologies are target decisions, **not evidence that the rebuild is complete**. Performance is measured, not assumed from Go/Solid selection.

## 2. Non-negotiable correctness boundaries

1. One stable seeded owner; no arbitrary client owner IDs or public/LAN exposure in MVP.
2. Preserve exact authored source, all whitespace and original UTF-16 half-open timed offsets. Go byte offsets are not browser offsets.
3. Publication requires matching **READY validated MP3 + actual measured duration + language/voice/rate/segmentation/fingerprint identity**.
4. Drafts may be incomplete; temporary previews are not drafts; failed revisions cannot replace published audio.
5. Metadata changes do not invalidate audio; Favorite is a focused set-value write and does not advance content version/newest timestamp.
6. Vocabulary uniqueness is per owner/language/normalized word key; same-language source links only. Manual collection works regardless of automatic match support.
7. Global Topics and max 10 associations; new cover maximum exactly **1,000,000 bytes**; preserve legacy oversized assets.
8. **UI is app-owned and simple.** CSS variables/modules and native browser elements are the default. Do not build an independently published library or custom interactive widget infrastructure.
9. Azure jobs are explicit, durable/idempotent; never automatically repeat an uncertain paid call.
10. Database and local media are a **single backup/restore unit**.

## 3. Work sequence — vertical slices

### Stage 0 — Inventory and protect source data

- Inspect Mandarin Reader schema/migrations, source text, MP3/cover paths, segmentation versions, vocabulary, Topic identity, old `native` difficulties and oversized cover files.
- Back up nonempty source data before modification. Choose import to fresh TaoTale DB or an in-place upgrade based on a rehearsal; keep historical byte/offset evidence and UUIDs.
- Capture golden Chinese regression vectors and representative performance baseline. No historical test run counts as a new pass.

**Exit:** a written migration inventory and recoverable baseline.

### Stage 1 — Runnable backend/client foundation + Iris UI

- Set up Go/Gin module, local Docker PostgreSQL volume/health, migration runner, pgxpool, sqlc and stable owner/language/Topic seed.
- Create SolidJS/Vite/Router shell and typed `/api/v1` client with `/library` home, route directory and small local preferences.
- Build **`tokens.css`, `global.css`, app-owned Button/Field/Card/Badge and the actual Library page in CSS Modules**. Use the Iris palette immediately so visual design evolves with real screens, not as a late skin.
- Native forms/selects/buttons; simple confirmation (`window.confirm` or native `<dialog>` only as needed). **No npm UI-package work, Kobalte, Tailwind, external theming API, component packaging or custom focus/overlay mechanics.**

**Exit:** local app boots, backend health is visible, empty Library shell looks polished in light/dark at desktop/tablet widths, no external UI library dependency.

### Stage 2 — Highest-risk audio feasibility gate

- Test **real Azure Go SDK** Chinese and Korean voices on target runtime. Capture boundary events, XML/SSML mapping, output bytes, decoded measured duration, cancellation and native library dependencies.
- Implement two independent versioned segmentation strategies and verified UTF-8/UTF-16 conversion with golden fixtures. Map timing without approximation. Maintain legacy Chinese strategy/fingerprint verification.

**Exit:** real test samples in both languages produce verifiable aligned assets or documented blockers; never advance by faking timestamps.

### Stage 3 — Create → Generate → Publish → Reader

- Implement item drafts, cover upload, Topic selection, voices and generation jobs; persist immutable intent/idempotency/MP3 and verified audio metadata. Complete the real backend/client publishing transaction.
- Style the Editor as a two-column writing studio with native inputs and clear Audio status; Reader as an uncluttered reading room with beautiful typography, iris phrase highlighting and native audio controls.
- Add edit metadata/working revision and version conflicts, reliable dirty navigation, safe media ranges and Reader text-only fallback. Keep old active asset on revision failure.
- Finish Library cards, Favorite filter/set behavior, sorting, search, stable paging and clear empty/error states.

**Exit:** both language flows work end to end and the published Reader survives restarts, metadata updates and theme switches without audio remount.

### Stage 4 — Vocabulary, Settings, visual consistency and safety

- Build language-scoped manual vocabulary, references/status/search, global Topics and source selection. Use **regular forms/checkbox lists/inline panels**, not custom combobox/popover framework. Enable auto-matching only for a tested language strategy.
- Implement Settings for theme/language/voice/rate/playback preferences, provider state and truthful **manual** backup/restore instructions.
- Apply consistent Iris cards, type, spacing, empty/pending/error states and responsive desktop/tablet layout to Vocabulary/Settings. Review full light/dark screenshots; keep interactions simple.
- Verify cleanup, local security, job recovery, DB/media paired restore, and Go/SQL/Solid performance. No Azure calls in routine CI.

**Exit:** all five MVP screens are cohesive and usable and important local failure/recovery cases are tested.

### Stage 5 — Contract, migration, final acceptance

- Lock **OpenAPI 3.1**, generate client types and verify Gin requests/errors/media responses. Validate migration on a disposable data copy, comparing IDs, word statuses, source/timing, legacy covers and original audio hashes.
- Run core end-to-end acceptance, real Azure Chinese/Korean smoke, desktop/tablet Iris light/dark visual check, loopback/security and paired restore verification.
- Record actual tests, performance evidence, screenshots, versions, migration revision and known limitations; cut over only after a recoverable rehearsal.

**Exit:** target tests pass, verified backups exist and no separate UI-library tasks block release.

## 4. Remaining decisions

| ID | Remaining choice | Recommended handling |
|---|---|---|
| OPEN-01 | Run Go on host vs within Compose | Pick easiest reliable local setup; preserve loopback and durable asset root |
| OPEN-02 | Go/Gin/pgx/sqlc/migration tool versions, OpenAPI codegen | Pin after minimal working vertical slice |
| OPEN-03 | Actual Azure Go SDK boundary/event reliability for both languages | Experiment before full audio port |
| OPEN-04 | Exact language segmenter versions and maximum text/chunk policy | Golden-test and publish versions only after verified provider mapping |
| OPEN-05 | MP3 decoder/probe on target OS | Choose verified duration/decoding evidence |
| OPEN-06 | Chinese optional automatic match and Korean supported behavior | Start manual; activate only demonstrated strategy |
| OPEN-07 | Legacy `native` values and >1MB covers | Report, preserve, and resolve without silent loss |
| OPEN-08 | `/topics` vs historical `/tags` aliases | Decide when locking TaoTale OpenAPI |
| OPEN-09 | Supported desktop/tablet browser/zoom baseline and screenshot set | Capture exact reference sizes during Stage 1 |
| OPEN-10 | Windows-specific local backup/restore commands | Test on a disposable volume and media directory |

**Resolved by the 25 Nov decision:** External UI-package name/API/licensing/theme delivery are **removed**, not pending decisions. CSS Modules + in-project Iris tokens/components are approved. The palette values and MVP visual rules are in [08](08-taotale-design-system.md).

## 5. Explicitly deferred work

Dashboard; Vocabulary Practice/SRS; AI Text/Audio Generation Workspace; Islands Builder; phone-specific UI; Japanese and further languages; scoring, word-level dictionary/TTS, public sharing, accounts/billing/cloud sync. An independent UI-component library can exist as another business/project, but **TaoTale does not need it and must not integrate it**.

Historical optional ideas (still not MVP): Reader word browser, vocabulary CSV/JSON export, A–B repeat, audio bookmarks, reader text-size preferences, richer diagnostics.

## 6. Governance and task tracking

The executable work breakdown is [15 — Implementation tasks](15-taotale-implementation-tasks.md). A checkbox is unchecked until verified with real implementation evidence; documentation is not a completion claim. Product scope changes update [01](01-taotale-overview-and-features.md) and [02](02-taotale-decision-register.md); CSS/UX changes update [03](03-taotale-client.md), [08](08-taotale-design-system.md) and [09](09-taotale-ui-states.md); acceptance changes update [12](12-taotale-quality-and-acceptance.md); the source index is [14](14-taotale-sources-and-coverage.md).
