#!/usr/bin/env bash
set -euo pipefail

ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)
SCRIPT="${ROOT}/.github/scripts/cloudflare-candidate.sh"
WORKFLOW="${ROOT}/.github/workflows/cloudflare-candidate.yml"
DEPLOY_WORKFLOW="${ROOT}/.github/workflows/deploy-cloudrun.yml"

fail() {
  printf 'not ok - %s\n' "$1" >&2
  exit 1
}

ok() {
  printf 'ok - %s\n' "$1"
}

[[ -f ${DEPLOY_WORKFLOW} ]] || fail "existing Cloud Run workflow must stay"
grep -q 'group: deploy-cloudrun-production' "${DEPLOY_WORKFLOW}" || fail "Cloud Run concurrency group changed"

[[ -f ${WORKFLOW} ]] || fail "candidate workflow missing"
if grep -Eq 'workflow_run|id-token:|wrangler|gcloud|deploy-cloudrun-production' "${WORKFLOW}"; then
  fail "candidate workflow can reach production"
fi
grep -q 'contents: read' "${WORKFLOW}" || fail "candidate workflow permissions"
grep -q 'workflow_dispatch' "${WORKFLOW}" || fail "candidate workflow must bind on dispatch"
if grep -Eq 'uses:[[:space:]]+[^@]+@v[0-9]' "${WORKFLOW}"; then
  fail "candidate actions must be pinned to commit SHAs"
fi
grep -q 'actions/checkout@11bd71901bbe5b1630ceea73d27597364c9af683' "${WORKFLOW}" || fail "checkout pin"
grep -q 'actions/setup-node@49933ea5288caeca8642d1e84afbd3f7d6820020' "${WORKFLOW}" || fail "node pin"
if grep -Eq 'image_digest:|git_sha:' "${WORKFLOW}"; then
  fail "candidate dispatch still accepts a hand-filled digest"
fi
IMAGE_WORKFLOW="${ROOT}/.github/workflows/cloudflare-image.yml"
[[ -f ${IMAGE_WORKFLOW} ]] || fail "image workflow missing"
grep -q 'workflow_dispatch' "${IMAGE_WORKFLOW}" || fail "image workflow must be manual"
if grep -q 'pull_request:' "${IMAGE_WORKFLOW}"; then
  fail "image workflow runs on pull request"
fi
if grep -Eq 'wrangler deploy|deploy-cloudrun|gcloud ' "${IMAGE_WORKFLOW}"; then
  fail "image workflow can publish production"
fi
grep -q 'containers build' "${IMAGE_WORKFLOW}" || fail "image workflow does not build the container"
(
  cd "${ROOT}"
  node --input-type=module -e 'import {readFileSync} from "node:fs"; const cfg=JSON.parse(readFileSync("cloudflare/wrangler.staging.jsonc","utf8")); if (cfg.name !== "needtoken-storefront-staging" || cfg.routes || cfg.workers_dev !== false || cfg.preview_urls !== false) process.exit(1);'
) || fail "staging worker config is not isolated from production routes"
ok "candidate workflow cannot deploy"

set +e
DEPLOY=true bash "${SCRIPT}" >/tmp/cloudflare-candidate-deploy.out 2>/tmp/cloudflare-candidate-deploy.err
status=$?
set -e
[[ ${status} -eq 2 ]] || fail "deploy request exited ${status}"
ok "deploy request exits 2"

workdir=$(mktemp -d)
trap 'rm -rf -- "${workdir}"' EXIT
git -C "${workdir}" init -q
git -C "${workdir}" config user.email "candidate@example.com"
git -C "${workdir}" config user.name "candidate"
printf 'one\n' >"${workdir}/README"
printf 'FROM scratch\n' >"${workdir}/Dockerfile"
git -C "${workdir}" add README Dockerfile
git -C "${workdir}" commit -q -m "one"
main_sha=$(git -C "${workdir}" rev-parse HEAD)
dockerfile_sha=$(git -C "${workdir}" rev-parse HEAD:Dockerfile)
base_branch=$(git -C "${workdir}" branch --show-current)
git -C "${workdir}" checkout -q -b feature
printf 'two\n' >"${workdir}/README"
git -C "${workdir}" commit -q -am "two"
feature_sha=$(git -C "${workdir}" rev-parse HEAD)
git -C "${workdir}" checkout -q "${base_branch}"
git -C "${workdir}" update-ref refs/heads/main "${main_sha}"

record="${workdir}/record"
full_ref="registry.example/needtoken/storefront-staging@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
zero_ref="registry.example/needtoken/storefront-staging@sha256:0000000000000000000000000000000000000000000000000000000000000000"
bare_ref="sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
set +e
(
  cd "${workdir}"
  GIT_SHA="${main_sha}" \
    IMAGE_DIGEST="${bare_ref}" \
    MAIN_REF="refs/heads/main" \
    bash "${SCRIPT}"
)
status=$?
set -e
[[ ${status} -eq 1 ]] || fail "digest shape alone exited ${status}"
ok "digest shape alone stays fail-closed"

(
  cd "${workdir}"
  GIT_SHA="${main_sha}" \
    IMAGE_DIGEST="${full_ref}" \
    DOCKERFILE_SHA="${dockerfile_sha}" \
    BUILD_REPOSITORY="NeedTokenAi/needtoken-storefront" \
    BUILD_SHA="${main_sha}" \
    BUILD_RUN_ID="43175381072" \
    BUILD_SOURCE="github-actions" \
    ISOLATION="isolated" \
    MAIN_REF="refs/heads/main" \
    CANDIDATE_RECORD="${record}" \
    bash "${SCRIPT}"
)
grep -q 'deploy=false' "${record}" || fail "record did not pin deploy=false"
grep -q 'rollback=false' "${record}" || fail "record claimed rollback"
grep -q 'isolation=isolated' "${record}" || fail "record did not pin isolated bindings"
grep -q "git_sha=${main_sha}" "${record}" || fail "record did not pin the main sha"
grep -q "dockerfile_sha=${dockerfile_sha}" "${record}" || fail "record did not pin the dockerfile blob"
grep -q "image_digest=${full_ref}" "${record}" || fail "record did not pin the registry ref"
grep -q 'source=github-actions' "${record}" || fail "record did not pin the build source"
ok "sha, dockerfile blob, and registry ref bind with deploy=false and rollback=false"

set +e
(
  cd "${workdir}"
  GIT_SHA="${main_sha}" \
    IMAGE_DIGEST="${bare_ref}" \
    DOCKERFILE_SHA="${dockerfile_sha}" \
    BUILD_REPOSITORY="NeedTokenAi/needtoken-storefront" \
    BUILD_SHA="${main_sha}" \
    BUILD_RUN_ID="43175381072" \
    BUILD_SOURCE="github-actions" \
    ISOLATION="isolated" \
    MAIN_REF="refs/heads/main" \
    bash "${SCRIPT}"
)
status=$?
set -e
[[ ${status} -eq 1 ]] || fail "bare sha256 exited ${status}"
ok "bare sha256 is refused"

set +e
(
  cd "${workdir}"
  GIT_SHA="${main_sha}" \
    IMAGE_DIGEST="${full_ref}" \
    DOCKERFILE_SHA="${dockerfile_sha}" \
    BUILD_REPOSITORY="NeedTokenAi/needtoken-storefront" \
    BUILD_SHA="${main_sha}" \
    BUILD_RUN_ID="43175381072" \
    BUILD_SOURCE="github-actions" \
    BUILD_RECORD_JSON="{}" \
    ISOLATION="isolated" \
    MAIN_REF="refs/heads/main" \
    bash "${SCRIPT}"
)
status=$?
set -e
[[ ${status} -eq 1 ]] || fail "repository JSON exited ${status}"
ok "repository JSON is refused"

set +e
(
  cd "${workdir}"
  GIT_SHA="${main_sha}" \
    IMAGE_DIGEST="${zero_ref}" \
    DOCKERFILE_SHA="${dockerfile_sha}" \
    BUILD_REPOSITORY="NeedTokenAi/needtoken-storefront" \
    BUILD_SHA="${main_sha}" \
    BUILD_RUN_ID="43175381072" \
    BUILD_SOURCE="github-actions" \
    ISOLATION="isolated" \
    MAIN_REF="refs/heads/main" \
    bash "${SCRIPT}"
)
status=$?
set -e
[[ ${status} -eq 1 ]] || fail "placeholder digest exited ${status}"
ok "placeholder digest stays fail-closed"

set +e
(
  cd "${workdir}"
  GIT_SHA="${main_sha}" \
    IMAGE_DIGEST="${full_ref}" \
    DOCKERFILE_SHA="aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" \
    BUILD_REPOSITORY="NeedTokenAi/needtoken-storefront" \
    BUILD_SHA="${main_sha}" \
    BUILD_RUN_ID="43175381072" \
    BUILD_SOURCE="github-actions" \
    ISOLATION="isolated" \
    MAIN_REF="refs/heads/main" \
    bash "${SCRIPT}"
)
status=$?
set -e
[[ ${status} -eq 1 ]] || fail "unrelated dockerfile sha exited ${status}"
ok "digest not tied to the Dockerfile is rejected"

set +e
(
  cd "${workdir}"
  git checkout -q feature
  GIT_SHA="${feature_sha}" \
    IMAGE_DIGEST="${full_ref}" \
    DOCKERFILE_SHA="${dockerfile_sha}" \
    BUILD_REPOSITORY="NeedTokenAi/needtoken-storefront" \
    BUILD_SHA="${feature_sha}" \
    BUILD_RUN_ID="43175381072" \
    BUILD_SOURCE="github-actions" \
    ISOLATION="isolated" \
    MAIN_REF="refs/heads/main" \
    bash "${SCRIPT}"
)
status=$?
set -e
[[ ${status} -eq 1 ]] || fail "non-main sha exited ${status}"
ok "sha outside main is rejected"

set +e
(
  cd "${workdir}"
  git checkout -q "${base_branch}"
  GIT_SHA="${main_sha}" \
    IMAGE_DIGEST="${full_ref}" \
    DOCKERFILE_SHA="${dockerfile_sha}" \
    BUILD_REPOSITORY="NeedTokenAi/needtoken-storefront" \
    BUILD_SHA="${main_sha}" \
    BUILD_RUN_ID="43175381072" \
    BUILD_SOURCE="github-actions" \
    ISOLATION="shared" \
    MAIN_REF="refs/heads/main" \
    bash "${SCRIPT}"
)
status=$?
set -e
[[ ${status} -eq 1 ]] || fail "shared isolation exited ${status}"
ok "isolation other than isolated is rejected"
