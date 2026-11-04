#!/usr/bin/env bash
# 决定这次运行能不能部署生产。
# MODE=gate    过期的 main 队列记为成功跳过（proceed=false）
# MODE=recheck 部署中途再查一次；main 已经往前走则 exit 10，调用方不得切流量
#
# 不读取 GCP 凭证。PR / dev / 手动重跑 ci 在进这个脚本之前就应被工作流 if 挡住；
# 这里再挡一层，避免只看 workflow_run 的总结论。
set -euo pipefail

MODE=${MODE:-gate}
REPO=${GITHUB_REPOSITORY:?GITHUB_REPOSITORY is required}
EVENT=${GITHUB_EVENT_NAME:?GITHUB_EVENT_NAME is required}
RUN_ID=${GITHUB_RUN_ID:?GITHUB_RUN_ID is required}
REF=${GITHUB_REF:-}

REQUIRED_JOBS=(
  "Verify installer"
  "Verify API"
  "Verify release config"
  "Verify fullstack build"
)

log() { printf '%s\n' "$*"; }

die() {
  printf '::error::%s\n' "$*" >&2
  exit 1
}

out() {
  local key=$1 value=$2
  printf '%s=%s\n' "$key" "$value"
  if [[ -n ${GITHUB_OUTPUT:-} ]]; then
    printf '%s=%s\n' "$key" "$value" >>"$GITHUB_OUTPUT"
  fi
}

is_sha() { [[ ${1:-} =~ ^[0-9a-f]{40}$ ]]; }

summary() {
  [[ -n ${GITHUB_STEP_SUMMARY:-} ]] || return 0
  printf '%s\n' "$1" >>"$GITHUB_STEP_SUMMARY"
}

finish_skip() {
  local reason=$1
  log "skip: ${reason}"
  out proceed false
  out sha "${SHA:-}"
  out allow_behind false
  out reason "$reason"
  summary "### 生产部署已跳过"
  summary ""
  summary "$reason"
  exit 0
}

cancel_queued_auto_deploys() {
  local id sha rows
  # 只取消尚未开始、且 head SHA 已是 main 祖先的自动部署。
  # 不取消 in_progress：那次可能正在切流量。认不出 SHA 的排队运行也留下，交给它自己的过期检查。
  if ! rows=$(gh api "repos/${REPO}/actions/workflows/deploy-cloudrun.yml/runs?status=queued&per_page=50" \
    --jq '.workflow_runs[] | select(.event == "workflow_run") | [.id, .head_sha] | @tsv'); then
    log "warning: unable to list queued deploy runs"
    return 0
  fi
  while IFS=$'\t' read -r id sha; do
    [[ -z ${id:-} || $id == "$RUN_ID" ]] && continue
    if [[ ! $sha =~ ^[0-9a-f]{40}$ || $sha == "$MAIN" ]]; then
      log "leave queued deploy run ${id} (sha ${sha:-missing})"
      continue
    fi
    if ! git merge-base --is-ancestor "$sha" "$MAIN" 2>/dev/null; then
      log "leave queued deploy run ${id}; ${sha} is not an ancestor of main"
      continue
    fi
    log "cancel queued deploy run ${id} for stale sha ${sha}"
    gh api --method POST "repos/${REPO}/actions/runs/${id}/cancel" >/dev/null \
      || log "warning: failed to cancel run ${id}"
  done <<<"$rows"
}

verify_ci_run() {
  local run_id=$1
  local meta event branch head conclusion path name head_repo
  meta=$(gh api "repos/${REPO}/actions/runs/${run_id}")
  event=$(jq -r '.event' <<<"$meta")
  branch=$(jq -r '.head_branch' <<<"$meta")
  head=$(jq -r '.head_sha' <<<"$meta")
  conclusion=$(jq -r '.conclusion' <<<"$meta")
  path=$(jq -r '.path' <<<"$meta")
  name=$(jq -r '.name' <<<"$meta")
  head_repo=$(jq -r '.head_repository.full_name // empty' <<<"$meta")

  [[ $event == push ]] || die "ci run ${run_id} event is ${event}, want push"
  [[ $branch == main ]] || die "ci run ${run_id} branch is ${branch}, want main"
  [[ $head == "$SHA" ]] || die "ci run ${run_id} head ${head} != ${SHA}"
  [[ $conclusion == success ]] || die "ci run ${run_id} conclusion is ${conclusion}"
  [[ $path == .github/workflows/ci.yml ]] || die "ci run ${run_id} path is ${path}"
  [[ $name == ci ]] || die "ci run ${run_id} name is ${name}"
  if [[ -n $head_repo && $head_repo != "$REPO" ]]; then
    die "ci run ${run_id} head repository is ${head_repo}"
  fi

  local jobs_tsv line job_name job_conclusion job_status
  jobs_tsv=$(gh api --paginate "repos/${REPO}/actions/runs/${run_id}/jobs?per_page=100" \
    --jq '.jobs[] | [.name, (.conclusion // ""), .status] | @tsv')
  [[ -n $jobs_tsv ]] || die "ci run ${run_id} has no jobs"

  declare -A seen=()
  while IFS= read -r line; do
    [[ -z $line ]] && continue
    IFS=$'\t' read -r job_name job_conclusion job_status <<<"$line"
    seen["$job_name"]=1
    if [[ $job_status != completed || $job_conclusion != success ]]; then
      die "ci job '${job_name}' status=${job_status} conclusion=${job_conclusion}"
    fi
  done <<<"$jobs_tsv"

  local required
  for required in "${REQUIRED_JOBS[@]}"; do
    [[ -n ${seen[$required]:-} ]] || die "missing successful ci job '${required}'"
  done
}

safe_context() {
  local value=${1:-}
  local pattern='^[A-Za-z0-9._:/() -]+$'
  [[ $value =~ $pattern ]]
}

check_conclusion_for_context() {
  local context=$1
  local short=$context
  local checks id name conclusion status best found_conclusion found_status
  best=-1
  found_conclusion=""
  found_status=""
  if [[ $context == *" / "* ]]; then
    short=${context##* / }
  fi
  checks=$(gh api --paginate "repos/${REPO}/commits/${SHA}/check-runs?per_page=100" \
    --jq '.check_runs[] | [.id, .name, (.conclusion // ""), .status] | @tsv')
  while IFS=$'\t' read -r id name conclusion status; do
    [[ -z ${id:-} ]] && continue
    if [[ $name == "$context" || $name == "$short" ]]; then
      if [[ $id -gt $best ]]; then
        best=$id
        found_conclusion=$conclusion
        found_status=$status
      fi
    fi
  done <<<"$checks"

  if [[ $best -lt 0 ]]; then
    local state
    state=$(gh api "repos/${REPO}/commits/${SHA}/status" \
      --jq ".statuses[]? | select(.context == \"${context}\") | .state" | head -n 1 || true)
    if [[ $state == success ]]; then
      return 0
    fi
    if [[ -n $state ]]; then
      die "required status check '${context}' state=${state}"
    fi
    die "required status check '${context}' is missing on ${SHA}"
  fi
  if [[ $found_status != completed || $found_conclusion != success ]]; then
    die "required status check '${context}' status=${found_status} conclusion=${found_conclusion}"
  fi
}

enforce_rulesets() {
  local rules rc errfile contexts context
  errfile=$(mktemp)
  set +e
  rules=$(gh api "repos/${REPO}/rules/branches/main" 2>"$errfile")
  rc=$?
  set -e
  if [[ $rc -ne 0 ]]; then
    log "warning: could not read branch rules for main ($(tr '\n' ' ' <"$errfile" | head -c 240))"
    log "warning: enforcing the ci workflow jobs only; confirm rulesets if branch protection adds more checks"
    rm -f "$errfile"
    return 0
  fi
  rm -f "$errfile"
  contexts=$(jq -r '.[]? | select(.type == "required_status_checks") | .parameters.required_status_checks[]?.context' <<<"$rules")
  [[ -n $contexts ]] || return 0
  while IFS= read -r context; do
    [[ -z $context ]] && continue
    safe_context "$context" || die "refusing unusual required check context: ${context}"
    check_conclusion_for_context "$context"
  done <<<"$contexts"
}

SHA=""
ALLOW=false
case "$EVENT" in
  workflow_dispatch)
    [[ $REF == refs/heads/main ]] || die "workflow_dispatch must be started from refs/heads/main (got ${REF})"
    SHA=${INPUT_SHA:-}
    is_sha "$SHA" || die "git_sha must be a full 40-character lowercase commit SHA"
    if [[ ${INPUT_ALLOW_BEHIND:-false} == true ]]; then
      ALLOW=true
    fi
    ;;
  workflow_run)
    [[ ${WR_HEAD_REPO:-} == "$REPO" ]] || die "refusing workflow_run from ${WR_HEAD_REPO:-unknown}"
    [[ ${WR_EVENT:-} == push ]] || die "refusing workflow_run event ${WR_EVENT:-unknown}"
    [[ ${WR_BRANCH:-} == main ]] || die "refusing workflow_run branch ${WR_BRANCH:-unknown}"
    [[ ${WR_CONCLUSION:-} == success ]] || die "refusing workflow_run conclusion ${WR_CONCLUSION:-unknown}"
    SHA=${WR_SHA:-}
    is_sha "$SHA" || die "workflow_run head_sha is not a full SHA"
    ALLOW=false
    ;;
  *)
    die "unsupported event ${EVENT}"
    ;;
esac

if [[ -n ${DEPLOY_SHA:-} ]]; then
  is_sha "$DEPLOY_SHA" || die "DEPLOY_SHA is not a full SHA"
  [[ $DEPLOY_SHA == "$SHA" ]] || die "DEPLOY_SHA ${DEPLOY_SHA} does not match event SHA ${SHA}"
fi
if [[ -n ${DEPLOY_ALLOW_BEHIND:-} ]]; then
  if [[ $DEPLOY_ALLOW_BEHIND == true ]]; then
    ALLOW=true
  else
    ALLOW=false
  fi
fi

cd "$(git rev-parse --show-toplevel)"
git fetch --no-tags origin '+refs/heads/main:refs/remotes/origin/main'
MAIN=$(git rev-parse origin/main)
is_sha "$MAIN" || die "origin/main is not a full SHA"

if [[ $SHA != "$MAIN" ]]; then
  if [[ $(git rev-parse --is-shallow-repository) == true ]]; then
    git fetch --no-tags --unshallow origin
    MAIN=$(git rev-parse origin/main)
  fi
  git fetch --no-tags origin "$SHA"
  git cat-file -e "${SHA}^{commit}" || die "SHA ${SHA} is not a commit"
  git merge-base --is-ancestor "$SHA" "$MAIN" || die "SHA ${SHA} is not contained in origin/main (${MAIN})"
fi

if [[ $SHA != "$MAIN" && $ALLOW != true ]]; then
  if [[ $MODE == recheck ]]; then
    log "stale: origin/main is ${MAIN}, this run is ${SHA}"
    exit 10
  fi
  finish_skip "stale: origin/main ${MAIN} has moved past ${SHA}"
fi

if [[ $EVENT == workflow_run ]]; then
  [[ ${WR_RUN_ID:-} =~ ^[0-9]+$ ]] || die "workflow_run id is missing"
  verify_ci_run "$WR_RUN_ID"
else
  list_url="repos/${REPO}/actions/workflows/ci.yml/runs?head_sha=${SHA}&event=push&status=success&per_page=20"
  found=$(gh api "$list_url" \
    --jq "[.workflow_runs[] | select(.head_branch == \"main\" and .head_sha == \"${SHA}\")][0].id")
  [[ $found =~ ^[0-9]+$ ]] || die "no successful push-to-main ci run for ${SHA}"
  verify_ci_run "$found"
fi

enforce_rulesets

if [[ $MODE == gate && $SHA == "$MAIN" ]]; then
  cancel_queued_auto_deploys || log "warning: unable to list queued deploy runs"
fi

out proceed true
out sha "$SHA"
out allow_behind "$ALLOW"
out reason "ci green for ${SHA}"
log "preflight ok: ${SHA} (main ${MAIN}, allow_behind ${ALLOW})"
