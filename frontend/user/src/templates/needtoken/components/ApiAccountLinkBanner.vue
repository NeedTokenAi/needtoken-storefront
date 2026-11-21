<template>
  <section
    v-if="visible"
    class="nt-account-link"
    :class="{
      'is-compact': compact,
      'is-warning': hasPendingOrders,
      'is-linked': isLinked,
    }"
    :aria-label="t('needtoken.apiAccountLink.title')"
  >
    <div class="nt-account-link__icon" aria-hidden="true">
      <AlertTriangle v-if="hasPendingOrders" :size="18" />
      <CheckCircle2 v-else-if="isLinked" :size="18" />
      <Sparkles v-else :size="18" />
    </div>

    <div class="nt-account-link__body">
      <div class="nt-account-link__head">
        <h3>{{ headingText }}</h3>
        <span v-if="isLinked && status?.native_user_id" class="nt-account-link__badge">
          API #{{ status.native_user_id }}
        </span>
      </div>
      <p>{{ descriptionText }}</p>

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
import type { ApiConsoleTargetPath } from '../lib/apiAccountLinkPolicy'

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

const { t } = useI18n()
const userAuth = useUserAuthStore()
const appStore = useAppStore()
const { status, starting, ssoLoading, ssoTarget, errorMessage, load, beginLinking, openApiConsoleSSO } = useApiAccountLink()

const quickLinks: readonly { target: ApiConsoleTargetPath; labelKey: string; icon: typeof KeyRound }[] = [
  { target: '/keys', labelKey: 'needtoken.apiAccountLink.shortcutKeys', icon: KeyRound },
  { target: '/dashboard', labelKey: 'needtoken.apiAccountLink.shortcutDashboard', icon: Activity },
  { target: '/subscriptions', labelKey: 'needtoken.apiAccountLink.shortcutSubscriptions', icon: CreditCard },
]

watch(
  () => [userAuth.isAuthenticated, appStore.isResellerTenant] as const,
  ([authed, reseller]) => {
    if (!authed || reseller || getActiveTemplate() !== 'needtoken') {
      if (!authed) resetApiAccountLinkCache()
      return
    }
    void load()
  },
  { immediate: true },
)

const isLinked = computed(() => Boolean(status.value?.linked))
const pendingOrders = computed(() => status.value?.pending_orders ?? 0)
const hasPendingOrders = computed(() => !isLinked.value && pendingOrders.value > 0)
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
  if (isLinked.value) {
    return t('needtoken.apiAccountLink.linkedHeading')
  }
  return t('needtoken.apiAccountLink.unlinkedHeading')
})

const descriptionText = computed(() => {
  if (hasPendingOrders.value) {
    return t('needtoken.apiAccountLink.pendingDescription')
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
.nt-account-link__body p {
  margin-top: 4px;
  font-size: 12px;
  line-height: 1.65;
  color: var(--ui-text-secondary, hsl(var(--muted-foreground)));
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
