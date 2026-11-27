#!/usr/bin/env bash
# 校验 storefront 高危路径分类器（fail-closed）与低风险路径放行边界。
set -euo pipefail

root=$(cd "$(dirname "$0")/.." && pwd)
# shellcheck source=../lib/high-risk-paths.sh
source "$root/lib/high-risk-paths.sh"

fail() {
  printf 'FAIL: %s\n' "$*" >&2
  exit 1
}

high_risk_cases=(
  ".github/workflows/ci.yml"
  ".github/needtoken-fork-rules.md"
  "cloudflare/wrangler.jsonc"
  "Dockerfile"
  "docker-compose.yml"
  ".goreleaser.yaml"
  "config.yml.example"
  "internal/models/order.go"
  "internal/database/migrate.go"
  "internal/authz/bootstrap.go"
  "internal/modules/payment/service.go"
  "internal/modules/wallet/service.go"
  "internal/modules/order/service.go"
  "internal/modules/upstreamapi/handler.go"
  "internal/upstream/client.go"
  "internal/bootstrap/app.go"
  "internal/workflows/checkout.go"
  "internal/modules/catalog/product/store/gormstore/repo.go"
  "scripts/init.sql"
  "internal/modules/diagnostics/infrastructure/statuspage/series.go"
  "internal/constants/constants.go"
  "internal/app/container/bootstrap.go"
  "internal/app/container/bootstrap_test.go"
  "internal/modules/payment/infrastructure/gateway/adapters/moincash/adapter.go"
  "internal/modules/opsbridge/application/service.go"
  "internal/modules/apipublic/transport/http/handler.go"
  "internal/modules/identity/userauth/application/service.go"
  "internal/modules/giftcard/application/redeem.go"
  "internal/modules/giftcard/transport/http/admin_handler.go"
  "internal/app/httpserver/router.go"
  "internal/app/httpserver/routes_storefront.go"
  "internal/app/httpserver/routes_channel.go"
  "frontend/user/src/templates/needtoken/layout/NeedtokenLayout.vue"
  "frontend/user/src/templates/needtoken/Rankings.vue"
  "frontend/user/src/templates/needtoken/styles/needtoken.css"
  "frontend/user/src/templates/needtoken/AccountLinkCallback.vue"
  "frontend/user/src/router/index.ts"
  "frontend/user/src/api/order.ts"
  "frontend/user/src/api/auth.ts"
  "frontend/user/src/stores/userAuth.ts"
  "frontend/user/src/stores/userProfile.ts"
  "frontend/user/src/composables/usePersonalCenter.ts"
  "frontend/user/src/components/AssetVaultClaimPanel.vue"
  "frontend/user/src/components/SubscriptionAuthorizationPanel.vue"
  "frontend/user/src/utils/subscriptionAuthorization.ts"
  "frontend/user/src/views/OrderDetail.vue"
  "frontend/user/src/views/GuestOrderDetail.vue"
  "frontend/user/src/views/PersonalCenter.vue"
  "frontend/user/src/views/personal/GiftCardPanel.vue"
  "frontend/user/index.html"
  "frontend/user/src/App.vue"
  "frontend/user/public/dj.svg"
  "frontend/user/src/main.ts"
  "frontend/user/src/stores/app.ts"
  "frontend/user/src/utils/storefrontStartup.ts"
  "frontend/user/src/components/Footer.vue"
  "frontend/user/src/templates/vault/layout/VaultLayout.vue"
  "frontend/user/src/utils/theme.ts"
  "frontend/user/src/utils/themeState.ts"
  "internal/web/handler.go"
  "internal/modules/captcha/application/service.go"
  "internal/modules/identity/telegramauth/application/service.go"
  "internal/modules/identity/userauth/transport/http/user_login_handler.go"
  "frontend/user/src/components/captcha/TurnstileCaptcha.vue"
  "frontend/admin/src/components/captcha/TurnstileCaptcha.vue"
  "frontend/admin/src/views/Login.vue"
  "frontend/user/src/utils/chunkRecovery.ts"
  "frontend/admin/src/utils/chunkRecovery.ts"
  "internal/modules/radarsubscription/application/service.go"
  "internal/modules/notification/infrastructure/smtp/radar_reset_template.go"
  "internal/app/radar_notify_service.go"
  "internal/app/paid_fulfillment_service.go"
  "frontend/user/src/api/radar.ts"
  "frontend/user/src/api/wallet.ts"
  "frontend/user/src/style.css"
  "frontend/user/src/components/ui/select/SelectContent.vue"
  "frontend/user/src/composables/useCheckout.ts"
  "frontend/user/src/types/telegram-webapp.d.ts"
  "frontend/user/src/views/personal/WalletPanel.vue"
)

for path in "${high_risk_cases[@]}" "internal/modules/identity/user/contract/store.go"; do
  is_high_risk_path "$path" || fail "expected high-risk: $path"
done

low_risk_cases=(
  "README.md"
  "frontend/user/src/views/Home.vue"
  "frontend/admin/src/i18n/index.ts"
  "internal/i18n/messages.go"
  "internal/modules/catalog/product/application/service.go"
  "internal/shared/timeutil/clock.go"
)

for path in "${low_risk_cases[@]}"; do
  if is_high_risk_path "$path"; then
    fail "expected low-risk: $path"
  fi
done

filtered=$(printf '%s\n' "README.md" "internal/modules/payment/a.go" "frontend/user/src/views/Home.vue" ".github/workflows/ci.yml" | filter_high_risk_paths)
expected=$(printf '%s\n' "internal/modules/payment/a.go" ".github/workflows/ci.yml")
[[ $filtered == "$expected" ]] || fail "unexpected filter output: $filtered"

echo "high-risk-paths ok"
