#!/usr/bin/env bash
set -euo pipefail

ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)
SCRIPT="${ROOT}/.github/scripts/cloudflare-image-record.sh"
WORKFLOW="${ROOT}/.github/workflows/cloudflare-image.yml"

fail() {
  printf 'not ok - %s\n' "$1" >&2
  exit 1
}

ok() {
  printf 'ok - %s\n' "$1"
}

if [[ -e ${WORKFLOW} ]]; then
  fail "image workflow must stay deleted; Workers Builds publishes the Dockerfile"
fi
if [[ -e ${ROOT}/.github/workflows/deploy-cloudrun.yml ]]; then
  fail "cloud run workflow must stay deleted"
fi
ok "image and cloud run workflows are not a second publisher"

set +e
DEPLOY=true bash "${SCRIPT}" >/tmp/cloudflare-image-record-deploy.out 2>/tmp/cloudflare-image-record-deploy.err
status=$?
set -e
[[ ${status} -eq 2 ]] || fail "deploy request exited ${status}"
ok "deploy request exits 2"

sha=$(git -C "${ROOT}" rev-parse HEAD)
digest="sha256:$(printf 'b%.0s' {1..64})"
summary=$(mktemp)
record=$(mktemp)
out=$(
  cd "${ROOT}"
  BUILD_REPOSITORY=NeedTokenAi/needtoken-storefront \
    BUILD_SHA="${sha}" \
    BUILD_RUN_ID=43175381072 \
    BUILD_SOURCE=github-actions \
    IMAGE_DIGEST="${digest}" \
    GITHUB_STEP_SUMMARY="${summary}" \
    BUILD_RECORD_PATH="${record}" \
    bash "${SCRIPT}"
)
expected=$(printf 'repository=NeedTokenAi/needtoken-storefront\ndigest=%s\nsha=%s\nrun_id=43175381072\n' "${digest}" "${sha}")
[[ ${out} == "${expected}" ]] || fail "stdout is not the four-field record"
[[ $(<"${summary}") == "${expected}" ]] || fail "job summary is not the four-field record"
[[ $(<"${record}") == "${expected}" ]] || fail "artifact is not the four-field record"
if grep -Eq 'registry\.cloudflare\.com|account' "${summary}" "${record}"; then
  fail "record contains an account id"
fi
ok "record keeps repository, sha256 digest, source SHA, and run id"

set +e
(
  cd "${ROOT}"
  BUILD_REPOSITORY=NeedTokenAi/needtoken-storefront \
    BUILD_SHA="${sha}" \
    BUILD_RUN_ID=43175381072 \
    IMAGE_DIGEST="registry.example/needtoken/needtoken-storefront@${digest}" \
    bash "${SCRIPT}" >/tmp/cloudflare-image-record-ref.out 2>/tmp/cloudflare-image-record-ref.err
)
status=$?
set -e
[[ ${status} -eq 1 ]] || fail "registry address exited ${status}"
ok "registry address is refused"
