<template>
  <div class="payment-method-strip" data-payment-presentation="static" :class="{ 'is-compact': compact, 'is-mobile-inline': mobileInline }">
    <div class="payment-methods" role="list" :aria-label="t('needtoken.paymentMarks.label')">
      <span class="payment-label" aria-hidden="true">{{ t('needtoken.paymentMarks.label') }}</span>
      <span v-for="method in methods" :key="method" class="payment-method" role="listitem">
        <img :src="brandAssets[method]" class="payment-logo" :class="method" alt="" width="21" height="21" />
        <span class="payment-method-label" :translate="method === 'usdt' ? 'no' : undefined">{{ t(`needtoken.paymentMarks.${method}`) }}</span>
      </span>
      <span class="payment-networks" role="listitem" translate="no">
        <span class="network-logos" aria-hidden="true"><img v-for="network in networks" :key="network.label" :src="network.asset" alt="" width="18" height="18" /></span>
        <span class="network-label">{{ networks.map(network => network.label).join(' · ') }}</span>
      </span>
    </div>
    <p v-if="!compact">{{ t('needtoken.paymentMarks.checkoutNote') }}</p>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

defineProps<{ compact?: boolean; mobileInline?: boolean }>()
const { t } = useI18n()

// Merchant-selected display marks. Checkout continues to resolve actual
// payment channels, limits and eligibility through the existing order APIs.
const methods = ['alipay', 'usdt'] as const
const brandAssets = { alipay: '/payment/alipay.svg', usdt: '/payment/usdt.svg' }
const networks = [
  { label: 'BEP20', asset: '/payment/bnb.svg' },
  { label: 'TRC20', asset: '/payment/tron.svg' },
] as const
</script>

<style scoped>
.payment-method-strip { min-width: 0; max-width: 100%; text-align: center; }
.payment-methods { display: flex; align-items: center; justify-content: center; flex-wrap: wrap; gap: 8px 16px; }
.payment-label { color: var(--ui-text-muted); font-size: 12px; font-weight: 400; }
.payment-method, .payment-networks { display: inline-flex; align-items: center; gap: 8px; color: var(--ui-text-secondary); font-size: 12px; line-height: 20px; font-variant-numeric: tabular-nums; white-space: nowrap; }
.payment-logo { display: block; width: 20px; height: 20px; flex: none; }
.network-logos { display: inline-flex; align-items: center; padding-right: 1px; }
.network-logos img { display: block; width: 18px; height: 18px; }
.network-logos img + img { margin-left: -4px; border: 1px solid var(--ui-bg-page, #0d1217); border-radius: 50%; }
.payment-method-strip p { margin-top: 8px; color: var(--ui-text-muted); font-size: 11px; line-height: 20px; }
.is-compact .payment-methods { justify-content: flex-end; flex-wrap: nowrap; gap: 8px 16px; overflow-x: auto; scrollbar-width: thin; }
.is-compact .payment-label { display: none; }
@media (max-width: 640px) {
  .payment-methods, .is-compact .payment-methods { justify-content: flex-start; gap: 8px 12px; }
  .payment-method, .payment-networks { font-size: 11px; }
  .payment-label { font-size: 11px; }
  .payment-logo { width: 19px; height: 19px; }
  .is-mobile-inline .payment-label { display: none; }
  .payment-method-strip.is-mobile-inline .payment-methods { justify-content: flex-start; flex-wrap: nowrap; gap: 8px; overflow-x: auto; scrollbar-width: thin; }
  .is-mobile-inline .payment-method, .is-mobile-inline .payment-networks { flex: none; gap: 6px; }
  .is-mobile-inline .payment-logo { width: 18px; height: 18px; }
}
@media (max-width: 359px) {
  .is-mobile-inline .network-label { position: absolute; width: 1px; height: 1px; padding: 0; margin: -1px; overflow: hidden; clip: rect(0, 0, 0, 0); white-space: nowrap; border: 0; }
}
</style>
