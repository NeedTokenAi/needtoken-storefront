<template>
  <article class="nt-service-card" :class="{ 'nt-service-card--muted': availability !== 'available' }" :data-sku-id="selectedSkuId || undefined" :data-brand="brand" @pointermove="onCardPointerMove">
    <div class="nt-service-card__heading">
      <div class="nt-service-card__identity">
        <div class="nt-service-card__icon" aria-hidden="true">
          <ProductBrandMark v-if="brand !== 'unknown'" :brand="brand" />
          <img v-else-if="image && !imageErrored" :src="image" alt="" loading="lazy" @error="imageErrored = true" />
          <Zap v-else-if="kind === 'recharge'" :size="20" :stroke-width="1.7" />
          <PackageCheck v-else :size="20" :stroke-width="1.7" />
        </div>
        <span class="nt-service-card__label">{{ categoryLabel || t(`needtoken.catalogServices.${kind}.label`) }}</span>
      </div>
      <span v-if="selectedSku && skus.length === 1" class="nt-service-card__single-option">{{ skuLabel(selectedSku, 0) }}</span>
    </div>

    <div class="nt-service-card__body">
      <h2 aria-live="polite">{{ titleParts.before }}<span v-if="titleParts.tier" class="nt-service-card__tier" :data-tier="titleParts.plan">{{ titleParts.tier }}</span>{{ titleParts.after }}</h2>
      <div v-if="variantAxes.length" class="nt-service-card__selectors">
        <fieldset v-for="axis in variantAxes" :key="axis.id" class="nt-service-card__axis">
          <legend class="sr-only">{{ t(`needtoken.variants.${axis.id}Label`) }}</legend>
          <span class="nt-service-card__axis-label" aria-hidden="true">{{ t(`needtoken.variants.${axis.id}Label`) }}</span>
          <div class="nt-service-card__options">
            <label v-for="value in axis.options" :key="value" class="nt-service-card__option" :class="{ 'is-selected': axis.valuesBySku[selectedSkuId] === value }" :aria-disabled="!axisOptionAvailable(axis.id, value)">
              <input type="radio" :name="`service-sku-${product.id}-${axis.id}`" :value="value" :checked="axis.valuesBySku[selectedSkuId] === value" :disabled="!axisOptionAvailable(axis.id, value)" @change="selectAxis(axis.id, value)" />
              <span>{{ axisLabel(axis.id, value) }}</span>
            </label>
          </div>
        </fieldset>
      </div>
      <fieldset v-else-if="skus.length > 1" class="nt-service-card__options nt-service-card__flat-options">
        <legend class="sr-only">{{ t('needtoken.catalogServices.chooseOption') }}</legend>
        <label v-for="(sku, index) in skus" :key="sku.id" class="nt-service-card__option" :class="{ 'is-selected': selectedSkuId === sku.id }">
          <input v-model="selectedSkuId" type="radio" :name="`service-sku-${product.id}`" :value="sku.id" />
          <span>{{ skuLabel(sku, index) }}</span>
        </label>
      </fieldset>
      <p class="nt-service-card__description" :title="displayDescription">{{ displayDescription }}</p>
      <ul v-if="presentation.features.length" class="nt-service-card__features">
        <li v-for="(feature, index) in presentation.features.slice(0, 2)" :key="index"><Check :size="15" aria-hidden="true" /><span :title="feature">{{ feature }}</span></li>
      </ul>
      <div v-else class="nt-service-card__details">
        <span><Check :size="14" aria-hidden="true" />{{ deliveryText || t(`needtoken.catalogServices.${kind}.delivery`) }}</span>
        <span><ShieldCheck :size="14" aria-hidden="true" />{{ t('needtoken.catalogServices.termsInDetail') }}</span>
      </div>
    </div>

    <div class="nt-service-card__price-row">
      <div class="nt-service-card__price-block" aria-live="polite">
        <p v-if="isConverted && price" class="nt-service-card__price-note">{{ t('needtoken.variants.convertedPrice') }}</p>
        <p class="nt-service-card__price"><span :key="`${selectedSkuId}-${price?.amount ?? ''}`" class="nt-service-card__price-value">{{ price ? format(price.amount) || t('needtoken.catalogServices.pricePending') : t('needtoken.catalogServices.pricePending') }}</span><span v-if="priceSuffix && price" class="nt-service-card__price-suffix">/ {{ priceSuffix }}</span></p>
        <p v-if="price?.originalAmount" class="nt-service-card__discount">
          <s>{{ format(price.originalAmount) }}</s>
          <span>{{ price.kind === 'member' ? t('products.memberPriceTag') : t('products.promotionTag') }}</span>
        </p>
        <p v-if="isConverted && price" class="nt-service-card__reference">{{ t('needtoken.variants.settlementPrice', { price: formatSettlement(price.amount) }) }}</p>
        <p v-if="officialReference" class="nt-service-card__reference">{{ t('needtoken.variants.officialReference', { price: officialReference }) }}</p>
      </div>
      <RouterLink v-if="availability === 'available' && selectedSku" :to="purchaseTarget" class="nt-service-card__cta">
        {{ ctaLabel || t(`needtoken.catalogServices.${kind}.cta`) }}<ArrowUpRight :size="15" aria-hidden="true" />
      </RouterLink>
      <button v-else type="button" class="nt-service-card__cta" disabled>
        {{ availability === 'sold_out' ? t('products.stockStatus.outOfStock') : t('needtoken.catalogServices.availabilityPending') }}
      </button>
    </div>
    <RouterLink :to="{ path: `/products/${product.slug}`, query: { sku: String(selectedSkuId || 0) } }" class="nt-service-card__read-more">{{ t('needtoken.catalogServices.viewDetails') }}</RouterLink>
  </article>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { ArrowUpRight, Check, PackageCheck, ShieldCheck, Zap } from 'lucide-vue-next'
import { useLocalized } from '../../../composables/useProduct'
import { useAppStore } from '../../../stores/app'
import { useUserAuthStore } from '../../../stores/userAuth'
import { getFirstImageUrl } from '../../../utils/image'
import { formatServiceMoney } from '../lib/displayMoney'
import { useDisplayCurrency } from '../composables/useDisplayCurrency'
import { selectServiceVariant, serviceCardTitle, serviceSkuPresentation, serviceVariantAxes, serviceVariantLabel, type ServiceVariantAxis } from '../lib/serviceVariants'
import { apiPlanMemberLoginPath } from '../lib/apiPlanCatalog'
import { homeCardBrand, homeCardPriceSuffix, homeCardTitle } from '../lib/homeCardPresentation'
import ProductBrandMark from './ProductBrandMark.vue'
import {
  activeServiceSkus, serviceAvailability, serviceDescriptionText, serviceSkuPrice,
  type ServiceKind, type ServiceCardProduct, type ServiceSku,
} from '../lib/serviceCatalog'

const props = defineProps<{ product: ServiceCardProduct; kind: ServiceKind; categoryLabel?: string; memberOnly?: boolean; ctaLabel?: string; deliveryText?: string; descriptionFallback?: string }>()
const { t } = useI18n()
const appStore = useAppStore()
const userAuthStore = useUserAuthStore()
const { getLocalizedText } = useLocalized()
const { format, formatSettlement, isConverted } = useDisplayCurrency()
const selectedSkuId = ref(0)
const imageErrored = ref(false)
const skus = computed(() => activeServiceSkus(props.product))
const selectedSku = computed(() => skus.value.find(sku => sku.id === selectedSkuId.value) || null)
const purchaseTarget = computed(() => props.memberOnly && !userAuthStore.isAuthenticated
  ? apiPlanMemberLoginPath(props.product.slug, selectedSku.value?.id || 0)
  : { path: `/products/${props.product.slug}`, query: { sku: String(selectedSku.value?.id || 0), checkout: '1' } })
const availability = computed(() => serviceAvailability(props.product, selectedSku.value))
const price = computed(() => serviceSkuPrice(selectedSku.value))
const image = computed(() => getFirstImageUrl(props.product.images))
const description = computed(() => serviceDescriptionText(getLocalizedText(props.product.description)))
const presentation = computed(() => serviceSkuPresentation(selectedSku.value, appStore.locale))
const titleParts = computed(() => homeCardTitle(serviceCardTitle(props.product, selectedSku.value, appStore.locale)))
const brand = computed(() => homeCardBrand(props.product, selectedSku.value, appStore.locale))
const priceSuffix = computed(() => homeCardPriceSuffix(presentation.value.priceSuffix))
const displayDescription = computed(() => presentation.value.description || description.value || props.descriptionFallback || t(`needtoken.catalogServices.${props.kind}.descriptionFallback`))
const variantAxes = computed(() => serviceVariantAxes(skus.value, appStore.locale))
const officialReference = computed(() => presentation.value.officialPriceUsd ? formatServiceMoney(presentation.value.officialPriceUsd, 'USD', appStore.locale) : null)
const axisLabel = (id: ServiceVariantAxis['id'], value: string) => id === 'period' && (value === 'month' || value === 'year')
  ? t(`needtoken.variants.${value === 'month' ? 'monthly' : 'yearly'}`) : value
const skuLabel = (sku: ServiceSku, index: number) => axisLabel('period', serviceVariantLabel(sku, appStore.locale, skus.value.length > 1
  ? t('needtoken.variants.optionNumber', { number: index + 1 })
  : t('needtoken.catalogServices.standardOption')))
const axisOptionAvailable = (id: ServiceVariantAxis['id'], value: string) => selectServiceVariant(skus.value, variantAxes.value, selectedSkuId.value, id, value) !== null
const selectAxis = (id: ServiceVariantAxis['id'], value: string) => {
  const next = selectServiceVariant(skus.value, variantAxes.value, selectedSkuId.value, id, value)
  if (next !== null) selectedSkuId.value = next
}

watch(skus, (current) => {
  if (!current.some(sku => sku.id === selectedSkuId.value)) {
    selectedSkuId.value = (current.find(sku => serviceAvailability(props.product, sku) === 'available') || current[0])?.id || 0
  }
}, { immediate: true })
watch(image, () => { imageErrored.value = false })
function onCardPointerMove(event: PointerEvent) {
  if (event.pointerType === 'touch') return
  const el = event.currentTarget as HTMLElement | null
  if (!el) return
  const rect = el.getBoundingClientRect()
  el.style.setProperty('--mx', `${Math.round(event.clientX - rect.left)}px`)
  el.style.setProperty('--my', `${Math.round(event.clientY - rect.top)}px`)
}
</script>

<style scoped>
.nt-service-card { display: flex; overflow: hidden; min-width: 0; max-width: 100%; flex-direction: column; position: relative; padding: 20px; border: 1px solid var(--nt-card-border); border-radius: 20px; background: var(--ui-bg-elevated); box-shadow: var(--nt-card-shadow); transition: transform var(--nt-duration-rebound, 480ms) var(--nt-ease-spring), border-color var(--nt-motion-enter) var(--nt-motion-ease), background-color var(--nt-motion-enter) var(--nt-motion-ease), box-shadow var(--nt-motion-enter) var(--nt-motion-ease); }
.nt-service-card::before { content: ''; position: absolute; top: 0; left: 16%; right: 16%; height: 1px; background: var(--nt-card-top-sheen); opacity: 0; transition: opacity 240ms var(--nt-motion-ease); pointer-events: none; z-index: 2; }
.nt-service-card::after { content: ''; position: absolute; inset: 0; border-radius: inherit; background: radial-gradient(280px circle at var(--mx, 50%) var(--my, 0%), var(--nt-card-spotlight), transparent 72%); opacity: 0; transition: opacity 260ms var(--nt-motion-ease); pointer-events: none; z-index: 0; }
.nt-service-card > * { position: relative; z-index: 1; }
@media (hover: hover) {
  .nt-service-card:hover { transform: translateY(-3px); border-color: var(--nt-card-border-hover); background: var(--nt-surface-hover); box-shadow: var(--nt-card-shadow-hover); }
  .nt-service-card:hover::before, .nt-service-card:hover::after { opacity: 1; }
}
.nt-service-card:focus-within { border-color: var(--nt-card-border-hover); background: var(--nt-surface-hover); box-shadow: var(--nt-card-shadow-hover); }
.nt-service-card:focus-within::before { opacity: 1; }
.nt-service-card__heading { display: flex; align-items: center; gap: 8px; min-width: 0; margin-bottom: 8px; }
.nt-service-card__identity { display: flex; flex: 1 1 90px; align-items: center; gap: 8px; min-width: 0; }
.nt-service-card__icon { display: grid; width: 28px; height: 28px; flex-shrink: 0; place-items: center; overflow: hidden; color: var(--ui-accent); border: 1px solid var(--ui-border); background: var(--ui-bg-soft); border-radius: 8px; }
.nt-service-card__icon > svg { width: 17px; height: 17px; }
.nt-service-card__icon img { width: 100%; height: 100%; object-fit: contain; }
.nt-service-card__icon > :is(img, svg, .product-brand-mark) { transition: transform 440ms var(--nt-ease-spring); }
.nt-service-card:is(:hover, :focus-within) .nt-service-card__icon > :is(img, svg, .product-brand-mark) { transform: translateY(-2px) rotate(-4deg) scale(1.04); }
.nt-service-card__label { min-width: 0; color: var(--ui-text-muted); font-size: 12px; overflow-wrap: anywhere; line-height: 1.4; }
.nt-service-card__selectors { display: grid; gap: 6px; margin-top: 11px; }
.nt-service-card__axis { display: flex; align-items: center; gap: 8px; min-inline-size: 0; margin: 0; padding: 0; border: 0; }
.nt-service-card__axis-label { flex-shrink: 0; color: var(--ui-text-muted); font-size: 12px; line-height: 1.4; }
.nt-service-card__options { display: flex; flex-wrap: wrap; gap: 3px; min-inline-size: 0; max-width: 100%; margin: 0; padding: 3px; border: 1px solid var(--ui-border); border-radius: 12px; background: var(--nt-surface-subtle); }
.nt-service-card__flat-options { margin-top: 11px; width: fit-content; }
.nt-service-card__option { position: relative; display: inline-flex; align-items: center; justify-content: center; min-width: 32px; min-height: 32px; max-width: 100%; padding: 5px 9px; border: 1px solid transparent; border-radius: 8px; color: var(--ui-text-secondary); font-size: 12px; line-height: 1.35; text-align: center; cursor: pointer; transition: color .15s, background .15s, border-color .15s, transform var(--nt-duration-rebound, 480ms) var(--nt-ease-spring); }
.nt-service-card__option:not([aria-disabled="true"]):active { transform: scale(0.95); transition-duration: var(--nt-duration-press, 80ms); }
.nt-service-card__option > span { min-width: 0; overflow-wrap: anywhere; }
.nt-service-card__option input { position: absolute; opacity: 0; width: 1px; height: 1px; }
.nt-service-card__option:focus-within { outline: 2px solid var(--ui-accent); outline-offset: 2px; }
.nt-service-card__option[aria-disabled="true"] { color: var(--ui-text-muted); opacity: .4; cursor: not-allowed; }
.nt-service-card__option:not(.is-selected):not([aria-disabled="true"]):hover { color: var(--ui-text-primary); border-color: var(--nt-accent-border); background: var(--ui-bg-soft); }
.nt-service-card__option.is-selected { color: var(--ui-accent); border-color: var(--nt-accent-border-strong); background: var(--ui-accent-soft); font-weight: 600; }
.nt-service-card__single-option { min-width: 0; max-width: 48%; padding: 5px 8px; border: 1px solid var(--nt-accent-border); border-radius: 9px; color: var(--ui-accent); background: var(--nt-accent-subtle); font-size: 12px; overflow-wrap: anywhere; }
.nt-service-card__body { min-width: 0; flex: 1; }
h2 { margin: 0; color: var(--ui-text-primary); font-size: 18px; font-weight: 600; line-height: 1.4; letter-spacing: 0; overflow-wrap: anywhere; transition: color var(--nt-motion-fast) var(--nt-motion-ease); }
.nt-service-card__tier { color: var(--nt-tier-default); font-size: .94em; white-space: nowrap; }
.nt-service-card__tier[data-tier="5x"] { color: var(--nt-tier-5x); }
.nt-service-card__tier[data-tier="20x"] { color: var(--nt-tier-20x); }
.nt-service-card__tier[data-tier="50x"] { color: var(--nt-tier-50x); }
.nt-service-card:is(:hover, :focus-within) h2 { color: var(--nt-accent-ink-hover); }
.nt-service-card__description { display: -webkit-box; -webkit-box-orient: vertical; -webkit-line-clamp: 2; overflow: hidden; margin-top: 9px; min-height: 43px; font-size: 13px; line-height: 1.65; color: var(--ui-text-secondary); overflow-wrap: anywhere; }
.nt-service-card__features { display: grid; gap: 6px; margin-top: 11px; color: var(--ui-text-secondary); font-size: 12px; }
.nt-service-card__features li { display: flex; align-items: flex-start; gap: 7px; min-width: 0; line-height: 1.6; }
.nt-service-card__features li > svg { flex-shrink: 0; margin-top: 2px; color: var(--ui-accent); }
.nt-service-card__features li > span { display: -webkit-box; -webkit-box-orient: vertical; -webkit-line-clamp: 1; overflow: hidden; min-width: 0; overflow-wrap: anywhere; }
.nt-service-card__details { display: grid; gap: 6px; margin-top: 10px; color: var(--ui-text-secondary); font-size: 12px; }
.nt-service-card__details > span { display: flex; align-items: flex-start; gap: 7px; min-width: 0; line-height: 1.6; overflow-wrap: anywhere; }
.nt-service-card__details svg { flex-shrink: 0; margin-top: 2px; color: var(--ui-accent); }
.nt-service-card__price-row { display: flex; flex-wrap: wrap; gap: 10px 9px; align-items: flex-end; justify-content: space-between; min-width: 0; margin-top: auto; padding-top: 12px; }
.nt-service-card__price-block { flex: 1 1 120px; min-width: 0; max-width: 100%; }
.nt-service-card__price { display: flex; flex-wrap: wrap; align-items: baseline; gap: 4px; min-width: 0; color: var(--ui-text-primary); font-weight: 600; font-size: clamp(28px, 2.4vw, 34px); line-height: 1.15; letter-spacing: 0; font-variant-numeric: tabular-nums; }
.nt-service-card__price > span { min-width: 0; overflow-wrap: anywhere; }
.nt-service-card__price-value { display: inline-block; animation: nt-price-tick 200ms var(--nt-ease-spring-soft) backwards; }
@keyframes nt-price-tick { from { opacity: 0.45; transform: translateY(4px); } to { opacity: 1; transform: translateY(0); } }
.nt-service-card__price .nt-service-card__price-suffix { color: var(--ui-text-muted); font-size: 12px; font-weight: 500; letter-spacing: 0; }
.nt-service-card__price-note { margin-bottom: 4px; color: var(--ui-text-muted); font-size: 10px; }
.nt-service-card__discount { display: flex; flex-wrap: wrap; align-items: center; gap: 4px 7px; margin-top: 5px; color: var(--ui-text-muted); font-size: 11px; overflow-wrap: anywhere; }
.nt-service-card__discount span { color: var(--ui-accent); }
.nt-service-card__reference { margin-top: 5px; color: var(--ui-text-muted); font-size: 10px; line-height: 1.4; overflow-wrap: anywhere; }
.nt-service-card__cta { display: flex; flex-shrink: 0; align-items: center; justify-content: center; gap: 6px; min-height: 40px; max-width: 100%; padding: 8px 12px; border: 1px solid var(--nt-purchase-border); border-radius: 11px; background: var(--nt-purchase-bg); color: var(--nt-purchase-fg); font-size: 12px; font-weight: 650; text-align: center; transition: background-color .15s, border-color .15s, transform var(--nt-duration-rebound, 480ms) var(--nt-ease-spring); }
.nt-service-card__cta svg { transition: transform 320ms var(--nt-ease-spring); }
.nt-service-card__cta:hover:not(:disabled) { background: var(--nt-purchase-bg-hover); border-color: var(--nt-purchase-bg-hover); }
.nt-service-card__cta:hover:not(:disabled) svg { transform: translate(2px, -2px); }
.nt-service-card__cta:active:not(:disabled) { transform: scale(0.965); transition-duration: var(--nt-duration-press, 80ms); }
.nt-service-card__cta:focus-visible, .nt-service-card__read-more:focus-visible { outline: 2px solid var(--ui-accent); outline-offset: 4px; }
.nt-service-card__cta:disabled { color: var(--nt-purchase-disabled-fg); background: var(--nt-purchase-disabled-bg); border-color: var(--nt-purchase-disabled-border); cursor: not-allowed; }
.nt-service-card__read-more { display: flex; justify-content: center; align-items: center; min-height: 32px; margin-top: 4px; text-align: center; color: var(--ui-text-muted); font-size: 12px; }
.nt-service-card__read-more:hover { color: var(--ui-text-primary); }
@media (forced-colors: active) { .nt-service-card__option.is-selected { color: ButtonText; background: ButtonFace; border-color: Highlight; outline: 2px solid Highlight; outline-offset: 1px; } }
@media (hover: none), (pointer: coarse) { .nt-service-card__option, .nt-service-card__cta, .nt-service-card__read-more { min-height: 44px; } .nt-service-card__option { min-width: 44px; } }
@media (prefers-reduced-motion: reduce) {
  .nt-service-card, .nt-service-card::before, .nt-service-card::after, .nt-service-card h2, .nt-service-card__icon > :is(img, svg, .product-brand-mark), .nt-service-card__option, .nt-service-card__cta, .nt-service-card__cta svg { transition: none; }
  .nt-service-card::after { display: none; }
  .nt-service-card__price-value { animation: none; }
  .nt-service-card:hover, .nt-service-card:is(:hover, :focus-within) .nt-service-card__icon > :is(img, svg, .product-brand-mark), .nt-service-card__option:active, .nt-service-card__cta:hover svg, .nt-service-card__cta:active { transform: none; }
}
@media (max-width: 640px) { .nt-service-card { padding: 20px; } h2 { font-size: 19px; } .nt-service-card__description { font-size: 13px; min-height: 43px; } .nt-service-card__price { font-size: 30px; } .nt-service-card__option, .nt-service-card__cta, .nt-service-card__read-more { min-height: 44px; } .nt-service-card__option { min-width: 44px; } }
</style>
