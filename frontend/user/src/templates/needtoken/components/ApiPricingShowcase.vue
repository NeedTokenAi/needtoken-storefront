<template>
  <section id="api-pricing-showcase" class="pricing-showcase" aria-labelledby="api-pricing-showcase-title">
    <header class="showcase-head">
      <div>
        <span class="showcase-eyebrow">{{ t('needtoken.apiServices.pricing.eyebrow') }}</span>
        <h2 id="api-pricing-showcase-title">{{ t('needtoken.apiServices.pricing.title') }}</h2>
        <p>{{ t('needtoken.apiServices.pricing.subtitle') }}</p>
      </div>
      <div class="showcase-head-right">
        <RouterLink to="/models" class="view-all-btn">
          <span>{{ t('needtoken.apiServices.viewAllModels') }}</span>
          <ArrowRight :size="15" aria-hidden="true" />
        </RouterLink>
      </div>
    </header>

    <!-- 计价模式切换：按量实价 vs 活跃套餐倍数等效折后价 -->
    <div class="pricing-mode-bar">
      <div class="mode-pills" role="group" :aria-label="t('needtoken.apiServices.pricing.modeLabel')">
        <button
          type="button"
          :class="['mode-pill', { active: selectedMultiplier === 1 }]"
          @click="selectedMultiplier = 1"
        >
          <span>{{ t('needtoken.apiServices.pricing.paygMode') }}</span>
        </button>
        <button
          v-for="tier in planMultiplierOptions"
          :key="tier.key"
          type="button"
          :class="['mode-pill', { active: selectedMultiplier === tier.multiplier }]"
          @click="selectedMultiplier = tier.multiplier"
        >
          <Sparkles :size="13" aria-hidden="true" />
          <span>{{ tier.label }}</span>
        </button>
      </div>
      <span class="mode-hint">{{ pricingModeHint }}</span>
    </div>

    <!-- 精选 6 宫格模型折扣卡片（参考图 7） -->
    <div v-if="featuredCards.length > 0" class="featured-grid">
      <article v-for="card in featuredCards" :key="card.key" class="featured-card">
        <div class="featured-card-top">
          <div class="featured-model-meta">
            <span class="vendor-dot" :data-vendor="card.vendor.toLowerCase()" aria-hidden="true">{{ card.vendor.slice(0, 1) }}</span>
            <div class="min-w-0">
              <h3 class="featured-model-name" :title="card.modelId" translate="no">{{ card.modelId }}</h3>
              <p class="featured-model-sub">
                <span>{{ card.vendor }}</span>
                <template v-if="card.contextLength"> · {{ card.contextLength }} context</template>
              </p>
            </div>
          </div>
          <span v-if="card.displayDiscount !== null" class="discount-badge">{{ card.displayDiscount }}% off</span>
          <span v-else class="group-badge">{{ card.group }}</span>
        </div>

        <div class="featured-prices">
          <template v-if="card.unit === 'per_million_tokens'">
            <div class="price-box">
              <span class="price-label">INPUT / 1M TOKENS</span>
              <span v-if="card.officialInputPrice !== null && card.displayDiscount !== null" class="price-list">{{ formatUsd(card.officialInputPrice) }}</span>
              <strong class="price-actual">{{ formatUsd(card.displayInputPrice) }}</strong>
            </div>
            <div class="price-box">
              <span class="price-label">OUTPUT / 1M TOKENS</span>
              <span v-if="card.officialOutputPrice !== null && card.displayDiscount !== null" class="price-list">{{ formatUsd(card.officialOutputPrice) }}</span>
              <strong class="price-actual">{{ formatUsd(card.displayOutputPrice) }}</strong>
            </div>
          </template>
          <template v-else-if="card.unit === 'per_call'">
            <div class="price-box">
              <span class="price-label">PER CALL</span>
              <span v-if="card.officialInputPrice !== null && card.displayDiscount !== null" class="price-list">{{ formatUsd(card.officialInputPrice) }}</span>
              <strong class="price-actual">{{ formatUsd(card.displayInputPrice) }}</strong>
            </div>
          </template>
          <template v-else>
            <div class="price-box">
              <span class="price-label">PRICING</span>
              <strong class="price-actual text-sm">{{ t('needtoken.models.conditionalPrice') }}</strong>
            </div>
          </template>
        </div>

        <button type="button" class="quick-start-btn" @click="quickStartModel(card.modelId)">
          <ArrowUpRight :size="14" aria-hidden="true" />
          <span>{{ t('needtoken.apiServices.pricing.quickStart') }}</span>
        </button>
      </article>
    </div>

    <!-- 动态全模型比价表格（参考图 8） -->
    <div v-if="allOffers.length > 0" class="compare-block">
      <div class="compare-toolbar">
        <div class="vendor-tabs" role="tablist" :aria-label="t('needtoken.models.vendor')">
          <button
            type="button"
            role="tab"
            :aria-selected="activeVendor === '__featured__'"
            :class="['vendor-tab', { active: activeVendor === '__featured__' }]"
            @click="activeVendor = '__featured__'"
          >
            {{ t('needtoken.apiServices.pricing.featuredTab') }}
          </button>
          <button
            v-for="v in vendorList"
            :key="v"
            type="button"
            role="tab"
            :aria-selected="activeVendor === v"
            :class="['vendor-tab', { active: activeVendor === v }]"
            @click="activeVendor = v"
          >
            {{ v }}
          </button>
          <button
            type="button"
            role="tab"
            :aria-selected="activeVendor === '__all__'"
            :class="['vendor-tab', { active: activeVendor === '__all__' }]"
            @click="activeVendor = '__all__'"
          >
            {{ t('needtoken.models.allVendors') }}
          </button>
        </div>
        <div class="compare-search">
          <Search :size="14" aria-hidden="true" />
          <input
            v-model="keyword"
            type="search"
            :placeholder="t('needtoken.apiServices.pricing.searchPlaceholder')"
            :aria-label="t('needtoken.models.search')"
          />
        </div>
      </div>

      <div class="compare-table-wrap">
        <table class="compare-table">
          <thead>
            <tr>
              <th>Model</th>
              <th>Context</th>
              <th>{{ t('needtoken.apiServices.pricing.colOfficialInput') }}</th>
              <th>{{ t('needtoken.apiServices.pricing.colOfficialOutput') }}</th>
              <th class="highlight-col">{{ t('needtoken.apiServices.pricing.colNeedInput') }}</th>
              <th class="highlight-col">{{ t('needtoken.apiServices.pricing.colNeedOutput') }}</th>
              <th>{{ t('needtoken.apiServices.pricing.colProvider') }}</th>
              <th>Uptime (SLA)</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="row in tableRows" :key="row.key">
              <td>
                <div class="table-model-cell">
                  <div class="flex items-center gap-1.5">
                    <strong class="table-model-id" translate="no">{{ row.modelId }}</strong>
                    <button
                      type="button"
                      class="copy-mini"
                      :aria-label="t('needtoken.models.copyModelId', { model: row.modelId })"
                      @click="copyModel(row.modelId)"
                    >
                      <Check v-if="copiedId === row.modelId" :size="12" />
                      <Copy v-else :size="12" />
                    </button>
                  </div>
                  <span class="table-model-vendor">{{ row.vendor }}</span>
                </div>
              </td>
              <td class="mono-muted">{{ row.contextLength || '—' }}</td>
              <template v-if="row.unit === 'per_million_tokens'">
                <td class="list-col">
                  <div>{{ row.officialInputPrice !== null ? formatUsd(row.officialInputPrice) : '—' }}</div>
                  <small v-if="row.officialCachePrice !== null">Cache {{ formatUsd(row.officialCachePrice) }}</small>
                </td>
                <td class="list-col">
                  <div>{{ row.officialOutputPrice !== null ? formatUsd(row.officialOutputPrice) : '—' }}</div>
                </td>
                <td class="highlight-col actual-col">
                  <div class="actual-price-line">
                    <strong>{{ formatUsd(row.displayInputPrice) }}</strong>
                    <span v-if="row.displayDiscount !== null" class="discount-mini">{{ row.displayDiscount }}% off</span>
                  </div>
                  <small v-if="row.displayCachePrice !== null">Cache {{ formatUsd(row.displayCachePrice) }}</small>
                </td>
                <td class="highlight-col actual-col">
                  <div class="actual-price-line">
                    <strong>{{ formatUsd(row.displayOutputPrice) }}</strong>
                    <span v-if="row.displayDiscount !== null" class="discount-mini">{{ row.displayDiscount }}% off</span>
                  </div>
                </td>
              </template>
              <template v-else-if="row.unit === 'per_call'">
                <td colspan="2" class="list-col">{{ row.officialInputPrice !== null ? `${formatUsd(row.officialInputPrice)} / call` : '—' }}</td>
                <td colspan="2" class="highlight-col actual-col">
                  <div class="actual-price-line">
                    <strong>{{ formatUsd(row.displayInputPrice) }}</strong>
                    <small>/ {{ t('needtoken.models.perCallShort') }}</small>
                    <span v-if="row.displayDiscount !== null" class="discount-mini">{{ row.displayDiscount }}% off</span>
                  </div>
                </td>
              </template>
              <template v-else>
                <td colspan="4" class="mono-muted">{{ t('needtoken.models.conditionalPrice') }}</td>
              </template>
              <td>
                <span class="provider-chip">{{ row.group }}</span>
              </td>
              <td>
                <div v-if="row.uptimePct !== null" class="sla-cell">
                  <span class="sla-pct" :class="row.uptimePct >= 99 ? 'is-good' : row.uptimePct >= 90 ? 'is-warn' : 'is-bad'">
                    {{ row.uptimePct.toFixed(2) }}%
                  </span>
                  <div class="sla-bars" aria-hidden="true">
                    <i
                      v-for="(slot, idx) in row.slaSlots"
                      :key="idx"
                      :class="`is-${slot}`"
                    />
                  </div>
                </div>
                <span v-else class="mono-muted text-xs">{{ t('needtoken.apiServices.pricing.slaMonitoring') }}</span>
              </td>
            </tr>
            <tr v-if="!tableRows.length">
              <td colspan="8" class="empty-row">{{ t('needtoken.models.empty') }}</td>
            </tr>
          </tbody>
        </table>
      </div>

      <div v-if="hasMoreRows" class="compare-footer">
        <RouterLink to="/models" class="compare-more-link">
          <span>{{ t('needtoken.apiServices.pricing.viewFullCatalog', { count: allOffers.length }) }}</span>
          <ArrowRight :size="14" aria-hidden="true" />
        </RouterLink>
      </div>
    </div>

    <div v-else-if="pricing.isLoading.value" class="pricing-loading">
      {{ t('needtoken.common.loading') }}
    </div>
    <div v-else class="pricing-fallback">
      <p>{{ t('needtoken.apiServices.hotModelsFallbackHint') }}</p>
      <RouterLink to="/models" class="view-all-btn">{{ t('needtoken.apiServices.viewAllModels') }} <ArrowRight :size="14" /></RouterLink>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { ArrowRight, ArrowUpRight, Check, Copy, Search, Sparkles } from 'lucide-vue-next'
import {
  effectiveDiscountPercent,
  effectivePlanOfferPrice,
  formatUsd,
  selectOffersByGroup,
  sortOffers,
  type ModelOffer,
} from '../lib/pricing'
import { usePricing } from '../lib/usePricing'
import { historyFor, uptimeFor, type PoolLevel } from '../lib/poolStatus'
import { usePoolStatus } from '../lib/usePoolStatus'

const emit = defineEmits<{
  'quick-start': [modelId: string]
}>()

const { t } = useI18n()
const pricing = usePricing()
const pool = usePoolStatus()

const selectedMultiplier = ref<number>(1)
const activeVendor = ref<string>('__featured__')
const keyword = ref('')
const copiedId = ref('')
let copyTimer: ReturnType<typeof setTimeout> | undefined

interface DisplayOffer extends ModelOffer {
  displayInputPrice: number | null
  displayOutputPrice: number | null
  displayCachePrice: number | null
  displayDiscount: number | null
  uptimePct: number | null
  slaSlots: PoolLevel[]
}

const planMultiplierOptions = computed(() => {
  const seen = new Set<number>()
  const opts: Array<{ key: string; multiplier: number; label: string }> = []
  for (const plan of pricing.activePlans.value) {
    const mult = Number(plan.multiplier)
    if (!Number.isFinite(mult) || mult <= 1.05) continue
    const rounded = Math.round(mult * 10) / 10
    if (seen.has(rounded)) continue
    seen.add(rounded)
    opts.push({
      key: `plan-${rounded}`,
      multiplier: rounded,
      label: t('needtoken.apiServices.pricing.planMultiplierOption', { mult: `${rounded}x` }),
    })
  }
  if (!opts.length) {
    // Fallback preview tiers when active plans have not yet loaded from glue
    opts.push(
      { key: 'default-2x', multiplier: 2, label: t('needtoken.apiServices.pricing.planMultiplierOption', { mult: '2x' }) },
      { key: 'default-3x', multiplier: 3, label: t('needtoken.apiServices.pricing.planMultiplierOption', { mult: '3x' }) },
    )
  }
  return opts.sort((a, b) => a.multiplier - b.multiplier).slice(0, 3)
})

const pricingModeHint = computed(() => {
  if (selectedMultiplier.value <= 1) {
    return t('needtoken.apiServices.pricing.paygHint')
  }
  return t('needtoken.apiServices.pricing.planHint', { mult: `${selectedMultiplier.value}x` })
})

const allOffers = computed<DisplayOffer[]>(() => {
  const deduped = selectOffersByGroup(pricing.offers.value, '')
  const sorted = sortOffers(deduped, 'default', pricing.featuredModels.value)
  const mult = selectedMultiplier.value
  return sorted.map((offer) => {
    const displayInputPrice = effectivePlanOfferPrice(offer.inputPrice, mult)
    const displayOutputPrice = effectivePlanOfferPrice(offer.outputPrice, mult)
    const displayCachePrice = effectivePlanOfferPrice(offer.cachePrice, mult)
    const baselineOfficial = offer.officialInputPrice ?? (mult > 1 ? offer.inputPrice : null)
    const displayDiscount = effectiveDiscountPercent(baselineOfficial, displayInputPrice)
    const uptimePct = uptimeFor(pool.view.value, offer.group)
    const slaSlots = historyFor(pool.view.value, offer.group).slice(-24)
    return {
      ...offer,
      displayInputPrice,
      displayOutputPrice,
      displayCachePrice,
      displayDiscount,
      uptimePct,
      slaSlots,
    }
  })
})

const featuredCards = computed(() => {
  const featuredNames = pricing.featuredModels.value
  const picked: DisplayOffer[] = []
  const byId = new Map(allOffers.value.map(o => [o.modelId, o]))
  for (const name of featuredNames) {
    const found = byId.get(name)
    if (found) picked.push(found)
    if (picked.length >= 6) break
  }
  if (picked.length < 6) {
    const used = new Set(picked.map(p => p.modelId))
    for (const offer of allOffers.value) {
      if (used.has(offer.modelId)) continue
      picked.push(offer)
      if (picked.length >= 6) break
    }
  }
  return picked
})

const vendorList = computed(() => {
  const counts = new Map<string, number>()
  for (const o of allOffers.value) {
    counts.set(o.vendor, (counts.get(o.vendor) || 0) + 1)
  }
  return [...counts.entries()]
    .sort((a, b) => b[1] - a[1] || a[0].localeCompare(b[0]))
    .map(([v]) => v)
    .slice(0, 7)
})

const filteredTableOffers = computed(() => {
  const q = keyword.value.trim().toLowerCase()
  let list = allOffers.value
  if (activeVendor.value === '__featured__' && !q) {
    const featuredSet = new Set(pricing.featuredModels.value)
    const matched = list.filter(o => featuredSet.has(o.modelId))
    list = matched.length >= 4 ? matched : list.slice(0, 12)
  } else if (activeVendor.value !== '__all__' && activeVendor.value !== '__featured__') {
    list = list.filter(o => o.vendor === activeVendor.value)
  }
  if (q) {
    list = list.filter(o => o.modelId.toLowerCase().includes(q) || o.vendor.toLowerCase().includes(q))
  }
  return list
})

const tableRows = computed(() => filteredTableOffers.value.slice(0, 12))
const hasMoreRows = computed(() => filteredTableOffers.value.length > 12 || allOffers.value.length > 12)

const copyModel = async (modelId: string) => {
  if (copyTimer) clearTimeout(copyTimer)
  try {
    await navigator.clipboard.writeText(modelId)
    copiedId.value = modelId
    copyTimer = setTimeout(() => { copiedId.value = '' }, 1500)
  } catch {
    copiedId.value = ''
  }
}

const quickStartModel = (modelId: string) => {
  emit('quick-start', modelId)
  const target = document.getElementById('api-connect')
  if (target) {
    target.scrollIntoView({ behavior: 'smooth', block: 'start' })
  }
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
.pricing-showcase { padding-block: 48px; border-top: 1px solid var(--ui-border); }
.showcase-head { display: flex; align-items: flex-end; justify-content: space-between; flex-wrap: wrap; gap: 16px; }
.showcase-eyebrow { font-size: 11.5px; font-weight: 700; letter-spacing: .12em; text-transform: uppercase; color: var(--ui-accent); }
.showcase-head h2 { margin-top: 8px; font-size: clamp(24px, 2.8vw, 32px); font-weight: 700; letter-spacing: -.025em; color: var(--ui-text-primary); }
.showcase-head p { margin-top: 8px; max-width: 68ch; font-size: 14px; line-height: 1.75; color: var(--ui-text-secondary); }
.view-all-btn { display: inline-flex; align-items: center; gap: 6px; min-height: 38px; padding: 0 14px; border: 1px solid var(--ui-border-strong); border-radius: 10px; font-size: 13px; font-weight: 600; color: var(--ui-accent); background: var(--ui-bg-elevated); }
.view-all-btn:hover { border-color: var(--ui-accent); background: var(--ui-bg-soft); }

.pricing-mode-bar { display: flex; align-items: center; justify-content: space-between; flex-wrap: wrap; gap: 12px; margin-top: 22px; }
.mode-pills { display: inline-flex; flex-wrap: wrap; gap: 6px; padding: 4px; border: 1px solid var(--ui-border); border-radius: 12px; background: var(--ui-bg-soft); }
.mode-pill { display: inline-flex; align-items: center; gap: 6px; min-height: 34px; padding: 0 13px; border-radius: 9px; font-size: 12.5px; font-weight: 600; color: var(--ui-text-secondary); cursor: pointer; transition: all 150ms; }
.mode-pill:hover { color: var(--ui-text-primary); }
.mode-pill.active { background: var(--ui-bg-elevated); color: var(--ui-accent); box-shadow: 0 1px 3px rgba(15, 23, 42, 0.08); }
.mode-hint { font-size: 12.5px; color: var(--ui-text-muted); }

/* 精选 6 宫格卡片（参考截图 7） */
.featured-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  margin-top: 20px;
  border: 1px dashed var(--ui-border-strong);
  border-radius: 18px;
  overflow: hidden;
  background: var(--ui-bg-elevated);
}
.featured-card {
  display: flex;
  flex-direction: column;
  gap: 18px;
  padding: 22px;
  border-right: 1px dashed var(--ui-border);
  border-bottom: 1px dashed var(--ui-border);
  transition: background 150ms;
}
.featured-card:nth-child(3n) { border-right: 0; }
.featured-card:nth-last-child(-n+3) { border-bottom: 0; }
.featured-card:hover { background: var(--ui-bg-soft); }
.featured-card-top { display: flex; align-items: flex-start; justify-content: space-between; gap: 10px; }
.featured-model-meta { display: flex; align-items: center; gap: 10px; min-width: 0; }
.vendor-dot {
  display: grid;
  place-items: center;
  flex: none;
  width: 30px;
  height: 30px;
  border-radius: 8px;
  background: var(--ui-bg-soft);
  border: 1px solid var(--ui-border);
  font: 700 12px var(--nt-font-mono);
  color: var(--ui-accent);
}
.featured-model-name { font-size: 15px; font-weight: 700; color: var(--ui-text-primary); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.featured-model-sub { margin-top: 2px; font-size: 12px; color: var(--ui-text-muted); }
.discount-badge {
  flex: none;
  padding: 3px 8px;
  border-radius: 6px;
  background: #18181b;
  color: #fff;
  font: 700 11.5px var(--nt-font-mono);
  white-space: nowrap;
}
.dark .discount-badge { background: #10b981; color: #052e16; }
.group-badge {
  flex: none;
  padding: 3px 8px;
  border-radius: 6px;
  background: var(--ui-bg-soft);
  border: 1px solid var(--ui-border);
  color: var(--ui-text-secondary);
  font: 600 11px var(--nt-font-mono);
}
.featured-prices { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 14px; padding-top: 12px; border-top: 1px dashed var(--ui-border); }
.price-box { display: flex; flex-direction: column; gap: 4px; }
.price-label { font: 600 10.5px var(--nt-font-mono); letter-spacing: .04em; color: var(--ui-text-muted); }
.price-list { font: 500 12.5px var(--nt-font-mono); color: var(--ui-text-muted); text-decoration: line-through; }
.price-actual { font: 750 22px/1.15 var(--nt-font-mono); color: var(--ui-text-primary); letter-spacing: -.03em; }
.quick-start-btn {
  margin-top: auto;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  height: 36px;
  border: 1px dashed var(--ui-border-strong);
  border-radius: 9px;
  background: transparent;
  font-size: 12.5px;
  font-weight: 600;
  color: var(--ui-text-secondary);
  cursor: pointer;
  transition: all 150ms;
}
.quick-start-btn:hover { border-style: solid; border-color: var(--ui-accent); color: var(--ui-accent); background: var(--ui-bg-elevated); }

/* 全模型对比表（参考截图 8） */
.compare-block { margin-top: 28px; }
.compare-toolbar { display: flex; align-items: center; justify-content: space-between; flex-wrap: wrap; gap: 12px; }
.vendor-tabs { display: flex; flex-wrap: wrap; gap: 6px; }
.vendor-tab {
  min-height: 34px;
  padding: 0 12px;
  border: 1px solid var(--ui-border);
  border-radius: 8px;
  background: var(--ui-bg-elevated);
  color: var(--ui-text-secondary);
  font-size: 12.5px;
  font-weight: 600;
  cursor: pointer;
  transition: all 150ms;
}
.vendor-tab:hover { color: var(--ui-text-primary); border-color: var(--ui-border-strong); }
.vendor-tab.active { background: #18181b; border-color: #18181b; color: #fff; }
.dark .vendor-tab.active { background: var(--ui-accent); border-color: var(--ui-accent); color: var(--ui-text-on-accent); }
.compare-search {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  height: 34px;
  padding: 0 11px;
  border: 1px solid var(--ui-border);
  border-radius: 8px;
  background: var(--ui-bg-elevated);
  color: var(--ui-text-muted);
  min-width: 220px;
}
.compare-search input { border: 0; background: transparent; color: var(--ui-text-primary); font-size: 12.5px; width: 100%; outline: none; }

.compare-table-wrap { margin-top: 14px; overflow-x: auto; border: 1px solid var(--ui-border); border-radius: 16px; background: var(--ui-bg-elevated); }
.compare-table { width: 100%; min-width: 820px; border-collapse: collapse; text-align: left; font-size: 13px; }
.compare-table th { padding: 13px 14px; border-bottom: 1px solid var(--ui-border); font-size: 12px; font-weight: 600; color: var(--ui-text-muted); white-space: nowrap; }
.compare-table td { padding: 14px; border-bottom: 1px dashed var(--ui-border); vertical-align: middle; }
.compare-table tbody tr:last-child td { border-bottom: 0; }
.highlight-col {
  background: linear-gradient(180deg, rgba(16, 185, 129, 0.08), rgba(249, 115, 22, 0.05));
}
.dark .highlight-col {
  background: linear-gradient(180deg, rgba(16, 185, 129, 0.12), rgba(249, 115, 22, 0.08));
}
.compare-table th.highlight-col { color: var(--ui-text-primary); font-weight: 700; }
.table-model-cell { display: flex; flex-direction: column; gap: 2px; }
.table-model-id { font: 650 13.5px var(--nt-font-mono); color: var(--ui-text-primary); }
.table-model-vendor { font-size: 11.5px; color: var(--ui-text-muted); }
.copy-mini { display: inline-grid; place-items: center; width: 22px; height: 22px; border-radius: 5px; color: var(--ui-text-muted); }
.copy-mini:hover { color: var(--ui-accent); background: var(--ui-bg-soft); }
.mono-muted { font: 500 12.5px var(--nt-font-mono); color: var(--ui-text-muted); }
.list-col { font-family: var(--nt-font-mono); color: var(--ui-text-muted); white-space: nowrap; }
.list-col small { display: block; margin-top: 2px; font-size: 11px; opacity: .85; }
.actual-col { font-family: var(--nt-font-mono); white-space: nowrap; }
.actual-price-line { display: inline-flex; align-items: center; gap: 6px; }
.actual-col strong { font-size: 14px; font-weight: 750; color: var(--ui-text-primary); }
.actual-col small { display: block; margin-top: 2px; font-size: 11px; color: var(--ui-accent); }
.discount-mini { padding: 2px 6px; border-radius: 5px; background: #18181b; color: #fff; font: 700 10.5px var(--nt-font-mono); }
.dark .discount-mini { background: #10b981; color: #052e16; }
.provider-chip { display: inline-flex; padding: 3px 8px; border-radius: 999px; border: 1px solid var(--ui-border); background: var(--ui-bg-soft); font: 600 11px var(--nt-font-mono); color: var(--ui-text-secondary); }

.sla-cell { display: flex; flex-direction: column; gap: 5px; min-width: 116px; }
.sla-pct { font: 700 12.5px var(--nt-font-mono); }
.sla-pct.is-good { color: #059669; }
.dark .sla-pct.is-good { color: #34d399; }
.sla-pct.is-warn { color: #d97706; }
.sla-pct.is-bad { color: #dc2626; }
.sla-bars { display: flex; align-items: center; gap: 2px; height: 12px; }
.sla-bars i { display: block; width: 3px; height: 100%; border-radius: 2px; background: var(--ui-border-strong); }
.sla-bars i.is-available { background: #10b981; }
.sla-bars i.is-busy { background: #f59e0b; }
.sla-bars i.is-unavailable { background: #ef4444; }
.empty-row { text-align: center; padding: 32px !important; color: var(--ui-text-muted); }

.compare-footer { display: flex; justify-content: center; margin-top: 16px; }
.compare-more-link { display: inline-flex; align-items: center; gap: 6px; padding: 8px 16px; border: 1px solid var(--ui-border-strong); border-radius: 10px; font-size: 13px; font-weight: 600; color: var(--ui-accent); background: var(--ui-bg-elevated); }
.compare-more-link:hover { border-color: var(--ui-accent); background: var(--ui-bg-soft); }
.pricing-loading, .pricing-fallback { margin-top: 20px; padding: 28px; border: 1px solid var(--ui-border); border-radius: 16px; background: var(--ui-bg-elevated); text-align: center; color: var(--ui-text-muted); display: flex; flex-direction: column; align-items: center; gap: 12px; }

@media (max-width: 960px) {
  .featured-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .featured-card:nth-child(3n) { border-right: 1px dashed var(--ui-border); }
  .featured-card:nth-child(2n) { border-right: 0; }
}
@media (max-width: 640px) {
  .featured-grid { grid-template-columns: minmax(0, 1fr); }
  .featured-card { border-right: 0 !important; }
  .compare-search { width: 100%; }
}
</style>
