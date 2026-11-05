<template>
  <section ref="calculatorEl" class="nt-api-cost" aria-labelledby="api-cost-title">
    <header class="cost-heading"><span>{{ t('needtoken.apiCost.eyebrow') }}</span><h2 id="api-cost-title">{{ t('needtoken.apiCost.title') }}</h2><p>{{ t('needtoken.apiCost.intro') }}</p></header>
    <div class="cost-modes" role="group" :aria-label="t('needtoken.apiCost.priceSource')"><button v-for="source in sources" :key="source" type="button" :aria-pressed="mode === source" @click="mode = source">{{ t(`needtoken.apiCost.mode.${source}`) }}</button></div>

    <div class="cost-grid">
      <div class="cost-inputs">
        <section v-if="mode === 'public'" class="cost-source" :aria-label="t('needtoken.apiCost.priceSource')">
          <div class="cost-source-head"><h3>{{ t('needtoken.apiCost.selectModel') }}</h3><button type="button" :disabled="pricing.isLoading.value" @click="pricing.reload()"><RotateCw :size="13" aria-hidden="true" />{{ t('needtoken.common.retry') }}</button></div>
          <p v-if="pricing.isLoading.value" role="status">{{ t('needtoken.apiCost.loading') }}</p>
          <p v-else-if="!liveReady" class="cost-status" role="status">{{ t(pricing.state.value === 'error' ? 'needtoken.apiCost.sourceUnavailable' : 'needtoken.apiCost.sourceStale') }}</p>
          <p v-else-if="!rates.length" class="cost-status" role="status">{{ t('needtoken.apiCost.noScalarRates') }}</p>
          <div class="cost-model-controls"><input v-model="modelSearch" type="search" :placeholder="t('needtoken.apiCost.searchModel')" :aria-label="t('needtoken.apiCost.searchModel')" /><NeedtokenSelect v-model="selectedKey" :options="modelOptions" :label="t('needtoken.apiCost.selectModel')" :disabled="!liveReady || !rates.length" align="start" /></div>
          <p v-if="filteredRates.length > 100" class="cost-small">{{ t('needtoken.apiCost.searchHint') }}</p>
          <p v-if="selectedRate" class="cost-small">{{ t('needtoken.apiCost.groupFactor', { group: selectedRate.group, factor: selectedRate.groupRatio }) }}</p>
        </section>
        <section v-else class="cost-source">
          <h3>{{ t('needtoken.apiCost.manualTitle') }}</h3><p>{{ t('needtoken.apiCost.manualHint') }}</p>
          <NeedtokenSelect v-model="manualUnit" :options="unitOptions" :label="t('needtoken.apiCost.billingUnit')" align="start" />
          <div class="cost-fields"><label>{{ t(manualUnit === 'per_call' ? 'needtoken.apiCost.manualCall' : 'needtoken.apiCost.manualInput') }}<input v-model="manualInput" type="number" min="0" step="any" inputmode="decimal" placeholder="—" /></label><label v-if="manualUnit === 'per_million_tokens'">{{ t('needtoken.apiCost.manualOutput') }}<input v-model="manualOutput" type="number" min="0" step="any" inputmode="decimal" placeholder="—" /></label></div>
        </section>

        <section class="cost-usage"><h3>{{ t('needtoken.apiCost.usageTitle') }}</h3><div class="cost-fields" :class="{ 'cost-fields-three': !perCall }">
          <label>{{ t('needtoken.apiCost.calls') }}<input v-model="calls" type="number" min="0" max="1000000000" step="1" inputmode="numeric" /></label>
          <template v-if="!perCall"><label>{{ t('needtoken.apiCost.inputTokens') }}<input v-model="inputTokens" type="number" min="0" max="1000000000" step="1" inputmode="numeric" /></label><label>{{ t('needtoken.apiCost.outputTokens') }}<input v-model="outputTokens" type="number" min="0" max="1000000000" step="1" inputmode="numeric" /></label></template>
        </div><p v-if="!perCall" class="cost-small">{{ t('needtoken.apiCost.inputHint') }}</p></section>

        <details v-if="!perCall" class="cost-cache"><summary>{{ t('needtoken.apiCost.cacheTitle') }}</summary><p class="cost-small">{{ t('needtoken.apiCost.cacheHint') }}</p><div class="cost-fields"><label>{{ t('needtoken.apiCost.cacheReadTokens') }}<input v-model="cacheReadTokens" type="number" min="0" max="1000000000" step="1" inputmode="numeric" /></label><label>{{ t('needtoken.apiCost.cacheWriteTokens') }}<input v-model="cacheWriteTokens" type="number" min="0" max="1000000000" step="1" inputmode="numeric" /></label></div><div v-if="mode === 'manual'" class="cost-fields"><label>{{ t('needtoken.apiCost.manualCacheRead') }}<input v-model="manualCacheRead" type="number" min="0" step="any" inputmode="decimal" placeholder="—" /></label><label>{{ t('needtoken.apiCost.manualCacheWrite') }}<input v-model="manualCacheWrite" type="number" min="0" step="any" inputmode="decimal" placeholder="—" /></label></div></details>
      </div>

      <aside class="cost-result" aria-live="polite" aria-atomic="true">
        <div class="cost-result-head"><span>{{ t('needtoken.apiCost.resultTitle') }}</span><NeedtokenSelect :model-value="currency" :options="currencyOptions" :label="t('needtoken.apiCost.resultCurrency')" compact @update:model-value="selectCurrency" /></div>
        <span class="cost-source-badge">{{ t(`needtoken.apiCost.mode.${mode}`) }}</span>
        <strong class="cost-total">{{ convertedEstimate === null ? '—' : formatAmount(convertedEstimate) }}</strong><p class="cost-small">{{ t('needtoken.apiCost.perMonth') }}</p>
        <p v-if="!currentRate" class="cost-status">{{ t(mode === 'manual' ? 'needtoken.apiCost.enterManualRates' : 'needtoken.apiCost.choosePublicRate') }}</p><p v-else-if="!estimate" class="cost-status">{{ t('needtoken.apiCost.invalidUsage') }}</p>
        <dl v-if="currentRate" class="cost-rate-list"><div><dt>{{ t(perCall ? 'needtoken.apiCost.rateCall' : 'needtoken.apiCost.rateInput') }}</dt><dd>{{ formatUsd(currentRate.inputUsd) }}</dd></div><template v-if="!perCall"><div><dt>{{ t('needtoken.apiCost.rateOutput') }}</dt><dd>{{ formatUsd(currentRate.outputUsd) }}</dd></div><div><dt>{{ t('needtoken.apiCost.rateCacheRead') }}</dt><dd>{{ formatUsd(currentRate.cacheReadUsd) }}</dd></div><div><dt>{{ t('needtoken.apiCost.rateCacheWrite') }}</dt><dd>{{ formatUsd(currentRate.cacheWriteUsd) }}</dd></div></template></dl>
        <p v-if="estimate && !perCall" class="cost-small">{{ t('needtoken.apiCost.volume', { input: formatCount(estimate.monthlyInput), output: formatCount(estimate.monthlyOutput) }) }}</p>

        <div class="cost-comparison"><h3>{{ t('needtoken.apiCost.comparisonTitle') }}</h3><label>{{ t('needtoken.apiCost.comparisonAmount', { currency }) }}<input v-model="comparisonAmount" type="number" min="0" step="0.01" inputmode="decimal" :placeholder="t('needtoken.apiCost.comparisonPlaceholder')" /></label><p v-if="comparison" class="cost-difference">{{ t(`needtoken.apiCost.comparison.${comparison.kind}`, { amount: formatAmount(comparison.difference) }) }}</p><p class="cost-small">{{ t('needtoken.apiCost.comparisonHint') }}</p></div>
      </aside>
    </div>

    <footer class="cost-notes"><p><Calculator :size="14" aria-hidden="true" />{{ t(perCall ? 'needtoken.apiCost.formulaCall' : 'needtoken.apiCost.formulaTokens') }}</p><p>{{ t('needtoken.apiCost.scopeNote') }}</p><p v-if="currency === 'CNY'">{{ t('needtoken.apiCost.fx', { rate: fx.rate.value, time: fx.rateUpdatedAt.value ? formatDate(fx.rateUpdatedAt.value) : t('needtoken.apiCost.timeUnknown') }) }}</p><p v-else-if="!fx.rate.value">{{ t('needtoken.apiCost.fxMissing') }}</p><p v-if="mode === 'public'"><a :href="PRICING_URL" target="_blank" rel="noopener noreferrer">{{ t('needtoken.apiCost.sourceLink') }}</a><span v-if="pricing.loadedAt.value"> · {{ t('needtoken.apiCost.loadedAt', { time: formatDate(pricing.loadedAt.value) }) }}</span></p><p v-else>{{ t('needtoken.apiCost.manualSource') }}</p></footer>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { Calculator, RotateCw } from 'lucide-vue-next'
import { useAppStore } from '../../../stores/app'
import { useDisplayCurrency } from '../composables/useDisplayCurrency'
import { BLOCKED_MODELS, DEFAULT_MODELS, PRICING_URL, SUPPLY_LINE_BY_GROUP } from '../config'
import { buildApiCostRates, compareApiCost, convertApiCost, estimateApiCost, parseApiRate, type ApiCostRate } from '../lib/apiCost'
import { formatServiceMoney } from '../lib/displayMoney'
import { formatUsd } from '../lib/pricing'
import { usePricing } from '../lib/usePricing'
import NeedtokenSelect from './NeedtokenSelect.vue'

const { t } = useI18n(), app = useAppStore(), pricing = usePricing(), fx = useDisplayCurrency()
const sources = ['public', 'manual'] as const
const mode = ref<'public' | 'manual'>('public')
const selectedKey = ref(''), modelSearch = ref('')
const manualUnit = ref('per_million_tokens')
const manualInput = ref(''), manualOutput = ref(''), manualCacheRead = ref(''), manualCacheWrite = ref('')
const calls = ref('1000'), inputTokens = ref('1500'), outputTokens = ref('500'), cacheReadTokens = ref('0'), cacheWriteTokens = ref('0')
const comparisonAmount = ref(''), currencyPreference = ref<'CNY' | 'USD'>('CNY'), clock = ref(Date.now())
const calculatorEl = ref<HTMLElement | null>(null)
let freshnessTimer: ReturnType<typeof setTimeout> | undefined
let visibilityObserver: IntersectionObserver | undefined
const liveReady = computed(() => pricing.state.value === 'ready' && !!pricing.loadedAt.value && Math.max(0, clock.value - pricing.loadedAt.value.getTime()) < 300_000)
const rates = computed(() => buildApiCostRates(pricing.response.value || {}, { defaultModels: DEFAULT_MODELS, blockedModels: BLOCKED_MODELS, supplyLineByGroup: SUPPLY_LINE_BY_GROUP }))
const filteredRates = computed(() => rates.value.filter(rate => `${rate.modelId} ${rate.group}`.toLowerCase().includes(modelSearch.value.trim().toLowerCase())))
const selectedRate = computed(() => rates.value.find(rate => rate.key === selectedKey.value) || null)
const modelOptions = computed(() => {
  const rows = filteredRates.value.slice(0, 100)
  if (selectedRate.value && !rows.some(rate => rate.key === selectedKey.value)) rows.unshift(selectedRate.value)
  return [{ value: '', label: t('needtoken.apiCost.selectModel') }, ...rows.map(rate => ({ value: rate.key, label: `${rate.modelId} · ${rate.group}` }))]
})
const unitOptions = computed(() => [{ value: 'per_million_tokens', label: t('needtoken.apiCost.tokenUnit') }, { value: 'per_call', label: t('needtoken.apiCost.callUnit') }])
const currency = computed<'CNY' | 'USD'>(() => currencyPreference.value === 'CNY' && fx.rate.value ? 'CNY' : 'USD')
// Never reinterpret a buyer-entered monthly budget in a different currency.
watch(currency, () => { comparisonAmount.value = '' })
watch(manualUnit, () => { manualInput.value = ''; manualOutput.value = ''; manualCacheRead.value = ''; manualCacheWrite.value = '' })
const currencyOptions = computed(() => [{ value: 'USD', label: 'USD' }, { value: 'CNY', label: 'CNY', disabled: !fx.rate.value }])
function selectCurrency(value: string) { if (value === 'USD' || value === 'CNY' && fx.rate.value) currencyPreference.value = value }
const manualRate = computed<ApiCostRate | null>(() => {
  const input = parseApiRate(manualInput.value), output = parseApiRate(manualOutput.value)
  if (input === null || manualUnit.value !== 'per_call' && output === null) return null
  return { key: 'manual', modelId: '', group: '', groupRatio: 1, unit: manualUnit.value === 'per_call' ? 'per_call' : 'per_million_tokens', inputUsd: input, outputUsd: output, cacheReadUsd: parseApiRate(manualCacheRead.value), cacheWriteUsd: parseApiRate(manualCacheWrite.value), sourceVersion: null }
})
const currentRate = computed(() => mode.value === 'manual' ? manualRate.value : liveReady.value ? selectedRate.value : null)
const perCall = computed(() => (mode.value === 'manual' ? manualUnit.value : selectedRate.value?.unit) === 'per_call')
const estimate = computed(() => estimateApiCost(currentRate.value, { calls: calls.value, input: inputTokens.value, output: outputTokens.value, cacheRead: cacheReadTokens.value, cacheWrite: cacheWriteTokens.value }))
const convertedEstimate = computed(() => estimate.value ? convertApiCost(estimate.value.monthlyUsd, currency.value, fx.rate.value) : null)
const comparison = computed(() => compareApiCost(convertedEstimate.value, comparisonAmount.value))
const formatAmount = (amount: number) => amount > 0 && amount < 0.01 ? t('needtoken.apiCost.smallAmount', { price: formatServiceMoney('0.01', currency.value, app.locale) }) : formatServiceMoney(amount.toFixed(10), currency.value, app.locale) || '—'
const formatCount = (value: number) => new Intl.NumberFormat(app.locale).format(value)
const formatDate = (value: string | Date) => new Date(value).toLocaleString(app.locale)
watch(pricing.loadedAt, loadedAt => {
  if (freshnessTimer) clearTimeout(freshnessTimer)
  clock.value = Date.now()
  const remaining = loadedAt ? 300_000 - Math.max(0, clock.value - loadedAt.getTime()) : 0
  if (remaining > 0) freshnessTimer = setTimeout(() => { clock.value = Date.now() }, remaining + 5)
}, { immediate: true })
function loadVisiblePricing() {
  visibilityObserver?.disconnect()
  void pricing.reload().finally(() => { clock.value = Date.now() })
}
onMounted(() => {
  if (!calculatorEl.value || !('IntersectionObserver' in window)) { loadVisiblePricing(); return }
  visibilityObserver = new IntersectionObserver(entries => { if (entries.some(entry => entry.isIntersecting)) loadVisiblePricing() }, { rootMargin: '180px' })
  visibilityObserver.observe(calculatorEl.value)
})
onUnmounted(() => { visibilityObserver?.disconnect(); if (freshnessTimer) clearTimeout(freshnessTimer) })
</script>

<style scoped>
.nt-api-cost { border: 1px solid var(--ui-border); border-radius: 23px; padding: clamp(20px, 4vw, 36px); background: linear-gradient(135deg, #2dd4a708, #ffffff02 60%), var(--ui-bg-elevated); min-width: 0; }
.cost-heading > span { font-size: 11px; color: var(--ui-accent); letter-spacing: .1em; }.cost-heading h2 { margin-top: 9px; font-size: clamp(24px, 3vw, 32px); font-weight: 700; letter-spacing: -.035em; color: var(--ui-text-primary); }.cost-heading > p { max-width: 70ch; margin-top: 10px; color: var(--ui-text-secondary); font-size: 13px; line-height: 1.85; }
.cost-modes { display: inline-flex; gap: 5px; padding: 4px; margin-top: 22px; border: 1px solid var(--ui-border); border-radius: 11px; background: var(--ui-bg-soft); }.cost-modes button { min-height: 36px; padding: 8px 13px; border-radius: 7px; color: var(--ui-text-muted); font-size: 12px; }.cost-modes button[aria-pressed=true] { color: var(--ui-accent); background: var(--ui-accent-soft); }
.cost-grid { display: grid; grid-template-columns: minmax(0, 1fr) minmax(240px, .64fr); gap: 28px; margin-top: 23px; }.cost-inputs, .cost-source, .cost-usage, .cost-cache, .cost-result { min-width: 0; }.cost-inputs { display: grid; align-content: start; gap: 23px; }.cost-inputs h3, .cost-comparison h3 { font-size: 13px; font-weight: 600; color: var(--ui-text-primary); }.cost-source > p { margin: 9px 0; color: var(--ui-text-muted); font-size: 12px; line-height: 1.75; }.cost-source-head { display: flex; align-items: center; justify-content: space-between; gap: 14px; }.cost-source-head button { display: inline-flex; align-items: center; gap: 5px; padding: 5px; font-size: 12px; color: var(--ui-accent); }.cost-source-head button:disabled { opacity: .5; }.cost-model-controls { display: grid; grid-template-columns: minmax(110px, .65fr) minmax(0, 1fr); gap: 9px; margin-top: 12px; }
.cost-fields { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 12px; margin-top: 12px; }.cost-fields-three { grid-template-columns: repeat(3, minmax(0, 1fr)); }.nt-api-cost label { display: grid; align-content: start; gap: 7px; min-width: 0; font-size: 12px; line-height: 1.65; color: var(--ui-text-secondary); }.nt-api-cost input { min-width: 0; width: 100%; min-height: 41px; border: 1px solid var(--ui-border-strong); border-radius: 9px; padding: 9px 11px; color: var(--ui-text-primary); background: var(--ui-bg-soft); font-size: 13px; }.nt-api-cost input::placeholder { color: var(--ui-text-muted); }.cost-small { margin-top: 9px; font-size: 12px; line-height: 1.8; color: var(--ui-text-muted); }.cost-status { margin-top: 12px; font-size: 12px; line-height: 1.75; color: #e5c48c; }.cost-cache { border-top: 1px solid var(--ui-border); padding-top: 16px; }.cost-cache summary { cursor: pointer; color: var(--ui-text-secondary); font-size: 12px; }
.cost-result { border: 1px solid var(--ui-border); border-radius: 17px; padding: 22px; background: var(--ui-bg-soft); }.cost-result-head { display: flex; align-items: center; justify-content: space-between; gap: 10px; color: var(--ui-text-secondary); font-size: 12px; }.cost-source-badge { display: inline-flex; margin-top: 18px; padding: 4px 8px; border: 1px solid #2dd4a733; border-radius: 6px; color: var(--ui-accent); font-size: 12px; }.cost-total { display: block; margin-top: 12px; font-size: clamp(30px, 4vw, 43px); line-height: 1.25; font-weight: 700; letter-spacing: -.045em; color: var(--ui-text-primary); overflow-wrap: anywhere; }.cost-rate-list { display: grid; gap: 7px; margin-top: 18px; padding-top: 15px; border-top: 1px solid var(--ui-border); font-size: 12px; line-height: 1.6; }.cost-rate-list > div { display: flex; justify-content: space-between; align-items: baseline; gap: 12px; }.cost-rate-list dt { color: var(--ui-text-muted); }.cost-rate-list dd { color: var(--ui-text-secondary); font-variant-numeric: tabular-nums; overflow-wrap: anywhere; text-align: right; }.cost-comparison { margin-top: 21px; padding-top: 18px; border-top: 1px solid var(--ui-border); }.cost-comparison label { margin-top: 11px; }.cost-difference { margin-top: 12px; color: var(--ui-accent); font-size: 13px; font-weight: 550; line-height: 1.75; }
.cost-notes { display: grid; gap: 9px; margin-top: 25px; padding-top: 19px; border-top: 1px solid var(--ui-border); color: var(--ui-text-muted); font-size: 12px; line-height: 1.85; overflow-wrap: anywhere; }.cost-notes > p:first-child { display: flex; align-items: flex-start; gap: 7px; color: var(--ui-text-secondary); }.cost-notes svg { flex: none; margin-top: 2px; }.cost-notes a { color: var(--ui-accent); }.nt-api-cost :is(button,input):focus-visible { outline: 2px solid var(--ui-accent); outline-offset: 3px; }
@media(max-width: 800px) { .cost-grid { grid-template-columns: minmax(0, 1fr); gap: 23px; }.cost-result { padding: 20px; }.cost-model-controls { grid-template-columns: minmax(0, 1fr); } }
@media(max-width: 460px) { .cost-fields-three { grid-template-columns: minmax(0, 1fr); }.cost-fields { gap: 10px; }.cost-modes button { padding: 8px 10px; font-size: 12px; }.nt-api-cost { border-radius: 17px; }.cost-result { padding: 17px; } }
</style>
