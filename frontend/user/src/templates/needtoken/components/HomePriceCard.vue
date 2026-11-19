<template>
  <component :is="available ? 'router-link' : 'article'" :to="available ? target : undefined" :tabindex="available ? undefined : 0" :aria-disabled="!available || undefined" class="home-price-card" :class="{ 'is-unavailable': !available }" :data-product-id="product.id" :data-sku-id="sku?.id" :data-brand="brand" :data-plan="plan">
    <div class="home-price-copy">
      <div class="home-price-topline">
        <span class="home-price-mark" aria-hidden="true">
          <ProductBrandMark :brand="brand" />
        </span>
        <span class="home-price-name" :title="title">{{ titleParts.before }}<span v-if="titleParts.tier" class="home-price-tier" :data-tier="titleParts.plan">{{ titleParts.tier }}</span>{{ titleParts.after }}</span>
        <span class="home-price-invoice" :title="t('needtoken.homeLibrary.invoice')"><ReceiptText :size="13" aria-hidden="true" /><span class="home-price-invoice-label">{{ t('needtoken.homeLibrary.invoice') }}</span></span>
      </div>
      <div class="home-price-meta">
        <span v-if="officialReference" class="home-price-official">{{ t('needtoken.homeLibrary.officialChip', { price: officialChip }) }}</span>
        <span v-if="variantLabel" class="home-price-variant">{{ variantLabel }}</span>
        <span v-if="!available" class="home-price-state">{{ availability === 'sold_out' ? t('products.stockStatus.outOfStock') : t('needtoken.catalogServices.availabilityPending') }}</span>
        <div class="home-price-art" aria-hidden="true">
          <span class="home-price-art__fallback" :class="{ 'is-hidden': imageLoaded && !imageFailed }"><PlanArtwork :plan="plan" :brand="brand" /></span>
          <img v-if="image && !imageFailed" :src="image" :class="{ 'is-loaded': imageLoaded }" width="88" height="80" decoding="async" alt="" @load="imageLoaded = true" @error="imageFailed = true" />
        </div>
      </div>
      <p class="home-price-amount">
        <span class="home-price-value"><small v-if="priceParts.prefix" class="home-price-currency">{{ priceParts.prefix }}</small>{{ priceParts.value }}</span>
        <small v-if="priceSuffix && price">/ {{ priceSuffix }}</small>
      </p>
    </div>
  </component>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { ReceiptText } from 'lucide-vue-next'
import { useAppStore } from '../../../stores/app'
import { getFirstImageUrl } from '../../../utils/image'
import { useDisplayCurrency } from '../composables/useDisplayCurrency'
import { formatServiceMoney } from '../lib/displayMoney'
import { serviceAvailability, serviceSkuPrice, type ServiceKind, type ServiceCardProduct } from '../lib/serviceCatalog'
import { serviceCardTitle, serviceSkuPresentation, serviceVariantLabel } from '../lib/serviceVariants'
import { homeCardBrand, homeCardPlan, homeCardPriceSuffix, homeCardSku, homeCardTitle } from '../lib/homeCardPresentation'
import ProductBrandMark from './ProductBrandMark.vue'
import PlanArtwork from './PlanArtwork.vue'

const props = defineProps<{ product: ServiceCardProduct; kind: ServiceKind }>()
const { t } = useI18n()
const appStore = useAppStore()
const { format } = useDisplayCurrency()
const imageFailed = ref(false)
const imageLoaded = ref(false)
const sku = computed(() => homeCardSku(props.product))
const availability = computed(() => serviceAvailability(props.product, sku.value))
const available = computed(() => availability.value === 'available' && !!sku.value)
const price = computed(() => serviceSkuPrice(sku.value))
const presentation = computed(() => serviceSkuPresentation(sku.value, appStore.locale))
const title = computed(() => serviceCardTitle(props.product, sku.value, appStore.locale))
const variantLabel = computed(() => props.product.cardHasVariants && presentation.value.title && sku.value
  ? serviceVariantLabel(sku.value, appStore.locale, '') : '')
const titleParts = computed(() => homeCardTitle(title.value))
const brand = computed(() => homeCardBrand(props.product, sku.value, appStore.locale))
const plan = computed(() => homeCardPlan(title.value, sku.value, appStore.locale))
const priceSuffix = computed(() => homeCardPriceSuffix(presentation.value.priceSuffix))
const priceText = computed(() => price.value ? format(price.value.amount) || t('needtoken.catalogServices.pricePending') : t('needtoken.catalogServices.pricePending'))
const priceParts = computed(() => {
  const match = /^(\D*?)(\d[\s\S]*)$/.exec(priceText.value)
  return match ? { prefix: match[1]!, value: match[2]! } : { prefix: '', value: priceText.value }
})
const officialReference = computed(() => presentation.value.officialPriceUsd ? formatServiceMoney(presentation.value.officialPriceUsd, 'USD', appStore.locale) : '')
const officialChip = computed(() => priceSuffix.value && officialReference.value ? `${officialReference.value}/${priceSuffix.value}` : officialReference.value)
const image = computed(() => getFirstImageUrl(props.product.images))
watch(image, () => { imageFailed.value = false; imageLoaded.value = false })
const target = computed(() => ({ path: `/products/${props.product.slug}`, query: { sku: String(sku.value?.id || 0), checkout: '1' } }))
</script>

<style scoped>
.home-price-card { position: relative; display: flex; min-width: 0; min-height: 156px; padding: 16px; border: 1px solid var(--nt-card-border); border-radius: 18px; background: var(--ui-bg-elevated); color: var(--ui-text-primary); box-shadow: var(--nt-card-shadow); transition: transform var(--nt-motion-enter, 180ms) var(--nt-motion-ease, ease-out), border-color var(--nt-motion-enter, 180ms), background-color var(--nt-motion-enter, 180ms), box-shadow var(--nt-motion-enter, 180ms); }
.home-price-card.is-unavailable { border-color: var(--ui-border); opacity: 0.78; }
.home-price-copy { display: flex; flex-direction: column; align-items: flex-start; min-width: 0; width: 100%; }
.home-price-topline { display: grid; grid-template-columns: 22px minmax(0, 1fr) auto; align-items: center; gap: 8px; width: 100%; min-width: 0; }
.home-price-mark { display: grid; width: 22px; height: 22px; place-items: center; transition: transform var(--nt-motion-enter, 180ms) var(--nt-motion-ease, ease-out); }
.home-price-mark :deep(.product-brand-mark), .home-price-mark :deep(.product-brand-symbol) { width: 100%; height: 100%; }
.home-price-name { min-width: 0; color: var(--ui-text-primary); font-size: 16px; font-weight: 600; line-height: 20px; letter-spacing: 0; overflow-wrap: anywhere; transition: color var(--nt-motion-enter, 180ms); }
.home-price-tier { color: var(--nt-tier-default); font-size: .9em; font-weight: 500; white-space: nowrap; }
.home-price-tier[data-tier="5x"] { color: var(--nt-tier-5x); }
.home-price-tier[data-tier="20x"] { color: var(--nt-tier-20x); }
.home-price-tier[data-tier="50x"] { color: var(--nt-tier-50x); }
.home-price-invoice { display: inline-flex; align-items: center; gap: 4px; color: var(--ui-text-muted); font-size: 11px; line-height: 16px; white-space: nowrap; }
.home-price-invoice svg { flex: none; }
.home-price-meta { position: relative; display: flex; flex-wrap: wrap; align-content: flex-start; align-items: center; gap: 4px 6px; width: 100%; min-height: 56px; max-width: 100%; margin-top: 8px; padding-right: 76px; }
.home-price-official { padding: 1px 6px; border: 1px solid var(--ui-border); border-radius: 999px; background: var(--nt-surface-subtle); color: var(--ui-text-secondary); font-size: 11px; line-height: 16px; overflow-wrap: anywhere; }
.home-price-variant { color: var(--ui-text-muted); font-size: 11px; line-height: 18px; overflow-wrap: anywhere; }
.home-price-state { display: inline-flex; align-items: center; gap: 4px; color: var(--nt-warning-ink); font-size: 11px; line-height: 18px; }
.home-price-state::before { content: ''; width: 4px; height: 4px; flex: none; border-radius: 50%; background: var(--nt-warning-ink); }
.home-price-amount { position: relative; z-index: 1; display: flex; flex-wrap: wrap; align-items: baseline; gap: 2px 6px; max-width: 100%; margin-top: auto; padding-top: 4px; color: var(--ui-text-primary); font-size: 28px; font-weight: 600; line-height: 32px; font-variant-numeric: tabular-nums lining-nums; }
.home-price-value { min-width: 0; overflow-wrap: anywhere; }
.home-price-amount > small { color: var(--ui-text-muted); font-size: 12px; line-height: 16px; font-weight: 400; }
.home-price-currency { margin-right: 2px; font-size: .58em; font-weight: 500; line-height: 1; white-space: pre; }
.home-price-art { position: absolute; right: 0; top: 0; width: 72px; height: 56px; pointer-events: none; }
.home-price-art__fallback { position: absolute; inset: 0; transition: opacity var(--nt-motion-enter, 180ms); }
.home-price-art__fallback.is-hidden { opacity: 0; }
.home-price-art img { position: absolute; right: 0; bottom: 0; width: 64px; height: 52px; opacity: 0; object-fit: contain; filter: drop-shadow(0 3px 6px rgba(0, 0, 0, 0.14)); transform-origin: 50% 75%; transition: opacity var(--nt-motion-enter, 180ms), transform var(--nt-motion-enter, 180ms) var(--nt-motion-ease, ease-out); }
.home-price-art img.is-loaded { opacity: 1; }
.home-price-card:focus-visible { outline: 3px solid var(--ui-accent); outline-offset: 4px; transform: translateY(-2px); border-color: var(--nt-card-border-hover); background: var(--nt-surface-hover); box-shadow: var(--nt-card-shadow-hover); }
.home-price-card:focus-visible .home-price-mark { transform: translateY(-1px) rotate(-4deg); }
.home-price-card:focus-visible .home-price-name { color: var(--nt-accent-ink-hover); }
.home-price-card:focus-visible .home-price-art img { transform: translateY(-2px) rotate(-2deg); }
@media (hover: hover) {
  .home-price-card:hover { transform: translateY(-2px); border-color: var(--nt-card-border-hover); background: var(--nt-surface-hover); box-shadow: var(--nt-card-shadow-hover); }
  .home-price-card:hover .home-price-mark { transform: translateY(-1px) rotate(-4deg); }
  .home-price-card:hover .home-price-name { color: var(--nt-accent-ink-hover); }
  .home-price-card:hover .home-price-art img { transform: translateY(-2px) rotate(-2deg); }
}
a.home-price-card:active { transform: translateY(1px); box-shadow: var(--nt-card-shadow); }
a.home-price-card:active .home-price-mark, a.home-price-card:active .home-price-art img { transform: translateY(0); }
@media (min-width: 1440px) {
  .home-price-card { min-height: 132px; padding: 12px; border-radius: 16px; }
  .home-price-topline { grid-template-columns: 20px minmax(0, 1fr) 13px; gap: 6px; }
  .home-price-mark { width: 20px; height: 20px; }
  .home-price-name { font-size: 14px; }
  .home-price-invoice-label { position: absolute; width: 1px; height: 1px; padding: 0; overflow: hidden; clip: rect(0, 0, 0, 0); white-space: nowrap; }
  .home-price-meta { min-height: 44px; gap: 4px; padding-right: 60px; }
  .home-price-official { padding-inline: 5px; }
  .home-price-amount { padding-top: 4px; font-size: 24px; line-height: 28px; gap: 2px 4px; }
  .home-price-amount > small { font-size: 11px; }
  .home-price-art { right: 0; top: 0; width: 56px; height: 44px; }
  .home-price-art img { width: 52px; height: 42px; }
}
@media (max-width: 640px) { .home-price-card { min-height: 156px; } }
@media (max-width: 359px) {
  .home-price-card { padding: 14px; }
  .home-price-name { font-size: 15px; }
  .home-price-invoice-label { position: absolute; width: 1px; height: 1px; padding: 0; overflow: hidden; clip: rect(0, 0, 0, 0); white-space: nowrap; }
  .home-price-amount { font-size: 26px; line-height: 30px; }
}
@media (prefers-reduced-motion: reduce) {
  .home-price-card, .home-price-name, .home-price-mark, .home-price-art img, .home-price-art__fallback { transition: none; }
  .home-price-card:is(:hover, :focus-visible, :active), .home-price-card:is(:hover, :focus-visible, :active) :is(.home-price-mark, .home-price-art img) { transform: none; }
}
</style>
