#!/bin/bash
# Push .deploy/ directory to cf-pages branch for Cloudflare deployment
#
# Usage:
#   push-to-cf-pages.sh <github-token> <github-repository>
# Where:
#   github-token is the access token for pushing
#   github-repository is the full repo name (e.g., org/repo)

set -o errexit
set -o nounset
set -o pipefail

exec 3>&1 4>&2

GITHUB_TOKEN=${1:?missing github-token argument}
GITHUB_REPO=${2:?missing github-repository argument}

cd .deploy
git init
git config user.name 'github-actions[bot]'
git config user.email 'github-actions[bot]@users.noreply.github.com'
git add -A
git commit -m "Deploy $(date -u +%Y-%m-%dT%H:%M:%SZ)"
git remote add origin "https://x-access-token:${GITHUB_TOKEN}@github.com/${GITHUB_REPO}.git"
git push origin HEAD:cf-pages --force

exit 0
