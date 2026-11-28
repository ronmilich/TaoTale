# TaoTale

The SolidJS app lives in [frontend](frontend/README.md), the Go service in [backend](backend/README.md), and the Compose stack in [infra](infra). The [roadmap](docs/taotale-roadmap.md) tracks milestone status.

## Run locally on Windows

Start Docker Desktop. On first setup, copy `infra/.env.example` to `infra/.env` and set `POSTGRES_PASSWORD`. Keep the same password when reusing the database volume.

Run this from the repository root.

```powershell
.\taotale.cmd
```

This builds the frontend and backend and starts PostgreSQL. The backend serves the built frontend on the same port as the API. With the default port, open the [TaoTale Library](http://127.0.0.1:8080/library).

To generate audio with Azure Speech, set `TAOTALE_BACKEND_TARGET=speech-runtime` in `infra/.env` and fill in `AZURE_SPEECH_KEY` and `AZURE_SPEECH_REGION`. Run the same start command to rebuild with the speech runtime. Settings shows its local readiness and provides backup and restore commands.

To stop the stack and keep your database and recordings, run this from the root.

```powershell
.\taotale.cmd stop
```

## Development checks

Install Node 24.19.0 from `.nvmrc`, pnpm 11.25.0 from `frontend/package.json`, and Go 1.27.4 from `backend/go.mod`. Docker runs the pinned Gitleaks scanner and the local database stack.

- In `frontend/`, run `pnpm install --frozen-lockfile`. Run `pnpm format` to format source. The CI checks are `pnpm check:ui`, `pnpm check:ui:test`, `pnpm format:check`, `pnpm lint`, `pnpm typecheck`, `pnpm test --run`, and `pnpm build`.
- In `backend/`, run `gofmt -w ./cmd ./db ./internal`, then `go mod verify`, `go vet ./...`, and `go test -race ./...`. CI also runs Staticcheck v0.8.2.
- From the repository root, run `sh scripts/sqlc.sh diff` to check generated database queries and `bash scripts/test-db.sh` to test migration, seed and restore against disposable PostgreSQL.
- Update `backend/api/openapi.yaml` before implementing handlers. In `frontend/`, run `pnpm api:generate` to refresh the declarations and `pnpm api:check` to lint the contract and check freshness.
- From the repository root, run `git config core.hooksPath .githooks` to enable the pre-commit hook. It checks staged whitespace and secrets, Go formatting and vet, and frontend formatting, lint and UI dependencies. Run `sh scripts/check-secrets.sh` in Git Bash or another POSIX shell to scan Git history.

The frontend uses native controls and app-owned styling. Document the purpose of a new runtime dependency before adding it to the allowlist in `frontend/scripts/check-ui.mjs`.
