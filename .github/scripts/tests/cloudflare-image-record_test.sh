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

[[ -f ${WORKFLOW} ]] || fail "image workflow missing"
grep -q 'workflow_dispatch' "${WORKFLOW}" || fail "image workflow must be manual"
if grep -q 'pull_request:' "${WORKFLOW}"; then
  fail "image workflow runs on pull request"
fi
if grep -Eq 'wrangler deploy|deploy-cloudrun|gcloud ' "${WORKFLOW}"; then
  fail "image workflow can publish production"
fi
grep -q 'containers build' "${WORKFLOW}" || fail "image workflow does not build the container"
if grep -Eq 'image_digest:|git_sha:' "${WORKFLOW}"; then
  fail "image workflow accepts a hand-filled digest"
fi
grep -q 'tag="needtoken-storefront:${GITHUB_SHA}"' "${WORKFLOW}" || fail "tag must be the production image and source SHA"
if grep -q 'storefront-staging' "${WORKFLOW}"; then
  fail "image workflow uses a staging image name"
fi
grep -q 'containers build -p -t "${tag}"' "${WORKFLOW}" || fail "push must use the immutable tag"
grep -q 'CLOUDFLARE_ACCOUNT_ID: ${{ secrets.CLOUDFLARE_ACCOUNT_ID }}' "${WORKFLOW}" || fail "account id must stay a secret"
if grep -q 'vars:' "${WORKFLOW}"; then
  fail "account id was turned into a variable"
fi
grep -q 'actions/upload-artifact@ea165f8d65b6e75b540449e92b4886f43607fa02' "${WORKFLOW}" || fail "build record is not uploaded"
grep -q 'name: image-build-record' "${WORKFLOW}" || fail "artifact name missing"
grep -q 'build-record.txt' "${WORKFLOW}" || fail "build record path missing"
grep -q 'contents: read' "${WORKFLOW}" || fail "contents permission"
grep -q 'actions: write' "${WORKFLOW}" || fail "artifact upload permission"
ok "image workflow records a production digest and does not deploy"

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
