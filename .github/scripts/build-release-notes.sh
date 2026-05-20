#!/usr/bin/env bash
# Builds release-notes.md for a version tag push (expects fetch-depth: 0 checkout).
set -euo pipefail

: "${GITHUB_REPOSITORY:?}"
: "${GITHUB_REF:?}"
: "${GITHUB_SHA:?}"

REGISTRY="${REGISTRY:-ghcr.io}"
REPO_LC="${GITHUB_REPOSITORY,,}"
CURRENT_TAG="${GITHUB_REF#refs/tags/}"
IMAGE="${REGISTRY}/${REPO_LC}:${CURRENT_TAG}"
IMAGE_SHA="${REGISTRY}/${REPO_LC}:sha-${GITHUB_SHA:0:7}"
REPO_URL="https://github.com/${GITHUB_REPOSITORY}"

PREV_TAG=""
while IFS= read -r tag; do
  if [[ "${tag}" == "${CURRENT_TAG}" ]]; then
    break
  fi
  PREV_TAG="${tag}"
done < <(git tag -l 'v*' --sort=v:refname)

changelog() {
  local range=()
  if [[ -n "${PREV_TAG}" ]]; then
    range=("${PREV_TAG}..${CURRENT_TAG}")
  fi

  local entries
  if [[ ${#range[@]} -gt 0 ]]; then
    entries="$(git log "${range[@]}" \
      --pretty=format:'- %s ([%h]('"${REPO_URL}"'/commit/%H))' \
      --no-merges 2>/dev/null || true)"
  else
    entries="$(git log \
      --pretty=format:'- %s ([%h]('"${REPO_URL}"'/commit/%H))' \
      --no-merges 2>/dev/null || true)"
  fi

  if [[ -z "${entries}" ]]; then
    echo "_No commits since the previous tag (empty range or merge-only)._"
  else
    printf '%s\n' "${entries}"
  fi
}

{
  echo "## Container image"
  echo
  echo "| Tag | Image |"
  echo "|-----|-------|"
  echo "| \`${CURRENT_TAG}\` | \`${IMAGE}\` |"
  echo "| \`sha-${GITHUB_SHA:0:7}\` | \`${IMAGE_SHA}\` |"
  echo
  echo "### Пример запуска"
  echo
  echo '```bash'
  echo "docker run --rm \\"
  echo "  -e FB_APP_ID=... \\"
  echo "  -e FB_USER_ACCESS_TOKEN=... \\"
  echo "  -e FBINSTANT_ZIP_PATH=/data/game.zip \\"
  echo "  -e CONFIG_JSON_FILE=/data/config.json \\"
  echo "  -v \"\$(pwd)/build:/data:ro\" \\"
  echo "  ${IMAGE}"
  echo '```'
  echo
  echo "## Changelog"
  echo
  if [[ -n "${PREV_TAG}" ]]; then
    echo "Changes since [\`${PREV_TAG}\`](${REPO_URL}/releases/tag/${PREV_TAG}):"
  else
    echo "Initial release:"
  fi
  echo
  changelog
  echo
  if [[ -n "${PREV_TAG}" ]]; then
    echo "**Full diff:** ${REPO_URL}/compare/${PREV_TAG}...${CURRENT_TAG}"
  else
    echo "**Tag commit:** ${REPO_URL}/commit/${GITHUB_SHA}"
  fi
} > release-notes.md
