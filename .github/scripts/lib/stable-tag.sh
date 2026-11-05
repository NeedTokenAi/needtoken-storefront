#!/usr/bin/env bash
# 从 stdin 的 tag 列表里选出最高的稳定语义化版本。
# 只接受 vMAJOR.MINOR.PATCH。v1.5.2-rc.1、v0.0.9-beta 这类预发布标签会被丢掉。
# 依赖 GNU sort -V（GitHub Actions ubuntu-latest 自带）。

select_stable_tag() {
  local picked
  picked=$(grep -E '^v[0-9]+\.[0-9]+\.[0-9]+$' | LC_ALL=C sort -V | tail -n 1 || true)
  printf '%s\n' "$picked"
}
