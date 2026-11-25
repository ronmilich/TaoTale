# 08 — TaoTale Iris visual design system

**Status:** Application-owned MVP visual specification, revised **25 November 2026**. **Authority:** [01 — Product](01-taotale-overview-and-features.md), [03 — Client](03-taotale-client.md), [09 — UI states](09-taotale-ui-states.md).

## 1. Vision: Iris Reading Studio

**TaoTale should look premium and inviting without a complex UI implementation.** The visual identity takes inspiration from an iris flower: deep violet, luminous lavender, soft periwinkle, a touch of orchid pink and quiet near-white paper. The reader is the hero. The surrounding app resembles a thoughtfully designed personal library rather than a generic admin dashboard.

**Experience keywords:** literary, calm, refined, modern, warm, slightly playful, spacious. Use one or two visual flourishes per screen, not a gradient on every card. Keep detailed content readable and practical.

**Ownership and exclusions:** **All styling and generic components are written directly inside TaoTale's SolidJS frontend.** There is **no external UI library**, UI npm package, CSS theme SDK, custom accessibility framework, Kobalte/Radix primitives, Tailwind, CSS-in-JS, floating UI engine, animation package or reusable public component contract. The goal is a few well-styled app components plus excellent page composition.

Use **plain CSS + CSS Modules + CSS custom properties**. JavaScript is for actual app behavior: routes, form state, API calls, audio, selection and Reader synchronization. **No specialized JavaScript to recreate buttons, selects, menus, tooltips, focus traps or animations.** Native browser controls and normal semantic HTML are sufficient.

## 2. Iris palette

### 2.1 Brand and neutral colors

**Light theme** is the primary design target. Dark is a coordinated alternative, not a color inversion.

| Token | Light | Dark | Purpose |
|---|---|---|---|
| `--color-bg` | `#F8F7FC` | `#151322` | App canvas |
| `--color-surface` | `#FFFFFF` | `#211E33` | Cards, forms, Reader paper |
| `--color-surface-soft` | `#F1EEFA` | `#2B2642` | Soft panels, filters, inputs |
| `--color-surface-raised` | `#FFFCFF` | `#322B4A` | Selected/raised sections |
| `--color-text` | `#29243D` | `#F3F0FA` | Primary headings and body |
| `--color-text-muted` | `#69647D` | `#B9B2CC` | Labels, secondary metadata |
| `--color-border` | `#E3DFF0` | `#413B58` | Dividers and outlines |
| `--color-primary` | `#6C4BCB` | `#BCA7FF` | Main call to action / links |
| `--color-primary-hover` | `#5A3DB1` | `#D1C2FF` | Hover state |
| `--color-primary-pressed` | `#49318D` | `#A18CEB` | Pressed state |
| `--color-on-primary` | `#FFFFFF` | `#211633` | Button content |
| `--color-primary-soft` | `#EDE7FC` | `#383052` | Selected filters / gentle emphasis |
| `--color-periwinkle` | `#8186DC` | `#A9B0FF` | Secondary brand accent |
| `--color-orchid` | `#BA79B7` | `#DFA1DA` | Warm floral accent |
| `--color-orchid-soft` | `#F8EAF6` | `#4A304C` | Decorative cards/chips |
| `--color-highlight` | `#E9E3FF` | `#554675` | Reader active phrase wash |
| `--color-highlight-text` | `#312451` | `#FFF9FF` | Highlighted text |
| `--color-danger` | `#B34B64` | `#F0A1B4` | Destructive/error emphasis |
| `--color-success` | `#28775E` | `#93D5B7` | Confirmed / Acquired |
| `--color-warning` | `#936123` | `#F0C98A` | Nonterminal warnings |

**Palette rules:** Primary violet is reserved for selection, current navigation and primary buttons. Orchid and periwinkle provide small supporting accents; avoid using all three in the same control. Make space for ivory/white paper so the palette does not become visually noisy. Avoid pure black, pure white as the page backdrop, saturated neon purple, large high-contrast gradients and multiple heavy box shadows.

### 2.2 Brand flourishes

- One gentle **iris gradient** may appear on the Library hero panel or a cover placeholder: `linear-gradient(125deg, #6C4BCB 0%, #8D83DC 60%, #C69BCF 100%)`. Keep text clearly readable on any gradient.
- Favor illustration by **abstract petal geometry**: overlapping translucent oval shapes, small halo rings and very subtle paper texture. CSS pseudo-elements or one static SVG are sufficient. Do not ship a canvas engine, animation library or complex SVG JS.
- Reading material should feel like paper. Use quiet borders, generous breathing room, subtle tint, rounded corners and beautiful type hierarchy rather than extensive glassmorphism.
- Cover placeholders can use one of four muted preselected variants (violet, periwinkle, orchid, neutral) selected deterministically. Uploaded images retain their original color.

### 2.3 Topic chips and vocabulary

Topics remain user-global database records. Preserve the existing four persisted color values (light/dark text and background) and server-side validation from [05](05-taotale-database.md); CSS only displays those values. Offer a few attractive presets, such as:

| Preset | Light background / text | Dark background / text |
|---|---|---|
| Iris | `#ECE7FA` / `#554095` | `#3E345C` / `#E4D8FF` |
| Periwinkle | `#E7EAFE` / `#444D9E` | `#31385C` / `#DCE0FF` |
| Orchid | `#F6E6F3` / `#813E7B` | `#513450` / `#F4C8EC` |
| Sage | `#E1F0E9` / `#286449` | `#294C3F` / `#B9E9D2` |
| Amber | `#FFF0DA` / `#835921` | `#54422B` / `#F8DEB1` |
| Slate | `#EDECF3` / `#59566B` | `#3A3749` / `#DEDAED` |

Vocabulary status is communicated with **text plus styling**. Learning uses a dotted iris underline and soft lavender status; Acquired uses a solid sage/green underline and green-tinted status. On active timed spans, keep those underlines readable over the phrase highlight. Do not invent vocabulary matching; display only verified/manual associations.

## 3. Typography and spacing

| Element | Recommended treatment |
|---|---|
| Interface font | `Inter`, `ui-sans-serif`, `system-ui`, `Segoe UI`, sans-serif; system fonts are perfectly acceptable if no font is bundled |
| Display / Library title | 32–38px, weight 700, line-height 1.14, subtle negative tracking |
| Page title | 26–30px, 650–700, 1.2 |
| Section title | 18–21px, 600–650, 1.35 |
| Normal body | 15–16px, 400, 1.55–1.65 |
| Card title | 17–19px, 600–650, 1.4; clamp only metadata previews, never Reader source |
| Metadata | 12–13px, medium weight; contrast through text color, not tiny opacity |
| Reader source | 21–24px, line-height 1.85–2.0; large enough for Han and Hangul |

**Reading fonts:** use an appropriate Simplified Chinese serif stack (e.g. Noto Serif SC when locally available, then Songti SC / SimSun / serif) and a separately tested Korean font stack (e.g. Noto Serif KR where available, then Batang / serif). Avoid requiring remote font CDNs or shipping heavy CJK bundles merely to meet a style goal. Exact original characters, whitespace, selection and UTF-16 offsets remain unchanged.

**Rhythm:** spacing steps **4 / 8 / 12 / 16 / 20 / 24 / 32 / 40 / 48 / 64 px**. Use 24px card gaps at comfortable desktop sizes; 16px on tablets. Forms have 8px label-to-control gap and 20–24px between field groups. Avoid long dense forms without section breaks.

**Shape:** buttons/inputs `10–12px` radius; cards `18px`; hero panels `24px`; chips `999px`; Reader paper `20px`. Borders generally `1px solid var(--color-border)`. A small consistent shadow is enough: `0 8px 30px rgba(53, 35, 91, .06)` in light mode and restrained dark-mode shadows.

## 4. CSS implementation contract

Import exactly one global token stylesheet before feature styles. Component appearance comes from CSS Modules, not dynamic runtime style objects. Example starter tokens:

```css
/* shared/styles/tokens.css — representative implementation */
:root,
[data-theme="light"] {
  color-scheme: light;
  --color-bg: #F8F7FC;
  --color-surface: #FFFFFF;
  --color-surface-soft: #F1EEFA;
  --color-surface-raised: #FFFCFF;
  --color-text: #29243D;
  --color-text-muted: #69647D;
  --color-border: #E3DFF0;
  --color-primary: #6C4BCB;
  --color-primary-hover: #5A3DB1;
  --color-primary-pressed: #49318D;
  --color-on-primary: #FFFFFF;
  --color-primary-soft: #EDE7FC;
  --color-periwinkle: #8186DC;
  --color-orchid: #BA79B7;
  --color-orchid-soft: #F8EAF6;
  --color-highlight: #E9E3FF;
  --color-highlight-text: #312451;
  --color-danger: #B34B64;
  --color-success: #28775E;
  --color-warning: #936123;
  --radius-control: 11px;
  --radius-card: 18px;
  --shadow-card: 0 8px 30px rgba(53, 35, 91, .06);
}
[data-theme="dark"] {
  color-scheme: dark;
  --color-bg: #151322;
  --color-surface: #211E33;
  --color-surface-soft: #2B2642;
  --color-surface-raised: #322B4A;
  --color-text: #F3F0FA;
  --color-text-muted: #B9B2CC;
  --color-border: #413B58;
  --color-primary: #BCA7FF;
  --color-primary-hover: #D1C2FF;
  --color-primary-pressed: #A18CEB;
  --color-on-primary: #211633;
  --color-primary-soft: #383052;
  --color-periwinkle: #A9B0FF;
  --color-orchid: #DFA1DA;
  --color-orchid-soft: #4A304C;
  --color-highlight: #554675;
  --color-highlight-text: #FFF9FF;
  --color-danger: #F0A1B4;
  --color-success: #93D5B7;
  --color-warning: #F0C98A;
  --shadow-card: 0 10px 30px rgba(0, 0, 0, .12);
}
```

```css
/* Example app-local Button.module.css */
.button {
  min-height: 42px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 10px 16px;
  border-radius: var(--radius-control);
  border: 1px solid transparent;
  font: inherit;
  font-weight: 600;
  cursor: pointer;
  transition: background-color 150ms ease, box-shadow 150ms ease,
              transform 150ms ease, border-color 150ms ease;
}
.button:active:not(:disabled) { transform: translateY(1px); }
.button:disabled { opacity: .55; cursor: not-allowed; }
.primary { background: var(--color-primary); color: var(--color-on-primary); }
.primary:hover:not(:disabled) { background: var(--color-primary-hover); }
.secondary { background: var(--color-surface); color: var(--color-text);
             border-color: var(--color-border); }
.secondary:hover:not(:disabled) { background: var(--color-surface-soft); }
.ghost { background: transparent; color: var(--color-primary); }
.ghost:hover:not(:disabled) { background: var(--color-primary-soft); }
.button:focus-visible { outline: 2px solid var(--color-periwinkle); outline-offset: 2px; }
@media (prefers-reduced-motion: reduce) { .button { transition: none; } }
```

Use only as much CSS as the product needs. Basic focus outlines and reduced-motion media queries are **CSS**, not a specialized JavaScript accessibility/motion system. Theme changes swap `data-theme`, preserving the existing DOM and audio.

## 5. Small in-app component catalog

Build these **only when a screen needs them**:

| Element | Implementation | Visual goal |
|---|---|---|
| `Button` | Native `button` with app-local variants | Confident violet primary, light secondary, calm ghost |
| `TextField` / `TextArea` | Native `input`/`textarea` + label/help/error markup | Clear white/soft input surfaces, iris focus treatment |
| `SelectField` | Native `select` | Consistent height/radius with input; no custom dropdown engine |
| `Card` | Styled container | Paper surface, subtle border/shadow, 18px radius |
| `Badge` / `TopicChip` | Styled text/span | Small rounded colored metadata |
| `FavoriteButton` | Domain-specific native button | Always-visible recognizable star and pending/error state |
| `Modal` (optional) | Tiny wrapper around native `<dialog>` | Simple confirmations/forms; no custom focus/portal logic |
| `LoadingSkeleton` | CSS placeholder shape | Quiet loading, no pulsing library needed |
| `EmptyState` | Plain content + illustration or static SVG | Helpful message and one clear action |
| `AudioPlayer` | Native `<audio>` + TaoTale buttons/range | Fixed, elegant reading controls; actual timing belongs to Reader logic |

Do not prebuild a full component inventory (combobox, nested menu, calendar, drag/drop, toast scheduler, accessibility primitives, theme SDK). A `TopicManager` and Reader transcript are **feature components** and remain within TaoTale.

## 6. Page compositions

### 6.1 Shell and navigation

A slim top navigation bar with TaoTale wordmark or simple iris icon, **Library**, **Vocabulary**, **Settings** and a restrained language switcher. Active navigation has a soft violet tint and a clear indicator. Use `position: sticky` only if it helps; avoid multi-level app chrome. Content width generally **1280–1440px**, side gutters 24–40px desktop and 16–24px tablet. Background is `--color-bg`.

### 6.2 Library — signature screen

- A welcoming heading such as **Your reading library** and a small, gentle violet/orchid decorative flourish. One **New article** primary action visibly stands out.
- Above content: language selector, title search and a concise row of native filter controls (difficulty, Topics, status, Favorites) plus Sort. Avoid elaborate popovers and nested menus.
- Responsive card grid: approximately **4 columns** on wide desktop, **3 or 2** as space narrows, with CSS `grid-template-columns: repeat(auto-fit, minmax(245px, 1fr))`; no phone-specific layout obligation.
- Card: 1.75–1.9:1 illustrated/image cover, title, two-line description, difficulty/topics, date and published duration. Favorite star is **always visible**, not revealed only on hover. Hover adds a subtle elevation, not a dramatic jump.
- Drafts receive a quiet outlined **Draft** badge and Continue Editing action; published items receive Open Reader. Show meaningful true-empty and filter-empty states with attractive small illustrations or static SVG.

### 6.3 Create/Edit — focused writing studio

- Two-column wide desktop composition: **left** source textarea and Audio panel; **right** Cover, Topics, difficulty, description/notes. At tablet sizes stack into one comfortable column.
- Source editor is the strongest element: roomy paper panel, 16–18px editing text, generous padding, clear border, optional quiet character count. Preserve exact source text.
- Audio generation card groups language-compatible voice, rate, **Generate Audio**, state/progress, measured duration and Preview. Use plain explanatory text for Outdated/Failed/Pending; no complex wizard or progress animation.
- Keep a small clear action area for **Save Draft** and **Publish**. Publish is enabled only for a validated matching READY candidate. Working revision actions should not risk the existing published recording.
- Cover selection uses native file input and a styled preview. Keep the existing cover after invalid replacement. Topics use simple multiselect checkboxes or a straightforward manager panel; no autocomplete engine is required.

### 6.4 Reader — quiet reading room

- Hide distracting global navigation if needed. Reader shell ~960px; prose column approximately **720–780px**, centered in a surface with generous margins. Paper should dominate.
- Title at 30–34px; metadata quieter. Transcript 21–24px with generous line-height, original paragraphs and selectable text. Color highlights should feel like soft translucent ink, not a flashing karaoke banner.
- Native audio control area or a slim custom presentation around `HTMLAudioElement`, docked/visible without covering final prose. Play/pause, seek, ±5s, rate, time and download. Keep audio mounted when changing theme or opening panels.
- Selection-based **Add to Vocabulary** can use a persistent action near transcript controls or one simple contextual inline panel; avoid precision popup positioning infrastructure. Notes and Favorite are simple actions.
- When audio or alignment is invalid, preserve beautiful text-only reading. Never render invented timing.

### 6.5 Vocabulary and Settings

- Vocabulary: a comfortable summary row of Learning / Acquired / Total; compact search and filters, clean rows with word, reading, meaning, status and topics. Table/card treatment depends on width; avoid an enterprise-heavy data grid.
- Word edit is a clear regular form or a native dialog; source-item references remain visible. Status changes are explicit and reversible.
- Settings: grouped paper panels for Language, Appearance, Audio and Local Data. Use native light/dark/system select or three plainly styled radio options. The backup section **documents** the manual database+media procedure; no nonfunctional Backup button.

## 7. Layout, states and simple quality bar

- **Light and dark** both feel intentional; theme switching does not remount audio or clear forms.
- At desktop (~1280/1440px) and tablet (~768–1024px), there is no disruptive horizontal scrolling, clipped labels or hidden primary action. Phone-specific design and testing remain post-MVP.
- Search/filters, cards, buttons, forms, status and pagination are consistent across pages. Use icons sparingly and clearly; a labeled action is better than an ambiguous icon-only menu.
- Include loading, empty, filtered-empty, pending, success and error states in the same design language. Explain errors near affected actions, not as transient colored decorations alone.
- Use readable contrast and normal native focus states as **basic usability**, without formal WCAG certification, special ARIA framework or custom keyboard event choreography.
- Microinteractions stay in CSS (150–200ms hover/background changes), and must never be required for functionality.
- Protect the most important features: Reader alignment, source preservation, versioned publishing, user data and audio resume. Visual polish must not introduce correctness risk.

## 8. Implementation order and approval checklist

1. Define `tokens.css` and global typography/light-dark theme on the app root.
2. Implement the shell and **one polished Library card** with real responsive CSS.
3. Build the few local native control wrappers used by that page: Button, TextField, Select, Badge, Card.
4. Establish Editor layouts and generate/status panels, then Reader typography/highlight/player.
5. Style Vocabulary and Settings, including empty/error/pending states.
6. Manually review screenshots of all MVP screens in light and dark at desktop and tablet widths; fix hierarchy, spacing and consistency. Test that Reader playback survives theme and panel changes.

Review the TaoTale screens for reading comfort, consistent visuals and clear native interactions. The [TaoTale roadmap](taotale-roadmap.md) tracks the implementation and visual review.

## 9. Source and scope notes

This replaces the earlier **Quiet Study sage theme** and all independent npm component-library requirements. Historical Mandarin Reader visual references may inform layout, but their framework, accessibility infrastructure and UI-library assumptions are **not normative**. The application remains SolidJS, and the product/data/audio specification in [01](01-taotale-overview-and-features.md) and [06](06-taotale-segmentation-and-audio.md) is unchanged by the palette decision.
