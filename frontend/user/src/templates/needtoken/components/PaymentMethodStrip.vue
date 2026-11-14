<template>
  <div v-if="methods.length" class="payment-method-strip" :class="{ 'is-compact': compact, 'is-mobile-inline': mobileInline, 'is-dense': methods.length > 2 }">
    <div class="payment-methods" role="list" :aria-label="t('needtoken.paymentMarks.label')">
      <span class="payment-label" aria-hidden="true">{{ t('needtoken.paymentMarks.label') }}</span>
      <span v-for="method in methods" :key="method" class="payment-method" role="listitem">
        <img v-if="brandAssets[method]" :src="brandAssets[method]" class="payment-logo" :class="method" alt="" width="21" height="21" />
        <Coins v-else-if="method === 'crypto'" class="payment-logo crypto" :stroke-width="1.7" aria-hidden="true" />
        <CreditCard v-else class="payment-logo card" :stroke-width="1.7" aria-hidden="true" />
        <span class="payment-method-label" :translate="method === 'paypal' || method === 'usdt' ? 'no' : undefined">{{ t(`needtoken.paymentMarks.${method}`) }}</span>
      </span>
      <span v-if="networks.length" class="payment-networks" role="listitem" translate="no">
        <span class="network-logos" aria-hidden="true"><img v-for="network in networks" :key="network" :src="network === 'bep20' ? '/payment/bnb.svg' : '/payment/tron.svg'" alt="" width="18" height="18" /></span>
        <span class="network-label">{{ networks.map(network => network.toUpperCase()).join(' · ') }}</span>
      </span>
    </div>
    <p v-if="!compact">{{ t('needtoken.paymentMarks.checkoutNote') }}</p>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { Coins, CreditCard } from 'lucide-vue-next'
import { useAppStore } from '../../../stores/app'
import { paymentMarks, paymentNetworks, type PaymentMark } from '../lib/paymentMarks'

defineProps<{ compact?: boolean; mobileInline?: boolean }>()
const { t } = useI18n()
const appStore = useAppStore()
const channels = computed(() => appStore.config?.wallet_only_payment ? [] : appStore.config?.payment_channels)
const methods = computed(() => paymentMarks(channels.value))
const networks = computed(() => paymentNetworks(channels.value))
const brandAssets: Partial<Record<PaymentMark, string>> = {
  alipay: '/payment/alipay.svg', wechat: '/payment/wechat.svg',
  usdt: '/payment/usdt.svg', paypal: '/payment/paypal.svg',
}
</script>

<style scoped>
.payment-method-strip { text-align: center; }
.payment-methods { display: flex; align-items: center; justify-content: center; flex-wrap: wrap; gap: 8px 16px; }
.payment-label { color: var(--ui-text-muted); font-size: 12px; font-weight: 400; }
.payment-method, .payment-networks { display: inline-flex; align-items: center; gap: 8px; color: var(--ui-text-secondary); font-size: 12px; line-height: 20px; font-variant-numeric: tabular-nums; white-space: nowrap; }
.payment-logo { display: block; width: 20px; height: 20px; flex: none; }
.payment-logo.alipay { filter: invert(39%) sepia(98%) saturate(2818%) hue-rotate(201deg) brightness(101%) contrast(101%); }
.payment-logo.wechat { filter: invert(68%) sepia(65%) saturate(480%) hue-rotate(87deg) brightness(94%) contrast(100%); }
.payment-logo.paypal { filter: invert(66%) sepia(47%) saturate(1459%) hue-rotate(186deg) brightness(107%) contrast(100%); }
.crypto { color: #a8bcbd; }
.card { color: #b8c5d9; }
.network-logos { display: inline-flex; align-items: center; padding-right: 1px; }
.network-logos img { display: block; width: 18px; height: 18px; }
.network-logos img + img { margin-left: -4px; border: 1px solid var(--ui-bg-page, #0d1217); border-radius: 50%; }
.payment-method-strip p { margin-top: 8px; color: var(--ui-text-muted); font-size: 11px; line-height: 20px; }
.is-compact .payment-methods { justify-content: flex-end; gap: 8px 16px; }
@media (max-width: 640px) { .payment-methods, .is-compact .payment-methods { justify-content: flex-start; gap: 8px 12px; } .payment-method, .payment-networks { font-size: 11px; } .payment-label { font-size: 11px; } .payment-logo { width: 19px; height: 19px; } }
@media (max-width: 640px) {
  .is-mobile-inline .payment-label { display: none; }
  .is-mobile-inline .payment-methods { flex-wrap: nowrap; gap: 8px; }
  .is-mobile-inline .payment-method, .is-mobile-inline .payment-networks { gap: 6px; }
  .is-mobile-inline .payment-logo { width: 18px; height: 18px; }
  .is-mobile-inline.is-dense .payment-methods { gap: 6px; }
  .is-mobile-inline.is-dense .payment-method-label { position: absolute; width: 1px; height: 1px; padding: 0; margin: -1px; overflow: hidden; clip: rect(0, 0, 0, 0); white-space: nowrap; border: 0; }
}
@media (max-width: 359px) {
  .is-mobile-inline .network-label { position: absolute; width: 1px; height: 1px; padding: 0; margin: -1px; overflow: hidden; clip: rect(0, 0, 0, 0); white-space: nowrap; border: 0; }
}
</style>
