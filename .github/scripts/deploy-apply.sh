#!/usr/bin/env bash
# 把已经推送的 digest 做成一个不接流量的 Cloud Run 修订版本，冒烟通过后再切 100%。
# 只改镜像。不重建服务，不重写 env / secret / 资源 / 扩缩，不跑管理端重置命令。
set -euo pipefail

SCRIPT_DIR=$(cd "$(dirname "$0")" && pwd)
cd "$(git rev-parse --show-toplevel)"

SHA=${SHA:?}
IMAGE=${IMAGE:?}
PROJECT=${GCP_PROJECT_ID:?}
REGION=${GCP_REGION:?}
SERVICE=${CLOUD_RUN_SERVICE:?}
RUN_ID=${GITHUB_RUN_ID:?}
RUN_URL="${GITHUB_SERVER_URL:-https://github.com}/${GITHUB_REPOSITORY:?}/actions/runs/${RUN_ID}"

[[ $PROJECT =~ ^[a-z][a-z0-9-]{4,28}[a-z0-9]$ ]] || {
  echo "unexpected GCP_PROJECT_ID" >&2
  exit 1
}
[[ $REGION =~ ^[a-z]+-[a-z0-9]+$ ]] || {
  echo "unexpected GCP_REGION" >&2
  exit 1
}
[[ $SERVICE =~ ^[a-z][a-z0-9-]{0,48}[a-z0-9]$ ]] || {
  echo "unexpected CLOUD_RUN_SERVICE" >&2
  exit 1
}
[[ -f build-record.json ]] || {
  echo "build-record.json missing" >&2
  exit 1
}

digest=$(jq -r '.image_digest' build-record.json)
built_image=$(jq -r '.image' build-record.json)
built_sha=$(jq -r '.source_sha' build-record.json)
image_config=$(jq -r '.image_config_digest // empty' build-record.json)
[[ $built_image == "$IMAGE" && $built_sha == "$SHA" ]] || {
  echo "build record does not match this deploy" >&2
  exit 1
}
[[ $digest =~ ^sha256:[0-9a-f]{64}$ ]] || {
  echo "build record digest is invalid" >&2
  exit 1
}

promoted=false
traffic_percent=0
revision=""
previous_revision=""
smoke_url=""
skipped_reason=""
smoke='{"health":"not-run","homepage":"not-run","public_config":"not-run"}'
tag_set=0
record_written=0

export CLOUDSDK_CORE_DISABLE_PROMPTS=1

write_record() {
  [[ $record_written -eq 1 ]] && return 0
  if [[ ! $traffic_percent =~ ^[0-9]+$ ]]; then
    traffic_percent=0
  fi
  if [[ $promoted != true && $promoted != false ]]; then
    promoted=false
  fi
  if ! jq -e . >/dev/null <<<"$smoke"; then
    smoke='{"health":"unknown"}'
  fi
  jq -n \
    --arg source_sha "$SHA" \
    --arg main_sha "${MAIN_SHA:-}" \
    --arg run_id "$RUN_ID" \
    --arg run_url "$RUN_URL" \
    --arg image "$IMAGE" \
    --arg image_digest "$digest" \
    --arg image_config_digest "$image_config" \
    --arg revision "$revision" \
    --arg previous_revision "$previous_revision" \
    --arg smoke_url "$smoke_url" \
    --arg skipped_reason "$skipped_reason" \
    --argjson traffic_percent "$traffic_percent" \
    --argjson promoted "$promoted" \
    --argjson smoke "$smoke" \
    '{
      source_sha:$source_sha,
      main_sha:$main_sha,
      github_run_id:$run_id,
      github_run_url:$run_url,
      image:$image,
      image_digest:$image_digest,
      image_config_digest:$image_config_digest,
      revision:$revision,
      previous_revision:$previous_revision,
      traffic_percent:$traffic_percent,
      promoted:$promoted,
      smoke_url:$smoke_url,
      skipped_reason:$skipped_reason,
      smoke:$smoke,
      note:"新修订版本启动时会连接现有生产库并执行 AutoMigrate。--no-traffic 不隔离数据库。本记录不包含重置管理员或重建数据库的操作。"
    }' >deploy-record.json

  if [[ -n ${GITHUB_STEP_SUMMARY:-} ]]; then
    cat >>"$GITHUB_STEP_SUMMARY" <<EOF
### 部署记录

- source SHA: \`${SHA}\`
- main SHA: \`${MAIN_SHA:-unknown}\`
- image digest: \`${digest}\`
- revision: \`${revision:-none}\`
- previous revision: \`${previous_revision:-none}\`
- traffic: \`${traffic_percent}%\`
- promoted: \`${promoted}\`
- smoke URL: \`${smoke_url:-none}\`
- smoke: \`$(jq -c . <<<"$smoke")\`
- note: ${skipped_reason:-切流量前会再核对 main；未调用 admin reset-password / reset-2fa，未重建数据库}

EOF
  fi
  record_written=1
}

remove_smoke_tag() {
  [[ $tag_set -eq 1 ]] || return 0
  tag_set=0
  gcloud run services update-traffic "$SERVICE" \
    --project="$PROJECT" \
    --region="$REGION" \
    --platform=managed \
    --remove-tags=smoke \
    --quiet \
    || echo "warning: failed to remove smoke tag" >&2
}

cleanup() {
  local rc=$?
  remove_smoke_tag || true
  write_record || true
  exit "$rc"
}
trap cleanup EXIT

run_recheck() {
  local rc
  set +e
  MODE=recheck "$SCRIPT_DIR/deploy-preflight.sh"
  rc=$?
  set -e
  RECHECK_RC=$rc
}

stale_exit() {
  local why=$1
  skipped_reason=$why
  promoted=false
  echo "::warning::${why}"
  exit 0
}

run_recheck
MAIN_SHA=$(git rev-parse origin/main)
if [[ $RECHECK_RC -eq 10 ]]; then
  stale_exit "stale before Cloud Run update: origin/main moved past ${SHA}"
elif [[ $RECHECK_RC -ne 0 ]]; then
  exit "$RECHECK_RC"
fi

set +e
svc_name=$(gcloud run services describe "$SERVICE" \
  --project="$PROJECT" --region="$REGION" --platform=managed \
  --format='value(metadata.name)' 2>/tmp/gcloud-describe.err)
svc_rc=$?
set -e
if [[ $svc_rc -ne 0 || $svc_name != "$SERVICE" ]]; then
  echo "Cloud Run service ${SERVICE} not found in ${PROJECT}/${REGION}; refusing to create it" >&2
  cat /tmp/gcloud-describe.err >&2 || true
  exit 1
fi

before=$(mktemp)
gcloud run services describe "$SERVICE" \
  --project="$PROJECT" --region="$REGION" --platform=managed \
  --format=json >"$before"
previous_revision=$(jq -r '[.status.traffic[]? | select((.percent // 0) > 0)] | max_by(.percent) | .revisionName // empty' "$before")

gcloud run services update "$SERVICE" \
  --project="$PROJECT" \
  --region="$REGION" \
  --platform=managed \
  --image="${IMAGE}@${digest}" \
  --no-traffic \
  --quiet \
  --format=json >"$before"

revision=$(jq -r '.status.latestCreatedRevisionName // empty' "$before")
[[ $revision =~ ^[a-z0-9-]+$ ]] || {
  echo "did not get a revision name" >&2
  exit 1
}

served=$(gcloud run revisions describe "$revision" \
  --project="$PROJECT" --region="$REGION" --platform=managed \
  --format='value(spec.containers[0].image)')
[[ $served == *"@${digest}" ]] || {
  echo "revision ${revision} image is ${served}, want @${digest}" >&2
  exit 1
}

gcloud run services update-traffic "$SERVICE" \
  --project="$PROJECT" \
  --region="$REGION" \
  --platform=managed \
  --update-tags="smoke=${revision}" \
  --quiet
tag_set=1

gcloud run services describe "$SERVICE" \
  --project="$PROJECT" --region="$REGION" --platform=managed \
  --format=json >"$before"
smoke_url=$(jq -r '.status.traffic[]? | select(.tag == "smoke") | .url' "$before" | head -n 1)
if [[ -z $smoke_url ]]; then
  host=$(jq -r '.status.url' "$before")
  host=${host#https://}
  smoke_url="https://smoke---${host}"
fi
[[ $smoke_url == https://* ]] || {
  echo "no smoke URL for ${revision}" >&2
  exit 1
}

set +e
SMOKE_JSON="$PWD/smoke-result.json" "$SCRIPT_DIR/deploy-smoke.sh" "$smoke_url"
smoke_rc=$?
set -e
if [[ -f smoke-result.json ]]; then
  smoke=$(jq -c . smoke-result.json)
fi
if [[ $smoke_rc -ne 0 ]]; then
  skipped_reason="smoke failed; traffic was not changed"
  echo "::error::${skipped_reason}" >&2
  exit 1
fi

run_recheck
MAIN_SHA=$(git rev-parse origin/main)
if [[ $RECHECK_RC -eq 10 ]]; then
  stale_exit "smoke passed but origin/main moved past ${SHA}; left ${revision} at 0% traffic"
elif [[ $RECHECK_RC -ne 0 ]]; then
  exit "$RECHECK_RC"
fi

gcloud run services update-traffic "$SERVICE" \
  --project="$PROJECT" \
  --region="$REGION" \
  --platform=managed \
  --to-revisions="${revision}=100" \
  --quiet

gcloud run services describe "$SERVICE" \
  --project="$PROJECT" --region="$REGION" --platform=managed \
  --format=json >"$before"
traffic_percent=$(jq -r --arg rev "$revision" '[.status.traffic[]? | select(.revisionName == $rev) | .percent] | add // 0' "$before")
[[ $traffic_percent == 100 ]] || {
  echo "revision ${revision} traffic is ${traffic_percent}, want 100" >&2
  exit 1
}
promoted=true
skipped_reason=""
exit 0
