# 03 — TaoTale client architecture

**Status:** Approved frontend direction; implementation details and package versions are subject to verification.

**Authority:** [`01-taotale-overview-and-features.md`](01-taotale-overview-and-features.md) defines MVP scope and product behavior; [`02-taotale-decision-register.md`](02-taotale-decision-register.md) tracks adopted and historical decisions. This document defines **TaoTale application architecture**, not the reusable UI component library. The standalone **SolidJS-only npm UI library**, its styling/theming APIs and TaoTale Quiet Study design are specified in [`08-taotale-design-system.md`](08-taotale-design-system.md). TaoTale **consumes that package**; it does **not** implement or directly depend on Kobalte.

## 1. Architectural decisions

TaoTale is a **locally hosted, single-user, desktop/tablet web application**. Its UI is a client-side SPA talking to a local backend through a typed HTTP API; it never connects directly to PostgreSQL or accesses the backend's media directory. The existing internal `User` domain record is resolved by the backend. **No login, onboarding, billing, hosted SaaS shell, or phone-size support is required for MVP.**

### 1.1 Selected frontend stack

| Concern | Decision | Rationale / boundary |
|---|---|---|
| UI framework | **SolidJS** with strict **TypeScript** | Fine-grained reactive DOM updates; no React runtime or React compatibility layer. |
| Build and development | **Vite** | SPA build, dev server, lazy chunks; backend can serve the production build from the same origin. |
| Routing | **`@solidjs/router`** | Official Solid router for URL state, nested views, navigation and leave guards. |
| Styling | **No Tailwind CSS** | Consume the UI package's published CSS and theming contract. TaoTale may author CSS for **app-specific layouts/domain components**; package internals and token/override rules are specified in 08. |
| Generic UI and design system | **Independent, versioned, npm-published SolidJS-only UI library** | TaoTale imports its public Button/Input/Dialog/Tooltip/Menu/etc. components. The package owns generic behavior, styling, accessibility, focus, overlays and transitions; TaoTale supplies its Quiet Study theme. |
| Headless primitives | **No Kobalte — in TaoTale or in the UI package** | No app import or declared dependency on `@kobalte/core` or any other headless UI library. The UI package implements its own accessible primitives (TT-032); its internals are not an app API. |
| Local/derived state | `createSignal`, `createStore`, `createMemo`, `createContext` as appropriate | Prefer Solid primitives and component-local state; no default global-state dependency. |
| Asynchronous state | Solid resources and/or Solid Router `query` + `createAsync` + targeted revalidation | Use the primitives already available in the selected stack; do not introduce a general-purpose request-cache library by default. |
| HTTP transport | Browser **`fetch`** + narrow typed API wrapper | Same-origin `/api/v1`; typed errors, cancellation and explicit mutation behavior without Axios. |
| API contracts | Generated TypeScript types from the backend's OpenAPI specification | Keep the design-first contract principle, including backend/client compatibility checks. |
| Form handling | Native form elements, Solid signals/stores and reusable validation helpers | No React Hook Form or new form library unless a demonstrable requirement warrants one. |
| Preferences | Small typed local-storage adapter + Solid state | No Zustand, Redux, or application-wide persistence framework. |
| Audio | Native `HTMLAudioElement` | No audio-player framework; use browser media APIs directly. |

**Not selected as TaoTale app dependencies:** React, React Compiler, React Router, MUI, Zustand, TanStack Query, Axios, React Hook Form, **Kobalte (`@kobalte/core`) or Tailwind CSS**. The separate UI package does not use Kobalte either (TT-032). Removing historical dependencies does not remove associated user-facing behavior.

### 1.2 Minimal-dependency policy

1. **Solid-first:** use Solid's native reactivity, control flow, contexts, stores, resources, effects, lazy loading and lifecycle helpers before reaching for another runtime package.
2. **Platform-first:** prefer HTML forms, semantic elements, `fetch`, `AbortController`, `URLSearchParams`, `Intl`, CSS, and `HTMLAudioElement` where they solve the problem correctly.
3. **Use the npm UI library for generic controls:** import its public components for dialogs, popovers, menus, select/combobox, tooltips, buttons, inputs and transitions. **Do not import Kobalte** (or any other headless UI library) or duplicate those accessibility/overlay mechanisms in TaoTale. The package implements them itself (TT-032).
4. **Avoid component-library stacking:** no second headless library, MUI, Tailwind, alternate design-system framework, CSS-in-JS runtime, animation framework or broad icon package without a demonstrated gap.
5. **Require justification for a new application runtime dependency:** identify why Solid, browser APIs and the **UI package public APIs** are insufficient; check bundle/maintenance impact and a measurable benefit. If a generic capability is missing, consider adding it **to the independent UI library**, not bypassing it.
6. **Development tooling is a separate category:** tests, type generation, linting, formatting and build tools are allowed when useful; minimizing *runtime* dependencies must not compromise quality.
7. **Do not reimplement hard solved problems poorly:** the goal is low total complexity and good performance, **not** the smallest possible `package.json` at the expense of accessibility, correctness or security.

**Version policy:** pin compatible **SolidJS, Solid Router, Vite and npm UI-library versions** when initializing the client. Test package peer-dependency compatibility and bundled CSS/types in the consuming app. No historical React-era version pins apply.

## 2. Client organization and dependency boundaries

Use **feature-oriented modules with a thin route layer**. Keep components unique to a route or feature close to that feature. Generic reusable control implementations belong in the **independent UI-library project**; TaoTale may share app-specific domain compositions when there is a concrete reuse case.

```text
src/
  app/
    App.tsx                 # root/router/shell composition
    routes.tsx              # route definitions and lazy imports
    theme/                  # apply Quiet Study tokens via the UI library public theme API
  routes/
    library/                # route composition, route-only components
    create/                 # create/edit route compositions
    reader/                 # published-item route composition
    vocabulary/             # vocabulary route composition
    settings/               # settings/local-data route composition
  features/
    library/                # items, filters, pagination, favorites, API
    editor/                 # create/draft/revision forms and dirty state
    tts/                    # voice selection and generation UI/state
    reader/                 # transcript, timing, playback integration
    vocabulary/             # entries, references, selection, status
    topics/                 # topic selection and management
    settings/               # preferences and local-backup actions
  components/
    ui/                     # optional THIN app-specific wrappers of npm UI exports; no generic reimplementations
  shared/
    api/                    # fetch wrapper, generated contract types
    lib/                    # pure utilities, validators, URL helpers
    styles/                 # app-specific layouts + UI package stylesheet/theme import
  assets/                   # local app assets and font subsets
```

- **Dependency direction:** app/routes → features → shared **and public exports of the external UI package**. Optional `components/ui` contains only thin application-specific wrappers/configuration; it must not implement generic Button, Tooltip, Dialog, Menu, focus traps or transitions. The independent UI package must never import TaoTale modules.
- A route composes features and owns URL integration; it should not become a large repository of domain logic. A route-only component may live under that route. Domain-reusable components belong to the relevant feature.
- Cross-feature interactions use explicit public functions/interfaces, not imports of another feature's private internals. Avoid circular dependencies.
- Keep text offsets, segmentation/timing validation, highlighting, vocabulary matching, DTO conversion and URL normalization as **pure, testable functions** outside components. **Reader, AudioPlayer, LibraryItemCard, TopicManager and VocabularyEntryDetail are TaoTale feature components**, not exports of the generic npm UI package.
- Component names use `PascalCase`. Name Solid reactive factories `createX` where appropriate, and context accessors `useX` only when they are actual context helpers; do not import React hook patterns by convention.
- Avoid a generic abstraction merely for one call site. Use composition and small, typed functions rather than introducing extra frameworks or service layers.

## 3. SolidJS reactivity and performance rules

Solid components **set up reactive bindings**; they do not run again for every signal update like React function components. Write code for that model rather than copying React idioms.

- `createSignal` for small mutable values (active segment ID, modal visibility, selected IDs, local input). `createStore` for structured editable state when property-level updates help.
- `createMemo` for expensive or multiply-consumed derivations. Avoid `createEffect` for deriving state that can be expressed directly or through a memo; reserve effects for integration with external systems such as storage, audio, document title or observers.
- Keep reads of reactive state **inside tracked scopes** (JSX expressions, memos, resources/effects). Do not destructure reactive props in ways that lose tracking; use `props.foo`, or Solid helpers when needed.
- Use `<For>` for item/vocabulary/segment lists and `<Show>` / `<Switch>` for conditional UI. Preserve stable item identities; do not replace whole transcript trees on each playback tick.
- Isolate high-frequency changes. An audio progress indicator can update frequently; the Library, Reader header, entire transcript, notes and vocabulary data should not recompute because `currentTime` changed.
- Keep event handlers and local state near the component that needs them. Share reactive state through a small context **only** when distant branches truly depend on it; avoid one giant global store.
- `onCleanup` must remove media event listeners, animation frames, observers and subscriptions. Cancel obsolete fetches; guard against stale responses overwriting newer route/filter results.
- Prefer native `classList`, conditional class bindings and CSS attributes over a runtime class-name construction library.
- For large lists, paginate before considering virtualization. Add virtualization only after profiling indicates DOM size is a problem; it increases keyboard navigation and accessibility complexity.

**Performance principle:** measure user-observable behavior (initial JS/CSS, route readiness, input responsiveness, memory, and playback stability), not just framework microbenchmarks. SolidJS helps but does not excuse redundant requests or expensive text processing.

## 4. Routes and canonical URL state

### 4.1 MVP routes

| Route | Purpose |
|---|---|
| `/` | Redirect to `/library`. |
| `/library` | Language-scoped mixed Library of saved drafts and published items, including Favorites. |
| `/create` | New unsaved item editor. |
| `/create?draftId={id}` | Continue a saved draft. |
| `/items/{id}` | Published Reader; a draft points to Continue Editing. |
| `/items/{id}/edit` | Published metadata edit or entry to working revision. |
| `/items/{id}/edit?revisionId={id}` | Existing saved working revision. |
| `/vocabulary` | Language-scoped vocabulary dashboard. |
| `/settings` | Local preferences, provider settings and storage/backup information. |
| Unknown | Not-found with route back to Library. |

**Not in MVP:** `/generate` (AI text/audio workspace), Dashboard route, Vocabulary Practice route, Language Islands Builder, sign-in/onboarding and phone-specific navigation. These must not be included as lazy chunks or dead-end MVP navigation items.

### 4.2 URL behavior

- Use `@solidjs/router` for route transitions and `useSearchParams`/`useLocation` for reactive URL state. Do not duplicate authoritative filter/page values in another global store.
- **Library:** selected language, `search` (max 120 code points), `difficulty` (`beginner` / `intermediate` / `advanced` or absent), topic IDs, status (`draft` / `published` / all), **`favoritesOnly`**, sort (`newest`, `oldest`, `title_asc`, `title_desc`), page ≥1. An absent difficulty means *no difficulty filter*; item difficulty itself may be Unspecified (`null`).
- **Vocabulary:** selected language, query, `Learning` / `Acquired` / all, topic IDs, sort, page and selected entry if needed. It has a **separate route**, not a Library tab; filter names need not inherit the old `v*` namespace.
- Explicitly select learning language for new items; persist a last-used default in local preferences. Published Reader language always comes from the item's stored data, never a conflicting global selection.
- Canonicalize recognized values: deduplicate/sort selected topic IDs, omit default filters, clamp exhausted pages, reject invalid search values, keep deterministic result ordering. Topic filter OR applies within topics, AND across groups.
- Debounced search typing (initial target 300 ms) uses URL history replace; deliberate filter and navigation changes use push where appropriate. Reset page to 1 on filter change. Preserve browser Back/Forward and focus behavior.
- On Save Draft, replace the unsaved route with its saved draft URL; on Publish, navigate to the Reader without leaving a stale editable route in history.
- If supporting bookmarked legacy `/library?tab=vocabulary` URLs, redirect them to `/vocabulary` with a **one-time, explicit URL migration**; do not maintain two sources of vocabulary navigation state indefinitely.

## 5. Async data, caching, mutations and error handling

**No extra query-cache dependency by default.** Use `createResource` for isolated async data and Solid Router `query` + `createAsync` + `revalidate` when several routes/components need coherent reuse. Choose **one authoritative retrieval strategy per resource**, not a custom cache layered on top of a router cache.

- The **backend is authoritative** for items, drafts/revisions, topics, voices, favorite flags, vocabulary, generation jobs, and audio metadata. Client preferences are separate.
- **Library:** one paginated item request (initial page size 8) with embedded topic/cover/audio metadata. **Vocabulary:** server-paginated results (initial page size 10) and separate language-wide counts. Reader requests its item's linked vocabulary as a dedicated projection; avoid N+1 network requests per word or card.
- Use contract-generated request/response types; write a small API boundary for HTTP status, typed error code/message, request ID, JSON parsing and cancellation (`AbortController`). Use same-origin relative `/api/v1` and asset-pinned media routes.
- Treat a successful mutation response as confirmation before showing *Saved*, *Favorited*, *Published*, etc. A failed favorite toggle restores/displays the last confirmed flag; it must not invalidate or replace an audio asset.
- After a successful mutation, **revalidate only affected data**: for example favorite changes affect relevant Library/Reader item data; topic edits affect visible topic labels; vocabulary edits affect list/detail/stats/Reader projection. Do not remount or reset the audio player to refresh metadata.
- **No automatic retry of paid TTS generation or writes.** For an unknown generation outcome, recover with the original immutable request/key instead of unintentionally issuing another Azure call. Read-only requests may offer explicit Retry.
- Offline or backend-unavailable mutations fail visibly; do not queue them behind the user's back. Keep stale but valid previously loaded data with an error indicator and Retry when appropriate.
- Image upload is independent, validated on client and server. For **new or replacement covers, reject files >1,000,000 bytes**, invalid signatures or dimensions without truncation; preserve the editable form and the existing cover on failure. Historical oversized covers remain readable until deliberately replaced/migrated.
- Use backend generation state (`Pending`, `Processing`, `Ready`, `Failed`, `Outdated`) and a bounded polling/revalidation mechanism while a job is active. Stop polling on terminal state, route disposal or disconnection.

## 6. External design-system and UI-library integration boundary

The **authoritative specification already exists** in [`08-taotale-design-system.md`](08-taotale-design-system.md). It distinguishes (a) an independently developed, **SolidJS-only**, npm-published UI component library and (b) the TaoTale-specific **Quiet Study** theme and page composition. This client document defines the **consumer integration**, not generic component implementations or the library's public styling API.

- **TaoTale uses the published UI package** for generic controls (Button, Input, Dialog, AlertDialog, Tooltip, Popover, Menu, Select, etc.) where available. Add the UI package as a versioned app dependency; import only **public package exports**, not its source files, private CSS selectors or undocumented internals.
- **No Kobalte:** do not declare `@kobalte/core` as an application dependency, import its primitives, or wrap Kobalte in feature code. **The standalone UI library does not use Kobalte either**: it implements its accessible primitives in-house (TT-032), and TaoTale never depends on its internals.
- **No Tailwind CSS:** neither TaoTale nor the UI library requires Tailwind as a runtime/build/styling dependency. The UI package publishes its own styled components/CSS and documented theme tokens, variants, parts and customization hooks as specified in 08. TaoTale authors **only application-specific styles** for page composition and domain features.
- **Ownership:** the UI library owns reusable ARIA/keyboard/focus behavior, tooltip timers, overlay dismissal/positioning, transition/presence behavior and reduced-motion defaults. TaoTale supplies domain data, dialogs' content/actions, navigation, reading/text-selection rules, audio synchronization, error states and **end-to-end accessible composition**.
- **Theme:** TaoTale applies Quiet Study using the package's **public theme API/CSS custom properties**, with light/dark/system selection persisted locally and applied before first paint. Do not fork package components or reach into private classes merely to change their appearance.
- **Missing controls:** if a capability is generally reusable, propose/implement it in the npm library and publish a compatible package release. If it is TaoTale-specific (e.g., timed transcript, audio player, TopicManager, LibraryItemCard), implement it in the app by composing public library controls.
- **Testing:** verify the packaged library as an installed dependency (`npm pack`/prerelease as needed), with CSS, TypeScript declarations, peer Solid runtime, theme overrides, keyboard navigation and reduced motion. The **UI-library project** owns generic component tests; TaoTale owns integration, route and workflow tests.

**Unselected details**—npm package name, exact CSS export/token naming, component variants and packaging implementation—remain in document 08, not here. These are not grounds to add local Kobalte components while waiting for the library.

## 7. Create/Edit and dirty-navigation behavior

Use one editor supporting **new item, saved draft, published metadata edit, and working revision**. Do not bring back React Hook Form solely to mimic historical implementation.

- Maintain a typed `createStore`/signals form model and a separately captured confirmed baseline. Derive `isDirty`, source-affecting changes, metadata-only changes, and validation with memoized/pure helpers. Do not conflate **synthesis inputs** with a favorite/title/topic edit.
- Inputs update from Solid event handlers. For text inputs that need immediate state, use `onInput` rather than relying on blur-driven `change` behavior.
- Explicit Save Draft, Generate Audio and Publish only; **no automatic draft persistence**. Preserve all unsaved input after API failures. Read draft/revision source, voice and speaking rate from persisted state, including intentionally unset values.
- Saved-target generation must use persisted synthesis inputs. Unsaved source/voice/rate changes block generation/publishing until saved or reconciled according to the backend workflow. Show fingerprint mismatch/outdated state instead of pretending preview is current.
- Switching item language is a new-item action; never silently retag a published item's text, vocabulary or old audio into another language. Voices and synthesis controls are filtered by that language.
- Apply the selected **language-specific, versioned segmentation** on the backend; the client presents segment/timing diagnostics but never substitutes its own different segmentation for a published recording.
- Use Solid Router `useBeforeLeave` for internal route changes including Back; implement Save & Leave / Leave Without Saving / Stay. Only continue blocked navigation after a successful save or explicit discard. Use native `beforeunload` for refresh/close/external navigation. Never falsely claim that reload work is automatically recoverable.
- Version conflicts surface a review/reload choice. A failed reload must not erase dirty form data; any targeted rebase (such as topic detachment) must be justified by confirmed server state.
- Keep Notes independently editable with explicit save/cancel and equivalent dirty-state protection.

## 8. Reader rendering and audio performance

The Reader is the most performance-sensitive part of TaoTale. **Audio playback must remain independent of most reactive UI updates.**

- Hold a **native `HTMLAudioElement` reference** and attach playback listeners once per element/asset; clean up on disposal. Audio asset URL is pinned to the immutable recording ID.
- Reconstruct ordered, exact-source text runs from persisted source offsets and timed segments **once per recording**, not per animation frame. Preserve whitespace and blank paragraphs (`white-space: pre-wrap`). Do not use different client-side segmentation logic to recalculate historical timings.
- Use a precomputed ordered timing index and binary search for current segment. On media time updates, publish a reactive **active segment ID/index only when it changes**. Keep rapidly updating progress/time readout in a small isolated child.
- Prefer media events or one playback-scoped `requestAnimationFrame` loop only if smooth progress needs it; stop on pause, end, error, hidden/unmounted state. Do not create a signal/effect for every time tick across the whole transcript.
- Preserve play/pause, native timeline, ±5-second seek, speeds, current/duration, download, no-autoplay restore and position persistence per `(itemId, audioAssetId)`.
- Keep original selection semantics: UTF-16 source offsets, phrase selections spanning timed runs, selection/drag suppressing click-to-seek; keyboard-visible Add to Vocabulary remains available.
- Compute vocabulary annotation matches only when source text or relevant linked spellings change. Status and popup decoration should not recompute matching ranges. Preserve segment highlighting independently of word highlighting.
- **Language-aware boundary:** Reader uses persisted segment spans regardless of language; any automatic vocabulary matching requires a separately verified strategy for `zh-CN` or `ko-KR`. Manual phrase collection is always available. Do not transfer Mandarin substring matching blindly to Korean.
- Render overlays/popups outside transcript text flow using **public components from the npm UI library**, avoiding text offsets or DOM selection disruption. One active popup at a time; preserve keyboard focus on dismissal.
- If audio alignment is malformed or media missing, **still show readable exact text**, while disabling misleading seeks/playback and showing a recoverable error. Vocabulary collection and Notes remain usable.
- Favorites and Notes changes must not reset the recording or rebuild the transcript; Reader and Library read the **same confirmed favorite flag** from backend data.

## 9. Client-local preferences and storage

Most study data belongs in **local PostgreSQL and filesystem assets through the backend API**. Browser storage holds only small client/device preferences and the position of a specific recording. Do not use it as an article database.

| Suggested new key | Data |
|---|---|
| `taotale.preferences.language` | Last selected learning language (`zh-CN` / `ko-KR`). |
| `taotale.preferences.synthesis` | Per-language voice ID and synthesis rate defaults. |
| `taotale.preferences.player` | Playback-rate default. |
| `taotale.preferences.reader` | Learning/Acquired popup visibility and related display preferences. |
| `taotale.preferences.theme` | Light / dark / system. |
| `taotale.playback.v1.{itemId}.{audioAssetId}` | Last position and update time; approximately every 5 seconds / on pause. |

- Centralize schema checks and fallbacks in a tiny typed storage utility; storage may be unavailable or malformed and must not crash startup. Do not persist provider secrets or generation credentials in browser local storage.
- If needed for the migration, read historical `mr.*` keys **once**, validate/migrate their values, and write new `taotale.*` keys; do not keep both naming schemes indefinitely.
- Playback restore must not autoplay; when the asset changes or the recording ends, reset/clear the old position. No cross-device playback sync is promised.
- Backups/restores are initiated through backend endpoints or documented local tooling. **The browser does not connect to PostgreSQL, Docker, or the server filesystem directly.**

## 10. Loading, performance evidence and testing

### 10.1 Loading discipline

- Lazy-load **Library, Create/Edit, Reader, Vocabulary and Settings** route modules with `lazy`; load heavier editors, specialized dialogs, topic manager or vocabulary details on demand where beneficial. Do not ship unused AI/dashboard/practice/island code in MVP.
- Keep the primary application shell small. Prefer a useful loading skeleton and an actionable error/retry experience over a large all-purpose loading library.
- Serve the built SPA and backend API through the same local origin when deployed; Vite's development proxy may target the local backend. Keep backend and PostgreSQL bound to loopback by default.
- Optimize cover thumbnails, lazy-load offscreen images, avoid eager Reader word-detail calls, and use local/subset CJK fonts only in contexts needing them. Korean typography requires an appropriate fallback/font strategy; do not apply Chinese serif fonts indiscriminately.
- Apply theme before displaying the app and preserve mounted media playback during small metadata/theme interactions.

### 10.2 Performance targets and instrumentation

Before setting hard numeric performance budgets, capture repeatable baselines on a representative PC and browsers. Record dataset shape, build version, network conditions and traces. Benchmark at least:

1. Cold startup: JS/CSS transferred, route usable time, initial memory.
2. Library: filtered/sorted paginated cards, Favorite toggle, stale-result refresh and navigation.
3. Editor: long text input, frequent keystrokes, topic selection, dirty checking and cover upload.
4. Reader: a 1,500-UTF-16-unit text with many timed segments/vocabulary annotations, continuous audio playback, seek, selection and popup operations.
5. Vocabulary: representative large **database**, server-paginated table, search/filter and bulk status updates.
6. Theme and language changes: no unnecessary remount, audio interruption or large JS layout work.

**Release gates:** responsive input during playback, no accumulating listeners/timers, stable browser memory across repeat navigations, no per-card request storm, tested keyboard and screen-reader interactions, and a documented bundle/performance regression review. Establish numerical budgets after the first working Solid build, then treat regressions as actionable.

### 10.3 Test strategy

- **Pure unit tests:** source-offset handling, timing lookup, filter normalization, dirty-state derivation, storage validation, status transitions, language-aware annotation boundaries.
- **Component tests:** TaoTale feature composition/integration (Reader actions, forms, Library cards, vocabulary). **Generic Button/Dialog/Tooltip/Menu behavior is tested in the independent UI-library project**, with consumer smoke tests in TaoTale. Choose Solid-compatible test tools as needed.
- **Browser E2E:** Playwright for Library/Favorites, drafts, editor guards, Reader/audio, vocabulary, error/restore flows, keyboard/focus, light/dark and desktop/tablet viewports. Add automated accessibility checks where practical; they do not replace manual keyboard testing.
- **API contract tests:** generated client types stay in sync with backend OpenAPI. Mock providers at boundaries for deterministic development; do not trigger real paid Azure generation as a side effect of ordinary tests.
- Keep linting/formatting as development tools (e.g. oxlint/Prettier after confirming Solid/TSX support); do not adopt a runtime utility library just because the former React project used it.

## 11. Current MVP boundaries and inherited behavior

| Concern | MVP requirement | Explicitly deferred / removed |
|---|---|---|
| Supported learning languages | Simplified Mandarin and Korean, each with compatible voices and versioned backend segmentation. | Japanese and additional languages until their capabilities are implemented/tested. |
| Difficulty | Optional **Beginner**, **Intermediate**, **Advanced**. | `Native` option; automatic difficulty inference. |
| Library | Mixed drafts/published, Favorites, topics, filters, paging, covers ≤1 MB on new upload. | Dashboard, public sharing, advanced collections. |
| Content creation | Manual text, Azure TTS via adapter, verified publish/revisions. | AI Text/Audio Generation Workspace (`/generate`) and Language Islands Builder. |
| Reader | Synchronized, validated segment highlighting and vocabulary collection. | Word-level dictionary/TTS, automatic acquisition. |
| Vocabulary | Language-scoped manual words, refs, statuses and administration. | Vocabulary Practice and SRS. |
| Devices | Desktop and tablet browser support. | Phone/small-screen acceptance and phone-specific interaction patterns. |
| Infrastructure | Local backend, Docker-hosted PostgreSQL, local assets, internally resolved User. | Public SaaS UI, registration, billing or client-side database access. |

The **Azure TTS adapter** remains part of the backend integration in MVP. The **chosen AI-provider adapter** may remain available as an architectural boundary but **does not create a required AI UI, startup connection, login flow or AI generation feature for MVP**.

## 12. Architecture risks and safeguards

| Risk / anti-pattern | Safeguard |
|---|---|
| React mental model copied into Solid | Review for destructured props, mistaken component re-renders, hooks and unnecessary effects; write Solid-native reactive code. |
| Minimal dependencies interpreted as hand-building inaccessible controls | Import accessible generic controls from the **independent UI library**; do not write app-level Kobalte wrappers or inaccessible substitutes. The library builds and tests these primitives itself (TT-032). |
| One global store for all app state | Route URL state, server-derived data, editor state, media state and preferences each have a clear owner. |
| Duplicate query cache layers | Use router queries or isolated resources with documented targeted refresh; no automatic extra cache package. |
| Audio time broadcasts to whole app | Native media element + isolated progress display + active segment changes only. |
| Theme or metadata update remounts Reader | Stable route/player identity and scoped reactive updates. |
| Styling conventions diverge across features | Use **public npm UI components** and the theme contract in 08; avoid duplicated generic controls, private package selectors and uncoordinated styling conventions. |
| Client assumes language-neutral segmentation or vocabulary morphology | Persist server timing; isolate language-aware matching; manual selection remains fallback. |
| New `/generate`, Dashboard, Practice or Island modules creep into navigation | Keep them outside MVP routes, bundles and acceptance criteria. |
| Local-only app exposes backend or secrets | Loopback-only server; same-origin API; no provider tokens in client storage. |
| Tooling choices dictated by legacy React code | Treat React-era stack as superseded; app dependencies are **SolidJS/Router/Vite/UI package**, not direct Kobalte or Tailwind. |
| UI package update breaks theme, focus or playback interactions | Pin compatible UI-library version; test external-package installation, theme CSS, overlay focus, and Reader stability before upgrade. |

## 13. Migration plan and unresolved choices

### 13.1 Migration sequence

1. Initialize **SolidJS + TypeScript + Vite + Solid Router**, strict TSX checks and a minimal app CSS entry. **Do not install Kobalte or Tailwind in TaoTale.**
2. Install and pin a build of the **independent SolidJS-only npm UI package** (published version or packaged prerelease). Verify `npm pack`/consumer installation, ESM/types/CSS, theme API, keyboard/ARIA, transitions and no duplicate Solid runtime; apply the Quiet Study theme from 08.
3. Establish the typed `fetch` API boundary and local development/production hosting. Verify API and generated types, including error handling and asset URLs.
4. Compose package UI primitives into TaoTale feature components and implement **Library** with language selection, Favorites, filters, pagination and saved-item actions. Do **not** recreate the package's generic buttons/menus/dialogs in-app.
5. Implement **Create/Edit**, drafts, covers, voice selection, Azure TTS job states and conflict/dirty guards using Solid stores/resources and generic UI-package controls.
6. Implement the **Reader** with persistent native media element, app-owned timed-segment renderer, Notes, Favorites and manual vocabulary collection.
7. Implement **Vocabulary**, Topics and Settings; check language scoping, storage keys, keyboard/tablet accessibility.
8. Complete profiling, dependency/bundle review, failure-state testing, local restart/restore and regression acceptance. Do **not** gate MVP on postponed views.

### 13.2 Open implementation choices (not reversals of approved decisions)

- Pin exact **SolidJS, Solid Router, Vite and external UI-library package versions** after checking compatibility, exports and TSX tooling. A local SPA does not require SolidStart/SSR by default.
- Choose a consistent rule for **route queries vs component resources**, including targeted invalidation and stale-data presentation, to avoid accidental duplicate data ownership.
- Use the **existing `08-taotale-design-system.md`** for the UI package CSS/theming API, slots, styling overrides, animations and accessibility contracts. TaoTale still needs an implementation convention for **app-specific CSS** but must not create a second generic styling system or use Tailwind.
- Define testable **bundle and interaction performance budgets** from a first functioning build, rather than claiming unmeasured speedups.
- Confirm client-facing fields/routes from the backend contract for languages, favorite toggling, difficulty, revision state and **1 MB covers** before implementation.

### 13.3 Reference documentation

- [SolidJS — Fine-grained reactivity](https://docs.solidjs.com/advanced-concepts/fine-grained-reactivity)
- [SolidJS — `createResource`](https://docs.solidjs.com/reference/basic-reactivity/create-resource)
- [Solid Router — Search parameters](https://docs.solidjs.com/solid-router/concepts/search-parameters)
- [Solid Router — Leave guards](https://docs.solidjs.com/solid-router/reference/primitives/use-before-leave)
- [Solid Router — Query revalidation](https://docs.solidjs.com/solid-router/data-fetching/revalidation)
- [`08-taotale-design-system.md`](08-taotale-design-system.md) — independent SolidJS-only npm library with in-house accessible primitives (no Kobalte), theming, component API and TaoTale Quiet Study visuals.

---

**Maintenance note:** the matching current decisions are recorded in [`02-taotale-decision-register.md`](02-taotale-decision-register.md), especially TT-027–032: **SolidJS**, minimal runtime dependencies, **external npm UI package**, **no Kobalte in TaoTale or the library**, **no Tailwind**, and the separate design-system contract in 08. The overview remains authoritative for MVP scope; the npm library is responsible for reusable generic accessible UI.
