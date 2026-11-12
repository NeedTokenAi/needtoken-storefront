#!/usr/bin/env bash
# Record the sha256 digest from this GitHub Actions run.
# The record is the repository, digest, source SHA, and run id. It has no account id.
# A registry address is not a build record. This script cannot publish a Worker.
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
IMAGE_DIGEST=${IMAGE_DIGEST:?IMAGE_DIGEST is required}
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
if [[ ${IMAGE_DIGEST} == */* || ${IMAGE_DIGEST} == *@* || ${IMAGE_DIGEST} == *registry.cloudflare.com* ]]; then
  printf 'digest must not include a registry address\n' >&2
  exit 1
fi
if [[ ! ${IMAGE_DIGEST} =~ ^sha256:[0-9a-f]{64}$ ]]; then
  printf 'IMAGE_DIGEST must be the sha256 digest\n' >&2
  exit 1
fi

record=$(printf 'repository=%s\ndigest=%s\nsha=%s\nrun_id=%s\n' \
  "${BUILD_REPOSITORY}" "${IMAGE_DIGEST}" "${BUILD_SHA}" "${BUILD_RUN_ID}")

if [[ -n ${GITHUB_OUTPUT:-} ]]; then
  printf '%s\n' "${record}" >>"${GITHUB_OUTPUT}"
fi
if [[ -n ${GITHUB_STEP_SUMMARY:-} ]]; then
  printf '%s\n' "${record}" >>"${GITHUB_STEP_SUMMARY}"
fi
record_path=${BUILD_RECORD_PATH:-build-record.txt}
printf '%s\n' "${record}" >"${record_path}"
printf '%s\n' "${record}"
