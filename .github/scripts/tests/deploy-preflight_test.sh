#!/usr/bin/env bash
# 门禁：过期 SHA 在调用 GitHub API 之前就跳过；当前 main 必须看到 ci 的全部 job。
set -euo pipefail

root=$(cd "$(dirname "$0")/../.." && pwd)
preflight="$root/scripts/deploy-preflight.sh"
repo_name=NeedTokenAi/needtoken-storefront

fail() {
  printf 'FAIL: %s\n' "$*" >&2
  exit 1
}

assert_rc() {
  local want=$1
  local rc=$2
  [[ $rc -eq $want ]] || fail "exit $rc, want $want; log: $log"
}

run_preflight() {
  set +e
  log=$(
    cd "${WORK:-.}" && env \
      GITHUB_REPOSITORY="$repo_name" \
      GITHUB_EVENT_NAME="$EVENT" \
      GITHUB_RUN_ID="${GITHUB_RUN_ID:-1}" \
      GITHUB_REF="${REF:-refs/heads/main}" \
      GITHUB_OUTPUT="$output" \
      GITHUB_STEP_SUMMARY="$summary" \
      INPUT_SHA="${INPUT_SHA:-}" \
      INPUT_ALLOW_BEHIND="${INPUT_ALLOW_BEHIND:-false}" \
      WR_SHA="${WR_SHA:-}" \
      WR_EVENT="${WR_EVENT:-}" \
      WR_BRANCH="${WR_BRANCH:-}" \
      WR_CONCLUSION="${WR_CONCLUSION:-}" \
      WR_HEAD_REPO="${WR_HEAD_REPO:-}" \
      WR_RUN_ID="${WR_RUN_ID:-}" \
      DEPLOY_SHA="${DEPLOY_SHA:-}" \
      DEPLOY_ALLOW_BEHIND="${DEPLOY_ALLOW_BEHIND:-}" \
      MODE="${MODE:-gate}" \
      PATH="$mockbin:$PATH" \
      EXPECT_SHA="${EXPECT_SHA:-}" \
      FIXTURE_JOBS="${FIXTURE_JOBS:-}" \
      GH_LOG="$ghlog" \
      bash "$preflight" 2>&1
  )
  rc=$?
  set -e
}

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
output=$tmp/out
summary=$tmp/summary
ghlog=$tmp/gh.log
mockbin=$tmp/bin
mkdir -p "$mockbin"
: >"$output"
: >"$summary"
: >"$ghlog"

cat >"$mockbin/gh" <<'EOS'
#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' "$*" >>"${GH_LOG:?}"
[[ ${1:-} == api ]] || { echo "unexpected gh $*" >&2; exit 1; }
shift
jqfilter=""
url=""
while [[ $# -gt 0 ]]; do
  case $1 in
    --method) shift 2 ;;
    --paginate) shift ;;
    --jq) jqfilter=$2; shift 2 ;;
    *) url=$1; shift ;;
  esac
done
emit() {
  if [[ -n $jqfilter ]]; then
    jq -r "$jqfilter" <<<"$1"
  else
    printf '%s\n' "$1"
  fi
}
if [[ $url == *"/jobs"* ]]; then
  emit "$(cat "${FIXTURE_JOBS:?}")"
elif [[ $url == *"/actions/runs/99" ]]; then
  emit "$(jq -n --arg sha "${EXPECT_SHA:?}" '{event:"push",head_branch:"main",head_sha:$sha,conclusion:"success",path:".github/workflows/ci.yml",name:"ci",head_repository:{full_name:"NeedTokenAi/needtoken-storefront"}}')"
elif [[ $url == *"/rules/branches/main" ]]; then
  echo "rules unavailable" >&2
  exit 1
elif [[ $url == *"/actions/workflows/ci.yml/runs"* ]]; then
  sha=$(sed -n 's/.*head_sha=\([0-9a-f]\{40\}\).*/\1/p' <<<"$url")
  emit "$(jq -n --arg sha "$sha" '{workflow_runs:[{id:99,head_branch:"main",head_sha:$sha,event:"push",conclusion:"success"}]}')"
elif [[ $url == *"status=queued"* ]]; then
  emit '{"workflow_runs":[]}'
else
  echo "unexpected gh url: $url" >&2
  exit 1
fi
EOS
chmod +x "$mockbin/gh"

jobs_ok=$tmp/jobs-ok.json
jobs_missing=$tmp/jobs-missing.json
cat >"$jobs_ok" <<'EOF'
{"jobs":[
  {"name":"Verify installer","conclusion":"success","status":"completed"},
  {"name":"Verify API","conclusion":"success","status":"completed"},
  {"name":"Verify release config","conclusion":"success","status":"completed"},
  {"name":"Verify fullstack build","conclusion":"success","status":"completed"}
]}
EOF
cat >"$jobs_missing" <<'EOF'
{"jobs":[
  {"name":"Verify installer","conclusion":"success","status":"completed"},
  {"name":"Verify release config","conclusion":"success","status":"completed"},
  {"name":"Verify fullstack build","conclusion":"success","status":"completed"}
]}
EOF

EVENT=workflow_run
WR_HEAD_REPO=evil/fork
WR_EVENT=push
WR_BRANCH=main
WR_CONCLUSION=success
WR_SHA=0123456789abcdef0123456789abcdef01234567
WR_RUN_ID=99
run_preflight
assert_rc 1 "$rc"
printf '%s\n' "$log" | grep -q 'evil/fork' || fail "fork was not rejected"

EVENT=workflow_dispatch
REF=refs/heads/dev
INPUT_SHA=0123456789abcdef0123456789abcdef01234567
WR_HEAD_REPO=""
run_preflight
assert_rc 1 "$rc"
printf '%s\n' "$log" | grep -q 'refs/heads/main' || fail "non-main dispatch was not rejected"

# 本地 main：旧提交必须在打 GitHub API 之前跳过。
origin=$tmp/origin.git
work=$tmp/work
git init --bare -q "$origin"
git init -q -b main "$work"
git -C "$work" config user.email "test@example.com"
git -C "$work" config user.name "test"
echo one >"$work/file"
git -C "$work" add file
git -C "$work" commit -q -m one
old=$(git -C "$work" rev-parse HEAD)
echo two >>"$work/file"
git -C "$work" add file
git -C "$work" commit -q -m two
new=$(git -C "$work" rev-parse HEAD)
git -C "$work" remote add origin "$origin"
git -C "$origin" config uploadpack.allowReachableSHA1InWant true
git -C "$origin" config uploadpack.allowAnySHA1InWant true
git -C "$work" push -q origin main
WORK=$work

cat >"$mockbin/gh" <<'EOS'
#!/usr/bin/env bash
echo "gh should not be called: $*" >&2
exit 99
EOS
chmod +x "$mockbin/gh"

EVENT=workflow_run
REF=refs/heads/main
WR_HEAD_REPO=$repo_name
WR_EVENT=push
WR_BRANCH=main
WR_CONCLUSION=success
WR_SHA=$old
WR_RUN_ID=99
MODE=gate
: >"$output"
run_preflight
assert_rc 0 "$rc"
grep -q '^proceed=false$' "$output" || fail "stale gate should skip: $(cat "$output")"
printf '%s\n' "$log" | grep -q 'stale:' || fail "missing stale note"

MODE=recheck
: >"$output"
run_preflight
assert_rc 10 "$rc"

# 不在 main 上的提交直接失败，仍然不调用 gh。
git -C "$work" checkout -q -b side "$old"
echo side >"$work/side"
git -C "$work" add side
git -C "$work" commit -q -m side
side=$(git -C "$work" rev-parse HEAD)
git -C "$work" push -q origin side
git -C "$work" checkout -q main
MODE=gate
WR_SHA=$side
: >"$output"
run_preflight
assert_rc 1 "$rc"
printf '%s\n' "$log" | grep -q 'not contained' || fail "side commit should be rejected"

# 当前 main：四个 ci job 都成功才放行。mock 恢复。
cat >"$mockbin/gh" <<'EOS'
#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' "$*" >>"${GH_LOG:?}"
[[ ${1:-} == api ]] || { echo "unexpected gh $*" >&2; exit 1; }
shift
jqfilter=""
url=""
while [[ $# -gt 0 ]]; do
  case $1 in
    --method) shift 2 ;;
    --paginate) shift ;;
    --jq) jqfilter=$2; shift 2 ;;
    *) url=$1; shift ;;
  esac
done
emit() {
  if [[ -n $jqfilter ]]; then
    jq -r "$jqfilter" <<<"$1"
  else
    printf '%s\n' "$1"
  fi
}
if [[ $url == *"/jobs"* ]]; then
  emit "$(cat "${FIXTURE_JOBS:?}")"
elif [[ $url == *"/actions/runs/99" ]]; then
  emit "$(jq -n --arg sha "${EXPECT_SHA:?}" '{event:"push",head_branch:"main",head_sha:$sha,conclusion:"success",path:".github/workflows/ci.yml",name:"ci",head_repository:{full_name:"NeedTokenAi/needtoken-storefront"}}')"
elif [[ $url == *"/rules/branches/main" ]]; then
  echo "rules unavailable" >&2
  exit 1
elif [[ $url == *"/actions/workflows/ci.yml/runs"* ]]; then
  sha=$(sed -n 's/.*head_sha=\([0-9a-f]\{40\}\).*/\1/p' <<<"$url")
  emit "$(jq -n --arg sha "$sha" '{workflow_runs:[{id:99,head_branch:"main",head_sha:$sha,event:"push",conclusion:"success"}]}')"
elif [[ $url == *"status=queued"* ]]; then
  emit '{"workflow_runs":[]}'
else
  echo "unexpected gh url: $url" >&2
  exit 1
fi
EOS
chmod +x "$mockbin/gh"

EVENT=workflow_run
MODE=gate
WR_SHA=$new
EXPECT_SHA=$new
FIXTURE_JOBS=$jobs_ok
: >"$output"
: >"$ghlog"
run_preflight
assert_rc 0 "$rc"
grep -q '^proceed=true$' "$output" || fail "current main should proceed: $(cat "$output")"
grep -q "^sha=${new}$" "$output" || fail "sha output mismatch"

FIXTURE_JOBS=$jobs_missing
: >"$output"
run_preflight
assert_rc 1 "$rc"
printf '%s\n' "$log" | grep -q 'Verify API' || fail "missing job should be named"

EVENT=workflow_dispatch
REF=refs/heads/main
INPUT_SHA=$old
INPUT_ALLOW_BEHIND=true
EXPECT_SHA=$old
FIXTURE_JOBS=$jobs_ok
MODE=gate
: >"$output"
: >"$ghlog"
run_preflight
assert_rc 0 "$rc"
grep -q '^proceed=true$' "$output" || fail "allow_behind should proceed"
grep -q '^allow_behind=true$' "$output" || fail "allow_behind output"
if grep -q 'method POST' "$ghlog"; then
  fail "behind-main dispatch must not cancel other runs"
fi

if grep -R -n -E '^[[:space:]]*pull_request_target:|credentials_json:|gcloud run deploy|gcloud run services replace|--set-env-vars|--set-secrets' \
  "$root/workflows" "$root/scripts" --exclude-dir=tests; then
  fail "deploy/sync files contain a forbidden production operation"
fi
grep -q -- '--no-traffic' "$root/scripts/deploy-apply.sh" || fail "missing --no-traffic"
grep -q 'update-traffic' "$root/scripts/deploy-apply.sh" || fail "missing traffic shift"

echo "deploy-preflight ok"
