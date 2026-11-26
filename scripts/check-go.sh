#!/bin/sh
set -eu
unformatted=$(gofmt -l ./cmd ./db ./internal)
if [ -n "$unformatted" ]; then
    printf 'Unformatted Go files:\n%s\n' "$unformatted" >&2
    exit 1
fi
