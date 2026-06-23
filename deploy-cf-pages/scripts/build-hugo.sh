#!/bin/bash
# Build a Hugo-only site for Cloudflare deployment
#
# Usage:
#   build-hugo.sh
#
# Expects Hugo to have already run (public/ exists).
# Copies wrangler config and public/ into .deploy/ for push.

set -o errexit
set -o nounset
set -o pipefail

exec 3>&1 4>&2

mkdir -p .deploy
cp wrangler.jsonc .deploy/
mv public .deploy/public

exit 0
