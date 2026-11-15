<template>
  <div class="payment-method-strip" :data-payment-state="state" :class="{ 'is-compact': compact, 'is-mobile-inline': mobileInline, 'is-dense': methods.length > 2 }">
    <div v-if="state === 'ready'" class="payment-methods" role="list" :aria-label="t('needtoken.paymentMarks.label')" :title="t('needtoken.paymentMarks.checkoutNote')">
      <span class="payment-label" aria-hidden="true">{{ t('needtoken.paymentMarks.label') }}</span>
      <span v-for="method in methods" :key="method.key" class="payment-method" :class="{ 'is-custom': !method.mark }" role="listitem" :title="methodLabel(method)">
        <img v-if="assetFor(method.mark)" :src="assetFor(method.mark)" class="payment-logo" :class="method.mark || undefined" alt="" width="21" height="21" />
        <Coins v-else-if="method.mark === 'crypto'" class="payment-logo crypto" :stroke-width="1.7" aria-hidden="true" />
        <CreditCard v-else-if="method.mark === 'card' || method.mark === 'unionpay'" class="payment-logo card" :stroke-width="1.7" aria-hidden="true" />
        <Wallet v-else class="payment-logo" :stroke-width="1.7" aria-hidden="true" />
        <span class="payment-method-label" :translate="method.mark === 'paypal' || method.mark === 'usdt' ? 'no' : undefined">{{ methodLabel(method) }}</span>
      </span>
      <span v-if="networks.length" class="payment-networks" role="listitem" translate="no">
        <span class="network-logos" aria-hidden="true"><img v-for="network in networks" :key="network" :src="network === 'bep20' ? '/payment/bnb.svg' : '/payment/tron.svg'" alt="" width="18" height="18" /></span>
        <span class="network-label">{{ networks.map(network => network.toUpperCase()).join(' · ') }}</span>
      </span>
    </div>
    <div v-else class="payment-status" role="status">
      <template v-if="state === 'loading'"><span class="payment-label">{{ t('needtoken.paymentMarks.label') }}</span><span class="payment-skeleton" aria-hidden="true" /><span class="sr-only">{{ t('needtoken.paymentMarks.loading') }}</span></template>
      <template v-else><Wallet v-if="state === 'wallet'" :size="17" aria-hidden="true" /><span class="payment-state-copy" :title="t(`needtoken.paymentMarks.${state}`)">{{ t(`needtoken.paymentMarks.${state}`) }}</span><button v-if="state === 'empty' || state === 'unavailable'" type="button" data-payment-retry :aria-label="t('needtoken.paymentMarks.retry')" :title="t('needtoken.paymentMarks.retry')" @click="refresh"><RefreshCw :size="14" aria-hidden="true" /></button></template>
    </div>
    <p v-if="!compact && state === 'ready'">{{ t('needtoken.paymentMarks.checkoutNote') }}</p>
  </div>
</template>

<script setup lang="ts">
import { computed, onUnmounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { Coins, CreditCard, RefreshCw, Wallet } from 'lucide-vue-next'
import { useAppStore } from '../../../stores/app'
import { displayPaymentChannels, paymentDisplayState, publicPaymentMethods, paymentNetworks, type PaymentMark, type PublicPaymentMethod } from '../lib/paymentMarks'

defineProps<{ compact?: boolean; mobileInline?: boolean }>()
const { t } = useI18n()
const appStore = useAppStore()
const refreshing = ref(false), refreshFailed = ref(false)
let disposed = false
const state = computed(() => refreshing.value ? 'loading' : refreshFailed.value ? 'unavailable' : paymentDisplayState(appStore.config, appStore.loading))
watch(() => appStore.config, () => { refreshFailed.value = false })
const channels = computed(() => displayPaymentChannels(appStore.config))
const methods = computed(() => publicPaymentMethods(channels.value))
const networks = computed(() => paymentNetworks(channels.value))
const brandAssets: Partial<Record<PaymentMark, string>> = {
  alipay: '/payment/alipay.svg', wechat: '/payment/wechat.svg',
  usdt: '/payment/usdt.svg', paypal: '/payment/paypal.svg',
}
const assetFor = (mark: PaymentMark | null) => mark ? brandAssets[mark] : undefined
const methodLabel = (method: PublicPaymentMethod) => method.label || t(`needtoken.paymentMarks.${method.mark}`)
async function refresh() {
  if (refreshing.value) return
  refreshing.value = true
  refreshFailed.value = false
  const success = await appStore.loadConfig(true)
  if (disposed) return
  refreshFailed.value = !success
  refreshing.value = false
}
onUnmounted(() => { disposed = true })
</script>

<style scoped>
.payment-method-strip { min-width: 0; max-width: 100%; text-align: center; }
.payment-methods { display: flex; align-items: center; justify-content: center; flex-wrap: wrap; gap: 8px 16px; }
.payment-label { color: var(--ui-text-muted); font-size: 12px; font-weight: 400; }
.payment-method, .payment-networks { display: inline-flex; align-items: center; gap: 8px; color: var(--ui-text-secondary); font-size: 12px; line-height: 20px; font-variant-numeric: tabular-nums; white-space: nowrap; }
.payment-logo { display: block; width: 20px; height: 20px; flex: none; }
.payment-status { display: flex; align-items: center; justify-content: center; gap: 8px; min-width: 0; min-height: 28px; color: var(--ui-text-muted); font-size: 12px; }
.is-compact .payment-status { justify-content: flex-end; }
.payment-status > svg { flex: none; }
.payment-state-copy { min-width: 0; overflow: hidden; white-space: nowrap; text-overflow: ellipsis; }
.payment-status button { display: grid; place-items: center; width: 28px; height: 28px; flex: none; border-radius: 6px; color: var(--ui-text-secondary); }
.payment-status button:hover { color: var(--ui-accent); background: var(--ui-bg-soft); }
.payment-status button:focus-visible { outline: 2px solid var(--ui-accent); outline-offset: 2px; }
.payment-skeleton { width: 88px; height: 16px; border-radius: 4px; background: var(--ui-bg-soft); }
.payment-method.is-custom .payment-method-label { max-width: 112px; overflow: hidden; text-overflow: ellipsis; }
.payment-logo.paypal { filter: invert(66%) sepia(47%) saturate(1459%) hue-rotate(186deg) brightness(107%) contrast(100%); }
.crypto { color: #a8bcbd; }
.card { color: #b8c5d9; }
.network-logos { display: inline-flex; align-items: center; padding-right: 1px; }
.network-logos img { display: block; width: 18px; height: 18px; }
.network-logos img + img { margin-left: -4px; border: 1px solid var(--ui-bg-page, #0d1217); border-radius: 50%; }
.payment-method-strip p { margin-top: 8px; color: var(--ui-text-muted); font-size: 11px; line-height: 20px; }
.is-compact .payment-methods { justify-content: flex-end; flex-wrap: nowrap; gap: 8px 16px; overflow-x: auto; scrollbar-width: thin; }
@media (max-width: 640px) { .payment-methods, .is-compact .payment-methods { justify-content: flex-start; gap: 8px 12px; } .payment-method, .payment-networks { font-size: 11px; } .payment-label { font-size: 11px; } .payment-logo { width: 19px; height: 19px; } }
@media (max-width: 640px) {
  .is-mobile-inline .payment-label { display: none; }
  .payment-method-strip.is-mobile-inline .payment-methods { justify-content: flex-start; flex-wrap: nowrap; gap: 8px; overflow-x: auto; scrollbar-width: thin; }
  .is-mobile-inline .payment-method, .is-mobile-inline .payment-networks { flex: none; }
  .is-mobile-inline .payment-method, .is-mobile-inline .payment-networks { gap: 6px; }
  .is-mobile-inline .payment-logo { width: 18px; height: 18px; }
  .is-mobile-inline.is-dense .payment-methods { gap: 6px; }
  .is-mobile-inline.is-dense .payment-method:not(.is-custom) .payment-method-label { position: absolute; width: 1px; height: 1px; padding: 0; margin: -1px; overflow: hidden; clip: rect(0, 0, 0, 0); white-space: nowrap; border: 0; }
  .is-mobile-inline .payment-status { font-size: 11px; }
  .is-mobile-inline .payment-method.is-custom .payment-method-label { max-width: 88px; }
}
@media (max-width: 359px) {
  .is-mobile-inline .network-label { position: absolute; width: 1px; height: 1px; padding: 0; margin: -1px; overflow: hidden; clip: rect(0, 0, 0, 0); white-space: nowrap; border: 0; }
}
</style>
