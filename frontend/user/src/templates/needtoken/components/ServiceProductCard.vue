<template>
  <article class="nt-service-card" :class="{ 'nt-service-card--muted': availability !== 'available' }">
    <div class="nt-service-card__heading">
      <div class="nt-service-card__icon" aria-hidden="true">
        <img v-if="image && !imageErrored" :src="image" alt="" loading="lazy" @error="imageErrored = true" />
        <Zap v-else-if="kind === 'recharge'" :size="22" :stroke-width="1.7" />
        <PackageCheck v-else :size="22" :stroke-width="1.7" />
      </div>
      <span class="nt-service-card__label">{{ categoryLabel || t(`needtoken.catalogServices.${kind}.label`) }}</span>
      <span v-if="availability === 'sold_out'" class="nt-service-card__status">{{ t('products.stockStatus.outOfStock') }}</span>
    </div>

    <h2>{{ getLocalizedText(product.title) }}</h2>
    <p class="nt-service-card__description">{{ description || t(`needtoken.catalogServices.${kind}.descriptionFallback`) }}</p>

    <div v-if="skus.length > 1" class="nt-service-card__options" :aria-label="t('needtoken.catalogServices.chooseOption')">
      <label v-for="sku in skus" :key="sku.id" class="nt-service-card__option" :class="{ 'is-selected': selectedSkuId === sku.id }">
        <input v-model="selectedSkuId" type="radio" :name="`service-sku-${product.id}`" :value="sku.id" />
        <span>{{ skuLabel(sku) }}</span>
      </label>
    </div>
    <p v-else class="nt-service-card__single-option">{{ selectedSku ? skuLabel(selectedSku) : t('needtoken.catalogServices.optionsPending') }}</p>

    <div class="nt-service-card__details">
      <span><Check :size="14" aria-hidden="true" />{{ t(`needtoken.catalogServices.${kind}.delivery`) }}</span>
      <span><ShieldCheck :size="14" aria-hidden="true" />{{ t('needtoken.catalogServices.termsInDetail') }}</span>
    </div>
    <div v-if="product.tags?.length" class="nt-service-card__tags">
      <span v-for="tag in product.tags.slice(0, 3)" :key="tag">{{ tag }}</span>
    </div>

    <div class="nt-service-card__price-row">
      <div>
        <p class="nt-service-card__price">{{ price ? formatServiceMoney(price.amount, siteCurrency, appStore.locale) : t('needtoken.catalogServices.pricePending') }}</p>
        <p v-if="price?.originalAmount" class="nt-service-card__discount">
          <s>{{ formatServiceMoney(price.originalAmount, siteCurrency, appStore.locale) }}</s>
          <span>{{ price.kind === 'member' ? t('products.memberPriceTag') : t('products.promotionTag') }}</span>
        </p>
      </div>
      <RouterLink v-if="availability === 'available' && selectedSku" :to="{ path: `/products/${product.slug}`, query: { sku: String(selectedSku.id) } }" class="nt-service-card__cta">
        {{ t(`needtoken.catalogServices.${kind}.cta`) }}<ArrowUpRight :size="15" aria-hidden="true" />
      </RouterLink>
      <button v-else type="button" class="nt-service-card__cta" disabled>
        {{ availability === 'sold_out' ? t('products.stockStatus.outOfStock') : t('needtoken.catalogServices.availabilityPending') }}
      </button>
    </div>
    <RouterLink :to="`/products/${product.slug}`" class="nt-service-card__read-more">{{ t('needtoken.catalogServices.viewDetails') }}</RouterLink>
  </article>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { ArrowUpRight, Check, PackageCheck, ShieldCheck, Zap } from 'lucide-vue-next'
import { useLocalized } from '../../../composables/useProduct'
import { useAppStore } from '../../../stores/app'
import { getFirstImageUrl } from '../../../utils/image'
import { buildSkuDisplayText } from '../../../utils/sku'
import { formatServiceMoney } from '../lib/displayMoney'
import {
  activeServiceSkus, serviceAvailability, serviceDescriptionText, serviceSkuPrice,
  type ServiceKind, type ServiceProduct, type ServiceSku,
} from '../lib/serviceCatalog'

const props = defineProps<{ product: ServiceProduct; kind: ServiceKind; categoryLabel?: string }>()
const { t } = useI18n()
const appStore = useAppStore()
const { getLocalizedText, siteCurrency } = useLocalized()
const selectedSkuId = ref(0)
const imageErrored = ref(false)
const skus = computed(() => activeServiceSkus(props.product))
const selectedSku = computed(() => skus.value.find(sku => sku.id === selectedSkuId.value) || null)
const availability = computed(() => serviceAvailability(props.product, selectedSku.value))
const price = computed(() => serviceSkuPrice(selectedSku.value))
const image = computed(() => getFirstImageUrl(props.product.images))
const description = computed(() => serviceDescriptionText(getLocalizedText(props.product.description)))
const skuLabel = (sku: ServiceSku) => buildSkuDisplayText({
  skuCode: sku.sku_code === 'default' ? '' : sku.sku_code,
  specValues: sku.spec_values,
  fallback: t('needtoken.catalogServices.standardOption'),
  locale: appStore.locale,
})

watch(() => props.product, () => {
  if (!skus.value.some(sku => sku.id === selectedSkuId.value)) {
    selectedSkuId.value = (skus.value.find(sku => serviceAvailability(props.product, sku) === 'available') || skus.value[0])?.id || 0
  }
  imageErrored.value = false
}, { immediate: true })
</script>

<style scoped>
.nt-service-card { display: flex; min-width: 0; flex-direction: column; position: relative; padding: 22px; border: 1px solid rgba(174, 197, 213, .17); border-radius: 24px; background: linear-gradient(150deg, rgba(255, 255, 255, .045), rgba(255, 255, 255, .014)); box-shadow: inset 0 1px 0 rgba(255, 255, 255, .04); transition: border-color .2s, transform .2s, background .2s; }
.nt-service-card:hover { border-color: rgba(45, 212, 167, .42); background: linear-gradient(150deg, rgba(45, 212, 167, .06), rgba(255, 255, 255, .018)); transform: translateY(-3px); }
.nt-service-card__heading { display: flex; align-items: center; gap: 10px; margin-bottom: 14px; }
.nt-service-card__icon { display: grid; width: 36px; height: 36px; flex-shrink: 0; place-items: center; overflow: hidden; color: var(--ui-accent); border: 1px solid rgba(45, 212, 167, .2); background: rgba(45, 212, 167, .08); border-radius: 11px; }
.nt-service-card__icon img { width: 100%; height: 100%; object-fit: contain; }
.nt-service-card__label { min-width: 0; color: var(--ui-text-muted); font-size: 12px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.nt-service-card__status { margin-left: auto; font-size: 11px; color: var(--ui-text-muted); white-space: nowrap; }
h2 { color: var(--ui-text-primary); font-size: 21px; font-weight: 700; line-height: 1.3; letter-spacing: -.035em; overflow-wrap: anywhere; }
.nt-service-card__description { display: -webkit-box; -webkit-line-clamp: 2; -webkit-box-orient: vertical; min-height: 40px; margin-top: 8px; overflow: hidden; font-size: 12px; line-height: 1.65; color: var(--ui-text-secondary); }
.nt-service-card__options { display: flex; gap: 5px; max-width: 100%; min-height: 34px; margin-top: 14px; overflow-x: auto; scrollbar-width: thin; }
.nt-service-card__option { position: relative; flex-shrink: 0; display: inline-flex; align-items: center; padding: 7px 11px; border: 1px solid var(--ui-border); border-radius: 10px; font-size: 12px; color: var(--ui-text-secondary); cursor: pointer; transition: .15s; }
.nt-service-card__option input { position: absolute; opacity: 0; width: 1px; height: 1px; }
.nt-service-card__option:has(input:focus-visible) { outline: 2px solid var(--ui-accent); outline-offset: 2px; }
.nt-service-card__option.is-selected { color: var(--ui-accent); border-color: rgba(45, 212, 167, .45); background: rgba(45, 212, 167, .09); }
.nt-service-card__single-option { margin-top: 14px; min-height: 34px; display: flex; align-items: center; color: var(--ui-text-muted); font-size: 12px; }
.nt-service-card__price-row { display: flex; flex-wrap: wrap; gap: 12px; align-items: center; justify-content: space-between; margin-top: auto; padding-top: 22px; }
.nt-service-card__price { color: var(--ui-text-primary); font-weight: 750; font-size: clamp(22px, 2vw, 27px); line-height: 1.2; letter-spacing: -.04em; overflow-wrap: anywhere; font-variant-numeric: tabular-nums; }
.nt-service-card__discount { display: flex; align-items: center; gap: 8px; font-size: 11px; margin-top: 8px; color: var(--ui-text-muted); }
.nt-service-card__discount span { color: var(--ui-accent); }
.nt-service-card__details { display: flex; flex-wrap: wrap; gap: 7px 14px; margin-top: 14px; padding-top: 12px; border-top: 1px solid var(--ui-border); color: var(--ui-text-secondary); font-size: 10px; }
.nt-service-card__details > span { display: flex; align-items: center; gap: 6px; }
.nt-service-card__details svg { color: var(--ui-accent); flex-shrink: 0; }
.nt-service-card__tags { display: flex; flex-wrap: wrap; gap: 5px; margin-top: 13px; }
.nt-service-card__tags span { max-width: 100%; overflow-wrap: anywhere; padding: 3px 7px; border-radius: 6px; background: rgba(255,255,255,.04); color: var(--ui-text-muted); font-size: 10px; }
.nt-service-card__cta { display: flex; flex-shrink: 0; align-items: center; justify-content: center; gap: 6px; min-height: 44px; padding: 9px 12px; border: 1px solid rgba(45,212,167,.32); border-radius: 12px; background: var(--ui-accent); color: #04110d; font-size: 12px; font-weight: 650; transition: filter .15s; }
.nt-service-card__cta:hover { filter: brightness(1.08); }
.nt-service-card__cta:focus-visible, .nt-service-card__read-more:focus-visible { outline: 2px solid var(--ui-accent); outline-offset: 4px; }
.nt-service-card__cta:disabled { color: var(--ui-text-muted); background: rgba(255,255,255,.04); border-color: var(--ui-border); cursor: not-allowed; }
.nt-service-card__read-more { margin-top: 10px; text-align: center; color: var(--ui-text-muted); font-size: 11px; }
.nt-service-card__read-more:hover { color: var(--ui-text-primary); }
@media (prefers-reduced-motion: reduce) { .nt-service-card { transition: none; } .nt-service-card:hover { transform: none; } }
@media (min-width: 1024px) {
  .nt-service-card { padding: 18px; border-radius: 20px; }
  .nt-service-card__heading { margin-bottom: 10px; }
  .nt-service-card__icon { width: 28px; height: 28px; border-radius: 9px; }
  .nt-service-card__icon > svg { width: 18px; height: 18px; }
  .nt-service-card__description { min-height: 20px; margin-top: 7px; }
  .nt-service-card__options, .nt-service-card__single-option { min-height: 32px; margin-top: 10px; }
  .nt-service-card__option { padding: 6px 10px; }
  .nt-service-card__details, .nt-service-card__tags { display: none; }
  .nt-service-card__price-row { padding-top: 16px; }
  .nt-service-card__read-more { margin-top: 7px; }
}
@media (max-width: 640px) { .nt-service-card { padding: 22px; } .nt-service-card__price { font-size: 32px; } }
</style>
