#!/bin/sh
set -e

# Docker runs as root; the workspace is owned by the runner user.
# Mark it safe so go/git operations work inside the container.
git config --global --add safe.directory "${GITHUB_WORKSPACE:-/github/workspace}" 2>/dev/null || true

target="${INPUT_TARGET:-check}"

# shellcheck disable=SC2086
exec make -f /opt/go-tooling/tools.mk ${target} \
  GO_PKGS="${INPUT_PACKAGES:-./...}" \
  ${INPUT_ARGS}
