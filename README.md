# TaoTale

The SolidJS app lives in [frontend](frontend/README.md), the Go service in [backend](backend/README.md), and the Compose stack in [infra](infra). The [roadmap](docs/taotale-roadmap.md) tracks milestone status.

## Development checks

Install Node 24.19.0 from `.nvmrc`, pnpm 11.25.0 from `frontend/package.json`, and Go 1.27.4 from `backend/go.mod`. Docker runs the pinned Gitleaks scanner and the local database stack.

- In `frontend/`, run `pnpm install --frozen-lockfile`. Run `pnpm format` to format source. The CI checks are `pnpm check:ui`, `pnpm check:ui:test`, `pnpm format:check`, `pnpm lint`, `pnpm typecheck`, `pnpm test --run`, and `pnpm build`.
- In `backend/`, run `gofmt -w ./cmd ./internal`, then `go mod verify`, `go vet ./...`, and `go test ./...`.
- From the repository root, run `git config core.hooksPath .githooks` to enable the pre-commit hook. It checks staged whitespace and secrets, Go formatting and vet, and frontend formatting, lint and UI dependencies. Run `sh scripts/check-secrets.sh` in Git Bash or another POSIX shell to scan Git history.

The frontend uses native controls and app-owned styling. Document the purpose of a new runtime dependency before adding it to the allowlist in `frontend/scripts/check-ui.mjs`.
