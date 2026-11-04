#!/usr/bin/env bash
# 对尚未承接生产流量的修订版本做只读冒烟。
# 只请求 /health、首页和公开配置。不导入商品，不改销售开关，不带管理端凭证。
set -euo pipefail

BASE=${1:?revision base URL}
BASE=${BASE%/}
[[ $BASE == https://* ]] || {
  printf 'smoke URL must be https, got %q\n' "$BASE" >&2
  exit 1
}

out_json=${SMOKE_JSON:-smoke-result.json}
health=fail
homepage=fail
public_config=fail
detail=""

write() {
  jq -n \
    --arg base "$BASE" \
    --arg health "$health" \
    --arg homepage "$homepage" \
    --arg public_config "$public_config" \
    --arg detail "$detail" \
    '{base_url:$base,health:$health,homepage:$homepage,public_config:$public_config,detail:$detail}' \
    >"$out_json"
}

finish() {
  local rc=$1
  write
  exit "$rc"
}

fetch() {
  local path=$1 dest=$2 accept=$3
  local attempt code
  for attempt in 1 2 3 4 5 6 7 8 9 10 11 12; do
    code=$(curl -sS -L --compressed --max-time 20 \
      -H "Accept: ${accept}" \
      -A "needtoken-deploy-smoke" \
      -o "$dest" -w '%{http_code}' "${BASE}${path}" || true)
    if [[ $code == 200 ]]; then
      return 0
    fi
    detail="${path} HTTP ${code:-curl-error} (attempt ${attempt})"
    sleep 5
  done
  return 1
}

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

if ! fetch /health "$tmp/health.json" 'application/json'; then
  detail="GET /health failed: ${detail}"
  finish 1
fi
if ! jq -e '.status == "ok"' "$tmp/health.json" >/dev/null; then
  detail=$(printf 'GET /health body: %s' "$(head -c 300 "$tmp/health.json")")
  finish 1
fi
health=pass

if ! fetch / "$tmp/home.html" 'text/html'; then
  detail="GET / failed: ${detail}"
  finish 1
fi
if ! grep -q 'id="app"' "$tmp/home.html"; then
  detail="homepage HTML has no #app"
  finish 1
fi
if grep -q '/src/main.ts' "$tmp/home.html"; then
  detail="homepage still references /src/main.ts (dev shell, not a production build)"
  finish 1
fi
if ! grep -Eq '/assets/[A-Za-z0-9._-]+\.js' "$tmp/home.html"; then
  detail="homepage HTML does not reference a built /assets/*.js file"
  finish 1
fi
homepage=pass

if ! fetch /api/v1/public/config "$tmp/config.json" 'application/json'; then
  detail="GET /api/v1/public/config failed: ${detail}"
  finish 1
fi
status_code=$(jq -r '.status_code // empty' "$tmp/config.json")
template=$(jq -r '.data.storefront_template // empty' "$tmp/config.json")
if [[ $status_code != 0 || $template != needtoken ]]; then
  detail="public config status_code=${status_code:-missing} storefront_template=${template:-missing}"
  finish 1
fi
public_config=pass
detail="ok"
finish 0
