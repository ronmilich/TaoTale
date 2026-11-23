# TaoTale — implementation roadmap

**Status:** Planned — no milestone started (created 23 November 2026). This describes target work. It is not evidence that anything is implemented (12 §1).

**Scope authority:** [01 — Overview](../01-taotale-overview-and-features.md) decides MVP vs post-MVP; [02 — Decision register](../02-taotale-decision-register.md) records decisions. Technical sources are [03](../03-taotale-client.md)–[06](../06-taotale-segmentation-and-audio.md) and [08](../08-taotale-design-system.md)–[12](../12-taotale-quality-and-acceptance.md). This roadmap breaks the stage plans in 01 §9 and 13 §3 into small, gated milestones.

**Depends on:** the [UI Library roadmap](ui-library-roadmap.md). **Every TaoTale milestone has a UI Library gate.** It cannot start until that library milestone is Done.

---

## 1. Scope summary

- **MVP:**
  - a local, single-user app: Go/Gin + PostgreSQL in Docker + local media files, and a SolidJS client built on the UI library;
  - Library: drafts and published items, Favorites, difficulty, topics, filters;
  - Create/Edit with explicit Azure TTS generation and verified publication; working revisions;
  - Reader with synchronized highlighting;
  - language-scoped Vocabulary and shared Topics;
  - Settings / local data and a tested paired backup and restore;
  - import of the legacy Mandarin Reader data;
  - learning languages `zh-CN` and `ko-KR`; desktop and tablet layouts.
- **Post-MVP (Phase 11):** Dashboard, Vocabulary Practice, AI Text/Audio workspace, Language Islands, phone support, Japanese, optional backlog.
- **Never in the MVP:** login or accounts, billing, cloud hosting, LAN exposure, or routes for deferred features (01 §8).

## 2. UI Library gating rules

1. **One gate per milestone.** Each TaoTale milestone names one UI Library milestone (UL-xx) as its gate. It cannot start until that UL milestone is **Done**.
2. **Done means released.** For a UL code milestone, Done means its exit criteria are met **and** the version is published and verified by the library's consumer smoke test. TaoTale installs that exact version from the registry, or the packed prerelease from the library's release pipeline. Local linking (`npm link`, workspace paths) never satisfies a gate (08 §8.6). For UL decision milestones, Done means the decision record is approved.
3. **Two gate types, both binding:**
   - **Hard:** the milestone needs what the gate delivers — components, APIs, decisions or audit fixes.
   - **Sequencing:** library-first ordering, so the library always stays ahead. This also applies to backend-only work.

   Changing a sequencing gate requires a dated decision in 02. Hard gates follow from TT-028–031 and cannot be bypassed.
4. **No bypass.** If TaoTale needs a generic capability the library lacks:
   1. add it to a UL milestone that hasn't started, or create a new UL milestone;
   2. release it;
   3. then continue.

   Never write generic controls in TaoTale, never import `@kobalte/core` or any other headless UI library, never add Tailwind, and never use private library selectors (TT-028–030, TT-032, 03 §6).
5. **Domain components stay in TaoTale** — LibraryItemCard, Transcript, AudioPlayer, TopicManager, vocabulary dialogs. They are composed from public library exports (08 §4.1).
6. **Library defects found in TaoTale** are fixed in the library and released as a patch. TaoTale upgrades its pinned version only after the upgrade checks in 03 §12: install, theme CSS, overlay focus and Reader stability.
7. **The library never depends on TaoTale.** TaoTale feedback goes into the library backlog; it is never a library gate.
8. **Gate log.** When a milestone starts, record the gating UL milestone and the exact library version in the milestone's PR or release notes.

## 3. Integrated execution sequence (library-first)

```mermaid
flowchart TB
  subgraph LIB["UI Library — leads"]
    U1["UL-01–03<br/>charter · name · licence"]
    U2["UL-04–11<br/>spike · ADRs · repo · build · CI · release · docs"]
    U3["UL-12–17<br/>tokens · in-house behavior layer"]
    U4["UL-18–23<br/>theming · customization · foundation components 0.1–0.4"]
    U5["UL-24–31<br/>overlay engine · overlays · integration release 0.12"]
    U6["UL-32–35<br/>select · feedback · pagination · slider"]
    U7["UL-36–40<br/>combobox · dropzone · table · toast · tabs"]
    U8["UL-41–44<br/>docs · a11y audit · performance · 1.0 RC"]
    U9["UL-45 →<br/>1.0 GA · launch · post-1.0"]
    U1 --> U2 --> U3 --> U4 --> U5 --> U6 --> U7 --> U8 --> U9
  end
  subgraph APP["TaoTale — gated"]
    T1["TAO-01–03<br/>legacy · specs · Azure spike"]
    T2["TAO-04–08, 11<br/>repo · Docker · Go · SQL · OpenAPI · Unicode"]
    T3["TAO-12–15<br/>segmentation · TTS adapter · alignment"]
    T4["TAO-09, 16–20<br/>client scaffold · items · generation"]
    T5["TAO-10<br/>app shell · Quiet Study"]
    T6["TAO-21–27<br/>editor · list · publish · Reader"]
    T7["TAO-28–41<br/>library · revisions · vocabulary"]
    T8["TAO-42–51<br/>settings · backup · legacy · hardening · MVP"]
    T9["TAO-52–61<br/>post-MVP"]
    T1 --> T2 --> T3 --> T4 --> T5 --> T6 --> T7 --> T8 --> T9
  end
  U1 -. gate .-> T1
  U2 -. gate .-> T2
  U3 -. gate .-> T3
  U4 -. gate .-> T4
  U5 -. gate .-> T5
  U6 -. gate .-> T6
  U7 -. gate .-> T7
  U8 -. gate .-> T8
  U9 -. gate .-> T9
```

The diagram groups milestones into blocks. The table below shows the exact order and the gate each TaoTale block waits for:

| Step | Project | Milestones | Gate / notes |
|---|---|---|---|
| 1 | UI Library | UL-01 → UL-03 | Charter, name, licence. |
| 2 | TaoTale | TAO-01 → TAO-03 | Gate UL-03. Legacy inventory, spec reconciliation, and the Azure spike — the earliest de-risking. |
| 3 | UI Library | UL-04 → UL-07 | Spike, ADRs, conventions, repository. |
| 4 | TaoTale | TAO-04 → TAO-06 | Gate UL-07. Repository, Docker PostgreSQL, Go skeleton. |
| 5 | UI Library | UL-08 → UL-11 | Packaging, CI, release pipeline, docs site. |
| 6 | TaoTale | TAO-07, TAO-08, TAO-11 | Gate UL-11. Migrations/sqlc, OpenAPI, Unicode offsets and fingerprints. |
| 7 | UI Library | UL-12 → UL-15 | Tokens, default themes, core and accessibility utilities. |
| 8 | TaoTale | TAO-12 → TAO-15 | Gate UL-15. Segmentation (zh, ko), TTS adapter, alignment validation. |
| 9 | UI Library | UL-16 → UL-20 | Interaction and collection primitives, theming API, customization contract, Button → **`0.1.0`**. |
| 10 | TaoTale | TAO-09, TAO-16, TAO-17 | Gate UL-20. Client scaffold on the published package; items and topics API. |
| 11 | UI Library | UL-21 → UL-23 | Fields, choice controls, display primitives. |
| 12 | TaoTale | TAO-18 → TAO-20 | Gate UL-23. Generation admission, worker, streaming and previews. |
| 13 | UI Library | UL-24 → UL-31 | In-house overlay engine (layering, focus, positioning), overlays and the integration-readiness release **`0.12.0`**. |
| 14 | TaoTale | TAO-10 | Gate UL-31. App shell and Quiet Study theme. |
| 15 | UI Library | UL-32 → UL-33 | Select; Alert, Progress, EmptyState. |
| 16 | TaoTale | TAO-21 | Gate UL-33. Editor drafts. |
| 17 | UI Library | UL-34 | Pagination. |
| 18 | TaoTale | TAO-22 | Gate UL-34. Library list. |
| 19 | UI Library | UL-35 | Slider. |
| 20 | TaoTale | TAO-23 → TAO-27 | Gate UL-35. Audio generation UI, publication, Reader. |
| 21 | UI Library | UL-36 | Combobox. |
| 22 | TaoTale | TAO-28, TAO-29 | Gate UL-36. Topics UI; Library filters and Favorites. |
| 23 | UI Library | UL-37 | FileDropzone. |
| 24 | TaoTale | TAO-30 → TAO-35 | Gate UL-37. Covers, deletion, metadata edit, working revisions. |
| 25 | UI Library | UL-38 → UL-39 | Table; Toast. |
| 26 | TaoTale | TAO-36 → TAO-41 | Gate UL-39. Vocabulary. |
| 27 | UI Library | UL-40 → UL-41 | Tabs; documentation completion. |
| 28 | TaoTale | TAO-42 → TAO-46 | Gate UL-41. Settings, resilience, backup, legacy import. |
| 29 | UI Library | UL-42 | Accessibility audit and conformance report. |
| 30 | TaoTale | TAO-47 | Gate UL-42. Accessibility and device QA. |
| 31 | UI Library | UL-43 | Performance budgets and compatibility. |
| 32 | TaoTale | TAO-48 → TAO-50 | Gate UL-43. Performance, faults/security, real Azure smoke. |
| 33 | UI Library | UL-44 | API freeze, **`1.0.0-rc`**. |
| 34 | TaoTale | TAO-51 | Gate UL-44. **MVP release**, built on the RC. |
| 35 | UI Library | UL-45 | **`1.0.0` GA** and commercial launch. |
| 36+ | Both | UL-46 → UL-53, TAO-52 → TAO-61 | Post-1.0 and post-MVP; each TaoTale item starts after its gate. |

Within a block, milestones also follow their own *Depends on* lists. With more than one developer, the two projects can overlap only as far as the gates allow: a TaoTale milestone never starts before its gate is Done.

## 4. Milestone summary

| ID | Milestone | Phase | UL gate | Gate type | Depends on | Status |
|---|---|---|---|---|---|---|
| TAO-01 | Legacy inventory and migration strategy | 0 | UL-03 | Sequencing | — | Planned |
| TAO-02 | Specification reconciliation | 0 | UL-03 | Hard | — | Planned |
| TAO-03 | Azure Go SDK feasibility spike | 0 | UL-03 | Sequencing | — | Planned |
| TAO-04 | Repository and tooling foundation | 1 | UL-07 | Sequencing (hard input: UL-05) | TAO-02 | Planned |
| TAO-05 | Local Docker PostgreSQL and media root | 1 | UL-07 | Sequencing | TAO-04 | Planned |
| TAO-06 | Go/Gin backend skeleton | 1 | UL-07 | Sequencing | TAO-04, TAO-05 | Planned |
| TAO-07 | SQL migrations, sqlc and seeds | 1 | UL-11 | Sequencing | TAO-06 | Planned |
| TAO-08 | OpenAPI contract pipeline | 1 | UL-11 | Sequencing | TAO-07 | Planned |
| TAO-09 | SolidJS client scaffold on the published library | 1 | UL-20 | Hard | TAO-04, TAO-08 | Planned |
| TAO-10 | App shell, routing and Quiet Study theme | 1 | UL-31 | Hard | TAO-09 | Planned |
| TAO-11 | Unicode offsets and fingerprints | 2 | UL-11 | Sequencing | TAO-01, TAO-06 | Planned |
| TAO-12 | Segmentation registry and Mandarin strategies | 2 | UL-15 | Sequencing | TAO-11 | Planned |
| TAO-13 | Korean segmentation strategy | 2 | UL-15 | Sequencing | TAO-12 | Planned |
| TAO-14 | TTS provider adapter, voice catalog and mock provider | 2 | UL-15 | Sequencing | TAO-03, TAO-08 | Planned |
| TAO-15 | MP3 measurement and alignment validation | 2 | UL-15 | Sequencing | TAO-12, TAO-13, TAO-14 | Planned |
| TAO-16 | Library item and draft API | 3 | UL-20 | Sequencing | TAO-07, TAO-08 | Planned |
| TAO-17 | Topics API | 3 | UL-20 | Sequencing | TAO-07, TAO-08 | Planned |
| TAO-18 | Generation admission and idempotency | 3 | UL-23 | Sequencing | TAO-15, TAO-16 | Planned |
| TAO-19 | Generation worker, READY pipeline and job recovery | 3 | UL-23 | Sequencing | TAO-18 | Planned |
| TAO-20 | Audio streaming, downloads and temporary previews | 3 | UL-23 | Sequencing | TAO-19 | Planned |
| TAO-21 | Editor: new item and drafts | 4 | UL-33 | Hard | TAO-10, TAO-16 | Planned |
| TAO-22 | Library list (basic) | 4 | UL-34 | Hard | TAO-10, TAO-16 | Planned |
| TAO-23 | Editor: audio generation and preview | 4 | UL-35 | Hard | TAO-14, TAO-20, TAO-21 | Planned |
| TAO-24 | Publication | 4 | UL-35 | Sequencing | TAO-23 | Planned |
| TAO-25 | Reader page and transcript rendering | 4 | UL-35 | Sequencing | TAO-24 | Planned |
| TAO-26 | Audio player and synchronized highlighting | 4 | UL-35 | Hard | TAO-25 | Planned |
| TAO-27 | Reader actions: Favorite, Notes, Download | 4 | UL-35 | Sequencing | TAO-26 | Planned |
| TAO-28 | Topics UI | 5 | UL-36 | Hard | TAO-17, TAO-21 | Planned |
| TAO-29 | Library search, filters, sort and Favorites | 5 | UL-36 | Hard | TAO-22, TAO-27, TAO-28 | Planned |
| TAO-30 | Cover upload pipeline | 5 | UL-37 | Sequencing | TAO-16 | Planned |
| TAO-31 | Cover UI | 5 | UL-37 | Hard | TAO-21, TAO-22, TAO-30 | Planned |
| TAO-32 | Deletion and downloads in the Library | 5 | UL-37 | Sequencing | TAO-20, TAO-22 | Planned |
| TAO-33 | Published metadata edit | 6 | UL-37 | Sequencing | TAO-24, TAO-28, TAO-31 | Planned |
| TAO-34 | Working revisions: backend | 6 | UL-37 | Sequencing | TAO-19, TAO-24 | Planned |
| TAO-35 | Working revisions: client | 6 | UL-37 | Sequencing | TAO-23, TAO-33, TAO-34 | Planned |
| TAO-36 | Vocabulary data model and core API | 7 | UL-39 | Sequencing | TAO-16, TAO-17 | Planned |
| TAO-37 | Vocabulary list, statistics and bulk API | 7 | UL-39 | Sequencing | TAO-36 | Planned |
| TAO-38 | Reader vocabulary collection | 7 | UL-39 | Sequencing | TAO-26, TAO-36 | Planned |
| TAO-39 | Vocabulary view | 7 | UL-39 | Hard (with UL-38) | TAO-28, TAO-37 | Planned |
| TAO-40 | Reader vocabulary panel | 7 | UL-39 | Sequencing | TAO-38 | Planned |
| TAO-41 | Verified Chinese automatic matching *(conditional)* | 7 | UL-39 | Sequencing | TAO-12, TAO-40 | Planned |
| TAO-42 | Settings and local data | 8 | UL-41 | Sequencing | TAO-10, TAO-14, TAO-26, TAO-40 | Planned |
| TAO-43 | Cleanup, recovery and integrity checks | 8 | UL-41 | Sequencing | TAO-19, TAO-20, TAO-30, TAO-32 | Planned |
| TAO-44 | Paired backup and restore | 8 | UL-41 | Sequencing | TAO-43 | Planned |
| TAO-45 | Legacy import tooling | 9 | UL-41 | Sequencing | TAO-01, TAO-11, TAO-12, TAO-34, TAO-36 | Planned |
| TAO-46 | Legacy verification and rehearsal | 9 | UL-41 | Sequencing | TAO-26, TAO-45 | Planned |
| TAO-47 | Accessibility and supported-device QA | 10 | UL-42 | Hard | TAO-21 – TAO-42 | Planned |
| TAO-48 | Performance baselines and budgets | 10 | UL-43 | Hard | TAO-46, TAO-47 | Planned |
| TAO-49 | Fault-injection, security and privacy verification | 10 | UL-43 | Sequencing | TAO-44, TAO-46 | Planned |
| TAO-50 | Real Azure bilingual release smoke | 10 | UL-43 | Sequencing | TAO-49 | Planned |
| TAO-51 | MVP acceptance, release report and cutover | 10 | UL-44 | Hard | TAO-47 – TAO-50 | Planned |
| TAO-52 | Post-MVP scope approval and ordering | 11 | UL-45 | Sequencing | TAO-51 | Planned |
| TAO-53 | Dashboard | 11 | UL-45 | Sequencing | TAO-52 | Planned |
| TAO-54 | Vocabulary Practice | 11 | UL-48 | Hard | TAO-52 | Planned |
| TAO-55 | AI provider re-validation spike | 11 | UL-45 | Sequencing | TAO-52 | Planned |
| TAO-56 | AI text-generation backend | 11 | UL-45 | Sequencing | TAO-55 | Planned |
| TAO-57 | AI workspace UI (`/generate`) | 11 | UL-45 | Hard | TAO-56 | Planned |
| TAO-58 | Language Islands Builder | 11 | UL-48 | Hard | TAO-52 | Planned |
| TAO-59 | Phone and small-screen support | 11 | UL-47 (includes UL-46) | Hard | TAO-52 | Planned |
| TAO-60 | Japanese (`ja-JP`) support | 11 | UL-45 | Sequencing | TAO-52 | Planned |
| TAO-61 | Optional enhancement backlog | 11 | per item (set when scheduled) | — | TAO-52 | Planned |

## 5. UI component needs by screen

This shows which UI Library components each part of TaoTale uses, and therefore where the hard gates come from.

| Screen / capability | UI Library components used | Hard gate |
|---|---|---|
| Client scaffold | Button, theming API, CSS delivery | UL-20 |
| App shell | Button, IconButton, Tooltip, DropdownMenu (language switcher), theming + first-paint helper, VisuallyHidden (skip link) | UL-31 |
| Editor: drafts | Field, TextField, Textarea with counter, Select (language, difficulty), AlertDialog (dirty guard), Alert | UL-33 |
| Library list | Card, Badge, Skeleton, Pagination, EmptyState, DropdownMenu (card actions), Alert | UL-34 |
| Editor audio and preview player | Select (voice, rate), Badge (status), Progress, Slider (timeline), IconButton, Tooltip | UL-35 |
| Reader | Slider, IconButton, Tooltip, DropdownMenu (speed), Dialog (Notes), Alert (fallback) | UL-35 |
| Topics | Combobox (multi-select + create), Chip with consumer colors, Dialog, RadioGroup (swatches), AlertDialog | UL-36 |
| Library filters and Favorites | TextField (search), Select (sort, difficulty), ToggleGroup (status), Switch (Favorites only), Combobox (topics), IconButton + Tooltip (star) | UL-36 |
| Covers | FileDropzone, Alert | UL-37 |
| Reader vocabulary collection and panel | Popover with a virtual anchor (selection), Dialog (Add), ToggleGroup (display filter), Badge | UL-28 (in place by UL-39) |
| Vocabulary view | Table with row selection, Checkbox, Pagination, TextField, Select, Combobox, Dialog, AlertDialog, Toast | UL-39 |
| Settings | ToggleGroup (theme), Select (language, voice, rate, speed), Switch, Alert, Toast (copy confirmation) | UL-39 |

## 6. Definition of Done (every milestone)

1. **Gate respected:** the UL gate was Done before work started; the exact library version is pinned and logged (rule 8).
2. **Library boundary:** only public exports of the pinned UI library. No `@kobalte/core`, no Tailwind, no private selectors, no local generic controls — the dependency guard from TAO-04 is green.
3. **Contract first:** OpenAPI updated before handlers; generated TypeScript types and sqlc code fresh; SQL migrations forward-only and reviewed.
4. **Tests at the right layer:**
   - Go unit tests;
   - integration tests on a disposable real PostgreSQL;
   - Solid component tests;
   - Playwright end-to-end tests for user-visible flows;
   - mock TTS only. The only paid Azure calls are in TAO-03 and TAO-50, plus any owner-approved check in TAO-14.
5. **Accessibility for new UI:** keyboard path, visible focus, labels and status semantics, cues beyond color, and an automated axe check in light and dark themes.
6. **Privacy and safety:** no private study text, secrets or absolute paths in logs, fixtures or commits; loopback-only defaults intact.
7. **Scope guard:** nothing from Dashboard, Practice, AI workspace, Islands, phone layouts, auth or billing appears in MVP routes, bundles or schema.
8. **Docs updated in the same change** (13 §6): affected specs, 02 entries for any open items closed, and this roadmap's status column.
9. **Evidence, not claims:** record actual results. Skipped or unexecuted tests are never reported as passing (12 §6).

---

## 7. Phases and milestones

### Phase 0 — Preparation and de-risking

*Maps to 01 §9 Stage 0 and 13 §3 Stage 0, with the 13 Stage 2 Azure spike pulled forward.*

> **Operational note:** if the legacy Mandarin Reader data is not already backed up, back it up now. That is a safety action, not roadmap work, and it waits for no gate.

#### TAO-01 — Legacy inventory and migration strategy

**Gate:** UL-03 (sequencing)

**Goal:** Know exactly what legacy data exists and how it will move into TaoTale without loss.

**Scope**
- A verified, restorable copy of the legacy PostgreSQL database and media, stored outside the project and outside Docker volumes.
- Inventory, using counts and IDs only (no private text in the repository):
  - schema and Prisma migration history, and UUIDs;
  - `native` and other unexpected difficulty values;
  - covers larger than 1,000,000 bytes;
  - pinyin fields;
  - segmentation and fingerprint versions;
  - replay/idempotency rows;
  - manual and automatic vocabulary references;
  - orphan files.
- Freeze legacy golden vectors (v1 Chinese segmentation, legacy fingerprint format) using synthetic or owner-approved texts. Capture a legacy performance baseline for later comparison.
- Choose between importing into a fresh TaoTale schema and upgrading in place. 13 Stage 0 prefers rehearsing on a disposable copy.

**Exit criteria**
- [ ] Backup restore verified on a disposable instance.
- [ ] Inventory report and migration strategy recorded as a new TT- decision in 02, with open items for 02 OPEN-07 listed.
- [ ] Legacy golden vectors committed as fixtures.

**Depends on:** —

#### TAO-02 — Specification reconciliation

**Gate:** UL-03 (hard — records the library's name and licence)

**Goal:** Bring the specification set in line with current decisions before any code starts.

*Already done on 23 November 2026, ahead of this milestone: the 33 broken cross-links were fixed, the no-Kobalte decision was recorded as TT-032 across 02, 03, 07, 08, 13 and 14, and both roadmaps were registered in 14 §2.*

**Scope**
- Record the UI library's name, npm package, licence and repository decision in 02. This closes the naming/licensing part of 02 OPEN-10 and 13 OPEN-09.
- Unify the two colliding `OPEN-xx` numbering schemes (02 §6 and 13 §4 use the same IDs for different items): keep a single register in 02 and make 13 reference it.
- Remove the stale "Documentation follow-up" note at the end of 08 — 02 and 03 already describe the external package.

**Exit criteria**
- [ ] Decisions recorded in 02 with TT- IDs.
- [ ] One `OPEN-xx` numbering scheme across 02 and 13.
- [ ] No stale notes or broken links remain.

**Depends on:** —

#### TAO-03 — Azure Go SDK feasibility spike

**Gate:** UL-03 (sequencing)

**Goal:** Before building the backend, prove that Go can produce MP3 audio **and** reliable word-boundary timing for both `zh-CN` and `ko-KR` (13 Stage 2, 13 OPEN-03, BE-OPEN-03).

**Scope**
- A throwaway Go program in `spikes/` on the target OS. Record the SDK's native/CGO dependencies and installation steps.
- Synthesize synthetic Mandarin and Korean passages that cover punctuation, quotes, numbers, line breaks, `&`, `<` and emoji, using candidate voices at rates 0.75, 1 and 1.25.
- Capture the MP3 bytes and every boundary event (offsets, offset units, ticks). Check:
  - SSML escaping and offset mapping;
  - event ordering;
  - cancellation and timeout;
  - two syntheses running concurrently.

  Compare with the legacy Node output where available.
- **Billable:** run only with explicit owner approval, with credentials from the local environment only. Record the number of requests and their cost.

**Exit criteria**
- [ ] Evidence report (event dumps, observations per language) kept outside source control, except for small approved fixtures.
- [ ] Go/no-go decision per language recorded in 02. If Korean fails, record it as a blocker with a chosen alternative — never fabricate timing and never silently drop the language (06 §9).

**Depends on:** —

### Phase 1 — Foundations

*Maps to 01 §9 Stage 1 and 13 §3 Stage 1.*

#### TAO-04 — Repository and tooling foundation

**Gate:** UL-07 (sequencing). Hard input: UL-05's repository topology and pre-release consumption model.

**Goal:** TaoTale's repository structure and guardrails.

**Scope**
- Layout per 11 §2, adjusted to UL-05: `frontend/`, `backend/`, `infra/`, `docs/`, `spikes/`, and a gitignored `local-data/`.
- Pin the Go, Node and package-manager versions (02 OPEN-01).
- `.gitignore` for env files, media and dumps; secret scanning; formatting and lint hooks.
- CI skeleton with the stages from 12 §6 as placeholders.
- **Dependency guard:** CI fails if `@kobalte/core` (or any other headless UI library) or `tailwindcss` appears in TaoTale's manifests, imports or resolved dependency tree. Checking the tree also confirms the UI library ships without Kobalte (TT-032). Add a rule that every new runtime dependency needs a written justification (03 §1.2).
- How TaoTale gets library builds: registry versions pinned exactly, or prerelease tarballs from the library's release pipeline.

**Exit criteria**
- [ ] CI green on the skeleton.
- [ ] The dependency guard is proven by a deliberately failing test.

**Depends on:** TAO-02

#### TAO-05 — Local Docker PostgreSQL and media root

**Gate:** UL-07 (sequencing)

**Goal:** Persistent, loopback-only local infrastructure.

**Scope**
- Decide whether Go runs on the host or in Compose (02 OPEN-02, 13 OPEN-01).
- Compose service:
  - pinned PostgreSQL image, named volume, health check;
  - non-default credentials from a gitignored env file;
  - port bound to `127.0.0.1` only, or not published at all if Go runs in Compose.
- `MEDIA_ROOT` layout (`staging/`, `temporary/audio`, `audio/`, `temporary/images`, `images/`) with restrictive permissions, outside the source tree and container layers.
- Start/stop documentation that marks `docker compose down -v` as forbidden for routine use.

**Exit criteria**
- [ ] Recreating or redeploying the container keeps the data.
- [ ] The database port is unreachable from the LAN.
- [ ] Decision recorded.

**Depends on:** TAO-04

#### TAO-06 — Go/Gin backend skeleton

**Gate:** UL-07 (sequencing)

**Goal:** A runnable HTTP service that is safe by default.

**Scope**
- Go module and layout per 04 §2. Typed configuration validation: loopback by default, wildcard bind rejected, absolute `MEDIA_ROOT`.
- Gin middleware:
  - request ID, recoverer, 100 KB JSON cap, timeouts;
  - metadata-only structured logs;
  - Host/Origin validation for state-changing requests;
  - strict JSON decoding that rejects unknown fields and invalid UTF-8;
  - the error envelope from 10 §1.
- `pgxpool`, `/health/live`, `/health/ready` (database + media root) and graceful shutdown.
- CI: formatting, vet, static analysis, unit tests and the race detector.

**Exit criteria**
- [ ] Service starts on loopback and the health endpoints respond.
- [ ] Logs contain no request bodies.
- [ ] CI green.

**Depends on:** TAO-04, TAO-05

#### TAO-07 — SQL migrations, sqlc and seeds

**Gate:** UL-11 (sequencing)

**Goal:** A versioned schema workflow with idempotent seeding.

**Scope**
- Choose the migration runner (BE-OPEN-01). Configure sqlc with `pgx/v5`, and add a CI check that generated code hasn't drifted.
- First migrations: `users`, `learning_languages` (`zh-CN`, `ko-KR`), and `topics` (name key, system key, four colors).
- Seeds: one stable user, the enabled languages, and eight protected default topics. Rerunning the seed keeps IDs and user edits (NFR-15).
- Integration tests on a disposable PostgreSQL. Readiness checks the schema version.
- An early `pg_dump`/`pg_restore` smoke test into a fresh volume (01 §9 Stage 1 exit). The full paired procedure comes in TAO-44.

**Exit criteria**
- [ ] Migrate + seed is deterministic, both fresh and repeated.
- [ ] Drift check enforced in CI; dump/restore smoke passes.

**Depends on:** TAO-06

#### TAO-08 — OpenAPI contract pipeline

**Gate:** UL-11 (sequencing)

**Goal:** A contract-first API workflow (10 §7).

**Scope**
- The canonical TaoTale OpenAPI 3.1 file with conventions, the error envelope, `X-Request-Id`, health, `GET /languages` and `GET /topics`.
- Linting; TypeScript type generation for the client; and the Go conformance approach — generated interfaces or request/response validation tests (13 OPEN-02).
- CI fails on stale generated types or on a mismatch between contract and implementation.
- Rule for all later milestones: extend the contract before writing handlers.

**Exit criteria**
- [ ] `GET /languages` implemented and conformance-tested.
- [ ] Freshness checks enforced in CI.

**Depends on:** TAO-07

#### TAO-09 — SolidJS client scaffold on the published library

**Gate:** UL-20 (hard)

**Goal:** The client builds on the real, published UI package (03 §13.1 steps 1–3).

**Scope**
- Vite + SolidJS + strict TypeScript + `@solidjs/router`. Verify that lint and formatting handle Solid TSX. Set up component and end-to-end test tooling.
- Install the pinned UI library release from the registry. Verify ESM, types, CSS delivery and a single Solid runtime in TaoTale's build.
- A typed `fetch` wrapper (`/api/v1`, error envelope, request ID, `AbortController`) using the generated types.
- A typed local-preferences utility with validation and safe fallbacks (`taotale.preferences.*`, 03 §9).
- Serving:
  - in development, a proxy to Gin with an explicit origin allowlist;
  - in production, Gin serves the built SPA from the same origin, with an SPA fallback that never swallows `/api` 404s.

**Exit criteria**
- [ ] A production-like build served by Gin renders a page using the library Button.
- [ ] Refreshing a deep link works.
- [ ] Dependency guard passes.

**Depends on:** TAO-04, TAO-08

#### TAO-10 — App shell, routing and Quiet Study theme

**Gate:** UL-31 (hard)

**Goal:** The MVP's navigable, themed shell (08 §9.3 step 4).

**Scope**
- Lazy routes from 03 §4.1: `/` → `/library`, `/library`, `/create`, `/items/:id`, `/items/:id/edit`, `/vocabulary`, `/settings`, and not-found. No routes or code chunks for deferred features, and no sign-in page.
- Header navigation, and an active-language switcher (DropdownMenu radio items) fed by `/languages` and saved locally.
- Theme setting (light, dark or system), saved locally and applied before first paint with the library's first-paint helper.
- Quiet Study theme (08 §6.1), defined through the library's public theme API:
  - focus ring;
  - fix the known failing color pair (secondary text on subtle surface);
  - automated contrast check of every pair TaoTale uses.
- A convention for app-specific CSS (no Tailwind, no second generic styling system), base typography and a skip link.

**Exit criteria**
- [ ] Switching themes doesn't remount a stateful test component, and there is no theme flash on load.
- [ ] Navigation works by keyboard; contrast checks pass.

**Depends on:** TAO-09

### Phase 2 — Text and audio core

*Maps to 01 §9 Stage 2 and 13 §3 Stage 2.*

#### TAO-11 — Unicode offsets and fingerprints

**Gate:** UL-11 (sequencing)

**Goal:** The exact-text building blocks everything audio-related relies on.

**Scope**
- A Go converter between UTF-8 bytes, Unicode scalars and UTF-16 units (06 §2). It rejects offsets that fall inside a character or a surrogate pair, and rejects invalid UTF-8. Fixtures F3.
- The new canonical fingerprint serializer (schema-versioned, with the fields from 06 §4), with golden vectors for `1` vs `1.0`, non-ASCII text, `<`, `&`, emoji and CRLF.
- A legacy fingerprint verifier that reproduces the Mandarin Reader format, using the frozen vectors from TAO-01.

**Exit criteria**
- [ ] Golden vectors pass.
- [ ] Offset convention documented; no Go byte offsets are ever persisted.

**Depends on:** TAO-01, TAO-06

#### TAO-12 — Segmentation registry and Mandarin strategies

**Gate:** UL-15 (sequencing)

**Goal:** Versioned Chinese segmentation that keeps old recordings valid.

**Scope**
- A registry keyed by language code + strategy ID/version. No fallback to Chinese rules, and no Japanese placeholder.
- `zh-cn-legacy-v1`, reproducing the historical behavior exactly (06 §3.1), for existing recordings.
- `zh-cn-phrase` v1 for new generations. Its punctuation, quote and paragraph rules are frozen through owner-approved golden vectors (02 OPEN-03).
- Fixtures F1-CN and F2, including exact reconstruction of the gaps between segments.

**Exit criteria**
- [ ] Golden outputs approved; the decision is recorded in 02.

**Depends on:** TAO-11

#### TAO-13 — Korean segmentation strategy

**Gate:** UL-15 (sequencing)

**Goal:** An independent Korean strategy — not a fork of the Chinese one (06 §3.2).

**Scope**
- `ko-kr-phrase` v1, which handles:
  - sentence punctuation (ASCII and full-width), closing quotes and brackets, paragraphs;
  - decimals, abbreviations, ellipses and parentheticals;
  - mixed Latin text and digits;
  - long lines without punctuation, under a bounded policy.

  No morphology.
- Fixtures F1-KO; owner-approved golden vectors.

**Exit criteria**
- [ ] Golden vectors approved, and the Korean rules demonstrably differ from the Chinese rules where intended.

**Depends on:** TAO-12

#### TAO-14 — TTS provider adapter, voice catalog and mock provider

**Gate:** UL-15 (sequencing)

**Goal:** Azure kept behind a provider-neutral interface (TT-010).

**Scope**
- A domain `TTSProvider` interface: list voices, and synthesize → MP3 + boundary events + typed errors.
- The Azure adapter, productionized from TAO-03: backend-only credentials, region, timeout, cancellation and error mapping.
- A per-language voice allowlist with verified `zh-CN`/`ko-KR` voices and the rates each voice supports (02 OPEN-08), served by `GET /tts/voices?languageCode=`.
- A deterministic mock provider, visibly labelled "(mock)", used by every automated test. No paid calls in CI.
- With credentials missing, generation reports `TTS_UNAVAILABLE` and reading is unaffected.

**Exit criteria**
- [ ] Contract conformance passes; mock-based tests green.

**Depends on:** TAO-03, TAO-08

#### TAO-15 — MP3 measurement and alignment validation

**Gate:** UL-15 (sequencing)

**Goal:** Audio is marked READY only on verified evidence (06 §5–§6).

**Scope**
- Choose and verify an MP3 decoder/probe that measures duration (BE-OPEN-05, 13 OPEN-05).
- SSML generation with escaping and an index map back to the source text.
- Boundary normalization: validate the raw events, coalesce events per segment, and apply floor/ceil only after validation.
- The READY validation gate with a typed `ALIGNMENT_UNRELIABLE` error. The legacy 50 ms clamp applies only when validating legacy assets.
- Fixtures F4 and F5; replay the real outputs recorded in TAO-03.
- Decide the source-length policy: keep 1,500 UTF-16 units or plan chunking (02 OPEN-04).

**Exit criteria**
- [ ] The recorded `zh-CN` and `ko-KR` outputs validate; corrupt fixtures fail with typed errors.
- [ ] Tolerance policy documented.

**Depends on:** TAO-12, TAO-13, TAO-14

### Phase 3 — Items and generation backend

*Maps to 01 §9 Stage 2 and 13 §3 Stage 3.*

#### TAO-16 — Library item and draft API

**Gate:** UL-20 (sequencing)

**Goal:** Persist language-scoped items and drafts.

**Scope**
- Migration for `library_items` and `item_topics`, plus sqlc queries.
- Endpoints:
  - `POST /items`: a draft; language required and immutable;
  - `GET /items/{id}`;
  - versioned `PATCH`, which tells omitted, `null`, empty and `false` apart;
  - `DELETE`;
  - a basic `GET /items`: language required, newest first with stable ties, 8 per page, bounded page size.
- Validation:
  - limits from 01 §7, counting Unicode code points or UTF-16 units as specified;
  - difficulty is one of the three levels or `null` — `native` is rejected;
  - at most 10 owned topics;
  - the owner is resolved by the server, and unknown or foreign IDs return the same not-found.

**Exit criteria**
- [ ] Integration tests, including `409 VERSION_CONFLICT`.
- [ ] Contract conformance passes.

**Depends on:** TAO-07, TAO-08

#### TAO-17 — Topics API

**Gate:** UL-20 (sequencing)

**Goal:** One user-global topic catalog shared by both languages.

**Scope**
- `GET/POST /topics` and `PATCH/DELETE /topics/{id}`, with these rules:
  - names are unique case-insensitively and 1–40 code points long;
  - preset colors are resolved by the server (08 §6.2); custom colors are checked for contrast;
  - default topics are protected; deletion removes joins only;
  - duplicate errors include the existing ID; usage counts are returned.
- Settle `/topics` naming and whether any `/tags` alias is needed (13 OPEN-08).

**Exit criteria**
- [ ] Tests pass, including a seed rerun that keeps renamed or recolored defaults.

**Depends on:** TAO-07, TAO-08

#### TAO-18 — Generation admission and idempotency

**Gate:** UL-23 (sequencing)

**Goal:** Paid TTS requests are durable, never duplicated, and capped.

**Scope**
- Migrations for `audio_generations`, `audio_assets` (segments stored as JSONB) and `idempotency_records`.
- Endpoints:
  - `POST /items/{id}/audio/generate`: targets a saved draft, requires an `Idempotency-Key`, snapshots the saved inputs immutably (including strategy ID/version), returns `202`;
  - `GET …/generations/{id}`.
- Admission transaction:
  - writes PENDING and reserves the key; allows one active job per target;
  - same key + same intent replays the original;
  - same key + different intent → `409 IDEMPOTENCY_KEY_REUSED`;
  - tombstones remain after replay records expire.
- Cost caps: concurrency (initially 2) and an hourly limit (initially 60) return `429 TTS_RATE_LIMITED` before any provider call.
- Computed `NOT_GENERATED` and `OUTDATED` states.

**Exit criteria**
- [ ] Concurrent duplicate requests create exactly one job; cap tests pass.
- [ ] Contract conformance passes.

**Depends on:** TAO-15, TAO-16

#### TAO-19 — Generation worker, READY pipeline and job recovery

**Gate:** UL-23 (sequencing)

**Goal:** Turn admitted jobs into verified, immutable audio — or typed failures.

**Scope**
- In-process worker:
  1. atomically claim a job → PROCESSING with a heartbeat;
  2. call the provider outside any SQL transaction;
  3. staging file → MP3 check and measured duration → alignment validation → immutable promotion;
  4. one transaction creates the asset and sets READY (DB-1, DB-2).
- Typed FAILED codes; staging cleanup; no automatic resubmission.
- Startup reconciliation: interrupted jobs become FAILED with `GENERATION_INTERRUPTED` and are never resubmitted. Readiness waits until reconciliation finishes.

**Exit criteria**
- [ ] NFR-13: the same key never calls the provider twice (counted through the mock).
- [ ] A restart mid-job is handled correctly.
- [ ] A failed promotion never affects published assets.

**Depends on:** TAO-18

#### TAO-20 — Audio streaming, downloads and temporary previews

**Gate:** UL-23 (sequencing)

**Goal:** Safe playback and preview flows.

**Scope**
- Asset-pinned audio routes with single-range `206`/`416` responses, `audio/mpeg` and private caching. Downloads use safe, server-generated filenames.
- Temporary previews: `POST /tts/previews`, plus status, audio and download routes. Previews expire after 24 hours, are claimed on Save Draft (DB-3), and return a typed `PREVIEW_NOT_FOUND` once gone.

**Exit criteria**
- [ ] Range and pinning tests pass: a stale URL never serves new bytes.
- [ ] Claim and expiry tests pass.

**Depends on:** TAO-19

### Phase 4 — Create, publish and read

*Maps to 01 §9 Stage 2 and 13 §3 Stage 3.*

#### TAO-21 — Editor: new item and drafts

**Gate:** UL-33 (hard)

**Goal:** Explicit, safe draft editing (09 §5).

**Scope**
- `/create` and `/create?draftId=`. One store-based editor with a confirmed baseline, and memoized checks for "dirty" and "synthesis inputs changed" (03 §7).
- Fields:
  - language (new items only);
  - title;
  - exact source: a Textarea of at least 6 rows, with counters in the right counting unit;
  - description and notes;
  - difficulty: Beginner, Intermediate, Advanced or Unspecified.
- Saving:
  - explicit Save Draft, which replaces the URL with the `draftId`;
  - inline validation that focuses the first invalid field;
  - inputs kept on failure; a review step on version conflicts.
- Dirty guard: in-app navigation goes through `useBeforeLeave` and an AlertDialog (Save Draft & Leave / Leave Without Saving / Stay). Closing or reloading the tab uses the browser's native `beforeunload` (09 §9).
- Layout per 08 §7.2, for desktop and tablet.

**Exit criteria**
- [ ] End-to-end test: create → save → reload → continue.
- [ ] Every dirty-guard choice tested; no autosave.

**Depends on:** TAO-10, TAO-16

#### TAO-22 — Library list (basic)

**Gate:** UL-34 (hard)

**Goal:** The main entry point lists saved items.

**Scope**
- `/library`: a language-scoped grid (max ~1480 px, 24 px gaps, responsive columns for desktop and tablet).
- LibraryItemCard (a TaoTale component):
  - placeholder cover; title, or "Untitled draft";
  - description, difficulty, draft/published badge, dates;
  - an actions menu with Continue Editing or Open in Reader.
- Page number kept in the URL and clamped to the valid range.
- States: skeleton, true empty (offering Create New Item), error with Retry, and stale data while refreshing (09 §2).

**Exit criteria**
- [ ] End-to-end test with seeded data.
- [ ] One list request per page — no per-card requests.
- [ ] Fully keyboard-operable.

**Depends on:** TAO-10, TAO-16

#### TAO-23 — Editor: audio generation and preview

**Gate:** UL-35 (hard)

**Goal:** Explicit, cost-safe audio generation from the editor.

**Scope**
- Voice and rate pickers filtered by the item's language, defaulting to the per-language preferences saved locally.
- Generate Audio:
  - uses one idempotency key per intent;
  - after an unknown outcome, recovers with the same key and body;
  - never retries automatically;
  - unsaved synthesis inputs block generation until saved.
- Status shown with text labels (Not generated, Pending, Processing, Ready, Failed, Outdated). Bounded polling stops at a final state or when the page closes.
- Preview player (TaoTale's AudioPlayer, basic: play/pause, timeline, duration) on the asset-pinned candidate; candidate download; a temporary-preview path for unsaved items.
- Provider-unavailable and rate-limit states. A persistent explanation of why Publish is disabled.

**Exit criteria**
- [ ] End-to-end test with the mock: generate → Ready → preview.
- [ ] A backend restart mid-job shows the interruption without a duplicate provider call.
- [ ] Editing the source marks the audio Outdated.

**Depends on:** TAO-14, TAO-20, TAO-21

#### TAO-24 — Publication

**Gate:** UL-35 (sequencing)

**Goal:** Publish only verified, matching audio.

**Scope**
- Backend:
  - `POST /items/{id}/publish`: takes the expected version and re-checks the matching READY asset inside the transaction (DB-4);
  - direct `POST /items/publish`: claims a preview and requires an idempotency key.
- Client: Publish to Library, with a persistent explanation when disabled. On success, navigate to `/items/{id}` and replace the editor entry in history. On failure, the draft and its audio stay intact.

**Exit criteria**
- [ ] A mismatched or stale candidate can't be published (NFR-05).
- [ ] End-to-end test: create → generate → publish lands on the Reader route.

**Depends on:** TAO-23

#### TAO-25 — Reader page and transcript rendering

**Gate:** UL-35 (sequencing)

**Goal:** Exact, readable text, even when the audio isn't usable.

**Scope**
- Route states: a published item opens the Reader; a draft offers Continue Editing; an unknown ID shows not-found.
- Layout: a ~960 px shell, a ~780 px transcript, a fixed player area with bottom space, and no global header on the exact Reader route.
- Exact-source rendering from the source plus stored segments, validated against the JS string, with `pre-wrap` and blank lines kept. The client never re-segments the text.
- Language-appropriate local fonts: a Noto Serif SC subset for `zh-CN`; a separately tested Korean font stack.
- Fallback: if the alignment is corrupt or the media is missing, show readable text, disable seeking, and show an error with Retry.

**Exit criteria**
- [ ] `zh-CN` and `ko-KR` fixtures render byte-exact text.
- [ ] Fallback covered by an end-to-end test.

**Depends on:** TAO-24

#### TAO-26 — Audio player and synchronized highlighting

**Gate:** UL-35 (hard)

**Goal:** Playback that stays in sync with the text without slowing the UI down (03 §8, 06 §7).

**Scope**
- AudioPlayer on a native `HTMLAudioElement`: play/pause, ±5 s, speeds 0.75/1/1.25/1.5, timeline, elapsed/total, download.
- Sync engine:
  - a precomputed timing index searched with binary search;
  - the active-segment signal changes only when the segment changes;
  - a `requestAnimationFrame` loop runs only while playing;
  - the progress readout is isolated from the rest of the UI.
- Clicking or keyboard-activating a timed segment seeks to it and keeps the play state. Selecting or dragging text never triggers a seek. Ending playback clears the highlight; the next Play starts from 0.
- Resume position saved per `(itemId, audioAssetId)` every ~5 s and on pause. No autoplay when restoring.

**Exit criteria**
- [ ] NFR-03 met at every speed, for both `zh-CN` and `ko-KR`.
- [ ] An instrumented test proves the transcript doesn't re-render on each tick (NFR-04).
- [ ] Listeners and animation frames are cleaned up on navigation.

**Depends on:** TAO-25

#### TAO-27 — Reader actions: Favorite, Notes and Download

**Gate:** UL-35 (sequencing)

**Goal:** Metadata actions in the Reader that never interrupt playback.

**Scope**
- Backend `PUT /items/{id}/favorite` sets the value:
  - idempotent;
  - no change to `updatedAt` or the version;
  - never overwritten by later content updates (DB-6).
- Reader header:
  - Favorite / Remove from Favorites, with pending, confirmed and failure states;
  - a Notes dialog with explicit edit/save/cancel and dirty protection;
  - Download Audio and Back to Library.
- Favorite, Notes, theme and dialog changes never remount the player or the transcript.

**Exit criteria**
- [ ] The favorite survives a restart without a version change.
- [ ] Audio keeps playing through Notes and theme changes (tested).

**Depends on:** TAO-26

### Phase 5 — Library completeness

*Maps to 01 §9 Stage 3 and 13 §3 Stage 3.*

#### TAO-28 — Topics UI

**Gate:** UL-36 (hard)

**Goal:** Manage topics and assign them to items.

**Scope**
- Topic chips that use the stored light/dark colors (the library Chip with consumer colors).
- TopicAutocomplete in the editor: a multi-select Combobox with inline create, max 10.
- TopicManager dialog:
  - create, rename, and recolor via named swatches (RadioGroup);
  - delete custom topics, showing how many items and words use them;
  - default topics are protected;
  - a duplicate-name conflict keeps what the user typed.

**Exit criteria**
- [ ] End-to-end topic lifecycle passes.
- [ ] The same topic works on `zh-CN` and `ko-KR` items.

**Depends on:** TAO-17, TAO-21

#### TAO-29 — Library search, filters, sort and Favorites

**Gate:** UL-36 (hard)

**Goal:** The full Library browsing experience (01 §4.1).

**Scope**
- Backend `GET /items`:
  - filters: literal escaped title search (≤120 code points), status, difficulty including Unspecified, topics (OR), `favoritesOnly`;
  - sorts: newest, oldest, title A–Z and Z–A, with ID tie-breaks;
  - joins are batched, and indexes are justified by `EXPLAIN (ANALYZE, BUFFERS)`.
- Client URL state (03 §4.2):
  - parameters canonicalized;
  - search debounced 300 ms with history replace;
  - deliberate filter changes push a history entry;
  - the page resets or clamps.
- Filter bar: search, sort, difficulty, status, Favorites only and topics. An empty-filter state offers Clear filters.
- Card favorite star:
  - an icon button with a tooltip that sets the value and shows pending/failure;
  - it never opens the item;
  - unfavoriting under Favorites-only removes the card and keeps a sensible page.

**Exit criteria**
- [ ] Acceptance journey 5 (01 §10) automated.
- [ ] Favoriting never changes the newest-first order.

**Depends on:** TAO-22, TAO-27, TAO-28

#### TAO-30 — Cover upload pipeline

**Gate:** UL-37 (sequencing)

**Goal:** Safe, bounded cover images (TT-016).

**Scope**
- `POST /storage/images`:
  - bounded streaming, max 1,000,000 encoded bytes inclusive;
  - checks the real JPEG/PNG/WebP signature and the decoded dimensions (≤4096×4096);
  - protection against decompression bombs;
  - staging then promotion, with safe server-generated keys.
- `GET /storage/images/{id}` with ETag/304.
- A cover is attached or replaced through item updates, only after a successful save. Legacy oversized covers stay readable.

**Exit criteria**
- [ ] F6 fixtures pass, including 1,000,000 bytes accepted and 1,000,001 rejected.
- [ ] Path-traversal attempts are rejected.

**Depends on:** TAO-16

#### TAO-31 — Cover UI

**Gate:** UL-37 (hard)

**Goal:** Cover selection in the editor and covers on Library cards.

**Scope**
- An editor cover dropzone (the library FileDropzone):
  - client-side size, type and dimension checks that mirror the server's;
  - a thumbnail and remove/retry actions;
  - errors keep the old cover.
- Library cards load covers lazily and fall back to a placeholder when a cover fails.

**Exit criteria**
- [ ] Acceptance journey 7 automated.
- [ ] A failed replacement never removes the existing cover.

**Depends on:** TAO-21, TAO-22, TAO-30

#### TAO-32 — Deletion and downloads in the Library

**Gate:** UL-37 (sequencing)

**Goal:** Destructive and download actions done safely (09 §4).

**Scope**
- Delete:
  - an AlertDialog stating the consequences: the recording is removed, vocabulary entries survive;
  - a transactional delete;
  - after commit, release the item's own media without touching shared or still-referenced files.
- Download Audio for published items, with error states for missing media; correct pagination after a delete.

**Exit criteria**
- [ ] Delete covered by an end-to-end test.
- [ ] Media release never deletes referenced files (tested).

**Depends on:** TAO-20, TAO-22

### Phase 6 — Editing published content

*Maps to 01 §9 Stage 2–3 and 13 §3 Stage 3.*

#### TAO-33 — Published metadata edit

**Gate:** UL-37 (sequencing)

**Goal:** Change metadata without touching the recording (TT-023).

**Scope**
- `/items/{id}/edit` in metadata mode: title, description, notes, difficulty, topics, cover. The synthesis inputs are locked, with an "Edit Text & Voice" button that leads into a working revision.
- Versioned PATCH. Metadata changes never touch the active asset or its fingerprint. Review step on conflicts.

**Exit criteria**
- [ ] The metadata part of acceptance journey 4 is automated; the audio is unchanged after metadata edits.

**Depends on:** TAO-24, TAO-28, TAO-31

#### TAO-34 — Working revisions: backend

**Gate:** UL-37 (sequencing)

**Goal:** Edit text, voice or rate of published content without risking the published version.

**Scope**
- `working_revisions` migration. Endpoints: `POST /items/{id}/revisions` (200 existing / 201 new) and versioned GET/PATCH/DELETE.
- Revision generation. Candidate stream and download never fall back to the published bytes (D-12). A new revision starts as NOT_GENERATED (D-13).
- Revision publish (DB-5):
  - lock the item and the revision;
  - check the versions and the base asset;
  - swap source, voice, rate and asset atomically, keeping the metadata and favorite.

**Exit criteria**
- [ ] A failed or abandoned candidate leaves the published pair playable.
- [ ] A stale revision → 409.

**Depends on:** TAO-19, TAO-24

#### TAO-35 — Working revisions: client

**Gate:** UL-37 (sequencing)

**Goal:** The revision workflow in the editor (09 §5).

**Scope**
- Editor revision mode: Save Revision, Generate, Publish Revision, and Discard Revision (with confirmation). The dirty guard offers "Save Revision & Leave".
- The Reader keeps playing the published recording the whole time. Publishing a revision clears the old resume position.

**Exit criteria**
- [ ] Acceptance journey 4 automated end to end.

**Depends on:** TAO-23, TAO-33, TAO-34

### Phase 7 — Vocabulary

*Maps to 01 §9 Stage 2–3 and 13 §3 Stage 4.*

#### TAO-36 — Vocabulary data model and core API

**Gate:** UL-39 (sequencing)

**Goal:** Language-scoped vocabulary that can't mix languages (TT-007, TT-017).

**Scope**
- Migrations:
  - `vocabulary_entries`: unique per user + language + normalized key;
  - `vocabulary_references`: composite keys enforce the same owner and the same language, and `manual_linked OR auto_linked` must hold;
  - `vocabulary_topics`.
- Normalization — trim, NFC, collapse internal whitespace, case-sensitive — documented and tested. The original spelling is kept.
- Create, get, patch, delete; lookup; link/unlink references; the Reader projection `GET /items/{id}/vocabulary`; race-safe duplicate creation.
- Deleting an item removes its references but never the entries (re-test TAO-32).

**Exit criteria**
- [ ] F8 isolation tests pass: the same spelling in two languages is allowed, and a cross-language link is rejected.

**Depends on:** TAO-16, TAO-17

#### TAO-37 — Vocabulary list, statistics and bulk API

**Gate:** UL-39 (sequencing)

**Goal:** Server-side browsing and bulk management.

**Scope**
- `GET /vocabulary`:
  - language required;
  - search across spelling, reading and meaning;
  - status and topic filters, stable sorts, 10 per page.
- `GET /vocabulary/stats`.
- `POST /vocabulary/bulk`: atomic status change or delete, limited to owned, same-language IDs.
- Limits for reading/reading-system, meaning and note. Review the spelling limit for Korean phrases.

**Exit criteria**
- [ ] F7-scale query plans with no N+1 queries.
- [ ] Bulk atomicity tests pass.

**Depends on:** TAO-36

#### TAO-38 — Reader vocabulary collection

**Gate:** UL-39 (sequencing)

**Goal:** Collect words from the Reader explicitly (09 §6).

**Scope**
- Selection measured from DOM ranges in UTF-16, across timed segments and gaps. Selection rules are per language: the legacy Mandarin restrictions aren't applied to Korean without tests (D-07).
- Add to Vocabulary from a selection (a Popover anchored to it), and by keyboard from a focused segment (D-27).
- Lookup → link an existing word ("Already in Vocabulary", without overwriting its metadata), or open the Add dialog. Cancel creates nothing; the word and its reference are created atomically.

**Exit criteria**
- [ ] The collection part of acceptance journey 8 passes for both `zh-CN` and `ko-KR`.

**Depends on:** TAO-26, TAO-36

#### TAO-39 — Vocabulary view

**Gate:** UL-39 (hard, together with UL-38)

**Goal:** The `/vocabulary` management screen (08 §7.4).

**Scope**
- Summary stats; a server-paginated table (the library Table); search, status and topic filters; sorting; filters kept in the URL.
- Detail region:
  - edit word, reading, reading system, meaning, note and topics;
  - manual status;
  - references with drill-down to items;
  - delete.
- Bulk selection (a checkbox column with select-all) for status changes and deletion. A toast summarizes the result, in addition to the updated table.

**Exit criteria**
- [ ] End-to-end tests, including bulk operations, and a failed update that keeps the last confirmed status.

**Depends on:** TAO-28, TAO-37

#### TAO-40 — Reader vocabulary panel

**Gate:** UL-39 (sequencing)

**Goal:** See and manage the words linked to the open item.

**Scope**
- A Reader toolbar panel listing this item's linked words (manual references) with reading, meaning, topics, and manual Learning/Acquired toggles that show pending and failure states.
- Display settings (Both / Learning / Acquired), saved in `taotale.preferences.reader`.

**Exit criteria**
- [ ] Status changes refresh the panel and the Vocabulary view without remounting the audio.

**Depends on:** TAO-38

#### TAO-41 — Verified Chinese automatic matching (conditional)

**Gate:** UL-39 (sequencing)

**Goal:** Turn on automatic highlighting **only** where tests prove it is trustworthy (02 OPEN-05, 13 OPEN-06).

**Scope**
- Exact-match annotation for `zh-CN` (longest match, then earliest, then stable ID), behind a per-language capability flag.
- If enabled:
  - in-text underlines — Learning dotted, Acquired solid — with a legend;
  - word popovers rendered outside the transcript DOM;
  - publication-time reconciliation that keeps manual and automatic provenance separate;
  - `POST /items/{id}/vocabulary/sync`.
- For Korean (and for Chinese if not enabled): a typed "unsupported" response. No fake highlights or counts.

**Exit criteria**
- [ ] Decision recorded in 02.
- [ ] If enabled: annotation never disturbs selection, seeking or audio.
- [ ] This milestone may close as "Not enabled" without blocking the MVP.

**Depends on:** TAO-12, TAO-40

### Phase 8 — Settings, resilience and backup

*Maps to 01 §9 Stages 3–4 and 13 §3 Stage 4.*

#### TAO-42 — Settings and local data

**Gate:** UL-41 (sequencing). Uses UL-22, UL-32 and UL-39.

**Goal:** Local preferences and honest local-data information (01 §4.6, 09 §8).

**Scope**
- Preferences: theme, active language, per-language default voice and rate, default Reader speed and vocabulary popup visibility. All client-local, validated, with safe fallbacks.
- A system status endpoint with no secrets: Azure configured/available, database/schema status, media root health, app and segmentation versions.
- Local Data section: honest manual backup/restore guidance with copyable commands (a toast confirms the copy). No inert Backup button.
- Decide:
  - how much of the media path to show — 01 §4.6 asks to show it, while 10 and 11 forbid leaking absolute paths in errors and logs;
  - whether an in-app data reset exists. If yes, it needs confirmation and a prompt to back up first.

**Exit criteria**
- [ ] Corrupt or blocked browser storage falls back safely (tested).
- [ ] Decisions recorded.

**Depends on:** TAO-10, TAO-14, TAO-26, TAO-40

#### TAO-43 — Cleanup, recovery and integrity checks

**Gate:** UL-41 (sequencing)

**Goal:** The local data stays consistent over time (DB-9, 11 §5).

**Scope**
- Bounded cleanup (~every 60 min) of:
  - expired previews and unattached uploads;
  - superseded candidates (D-35);
  - orphan files.

  It uses fresh database checks and file leases, and never deletes active assets, revision bases or the latest candidates.
- A read-only integrity check comparing database references with files, checksums and durations.
- Disk-full, permission and missing-file handling with typed errors.

**Exit criteria**
- [ ] Fault fixtures for missing media and a full disk pass.
- [ ] Cleanup never deletes live media.

**Depends on:** TAO-19, TAO-20, TAO-30, TAO-32

#### TAO-44 — Paired backup and restore

**Gate:** UL-41 (sequencing)

**Goal:** Backups that are proven to restore (TT-005, 11 §7).

**Scope**
- Backup procedure/scripts:
  1. quiesce writes and the worker;
  2. a custom-format `pg_dump`;
  3. an archive of the media;
  4. a manifest: timestamp, schema revision, app and segmentation versions, checksums.

  Store backups outside Docker volumes and the source tree.
- Restore rehearsal into a fresh volume and an empty media root, followed by an integrity comparison. Commands for the target OS (02 OPEN-06, BE-OPEN-07).

**Exit criteria**
- [ ] NFR-08: the restored instance has zero missing active assets, and identical fingerprints, timings, favorites, topics and vocabulary.

**Depends on:** TAO-43

### Phase 9 — Legacy migration

*Maps to 13 §3 Stage 5. 01 §9 puts data migration in Stage 1. Here the import waits until the full schema exists; Stage 1's protection (backup and inventory) is done early in TAO-01.*

#### TAO-45 — Legacy import tooling

**Gate:** UL-41 (sequencing)

**Goal:** Bring Mandarin Reader data into TaoTale without loss (TT-022).

**Scope**
- An importer that reads a **copy** of the legacy database and media, and preserves:
  - UUIDs, timestamps and exact source bytes;
  - topics (tags → topics) and their colors;
  - statuses, and references (the legacy source pointer becomes a manual reference);
  - audio bytes, with their legacy fingerprint and `zh-cn-legacy-v1` identity.
- Backfill: `zh-CN`, `is_favorite=false`, pinyin → `reading` with `reading_system='pinyin'`, and the Learning/Acquired status.
- Report `native` or other unexpected difficulty values and covers over 1,000,000 bytes. Apply only owner-approved mappings (02 OPEN-07); never delete covers.
- Idempotent: running it twice gives the same result.

**Exit criteria**
- [ ] Dry-run report approved.
- [ ] The import runs twice on disposable copies with the same result.

**Depends on:** TAO-01, TAO-11, TAO-12, TAO-34, TAO-36

#### TAO-46 — Legacy verification and rehearsal

**Gate:** UL-41 (sequencing)

**Goal:** Evidence that the imported data behaves exactly as before.

**Scope**
- Compare counts, checksums, active assets, references and Reader offsets. Legacy published items must play with unchanged highlighting, and legacy covers must stay readable.
- Optional compatibility (03 §4.2, §9):
  - a one-time migration of legacy `mr.*` browser keys, if served from the same origin;
  - a redirect for legacy `/library?tab=vocabulary` URLs.

**Exit criteria**
- [ ] Rehearsal report shows zero loss.
- [ ] Rollback path documented.

**Depends on:** TAO-26, TAO-45

### Phase 10 — Hardening and MVP release

*Maps to 01 §9 Stage 4 and 13 §3 Stages 4–5.*

#### TAO-47 — Accessibility and supported-device QA

**Gate:** UL-42 (hard — the library's audit fixes are in place)

**Goal:** Meet NFR-11 and NFR-12 on every supported device.

**Scope**
- Accessibility:
  - keyboard-only journeys and visible focus;
  - focus never hidden behind the fixed player (WCAG 2.2 SC 2.4.11);
  - labels and status semantics;
  - contrast for Quiet Study and the topic palettes;
  - targets ≥44 px where specified; reduced motion.
- Devices: desktop at 1280/1440, representative tablet viewports, 200% zoom, touch; Chromium, Firefox and WebKit. Write down the supported matrix first (13 OPEN-10).

**Exit criteria**
- [ ] NFR-11 and NFR-12 met.
- [ ] No critical defects from automated or manual checks.

**Depends on:** TAO-21 to TAO-42

#### TAO-48 — Performance baselines and budgets

**Gate:** UL-43 (hard — the library's budgets are in place)

**Goal:** Measured performance with enforced budgets (03 §10.2, 12 §6).

**Scope**
- An F7 dataset generator: 1,000 published items and 100 drafts, 10,000 words and 100 topics across both languages.
- Measure NFR-01 to NFR-04 on the reference machine:
  - API p95 and first useful render;
  - Reader sync accuracy and input delay;
  - memory stability;
  - JS/CSS size, including the UI library.
- Set numeric budgets and add CI regression checks.

**Exit criteria**
- [ ] Baseline report published.
- [ ] Budgets recorded, and regressions fail CI.

**Depends on:** TAO-46, TAO-47

#### TAO-49 — Fault-injection, security and privacy verification

**Gate:** UL-43 (sequencing)

**Goal:** Evidence for the resilience and local-security claims (11 §6, §8).

**Scope**
- Faults:
  - restart mid-generation and abrupt termination;
  - full disk and permission errors;
  - missing or corrupt media and alignment;
  - database outage and migration checksum mismatch;
  - expired previews and duplicate keys.
- Security:
  - Host/Origin checks, DNS rebinding and cross-site writes;
  - path traversal, symlink escape and hostile uploads;
  - log redaction and no secrets in bundles or logs;
  - dependency, licence and secret scans, including a provenance check of the UI library.

**Exit criteria**
- [ ] Evidence recorded for NFR-06, NFR-09 and NFR-13.

**Depends on:** TAO-44, TAO-46

#### TAO-50 — Real Azure bilingual release smoke

**Gate:** UL-43 (sequencing)

**Goal:** Prove the real provider works end to end in both languages (06 §9).

**Scope**
- An explicit, billable, owner-approved run on the target OS for both `zh-CN` and `ko-KR`: create → generate → READY → publish → synchronized Reader → vocabulary → restart and reopen.
- A human listening check.

**Exit criteria**
- [ ] Evidence recorded.
- [ ] Any language that can't produce reliable timing blocks the release (12 §3).

**Depends on:** TAO-49

#### TAO-51 — MVP acceptance, release report and cutover

**Gate:** UL-44 (hard — TaoTale ships on the frozen 1.0 API)

**Goal:** Accept, document and switch over to the MVP.

**Scope**
- Upgrade to the library's 1.0 release candidate. Run the 12 journeys from 01 §10 and 12 §5, the NFR gates, and the scope gate (no deferred routes, login or billing).
- Release report (12 §6): commit, migration revision, OpenAPI hash, UI library version, real test counts, coverage, audits, performance, restore checksums, known limitations.
- Cut over from the legacy app using the verified import, keeping backups for rollback.
- Move to library `1.0.0` GA once it is published (a patch-level change).

**Exit criteria**
- [ ] Owner acceptance recorded in 02.

**Depends on:** TAO-47 to TAO-50

### Phase 11 — Post-MVP

*Maps to 01 §8, 07 and 13 §5.*

Each item needs a scope decision in 01 and 02 first (TAO-52). Each is broken into small milestones with full Goal/Scope/Exit sections when it is scheduled. None of them may be prebuilt during the MVP.

| ID | Milestone | Gate | Notes |
|---|---|---|---|
| TAO-52 | Post-MVP scope approval and ordering | UL-45 (sequencing) | Closes 02 OPEN-09; picks the order; updates 01 and 02. |
| TAO-53 | Dashboard | UL-45 (sequencing) | 01 §8.1: active language, Continue Reading, recent items, Favorites, per-language counts. |
| TAO-54 | Vocabulary Practice | UL-48 (hard: Collapsible) | 01 §8.2: manual recall and reveal, explicit status changes; no SRS or scoring. |
| TAO-55 | AI provider re-validation spike | UL-45 (sequencing) | 07 §6: re-check the provider, authorization and credential storage on the target OS. Billable only with approval. |
| TAO-56 | AI text-generation backend | UL-45 (sequencing) | 07 §4–§6: an isolated adapter, idempotent claims, output validation, no automatic writes. |
| TAO-57 | AI workspace UI (`/generate`) | UL-45 (hard) | 07 §3, §8: lazy route, reviewable candidates, data-sharing disclosure, hand-off to the existing draft/TTS/publish flow. |
| TAO-58 | Language Islands Builder | UL-48 (hard) | 01 §8.4: reuse items, jobs and the Reader; hide/reveal prompts for rehearsal. |
| TAO-59 | Phone and small-screen support | UL-47 (hard; includes UL-46) | Phone layouts and navigation, touch-friendly transcript interactions, testing on real devices. |
| TAO-60 | Japanese (`ja-JP`) support | UL-45 (sequencing) | Its own segmentation strategy, voices and fonts; Chinese rules are never reused. |
| TAO-61 | Optional enhancement backlog | per item | FEAT-01 to FEAT-07 (13 §5), one-click and encrypted backup, local usage tracking, LAN access (needs its own security decision). |

---

## 8. Open decisions and where they close

| Open item | Source | Closed in |
|---|---|---|
| Exact versions and build tooling | 02 OPEN-01, 13 OPEN-02 | TAO-04 (toolchain), TAO-07 (DB stack), TAO-08 (OpenAPI tooling), TAO-09 (client) |
| Go on the host vs in Compose; DB port; migration tool | 02 OPEN-02, 13 OPEN-01, BE-OPEN-01/02 | TAO-05, TAO-07 |
| Segmentation rules and offset convention | 02 OPEN-03, 13 OPEN-04, BE-OPEN-04 | TAO-11, TAO-12, TAO-13 |
| Source length vs chunking | 02 OPEN-04, 13 OPEN-04 | TAO-15 |
| Azure Go SDK runtime and boundary fidelity | 13 OPEN-03, BE-OPEN-03 | TAO-03 |
| MP3 decoder/probe | 13 OPEN-05, BE-OPEN-05 | TAO-15 |
| Automatic vocabulary matching | 02 OPEN-05, 13 OPEN-06, BE-OPEN-06 | TAO-41 |
| Backup workflow and packaging | 02 OPEN-06, BE-OPEN-07 | TAO-44 |
| Legacy difficulty values and oversized covers | 02 OPEN-07, 13 OPEN-07 | TAO-01 (report), TAO-45 (decision) |
| TTS limits, voices, provider configuration | 02 OPEN-08 | TAO-14, TAO-18 |
| Post-MVP order | 02 OPEN-09 | TAO-52 |
| UI package name, licence, repository | 02 OPEN-10, 13 OPEN-09 | UL-02, UL-03, UL-05 → recorded in TAO-02 |
| UI CSS delivery, token prefix, slots, variants | 02 OPEN-11 | UL-05, UL-06 → applied in TAO-09 and TAO-10 |
| `/topics` vs `/tags` | 13 OPEN-08 | TAO-17 |
| Tablet/browser matrix, performance thresholds | 13 OPEN-10 | TAO-47, TAO-48 |
| How much media path to show; in-app data reset | 01 §4.6 vs 10/11 | TAO-42 |

## 9. MVP acceptance traceability

The 01 §10 journeys, the milestones that build them, and where each is verified:

| # | Journey (01 §10) | Built in | Verified in |
|---|---|---|---|
| 1 | Local Docker/PostgreSQL startup, no login, data survives restart | TAO-05, TAO-06, TAO-07, TAO-10 | TAO-49, TAO-51 |
| 2 | Manual content: draft → Azure audio → validated → publish → read/listen/download | TAO-21, TAO-23, TAO-24, TAO-25, TAO-26, TAO-27 | TAO-50, TAO-51 |
| 3 | Language-specific segmentation | TAO-12, TAO-13, TAO-15, TAO-19 | TAO-50, TAO-51 |
| 4 | Revisions keep the published recording safe | TAO-33, TAO-34, TAO-35 | TAO-51 |
| 5 | Favorites persist and filter | TAO-27, TAO-29 | TAO-51 |
| 6 | Difficulty: three levels, no regeneration, no `native` | TAO-16, TAO-21, TAO-29 | TAO-51 |
| 7 | Cover limit 1,000,000 / 1,000,001 bytes | TAO-30, TAO-31 | TAO-51 |
| 8 | Vocabulary collected and managed per language | TAO-36 to TAO-40 | TAO-51 |
| 9 | Provider failure keeps reading working, with no duplicate billing | TAO-14, TAO-18, TAO-19 | TAO-49 |
| 10 | Persistence, fault recovery and restore | TAO-43, TAO-44 | TAO-49 |
| 11 | Supported-device and accessibility QA | all UI milestones | TAO-47 |
| 12 | Scope gate: no deferred features exposed | TAO-10 | TAO-51 |

## 10. Risks

| Risk | Mitigation |
|---|---|
| Library-first gating slows visible app progress — more so now that the library builds its own behavior layer (TT-032), which adds milestones before the first overlays | Backend work runs in sequencing-gated blocks; library components are ordered by TaoTale's needs; if the library slips, the owner may record a decision to relax a *sequencing* gate (never a hard gate). |
| Chinese or Korean alignment isn't feasible from Go | TAO-03 runs first; a failure is handled as an explicit blocker; timing is never fabricated. |
| UI library API churn during 0.x | Exact version pins, the upgrade checklist (03 §12), and the MVP ships on the 1.0 RC. |
| Generic controls creep into TaoTale while waiting for the library | CI dependency guard (TAO-04), the no-bypass rule, and review checklists. |
| Legacy data loss | Verified backup (TAO-01), rehearsals on copies only, an idempotent importer and a rollback path. |
| Duplicate paid Azure calls | Idempotency (TAO-18), the mock provider in CI, and explicit billable milestones only. |
| Post-MVP scope creep | Phase 11 needs a scope decision first; the scope gate in TAO-51. |
| The unauthenticated local service gets exposed | Loopback binding, Host/Origin checks, verified in TAO-49. |
