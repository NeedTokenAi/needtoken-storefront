#!/usr/bin/env bash
# 判定上游同步 PR 改动路径是否命中高危路径（fail-closed）。
# 命中任意高危路径时禁止自动合并，必须人工审阅后再用 Merge Commit 合并。

is_high_risk_path() {
  local path=$1
  case "$path" in
    # CI / 工作流 / 供应链与规则文件
    .github/*)
      return 0
      ;;
    # 容器 / Cloudflare 部署边界 / 发布与示例配置
    cloudflare/* | Dockerfile | Dockerfile.* | */Dockerfile | */Dockerfile.* | \
      docker-compose*.yml | compose*.yml | .goreleaser*.yaml | config.yml.example)
      return 0
      ;;
    # 任意 SQL 脚本 / 迁移
    *.sql | */migrations/* | migrations/*)
      return 0
      ;;
    # 数据库模型、存储层、鉴权、资金/订单/钱包/上游契约与装配
    internal/models/* | internal/database/* | internal/authz/* | \
      internal/modules/payment/* | internal/modules/wallet/* | internal/modules/order/* | \
      internal/modules/upstreamapi/* | internal/upstream/* | \
      internal/bootstrap/* | internal/workflows/* | */gormstore/*)
      return 0
      ;;
    # NeedToken 二开保护区
    internal/modules/diagnostics/* | internal/app/httpserver/routes_storefront.go | \
      frontend/user/src/templates/needtoken/* | frontend/user/index.html | \
      frontend/user/src/router/index.ts | frontend/user/src/views/OrderDetail.vue | \
      frontend/user/src/views/PersonalCenter.vue | \
      frontend/user/src/utils/theme.ts | frontend/user/src/utils/themeState.ts)
      return 0
      ;;
    *)
      return 1
      ;;
  esac
}

filter_high_risk_paths() {
  local line
  while IFS= read -r line; do
    [[ -n $line ]] || continue
    if is_high_risk_path "$line"; then
      printf '%s\n' "$line"
    fi
  done
}
