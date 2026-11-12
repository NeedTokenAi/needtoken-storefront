<template>
  <div class="nt-checkout">
    <header class="nt-checkout-head">
      <h1>{{ t('checkout.title') }}</h1>
      <CheckoutSteps current-step="checkout" :step-keys="isBuyNowMode ? ['checkout', 'payment'] : ['cart', 'checkout', 'payment']" />
    </header>

    <EmptyState v-if="cartItems.length === 0" icon="cart" :title="t('checkout.empty')" :action-label="t('checkout.emptyAction')" action-to="/products" />

    <div v-else class="nt-checkout-grid">
      <section class="nt-checkout-detail">
        <h2>{{ t('checkout.itemsTitle') }}</h2>
        <article v-for="item in cartItems" :key="cartItemKey(item)" class="nt-checkout-item" :class="{ 'is-warning': itemStockExceeded(item) }">
          <SmartImage :src="checkoutItemImage(item)" :alt="getLocalizedText(item.title)" img-class="h-14 w-14 rounded-lg object-cover" />
          <div>
            <router-link :to="`/products/${item.slug}`">{{ getLocalizedText(item.title) }}</router-link>
            <p>{{ t('checkout.quantityLabel') }}：{{ item.quantity }}</p>
            <p v-if="itemSkuDisplay(item)">{{ t('checkout.skuLabel') }}：{{ itemSkuDisplay(item) }}</p>
            <p v-if="itemStockHint(item)">{{ itemStockHint(item) }}</p>
            <p class="nt-checkout-item-price">{{ checkoutItemPriceParts(item).integer }}{{ checkoutItemPriceParts(item).decimal }} {{ checkoutItemCurrency }}</p>
          </div>
        </article>

        <CheckoutManualForm :manual-form-products="manualFormProducts" v-model="manualFormData" :submit-attempted="submitAttempted" :get-manual-field-label="getManualFieldLabel" :get-manual-field-placeholder="getManualFieldPlaceholder" :manual-field-error="manualFieldError" />

        <label v-if="!isResellerTenant" class="nt-checkout-field">{{ t('checkout.couponTitle') }}<Input v-model="couponCode" type="text" :placeholder="t('checkout.couponPlaceholder')" /></label>

        <div v-if="!userAuthStore.isAuthenticated" class="nt-checkout-guest">
          <h2>{{ t('checkout.modeTitle') }}</h2>
          <div class="nt-checkout-modes">
            <Button :variant="checkoutMode === 'guest' ? 'default' : 'secondary'" @click="checkoutMode = 'guest'">{{ t('checkout.guestPurchase') }}</Button>
            <Button as-child variant="secondary"><router-link to="/auth/login">{{ t('checkout.memberPurchase') }}</router-link></Button>
          </div>
          <div v-if="checkoutMode === 'guest'" class="nt-checkout-guest-fields">
            <Input v-model="guestEmail" type="email" :placeholder="t('checkout.guestEmailPlaceholder')" />
            <Input v-model="guestPassword" type="password" :placeholder="t('checkout.guestPasswordPlaceholder')" />
          </div>
          <div v-if="checkoutMode === 'guest' && guestCaptchaEnabled">
            <p>{{ t('auth.common.captchaLabel') }}</p>
            <ImageCaptcha v-if="captchaProvider === 'image'" ref="guestImageCaptchaRef" v-model="guestCaptchaPayload" :disabled="submitting" @config-stale="handleGuestCaptchaConfigStale" />
            <TurnstileCaptcha v-else-if="captchaProvider === 'turnstile'" ref="guestTurnstileRef" v-model="guestTurnstileToken" :site-key="guestTurnstileSiteKey" />
          </div>
          <p v-if="checkoutMode === 'guest' && guestEmail && !guestEmailValid">{{ t('error.email_invalid') }}</p>
        </div>
      </section>

      <aside class="nt-checkout-order">
        <h2>{{ t('checkout.submitTitle') }}</h2>
        <dl>
          <div><dt>{{ t('cart.itemsCount') }}</dt><dd>{{ totalItems }}</dd></div>
          <div><dt>{{ t('checkout.previewOriginal') }}</dt><dd>{{ formatPrice(previewOriginal, previewCurrency) }}</dd></div>
          <template v-if="!isResellerTenant">
            <div><dt>{{ t('checkout.previewCoupon') }}</dt><dd>{{ formatDiscountPrice(previewCoupon, previewCurrency) }}</dd></div>
            <div><dt>{{ t('checkout.previewPromotion') }}</dt><dd>{{ formatDiscountPrice(previewPromotion, previewCurrency) }}</dd></div>
            <div><dt>{{ t('checkout.previewWholesale') }}</dt><dd>{{ formatDiscountPrice(previewWholesale, previewCurrency) }}</dd></div>
          </template>
          <div v-if="Number(previewMemberDiscount) > 0"><dt>{{ t('checkout.previewMemberDiscount') }}</dt><dd>-{{ formatPrice(previewMemberDiscount, previewCurrency) }}</dd></div>
          <div class="is-total"><dt>{{ t('checkout.previewTotal') }}</dt><dd>{{ formatPrice(previewTotal, previewCurrency) }}</dd></div>
        </dl>
        <p v-if="previewLoading || couponRefreshing">{{ previewStatusText }}</p>
        <Alert v-if="checkoutAlert" :variant="pageAlertVariant(checkoutAlert.level)"><AlertDescription>{{ checkoutAlert.message }}</AlertDescription></Alert>

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
          <button v-for="channel in paymentChannels" :key="channel.id" type="button" :disabled="isChannelDisabledForAmount(channel)" :title="isChannelDisabledForAmount(channel) ? channelAmountLimitHint(channel) : ''" :class="{ selected: selectedChannelId === channel.id && !isChannelDisabledForAmount(channel) }" @click="handleSelectChannel(channel)">
            <img v-if="channel.icon" :src="getImageUrl(channel.icon)" alt="" />
            <span>{{ channel.name }}</span>
            <small v-if="channel.fee_policy === 'customer_surcharge'">{{ t('payment.feeLabel') }} {{ formatChannelFeeRate(channel) }} · {{ t('payment.fixedFeeLabel') }} {{ formatChannelFixedFee(channel) }}</small>
            <small v-if="isChannelDisabledForAmount(channel)">{{ channelAmountLimitHint(channel) }}</small>
          </button>
        </div>
        <p v-else-if="!walletOnlyPayment && requiresOnlineChannel">{{ t('checkout.noPaymentChannels') }}</p>
        <p v-if="!requiresOnlineChannel">{{ t('checkout.walletCoversAll') }}</p>

        <InvoiceRequestEditor :email="guestEmail || userAuthStore.user?.email" :disabled="submitting" />
        <Button size="lg" class="nt-button-purchase w-full" :disabled="!canSubmit" @click="handleSubmit">{{ submitting ? t('checkout.submitting') : t('checkout.submitButton') }}</Button>
      </aside>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { pageAlertVariant } from '../../utils/alerts'
import ImageCaptcha from '../../components/captcha/ImageCaptcha.vue'
import TurnstileCaptcha from '../../components/captcha/TurnstileCaptcha.vue'
import CheckoutManualForm from '../../components/checkout/CheckoutManualForm.vue'
import EmptyState from '../../components/EmptyState.vue'
import SmartImage from '../../components/SmartImage.vue'
import CheckoutSteps from '../../components/checkout/CheckoutSteps.vue'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { useCheckout } from '../../composables/useCheckout'
import InvoiceRequestEditor from './components/InvoiceRequestEditor.vue'

const { t } = useI18n()
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
  submitting, canSubmit, handleSubmit,
} = useCheckout()

void guestImageCaptchaRef
void guestTurnstileRef
</script>

<style scoped>
.nt-checkout { height: calc(100dvh - 64px); display: flex; flex-direction: column; width: min(1160px, 100%); margin: 0 auto; padding: 12px 16px; overflow: hidden; animation: checkout-in .28s ease; }
.nt-checkout-head { display: flex; align-items: center; justify-content: space-between; gap: 16px; flex: none; padding-bottom: 10px; }
.nt-checkout-head h1 { font-size: 22px; font-weight: 750; letter-spacing: -.03em; }
.nt-checkout-grid { flex: 1; min-height: 0; display: grid; grid-template-columns: minmax(0, 1.2fr) minmax(280px, 380px); gap: 14px; }
.nt-checkout-detail, .nt-checkout-order { min-height: 0; overflow: auto; padding: 14px; border: 1px solid var(--ui-border); border-radius: 16px; background: rgba(255,255,255,.03); }
.nt-checkout-detail h2, .nt-checkout-order h2, .nt-checkout-order h3 { font-size: 14px; font-weight: 700; }
.nt-checkout-item { display: flex; gap: 12px; margin-top: 10px; padding-bottom: 10px; border-bottom: 1px solid var(--ui-border); }
.nt-checkout-item.is-warning { color: #e2b340; }
.nt-checkout-item a { font-weight: 650; }
.nt-checkout-item p, .nt-checkout-guest p, .nt-checkout-order > p { margin-top: 4px; color: var(--ui-text-muted); font-size: 12px; line-height: 1.5; }
.nt-checkout-item-price { color: var(--ui-text-primary) !important; font-weight: 700; }
.nt-checkout-field, .nt-checkout-guest { display: grid; gap: 8px; margin-top: 14px; font-size: 13px; }
.nt-checkout-modes, .nt-checkout-guest-fields, .nt-checkout-channels { display: flex; flex-wrap: wrap; gap: 8px; }
.nt-checkout-guest-fields { display: grid; grid-template-columns: 1fr 1fr; }
.nt-checkout-order dl { display: grid; gap: 6px; margin: 10px 0; font-size: 12px; }
.nt-checkout-order dl div { display: flex; justify-content: space-between; gap: 10px; }
.nt-checkout-order .is-total { margin-top: 6px; padding-top: 8px; border-top: 1px solid var(--ui-border); font-size: 15px; font-weight: 750; }
.nt-checkout-wallet { display: grid; gap: 4px; margin: 8px 0; padding: 8px; border: 1px solid var(--ui-border); border-radius: 10px; font-size: 12px; }
.nt-checkout-channels { display: grid; grid-template-columns: 1fr 1fr; }
.nt-checkout-channels button { display: grid; gap: 2px; min-height: 44px; padding: 8px; border: 1px solid var(--ui-border); border-radius: 10px; text-align: left; font-size: 12px; }
.nt-checkout-channels button.selected { border-color: var(--ui-accent); background: rgba(45,212,167,.12); }
.nt-checkout-channels img { width: 18px; height: 18px; object-fit: contain; }
.nt-checkout-channels small { color: var(--ui-text-muted); }
.nt-checkout-order :deep(.nt-button-purchase) { margin-top: 12px; }
@keyframes checkout-in { from { opacity: 0; transform: translateY(10px); } to { opacity: 1; transform: none; } }
@media (max-width: 800px) {
  .nt-checkout { height: auto; min-height: calc(100dvh - 64px); overflow: visible; }
  .nt-checkout-head { align-items: flex-start; flex-direction: column; }
  .nt-checkout-grid { grid-template-columns: 1fr; overflow: visible; }
  .nt-checkout-detail, .nt-checkout-order { overflow: visible; }
  .nt-checkout-order { order: -1; }
  .nt-checkout-guest-fields, .nt-checkout-channels { grid-template-columns: 1fr; }
}
@media (prefers-reduced-motion: reduce) { .nt-checkout { animation: none; } }
</style>
