# 14 — TaoTale specification sources, precedence and coverage

**Status:** Document index / provenance handoff, revised **25 November 2026**. It is **not** proof of implemented features, tests, or completed migration.

## 1. Source precedence

1. **Latest explicit owner decision:** on **25 Nov 2026**, TaoTale's client UI becomes fully **app-owned**, deliberately simple, using **Iris colors** and no external UI library or specialized component accessibility JavaScript. This supersedes old independent-library/Quiet Study decisions.
2. **01 — Product** determines the MVP vs deferred feature boundary and critical content/audio invariants.
3. **02 — Decisions** records the technical choices and superseded Mandarin Reader history; **03 — Client** and **08 — Iris visual design** now own actual app UI implementation and appearance.
4. **04/05/06** own Go/PostgreSQL/audio semantics. **09/10/11/12** refine UI states, API, operations and test evidence. **13** orders work and **15** lists task-level execution.
5. Prior Mandarin Reader specs/repository are historical reference **only** where not superseded. A screenshot cannot authorize deferred features or a new UI library.

## 2. Current specification index

| # | File | Ownership |
|---|---|---|
| 01 | [Product overview](01-taotale-overview-and-features.md) | Features, scope, languages, publication and MVP acceptance |
| 02 | [Decision register](02-taotale-decision-register.md) | Adopted TaoTale choices + original D-01–D-61 traceability |
| 03 | [Client architecture](03-taotale-client.md) | SolidJS app **with local components/CSS Modules**, native controls, Reader and API |
| 04 | [Backend](04-taotale-backend.md) | Go/Gin, SQL services, TTS adapters/jobs and local HTTP |
| 05 | [Database](05-taotale-database.md) | PostgreSQL logical schema, persistence and migration |
| 06 | [Segmentation and audio](06-taotale-segmentation-and-audio.md) | Bilingual source offsets and validated immutable recording |
| 07 | [AI text generation](07-taotale-ai-text-generation.md) | **Deferred** future AI workspace, not MVP |
| 08 | [Iris visual design](08-taotale-design-system.md) | **App-owned** palette, typography, page composition, CSS/components |
| 09 | [UI states](09-taotale-ui-states.md) | Minimal native interactions, domain/error/Reader states |
| 10 | [API inventory](10-taotale-api.md) | Target REST operations, OpenAPI to be authored |
| 11 | [Operations and security](11-taotale-operations-and-security.md) | Local setup, secrets, media, jobs and paired backup |
| 12 | [Testing / acceptance](12-taotale-quality-and-acceptance.md) | Domain gates, desktop/tablet Iris visual and basic usability checks |
| 13 | [Rebuild roadmap](13-taotale-rebuild-decisions.md) | Stage ordering, risks and remaining decisions |
| 14 | [Sources and coverage](14-taotale-sources-and-coverage.md) | This index and provenance |
| 15 | [Implementation tasks](15-taotale-implementation-tasks.md) | **50 unchecked tasks**, stage exit criteria and explicitly cancelled UI-package work |

Files 04/05 have been standardized here to `04-taotale-backend.md` and `05-taotale-database.md`; links across this revision should point to those exact names.

## 3. Core traceability

| Topic | Requirements | Verification |
|---|---|---|
| Single user, local Docker/Go app | 01–05, 10–11 | 12, T005–T008, T038–T039 |
| No standalone UI library / no custom component behavior engine | 02, 03, 08, 09, 13 | 12/F9, T009–T012, T046–T047 |
| Iris palette, light/dark, five beautiful views | 03, 08, 09 | 12/NFR-11–12, F9, T011, T031, T040–T041, T047 |
| Mandarin + Korean and language ownership | 01, 04–06, 10 | 12/F1/F8, T013–T017, T032–T033 |
| Azure + source alignment + immutable audio | 01, 04–06, 09–11 | 12/F2–F5, T013–T028 |
| Draft/revision integrity + idempotency | 01, 04–06, 09–11 | 12/NFR-05/13, T021–T027 |
| Favorites independent of content/audio | 01, 05, 09, 10 | 12, T029–T030 |
| Difficulty + 1 MB cover | 01, 05, 09–12 | 12/F6, T019/T024 |
| Vocabulary and global Topics | 01, 04, 05, 09, 10 | 12/F8, T032–T036 |
| Local protection and paired backup | 01, 04, 10, 11 | 12/NFR-08/09, T038–T039/T049 |
| MVP vs post-MVP | 01, 02, 07, 13 | 12 and T050 release evidence |

## 4. Explicitly superseded historical rules

| Historical rule | Current ruling |
|---|---|
| Independent SolidJS-only npm UI library consumed by TaoTale | **Removed.** No package integration, separate repo, publishing, slot/theming API or peer-dependency smoke |
| Library-managed Kobalte Dialog/Tooltip/Menu and custom keyboard/ARIA/overlay behavior | **Removed.** Native HTML and small app-local SolidJS wrappers only |
| Quiet Study warm ivory/sage/mint theme | **Superseded** by app-owned **Iris violet/lavender/periwinkle/orchid** palette |
| UI package takes ownership of all generic controls | **Superseded.** TaoTale owns only the few controls its actual screens need |
| Formal UI-package accessibility and release tests | **Removed.** Basic readable, labeled, functional native UI and visual/interaction QA remain |
| React/MUI/Node/NestJS/Prisma | Superseded by **SolidJS + Go/Gin/pgx/sqlc/PostgreSQL** |
| Single Chinese language and Chinese-only `pinyin` identity | `zh-CN` + `ko-KR`; optional generic reading fields; language-scoped words |
| AI text workspace, practice and islands in MVP | Deferred; manual Azure TTS in Editor remains MVP |

## 5. Historical source references

Mandarin Reader pinned baseline: [historical repository tree](https://github.com/ronmilich/mandarin-reader/tree/dc71c1b1820345023585be95e72674bf1b3a4a00) (17 Nov 2026). The earlier original specs covered product requirements, D-01–D-61 decisions, React frontend, Node/Prisma backend, Chinese segmentation, design screenshots, OpenAPI, tests, backup setup and feature proposals. These are migration clues, **not** current requirements. Historical raw data and the actual running SQL/media state must be inspected before import.

**Key warnings:** Target logical schema is not executable migration SQL; target API inventory is not OpenAPI YAML; proposed Azure Go support is not verified until real tests pass; no task or test completion is implied by these documents. Use [15](15-taotale-implementation-tasks.md) for live progress tracking after implementation begins.
