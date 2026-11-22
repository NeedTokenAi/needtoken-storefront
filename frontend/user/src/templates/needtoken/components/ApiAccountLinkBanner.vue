<template>
  <section
    v-if="visible"
    class="nt-account-link"
    :class="{
      'is-compact': compact,
      'is-warning': hasPendingOrders || requiresExistingManualLink,
      'is-linked': isLinked,
    }"
    :aria-label="t('needtoken.apiAccountLink.title')"
  >
    <div class="nt-account-link__icon" aria-hidden="true">
      <AlertTriangle v-if="hasPendingOrders || requiresExistingManualLink" :size="18" />
      <CheckCircle2 v-else-if="isLinked" :size="18" />
      <Sparkles v-else :size="18" />
    </div>

    <div class="nt-account-link__body">
      <div class="nt-account-link__head">
        <h3>{{ headingText }}</h3>
        <span v-if="isLinked && displayNativeUserId" class="nt-account-link__badge">
          API #{{ displayNativeUserId }}
        </span>
        <span v-if="isLinked && summary?.group" class="nt-account-link__group-badge">
          {{ t('needtoken.apiAccountLink.groupBadge', { group: summary.group }) }}
        </span>
      </div>
      <p>{{ descriptionText }}</p>

      <!-- 已绑定 API 权益摘要卡（非紧凑模式展示额度、分组与活跃订阅） -->
      <div v-if="isLinked && !compact" class="nt-account-link__summary">
        <div v-if="summaryLoading && !summary" class="nt-account-link__summary-loading">
          <LoaderCircle :size="13" class="nt-account-link__spin" aria-hidden="true" />
          <span>{{ t('needtoken.apiAccountLink.summaryLoading') }}</span>
        </div>

        <template v-else-if="summary && summary.linked">
          <dl class="nt-account-link__metrics">
            <div class="nt-account-link__metric">
              <dt>{{ t('needtoken.apiAccountLink.remainingQuota') }}</dt>
              <dd class="nt-mono is-emphasis">{{ formatApiQuotaUsd(summary.quota) }}</dd>
            </div>
            <div class="nt-account-link__metric">
              <dt>{{ t('needtoken.apiAccountLink.usedQuota') }}</dt>
              <dd class="nt-mono">{{ formatApiQuotaUsd(summary.used_quota) }}</dd>
            </div>
            <div class="nt-account-link__metric">
              <dt>{{ t('needtoken.apiAccountLink.currentGroup') }}</dt>
              <dd class="nt-mono">{{ summary.group || 'default' }}</dd>
            </div>
            <div class="nt-account-link__metric">
              <dt>{{ t('needtoken.apiAccountLink.requestCount') }}</dt>
              <dd class="nt-mono">{{ formatRequestCount(summary.request_count) }}</dd>
            </div>
          </dl>

          <div v-if="activeSubscriptions.length > 0" class="nt-account-link__subs">
            <p class="nt-account-link__subs-title">{{ t('needtoken.apiAccountLink.activeSubscriptionsTitle') }}</p>
            <ul class="nt-account-link__subs-list">
              <li v-for="sub in activeSubscriptions" :key="sub.id" class="nt-account-link__sub-item">
                <div class="nt-account-link__sub-top">
                  <span class="nt-account-link__sub-name">
                    {{ sub.plan_title || t('needtoken.apiAccountLink.subscriptionFallbackName', { id: sub.plan_id }) }}
                  </span>
                  <span class="nt-account-link__sub-expiry">
                    {{ subscriptionExpiryText(sub.end_time) }}
                  </span>
                </div>
                <div class="nt-account-link__sub-bar" aria-hidden="true">
                  <div
                    class="nt-account-link__sub-bar-fill"
                    :style="{ width: `${computeSubscriptionUsagePercent(sub.amount_total, sub.amount_used)}%` }"
                  />
                </div>
                <div class="nt-account-link__sub-meta nt-mono">
                  <span>
                    {{
                      t('needtoken.apiAccountLink.subscriptionRemaining', {
                        remaining: formatApiQuotaUsd(computeSubscriptionRemainingQuota(sub.amount_total, sub.amount_used)),
                        total: formatApiQuotaUsd(sub.amount_total),
                      })
                    }}
                  </span>
                  <span v-if="sub.allow_wallet_overflow" class="nt-account-link__sub-overflow">
                    {{ t('needtoken.apiAccountLink.walletOverflowEnabled') }}
                  </span>
                </div>
              </li>
            </ul>
          </div>
        </template>
      </div>

      <div v-if="isLinked" class="nt-account-link__shortcuts" role="group" :aria-label="t('needtoken.apiAccountLink.shortcutsLabel')">
        <button
          v-for="item in quickLinks"
          :key="item.target"
          type="button"
          class="nt-account-link__chip"
          :disabled="busy"
          @click="handleSSO(item.target)"
        >
          <LoaderCircle v-if="ssoLoading && ssoTarget === item.target" :size="12" class="nt-account-link__spin" aria-hidden="true" />
          <component :is="item.icon" v-else :size="12" aria-hidden="true" />
          <span>{{ t(item.labelKey) }}</span>
          <ArrowUpRight :size="11" aria-hidden="true" />
        </button>
      </div>

      <p v-if="errorMessage" class="nt-account-link__error" role="alert">
        {{ errorMessage || t('needtoken.apiAccountLink.startFailed') }}
      </p>
    </div>

    <div class="nt-account-link__actions">
      <template v-if="!isLinked">
        <template v-if="requiresExistingManualLink">
          <button
            type="button"
            class="nt-account-link__cta"
            :disabled="busy"
            @click="handleManualBind"
          >
            <LoaderCircle v-if="starting" :size="15" class="nt-account-link__spin" aria-hidden="true" />
            <Link2 v-else :size="15" aria-hidden="true" />
            <span>{{ starting ? t('needtoken.apiAccountLink.redirecting') : t('needtoken.apiAccountLink.confirmExistingLinkAction') }}</span>
          </button>
        </template>
        <template v-else>
          <button
            type="button"
            class="nt-account-link__cta"
            :disabled="busy"
            @click="handleSSO('/dashboard')"
          >
            <LoaderCircle v-if="ssoLoading" :size="15" class="nt-account-link__spin" aria-hidden="true" />
            <Zap v-else :size="15" aria-hidden="true" />
            <span>{{ ssoLoading ? t('needtoken.apiAccountLink.ssoRedirecting') : ctaLabel }}</span>
          </button>
          <button
            type="button"
            class="nt-account-link__secondary"
            :disabled="busy"
            @click="handleManualBind"
          >
            <LoaderCircle v-if="starting" :size="13" class="nt-account-link__spin" aria-hidden="true" />
            <Link2 v-else :size="13" aria-hidden="true" />
            <span>{{ starting ? t('needtoken.apiAccountLink.redirecting') : t('needtoken.apiAccountLink.bindExistingAction') }}</span>
          </button>
        </template>
      </template>

      <button
        v-else
        type="button"
        class="nt-account-link__cta is-sso"
        :disabled="busy"
        @click="handleSSO('/dashboard')"
      >
        <LoaderCircle v-if="ssoLoading && ssoTarget === '/dashboard'" :size="15" class="nt-account-link__spin" aria-hidden="true" />
        <ArrowUpRight v-else :size="15" aria-hidden="true" />
        <span>{{ ssoLoading && ssoTarget === '/dashboard' ? t('needtoken.apiAccountLink.ssoRedirecting') : t('needtoken.apiAccountLink.enterConsoleSSO') }}</span>
      </button>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  Activity,
  AlertTriangle,
  ArrowUpRight,
  CheckCircle2,
  CreditCard,
  KeyRound,
  Link2,
  LoaderCircle,
  Sparkles,
  Zap,
} from 'lucide-vue-next'
import { useUserAuthStore } from '../../../stores/userAuth'
import { useAppStore } from '../../../stores/app'
import { getActiveTemplate } from '../../registry'
import { resetApiAccountLinkCache, useApiAccountLink } from '../composables/useApiAccountLink'
import {
  computeSubscriptionRemainingQuota,
  computeSubscriptionUsagePercent,
  formatApiQuotaUsd,
  formatSubscriptionExpiry,
  type ApiConsoleTargetPath,
} from '../lib/apiAccountLinkPolicy'

const props = withDefaults(
  defineProps<{
    nextPath?: string
    compact?: boolean
    onlyWhenActionNeeded?: boolean
  }>(),
  {
    nextPath: '/me',
    compact: false,
    onlyWhenActionNeeded: false,
  },
)

const { t, locale } = useI18n()
const userAuth = useUserAuthStore()
const appStore = useAppStore()
const {
  status,
  summary,
  ensureResult,
  summaryLoading,
  starting,
  ssoLoading,
  ssoTarget,
  errorMessage,
  load,
  ensure,
  loadSummary,
  beginLinking,
  openApiConsoleSSO,
} = useApiAccountLink()

const quickLinks: readonly { target: ApiConsoleTargetPath; labelKey: string; icon: typeof KeyRound }[] = [
  { target: '/keys', labelKey: 'needtoken.apiAccountLink.shortcutKeys', icon: KeyRound },
  { target: '/dashboard', labelKey: 'needtoken.apiAccountLink.shortcutDashboard', icon: Activity },
  { target: '/subscriptions', labelKey: 'needtoken.apiAccountLink.shortcutSubscriptions', icon: CreditCard },
]

const syncLinkAndSummary = async () => {
  const current = await load()
  if (!current || !current.enabled) {
    return
  }
  if (!current.linked) {
    const ensured = await ensure()
    if (ensured?.linked) {
      await loadSummary(true)
    }
    return
  }
  if (!props.compact) {
    await loadSummary()
  }
}

watch(
  () => [userAuth.isAuthenticated, appStore.isResellerTenant] as const,
  ([authed, reseller]) => {
    if (!authed || reseller || getActiveTemplate() !== 'needtoken') {
      if (!authed) resetApiAccountLinkCache()
      return
    }
    void syncLinkAndSummary()
  },
  { immediate: true },
)

const isLinked = computed(() => Boolean(status.value?.linked))
const displayNativeUserId = computed(() => status.value?.native_user_id || summary.value?.native_user_id || 0)
const pendingOrders = computed(() => status.value?.pending_orders ?? 0)
const hasPendingOrders = computed(() => !isLinked.value && pendingOrders.value > 0)
const requiresExistingManualLink = computed(() => !isLinked.value && ensureResult.value?.status === 'existing_requires_link')
const activeSubscriptions = computed(() => summary.value?.active_subscriptions ?? [])
const busy = computed(() => starting.value || ssoLoading.value)

const visible = computed(() => {
  if (getActiveTemplate() !== 'needtoken' || appStore.isResellerTenant || !userAuth.isAuthenticated) {
    return false
  }
  if (!status.value || !status.value.enabled) {
    return false
  }
  if (props.onlyWhenActionNeeded && isLinked.value) {
    return false
  }
  return true
})

const headingText = computed(() => {
  if (hasPendingOrders.value) {
    return t('needtoken.apiAccountLink.pendingHeading', { count: pendingOrders.value })
  }
  if (requiresExistingManualLink.value) {
    return t('needtoken.apiAccountLink.existingRequiresLinkHeading')
  }
  if (isLinked.value) {
    return t('needtoken.apiAccountLink.linkedHeading')
  }
  return t('needtoken.apiAccountLink.unlinkedHeading')
})

const descriptionText = computed(() => {
  if (hasPendingOrders.value) {
    return t('needtoken.apiAccountLink.pendingDescription')
  }
  if (requiresExistingManualLink.value) {
    return t('needtoken.apiAccountLink.existingRequiresLinkDescription')
  }
  if (isLinked.value) {
    return t('needtoken.apiAccountLink.linkedDescription')
  }
  return t('needtoken.apiAccountLink.unlinkedDescription')
})

const ctaLabel = computed(() =>
  hasPendingOrders.value
    ? t('needtoken.apiAccountLink.bindNowToActivate')
    : t('needtoken.apiAccountLink.oneClickOpenConsole'),
)

const formatRequestCount = (count?: number | null) => {
  const n = typeof count === 'number' && Number.isFinite(count) ? Math.max(0, Math.trunc(count)) : 0
  return n.toLocaleString('en-US')
}

const subscriptionExpiryText = (endTime?: number | null) => {
  const formatted = formatSubscriptionExpiry(endTime, String(locale.value || 'zh-CN'))
  if (!formatted) return t('needtoken.apiAccountLink.subscriptionPermanent')
  return t('needtoken.apiAccountLink.subscriptionValidUntil', { date: formatted })
}

const handleSSO = (target: ApiConsoleTargetPath) => {
  void openApiConsoleSSO(target, props.nextPath)
}

const handleManualBind = () => {
  void beginLinking(props.nextPath)
}
</script>

<style scoped>
.nt-account-link {
  display: flex;
  align-items: flex-start;
  gap: 14px;
  padding: 16px 18px;
  border: 1px solid var(--nt-accent-border, hsl(var(--border)));
  border-radius: 16px;
  background: var(--nt-accent-subtle, hsl(var(--card)));
  color: var(--ui-text-primary, hsl(var(--foreground)));
}
.nt-account-link.is-compact {
  padding: 12px 14px;
  border-radius: 12px;
}
.nt-account-link.is-warning {
  border-color: rgba(217, 119, 6, 0.45);
  background: rgba(245, 158, 11, 0.08);
}
.nt-account-link.is-linked {
  border-color: rgba(16, 185, 129, 0.34);
  background: rgba(16, 185, 129, 0.06);
}
.nt-account-link__icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  margin-top: 2px;
  flex: none;
  border-radius: 10px;
  background: rgba(255, 255, 255, 0.6);
  color: var(--ui-accent, hsl(var(--primary)));
}
.dark .nt-account-link__icon {
  background: rgba(255, 255, 255, 0.06);
}
.nt-account-link.is-warning .nt-account-link__icon {
  color: #d97706;
}
.nt-account-link.is-linked .nt-account-link__icon {
  color: #059669;
}
.nt-account-link__body {
  min-width: 0;
  flex: 1;
}
.nt-account-link__head {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
}
.nt-account-link__head h3 {
  font-size: 13.5px;
  font-weight: 680;
  line-height: 1.4;
}
.nt-account-link__badge {
  display: inline-flex;
  align-items: center;
  padding: 1px 8px;
  border-radius: 999px;
  border: 1px solid rgba(16, 185, 129, 0.35);
  font-family: var(--nt-font-mono, monospace);
  font-size: 11px;
  font-weight: 600;
  color: #059669;
}
.nt-account-link__group-badge {
  display: inline-flex;
  align-items: center;
  padding: 1px 8px;
  border-radius: 999px;
  border: 1px solid var(--ui-border, hsl(var(--border)));
  background: var(--ui-bg-elevated, hsl(var(--card)));
  font-family: var(--nt-font-mono, monospace);
  font-size: 11px;
  font-weight: 600;
  color: var(--ui-text-secondary, hsl(var(--muted-foreground)));
}
.nt-account-link__body p {
  margin-top: 4px;
  font-size: 12px;
  line-height: 1.65;
  color: var(--ui-text-secondary, hsl(var(--muted-foreground)));
}
.nt-account-link__summary {
  margin-top: 12px;
}
.nt-account-link__summary-loading {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: var(--ui-text-secondary, hsl(var(--muted-foreground)));
}
.nt-account-link__metrics {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 8px;
}
.nt-account-link__metric {
  padding: 9px 11px;
  border-radius: 10px;
  border: 1px solid rgba(16, 185, 129, 0.22);
  background: var(--ui-bg-elevated, hsl(var(--card)));
}
.nt-account-link__metric dt {
  font-size: 11px;
  color: var(--ui-text-secondary, hsl(var(--muted-foreground)));
}
.nt-account-link__metric dd {
  margin-top: 2px;
  font-size: 14px;
  font-weight: 700;
  color: var(--ui-text-primary, hsl(var(--foreground)));
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.nt-account-link__metric dd.is-emphasis {
  color: #059669;
}
.nt-account-link__subs {
  margin-top: 10px;
  padding: 10px 12px;
  border-radius: 10px;
  border: 1px solid var(--ui-border, hsl(var(--border)));
  background: var(--ui-bg-elevated, hsl(var(--card)));
}
.nt-account-link__subs-title {
  margin: 0 !important;
  font-size: 11.5px !important;
  font-weight: 650;
  color: var(--ui-text-primary, hsl(var(--foreground))) !important;
}
.nt-account-link__subs-list {
  margin: 8px 0 0;
  padding: 0;
  list-style: none;
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.nt-account-link__sub-item {
  padding-top: 8px;
  border-top: 1px solid var(--ui-border, hsl(var(--border)));
}
.nt-account-link__sub-item:first-child {
  padding-top: 0;
  border-top: 0;
}
.nt-account-link__sub-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  font-size: 12px;
}
.nt-account-link__sub-name {
  font-weight: 650;
  color: var(--ui-text-primary, hsl(var(--foreground)));
}
.nt-account-link__sub-expiry {
  font-size: 11px;
  color: var(--ui-text-secondary, hsl(var(--muted-foreground)));
}
.nt-account-link__sub-bar {
  margin-top: 6px;
  height: 5px;
  width: 100%;
  border-radius: 999px;
  background: rgba(148, 163, 184, 0.22);
  overflow: hidden;
}
.nt-account-link__sub-bar-fill {
  height: 100%;
  border-radius: 999px;
  background: #059669;
  transition: width 240ms ease;
}
.nt-account-link__sub-meta {
  margin-top: 5px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 6px;
  font-size: 11px;
  color: var(--ui-text-secondary, hsl(var(--muted-foreground)));
}
.nt-account-link__sub-overflow {
  padding: 1px 6px;
  border-radius: 999px;
  background: rgba(16, 185, 129, 0.12);
  color: #059669;
  font-weight: 600;
}
.nt-account-link__shortcuts {
  display: flex;
  flex-wrap: wrap;
  gap: 7px;
  margin-top: 10px;
}
.nt-account-link__chip {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  min-height: 28px;
  padding: 0 10px;
  border: 1px solid var(--ui-border, hsl(var(--border)));
  border-radius: 999px;
  background: var(--ui-bg-elevated, hsl(var(--card)));
  color: var(--ui-text-secondary, hsl(var(--foreground)));
  font-size: 11.5px;
  font-weight: 600;
  cursor: pointer;
  transition: border-color 140ms ease, color 140ms ease, background 140ms ease;
}
.nt-account-link__chip:hover:not(:disabled) {
  border-color: var(--ui-accent, hsl(var(--primary)));
  color: var(--ui-accent, hsl(var(--primary)));
}
.nt-account-link__chip:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}
.nt-account-link__error {
  color: #dc2626 !important;
  font-weight: 500;
}
.nt-account-link__actions {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 6px;
  flex: none;
}
.nt-account-link__cta {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  min-height: 38px;
  padding: 0 15px;
  border: 0;
  border-radius: 10px;
  background: var(--ui-accent, hsl(var(--primary)));
  color: #fff;
  font-size: 12.5px;
  font-weight: 650;
  white-space: nowrap;
  cursor: pointer;
  transition: opacity 120ms ease, transform 120ms ease;
}
.nt-account-link__cta.is-sso {
  background: #059669;
}
.nt-account-link.is-warning .nt-account-link__cta {
  background: #d97706;
}
.nt-account-link__cta:hover:not(:disabled) {
  opacity: 0.92;
}
.nt-account-link__cta:disabled,
.nt-account-link__secondary:disabled {
  opacity: 0.65;
  cursor: not-allowed;
}
.nt-account-link__secondary {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  min-height: 28px;
  padding: 0 8px;
  border: 0;
  background: transparent;
  color: var(--ui-text-muted, hsl(var(--muted-foreground)));
  font-size: 11.5px;
  font-weight: 500;
  white-space: nowrap;
  cursor: pointer;
}
.nt-account-link__secondary:hover:not(:disabled) {
  color: var(--ui-accent, hsl(var(--primary)));
}
.nt-account-link__spin {
  animation: nt-link-spin 1s linear infinite;
}
@keyframes nt-link-spin {
  to { transform: rotate(360deg); }
}
@media (max-width: 900px) {
  .nt-account-link__metrics {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}
@media (max-width: 720px) {
  .nt-account-link {
    flex-wrap: wrap;
    align-items: flex-start;
  }
  .nt-account-link__actions {
    width: 100%;
    flex-direction: row;
    flex-wrap: wrap;
    align-items: center;
    padding-left: 50px;
  }
}
@media (max-width: 480px) {
  .nt-account-link__actions {
    padding-left: 0;
  }
  .nt-account-link__cta {
    width: 100%;
  }
}
</style>
