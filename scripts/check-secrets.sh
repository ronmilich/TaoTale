#!/bin/sh
set -eu
root=$(git rev-parse --show-toplevel)
export MSYS_NO_PATHCONV=1
docker run --rm -v "$root:/repo:ro" \
    -e GIT_CONFIG_COUNT=1 -e GIT_CONFIG_KEY_0=safe.directory -e GIT_CONFIG_VALUE_0=/repo \
    ghcr.io/gitleaks/gitleaks:v8.30.1 git --redact --no-banner "$@" /repo
