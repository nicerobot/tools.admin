#!/bin/sh
set -e

# Docker runs as root; the workspace is owned by the runner user.
# Mark it safe so git operations work inside the container.
git config --global --add safe.directory "${GITHUB_WORKSPACE:-/github/workspace}"

command="${INPUT_COMMAND:?INPUT_COMMAND is required}"

case "${command}" in
  snapshot)
    exec admin-tools snapshot \
      --owner "${INPUT_OWNER:?INPUT_OWNER is required for snapshot}" \
      --settings-path "${INPUT_SETTINGS_PATH:-.github}"
    ;;
  create-pr)
    exec admin-tools create-pr \
      --settings-path "${INPUT_SETTINGS_PATH:-.github}" \
      --branch "${INPUT_BRANCH:-safe-settings/snapshot}" \
      --base "${INPUT_BASE:-main}"
    ;;
  cleanup-runs)
    # owner is optional: when omitted the tool auto-detects the current
    # repository from GITHUB_REPOSITORY, which GitHub injects into the run.
    args=""
    [ -n "${INPUT_OWNER:-}" ] && args="$args --owner ${INPUT_OWNER}"
    [ -n "${INPUT_CLEANUP_REPO:-}" ] && args="$args --repo ${INPUT_CLEANUP_REPO}"
    [ -n "${INPUT_CLEANUP_DAYS:-}" ] && args="$args --days ${INPUT_CLEANUP_DAYS}"
    [ -n "${INPUT_CLEANUP_KEEP:-}" ] && args="$args --keep ${INPUT_CLEANUP_KEEP}"
    [ "${INPUT_CLEANUP_DRY_RUN:-false}" = "true" ] && args="$args --dry-run"
    exec admin-tools cleanup-runs $args
    ;;
  *)
    echo "Unknown command: ${command}" >&2
    exit 1
    ;;
esac
