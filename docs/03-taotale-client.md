# 03 — TaoTale client architecture

**Status:** Approved simple frontend direction, revised **25 November 2026**. **Authority:** [01 — Product](01-taotale-overview-and-features.md), [02 — Decisions](02-taotale-decision-register.md), [08 — Iris design](08-taotale-design-system.md), [09 — UI states](09-taotale-ui-states.md).

## 1. Goal and architecture

TaoTale is a **local, single-user, desktop/tablet-oriented web application**. The frontend is a small SolidJS SPA. It consumes the Go backend over a same-origin typed REST API; it does not access PostgreSQL or private media paths directly. **All UI code and all component styling live inside TaoTale's frontend repository.** There is **no standalone UI library, UI npm package, headless component framework or component-package publication task**.

The design should feel like a polished reading product, not a component-system showcase: expressive **iris/violet/lavender**, calm neutral surfaces, refined typography, restrained shadows and generous spacing. **Beautiful appearance with minimal implementation complexity** is the explicit trade-off.

### 1.1 Stack

| Concern | Decision |
|---|---|
| Framework | **SolidJS**, strict TypeScript, Solid signals/resources and component-local state |
| Bundler | **Vite** |
| Routing | **`@solidjs/router`** |
| Styling | **CSS Modules** for components/pages; one small global stylesheet for reset, typography, layout tokens and theme variables |
| Design | Application-owned **Iris theme**, specified in [08](08-taotale-design-system.md), with light/dark/system modes |
| Controls | Ordinary native HTML: `button`, `input`, `textarea`, `select`, `label`, `details`, links, HTML audio; native `<dialog>` only when a real modal is needed |
| Generic UI | A **small in-project `src/components/ui/` folder**, created only for repeated needs (Button, Input, Card, Badge, Dialog wrapper, Spinner). These are private app components, **not a public API** |
| Local state | `createSignal` / `createStore` / `createMemo` where useful; no global-state dependency by default |
| Async data | `createResource` or router `query`/`createAsync`; choose a single approach for each resource |
| HTTP | Browser `fetch`, `AbortController`, narrow typed `/api/v1` wrapper, generated OpenAPI TypeScript types |
| Forms | Native forms, typed handlers and shared pure validation helpers |
| Audio | **Native `HTMLAudioElement`** and application-specific Reader synchronization |
| Preferences | Small validated `localStorage` adapter |

**No UI framework/library integration:** do not add Eroxo, a separate component package, Kobalte, MUI, Radix, Tailwind, CSS-in-JS runtime, tooltip/overlay positioning engine or animation framework to the MVP. There is no `npm pack`, external theme API, UI-library version pin, component SDK, slot contract, or component release workflow.

**No accessibility framework/project:** there is no WCAG certification, comprehensive ARIA abstraction, custom focus trap, custom keyboard-navigation system, menu/typeahead framework or overlay manager. Use native elements correctly, keep visible labels, readable contrast and obvious interaction states. **Do not remove ordinary browser keyboard behavior or meaningful error messages.** The domain-specific JavaScript for the Reader/audio, forms, fetching and navigation remains necessary.

## 2. Simple source organization

Keep route-specific code near its route, and extract shared UI only after a second real use case.

```text
frontend/
  src/
    app/
      App.tsx                 # shell and router
      routes.tsx
    routes/
      library/                # LibraryPage, LibraryCard, local CSS modules
      create/                 # editor page and local controls
      reader/                 # ReaderPage, transcript and player UI
      vocabulary/             # VocabularyPage and forms
      settings/               # settings and local-data guidance
    features/
      library/                # list filters, favorites, item requests
      editor/                 # draft/revision state, validation, generation UI
      reader/                 # audio clock, segments, selection, annotation
      vocabulary/             # words, language filtering, references
      topics/                 # topic selector/manager
      tts/                    # voices and job polling
    components/
      ui/                     # small app-local Button, Field, Card, Badge, Modal
      layout/                 # app navigation, page container
    shared/
      api/                    # typed fetch + OpenAPI types
      lib/                    # pure utilities and validators
      styles/
        global.css            # reset, base typography, common element styling
        tokens.css            # Iris tokens + light/dark themes
        utilities.css         # only a few reused layout helpers, if needed
    assets/                   # app visuals, SVGs, locally shipped fonts if chosen
```

**Rules:** `routes → features → shared`; domain components may import `components/ui`; avoid circular feature dependencies. Components unique to one page stay with that page. Do not build a generic design-system engine or add a shared abstraction for one use. Use regular `.module.css` files rather than runtime styling logic.

## 3. Component strategy: simple but polished

- **Native-first:** make `<button>` look excellent with CSS instead of building button behavior from scratch. Use native `<select>` rather than custom listboxes; use `<input type="file">` for covers, styled with a visible label; use `<input type="range">` for timeline/rate where appropriate.
- **App-local building blocks:** a `Button` may expose `variant="primary|secondary|ghost|danger"` and `size="sm|md"`; a `Card` mainly wraps a styled container. Props are chosen for TaoTale's current screens only. No polymorphic `as`, slots APIs, CSS state protocol, theme-provider package or exhaustive variants.
- **Dialogs:** prefer clear inline panels and normal page forms. For delete/discard confirmation, native `window.confirm` is acceptable and simplest. If a styled modal is important, use native `<dialog>` with a thin Solid wrapper invoking `showModal()` and `close()`. **Do not implement focus traps or portal stacks in JavaScript.**
- **Menus/tooltips:** prefer visible actions or a simple `<details>` dropdown; native `title` is optional for supplemental help. Do not add floating-placement, hover-timer, nested-menu or interactive tooltip systems.
- **Animation:** only subtle CSS `transition` on background, border, transform and shadow; avoid runtime motion libraries, elaborate enter/exit controllers and animation-driven state.
- **Icons:** a small set of static SVG assets or inline SVGs in the app. Keep icon use consistent; avoid an icon SDK unless measured reuse clearly warrants it.
- **Style integrity:** shared CSS variables and repeated border/radius/spacing rules keep the UI coherent. User-facing status always includes clear wording, not just a colored dot.

See [08](08-taotale-design-system.md) for actual Iris tokens and page-level styling rules. **Polish is achieved with visual consistency, spacing, typography, artwork, and careful screen composition—not a component infrastructure project.**

## 4. SolidJS and performance

Solid components set up fine-grained reactive bindings rather than rerunning like React components. Keep reactive reads in tracked JSX or memos; avoid effects for derivations. Use `createSignal` for input/visibility/playback values, `createStore` for structured forms, `<For>` for stable lists and `<Show>` for conditional fragments. Clean up media listeners, timers and active requests.

Reader audio clock updates must not rebuild the transcript or the whole page. Derive an active segment index from the verified timing data and update only the active visual treatment when it changes. Avoid per-card requests, unnecessary fetch cache layers and premature virtualization. Performance should be checked in the actual application, not justified by framework claims.

## 5. Routes and URL state

| Route | Purpose |
|---|---|
| `/` | Redirect to `/library` |
| `/library` | Language-scoped drafts/published items, Favorites, search/filters/sort/page |
| `/create` | New item editor |
| `/create?draftId={id}` | Continue saved draft |
| `/items/{id}` | Published Reader; draft routes to Continue Editing |
| `/items/{id}/edit` | Published metadata edit or entry to revision |
| `/items/{id}/edit?revisionId={id}` | Saved working revision |
| `/vocabulary` | Selected-language vocabulary |
| `/settings` | Local preferences, Azure availability and backup guidance |

Do not add Dashboard, Practice, AI workspace (`/generate`), Language Islands, login/billing or phone-only routes to MVP.

`@solidjs/router` owns URLs and browser history. Keep Library language, title search (max 120 code points), topics, difficulty, status, `favoritesOnly`, sort and page in query parameters. Vocabulary has its own language/query/status/topics/sort/page. Topic filter uses OR inside its group and AND against other filters. Reset page to 1 when filters change; use a modest ~300ms search debounce and history replace. Do not duplicate URL-owned filters in a global store. The Reader uses **its item's persisted language**, not a global selection.

## 6. Server data, errors and mutations

Use generated OpenAPI types and a small `fetch` helper for JSON parsing, typed error codes, request IDs and cancellation. Do not introduce Axios or a general-purpose query-cache dependency by default. Backend state is authoritative for items, Topics, words, Favorites, voices, jobs and audio metadata; browser storage is only for preferences and playback resume.

- **Library:** server-paginated mixed list (initial **8 cards/page**); stable filters/sort. **Vocabulary:** server-paginated list (initial **10 rows/page**) plus language-specific counts.
- Preserve previously confirmed data while a background refresh is in progress where appropriate. Distinguish initial loading, real empty, filtered empty and recoverable errors.
- Confirm mutations before showing Saved/Favorited/Published. A failed Favorite write shows last confirmed state. Update only affected data; never reset audio because the item title or Favorite changed.
- Reject new/replacement covers over **1,000,000 encoded bytes** or invalid type/dimensions; backend remains authoritative. Preserve older oversized legacy covers on read.
- Generation states: `NOT_GENERATED`/`OUTDATED` computed, and persisted `PENDING`/`PROCESSING`/`READY`/`FAILED`. Poll a known job only while relevant. Never auto-retry **paid Azure TTS** or an uncertain write; recover the original idempotency key/body first.
- Errors must be understandable, not raw SQL, stack traces or provider secrets. Offline writes fail explicitly; no hidden offline mutation queue.

## 7. Create/Edit behavior

Use one editor for **unsaved new item, saved draft, published metadata edit and saved working revision**. Maintain local form state plus the last confirmed baseline; derive dirty state and synthesis-affecting changes through pure functions. Inputs use `onInput` when immediate updates matter. There is **no autosave**.

- New item chooses `zh-CN` or `ko-KR`; only compatible voices appear. Saved/published item language does not silently change.
- Explicit **Save Draft**, **Generate Audio**, **Publish**; published text/voice/rate changes go through a working revision. Metadata (title, notes, description, difficulty, Topics, cover, Favorite) does not invalidate valid audio.
- Saved-target synthesis uses **saved** text/voice/rate; unsaved synthesis changes require save before Generate/Publish. READY must match the exact immutable candidate and verified alignment.
- Use straightforward inline field errors, status panels and disabled/pending actions with explanations. Do not create a generic form engine.
- Protect dirty editing against accidental navigation with a modest `useBeforeLeave` flow: **Save & Leave**, **Discard & Leave**, or **Stay**. A small native confirm is acceptable; navigate only after save succeeds. Browser reload/close uses native `beforeunload` warning. Preserve input after failure/version conflict.
- Notes use plain text and explicit save/cancel. Delete and revision discard confirm destructive consequences.

## 8. Reader and audio requirements

The Reader is **app-specific**, not a generic component library problem. Render the exact stored source with paragraph breaks and UTF-16 half-open segment ranges. Keep the native audio element mounted while themes, notes, popups and Favorites change. Source and recording are pinned to the published immutable asset.

- Native audio: play/pause, time, seek, ±5 seconds, rate, verified duration, download and errors; no autoplay on restore. Save resume by item **and audio asset ID**; reset on asset change/end.
- Clicking an eligible timed phrase can seek to its validated start; dragging/selecting text must not accidentally seek. No custom keyboard navigation or accessibility system for every segment is required. Ordinary native controls remain keyboard-usable.
- The smallest possible domain logic tracks player position, active span and text selection. **Do not add overlay-positioning infrastructure**; vocabulary collection may use a persistent action near the Reader toolbar plus a simple in-page panel or native dialog.
- Selecting words/phrases → look up by **item language** → link an existing word or create a new entry with explicit confirmation. Manual linking always works; language-specific automatic highlighting remains conditional on verified matching.
- Corrupt alignment or missing audio must leave **readable text and vocabulary actions** intact while disabling inaccurate playback seeking. No invented timestamp or silent client resegmentation.
- Active Reader highlighting should use a tasteful transparent iris wash; Learning and Acquired vocabulary have distinguishable underline styles. Keep the transcript clean and readable.

## 9. Preferences and theme

Store small validated local values: active language, light/dark/system theme, per-language voice/rate defaults, Reader playback speed and word-popup display setting. Use `taotale.preferences.*` keys; record resume at `taotale.playback.v1.{itemId}.{audioAssetId}`. Validate/correct invalid stored values; no provider secrets in localStorage.

A theme is a **CSS attribute/class on the application root** (`data-theme="light"|"dark"`), driven by a small preference setter; system mode resolves `prefers-color-scheme`. No theme-provider package, nested theme architecture or custom hydration mechanics. Set the resolved theme before showing the app when practical and **never remount the Reader/player** just to change colors.

## 10. Application QA and implementation boundaries

Test actual workflows (Library, Favorites, draft/publish, Reader/audio, Vocabulary, Settings), the CSS light/dark themes, desktop/tablet layout and the states in [09](09-taotale-ui-states.md). Check visible form labels, readable text, native controls, basic pointer and keyboard use, no destructive-action mistakes, no horizontal overflow and no Reader/player clipping. **No separate UI package tests, formal WCAG audit or custom overlay/focus-management test suite** is in scope. Test the complex domain logic thoroughly: timing, exact source offsets, idempotency, migration, recovery and backups.

Performance check: Library startup, filters, editor typing, active-segment updates, long transcript and switching theme during playback. Do not count old Mandarin Reader tests as a new TaoTale pass. See [12](12-taotale-quality-and-acceptance.md).

**Strict MVP boundary:** manual Library + Favorites + Editor/Azure generation + Reader + Vocabulary/Topics + Settings, with Chinese and Korean. Dashboard, Vocabulary Practice, AI text workspace, Language Islands, account/billing, phone-specific UI and separate UI-library development remain out of scope.
