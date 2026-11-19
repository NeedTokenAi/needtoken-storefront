#!/usr/bin/env bash
# 稳定 tag 选择：不能被 beta/rc 混进去，也不能按字符串序把 v1.9 排到 v1.10 前面；
# 同时支持可选 min_tag 版本底线过滤。
set -euo pipefail

root=$(cd "$(dirname "$0")/.." && pwd)
# shellcheck source=../lib/stable-tag.sh
source "$root/lib/stable-tag.sh"

got=$(printf '%s\n' v0.0.9-beta v1.4.9 v1.5.2 v1.5.2-rc.1 v1.10.0 v1.9.9 v1.10.0-rc.1 latest | select_stable_tag)
[[ $got == v1.10.0 ]] || {
  printf 'expected v1.10.0, got %q\n' "$got" >&2
  exit 1
}

got=$(printf '%s\n' v1.5.2 v1.5.1 v0.0.8-beta | select_stable_tag)
[[ $got == v1.5.2 ]] || {
  printf 'expected v1.5.2, got %q\n' "$got" >&2
  exit 1
}

if printf '%s\n' v1.2.3-rc.1 v0.0.9-beta | select_stable_tag | grep -q .; then
  echo "prerelease-only input must not yield a tag" >&2
  exit 1
fi

# 带 min_tag 底线：老分支 v0.13.2 低于 v1.0.0 时必须返回空
if printf '%s\n' v0.13.2 v1.0.0-rc.51 | select_stable_tag v1.0.0 | grep -q .; then
  echo "tag below min_tag must not be selected" >&2
  exit 1
fi

got=$(printf '%s\n' v0.13.2 v1.0.0-rc.51 v1.0.0 | select_stable_tag v1.0.0)
[[ $got == v1.0.0 ]] || {
  printf 'expected v1.0.0 when equal to min_tag, got %q\n' "$got" >&2
  exit 1
}

got=$(printf '%s\n' v0.13.2 v1.0.1 v1.2.0 | select_stable_tag v1.0.0)
[[ $got == v1.2.0 ]] || {
  printf 'expected v1.2.0 above min_tag, got %q\n' "$got" >&2
  exit 1
}

echo "stable-tag ok"
