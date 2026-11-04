#!/usr/bin/env bash
# 在干净工作区里构建 Dockerfile，推送「完整 SHA」标签，并记下 digest。
# 部署必须使用这里捕获的 digest，不能等会儿再按标签反查（标签可以被后一次推送挪走）。
#
# Dockerfile 没有声明 VITE_NEEDTOKEN_* ARG/ENV。这里不要加那些 --build-arg：
# 构建期用不到，运行时 Cloud Run 环境变量也改不了已经编进前端的常量。
# 默认值在 frontend/user/src/templates/needtoken/config.ts（https://api.needtoken.ai）。
# APP_VERSION 是 Dockerfile 里已有的 ARG，只写进 Go 版本字符串。
set -euo pipefail

SHA=${SHA:?}
IMAGE=${IMAGE:?}
PROJECT=${GCP_PROJECT_ID:?}
REGION=${GCP_REGION:?}
REPO=${GITHUB_REPOSITORY:?}
RUN_ID=${GITHUB_RUN_ID:?}

[[ $SHA =~ ^[0-9a-f]{40}$ ]] || {
  echo "bad SHA" >&2
  exit 1
}
[[ $(git rev-parse HEAD) == "$SHA" ]] || {
  echo "checkout HEAD $(git rev-parse HEAD) != ${SHA}" >&2
  exit 1
}
case "$IMAGE" in
  *@* | *:*)
    echo "ARTIFACT_REGISTRY_IMAGE must be a repository path without tag or digest" >&2
    exit 1
    ;;
esac
[[ $IMAGE == "${REGION}-docker.pkg.dev/${PROJECT}/"* ]] || {
  echo "image ${IMAGE} is not in ${REGION}-docker.pkg.dev/${PROJECT}/" >&2
  exit 1
}

cd "$(git rev-parse --show-toplevel)"
registry=${IMAGE%%/*}
gcloud auth configure-docker "$registry" --quiet

meta=$(mktemp)
trap 'rm -f "$meta"' EXIT

docker buildx build \
  --platform linux/amd64 \
  --provenance=false \
  --sbom=false \
  --push \
  --tag "${IMAGE}:${SHA}" \
  --build-arg "APP_VERSION=${SHA}" \
  --label "org.opencontainers.image.revision=${SHA}" \
  --label "org.opencontainers.image.source=https://github.com/${REPO}" \
  --metadata-file "$meta" \
  .

digest=$(jq -r '."containerimage.digest" // empty' "$meta")
[[ $digest =~ ^sha256:[0-9a-f]{64}$ ]] || {
  echo "docker buildx did not record a digest" >&2
  cat "$meta" >&2
  exit 1
}
config_digest=$(jq -r '."containerimage.config.digest" // empty' "$meta")

jq -n \
  --arg sha "$SHA" \
  --arg image "$IMAGE" \
  --arg digest "$digest" \
  --arg config "$config_digest" \
  --arg run_id "$RUN_ID" \
  --arg run_url "${GITHUB_SERVER_URL:-https://github.com}/${REPO}/actions/runs/${RUN_ID}" \
  '{source_sha:$sha,image:$image,image_digest:$digest,image_config_digest:$config,github_run_id:$run_id,github_run_url:$run_url}' \
  >build-record.json

if [[ -n ${GITHUB_STEP_SUMMARY:-} ]]; then
  cat >>"$GITHUB_STEP_SUMMARY" <<EOF
### 镜像

- source SHA: \`${SHA}\`
- tag: \`${IMAGE}:${SHA}\`
- digest: \`${digest}\`
- GitHub run: \`${RUN_ID}\`

EOF
fi

printf 'digest=%s\n' "$digest"
