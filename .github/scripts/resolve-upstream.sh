#!/usr/bin/env bash
# 把 UPSTREAM_CHANNEL 解析成一个固定的上游 tag 和 commit SHA。
# 不调用 /releases/latest：那个指针可以改，而且 Release 对象的 target_commitish
# 经常是分支名（v1.5.2 返回的是 main），不是提交。
set -euo pipefail

root=$(cd "$(dirname "$0")" && pwd)
# shellcheck source=lib/stable-tag.sh
source "$root/lib/stable-tag.sh"

CHANNEL=${UPSTREAM_CHANNEL:-stable-tags}
REPO=${UPSTREAM_REPO:-dujiao-next/dujiao-next}

case "$CHANNEL" in
  stable-tags | releases) ;;
  *)
    printf 'unsupported UPSTREAM_CHANNEL=%s (expected stable-tags or releases)\n' "$CHANNEL" >&2
    exit 1
    ;;
esac

tmp=$(mktemp)
trap 'rm -f "$tmp"' EXIT

if [[ $CHANNEL == stable-tags ]]; then
  # 只读 tag 名。调用方稍后单独 fetch 选中的那一个提交，这里不把 tag 写进 fork。
  gh api --paginate "repos/${REPO}/git/matching-refs/tags/v" --jq '.[].ref' \
    | sed 's#^refs/tags/##' >"$tmp"
else
  # 非 draft、非 prerelease，再用稳定 semver 过滤。仍然不看 "latest" 旗标。
  gh api --paginate "repos/${REPO}/releases?per_page=100" \
    --jq '.[] | select(.draft == false and .prerelease == false) | .tag_name' >"$tmp"
fi

tag=$(select_stable_tag <"$tmp")
if [[ -z $tag ]]; then
  echo "no stable semver tag found for ${REPO} via ${CHANNEL}" >&2
  exit 1
fi
[[ $tag =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]] || {
  printf 'refusing unexpected tag %q\n' "$tag" >&2
  exit 1
}

obj_type=$(gh api "repos/${REPO}/git/ref/tags/${tag}" --jq '.object.type')
obj_sha=$(gh api "repos/${REPO}/git/ref/tags/${tag}" --jq '.object.sha')
if [[ $obj_type == tag ]]; then
  # 附注 tag：再剥一层到 commit。轻量 tag 的 object.type 直接就是 commit。
  peel_type=$(gh api "repos/${REPO}/git/tags/${obj_sha}" --jq '.object.type')
  peel_sha=$(gh api "repos/${REPO}/git/tags/${obj_sha}" --jq '.object.sha')
  obj_type=$peel_type
  obj_sha=$peel_sha
fi

[[ $obj_type == commit ]] || {
  printf 'tag %s resolved to %s, want commit\n' "$tag" "$obj_type" >&2
  exit 1
}
[[ $obj_sha =~ ^[0-9a-f]{40}$ ]] || {
  printf 'tag %s resolved to invalid sha %q\n' "$tag" "$obj_sha" >&2
  exit 1
}

printf 'channel=%s\n' "$CHANNEL"
printf 'tag=%s\n' "$tag"
printf 'sha=%s\n' "$obj_sha"
