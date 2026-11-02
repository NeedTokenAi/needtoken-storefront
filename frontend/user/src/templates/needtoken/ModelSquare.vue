<template>
  <div class="mx-auto w-full max-w-[1200px] px-6 pb-10 pt-12">
    <header>
      <h1 class="text-[32px] font-bold text-foreground">{{ t('needtoken.models.title') }}</h1>
      <p class="mt-2 max-w-[70ch] text-[15px] text-muted-foreground">{{ t('needtoken.models.subtitle') }}</p>
    </header>

    <!-- 筛选与排序 -->
    <div class="mt-8 flex flex-wrap items-center gap-3">
      <input
        v-model="keyword"
        type="search"
        class="nt-mono h-10 w-full rounded-md border border-border bg-card px-3 text-[14px] text-foreground placeholder:text-muted-foreground sm:w-[240px]"
        :placeholder="t('needtoken.models.search')"
        :aria-label="t('needtoken.models.search')"
      />
      <select v-model="vendor" class="h-10 rounded-md border border-border bg-card px-3 text-[14px] text-foreground" :aria-label="t('needtoken.models.vendor')">
        <option value="">{{ t('needtoken.models.allVendors') }}</option>
        <option v-for="v in vendorOptions" :key="v" :value="v">{{ v }}</option>
      </select>
      <select v-model="supply" class="h-10 rounded-md border border-border bg-card px-3 text-[14px] text-foreground" :aria-label="t('needtoken.models.supply')">
        <option value="">{{ t('needtoken.models.allSupply') }}</option>
        <option v-for="s in supplyOptions" :key="s.value" :value="s.value">{{ s.label }}</option>
      </select>
      <select v-model="sort" class="h-10 rounded-md border border-border bg-card px-3 text-[14px] text-foreground" :aria-label="t('needtoken.models.sort')">
        <option value="default">{{ t('needtoken.models.sortDefault') }}</option>
        <option value="price_asc">{{ t('needtoken.models.sortPriceAsc') }}</option>
        <option value="price_desc">{{ t('needtoken.models.sortPriceDesc') }}</option>
      </select>
      <span class="ml-auto text-[12px] text-muted-foreground">
        <template v-if="pricing.loadedAt.value">{{ t('needtoken.models.updatedAt', { time: pricing.loadedAt.value.toLocaleTimeString() }) }}</template>
      </span>
    </div>

    <!-- 状态 -->
    <div v-if="pricing.isLoading.value" class="mt-10 rounded-xl border border-border px-6 py-12 text-center text-muted-foreground">{{ t('needtoken.common.loading') }}</div>
    <div v-else-if="pricing.state.value === 'error' && offers.length === 0" class="mt-10 rounded-xl border border-border px-6 py-12 text-center">
      <p class="text-muted-foreground">{{ t('needtoken.models.loadError') }}</p>
      <button type="button" class="mt-4 rounded-md border border-border px-4 py-2 text-[14px] text-foreground hover:bg-secondary" @click="pricing.reload()">{{ t('needtoken.common.retry') }}</button>
    </div>
    <div v-else-if="offers.length === 0" class="mt-10 rounded-xl border border-border px-6 py-12 text-center text-muted-foreground">{{ t('needtoken.common.pricingPending') }}</div>
    <div v-else-if="visible.length === 0" class="mt-10 rounded-xl border border-border px-6 py-12 text-center text-muted-foreground">{{ t('needtoken.models.noMatch') }}</div>

    <!-- 模型卡 -->
    <div v-else class="mt-8 grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
      <article v-for="offer in visible" :key="offer.key" class="flex flex-col rounded-xl border bg-card p-5" :class="compare.includes(offer.key) ? 'border-primary' : 'border-border'">
        <div class="flex items-start justify-between gap-3">
          <div class="min-w-0">
            <h2 class="nt-mono truncate text-[16px] font-semibold text-foreground">{{ offer.modelId }}</h2>
            <div class="mt-1 text-[13px] text-muted-foreground">{{ offer.vendor }}</div>
          </div>
          <span v-if="offer.isDefault" class="rounded bg-primary/15 px-1.5 py-0.5 text-[11px] text-primary">{{ t('needtoken.models.defaultBadge') }}</span>
        </div>
        <div class="mt-3 flex flex-wrap gap-2 text-[12px]">
          <span class="rounded border border-border px-2 py-0.5 text-muted-foreground">{{ supplyLabel(offer) }}</span>
<<<<<<< HEAD
          <span class="rounded border px-2 py-0.5" :class="statusClass(offer)" :title="statusTitle">{{ statusLabel(offer) }}</span>
=======
          <span class="rounded border border-border px-2 py-0.5 text-muted-foreground">{{ t('needtoken.models.statusPending') }}</span>
>>>>>>> origin/claude/project-thread-6r3cwt
        </div>
        <dl class="nt-mono mt-4 grid grid-cols-2 gap-3 text-[13px]">
          <div>
            <dt class="text-muted-foreground">{{ offer.unit === 'per_call' ? t('needtoken.models.columns.perCall') : t('needtoken.models.columns.input') }}</dt>
            <dd class="mt-0.5 text-[15px] text-foreground">{{ formatUsd(offer.inputPrice) }}</dd>
          </div>
          <div v-if="offer.unit !== 'per_call'">
            <dt class="text-muted-foreground">{{ t('needtoken.models.columns.output') }}</dt>
            <dd class="mt-0.5 text-[15px] text-foreground">{{ formatUsd(offer.outputPrice) }}</dd>
          </div>
        </dl>
        <div class="mt-5 flex flex-wrap items-center gap-2 pt-1">
          <RouterLink to="/plans" class="rounded-md bg-primary px-3 py-1.5 text-[13px] font-semibold text-primary-foreground hover:bg-primary/90">{{ t('needtoken.models.buy') }}</RouterLink>
          <RouterLink :to="{ path: '/docs', query: { model: offer.modelId }, hash: '#opencode' }" class="rounded-md border border-border px-3 py-1.5 text-[13px] text-foreground hover:bg-secondary">{{ t('needtoken.models.configure') }}</RouterLink>
          <label class="ml-auto flex cursor-pointer items-center gap-1.5 text-[13px] text-muted-foreground">
            <input type="checkbox" class="accent-[var(--ui-accent)]" :checked="compare.includes(offer.key)" :disabled="!compare.includes(offer.key) && compare.length >= MAX_COMPARE" @change="compare = toggleCompare(compare, offer.key)" />
            {{ t('needtoken.models.compare') }}
          </label>
        </div>
      </article>
    </div>

    <p class="mt-6 text-[12px] text-muted-foreground">{{ t('needtoken.common.priceUnitNote') }} {{ t('needtoken.models.sourceNote') }}</p>

    <!-- 比价 -->
    <section v-if="compared.length > 0" class="sticky bottom-4 mt-8 rounded-xl border border-primary bg-card p-4">
      <div class="flex items-center justify-between gap-3">
        <h2 class="text-[15px] font-semibold text-foreground">{{ t('needtoken.models.compareTitle', { n: compared.length, max: MAX_COMPARE }) }}</h2>
        <button type="button" class="text-[13px] text-muted-foreground hover:text-foreground" @click="compare = []">{{ t('needtoken.models.compareClear') }}</button>
      </div>
      <div class="mt-3 overflow-x-auto">
        <table class="w-full min-w-[520px] text-left text-[13px]">
          <thead class="text-muted-foreground">
            <tr>
              <th class="py-2 pr-4 font-medium">{{ t('needtoken.models.columns.model') }}</th>
              <th class="py-2 pr-4 font-medium">{{ t('needtoken.models.columns.supply') }}</th>
              <th class="py-2 pr-4 font-medium">{{ t('needtoken.models.columns.input') }}</th>
              <th class="py-2 pr-4 font-medium">{{ t('needtoken.models.columns.output') }}</th>
              <th class="py-2 font-medium"></th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="o in compared" :key="`c-${o.key}`" class="border-t border-border">
              <td class="nt-mono py-2 pr-4 text-foreground">{{ o.modelId }}</td>
              <td class="py-2 pr-4 text-muted-foreground">{{ supplyLabel(o) }}</td>
              <td class="nt-mono py-2 pr-4 text-foreground">{{ formatUsd(o.inputPrice) }}<span v-if="o.unit === 'per_call'" class="text-muted-foreground"> / {{ t('needtoken.models.perCallShort') }}</span></td>
              <td class="nt-mono py-2 pr-4 text-foreground">{{ formatUsd(o.outputPrice) }}</td>
              <td class="py-2 text-right"><button type="button" class="text-muted-foreground hover:text-foreground" :aria-label="t('needtoken.models.compareRemove')" @click="compare = toggleCompare(compare, o.key)">×</button></td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import { usePageSeo } from '../../composables/usePageSeo'
import { DEFAULT_MODELS } from './config'
import {
  filterOffers, formatUsd, MAX_COMPARE, sortOffers, supplyKeyOf, toggleCompare,
  type ModelOffer, type OfferSort,
} from './lib/pricing'
import { usePricing } from './lib/usePricing'
<<<<<<< HEAD
import { levelFor, type PoolLevel } from './lib/poolStatus'
import { usePoolStatus } from './lib/usePoolStatus'
=======
>>>>>>> origin/claude/project-thread-6r3cwt

const { t } = useI18n()
const route = useRoute()
const pricing = usePricing()
<<<<<<< HEAD
const pool = usePoolStatus()
=======
>>>>>>> origin/claude/project-thread-6r3cwt

usePageSeo({
  title: () => t('needtoken.models.title'),
  description: () => t('needtoken.models.subtitle'),
})

const keyword = ref(typeof route.query.q === 'string' ? route.query.q : '')
const vendor = ref('')
const supply = ref('')
const sort = ref<OfferSort>('default')
const compare = ref<string[]>([])

const offers = computed(() => pricing.offers.value)

const supplyLabel = (offer: ModelOffer) =>
  offer.supplyLine ? t(`needtoken.supply.${offer.supplyLine}`) : t('needtoken.models.groupLabel', { group: offer.group })

const vendorOptions = computed(() => [...new Set(offers.value.map(o => o.vendor))].sort())
const supplyOptions = computed(() => {
  const seen = new Map<string, string>()
  for (const o of offers.value) {
    const key = supplyKeyOf(o)
    if (!seen.has(key)) seen.set(key, supplyLabel(o))
  }
  return [...seen.entries()].map(([value, label]) => ({ value, label }))
})

const visible = computed(() =>
  sortOffers(filterOffers(offers.value, { vendor: vendor.value, supply: supply.value, keyword: keyword.value }), sort.value, DEFAULT_MODELS),
)

const compared = computed(() =>
  compare.value.map(key => offers.value.find(o => o.key === key)).filter((o): o is ModelOffer => Boolean(o)),
)

<<<<<<< HEAD
const levelClass: Record<PoolLevel, string> = {
  available: 'border-primary/60 text-primary',
  busy: 'border-amber-500/60 text-amber-500',
  unavailable: 'border-red-500/60 text-red-500',
  unknown: 'border-border text-muted-foreground',
}
const levelOf = (offer: ModelOffer) => levelFor(pool.view.value, offer.group)
const statusLabel = (offer: ModelOffer) => {
  const level = levelOf(offer)
  if (!level) return t('needtoken.models.statusPending')
  return pool.view.value?.stale ? t('needtoken.models.statusStale', { status: t(`needtoken.models.status.${level}`) }) : t(`needtoken.models.status.${level}`)
}
const statusClass = (offer: ModelOffer) => levelClass[levelOf(offer) ?? 'unknown']
const statusTitle = computed(() => {
  const at = pool.view.value?.updatedAt
  return at ? t('needtoken.models.statusUpdatedAt', { time: at.toLocaleTimeString() }) : undefined
})

onMounted(() => { void pricing.ensureLoaded(); void pool.ensureLoaded() })
=======
onMounted(() => { void pricing.ensureLoaded() })
>>>>>>> origin/claude/project-thread-6r3cwt
</script>
