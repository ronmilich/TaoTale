# 11 — TaoTale local operations, deployment and security

**Status:** Target MVP operating model, revised **25 November 2026**. **Authority:** [01 — Overview](01-taotale-overview-and-features.md), [04 — Go backend](04-taotale-backend.md), [05 — PostgreSQL model](05-taotale-database.md), [10 — API](10-taotale-api.md), and [12 — Quality](12-taotale-quality-and-acceptance.md). Commands and configuration keys shown here are **proposed implementation conventions**, not verified commands from an existing TaoTale repository.

## 1. Deployment boundary and selected stack

- **Local, single-user web app**: SolidJS + TypeScript + Vite client, compiled for same-origin serving; **Go + Gin** HTTP server; **PostgreSQL in Docker** with persistent named volume; **pgx/v5 + sqlc** and forward SQL migrations; **local filesystem** for MP3s, cover images and temporary uploads.
- The **SolidJS frontend owns its Iris CSS Modules and small native UI components**. No external UI component package, Kobalte/Tailwind, generic overlay manager or special accessibility JavaScript is part of the application. The Iris theme is local CSS, not a runtime CDN or npm dependency.
- Backend binds to **`127.0.0.1` by default**, serves `/api/v1` and the built SPA from the same origin where feasible. Docker PostgreSQL port is loopback-only if published to the host; keep PostgreSQL on a private Compose network if Go is containerized. **Never** publish Gin/PostgreSQL unauthenticated on LAN or internet. Tablet-sized responsive UI does not imply network access from a second tablet.
- A single persisted User row is seeded/resolved internally. No registration, sessions, OAuth, login, billing, cloud storage, managed jobs or hosted monitoring requirements in MVP.
- **MVP learning languages:** `zh-CN` and `ko-KR`. **Japanese, Dashboard, Vocabulary Practice, AI workspace, Language Islands and phone-sized layouts are not deployment blockers because they are deferred.**

## 2. Repository, local paths and processes

Repository structure:

```text
frontend/                       # SolidJS app with app-local Iris CSS/components
backend/                        # Go module (Gin handlers, services, sqlc output)
  db/migrations/                # reviewed forward-only .sql migrations
  db/queries/                   # sqlc input queries
  db/sqlc.yaml                  # sqlc configuration
  internal/                     # domain/services/adapters/storage/worker
infra/                          # compose.yaml and .env.example
docs/                           # numbered TaoTale specifications
spikes/                         # isolated feasibility experiments
local-data/                     # example of gitignored persisted media root
```

- Keep real database volume and media directory **outside version-controlled/source build paths**. Docker container replacement must not remove data; ordinary shutdown must **never** use `docker compose down -v`.
- Store **opaque, generated relative storage keys** in PostgreSQL, not client-provided file paths or audio BLOBs. Set absolute `MEDIA_ROOT`, create it with restrictive permissions, reject traversal and symlink/junction escapes, and serve media through checked HTTP routes.
- TT-036 selects Compose-run Go. The API reaches PostgreSQL via `db:5432` on the Compose network. Only the API publishes a host port, on loopback. The named PostgreSQL and media volumes live outside the source tree and container layers. See the [backend start and stop instructions](../backend/README.md).
- App should remain usable for **previously saved reading/audio without internet** as long as local frontend/backend/PostgreSQL and media are available. Azure synthesis remains an external network service.

## 3. Configuration and secrets

| Setting (proposed) | Constraint |
|---|---|
| `-listen` / `PORT` | Standalone Go defaults to `127.0.0.1:8080`. In Compose, `PORT` selects the published loopback port. |
| `TAOTALE_DEV_ORIGIN` | Optional single HTTP loopback origin for writes sent through a local dev proxy with a different Host. |
| `DATABASE_URL` or separate PG fields | Local PostgreSQL credentials; backend only, never exposed in browser build. |
| `MEDIA_ROOT` | Existing/creatable absolute persistent local directory, writable and safe. |
| `TTS_PROVIDER` | `azure` or explicit deterministic `mock` in test/dev only. |
| `AZURE_SPEECH_KEY`, `AZURE_SPEECH_REGION` | Backend-local secret configuration; required for real generation, **not** for read-only use. |
| `TTS_VOICE_CATALOG` or voice config | Explicit provider/locale allowlist; Chinese and Korean voices must be verified against region and timing behavior. |
| `TTS_MAX_CONCURRENT_GENERATIONS` | Initial cap **2**; configurable and bounded. |
| `TTS_MAX_GENERATIONS_PER_HOUR` | Historical default **60** as local cost-safety cap; not an entitlement. |
| `TTS_REQUEST_TIMEOUT` | Historical starting target **60s**; verify SDK cancellation/platform behavior. |
| `CLEANUP_INTERVAL` | Historical starting target **60 minutes**, bounded work. |
| `SERVE_FRONTEND_DIST` | Optional absolute path to built SolidJS static assets. |
| `LOG_LEVEL` | Structured safe metadata, default info. |

Exact variable names, file permissions and platform-specific credential storage are resolved in runnable setup instructions. No application `.env`, PostgreSQL password, provider key, user text or private media belongs in Git, CI output, frontend bundle or `VITE_*` environment variables. **Do not implement AI credential flows merely because a post-MVP design exists** ([07](07-taotale-ai-text-generation.md)).

## 4. First start and upgrades

1. Validate Docker availability, correct Compose project and **persistent PostgreSQL volume**. Start PostgreSQL; wait for DB readiness. Never reset the real database to solve startup failures.
2. Validate environment and `MEDIA_ROOT`; apply reviewed, versioned **SQL migrations** (not Prisma push, synchronize, or auto-generated ORM schema). Maintain migration checksums/version history.
3. Seed or resolve **exactly one stable local User**, enabled `zh-CN`/`ko-KR` languages and idempotent default Topics. Rerunning seed must preserve edited Topic names/colors, IDs and references.
4. Validate expected segmentation registry and legacy strategy support; reconcile unfinished TTS jobs and local staging/media references **without automatically repeating Azure calls**. Begin bounded cleanup and in-process worker.
5. Check `/health/live` and `/health/ready` (database, migration status, media root, seeded owner, completed recovery). If Azure credentials are absent, **generation endpoints may return `TTS_UNAVAILABLE` while Library/Reader remain usable**.
6. In development, use a Solid/Vite proxy to Gin and an explicitly permitted dev origin; in production-like local mode, serve static client and API same-origin. Test SPA deep links, 404 API paths, forbidden dotfiles and media path isolation.

On upgrade, stop or quiesce writes and generation, take paired backup, apply forward SQL migrations, seed idempotently, validate audio/refs, restart and verify readiness. **Do not treat changing the Go executable or Docker image as a license to recreate persistent volumes.**

## 5. Persistent jobs, provider and file lifecycle

- In one committed SQL operation, record `PENDING` job and immutable generation intent/idempotency evidence **before returning `202`**. The Go worker claims jobs with bounded concurrency and marks `PROCESSING` before calling Azure **outside** a database transaction.
- For success, stage MP3 bytes; decode/measure duration; validate timed boundaries against exact text/language/voice/rate/segmentation ID+version/fingerprint; promote immutable file; then atomically create asset row and set job `READY`. A provider success response alone is **not READY**.
- For failure/crash, classify and persist sanitized failure, remove unsafe staging files, preserve original published asset; **never auto-resubmit** a potentially charged request. Recover uncertain results with the original key/intent before offering a new explicit attempt.
- Startup recovery reconciles interrupted states without inferring a successful or unsuccessful charge. Separate local cleanup of old previews (historical default **24 hours**), unattached uploads and orphan media from durable active/revision/latest candidates.
- Filesystem and PostgreSQL commits are **not one atomic operation**. Prepare bytes before DB reference, release obsolete files after commit and use rechecks/leases to avoid deletion during streaming/copying. Validate media availability in readiness/integrity routines appropriately; a single missing legacy file should surface recoverable diagnostics, not be silently replaced.
- Stream audio using safe asset IDs and single-range `206` semantics; pin mutable item media URLs to expected asset version. A stale URL must not receive new recording bytes with old offsets.

## 6. Local security and privacy

| Threat | Protection |
|---|---|
| Network or DNS-rebinding access to unauthenticated app | Loopback-only listener and Docker port, explicit host/origin validation on browser mutations; no permissive CORS. Reassess immediately before LAN access. |
| Cross-site browser requests from another tab/site | Same-origin checks for state-changing requests, safe content types, CSRF-conscious API design; do not infer that localhost is immune. |
| Traversal, symlink escape or arbitrary file read | Generate server-side keys; secure root-relative path resolution/open; never accept caller-controlled path segments for filesystem access. |
| Malicious cover uploads | Streaming size cap **≤1,000,000 encoded bytes**, actual JPEG/PNG/WebP signature + dimensions ≤4096×4096; avoid decompression bombs and keep previous cover if replacement fails. |
| Private text/credentials in logs | Log only route/status/requestId/job ID/typed code and timing; redact source, title, notes, words, prompts, provider responses, secrets and absolute paths. |
| Accidentally billable retry | Idempotency and bounded worker admissions; only explicit Generate triggers paid synthesis, no hidden retries/fallback. |
| Frontend dependencies and style assets | Keep a small pinned SolidJS dependency set; review runtime packages, bundled local CSS/JS and SVGs; no external UI-component package/CDN is needed. |
| Misleading alignment | Never serve corrupt/mismatched audio as READY. Preserve text-only Reader fallback. |

TaoTale app-local CSS and native controls are presentation only, **never security validation or authorization**. No custom overlay/focus system is needed. Keep private Local Data/backups access within explicit local paths; backup archives can reveal all user-created text and audio.

## 7. Paired PostgreSQL + assets backup and restore

**Backup must represent one consistent point in time across database and filesystem.** Tested manual backup/restore is an MVP gate; an automated Settings button is optional/post-MVP.

Settings provides copyable PowerShell commands for the default Compose installation. They pause the API, save the paired database dump and media archive, and restore them into an independent Compose project. Preserve the browser profile and local setup credentials separately. TAO-44 owns the release manifest and full integrity comparison.

**Backup procedure (conceptual):**

1. Quiesce Gin writes and TTS worker (or stop backend cleanly); make sure no file promotions or cleanup are in progress.
2. Create a **PostgreSQL custom-format `pg_dump`** from the running Docker DB; preserve it as a **binary** file. Do **not** archive live PostgreSQL data directory as a database backup.
3. Archive the matching entire configured media root and required nonsecret configuration, and write a manifest of dump timestamp, schema revision, app/segmentation versions and media checksums. Keep secrets in a separately controlled secure location if recovery requires them.
4. Store the paired artifacts **outside** Docker's data volume and project source, with access restrictions. Verify the manifest/archives were created before resuming normal service.

**Restore rehearsal:** into a **fresh disposable DB volume and empty media root**, restore binary dump with `pg_restore`, restore media with exact relative keys, reconcile required migrations/seeds, then run read-only integrity checks while writes are stopped. Compare IDs, item/word/topic counts, references, cover sizes, MP3 hashes, measured duration, segmentation strategy/version and fingerprint against source. Recover historically released READY audit rows according to retention rules, not as missing active assets. Start local service, open Mandarin and Korean sample Readers, confirm notes/vocabulary/favorites and resume behavior. **Never restore destructively onto the only working copy.**

Settings should show honest manual instructions/status, **not** an inert Backup button pretending the process is implemented.

## 8. Verification, monitoring and incident response

- Structured logs, localhost health endpoints and optional process-local metrics are adequate for the personal MVP; no cloud monitoring/Redis/event bus required. Avoid recording private article contents in diagnostics.
- Test local restart, disk-full, permission errors, missing media, corrupt alignment, Docker volume persistence, migration checksum mismatch, database outage/restart, expired previews, duplicate generation and abrupt backend termination.
- Measure Gin/sqlc API p95, RSS/CPU, worker concurrency and route/network overhead separately from Azure provider latency. Go was selected for performance/control, **not** as proof that externally synthesized audio is faster.
- For release, capture actual command output, commit, schema revision, Iris frontend revision, browser versions, startup logs and paired restore evidence. **Historical Mandarin Reader test counts do not certify TaoTale.** See 12.

## 9. Deferred and unresolved operations

**Deferred:** app accounts/auth/HTTPS for public exposure, billing and multi-tenant quotas, hosted database/storage, cross-device sync, public dashboards, AI runtime credentials/inference, phone support, network-connected tablet deployment, high-availability worker coordination and one-click backup UI.

Remaining setup decisions include production Azure runtime packaging, MP3 decoding, backup paths and procedures, and Chinese automatic vocabulary matching. See the [decision register](02-taotale-decision-register.md#6-implementation-decision-register).
