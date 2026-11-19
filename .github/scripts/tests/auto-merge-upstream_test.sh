#!/usr/bin/env bash
# 覆盖上游同步 PR 自动合并门禁：
#   1. 低风险 + CI 全绿 + 双父 merge commit -> 必须用 merge_method=merge + 精确 sha 合并并删分支
#   2. 命中高危路径 -> 拒绝合并并幂等留评（重复执行更新同一条评论，不刷屏）
#   3. 变更文件数超过阈值 / 与 API 列表不一致 -> fail-closed 拒绝合并
#   4. CI 未完成或失败 -> 拒绝合并
#   5. 非双父 merge commit 或第二父提交篡改 -> 拒绝合并
#   6. UPSTREAM_AUTO_MERGE=false 熔断开关 -> 拒绝合并
set -euo pipefail

root=$(cd "$(dirname "$0")/.." && pwd)
script=$root/auto-merge-upstream.sh

fail() {
  printf 'FAIL: %s\n' "$*" >&2
  exit 1
}

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

UPSTREAM_SHA="1111111111111111111111111111111111111111"
BASE_SHA="2222222222222222222222222222222222222222"
HEAD_SHA="3333333333333333333333333333333333333333"
HEAD_BRANCH="sync/upstream/${UPSTREAM_SHA}"

cat >"$tmp/gh" <<'EOS'
#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' "$*" >>"${MOCK_LOG:?}"

cmd=${1:-}
shift || true

if [[ $cmd == pr && ${1:-} == list ]]; then
  cat "${MOCK_PR_LIST:?}"
  exit 0
fi

if [[ $cmd == api ]]; then
  method="GET"
  jqfilter=""
  endpoint=""
  while [[ $# -gt 0 ]]; do
    case $1 in
      -X) method=$2; shift 2 ;;
      --jq) jqfilter=$2; shift 2 ;;
      --paginate) shift ;;
      -f | -F) shift 2 ;;
      *)
        if [[ -z $endpoint ]]; then
          endpoint=$1
        fi
        shift
        ;;
    esac
  done

  out=""
  case "${method}:${endpoint}" in
    GET:repos/*/commits/*/check-runs)
      out=$(cat "${MOCK_CHECK_RUNS:?}")
      ;;
    GET:repos/*/commits/*/status)
      out=$(cat "${MOCK_STATUS:?}")
      ;;
    GET:repos/*/commits/*)
      out=$(cat "${MOCK_COMMIT:?}")
      ;;
    GET:repos/*/compare/*)
      out=$(cat "${MOCK_COMPARE:?}")
      ;;
    GET:repos/*/pulls/*/files)
      out=$(cat "${MOCK_FILES:?}")
      ;;
    GET:repos/*/pulls/*)
      out=$(cat "${MOCK_PR_DETAIL:?}")
      ;;
    GET:repos/*/issues/*/comments)
      out=$(cat "${MOCK_COMMENTS:?}")
      ;;
    POST:repos/*/issues/*/comments)
      printf '{"id":901}\n'
      exit 0
      ;;
    PATCH:repos/*/issues/comments/*)
      printf '{"id":901}\n'
      exit 0
      ;;
    PUT:repos/*/pulls/*/merge)
      printf '{"merged":true}\n'
      exit 0
      ;;
    DELETE:repos/*/git/refs/heads/*)
      exit 0
      ;;
    *)
      echo "unexpected mock api call: ${method} ${endpoint}" >&2
      exit 1
      ;;
  esac

  if [[ -n $jqfilter ]]; then
    jq -r "$jqfilter" <<<"$out"
  else
    printf '%s\n' "$out"
  fi
  exit 0
fi

echo "unexpected mock gh invocation: $cmd $*" >&2
exit 1
EOS
chmod +x "$tmp/gh"

reset_mocks() {
  : >"$tmp/gh.log"
  cat >"$tmp/pr_list.json" <<EOF
[{"number":42,"url":"https://example.test/pr/42","headRefOid":"${HEAD_SHA}","baseRefName":"main","isDraft":false}]
EOF
  cat >"$tmp/commit.json" <<EOF
{"parents":[{"sha":"${BASE_SHA}"},{"sha":"${UPSTREAM_SHA}"}]}
EOF
  printf '{"status":"identical"}\n' >"$tmp/compare.json"
  printf '{"changed_files":2}\n' >"$tmp/pr_detail.json"
  printf '[{"filename":"README.md"},{"filename":"frontend/user/src/App.vue"}]\n' >"$tmp/files.json"
  cat >"$tmp/check_runs.json" <<'EOF'
{"check_runs":[
  {"id":10,"name":"Verify installer","status":"completed","conclusion":"success"},
  {"id":11,"name":"Verify API","status":"completed","conclusion":"success"},
  {"id":12,"name":"Verify release config","status":"completed","conclusion":"success"},
  {"id":13,"name":"Verify fullstack build","status":"completed","conclusion":"success"}
]}
EOF
  printf '{"total_count":0,"state":"pending"}\n' >"$tmp/status.json"
  printf '[]\n' >"$tmp/comments.json"
}

run_gate() {
  env \
    PATH="$tmp:$PATH" \
    GITHUB_REPOSITORY="NeedTokenAi/needtoken-storefront" \
    HEAD_BRANCH="$HEAD_BRANCH" \
    HEAD_SHA="$HEAD_SHA" \
    REQUIRED_CHECKS=$'Verify installer\nVerify API\nVerify release config\nVerify fullstack build' \
    MOCK_LOG="$tmp/gh.log" \
    MOCK_PR_LIST="$tmp/pr_list.json" \
    MOCK_COMMIT="$tmp/commit.json" \
    MOCK_COMPARE="$tmp/compare.json" \
    MOCK_PR_DETAIL="$tmp/pr_detail.json" \
    MOCK_FILES="$tmp/files.json" \
    MOCK_CHECK_RUNS="$tmp/check_runs.json" \
    MOCK_STATUS="$tmp/status.json" \
    MOCK_COMMENTS="$tmp/comments.json" \
    "$@" \
    bash "$script"
}

# 1. 低风险 + CI 全绿 -> 必须以 merge_method=merge + 精确 sha 合并并删分支
reset_mocks
run_gate >/dev/null
grep -q "PUT repos/NeedTokenAi/needtoken-storefront/pulls/42/merge -f merge_method=merge -f sha=${HEAD_SHA}" "$tmp/gh.log" \
  || fail "case 1: expected merge commit API call"
grep -q "DELETE repos/NeedTokenAi/needtoken-storefront/git/refs/heads/${HEAD_BRANCH}" "$tmp/gh.log" \
  || fail "case 1: expected sync branch deletion"

# 2. 命中高危路径 -> 拒绝合并，首次 POST 评论，二次 PATCH 同一评论（不刷屏）
reset_mocks
printf '[{"filename":"README.md"},{"filename":"internal/modules/payment/pay.go"}]\n' >"$tmp/files.json"
run_gate >/dev/null
if grep -q "pulls/42/merge" "$tmp/gh.log"; then
  fail "case 2a: high-risk PR must not auto-merge"
fi
grep -q "POST repos/NeedTokenAi/needtoken-storefront/issues/42/comments" "$tmp/gh.log" \
  || fail "case 2a: expected risk gate comment"

printf '[{"id":901,"body":"<!-- needtoken-upstream-risk-gate -->\\nold"}]\n' >"$tmp/comments.json"
: >"$tmp/gh.log"
run_gate >/dev/null
grep -q "PATCH repos/NeedTokenAi/needtoken-storefront/issues/comments/901" "$tmp/gh.log" \
  || fail "case 2b: expected idempotent comment update"

# 3. 超过文件数阈值 -> fail-closed 拒绝合并
reset_mocks
printf '{"changed_files":2}\n' >"$tmp/pr_detail.json"
run_gate UPSTREAM_AUTO_MERGE_MAX_FILES=1 >/dev/null
if grep -q "pulls/42/merge" "$tmp/gh.log"; then
  fail "case 3a: PR exceeding max files must not auto-merge"
fi

# 3b. API 文件数与 changed_files 不一致（截断保护）-> fail-closed 拒绝合并
reset_mocks
printf '{"changed_files":5}\n' >"$tmp/pr_detail.json"
run_gate >/dev/null
if grep -q "pulls/42/merge" "$tmp/gh.log"; then
  fail "case 3b: truncated file list must not auto-merge"
fi

# 4. CI 仍有 in_progress 或失败 -> 拒绝合并
reset_mocks
cat >"$tmp/check_runs.json" <<'EOF'
{"check_runs":[
  {"id":10,"name":"Verify installer","status":"completed","conclusion":"success"},
  {"id":11,"name":"Verify API","status":"in_progress","conclusion":null},
  {"id":12,"name":"Verify release config","status":"completed","conclusion":"success"},
  {"id":13,"name":"Verify fullstack build","status":"completed","conclusion":"success"}
]}
EOF
run_gate >/dev/null
if grep -q "pulls/42/merge" "$tmp/gh.log"; then
  fail "case 4a: in_progress CI must not auto-merge"
fi

reset_mocks
cat >"$tmp/check_runs.json" <<'EOF'
{"check_runs":[
  {"id":10,"name":"Verify installer","status":"completed","conclusion":"success"},
  {"id":11,"name":"Verify API","status":"completed","conclusion":"failure"},
  {"id":12,"name":"Verify release config","status":"completed","conclusion":"success"},
  {"id":13,"name":"Verify fullstack build","status":"completed","conclusion":"success"}
]}
EOF
run_gate >/dev/null
if grep -q "pulls/42/merge" "$tmp/gh.log"; then
  fail "case 4b: failed CI must not auto-merge"
fi

# 5. 非双父 merge commit（如 squash 后的单父提交）-> 拒绝合并
reset_mocks
printf '{"parents":[{"sha":"%s"}]}\n' "$BASE_SHA" >"$tmp/commit.json"
run_gate >/dev/null
if grep -q "pulls/42/merge" "$tmp/gh.log"; then
  fail "case 5: non-merge commit must not auto-merge"
fi

# 6. 熔断开关 UPSTREAM_AUTO_MERGE=false -> 拒绝合并
reset_mocks
run_gate UPSTREAM_AUTO_MERGE=false >/dev/null
if grep -q "pulls/42/merge" "$tmp/gh.log"; then
  fail "case 6: kill switch must disable auto-merge"
fi

echo "auto-merge-upstream ok"
