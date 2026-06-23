#!/bin/bash
# Build a Hugo + Worker hybrid site for Cloudflare deployment
#
# Usage:
#   build-hugo-worker.sh
#
# Expects both Hugo and npm ci to have already run.
# Hugo output in public/ is included as Workers Static Assets.
# Copies worker source, config, Hugo output, and optional directories
# into .deploy/ for push.

set -o errexit
set -o nounset
set -o pipefail

exec 3>&1 4>&2

mkdir -p .deploy
cp wrangler.jsonc .deploy/
cp package.json package-lock.json tsconfig.json .deploy/
cp -r src .deploy/src
cp -r public .deploy/public
[[ -d migrations ]] && cp -r migrations .deploy/migrations || true

exit 0
