# 14 — TaoTale specification sources, precedence and coverage

**Status:** Documentation inventory and traceability companion, revised 23 November 2026. **This is a specifications handoff, not an implementation audit or test report.** The earlier Mandarin Reader consolidation remains historical reference; **TaoTale-approved owner decisions now take precedence.**

## 1. Source precedence and scope of evidence

When documents conflict, apply these rules **in order**:

1. **Latest explicit TaoTale owner decisions** — current user approvals, reflected in [01](01-taotale-overview-and-features.md) and [02](02-taotale-decision-register.md), override inherited Mandarin Reader plans, even if historical code does something else.
2. **Product scope:** [01 — TaoTale overview](01-taotale-overview-and-features.md) determines MVP vs post-MVP. A detailed older document or UI screenshot **cannot** add a deferred feature to MVP.
3. **Approved architectural boundaries:** [02 — Decision register](02-taotale-decision-register.md) plus [03 — Solid app](03-taotale-client.md), [04 — Go backend](04-backend.md), [05 — SQL schema](05-database.md), [06 — language-specific segmentation](06-taotale-segmentation-and-audio.md), [08 — UI library/design](08-taotale-design-system.md).
4. **Specific operational contracts:** [09 — UI states](09-taotale-ui-states.md), [10 — target API inventory](10-taotale-api.md), [11 — local operations](11-taotale-operations-and-security.md), [12 — acceptance](12-taotale-quality-and-acceptance.md) refine but cannot override higher-priority approved decisions.
5. **Post-MVP design reference:** [07 — AI text generation](07-taotale-ai-text-generation.md) is **not** an authorization to create AI routes/UI/schema in MVP. [13 — Rebuild roadmap](13-taotale-rebuild-decisions.md) sequences work but is not evidence that a stage has shipped.
6. **Historical Mandarin Reader repository/docs and image references** are source material only when not superseded, and must be checked against the actual code/migration SQL for import or parity work.

**Key distinction:** the standalone SolidJS-only **npm UI component library is publicly distributable**, but the **TaoTale application is single-user and local**. Do not infer public networking, auth, billing or hosted infrastructure from the library's distribution model.

## 2. Current TaoTale specification index

| # | Current document | Responsibility / scope | Status |
|---|---|---|---|
| 01 | [`01-taotale-overview-and-features.md`](01-taotale-overview-and-features.md) | Product, languages, Favorites, difficulty, local deployment, MVP/post-MVP boundary | **Primary product authority** |
| 02 | [`02-taotale-decision-register.md`](02-taotale-decision-register.md) | Adopted TT decisions, historical D-01–61 dispositions, open questions | **Decision authority** |
| 03 | [`03-taotale-client.md`](03-taotale-client.md) | SolidJS/TypeScript/Vite/Router architecture; consumes external UI package | **Client architecture** |
| 04 | [`04-backend.md`](04-backend.md) | Go/Gin services, Azure adapter, workers, storage, API and validation | **Backend architecture**; older filename retained |
| 05 | [`05-database.md`](05-database.md) | PostgreSQL Docker/pgx/sqlc logical entities, indexes, transactions, migration | **Target logical model**; not executable DDL |
| 06 | [`06-taotale-segmentation-and-audio.md`](06-taotale-segmentation-and-audio.md) | Mandarin/Korean strategies, UTF-16 offsets, provider timing/MP3 validation | **MVP audio contract** |
| 07 | [`07-taotale-ai-text-generation.md`](07-taotale-ai-text-generation.md) | AI text workspace design for later release | **Post-MVP reference** |
| 08 | [`08-taotale-design-system.md`](08-taotale-design-system.md) | Standalone SolidJS-only npm UI library, accessibility/theming plus Quiet Study app theme | **Library/UI design authority** |
| 09 | [`09-taotale-ui-states.md`](09-taotale-ui-states.md) | MVP loading/mutation/dirty/reader/vocabulary/favorite states and labels | **Target interaction contract** |
| 10 | [`10-taotale-api.md`](10-taotale-api.md) | Target Go REST operation inventory, language-scoped shapes/errors, OpenAPI plan | **Not a machine-readable OpenAPI file** |
| 11 | [`11-taotale-operations-and-security.md`](11-taotale-operations-and-security.md) | Local Compose/Go startup, secrets, media, cost controls, paired backup and restore | **Target operating model** |
| 12 | [`12-taotale-quality-and-acceptance.md`](12-taotale-quality-and-acceptance.md) | Go/SQL/Solid/UI-library tests, bilingual fixtures and MVP release gates | **Proposed tests; not executed** |
| 13 | [`13-taotale-rebuild-decisions.md`](13-taotale-rebuild-decisions.md) | Locked choices, staged vertical-slice implementation and unresolved details | **Roadmap / governance** |
| 14 | [`14-taotale-sources-and-coverage.md`](14-taotale-sources-and-coverage.md) | This authority/provenance/traceability inventory | **Companion index** |

**Naming policy:** Newly created Markdown specifications use **`number-taotale-filename.md`**. Existing 04/05 files in this working set predate the naming rule; references above use their *actual current filenames* rather than pretending they have already been renamed. Renaming them should be accompanied by link updates; it is not part of revising 09–14.

## 3. Historical baseline and provenance

The original Mandarin Reader consolidation cited a private repository at commit `dc71c1b1820345023585be95e72674bf1b3a4a00` (17 November 2026). [Pinned repository tree](https://github.com/ronmilich/mandarin-reader/tree/dc71c1b1820345023585be95e72674bf1b3a4a00). The links below identify **historical** sources; they are not claims that the new Go/Solid TaoTale repository or its OpenAPI/migrations already exist.

| Historical source (private repository) | Reused concept or legacy evidence | Current TaoTale owner document |
|---|---|---|
| [Product requirements](https://github.com/ronmilich/mandarin-reader/blob/dc71c1b1820345023585be95e72674bf1b3a4a00/docs/product/requirements.md) | Drafts, publication, Reader, vocabulary, constraints, original NFRs | 01, 09, 12 |
| [Decision register](https://github.com/ronmilich/mandarin-reader/blob/dc71c1b1820345023585be95e72674bf1b3a4a00/docs/development/decisions.md) | D-01–61 and D-25(a–u), lifecycle and historical stack choices | 02, 13 |
| [Original frontend architecture](https://github.com/ronmilich/mandarin-reader/blob/dc71c1b1820345023585be95e72674bf1b3a4a00/docs/architecture/frontend.md) | Routes, Reader event semantics, old React client and state behavior | 03, 09 |
| [Original backend architecture](https://github.com/ronmilich/mandarin-reader/blob/dc71c1b1820345023585be95e72674bf1b3a4a00/docs/architecture/backend.md) | Generation recovery, immutable media, idempotency, local service boundary | 04, 10, 11 |
| [Original DB architecture](https://github.com/ronmilich/mandarin-reader/blob/dc71c1b1820345023585be95e72674bf1b3a4a00/docs/architecture/database.md) and [Prisma schema](https://github.com/ronmilich/mandarin-reader/blob/dc71c1b1820345023585be95e72674bf1b3a4a00/backend/prisma/schema.prisma) | Legacy entity identity, indexes, reference provenance and migration input | 05 |
| [Original segmentation/synchronization](https://github.com/ronmilich/mandarin-reader/blob/dc71c1b1820345023585be95e72674bf1b3a4a00/docs/architecture/segmentation-and-sync.md) | Exact UTF-16/timing, Chinese v1 golden alignment | 06, 12 |
| [Original AI feature](https://github.com/ronmilich/mandarin-reader/blob/dc71c1b1820345023585be95e72674bf1b3a4a00/docs/features/ai-text-generation.md) | Deferred AI provider and future workspace behavior | 07 |
| [Original UI design](https://github.com/ronmilich/mandarin-reader/blob/dc71c1b1820345023585be95e72674bf1b3a4a00/docs/ui/design-system.md) / [Topic palettes](https://github.com/ronmilich/mandarin-reader/blob/dc71c1b1820345023585be95e72674bf1b3a4a00/frontend/src/shared/theme/tagColors.ts) | Quiet Study colors, tokens and Topic presets | 08 |
| [Original UI states](https://github.com/ronmilich/mandarin-reader/blob/dc71c1b1820345023585be95e72674bf1b3a4a00/docs/ui/states.md) | Dirty guard, error states and labels | 09 |
| [Original OpenAPI 3.1 YAML](https://github.com/ronmilich/mandarin-reader/blob/dc71c1b1820345023585be95e72674bf1b3a4a00/docs/api/openapi.yaml) | Historic **52** method/path pairs, old DTOs, routes and errors | 10; **new TaoTale YAML still to be authored** |
| [Original setup](https://github.com/ronmilich/mandarin-reader/blob/dc71c1b1820345023585be95e72674bf1b3a4a00/docs/operations/setup.md) | Paired backup, recovery/security and health intent | 11 |
| [Original testing spec](https://github.com/ronmilich/mandarin-reader/blob/dc71c1b1820345023585be95e72674bf1b3a4a00/docs/testing/specification.md) / [Acceptance ledger](https://github.com/ronmilich/mandarin-reader/blob/dc71c1b1820345023585be95e72674bf1b3a4a00/docs/development/acceptance-status.md) | Old F1–F7 fixtures and historical scoped results | 12; **no TaoTale pass credit** |
| [Original feature suggestions](https://github.com/ronmilich/mandarin-reader/blob/dc71c1b1820345023585be95e72674bf1b3a4a00/docs/issues/feature-suggestions.md) | FEAT-01–07 backlog, not current MVP requirements | 13 |

Other references in the historical repository include M12 Library, M13 Create, M14 Reader, M15 Vocabulary and M16 Topic screenshots and milestone reports. They may help replicate desirable UX, **but visual examples containing `/generate`/mobile layouts do not override current MVP scope**. The first text-to-speech application PRD and older archived materials are further historical context, not current architecture authority.

## 4. Critical cross-document traceability

| Requirement or decision | Primary specs | Verification / future evidence |
|---|---|---|
| Local one-user deployment and internal owner | 01, 02, 04, 05, 11 | Docker startup, loopback checks, migrations/seed, absence of account routes; 12 |
| Go + Gin, PostgreSQL/pgx/v5/sqlc, SQL migrations | 02, 04, 05, 10, 11 | Real Gin/sqlc integration, DDL/lock/rollback, benchmark; 12 |
| SolidJS-only app with external npm UI library | 02, 03, 08, 09 | Installable package exports/ESM/CSS/types, single Solid runtime, theme/a11y tests; 12 |
| No direct Kobalte or Tailwind in TaoTale | 02, 03, 08 | Dependency/import/bundle audit; 12 |
| Mandarin and Korean language selection | 01, 04, 05, 06, 10 | Valid language/voice lookup and isolated Library/Vocabulary, bilingual fixtures; 12 |
| Per-language segmentation + immutable validated audio | 01, 04, 05, 06, 10 | Real Azure Chinese/Korean offsets, Go UTF-16 vectors, MP3 measurement, Reader sync; 12 |
| Draft → preview/generate → published/revision integrity | 01, 04–06, 09–11 | Race/restart/idempotency and old-active-asset preservation; 12 |
| Favorites independent of audio/version | 01, 05, 09, 10 | Toggle both draft/published, restart/filter sort/version tests; 12 |
| Difficulty and 1 MB covers | 01, 05, 09–12 | Three levels only; cover 1,000,000/1,000,001 boundary; legacy import; 12 |
| Vocabulary language identity and Topic sharing | 01, 04, 05, 09, 10 | Cross-language duplicate/link rejection; manual collection/status; 12 |
| Local private media, no false online security claim | 01, 04, 10, 11 | Host/origin/path/bundle tests, network exposure review; 12 |
| Paired database/media backup and restore | 01, 05, 11 | Fresh Docker volume restore + file hash/record check; 12 |
| MVP/post-MVP boundary | 01, 02, 07, 08, 09–13 | No AI workspace/Practice/Dashboard/Islands/phone/login/billing in MVP routes or acceptance; 12 |

## 5. Resolved conflicts and implementation warnings

| Inherited old rule | Current approved TaoTale rule |
|---|---|
| Product named Mandarin Reader or Talezi, single `zh-CN` | **TaoTale** with `zh-CN` and `ko-KR`; item/word language explicit. |
| Node/NestJS/Prisma | **Go/Gin + PostgreSQL/pgx/v5/sqlc**, reviewed SQL migrations. |
| React/MUI/TanStack Query/React Hook Form | **SolidJS** and native Solid/browser primitives; external Solid-only npm UI package. |
| Kobalte directly in app or Tailwind CSS | **Neither in TaoTale**. UI package may use Kobalte internally; theming is via package public API. |
| UI system part of TaoTale application itself | Independent npm library owns reusable components, accessibility and transition plumbing; Quiet Study remains TaoTale-specific. |
| HSK or `native` as general difficulty | **Beginner / Intermediate / Advanced**, nullable Unspecified; legacy unsupported values reviewed explicitly. |
| 5 MiB cover uploads | **1,000,000 encoded bytes** for new/replacements; legacy images remain readable. |
| No Favorites | Persistent draft/published `isFavorite` without content-version/updated-date changes. |
| AI generated text and `/generate` considered implemented MVP | **Post-MVP** only; manual editor **Azure Generate Audio** remains MVP. |
| Common Mandarin segmentation for all speech | Separate ID/version-pinned Mandarin and Korean strategy; Japanese only when added. |
| Mandarin `pinyin` as all-language field | Optional generic reading + readingSystem; language-scoped vocabulary; Topics shared. |
| Mobile/375px or phone acceptance | **Desktop and tablet-sized** support; phone layouts post-MVP. |
| Live ChatGPT Windows token probe required to ship local app | Historical deferred AI experiment; no AI connection, credential provider or token store in MVP. |
| Completed old tests certify new release | No: **new Go/Solid/Korean and npm-package evidence** is required. |

## 6. Remaining source-of-truth gaps and maintenance procedure

- **Machine-readable API** is not yet authored: [`10`](10-taotale-api.md) is an operation inventory. Lock a new OpenAPI 3.1 YAML and regenerate client contract types; decide `/topics` rename/compatibility and route responses.
- **Physical SQL migrations and sqlc queries** are not created by [`05`](05-database.md). Write and test them against an actual copy of old PostgreSQL records, not merely the legacy Prisma schema summary.
- **Mandarin/Korean Go Azure word-boundary evidence**, MP3 decoding and exact segmentation golden versions must be collected. [`06`](06-taotale-segmentation-and-audio.md) is the contract, not provider certification.
- **Independent npm UI package** name/peer versions, public theme hooks and package release remain implementation decisions. [`08`](08-taotale-design-system.md) specifies requirements, **not a published package that has been verified**.
- **Performance, accessibility, restore and migration evidence** remain to be run and recorded. [`12`](12-taotale-quality-and-acceptance.md) is an acceptance plan, not an executed acceptance ledger.
- In legacy migration, preserve original UUIDs, published MP3/offset/strategy data, word statuses, Topics and large covers; review `native` difficulty rather than silently mapping it.
- When a new owner decision changes scope, update **01 and 02 first**, then impacted topical specs and their API/SQL/tests, and finally this traceability document. Maintain links to *actual existing filenames*; never silently claim the app, package, migrations or tests exist just because the specification describes them.

**Documentation revision note:** The files `09-ui-states(1).md` through `14-sources-and-coverage(1).md` were the historical input for this rewrite. They are not live TaoTale implementation evidence; the new files are named according to the `number-taotale-filename.md` rule.
