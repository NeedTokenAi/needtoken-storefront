<template>
  <component :is="available ? 'router-link' : 'article'" :to="available ? target : undefined" class="home-price-card" :class="{ 'is-unavailable': !available }">
    <div class="home-price-copy">
      <div class="home-price-topline">
        <span class="home-price-mark" aria-hidden="true">
          <Zap v-if="kind === 'recharge'" :size="18" :stroke-width="1.5" />
          <PackageCheck v-else :size="18" :stroke-width="1.5" />
        </span>
        <span class="home-price-name">{{ titleParts.name }}<span v-if="titleParts.tier" class="home-price-tier">{{ titleParts.tier }}</span></span>
        <span class="home-price-invoice" :title="t('needtoken.homeLibrary.invoice')"><ReceiptText :size="13" aria-hidden="true" />{{ t('needtoken.homeLibrary.invoice') }}</span>
      </div>
      <p v-if="officialReference" class="home-price-official">{{ t('needtoken.homeLibrary.officialChip', { price: officialChip }) }}</p>
      <p class="home-price-amount">
        <span>{{ priceText }}</span>
        <small v-if="presentation.priceSuffix && price">/ {{ presentation.priceSuffix }}</small>
      </p>
      <p v-if="!available" class="home-price-state">{{ availability === 'sold_out' ? t('products.stockStatus.outOfStock') : t('needtoken.catalogServices.availabilityPending') }}</p>
    </div>
    <div class="home-price-art" aria-hidden="true">
      <img v-if="image && !imageFailed" :src="image" width="68" height="68" decoding="async" alt="" @error="imageFailed = true" />
      <span v-else class="home-price-art-fallback"><Zap v-if="kind === 'recharge'" :size="22" /><PackageCheck v-else :size="22" /></span>
    </div>
  </component>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { PackageCheck, ReceiptText, Zap } from 'lucide-vue-next'
import { useLocalized } from '../../../composables/useProduct'
import { useAppStore } from '../../../stores/app'
import { getFirstImageUrl } from '../../../utils/image'
import { useDisplayCurrency } from '../composables/useDisplayCurrency'
import { formatServiceMoney } from '../lib/displayMoney'
import {
  activeServiceSkus,
  serviceAvailability,
  serviceSkuPrice,
  type ServiceKind,
  type ServiceProduct,
} from '../lib/serviceCatalog'
import { serviceSkuPresentation } from '../lib/serviceVariants'

const props = defineProps<{ product: ServiceProduct; kind: ServiceKind }>()
const { t } = useI18n()
const appStore = useAppStore()
const { getLocalizedText } = useLocalized()
const { format } = useDisplayCurrency()
const imageFailed = ref(false)
const skus = computed(() => activeServiceSkus(props.product))
const sku = computed(() => skus.value.find(row => serviceAvailability(props.product, row) === 'available') || skus.value[0] || null)
const availability = computed(() => serviceAvailability(props.product, sku.value))
const available = computed(() => availability.value === 'available' && !!sku.value)
const price = computed(() => serviceSkuPrice(sku.value))
const presentation = computed(() => serviceSkuPresentation(sku.value, appStore.locale))
const title = computed(() => presentation.value.title || getLocalizedText(props.product.title))
const titleParts = computed(() => {
  const match = title.value.match(/^(.*?)(\s+\d+(?:\.\d+)?[x×])$/i)
  return { name: match?.[1] || title.value, tier: match?.[2] || '' }
})
const priceText = computed(() => price.value ? format(price.value.amount) || t('needtoken.catalogServices.pricePending') : t('needtoken.catalogServices.pricePending'))
const officialReference = computed(() => presentation.value.officialPriceUsd ? formatServiceMoney(presentation.value.officialPriceUsd, 'USD', appStore.locale) : '')
const officialChip = computed(() => presentation.value.priceSuffix && officialReference.value ? `${officialReference.value}/${presentation.value.priceSuffix}` : officialReference.value)
const image = computed(() => getFirstImageUrl(props.product.images))
watch(image, () => { imageFailed.value = false })
const target = computed(() => ({ path: `/products/${props.product.slug}`, query: { sku: String(sku.value?.id || 0), checkout: '1' } }))
</script>

<style scoped>
.home-price-card { position: relative; display: flex; min-width: 0; min-height: 180px; padding: 20px; border: 1px solid #dde8e3; border-radius: 20px; background: #f8fbf9; color: #18251f; box-shadow: 0 2px 8px #00000012; transition: transform var(--nt-motion-enter, 180ms) var(--nt-motion-ease, ease-out), border-color var(--nt-motion-enter, 180ms), box-shadow var(--nt-motion-enter, 180ms); }
.home-price-card.is-unavailable { background: #e8edeb; color: #5f6d66; }
.home-price-copy { min-width: 0; display: flex; flex-direction: column; align-items: flex-start; width: 100%; }
.home-price-topline { display: flex; flex-wrap: wrap; align-items: center; gap: 8px; width: 100%; min-width: 0; }
.home-price-mark { display: grid; width: 24px; height: 24px; flex: none; place-items: center; color: #198266; transition: color var(--nt-motion-enter, 180ms), transform var(--nt-motion-enter, 180ms) var(--nt-motion-ease, ease-out); }
.home-price-name { flex: 1; min-width: 0; font-size: 18px; font-weight: 600; line-height: 1.35; letter-spacing: 0; overflow-wrap: anywhere; transition: color var(--nt-motion-enter, 180ms); }
.home-price-tier { color: #62736a; font-size: .9em; font-weight: 500; white-space: nowrap; transition: color var(--nt-motion-enter, 180ms); }
.home-price-invoice { display: inline-flex; align-items: center; gap: 4px; color: #62736b; font-size: 11px; white-space: nowrap; }
.home-price-official { margin-top: 12px; padding: 4px 8px; border: 1px solid #d8e1dc; border-radius: 999px; color: #5e6e65; font-size: 12px; line-height: 1.4; }
.home-price-amount { position: relative; z-index: 1; display: flex; flex-wrap: wrap; align-items: baseline; gap: 4px; max-width: 100%; margin-top: auto; padding-top: 24px; padding-right: 72px; color: #18251f; font-size: 30px; font-weight: 600; letter-spacing: 0; line-height: 1.15; font-variant-numeric: tabular-nums lining-nums; }
.home-price-amount > span { overflow-wrap: anywhere; }
.home-price-amount small { color: #62736b; font-size: 13px; font-weight: 400; letter-spacing: 0; }
.home-price-state { position: relative; z-index: 1; margin-top: 8px; color: #62736b; font-size: 12px; }
.home-price-art { position: absolute; right: 20px; bottom: 20px; width: 68px; height: 68px; pointer-events: none; }
.home-price-art img { width: 100%; height: 100%; object-fit: contain; filter: drop-shadow(0 4px 8px #00000012); transform-origin: 50% 75%; transition: transform var(--nt-motion-enter, 180ms) var(--nt-motion-ease, ease-out); }
.home-price-art-fallback { display: grid; place-items: center; height: 100%; color: #8ba99b; }
a.home-price-card:focus-visible { outline: 3px solid var(--ui-accent); outline-offset: 4px; transform: translateY(-2px); border-color: #b7cec1; box-shadow: 0 6px 16px #0000001c; }
a.home-price-card:focus-visible .home-price-mark { transform: translateY(-1px) rotate(-8deg); color: #12694e; }
a.home-price-card:focus-visible .home-price-name { color: #164d39; }
a.home-price-card:focus-visible .home-price-tier { color: #416653; }
a.home-price-card:focus-visible .home-price-art img { transform: translateY(-4px) rotate(-2deg); }
@media (hover: hover) {
  a.home-price-card:hover { transform: translateY(-2px); border-color: #b7cec1; box-shadow: 0 6px 16px #0000001c; }
  a.home-price-card:hover .home-price-mark { transform: translateY(-1px) rotate(-8deg); color: #12694e; }
  a.home-price-card:hover .home-price-name { color: #164d39; }
  a.home-price-card:hover .home-price-tier { color: #416653; }
  a.home-price-card:hover .home-price-art img { transform: translateY(-4px) rotate(-2deg); }
}
a.home-price-card:active { transform: translateY(1px); box-shadow: 0 1px 4px #00000012; }
a.home-price-card:active .home-price-mark, a.home-price-card:active .home-price-art img { transform: translateY(0); }
@media (min-width: 1440px) { .home-price-card { padding: 20px; } .home-price-topline { gap: 6px; } .home-price-mark { width: 24px; } .home-price-name { font-size: 16px; } .home-price-invoice { font-size: 10px; } }
@media (max-width: 640px) { .home-price-card { min-height: 176px; padding: 20px; } .home-price-name { font-size: 20px; } .home-price-amount { font-size: 32px; } }
@media (max-width: 359px) { .home-price-invoice { flex-basis: 100%; margin-left: 32px; } .home-price-name { font-size: 18px; } .home-price-amount { font-size: 28px; } }
@media (prefers-reduced-motion: reduce) {
  .home-price-card, .home-price-name, .home-price-tier, .home-price-mark, .home-price-art img { transition: none; }
  a.home-price-card:is(:hover, :focus-visible, :active), a.home-price-card:is(:hover, :focus-visible, :active) :is(.home-price-mark, .home-price-art img) { transform: none; }
}
</style>
