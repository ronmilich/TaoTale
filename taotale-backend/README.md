# TaoTale backend

The scaffold runs a Go/Gin HTTP service and PostgreSQL with Docker Compose. The API listens on the host's loopback interface. PostgreSQL is reachable inside the Compose network. Named volumes hold database files and media across restarts.

## Run locally

From this directory, copy `.env.example` to `.env` on first setup. Set `POSTGRES_PASSWORD` to a generated local password before starting PostgreSQL. Keep that value for subsequent starts with the same database volume. Set `PORT` to choose a host port (default 8080).

```sh
docker compose up --build -d
```

Open `http://127.0.0.1:8080/health/live` to check the process or `/health/ready` to check the database connection and writable media directory. Both routes are also available under `/api/v1`. The response contract is in [api/openapi.yaml](api/openapi.yaml).

Use `docker compose logs api` to inspect the service and `docker compose down` to stop the stack while keeping both volumes.
