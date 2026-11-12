#!/usr/bin/env bash
# Record a registry digest produced by this GitHub Actions run.
# A typed sha256 or a JSON file is not a build record. This script cannot publish a Worker.
set -euo pipefail

if [[ ${DEPLOY:-false} == "true" || ${DEPLOY:-false} == "1" ]]; then
  printf 'image workflow refuses to publish a worker\n' >&2
  exit 2
fi
if [[ -n ${BUILD_RECORD_JSON:-} ]]; then
  printf 'repository JSON is not a build record\n' >&2
  exit 1
fi

BUILD_REPOSITORY=${BUILD_REPOSITORY:?BUILD_REPOSITORY is required}
BUILD_SHA=${BUILD_SHA:?BUILD_SHA is required}
BUILD_RUN_ID=${BUILD_RUN_ID:?BUILD_RUN_ID is required}
IMAGE_REF=${IMAGE_REF:?IMAGE_REF is required}
BUILD_SOURCE=${BUILD_SOURCE:-github-actions}

if [[ ${BUILD_SOURCE} != "github-actions" ]]; then
  printf 'hand-filled build source is refused\n' >&2
  exit 1
fi
head_sha=$(git rev-parse HEAD)
if [[ ${BUILD_SHA} != "${head_sha}" ]]; then
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
if [[ ! ${IMAGE_REF} =~ ^[a-z0-9][a-z0-9._:-]*(/[a-z0-9][a-z0-9._:-]*)+@sha256:[0-9a-f]{64}$ ]]; then
  printf 'IMAGE_REF must be the full registry ref from ctx.container.images.dujiao\n' >&2
  exit 1
fi

if [[ -n ${GITHUB_OUTPUT:-} ]]; then
  {
    printf 'repository=%s\n' "${BUILD_REPOSITORY}"
    printf 'sha=%s\n' "${BUILD_SHA}"
    printf 'run_id=%s\n' "${BUILD_RUN_ID}"
    printf 'digest=%s\n' "${IMAGE_REF}"
  } >>"${GITHUB_OUTPUT}"
fi
printf 'repository=%s\nsha=%s\nrun_id=%s\ndigest=%s\nsource=github-actions\ndeploy=false\n' \
  "${BUILD_REPOSITORY}" "${BUILD_SHA}" "${BUILD_RUN_ID}" "${IMAGE_REF}"
