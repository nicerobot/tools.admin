#!/bin/bash
# Build .deploy/ directory with generated image assets and manifest
#
# Usage:
#   build.sh
#
# Copies all generated output (excluding source files, build config,
# and dotfiles) into .deploy/ and generates manifest.json.

set -o errexit
set -o nounset
set -o pipefail

exec 3>&1 4>&2

mkdir -p .deploy

# Copy generated asset directories
for dir in logo favicon platform; do
  [[ -d "${dir}" ]] && cp -r "${dir}" ".deploy/${dir}" || true
done

# Generate manifest.json listing every published file
cd .deploy

FILES=()
while IFS= read -r -d '' file; do
  FILES+=("${file}")
done < <(find . -type f ! -name manifest.json -print0 | sort -z)

{
  echo '{'
  echo '  "generated": "'$(date -u +%Y-%m-%dT%H:%M:%SZ)'",'
  echo '  "files": ['

  for i in "${!FILES[@]}"; do
    # Strip leading ./
    filepath="${FILES[$i]#./}"
    comma=','
    [[ $i -eq $(( ${#FILES[@]} - 1 )) ]] && comma=''
    echo "    \"${filepath}\"${comma}"
  done

  echo '  ]'
  echo '}'
} > manifest.json

echo "::notice::Staged ${#FILES[@]} files for gh-pages"

cd ..

exit 0
