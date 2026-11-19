<template>
  <div class="nt-purchase-page">
    <div class="purchase-topline">
      <button v-if="inlineCheckout" type="button" class="purchase-back" data-purchase-back :disabled="inlineSubmitting || inlineOrderCreated" @click="backToSelection"><ArrowLeft :size="16" aria-hidden="true" />{{ t('needtoken.purchaseFlow.backToPlans') }}</button>
      <RouterLink v-else :to="backToListTo" class="purchase-back"><ArrowLeft :size="15" />{{ t('productDetail.backToProducts') }}</RouterLink>
      <ol v-if="product && !loading" class="purchase-steps" :aria-label="t('needtoken.purchaseServices.stepsLabel')">
        <li :class="{ current: !inlineCheckout }" :aria-current="!inlineCheckout ? 'step' : undefined"><span>1</span>{{ t('needtoken.purchaseServices.stepChoose') }}</li>
        <li :class="{ current: inlineCheckout }" :aria-current="inlineCheckout ? 'step' : undefined"><ChevronRight :size="14" aria-hidden="true" /><span>2</span>{{ t('needtoken.purchaseServices.stepPay') }}</li>
        <li><ChevronRight :size="14" aria-hidden="true" /><span>3</span>{{ t('needtoken.purchaseServices.stepDelivery') }}</li>
      </ol>
    </div>

    <div v-if="loading" class="purchase-loading" role="status" :aria-label="t('needtoken.common.loading')">
      <div class="purchase-skeleton title-skeleton"></div>
      <div class="purchase-skeleton panel-skeleton"></div>
    </div>

    <template v-else-if="product">
      <header class="purchase-heading">
        <div class="purchase-heading-copy">
          <p class="purchase-category">{{ categoryName || t('needtoken.purchaseServices.eyebrow') }}</p>
          <h1>{{ selectedTitle }}</h1>
          <p v-if="selectedDescription" class="purchase-intro">{{ selectedDescription }}</p>
        </div>
        <img v-if="images[0]" class="purchase-product-image" :src="images[0]" :alt="getLocalizedText(product.title)" />
      </header>

      <Transition :name="stepTransitionName">
      <div v-show="!inlineCheckout" class="purchase-grid">
        <section class="purchase-main">
          <div class="purchase-section-head">
            <div><h2>{{ t('needtoken.purchaseServices.selectPlan') }}</h2><p>{{ t('needtoken.purchaseServices.planHint') }}</p></div>
            <div class="purchase-display-controls"><DisplayCurrencySelector compact /><span class="purchase-count">{{ t('needtoken.purchaseServices.optionCount', { count: activeSkus.length }) }}</span></div>
          </div>

          <p v-if="requestedSkuInvalid" class="purchase-message warning" role="alert"><Info :size="17" />{{ t('needtoken.purchaseServices.expiredSelection') }}</p>

          <div id="purchase-plan-options" class="purchase-options" tabindex="-1" role="group" :aria-label="t('productDetail.skuTitle')">
            <button v-for="sku in activeSkus" :key="sku.id" type="button" class="purchase-option"
              :class="{ selected: normalizeSkuId(sku.id) === selectedSkuId, unavailable: !isOptionPurchasable(sku) }"
              :aria-pressed="normalizeSkuId(sku.id) === selectedSkuId" @click="selectSku(sku)">
              <div class="option-top"><span :title="skuDisplayText(sku)">{{ skuDisplayText(sku) }}</span><span class="option-check"><Check v-if="normalizeSkuId(sku.id) === selectedSkuId" :size="13" /></span></div>
              <div class="option-price" :class="{ unconfirmed: !isKnownPurchasePrice(skuCatalogPrice(sku)) }">{{ isKnownPurchasePrice(skuCatalogPrice(sku)) ? formatDisplayPrice(skuCatalogPrice(sku)) : t('needtoken.purchaseServices.priceUnavailable') }}</div>
              <div v-if="hasSkuPromotionPrice(sku) && isKnownPurchasePrice(sku.price_amount)" class="option-original">{{ formatDisplayPrice(sku.price_amount) }}</div>
              <div class="option-stock">{{ optionStockText(sku) }}</div>
            </button>
          </div>
          <p v-if="!activeSkus.length" class="purchase-message" role="status">{{ t('productDetail.stockUnavailable') }}</p>

          <section v-if="selectedSku && (selectedDescription || selectedPresentation.features.length)" class="purchase-selected-details" aria-live="polite" aria-atomic="true">
            <h3>{{ skuDisplayText(selectedSku) }}</h3>
            <p v-if="selectedDescription" class="selected-description">{{ selectedDescription }}</p>
            <ul v-if="selectedPresentation.features.length" class="purchase-variant-features"><li v-for="(feature, index) in selectedPresentation.features" :key="index"><Check :size="14" aria-hidden="true" /><span>{{ feature }}</span></li></ul>
          </section>

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
        </section>

        <aside class="purchase-summary">
          <div class="summary-label">{{ t('needtoken.purchaseServices.orderSummary') }}</div>
          <h2>{{ selectedTitle }}</h2>
          <p class="summary-option">{{ selectedSku ? skuDisplayText(selectedSku) : t('productDetail.skuRequired') }}</p>
          <div class="summary-price" :class="{ unconfirmed: !selectedSku || !isKnownPurchasePrice(currentUnitPrice) }"><span :key="`${selectedSkuId}-${currentUnitPrice ?? ''}`" class="summary-price-value">{{ !selectedSku ? t('productDetail.skuRequired') : isKnownPurchasePrice(currentUnitPrice) ? formatDisplayPrice(currentUnitPrice) : t('needtoken.purchaseServices.priceUnavailable') }}</span><span class="summary-price-unit">{{ selectedPresentation.priceSuffix || t('needtoken.purchaseServices.unitPrice') }}</span></div>
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
          <InvoiceRequestEditor v-if="!inlineCheckout" />
          <p class="summary-note">{{ t('needtoken.purchaseServices.priceNote') }}</p>

          <p v-if="cannotPurchaseReason" class="purchase-message warning" role="alert">{{ cannotPurchaseReason }}</p>
          <p v-if="purchaseWarning" class="purchase-message warning" role="alert">{{ purchaseWarning }}</p>
          <button type="button" class="purchase-primary nt-button-purchase" :disabled="!requiresLogin && !canOrder" @click="submitPurchase">
            {{ requiresLogin ? t('productDetail.loginToBuy') : t('needtoken.purchaseServices.continueOrder') }}<ArrowRight :size="17" />
          </button>
          <button v-if="!requiresLogin" type="button" class="purchase-secondary" :disabled="!canOrder" @click="submitCart"><ShoppingBag :size="15" />{{ t('productDetail.addToCart') }}</button>
          <div class="summary-secure"><LockKeyhole :size="13" />{{ t('needtoken.purchaseServices.secureOrder') }}</div>
        </aside>
      </div>
      </Transition>


      <Transition :name="stepTransitionName">
      <div v-if="checkoutReady" v-show="inlineCheckout" ref="inlineRegion" class="inline-checkout" tabindex="-1" :aria-label="t('needtoken.purchaseServices.stepPay')">
        <p class="inline-step-hint" role="status">{{ t(inlineOrderCreated ? 'needtoken.purchaseFlow.orderCreated' : inlineSubmitting ? 'needtoken.purchaseFlow.submitting' : 'needtoken.purchaseFlow.reviewHint') }}</p>
        <CheckoutPanel embedded :active="inlineCheckout" @pending-change="inlineSubmitting = $event" @order-created="inlineOrderCreated = $event">
          <template #details>
            <div class="inline-plan-review">
              <span>{{ selectedSku ? skuDisplayText(selectedSku) : '' }} · {{ t('productDetail.quantity') }} {{ quantity }}</span>
              <button type="button" data-purchase-edit :disabled="inlineSubmitting || inlineOrderCreated" @click="backToSelection">{{ t('needtoken.purchaseFlow.changePlan') }}<ArrowLeft :size="14" aria-hidden="true" /></button>
            </div>
            <section class="inline-buyer-notice">
              <h2>{{ t('needtoken.purchaseServices.beforeOrder') }}</h2>
              <p>{{ t('needtoken.purchaseServices.orderHint') }}</p>
              <p v-if="product.fulfillment_type === 'upstream'">{{ t('needtoken.purchaseServices.serviceHint') }}</p>
              <p v-else-if="product.fulfillment_type === 'auto'">{{ t('needtoken.purchaseServices.digitalHint') }}</p>
              <ul v-if="selectedPresentation.features.length" class="purchase-variant-features"><li v-for="(feature, index) in selectedPresentation.features" :key="index"><Check :size="14" aria-hidden="true" /><span>{{ feature }}</span></li></ul>
              <div class="guidance-links"><RouterLink to="/notice">{{ t('needtoken.purchaseServices.notices') }}<ArrowUpRight :size="13" /></RouterLink><RouterLink to="/terms">{{ t('footer.terms') }}<ArrowUpRight :size="13" /></RouterLink></div>
            </section>
            <details v-if="getLocalizedText(product.content)" class="purchase-content inline-product-content" :open="!compactCheckout">
              <summary><span>{{ t('needtoken.purchaseServices.productInfo') }}</span><ChevronRight :size="16" aria-hidden="true" /></summary>
              <div class="prose dark:prose-invert max-w-none theme-prose" v-html="processHtmlForDisplay(getLocalizedText(product.content))"></div>
            </details>
          </template>
        </CheckoutPanel>
      </div>
      </Transition>

      <div v-if="!inlineCheckout" class="purchase-mobile-bar" :aria-label="t('needtoken.purchaseServices.orderSummary')">
        <div class="mobile-purchase-info">
          <strong>{{ !selectedSku ? t('productDetail.skuRequired') : isKnownPurchasePrice(currentUnitPrice) ? formatDisplayPrice(currentUnitPrice) : t('needtoken.purchaseServices.priceUnavailable') }}</strong>
          <span v-if="selectedSku">{{ skuDisplayText(selectedSku) }}</span>
        </div>
        <button type="button" class="mobile-purchase-action nt-button-purchase" :disabled="!requiresLogin && !canOrder" @click="submitPurchase">
          {{ requiresLogin ? t('productDetail.loginToBuy') : t('productDetail.buyNow') }}<ArrowRight :size="16" aria-hidden="true" />
        </button>
      </div>
    </template>

    <div v-else class="purchase-empty" role="status"><CircleHelp :size="34" /><h1>{{ t('productDetail.notFound') }}</h1><button type="button" class="purchase-secondary" @click="loadProduct"><RotateCw :size="15" />{{ t('errorBoundary.retry') }}</button></div>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
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
import InvoiceRequestEditor from './InvoiceRequestEditor.vue'
import CheckoutPanel from './CheckoutPanel.vue'
import { useInvoiceRequestStore } from '../../../stores/invoiceRequest'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const appStore = useAppStore()
const requestedSkuInvalid = ref(false)
const inlineCheckout = ref(false)
const stepDirection = ref<'forward' | 'backward'>('forward')
const stepTransitionName = computed(() => stepDirection.value === 'forward' ? 'nt-step-forward' : 'nt-step-backward')
const checkoutReady = ref(false)
const inlineSubmitting = ref(false)
const inlineOrderCreated = ref(false)
const inlineRegion = ref<HTMLElement | null>(null)
const compactCheckoutQuery = window.matchMedia('(max-width: 800px)')
const compactCheckout = ref(compactCheckoutQuery.matches)
const onCheckoutWidthChange = (event: MediaQueryListEvent) => { compactCheckout.value = event.matches }
onMounted(() => compactCheckoutQuery.addEventListener('change', onCheckoutWidthChange))
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
  compactCheckoutQuery.removeEventListener('change', onCheckoutWidthChange)
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
  addToCart, prepareBuyNow, loadProduct,
} = useProductDetail({ onLoaded: () => {
  applyRequestedSku()
  void loadSiteNotices()
  if (route.query.checkout === '1') void nextTick(() => resumeInlineCheckout())
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
  if (loading.value || !product.value) return
  applyRequestedSku()
  // A URL-driven SKU change must update the native draft before checkout can continue.
  if (inlineCheckout.value) void nextTick(() => {
    if (canOrder.value) void showInlineCheckout(false, false)
    else inlineCheckout.value = false
  })
})

const selectSku = (sku: any) => {
  if (inlineSubmitting.value || inlineOrderCreated.value) return
  if (!activeSkus.value.some((item: any) => normalizeSkuId(item.id) === normalizeSkuId(sku.id))) return
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
const invoiceStore = useInvoiceRequestStore()
const stepRoute = (checkout: boolean) => ({
  path: route.path,
  query: { ...route.query, sku: String(selectedSkuId.value), checkout: checkout ? '1' : undefined },
  hash: route.hash,
})
async function showInlineCheckout(updateRoute = true, focus = true) {
  if (inlineSubmitting.value || inlineOrderCreated.value || !canOrder.value || requiresLogin.value) return
  if (!prepareBuyNow()) return
  stepDirection.value = 'forward'
  checkoutReady.value = true
  inlineCheckout.value = true
  if (updateRoute) await router.replace(stepRoute(true))
  if (focus) {
    await nextTick()
    inlineRegion.value?.focus({ preventScroll: true })
    inlineRegion.value?.scrollIntoView({ block: 'start', behavior: window.matchMedia('(prefers-reduced-motion: reduce)').matches ? 'auto' : 'smooth' })
  }
}
function resumeInlineCheckout() {
  if (route.query.checkout === '1' && !inlineCheckout.value && !requiresLogin.value && canOrder.value) void showInlineCheckout(false, false)
}
async function backToSelection() {
  if (inlineSubmitting.value || inlineOrderCreated.value) return
  stepDirection.value = 'backward'
  inlineCheckout.value = false
  await router.replace(stepRoute(false))
  await nextTick()
  document.getElementById('purchase-plan-options')?.querySelector<HTMLButtonElement>('button.selected')?.focus({ preventScroll: true })
}
const submitPurchase = async () => {
  if (inlineSubmitting.value || inlineOrderCreated.value) return
  if (requiresLogin.value) {
    const redirect = router.resolve(stepRoute(true)).fullPath
    await router.push({ path: '/auth/login', query: { redirect } })
    return
  }
  if (!invoiceStore.ready) { invoiceStore.openEditor(); return }
  await showInlineCheckout()
}
watch(() => route.query.checkout, value => {
  if (inlineSubmitting.value || inlineOrderCreated.value) return
  if (value === '1') resumeInlineCheckout()
  else inlineCheckout.value = false
})
const submitCart = () => {
  if (canOrder.value) addToCart()
}
</script>

<style scoped>
.nt-purchase-page { max-width: 1200px; margin: 0 auto; padding: 32px 24px 64px; animation: purchase-in var(--nt-motion-enter, 180ms) var(--nt-motion-ease, ease-out); }
.purchase-topline { display: flex; align-items: center; justify-content: space-between; gap: 24px; min-width: 0; }
.purchase-back { display: inline-flex; align-items: center; gap: 8px; min-height: 32px; color: var(--ui-text-muted); font-size: 13px; transition: color var(--nt-motion-fast, 120ms) var(--nt-motion-ease, ease-out); }
.purchase-back:hover { color: var(--ui-text-primary); }
.purchase-heading { display: flex; align-items: center; justify-content: space-between; gap: 24px; padding: 32px 0; min-width: 0; }
.purchase-heading-copy { min-width: 0; }
.purchase-product-image { transition: transform var(--nt-motion-enter, 180ms) var(--nt-motion-ease, ease-out); width: 112px; height: 112px; flex: none; object-fit: contain; border-radius: 16px; background: var(--ui-bg-elevated); }
.purchase-heading h1 { margin-top: 8px; font-size: clamp(28px, 3.5vw, 40px); font-weight: 500; line-height: 1.25; overflow-wrap: anywhere; color: var(--ui-text-primary); }
.purchase-category { color: var(--ui-text-secondary); font-size: 13px; font-weight: 500; }
.purchase-intro { margin-top: 12px; max-width: 64ch; font-size: 14px; line-height: 1.8; color: var(--ui-text-secondary); overflow-wrap: anywhere; }
.purchase-steps { display: flex; flex-wrap: wrap; justify-content: flex-end; align-items: center; gap: 16px; font-size: 12px; color: var(--ui-text-muted); }
.purchase-steps li { display: flex; align-items: center; gap: 8px; transition: color 200ms var(--nt-motion-ease); }
.purchase-steps li > span { display: grid; place-items: center; width: 20px; height: 20px; border: 1px solid var(--ui-border-strong); border-radius: 50%; font-size: 11px; font-variant-numeric: tabular-nums; transition: border-color 200ms var(--nt-motion-ease), background-color 200ms var(--nt-motion-ease), transform 360ms var(--nt-ease-spring); }
.purchase-steps .current { color: var(--ui-accent); font-weight: 500; }
.purchase-steps .current > span { border-color: var(--ui-accent); background: var(--ui-accent-soft); transform: scale(1.06); }
.purchase-grid { display: grid; grid-template-columns: minmax(0, 1fr) minmax(300px, 360px); gap: 24px; align-items: start; }
.purchase-main, .purchase-summary { border: 1px solid var(--ui-border); border-radius: 20px; background: var(--ui-bg-elevated); box-shadow: var(--nt-card-shadow); }
.purchase-main { padding: 24px; min-width: 0; }
.purchase-section-head { display: flex; gap: 16px; justify-content: space-between; align-items: center; margin-bottom: 24px; }
.purchase-section-head h2 { font-size: 18px; font-weight: 500; }
.purchase-section-head p { margin-top: 8px; font-size: 12px; line-height: 1.7; color: var(--ui-text-muted); }
.purchase-count { flex: none; font-size: 12px; color: var(--ui-text-muted); font-variant-numeric: tabular-nums; }
.purchase-display-controls { display: flex; flex-wrap: wrap; justify-content: flex-end; align-items: center; gap: 12px; min-width: 0; }
.purchase-selected-details { margin-top: 24px; padding: 16px; border: 1px solid var(--ui-border); border-radius: 12px; background: var(--ui-bg-soft); overflow-wrap: anywhere; }
.purchase-selected-details h3 { color: var(--ui-text-primary); font-size: 13px; font-weight: 500; line-height: 1.6; }
.selected-description { margin-top: 8px; font-size: 13px; line-height: 1.8; color: var(--ui-text-secondary); white-space: pre-line; }
.purchase-variant-features { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 8px 16px; margin-top: 12px; }
.purchase-variant-features li { display: flex; align-items: flex-start; gap: 8px; font-size: 12px; line-height: 1.8; min-width: 0; overflow-wrap: anywhere; }
.purchase-variant-features svg { flex: none; margin-top: 4px; color: var(--ui-text-secondary); }
.summary-price-reference { margin-top: 8px; color: var(--ui-text-muted); font-size: 12px; line-height: 1.8; overflow-wrap: anywhere; }
.option-top > span:first-child, .summary-option { min-width: 0; overflow-wrap: anywhere; }
.purchase-options { display: grid; grid-template-columns: repeat(auto-fit, minmax(160px, 1fr)); gap: 12px; }
.purchase-option { padding: 16px; border-radius: 14px; border: 1px solid var(--ui-border); background: transparent; color: var(--ui-text-primary); text-align: left; min-width: 0; transition: border-color var(--nt-motion-enter, 180ms) var(--nt-motion-ease), background-color var(--nt-motion-enter, 180ms) var(--nt-motion-ease), box-shadow var(--nt-motion-enter, 180ms) var(--nt-motion-ease), transform var(--nt-duration-rebound, 480ms) var(--nt-ease-spring); }
@media (hover: hover) {
  .purchase-option:not(.selected):hover { border-color: var(--ui-border-strong); background: var(--nt-surface-subtle); transform: translateY(-2px); }
}
.purchase-option.selected { border-color: var(--ui-accent); background: var(--ui-accent-soft); box-shadow: 0 0 0 1px var(--nt-accent-border); }
.purchase-option:active { transform: scale(0.98); transition-duration: var(--nt-duration-press, 80ms); }
.purchase-option.unavailable { border-style: dashed; }
.purchase-option.unavailable:not(.selected) { color: var(--ui-text-muted); }
.option-top { display: flex; justify-content: space-between; align-items: start; gap: 8px; font-size: 13px; line-height: 1.5; font-weight: 500; overflow-wrap: anywhere; }
.option-top > span:first-child { display: -webkit-box; -webkit-line-clamp: 2; -webkit-box-orient: vertical; overflow: hidden; }
.option-check { width: 20px; height: 20px; border-radius: 50%; border: 1px solid var(--ui-border-strong); display: grid; place-items: center; flex: none; transition: border-color 180ms var(--nt-motion-ease), background-color 180ms var(--nt-motion-ease); }
.selected .option-check { border-color: var(--ui-accent); background: var(--ui-accent); color: var(--ui-text-on-accent); }
.selected .option-check svg { animation: nt-check-pop 280ms var(--nt-ease-spring) backwards; }
@keyframes nt-check-pop { from { opacity: 0; transform: scale(0.45); } to { opacity: 1; transform: scale(1); } }
.option-price { margin-top: 12px; font-size: 24px; line-height: 1.25; font-weight: 600; font-variant-numeric: tabular-nums; overflow-wrap: anywhere; }
.option-price.unconfirmed, .summary-price.unconfirmed { font-size: 16px; font-weight: 400; color: var(--ui-text-muted); }
.option-original { font-size: 12px; color: var(--ui-text-muted); text-decoration: line-through; margin-top: 4px; font-variant-numeric: tabular-nums; }
.option-stock { margin-top: 8px; font-size: 12px; line-height: 1.6; color: var(--ui-text-secondary); }
.purchase-option.selected .option-stock { color: var(--ui-text-secondary); }
.purchase-promotion { margin-top: 24px; padding: 16px; border: 1px solid var(--ui-border); border-radius: 12px; background: var(--ui-bg-soft); }
.purchase-promotion h3 { font-size: 12px; font-weight: 500; color: var(--ui-text-primary); }
.purchase-promotion ul { margin-top: 8px; font-size: 12px; line-height: 1.8; color: var(--ui-text-secondary); }
.purchase-guidance { padding-top: 24px; margin-top: 24px; border-top: 1px solid var(--ui-border); }
.purchase-guidance h2 { display: flex; gap: 8px; align-items: center; font-size: 16px; font-weight: 500; }
.purchase-guidance h2 svg { color: var(--ui-text-secondary); }
.purchase-guidance p { font-size: 13px; line-height: 1.8; color: var(--ui-text-secondary); margin-top: 12px; }
.guidance-links { display: flex; flex-wrap: wrap; gap: 12px 24px; margin-top: 16px; }
.guidance-links a, .purchase-section-head > a { display: inline-flex; align-items: center; gap: 4px; font-size: 12px; color: var(--ui-accent); }
.purchase-summary { position: sticky; top: 88px; padding: 24px; min-width: 0; }
.summary-label { font-size: 12px; color: var(--ui-text-muted); }
.purchase-summary h2 { font-size: 20px; font-weight: 500; line-height: 1.5; margin-top: 8px; overflow-wrap: anywhere; }
.summary-option { font-size: 13px; color: var(--ui-text-secondary); margin-top: 8px; line-height: 1.6; overflow-wrap: anywhere; }
.summary-price { margin-top: 24px; font-size: 40px; line-height: 1.2; font-weight: 600; font-variant-numeric: tabular-nums; overflow-wrap: anywhere; }
.summary-price-value { display: inline-block; animation: nt-summary-tick 200ms var(--nt-ease-spring-soft) backwards; }
@keyframes nt-summary-tick { from { opacity: 0.45; transform: translateY(4px); } to { opacity: 1; transform: translateY(0); } }
.summary-price > .summary-price-unit { margin-left: 8px; color: var(--ui-text-muted); font-size: 13px; font-weight: 400; }
.summary-original { font-size: 12px; color: var(--ui-text-muted); text-decoration: line-through; margin-top: 8px; font-variant-numeric: tabular-nums; }
.summary-badge { display: inline-flex; background: var(--ui-accent-soft); color: var(--ui-accent); font-size: 12px; border-radius: 4px; padding: 4px 8px; margin-top: 12px; }
.quantity-row { margin-top: 24px; border-top: 1px solid var(--ui-border); padding-top: 24px; display: flex; align-items: center; justify-content: space-between; gap: 12px; font-size: 13px; }
.quantity-control { display: flex; min-height: 44px; border: 1px solid var(--ui-border-strong); border-radius: 8px; transition: border-color var(--nt-motion-fast, 120ms) var(--nt-motion-ease, ease-out); }
.quantity-control:focus-within { border-color: var(--ui-accent); }
.quantity-control button { display: grid; place-items: center; width: 40px; border-radius: 7px; transition: background-color var(--nt-motion-fast, 120ms) var(--nt-motion-ease, ease-out); }
.quantity-control button:hover:not(:disabled) { background: var(--ui-bg-soft); }
.quantity-control button:active:not(:disabled) { background: var(--ui-bg-muted); }
.quantity-control button:disabled { color: var(--ui-text-muted); opacity: .5; cursor: not-allowed; }
.quantity-control input { width: 40px; text-align: center; background: transparent; border: 0; border-left: 1px solid var(--ui-border); border-right: 1px solid var(--ui-border); font-size: 13px; font-variant-numeric: tabular-nums; }
.summary-stock { margin-top: 8px; font-size: 12px; color: var(--ui-text-muted); text-align: right; }
.summary-note { margin-top: 16px; font-size: 12px; line-height: 1.8; color: var(--ui-text-muted); }
.purchase-primary, .purchase-secondary { width: 100%; min-height: 48px; display: flex; align-items: center; justify-content: center; gap: 8px; border-radius: 10px; font-size: 14px; font-weight: 500; transition: background-color var(--nt-motion-fast, 120ms) var(--nt-motion-ease, ease-out), border-color var(--nt-motion-fast, 120ms) var(--nt-motion-ease, ease-out), color var(--nt-motion-fast, 120ms) var(--nt-motion-ease, ease-out), transform var(--nt-duration-rebound, 480ms) var(--nt-ease-spring); }
.purchase-primary { margin-top: 16px; }
.purchase-secondary { border: 1px solid var(--ui-border-strong); color: var(--ui-text-secondary); margin-top: 12px; }
.purchase-secondary:hover:not(:disabled) { background: var(--ui-bg-soft); color: var(--ui-text-primary); }
.purchase-primary:active:not(:disabled), .purchase-secondary:active:not(:disabled) { transform: scale(0.98); transition-duration: var(--nt-duration-press, 80ms); }
.purchase-primary:disabled, .purchase-secondary:disabled { cursor: not-allowed; }
.purchase-secondary:disabled { color: var(--ui-text-muted); background: var(--ui-bg-soft); border-color: var(--ui-border); opacity: .6; }
.summary-secure { display: flex; align-items: center; justify-content: center; gap: 8px; margin-top: 16px; font-size: 12px; color: var(--ui-text-muted); }
.purchase-message { display: flex; gap: 8px; align-items: start; padding: 12px; border-radius: 8px; border: 1px solid var(--ui-border); font-size: 12px; line-height: 1.8; margin: 16px 0; color: var(--ui-text-muted); }
.purchase-message svg { flex: none; margin-top: 4px; }
.purchase-message.warning { color: var(--nt-warning-ink); background: var(--nt-warning-bg); border-color: var(--nt-warning-border); }
.purchase-content, .purchase-related { margin-top: 32px; padding-top: 24px; border-top: 1px solid var(--ui-border); }
.purchase-content :deep(.prose) { font-size: 14px; line-height: 1.8; }
.purchase-content :deep(img) { max-width: 100%; height: auto; border-radius: 8px; }
.purchase-content :deep(pre) { overflow-x: auto; }
.purchase-content :deep(table) { display: block; overflow-x: auto; }
.purchase-content :deep(a) { overflow-wrap: anywhere; }
.purchase-resources { display: grid; grid-template-columns: minmax(0, 1fr) minmax(0, 1fr); gap: 24px; }
.resource-group { min-width: 0; }
.resource-heading { display: flex; justify-content: space-between; align-items: center; gap: 12px; margin-bottom: 16px; }
.resource-heading h3 { display: flex; align-items: center; gap: 8px; font-size: 13px; font-weight: 500; }
.resource-heading h3 svg { color: var(--ui-text-secondary); }
.resource-heading > a { display: inline-flex; align-items: center; gap: 4px; color: var(--ui-accent); font-size: 12px; flex: none; }
.resource-state { border: 1px solid var(--ui-border); border-radius: 8px; padding: 16px; color: var(--ui-text-muted); font-size: 12px; line-height: 1.8; }
.related-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 12px; }
.notice-scope { color: var(--ui-text-muted); font-size: 12px; line-height: 1.8; margin-bottom: 12px; }
.notice-list { display: flex; flex-direction: column; gap: 12px; }
.notice-card { border: 1px solid var(--ui-border); border-radius: 8px; background: transparent; display: flex; justify-content: space-between; align-items: start; gap: 16px; padding: 16px; transition: border-color var(--nt-motion-fast, 120ms) var(--nt-motion-ease, ease-out), background-color var(--nt-motion-fast, 120ms) var(--nt-motion-ease, ease-out); }
.notice-card:hover { border-color: var(--ui-border-strong); background: var(--ui-bg-soft); }
.notice-card h4 { color: var(--ui-text-primary); font-size: 13px; line-height: 1.7; font-weight: 500; overflow-wrap: anywhere; }
.notice-card p { font-size: 12px; line-height: 1.7; margin-top: 8px; color: var(--ui-text-muted); }
.notice-card time { display: block; font-size: 12px; color: var(--ui-text-muted); margin-top: 8px; }
.notice-card > svg { flex: none; color: var(--ui-text-muted); margin-top: 4px; }
.notice-error button { display: inline-flex; align-items: center; gap: 4px; min-height: 32px; font-size: 12px; color: var(--ui-accent); margin-top: 8px; }
.related-card { border-radius: 8px; overflow: hidden; border: 1px solid var(--ui-border); background: transparent; transition: border-color var(--nt-motion-fast, 120ms) var(--nt-motion-ease, ease-out), background-color var(--nt-motion-fast, 120ms) var(--nt-motion-ease, ease-out); }
.related-card:hover { border-color: var(--ui-border-strong); background: var(--ui-bg-soft); }
.related-card > img { width: 100%; height: 100px; object-fit: cover; }
.related-card > div { padding: 16px; }
.related-card h4 { font-size: 14px; line-height: 1.7; font-weight: 500; display: flex; gap: 8px; align-items: start; }
.related-card h4 svg { flex: none; margin-top: 4px; color: var(--ui-text-muted); }
.related-card p { font-size: 12px; color: var(--ui-text-muted); line-height: 1.8; margin-top: 8px; }
.related-card time { display: block; font-size: 12px; color: var(--ui-text-muted); margin-top: 12px; }
.purchase-empty { display: flex; flex-direction: column; align-items: center; padding: 96px 24px; gap: 16px; color: var(--ui-text-muted); text-align: center; }
.purchase-empty h1 { font-size: 20px; font-weight: 500; color: var(--ui-text-primary); }
.purchase-empty button { max-width: 192px; }
.purchase-skeleton { border-radius: 12px; background: var(--ui-bg-soft); animation: purchase-pulse 1.4s ease-in-out infinite; }
.title-skeleton { width: 55%; height: 40px; margin: 48px 0 32px; }
.panel-skeleton { height: 408px; }
.purchase-mobile-bar { display: none; }
.mobile-purchase-info { min-width: 0; flex: 1; }
.mobile-purchase-info strong { display: block; font-size: 20px; font-weight: 600; color: var(--ui-text-primary); overflow-wrap: anywhere; font-variant-numeric: tabular-nums; }
.mobile-purchase-info span { display: block; margin-top: 4px; font-size: 12px; color: var(--ui-text-muted); overflow: hidden; white-space: nowrap; text-overflow: ellipsis; }
.mobile-purchase-action { display: inline-flex; align-items: center; justify-content: center; gap: 8px; min-height: 44px; max-width: 58%; flex: none; padding: 12px 16px; border-radius: 8px; font-size: 14px; font-weight: 500; }
.mobile-purchase-action:disabled { cursor: not-allowed; }
.nt-purchase-page :deep(svg) { stroke-width: 1.5; }
button:focus-visible, a:focus-visible, input:focus-visible { outline: 2px solid var(--ui-accent); outline-offset: 3px; }
@keyframes purchase-pulse { 50% { opacity: .6; } }
@keyframes purchase-in { from { opacity: 0; transform: translateY(4px); } to { opacity: 1; transform: none; } }
@media (max-width: 1050px) and (min-width: 851px) {
  .purchase-grid { grid-template-columns: minmax(0, 1fr) 312px; gap: 20px; }
  .purchase-main, .purchase-summary { padding: 20px; }
  .purchase-options { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .purchase-section-head { align-items: flex-start; flex-direction: column; }
}
@media (max-width: 850px) {
  .purchase-grid { grid-template-columns: minmax(0, 1fr); gap: 20px; }
  .purchase-summary { position: static; }
  .purchase-resources { grid-template-columns: minmax(0, 1fr); gap: 24px; }
  .notice-group { border-top: 1px solid var(--ui-border); padding-top: 24px; }
}
@media (max-width: 560px) {
  .nt-purchase-page { padding: 24px 16px 40px; }
  .purchase-topline { align-items: flex-start; flex-direction: column; gap: 16px; }
  .purchase-back { font-size: 13px; }
  .purchase-heading { padding: 24px 0; align-items: flex-start; gap: 16px; }
  .purchase-heading h1 { font-size: 28px; }
  .purchase-intro { font-size: 13px; }
  .purchase-product-image { width: 64px; height: 64px; border-radius: 12px; }
  .purchase-steps { gap: 8px; font-size: 11px; }
  .purchase-steps li { gap: 4px; }
  .purchase-steps li > svg { display: none; }
  .purchase-main, .purchase-summary { padding: 20px; }
  .purchase-section-head { align-items: flex-start; flex-direction: column; gap: 12px; }
  .purchase-options { grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 12px; }
  .purchase-option { padding: 16px; }
  .option-top { font-size: 12px; }
  .option-price { font-size: 24px; }
  .purchase-variant-features { grid-template-columns: minmax(0, 1fr); }
  .purchase-count { display: none; }
  .summary-price { font-size: 36px; }
  .related-grid { grid-template-columns: minmax(0, 1fr); }
  .related-card { display: flex; }
  .related-card > img { width: 96px; height: auto; object-fit: cover; }
}
@media (max-width: 360px) {
  .purchase-options { grid-template-columns: minmax(0, 1fr); }
  .purchase-product-image { width: 48px; height: 48px; }
}
.nt-step-forward-enter-active,
.nt-step-backward-enter-active {
  transition: opacity 220ms var(--nt-ease-emphasized), transform 280ms var(--nt-ease-spring-soft);
}
.nt-step-forward-leave-active,
.nt-step-backward-leave-active {
  display: none;
}
.nt-step-forward-enter-from {
  opacity: 0;
  transform: translateX(18px);
}
.nt-step-backward-enter-from {
  opacity: 0;
  transform: translateX(-18px);
}
.purchase-back:disabled { opacity: .5; cursor: not-allowed; }
.purchase-heading:hover .purchase-product-image { transform: translateY(-2px); }
.inline-checkout { scroll-margin-top: 88px; }
.inline-checkout:focus { outline: none; }
.inline-step-hint { margin-bottom: 20px; color: var(--ui-text-muted); font-size: 13px; line-height: 1.7; }
.inline-plan-review { display: flex; align-items: center; justify-content: space-between; flex-wrap: wrap; gap: 12px; padding-bottom: 20px; margin-bottom: 24px; border-bottom: 1px solid var(--ui-border); color: var(--ui-text-secondary); font-size: 13px; line-height: 1.6; }
.inline-plan-review button { display: inline-flex; align-items: center; gap: 8px; min-height: 32px; color: var(--ui-accent); }
.inline-plan-review button:disabled { opacity: .5; cursor: not-allowed; }
.inline-buyer-notice h2 { font-size: 18px; font-weight: 500; }
.inline-buyer-notice > p { margin-top: 12px; color: var(--ui-text-secondary); font-size: 13px; line-height: 1.8; }
.inline-product-content > summary { display: flex; align-items: center; justify-content: space-between; gap: 16px; min-height: 44px; color: var(--ui-text-secondary); cursor: pointer; list-style: none; font-size: 14px; }
.inline-product-content > summary::-webkit-details-marker { display: none; }
.inline-product-content > summary:hover { color: var(--ui-text-primary); }
.inline-product-content > summary > svg { flex: none; transition: transform 440ms var(--nt-ease-spring); }
.inline-product-content[open] > summary { margin-bottom: 16px; }
.inline-product-content[open] > summary > svg { transform: rotate(90deg); }
@media (prefers-reduced-motion: reduce) {
  .nt-purchase-page, .summary-price-value, .selected .option-check svg, .purchase-skeleton { animation: none; }
  .purchase-option, .purchase-primary, .purchase-secondary, .purchase-steps li, .purchase-steps li > span, .purchase-product-image, .inline-product-content > summary > svg, .nt-step-forward-enter-active, .nt-step-backward-enter-active { transition: none; }
  .nt-step-forward-enter-from, .nt-step-backward-enter-from, .purchase-option:hover, .purchase-option:active, .purchase-primary:active, .purchase-secondary:active, .purchase-heading:hover .purchase-product-image, .purchase-steps .current > span { transform: none; }
}

</style>
