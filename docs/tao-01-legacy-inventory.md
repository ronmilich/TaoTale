# TAO-01 legacy inventory and restore rehearsal

On 25 November 2026, the configured Mandarin Reader database ran on PostgreSQL 17.11. The source schema and implementation reference is Mandarin Reader revision `dc71c1b1820345023585be95e72674bf1b3a4a00`. The private backup folder is named `Mandarin Reader Backups/TaoTale-20261125-210008` under the owner's Documents folder. It contains `database.dump`, `media.tgz` and `verification.json`, including archive checksums and the IDs of the oversized covers.

## Inventory

| Legacy table | Rows | Findings |
|---|---:|---|
| `_prisma_migrations` | 13 | All applied, with no unfinished migrations. The applied IDs match the directories at the pinned revision. |
| `users` | 1 | Preserve the owner UUID. |
| `library_items` | 2 | Both published with active audio. One `INTERMEDIATE` and one unspecified. Zero `native` or unexpected values. |
| `working_revisions` | 0 | The schema supports one working revision per item. |
| `audio_assets` | 2 | Both use segmentation version 1 and match their generation fingerprints. |
| `audio_generations` | 5 | All READY and version 1. The pinned implementation reproduced all five fingerprints. Three records have no attached asset. |
| `media_images` | 2 | Both exceed 1,000,000 bytes. Both files are present at the recorded size. |
| `idempotency_records` | 17 | All have recorded responses and had expired by the snapshot. Generation idempotency keys live separately on the five generation rows. |
| `vocabulary_items` | 20 | All have nonblank pinyin. Seventeen also have a legacy `source_item_id`. |
| `vocabulary_references` | 21 | Sixteen manual-only, three automatic-only and two with both flags. |
| `topic_tags` | 11 | Preserve Topic UUIDs and their associations. |
| `item_tags` | 2 | Item-to-Topic associations. |
| `vocabulary_tags` | 5 | Vocabulary-to-Topic associations. |

The live schema has 33 UUID columns. The media tree contained two MP3s and two images, with zero orphan files, missing references or byte-size mismatches. Table digests include the original UUIDs, all stored text and the timing JSON.

## Restore verification

1. Confirmed no other clients were connected to the legacy database. Exported a repeatable-read, read-only snapshot and captured it with `pg_dump --format=custom --snapshot`. Archived the configured media directory and checked its file sizes and SHA-256 hashes before and after capture.
2. Restored with `pg_restore --exit-on-error --single-transaction --no-owner --no-acl` into an empty PostgreSQL 17.11 container with networking disabled. Extracted the media archive into an empty directory.
3. Compared all 13 table counts and SHA-256 digests of sorted row hashes, plus 211 schema signatures covering columns, constraints, indexes and enum members. Compared the media trees by file key, size and SHA-256. All comparisons matched. Removed the disposable restore container and extracted copy after verification.

The rehearsal uses disposable PostgreSQL ownership and permissions. Set up runtime roles separately during cutover. TaoTale import and Reader playback validation are tracked in TAO-45 and TAO-46.

## Compatibility and performance baseline

The [synthetic golden vectors](../testdata/legacy/mandarin-v1.json) were generated and checked against the pinned implementation. All eight segmentation cases reconstruct their source exactly from UTF-16 spans and gaps. All five fingerprint cases match SHA-256 over the exact UTF-8 JSON bytes. They cover punctuation, quotes, Han-hyphen rules, CR/LF, supplementary characters, JSON escaping and voice/rate changes.

On this machine, Node 24.19.0 queried the legacy PostgreSQL 17.11 container over loopback. Queries selected up to 25 rows for the legacy owner, ordered by `updated_at DESC, id DESC`. Each query used 5 warmups and 40 measured calls. Times include client round trips and reflect the small restored workload.

| Query | Rows | Median | p95 |
|---|---:|---:|---:|
| Library page | 2 | 1.393 ms | 2.217 ms |
| Vocabulary page | 20 | 1.348 ms | 1.914 ms |

For larger and browser-level comparisons, the pinned legacy repository also recorded [database plans](https://github.com/ronmilich/mandarin-reader/blob/dc71c1b1820345023585be95e72674bf1b3a4a00/docs/development/evidence/performance/database.json), [vocabulary matching](https://github.com/ronmilich/mandarin-reader/blob/dc71c1b1820345023585be95e72674bf1b3a4a00/docs/development/evidence/performance/matching.json) and [browser loading](https://github.com/ronmilich/mandarin-reader/blob/dc71c1b1820345023585be95e72674bf1b3a4a00/docs/development/evidence/performance/loading.json) results.

## Migration strategy

TT-033 selects an import into a fresh TaoTale schema. Use the restored copy as the read-only input for the rehearsal. Import records and references in dependency order, preserve UUIDs and timestamps, and use the legacy owner UUID when setting up the single local owner. Map legacy content and vocabulary to `zh-CN`, initialize Favorites to false and preserve pinyin as the reading with the pinyin reading system.

Keep source text byte-for-byte, including CR/LF and whitespace. Keep original MP3 and cover bytes, active pins, timed UTF-16 offsets and version-1 fingerprint identity. Preserve manual and automatic provenance independently and retain any legacy source-item link. Keep expired replay rows and released READY history in the verified snapshot. If the importer exposes that history, treat it as historical data without replaying requests or promoting history without an asset to a playable recording.

Before cutover, verify counts and UUIDs, file hashes, source and timing integrity, references, repeat import behavior and actual Reader playback. Retain the original database/media pair and the backup for rollback.

OPEN-07 remains for TAO-45. Preserve both oversized covers and their item references. Decide how imported covers are represented while enforcing the 1,000,000-byte limit on new or replacement covers. The snapshot has no unexpected difficulty values. The importer still needs an explicit policy for future snapshots containing `native` or other unsupported values, with the affected UUIDs and original values reported for review.
