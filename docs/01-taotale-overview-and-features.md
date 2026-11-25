# 01 — TaoTale: Product overview and features

**Status:** Proposed local-first, single-user MVP specification (updated **25 November 2026**). This is a **target product specification**, not a statement that every feature has been implemented.

The [specification index](14-taotale-sources-and-coverage.md) identifies the Mandarin Reader source baseline. The [TaoTale decision register](02-taotale-decision-register.md) records which legacy decisions carry forward.

**Priority legend:** **MVP** = intended in the first usable TaoTale release; **Post-MVP** / **Later** = explicitly deferred and not required for MVP acceptance. There is **no commercial / paid-launch gate** in the current roadmap.

## 1. Product and scope

### 1.1 Name and purpose

**TaoTale** is a personal, locally hosted web application for studying languages through reading, listening, and reusable vocabulary. For the MVP, the user adds personally interesting texts, generates natural-sounding audio with Azure TTS, and reads with synchronized text highlighting. AI-assisted text creation is planned after MVP.

**Core learning loop:** **Choose language → add text → create audio → read/listen → collect vocabulary → revisit.** Optional AI text generation and vocabulary practice follow after MVP.

**MVP visual and UI direction (25 Nov):** Every component is implemented **inside TaoTale** with SolidJS, native HTML and app-local CSS Modules; **no external UI library** or specialized generic component JavaScript. The approved look is the **Iris palette**—violet, lavender, periwinkle and orchid with beautifully composed light/dark reading surfaces. See [03](03-taotale-client.md) and [08](08-taotale-design-system.md).

**Product goal:** A dependable personal learning tool, **not** a complete language curriculum, proficiency evaluator, or public SaaS platform.

### 1.2 Current deployment and user model — approved direction

- **MVP:** **one user**, running the app for personal use. The user must exist as a persisted domain entity, but there is **no registration, onboarding account flow, login, logout, password recovery, roles, subscriptions, or payment integration**.
- **MVP:** the application, backend, **PostgreSQL database in Docker**, uploaded covers, generated MP3 files, timing metadata, previews and other owned assets are **stored and operated locally** on the user's machine. PostgreSQL data persists in a Docker volume. Do not require a hosted application server, managed database, cloud object bucket, or account service.
- **MVP:** a **desktop-first web application** supporting desktop and tablet-sized browser layouts. **Phone/small-screen support is post-MVP**, and there is no phone usability acceptance requirement. The default deployment is on the same PC; access from another physical device requires a separately approved, secured network setup. Responsive layout does **not** imply cross-device sync.
- **MVP learning languages:** **Simplified Mandarin (`zh-CN`)** and **Korean (`ko-KR`)**. New languages should be configurable through language-specific capabilities, not branches scattered throughout generic UI code.
- **MVP interface language:** English. Interface language and learning language are separate concepts.
- **MVP:** select learning language explicitly. A Library item belongs to one language; Vocabulary belongs to the user **and** language; Topics are shared across the user's languages.
- **Important distinction:** **local data storage does not guarantee offline generation**. MVP Azure TTS sends submitted text to an external service and requires connectivity. The deferred AI workspace will have the same external-provider consideration unless a future local model is configured.
- **Future possibility only:** if TaoTale is commercialized later, authentication, storage isolation, cloud deployment, legal/billing requirements and migration will be a separate phase. Do not design the current UI around hypothetical customers.

**Recommended deployment boundary:** bind the backend to `127.0.0.1` by default. No direct public exposure and no unauthenticated LAN access. Treat opening access to other devices as a new security requirement, not as a simple host setting.

### 1.3 Learning and engineering principles

1. **Explicit user actions:** speech synthesis, saving, publishing, deleting and changing vocabulary status do not happen silently. Future AI text generation will follow the same rule.
2. **Exact-source integrity:** authored text retains whitespace/paragraphs; generated speech and timed segments must match exact synthesis inputs.
3. **Stable published content:** failed revisions or generation attempts cannot break a playable, published Library item.
4. **Language-aware behavior:** voice catalogs, pronunciation display, **language-specific text segmentation rules**, and vocabulary recognition must reflect the item language. Chinese, Korean and future Japanese each have their own registered segmentation strategy and version.
5. **Manual learning labels:** difficulty and `Learning`/`Acquired` statuses are user-assigned; do not imply automatic mastery assessment.
6. **Lean local architecture:** preserve an internal user entity, use **Docker-hosted local PostgreSQL** and local filesystem assets, but avoid SaaS infrastructure, cloud persistence, billing, tenant-management and distributed services.
7. **Recoverable local data:** backups must include both the database and assets; local persistence by itself is not a backup.
8. **Feature reuse:** manual content uses the validated audio, Reader, Library and vocabulary pipeline. When the AI workspace arrives after MVP, generated articles must reuse that same pipeline, not create a parallel one.

### 1.4 Retain, change, add, postpone

| Treatment | Scope |
|---|---|
| **Retain in MVP** | Drafts, immutable audio assets, Azure TTS generation jobs, publication and working revisions, Reader, Library, manual vocabulary, shared Topics and Favorites. |
| **Change** | Project name → **TaoTale**; public multi-user SaaS → **single local user**; local **PostgreSQL via Docker** + filesystem assets; language-specific vocabulary and segmentation; generalized pronunciation field; difficulty → three manual levels; covers ≤1 MB. |
| **Add in MVP** | **Library Favorites**, per-language segmentation strategies, Docker/PostgreSQL persistence, local Settings and backup/restore controls. |
| **Remove from current scope** | Sign-up/sign-in/sign-out, onboarding/account recovery, subscriptions, billing/entitlements, public hosting, cloud media storage, multi-user authorization tests, public Terms/Privacy onboarding flows. |
| **Post-MVP / Later** | **Dashboard, Vocabulary Practice, AI Text / Audio Generation Workspace, Language Islands Builder**, **phone/small-screen support**, scoring, streaks, SRS, learning roadmaps, dictionary/word-level speech, sophisticated AI tutoring, cross-device sync and full offline inference. |

## 2. Language, difficulty, vocabulary and topic decisions

### 2.1 Scoping rules

| Data / feature | Scope | Rule |
|---|---|---|
| User | One local persistent record | Seed or resolve the default user on initialization; keep a stable `userId` for data ownership and future migration. |
| Learning language | App catalog | MVP: `zh-CN`, `ko-KR`; Japanese (`ja-JP`) requires its own language capabilities when added. |
| Library item | User + **one language** | Required; language should not change after publication. |
| Favorite flag | Per **Library item** | Independent of language, publication state, topics and difficulty. |
| Vocabulary entry / status | User + **one language** | Identical-looking strings in different languages remain separate entries. |
| Vocabulary references | Same language | An entry can reference several Library items of its own language. |
| Topic | User-global | Reuse History, Science, Travel, etc. across languages and content types. |
| Voices / synthesis preferences | Per language | Never show incompatible voice options. Azure TTS remains behind a provider adapter. |
| Text segmentation | Per language + algorithm version | Chinese and Korean have distinct MVP rule sets; Japanese must have its own rules when supported. |
| Reader resume | Local device + item + asset version | A replacement recording invalidates the previous position. |

**Data-model recommendation:** retain a `User` table/entity even with only one owner. All new records use the stable local `userId`; however, **do not take an arbitrary client-supplied `userId` as authority**. The backend resolves the default owner internally. No complex authentication abstraction is needed until the deployment model actually changes.

**Linguistic constraints:** Chinese does not normally separate words with spaces; Korean particles, inflections and spacing complicate string matching. Keep original display spelling, use documented Unicode normalization for lookup keys, and do not promise morphology-aware word recognition. Pinyin is a Mandarin-specific *reading system*, not a universal field name; use an optional generic reading/pronunciation field with an optional system label.

### 2.2 Manual difficulty levels — approved for MVP

Use this **ordered, language-independent, user-assigned three-level set** on Library items:

| Stored key | Display label | Intended interpretation |
|---|---|---|
| `beginner` | Beginner | Short, simple, controlled language. |
| `intermediate` | Intermediate | Wider vocabulary and grammar; some complexity. |
| `advanced` | Advanced | Complex constructions and less common vocabulary. |

- Difficulty is **optional** (`null` / Unspecified) and assigned or changed manually. Never automatically infer or overwrite it from text, AI output, HSK or TOPIK.
- It is **metadata**, not part of the audio fingerprint: changing difficulty must **not** force speech regeneration.
- Difficulty filters use these stable keys; display labels are separate from persisted values.
- **Post-MVP:** the AI workspace may accept these levels as **generation instructions**, but model compliance is not guaranteed or certified.
- **Optional future additions (not added to the MVP enum):** `elementary` between Beginner and Intermediate, and `upper_intermediate` between Intermediate and Advanced. **Recommendation: retain the three approved levels**; more tiers risk arbitrary classification. If later introduced, migrate filters and sorting explicitly.

## 3. Core domain concepts

| Concept | Definition / invariant |
|---|---|
| **Local user** | One persistent owner of the data and settings, resolved internally; no authentication screens. |
| **Learning language** | Language/locale with supported voices, a registered versioned segmentation strategy, and language-specific text and vocabulary capabilities. |
| **Library item** | Exact source text and metadata belonging to one language; MVP items are standard articles. |
| **Favorite** | Persistent Boolean bookmark on a Library item; has no effect on text, audio validity or publication. |
| **Draft** | Explicitly saved, potentially incomplete item; cannot open in Reader until published. |
| **Published item** | Item with a validated active text/audio/timing pair; publication means Reader-ready locally, not publicly shared. |
| **Working revision** | Independently saved candidate text/voice/rate for a published item; active audio remains unchanged until publishing the candidate succeeds. |
| **Temporary preview** | Generated output for unsaved work; expires if not claimed. |
| **Generation job** | Persisted Azure TTS task: `PENDING → PROCESSING → READY` or `FAILED`; retry semantics must prevent accidental repeated paid API calls. |
| **Audio asset** | Immutable MP3, measured duration, exact synthesis fingerprint, language-specific segmentation algorithm ID/version and timed segments, stored on local disk. |
| **Outdated audio** | Candidate audio that does not match its exact text/language/voice/rate/segmentation algorithm/version; not publishable. |
| **Vocabulary entry** | User- and language-specific word/phrase with manual `Learning` / `Acquired` state and optional metadata/references. |
| **Topic** | Reusable owner-global subject label with persisted coordinated light/dark colors. |
| **Language segmentation strategy** | Per-language, versioned rules mapping exact source-text spans to ordered segments; aligned timing is persisted with each recording. |

## 4. Main application views and feature requirements

**MVP views (only):**

| View | Purpose | MVP expectation |
|---|---|---|
| **1. Language Library** (`/library`) | Manage study material and main entry point | Drafts/published items, Favorites, difficulty/topics, filters, sorting, search, paging, CRUD. |
| **2. Create / Edit Library Item** | Prepare ordinary content | Metadata, source, Azure TTS synthesis, drafts, publishing, working revisions. |
| **3. Reader** | Read and listen to published content | Language-specific synchronized segments, playback, selection, vocabulary, notes, favorite toggle. |
| **4. Vocabulary** (`/vocabulary`) | Manage collected entries | Per-language word list, statuses, metadata, topics and item references. |
| **5. Settings / Local Data** | Configure local installation | Theme, language and voice defaults, provider configuration, PostgreSQL/asset backup and storage information. |

**MVP navigation:** `/` should redirect to `/library` or show the Library directly, with no Dashboard. The active language applies to Library, Create/Edit and Vocabulary; a Reader always uses the item's stored language. **Do not expose a `/generate` AI workspace, Vocabulary Practice view or Language Islands view in MVP navigation.** No account/sign-in landing page.

### 4.1 Library, search, and Favorites

- **MVP:** one language-scoped mixed Library of **saved drafts and published items**. Unsaved previews and working revisions do not appear as separate items.
- **MVP cards:** cover or placeholder, title, short description, manual difficulty, up to two topic chips (+N), updated/created time, draft/published indication and **favorite star**; show audio duration only for a valid published recording.
- **MVP search/filter:** literal case-insensitive title substring; draft/published status, difficulty, topics, **Favorites only**, Apply filters together; topic selections are OR within their group. Default sort by newest, plus oldest, title A–Z/Z–A; pagination must be deterministic.
- **MVP Favorites:** provide a visible star button on cards and in Reader. Toggling is explicit, idempotent and persisted in the **local database**, so it survives restart. Both drafts and published items may be favorited. Toggle does **not** update source content or invalidate audio. The favorite flag remains on an item through metadata edits and published revisions; it disappears when that item is deleted.
- **MVP:** favorite state, filters and pagination update immediately after a confirmed write. If the view is filtered to Favorites only, unfavoriting removes the card from those results; preserve sensible page position. Support keyboard controls and pointer input; do not make stars hover-only. Touch on tablets is supported, but phone-sized layouts are deferred. Favoriting an item **does not** change `updatedAt` or the default newest sort unless explicitly designed to do so.
- **MVP actions:** published → Open Reader, Edit, Download Audio, Delete; draft → Continue Editing, Delete. Confirm destructive deletion and make clear that vocabulary entries themselves survive a source-item deletion.
- **MVP:** skeleton/loading, Retry, actual-empty/filtered-empty, failed cover/audio and pagination-after-delete states.
- **Later:** favorite folders/collections, full-text article search, description/notes search, public sharing and collaborative libraries.

### 4.2 Create, drafts, Azure TTS generation, and publishing

**Editor fields:** learning language (new item), title, exact plain-text source, description, optional cover, optional manual difficulty, topics, notes (private plain text), compatible voice, synthesis rate. MVP supports standard articles only; no Language Island-specific fields or item-kind selector are required.

- **MVP:** explicit **Save Draft**, **Generate Audio**, **Publish** actions. Typing is not an implicit save. Use dirty-navigation safeguards and preserve unsaved values after errors.
- **MVP:** saved drafts can have incomplete title/text; drafts cannot open in Reader. A saved draft or revision survives navigation while generation runs.
- **MVP:** unsaved generated audio may be a temporary preview. Reuse the existing short-lived preview/claim semantics (legacy baseline: 24 hours), with cleanup; favor **Save Draft → Generate** in the main UI.
- **MVP:** **Azure TTS remains the speech provider behind a TTS adapter**; generation uses language-compatible voice and rate. Persist an immutable request snapshot and idempotency key, with visible `Pending`, `Processing`, `Ready`, `Failed`, `Outdated` states and bounded concurrency.
- **MVP:** segment text using the **registered strategy for the item language** (`zh-CN` rules for Chinese, `ko-KR` rules for Korean); record the algorithm/version and original text spans. Verify nonempty audio, measured duration and valid segment timing/alignment before marking `READY`; highlight **timed segments**, not falsely precise individual phonemes. Japanese will have its own strategy when supported.
- **MVP:** publish only nonblank titled text with a verified, matching `READY` asset. Source/language/voice/synthesis rate/segmentation strategy **or version** changes invalidate candidate audio; difficulty, favorite, title, description, topics, cover and notes do **not**.
- **MVP:** one independent **working revision** for published content. Publishing replaces active source + audio + timings atomically; failed regeneration never breaks the previous published recording. Reject stale writes with explicit conflict feedback.
- **MVP:** accept new or replacement covers only if the encoded file size is **≤1 MB (1,000,000 bytes)**; validate actual file type, dimensions and size on both client and server, without silent truncation. Store covers, MP3s and timing files locally; recover gracefully from missing files, upload/copy failures, invalid metadata and disk-full errors. Avoid overwriting immutable assets in place.
- **Later:** automatic autosave/recovery, rich text, multiple revisions, version history UI, user-uploaded recording alignment, voice cloning.

### 4.3 Reader

- **MVP:** published items only; drafts offer Continue Editing; missing IDs show not-found. There is no public/private permission distinction for a single trusted local user.
- **MVP:** render original source including whitespace and blank paragraphs; keep text readable even if audio or alignment is invalid.
- **MVP:** play/pause, timeline, seek ±5 seconds, speed, duration, MP3 download; activating a valid timed segment seeks to its start without breaking text selection. Highlight the active **segment**.
- **MVP:** persist playback position locally per item **and audio asset version**; do not autoplay upon restore. Clear obsolete position when a published revision changes the audio.
- **MVP:** provide **Favorite / Unfavorite** in the Reader header with the same persistence semantics as the Library star. No second independent favorite state.
- **MVP:** select a word/phrase → explicit Add to Vocabulary; use the Reader item's language. Lookup existing words and link the source without silently replacing metadata or status.
- **MVP:** show vocabulary highlights and popups only where language-specific matching is trustworthy. Manual text selection and lookup continue to work if automatic matching is unavailable. Do not transfer Mandarin substring heuristics untested to Korean.
- **MVP:** display known saved word, reading, meaning, topics and manual status; do not invent missing definitions. Keep notes edit/save/cancel explicit.
- **MVP:** desktop and tablet selection, keyboard navigation, visible focus state and reduced-motion behavior. Phone-specific interactions/layouts are deferred.
- **Later:** word-level TTS, dictionary integration, pronunciation assessment, automated speaking feedback, offline playback across other devices.

### 4.4 Vocabulary

- **MVP:** vocabulary unique by `(userId, languageCode, normalizedWordKey)`; entries use manually controlled `Learning` (default) and `Acquired` statuses.
- **MVP:** store original spelling, optional language-aware reading and system, optional meaning/note, Topics, timestamps, and references to Library items **in the same language**. Do not automatically convert Simplified/Traditional Chinese or infer Korean base forms.
- **MVP:** Reader selection → lookup → link existing entry idempotently or show an editable Add dialog. Save new entry and source reference atomically; cancel leaves both unchanged. Handle duplicate races explicitly.
- **MVP:** manual add/edit/delete, single/bulk status updates and delete, search/filter/sort/pagination, global language-specific counts and source-item drill-down.
- **MVP:** user-global Topics can label words in any language. Deleting an article removes its source references but never deletes its collected vocabulary.
- **MVP:** do not infer acquired status, meaning or morphological lemma from repeated appearances. Manual associations are the correctness baseline; enable language-tested automatic highlights if reliable.
- **Later:** dictionary translations, morphological matching, SRS exports, background Library scans, automatic proficiency transitions.

### 4.5 Topics

- **MVP:** Topics are **global to the single local user**, not language-specific. Initial examples: History, Science, Technology, Business, Culture, Daily Life, Travel, Health.
- **MVP:** create/rename/recolor/delete custom topics; case-insensitive uniqueness; readable light/dark colors; topic deletion removes joins only. Preserve the current default-topic protection behavior if still useful.
- **MVP:** topic associations work for Library items and vocabulary. Keep the existing maximum of 10 topic associations per entity initially.
- **Later:** topic folders, hierarchy, language-specific visibility and topic analytics.

### 4.6 Settings, local storage and backup

- **MVP:** settings for light/dark/system scheme, active language, per-language default voice and synthesis rate, Reader playback speed and vocabulary popup visibility.
- **MVP:** show configured local filesystem asset path, PostgreSQL/Docker storage and backup guidance, data reset/delete confirmation and Azure TTS connection status. **No account management page**.
- **MVP:** support **consistent backup and restore** of all locally owned data: database, article covers, audio, timings, settings and vocabulary. Provide a verifiable recovery test before treating backup as reliable. At minimum, a tested `pg_dump`/restore plus asset-backup procedure is required; one-click UI can follow the first stable release.
- **MVP:** on startup, resolve/create the single user idempotently and load stored data; do not introduce a registration flow.
- **Later:** encrypted backup, optional user-managed sync to a trusted storage location, remote access and multi-device synchronization (each is a separate architecture/security decision).

## 5. Local architecture, privacy, and reliability requirements

### 5.1 Local infrastructure — approved MVP stack

- **Database (decided): PostgreSQL, running locally in Docker.** Start it with a documented Docker Compose service; use a **persistent named volume or explicitly managed bind mount** for PostgreSQL's data directory. A database container is disposable; its persisted volume is not. Define a health check and deterministic migrations/seed for the single User record.
- **Connectivity:** the local backend uses the containerized PostgreSQL instance. If a database port is published to the host, bind it to `127.0.0.1` only; if the backend is also containerized, prefer Docker's internal network and do not publish the database port at all. Never expose a credential-free database on the LAN.
- **Assets:** store immutable MP3s, cover images, preview files and any filesystem timing artifacts in a **persistent local application data directory** (bind-mounted if the backend is in Docker), not inside ephemeral container layers or a frontend build directory. PostgreSQL stores metadata and safe asset identifiers/paths. Asset cleanup must not delete referenced published media.
- **User:** one internal seeded/resolved owner. Preserve `userId` on records and schema constraints; no login, session, tenant management, or account-service dependency.
- **Backend:** serve on loopback by default; restrict file APIs to the configured asset root and validate against path traversal. Provider secrets stay in backend config, never in frontend code or committed files.
- **No required infrastructure:** cloud PostgreSQL, cloud object storage, authentication provider, payment provider, Redis, microservices or SaaS monitoring services.
- **Offline boundary:** previously stored content should remain accessible while the local UI/backend/database are running even without internet. **Azure TTS requests require provider connectivity.** The future AI workspace may also require internet.

### 5.2 Language-specific text segmentation and synchronized audio — approved decision

- **One segmentation strategy per learning language**, selected through a registry/interface by `languageCode`. For MVP, implement and test separate **Simplified Mandarin (`zh-CN`)** and **Korean (`ko-KR`)** strategies. **Japanese (`ja-JP`)** needs its own rules if/when added; do not apply Chinese rules to Japanese just because both use Han characters.
- Each strategy decides **segment boundaries** appropriate to the language (punctuation, sentence/phrase boundaries, whitespace behavior and mixed-script edge cases), while preserving **the exact original source text**. Chinese text cannot be segmented reliably by spaces; Korean requires handling Hangul, sentence endings and punctuation. A generic regex may be shared internally only where language tests establish equivalent behavior.
- Persist the **strategy identifier and semantic version** with audio/timing assets and include them in synthesis/alignment fingerprints. Store ordered, nonoverlapping source offsets using one documented offset convention; preserve all untimed original text, including blank lines. TTS timing/alignment must be validated against these source spans; segmentation alone does **not** supply accurate speech timestamps.
- Source edits or a different strategy/version require fresh matching candidate generation **before republishing**. Do not silently invalidate, rewrite, or replace existing verified published recordings merely because application code introduces a newer segmentation algorithm; retain the pinned legacy strategy/version until an explicit regeneration/revision.
- Test each language with punctuation, multi-paragraph text, quotes, numbers, abbreviations, mixed scripts, emoji and provider timing edge cases. Failure to produce safe alignment must not fabricate precision: show readable text and disable unusable seek/highlight controls.
- **Keep segmentation distinct from vocabulary matching**. Sentence/audio segmentation can be language-specific without claiming Korean lemmatization or full Chinese word segmentation/dictionary capabilities.

### 5.3 Provider adapters, jobs, assets and integrity

- **TTS decision:** keep the existing **Azure TTS** integration, but access it through a clearly defined provider adapter so the Library/Editor/Reader do not depend on Azure-specific response types or credentials. Voice lookup, synthesis, provider timing and errors are normalized at the adapter boundary.
- **AI decision:** retain the **previously chosen AI provider behind an AI adapter**, but **the AI Text/Audio Generation Workspace and AI-generation product workflow are post-MVP**. Preserving the provider choice/adapter contract does **not** require adding AI routes, prompt UI or AI API calls to the MVP. The audio-generation action in the Editor remains MVP via Azure TTS.
- Persist generation jobs so browser refresh or backend restart cannot silently lose state; reconcile uncertain `PROCESSING` jobs without assuming an external request was or was not billed. Use stable idempotency keys, immutable generation intent and bounded concurrency; do not automatically repeat costly unknown-outcome requests.
- Verify nonempty audio, duration, exact source/language/voice/rate/segmentation fingerprint, and timed-segment boundaries before declaring `READY`. Atomically publish only a matching verified candidate.
- Use crash-safe file writes (temporary file, validate, atomic rename where supported) plus consistent database references and orphan cleanup. **PostgreSQL transactions cannot roll back a filesystem write.** Handle disk-full, missing/corrupt assets and interrupted jobs explicitly.
- **Backup as a unit:** use a consistent PostgreSQL backup (e.g., `pg_dump`, not a naïve copy of live PostgreSQL data files), and back up referenced asset files and config needed for recovery. Document restore order and verify a full restore in a clean Docker volume/application-data directory. Avoid destructive `docker compose down -v` as a routine stop command.
- Log actionable failures without dumping private study passages or provider credentials. No hosted observability stack is required.

### 5.4 Device support and accessibility

- **MVP:** support **desktop and tablet-sized browser layouts**. Test readable typography, native link/button/form interaction, usable zoom and clear visual states. Tablet actions must not depend exclusively on mouse hover. No custom keyboard/focus system or formal accessibility certification is required.
- **Post-MVP:** **phone/small-screen layouts**, phone-specific navigation, narrow touch transcript interactions and testing on actual phones. Avoid claiming phone support from incidental CSS responsiveness. Separately approve secured LAN access if the app needs to run on a second device.
- Use **simple app-owned native controls** with readable labels, clear status and consistent **Iris violet/lavender** styling. No standalone UI library or formal WCAG/custom component-focus implementation is required. Verify synchronized highlighting across Mandarin, Hangul, punctuation and mixed scripts.

### 5.5 Provider privacy and costs

- Azure TTS sends submitted text to an external service. Clearly distinguish **local storage** from **external synthesis** and do not send content without an explicit Generate Audio action.
- Keep provider secrets in local backend configuration, outside source control and browser bundles. The retained AI provider is used for **post-MVP** text generation, not implicitly called by MVP actions.
- Optional local tracking may measure synthesized characters and estimated Azure usage; AI token tracking is relevant when the AI workspace is implemented. No payments, billing ledger or subscription infrastructure.
- Review provider terms before any future public/commercial deployment; those concerns are not MVP launch gates for this personal product.

## 6. Data model and API invariants

**Recommended entities:** `User`, `LearningLanguage`, `UserLanguagePreference`, `LibraryItem`, `WorkingRevision`, `GenerationJob`, `AudioAsset`, `TimedSegment`, `SegmentationStrategyVersion` (logical descriptor/configuration, not necessarily a table), `VocabularyEntry`, `VocabularyReference`, `Topic`, and the relevant item/topic and vocabulary/topic joins.

- `User`: exactly one active local user record. All persisted owner-linked records refer to it. Preserve stable ownership fields to ease eventual migration, without currently requiring auth.
- `LibraryItem`: `id`, `userId`, `languageCode`, `isFavorite` (`false` by default), `difficulty` (nullable enum), draft/published state, source, metadata, active asset ID and optimistic `version`.
- `AudioAsset`: immutable local file identity and duration, source fingerprint, language, TTS provider/voice/rate, **segmentation strategy ID + version**, and timed-segment alignment. The active published item references an asset verified to match its source.
- `VocabularyEntry`: uniqueness `(userId, languageCode, normalizedWordKey)`; language-specific reading and independent Learning/Acquired state. References enforce matching language and owner.
- `Topic`: uniqueness `(userId, normalizedName)` and no required language column; associations are reusable by articles and words.
- `Favorite`: **no separate entity required** for the MVP. A Boolean on `LibraryItem` is adequate for a single user; an explicit favorite collection table can be introduced only if multiple lists/ordering later justify it.
- `GenerationJob`/previews have persisted ownership, immutable request intent, provider identity, strategy version, status, created/updated timestamps and cleanup rules; do not rely on frontend state for recovery.
- **Storage migration:** migrate existing database records to local **PostgreSQL** without regenerating verified audio or renumbering IDs. Retain existing audio/timing segmentation versions for published assets and represent legacy versions explicitly.
- Avoid exposing unrestricted local path parameters over APIs. Return app-scoped media references, and map them to files rooted in configured storage.
- **Important migration:** backfill all existing Mandarin data as `zh-CN`, seed `isFavorite = false`, map old `pinyin` to optional reading metadata and assign the one stable local `userId` without replacing existing IDs or audio links.

## 7. Input limits and preferences

Keep known Mandarin Reader input limits as an **initial implementation baseline** unless verified provider or usability tests show a reason to change them.

| Input | Starting limit / rule | MVP treatment |
|---|---|---|
| Title | 1–120 Unicode code points to publish; optional in draft | Retain. |
| Article source | Legacy ≤1,500 UTF-16 units | Re-evaluate for Korean; define Unicode-aware display and TTS-provider constraints before expanding. |
| Description / notes | ≤1,000 / ≤5,000 code points | Retain initially. |
| Cover | **JPEG/PNG/WebP ≤1 MB (1,000,000 bytes)**, max 4096×4096 | Reject oversized **new/replacement** uploads; validate actual bytes, extension-independent type and dimensions on client and backend. Do not silently compress/truncate. Existing larger images require a documented migration policy and must not silently disappear. |
| Topic count / name | ≤10 per item or entry; name 1–40 code points | Retain initially. |
| Vocabulary spelling | Legacy 1–30 code points | Re-evaluate Korean phrases; never silently truncate. |
| Reading / meaning / word note | Legacy ≤160 / ≤300 / ≤5,000 code points | Make reading language-aware, retain lengths initially. |
| Difficulty | `beginner`, `intermediate`, `advanced`, or `null` | Three fixed, ordered manual levels; no HSK/TOPIK mixing. Migrate legacy `native` values deliberately, never silently. |
| Favorite | Boolean | Defaults to `false`; toggle without audio invalidation. |
| Synthesis rate | 0.75 / 1 / 1.25 (legacy) | Validate per voice/provider. |
| Reader speed | 0.75 / 1 / 1.25 / 1.5 (legacy) | Retain initially. |
| AI inputs | Existing Chinese-oriented constraints | **Post-MVP only:** revisit per-language/provider limits when the AI workspace is implemented. |

Validate on both UI and backend, without invisible truncation. Persist theme, language, voice defaults and local playback preferences locally. No cloud account synchronization is promised.

## 8. Explicitly out of scope for the current MVP

- **Dashboard** (`/` as a personalized landing page): no recent/favorite widgets, learning analytics, Continue Reading dashboard or Dashboard-specific entry point. For MVP, `/` navigates to the Library.
- **Vocabulary Practice:** no practice route, reveal cards, practice sessions, review queues, scores or repetition logic. Normal Vocabulary management and manual Learning/Acquired status changes remain MVP.
- **AI Text / Audio Generation Workspace** (`/generate`): no prompt-based text generation, refinements, candidate workflows, AI vocabulary sampling or AI workspace UI. **Manual text editing and Azure TTS audio synthesis remain MVP.** Keep the chosen AI provider decision/adapter boundary for the later feature.
- **Language Islands Builder:** no Islands view/route, dedicated model, script builder, rehearsal UI or generated-dialogue workflow in MVP.
- **Phone/small-device support:** no phone-sized navigation/layout commitment or mobile browser QA gate; desktop and tablets remain supported.
- Authentication screens, registrations, sign-in/out, email flows, account recovery, multiple users, roles, organizations and collaboration.
- Hosted production SaaS, paid plans, payments, invoices, subscription management and cloud-hosted object storage/database.
- Cross-device data sync and remote network access; secure LAN use requires a separate security decision.
- Gamification, score/streak system, learning roadmap and automatic proficiency/difficulty assessment.
- SRS, scheduled flashcards, inferred acquisition/status and automated review intervals.
- Rich-text notes, shared/public libraries, advanced Favorite collections, description/notes search.
- Automatic simplified/traditional conversion, morphological lemmatization, automatic definitions/translations, word-level TTS, dictionary lookups and background Library-wide vocabulary scans.
- AI comprehension quizzes, grammar tutoring, full generation history, autonomous regeneration; these are **not** automatically included when the basic post-MVP AI workspace is introduced.
- Speech recognition/scoring, multi-turn AI conversation, independent per-sentence audio tracks, structured branching conversations, true offline AI/TTS inference, native mobile apps and arbitrary uploaded-audio alignment.

### 8.1 Planned post-MVP feature: Dashboard

**Status: deferred. Not a launch route or MVP acceptance criterion.** A future personal homepage may show the active language, Continue Reading, recent items, Favorites, shortcuts to Library/Create and per-language item/vocabulary counts. The MVP Library is the initial page instead. Streaks, goals, recommendations and learning analytics are later enhancements, not part of this basic Dashboard commitment.

### 8.2 Planned post-MVP feature: Vocabulary Practice

**Status: deferred.** A future language-scoped practice mode can start with `Learning` entries, present stored spelling, allow mental recall, reveal the saved meaning/reading, provide Next/Previous and explicit manual status changes. It should gracefully handle blank meanings and must not change `Acquired` automatically. SRS, scoring, scheduling and persistent flashcard decks require separate decisions.

### 8.3 Planned post-MVP feature: AI Text / Audio Generation Workspace

**Status: deferred. No `/generate` route, prompt workflow or AI calls required for MVP.** Retain the **chosen AI provider behind an adapter** and reuse the existing Azure TTS adapter and manual Library/Reader pipeline when the feature is implemented.

**Potential later workflow:** choose language, prompt/theme, length, optional style and **Beginner / Intermediate / Advanced** guidance; optionally include saved words from the *same language*; explicitly generate text; inspect/edit/refine candidates; then intentionally Save as Draft or proceed to the existing voice/generation/publishing pipeline. Required-word validation should be explicit, while grammar/difficulty/factual correctness remain advisory. AI text generation and TTS audio synthesis are independent, opt-in provider actions, with no silent save or paid retry.

### 8.4 Planned post-MVP feature: Language Islands Builder

**Status: deferred. Not part of MVP scope, navigation, schema, implementation stages, or acceptance criteria.** Do **not** add a `/islands` route, Island-specific UI/table, item-kind discriminator, rehearsal mode or island prompts merely to satisfy the MVP.

**Concept (future proposal):** A *Language Island* is a short, rehearsable passage or set of sentences about a familiar conversational situation (e.g., introducing yourself, discussing work or describing a hobby). Its intent is reusable spoken-language rehearsal, not a second vocabulary collection.

**Potential future workflow:** choose language → describe a situation → write or explicitly AI-generate a script → edit → save → synthesize audio → read/listen/rehearse; optional hide/reveal prompts for recall. Sending personal information to AI must be an explicit action.

**Design preference, not an MVP contract:** reuse Library drafts, verified generation jobs, published Reader assets, Favorites, Topics and language-scoped vocabulary. Consider an item-kind discriminator and Island metadata only **when this feature is prioritized**. Speech recognition, branching dialogue and Island-specific SRS are later possibilities.

## 9. Incremental implementation plan (existing Mandarin Reader → TaoTale)

| Stage | Deliverable | Exit condition |
|---|---|---|
| **0. Reconcile requirements** | Rename product; update decision register; fix MVP/post-MVP boundary, three-level difficulty, 1 MB cover rule, PostgreSQL/Docker, Azure/AI adapter choices and per-language segmentation. | No Dashboard, Practice, AI workspace, Islands, phone or SaaS requirements leak into MVP backlog. |
| **1. Local PostgreSQL and assets** | Docker Compose PostgreSQL with persistent volume, migrations/seeding, single User, local asset root, restore procedure and data migration from existing storage. | Data and verified media survive restarts/redeployment; a `pg_dump` plus asset restore works from a clean environment. |
| **2. Multilingual Reader + TTS parity** | `zh-CN`/`ko-KR` language metadata, separate versioned segmentation strategies, Azure TTS adapter, compatible voices, synchronized Reader and vocabulary scoping. | Both languages can Add → Generate Audio → Publish → Read → Save Vocabulary with validated highlighting. |
| **3. Core library polish** | Favorite toggle/filter, manual Beginner/Intermediate/Advanced difficulty, 1 MB cover validation, robust Library/Create/Vocabulary/Reader/Settings behavior. | Favorites and filters persist, only allowed difficulty values save, oversized covers reject without damaging existing content. |
| **4. Local hardening** | Disk and interrupted-job recovery, PostgreSQL+asset backups, provider credential handling, accessibility, desktop/tablet QA and regression tests. | Local failures cannot silently destroy verified published content; supported devices and restore path are tested. |

**Post-MVP boundary:** Dashboard, Vocabulary Practice, AI Text/Audio Generation Workspace, Language Islands Builder and phone/small-device support are separately scheduled features **after** the MVP; they are **not** completion conditions for any stage above.

**Migration safeguards:** preserve article/vocabulary IDs, audio fingerprints, working revisions, references and manual `Acquired` statuses. Backfill the user and language (`zh-CN`) and set `isFavorite = false` by default. Existing audio is pinned to its **original segmentation version**, not silently resegmented on upgrade. For legacy difficulty values, especially `native`, require explicit review/mapping or leave difficulty Unspecified; do **not** automatically map `native` to `advanced`. Existing covers over 1 MB must be retained until deliberately replaced or migrated, rather than deleted automatically.

## 10. Core MVP acceptance journeys

1. **Local Docker/PostgreSQL startup:** start PostgreSQL with Docker Compose; launch backend/UI locally; resolve the stable internal user; go directly to Library **without login or Dashboard**; restart containers/app and retain data.
2. **Manual content:** choose Mandarin or Korean → create/save draft → generate Azure TTS audio via adapter → validate source segments/timings → publish → read/listen/download.
3. **Language-specific segmentation:** Chinese and Korean texts use their respective strategy/version and correct source spans; mixed punctuation/line breaks preserve exact display; invalid alignment cannot be published as verified or produce misleading seeking.
4. **Revisions:** metadata-only changes preserve audio; text/voice/segmentation changes require a saved candidate and matching recording; a failed attempt leaves the previous published Reader intact.
5. **Favorites:** star a draft or published item → reopen Library/restart → state persists → Favorites-only filter shows it → unstar removes it from results without deleting the item.
6. **Difficulty:** assign Beginner, Intermediate, Advanced or Unspecified → save/filter → change without regenerating audio; `native` is not accepted as a new value.
7. **Cover images:** accept correctly typed images ≤1,000,000 bytes and within dimension bounds; reject 1,000,001-byte uploads or invalid files without silently overwriting existing media.
8. **Vocabulary:** collect a selected phrase in Mandarin or Korean → save/link → see it in language-specific Vocabulary → manually change status → deleting article leaves the entry.
9. **Provider failure:** disconnect internet or fail Azure request → existing local Reader/audio still works → generation shows recoverable error without automatic duplicate billing.
10. **Persistence and restore:** restart during job, simulate missing media and disk-full, then restore a consistent PostgreSQL backup + referenced asset set; corrupted states are detected with actionable recovery.
11. **Supported-device/accessibility QA:** desktop and tablet browsers support reading, selection, focus and key navigation; phone-sized layout is **not** an MVP acceptance requirement.
12. **Scope gate:** no Dashboard, Vocabulary Practice route, `/generate` AI workspace, Language Islands route, phone-specific UX, sign-in or payment integrations are required or exposed as finished MVP features.

## 11. Key risks and pitfalls

| Risk | Consequence | Default mitigation |
|---|---|---|
| **PostgreSQL data lives only in an ephemeral container** | Recreating a container wipes the study history. | Persist PostgreSQL data with a named volume/bind mount; document start/stop and forbid destructive volume removal in normal operations. |
| **Inconsistent PostgreSQL and asset backups** | Restored rows reference lost MP3s/covers. | Consistent `pg_dump` and corresponding asset snapshot; tested restore, integrity checks and migration plan. |
| **Local-only mistaken for offline-only** | Azure TTS fails without internet or content unexpectedly leaves the PC. | Explicit Generate action and provider disclosure; Reader works with saved files offline. |
| **Unauthenticated backend accidentally exposed on LAN** | Local private data can be accessed from the network. | Loopback bind, Docker network isolation, no published database port unless loopback-only. |
| **Wrong segmentation rules reused across languages** | Chinese/Korean sentences split incorrectly, leading to bad highlighting or alignment. | Registered language-specific, versioned strategies; source-offset and timing tests per language; separate Japanese rules later. |
| **Strategy upgrades silently invalidate published audio** | Existing playable articles become broken when segmentation code changes. | Pin published audio to original algorithm/version; regenerate only through explicit revision. |
| **Mandarin vocabulary substring assumptions reused for Korean** | False highlights and invalid word references. | Keep segmentation and vocabulary matching separate; manual linking remains correctness baseline. |
| **Azure implementation leaks into application domain** | Future provider changes require rewriting Editor/Reader. | Adapter for voices, synthesis, timing and errors; AI provider similarly isolated for later use. |
| **Oversized/invalid cover uploads** | Excess disk usage or inconsistent UI and stored files. | Enforce ≤1,000,000 bytes and file signatures/dimensions server-side; preserve legacy oversized covers until deliberate migration. |
| **Single-user shortcuts damage future data model** | Future data migrations become costly. | Keep stable internal User, `userId`, language and IDs without implementing SaaS infrastructure. |
| **Favorites coupled to revisions** | Star click invalidates audio or reshuffles the newest list. | Independent metadata flag with stable publication and sort semantics. |
| **Audio diverges from source** | Reader highlighting claims false timing. | Immutable source fingerprints, verified timed spans and atomic publish. |
| **Interrupted paid TTS jobs** | Orphan files and accidental duplicate Azure charges. | Durable idempotency/request state, capped concurrency, uncertain-outcome recovery. |
| **Post-MVP features creep back in** | Core Reader and migration stay unfinished. | Keep Dashboard, Practice, AI workspace, Islands and phone QA outside the MVP acceptance gate. |

## 12. Open implementation decisions (not blockers to this update)

The following are **implementation details to resolve**, not reversals of approved choices: **PostgreSQL with Docker is decided; Azure TTS and the chosen AI provider behind adapters are decided; per-language segmentation is decided; phone support is post-MVP.**

1. **PostgreSQL packaging:** choose Docker Compose service names, database/role credentials, stable named-volume location, migration tooling, and whether the backend runs on host or inside Docker. Keep all network exposure local.
2. **Existing data migration:** confirm current DB engine/schema, write repeatable migration and integrity checks into PostgreSQL, and document clean restore plus rollback. Preserve article IDs, original audio assets and segmentation versions.
3. **Precise segmentation rules:** specify and test Mandarin and Korean punctuation/paragraph cases, source-offset convention, source-text-to-TTS chunk mapping, and the policy for legacy segment versions; define Japanese rules when Japanese is added.
4. **Exact article length and TTS chunking:** legacy 1,500 UTF-16 units vs longer texts; verify Azure voice/timing behavior and cost/latency before changing the limit.
5. **Vocabulary highlighting strategy:** manual-linked-only vs separately verified per-language matching; begin with reliable manual selection.
6. **Backup UX:** tested manual `pg_dump`/restore + asset procedure first; one-click backup/import UI later if it adds value.
7. **Legacy difficulty/cover migration:** decide what happens to old `native` or other non-enum difficulty values and pre-existing >1 MB covers without implicit data loss.
8. **Future difficulty refinements:** consider `elementary` and `upper_intermediate` only if the three-level system proves too coarse.
9. **Later product modules:** schedule Dashboard, Vocabulary Practice, AI workspace, Language Islands, and phone support independently rather than prebuilding routes or schema for them.

## 13. References and maintenance

- Existing Mandarin Reader requirement and decision files remain the **implementation baseline** until each relevant behavior is migrated and tested.
- [Language Islands — Fluent in 3 Months](https://www.fluentin3months.com/language-islands/) — background on preparing focused conversational material; reference only for a deferred post-MVP concept; it does not define MVP behavior.
- [08 — Iris design](08-taotale-design-system.md) — application-owned colors, typography, page composition and minimal native components.
- [BCP 47 / RFC 5646](https://www.rfc-editor.org/info/rfc5646/) — language identifiers.
- [Azure Speech language support](https://learn.microsoft.com/en-us/azure/ai-services/speech-service/language-support?tabs=text-to-speech) — verify actual Azure voice and timing support for each supported language.

---

The [decision register](02-taotale-decision-register.md) records the local single-user scope, naming, difficulty, Favorites, storage, provider adapters, segmentation, cover limit and deferred features. Keep the detailed specs aligned with this overview and track implementation in the [roadmap](taotale-roadmap.md).
