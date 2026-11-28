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
    golang:1.27.4 sh -ec 'go test -count=1 -v ./internal/persistence -run "^TestPostgresMigrateAndSeed$"
                          go test -count=1 -v ./internal/httpapi -run "^TestLibraryPostgres$"
                          go test -count=1 -v ./internal/httpapi -run "^TestCoverPostgres$"
                          go test -count=1 -v ./internal/httpapi -run "^TestGenerationPostgres$"
                          go test -count=1 -v ./internal/httpapi -run "^TestVocabulary(Postgres|ScalePostgres)$"
                          apt-get update -qq
                          DEBIAN_FRONTEND=noninteractive apt-get install -y -qq --no-install-recommends ffmpeg >/dev/null
                          go test -count=1 -v ./internal/httpapi -run "^TestPreviewAudioPostgres$"
                          go test -count=1 -v ./internal/httpapi -run "^TestWorkingRevisionPostgres$"
                          go test -count=1 -v ./internal/httpapi -run "^TestVocabularyMatchingPostgres$"
                          go test -count=1 -v ./internal/generation -run "^TestWorkerPostgres$"'

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
            (SELECT jsonb_agg(i ORDER BY id) FROM public.library_items AS i),
            (SELECT jsonb_agg(r ORDER BY id) FROM public.working_revisions AS r),
            (SELECT jsonb_agg(i ORDER BY id) FROM public.media_images AS i),
            (SELECT jsonb_agg(it ORDER BY item_id, topic_id) FROM public.item_topics AS it),
            (SELECT jsonb_agg(v ORDER BY id) FROM public.vocabulary_entries AS v),
            (SELECT jsonb_agg(vr ORDER BY vocabulary_entry_id, library_item_id) FROM public.vocabulary_references AS vr),
            (SELECT jsonb_agg(vt ORDER BY vocabulary_entry_id, topic_id) FROM public.vocabulary_topics AS vt),
            (SELECT jsonb_agg(g ORDER BY id) FROM public.audio_generations AS g),
            (SELECT jsonb_agg(a ORDER BY id) FROM public.audio_assets AS a),
            (SELECT jsonb_agg(r ORDER BY user_id, operation, key) FROM public.idempotency_records AS r),
            (SELECT jsonb_agg(m ORDER BY version) FROM public.schema_migrations AS m)
        );"
}
original=$(snapshot "$source_db")
restored=$(snapshot "$restored_db")
[[ "$original" == "$restored" ]]
echo 'Restored catalog, items, working revisions, vocabulary, topic associations, audio generations, assets, receipts and migration history match the source database.'
