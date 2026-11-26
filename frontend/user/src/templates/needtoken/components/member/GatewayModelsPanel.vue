<template>
  <div class="space-y-5">
    <!-- 顶部标题与双视图切换 -->
    <div class="rounded-2xl border bg-card p-5 shadow-sm">
      <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h2 class="text-lg font-bold text-foreground">{{ t('needtoken.memberConsole.models.title') }}</h2>
          <p class="mt-1 text-xs text-muted-foreground">{{ t('needtoken.memberConsole.models.subtitle') }}</p>
        </div>
        <div class="flex items-center gap-1.5 rounded-xl border bg-muted/40 p-1">
          <button
            type="button"
            class="inline-flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-xs font-semibold transition-colors"
            :class="activeTab === 'catalog' ? 'bg-background text-primary shadow-xs' : 'text-muted-foreground hover:text-foreground'"
            @click="activeTab = 'catalog'"
          >
            <Box class="h-3.5 w-3.5" />
            <span>{{ t('needtoken.memberConsole.models.tabCatalog') }}</span>
          </button>
          <button
            type="button"
            class="inline-flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-xs font-semibold transition-colors"
            :class="activeTab === 'analytics' ? 'bg-background text-primary shadow-xs' : 'text-muted-foreground hover:text-foreground'"
            @click="switchToAnalytics"
          >
            <BarChart3 class="h-3.5 w-3.5" />
            <span>{{ t('needtoken.memberConsole.models.tabAnalytics') }}</span>
          </button>
        </div>
      </div>

      <!-- Tab 1 筛选栏：搜索 + 分组切换 + 计费类型 -->
      <div v-if="activeTab === 'catalog'" class="mt-4 space-y-3 border-t pt-4">
        <div class="flex flex-wrap items-center gap-2">
          <div class="relative min-w-[200px] flex-1">
            <Search class="pointer-events-none absolute left-3 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-muted-foreground" />
            <input
              v-model.trim="searchKeyword"
              type="text"
              :placeholder="t('needtoken.memberConsole.models.searchPlaceholder')"
              class="h-9 w-full rounded-xl border bg-background pl-8 pr-3 text-xs text-foreground outline-none focus:border-primary"
            />
          </div>
          <NeedtokenSelect
            v-model="selectedGroup"
            :options="groupSelectOptions"
            :label="t('needtoken.memberConsole.models.colGroups')"
            compact
            align="start"
            class="min-w-[210px]"
          />
          <NeedtokenSelect
            v-model="billingFilter"
            :options="billingSelectOptions"
            :label="t('needtoken.memberConsole.models.colBilling')"
            compact
            class="min-w-[140px]"
          />
          <Button variant="outline" size="sm" :disabled="modelsLoading" @click="loadModels">
            <RefreshCw class="h-3.5 w-3.5" :class="modelsLoading ? 'animate-spin' : ''" />
          </Button>
        </div>

        <div class="flex flex-wrap items-center justify-between gap-2 text-xs text-muted-foreground">
          <span>
            {{ t('needtoken.memberConsole.models.activeGroupLabel', { group: activeGroupDisplay }) }}
          </span>
          <span>{{ t('needtoken.memberConsole.models.totalCount', { count: filteredModels.length }) }}</span>
        </div>
      </div>

      <!-- Tab 2 筛选栏：统计周期 -->
      <div v-else class="mt-4 flex flex-wrap items-center justify-between gap-3 border-t pt-4">
        <p class="text-xs text-muted-foreground">{{ t('needtoken.memberConsole.models.analyticsHint') }}</p>
        <div class="flex items-center gap-1 rounded-xl border bg-muted/40 p-1">
          <button
            v-for="opt in dayOptions"
            :key="opt.days"
            type="button"
            class="rounded-lg px-3 py-1 text-xs font-semibold transition-colors"
            :class="analyticsDays === opt.days ? 'bg-background text-primary shadow-xs' : 'text-muted-foreground hover:text-foreground'"
            @click="changeAnalyticsDays(opt.days)"
          >
            {{ opt.label }}
          </button>
        </div>
      </div>
    </div>

    <div v-if="errorMsg" class="rounded-xl border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive">
      {{ errorMsg }}
    </div>

    <!-- Tab 1：可用模型与价格目录 -->
    <div v-if="activeTab === 'catalog'" class="rounded-2xl border bg-card shadow-sm">
      <div v-if="modelsLoading && !models.length" class="space-y-3 p-5">
        <div v-for="idx in 6" :key="idx" class="h-12 animate-pulse rounded-xl bg-muted" />
      </div>

      <div v-else-if="!filteredModels.length" class="px-6 py-12 text-center">
        <Box class="mx-auto h-9 w-9 text-muted-foreground/50" />
        <p class="mt-3 text-sm font-semibold text-foreground">{{ t('needtoken.memberConsole.models.emptyModels') }}</p>
      </div>

      <div v-else class="overflow-x-auto">
        <table class="w-full min-w-[720px] text-left text-xs">
          <thead class="border-b bg-muted/30 text-[11px] font-semibold uppercase tracking-wide text-muted-foreground">
            <tr>
              <th class="px-5 py-3">{{ t('needtoken.memberConsole.models.colModel') }}</th>
              <th class="px-4 py-3">{{ t('needtoken.memberConsole.models.colBilling') }}</th>
              <th class="px-4 py-3">{{ t('needtoken.memberConsole.models.colPrice') }}</th>
              <th class="px-4 py-3">{{ t('needtoken.memberConsole.models.colGroups') }}</th>
              <th class="px-5 py-3 text-right">{{ t('needtoken.memberConsole.models.colActions') }}</th>
            </tr>
          </thead>
          <tbody class="divide-y">
            <tr v-for="m in pagedModels" :key="m.model_name" class="transition-colors hover:bg-muted/20">
              <td class="px-5 py-3.5">
                <div class="flex items-center gap-2">
                  <span class="font-mono text-xs font-bold text-foreground">{{ m.model_name }}</span>
                  <button
                    type="button"
                    class="inline-flex items-center rounded-md border bg-background px-1.5 py-0.5 text-[10px] font-medium text-muted-foreground transition-colors hover:border-primary/40 hover:text-primary"
                    @click="copyModelId(m.model_name)"
                  >
                    <Check v-if="copiedModel === m.model_name" class="mr-0.5 h-3 w-3 text-primary" />
                    <Copy v-else class="mr-0.5 h-3 w-3" />
                    {{ copiedModel === m.model_name ? t('needtoken.common.copied') : t('needtoken.common.copy') }}
                  </button>
                </div>
                <p v-if="vendorName(m.vendor_id)" class="mt-0.5 text-[11px] text-muted-foreground">
                  {{ vendorName(m.vendor_id) }}
                </p>
              </td>
              <td class="px-4 py-3.5">
                <Badge :variant="m.quota_type === 1 ? 'warning' : 'secondary'" size="sm">
                  {{ m.quota_type === 1 ? t('needtoken.memberConsole.models.billingFixed') : t('needtoken.memberConsole.models.billingToken') }}
                </Badge>
              </td>
              <td class="px-4 py-3.5 font-mono">
                <template v-if="m.quota_type === 1">
                  <span class="font-bold text-primary">${{ formatUnitUsd(m.model_price * activeGroupRatio) }}</span>
                  <span class="text-muted-foreground"> / {{ t('needtoken.memberConsole.models.perCall') }}</span>
                </template>
                <template v-else>
                  <div class="flex flex-col gap-0.5">
                    <span>
                      <span class="text-muted-foreground">{{ t('needtoken.memberConsole.models.inputLabel') }}</span>
                      <strong class="ml-1 text-primary">${{ formatUnitUsd(m.model_ratio * 2 * activeGroupRatio) }}</strong>
                      <span class="text-[10.5px] text-muted-foreground"> /1M</span>
                    </span>
                    <span>
                      <span class="text-muted-foreground">{{ t('needtoken.memberConsole.models.outputLabel') }}</span>
                      <strong class="ml-1 text-foreground">${{ formatUnitUsd(m.model_ratio * 2 * (m.completion_ratio || 1) * activeGroupRatio) }}</strong>
                      <span class="text-[10.5px] text-muted-foreground"> /1M</span>
                    </span>
                  </div>
                </template>
              </td>
              <td class="px-4 py-3.5">
                <div class="flex flex-wrap gap-1">
                  <span
                    v-for="grp in visibleModelGroups(m).slice(0, 3)"
                    :key="grp"
                    class="rounded-md border bg-muted/40 px-1.5 py-0.5 text-[10.5px] font-medium text-muted-foreground"
                  >
                    {{ formatGroup(grp) }}
                  </span>
                </div>
              </td>
              <td class="px-5 py-3.5 text-right">
                <Button variant="outline" size="sm" class="h-7 px-2.5 text-xs" @click="tryInPlayground(m.model_name)">
                  <FlaskConical class="mr-1 h-3 w-3 text-primary" />
                  {{ t('needtoken.memberConsole.models.tryPlayground') }}
                </Button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <div v-if="totalModelPages > 1" class="flex items-center justify-between border-t px-5 py-3 text-xs text-muted-foreground">
        <span>{{ t('needtoken.memberConsole.usage.pageInfo', { page: modelPage, totalPages: totalModelPages, total: filteredModels.length }) }}</span>
        <div class="flex items-center gap-2">
          <Button variant="outline" size="sm" :disabled="modelPage <= 1" @click="modelPage--">
            {{ t('needtoken.memberConsole.usage.prevPage') }}
          </Button>
          <Button variant="outline" size="sm" :disabled="modelPage >= totalModelPages" @click="modelPage++">
            {{ t('needtoken.memberConsole.usage.nextPage') }}
          </Button>
        </div>
      </div>
    </div>

    <!-- Tab 2：模型调用分析 -->
    <div v-else class="space-y-4">
      <div class="grid grid-cols-1 gap-3 sm:grid-cols-3">
        <div class="rounded-2xl border bg-card p-5 shadow-sm">
          <p class="text-xs text-muted-foreground">{{ t('needtoken.memberConsole.usage.totalSpend') }}</p>
          <p class="mt-1.5 font-mono text-2xl font-black text-primary">{{ formatApiQuotaUsd(usage.total_quota) }}</p>
        </div>
        <div class="rounded-2xl border bg-card p-5 shadow-sm">
          <p class="text-xs text-muted-foreground">{{ t('needtoken.memberConsole.usage.totalRequests') }}</p>
          <p class="mt-1.5 font-mono text-2xl font-black text-foreground">{{ formatNumber(usage.total_requests) }}</p>
        </div>
        <div class="rounded-2xl border bg-card p-5 shadow-sm">
          <p class="text-xs text-muted-foreground">{{ t('needtoken.memberConsole.usage.totalTokens') }}</p>
          <p class="mt-1.5 font-mono text-2xl font-black text-foreground">{{ formatCompactNumber(usage.total_tokens) }}</p>
        </div>
      </div>

      <div class="rounded-2xl border bg-card shadow-sm">
        <div v-if="usageLoading && !usage.models.length" class="space-y-3 p-5">
          <div v-for="idx in 4" :key="idx" class="h-12 animate-pulse rounded-xl bg-muted" />
        </div>
        <div v-else-if="!usage.models.length" class="px-6 py-12 text-center text-sm text-muted-foreground">
          {{ t('needtoken.memberConsole.usage.emptyLogs') }}
        </div>
        <div v-else class="overflow-x-auto">
          <table class="w-full min-w-[640px] text-left text-xs">
            <thead class="border-b bg-muted/30 text-[11px] font-semibold uppercase tracking-wide text-muted-foreground">
              <tr>
                <th class="px-5 py-3">{{ t('needtoken.memberConsole.models.colModel') }}</th>
                <th class="px-4 py-3 text-right">{{ t('needtoken.memberConsole.usage.totalRequests') }}</th>
                <th class="px-4 py-3 text-right">{{ t('needtoken.memberConsole.usage.totalTokens') }}</th>
                <th class="px-4 py-3 text-right">{{ t('needtoken.memberConsole.usage.colCost') }}</th>
                <th class="px-5 py-3">{{ t('needtoken.memberConsole.models.colShare') }}</th>
              </tr>
            </thead>
            <tbody class="divide-y">
              <tr v-for="item in usage.models" :key="item.model_name" class="hover:bg-muted/20">
                <td class="px-5 py-3 font-mono font-semibold text-foreground">{{ item.model_name }}</td>
                <td class="px-4 py-3 text-right font-mono">{{ formatNumber(item.requests) }}</td>
                <td class="px-4 py-3 text-right font-mono text-muted-foreground">{{ formatCompactNumber(item.token_used) }}</td>
                <td class="px-4 py-3 text-right font-mono font-bold text-primary">{{ formatApiQuotaUsd(item.quota) }}</td>
                <td class="px-5 py-3">
                  <div class="flex items-center gap-2">
                    <div class="h-2 flex-1 overflow-hidden rounded-full bg-muted">
                      <div class="h-full rounded-full bg-primary" :style="{ width: `${sharePercent(item.quota)}%` }" />
                    </div>
                    <span class="w-10 text-right font-mono text-[11px] text-muted-foreground">{{ sharePercent(item.quota) }}%</span>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { BarChart3, Box, Check, Copy, FlaskConical, RefreshCw, Search } from 'lucide-vue-next'
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
  type GatewayGroupMetaItem,
  type GatewayModelPricingItem,
  type GatewayUsageResponse,
  type GatewayVendorItem,
} from '../../lib/gatewayConsoleApi'

const { t } = useI18n()
const router = useRouter()

const activeTab = ref<'catalog' | 'analytics'>('catalog')
const modelsLoading = ref(false)
const usageLoading = ref(false)
const errorMsg = ref('')

const models = ref<GatewayModelPricingItem[]>([])
const vendors = ref<GatewayVendorItem[]>([])
const usableGroups = ref<Record<string, string>>({})
const groupRatio = ref<Record<string, number>>({})
const groupMeta = ref<Record<string, GatewayGroupMetaItem>>({})
const defaultGroup = ref('default')

const searchKeyword = ref('')
const selectedGroup = ref('')
const billingFilter = ref('all')
const modelPage = ref(1)
const modelPageSize = 20

const analyticsDays = ref(7)
const usage = ref<GatewayUsageResponse>({
  days: 7,
  start_ts: 0,
  end_ts: 0,
  total_quota: 0,
  total_requests: 0,
  total_tokens: 0,
  quota_per_unit: 500_000,
  daily: [],
  models: [],
})

const dayOptions = computed(() => [
  { days: 1, label: t('needtoken.memberConsole.usage.range1d') },
  { days: 7, label: t('needtoken.memberConsole.usage.range7d') },
  { days: 30, label: t('needtoken.memberConsole.usage.range30d') },
])

const selectableGroups = computed(() =>
  buildSelectableGroupOptions({
    usableGroups: usableGroups.value,
    groupMeta: groupMeta.value,
    groupRatio: groupRatio.value,
    pricingModels: models.value,
    defaultGroup: defaultGroup.value,
  }),
)

const allowedGroupCodeSet = computed(() => new Set(selectableGroups.value.map(g => g.code)))

const groupSelectOptions = computed(() => {
  if (!selectableGroups.value.length) {
    const code = defaultGroup.value || 'default'
    return [{ value: code, label: formatConsoleGroupLabel(code, groupMeta.value, usableGroups.value, groupRatio.value) }]
  }
  return selectableGroups.value.map(g => ({ value: g.code, label: g.label }))
})

const billingSelectOptions = computed(() => [
  { value: 'all', label: t('needtoken.memberConsole.models.billingAll') },
  { value: 'token', label: t('needtoken.memberConsole.models.billingToken') },
  { value: 'fixed', label: t('needtoken.memberConsole.models.billingFixed') },
])

const formatGroup = (code: string) =>
  formatConsoleGroupLabel(code, groupMeta.value, usableGroups.value, groupRatio.value)

const activeGroupCode = computed(() => selectedGroup.value || selectableGroups.value[0]?.code || defaultGroup.value || 'default')

const activeGroupDisplay = computed(() => formatGroup(activeGroupCode.value))

const activeGroupRatio = computed(() => {
  const key = activeGroupCode.value
  const metaRatio = groupMeta.value[key]?.ratio
  if (typeof metaRatio === 'number' && Number.isFinite(metaRatio) && metaRatio > 0) return metaRatio
  const val = groupRatio.value[key]
  return typeof val === 'number' && Number.isFinite(val) && val > 0 ? val : 1
})

const vendorMap = computed(() => {
  const m = new Map<number, string>()
  for (const v of vendors.value) {
    m.set(v.id, v.name)
  }
  return m
})

const vendorName = (id?: number) => (id ? vendorMap.value.get(id) || '' : '')

const visibleModelGroups = (item: GatewayModelPricingItem): string[] => {
  const egs = Array.isArray(item.enable_groups) ? item.enable_groups : []
  if (egs.includes('all')) {
    return activeGroupCode.value ? [activeGroupCode.value] : []
  }
  const authorized = egs.filter(g => allowedGroupCodeSet.value.has(g))
  if (authorized.length) return authorized
  return activeGroupCode.value ? [activeGroupCode.value] : egs
}

const filteredModels = computed(() => {
  const kw = searchKeyword.value.toLowerCase()
  const grp = activeGroupCode.value
  const metaModels = new Set(groupMeta.value[grp]?.models || [])
  return models.value.filter((item) => {
    if (grp) {
      const egs = Array.isArray(item.enable_groups) ? item.enable_groups : []
      const inPricing = egs.length === 0 || egs.includes('all') || egs.includes(grp)
      const inMeta = metaModels.size > 0 && metaModels.has(item.model_name)
      if (!inPricing && !inMeta) {
        return false
      }
    }
    if (billingFilter.value === 'token' && item.quota_type !== 0) return false
    if (billingFilter.value === 'fixed' && item.quota_type !== 1) return false
    if (!kw) return true
    const grpLabels = visibleModelGroups(item).map(formatGroup).join(' ').toLowerCase()
    return (
      item.model_name.toLowerCase().includes(kw)
      || vendorName(item.vendor_id).toLowerCase().includes(kw)
      || grpLabels.includes(kw)
    )
  })
})

watch([searchKeyword, selectedGroup, billingFilter], () => {
  modelPage.value = 1
})

const totalModelPages = computed(() => Math.max(1, Math.ceil(filteredModels.value.length / modelPageSize)))
const pagedModels = computed(() => {
  const start = (modelPage.value - 1) * modelPageSize
  return filteredModels.value.slice(start, start + modelPageSize)
})

const formatUnitUsd = (val: number) => {
  if (!Number.isFinite(val) || val <= 0) return '0.00'
  if (val >= 1) return val.toFixed(2)
  if (val >= 0.01) return val.toFixed(4).replace(/0+$/, '').replace(/\.$/, '')
  return val.toFixed(6).replace(/0+$/, '').replace(/\.$/, '')
}

const formatNumber = (n: number) => (Number.isFinite(n) ? n.toLocaleString('en-US') : '0')
const formatCompactNumber = (n: number) => {
  if (!Number.isFinite(n) || n <= 0) return '0'
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(2)}M`
  if (n >= 1_000) return `${(n / 1_000).toFixed(1)}K`
  return String(Math.round(n))
}

const sharePercent = (quota: number) => {
  if (!usage.value.total_quota || usage.value.total_quota <= 0) return 0
  return Math.max(1, Math.min(100, Math.round((quota / usage.value.total_quota) * 100)))
}

const copiedModel = ref('')
let copyTimer: ReturnType<typeof setTimeout> | undefined
const copyModelId = async (modelId: string) => {
  if (copyTimer) clearTimeout(copyTimer)
  try {
    await navigator.clipboard.writeText(modelId)
    copiedModel.value = modelId
    copyTimer = setTimeout(() => { copiedModel.value = '' }, 1500)
  } catch {
    copiedModel.value = ''
  }
}

const tryInPlayground = (modelId: string) => {
  void router.push({ path: '/me/playground', query: { model: modelId } })
}

const loadModels = async () => {
  modelsLoading.value = true
  errorMsg.value = ''
  try {
    const res = await getGatewayModels()
    models.value = Array.isArray(res.models) ? res.models : []
    vendors.value = Array.isArray(res.vendors) ? res.vendors : []
    usableGroups.value = res.usable_groups || {}
    groupRatio.value = res.group_ratio || {}
    groupMeta.value = res.group_meta || {}
    defaultGroup.value = res.default_group || 'default'
    const available = selectableGroups.value
    if (!selectedGroup.value || !available.some(g => g.code === selectedGroup.value)) {
      selectedGroup.value = available[0]?.code || defaultGroup.value
    }
  } catch (err: any) {
    errorMsg.value = err?.response?.data?.msg || t('needtoken.memberConsole.loadFailed')
  } finally {
    modelsLoading.value = false
  }
}

const loadAnalytics = async () => {
  usageLoading.value = true
  try {
    usage.value = await getGatewayUsage(analyticsDays.value)
  } catch {
    // keep previous data
  } finally {
    usageLoading.value = false
  }
}

const switchToAnalytics = () => {
  activeTab.value = 'analytics'
  if (!usage.value.models.length) {
    void loadAnalytics()
  }
}

const changeAnalyticsDays = (days: number) => {
  analyticsDays.value = days
  void loadAnalytics()
}

onMounted(() => {
  void loadModels()
})
</script>
