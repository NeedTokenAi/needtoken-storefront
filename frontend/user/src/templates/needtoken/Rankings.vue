<template>
  <div class="mx-auto w-full max-w-[1200px] px-4 pb-12 pt-10 sm:px-6">
    <header class="flex flex-col justify-between gap-4 sm:flex-row sm:items-end">
      <div>
        <p class="text-[12px] font-semibold uppercase tracking-[0.14em] text-primary">{{ t('needtoken.rankings.eyebrow') }}</p>
        <h1 class="mt-2 text-[30px] font-bold tracking-tight text-foreground sm:text-[34px]">{{ t('needtoken.rankings.title') }}</h1>
        <p class="mt-2 max-w-[72ch] text-[14px] leading-relaxed text-muted-foreground">{{ t('needtoken.rankings.subtitle') }}</p>
      </div>
      <div class="flex flex-wrap items-center gap-2 text-[12px] text-muted-foreground">
        <span v-if="rankings.loadedAt.value" class="rounded-full border border-border bg-card px-3 py-1">
          {{ t('needtoken.rankings.updatedAt', { time: rankings.loadedAt.value.toLocaleTimeString() }) }}
        </span>
        <RouterLink
          to="/models"
          class="inline-flex items-center gap-1 rounded-full border border-border bg-card px-3 py-1 font-medium text-foreground transition-colors hover:border-primary/40 hover:text-primary"
        >
          <span>{{ t('needtoken.nav.models') }}</span>
          <ArrowUpRight class="h-3.5 w-3.5" aria-hidden="true" />
        </RouterLink>
      </div>
    </header>

    <!-- 周期切换栏 -->
    <section class="mt-6 flex flex-wrap items-center justify-between gap-3 rounded-2xl border border-border bg-card p-3.5">
      <div class="inline-flex flex-wrap gap-1.5" role="tablist" :aria-label="t('needtoken.rankings.periodLabel')">
        <button
          v-for="item in periodOptions"
          :key="item.value"
          type="button"
          role="tab"
          :aria-selected="rankings.period.value === item.value"
          class="inline-flex min-h-8 items-center rounded-full border px-3.5 py-1 text-[12.5px] font-medium transition-colors"
          :class="rankings.period.value === item.value
            ? 'border-primary bg-primary text-primary-foreground'
            : 'border-border bg-background/60 text-muted-foreground hover:text-foreground'"
          @click="selectPeriod(item.value)"
        >
          {{ t(item.labelKey) }}
        </button>
      </div>

      <button
        type="button"
        class="inline-flex min-h-8 items-center gap-1.5 rounded-full border border-border px-3 py-1 text-[12px] font-medium text-muted-foreground transition-colors hover:text-foreground disabled:opacity-60"
        :disabled="rankings.isRefreshing.value"
        @click="rankings.reload()"
      >
        <RefreshCw class="h-3.5 w-3.5" :class="rankings.isRefreshing.value ? 'animate-spin' : ''" aria-hidden="true" />
        <span>{{ t('needtoken.rankings.refresh') }}</span>
      </button>
    </section>

    <p
      v-if="(rankings.state.value === 'error' || rankings.isStale.value) && hasAnyData"
      role="status"
      class="mt-5 rounded-lg border border-amber-500/40 bg-amber-500/5 p-3 text-[12px] text-muted-foreground"
    >
      {{ t('needtoken.rankings.staleNotice') }}
    </p>

    <!-- 加载 / 错误 / 空态 -->
    <div v-if="rankings.isLoading.value" class="mt-8 rounded-2xl border border-border bg-card px-6 py-14 text-center text-muted-foreground">
      {{ t('needtoken.common.loading') }}
    </div>
    <div v-else-if="rankings.state.value === 'error' && !hasAnyData" class="mt-8 rounded-2xl border border-border bg-card px-6 py-14 text-center">
      <p class="text-muted-foreground">{{ t('needtoken.rankings.loadError') }}</p>
      <button type="button" class="mt-4 rounded-md border border-border px-4 py-2 text-[14px] text-foreground hover:bg-secondary" @click="rankings.reload()">
        {{ t('needtoken.common.retry') }}
      </button>
    </div>
    <div v-else-if="!hasAnyData" class="mt-8 rounded-2xl border border-border bg-card px-6 py-14 text-center text-muted-foreground">
      {{ t('needtoken.rankings.empty') }}
    </div>

    <template v-else>
      <!-- 核心指标卡 -->
      <section class="mt-6 grid grid-cols-2 gap-3 lg:grid-cols-4">
        <article class="rounded-2xl border border-border bg-card p-4">
          <p class="text-[11.5px] font-medium text-muted-foreground">{{ t('needtoken.rankings.summary.totalTokens') }}</p>
          <p class="nt-mono mt-1.5 text-[22px] font-bold text-foreground">{{ formatCompactTokens(totalTokens) }}</p>
        </article>
        <article class="rounded-2xl border border-border bg-card p-4">
          <p class="text-[11.5px] font-medium text-muted-foreground">{{ t('needtoken.rankings.summary.rankedModels') }}</p>
          <p class="nt-mono mt-1.5 text-[22px] font-bold text-foreground">{{ models.length }}</p>
        </article>
        <article class="rounded-2xl border border-border bg-card p-4">
          <p class="text-[11.5px] font-medium text-muted-foreground">{{ t('needtoken.rankings.summary.topModel') }}</p>
          <RouterLink
            v-if="models[0]"
            :to="modelSearchRoute(models[0].model_name)"
            class="nt-mono mt-1.5 block truncate text-[15px] font-bold text-primary hover:underline"
            :title="models[0].model_name"
          >
            {{ models[0].model_name }}
          </RouterLink>
          <p v-else class="mt-1.5 text-[15px] font-semibold text-muted-foreground">—</p>
        </article>
        <article class="rounded-2xl border border-border bg-card p-4">
          <p class="text-[11.5px] font-medium text-muted-foreground">{{ t('needtoken.rankings.summary.topVendor') }}</p>
          <p class="mt-1.5 truncate text-[16px] font-bold text-foreground">
            {{ vendors[0] ? `${vendors[0].vendor} (${formatSharePercent(vendors[0].share)})` : '—' }}
          </p>
        </article>
      </section>

      <!-- 模型消耗榜 Top 20 -->
      <section class="mt-6 rounded-2xl border border-border bg-card p-5">
        <div class="flex flex-wrap items-center justify-between gap-2">
          <div>
            <h2 class="flex items-center gap-2 text-[17px] font-bold text-foreground">
              <Trophy class="h-4.5 w-4.5 text-primary" aria-hidden="true" />
              <span>{{ t('needtoken.rankings.modelsSectionTitle') }}</span>
            </h2>
            <p class="mt-1 text-[12px] text-muted-foreground">{{ t('needtoken.rankings.modelsSectionHint') }}</p>
          </div>
          <span class="rounded-full border border-border bg-secondary/40 px-2.5 py-0.5 text-[11.5px] text-muted-foreground">
            Top {{ models.length }}
          </span>
        </div>

        <div class="mt-4 overflow-x-auto">
          <table class="w-full min-w-[720px] border-collapse text-left text-[13px]">
            <thead>
              <tr class="border-b border-border text-[11.5px] text-muted-foreground">
                <th class="py-2.5 pr-3 font-medium">{{ t('needtoken.rankings.columns.rank') }}</th>
                <th class="py-2.5 pr-4 font-medium">{{ t('needtoken.rankings.columns.model') }}</th>
                <th class="py-2.5 pr-4 font-medium">{{ t('needtoken.rankings.columns.price') }}</th>
                <th class="py-2.5 pr-4 font-medium">{{ t('needtoken.rankings.columns.tokens') }}</th>
                <th class="py-2.5 pr-4 font-medium">{{ t('needtoken.rankings.columns.share') }}</th>
                <th class="py-2.5 text-right font-medium">{{ t('needtoken.rankings.columns.growth') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="item in models"
                :key="item.model_name"
                class="border-b border-border/60 transition-colors last:border-b-0 hover:bg-secondary/35"
              >
                <td class="py-3 pr-3 align-middle">
                  <div class="flex items-center gap-2">
                    <span
                      class="nt-mono inline-flex h-6 w-6 items-center justify-center rounded-full text-[11.5px] font-bold"
                      :class="rankBadgeClass(item.rank)"
                    >
                      {{ item.rank }}
                    </span>
                    <span class="nt-mono text-[11px] font-semibold" :class="rankChangeClass(item)">
                      {{ rankChangeLabel(item) }}
                    </span>
                  </div>
                </td>
                <td class="py-3 pr-4 align-middle">
                  <RouterLink
                    :to="modelSearchRoute(item.model_name)"
                    class="group inline-flex max-w-[300px] items-center gap-1.5 font-semibold text-foreground hover:text-primary"
                  >
                    <span class="nt-mono truncate" :title="item.model_name">{{ item.model_name }}</span>
                    <ArrowUpRight class="h-3.5 w-3.5 shrink-0 opacity-45 transition-opacity group-hover:opacity-100" aria-hidden="true" />
                  </RouterLink>
                  <div class="mt-0.5 flex flex-wrap items-center gap-1.5 text-[11.5px] text-muted-foreground">
                    <span>{{ item.vendor }}</span>
                    <span v-if="item.category && item.category !== 'all'" class="rounded bg-secondary px-1.5 py-0.5 text-[10.5px]">
                      {{ item.category }}
                    </span>
                  </div>
                </td>
                <td class="nt-mono py-3 pr-4 align-middle text-[12px] text-muted-foreground">
                  <template v-if="priceHintOf(item.model_name)">
                    <span class="text-foreground">{{ priceHintOf(item.model_name) }}</span>
                  </template>
                  <span v-else>—</span>
                </td>
                <td class="nt-mono py-3 pr-4 align-middle font-semibold text-foreground">
                  {{ formatCompactTokens(item.total_tokens) }}
                </td>
                <td class="py-3 pr-4 align-middle">
                  <div class="flex min-w-[120px] items-center gap-2">
                    <div class="h-1.5 flex-1 overflow-hidden rounded-full bg-secondary">
                      <div
                        class="h-full rounded-full bg-primary transition-all"
                        :style="{ width: `${Math.max(2, Math.min(100, item.share * 100))}%` }"
                      />
                    </div>
                    <span class="nt-mono w-11 text-right text-[12px] text-muted-foreground">
                      {{ formatSharePercent(item.share) }}
                    </span>
                  </div>
                </td>
                <td class="nt-mono py-3 text-right align-middle text-[12.5px] font-semibold" :class="growthClass(item.growth_pct)">
                  {{ formatGrowthPercent(item.growth_pct) }}
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>

      <!-- 厂商占比榜 + 热门升降榜 -->
      <div class="mt-6 grid grid-cols-1 gap-6 lg:grid-cols-12">
        <!-- 厂商占比榜 -->
        <section class="rounded-2xl border border-border bg-card p-5 lg:col-span-7">
          <h2 class="flex items-center gap-2 text-[16px] font-bold text-foreground">
            <Layers class="h-4 w-4 text-primary" aria-hidden="true" />
            <span>{{ t('needtoken.rankings.vendorsSectionTitle') }}</span>
          </h2>
          <p class="mt-1 text-[12px] text-muted-foreground">{{ t('needtoken.rankings.vendorsSectionHint') }}</p>

          <div v-if="vendors.length === 0" class="mt-6 text-center text-[13px] text-muted-foreground">
            {{ t('needtoken.rankings.empty') }}
          </div>
          <div v-else class="mt-4 space-y-3.5">
            <article
              v-for="vendor in vendors"
              :key="vendor.vendor"
              class="rounded-xl border border-border/70 bg-secondary/20 p-3.5"
            >
              <div class="flex flex-wrap items-center justify-between gap-2">
                <div class="flex items-center gap-2">
                  <span class="nt-mono inline-flex h-5 w-5 items-center justify-center rounded-full bg-secondary text-[11px] font-bold text-foreground">
                    {{ vendor.rank }}
                  </span>
                  <span class="text-[14px] font-semibold text-foreground">{{ vendor.vendor }}</span>
                  <span class="rounded-full border border-border bg-card px-2 py-0.5 text-[11px] text-muted-foreground">
                    {{ t('needtoken.rankings.vendorModelsCount', { count: vendor.models_count }) }}
                  </span>
                </div>
                <div class="nt-mono flex items-center gap-3 text-[12.5px]">
                  <span class="font-semibold text-foreground">{{ formatSharePercent(vendor.share) }}</span>
                  <span class="text-muted-foreground">{{ formatCompactTokens(vendor.total_tokens) }}</span>
                  <span class="font-semibold" :class="growthClass(vendor.growth_pct)">{{ formatGrowthPercent(vendor.growth_pct) }}</span>
                </div>
              </div>
              <div class="mt-2.5 h-1.5 w-full overflow-hidden rounded-full bg-secondary">
                <div
                  class="h-full rounded-full bg-primary/85 transition-all"
                  :style="{ width: `${Math.max(2, Math.min(100, vendor.share * 100))}%` }"
                />
              </div>
              <div v-if="vendor.top_model" class="mt-2 flex items-center gap-1.5 text-[11.5px] text-muted-foreground">
                <span>{{ t('needtoken.rankings.topModelLabel') }}:</span>
                <RouterLink :to="modelSearchRoute(vendor.top_model)" class="nt-mono truncate font-medium text-primary hover:underline">
                  {{ vendor.top_model }}
                </RouterLink>
              </div>
            </article>
          </div>
        </section>

        <!-- 热门升降榜 -->
        <section class="flex flex-col gap-4 lg:col-span-5">
          <!-- 飙升榜 -->
          <div class="rounded-2xl border border-border bg-card p-5">
            <h2 class="flex items-center gap-2 text-[15.5px] font-bold text-foreground">
              <TrendingUp class="h-4 w-4 text-emerald-500" aria-hidden="true" />
              <span>{{ t('needtoken.rankings.moversTitle') }}</span>
            </h2>
            <p v-if="movers.length === 0" class="mt-4 text-[12.5px] text-muted-foreground">
              {{ t('needtoken.rankings.noMovers') }}
            </p>
            <ul v-else class="mt-3.5 space-y-2.5">
              <li
                v-for="mover in movers"
                :key="`up-${mover.model_name}`"
                class="flex items-center justify-between gap-2 rounded-xl border border-border/70 bg-secondary/20 px-3 py-2.5"
              >
                <div class="min-w-0 flex-1">
                  <RouterLink :to="modelSearchRoute(mover.model_name)" class="nt-mono block truncate text-[13px] font-semibold text-foreground hover:text-primary">
                    {{ mover.model_name }}
                  </RouterLink>
                  <p class="mt-0.5 text-[11px] text-muted-foreground">
                    {{ mover.vendor }} · #{{ mover.current_rank }}
                  </p>
                </div>
                <div class="nt-mono shrink-0 text-right">
                  <div class="text-[12px] font-bold text-emerald-600 dark:text-emerald-400">
                    ▲{{ Math.abs(mover.rank_delta) }}
                  </div>
                  <div class="text-[11px]" :class="growthClass(mover.growth_pct)">
                    {{ formatGrowthPercent(mover.growth_pct) }}
                  </div>
                </div>
              </li>
            </ul>
          </div>

          <!-- 回落榜 -->
          <div class="rounded-2xl border border-border bg-card p-5">
            <h2 class="flex items-center gap-2 text-[15.5px] font-bold text-foreground">
              <TrendingDown class="h-4 w-4 text-rose-500" aria-hidden="true" />
              <span>{{ t('needtoken.rankings.droppersTitle') }}</span>
            </h2>
            <p v-if="droppers.length === 0" class="mt-4 text-[12.5px] text-muted-foreground">
              {{ t('needtoken.rankings.noDroppers') }}
            </p>
            <ul v-else class="mt-3.5 space-y-2.5">
              <li
                v-for="dropper in droppers"
                :key="`down-${dropper.model_name}`"
                class="flex items-center justify-between gap-2 rounded-xl border border-border/70 bg-secondary/20 px-3 py-2.5"
              >
                <div class="min-w-0 flex-1">
                  <RouterLink :to="modelSearchRoute(dropper.model_name)" class="nt-mono block truncate text-[13px] font-semibold text-foreground hover:text-primary">
                    {{ dropper.model_name }}
                  </RouterLink>
                  <p class="mt-0.5 text-[11px] text-muted-foreground">
                    {{ dropper.vendor }} · #{{ dropper.current_rank }}
                  </p>
                </div>
                <div class="nt-mono shrink-0 text-right">
                  <div class="text-[12px] font-bold text-rose-600 dark:text-rose-400">
                    ▼{{ Math.abs(dropper.rank_delta) }}
                  </div>
                  <div class="text-[11px]" :class="growthClass(dropper.growth_pct)">
                    {{ formatGrowthPercent(dropper.growth_pct) }}
                  </div>
                </div>
              </li>
            </ul>
          </div>
        </section>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import { ArrowUpRight, Layers, RefreshCw, TrendingDown, TrendingUp, Trophy } from 'lucide-vue-next'
import { usePageSeo } from '../../composables/usePageSeo'
import { formatUsd } from './lib/pricing'
import { usePricing } from './lib/usePricing'
import {
  findBestOfferForModel,
  formatCompactTokens,
  formatGrowthPercent,
  formatSharePercent,
  normalizeRankingPeriod,
  resolveRankChange,
  type RankedModel,
  type RankingPeriod,
} from './lib/rankings'
import { useRankings } from './lib/useRankings'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const pricing = usePricing()
const rankings = useRankings(normalizeRankingPeriod(route.query.period))

usePageSeo({
  title: () => t('needtoken.rankings.title'),
  description: () => t('needtoken.rankings.subtitle'),
  canonicalPath: () => '/rankings',
})

const periodOptions: readonly { value: RankingPeriod; labelKey: string }[] = [
  { value: 'today', labelKey: 'needtoken.rankings.periods.today' },
  { value: 'week', labelKey: 'needtoken.rankings.periods.week' },
  { value: 'month', labelKey: 'needtoken.rankings.periods.month' },
  { value: 'year', labelKey: 'needtoken.rankings.periods.year' },
]

const models = computed(() => rankings.snapshot.value.models)
const vendors = computed(() => rankings.snapshot.value.vendors)
const movers = computed(() => rankings.snapshot.value.top_movers)
const droppers = computed(() => rankings.snapshot.value.top_droppers)
const hasAnyData = computed(() => models.value.length > 0 || vendors.value.length > 0)

const totalTokens = computed(() => {
  const fromVendors = vendors.value.reduce((sum, item) => sum + item.total_tokens, 0)
  if (fromVendors > 0) return fromVendors
  return models.value.reduce((sum, item) => sum + item.total_tokens, 0)
})

const modelSearchRoute = (modelName: string) => ({
  path: '/models',
  query: { search: modelName },
})

const selectPeriod = (next: RankingPeriod) => {
  void rankings.setPeriod(next)
  if (route.query.period !== next) {
    void router.replace({ query: { ...route.query, period: next } })
  }
}

watch(
  () => route.query.period,
  (raw) => {
    const normalized = normalizeRankingPeriod(raw)
    if (normalized !== rankings.period.value) {
      void rankings.setPeriod(normalized)
    }
  },
)

const rankBadgeClass = (rank: number) => {
  if (rank === 1) return 'bg-amber-500/20 text-amber-600 dark:text-amber-400'
  if (rank === 2) return 'bg-slate-400/20 text-slate-600 dark:text-slate-300'
  if (rank === 3) return 'bg-orange-500/20 text-orange-600 dark:text-orange-400'
  return 'bg-secondary text-muted-foreground'
}

const rankChangeLabel = (item: RankedModel) => {
  const change = resolveRankChange(item.rank, item.previous_rank)
  if (change.kind === 'new') return t('needtoken.rankings.rankNew')
  if (change.kind === 'up') return `▲${change.delta}`
  if (change.kind === 'down') return `▼${change.delta}`
  return '—'
}

const rankChangeClass = (item: RankedModel) => {
  const change = resolveRankChange(item.rank, item.previous_rank)
  if (change.kind === 'new') return 'text-primary'
  if (change.kind === 'up') return 'text-emerald-600 dark:text-emerald-400'
  if (change.kind === 'down') return 'text-rose-600 dark:text-rose-400'
  return 'text-muted-foreground/60'
}

const growthClass = (pct: number) => {
  if (!Number.isFinite(pct) || Math.abs(pct) < 0.05) return 'text-muted-foreground'
  return pct > 0 ? 'text-emerald-600 dark:text-emerald-400' : 'text-rose-600 dark:text-rose-400'
}

const priceHintOf = (modelName: string): string => {
  const offer = findBestOfferForModel(pricing.offers.value, modelName)
  if (!offer) return ''
  if (offer.unit === 'per_call') {
    return `${formatUsd(offer.inputPrice)} / ${t('needtoken.models.perCallShort')}`
  }
  if (offer.unit === 'per_million_tokens') {
    return `${formatUsd(offer.inputPrice)} / ${formatUsd(offer.outputPrice)}`
  }
  return t('needtoken.models.units.conditional')
}

onMounted(() => {
  void rankings.ensureLoaded()
  void pricing.ensureLoaded()
})
</script>
