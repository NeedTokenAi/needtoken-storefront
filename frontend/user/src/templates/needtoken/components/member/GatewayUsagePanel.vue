<template>
  <div class="space-y-5">
    <!-- 顶部统计概览 + 每日趋势图 -->
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

      <!-- 每日趋势可视化柱状图 -->
      <div v-if="usage.daily.length" class="mt-5 border-t pt-4">
        <div class="mb-3 flex flex-wrap items-center justify-between gap-2">
          <p class="text-xs font-semibold text-muted-foreground">{{ t('needtoken.memberConsole.usage.dailyTrend') }}</p>
          <div class="flex items-center gap-1 rounded-lg border bg-muted/40 p-0.5 text-[11px]">
            <button
              type="button"
              class="rounded-md px-2 py-0.5 font-semibold transition-colors"
              :class="trendMetric === 'quota' ? 'bg-background text-primary shadow-xs' : 'text-muted-foreground'"
              @click="trendMetric = 'quota'"
            >
              {{ t('needtoken.memberConsole.usage.colCost') }}
            </button>
            <button
              type="button"
              class="rounded-md px-2 py-0.5 font-semibold transition-colors"
              :class="trendMetric === 'requests' ? 'bg-background text-primary shadow-xs' : 'text-muted-foreground'"
              @click="trendMetric = 'requests'"
            >
              {{ t('needtoken.memberConsole.usage.totalRequests') }}
            </button>
          </div>
        </div>
        <div class="flex h-28 items-end gap-1.5 overflow-x-auto pb-1">
          <div
            v-for="d in usage.daily"
            :key="d.date"
            class="group flex min-w-[28px] flex-1 flex-col items-center gap-1"
            :title="`${d.date} · ${formatApiQuotaUsd(d.quota)} · ${formatNumber(d.requests)} ${t('needtoken.memberConsole.usage.reqUnit')}`"
          >
            <div class="flex h-20 w-full items-end justify-center rounded-md bg-muted/30 px-1 pt-1">
              <div
                class="w-full max-w-[26px] rounded-t-md bg-primary/80 transition-all group-hover:bg-primary"
                :style="{ height: `${dailyBarPercent(d)}%` }"
              />
            </div>
            <span class="font-mono text-[10px] text-muted-foreground">{{ d.date.slice(5) }}</span>
          </div>
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

    <!-- 三合一日志区：调用日志 / 异步任务 / 绘图任务 -->
    <div class="rounded-2xl border bg-card shadow-sm">
      <div class="flex flex-col gap-3 border-b px-5 py-3.5 lg:flex-row lg:items-center lg:justify-between">
        <div class="flex flex-wrap items-center gap-1.5">
          <button
            type="button"
            class="inline-flex items-center gap-1.5 rounded-xl px-3 py-1.5 text-xs font-semibold transition-colors"
            :class="logTab === 'common' ? 'bg-primary text-primary-foreground' : 'bg-muted/60 text-muted-foreground hover:text-foreground'"
            @click="switchLogTab('common')"
          >
            <FileText class="h-3.5 w-3.5" />
            <span>{{ t('needtoken.memberConsole.usage.tabCommon') }}</span>
          </button>
          <button
            type="button"
            class="inline-flex items-center gap-1.5 rounded-xl px-3 py-1.5 text-xs font-semibold transition-colors"
            :class="logTab === 'task' ? 'bg-primary text-primary-foreground' : 'bg-muted/60 text-muted-foreground hover:text-foreground'"
            @click="switchLogTab('task')"
          >
            <ListTodo class="h-3.5 w-3.5" />
            <span>{{ t('needtoken.memberConsole.usage.tabTask') }}</span>
          </button>
          <button
            type="button"
            class="inline-flex items-center gap-1.5 rounded-xl px-3 py-1.5 text-xs font-semibold transition-colors"
            :class="logTab === 'drawing' ? 'bg-primary text-primary-foreground' : 'bg-muted/60 text-muted-foreground hover:text-foreground'"
            @click="switchLogTab('drawing')"
          >
            <ImageIcon class="h-3.5 w-3.5" />
            <span>{{ t('needtoken.memberConsole.usage.tabDrawing') }}</span>
          </button>
        </div>

        <!-- Tab 1 筛选 -->
        <form v-if="logTab === 'common'" class="flex flex-wrap items-center gap-2" @submit.prevent="searchLogs">
          <NeedtokenSelect
            v-if="groupFilterOptions.length > 1"
            v-model="filterGroup"
            :options="groupFilterOptions"
            :label="t('needtoken.memberConsole.usage.allGroups')"
            compact
            align="start"
            class="min-w-[165px]"
          />
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

        <!-- Tab 2 筛选 -->
        <form v-else-if="logTab === 'task'" class="flex flex-wrap items-center gap-2" @submit.prevent="searchTasks">
          <input
            v-model.trim="taskIdFilter"
            type="text"
            :placeholder="t('needtoken.memberConsole.usage.taskIdPlaceholder')"
            class="h-8 w-44 rounded-lg border bg-background px-2.5 text-xs text-foreground outline-none focus:border-primary"
          />
          <NeedtokenSelect
            v-model="taskStatusFilter"
            :options="taskStatusOptions"
            :label="t('needtoken.memberConsole.usage.colStatus')"
            compact
            class="min-w-[130px]"
          />
          <Button type="submit" variant="outline" size="sm" :disabled="tasksLoading">
            <Search class="mr-1 h-3.5 w-3.5" />
            {{ t('needtoken.memberConsole.usage.filterAction') }}
          </Button>
        </form>

        <!-- Tab 3 筛选 -->
        <form v-else class="flex flex-wrap items-center gap-2" @submit.prevent="searchDrawings">
          <input
            v-model.trim="drawingIdFilter"
            type="text"
            :placeholder="t('needtoken.memberConsole.usage.drawingIdPlaceholder')"
            class="h-8 w-44 rounded-lg border bg-background px-2.5 text-xs text-foreground outline-none focus:border-primary"
          />
          <Button type="submit" variant="outline" size="sm" :disabled="drawingsLoading">
            <Search class="mr-1 h-3.5 w-3.5" />
            {{ t('needtoken.memberConsole.usage.filterAction') }}
          </Button>
        </form>
      </div>

      <div v-if="errorMsg" class="m-4 rounded-xl border border-destructive/30 bg-destructive/5 px-4 py-3 text-xs text-destructive">
        {{ errorMsg }}
      </div>

      <!-- Tab 1: 普通调用日志 -->
      <template v-if="logTab === 'common'">
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
                  <div class="flex items-center gap-1.5">
                    <span class="font-mono font-semibold text-foreground">{{ row.model_name || '-' }}</span>
                    <span v-if="row.is_stream" class="rounded bg-primary/10 px-1 py-0.5 text-[10px] font-semibold text-primary">SSE</span>
                  </div>
                  <span v-if="row.group" class="text-[10.5px] text-muted-foreground">{{ formatGroup(row.group) }}</span>
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
      </template>

      <!-- Tab 2: 异步任务日志 -->
      <template v-else-if="logTab === 'task'">
        <div v-if="tasksLoading && !tasks.length" class="space-y-2.5 p-5">
          <div v-for="idx in 4" :key="idx" class="h-11 animate-pulse rounded-lg bg-muted" />
        </div>
        <div v-else-if="!tasks.length" class="px-6 py-12 text-center text-sm text-muted-foreground">
          {{ t('needtoken.memberConsole.usage.emptyTasks') }}
        </div>
        <div v-else class="overflow-x-auto">
          <table class="w-full min-w-[680px] text-left text-xs">
            <thead class="border-b bg-muted/30 text-[11px] font-semibold uppercase tracking-wide text-muted-foreground">
              <tr>
                <th class="px-4 py-2.5">{{ t('needtoken.memberConsole.usage.colTime') }}</th>
                <th class="px-4 py-2.5">{{ t('needtoken.memberConsole.usage.colTaskId') }}</th>
                <th class="px-4 py-2.5">{{ t('needtoken.memberConsole.usage.colPlatformAction') }}</th>
                <th class="px-4 py-2.5">{{ t('needtoken.memberConsole.usage.colStatus') }}</th>
                <th class="px-4 py-2.5 text-right">{{ t('needtoken.memberConsole.usage.colCost') }}</th>
              </tr>
            </thead>
            <tbody class="divide-y">
              <tr v-for="item in tasks" :key="item.task_id" class="hover:bg-muted/20">
                <td class="whitespace-nowrap px-4 py-2.5 font-mono text-[11px] text-muted-foreground">
                  {{ formatLogTime(item.submit_time || item.created_at || 0) }}
                </td>
                <td class="px-4 py-2.5 font-mono font-semibold text-foreground">{{ item.task_id }}</td>
                <td class="px-4 py-2.5">
                  <span class="font-medium text-foreground">{{ item.platform || '-' }}</span>
                  <span v-if="item.action" class="ml-1.5 text-muted-foreground">· {{ item.action }}</span>
                  <p v-if="item.fail_reason" class="mt-0.5 max-w-xs truncate text-[11px] text-destructive">{{ item.fail_reason }}</p>
                </td>
                <td class="px-4 py-2.5">
                  <Badge :variant="taskStatusVariant(item.status)" size="sm">{{ item.status || '-' }}</Badge>
                  <span v-if="item.progress" class="ml-1.5 font-mono text-[11px] text-muted-foreground">{{ item.progress }}</span>
                </td>
                <td class="whitespace-nowrap px-4 py-2.5 text-right font-mono font-semibold text-foreground">
                  {{ formatApiQuotaUsd(item.quota || 0) }}
                </td>
              </tr>
            </tbody>
          </table>
        </div>
        <div v-if="taskTotalPages > 1" class="flex items-center justify-between border-t px-5 py-3 text-xs text-muted-foreground">
          <span>{{ t('needtoken.memberConsole.usage.pageInfo', { page: taskPage, totalPages: taskTotalPages, total: tasksTotal }) }}</span>
          <div class="flex items-center gap-2">
            <Button variant="outline" size="sm" :disabled="taskPage <= 1 || tasksLoading" @click="taskPage--; loadTasks()">
              {{ t('needtoken.memberConsole.usage.prevPage') }}
            </Button>
            <Button variant="outline" size="sm" :disabled="taskPage >= taskTotalPages || tasksLoading" @click="taskPage++; loadTasks()">
              {{ t('needtoken.memberConsole.usage.nextPage') }}
            </Button>
          </div>
        </div>
      </template>

      <!-- Tab 3: 绘图任务日志 -->
      <template v-else>
        <div v-if="drawingsLoading && !drawings.length" class="space-y-2.5 p-5">
          <div v-for="idx in 4" :key="idx" class="h-11 animate-pulse rounded-lg bg-muted" />
        </div>
        <div v-else-if="!drawings.length" class="px-6 py-12 text-center text-sm text-muted-foreground">
          {{ t('needtoken.memberConsole.usage.emptyDrawings') }}
        </div>
        <div v-else class="overflow-x-auto">
          <table class="w-full min-w-[680px] text-left text-xs">
            <thead class="border-b bg-muted/30 text-[11px] font-semibold uppercase tracking-wide text-muted-foreground">
              <tr>
                <th class="px-4 py-2.5">{{ t('needtoken.memberConsole.usage.colTime') }}</th>
                <th class="px-4 py-2.5">{{ t('needtoken.memberConsole.usage.colTaskId') }}</th>
                <th class="px-4 py-2.5">{{ t('needtoken.memberConsole.usage.colPrompt') }}</th>
                <th class="px-4 py-2.5">{{ t('needtoken.memberConsole.usage.colStatus') }}</th>
                <th class="px-4 py-2.5 text-right">{{ t('needtoken.memberConsole.usage.colCost') }}</th>
              </tr>
            </thead>
            <tbody class="divide-y">
              <tr v-for="item in drawings" :key="item.id || item.mj_id" class="hover:bg-muted/20">
                <td class="whitespace-nowrap px-4 py-2.5 font-mono text-[11px] text-muted-foreground">
                  {{ formatLogTime(item.submit_time) }}
                </td>
                <td class="px-4 py-2.5">
                  <div class="font-mono font-semibold text-foreground">{{ item.mj_id || `#${item.id}` }}</div>
                  <span class="text-[10.5px] text-muted-foreground">{{ item.action || 'IMAGINE' }}</span>
                </td>
                <td class="px-4 py-2.5">
                  <p class="max-w-xs truncate text-foreground">{{ item.prompt || item.prompt_en || '-' }}</p>
                  <a
                    v-if="item.image_url && /^https?:\/\//i.test(item.image_url)"
                    :href="item.image_url"
                    target="_blank"
                    rel="noopener noreferrer"
                    class="mt-0.5 inline-block text-[11px] font-medium text-primary hover:underline"
                  >
                    {{ t('needtoken.memberConsole.usage.openImage') }} ↗
                  </a>
                </td>
                <td class="px-4 py-2.5">
                  <Badge :variant="taskStatusVariant(item.status)" size="sm">{{ item.status || '-' }}</Badge>
                  <span v-if="item.progress" class="ml-1.5 font-mono text-[11px] text-muted-foreground">{{ item.progress }}</span>
                </td>
                <td class="whitespace-nowrap px-4 py-2.5 text-right font-mono font-semibold text-foreground">
                  {{ formatApiQuotaUsd(item.quota || 0) }}
                </td>
              </tr>
            </tbody>
          </table>
        </div>
        <div v-if="drawingTotalPages > 1" class="flex items-center justify-between border-t px-5 py-3 text-xs text-muted-foreground">
          <span>{{ t('needtoken.memberConsole.usage.pageInfo', { page: drawingPage, totalPages: drawingTotalPages, total: drawingsTotal }) }}</span>
          <div class="flex items-center gap-2">
            <Button variant="outline" size="sm" :disabled="drawingPage <= 1 || drawingsLoading" @click="drawingPage--; loadDrawings()">
              {{ t('needtoken.memberConsole.usage.prevPage') }}
            </Button>
            <Button variant="outline" size="sm" :disabled="drawingPage >= drawingTotalPages || drawingsLoading" @click="drawingPage++; loadDrawings()">
              {{ t('needtoken.memberConsole.usage.nextPage') }}
            </Button>
          </div>
        </div>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { FileText, Image as ImageIcon, ListTodo, RefreshCw, Search } from 'lucide-vue-next'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import NeedtokenSelect from '../NeedtokenSelect.vue'
import { formatApiQuotaUsd } from '../../lib/apiAccountLinkPolicy'
import {
  buildSelectableGroupOptions,
  formatConsoleGroupLabel,
} from '../../lib/gatewayConsoleHelpers'
import {
  getGatewayModels,
  getGatewayUsage,
  listGatewayDrawings,
  listGatewayLogs,
  listGatewayTasks,
  type GatewayDrawingItem,
  type GatewayGroupMetaItem,
  type GatewayLogItem,
  type GatewayTaskItem,
  type GatewayUsageDailyItem,
  type GatewayUsageResponse,
} from '../../lib/gatewayConsoleApi'

const { t, locale } = useI18n()
const selectedDays = ref(7)
const trendMetric = ref<'quota' | 'requests'>('quota')
const logTab = ref<'common' | 'task' | 'drawing'>('common')

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

const usableGroups = ref<Record<string, string>>({})
const groupRatio = ref<Record<string, number>>({})
const groupMeta = ref<Record<string, GatewayGroupMetaItem>>({})
const defaultGroup = ref('default')

const logs = ref<GatewayLogItem[]>([])
const logsTotal = ref(0)
const page = ref(1)
const pageSize = 20
const logsLoading = ref(false)
const filterGroup = ref('')
const filterModel = ref('')
const filterToken = ref('')
const errorMsg = ref('')

const tasks = ref<GatewayTaskItem[]>([])
const tasksTotal = ref(0)
const taskPage = ref(1)
const tasksLoading = ref(false)
const taskIdFilter = ref('')
const taskStatusFilter = ref('')

const drawings = ref<GatewayDrawingItem[]>([])
const drawingsTotal = ref(0)
const drawingPage = ref(1)
const drawingsLoading = ref(false)
const drawingIdFilter = ref('')

const dayOptions = computed(() => [
  { days: 1, label: t('needtoken.memberConsole.usage.range1d') },
  { days: 7, label: t('needtoken.memberConsole.usage.range7d') },
  { days: 30, label: t('needtoken.memberConsole.usage.range30d') },
])

const groupFilterOptions = computed(() => {
  const opts = buildSelectableGroupOptions({
    usableGroups: usableGroups.value,
    groupMeta: groupMeta.value,
    groupRatio: groupRatio.value,
    defaultGroup: defaultGroup.value,
  })
  return [
    { value: '', label: t('needtoken.memberConsole.usage.allGroups') },
    ...opts.map(g => ({ value: g.code, label: g.label })),
  ]
})

const taskStatusOptions = computed(() => [
  { value: '', label: t('needtoken.memberConsole.usage.allStatuses') },
  { value: 'SUCCESS', label: 'SUCCESS' },
  { value: 'IN_PROGRESS', label: 'IN_PROGRESS' },
  { value: 'FAILURE', label: 'FAILURE' },
])

const formatGroup = (code: string) =>
  formatConsoleGroupLabel(code, groupMeta.value, usableGroups.value, groupRatio.value)

watch(filterGroup, () => {
  if (logTab.value === 'common') {
    page.value = 1
    void loadLogs()
  }
})

watch(taskStatusFilter, () => {
  if (logTab.value === 'task') {
    taskPage.value = 1
    void loadTasks()
  }
})

const topModels = computed(() => (usage.value.models || []).slice(0, 6))
const totalPages = computed(() => Math.max(1, Math.ceil(logsTotal.value / pageSize)))
const taskTotalPages = computed(() => Math.max(1, Math.ceil(tasksTotal.value / pageSize)))
const drawingTotalPages = computed(() => Math.max(1, Math.ceil(drawingsTotal.value / pageSize)))

const dailyBarPercent = (item: GatewayUsageDailyItem) => {
  const values = usage.value.daily.map(d => (trendMetric.value === 'quota' ? d.quota : d.requests))
  const maxVal = Math.max(0, ...values)
  const current = trendMetric.value === 'quota' ? item.quota : item.requests
  if (maxVal <= 0 || current <= 0) return 4
  return Math.max(8, Math.round((current / maxVal) * 100))
}

const modelBarPercent = (quota: number) => {
  const maxQ = topModels.value[0]?.quota || 0
  if (maxQ <= 0) return 0
  return Math.max(4, Math.round((quota / maxQ) * 100))
}

const taskStatusVariant = (status?: string) => {
  const s = String(status || '').toUpperCase()
  if (s === 'SUCCESS' || s === 'COMPLETED') return 'success'
  if (s === 'FAILURE' || s === 'FAILED') return 'destructive'
  return 'secondary'
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
  const ms = ts > 1_000_000_000_000 ? ts : ts * 1000
  try {
    return new Intl.DateTimeFormat(locale.value || 'zh-CN', {
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
    }).format(new Date(ms))
  } catch {
    return new Date(ms).toISOString().slice(5, 19).replace('T', ' ')
  }
}

const loadGroupMetadata = async () => {
  try {
    const res = await getGatewayModels()
    usableGroups.value = res.usable_groups || {}
    groupRatio.value = res.group_ratio || {}
    groupMeta.value = res.group_meta || {}
    defaultGroup.value = res.default_group || 'default'
  } catch {
    // non-blocking metadata enrichment
  }
}

const loadUsage = async () => {
  try {
    usage.value = await getGatewayUsage(selectedDays.value)
  } catch {
    // keep previous state
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
      type: 2,
      group: filterGroup.value || undefined,
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

const loadTasks = async () => {
  tasksLoading.value = true
  errorMsg.value = ''
  try {
    const res = await listGatewayTasks({
      p: taskPage.value,
      page_size: pageSize,
      task_id: taskIdFilter.value || undefined,
      status: taskStatusFilter.value || undefined,
    })
    tasks.value = Array.isArray(res.items) ? res.items : []
    tasksTotal.value = Number(res.total || 0)
  } catch (err: any) {
    errorMsg.value = err?.response?.data?.msg || err?.message || t('needtoken.memberConsole.loadFailed')
  } finally {
    tasksLoading.value = false
  }
}

const loadDrawings = async () => {
  drawingsLoading.value = true
  errorMsg.value = ''
  try {
    const res = await listGatewayDrawings({
      p: drawingPage.value,
      page_size: pageSize,
      mj_id: drawingIdFilter.value || undefined,
    })
    drawings.value = Array.isArray(res.items) ? res.items : []
    drawingsTotal.value = Number(res.total || 0)
  } catch (err: any) {
    errorMsg.value = err?.response?.data?.msg || err?.message || t('needtoken.memberConsole.loadFailed')
  } finally {
    drawingsLoading.value = false
  }
}

const switchLogTab = (tab: 'common' | 'task' | 'drawing') => {
  logTab.value = tab
  errorMsg.value = ''
  if (tab === 'task' && !tasks.value.length) void loadTasks()
  if (tab === 'drawing' && !drawings.value.length) void loadDrawings()
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

const searchTasks = () => {
  taskPage.value = 1
  void loadTasks()
}

const searchDrawings = () => {
  drawingPage.value = 1
  void loadDrawings()
}

const goPage = (nextPage: number) => {
  page.value = Math.max(1, Math.min(totalPages.value, nextPage))
  void loadLogs()
}

const reloadAll = () => {
  void Promise.all([loadUsage(), loadLogs(), loadGroupMetadata()])
}

onMounted(() => {
  reloadAll()
})
</script>
