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
BUILD_REPOSITORY=${BUILD_REPOSITORY:?BUILD_REPOSITORY is required}
BUILD_SHA=${BUILD_SHA:?BUILD_SHA is required}
BUILD_RUN_ID=${BUILD_RUN_ID:?BUILD_RUN_ID is required}
BUILD_SOURCE=${BUILD_SOURCE:?BUILD_SOURCE is required}
ISOLATION=${ISOLATION:?ISOLATION is required}
MAIN_REF=${MAIN_REF:-origin/main}
if [[ -n ${BUILD_RECORD_JSON:-} ]]; then
  printf 'repository JSON is not a build record\n' >&2
  exit 1
fi

if [[ ${ISOLATION} != "isolated" ]]; then
  printf 'candidate isolation must be isolated\n' >&2
  exit 1
fi
if [[ ${BUILD_SOURCE} != "github-actions" ]]; then
  printf 'hand-filled build source is refused\n' >&2
  exit 1
fi
if [[ ${BUILD_SHA} != "${GIT_SHA}" ]]; then
  printf 'hand-filled SHA is refused\n' >&2
  exit 1
fi
if [[ ! ${BUILD_REPOSITORY} =~ ^[^/[:space:]]+/[^/[:space:]]+$ ]]; then
  printf 'BUILD_REPOSITORY must be owner/name\n' >&2
  exit 1
fi
if [[ ! ${BUILD_RUN_ID} =~ ^[1-9][0-9]*$ ]]; then
  printf 'BUILD_RUN_ID must be the Actions run id\n' >&2
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
if [[ ! ${IMAGE_DIGEST} =~ ^[a-z0-9][a-z0-9._:-]*(/[a-z0-9][a-z0-9._:-]*)+@sha256:[0-9a-f]{64}$ ]]; then
  printf 'IMAGE_DIGEST must be the full registry ref, not a hand-filled sha256\n' >&2
  exit 1
fi
if [[ ${IMAGE_DIGEST} == *@sha256:0000000000000000000000000000000000000000000000000000000000000000 ]]; then
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
  printf 'git_sha=%s\nimage_digest=%s\ndockerfile_sha=%s\nrepository=%s\nrun_id=%s\nsource=github-actions\nisolation=isolated\ndeploy=false\nrollback=false\n' \
    "${GIT_SHA}" "${IMAGE_DIGEST}" "${DOCKERFILE_SHA}" "${BUILD_REPOSITORY}" "${BUILD_RUN_ID}" >"${CANDIDATE_RECORD}"
fi
printf 'bound git_sha=%s image_digest=%s isolation=isolated deploy=false rollback=false\n' \
  "${GIT_SHA}" "${IMAGE_DIGEST}"
