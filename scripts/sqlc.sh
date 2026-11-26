#!/bin/sh
set -eu
root=$(git rev-parse --show-toplevel)
export MSYS_NO_PATHCONV=1
docker run --rm -v "$root/backend:/src" -w /src/db sqlc/sqlc:1.31.1 "$@"
