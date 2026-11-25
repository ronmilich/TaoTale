# TaoTale backend

The Go/Gin service and PostgreSQL run in Docker Compose. The API publishes to `127.0.0.1`. The API reaches PostgreSQL as `db:5432` on the Compose network. Use `docker compose exec db psql -U taotale -d taotale` for local SQL access.

## Run locally

From [infra](../infra), copy `.env.example` to the gitignored `.env` on first setup. Set `POSTGRES_PASSWORD` to a generated local password before starting PostgreSQL. Keep that value for subsequent starts with the same database volume. PostgreSQL reads it when it initializes a fresh volume. Set `PORT` to choose the API's host port (default 8080).

```sh
docker compose up --build --wait
```

Open `http://127.0.0.1:8080/health/live` to check the process or `/health/ready` to check the database connection and writable media directory. Both routes are also available under `/api/v1`. The response contract is in [api/openapi.yaml](api/openapi.yaml).

Standalone Go runs default to `127.0.0.1:8080`. Compose binds the Go process to the private address of its `api` service and publishes the host port on loopback. Every request must use a loopback Host. For a write with an Origin header, the origin must match the request's HTTP origin or the optional `TAOTALE_DEV_ORIGIN` in `infra/.env` (for example, `http://localhost:3000`). Configure it only when the local dev proxy uses a different Host. The normal Vite proxy preserves the browser's Host.

API logs contain the server-generated request ID, method, route template, status and duration. JSON handlers use the shared decoder for one UTF-8 object up to 100 KiB with known fields. PostgreSQL or media failures return a sanitized `503 SERVICE_UNAVAILABLE` envelope with the same request ID as the response header.

PostgreSQL 17.11 stores data in the `taotale-backend_postgres` named volume. The API runs as UID 10001 and initializes its absolute `MEDIA_ROOT` at `/var/lib/taotale/media` in the `taotale-backend_media` named volume. The root, `staging/`, `temporary/audio/`, `audio/`, `temporary/images/`, and `images/` use owner-only directory permissions (`0700`).

Use `docker compose logs api` to inspect the service. Use `docker compose stop` to stop it, or `docker compose down` to remove containers while keeping both volumes. Never use `docker compose down -v` for routine shutdown. It deletes the database and media volumes.
