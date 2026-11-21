<template>
  <div class="mx-auto w-full max-w-[1200px] px-4 pb-12 pt-10 sm:px-6">
    <header class="flex flex-col justify-between gap-4 sm:flex-row sm:items-end">
      <div>
        <p class="text-[12px] font-semibold uppercase tracking-[0.14em] text-primary">{{ t('needtoken.models.eyebrow') }}</p>
        <h1 class="mt-2 text-[30px] font-bold tracking-tight text-foreground sm:text-[34px]">{{ t('needtoken.models.title') }}</h1>
        <p class="mt-2 max-w-[72ch] text-[14px] leading-relaxed text-muted-foreground">{{ t('needtoken.models.subtitle') }}</p>
      </div>
      <div class="flex flex-wrap items-center gap-2 text-[12px] text-muted-foreground">
        <span v-if="uniqueModelCount > 0" class="rounded-full border border-border bg-card px-3 py-1 font-medium text-foreground">
          {{ t('needtoken.models.statsModels', { count: uniqueModelCount, vendors: vendorStats.length }) }}
        </span>
        <span v-if="pricing.loadedAt.value">
          {{ t('needtoken.models.updatedAt', { time: pricing.loadedAt.value.toLocaleTimeString() }) }}
        </span>
      </div>
    </header>

    <!-- 分组切换器（展示 usable_group 说明与 group_ratio 倍率，切换实时重算单价） -->
    <section
      v-if="pricing.groups.value.length > 0"
      class="mt-6 rounded-2xl border border-border bg-card p-4"
      :aria-label="t('needtoken.models.groupSwitcherLabel')"
    >
      <div class="flex flex-wrap items-center justify-between gap-2">
        <span class="text-[12px] font-semibold text-foreground">{{ t('needtoken.models.groupSwitcherTitle') }}</span>
        <span class="text-[12px] text-muted-foreground">{{ activeGroupHint }}</span>
      </div>
      <div class="mt-3 flex flex-wrap gap-2">
        <button
          type="button"
          class="inline-flex min-h-8 items-center gap-1.5 rounded-full border px-3 py-1 text-[12px] font-medium transition-colors"
          :class="selectedGroup === '' ? 'border-primary bg-primary/10 text-primary' : 'border-border bg-background/60 text-muted-foreground hover:text-foreground'"
          :aria-pressed="selectedGroup === ''"
          @click="selectedGroup = ''"
        >
          <span>{{ t('needtoken.models.groupRecommended') }}</span>
        </button>
        <button
          v-for="grp in pricing.groups.value"
          :key="grp.name"
          type="button"
          class="inline-flex min-h-8 items-center gap-1.5 rounded-full border px-3 py-1 text-[12px] font-medium transition-colors"
          :class="selectedGroup === grp.name ? 'border-primary bg-primary/10 text-primary' : 'border-border bg-background/60 text-muted-foreground hover:text-foreground'"
          :aria-pressed="selectedGroup === grp.name"
          @click="selectedGroup = grp.name"
        >
          <span class="h-2 w-2 rounded-full" :class="groupDotClass(grp.name)" aria-hidden="true" />
          <span class="nt-mono">{{ grp.name }}</span>
          <span class="nt-mono opacity-75">×{{ grp.ratio }}</span>
        </button>
        <button
          v-if="pricing.groups.value.length > 1"
          type="button"
          class="inline-flex min-h-8 items-center gap-1.5 rounded-full border px-3 py-1 text-[12px] font-medium transition-colors"
          :class="selectedGroup === '__all_offers__' ? 'border-primary bg-primary/10 text-primary' : 'border-border bg-background/60 text-muted-foreground hover:text-foreground'"
          :aria-pressed="selectedGroup === '__all_offers__'"
          @click="selectedGroup = '__all_offers__'"
        >
          <span>{{ t('needtoken.models.groupExpandAll') }}</span>
        </button>
      </div>
    </section>

    <!-- 厂商筛选胶囊栏 -->
    <nav
      v-if="vendorStats.length > 0"
      class="mt-4 flex items-center gap-2 overflow-x-auto pb-1"
      :aria-label="t('needtoken.models.vendor')"
    >
      <button
        type="button"
        class="inline-flex min-h-8 shrink-0 items-center gap-1.5 rounded-full border px-3.5 py-1 text-[12.5px] font-medium transition-colors"
        :class="vendor === '' ? 'border-primary bg-primary text-primary-foreground' : 'border-border bg-card text-muted-foreground hover:text-foreground'"
        :aria-pressed="vendor === ''"
        @click="vendor = ''"
      >
        <span>{{ t('needtoken.models.allVendors') }}</span>
        <span class="nt-mono text-[11px] opacity-80">{{ groupScopedOffers.length }}</span>
      </button>
      <button
        v-for="item in vendorStats"
        :key="item.name"
        type="button"
        class="inline-flex min-h-8 shrink-0 items-center gap-1.5 rounded-full border px-3.5 py-1 text-[12.5px] font-medium transition-colors"
        :class="vendor === item.name ? 'border-primary bg-primary text-primary-foreground' : 'border-border bg-card text-muted-foreground hover:text-foreground'"
        :aria-pressed="vendor === item.name"
        @click="vendor = vendor === item.name ? '' : item.name"
      >
        <span>{{ item.name }}</span>
        <span class="nt-mono text-[11px] opacity-80">{{ item.count }}</span>
      </button>
    </nav>

    <!-- 搜索、计费方式与排序 -->
    <div class="mt-4 flex flex-wrap items-center gap-3">
      <input
        v-model="keyword"
        type="search"
        class="nt-mono h-10 w-full rounded-lg border border-border bg-card px-3.5 text-[13.5px] text-foreground placeholder:text-muted-foreground sm:w-[260px]"
        :placeholder="t('needtoken.models.search')"
        :aria-label="t('needtoken.models.search')"
      />

      <div class="inline-flex rounded-lg border border-border bg-card p-1" role="group" :aria-label="t('needtoken.models.billingFilterLabel')">
        <button
          v-for="mode in billingModes"
          :key="mode.value"
          type="button"
          class="rounded-md px-2.5 py-1 text-[12px] font-medium transition-colors"
          :class="billingType === mode.value ? 'bg-primary/15 text-primary' : 'text-muted-foreground hover:text-foreground'"
          :aria-pressed="billingType === mode.value"
          @click="billingType = mode.value"
        >
          {{ t(mode.labelKey) }}
        </button>
      </div>

      <NeedtokenSelect
        v-if="supplyOptions.length > 1 && selectedGroup === '__all_offers__'"
        v-model="supply"
        :options="supplySelectOptions"
        :label="t('needtoken.models.supply')"
        align="start"
      />
      <NeedtokenSelect
        :model-value="sort"
        :options="sortSelectOptions"
        :label="t('needtoken.models.sort')"
        align="start"
        @update:model-value="selectSort"
      />

      <button
        v-if="hasActiveFilters"
        type="button"
        class="text-[12px] font-medium text-primary hover:underline"
        @click="resetFilters"
      >
        {{ t('products.clearFilters') }}
      </button>
    </div>

    <p
      v-if="(pricing.state.value === 'error' || pricing.isStale.value) && offers.length"
      role="status"
      class="mt-5 rounded-lg border border-amber-500/40 bg-amber-500/5 p-3 text-[12px] text-muted-foreground"
    >
      {{ t('needtoken.models.stalePrice') }}
    </p>

    <!-- 状态 -->
    <div v-if="pricing.isLoading.value" class="mt-10 rounded-xl border border-border px-6 py-12 text-center text-muted-foreground">{{ t('needtoken.common.loading') }}</div>
    <div v-else-if="pricing.state.value === 'error' && offers.length === 0" class="mt-10 rounded-xl border border-border px-6 py-12 text-center">
      <p class="text-muted-foreground">{{ t('needtoken.models.loadError') }}</p>
      <button type="button" class="mt-4 rounded-md border border-border px-4 py-2 text-[14px] text-foreground hover:bg-secondary" @click="pricing.reload()">{{ t('needtoken.common.retry') }}</button>
    </div>
    <div v-else-if="offers.length === 0" class="mt-10 rounded-xl border border-border px-6 py-12 text-center text-muted-foreground">{{ t('needtoken.common.pricingPending') }}</div>
    <div v-else-if="visible.length === 0" class="mt-10 rounded-xl border border-border px-6 py-12 text-center text-muted-foreground">{{ t('needtoken.models.noMatch') }}</div>

    <!-- 模型卡网格 -->
    <div v-else class="mt-6 grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
      <article
        v-for="offer in visible"
        :key="offer.key"
        class="nt-model-card flex flex-col rounded-2xl border bg-card p-5"
        :class="compare.includes(offer.key) ? 'border-primary' : 'border-border'"
      >
        <div class="flex items-start justify-between gap-2">
          <div class="min-w-0 flex-1">
            <div class="flex items-center gap-1.5">
              <h2 class="nt-model-name nt-mono truncate text-[15.5px] font-semibold text-foreground" :title="offer.modelId">
                {{ offer.modelId }}
              </h2>
              <button
                type="button"
                class="inline-flex h-6 w-6 shrink-0 items-center justify-center rounded-md text-muted-foreground hover:bg-secondary hover:text-foreground"
                :aria-label="t('needtoken.models.copyModelId', { model: offer.modelId })"
                :title="copiedModelId === offer.modelId ? t('needtoken.models.copied') : t('needtoken.models.copyIdShort')"
                @click="copyModelId(offer.modelId)"
              >
                <Check v-if="copiedModelId === offer.modelId" class="h-3.5 w-3.5 text-emerald-500" aria-hidden="true" />
                <Copy v-else class="h-3.5 w-3.5" aria-hidden="true" />
              </button>
            </div>
            <div class="mt-1 flex flex-wrap items-center gap-2 text-[12px] text-muted-foreground">
              <span class="font-medium text-foreground/85">{{ offer.vendor }}</span>
              <span>·</span>
              <span class="nt-mono">{{ supplyLabel(offer) }} (×{{ offer.groupRatio }})</span>
            </div>
          </div>

          <div class="flex shrink-0 items-center gap-1.5">
            <span v-if="offer.unit === 'per_call'" class="rounded-full border border-amber-500/40 bg-amber-500/10 px-2 py-0.5 text-[11px] font-medium text-amber-600 dark:text-amber-400">
              {{ t('needtoken.models.perCallBadge') }}
            </span>
            <span v-if="offer.isDefault" class="rounded-full bg-primary/15 px-2 py-0.5 text-[11px] font-medium text-primary">
              {{ t('needtoken.models.defaultBadge') }}
            </span>
          </div>
        </div>

        <!-- 号池健康灯 + 支持端点协议徽标 -->
        <div class="mt-3 flex flex-wrap items-center gap-1.5 text-[11px]">
          <span class="inline-flex items-center gap-1.5 rounded-full border px-2 py-0.5" :class="statusClass(offer)" :title="statusTitle">
            <span class="h-1.5 w-1.5 rounded-full" :class="groupDotClass(offer.group)" aria-hidden="true" />
            <span>{{ statusLabel(offer) }}</span>
          </span>
          <span
            v-for="ep in offer.endpoints"
            :key="`${offer.key}-${ep.label}`"
            class="nt-mono rounded border border-border bg-secondary/50 px-1.5 py-0.5 text-muted-foreground"
            :title="ep.path"
          >
            {{ ep.label }}
          </span>
        </div>

        <!-- 模型描述与标签 -->
        <p v-if="offer.description" class="mt-2.5 line-clamp-2 text-[12px] leading-relaxed text-muted-foreground">
          {{ offer.description }}
        </p>
        <div v-if="offer.tags.length > 0" class="mt-2 flex flex-wrap gap-1.5 text-[11px]">
          <span
            v-for="tag in offer.tags.slice(0, 4)"
            :key="`${offer.key}-tag-${tag}`"
            class="rounded bg-primary/8 px-1.5 py-0.5 text-muted-foreground"
          >
            #{{ tag }}
          </span>
        </div>

        <!-- 计费详情 -->
        <div v-if="offer.unit === 'conditional'" class="mt-4 rounded-lg bg-secondary/60 p-3 text-[12px] text-muted-foreground">
          <p class="font-medium text-foreground">{{ t('needtoken.models.conditionalPrice') }}</p>
          <p class="mt-1">{{ t(`needtoken.models.conditions.${offer.condition || 'unknown'}`) }}</p>
          <p v-if="offer.usageUnits.length" class="nt-mono mt-2 break-words">{{ t('needtoken.models.usageFields') }}: {{ offer.usageUnits.join(', ') }}</p>
        </div>

        <div v-else-if="offer.unit === 'per_call'" class="nt-mono mt-4 rounded-lg border border-border/70 bg-secondary/35 px-3.5 py-2.5">
          <div class="text-[11px] text-muted-foreground">{{ t('needtoken.models.columns.perCall') }}</div>
          <div class="mt-0.5 flex items-baseline gap-1.5">
            <span class="text-[18px] font-bold text-foreground">{{ formatUsd(offer.inputPrice) }}</span>
            <span class="text-[11px] text-muted-foreground">/ {{ t('needtoken.models.perCallShort') }}</span>
          </div>
        </div>

        <dl
          v-else
          class="nt-mono mt-4 grid gap-2.5 rounded-lg border border-border/70 bg-secondary/35 p-3 text-[12px]"
          :class="offer.cachePrice !== null ? 'grid-cols-3' : 'grid-cols-2'"
        >
          <div>
            <dt class="text-[11px] text-muted-foreground">{{ t('needtoken.models.columns.input') }}</dt>
            <dd class="mt-0.5 text-[14.5px] font-semibold text-foreground">{{ formatUsd(offer.inputPrice) }}</dd>
          </div>
          <div>
            <dt class="text-[11px] text-muted-foreground">{{ t('needtoken.models.columns.output') }}</dt>
            <dd class="mt-0.5 text-[14.5px] font-semibold text-foreground">{{ formatUsd(offer.outputPrice) }}</dd>
          </div>
          <div v-if="offer.cachePrice !== null">
            <dt class="text-[11px] text-emerald-600 dark:text-emerald-400">{{ t('needtoken.models.columns.cacheHit') }}</dt>
            <dd class="mt-0.5 text-[14.5px] font-semibold text-emerald-600 dark:text-emerald-400">{{ formatUsd(offer.cachePrice) }}</dd>
          </div>
        </dl>

        <p class="mt-2 text-[11px] text-muted-foreground">{{ unitLabel(offer) }}</p>

        <!-- 操作区 -->
        <div class="mt-auto flex flex-wrap items-center gap-2 pt-4">
          <button
            type="button"
            class="inline-flex items-center gap-1.5 rounded-lg bg-primary px-3 py-1.5 text-[12.5px] font-semibold text-primary-foreground hover:bg-primary/90 disabled:opacity-60"
            :disabled="ssoLoading"
            @click="invokeModelInConsole(offer.modelId)"
          >
            <LoaderCircle v-if="ssoLoading && invokingModel === offer.modelId" class="h-3.5 w-3.5 animate-spin" aria-hidden="true" />
            <span>{{ t('needtoken.models.callInConsole') }}</span>
            <ArrowUpRight class="h-3.5 w-3.5" aria-hidden="true" />
          </button>
          <RouterLink
            :to="{ path: '/docs', query: { model: offer.modelId }, hash: '#opencode' }"
            class="rounded-lg border border-border px-3 py-1.5 text-[12.5px] text-foreground hover:bg-secondary"
          >
            {{ t('needtoken.models.configure') }}
          </RouterLink>
          <label class="ml-auto flex cursor-pointer items-center gap-1.5 text-[12px] text-muted-foreground">
            <input
              type="checkbox"
              class="accent-[var(--ui-accent)]"
              :checked="compare.includes(offer.key)"
              :disabled="!compare.includes(offer.key) && compare.length >= MAX_COMPARE"
              @change="compare = toggleCompare(compare, offer.key)"
            />
            {{ t('needtoken.models.compare') }}
          </label>
        </div>
      </article>
    </div>

    <p class="mt-6 text-[12px] text-muted-foreground">{{ t('needtoken.common.priceUnitNote') }} {{ t('needtoken.models.sourceNote') }}</p>

    <!-- 比价抽屉 -->
    <section v-if="compared.length > 0" class="sticky bottom-4 z-20 mt-8 rounded-2xl border border-primary bg-card p-4 shadow-lg">
      <div class="flex items-center justify-between gap-3">
        <h2 class="text-[15px] font-semibold text-foreground">{{ t('needtoken.models.compareTitle', { n: compared.length, max: MAX_COMPARE }) }}</h2>
        <button type="button" class="text-[13px] text-muted-foreground hover:text-foreground" @click="compare = []">{{ t('needtoken.models.compareClear') }}</button>
      </div>
      <div class="mt-3 overflow-x-auto">
        <table class="w-full min-w-[580px] text-left text-[13px]">
          <thead class="text-muted-foreground">
            <tr>
              <th class="py-2 pr-4 font-medium">{{ t('needtoken.models.columns.model') }}</th>
              <th class="py-2 pr-4 font-medium">{{ t('needtoken.models.columns.supply') }}</th>
              <th class="py-2 pr-4 font-medium">{{ t('needtoken.models.columns.unit') }}</th>
              <th class="py-2 pr-4 font-medium">{{ t('needtoken.models.columns.input') }}</th>
              <th class="py-2 pr-4 font-medium">{{ t('needtoken.models.columns.output') }}</th>
              <th class="py-2 pr-4 font-medium">{{ t('needtoken.models.columns.cacheHit') }}</th>
              <th class="py-2 font-medium"></th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="o in compared" :key="`c-${o.key}`" class="border-t border-border">
              <td class="nt-mono py-2 pr-4 text-foreground">{{ o.modelId }}</td>
              <td class="py-2 pr-4 text-muted-foreground">{{ supplyLabel(o) }} (×{{ o.groupRatio }})</td>
              <td class="py-2 pr-4 text-muted-foreground">{{ unitLabel(o) }}</td>
              <td v-if="o.unit === 'conditional'" colspan="3" class="py-2 pr-4 text-muted-foreground">{{ t('needtoken.models.conditionalPrice') }}</td>
              <template v-else>
                <td class="nt-mono py-2 pr-4 text-foreground">{{ formatUsd(o.inputPrice) }}</td>
                <td class="nt-mono py-2 pr-4 text-foreground">{{ formatUsd(o.outputPrice) }}</td>
                <td class="nt-mono py-2 pr-4 text-emerald-600 dark:text-emerald-400">{{ formatUsd(o.cachePrice) }}</td>
              </template>
              <td class="py-2 text-right">
                <button type="button" class="text-muted-foreground hover:text-foreground" :aria-label="t('needtoken.models.compareRemove')" @click="compare = toggleCompare(compare, o.key)">×</button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import { ArrowUpRight, Check, Copy, LoaderCircle } from 'lucide-vue-next'
import { usePageSeo } from '../../composables/usePageSeo'
import { useUserAuthStore } from '../../stores/userAuth'
import { useAppStore } from '../../stores/app'
import { CONSOLE_URL, DEFAULT_MODELS } from './config'
import {
  filterOffers,
  formatUsd,
  MAX_COMPARE,
  selectOffersByGroup,
  sortOffers,
  supplyKeyOf,
  toggleCompare,
  type ModelOffer,
  type OfferSort,
} from './lib/pricing'
import { usePricing } from './lib/usePricing'
import { levelFor, type PoolLevel } from './lib/poolStatus'
import { usePoolStatus } from './lib/usePoolStatus'
import { useApiAccountLink } from './composables/useApiAccountLink'
import NeedtokenSelect from './components/NeedtokenSelect.vue'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const userAuth = useUserAuthStore()
const appStore = useAppStore()
const pricing = usePricing()
const pool = usePoolStatus()
const { ssoLoading, openApiConsoleSSO } = useApiAccountLink()

usePageSeo({
  title: () => t('needtoken.models.title'),
  description: () => t('needtoken.models.subtitle'),
  canonicalPath: () => '/models',
})

const keyword = ref(typeof route.query.q === 'string' ? route.query.q : '')
const vendor = ref(typeof route.query.vendor === 'string' ? route.query.vendor : '')
const selectedGroup = ref(typeof route.query.group === 'string' ? route.query.group : '')
const billingType = ref<'' | 'token' | 'per_call'>('')
const supply = ref('')
const sort = ref<OfferSort>('default')
const compare = ref<string[]>([])
const copiedModelId = ref('')
const invokingModel = ref('')
let copyTimer: ReturnType<typeof setTimeout> | undefined

const billingModes = [
  { value: '' as const, labelKey: 'needtoken.models.billingAll' },
  { value: 'token' as const, labelKey: 'needtoken.models.billingToken' },
  { value: 'per_call' as const, labelKey: 'needtoken.models.billingPerCall' },
]

const offers = computed(() => pricing.offers.value)
const groupScopedOffers = computed(() => selectOffersByGroup(offers.value, selectedGroup.value))
const uniqueModelCount = computed(() => new Set(offers.value.map(o => o.modelId)).size)

const activeGroupHint = computed(() => {
  if (!selectedGroup.value) return t('needtoken.models.groupRecommendedHint')
  if (selectedGroup.value === '__all_offers__') return t('needtoken.models.groupExpandAllHint')
  const matched = pricing.groups.value.find(g => g.name === selectedGroup.value)
  if (!matched) return ''
  return matched.description
    ? `${matched.description} · ${t('needtoken.models.groupMultiplier', { ratio: matched.ratio })}`
    : t('needtoken.models.groupMultiplier', { ratio: matched.ratio })
})

const supplyLabel = (offer: ModelOffer) =>
  offer.supplyLine ? t(`needtoken.supply.${offer.supplyLine}`) : offer.group

const unitLabel = (offer: ModelOffer) => t(`needtoken.models.units.${offer.unit}`)

const PREFERRED_VENDOR_ORDER = ['OpenAI', 'Anthropic', 'Google', 'DeepSeek', 'xAI', 'Qwen', 'Zhipu', 'Moonshot', 'Meta', 'Mistral']

const vendorStats = computed(() => {
  const counts = new Map<string, number>()
  for (const offer of groupScopedOffers.value) {
    counts.set(offer.vendor, (counts.get(offer.vendor) || 0) + 1)
  }
  return [...counts.entries()]
    .map(([name, count]) => ({ name, count }))
    .sort((a, b) => {
      const rankA = PREFERRED_VENDOR_ORDER.indexOf(a.name)
      const rankB = PREFERRED_VENDOR_ORDER.indexOf(b.name)
      if (rankA !== -1 && rankB !== -1) return rankA - rankB
      if (rankA !== -1) return -1
      if (rankB !== -1) return 1
      return b.count - a.count || a.name.localeCompare(b.name)
    })
})

const supplyOptions = computed(() => {
  const seen = new Map<string, string>()
  for (const o of offers.value) {
    const key = supplyKeyOf(o)
    if (!seen.has(key)) seen.set(key, supplyLabel(o))
  }
  return [...seen.entries()].map(([value, label]) => ({ value, label }))
})
const supplySelectOptions = computed(() => [{ value: '', label: t('needtoken.models.allSupply') }, ...supplyOptions.value])
const sortSelectOptions = computed(() => [
  { value: 'default', label: t('needtoken.models.sortDefault') },
  { value: 'price_asc', label: t('needtoken.models.sortPriceAsc') },
  { value: 'price_desc', label: t('needtoken.models.sortPriceDesc') },
])
const selectSort = (value: string) => {
  if (value === 'default' || value === 'price_asc' || value === 'price_desc') sort.value = value
}

const hasActiveFilters = computed(() =>
  Boolean(keyword.value.trim() || vendor.value || billingType.value || supply.value || selectedGroup.value),
)

const resetFilters = () => {
  keyword.value = ''
  vendor.value = ''
  billingType.value = ''
  supply.value = ''
  selectedGroup.value = ''
}

const visible = computed(() =>
  sortOffers(
    filterOffers(groupScopedOffers.value, {
      vendor: vendor.value,
      supply: selectedGroup.value === '__all_offers__' ? supply.value : '',
      billingType: billingType.value,
      keyword: keyword.value,
    }),
    sort.value,
    DEFAULT_MODELS,
  ),
)

const compared = computed(() =>
  compare.value.map(key => offers.value.find(o => o.key === key)).filter((o): o is ModelOffer => Boolean(o)),
)

const levelClass: Record<PoolLevel, string> = {
  available: 'border-emerald-500/45 text-emerald-600 dark:text-emerald-400',
  busy: 'border-amber-500/50 text-amber-600 dark:text-amber-400',
  unavailable: 'border-red-500/50 text-red-600 dark:text-red-400',
  unknown: 'border-border text-muted-foreground',
}
const dotClassMap: Record<PoolLevel, string> = {
  available: 'bg-emerald-500',
  busy: 'bg-amber-500',
  unavailable: 'bg-red-500',
  unknown: 'bg-muted-foreground/45',
}
const levelOf = (offer: ModelOffer) => levelFor(pool.view.value, offer.group)
const groupDotClass = (group: string) => dotClassMap[levelFor(pool.view.value, group) ?? 'unknown']
const statusLabel = (offer: ModelOffer) => {
  const level = levelOf(offer)
  if (!level) return t('needtoken.models.statusPending')
  return pool.view.value?.stale
    ? t('needtoken.models.statusStale', { status: t(`needtoken.models.status.${level}`) })
    : t(`needtoken.models.status.${level}`)
}
const statusClass = (offer: ModelOffer) => levelClass[levelOf(offer) ?? 'unknown']
const statusTitle = computed(() => {
  const at = pool.view.value?.updatedAt
  return at ? t('needtoken.models.statusUpdatedAt', { time: at.toLocaleTimeString() }) : undefined
})

const copyModelId = async (modelId: string) => {
  if (copyTimer) clearTimeout(copyTimer)
  try {
    await navigator.clipboard.writeText(modelId)
    copiedModelId.value = modelId
    copyTimer = setTimeout(() => {
      copiedModelId.value = ''
    }, 1600)
  } catch {
    copiedModelId.value = ''
  }
}

const invokeModelInConsole = async (modelId: string) => {
  if (appStore.isResellerTenant) {
    window.open(`${CONSOLE_URL}/playground`, '_blank', 'noopener,noreferrer')
    return
  }
  if (!userAuth.isAuthenticated) {
    void router.push({ path: '/auth/login', query: { redirect: `/models?q=${encodeURIComponent(modelId)}` } })
    return
  }
  invokingModel.value = modelId
  await openApiConsoleSSO('/playground', '/models')
  invokingModel.value = ''
}

onMounted(() => {
  void pricing.ensureLoaded()
  void pool.ensureLoaded()
})

onUnmounted(() => {
  if (copyTimer) clearTimeout(copyTimer)
})
</script>

<style scoped>
.nt-model-card {
  transition: background-color var(--nt-motion-fast) var(--nt-motion-ease), border-color var(--nt-motion-fast) var(--nt-motion-ease);
}
.nt-model-card:is(:hover, :focus-within) {
  background: var(--nt-surface-hover);
  border-color: var(--nt-accent-border-strong, var(--ui-border-strong));
}
.nt-model-name {
  transition: color var(--nt-motion-fast) var(--nt-motion-ease);
}
.nt-model-card:is(:hover, :focus-within) .nt-model-name {
  color: var(--nt-accent-ink-hover);
}
@media (prefers-reduced-motion: reduce) {
  .nt-model-card,
  .nt-model-name {
    transition: none;
  }
}
</style>
