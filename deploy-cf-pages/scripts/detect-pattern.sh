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

[[ -f src/index.ts ]] && [[ -f tsconfig.json ]] && [[ -f package.json ]] && IS_WORKER=true || true

for HUGO_CONFIG in hugo.json hugo.toml hugo.yaml config.toml config.yaml; do
  [[ -f "${HUGO_CONFIG}" ]] && { IS_HUGO=true; break; } || true
done

${IS_WORKER} && ${IS_HUGO} && { echo 'hugo-worker'; exit 0; } || true
${IS_WORKER} && { echo 'worker'; exit 0; } || true

echo 'hugo'

exit 0
