#!/usr/bin/env bash
set -euo pipefail
root=$(git rev-parse --show-toplevel)
export MSYS_NO_PATHCONV=1
source_db=
restored_db=
test_network=
cleanup() {
    for container in "$source_db" "$restored_db"; do
        if [[ -n "$container" ]]; then docker rm -fv "$container" >/dev/null; fi
    done
    if [[ -n "$test_network" ]]; then docker network rm "$test_network" >/dev/null; fi
}
trap cleanup EXIT

fresh_postgres() {
    docker run -d "$@" -e POSTGRES_HOST_AUTH_METHOD=trust \
        -e POSTGRES_DB=taotale_test --mount type=volume,target=/var/lib/postgresql/data postgres:17.11
}

wait_postgres() {
    for ((attempt = 0; attempt < 60; attempt++)); do
        if docker exec "$1" pg_isready -h 127.0.0.1 -U postgres -d taotale_test >/dev/null; then
            return 0
        fi
        sleep 1
    done
    echo 'Disposable PostgreSQL did not start.' >&2
    return 1
}

test_network=$(docker network create "taotale-db-check-$$-$(date +%s)")
source_db=$(fresh_postgres --network "$test_network" --network-alias postgres)
wait_postgres "$source_db"
docker run --rm --network "$test_network" -v "$root/backend:/src:ro" -w /src \
    -e 'TAOTALE_TEST_DATABASE_URL=postgres://postgres@postgres/taotale_test?sslmode=disable' \
    golang:1.27.4 go test -count=1 -v ./internal/persistence -run '^TestPostgresMigrateAndSeed$'

restored_db=$(fresh_postgres --network none)
wait_postgres "$restored_db"
docker exec "$source_db" pg_dump -U postgres -d taotale_test -Fc |
    docker exec -i "$restored_db" pg_restore -U postgres -d taotale_test --exit-on-error

snapshot() {
    docker exec "$1" psql -XAtq -v ON_ERROR_STOP=1 -U postgres -d taotale_test -c "
        SET TIME ZONE 'UTC';
        SELECT jsonb_build_array(
            (SELECT jsonb_agg(u ORDER BY id) FROM public.users AS u),
            (SELECT jsonb_agg(l ORDER BY code) FROM public.learning_languages AS l),
            (SELECT jsonb_agg(t ORDER BY id) FROM public.topics AS t),
            (SELECT jsonb_agg(m ORDER BY version) FROM public.schema_migrations AS m)
        );"
}
original=$(snapshot "$source_db")
restored=$(snapshot "$restored_db")
[[ "$original" == "$restored" ]]
echo 'Restored catalog, edited metadata and migration history match the source database.'
