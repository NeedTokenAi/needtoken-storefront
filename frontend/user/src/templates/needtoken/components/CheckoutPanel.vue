<template>
  <div class="nt-checkout" :class="{ 'is-embedded': embedded }">
    <header v-if="!embedded" class="nt-checkout-head">
      <div><h1>{{ t('checkout.title') }}</h1><p>{{ t('checkout.subtitle') }}</p></div>
      <CheckoutSteps current-step="checkout" :step-keys="isBuyNowMode ? ['checkout', 'payment'] : ['cart', 'checkout', 'payment']" />
    </header>

    <section v-if="paymentTarget" class="nt-checkout-handoff" role="status" aria-live="polite" :aria-busy="submitting">
      <LoaderCircle v-if="submitting" :size="28" class="nt-checkout-spinner" aria-hidden="true" /><LockKeyhole v-else :size="28" aria-hidden="true" />
      <h2>{{ t('needtoken.purchaseFlow.orderCreated') }}</h2>
      <p>{{ t(paymentNavigationFailed ? 'needtoken.purchaseFlow.paymentRouteFailed' : 'needtoken.purchaseFlow.openingPayment') }}</p>
      <Button v-if="paymentNavigationFailed" type="button" class="nt-button-purchase" :disabled="submitting" @click="retryPaymentNavigation">{{ t('needtoken.purchaseFlow.continuePayment') }}<ArrowRight :size="16" aria-hidden="true" /></Button>
    </section>
    <EmptyState v-else-if="cartItems.length === 0" icon="cart" :title="t('checkout.empty')" :action-label="t('checkout.emptyAction')" action-to="/products" />

    <fieldset v-else class="nt-checkout-grid" :disabled="submitting" :aria-busy="submitting" :aria-label="t('checkout.submitTitle')">
      <section class="nt-checkout-detail" :aria-label="t('checkout.itemsTitle')">
        <h2 v-if="!embedded">{{ t('checkout.itemsTitle') }}</h2>
        <slot name="details" />
        <article v-for="item in embedded ? [] : cartItems" :key="cartItemKey(item)" class="nt-checkout-item" :class="{ 'is-warning': itemStockExceeded(item) }">
          <div class="nt-checkout-item-image"><SmartImage :src="checkoutItemImage(item)" :alt="getLocalizedText(item.title)" img-class="h-full w-full object-contain" /></div>
          <div class="nt-checkout-item-copy">
            <router-link :to="`/products/${item.slug}`">{{ getLocalizedText(item.title) }}</router-link>
            <p v-if="itemSkuDisplay(item)">{{ t('checkout.skuLabel') }}：{{ itemSkuDisplay(item) }}</p>
            <p>{{ t('checkout.quantityLabel') }}：{{ item.quantity }}</p>
            <p v-if="itemStockHint(item)" class="nt-checkout-stock">{{ itemStockHint(item) }}</p>
            <p class="nt-checkout-item-price">{{ checkoutItemPriceParts(item).integer }}{{ checkoutItemPriceParts(item).decimal }} <small>{{ checkoutItemCurrency }}</small></p>
          </div>
        </article>

        <div class="nt-checkout-manual">
          <CheckoutManualForm :manual-form-products="manualFormProducts" v-model="manualFormData" :submit-attempted="submitAttempted" :get-manual-field-label="getManualFieldLabel" :get-manual-field-placeholder="getManualFieldPlaceholder" :manual-field-error="manualFieldError" />
        </div>

        <section v-if="!embedded" class="nt-checkout-guidance">
          <h2><ShieldCheck :size="19" aria-hidden="true" />{{ t('needtoken.purchaseServices.beforeOrder') }}</h2>
          <p>{{ t('needtoken.purchaseServices.orderHint') }}</p>
          <div class="nt-checkout-guidance-links">
            <router-link to="/notice">{{ t('needtoken.purchaseServices.notices') }}<ArrowUpRight :size="14" aria-hidden="true" /></router-link>
            <router-link to="/terms">{{ t('footer.terms') }}<ArrowUpRight :size="14" aria-hidden="true" /></router-link>
          </div>
        </section>
      </section>

      <aside class="nt-checkout-order" :aria-label="t('checkout.submitTitle')">
        <div class="nt-checkout-total" :class="{ 'is-refreshing': previewLoading || couponRefreshing }" :aria-busy="previewLoading || couponRefreshing">
          <h2>{{ t('checkout.previewTotal') }}</h2>
          <p>{{ formatPrice(previewTotal, previewCurrency) }}</p>
        </div>
        <dl class="nt-checkout-breakdown">
          <div><dt>{{ t('cart.itemsCount') }}</dt><dd>{{ totalItems }}</dd></div>
          <div><dt>{{ t('checkout.previewOriginal') }}</dt><dd>{{ formatPrice(previewOriginal, previewCurrency) }}</dd></div>
          <template v-if="!isResellerTenant">
            <div v-if="Number(previewCoupon) > 0" class="is-discount"><dt>{{ t('checkout.previewCoupon') }}</dt><dd>{{ formatDiscountPrice(previewCoupon, previewCurrency) }}</dd></div>
            <div v-if="Number(previewPromotion) > 0" class="is-discount"><dt>{{ t('checkout.previewPromotion') }}</dt><dd>{{ formatDiscountPrice(previewPromotion, previewCurrency) }}</dd></div>
            <div v-if="Number(previewWholesale) > 0" class="is-discount"><dt>{{ t('checkout.previewWholesale') }}</dt><dd>{{ formatDiscountPrice(previewWholesale, previewCurrency) }}</dd></div>
          </template>
          <div v-if="Number(previewMemberDiscount) > 0" class="is-discount"><dt>{{ t('checkout.previewMemberDiscount') }}</dt><dd>-{{ formatPrice(previewMemberDiscount, previewCurrency) }}</dd></div>
        </dl>
        <p v-if="previewLoading || couponRefreshing" class="nt-checkout-status" role="status">{{ previewStatusText }}</p>
        <Alert v-if="checkoutAlert" :variant="pageAlertVariant(checkoutAlert.level)"><AlertDescription>{{ checkoutAlert.message }}</AlertDescription></Alert>

        <section class="nt-checkout-payment" :aria-label="t('checkout.paymentMethod')">
          <h3>{{ t('checkout.paymentMethod') }}</h3>
          <div v-if="showBalanceOption" class="nt-checkout-wallet">
            <div><span>{{ t('payment.walletBalanceLabel') }}</span><strong>{{ walletLoading ? t('common.loading') : formatPrice(walletBalance, previewCurrency) }}</strong></div>
            <label><input v-model="useBalance" type="checkbox" :disabled="walletOnlyPayment" />{{ t('payment.useBalance') }}</label>
            <p v-if="walletOnlyPayment">{{ t('payment.walletOnlyHint') }}</p>
            <p v-if="useBalance">{{ t('payment.walletDeductLabel') }}：{{ expectedWalletPaidDisplay }}</p>
            <p v-if="useBalance && !walletOnlyPayment">{{ t('payment.onlinePayLabel') }}：{{ expectedOnlinePayDisplay }}</p>
            <p v-if="walletOnlyPayment && expectedOnlinePayCents > 0">{{ t('payment.walletInsufficientHint') }}</p>
          </div>
          <div v-if="!walletOnlyPayment && requiresOnlineChannel && paymentChannels.length" class="nt-checkout-channels">
            <button v-for="channel in paymentChannels" :key="channel.id" type="button" :disabled="isChannelDisabledForAmount(channel)" :aria-pressed="selectedChannelId === channel.id && !isChannelDisabledForAmount(channel)" :title="isChannelDisabledForAmount(channel) ? channelAmountLimitHint(channel) : ''" :class="{ selected: selectedChannelId === channel.id && !isChannelDisabledForAmount(channel) }" @click="handleSelectChannel(channel)">
              <span class="nt-checkout-channel-name"><img v-if="channel.icon" :src="getImageUrl(channel.icon)" alt="" /><span>{{ channel.name }}</span><Check v-if="selectedChannelId === channel.id && !isChannelDisabledForAmount(channel)" :size="15" class="nt-checkout-channel-check" aria-hidden="true" /></span>
              <small v-if="channel.fee_policy === 'customer_surcharge'">{{ t('payment.feeLabel') }} {{ formatChannelFeeRate(channel) }} · {{ t('payment.fixedFeeLabel') }} {{ formatChannelFixedFee(channel) }}</small>
              <small v-if="isChannelDisabledForAmount(channel)">{{ channelAmountLimitHint(channel) }}</small>
            </button>
          </div>
          <p v-else-if="!walletOnlyPayment && requiresOnlineChannel" class="nt-checkout-status">{{ t('checkout.noPaymentChannels') }}</p>
          <p v-if="!requiresOnlineChannel" class="nt-checkout-status">{{ t('checkout.walletCoversAll') }}</p>
        </section>

        <section v-if="!userAuthStore.isAuthenticated" class="nt-checkout-guest" :aria-label="t('checkout.modeTitle')">
          <h3>{{ t('checkout.modeTitle') }}</h3>
          <div class="nt-checkout-modes">
            <Button type="button" :variant="checkoutMode === 'guest' ? 'default' : 'secondary'" :aria-pressed="checkoutMode === 'guest'" @click="checkoutMode = 'guest'">{{ t('checkout.guestPurchase') }}</Button>
            <Button as-child variant="secondary"><router-link :to="memberLoginTarget">{{ t('checkout.memberPurchase') }}</router-link></Button>
          </div>
          <div v-if="checkoutMode === 'guest'" class="nt-checkout-guest-fields">
            <label for="nt-checkout-email">{{ t('checkout.guestEmailPlaceholder') }}<Input id="nt-checkout-email" v-model="guestEmail" type="email" autocomplete="email" :aria-invalid="!!guestEmail && !guestEmailValid" :placeholder="t('checkout.guestEmailPlaceholder')" /></label>
            <p v-if="guestEmail && !guestEmailValid" class="nt-checkout-field-error" role="alert">{{ t('error.email_invalid') }}</p>
            <label for="nt-checkout-password">{{ t('checkout.guestPasswordPlaceholder') }}<Input id="nt-checkout-password" v-model="guestPassword" type="password" autocomplete="new-password" :placeholder="t('checkout.guestPasswordPlaceholder')" /></label>
            <p>{{ t('checkout.guestInstructions.password') }}</p>
          </div>
          <div v-if="checkoutMode === 'guest' && guestCaptchaEnabled" class="nt-checkout-captcha">
            <p>{{ t('auth.common.captchaLabel') }}</p>
            <ImageCaptcha v-if="captchaProvider === 'image'" ref="guestImageCaptchaRef" v-model="guestCaptchaPayload" :disabled="submitting" @config-stale="handleGuestCaptchaConfigStale" />
            <TurnstileCaptcha v-else-if="captchaProvider === 'turnstile'" ref="guestTurnstileRef" v-model="guestTurnstileToken" :site-key="guestTurnstileSiteKey" />
          </div>
        </section>

        <label v-if="!isResellerTenant" class="nt-checkout-field">{{ t('checkout.couponTitle') }}<Input v-model="couponCode" type="text" :placeholder="t('checkout.couponPlaceholder')" /></label>
        <InvoiceRequestEditor v-if="active" :email="guestEmail || userAuthStore.user?.email" :disabled="submitting" />
        <Button size="lg" class="nt-button-purchase nt-checkout-submit w-full" :disabled="!canSubmit" :aria-busy="submitting" @click="handleSubmit">{{ submitting ? t('checkout.submitting') : t('checkout.submitButton') }}<LoaderCircle v-if="submitting" :size="18" class="nt-checkout-spinner" aria-hidden="true" /><ArrowRight v-else :size="18" aria-hidden="true" /></Button>
        <p class="nt-checkout-secure"><LockKeyhole :size="13" aria-hidden="true" />{{ t('needtoken.purchaseServices.secureOrder') }}</p>
        <p class="nt-checkout-submit-hint">{{ t('checkout.submitHint') }}</p>
      </aside>
    </fieldset>
  </div>
</template>

<script setup lang="ts">
import { computed, watch } from 'vue'
import { onBeforeRouteLeave, onBeforeRouteUpdate, useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { ArrowRight, ArrowUpRight, Check, LoaderCircle, LockKeyhole, ShieldCheck } from 'lucide-vue-next'
import { pageAlertVariant } from '../../../utils/alerts'
import ImageCaptcha from '../../../components/captcha/ImageCaptcha.vue'
import TurnstileCaptcha from '../../../components/captcha/TurnstileCaptcha.vue'
import CheckoutManualForm from '../../../components/checkout/CheckoutManualForm.vue'
import EmptyState from '../../../components/EmptyState.vue'
import SmartImage from '../../../components/SmartImage.vue'
import CheckoutSteps from '../../../components/checkout/CheckoutSteps.vue'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { useCheckout } from '../../../composables/useCheckout'
import InvoiceRequestEditor from './InvoiceRequestEditor.vue'

const props = withDefaults(defineProps<{ embedded?: boolean; active?: boolean }>(), { embedded: false, active: true })
const emit = defineEmits<{ pendingChange: [pending: boolean]; orderCreated: [created: boolean] }>()
const route = useRoute()
const { t } = useI18n()
const memberLoginTarget = computed(() => ({ path: '/auth/login', query: { redirect: route.fullPath } }))
const {
  userAuthStore, getLocalizedText, formatPrice, getImageUrl,
  isBuyNowMode, cartItems, totalItems, cartItemKey, checkoutItemImage, itemSkuDisplay,
  itemStockExceeded, itemStockHint,
  checkoutItemCurrency, checkoutItemPriceParts,
  manualFormProducts, manualFormData, submitAttempted, getManualFieldLabel, getManualFieldPlaceholder, manualFieldError,
  couponCode, isResellerTenant,
  checkoutMode, guestEmail, guestPassword, guestEmailValid,
  guestCaptchaEnabled, captchaProvider, guestCaptchaPayload, guestTurnstileToken, guestTurnstileSiteKey,
  guestImageCaptchaRef, guestTurnstileRef, handleGuestCaptchaConfigStale,
  previewCurrency, previewOriginal, previewCoupon, previewPromotion, previewWholesale, previewMemberDiscount, previewTotal,
  previewLoading, couponRefreshing, previewStatusText, formatDiscountPrice, checkoutAlert,
  showBalanceOption, walletLoading, walletBalance, useBalance, walletOnlyPayment,
  expectedWalletPaidDisplay, expectedOnlinePayDisplay, expectedOnlinePayCents,
  requiresOnlineChannel, paymentChannels, selectedChannelId, isChannelDisabledForAmount, channelAmountLimitHint,
  handleSelectChannel, formatChannelFeeRate, formatChannelFixedFee,
  submitting, paymentTarget, paymentNavigationFailed, retryPaymentNavigation, canSubmit, handleSubmit,
} = useCheckout({ source: props.embedded ? 'buynow' : undefined })
watch(submitting, value => emit('pendingChange', value), { immediate: true, flush: 'sync' })
watch(paymentTarget, value => emit('orderCreated', !!value), { immediate: true, flush: 'sync' })
onBeforeRouteUpdate(() => !props.embedded || (!submitting.value && !paymentTarget.value))
onBeforeRouteLeave(to => !props.embedded || !submitting.value || to.path === '/pay')

void guestImageCaptchaRef
void guestTurnstileRef
</script>

<style scoped>
.nt-checkout-handoff { display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 20px; min-height: 320px; padding: 40px 24px; border: 1px solid var(--ui-border); border-radius: 20px; background: var(--ui-bg-elevated); text-align: center; }
.nt-checkout-handoff > svg { color: var(--ui-accent); }
.nt-checkout-handoff h2 { font-size: 24px; font-weight: 500; }
.nt-checkout-handoff > p { max-width: 52ch; color: var(--ui-text-secondary); font-size: 14px; line-height: 1.8; }

.nt-checkout.is-embedded { width: 100%; padding: 0; }
.is-embedded .nt-checkout-grid { grid-template-columns: minmax(0, 1fr) minmax(320px, 400px); }
@media (max-width: 800px) { .is-embedded .nt-checkout-grid { grid-template-columns: 1fr; } }
.nt-checkout { width: min(1200px, 100%); margin: 0 auto; padding: 32px 24px 64px; animation: checkout-in var(--nt-motion-enter, 180ms) var(--nt-motion-ease, ease-out); }
.nt-checkout-head { display: flex; align-items: center; justify-content: space-between; gap: 24px; margin-bottom: 32px; }
.nt-checkout-head h1 { font-size: 32px; font-weight: 500; line-height: 1.25; }
.nt-checkout-head p { margin-top: 8px; color: var(--ui-text-secondary); font-size: 14px; line-height: 1.7; }
.nt-checkout-grid { display: grid; min-width: 0; margin: 0; padding: 0; border: 0; grid-template-columns: minmax(0, 1fr) minmax(320px, 400px); gap: 24px; align-items: start; }
.nt-checkout-detail, .nt-checkout-order { min-width: 0; padding: 24px; border: 1px solid var(--ui-border); border-radius: 20px; background: var(--ui-bg-elevated); box-shadow: var(--nt-card-shadow); }
.nt-checkout-order { position: sticky; top: 88px; }
.nt-checkout-detail h2 { font-size: 18px; font-weight: 500; }
.nt-checkout-order h3 { font-size: 14px; font-weight: 500; }
.nt-checkout-item { display: flex; align-items: flex-start; gap: 16px; padding: 24px 0; border-bottom: 1px solid var(--ui-border); }
.nt-checkout-item-image { width: 80px; height: 80px; flex: none; overflow: hidden; border-radius: 12px; background: var(--ui-bg-soft); }
.nt-checkout-item-copy { min-width: 0; flex: 1; }
.nt-checkout-item a { font-size: 18px; font-weight: 500; line-height: 1.5; overflow-wrap: anywhere; transition: color var(--nt-motion-fast, 120ms) var(--nt-motion-ease, ease-out); }
.nt-checkout-item a:hover { color: var(--ui-accent); }
.nt-checkout-item p { margin-top: 8px; color: var(--ui-text-secondary); font-size: 13px; line-height: 1.6; overflow-wrap: anywhere; }
.nt-checkout-item.is-warning .nt-checkout-stock { color: var(--nt-warning-ink); }
.nt-checkout-item .nt-checkout-item-price { margin-top: 12px; color: var(--ui-text-primary); font-size: 24px; font-weight: 600; font-variant-numeric: tabular-nums; }
.nt-checkout-item-price small { color: var(--ui-text-muted); font-size: 12px; font-weight: 400; }
.nt-checkout-manual:has(> *) { margin-top: 24px; }
.nt-checkout-guidance { padding-top: 24px; }
.nt-checkout-guidance h2 { display: flex; align-items: center; gap: 8px; font-size: 16px; font-weight: 500; }
.nt-checkout-guidance h2 svg { color: var(--ui-text-secondary); }
.nt-checkout-guidance p { margin-top: 12px; color: var(--ui-text-secondary); font-size: 13px; line-height: 1.8; }
.nt-checkout-guidance-links { display: flex; flex-wrap: wrap; gap: 12px 24px; margin-top: 16px; }
.nt-checkout-guidance-links a { display: inline-flex; align-items: center; gap: 4px; min-height: 32px; color: var(--ui-accent); font-size: 12px; }
.nt-checkout-guidance-links a:hover { text-decoration: underline; text-underline-offset: 4px; }
.nt-checkout-total h2 { color: var(--ui-text-secondary); font-size: 13px; font-weight: 400; }
.nt-checkout-total > p { margin-top: 12px; font-size: clamp(32px, 3vw, 40px); line-height: 1.2; font-weight: 600; font-variant-numeric: tabular-nums; overflow-wrap: anywhere; transition: opacity var(--nt-motion-fast, 120ms) var(--nt-motion-ease, ease-out); }
.nt-checkout-total.is-refreshing > p { opacity: .55; }
.nt-checkout-breakdown { display: grid; gap: 12px; margin: 24px 0; padding-bottom: 24px; border-bottom: 1px solid var(--ui-border); font-size: 13px; color: var(--ui-text-secondary); }
.nt-checkout-breakdown div { display: flex; justify-content: space-between; gap: 16px; }
.nt-checkout-breakdown dt { min-width: 0; }
.nt-checkout-breakdown dd { flex: none; color: var(--ui-text-primary); text-align: right; font-variant-numeric: tabular-nums; }
.nt-checkout-breakdown .is-discount dd { color: var(--ui-accent); }
.nt-checkout-field, .nt-checkout-guest { display: grid; gap: 12px; margin-top: 24px; font-size: 13px; }
.nt-checkout-modes { display: flex; flex-wrap: wrap; gap: 8px; }
.nt-checkout-modes :deep(button), .nt-checkout-modes :deep(a) { min-height: 40px; border-radius: 10px; font-size: 13px; font-weight: 500; box-shadow: none; transition: background-color var(--nt-motion-fast, 120ms) var(--nt-motion-ease, ease-out), color var(--nt-motion-fast, 120ms) var(--nt-motion-ease, ease-out), transform var(--nt-duration-rebound, 480ms) var(--nt-ease-spring); }
.nt-checkout-modes :deep(button:active:not(:disabled)), .nt-checkout-modes :deep(a:active) { transform: scale(0.97); transition-duration: var(--nt-duration-press, 80ms); }
.nt-checkout-guest-fields { display: grid; gap: 16px; margin-top: 4px; animation: nt-panel-reveal 220ms var(--nt-ease-emphasized) backwards; }
@keyframes nt-panel-reveal { from { opacity: 0; transform: translateY(6px); } to { opacity: 1; transform: translateY(0); } }
.nt-checkout-guest-fields label { display: grid; gap: 8px; font-size: 13px; color: var(--ui-text-secondary); }
.nt-checkout-guest-fields > p, .nt-checkout-captcha > p, .nt-checkout-status { color: var(--ui-text-muted); font-size: 12px; line-height: 1.8; }
.nt-checkout-guest-fields > p { margin-top: -8px; }
.nt-checkout-guest-fields > .nt-checkout-field-error { color: var(--nt-warning-ink); }
.nt-checkout-captcha > p { margin-bottom: 8px; }
.nt-checkout-wallet { display: grid; gap: 12px; margin-top: 12px; padding: 16px; border: 1px solid var(--ui-border); border-radius: 12px; font-size: 13px; }
.nt-checkout-wallet > div { display: flex; justify-content: space-between; gap: 12px; }
.nt-checkout-wallet strong { font-weight: 500; font-variant-numeric: tabular-nums; }
.nt-checkout-wallet label { display: flex; align-items: center; gap: 8px; min-height: 32px; cursor: pointer; }
.nt-checkout-wallet input { accent-color: var(--ui-accent); width: 16px; height: 16px; }
.nt-checkout-wallet p { color: var(--ui-text-muted); font-size: 12px; line-height: 1.8; }
.nt-checkout-channels { display: grid; grid-template-columns: repeat(auto-fit, minmax(136px, 1fr)); gap: 12px; margin-top: 12px; }
.nt-checkout-channels button { display: grid; align-content: center; gap: 8px; min-width: 0; min-height: 56px; padding: 12px; border: 1px solid var(--ui-border); border-radius: 12px; background: transparent; text-align: left; font-size: 13px; transition: border-color var(--nt-motion-enter, 180ms) var(--nt-motion-ease), background-color var(--nt-motion-enter, 180ms) var(--nt-motion-ease), color var(--nt-motion-fast, 120ms) var(--nt-motion-ease), box-shadow var(--nt-motion-enter, 180ms) var(--nt-motion-ease), transform var(--nt-duration-rebound, 480ms) var(--nt-ease-spring); }
.nt-checkout-channel-name { display: flex; align-items: center; gap: 8px; }
.nt-checkout-channel-name > span { min-width: 0; overflow-wrap: anywhere; }
@media (hover: hover) {
  .nt-checkout-channels button:hover:not(:disabled):not(.selected) { border-color: var(--ui-border-strong); background: var(--ui-bg-soft); transform: translateY(-1px); }
}
.nt-checkout-channels button.selected { border-color: var(--ui-accent); background: var(--ui-accent-soft); box-shadow: 0 0 0 1px var(--nt-accent-border); }
.nt-checkout-channels button:active:not(:disabled) { transform: scale(0.975); transition-duration: var(--nt-duration-press, 80ms); }
.nt-checkout-channels button:disabled { color: var(--ui-text-muted); background: var(--ui-bg-soft); opacity: .6; cursor: not-allowed; }
.nt-checkout-channel-check { margin-left: auto; color: var(--ui-accent); flex: none; animation: nt-checkout-check-pop 280ms var(--nt-ease-spring) backwards; }
@keyframes nt-checkout-check-pop { from { opacity: 0; transform: scale(0.5); } to { opacity: 1; transform: scale(1); } }
.nt-checkout-channels img { width: 24px; height: 24px; object-fit: contain; flex: none; }
.nt-checkout-channels small { color: var(--ui-text-muted); font-size: 12px; line-height: 1.6; }
.nt-checkout-status { margin: 12px 0; }
.nt-checkout-order :deep(input:not([type=checkbox])) { min-height: 44px; border-radius: 8px; font-size: 14px; box-shadow: none; transition: border-color var(--nt-motion-fast, 120ms) var(--nt-motion-ease, ease-out), background-color var(--nt-motion-fast, 120ms) var(--nt-motion-ease, ease-out), color var(--nt-motion-fast, 120ms) var(--nt-motion-ease, ease-out); }
.nt-checkout-order :deep(input:not([type=checkbox]):hover:not(:disabled):not(:focus-visible)) { border-color: var(--ui-border-strong); }
.nt-checkout-order :deep(input[aria-invalid="true"]) { border-color: var(--nt-warning-ink); }
.nt-checkout-order :deep(input:disabled) { color: var(--ui-text-muted); cursor: not-allowed; }
.nt-checkout-order :deep(.nt-invoice-editor) { margin-top: 24px; padding-top: 24px; }
.nt-checkout-order :deep(.nt-checkout-submit) { display: flex; align-items: center; justify-content: center; gap: 12px; min-height: 48px; margin-top: 24px; border-radius: 8px; font-size: 14px; transition: background-color var(--nt-motion-fast, 120ms) var(--nt-motion-ease, ease-out), color var(--nt-motion-fast, 120ms) var(--nt-motion-ease, ease-out), transform var(--nt-motion-fast, 120ms) var(--nt-motion-ease, ease-out); }
.nt-checkout-order :deep(.nt-checkout-submit[aria-busy="true"]) { cursor: progress; }
.nt-checkout-spinner { animation: checkout-spin .8s linear infinite; }
.nt-checkout-secure { display: flex; align-items: center; justify-content: center; gap: 8px; margin-top: 16px; color: var(--ui-text-muted); font-size: 12px; }
.nt-checkout-submit-hint { margin-top: 12px; color: var(--ui-text-muted); text-align: center; font-size: 12px; line-height: 1.8; }
.nt-checkout :deep(svg) { stroke-width: 1.5; }
.nt-checkout :is(button, a, input):focus-visible { outline: 2px solid var(--ui-accent); outline-offset: 3px; }
@keyframes checkout-in { from { opacity: 0; transform: translateY(4px); } to { opacity: 1; transform: none; } }
@keyframes checkout-spin { to { transform: rotate(360deg); } }
@media (max-width: 1000px) and (min-width: 801px) {
  .nt-checkout-grid { grid-template-columns: minmax(0, 1fr) 336px; gap: 20px; }
  .nt-checkout-detail, .nt-checkout-order { padding: 20px; }
}
@media (max-width: 800px) {
  .nt-checkout { padding: 24px 20px 48px; }
  .nt-checkout-head { align-items: flex-start; flex-direction: column; gap: 20px; }
  .nt-checkout-grid { grid-template-columns: minmax(0, 1fr); gap: 20px; }
  .nt-checkout-order { position: static; }
  .nt-checkout-total > p { font-size: 36px; }
}
@media (max-width: 480px) {
  .nt-checkout { padding-inline: 16px; }
  .nt-checkout-head h1 { font-size: 28px; }
  .nt-checkout-detail, .nt-checkout-order { padding: 20px; }
  .nt-checkout-item { gap: 16px; }
  .nt-checkout-item-image { width: 56px; height: 56px; border-radius: 8px; }
  .nt-checkout-item a { font-size: 16px; }
  .nt-checkout-channels { grid-template-columns: minmax(0, 1fr); }
}
@media (prefers-reduced-motion: reduce) {
  .nt-checkout, .nt-checkout-spinner, .nt-checkout-guest-fields, .nt-checkout-channel-check { animation: none; }
  .nt-checkout :deep(button), .nt-checkout :deep(a), .nt-checkout :deep(input), .nt-checkout-total > p, .nt-checkout-channels button { transition: none; }
  .nt-checkout :deep(button:active), .nt-checkout :deep(a:active), .nt-checkout-channels button:hover, .nt-checkout-channels button:active { transform: none; }
}
</style>
