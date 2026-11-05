#!/usr/bin/env bash
# 干净合并会推分支并要求开 PR；冲突不推送，并留下给 Codex 的说明。
set -euo pipefail

root=$(cd "$(dirname "$0")/.." && pwd)
sync=$root/sync-upstream.sh
fail() {
  printf 'FAIL: %s\n' "$*" >&2
  exit 1
}

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

cat >"$tmp/gh" <<'EOS'
#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' "$*" >>"${GH_LOG:?}"
sub=${1:-}
action=${2:-}
shift || true
shift || true
jqfilter=""
while [[ $# -gt 0 ]]; do
  case $1 in
    --jq) jqfilter=${2:-}; shift 2 ;;
    --body-file) cp "$2" "${GH_BODY:?}"; shift 2 ;;
    *) shift ;;
  esac
done
if [[ $sub == pr && $action == create ]]; then
  printf '%s\n' "https://example.test/pr/1"
  exit 0
fi
if [[ $sub == issue && $action == create ]]; then
  printf '%s\n' "https://example.test/issue/1"
  exit 0
fi
if [[ $sub == workflow ]]; then
  exit 0
fi
if [[ -n $jqfilter ]]; then
  jq -r "$jqfilter" <<<"[]"
fi
EOS
chmod +x "$tmp/gh"

cat >"$tmp/resolve.sh" <<'EOS'
#!/usr/bin/env bash
set -euo pipefail
printf 'channel=stable-tags\n'
printf 'tag=v9.9.9\n'
printf 'sha=%s\n' "${UPSTREAM_SHA:?}"
EOS
chmod +x "$tmp/resolve.sh"

prepare() {
  local name=$1
  git init --bare -q -b main "$tmp/$name-our.git"
  git init --bare -q -b main "$tmp/$name-up.git"
  git -C "$tmp/$name-up.git" config uploadpack.allowReachableSHA1InWant true
  git -C "$tmp/$name-up.git" config uploadpack.allowAnySHA1InWant true
  git init -q -b main "$tmp/$name-src"
  git -C "$tmp/$name-src" config user.email "test@example.com"
  git -C "$tmp/$name-src" config user.name "test"
  printf 'base\n' >"$tmp/$name-src/shared.txt"
  printf 'ours\n' >"$tmp/$name-src/ours.txt"
  git -C "$tmp/$name-src" add shared.txt ours.txt
  git -C "$tmp/$name-src" commit -q -m base
  git -C "$tmp/$name-src" remote add origin "$tmp/$name-our.git"
  git -C "$tmp/$name-src" remote add upstream "$tmp/$name-up.git"
  git -C "$tmp/$name-src" push -q origin main
  git -C "$tmp/$name-src" push -q upstream main
}

prepare clean
git clone -q "$tmp/clean-up.git" "$tmp/clean-upstream-work"
git -C "$tmp/clean-upstream-work" config user.email "test@example.com"
git -C "$tmp/clean-upstream-work" config user.name "test"
printf 'upstream\n' >"$tmp/clean-upstream-work/upstream.txt"
git -C "$tmp/clean-upstream-work" add upstream.txt
git -C "$tmp/clean-upstream-work" commit -q -m upstream
git -C "$tmp/clean-upstream-work" push -q origin HEAD:main
clean_sha=$(git -C "$tmp/clean-upstream-work" rev-parse HEAD)

git -C "$tmp/clean-src" commit -q --allow-empty -m "our change"
git -C "$tmp/clean-src" push -q origin main
clean_main=$(git -C "$tmp/clean-src" rev-parse origin/main)

: >"$tmp/clean-gh.log"
set +e
(
  cd "$tmp/clean-src"
  UPSTREAM_SHA=$clean_sha \
  UPSTREAM_URL="$tmp/clean-up.git" \
  RESOLVE_UPSTREAM="$tmp/resolve.sh" \
  GITHUB_REPOSITORY=NeedTokenAi/needtoken-storefront \
  SYNC_AUTH_MODE=pat \
  GH_LOG="$tmp/clean-gh.log" \
  GH_BODY="$tmp/clean-body.md" \
  PATH="$tmp:$PATH" \
  bash "$sync"
)
clean_rc=$?
set -e
[[ $clean_rc -eq 0 ]] || fail "clean sync exit $clean_rc"
git --git-dir="$tmp/clean-our.git" rev-parse "refs/heads/sync/upstream/${clean_sha}" >/dev/null \
  || fail "clean branch was not pushed"
[[ $(git --git-dir="$tmp/clean-our.git" rev-parse refs/heads/main) == "$clean_main" ]] \
  || fail "sync pushed main"
parents=$(git --git-dir="$tmp/clean-our.git" rev-list --parents -n 1 "refs/heads/sync/upstream/${clean_sha}")
[[ $(wc -w <<<"$parents") -eq 3 ]] || fail "expected merge commit, got $parents"
grep -q 'pr create' "$tmp/clean-gh.log" || fail "PR was not created"
grep -q 'Create a merge commit' "$tmp/clean-body.md" || fail "PR body missing merge instructions"
if grep -q '^workflow ' "$tmp/clean-gh.log"; then
  fail "PAT mode should not dispatch ci"
fi

prepare conflict
git clone -q "$tmp/conflict-up.git" "$tmp/conflict-upstream-work"
git -C "$tmp/conflict-upstream-work" config user.email "test@example.com"
git -C "$tmp/conflict-upstream-work" config user.name "test"
printf 'upstream-edit\n' >"$tmp/conflict-upstream-work/shared.txt"
git -C "$tmp/conflict-upstream-work" add shared.txt
git -C "$tmp/conflict-upstream-work" commit -q -m upstream
git -C "$tmp/conflict-upstream-work" push -q origin HEAD:main
conflict_sha=$(git -C "$tmp/conflict-upstream-work" rev-parse HEAD)

printf 'our-edit\n' >"$tmp/conflict-src/shared.txt"
git -C "$tmp/conflict-src" add shared.txt
git -C "$tmp/conflict-src" commit -q -m ours
git -C "$tmp/conflict-src" push -q origin main
our_sha=$(git -C "$tmp/conflict-src" rev-parse HEAD)

: >"$tmp/conflict-gh.log"
set +e
(
  cd "$tmp/conflict-src"
  UPSTREAM_SHA=$conflict_sha \
  UPSTREAM_URL="$tmp/conflict-up.git" \
  RESOLVE_UPSTREAM="$tmp/resolve.sh" \
  GITHUB_REPOSITORY=NeedTokenAi/needtoken-storefront \
  SYNC_AUTH_MODE=github_token \
  GH_LOG="$tmp/conflict-gh.log" \
  GH_BODY="$tmp/conflict-body.md" \
  PATH="$tmp:$PATH" \
  bash "$sync"
)
conflict_rc=$?
set -e
[[ $conflict_rc -eq 1 ]] || fail "conflict sync exit $conflict_rc, want 1"
if git --git-dir="$tmp/conflict-our.git" rev-parse "refs/heads/sync/upstream/${conflict_sha}" >/dev/null 2>&1; then
  fail "conflict branch was pushed"
fi
grep -q 'issue create' "$tmp/conflict-gh.log" || fail "conflict issue was not opened"
grep -q 'Codex' "$tmp/conflict-body.md" || fail "issue does not mention Codex"
grep -q "$our_sha" "$tmp/conflict-body.md" || fail "issue missing our sha"
grep -q "$conflict_sha" "$tmp/conflict-body.md" || fail "issue missing upstream sha"
grep -q 'shared.txt' "$tmp/conflict-body.md" || fail "issue missing conflict file"
if grep -q '<<<<<<<' "$tmp/conflict-body.md"; then
  fail "issue should list files, not paste conflict markers from an index we aborted"
fi

echo "sync-merge ok"
