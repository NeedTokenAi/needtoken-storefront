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
      <span v-if="selectedSku && skus.length === 1" class="nt-service-card__single-option">{{ skuLabel(selectedSku, 0) }}</span>
    </div>

    <div class="nt-service-card__body">
      <h2 aria-live="polite">{{ presentation.title || getLocalizedText(product.title) }}</h2>
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
        <p class="nt-service-card__price"><span>{{ price ? format(price.amount) || t('needtoken.catalogServices.pricePending') : t('needtoken.catalogServices.pricePending') }}</span><span v-if="presentation.priceSuffix && price" class="nt-service-card__price-suffix">{{ presentation.priceSuffix }}</span></p>
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
import { selectServiceVariant, serviceSkuPresentation, serviceVariantAxes, serviceVariantLabel, type ServiceVariantAxis } from '../lib/serviceVariants'
import { apiPlanMemberLoginPath } from '../lib/apiPlanCatalog'
import {
  activeServiceSkus, serviceAvailability, serviceDescriptionText, serviceSkuPrice,
  type ServiceKind, type ServiceProduct, type ServiceSku,
} from '../lib/serviceCatalog'

const props = defineProps<{ product: ServiceProduct; kind: ServiceKind; categoryLabel?: string; memberOnly?: boolean; ctaLabel?: string; deliveryText?: string; descriptionFallback?: string }>()
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
  : { path: `/products/${props.product.slug}`, query: { sku: String(selectedSku.value?.id || 0) } })
const availability = computed(() => serviceAvailability(props.product, selectedSku.value))
const price = computed(() => serviceSkuPrice(selectedSku.value))
const image = computed(() => getFirstImageUrl(props.product.images))
const description = computed(() => serviceDescriptionText(getLocalizedText(props.product.description)))
const presentation = computed(() => serviceSkuPresentation(selectedSku.value, appStore.locale))
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
</script>

<style scoped>
.nt-service-card { display: flex; min-width: 0; max-width: 100%; flex-direction: column; position: relative; padding: 18px; border: 1px solid rgba(174, 197, 213, .17); border-radius: 20px; background: linear-gradient(150deg, rgba(255, 255, 255, .045), rgba(255, 255, 255, .014)); box-shadow: inset 0 1px 0 rgba(255, 255, 255, .04); transition: border-color .2s, transform .2s, background .2s; }
.nt-service-card:hover { border-color: rgba(45, 212, 167, .42); background: linear-gradient(150deg, rgba(45, 212, 167, .06), rgba(255, 255, 255, .018)); transform: translateY(-2px); }
.nt-service-card__heading { display: flex; align-items: center; gap: 8px; min-width: 0; margin-bottom: 8px; }
.nt-service-card__identity { display: flex; flex: 1 1 90px; align-items: center; gap: 8px; min-width: 0; }
.nt-service-card__icon { display: grid; width: 28px; height: 28px; flex-shrink: 0; place-items: center; overflow: hidden; color: var(--ui-accent); border: 1px solid rgba(45, 212, 167, .2); background: rgba(45, 212, 167, .08); border-radius: 8px; }
.nt-service-card__icon > svg { width: 17px; height: 17px; }
.nt-service-card__icon img { width: 100%; height: 100%; object-fit: contain; }
.nt-service-card__label { min-width: 0; color: var(--ui-text-muted); font-size: 12px; overflow-wrap: anywhere; line-height: 1.4; }
.nt-service-card__selectors { display: grid; gap: 6px; margin-top: 11px; }
.nt-service-card__axis { display: flex; align-items: center; gap: 8px; min-inline-size: 0; margin: 0; padding: 0; border: 0; }
.nt-service-card__axis-label { flex-shrink: 0; color: var(--ui-text-muted); font-size: 12px; line-height: 1.4; }
.nt-service-card__options { display: flex; flex-wrap: wrap; gap: 3px; min-inline-size: 0; max-width: 100%; margin: 0; padding: 3px; border: 1px solid var(--ui-border); border-radius: 12px; background: rgba(255, 255, 255, .035); }
.nt-service-card__flat-options { margin-top: 11px; width: fit-content; }
.nt-service-card__option { position: relative; display: inline-flex; align-items: center; justify-content: center; min-width: 32px; min-height: 32px; max-width: 100%; padding: 5px 9px; border: 1px solid transparent; border-radius: 8px; color: #b5c1cc; font-size: 12px; line-height: 1.35; text-align: center; cursor: pointer; transition: color .15s, background .15s, border-color .15s; }
.nt-service-card__option > span { min-width: 0; overflow-wrap: anywhere; }
.nt-service-card__option input { position: absolute; opacity: 0; width: 1px; height: 1px; }
.nt-service-card__option:focus-within { outline: 2px solid #8cf1d1; outline-offset: 2px; }
.nt-service-card__option[aria-disabled="true"] { color: var(--ui-text-muted); opacity: .4; cursor: not-allowed; }
.nt-service-card__option:not(.is-selected):not([aria-disabled="true"]):hover { color: #edfff8; border-color: #5caa944d; }
.nt-service-card__option.is-selected { color: #e4fff4; border-color: #72e4bf; background: linear-gradient(150deg, #3a6b5c, #245443); box-shadow: inset 0 1px 0 #baffdf40, 0 0 0 1px #74efc619, 0 2px 10px #2dd4a712; }
.nt-service-card__single-option { min-width: 0; max-width: 48%; padding: 5px 8px; border: 1px solid rgba(45, 212, 167, .18); border-radius: 9px; color: var(--ui-accent); background: rgba(45, 212, 167, .06); font-size: 12px; overflow-wrap: anywhere; }
.nt-service-card__body { min-width: 0; flex: 1; }
h2 { margin: 0; color: var(--ui-text-primary); font-size: 21px; font-weight: 700; line-height: 1.3; letter-spacing: -.035em; overflow-wrap: anywhere; }
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
.nt-service-card__price { display: flex; flex-wrap: wrap; align-items: baseline; gap: 3px 5px; min-width: 0; color: var(--ui-text-primary); font-weight: 750; font-size: clamp(28px, 2.4vw, 34px); line-height: 1.15; letter-spacing: -.045em; font-variant-numeric: tabular-nums; }
.nt-service-card__price > span { min-width: 0; overflow-wrap: anywhere; }
.nt-service-card__price .nt-service-card__price-suffix { color: var(--ui-text-muted); font-size: 12px; font-weight: 500; letter-spacing: 0; }
.nt-service-card__price-note { margin-bottom: 4px; color: var(--ui-text-muted); font-size: 10px; }
.nt-service-card__discount { display: flex; flex-wrap: wrap; align-items: center; gap: 4px 7px; margin-top: 5px; color: var(--ui-text-muted); font-size: 11px; overflow-wrap: anywhere; }
.nt-service-card__discount span { color: var(--ui-accent); }
.nt-service-card__reference { margin-top: 5px; color: var(--ui-text-muted); font-size: 10px; line-height: 1.4; overflow-wrap: anywhere; }
.nt-service-card__cta { display: flex; flex-shrink: 0; align-items: center; justify-content: center; gap: 6px; min-height: 40px; max-width: 100%; padding: 8px 12px; border: 1px solid rgba(45,212,167,.32); border-radius: 11px; background: var(--ui-accent); color: #04110d; font-size: 12px; font-weight: 650; text-align: center; transition: filter .15s; }
.nt-service-card__cta:hover { filter: brightness(1.08); }
.nt-service-card__cta:focus-visible, .nt-service-card__read-more:focus-visible { outline: 2px solid var(--ui-accent); outline-offset: 4px; }
.nt-service-card__cta:disabled { color: var(--ui-text-muted); background: rgba(255,255,255,.04); border-color: var(--ui-border); cursor: not-allowed; }
.nt-service-card__read-more { display: flex; justify-content: center; align-items: center; min-height: 32px; margin-top: 4px; text-align: center; color: var(--ui-text-muted); font-size: 12px; }
.nt-service-card__read-more:hover { color: var(--ui-text-primary); }
@media (forced-colors: active) { .nt-service-card__option.is-selected { color: ButtonText; background: ButtonFace; border-color: Highlight; outline: 2px solid Highlight; outline-offset: 1px; } }
@media (hover: none), (pointer: coarse) { .nt-service-card__option, .nt-service-card__cta, .nt-service-card__read-more { min-height: 44px; } .nt-service-card__option { min-width: 44px; } }
@media (prefers-reduced-motion: reduce) { .nt-service-card { transition: none; } .nt-service-card:hover { transform: none; } }
@media (max-width: 640px) { .nt-service-card { padding: 20px; } h2 { font-size: 23px; } .nt-service-card__description { font-size: 13px; min-height: 43px; } .nt-service-card__price { font-size: 34px; } .nt-service-card__option, .nt-service-card__cta, .nt-service-card__read-more { min-height: 44px; } .nt-service-card__option { min-width: 44px; } }
</style>
