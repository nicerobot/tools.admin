#!/bin/bash
# Detect the build pattern for a Cloudflare deployment
#
# Usage:
#   detect-pattern.sh
#
# Outputs one of: hugo-worker, worker, hugo
# Detection logic:
#   hugo-worker: worker files AND Hugo config both present
#   worker: src/index.ts + tsconfig.json + package.json
#   hugo: everything else (default)

set -o errexit
set -o nounset
set -o pipefail

exec 3>&1 4>&2

IS_WORKER=false
IS_HUGO=false

if [[ -f src/index.ts ]]; then [[ -f tsconfig.json ]] && [[ -f package.json ]] && IS_WORKER=true; fi

for HUGO_CONFIG in hugo.json hugo.toml hugo.yaml config.toml config.yaml; do
  if [[ -f "${HUGO_CONFIG}" ]]; then
    IS_HUGO=true
    break
  fi
done

if ${IS_WORKER} && ${IS_HUGO}; then
  echo 'hugo-worker'
  exit 0
fi
if ${IS_WORKER}; then
  echo 'worker'
  exit 0
fi

echo 'hugo'

exit 0
