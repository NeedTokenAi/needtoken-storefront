#!/usr/bin/env bash
# Bind a checked-out commit to the digest's build source.
# A digest shape alone is not enough. This script cannot deploy and cannot roll back.
set -euo pipefail

if [[ ${DEPLOY:-false} == "true" || ${DEPLOY:-false} == "1" ]]; then
  printf 'cloudflare candidate refuses to deploy\n' >&2
  exit 2
fi

GIT_SHA=${GIT_SHA:?GIT_SHA is required}
IMAGE_DIGEST=${IMAGE_DIGEST:?IMAGE_DIGEST is required}
DOCKERFILE_SHA=${DOCKERFILE_SHA:?DOCKERFILE_SHA is required to tie the digest to the build source}
ISOLATION=${ISOLATION:?ISOLATION is required}
MAIN_REF=${MAIN_REF:-origin/main}

if [[ ${ISOLATION} != "isolated" ]]; then
  printf 'candidate isolation must be isolated\n' >&2
  exit 1
fi
if [[ ! ${GIT_SHA} =~ ^[0-9a-f]{40}$ ]]; then
  printf 'GIT_SHA must be a 40 character commit\n' >&2
  exit 1
fi
if [[ ! ${DOCKERFILE_SHA} =~ ^[0-9a-f]{40}$ ]]; then
  printf 'DOCKERFILE_SHA must be the Dockerfile blob at GIT_SHA\n' >&2
  exit 1
fi
if [[ ! ${IMAGE_DIGEST} =~ ^sha256:[0-9a-f]{64}$ ]]; then
  printf 'IMAGE_DIGEST must be a sha256 digest\n' >&2
  exit 1
fi
if [[ ${IMAGE_DIGEST} == sha256:0000000000000000000000000000000000000000000000000000000000000000 ]]; then
  printf 'placeholder digest stays fail-closed\n' >&2
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
head_sha=$(git rev-parse HEAD)
if [[ ${head_sha} != "${GIT_SHA}" ]]; then
  printf 'GIT_SHA %s is not the checked out source %s\n' "${GIT_SHA}" "${head_sha}" >&2
  exit 1
fi
source_dockerfile=$(git rev-parse "${GIT_SHA}:Dockerfile")
if [[ ${source_dockerfile} != "${DOCKERFILE_SHA}" ]]; then
  printf 'digest is not tied to the Dockerfile at %s\n' "${GIT_SHA}" >&2
  exit 1
fi

if [[ -n ${CANDIDATE_RECORD:-} ]]; then
  printf 'git_sha=%s\nimage_digest=%s\ndockerfile_sha=%s\nisolation=isolated\ndeploy=false\nrollback=false\n' \
    "${GIT_SHA}" "${IMAGE_DIGEST}" "${DOCKERFILE_SHA}" >"${CANDIDATE_RECORD}"
fi
printf 'bound git_sha=%s image_digest=%s isolation=isolated deploy=false rollback=false\n' \
  "${GIT_SHA}" "${IMAGE_DIGEST}"
