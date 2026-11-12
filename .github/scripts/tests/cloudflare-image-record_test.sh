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
ok "image workflow records a build and does not deploy"

set +e
DEPLOY=true bash "${SCRIPT}" >/tmp/cloudflare-image-record-deploy.out 2>/tmp/cloudflare-image-record-deploy.err
status=$?
set -e
[[ ${status} -eq 2 ]] || fail "deploy request exited ${status}"
ok "deploy request exits 2"

sha=$(git -C "${ROOT}" rev-parse HEAD)
digest="registry.example/needtoken/storefront-staging@sha256:$(printf 'b%.0s' {1..64})"
out=$(
  cd "${ROOT}"
  BUILD_REPOSITORY=NeedTokenAi/needtoken-storefront \
    BUILD_SHA="${sha}" \
    BUILD_RUN_ID=43175381072 \
    BUILD_SOURCE=github-actions \
    IMAGE_REF="${digest}" \
    bash "${SCRIPT}"
)
grep -q "repository=NeedTokenAi/needtoken-storefront" <<<"${out}" || fail "repository was not recorded"
grep -q "sha=${sha}" <<<"${out}" || fail "sha was not recorded"
grep -q "run_id=43175381072" <<<"${out}" || fail "run id was not recorded"
grep -q "digest=${digest}" <<<"${out}" || fail "digest was not recorded"
grep -q "source=github-actions" <<<"${out}" || fail "source was not recorded"
grep -q "deploy=false" <<<"${out}" || fail "deploy was not refused"
ok "record keeps repository, sha, run id, and digest"

set +e
(
  cd "${ROOT}"
  BUILD_REPOSITORY=NeedTokenAi/needtoken-storefront \
    BUILD_SHA="${sha}" \
    BUILD_RUN_ID=43175381072 \
    IMAGE_REF="sha256:${digest##*sha256:}" \
    bash "${SCRIPT}" >/tmp/cloudflare-image-record-bare.out 2>/tmp/cloudflare-image-record-bare.err
)
status=$?
set -e
[[ ${status} -eq 1 ]] || fail "bare sha256 exited ${status}"
ok "bare sha256 is refused"
