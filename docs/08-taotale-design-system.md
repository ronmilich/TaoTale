# 08 — TaoTale design system and SolidJS UI library

**Status:** Product and architecture specification for the standalone UI library and its TaoTale integration. Package name, exact public prop names, and first release version remain to be selected.

**Authority:** `01-taotale-overview-and-features.md` determines TaoTale's MVP scope; `03-client.md` determines the SolidJS application architecture; this document defines (1) the reusable UI-library contract and (2) the TaoTale-specific theme, visual rules, and composition. The old Mandarin Reader design and D-24/39–46/48–52/61 are historical references, not permission to reintroduce deferred features.

## 1. Core decision and product boundaries

TaoTale will **consume an independently developed, published npm UI-component library**. That library is itself a reusable product for other developers building **SolidJS applications only**. It is **opinionated**: components have polished defaults, coherent styling, predictable interaction states, and accessibility behavior out of the box. It is also **customizable**: developers can change themes, tokens, variants, component styles, and composition without forking the library.

- **Framework:** SolidJS and TypeScript. No React, Vue, Angular, Web Components compatibility layer, framework-neutral virtual DOM, or dual framework builds.
- **Distribution:** a separately versioned package published to the public npm registry. The npm package name and repository topology are **TBD**. The library must be usable outside TaoTale, independently of TaoTale's app, backend, and domain data.
- **Styling:** authored/compiled CSS with a public customization contract. **No Tailwind CSS** or consumer Tailwind configuration required. Do not choose a runtime CSS-in-JS engine by default. Internal CSS organization (plain CSS versus CSS Modules, etc.) is an implementation choice as long as published CSS and override mechanisms are reliable.
- **Behavior:** the library implements reusable client-side UI behavior such as keyboard interaction, focus management, overlay positioning/presence, tooltip timing, menus, accessible dialogs, transitions, and reduced-motion handling. Prefer native browser capabilities and SolidJS primitives; use **Kobalte internally where appropriate** rather than reimplementing complex accessibility from scratch.
- **Responsibility:** the library owns reusable UI mechanics and presentation APIs; **TaoTale owns product workflows**, language-aware reading, text selection, audio playback and synchronization, vocabulary matching, persistence, API requests, routes, and local preferences.
- **Performance:** minimize runtime dependencies and avoid work on unrelated reactive updates. Dependency count is a constraint, not a reason to compromise accessible interaction behavior.

### 1.1 Responsibility matrix

| Concern | SolidJS UI library | TaoTale application |
|---|---|---|
| Buttons, input controls, labels, cards, badges, feedback | Implementation, variants, states, styles | Choose and compose |
| Dialog, menu, popover, tooltip, select/combobox | Reusable focus/ARIA/keyboard/overlay behavior and styles | Supply content, conditions, domain actions |
| Transitions, presence, reduced-motion support | Reusable, configurable behavior | Choose when/how components open and close |
| Theme tokens, theming API, component override hooks | Stable public contract and sensible generic defaults | Supply Quiet Study theme and persisted preference |
| Library cards, Reader transcript, synchronized player, vocabulary UI | **Not domain components** | Build by composing library controls and app-specific components |
| Routes, API calls, mutations, error recovery, persistence | No dependencies or assumptions | Complete ownership |
| Design-system documentation | Public component/API/theming docs | App-specific layouts, theme palette, UX acceptance |
| Accessibility | Accessible defaults and documented responsibilities | Accessible page composition and end-to-end verification |

The UI package **must not import** TaoTale modules, schemas, endpoints, fonts, topic models, or reading vocabulary concepts. TaoTale may depend on the UI package; never the reverse.

## 2. Library implementation and packaging contract

### 2.1 SolidJS-first architecture

- Implement components as native SolidJS components using fine-grained reactive primitives (`createSignal`, `createMemo`, `createEffect` only for actual side effects, context for shared configuration); avoid React-style repeated component rerenders, mirrored state, or a second reactive runtime.
- Prefer semantic HTML (`button`, `a`, `input`, `label`, `fieldset`, native forms) for controls that do not need a headless primitive.
- Where behavior is nontrivial, wrap/compose selected `@kobalte/core` primitives. **Kobalte is an internal implementation choice**, not a mandatory consumer API and not a license to export its internal structure as the library's public contract.
- Do not require developers to install Kobalte directly, learn Kobalte-specific composition for basic library components, or maintain ARIA/focus/tooltip timing themselves. Preserve useful advanced composition slots rather than hiding essential capabilities.
- The package exposes predictable controlled and uncontrolled forms where meaningful (`value`/`defaultValue`/change callback; `open`/`defaultOpen`/open-change callback). Never silently mix both modes.
- Components support `class`, `style`, appropriate native HTML attributes/events and documented slots/parts. Forward refs where supported and merge consumers' handlers without suppressing necessary internal behavior.
- Ensure disabled, pending, readonly, required, invalid, loading, hover, focus-visible, checked, selected, and expanded states have documented semantics. Do not style state using color alone.
- Favor composition and small focused primitives over an enormous all-purpose configuration object. New abstractions require at least one real reuse case.

### 2.2 Public package distribution

- Publish **ES modules, TypeScript declarations, and distributable CSS**. Use explicit `package.json` `exports`, `files`, and CSS entry points. Avoid shipping raw TSX requiring special consumer transpilation unless deliberately documented and tested.
- Declare `solid-js` as a **peer dependency**, preventing an accidentally bundled second Solid runtime. Keep non-peer dependencies as few as practical. Verify compatible Solid/Kobalte versions before release; do not copy legacy React-era versions.
- Permit per-component imports (subpath exports or otherwise verified tree-shakeable exports). Avoid one barrel that eagerly pulls all dialogs, icons, motion behavior, and CSS into a simple button usage.
- If CSS is imported from component entry points, publish correct `sideEffects` metadata so bundlers do not discard it; alternatively provide a clearly documented required stylesheet import. **The styling delivery mechanism must be proven in a real consuming app.**
- Library code must be safe to import without a browser global at module initialization. Browser-only DOM work belongs in client effects/handlers; treat SSR/hydration compatibility as a documented design concern even though TaoTale MVP is a local SPA.
- Use semantic versioning. Public props, CSS variables, documented slots/state attributes, and exported styles are compatibility-sensitive APIs. Breaking changes require a major version or a documented migration policy.
- Before publishing, validate the generated tarball (`npm pack`) with a separate minimal SolidJS consumer, including types, CSS, import paths, actual behavior, and no duplicate Solid runtime.
- The package's public README/reference should cover installation, component examples, light/dark themes, custom theme overrides, component-level style overrides, accessibility assumptions, and migration notes.

**No package name, licensing model, CSS build tool, icon library, or motion library is approved by this document.** Select them deliberately after a small working library prototype.

## 3. Public styling and customization architecture

The library is **styled by default**, but styling must never be welded to the TaoTale brand. A developer must be able to make a distinct-looking application without editing source files or relying on private class names.

### 3.1 Customization layers (in priority order)

1. **Design tokens:** documented semantic CSS custom properties for surfaces, text, actions, borders, focus, feedback, spacing, radii, typography, and motion. Brand shades should not be hardcoded inside reusable components.
2. **Theme scopes:** apply light/dark/custom themes at an app root or a subtree; nested theme scopes must be possible without globally rewriting `:root`. Use semantic tokens rather than conditionals scattered through JSX. Expose a controlled theme mechanism; persistence belongs to the consumer.
3. **Component variants and sizes:** supported props for common style changes such as `variant`, `size`, `tone` or equivalent; avoid unconstrained permutations and document defaults.
4. **Slots/parts and state hooks:** stable documented selectors or a slots/classes prop for relevant internal parts (trigger, content, label, icon, overlay, etc.). Expose only supported customization surfaces; do not require users to target volatile DOM implementation details.
5. **Direct style overrides:** allow a consumer's class, inline style and CSS overrides to work with documented cascade/layer precedence. Avoid `!important` as a normal styling mechanism.

**Provisional naming guidance:** namespace public CSS custom properties and selectors to avoid collisions (e.g. `--ui-color-surface`, `data-ui-slot="dialog-content"`). The definitive prefix is chosen alongside the npm package name, before v1; example names here are not a committed API.

### 3.2 Themes, animations, and configuration

- Ship attractive, accessible **generic defaults** in the UI package. Ship TaoTale's Quiet Study values as a **consumer-owned theme**, not as defaults hardwired into the library. Optional reusable theme presets may be introduced later.
- A theme change should update CSS variables/attributes without remounting component trees or destroying dialog/input/player state.
- Use CSS transitions/animations for ordinary visual effects. Provide reusable entry/exit **presence handling** for overlays that must stay mounted until an exit animation completes; define focus restoration and cleanup when a transition finishes or is canceled.
- Support configurable placement, collision handling, delays, dismiss behavior and overlay layering where appropriate. Prefer existing Kobalte capabilities before adding a positioning or animation dependency.
- Honor `prefers-reduced-motion` and offer an explicit disable/reduced-motion override for consumers. Animation must not block interaction or be required to understand state.
- Provide consistent overlay z-index and portal behavior across tooltip, popover, select, dialog, and toast. A portaled component must still receive the intended theme variables; document how a nested theme scope works with portals.
- Use logical CSS properties and avoid assuming left-to-right text; preserve native focus indicators, zoom and font scaling.

## 4. Library accessibility and interaction requirements

**Target:** WCAG 2.2 AA for supported components and documented compositions, with conformance evaluated per component and use case. Using Kobalte is a foundation, **not an automatic accessibility guarantee** for every custom wrapper or consumer composition.

- Keyboard activation, arrow-key navigation for composite widgets, Tab behavior, Escape dismiss, and focus return follow the appropriate native/WAI-ARIA patterns.
- Dialogs provide modal semantics, focus containment, initial focus policy, inert background behavior, Escape/close behavior, and reliable focus restoration; nested overlays must not steal focus unexpectedly.
- Tooltips open on hover **and keyboard focus**; support configurable delay, safe dismissal, Escape, accessible descriptions, one-active-tooltip behavior, and no focusable interactive content disguised as a tooltip. Use popovers/dialogs for interactive content.
- Menus, listboxes, selects, comboboxes and tabs implement the correct roles, labels, selection states and typeahead/key handling where applicable. Do not claim a universal keyboard pattern for unlike widgets.
- Every icon-only interactive control requires an accessible name. Labels, error messages, helper descriptions, loading status and live announcements must be properly associated.
- Ensure visible `:focus-visible`, pointer/touch affordances, contrast, readable disabled states, and correct behavior at 200% zoom. Do not depend on color, hover, or animation to communicate essential information.
- Defaults must not erase application labels, `aria-*` props or keyboard handlers. Developers are responsible for meaningful labels, content, page hierarchy, form validation messages and correct feature integration.
- Motion and transitions should never trap focus or create invisible-but-focusable elements; teardown after exit must restore focus predictably.

### 4.1 Component roadmap for the library

This roadmap is **library development scope**, not a promise that every control is required for TaoTale's first MVP build. Build components incrementally based on verified app needs and reusable value.

| Priority | Component types | Behavior guidance |
|---|---|---|
| Foundation | Button, IconButton, TextField, Textarea, Checkbox, Radio, Switch, Label, Badge, Card, Separator, Spinner/Skeleton | Native HTML/Solid-first, clear state and labeling contracts |
| Core overlays | Dialog/AlertDialog, Tooltip, Popover, DropdownMenu | Kobalte-backed where useful; focus, dismissal, placement, presence/exit |
| Selection and navigation | Select, Combobox/Autocomplete, Tabs, Pagination | Correct roles, keyboard selection, focus and controlled state |
| Feedback | Alert, Toast, Progress | Announcements, queue/dismiss handling, accessible feedback |
| Later as needed | Slider, calendar/date picker, data table primitives, drag-and-drop, complex navigation, rich motion | Add only for demonstrated consumer needs; evaluate native options and bundle cost |

**TaoTale-specific components remain in TaoTale:** LibraryItemCard, Favorites toggle backed by an API, TopicManager, VocabularyEntryDetail, Transcript, AnnotatedText, AudioPlayer, GenerateAudio workflow, publication/revision forms, and language-specific controls. They may compose UI-library components and styling tokens, but their domain logic must not ship in the generic npm package.

## 5. Library performance, QA, and release gates

- Measure shipped JavaScript and CSS size by import path; compare a consumer importing **only Button** versus an app importing Dialog/Select/Tooltip. Set numeric budgets after baseline measurements, rather than inventing unverified targets.
- Avoid global event listeners when no relevant overlay is mounted; clean up listeners, observers, timers and portals on disposal. Don't create effects that track irrelevant signals or rerender large component trees for a single state change.
- Favor CSS-based state and animations over frame-by-frame JavaScript work. Ensure animations, tooltip positioning, and focus changes do not interfere with a consuming app's audio playback or text selection.
- Test each public component in light/dark and custom themes, keyboard-only navigation, screen-reader-relevant ARIA state, mouse and touch, reduced motion, 200% zoom, and disabled/loading/error states.
- Use component tests plus browser integration tests (e.g. Vitest and Playwright) with automated accessibility checks and manual keyboard/focus review. Test both internal Kobalte wrappers and the **published package in a consuming app**.
- Test theming and CSS overrides as contractual features: nested themes, portal inheritance, overridden variants, custom token values, and fallback when a token is missing.
- Release only after `npm pack` consumer smoke tests, TypeScript declaration checks, accessibility regression checks, and versioned changes/changelog. Avoid publishing TaoTale-only assets, user data or secrets.

## 6. TaoTale-specific visual identity: Quiet Study

This section is **application theme/design**, not the UI-library's universal default appearance. The library provides the styling and theming API; TaoTale applies these values through that API.

Design direction: calm, minimalist, reading-first; warm ivory/sage in light mode and charcoal/mint in dark mode. Avoid heavy elevation, strong gradients, saturated accents, or pure white/black page backgrounds. Reader text is the primary focus; metadata is quieter. Support light, dark and system preference. TaoTale stores the preference locally and applies the resolved theme before first paint where feasible; switching must not remount the Reader/player.

### 6.1 TaoTale semantic color tokens

| Semantic token | Light | Dark |
|---|---|---|
| Background | `#F5F3EE` | `#151C1B` |
| Paper | `#FFFEFB` | `#202A27` |
| Subtle surface | `#ECEFEA` | `#2A3530` |
| Primary | `#356F63` | `#91C9B2` |
| Primary text contrast | `#FFFFFF` | `#142922` |
| Secondary | `#556F87` | `#A7B6DA` |
| Text primary | `#26332F` | `#E7EFE7` |
| Text secondary | `#60716A` | `#ACBAB0` |
| Divider | `#DADFD7` | `#3A4841` |
| Reader active highlight | `#E8E5B3` | `#48472F` |
| Reader highlight text | `#26332F` | `#FBF0C8` |
| Warm accent | `#B77C4D` | `#D8AC83` |
| Vocabulary Learning chip bg/text | `#FBE9D5` / `#9B430F` | `#2C4D73` / `#E6F0FF` |
| Vocabulary Acquired chip bg/text | `#D7EBE4` / `#20594F` | `#285A4B` / `#D9F2E9` |
| Reader Learning foreground | `#8B4513` | `#E6B27D` |
| Reader Acquired foreground | `#356F63` | `#91C9B2` |

**Usage gates:** ordinary text contrast at least 4.5:1; large text at least 3:1; meaningful non-text control borders/icons at least 3:1. The historical light secondary color on subtle surface did not reach 4.5:1, so use primary text or adjust tokens for that combination. The warm accent is decorative/large text unless an exact combination passes normal-text contrast. Use a lighter accessible derivative for dark-theme primary hover, not a default darkening that reduces contrast. Focus ring: clear, e.g. 2px with 2px offset, tested against adjacent surfaces. Color must not be the only status cue. Check actual nested vocabulary markup against active Reader highlights.

### 6.2 TaoTale topic palettes

Topics are user-global reusable labels, not library color names. The **TaoTale backend** stores and validates exact light/dark text/background combinations; the generic UI package only renders the supplied colors and handles accessible presentation. Do not make the package depend on topic IDs or server presets.

| Preset | Light background / text | Dark background / text |
|---|---|---|
| Sage | `#E1EFE6` / `#285447` | `#2B473D` / `#B7E3CE` |
| Mint | `#D9F1EA` / `#215C50` | `#214740` / `#B1ECDD` |
| Sky | `#E1EEF9` / `#28597C` | `#294455` / `#C1E2F7` |
| Blue | `#E5EEFA` / `#274F84` | `#2A3F59` / `#BFD9FA` |
| Amber | `#F9EED8` / `#80551F` | `#51422B` / `#F2D49B` |
| Peach | `#FCE8DC` / `#814629` | `#51382F` / `#FFD0AE` |
| Lavender | `#EAE7FA` / `#564889` | `#3D3654` / `#D9C9F8` |
| Rose | `#F9E5E6` / `#8B3D4B` | `#54343C` / `#F5BBC6` |
| Slate | `#E9EEEA` / `#465A51` | `#32443B` / `#D5E2DC` |

Seeded defaults and naming/contrast constraints remain TaoTale responsibilities. Creation initially defaults to Sage; historical migration used Slate. The library must not infer colors from a topic's name.

### 6.3 Typography, spacing, and motion

- UI: Inter/system-ui/sans-serif, page heading ~28px/1.3, section ~20px/1.4, body 15–16px/1.6, metadata 13px/1.5, buttons 14px/1.4.
- Transcript: **language-appropriate local fonts**. For Simplified Chinese, retain Noto Serif SC 400 with sensible CJK fallbacks; test a Korean font stack separately. Do not assume one Mandarin font is correct for Korean or future Japanese. Original text must remain selectable and its exact characters/spacing unchanged.
- Reader text ~22px/1.8 on supported desktop/tablet layouts, width around 780px, normal tracking. Font changes must not affect stored UTF-16 offsets or synchronization data.
- Spacing scale: 4/8/12/16/24/32/48 px; base rhythm 8px. Radius ~12px default, card 16px, input/button 10px, chips pill. Low-elevation surfaces, subtle borders and limited shadows.
- Form labels remain visible; use helper/count/error rows. Controls need appropriately large targets (aim ≥44px for common touch targets) and robust focus outlines; support tablets.
- Motion should be subtle and respect reduced-motion preference. The package owns reusable entry/exit behavior; TaoTale chooses when transitions add value.

## 7. TaoTale MVP page composition

**MVP views:** Library, Create/Edit, Reader, Vocabulary and Settings. A Dashboard, Vocabulary Practice, AI Text/Audio Generation Workspace, Language Islands Builder and phone-sized application layouts are **post-MVP**. **Manual Azure Generate Audio within Create/Edit remains in MVP.** Do not add out-of-scope actions to navigation or acceptance criteria because a historical mockup includes them.

### 7.1 Library

- Centered maximum ~1480px; desktop/tablet gutters and responsive grid that remains usable at supported viewport/zoom combinations; 24px grid gaps. Four columns may fit at wide desktop sizes, with fewer on narrower supported tablet widths.
- Mixed saved drafts and published items, scoped to the selected learning language. Card ratio ~1.9:1 cover/placeholder; actual cover upload cap **1 MB (1,000,000 bytes)**, enforced by the application/server, not a generic Card component.
- Card title, description, optional manually assigned **Beginner / Intermediate / Advanced** difficulty (or unspecified), up to two topic chips +N, date, Draft state and published playable duration only when available.
- Persistent **Favorite/Unfavorite** control visible and keyboard accessible on draft and published cards. The icon-only library Button/Tooltip supplies correct interaction/accessibility mechanics; TaoTale supplies the API write, status and cache behavior.
- Search, language choice, topic/difficulty/status/Favorites filters, sort, pagination, Create and item action menus. Omit the old AI generation workspace entry point.
- Distinguish initial loading, stale-on-refresh, empty collection, no filter matches, invalid cover, failed audio, and destructive-operation confirmation.

### 7.2 Create/Edit and publishing

- Wide layout max ~1440px: Content + Audio on left, Cover + Details on right; adapt column layout for tablets and zoom, with clear reading order.
- Reusable editor modes: unsaved new, draft, published metadata and working revision. Preserve explicit Save Draft, Generate Audio, Publish, Save Revision and dirty-navigation decisions.
- Plain source textarea large enough for reading/editing, at least six rows; language-specific font readability, exact source preservation, and counters/validation errors. Do not add a generic rich-text editor.
- Audio status panel, voice/rate selector, generation feedback, READY preview, optional segment review and disabled actions with persistent reason. Library handles the generation workflow; the UI package provides generic buttons, feedback, dialogs, etc.
- Cover dropzone shows selected thumbnail, file limits and error states. App validates ≤1 MB, allowed formats and dimensions; the UI package may provide a generic dropzone later but must not encode TaoTale cover rules.
- TopicManager/TopicAutocomplete are TaoTale feature components using generic accessible dialog/combobox/checkbox/chip primitives.

### 7.3 Reader

- Centered shell ~960px, transcript ~780px, fixed player aligned to shell; reserve sufficient bottom space so the last paragraph is never hidden by the player. Hide the global app header on the exact Reader route where specified.
- Original source rendered with natural inline flow and preserved whitespace/blank lines. Timed runs can highlight while playing; vocabulary annotation underlines coexist with active playback background. Invalid alignment retains readable text.
- Use a native HTML audio element and app-specific high-frequency synchronization outside the general-purpose library. Components such as Buttons, Tooltip, Slider, Dialog and Popover may be reused; **the UI package does not implement speech/text alignment or transcript DOM offsets**.
- Accessible Favorite toggle in the Reader header, Notes dialog, vocabulary actions, lookup popovers, settings and optional browse dialogs; overlay portals must not corrupt text selection or cause audio resets.
- Learning: dotted underline; Acquired: solid underline; independent semantic color and legend, with adequate contrast on active highlight. Popovers show actual saved meaning and language-aware reading, with missing metadata made explicit.
- Active seek targets are keyboard reachable; selection/drag must suppress accidental seek and popup. Overlay close restores focus. Reader shortcuts and audio states are app-level responsibilities.

### 7.4 Vocabulary, Topics and Settings

- Vocabulary: summary stats, server-paginated list (initial 10 rows/page), search/status/topic filters, editable details, source references and bulk operations. Structured list/table with clear keyboard focus, labeled status chips and topic count pills. Optional layout for detail region at supported desktop/tablet widths.
- Topics: accessible autocomplete and manager, named light/dark swatches, default-topic protection, maximum 10 linked topics, confirmation on custom deletion. Never make topic CRUD a UI-library dependency.
- Settings: app theme (light/dark/system), selected language, per-language TTS defaults, playback speed, vocabulary popup controls, and local backup/status guidance. No authentication or billing screen.

## 8. Library-to-application integration requirements

1. **The npm package is a dependency** of TaoTale, pinned according to the application's normal dependency policy; do not copy/fork UI package source into each feature directory.
2. TaoTale defines its own Quiet Study theme using the library's public CSS tokens/theme API. Keep any TaoTale-only design tokens/styles in the application, not the generic package.
3. Shared generic UI controls are imported from the package. `src/components/ui` in TaoTale, if retained, should contain **thin app-specific compositions or aliases**, not duplicate implementations of Button, Dialog, Menu, Tooltip, etc.
4. TaoTale feature code may import public library exports only; do not couple to Kobalte internals or undocumented package paths. For missing capabilities, prefer improving the reusable package if broadly applicable; otherwise keep the feature implementation local.
5. Theme switching, dialog open/close, tooltips, and animations must not reinitialize the Reader audio element, invalidate transcript segment offsets, or cause unrelated Library/Vocabulary refetches.
6. During development, test the library both in its own isolated component/examples environment **and** as an installed dependency in TaoTale. No unpublished package API is assumed stable merely because local linking works.
7. Public library accessibility and theme APIs should be documented and covered by compatibility tests before app features depend on them heavily.

## 9. Acceptance criteria and rollout

### 9.1 UI library readiness for first TaoTale integration

- A separately buildable npm package installs into a clean SolidJS consumer; ESM, types and CSS resolve correctly and there is one Solid runtime.
- Core Button, input, Tooltip, Dialog and Menu/Popover components have accessible keyboard/focus behavior, appropriate states and working public styling hooks. Expand the component set only as TaoTale needs it.
- A consumer can change light/dark theme colors **and** restyle at least one multi-part component without changing package source or using private selectors.
- Transitions and reduced motion work without stray focus, dangling overlays or layout flicker. No global runtime Tailwind requirement.
- The release process can test the packed artifact, compare bundle sizes, and publish with clear versioning.

### 9.2 TaoTale MVP acceptance

- All MVP screens consume the package where generic controls apply. App-specific Reader/audio/vocabulary behavior remains correct and performant.
- Quiet Study tokens and topic palettes are applied accessibly, persisted theme preference behaves correctly, and toolbar/dialog controls work with keyboard and pointer at supported desktop/tablet sizes and zoom.
- Library Favorites, manual three-level difficulty, ≤1 MB covers, audio generation, publication, Reader and Vocabulary are present. AI Workspace, Language Islands Builder, Dashboard, Vocabulary Practice, and phone support remain deferred.
- Switching themes or showing library overlays does not restart audio, corrupt text selection, or remount large unrelated subtrees.

### 9.3 Suggested implementation sequence

1. Initialize the independent SolidJS npm library: package build/exports/types/CSS and a tiny consumer playground.
2. Establish generic CSS tokens, theming/override hooks and one styled native component (`Button`). Validate external customization and bundling.
3. Implement Kobalte-backed Tooltip, Dialog, Popover/Menu with controlled state, focus, presence and reduced-motion tests.
4. Publish an initial version (or install a packaged prerelease) and integrate it into TaoTale's app shell.
5. Add Select/Combobox, forms, feedback and other primitives **as actual TaoTale screens need them**; avoid building a large speculative catalog before the application works.
6. Apply Quiet Study colors/layouts in TaoTale and run end-to-end accessibility/performance checks.

## 10. Decisions to finalize before the first public npm release

- Public package name, repository ownership/license and publication workflow.
- Public token prefix, slot/selector naming, component variant contract and explicit CSS import strategy.
- Internal styling/build implementation and whether a small example/docs site is bundled with the project.
- Exact Kobalte dependency strategy and compatible versions, animation/presence implementation, and which components are part of the v1 stability commitment.
- Scope of component-level theming helpers and whether an `unstyled` mode is supported. **Not required** merely because the library supports overrides.

These choices are left open intentionally; no TaoTale MVP requirement depends on inventing their final names now.

## 11. Reference and provenance

- Historical design source: `08-design-system(1).md` (Quiet Study tokens, topic palettes and screen layout guidance); framework-specific MUI/React-era details are not normative.
- Kobalte: [introduction](https://kobalte.dev/docs/core/overview/introduction), [styling and component states](https://kobalte.dev/docs/core/overview/styling), [tooltip](https://kobalte.dev/docs/core/components/tooltip), [dialog](https://kobalte.dev/docs/core/components/dialog).
- SolidJS: [styling components](https://docs.solidjs.com/guides/styling-your-components), [context](https://docs.solidjs.com/concepts/context).
- npm: [package metadata and peer dependencies](https://docs.npmjs.com/files/package.json/).

**Documentation follow-up:** `03-client.md` and `02-decision-register.md` should explicitly replace the historical “in-app custom UI primitives” wording with “consume the standalone SolidJS UI package” at their next reconciliation. They are **not modified by this document**.
