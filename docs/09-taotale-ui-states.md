# 09 — TaoTale UI states and interaction contract

**Status:** Target MVP behavior, revised 23 November 2026. **Authority:** [01 — Overview](01-taotale-overview-and-features.md), [03 — Client](03-taotale-client.md), [06 — Segmentation and audio](06-taotale-segmentation-and-audio.md), and [08 — Design system](08-taotale-design-system.md). This document defines application states and interactions, **not** the reusable npm package's component styling or implementation.

## 1. Scope and UI ownership

TaoTale is a **single-user, locally hosted** SolidJS application. MVP views: **Library**, **Create/Edit**, **Reader**, **Vocabulary**, and **Settings / Local Data**. `/` enters `/library`; no account/onboarding screen. **Dashboard, Vocabulary Practice, AI text/audio workspace, Language Islands Builder and phone-specific UX are post-MVP.** Manual **Generate Audio** in Create/Edit remains MVP.

- The **independent, SolidJS-only npm UI library** owns generic Button/Dialog/Tooltip/Menu/Popover behavior, keyboard and focus mechanics, transitions, theme customization and accessible primitives. **TaoTale does not import Kobalte directly or use Tailwind.**
- TaoTale owns domain state, validation, navigation, generation progress, local persistence, transcript selection, audio synchronization, vocabulary interaction and API calls. Application-specific composites (Reader, LibraryCard, TopicManager, vocabulary dialogs) stay in TaoTale.
- Use semantic labels, associated validation messages, focus management and live status updates; do not convey status by color alone. Honor reduced-motion. Do not let UI-library overlays remount or interrupt the Reader audio element.

## 2. Global state and error contract

| State | User-visible behavior |
|---|---|
| Initial loading | Skeleton/short progress announcement; do not show a false empty state. |
| Ready | Display server-confirmed data and enabled actions only when their prerequisites hold. |
| True empty | Explain there is no saved content; offer the relevant creation action. |
| Filter empty | Explain no results match current filters; offer Clear filters without implying no data exists. |
| Background loading | Keep valid previous results visible, indicate refresh without flashing content. |
| Recoverable read failure | Preserve previously confirmed data where safe and show **Retry**. Never invent data or timing. |
| Mutation pending | Disable duplicate/conflicting actions, preserve editable input; distinguish network pending from confirmed success. |
| Validation failure | Inline field/code/message, focus first invalid control, keep all user input. |
| Version conflict | Explain the record changed, allow review/reload/reapply; **never** silently overwrite. |
| Uncertain write/generation outcome | Retain immutable request body and idempotency key, recover status with the **same** intent; do not automatically resubmit paid work. |
| Offline / backend unavailable | Existing loaded content remains readable where possible; reads can retry on reconnection; writes fail explicitly, never queue silently. |
| Unsupported route or missing item | Accessible not-found recovery to Library. A draft Reader URL directs to Continue Editing. |

All errors use sanitized, actionable messages and may show a request ID; **never** surface SQL, server paths, Azure credentials, private source text or stack traces. Prefer stable labels over ambiguous generic toast-only feedback.

## 3. Shell and language selection

- Application opens to the **selected learning language**'s Library. MVP languages: **Simplified Mandarin (`zh-CN`)** and **Korean (`ko-KR`)**. A language selector changes Library, Vocabulary, Create defaults and compatible voices; it does **not** change an existing Reader item's stored language.
- Show language-loading/empty/unavailable states honestly. If a language is disabled or missing from the server catalog, offer an available language without rewriting persisted items.
- Route transitions preserve the relevant URL filters and can retain the previous view during lazy loading. Theme may be light/dark/system and must apply without unmounting Reader/audio or resetting forms.
- Desktop and **tablet-sized** layouts, keyboard, zoom and pointer/touch use are in MVP; phone layouts are not acceptance requirements. Hover-only actions are insufficient for tablets.

## 4. Language Library and Favorites

| Situation | Required behavior |
|---|---|
| Initial / refreshed list | Load one paginated, language-scoped mixed list of drafts and published items; maintain stable previous cards during refresh. |
| Card presentation | Cover/fallback, title or **Untitled draft**, description, manual difficulty, up to two topics +N, draft/published indication, favorite button. Duration/play only for published matching READY audio. |
| Filters | Search **literal title substring**, selected Topics (OR), difficulty (`beginner`/`intermediate`/`advanced`/Unspecified), status and **Favorites only**; combine groups with AND. |
| Sort/pagination | Newest default, oldest, title A–Z/Z–A; stable ID tie-breaker; reset/clamp page when result count changes. |
| Favorite toggle | Both drafts and published items; use explicit **set true/false**, not a blind inverse toggle. Show pending, confirmed state and retry on failure. It must not change audio, item content version or newest ordering. |
| Unfavorite in Favorites-only filter | After confirmed success, remove from filtered results; maintain a sensible page. Failure keeps last confirmed state. |
| Delete | Require confirmation with consequences. Deleting an item removes its own recording/joins when released but **keeps vocabulary entries**; preserve failed deletion state. |
| Download | Only available for validated published audio. Expose unavailable/missing-media errors, not a bogus duration or disabled-but-clickable link. |

A card's Favorite control must be separately keyboard/pointer accessible and must not accidentally navigate to Reader. Loading or failure of one cover must not blank the collection.

## 5. Create/Edit, drafts and audio generation

**Four editor modes:** unsaved new item, saved draft, published metadata edit, saved working revision. All share deliberate Save/Generate/Publish rules; **no autosave** and no persistence merely from choosing an image or playing a preview.

| Mode or state | Behavior |
|---|---|
| New, clean / dirty | Title and source may be incomplete locally; Save Draft is explicit. New item chooses one enabled language, which governs voices and segmenter. |
| Upload pending / invalid / failed | Validate encoded cover **≤1,000,000 bytes** plus actual JPEG/PNG/WebP format and dimensions; keep fields/old cover on failure, offer remove/retry. Existing oversized legacy cover remains viewable. |
| Temporary preview | Generate may create an expiring, unsaved preview; it is **not** a draft. Distinguish Ready/Failed/Expired/Claimed. Saving a draft may claim a matching preview. |
| Saved draft | Restores all saved fields and latest generation. If source/voice/rate changes, saved-target generation needs the current synthesis inputs saved first. Draft cannot open in Reader. |
| Generation | `NOT_GENERATED` (computed), `PENDING`, `PROCESSING`, `READY`, `FAILED`, `OUTDATED` (computed). Interrupted jobs are `FAILED` with a specific interruption code. Poll the same generation ID; prevent concurrent duplicate submission. |
| Ready candidate | Preview/download only when validated audio **matches** exact text, item language, voice, rate, segmentation strategy/version and fingerprint format. Display measured duration, not an estimate. |
| Publish disabled | Persistent explanation: missing title/text, unsupported voice, unsaved synthesis changes, no matching READY asset, invalid alignment or pending mutation. |
| Published metadata edit | Title, description, notes, difficulty, topics, cover and Favorite can change **without** audio regeneration. Text/voice/rate use a working revision. |
| Working revision | Has its own saved candidate/version/job. Original published Reader remains playable through generation, failure, cancellation or discard. Atomic publish replaces only the validated source/audio pair. |
| Conflict / unknown result | Preserve unsaved values; don't auto-merge. Unknown paid-result recovery uses original key/body. |

Changing manually assigned difficulty does not modify generation identity. `native` is **not** a valid new difficulty value. Avoid accidental language changes to published material; creating a new item in another language is preferred.

## 6. Reader

- **Text:** render exact original characters, whitespace and blank lines. Valid UTF-16 source spans support timed highlighted segments; the Reader never resegments historical audio on the client. On missing/corrupt audio or invalid alignment, show **readable plain text** and disable misleading timed seeking, while keeping vocabulary operations available.
- **Playback:** native HTML audio; play/pause, timeline, ±5 seconds, playback-rate control, elapsed/verified total, download, loading/buffering/error. No autoplay on restore. Resume position keyed by **item + immutable audio asset**; replacement asset resets previous resume. End clears active highlight.
- **Seeking/selection:** segment click/keyboard activation seeks only a valid timed span and preserves paused/playing state. Text dragging or non-collapsed native selection suppresses seek and hover popups. Keep keyboard focus visible and text selectable, including Hangul, Han and mixed scripts.
- **Vocabulary:** explicit selection → Add to Vocabulary; lookup uses **Reader item language**. Existing word links to item without overwriting spelling/reading/meaning/status. New entry opens editable form; Cancel creates nothing. Reading/meaning missing states are explicit. Learning/Acquired is a **manual**, reversible status.
- **Annotation:** only use a verified per-language matcher. Historical Mandarin exact substring highlighting may be enabled after tests; **do not apply it blindly to Korean**. Manual linked entries and selection work independently of optional auto-matching. Word popup remains outside transcript DOM and never moves selection offsets.
- **Favorite:** Reader header shares the Library item's persisted `isFavorite`; show busy/failure state and avoid an independent favorite store.
- **Notes:** plain text, explicit edit/save/cancel; closing presentation must not silently discard dirty content. Vocabulary/Notes/theme overlays do not reset playback.

## 7. Vocabulary and Topics

- Scope list/search/stats and entry creation to the **active language**; Topics remain **global for the one user**. Two entries with the same spelling in different languages are distinct.
- Vocabulary: Initial/loading/stale/error/true-empty/filter-empty, Learning vs Acquired, search in word/reading/meaning, topic filters, deterministic pagination (initial **10 rows/page**), details/notes/source references, add/edit/delete, bulk status/delete. Disable repeated writes and preserve confirmed status on failure.
- Reader collection: normalize identity according to the selected language's documented key rule, enforce duplicate safety server-side, and reject source links across languages. User-visible duplicate state offers linking the existing entry, not creating a second record.
- Manual source linking/unlinking is supported; automatic publication matching/sync is **only** offered for a language with a verified matcher. If unsupported (including Korean initially), do not display a fake success count or imply morphology-aware matching.
- Topic management: default Topics cannot be deleted; custom Topics can be renamed, recolored and removed with a confirmation showing item/vocabulary associations. Deletion removes joins, not entities. Duplicate name conflicts retain user input and refresh catalog. Limit linked Topics to **10** per item/entry.

## 8. Settings and local data

- Preferences: active learning language, theme (light/dark/system), per-language preferred voice and synthesis rate, Reader playback speed, vocabulary popup visibility. Validate persisted client-local values; corrupt, blocked or obsolete storage values fall back safely. No login/profile/billing flow.
- Display storage/configuration status and **documented** backup/restore guidance for the **local PostgreSQL volume and asset directory**. A **tested manual backup/restore** is an MVP requirement; **one-click backup UI is not**. Do not advertise a button for unimplemented backup logic.
- Offline means locally saved text/audio can be used when the app/database run; **new Azure synthesis requires network** and explicit user action. Missing provider credentials must not block reading saved material.

## 9. Navigation guards, confirmation and accessibility

| Dirty context | Primary leave action | Alternatives |
|---|---|---|
| Unsaved new item (eligible draft) | **Save Draft & Leave** | Leave Without Saving; Stay |
| Saved draft / published metadata / Notes | **Save Changes & Leave** (new draft may say Save Draft) | Leave Without Saving; Stay |
| Working revision | **Save Revision & Leave** | Leave Without Saving; Stay |

Intercept in-app navigation including Back; navigate **only after successful save**. Use native `beforeunload` for tab close/reload/external navigation; no custom browser-close dialog. Discard removes only unsaved local changes, not persisted records. Confirm deletion/discard revision/source unlink with accurate consequences. A dialog from the npm UI package must handle focus trap, Escape/close, accessible labels and return focus; the TaoTale caller decides whether the action is allowed while a mutation is pending.

## 10. Canonical labels and acceptance examples

**Examples:** Library, Vocabulary, Settings, Create New Item, Save Draft, Generate Audio, Publish to Library, Edit Text & Voice, Save Revision, Publish Revision, Discard Revision, Favorite, Remove from Favorites, Favorites only, Beginner, Intermediate, Advanced, Unspecified, Open in Reader, Continue Editing, Back to Library, Download Audio, Add to Vocabulary, Already in Vocabulary, Learning, Acquired, Retry, Outdated audio, Audio unavailable, No items match your filters.

**Critical UI acceptance:** (1) favorite persisted across restart without content/audio version change; (2) two languages never mix list/voice/word contexts; (3) corrupt alignment preserves readable transcript; (4) dirty guard never promises an unconfirmed save; (5) invalid 1,000,001-byte cover never replaces valid cover; (6) failed revision retains previous recording; (7) Reader audio stays mounted during theme/overlay changes; (8) no AI, Dashboard, practice, islands, login, billing or phone-only flows in MVP.

**Related documents:** [10 — API](10-taotale-api.md), [11 — Operations](11-taotale-operations-and-security.md), [12 — Quality](12-taotale-quality-and-acceptance.md).
