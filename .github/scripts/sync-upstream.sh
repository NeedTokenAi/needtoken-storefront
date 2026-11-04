#!/usr/bin/env bash
# 把上游一个固定提交用普通 merge commit 合到分支上，并开 PR。
# 不编译、不测试、不执行合并进来的代码。没有 GCP 凭证。
# 冲突时不推送，避免把冲突标记送上 fork，更不会推到 main。
set -euo pipefail

REPO=${GITHUB_REPOSITORY:?}
UPSTREAM_REPO=${UPSTREAM_REPO:-dujiao-next/dujiao-next}
SYNC_AUTH_MODE=${SYNC_AUTH_MODE:-github_token}

[[ $UPSTREAM_REPO =~ ^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$ ]] || {
  echo "unexpected UPSTREAM_REPO" >&2
  exit 1
}
[[ $SYNC_AUTH_MODE == pat || $SYNC_AUTH_MODE == github_token ]] || {
  echo "unexpected SYNC_AUTH_MODE" >&2
  exit 1
}

cd "$(git rev-parse --show-toplevel)"
upstream_url=${UPSTREAM_URL:-https://github.com/${UPSTREAM_REPO}.git}
script_dir=$(cd "$(dirname "$0")" && pwd)
resolver=${RESOLVE_UPSTREAM:-$script_dir/resolve-upstream.sh}

resolved=$("$resolver")
channel=$(printf '%s\n' "$resolved" | sed -n 's/^channel=//p')
tag=$(printf '%s\n' "$resolved" | sed -n 's/^tag=//p')
upstream_sha=$(printf '%s\n' "$resolved" | sed -n 's/^sha=//p')

[[ $channel == stable-tags || $channel == releases ]] || {
  echo "resolver returned unexpected channel" >&2
  exit 1
}
[[ $tag =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]] || {
  echo "resolver returned unexpected tag" >&2
  exit 1
}
[[ $upstream_sha =~ ^[0-9a-f]{40}$ ]] || {
  echo "resolver returned unexpected sha" >&2
  exit 1
}

log() { printf '%s\n' "$*"; }
note() {
  [[ -n ${GITHUB_STEP_SUMMARY:-} ]] || return 0
  printf '%s\n' "$1" >>"$GITHUB_STEP_SUMMARY"
}

git fetch --no-tags origin '+refs/heads/main:refs/remotes/origin/main'
our_sha=$(git rev-parse origin/main)
[[ $our_sha =~ ^[0-9a-f]{40}$ ]] || {
  echo "origin/main is not a sha" >&2
  exit 1
}

# 只要这一个提交，不 fetch 上游的全部 tag，后面的 push 也只推我们的分支。
git fetch --no-tags "$upstream_url" "$upstream_sha"
git cat-file -e "${upstream_sha}^{commit}"

note "### 上游同步"
note ""
note "- 通道: \`${channel}\`"
note "- 上游: \`${UPSTREAM_REPO}\` \`${tag}\` \`${upstream_sha}\`"
note "- 我们的 main: \`${our_sha}\`"

if git merge-base --is-ancestor "$upstream_sha" origin/main; then
  log "upstream ${tag} ${upstream_sha} is already contained in main ${our_sha}"
  note ""
  note "该上游提交已经在 main 里，本次不需要开 PR。"
  exit 0
fi

open_sync=$(gh pr list --repo "$REPO" --base main --state open --limit 100 \
  --json number,headRefName,url \
  --jq '.[] | select(.headRefName | startswith("sync/upstream/")) | "\(.number)\t\(.url)\t\(.headRefName)"')
if [[ -n $open_sync ]]; then
  log "an upstream sync PR is already open:"
  printf '%s\n' "$open_sync"
  note ""
  note "已有打开的同步 PR，本次不再开第二个："
  note ""
  note '```'
  note "$open_sync"
  note '```'
  exit 0
fi

branch="sync/upstream/${upstream_sha}"
export GIT_AUTHOR_NAME='github-actions[bot]'
export GIT_AUTHOR_EMAIL='41898282+github-actions[bot]@users.noreply.github.com'
export GIT_COMMITTER_NAME=$GIT_AUTHOR_NAME
export GIT_COMMITTER_EMAIL=$GIT_AUTHOR_EMAIL

open_conflict_issue() {
  local files=$1 merge_log=$2
  local existing body
  existing=$(gh issue list --repo "$REPO" --state open --limit 100 --json number,url,title \
    --jq ".[] | select(.title | contains(\"${upstream_sha}\")) | .url" | head -n 1)
  if [[ -n $existing ]]; then
    log "conflict issue already open: ${existing}"
    note ""
    note "合并冲突仍在。已有 issue: ${existing}"
    return 0
  fi
  body=$(mktemp)
  {
    printf '%s\n' "上游同步无法自动合并，需要 Codex 解决冲突。"
    printf '%s\n' ""
    printf '%s\n' "- 我们的 main: \`${our_sha}\`"
    printf '%s\n' "- 上游: \`${UPSTREAM_REPO}\` \`${tag}\` \`${upstream_sha}\`"
    printf '%s\n' "- 通道: \`${channel}\`"
    printf '%s\n' "- 分支（未推送）: \`${branch}\`"
    printf '%s\n' ""
    printf '%s\n' "冲突文件："
    printf '%s\n' ""
    printf '%s\n' '```'
    printf '%s\n' "${files:-（git 没有留下未合并路径；见下面的 merge 输出）}"
    printf '%s\n' '```'
    printf '%s\n' ""
    printf '%s\n' "merge 输出："
    printf '%s\n' ""
    printf '%s\n' '```'
    printf '%s\n' "$merge_log"
    printf '%s\n' '```'
    printf '%s\n' ""
    printf '%s\n' "不要把带冲突标记的分支推到 main，也不要 force-push。请从当前 main 合并该上游提交，解决冲突后另开 PR，并用 merge commit 合并（不要 squash，不要 rebase）。"
  } >"$body"
  gh issue create --repo "$REPO" --title "上游同步冲突：${tag} ${upstream_sha}" --body-file "$body"
  rm -f "$body"
}

ensure_ci() {
  local head=$1
  if [[ $SYNC_AUTH_MODE == pat ]]; then
    log "PR opened with SYNC_GITHUB_TOKEN; pull_request should start ci"
    note ""
    note "使用 SYNC_GITHUB_TOKEN 开了 PR，\`pull_request\` 会触发 ci。"
    return 0
  fi
  gh workflow run ci.yml --ref "$head"
  log "GITHUB_TOKEN cannot trigger pull_request workflows; dispatched ci on ${head}"
  note ""
  note "没有 SYNC_GITHUB_TOKEN。已对 \`${head}\` 执行 \`workflow_dispatch\` 触发 ci。默认 GITHUB_TOKEN 开的 PR 不会自动跑其他工作流。"
}

create_pr() {
  local head=$1
  local existing body url
  existing=$(gh pr list --repo "$REPO" --base main --head "$head" --state open --json url \
    --jq '.[0].url // empty')
  if [[ -n $existing ]]; then
    log "PR already open: ${existing}"
    note ""
    note "PR: ${existing}"
    ensure_ci "$head"
    return 0
  fi
  body=$(mktemp)
  cat >"$body" <<EOF
把上游 \`${UPSTREAM_REPO}\` 的 \`${tag}\`（\`${upstream_sha}\`）用普通 merge commit 合入 main。

- 通道: \`${channel}\`（不是 \`/releases/latest\`）
- 开分支时的 main: \`${our_sha}\`
- 头分支: \`${head}\`

## 合并方式

请使用 **Create a merge commit**。不要 Squash，不要 Rebase。Squash 会丢掉这次合并的两个父提交，下一次同步无法判断上游提交是否已经在 main 里。

## CI

同步任务本身不编译、不执行合并进来的代码。请等 ci（installer / api / release-config / fullstack）在这个 SHA 上通过后再合并。

生产部署只在该提交进入 main 且 push 触发的 ci 成功之后才会发生。合并前请确认 Cloud Build 触发器 \`needtoken-storefront-asia-southeast1-NeedTokenAi\` 不会和 Actions 抢着切 100% 流量。涉及资金或数据库迁移时，先在隔离数据库演练。
EOF
  url=$(gh pr create --repo "$REPO" --base main --head "$head" --title "sync: upstream ${UPSTREAM_REPO} ${tag}" --body-file "$body")
  rm -f "$body"
  log "opened ${url}"
  note ""
  note "PR: ${url}"
  ensure_ci "$head"
}

if git ls-remote --exit-code --heads origin "$branch" >/dev/null 2>&1; then
  git fetch --no-tags origin "+refs/heads/${branch}:refs/remotes/origin/${branch}"
  if git merge-base --is-ancestor "$our_sha" "origin/${branch}" \
    && git merge-base --is-ancestor "$upstream_sha" "origin/${branch}"; then
    log "remote ${branch} already contains main and upstream; opening a PR without pushing"
    create_pr "$branch"
    exit 0
  fi
  echo "remote ${branch} exists but is not a merge of current main and ${upstream_sha}; refusing to force-push" >&2
  note ""
  note "远程分支 \`${branch}\` 已存在，且不是当前 main 与该上游提交的合并。没有 force-push。请人工删掉废弃分支后再跑。"
  exit 1
fi

# --no-track：起点是 origin/main 时，默认跟踪会把下一次不带 refspec 的 push 指到 main。
git checkout --no-track -B "$branch" origin/main
merge_msg="merge: upstream ${UPSTREAM_REPO} ${tag} (${upstream_sha})"
merge_log=$(mktemp)
set +e
GIT_MERGE_AUTOEDIT=no git merge --no-ff "$upstream_sha" -m "$merge_msg" >"$merge_log" 2>&1
merge_rc=$?
set -e
cat "$merge_log"

if [[ $merge_rc -ne 0 ]]; then
  files=$(git diff --name-only --diff-filter=U || true)
  git merge --abort >/dev/null 2>&1 || git reset --hard origin/main
  git checkout --detach origin/main
  git branch -D "$branch" >/dev/null 2>&1 || true
  open_conflict_issue "$files" "$(cat "$merge_log")"
  rm -f "$merge_log"
  note ""
  note "合并冲突，没有推送分支。Codex 需要按 issue 里的文件列表解决。"
  exit 1
fi
rm -f "$merge_log"

git diff --quiet
git diff --cached --quiet
[[ $(git rev-parse 'HEAD^1') == "$our_sha" ]] || {
  echo "merge commit first parent is not our main" >&2
  exit 1
}
[[ $(git rev-parse 'HEAD^2') == "$upstream_sha" ]] || {
  echo "merge commit second parent is not the upstream sha" >&2
  exit 1
}
git fetch --no-tags origin '+refs/heads/main:refs/remotes/origin/main'
if [[ $(git rev-parse origin/main) != "$our_sha" ]]; then
  log "origin/main moved during sync; not pushing ${branch}"
  note ""
  note "同步过程中 main 又更新了，没有推送。下次调度会重试。"
  exit 0
fi

# 只推这个分支，不推 tag，也不推其它 ref。
git push origin "refs/heads/${branch}:refs/heads/${branch}"
create_pr "$branch"
