<template>
  <article class="nt-service-card" :class="{ 'nt-service-card--muted': availability !== 'available' }" :data-sku-id="selectedSkuId || undefined">
    <div class="nt-service-card__heading">
      <div class="nt-service-card__identity">
        <div class="nt-service-card__icon" aria-hidden="true">
          <img v-if="image && !imageErrored" :src="image" alt="" loading="lazy" @error="imageErrored = true" />
          <Zap v-else-if="kind === 'recharge'" :size="20" :stroke-width="1.7" />
          <PackageCheck v-else :size="20" :stroke-width="1.7" />
        </div>
        <span class="nt-service-card__label">{{ categoryLabel || t(`needtoken.catalogServices.${kind}.label`) }}</span>
      </div>
      <fieldset v-if="skus.length > 1" class="nt-service-card__options">
        <legend class="sr-only">{{ t('needtoken.catalogServices.chooseOption') }}</legend>
        <label v-for="(sku, index) in skus" :key="sku.id" class="nt-service-card__option" :class="{ 'is-selected': selectedSkuId === sku.id }">
          <input v-model="selectedSkuId" type="radio" :name="`service-sku-${product.id}`" :value="sku.id" />
          <span>{{ skuLabel(sku, index) }}</span>
        </label>
      </fieldset>
      <span v-else-if="selectedSku" class="nt-service-card__single-option">{{ skuLabel(selectedSku, 0) }}</span>
    </div>

    <div class="nt-service-card__body" aria-live="polite">
      <h2>{{ presentation.title || getLocalizedText(product.title) }}</h2>
      <p class="nt-service-card__description">{{ presentation.description || description || t(`needtoken.catalogServices.${kind}.descriptionFallback`) }}</p>
      <ul v-if="presentation.features.length" class="nt-service-card__features">
        <li v-for="(feature, index) in presentation.features.slice(0, 3)" :key="index"><Check :size="15" aria-hidden="true" /><span>{{ feature }}</span></li>
      </ul>
      <div v-else class="nt-service-card__details">
        <span><Check :size="14" aria-hidden="true" />{{ t(`needtoken.catalogServices.${kind}.delivery`) }}</span>
        <span><ShieldCheck :size="14" aria-hidden="true" />{{ t('needtoken.catalogServices.termsInDetail') }}</span>
      </div>
      <div v-if="product.tags?.length" class="nt-service-card__tags">
        <span v-for="tag in product.tags.slice(0, 3)" :key="tag">{{ tag }}</span>
      </div>
    </div>

    <div class="nt-service-card__price-row">
      <div class="nt-service-card__price-block">
        <p v-if="isConverted && price" class="nt-service-card__price-note">{{ t('needtoken.variants.convertedPrice') }}</p>
        <p class="nt-service-card__price"><span>{{ price ? format(price.amount) || t('needtoken.catalogServices.pricePending') : t('needtoken.catalogServices.pricePending') }}</span><span v-if="presentation.priceSuffix && price" class="nt-service-card__price-suffix">{{ presentation.priceSuffix }}</span></p>
        <p v-if="price?.originalAmount" class="nt-service-card__discount">
          <s>{{ format(price.originalAmount) }}</s>
          <span>{{ price.kind === 'member' ? t('products.memberPriceTag') : t('products.promotionTag') }}</span>
        </p>
        <p v-if="isConverted && price" class="nt-service-card__reference">{{ t('needtoken.variants.settlementPrice', { price: formatSettlement(price.amount) }) }}</p>
        <p v-if="officialReference" class="nt-service-card__reference">{{ t('needtoken.variants.officialReference', { price: officialReference }) }}</p>
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
import { formatServiceMoney } from '../lib/displayMoney'
import { useDisplayCurrency } from '../composables/useDisplayCurrency'
import { serviceSkuPresentation, serviceVariantLabel } from '../lib/serviceVariants'
import {
  activeServiceSkus, serviceAvailability, serviceDescriptionText, serviceSkuPrice,
  type ServiceKind, type ServiceProduct, type ServiceSku,
} from '../lib/serviceCatalog'

const props = defineProps<{ product: ServiceProduct; kind: ServiceKind; categoryLabel?: string }>()
const { t } = useI18n()
const appStore = useAppStore()
const { getLocalizedText } = useLocalized()
const { format, formatSettlement, isConverted } = useDisplayCurrency()
const selectedSkuId = ref(0)
const imageErrored = ref(false)
const skus = computed(() => activeServiceSkus(props.product))
const selectedSku = computed(() => skus.value.find(sku => sku.id === selectedSkuId.value) || null)
const availability = computed(() => serviceAvailability(props.product, selectedSku.value))
const price = computed(() => serviceSkuPrice(selectedSku.value))
const image = computed(() => getFirstImageUrl(props.product.images))
const description = computed(() => serviceDescriptionText(getLocalizedText(props.product.description)))
const presentation = computed(() => serviceSkuPresentation(selectedSku.value, appStore.locale))
const officialReference = computed(() => presentation.value.officialPriceUsd ? formatServiceMoney(presentation.value.officialPriceUsd, 'USD', appStore.locale) : null)
const skuLabel = (sku: ServiceSku, index: number) => serviceVariantLabel(sku, appStore.locale, skus.value.length > 1
  ? t('needtoken.variants.optionNumber', { number: index + 1 })
  : t('needtoken.catalogServices.standardOption'))

watch(skus, (current) => {
  if (!current.some(sku => sku.id === selectedSkuId.value)) {
    selectedSkuId.value = (current.find(sku => serviceAvailability(props.product, sku) === 'available') || current[0])?.id || 0
  }
}, { immediate: true })
watch(image, () => { imageErrored.value = false })
</script>

<style scoped>
.nt-service-card { display: flex; min-width: 0; max-width: 100%; flex-direction: column; position: relative; padding: 24px; border: 1px solid rgba(174, 197, 213, .17); border-radius: 24px; background: linear-gradient(150deg, rgba(255, 255, 255, .045), rgba(255, 255, 255, .014)); box-shadow: inset 0 1px 0 rgba(255, 255, 255, .04); transition: border-color .2s, transform .2s, background .2s; }
.nt-service-card:hover { border-color: rgba(45, 212, 167, .42); background: linear-gradient(150deg, rgba(45, 212, 167, .06), rgba(255, 255, 255, .018)); transform: translateY(-3px); }
.nt-service-card__heading { display: flex; flex-wrap: wrap; align-items: center; gap: 12px; min-width: 0; margin-bottom: 22px; }
.nt-service-card__identity { display: flex; flex: 1 1 90px; align-items: center; gap: 9px; min-width: 0; }
.nt-service-card__icon { display: grid; width: 34px; height: 34px; flex-shrink: 0; place-items: center; overflow: hidden; color: var(--ui-accent); border: 1px solid rgba(45, 212, 167, .2); background: rgba(45, 212, 167, .08); border-radius: 10px; }
.nt-service-card__icon img { width: 100%; height: 100%; object-fit: contain; }
.nt-service-card__label { min-width: 0; color: var(--ui-text-muted); font-size: 11px; overflow-wrap: anywhere; line-height: 1.5; }
.nt-service-card__options { display: flex; flex: 0 1 auto; flex-wrap: wrap; gap: 3px; min-inline-size: 0; max-width: 100%; margin: 0 0 0 auto; padding: 4px; border: 1px solid var(--ui-border); border-radius: 15px; background: rgba(255, 255, 255, .04); }
.nt-service-card__option { position: relative; display: inline-flex; align-items: center; justify-content: center; min-width: 0; max-width: 100%; padding: 7px 10px; border: 1px solid transparent; border-radius: 11px; color: var(--ui-text-muted); font-size: 12px; line-height: 1.35; text-align: center; cursor: pointer; transition: color .15s, background .15s, border-color .15s; }
.nt-service-card__option > span { min-width: 0; overflow-wrap: anywhere; }
.nt-service-card__option input { position: absolute; opacity: 0; width: 1px; height: 1px; }
.nt-service-card__option:has(input:focus-visible) { outline: 2px solid var(--ui-accent); outline-offset: 2px; }
.nt-service-card__option.is-selected { color: var(--ui-text-primary); border-color: rgba(45, 212, 167, .26); background: rgba(45, 212, 167, .12); box-shadow: inset 0 1px 0 rgba(255, 255, 255, .05); }
.nt-service-card__single-option { margin-left: auto; min-width: 0; max-width: 100%; padding: 7px 11px; border: 1px solid rgba(45, 212, 167, .18); border-radius: 12px; color: var(--ui-accent); background: rgba(45, 212, 167, .06); font-size: 11px; overflow-wrap: anywhere; }
.nt-service-card__body { min-width: 0; flex: 1; }
h2 { color: var(--ui-text-primary); font-size: 23px; font-weight: 700; line-height: 1.3; letter-spacing: -.035em; overflow-wrap: anywhere; }
.nt-service-card__description { margin-top: 11px; min-height: 42px; font-size: 12px; line-height: 1.75; color: var(--ui-text-secondary); overflow-wrap: anywhere; }
.nt-service-card__features { display: grid; gap: 10px; margin-top: 19px; color: var(--ui-text-secondary); font-size: 12px; }
.nt-service-card__features li { display: flex; align-items: flex-start; gap: 8px; min-width: 0; line-height: 1.7; }
.nt-service-card__features li > svg { flex-shrink: 0; margin-top: 2px; color: var(--ui-accent); }
.nt-service-card__features li > span { min-width: 0; overflow-wrap: anywhere; }
.nt-service-card__details { display: grid; gap: 9px; margin-top: 17px; color: var(--ui-text-secondary); font-size: 11px; }
.nt-service-card__details > span { display: flex; align-items: flex-start; gap: 7px; min-width: 0; line-height: 1.6; overflow-wrap: anywhere; }
.nt-service-card__details svg { flex-shrink: 0; margin-top: 2px; color: var(--ui-accent); }
.nt-service-card__tags { display: flex; flex-wrap: wrap; gap: 5px; margin-top: 15px; }
.nt-service-card__tags span { max-width: 100%; overflow-wrap: anywhere; padding: 3px 7px; border-radius: 6px; background: rgba(255,255,255,.04); color: var(--ui-text-muted); font-size: 10px; }
.nt-service-card__price-row { display: flex; flex-wrap: wrap; gap: 18px 12px; align-items: flex-end; justify-content: space-between; min-width: 0; margin-top: auto; padding-top: 25px; }
.nt-service-card__price-block { flex: 1 1 140px; min-width: 0; max-width: 100%; }
.nt-service-card__price { display: flex; flex-wrap: wrap; align-items: baseline; gap: 4px 6px; min-width: 0; color: var(--ui-text-primary); font-weight: 750; font-size: clamp(28px, 2.4vw, 35px); line-height: 1.2; letter-spacing: -.045em; font-variant-numeric: tabular-nums; }
.nt-service-card__price > span { min-width: 0; overflow-wrap: anywhere; }
.nt-service-card__price .nt-service-card__price-suffix { color: var(--ui-text-muted); font-size: 12px; font-weight: 500; letter-spacing: 0; }
.nt-service-card__price-note { margin-bottom: 5px; color: var(--ui-text-muted); font-size: 10px; }
.nt-service-card__discount { display: flex; flex-wrap: wrap; align-items: center; gap: 5px 8px; margin-top: 8px; color: var(--ui-text-muted); font-size: 11px; overflow-wrap: anywhere; }
.nt-service-card__discount span { color: var(--ui-accent); }
.nt-service-card__reference { margin-top: 8px; color: var(--ui-text-muted); font-size: 10px; line-height: 1.5; overflow-wrap: anywhere; }
.nt-service-card__cta { display: flex; flex-shrink: 0; align-items: center; justify-content: center; gap: 6px; min-height: 44px; max-width: 100%; padding: 10px 13px; border: 1px solid rgba(45,212,167,.32); border-radius: 12px; background: var(--ui-accent); color: #04110d; font-size: 12px; font-weight: 650; text-align: center; transition: filter .15s; }
.nt-service-card__cta:hover { filter: brightness(1.08); }
.nt-service-card__cta:focus-visible, .nt-service-card__read-more:focus-visible { outline: 2px solid var(--ui-accent); outline-offset: 4px; }
.nt-service-card__cta:disabled { color: var(--ui-text-muted); background: rgba(255,255,255,.04); border-color: var(--ui-border); cursor: not-allowed; }
.nt-service-card__read-more { margin-top: 14px; text-align: center; color: var(--ui-text-muted); font-size: 11px; }
.nt-service-card__read-more:hover { color: var(--ui-text-primary); }
@media (min-width: 1024px) { .nt-service-card { padding: 22px; border-radius: 22px; } .nt-service-card__heading { gap: 9px; } .nt-service-card__option { padding: 6px 9px; } h2 { font-size: 21px; } }
@media (prefers-reduced-motion: reduce) { .nt-service-card { transition: none; } .nt-service-card:hover { transform: none; } }
@media (max-width: 640px) { .nt-service-card { padding: 22px; } .nt-service-card__price { font-size: 34px; } }
</style>
