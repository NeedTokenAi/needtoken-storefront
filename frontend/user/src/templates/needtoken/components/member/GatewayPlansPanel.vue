<template>
  <div class="space-y-5">
    <!-- 顶部汇总卡 -->
    <div class="rounded-2xl border bg-card p-5 shadow-sm sm:p-6">
      <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h2 class="text-lg font-bold text-foreground">{{ t('needtoken.memberConsole.subscriptions.title') }}</h2>
          <p class="mt-1 text-xs text-muted-foreground">{{ t('needtoken.memberConsole.subscriptions.subtitle') }}</p>
        </div>
        <div class="flex items-center gap-2">
          <Button variant="outline" size="sm" :disabled="loading" @click="loadSubscriptions">
            <RefreshCw class="mr-1.5 h-3.5 w-3.5" :class="loading ? 'animate-spin' : ''" />
            {{ t('needtoken.common.refresh') }}
          </Button>
          <Button as-child size="sm">
            <RouterLink to="/plans">
              <Sparkles class="mr-1.5 h-3.5 w-3.5" />
              {{ t('needtoken.memberConsole.quick.buyPlans') }}
            </RouterLink>
          </Button>
        </div>
      </div>

      <div class="mt-4 grid grid-cols-1 gap-3 sm:grid-cols-3">
        <div class="rounded-xl border border-primary/25 bg-primary/5 p-4">
          <p class="text-xs text-muted-foreground">{{ t('needtoken.memberConsole.subscriptions.activeRemaining') }}</p>
          <p class="mt-1 font-mono text-xl font-black text-primary">{{ formatApiQuotaUsd(activeRemainingQuota) }}</p>
          <p class="mt-1 text-[11px] text-muted-foreground">
            {{ t('needtoken.memberConsole.subscriptions.activeCount', { count: activeSubs.length }) }}
          </p>
        </div>
        <div class="rounded-xl border bg-muted/20 p-4">
          <p class="text-xs text-muted-foreground">{{ t('needtoken.memberConsole.subscriptions.walletQuota') }}</p>
          <p class="mt-1 font-mono text-xl font-black text-foreground">{{ formatApiQuotaUsd(userQuota) }}</p>
          <p class="mt-1 text-[11px] text-muted-foreground">{{ t('needtoken.memberConsole.subscriptions.walletFallbackHint') }}</p>
        </div>
        <div class="rounded-xl border bg-muted/20 p-4">
          <p class="text-xs text-muted-foreground">{{ t('needtoken.memberConsole.subscriptions.defaultGroup') }}</p>
          <p class="mt-1 font-mono text-xl font-black text-foreground">{{ userGroup || 'default' }}</p>
          <p class="mt-1 text-[11px] text-muted-foreground">
            {{ t('needtoken.memberConsole.subscriptions.historyCount', { count: allSubs.length }) }}
          </p>
        </div>
      </div>
    </div>

    <div v-if="errorMsg" class="rounded-xl border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive">
      {{ errorMsg }}
    </div>

    <!-- 生效中的套餐 -->
    <div class="rounded-2xl border bg-card p-5 shadow-sm sm:p-6">
      <h3 class="text-base font-bold text-foreground">{{ t('needtoken.memberConsole.plans.title') }}</h3>
      <p class="mt-0.5 text-xs text-muted-foreground">{{ t('needtoken.memberConsole.plans.subtitle') }}</p>

      <div v-if="loading && !allSubs.length" class="mt-4 space-y-3">
        <div v-for="idx in 2" :key="idx" class="h-24 animate-pulse rounded-xl bg-muted" />
      </div>

      <div v-else-if="!activeSubs.length" class="mt-4 flex flex-col items-start justify-between gap-3 rounded-xl border border-dashed p-5 sm:flex-row sm:items-center">
        <div>
          <p class="text-sm font-semibold text-foreground">{{ t('needtoken.memberConsole.plans.emptyTitle') }}</p>
          <p class="mt-1 text-xs text-muted-foreground">{{ t('needtoken.memberConsole.plans.emptyHint') }}</p>
        </div>
        <Button as-child size="sm">
          <RouterLink to="/plans">{{ t('needtoken.memberConsole.quick.buyPlans') }}</RouterLink>
        </Button>
      </div>

      <div v-else class="mt-4 grid grid-cols-1 gap-4 md:grid-cols-2">
        <div
          v-for="sub in activeSubs"
          :key="sub.id"
          class="rounded-xl border bg-muted/20 p-4"
        >
          <div class="flex items-start justify-between gap-2">
            <div class="min-w-0">
              <p class="truncate text-sm font-bold text-foreground">
                {{ sub.plan_title || t('needtoken.apiAccountLink.subscriptionFallbackName', { id: sub.plan_id }) }}
              </p>
              <div v-if="splitGroups(sub.allowed_groups).length" class="mt-1.5 flex flex-wrap gap-1">
                <span
                  v-for="grp in splitGroups(sub.allowed_groups)"
                  :key="grp"
                  class="rounded-md border bg-background px-1.5 py-0.5 font-mono text-[10.5px] font-medium text-muted-foreground"
                >
                  {{ grp }}
                </span>
              </div>
            </div>
            <Badge variant="success" size="sm">{{ statusLabel(sub.status) }}</Badge>
          </div>

          <div class="mt-3 h-2 w-full overflow-hidden rounded-full bg-muted">
            <div
              class="h-full rounded-full bg-primary transition-all"
              :style="{ width: `${computeSubscriptionUsagePercent(sub.amount_total, sub.amount_used)}%` }"
            />
          </div>

          <div class="mt-2 flex flex-wrap items-center justify-between gap-2 text-xs">
            <span class="font-mono font-semibold text-foreground">
              {{
                t('needtoken.apiAccountLink.subscriptionRemaining', {
                  remaining: formatApiQuotaUsd(computeSubscriptionRemainingQuota(sub.amount_total, sub.amount_used)),
                  total: formatApiQuotaUsd(sub.amount_total),
                })
              }}
            </span>
            <span class="text-[11px] text-muted-foreground">{{ expiryText(sub.end_time) }}</span>
          </div>

          <div class="mt-2.5 flex flex-wrap items-center justify-between gap-2 border-t pt-2.5 text-[11px] text-muted-foreground">
            <span>{{ t('needtoken.memberConsole.subscriptions.startAt') }} {{ formatUnix(sub.start_time) }}</span>
            <span v-if="sub.allow_wallet_overflow" class="rounded-full bg-primary/10 px-2 py-0.5 font-semibold text-primary">
              {{ t('needtoken.apiAccountLink.walletOverflowEnabled') }}
            </span>
          </div>
        </div>
      </div>
    </div>

    <!-- 历史订阅记录 -->
    <div class="rounded-2xl border bg-card shadow-sm">
      <div class="flex flex-wrap items-center justify-between gap-3 border-b px-5 py-3.5">
        <h3 class="text-sm font-bold text-foreground">{{ t('needtoken.memberConsole.subscriptions.historyTitle') }}</h3>
        <select
          v-model="statusFilter"
          class="h-8 rounded-lg border bg-background px-2.5 text-xs text-foreground outline-none focus:border-primary"
        >
          <option value="all">{{ t('needtoken.memberConsole.subscriptions.filterAll') }}</option>
          <option value="active">{{ t('needtoken.memberConsole.subscriptions.statusActive') }}</option>
          <option value="depleted">{{ t('needtoken.memberConsole.subscriptions.statusDepleted') }}</option>
          <option value="expired">{{ t('needtoken.memberConsole.subscriptions.statusExpired') }}</option>
        </select>
      </div>

      <div v-if="!filteredHistory.length" class="px-6 py-10 text-center text-xs text-muted-foreground">
        {{ t('needtoken.memberConsole.subscriptions.emptyHistory') }}
      </div>

      <div v-else class="overflow-x-auto">
        <table class="w-full min-w-[640px] text-left text-xs">
          <thead class="border-b bg-muted/30 text-[11px] font-semibold uppercase tracking-wide text-muted-foreground">
            <tr>
              <th class="px-5 py-3">{{ t('needtoken.memberConsole.subscriptions.colPlan') }}</th>
              <th class="px-4 py-3">{{ t('needtoken.memberConsole.subscriptions.colStatus') }}</th>
              <th class="px-4 py-3">{{ t('needtoken.memberConsole.subscriptions.colUsage') }}</th>
              <th class="px-4 py-3">{{ t('needtoken.memberConsole.subscriptions.colPeriod') }}</th>
              <th class="px-5 py-3">{{ t('needtoken.memberConsole.subscriptions.colGroups') }}</th>
            </tr>
          </thead>
          <tbody class="divide-y">
            <tr v-for="sub in filteredHistory" :key="sub.id" class="hover:bg-muted/20">
              <td class="px-5 py-3 font-semibold text-foreground">
                {{ sub.plan_title || t('needtoken.apiAccountLink.subscriptionFallbackName', { id: sub.plan_id }) }}
              </td>
              <td class="px-4 py-3">
                <Badge :variant="statusVariant(sub.status)" size="sm">{{ statusLabel(sub.status) }}</Badge>
              </td>
              <td class="px-4 py-3 font-mono">
                {{ formatApiQuotaUsd(sub.amount_used) }} / {{ formatApiQuotaUsd(sub.amount_total) }}
              </td>
              <td class="px-4 py-3 font-mono text-[11px] text-muted-foreground">
                {{ formatUnix(sub.start_time) }} → {{ expiryText(sub.end_time) }}
              </td>
              <td class="px-5 py-3 font-mono text-[11px] text-muted-foreground">
                {{ sub.allowed_groups || '—' }}
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { RefreshCw, Sparkles } from 'lucide-vue-next'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  computeSubscriptionRemainingQuota,
  computeSubscriptionUsagePercent,
  formatApiQuotaUsd,
  formatSubscriptionExpiry,
} from '../../lib/apiAccountLinkPolicy'
import { sumSubscriptionRemainingQuota } from '../../lib/gatewayConsoleHelpers'
import {
  getGatewaySubscriptions,
  type GatewaySubscriptionItem,
} from '../../lib/gatewayConsoleApi'

const { t, locale } = useI18n()

const loading = ref(false)
const errorMsg = ref('')
const activeSubs = ref<GatewaySubscriptionItem[]>([])
const allSubs = ref<GatewaySubscriptionItem[]>([])
const userGroup = ref('default')
const userQuota = ref(0)
const statusFilter = ref<'all' | 'active' | 'depleted' | 'expired'>('all')

const activeRemainingQuota = computed(() => sumSubscriptionRemainingQuota(activeSubs.value))

const filteredHistory = computed(() => {
  if (statusFilter.value === 'all') return allSubs.value
  return allSubs.value.filter(s => String(s.status || '').toLowerCase() === statusFilter.value)
})

const splitGroups = (raw?: string | null) =>
  raw ? raw.split(',').map(s => s.trim()).filter(Boolean) : []

const formatUnix = (ts?: number) => {
  if (!ts || ts <= 0) return '—'
  return new Date(ts * 1000).toLocaleDateString()
}

const expiryText = (endTime?: number) => {
  const formatted = formatSubscriptionExpiry(endTime, String(locale.value || 'zh-CN'))
  if (!formatted) return t('needtoken.apiAccountLink.subscriptionPermanent')
  return formatted
}

const statusLabel = (raw?: string) => {
  const s = String(raw || '').toLowerCase()
  if (s === 'active') return t('needtoken.memberConsole.subscriptions.statusActive')
  if (s === 'depleted') return t('needtoken.memberConsole.subscriptions.statusDepleted')
  if (s === 'expired') return t('needtoken.memberConsole.subscriptions.statusExpired')
  return raw || '—'
}

const statusVariant = (raw?: string) => {
  const s = String(raw || '').toLowerCase()
  if (s === 'active') return 'success'
  if (s === 'depleted') return 'warning'
  return 'secondary'
}

const loadSubscriptions = async () => {
  loading.value = true
  errorMsg.value = ''
  try {
    const res = await getGatewaySubscriptions()
    activeSubs.value = Array.isArray(res.active_subscriptions) ? res.active_subscriptions : []
    allSubs.value = Array.isArray(res.all_subscriptions) ? res.all_subscriptions : []
    userGroup.value = res.user_group || 'default'
    userQuota.value = res.user_quota || 0
  } catch (err: any) {
    errorMsg.value = err?.response?.data?.msg || t('needtoken.memberConsole.loadFailed')
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  void loadSubscriptions()
})
</script>
