<template>
  <component :is="available ? 'router-link' : 'article'" :to="available ? target : undefined" class="home-price-card" :class="{ 'is-unavailable': !available }">
    <div class="home-price-copy">
      <div class="home-price-topline">
        <span class="home-price-mark" aria-hidden="true">
          <img v-if="image && !imageFailed" :src="image" alt="" @error="imageFailed = true" />
          <Zap v-else-if="kind === 'recharge'" :size="16" />
          <PackageCheck v-else :size="16" />
        </span>
        <span class="home-price-name">{{ title }}</span>
        <span class="home-price-invoice"><ReceiptText :size="13" aria-hidden="true" />{{ t('needtoken.homeLibrary.invoice') }}</span>
      </div>
      <p v-if="officialReference" class="home-price-official">{{ t('needtoken.homeLibrary.officialChip', { price: officialChip }) }}</p>
      <p class="home-price-amount">
        <span>{{ priceText }}</span>
        <small v-if="presentation.priceSuffix && price">/ {{ presentation.priceSuffix }}</small>
      </p>
      <p v-if="!available" class="home-price-state">{{ availability === 'sold_out' ? t('products.stockStatus.outOfStock') : t('needtoken.catalogServices.availabilityPending') }}</p>
    </div>
    <div class="home-price-art" aria-hidden="true">
      <img v-if="image && !imageFailed" :src="image" alt="" @error="imageFailed = true" />
      <span v-else class="home-price-art-fallback"><Zap v-if="kind === 'recharge'" :size="22" /><PackageCheck v-else :size="22" /></span>
    </div>
  </component>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
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
const priceText = computed(() => price.value ? format(price.value.amount) || t('needtoken.catalogServices.pricePending') : t('needtoken.catalogServices.pricePending'))
const officialReference = computed(() => presentation.value.officialPriceUsd ? formatServiceMoney(presentation.value.officialPriceUsd, 'USD', appStore.locale) : '')
const officialChip = computed(() => presentation.value.priceSuffix && officialReference.value ? `${officialReference.value}/${presentation.value.priceSuffix}` : officialReference.value)
const image = computed(() => getFirstImageUrl(props.product.images))
const target = computed(() => ({ path: `/products/${props.product.slug}`, query: { sku: String(sku.value?.id || 0) } }))
</script>

<style scoped>
.home-price-card { position: relative; display: flex; align-items: stretch; min-width: 320px; width: 340px; min-height: 168px; padding: 18px 18px 16px; border: 1px solid rgba(255,255,255,.72); border-radius: 22px; background: linear-gradient(180deg, #fbfcfb, #f3f6f4); color: #172027; scroll-snap-align: start; box-shadow: 0 12px 28px rgba(0,0,0,.18); transition: transform .24s ease, box-shadow .24s ease; }
a.home-price-card:hover { transform: translateY(-6px); box-shadow: 0 20px 36px rgba(0,0,0,.28); }
.home-price-card.is-unavailable { opacity: .72; }
.home-price-copy { min-width: 0; display: flex; flex-direction: column; align-items: flex-start; width: 100%; }
.home-price-topline { display: flex; align-items: center; gap: 8px; width: 100%; min-width: 0; }
.home-price-mark { display: grid; width: 28px; height: 28px; flex: none; place-items: center; overflow: hidden; border-radius: 8px; background: #e7f8f2; color: #0f9f78; }
.home-price-mark img { width: 100%; height: 100%; object-fit: cover; }
.home-price-name { min-width: 0; font-size: 16px; font-weight: 750; line-height: 1.3; letter-spacing: -.02em; overflow-wrap: anywhere; }
.home-price-invoice { display: inline-flex; align-items: center; gap: 4px; margin-left: auto; color: #6b7785; font-size: 12px; white-space: nowrap; }
.home-price-official { margin-top: 14px; padding: 3px 10px; border: 1px solid #d5dee6; border-radius: 999px; color: #5d6b78; font-size: 12px; }
.home-price-amount { display: flex; flex-wrap: wrap; align-items: baseline; gap: 4px; margin-top: auto; padding-top: 18px; padding-right: 88px; color: #12181e; font-size: 28px; font-weight: 750; letter-spacing: -.045em; line-height: 1.05; }
.home-price-amount small { color: #6b7785; font-size: 13px; font-weight: 550; letter-spacing: 0; }
.home-price-state { margin-top: 6px; color: #6b7785; font-size: 11px; }
.home-price-art { position: absolute; right: 16px; bottom: 16px; width: 78px; height: 68px; }
.home-price-art img, .home-price-art-fallback { width: 72px; height: 64px; border-radius: 14px; background: #fff; box-shadow: 0 10px 18px rgba(23, 32, 39, .12); object-fit: cover; transition: transform .28s ease; }
.home-price-art-fallback { display: grid; place-items: center; color: #8aa398; }
a.home-price-card:hover .home-price-art img, a.home-price-card:hover .home-price-art-fallback { transform: translate(-4px, -6px) rotate(-8deg); }
a.home-price-card:focus-visible { outline: 2px solid #2dd4a7; outline-offset: 3px; }
@media (prefers-reduced-motion: reduce) {
  .home-price-card, .home-price-art img, .home-price-art-fallback { transition: none; }
  a.home-price-card:hover, a.home-price-card:hover .home-price-art img, a.home-price-card:hover .home-price-art-fallback { transform: none; }
}
</style>
