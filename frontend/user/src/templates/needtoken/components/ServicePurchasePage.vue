<template>
  <div class="nt-purchase-page">
    <RouterLink :to="backToListTo" class="purchase-back"><ArrowLeft :size="15" />{{ t('productDetail.backToProducts') }}</RouterLink>

    <div v-if="loading" class="purchase-loading" role="status" :aria-label="t('needtoken.common.loading')">
      <div class="purchase-skeleton title-skeleton"></div>
      <div class="purchase-skeleton panel-skeleton"></div>
    </div>

    <template v-else-if="product">
      <header class="purchase-heading">
        <div class="purchase-eyebrow"><span></span>{{ categoryName || t('needtoken.purchaseServices.eyebrow') }}</div>
        <h1>{{ selectedTitle }}</h1>
        <p v-if="selectedDescription">{{ selectedDescription }}</p>
        <ol class="purchase-steps" :aria-label="t('needtoken.purchaseServices.stepsLabel')">
          <li class="current"><span>1</span>{{ t('needtoken.purchaseServices.stepChoose') }}</li>
          <li><ChevronRight :size="14" aria-hidden="true" /><span>2</span>{{ t('needtoken.purchaseServices.stepPay') }}</li>
          <li><ChevronRight :size="14" aria-hidden="true" /><span>3</span>{{ t('needtoken.purchaseServices.stepDelivery') }}</li>
        </ol>
      </header>

      <div class="purchase-grid">
        <section class="purchase-main">
          <div class="purchase-section-head">
            <div><h2>{{ t('needtoken.purchaseServices.selectPlan') }}</h2><p>{{ t('needtoken.purchaseServices.planHint') }}</p></div>
            <div class="purchase-display-controls"><DisplayCurrencySelector compact /><span class="purchase-count">{{ t('needtoken.purchaseServices.optionCount', { count: activeSkus.length }) }}</span></div>
          </div>

          <p v-if="requestedSkuInvalid" class="purchase-message warning" role="alert"><Info :size="17" />{{ t('needtoken.purchaseServices.expiredSelection') }}</p>

          <div class="purchase-options" role="group" :aria-label="t('productDetail.skuTitle')">
            <button v-for="sku in activeSkus" :key="sku.id" type="button" class="purchase-option"
              :class="{ selected: normalizeSkuId(sku.id) === selectedSkuId, unavailable: !isOptionPurchasable(sku) }"
              :aria-pressed="normalizeSkuId(sku.id) === selectedSkuId" :disabled="!isOptionPurchasable(sku)" @click="selectSku(sku)">
              <div class="option-top"><span>{{ skuDisplayText(sku) }}</span><span class="option-check"><Check v-if="normalizeSkuId(sku.id) === selectedSkuId" :size="13" /></span></div>
              <div class="option-price" :class="{ unconfirmed: !isKnownPurchasePrice(skuCatalogPrice(sku)) }">{{ isKnownPurchasePrice(skuCatalogPrice(sku)) ? formatDisplayPrice(skuCatalogPrice(sku)) : t('needtoken.purchaseServices.priceUnavailable') }}</div>
              <p v-if="skuPresentation(sku).description" class="option-description">{{ skuPresentation(sku).description }}</p>
              <div v-if="hasSkuPromotionPrice(sku) && isKnownPurchasePrice(sku.price_amount)" class="option-original">{{ formatDisplayPrice(sku.price_amount) }}</div>
              <div class="option-stock">{{ optionStockText(sku) }}</div>
            </button>
          </div>
          <p v-if="!activeSkus.length" class="purchase-message" role="status">{{ t('productDetail.stockUnavailable') }}</p>

          <div v-if="selectedPresentation.features.length" class="purchase-variant-features" aria-live="polite"><p v-for="(feature, index) in selectedPresentation.features" :key="index"><Check :size="15" aria-hidden="true" /><span>{{ feature }}</span></p></div>

          <div v-if="hasPromotionRules(product)" class="purchase-promotion">
            <h3>{{ t('products.promotionRulesTitle') }}</h3>
            <ul><li v-for="rule in getPromotionRules(product)" :key="rule.id">{{ formatPromotionRule(rule) }}</li></ul>
          </div>

          <div v-if="selectedSkuWholesaleRules.length" class="purchase-promotion">
            <h3>{{ t('products.wholesaleTag') }}</h3>
            <ul><li v-for="(tier, index) in selectedSkuWholesaleRules" :key="index">{{ formatWholesaleTier(tier) }}</li></ul>
          </div>

          <section class="purchase-guidance">
            <h2><ShieldCheck :size="18" />{{ t('needtoken.purchaseServices.beforeOrder') }}</h2>
            <p>{{ t('needtoken.purchaseServices.orderHint') }}</p>
            <p v-if="product.fulfillment_type === 'upstream'">{{ t('needtoken.purchaseServices.serviceHint') }}</p>
            <p v-else-if="product.fulfillment_type === 'auto'">{{ t('needtoken.purchaseServices.digitalHint') }}</p>
            <div class="guidance-links"><RouterLink to="/notice">{{ t('needtoken.purchaseServices.notices') }}<ArrowUpRight :size="13" /></RouterLink><RouterLink to="/terms">{{ t('footer.terms') }}<ArrowUpRight :size="13" /></RouterLink></div>
          </section>
        </section>

        <aside class="purchase-summary">
          <div class="summary-label">{{ t('needtoken.purchaseServices.orderSummary') }}</div>
          <h2>{{ selectedTitle }}</h2>
          <p class="summary-option">{{ selectedSku ? skuDisplayText(selectedSku) : t('productDetail.skuRequired') }}</p>
          <div class="summary-price" :class="{ unconfirmed: !selectedSku || !isKnownPurchasePrice(currentUnitPrice) }">{{ !selectedSku ? t('productDetail.skuRequired') : isKnownPurchasePrice(currentUnitPrice) ? formatDisplayPrice(currentUnitPrice) : t('needtoken.purchaseServices.priceUnavailable') }}<span>{{ selectedPresentation.priceSuffix || t('needtoken.purchaseServices.unitPrice') }}</span></div>
          <p v-if="isConverted && selectedSku && isKnownPurchasePrice(currentUnitPrice)" class="summary-price-reference">{{ t('needtoken.variants.settlementPrice', { price: formatSettlement(currentUnitPrice) }) }}</p>
          <p v-if="officialReference" class="summary-price-reference">{{ t('needtoken.variants.officialReference', { price: officialReference }) }}</p>
          <div v-if="hasSelectedDiscount" class="summary-original">{{ formatDisplayPrice(selectedSku?.price_amount) }}</div>
          <div v-if="showSelectedSkuMemberBadge" class="summary-badge">{{ t('products.memberPriceTag') }}</div>

          <div class="quantity-row">
            <label for="nt-purchase-quantity">{{ t('productDetail.quantity') }}</label>
            <div class="quantity-control">
              <button type="button" :aria-label="t('needtoken.purchaseServices.decreaseQuantity')" :disabled="quantity <= quantityEffectiveMin" @click="quantity = Math.max(quantityEffectiveMin, quantity - 1)"><Minus :size="14" /></button>
              <input id="nt-purchase-quantity" type="text" inputmode="numeric" :value="quantity" @change="handleQuantityInput($event)" @keydown.enter.prevent="($event.target as HTMLInputElement)?.blur()" />
              <button type="button" :aria-label="t('needtoken.purchaseServices.increaseQuantity')" :disabled="quantityEffectiveLimit !== null && quantity >= quantityEffectiveLimit" @click="quantity += 1"><Plus :size="14" /></button>
            </div>
          </div>
          <p class="summary-stock">{{ selectedSku ? optionStockText(selectedSku) : t('productDetail.skuRequired') }}</p>
          <p class="summary-note">{{ t('needtoken.purchaseServices.priceNote') }}</p>

          <p v-if="cannotPurchaseReason" class="purchase-message warning" role="alert">{{ cannotPurchaseReason }}</p>
          <p v-if="purchaseWarning" class="purchase-message warning" role="alert">{{ purchaseWarning }}</p>
          <button type="button" class="purchase-primary" :disabled="!requiresLogin && !canOrder" @click="submitPurchase">
            {{ requiresLogin ? t('productDetail.loginToBuy') : t('needtoken.purchaseServices.continueOrder') }}<ArrowRight :size="17" />
          </button>
          <button v-if="!requiresLogin" type="button" class="purchase-secondary" :disabled="!canOrder" @click="submitCart"><ShoppingBag :size="15" />{{ t('productDetail.addToCart') }}</button>
          <div class="summary-secure"><LockKeyhole :size="13" />{{ t('needtoken.purchaseServices.secureOrder') }}</div>
        </aside>
      </div>

      <section v-if="getLocalizedText(product.content)" class="purchase-content">
        <div class="purchase-section-head"><div><h2>{{ t('needtoken.purchaseServices.productInfo') }}</h2><p>{{ t('needtoken.purchaseServices.infoHint') }}</p></div></div>
        <div class="prose dark:prose-invert max-w-none theme-prose" v-html="processHtmlForDisplay(getLocalizedText(product.content))"></div>
      </section>

      <section class="purchase-related">
        <div class="purchase-section-head"><div><h2>{{ t('needtoken.purchaseServices.resourcesTitle') }}</h2><p>{{ t('needtoken.purchaseServices.resourcesHint') }}</p></div></div>
        <div class="purchase-resources">
          <section class="resource-group">
            <div class="resource-heading"><h3>{{ t('productDetail.relatedPosts') }}</h3><RouterLink to="/blog">{{ t('needtoken.purchaseServices.allGuides') }}<ArrowUpRight :size="14" /></RouterLink></div>
            <p v-if="!relatedPosts.length" class="resource-state" role="status">{{ t('needtoken.purchaseServices.noProductGuides') }}</p>
            <div v-else class="related-grid"><RouterLink v-for="post in relatedPosts" :key="post.id" :to="`/blog/${post.slug}`" class="related-card">
              <img v-if="post.thumbnail" :src="getImageUrl(post.thumbnail)" :alt="getLocalizedText(post.title)" loading="lazy" />
              <div><h4>{{ getLocalizedText(post.title) }}<ArrowUpRight :size="15" /></h4><p v-if="post.summary">{{ getLocalizedText(post.summary) }}</p><time v-if="post.published_at">{{ formatRelatedPostDate(post.published_at) }}</time></div>
            </RouterLink></div>
          </section>
          <section class="resource-group notice-group">
            <div class="resource-heading"><h3><Bell :size="15" />{{ t('needtoken.purchaseServices.siteNotices') }}</h3><RouterLink to="/notice">{{ t('needtoken.purchaseServices.allNotices') }}<ArrowUpRight :size="14" /></RouterLink></div>
            <p class="notice-scope">{{ t('needtoken.purchaseServices.siteNoticeScope') }}</p>
            <p v-if="siteNoticeState === 'idle' || siteNoticeState === 'loading'" class="resource-state" role="status">{{ t('needtoken.common.loading') }}</p>
            <div v-else-if="siteNoticeState === 'error'" class="resource-state notice-error" role="status"><p>{{ t('needtoken.purchaseServices.noticeLoadFailed') }}</p><button type="button" @click="loadSiteNotices"><RotateCw :size="12" />{{ t('errorBoundary.retry') }}</button></div>
            <p v-else-if="!siteNotices.length" class="resource-state" role="status">{{ t('needtoken.purchaseServices.noSiteNotices') }}</p>
            <div v-else class="notice-list"><RouterLink v-for="post in siteNotices" :key="post.id" :to="`/blog/${post.slug}`" class="notice-card">
              <div><h4>{{ getLocalizedText(post.title) || t('nav.notice') }}</h4><p v-if="post.summary">{{ getLocalizedText(post.summary) }}</p><time v-if="post.published_at">{{ formatRelatedPostDate(post.published_at) }}</time></div><ArrowUpRight :size="14" />
            </RouterLink></div>
          </section>
        </div>
      </section>

      <div class="purchase-mobile-bar" :aria-label="t('needtoken.purchaseServices.orderSummary')">
        <div class="mobile-purchase-info">
          <strong>{{ !selectedSku ? t('productDetail.skuRequired') : isKnownPurchasePrice(currentUnitPrice) ? formatDisplayPrice(currentUnitPrice) : t('needtoken.purchaseServices.priceUnavailable') }}</strong>
          <span v-if="selectedSku">{{ skuDisplayText(selectedSku) }}</span>
        </div>
        <button type="button" class="mobile-purchase-action" :disabled="!requiresLogin && !canOrder" @click="submitPurchase">
          {{ requiresLogin ? t('productDetail.loginToBuy') : t('productDetail.buyNow') }}<ArrowRight :size="16" aria-hidden="true" />
        </button>
      </div>
    </template>

    <div v-else class="purchase-empty" role="status"><CircleHelp :size="34" /><h1>{{ t('productDetail.notFound') }}</h1><button type="button" class="purchase-secondary" @click="loadProduct"><RotateCw :size="15" />{{ t('errorBoundary.retry') }}</button></div>
  </div>
</template>

<script setup lang="ts">
import { computed, onUnmounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import { ArrowLeft, ArrowRight, ArrowUpRight, Bell, Check, ChevronRight, CircleHelp, Info, LockKeyhole, Minus, Plus, RotateCw, ShieldCheck, ShoppingBag } from 'lucide-vue-next'
import { postAPI } from '../../../api/product'
import { useAppStore } from '../../../stores/app'
import { useProductDetail } from '../../../composables/useProductDetail'
import { usePageSeo } from '../../../composables/usePageSeo'
import { getImageUrl } from '../../../utils/image'
import { amountToCents } from '../../../utils/money'
import { processHtmlForDisplay } from '../../../utils/content'
import { isKnownPurchasePrice, resolveRequestedSku } from '../lib/purchaseSelection'
import { formatServiceMoney } from '../lib/displayMoney'
import { serviceDescriptionText } from '../lib/serviceCatalog'
import { serviceSkuPresentation, serviceVariantLabel } from '../lib/serviceVariants'
import { useDisplayCurrency } from '../composables/useDisplayCurrency'
import DisplayCurrencySelector from './DisplayCurrencySelector.vue'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const appStore = useAppStore()
const requestedSkuInvalid = ref(false)
const siteNotices = ref<any[]>([])
const siteNoticeState = ref<'idle' | 'loading' | 'ready' | 'error'>('idle')
let noticeGeneration = 0
let pageActive = true

async function loadSiteNotices() {
  if (!pageActive) return
  const generation = ++noticeGeneration
  siteNoticeState.value = 'loading'
  try {
    // These are explicitly site-wide notices, not product associations. Use
    // the same public endpoint and type contract as the native Notice page.
    const response = await postAPI.list({ type: 'notice', page: 1, page_size: 3 })
    const rows = response.data?.data
    if (!Array.isArray(rows) || rows.some((row) => !row || row.type !== 'notice' || typeof row.slug !== 'string' || !row.slug.trim())) {
      throw new Error('invalid public notice response')
    }
    if (!pageActive || generation !== noticeGeneration) return
    siteNotices.value = rows.slice(0, 3)
    siteNoticeState.value = 'ready'
  } catch {
    if (!pageActive || generation !== noticeGeneration) return
    siteNoticeState.value = 'error'
  }
}

onUnmounted(() => {
  pageActive = false
  noticeGeneration += 1
})

// Keep native purchase/pricing/stock/authentication logic shared with the
// original storefront. This view only changes presentation and link selection.
const {
  getLocalizedText, formatRelatedPostDate,
  hasSkuPromotionPrice, getSkuPromotionPriceAmount, hasPromotionRules, getPromotionRules,
  formatPromotionRule, normalizeSkuId,
  loading, product, relatedPosts, selectedSkuId, quantity, purchaseWarning, activeSkus, selectedSku,
  selectedSkuMemberPrice, hasMemberPrice, hasSelectedSkuWholesalePrice, selectedSkuWholesaleFinalPrice,
  selectedSkuWholesaleRules, selectedSkuPromotionFinalPrice, showSelectedSkuMemberBadge,
  isSkuPurchasable, skuStockText, quantityEffectiveLimit, quantityEffectiveMin, handleQuantityInput,
  requiresLogin, canPurchase, cannotPurchaseReason, categoryName, backToListTo, images,
  addToCart, buyNow, goLogin, loadProduct,
} = useProductDetail({ onLoaded: () => {
  applyRequestedSku()
  void loadSiteNotices()
} })

const { format, formatSettlement, isConverted } = useDisplayCurrency()
const formatDisplayPrice = (amount: unknown) => {
  const result = format(amount)
  return result === null ? t('needtoken.purchaseServices.priceUnavailable') : `${isConverted.value ? '≈ ' : ''}${result}`
}
const skuPresentation = (sku: any) => serviceSkuPresentation(sku, appStore.locale)
const selectedPresentation = computed(() => skuPresentation(selectedSku.value))
const selectedTitle = computed(() => selectedPresentation.value.title || getLocalizedText(product.value?.title))
const selectedDescription = computed(() => selectedPresentation.value.description || serviceDescriptionText(getLocalizedText(product.value?.description)))
const officialReference = computed(() => selectedPresentation.value.officialPriceUsd ? formatServiceMoney(selectedPresentation.value.officialPriceUsd, 'USD', appStore.locale) : null)
const skuDisplayText = (sku: any) => serviceVariantLabel(sku, appStore.locale, t('needtoken.variants.optionNumber', { number: activeSkus.value.findIndex((item: any) => normalizeSkuId(item.id) === normalizeSkuId(sku.id)) + 1 }))
const formatWholesaleTier = (tier: any) => t('products.wholesaleTier', { count: Number(tier?.min_quantity || 0), price: formatDisplayPrice(tier?.unit_price) })

usePageSeo({
  title: () => getLocalizedText(product.value?.title),
  description: () => getLocalizedText(product.value?.seo_meta?.description) || getLocalizedText(product.value?.description),
  image: () => images.value[0],
  canonicalPath: () => `/products/${product.value?.slug || String(route.params.slug || '')}`,
  type: () => 'product',
})

function applyRequestedSku() {
  const requested = resolveRequestedSku(route.query.sku, activeSkus.value)
  requestedSkuInvalid.value = requested.kind === 'invalid'
  if (requested.kind === 'valid') selectedSkuId.value = requested.id
  if (requested.kind === 'invalid') selectedSkuId.value = 0
}

watch(() => route.query.sku, () => {
  if (!loading.value && product.value) applyRequestedSku()
})

const selectSku = (sku: any) => {
  if (!isOptionPurchasable(sku)) return
  selectedSkuId.value = normalizeSkuId(sku.id)
  requestedSkuInvalid.value = false
  void router.replace({ query: { ...route.query, sku: String(selectedSkuId.value) } })
}

// Public sold-out flags remain authoritative even when exact stock is hidden.
const isOptionPurchasable = (sku: any) => isKnownPurchasePrice(sku?.price_amount) && isKnownPurchasePrice(skuCatalogPrice(sku)) && sku?.is_sold_out !== true && sku?.stock_status !== 'out_of_stock' && isSkuPurchasable(sku)
const optionStockText = (sku: any) => sku?.is_sold_out === true ? t('productDetail.skuStockOut') : skuStockText(sku)
const skuCatalogPrice = (sku: any) => hasSkuPromotionPrice(sku) ? getSkuPromotionPriceAmount(sku) : sku.price_amount
const currentUnitPrice = computed(() => {
  if (!selectedSku.value) return null
  if (hasSelectedSkuWholesalePrice.value) return selectedSkuWholesaleFinalPrice.value
  if (hasSkuPromotionPrice(selectedSku.value)) return selectedSkuPromotionFinalPrice.value
  if (hasMemberPrice.value) return selectedSkuMemberPrice.value
  return selectedSku.value.price_amount
})
const hasSelectedDiscount = computed(() => {
  const current = amountToCents(currentUnitPrice.value)
  const original = amountToCents(selectedSku.value?.price_amount)
  return current !== null && original !== null && current < original
})
const canOrder = computed(() => canPurchase.value && !!selectedSku.value && isKnownPurchasePrice(currentUnitPrice.value) && isOptionPurchasable(selectedSku.value) && !requestedSkuInvalid.value)
const submitPurchase = () => {
  if (requiresLogin.value) return goLogin()
  if (canOrder.value) buyNow()
}
const submitCart = () => {
  if (canOrder.value) addToCart()
}
</script>

<style scoped>
.nt-purchase-page { max-width: 1120px; margin: 0 auto; padding: 32px 24px 36px; }
.purchase-back { display: inline-flex; align-items: center; gap: 7px; color: var(--ui-text-muted); font-size: 12px; transition: color .18s; }
.purchase-back:hover { color: var(--ui-accent); }
.purchase-heading { padding: 32px 0 28px; }
.purchase-eyebrow { display: flex; align-items: center; gap: 7px; font-size: 11px; color: var(--ui-accent); letter-spacing: .07em; }
.purchase-eyebrow > span { width: 5px; height: 5px; border-radius: 50%; background: currentColor; }
.purchase-heading h1 { margin-top: 12px; font-size: clamp(27px, 4vw, 39px); font-weight: 700; letter-spacing: -.04em; line-height: 1.2; overflow-wrap: anywhere; }
.purchase-heading > p { margin-top: 12px; font-size: 14px; line-height: 1.8; color: var(--ui-text-secondary); max-width: 750px; }
.purchase-steps { display: flex; flex-wrap: wrap; align-items: center; gap: 17px; margin-top: 23px; font-size: 11px; color: var(--ui-text-muted); }
.purchase-steps li { display: flex; align-items: center; gap: 7px; }
.purchase-steps li > span { display: grid; place-items: center; width: 18px; height: 18px; border: 1px solid var(--ui-border-strong); border-radius: 50%; font-size: 10px; }
.purchase-steps .current { color: var(--ui-accent); }
.purchase-steps .current > span { border-color: var(--ui-accent); background: var(--ui-accent-soft); }
.purchase-grid { display: grid; grid-template-columns: minmax(0, 1fr) 338px; gap: 26px; align-items: start; }
.purchase-main, .purchase-summary, .purchase-content, .purchase-related { border: 1px solid var(--ui-border); border-radius: 20px; background: linear-gradient(140deg, rgba(255,255,255,.025), rgba(255,255,255,.005)), var(--ui-bg-elevated); }
.purchase-main { padding: 25px; min-width: 0; }
.purchase-section-head { display: flex; gap: 14px; justify-content: space-between; align-items: center; margin-bottom: 21px; }
.purchase-section-head h2 { font-size: 17px; font-weight: 600; letter-spacing: -.02em; }
.purchase-section-head p { margin-top: 7px; font-size: 12px; line-height: 1.65; color: var(--ui-text-muted); }
.purchase-count { flex: none; font-size: 10px; color: var(--ui-text-muted); border: 1px solid var(--ui-border); padding: 5px 8px; border-radius: 7px; }
.purchase-display-controls { display: flex; flex-wrap: wrap; justify-content: flex-end; align-items: center; gap: 9px; min-width: 0; }
.option-description { margin-top: 11px; font-size: 12px; line-height: 1.7; font-weight: 400; color: var(--ui-text-secondary); overflow-wrap: anywhere; }
.purchase-variant-features { display: grid; gap: 10px; margin-top: 20px; padding: 17px; border: 1px solid var(--ui-border); border-radius: 12px; background: var(--ui-bg-soft); }
.purchase-variant-features p { display: flex; align-items: flex-start; gap: 8px; font-size: 12px; line-height: 1.75; min-width: 0; overflow-wrap: anywhere; }
.purchase-variant-features svg { flex: none; margin-top: 3px; color: var(--ui-accent); }
.summary-price-reference { margin-top: 9px; color: var(--ui-text-muted); font-size: 11px; line-height: 1.7; overflow-wrap: anywhere; }
.option-top > span:first-child, .summary-option { min-width: 0; overflow-wrap: anywhere; }
.purchase-options { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 12px; }
.purchase-option { padding: 19px; border-radius: 13px; border: 1px solid var(--ui-border-strong); background: rgba(255,255,255,.015); color: var(--ui-text-primary); text-align: left; transition: border-color .18s, background .18s; min-width: 0; }
.purchase-option:hover:not(:disabled) { border-color: var(--ui-accent); background: rgba(45,212,167,.03); }
.purchase-option.selected { border-color: var(--ui-accent); background: linear-gradient(135deg, rgba(45,212,167,.1), rgba(45,212,167,.025)); box-shadow: 0 0 0 1px rgba(45,212,167,.15); }
.purchase-option.unavailable { opacity: .45; cursor: not-allowed; border-style: dashed; }
.option-top { display: flex; justify-content: space-between; align-items: start; gap: 10px; font-size: 14px; line-height: 1.6; font-weight: 600; overflow-wrap: anywhere; }
.option-check { width: 19px; height: 19px; border-radius: 50%; border: 1px solid var(--ui-border-strong); display: grid; place-items: center; flex: none; margin-top: 1px; }
.selected .option-check { border-color: var(--ui-accent); background: var(--ui-accent); color: #071510; }
.option-price { margin-top: 23px; font-size: 23px; line-height: 1.25; font-weight: 650; letter-spacing: -.04em; overflow-wrap: anywhere; }
.option-price.unconfirmed, .summary-price.unconfirmed { font-size: 17px; font-weight: 500; letter-spacing: 0; color: var(--ui-text-muted); }
.option-original { font-size: 11px; color: var(--ui-text-muted); text-decoration: line-through; margin-top: 5px; }
.option-stock { margin-top: 12px; font-size: 10px; color: var(--ui-text-muted); }
.purchase-promotion { margin-top: 21px; padding: 14px 15px; border: 1px solid var(--ui-border); border-radius: 10px; background: var(--ui-bg-soft); }
.purchase-promotion h3 { font-size: 11px; font-weight: 600; color: var(--ui-accent); }
.purchase-promotion ul { margin-top: 7px; font-size: 11px; line-height: 1.9; color: var(--ui-text-secondary); }
.purchase-guidance { padding-top: 24px; margin-top: 25px; border-top: 1px solid var(--ui-border); }
.purchase-guidance h2 { display: flex; gap: 8px; align-items: center; font-size: 13px; font-weight: 600; }
.purchase-guidance h2 svg { color: var(--ui-accent); }
.purchase-guidance p { font-size: 12px; line-height: 1.85; color: var(--ui-text-muted); margin-top: 9px; }
.guidance-links { display: flex; gap: 20px; margin-top: 16px; }
.guidance-links a, .purchase-section-head > a { display: inline-flex; align-items: center; gap: 3px; font-size: 11px; color: var(--ui-accent); }
.purchase-summary { position: sticky; top: 90px; padding: 25px; min-width: 0; }
.summary-label { font-size: 10px; letter-spacing: .1em; color: var(--ui-text-muted); }
.purchase-summary h2 { font-size: 16px; font-weight: 600; margin-top: 14px; overflow-wrap: anywhere; }
.summary-option { font-size: 12px; color: var(--ui-accent); margin-top: 7px; line-height: 1.6; overflow-wrap: anywhere; }
.summary-price { margin-top: 25px; font-size: 31px; line-height: 1.2; font-weight: 650; letter-spacing: -.045em; overflow-wrap: anywhere; }
.summary-price > span { display: block; margin-top: 7px; color: var(--ui-text-muted); font-size: 10px; font-weight: 400; letter-spacing: 0; }
.summary-original { font-size: 12px; color: var(--ui-text-muted); text-decoration: line-through; margin-top: 8px; }
.summary-badge { display: inline-flex; background: var(--ui-accent-soft); color: var(--ui-accent); font-size: 10px; border-radius: 5px; padding: 3px 7px; margin-top: 10px; }
.quantity-row { margin-top: 24px; border-top: 1px solid var(--ui-border); padding-top: 20px; display: flex; align-items: center; justify-content: space-between; gap: 10px; font-size: 12px; }
.quantity-control { display: flex; height: 33px; border: 1px solid var(--ui-border-strong); border-radius: 7px; overflow: hidden; }
.quantity-control button { display: grid; place-items: center; width: 30px; transition: background .18s; }
.quantity-control button:hover:not(:disabled) { background: var(--ui-bg-soft); }
.quantity-control button:disabled { opacity: .25; cursor: not-allowed; }
.quantity-control input { width: 40px; text-align: center; background: transparent; border: 0; border-left: 1px solid var(--ui-border); border-right: 1px solid var(--ui-border); font-size: 12px; }
.summary-stock { margin-top: 9px; font-size: 10px; color: var(--ui-text-muted); text-align: right; }
.summary-note { margin-top: 15px; font-size: 10px; line-height: 1.7; color: var(--ui-text-muted); }
.purchase-primary, .purchase-secondary { width: 100%; min-height: 45px; display: flex; align-items: center; justify-content: center; gap: 8px; border-radius: 9px; font-size: 13px; font-weight: 600; transition: background .18s; }
.purchase-primary { background: var(--ui-accent); color: #04110d; margin-top: 20px; }
.purchase-primary:hover:not(:disabled) { background: var(--ui-accent-hover); }
.purchase-secondary { border: 1px solid var(--ui-border-strong); color: var(--ui-text-secondary); margin-top: 10px; }
.purchase-secondary:hover:not(:disabled) { background: var(--ui-bg-soft); }
.purchase-primary:disabled, .purchase-secondary:disabled { opacity: .4; cursor: not-allowed; }
.summary-secure { display: flex; align-items: center; justify-content: center; gap: 5px; margin-top: 16px; font-size: 10px; color: var(--ui-text-muted); }
.purchase-message { display: flex; gap: 7px; align-items: start; padding: 12px; border-radius: 8px; border: 1px solid var(--ui-border); font-size: 12px; line-height: 1.7; margin: 14px 0; color: var(--ui-text-muted); }
.purchase-message svg { flex: none; margin-top: 2px; }
.purchase-message.warning { color: #f0cd8c; background: rgba(230,180,80,.05); border-color: rgba(230,180,80,.2); }
.purchase-content, .purchase-related { margin-top: 26px; padding: 25px; }
.purchase-content :deep(.prose) { font-size: 14px; line-height: 1.85; }
.purchase-content :deep(img) { max-width: 100%; height: auto; border-radius: 10px; }
.purchase-content :deep(pre) { overflow-x: auto; }
.purchase-content :deep(table) { display: block; overflow-x: auto; }
.purchase-content :deep(a) { overflow-wrap: anywhere; }
.purchase-resources { display: grid; grid-template-columns: minmax(0, 1fr) minmax(0, 1fr); gap: 26px; }
.resource-group { min-width: 0; }
.resource-heading { display: flex; justify-content: space-between; align-items: center; gap: 12px; margin-bottom: 14px; }
.resource-heading h3 { display: flex; align-items: center; gap: 7px; font-size: 12px; font-weight: 600; }
.resource-heading h3 svg { color: var(--ui-accent); }
.resource-heading > a { display: inline-flex; align-items: center; gap: 3px; color: var(--ui-accent); font-size: 10px; flex: none; }
.resource-state { border: 1px solid var(--ui-border); border-radius: 10px; padding: 17px; color: var(--ui-text-muted); font-size: 12px; line-height: 1.8; }
.related-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 11px; }
.notice-scope { color: var(--ui-text-muted); font-size: 11px; line-height: 1.8; margin: -2px 0 12px; }
.notice-list { display: flex; flex-direction: column; gap: 9px; }
.notice-card { border: 1px solid var(--ui-border); border-radius: 10px; background: var(--ui-bg-soft); display: flex; justify-content: space-between; align-items: start; gap: 14px; padding: 14px; transition: border-color .18s; }
.notice-card:hover { border-color: var(--ui-accent); }
.notice-card h4 { color: var(--ui-text-primary); font-size: 12px; line-height: 1.7; font-weight: 500; overflow-wrap: anywhere; }
.notice-card p { font-size: 11px; line-height: 1.7; margin-top: 5px; color: var(--ui-text-muted); }
.notice-card time { display: block; font-size: 10px; color: var(--ui-text-muted); margin-top: 8px; }
.notice-card > svg { flex: none; color: var(--ui-text-muted); margin-top: 3px; }
.notice-error button { display: inline-flex; align-items: center; gap: 5px; font-size: 11px; color: var(--ui-accent); margin-top: 9px; }
.related-card { border-radius: 11px; overflow: hidden; border: 1px solid var(--ui-border); background: var(--ui-bg-soft); transition: border-color .18s; }
.related-card:hover { border-color: var(--ui-accent); }
.related-card > img { width: 100%; height: 100px; object-fit: cover; }
.related-card > div { padding: 15px; }
.related-card h4 { font-size: 13px; line-height: 1.65; font-weight: 600; display: flex; gap: 7px; align-items: start; }
.related-card h4 svg { flex: none; margin-top: 3px; color: var(--ui-text-muted); }
.related-card p { font-size: 11px; color: var(--ui-text-muted); line-height: 1.8; margin-top: 7px; }
.related-card time { display: block; font-size: 10px; color: var(--ui-text-muted); margin-top: 12px; }
.purchase-empty { display: flex; flex-direction: column; align-items: center; padding: 100px 20px; gap: 17px; color: var(--ui-text-muted); text-align: center; }
.purchase-empty h1 { font-size: 19px; color: var(--ui-text-primary); }
.purchase-empty button { max-width: 180px; }
.purchase-skeleton { border-radius: 12px; background: var(--ui-bg-soft); animation: purchase-pulse 1.4s ease-in-out infinite; }
.title-skeleton { width: 55%; height: 40px; margin: 50px 0 36px; }
.panel-skeleton { height: 410px; }
.purchase-mobile-bar { display: none; }
.mobile-purchase-info { min-width: 0; flex: 1; }
.mobile-purchase-info strong { display: block; font-size: 20px; font-weight: 700; color: var(--ui-text-primary); overflow-wrap: anywhere; }
.mobile-purchase-info span { display: block; margin-top: 4px; font-size: 11px; color: var(--ui-text-muted); overflow: hidden; white-space: nowrap; text-overflow: ellipsis; }
.mobile-purchase-action { display: inline-flex; align-items: center; justify-content: center; gap: 8px; min-height: 44px; max-width: 58%; flex: none; padding: 10px 18px; border-radius: 10px; background: var(--ui-accent); color: #04110d; font-size: 13px; font-weight: 650; }
.mobile-purchase-action:disabled { opacity: .4; cursor: not-allowed; }
.mobile-purchase-action:hover:not(:disabled) { background: var(--ui-accent-hover); }
button:focus-visible, a:focus-visible, input:focus-visible { outline: 2px solid var(--ui-accent); outline-offset: 4px; }
@keyframes purchase-pulse { 50% { opacity: .5; } }
@media (max-width: 850px) { .purchase-grid { grid-template-columns: minmax(0, 1fr); gap: 18px; } .purchase-summary { position: static; } .purchase-summary h2 { font-size: 19px; } .summary-price { font-size: 35px; } .summary-price.unconfirmed { font-size: 17px; } .purchase-resources { grid-template-columns: minmax(0, 1fr); gap: 23px; } .notice-group { border-top: 1px solid var(--ui-border); padding-top: 23px; } }
@media (max-width: 850px) { .purchase-mobile-bar { position: fixed; inset: auto 0 0; z-index: 45; display: flex; align-items: center; gap: 16px; padding: 12px 16px calc(12px + env(safe-area-inset-bottom)); border-top: 1px solid var(--ui-border-strong); background: rgba(14, 18, 24, .96); backdrop-filter: blur(18px); } :global(body:has(.purchase-mobile-bar)) { padding-bottom: calc(80px + env(safe-area-inset-bottom)); } }
@media (max-width: 560px) { .nt-purchase-page { padding: 23px 16px 25px; } .purchase-heading { padding-top: 25px; } .purchase-steps { gap: 8px; font-size: 10px; } .purchase-main, .purchase-summary, .purchase-content, .purchase-related { border-radius: 15px; padding: 19px; } .purchase-options { gap: 9px; } .purchase-option { padding: 14px; } .option-top { font-size: 12px; } .option-price { font-size: 19px; } .purchase-count { display: none; } .related-grid { grid-template-columns: minmax(0, 1fr); } .related-card { display: flex; } .related-card > img { width: 90px; height: auto; object-fit: cover; } }
@media (prefers-reduced-motion: reduce) { *, *::before, *::after { animation: none !important; transition: none !important; } }
</style>
