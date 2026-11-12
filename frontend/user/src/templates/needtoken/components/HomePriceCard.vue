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
      </div>
      <span class="home-price-invoice">{{ t('needtoken.catalogServices.invoiceNote') }}</span>
      <p v-if="officialReference" class="home-price-official">{{ t('needtoken.variants.officialReference', { price: officialReference }) }}</p>
      <p class="home-price-amount">
        <span>{{ priceText }}</span>
        <small v-if="presentation.priceSuffix && price">{{ presentation.priceSuffix }}</small>
      </p>
      <p v-if="!available" class="home-price-state">{{ availability === 'sold_out' ? t('products.stockStatus.outOfStock') : t('needtoken.catalogServices.availabilityPending') }}</p>
    </div>
    <div v-if="image && !imageFailed" class="home-price-art" aria-hidden="true">
      <img :src="image" alt="" @error="imageFailed = true" />
    </div>
  </component>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { PackageCheck, Zap } from 'lucide-vue-next'
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
const image = computed(() => getFirstImageUrl(props.product.images))
const target = computed(() => ({ path: `/products/${props.product.slug}`, query: { sku: String(sku.value?.id || 0) } }))
</script>

<style scoped>
.home-price-card { display: flex; align-items: stretch; justify-content: space-between; gap: 12px; min-width: 250px; width: 280px; min-height: 148px; padding: 16px; border: 1px solid rgba(174, 197, 213, .18); border-radius: 18px; background: linear-gradient(180deg, rgba(255,255,255,.07), rgba(255,255,255,.025)); color: var(--ui-text-primary); scroll-snap-align: start; transition: border-color .18s, transform .18s; }
a.home-price-card:hover { border-color: rgba(45, 212, 167, .5); transform: translateY(-2px); }
.home-price-card.is-unavailable { opacity: .72; }
.home-price-copy { min-width: 0; display: flex; flex-direction: column; align-items: flex-start; }
.home-price-topline { display: flex; align-items: center; gap: 8px; min-width: 0; max-width: 100%; }
.home-price-mark { display: grid; width: 28px; height: 28px; flex: none; place-items: center; overflow: hidden; border-radius: 8px; background: rgba(45, 212, 167, .1); color: var(--ui-accent); }
.home-price-mark img { width: 100%; height: 100%; object-fit: cover; }
.home-price-name { min-width: 0; font-size: 15px; font-weight: 700; line-height: 1.35; letter-spacing: -.02em; overflow-wrap: anywhere; }
.home-price-invoice { margin-top: 8px; padding: 2px 7px; border-radius: 999px; background: rgba(255,255,255,.06); color: var(--ui-text-muted); font-size: 11px; }
.home-price-official { margin-top: 10px; color: var(--ui-text-muted); font-size: 12px; }
.home-price-amount { display: flex; flex-wrap: wrap; align-items: baseline; gap: 4px; margin-top: auto; padding-top: 12px; font-size: 26px; font-weight: 750; letter-spacing: -.04em; line-height: 1.1; }
.home-price-amount small { color: var(--ui-text-muted); font-size: 12px; font-weight: 500; letter-spacing: 0; }
.home-price-state { margin-top: 6px; color: var(--ui-text-muted); font-size: 11px; }
.home-price-art { width: 72px; flex: none; display: grid; place-items: center; }
.home-price-art img { width: 72px; height: 72px; object-fit: cover; border-radius: 12px; }
a.home-price-card:focus-visible { outline: 2px solid var(--ui-accent); outline-offset: 3px; }
@media (prefers-reduced-motion: reduce) { .home-price-card { transition: none; } a.home-price-card:hover { transform: none; } }
</style>
