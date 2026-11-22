<template>
  <div class="space-y-5">
    <!-- 顶部统计概览 -->
    <div class="rounded-2xl border bg-card p-5 shadow-sm">
      <div class="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h2 class="text-lg font-bold text-foreground">{{ t('needtoken.memberConsole.usage.title') }}</h2>
          <p class="mt-1 text-xs text-muted-foreground">{{ t('needtoken.memberConsole.usage.subtitle') }}</p>
        </div>
        <div class="flex items-center gap-1.5 rounded-xl border bg-muted/40 p-1">
          <button
            v-for="opt in dayOptions"
            :key="opt.days"
            type="button"
            class="rounded-lg px-3 py-1.5 text-xs font-semibold transition-colors"
            :class="selectedDays === opt.days ? 'bg-background text-primary shadow-xs' : 'text-muted-foreground hover:text-foreground'"
            @click="changeDays(opt.days)"
          >
            {{ opt.label }}
          </button>
        </div>
      </div>

      <div class="mt-4 grid grid-cols-1 gap-3 sm:grid-cols-3">
        <div class="rounded-xl border bg-muted/20 p-4">
          <p class="text-xs text-muted-foreground">{{ t('needtoken.memberConsole.usage.totalSpend') }}</p>
          <p class="mt-1 font-mono text-xl font-black text-primary">{{ formatApiQuotaUsd(usage.total_quota) }}</p>
        </div>
        <div class="rounded-xl border bg-muted/20 p-4">
          <p class="text-xs text-muted-foreground">{{ t('needtoken.memberConsole.usage.totalRequests') }}</p>
          <p class="mt-1 font-mono text-xl font-black text-foreground">{{ formatNumber(usage.total_requests) }}</p>
        </div>
        <div class="rounded-xl border bg-muted/20 p-4">
          <p class="text-xs text-muted-foreground">{{ t('needtoken.memberConsole.usage.totalTokens') }}</p>
          <p class="mt-1 font-mono text-xl font-black text-foreground">{{ formatCompactNumber(usage.total_tokens) }}</p>
        </div>
      </div>

      <!-- 模型消耗分布 -->
      <div v-if="topModels.length" class="mt-5 border-t pt-4">
        <p class="mb-3 text-xs font-semibold text-muted-foreground">{{ t('needtoken.memberConsole.usage.modelBreakdown') }}</p>
        <div class="space-y-2.5">
          <div v-for="m in topModels" :key="m.model_name" class="space-y-1">
            <div class="flex items-center justify-between gap-2 text-xs">
              <span class="truncate font-mono font-semibold text-foreground">{{ m.model_name }}</span>
              <span class="shrink-0 font-mono text-muted-foreground">
                {{ formatApiQuotaUsd(m.quota) }} · {{ formatNumber(m.requests) }} {{ t('needtoken.memberConsole.usage.reqUnit') }}
              </span>
            </div>
            <div class="h-1.5 w-full overflow-hidden rounded-full bg-muted">
              <div
                class="h-full rounded-full bg-primary transition-all"
                :style="{ width: `${modelBarPercent(m.quota)}%` }"
              />
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 调用明细日志 -->
    <div class="rounded-2xl border bg-card shadow-sm">
      <div class="flex flex-wrap items-center justify-between gap-3 border-b px-5 py-3.5">
        <h3 class="text-sm font-bold text-foreground">{{ t('needtoken.memberConsole.usage.logsTitle') }}</h3>
        <form class="flex flex-wrap items-center gap-2" @submit.prevent="searchLogs">
          <input
            v-model="filterModel"
            type="text"
            :placeholder="t('needtoken.memberConsole.usage.modelFilterPlaceholder')"
            class="h-8 w-44 rounded-lg border bg-background px-2.5 text-xs text-foreground outline-none focus:border-primary"
          />
          <input
            v-model="filterToken"
            type="text"
            :placeholder="t('needtoken.memberConsole.usage.tokenFilterPlaceholder')"
            class="h-8 w-36 rounded-lg border bg-background px-2.5 text-xs text-foreground outline-none focus:border-primary"
          />
          <Button type="submit" variant="outline" size="sm" :disabled="logsLoading">
            <Search class="mr-1 h-3.5 w-3.5" />
            {{ t('needtoken.memberConsole.usage.filterAction') }}
          </Button>
          <Button type="button" variant="ghost" size="sm" :disabled="logsLoading" @click="reloadAll">
            <RefreshCw class="h-3.5 w-3.5" :class="logsLoading ? 'animate-spin' : ''" />
          </Button>
        </form>
      </div>

      <div v-if="errorMsg" class="m-4 rounded-xl border border-destructive/30 bg-destructive/5 px-4 py-3 text-xs text-destructive">
        {{ errorMsg }}
      </div>

      <div v-if="logsLoading && !logs.length" class="space-y-2.5 p-5">
        <div v-for="idx in 5" :key="idx" class="h-11 animate-pulse rounded-lg bg-muted" />
      </div>

      <div v-else-if="!logs.length" class="px-6 py-12 text-center text-sm text-muted-foreground">
        {{ t('needtoken.memberConsole.usage.emptyLogs') }}
      </div>

      <div v-else class="overflow-x-auto">
        <table class="w-full min-w-[680px] text-left text-xs">
          <thead class="border-b bg-muted/30 text-[11px] font-semibold uppercase tracking-wide text-muted-foreground">
            <tr>
              <th class="px-4 py-2.5">{{ t('needtoken.memberConsole.usage.colTime') }}</th>
              <th class="px-4 py-2.5">{{ t('needtoken.memberConsole.usage.colModel') }}</th>
              <th class="px-4 py-2.5">{{ t('needtoken.memberConsole.usage.colKey') }}</th>
              <th class="px-4 py-2.5">{{ t('needtoken.memberConsole.usage.colTokens') }}</th>
              <th class="px-4 py-2.5">{{ t('needtoken.memberConsole.usage.colDuration') }}</th>
              <th class="px-4 py-2.5 text-right">{{ t('needtoken.memberConsole.usage.colCost') }}</th>
            </tr>
          </thead>
          <tbody class="divide-y">
            <tr v-for="row in logs" :key="row.id" class="hover:bg-muted/20">
              <td class="whitespace-nowrap px-4 py-2.5 font-mono text-[11px] text-muted-foreground">
                {{ formatLogTime(row.created_at) }}
              </td>
              <td class="px-4 py-2.5">
                <div class="font-mono font-semibold text-foreground">{{ row.model_name || '-' }}</div>
                <span v-if="row.group" class="text-[10.5px] text-muted-foreground">{{ row.group }}</span>
              </td>
              <td class="px-4 py-2.5 text-muted-foreground">{{ row.token_name || '-' }}</td>
              <td class="whitespace-nowrap px-4 py-2.5 font-mono text-muted-foreground">
                {{ formatNumber(row.prompt_tokens) }} / {{ formatNumber(row.completion_tokens) }}
              </td>
              <td class="whitespace-nowrap px-4 py-2.5 font-mono text-muted-foreground">
                {{ row.use_time > 0 ? `${row.use_time}s` : '-' }}
              </td>
              <td class="whitespace-nowrap px-4 py-2.5 text-right font-mono font-semibold text-foreground">
                {{ formatApiQuotaUsd(row.quota) }}
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <div v-if="totalPages > 1" class="flex items-center justify-between border-t px-5 py-3 text-xs text-muted-foreground">
        <span>{{ t('needtoken.memberConsole.usage.pageInfo', { page, totalPages, total: logsTotal }) }}</span>
        <div class="flex items-center gap-2">
          <Button variant="outline" size="sm" :disabled="page <= 1 || logsLoading" @click="goPage(page - 1)">
            {{ t('needtoken.memberConsole.usage.prevPage') }}
          </Button>
          <Button variant="outline" size="sm" :disabled="page >= totalPages || logsLoading" @click="goPage(page + 1)">
            {{ t('needtoken.memberConsole.usage.nextPage') }}
          </Button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { RefreshCw, Search } from 'lucide-vue-next'
import { Button } from '@/components/ui/button'
import { formatApiQuotaUsd } from '../../lib/apiAccountLinkPolicy'
import {
  getGatewayUsage,
  listGatewayLogs,
  type GatewayLogItem,
  type GatewayUsageResponse,
} from '../../lib/gatewayConsoleApi'

const { t, locale } = useI18n()
const selectedDays = ref(7)
const usage = ref<GatewayUsageResponse>({
  days: 7,
  start_ts: 0,
  end_ts: 0,
  total_quota: 0,
  total_requests: 0,
  total_tokens: 0,
  quota_per_unit: 500000,
  daily: [],
  models: [],
})

const logs = ref<GatewayLogItem[]>([])
const logsTotal = ref(0)
const page = ref(1)
const pageSize = 20
const logsLoading = ref(false)
const filterModel = ref('')
const filterToken = ref('')
const errorMsg = ref('')

const dayOptions = computed(() => [
  { days: 1, label: t('needtoken.memberConsole.usage.range1d') },
  { days: 7, label: t('needtoken.memberConsole.usage.range7d') },
  { days: 30, label: t('needtoken.memberConsole.usage.range30d') },
])

const topModels = computed(() => (usage.value.models || []).slice(0, 6))
const totalPages = computed(() => Math.max(1, Math.ceil(logsTotal.value / pageSize)))

const modelBarPercent = (quota: number) => {
  const maxQ = topModels.value[0]?.quota || 0
  if (maxQ <= 0) return 0
  return Math.max(4, Math.round((quota / maxQ) * 100))
}

const formatNumber = (n?: number) => {
  const num = typeof n === 'number' && Number.isFinite(n) ? n : 0
  return new Intl.NumberFormat(locale.value || 'zh-CN').format(num)
}

const formatCompactNumber = (n?: number) => {
  const num = typeof n === 'number' && Number.isFinite(n) ? n : 0
  if (num >= 1_000_000) return `${(num / 1_000_000).toFixed(2)}M`
  if (num >= 1_000) return `${(num / 1_000).toFixed(1)}K`
  return String(num)
}

const formatLogTime = (ts: number) => {
  if (!ts) return '-'
  try {
    return new Intl.DateTimeFormat(locale.value || 'zh-CN', {
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
    }).format(new Date(ts * 1000))
  } catch {
    return new Date(ts * 1000).toISOString().slice(5, 19).replace('T', ' ')
  }
}

const loadUsage = async () => {
  try {
    usage.value = await getGatewayUsage(selectedDays.value)
  } catch {
    // Keep last state; log list handles surfacing connection errors.
  }
}

const loadLogs = async () => {
  logsLoading.value = true
  errorMsg.value = ''
  try {
    const startTs = Math.floor(Date.now() / 1000) - selectedDays.value * 86400
    const res = await listGatewayLogs({
      p: page.value,
      page_size: pageSize,
      model_name: filterModel.value.trim() || undefined,
      token_name: filterToken.value.trim() || undefined,
      start_timestamp: startTs,
    })
    logs.value = Array.isArray(res.items) ? res.items : []
    logsTotal.value = Number(res.total || 0)
  } catch (err: any) {
    errorMsg.value = err?.response?.data?.msg || err?.message || t('needtoken.memberConsole.loadFailed')
  } finally {
    logsLoading.value = false
  }
}

const changeDays = (days: number) => {
  if (selectedDays.value === days) return
  selectedDays.value = days
  page.value = 1
  void Promise.all([loadUsage(), loadLogs()])
}

const searchLogs = () => {
  page.value = 1
  void loadLogs()
}

const goPage = (nextPage: number) => {
  page.value = Math.max(1, Math.min(totalPages.value, nextPage))
  void loadLogs()
}

const reloadAll = () => {
  void Promise.all([loadUsage(), loadLogs()])
}

onMounted(() => {
  reloadAll()
})
</script>
