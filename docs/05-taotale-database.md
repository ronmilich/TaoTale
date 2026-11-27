# 05 — TaoTale database and persistence

The first physical migration creates the owner, learning language catalog and Topics. Later sections describe the logical schema for subsequent milestones. [Catalog migration](../backend/db/migrations/0001_catalog.sql)

**Authority:** [`01-taotale-overview-and-features.md`](01-taotale-overview-and-features.md) defines product behavior; [`02-decision-register.md`](02-taotale-decision-register.md) tracks decisions; [`04-taotale-backend.md`](04-taotale-backend.md) defines the **Go + Gin + pgx/v5 + sqlc** architecture. This replaces the old **Prisma-centric** database document without discarding valid data-integrity decisions from Mandarin Reader.

## 1. Storage and modeling rules

- **PostgreSQL running locally in Docker**, with a durable named volume/bind mount, backup/restore and forward SQL migrations. The **Go backend** connects using **pgx/v5**; SQL queries live in `.sql` files and **sqlc** generates typed methods. **No Prisma ORM/schema/migrations, GORM or database blobs for images/MP3.**
- **One stable local `User`**, resolved or seeded by the backend. All owned domain records carry `user_id` or are reachable through an owner-scoped parent. No account registration, password table, sessions, subscriptions or billing data in MVP. Retain IDs/ownership for migration flexibility.
- All IDs are UUIDs, timestamps are `timestamptz`, SQL columns/tables are `snake_case`; external JSON is camelCase. Exact source text is stored unnormalized. Client-side offsets are **UTF-16 code-unit indices**, not PostgreSQL character positions or Go UTF-8 byte offsets.
- `LearningLanguage` is a **small seeded reference table** (`zh-CN`, `ko-KR` for MVP), while executable segmenter and voice capabilities live in Go. Keep the seeded codes synchronized with the code registry; Japanese `ja-JP` is **not enabled until implemented**. A separate language row does not imply a separate user.
- Library and vocabulary are language-scoped. Browser-held **per-language synthesis preferences stay client-local for MVP**, as specified in 03; server persistence is optional and must not compete with browser state. **Topics are user-global**. Favoriting is a Boolean on an item, **not** a separate table. No Language Island, practice session, Dashboard aggregate or AI-generated article entity in MVP.
- Store safe relative paths/keys and file metadata in PostgreSQL; store MP3s, covers and temporary preview bytes in a persistent filesystem data root. `AudioAsset` stores immutable alignment metadata, not the MP3 body.

## 2. Entity audit and updated logical model

| Entity / SQL suggestion | Key fields and relationships | TaoTale decision |
|---|---|---|
| **User** (`users`) | `id`, optional local display name, `created_at`, `updated_at` | **Retain.** Seed exactly one stable local owner; **remove required unique email/auth fields**. No account UI. |
| **LearningLanguage** (`learning_languages`) | `code` PK (`zh-CN`, `ko-KR`), `display_name`, `enabled` | **New.** FK target/seed catalog; language capabilities are registered in Go. No auto-enabled Japanese. |
| **UserLanguagePreference** (`user_language_preferences`) | `(user_id, language_code)` PK, nullable default voice/rate, timestamps | **Optional future entity, not required for MVP.** Current per-language synthesis defaults are stored in the client as specified in 03; do not add a second source of truth. |
| **LibraryItem** (`library_items`) | `id`, `user_id`, **`language_code`**, `title` nullable, **exact `source_text`**, description/notes nullable, **`difficulty` nullable** (`beginner`, `intermediate`, `advanced`), **`is_favorite BOOLEAN NOT NULL DEFAULT FALSE`**, nullable `cover_image_id`, `voice_id`, `speaking_rate`, `active_audio_asset_id`, `version`, `published_at`, timestamps | **Update.** One language per item; mixed drafts/published. Favorite also allowed on drafts. No item-kind/Island fields. |
| **WorkingRevision** (`working_revisions`) | `id`, `item_id UNIQUE`, `base_audio_asset_id`, candidate exact text, candidate voice/rate, `version`, timestamps | **Retain/adapt.** Candidate inherits item language; one per published item. Published active pair stays unchanged until atomic publish. |
| **AudioGeneration** (`audio_generations`) | `id`, `owner_id`, `language_code`, `provider_key`, `target_type`, nullable `target_id`, immutable source snapshot, voice/rate, **`segmentation_strategy_id`**, **`segmentation_version`**, **`fingerprint_format_version`**, `source_fingerprint`, `idempotency_key`, status, nullable `audio_asset_id`, errors, claim/expiry/heartbeat/worker timestamps | **Update.** Persistent TTS job, not AI job. Candidate identity is language/strategy-aware. `OUTDATED` computed; `FAILED` records interruption. |
| **AudioAsset** (`audio_assets`) | `id`, `owner_id`, `language_code`, `provider_key`, `storage_key UNIQUE`, MP3 MIME/size, measured `duration_ms`, exact voice/rate/fingerprint/strategy ID+version, **`fingerprint_format_version`**, **`segments JSONB`**, `created_at` | **Update.** Immutable bytes and timing pair. Metadata must identify the exact source/segmentation; keep legacy fingerprint reader. |
| **Topic** (`topics`) | `id`, `user_id`, `name`, normalized `name_key`, system/default indicators, validated light/dark text/background color values, timestamps | **Rename/retain `TopicTag`.** **No language column.** Global catalog reusable across Chinese/Korean. |
| **ItemTopic** (`item_topics`) | `(item_id, topic_id)` PK | Renamed `ItemTag`; join only, delete cascade. Max 10 per item at service/transaction boundary. |
| **VocabularyEntry** (`vocabulary_entries`) | `id`, `user_id`, **`language_code`**, display `word`, **`normalized_word_key`**, optional **`reading`**, **`reading_system`** (e.g. pinyin/romanization), meaning/note, `status` `LEARNING`/`ACQUIRED`, nullable **legacy** `source_item_id`, timestamps | **Update `VocabularyItem`.** Uniqueness **`(user_id, language_code, normalized_word_key)`**. No universal `pinyin`; do not auto-convert scripts/lemmas. |
| **VocabularyReference** (`vocabulary_references`) | `(vocabulary_entry_id, library_item_id)` PK, `added_at`, `manual_linked`, `auto_linked` | **Retain/adapt.** Manual association works for both languages; matching owner + language must be checked/enforced. Auto provenance only from verified per-language matcher. |
| **VocabularyTopic** (`vocabulary_topics`) | `(vocabulary_entry_id, topic_id)` PK | Renamed `VocabularyTag`; reusable global Topics. |
| **MediaImage** (`media_images`) | `id`, `owner_id`, generated `storage_key UNIQUE`, `content_type`, `size_bytes`, `width`, `height`, nullable `expires_at`, `created_at` | **Retain with revised upload gate.** New/replacement encoded cover **≤1,000,000 bytes**; preserve legacy oversized images. |
| **IdempotencyRecord** (`idempotency_records`) | `id`, `owner_id`, `key`, `operation`, `request_hash`, nullable response status/body, `created_at`, `expires_at`, unique `(owner_id,key)` | **Retain.** Prevent duplicate billable TTS calls when request outcome is unknown. |

**Not new SQL tables in MVP:** `Favorite` (item flag), `TimedSegment` (JSONB inside immutable asset is adequate now), `SegmentationStrategyVersion` (immutable code registry + persisted ID/version is adequate now), `Voice` (provider/config catalog), `LibraryItemKind`, `LanguageIsland`, `PracticeSession`, `AIConnection`, `AIArticleHistory`, account tokens, billing/entitlement records, and per-occurrence vocabulary matches. Add tables only when a measured requirement justifies them.

**Naming migration:** The names above are **target logical names**; existing physical table/column names may be preserved where renaming creates unnecessary risk. Whichever SQL names are chosen, keep `sqlc` input, migrations and OpenAPI DTO mappings consistent. No generated ORM models are canonical.

### Legacy-to-TaoTale entity changes

| Old field/entity | New treatment |
|---|---|
| `User.email UNIQUE NOT NULL` | No email requirement; keep optional legacy email only if migration needs it. No auth schema. |
| `LibraryItem` without language | Require `language_code`; backfill historical data as `zh-CN`. |
| Library item without favorite | Add `is_favorite BOOLEAN NOT NULL DEFAULT FALSE`, no favorite row or audio fingerprint change. |
| Old difficulty including `native` | Allow **only** `beginner`, `intermediate`, `advanced` or `NULL`; report and explicitly resolve unsupported legacy values. Never silently map `native`. |
| `TopicTag`, `ItemTag`, `VocabularyTag` | Keep semantics; may rename to `Topic`, `ItemTopic`, `VocabularyTopic`. **Do not add `language_code` to Topics.** |
| `VocabularyItem.pinyin` | Migrate to `reading`; assign `reading_system='pinyin'` for known Mandarin pinyin values; allow null for both languages. |
| `isAcquired BOOLEAN` | Preserve exact Learning/Acquired state; a two-value `status` enum/text check is preferred for a clearer API, but preserve old storage until a tested migration. |
| Unique `(user_id, normalized_word)` | Change to `(user_id, language_code, normalized_word_key)`; spelling collisions across languages are permitted. |
| `AudioGeneration` / `AudioAsset` without language/strategy ID | Add both; legacy Mandarin assets use an explicit legacy fingerprint/segmentation identity, **without regenerating or rewriting their timings**. |
| Cover max 5 MiB | New or replacement image must be ≤1,000,000 encoded bytes. Existing oversized files stay referenced/readable. |
| Existing `sourceItemId` legacy pointer | Preserve and backfill a manual reference if present; optionally retire in a later validated migration. |

## 3. Detailed entity invariants

### User, Language and preferences

- Seed one stable User idempotently; do not generate a new ID at each process start. Requests never select an owner from arbitrary input. Retain owner FKs on all owned records even though only one user exists.
- `learning_languages.code` is an exact supported BCP-47 locale key (`zh-CN`, `ko-KR` in MVP). Seed only the languages implemented by the Go segmentation/voice registry. Foreign keys from Library, Vocabulary and language preferences prevent unknown values; disabling a catalog entry must not orphan historical items.
- Per-language TTS defaults, playback position, theme and popup display are small, validated **client-local** values in MVP per 03. If preference sync is introduced later, migrate their authority deliberately; **do not create duplicate competing preference authorities**.

### Library item / Favorites

- Item language is **required and immutable once created** for the MVP. Drafts can omit title/text/voice/rate; published items cannot.
- Store exact authored text. Publication is determined consistently by `published_at` (and any retained `status` column must agree). Published requires nonblank title/source, compatible voice/rate and matching verified active asset. Drafts cannot enter Reader even if generation is READY.
- `difficulty` is nullable and restricted to `beginner`, `intermediate`, `advanced`. `NULL` means Unspecified. Difficulty, `is_favorite`, notes, Topics, description, cover and title are **metadata**, not synthesis inputs.
- `is_favorite` is persisted for **both drafts and published items**. Favoriting must not bump content version, `updated_at` or default newest order. Use a focused SQL update (`SET is_favorite = $value`) with idempotent semantics and return the confirmed value.
- `version` on item/revision guards content/metadata edit conflicts. A focused favorite update is independent of the content optimistic version; subsequent ordinary item updates **must not overwrite favorite** with stale values. Ensure `sqlc` updates only requested fields rather than full stale-row replacement.
- One active asset per published item; immutable asset can be retained as revision base. A failure to publish a new candidate leaves the old item/source/audio pair unchanged.

### Audio generation, fingerprints, segments and media

- Persist statuses **`PENDING`, `PROCESSING`, `READY`, `FAILED`**; `OUTDATED` is computed from target and latest candidate. `GENERATION_INTERRUPTED` is a failure reason, not a new status.
- Persist the complete immutable intent: exact text, language, provider, voice, speaking rate, segmentation strategy ID/version and fingerprint schema version. New fingerprint format must include all these; old assets retain their **legacy** fingerprint validator (do not recompute the old SHA-256 by inventing new fields).
- Store `segments JSONB` as ordered records, for example `{index, text, textStart, textEnd, startMs, endMs}`. **`textStart`/`textEnd` are UTF-16 units**, end-exclusive; audio time bounds are integer milliseconds. Preserve punctuation/whitespace as untimed original text where appropriate. JSON array positions and values are validated on READY and again at publication, not assumed correct because JSON is well-formed.
- Audio files are immutable under generated relative keys. `duration_ms` is measured from persisted playable output; content type is `audio/mpeg`. A READY candidate must refer to verified bytes and alignment, except for explicitly documented old audit rows whose released output is no longer live.
- Preview targets may be unattached and expire; saved draft/revision latest candidates can be retained even when now outdated. Retain published active asset and revision base. When a READY asset has been released for audit, do **not** expose it as playable.
- Avoid storing both `AudioGeneration.audio_asset_id` and an additional unconstrained reverse pointer that can contradict it. If both directions exist for migration compatibility, add unique/FK consistency checks and a single write path.

### Vocabulary and references

- The uniqueness key is **per user and per language**. Form `normalized_word_key` with a documented Go normalization procedure (trim, NFC, collapse internal whitespace); preserve original display word unchanged except the established outer-trim behavior. Keys remain case-sensitive unless a language-specific normalization change is explicitly approved. Never assume Korean inflection removal, Chinese script conversion or word-boundary inference.
- `reading`/`reading_system` are **optional** (Mandarin pinyin, Korean Revised Romanization if manually entered, future Japanese furigana/romaji metadata as separately designed). `meaning` and notes are user-entered, not inferred.
- `status` is manual Learning (default) or Acquired; toggling in either direction must not modify library source or audio. Bulk operations check IDs/owner/language.
- A vocabulary reference may point only to a Library item with the **same owner and `language_code`**. Preserve `manual_linked` and `auto_linked` independently with **at least one true**; legacy original-source relationships are manual.
- **Manual collection/linking is the MVP correctness baseline.** Any automatic publication reconciliation or Reader matching uses a separately verified per-language strategy; do not apply Mandarin substring matching to Korean. If disabled for a language, keep manual references intact and do not treat missing matching as a publish failure.
- Deleting a Library item removes reference joins / clears legacy source pointers, but **does not delete words**. Deleting a word only removes its joins. Topic deletion removes joins only.

### Topics and images

- Topic name uniqueness is **`(user_id, name_key)`**, case-insensitive after trim/casefold by a documented canonicalization. Do not scope Topics by language. System default keys are stable and protected from deletion, while approved renames/recolors remain supported.
- Four saved light/dark colors (light background/text, dark background/text) retain their validated contrast. Topic association limit defaults to ten per item/word; enforce in transactional services if SQL alone cannot safely express it.
- Image file metadata is immutable after upload; item associations can change. New/replacement upload rejects **`size_bytes > 1,000,000`**, invalid signature/MIME, unsafe dimensions or paths. A DB `CHECK (size_bytes <= 1000000)` would **incorrectly reject historical oversized covers** and should not be added without a separate migration policy. Enforce new-upload limits in the Go service; existing metadata may exceed the threshold.

## 4. Keys, constraints and indexes

**Required uniqueness / foreign keys**

1. Every owned entity has a User FK; join tables connect only valid owned parents. `LearningLanguage.code` is referenced by language-scoped records.
2. `working_revisions(item_id)` unique; one persisted working revision per published item.
3. `topics(user_id,name_key)` unique and, for default Topics, `topics(user_id,system_key)` unique where `system_key IS NOT NULL`.
4. `vocabulary_entries(user_id,language_code,normalized_word_key)` unique. Case-sensitivity must match the documented normalized key; do not rely on a nondeterministic database collation.
5. `idempotency_records(user_id,key)` unique; `audio_generations(owner_id,idempotency_key)` unique where a key exists. Cross-operation key behavior must match the API contract.
6. Media `storage_key` unique and one output per live generation; asset references never point to nonexistent file **silently** (SQL FK protects DB row, not actual file existence).
7. Both provenance flags on `vocabulary_references` cannot be false: `CHECK (manual_linked OR auto_linked)`.
8. Enforce **cross-language and cross-owner reference consistency** via composite FKs/unique parent keys (e.g. include `user_id,language_code` in the association and FKs) **or a carefully tested transaction/trigger design**. An ordinary FK on each ID separately does *not* stop linking a Chinese word to a Korean item.
9. Published-state constraints should cover status/published timestamp parity and required fields where feasible; **asset fingerprint and media integrity require service-level verification** too.

**Initial indexes (refine after profiling)**

| Query | Initial index |
|---|---|
| Language-scoped Library default | `(user_id, language_code, created_at DESC, id DESC)` |
| Favorites-only Library | Partial `(user_id, language_code, created_at DESC, id DESC) WHERE is_favorite = TRUE` **if explain plans warrant** |
| Published/draft filtering | `(user_id, language_code, published_at, created_at DESC, id DESC)` only if justified by workload |
| Vocabulary uniqueness and lookup | `UNIQUE (user_id, language_code, normalized_word_key)` |
| Vocabulary page/statistics | `(user_id, language_code, created_at DESC, id DESC)` |
| Reference lookup by article | `(library_item_id, vocabulary_entry_id)` and relevant reverse join key |
| Generation latest by target | `(target_type, target_id, created_at DESC, id DESC)` plus owner/language filters |
| Pending worker admission | Partial by `(created_at, id)` where status is `PENDING`, as required by chosen worker query |
| Expiration cleanup | `expires_at` on previews/images/idempotency rows where measured useful |
| Topics by owner | Unique name key and stable default system key, plus reverse joins |

Library title search and vocabulary word/reading/meaning search use literal, escaped case-insensitive search and bounded pagination; do not add trigram/full-text extensions until benchmarks justify them. Tie-break all paginated sorts by stable IDs. Use **batch SQL joins/aggregations** instead of N+1 per article/word. For a single user, avoid precomputed statistics tables unless measured need warrants them.

TAO-29 ran `EXPLAIN (ANALYZE, BUFFERS)` on the generated page and count queries with 20,000 synthetic items split between the two languages. The existing `library_items_recent` index returned an eight-item newest page in 0.10 ms and a Favorites-only newest page in 0.10 ms. A combined title, status, difficulty and topic filter took 13.55 ms for the title-sorted page and 13.40 ms for the count. It used the item-topic primary key for membership checks. The measured plans supported retaining the existing indexes.

## 5. Transaction and filesystem consistency (DB-1–DB-9 adapted)

| ID | Invariant enforced by Go service + SQL |
|---|---|
| **DB-1** | File first, then DB reference: write to staging; validate; atomically move/copy to completed immutable key; only then commit referencing row. Failed DB commit may leave an orphan file for later cleanup. Replace/delete old files only after commit. |
| **DB-2** | Insert verified `AudioAsset` and set generation `READY`, output link and completion time **in one SQL transaction**. |
| **DB-3** | Claim temporary preview atomically; verify owner, expiry, unclaimed status, snapshot/fingerprint and valid target before linking to a saved draft or direct publication. Never duplicate a paid generation. |
| **DB-4** | Publish item only after candidate file preparation; transaction revalidates matching `READY` asset/source/language/voice/rate/strategy, sets active asset and `published_at`, and conditionally synchronizes vocabulary **only for enabled/tested language matcher**. Roll back if a required step fails. |
| **DB-5** | Revision publish locks item + revision, checks expected versions and unchanged base asset, atomically replaces active source/voice/rate/asset, and retains current metadata/`is_favorite`. Old published bytes remain valid if publish fails. |
| **DB-6** | `UPDATE ... WHERE version=$expected RETURNING ...` rejects stale writes, with `409 VERSION_CONFLICT` and no implicit merge. **Favorite toggle is a separate metadata-only operation** that must not be lost by later content updates. |
| **DB-7** | Generation admission is atomic with same-target in-flight check and idempotency key reservation. Unique/partial indexes and row/advisory locks as necessary prevent duplicate live jobs even if requests race. |
| **DB-8** | On restart, reconcile unfinished jobs; never automatically repeat an Azure request whose billing outcome may be unknown. Committed `READY` remains intact. |
| **DB-9** | Retain active published, revision-base and latest surviving candidate assets, unexpired previews and in-use files; only release genuinely unreferenced media after fresh DB checks. Audit metadata may outlive released bytes. |

**Transaction implementation:** start `pgx.Tx` from `pgxpool`; bind sqlc `Queries` to that transaction (`WithTx` or equivalent generated constructor); commit/rollback explicitly. External Azure calls and long file operations **must not hold SQL transactions open**. Use a consistent lock order for publication, reconciliation and media cleanup. Single-user deployment lowers expected contention, but it does not remove duplicate-request or crash-race risk.

### Language-aware vocabulary reconciliation

When an automatic matcher is **enabled and verified** for an individual language:

- Match only same-language words against the **exact published source** under that language's declared matching rules. Preserve separate `manual_linked` and `auto_linked` flags. Never rewrite spelling, reading, meaning, `status` or topic metadata.
- Add/update automatic references transactionally on publication; missing matches remove only the `auto_linked` flag, and delete joins only when neither provenance flag remains. Existing manual references stay untouched.
- Explicit manual link upgrades an automatic link to manual. Explicit unlink removes the association; a later *enabled* sync can rediscover it according to the documented policy. No background Library-wide scans for MVP.
- If a language's matcher is not approved (especially Korean in MVP), skip automatic reconciliation **without blocking item publication**. Manual references/highlights based on known links remain available.

## 6. sqlc workflow

The project pins sqlc v1.31.1 and checks `sqlc diff` in GitHub CI. The following example shows how the configuration links ordered SQL migrations to pgx/v5 query generation. [Actual configuration](../backend/db/sqlc.yaml)

```yaml
# backend/db/sqlc.yaml
version: "2"
sql:
  - engine: "postgresql"
    schema: "migrations"
    queries: "queries"
    gen:
      go:
        package: "queries"
        out: "../internal/persistence/queries"
        sql_package: "pgx/v5"
```

```sql
-- backend/db/queries/library.sql
-- name: SetItemFavorite :one
UPDATE library_items
SET is_favorite = sqlc.arg(is_favorite)
WHERE id = sqlc.arg(item_id) AND user_id = sqlc.arg(user_id)
RETURNING id, is_favorite;

-- name: GetVocabularyByKey :one
SELECT id, word, reading, reading_system, status
FROM vocabulary_entries
WHERE user_id = sqlc.arg(user_id)
  AND language_code = sqlc.arg(language_code)
  AND normalized_word_key = sqlc.arg(normalized_word_key)
LIMIT 1;
```

These examples show the intended **SQL-first** pattern. Real queries must match final migrations, handle `pgx` nullable types and enforce full association invariants. `sqlc` generates methods and checks query syntax/type mapping; it **does not** enforce application lifecycle rules, generate migrations, or verify files exist.

## 7. Migration, seeding, backup and integrity checks

**From the Mandarin Reader Prisma implementation:**

1. Make a complete copy/backup of its PostgreSQL DB and local media; inventory actual tables, Prisma migration SQL, rows, foreign keys, generated audio alignments and on-disk files. Design physical SQL migration against the real schema, not solely this overview.
2. Preserve UUIDs, created/updated timestamps, exact source bytes, topic identities/colors, vocabulary statuses and manual/provenance refs. Keep the existing User ID or deterministically select the one original owner.
3. Seed/reference `zh-CN` and `ko-KR`; backfill **all legacy Mandarin items, vocabulary and assets to `zh-CN`**, and introduce the language-safe composite vocabulary uniqueness and reference constraints **after validating historic joins**.
4. Backfill `is_favorite=false`, nullable three-level difficulty, generic reading metadata (`pinyin` → `reading`, `reading_system='pinyin'` where data exists) and language-scoped synthesis settings. **Explicitly report `native` and other unexpected difficulty values**, do not silently reinterpret them.
5. Preserve and mark existing audio/timing fingerprint and segmentation format as **legacy**; keep a compatibility validator and historical timing bytes. Never regenerate paid Azure assets merely because the application has new code.
6. Retain old covers above **1,000,000 bytes**; reject only new/replacement oversized uploads. Verify old file references and hash/byte integrity.
7. Validate migration and seed **twice** on nonempty DB, compare entity counts, refs, active assets and Reader timing offsets. Tests use disposable Docker volumes/databases, never the user's only data.

**Backup rule:** `pg_dump`/restore of the database is necessary but **not sufficient**: include the **consistent set of referenced local media files** and configuration required to decode/serve them. Avoid writes during a simple manual backup (stop the app/worker and quiesce file mutations), or implement a snapshot/manifest protocol. Restore the DB and asset tree together into a fresh volume/root, validate checksums/reference integrity and then start the app. Never treat `docker compose down -v` as normal shutdown.

## 8. Acceptance tests / unresolved schema choices

**Must pass before migration is considered complete:**

- Boot with clean local Docker PostgreSQL; user/language/topic seeds are idempotent and data survives container recreation.
- Chinese and Korean item/vocabulary records are separately filtered and uniqueness permits identical-looking spelling across languages, but **rejects cross-language references**.
- Favorites on draft/published items persist through restart and revisions, never alter active audio, `updated_at` or normal newest ordering; concurrent metadata edits do not reset favorites.
- `difficulty` accepts only `NULL`, `beginner`, `intermediate`, `advanced`; historical `native` is explicitly surfaced for migration decision.
- Cover 1,000,000-byte boundary test; preserve existing larger files without error when merely reading.
- Publish requires matching verified MP3, duration and segment timings; stale revision fails without breaking active Reader; restart does not double-bill Azure.
- Legacy Mandarin segment offsets/fingerprints remain readable; new Mandarin/Korean strategies store independent IDs/versions and correct UTF-16 offsets.
- SQL transactions roll back partial publication/vocabulary links; file-orphan cleanup does not erase live media; a real DB + media restore succeeds.

TT-037 records the migration runner, first catalog and sqlc pin. Later schema choices include difficulty/status representation, legacy `source_item_id` retention, language-safe composite FKs and MP3 integrity metadata.

**Maintenance:** Keep the logical invariants intact when changing code generation. Update the ordered migration SQL, `sqlc` queries and generated Go, API contract, domain behavior and regression tests together. TT-025 and TT-026 record the stack in the [decision register](02-taotale-decision-register.md).
