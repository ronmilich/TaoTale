# 09 — TaoTale UI states and interaction contract

**Status:** Simple MVP interaction specification, revised **25 November 2026**. **Authority:** [01 — Product](01-taotale-overview-and-features.md), [03 — Client](03-taotale-client.md), [06 — Audio](06-taotale-segmentation-and-audio.md), [08 — Iris design](08-taotale-design-system.md).

## 1. Scope and interaction philosophy

TaoTale is one locally hosted SolidJS application. MVP screens: **Library**, **Create/Edit**, **Reader**, **Vocabulary**, **Settings / Local Data**. `/` enters `/library`. No account, AI workspace, Dashboard, Practice, Islands or phone-specific navigation. Manual Azure **Generate Audio** is in the Editor.

**Simple UI implementation:** ordinary app-owned CSS Modules and native controls. No external UI package, Kobalte/headless controls, custom menu/tooltip/combobox engines, focus-trap code, overlay portal system, or specialized accessibility JS. Use a native `select`, regular forms, inline panels, `window.confirm` for quick destructive confirmation, or native `<dialog>` when a polished modal materially improves a screen. Keep labels, messages and obvious button states. Reader/audio JavaScript remains domain-specific and essential.

All screens follow the Iris palette in [08](08-taotale-design-system.md): clear titles, spacious white/soft-lavender surfaces, violet main actions, subtle CSS hover/pressed transitions, and accessible-to-read typography without a formal accessibility certification project.

## 2. Common states

| State | Required behavior |
|---|---|
| Initial load | Skeleton or compact loading text; never show a false empty list |
| Ready | Render server-confirmed content and actions whose prerequisites are satisfied |
| Actual empty | Explain that no saved items/words exist; offer a clear create action |
| Filtered empty | Explain no matches; offer Clear filters |
| Refresh | Keep previous confirmed content while fetching if safe; show small update indicator |
| Read error | Keep valid previous data if available; provide Retry |
| Save/generation pending | Disable conflicting repeat actions; retain all editor input |
| Validation error | Show message beside the field; keep entered data |
| Save confirmed | Only show Saved/Published/Favorited after backend confirmation |
| Version conflict | Explain item changed elsewhere and offer reload/review; never silently overwrite |
| Uncertain paid generation | Reuse original request body/idempotency key for recovery; never auto-send a second Azure request |
| Backend offline | Saved data already in memory can stay visible; writes fail clearly, no fake offline queue |
| Missing route/item | Simple not-found page linking to Library; drafts point to Continue Editing |

Errors are human-readable and sanitized: no secrets, full private text, stack traces, SQL or absolute paths. Status has words, not only color.

## 3. Shell and learning language

Active language controls Library, Vocabulary and defaults when creating a new item: **Simplified Mandarin `zh-CN`** and **Korean `ko-KR`**. A published Reader uses the item's stored language. Theme options: **Light / Dark / System**, stored in local preferences; apply theme by CSS `data-theme` without remounting audio or resetting forms.

Use desktop and tablet layouts, normal links/buttons/forms and touch-friendly spacing. No phone-specific acceptance gate. Language disabled/unavailable states never pretend content is usable; offer another enabled language.

## 4. Library

| Situation | Behavior |
|---|---|
| List | Language-scoped drafts and published items; **8 cards/page** initially |
| Card | Cover/fallback, title or Untitled draft, short description, difficulty, up to 2 topics +N, Favorite, draft/published status; duration only for validated audio |
| Filters | Literal title search, Topics (OR), manual difficulty or Unspecified, status, Favorites-only; groups combined with AND |
| Sort | Newest (default), oldest, title A–Z/Z–A; stable page and tie ordering |
| Favorite | Both drafts/published; explicit `true`/`false` API write; visible pending/error state; no audio/version/updatedAt change |
| Unfavorite in Favorites-only | Remove after confirmed success and clamp page |
| Delete | Native confirmation or simple dialog describing that linked vocabulary **entries remain**; show error if deletion fails |
| Download | Only for verified published audio; no fictitious duration or invalid playable link |

A Favorite star is a visible button rather than a hidden hover-only control. Cover load failure does not hide the collection. Use gentle hover and selected styles from 08.

## 5. Create/Edit and audio

Modes: **new unsaved**, **saved draft**, **published metadata**, **working revision**. No autosave. All use a straightforward page form and visible Save/Generate/Publish actions.

| State | Behavior |
|---|---|
| New, incomplete | Explicit Save Draft accepts incomplete title/text; choose enabled item language |
| Invalid cover | Reject new/replacement image above **1,000,000 bytes**, wrong format or unsafe dimensions; retain previous cover and form |
| Temporary preview | Unsaved temporary generation is not a draft; Ready/Failed/Expired/Claimed clearly indicated |
| Draft | Restore saved fields and generation status; source/voice/rate changes must be saved before saved-target generation |
| Generation | `NOT_GENERATED`, persisted `PENDING`/`PROCESSING`/`READY`/`FAILED`, computed `OUTDATED`; show progress as text, not fabricated % |
| Ready | Preview/download only if matching validated source/language/voice/rate/strategy and actual measured duration |
| Cannot publish | State the missing requirement: title/source, unsupported voice, unsaved synthesis changes, unmatched audio, invalid alignment or pending write |
| Published metadata | Title/description/notes/difficulty/topics/cover/Favorite change **without** re-generating audio |
| Working revision | Original published source/audio stays playable until a validated revision is atomically published |
| Conflict or unknown outcome | Preserve input; never auto-merge or re-bill |

`native` is not a supported difficulty. Saving a new item with another learning language is distinct from changing published content language.

## 6. Reader

- Render exact original text, whitespace and paragraphs. Use **verified persisted UTF-16 timed ranges** for the iris phrase highlight. No client resegmentation of historical assets. With corrupt alignment/audio, readable text and vocabulary selection still work; disable inaccurate timed actions.
- Native audio: play/pause, ±5s, timeline, rate, elapsed/verified total, download, buffer/error. No autoplay when reopening. Resume keyed by **item ID + audio asset ID**; a new asset resets old resume.
- Clicking a timed phrase can seek. If the user is dragging/selecting text, suppress accidental seek. Keep the transcript selectable; do not add custom composite keyboard navigation across every segment.
- **Add to Vocabulary** uses native text selection and a visible action/panel or native dialog; Reader language governs lookup. Existing word gets linked without overwriting metadata; new word requires explicit save. Cancel creates nothing. Manual status Learning/Acquired is reversible.
- The Reader measures the selection against the original UTF-16 text and trims outer whitespace. Mandarin selections reject internal line breaks and `，。！？；`. Korean selections preserve the punctuation and spacing of the selected phrase.
- Article vocabulary lists manually linked words. The Both, Learning and Acquired display choice is saved in `taotale.preferences.reader`. Confirmed status changes refresh open vocabulary views.
- Mandarin articles also show automatically linked words and offer **Sync vocabulary** to check the current published text. Learning highlights use dotted underlines and Acquired highlights use solid underlines. The display choice applies to both the panel and the underlines. Updates to underline ranges wait for an active transcript selection to clear.
- Automatic annotations only when a **verified language-specific matching strategy** exists. Do not apply Chinese substring behavior to Korean by assumption. A word panel should not modify transcript DOM offsets or reset playback.
- Favorite reflects the same persisted item flag as Library. Notes use plain text Save/Cancel and do not change audio. Theme/notes/panels must not remount the audio element.

## 7. Vocabulary and Topics

Vocabulary list/search/stats are scoped to active language; **10 rows/page** initially. Search word/reading/meaning, filter by Learning/Acquired/Topics and offer simple create/edit/delete/bulk actions. Manual source links require the same language; duplicate words provide a link-existing option. Errors retain user input.

Topics are **one user-global catalog**. Default Topics are protected; custom Topics can be renamed/recolored/deleted with a clear confirmation. Deletion removes associations, not the Library items or Vocabulary entries. Maximum **10 Topic associations** per item or entry. Use a simple checkbox list and edit form instead of custom autocomplete or complex menus.

## 8. Settings and local data

Appearance (light/dark/system), active language, per-language Azure voice/rate defaults, Reader playback speed and word-popup visibility are small validated browser preferences. Show whether local backend/DB/media and Azure generation are available. State clearly that **manual paired PostgreSQL+media backup/restore** is required; do not show a nonfunctional one-click Backup action. Previously saved reading and audio work without internet when local services are running; Azure generation needs an external connection.

## 9. Leaving, confirming, and preventing loss

When an editor or Notes form is dirty, use `@solidjs/router` leave protection and a **simple Save & Leave / Discard & Leave / Stay** choice. A minimal native confirm is acceptable for the first version; a compact app-owned `<dialog>` is an optional visual upgrade. Navigate only after successful save or explicit discard. For tab close or reload, use the **native `beforeunload` warning**. Do not build custom focus restoration, Escape dispatcher or overlay stack machinery. Confirm destructive item deletion and working-revision discard.

## 10. Acceptance examples and wording

Canonical labels: Library, Vocabulary, Settings, Create New Item, Save Draft, Generate Audio, Publish to Library, Edit Text & Voice, Save Revision, Publish Revision, Discard Revision, Favorite, Remove from Favorites, Favorites only, Beginner, Intermediate, Advanced, Unspecified, Open in Reader, Continue Editing, Back to Library, Download Audio, Add to Vocabulary, Already in Vocabulary, Learning, Acquired, Retry, Outdated audio, Audio unavailable, No items match your filters.

**Critical acceptance:** (1) Favorite survives restart without version/audio change; (2) Chinese/Korean contexts stay separate; (3) invalid timing preserves readable transcript; (4) leaving never falsely promises a save; (5) invalid 1,000,001-byte replacement cover preserves old cover; (6) failed revision leaves published recording safe; (7) theme/panels do not remount Reader audio; (8) Iris appearance is cohesive light/dark on desktop/tablet; (9) no external UI package or complicated custom component behavior; (10) no post-MVP views in the shipped shell.

Related specifications include the [API](10-taotale-api.md), [quality gates](12-taotale-quality-and-acceptance.md), [rebuild plan](13-taotale-rebuild-decisions.md) and [TaoTale roadmap](taotale-roadmap.md).
