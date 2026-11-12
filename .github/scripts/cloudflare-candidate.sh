#!/usr/bin/env bash
# Bind a main commit and image digest for a later, separate decision.
# This script cannot deploy. It does not call wrangler, gcloud, or any registry.
set -euo pipefail

if [[ ${DEPLOY:-false} == "true" || ${DEPLOY:-false} == "1" ]]; then
  printf 'cloudflare candidate refuses to deploy\n' >&2
  exit 2
fi

GIT_SHA=${GIT_SHA:?GIT_SHA is required}
IMAGE_DIGEST=${IMAGE_DIGEST:?IMAGE_DIGEST is required}
MAIN_REF=${MAIN_REF:-origin/main}

if [[ ! ${GIT_SHA} =~ ^[0-9a-f]{40}$ ]]; then
  printf 'GIT_SHA must be a 40 character commit\n' >&2
  exit 1
fi
if [[ ! ${IMAGE_DIGEST} =~ ^sha256:[0-9a-f]{64}$ ]]; then
  printf 'IMAGE_DIGEST must be a sha256 digest\n' >&2
  exit 1
fi
if ! git rev-parse --verify --quiet "${MAIN_REF}^{commit}" >/dev/null; then
  printf 'main ref %s is not available\n' "${MAIN_REF}" >&2
  exit 1
fi
if ! git merge-base --is-ancestor "${GIT_SHA}" "${MAIN_REF}"; then
  printf 'GIT_SHA %s is not on %s\n' "${GIT_SHA}" "${MAIN_REF}" >&2
  exit 1
fi

if [[ -n ${CANDIDATE_RECORD:-} ]]; then
  printf 'git_sha=%s\nimage_digest=%s\ndeploy=false\n' "${GIT_SHA}" "${IMAGE_DIGEST}" >"${CANDIDATE_RECORD}"
fi
printf 'bound git_sha=%s image_digest=%s deploy=false\n' "${GIT_SHA}" "${IMAGE_DIGEST}"
