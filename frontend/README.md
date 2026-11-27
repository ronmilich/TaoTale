# TaoTale frontend

The SolidJS app uses Vite's existing client build, Solid Router's lazy file routes and app-owned CSS Modules. The Iris tokens and base styles live in `src/styles/global.css`.

## Run and verify

Start Compose from `infra/` for PostgreSQL and the Gin API on port 8080. In `frontend/`, run `pnpm install --frozen-lockfile` and `pnpm dev`. Open `http://127.0.0.1:3000`. The development server accepts Origins from that URL and `http://localhost:3000`. The API proxy preserves Host so Gin can validate the browser origin on writes.

For local checks, run `pnpm api:check`, `pnpm format:check`, `pnpm lint`, `pnpm typecheck`, `pnpm test --run` and `pnpm build`. The OpenAPI generator runs in `tools/contract/` with its pinned TypeScript 5 peer while the client uses TypeScript 7.

Compose builds and serves `dist/client` from Gin on the same origin as the API. For browser tests, set `TAOTALE_BACKEND_TARGET=mock-runtime` in the Compose environment. This image includes the deterministic speech provider and FFmpeg. Run it with disposable PostgreSQL and media volumes because the tests create drafts and audio assets. Install Chromium with `pnpm exec playwright install chromium`, then run `pnpm test:e2e`. Set `TAOTALE_E2E_URL` to the test stack’s origin, or use the default `http://127.0.0.1:18080`.

## Library sample cover

Built-in ImageGen created the [sample cover](src/features/library/hangzhou-tea.webp).

### Generation prompt

> Photorealistic editorial landscape cover photograph for a calm modern language-reading library card, 1.85:1 horizontal composition. Early morning in a quiet tea room in Hangzhou, seen from a dark walnut table beside an open traditional lattice window. A single celadon teacup and saucer at lower left, jasmine petals, and softly blurred purple irises with green leaves just outside the window. Warm ivory window light, muted lilac and sage accents, restrained film grain, natural depth of field, sophisticated travel editorial photography. Main objects within central band for landscape cropping. No people, no text, no lettering, no border, no graphics.
