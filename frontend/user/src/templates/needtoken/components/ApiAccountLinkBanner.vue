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
      <Link2 v-else :size="18" />
    </div>

    <div class="nt-account-link__body">
      <div class="nt-account-link__head">
        <h3>{{ headingText }}</h3>
        <span v-if="isLinked && status?.native_user_id" class="nt-account-link__badge">
          API #{{ status.native_user_id }}
        </span>
      </div>
      <p>{{ descriptionText }}</p>
      <p v-if="errorMessage" class="nt-account-link__error" role="alert">
        {{ errorMessage || t('needtoken.apiAccountLink.startFailed') }}
      </p>
    </div>

    <div class="nt-account-link__actions">
      <button
        v-if="!isLinked"
        type="button"
        class="nt-account-link__cta"
        :disabled="starting"
        @click="handleStart"
      >
        <LoaderCircle v-if="starting" :size="15" class="nt-account-link__spin" aria-hidden="true" />
        <Link2 v-else :size="15" aria-hidden="true" />
        <span>{{ starting ? t('needtoken.apiAccountLink.redirecting') : ctaLabel }}</span>
      </button>
      <a
        v-else
        :href="CONSOLE_URL"
        target="_blank"
        rel="noopener noreferrer"
        class="nt-account-link__console"
      >
        <span>{{ t('needtoken.plans.console') }}</span>
        <ArrowUpRight :size="14" aria-hidden="true" />
      </a>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { AlertTriangle, ArrowUpRight, CheckCircle2, Link2, LoaderCircle } from 'lucide-vue-next'
import { useUserAuthStore } from '../../../stores/userAuth'
import { useAppStore } from '../../../stores/app'
import { getActiveTemplate } from '../../registry'
import { CONSOLE_URL } from '../config'
import { resetApiAccountLinkCache, useApiAccountLink } from '../composables/useApiAccountLink'

const props = withDefaults(
  defineProps<{
    nextPath?: string
    compact?: boolean
    onlyWhenActionNeeded?: boolean
  }>(),
  {
    nextPath: '/plans',
    compact: false,
    onlyWhenActionNeeded: false,
  },
)

const { t } = useI18n()
const userAuth = useUserAuthStore()
const appStore = useAppStore()
const { status, starting, errorMessage, load, beginLinking } = useApiAccountLink()

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
    : t('needtoken.apiAccountLink.bindAction'),
)

const handleStart = () => {
  void beginLinking(props.nextPath)
}
</script>

<style scoped>
.nt-account-link {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 14px 16px;
  border: 1px solid var(--nt-accent-border, hsl(var(--border)));
  border-radius: 14px;
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
  border-color: rgba(16, 185, 129, 0.32);
  background: rgba(16, 185, 129, 0.06);
}
.nt-account-link__icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 34px;
  height: 34px;
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
  font-size: 13px;
  font-weight: 650;
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
.nt-account-link__error {
  color: #dc2626 !important;
  font-weight: 500;
}
.nt-account-link__actions {
  flex: none;
}
.nt-account-link__cta {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-height: 36px;
  padding: 0 14px;
  border: 0;
  border-radius: 10px;
  background: var(--ui-accent, hsl(var(--primary)));
  color: #fff;
  font-size: 12px;
  font-weight: 600;
  cursor: pointer;
  transition: opacity 120ms ease;
}
.nt-account-link.is-warning .nt-account-link__cta {
  background: #d97706;
}
.nt-account-link__cta:hover:not(:disabled) {
  opacity: 0.9;
}
.nt-account-link__cta:disabled {
  opacity: 0.65;
  cursor: not-allowed;
}
.nt-account-link__console {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  font-size: 12px;
  font-weight: 600;
  color: var(--ui-accent, hsl(var(--primary)));
  white-space: nowrap;
}
.nt-account-link__spin {
  animation: nt-link-spin 1s linear infinite;
}
@keyframes nt-link-spin {
  to { transform: rotate(360deg); }
}
@media (max-width: 640px) {
  .nt-account-link {
    flex-wrap: wrap;
    align-items: flex-start;
  }
  .nt-account-link__actions {
    width: 100%;
    padding-left: 48px;
  }
}
</style>
