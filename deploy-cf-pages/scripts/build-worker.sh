#!/bin/bash
# Build a Worker site for Cloudflare deployment
#
# Usage:
#   build-worker.sh
#
# Expects npm ci to have already run.
# Runs make minify if the target exists.
# Copies worker source, config, and optional directories into .deploy/ for push.

set -o errexit
set -o nounset
set -o pipefail

exec 3>&1 4>&2

if [[ -f Makefile ]]; then grep -q '^minify:' Makefile && make minify; fi

mkdir -p .deploy
cp wrangler.jsonc .deploy/
cp package.json package-lock.json tsconfig.json .deploy/
cp -r src .deploy/src
if [[ -d public ]]; then cp -r public .deploy/public; fi
if [[ -d migrations ]]; then cp -r migrations .deploy/migrations; fi

exit 0
