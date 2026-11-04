<template>
  <div class="payment-method-strip">
    <div v-if="methods.length" class="payment-methods" role="list" :aria-label="t('needtoken.paymentMarks.label')">
      <span v-for="method in methods" :key="method" class="payment-method" role="listitem">
        <svg v-if="method === 'alipay'" viewBox="0 0 24 24" class="payment-logo alipay" aria-hidden="true">
          <rect width="24" height="24" rx="6" fill="currentColor" />
          <path d="M7 7h10M12 4.5v5M7.5 10h8.3c-1 4.6-4.3 7.1-8.8 8M9 12.2c2 2.6 5.1 4.4 9.5 5.8" fill="none" stroke="white" stroke-width="1.7" stroke-linecap="round" />
        </svg>
        <svg v-else-if="method === 'wechat'" viewBox="0 0 24 24" class="payment-logo wechat" aria-hidden="true">
          <path d="M2 9a8 7 0 0 1 16-1.3A7.6 7.6 0 0 0 8.6 16l-3 1 .6-3A6.5 6.5 0 0 1 2 9Z" fill="currentColor" />
          <path d="M10 14a6 5.3 0 1 1 10.1 3.8l.4 2.3-2.6-.8A6.5 6.5 0 0 1 10 14Z" fill="currentColor" />
          <circle cx="7" cy="8" r="1" fill="white" /><circle cx="12" cy="8" r="1" fill="white" />
          <circle cx="14" cy="13.5" r=".85" fill="white" /><circle cx="18" cy="13.5" r=".85" fill="white" />
        </svg>
        <svg v-else-if="method === 'unionpay'" viewBox="0 0 30 22" class="payment-logo unionpay" aria-hidden="true">
          <path d="M4 2h9L9 20H0Z" fill="#dc354b" /><path d="M13 2h9l-4 18H9Z" fill="#2479b0" /><path d="M22 2h8l-4 18h-8Z" fill="#1e9272" />
          <path d="m8 7-1.2 5.4c-.6 2.4 3.4 2.4 4 0L12 7m4-0.1h2c3.1 0 2.4 4.5-.5 4.5H15m1-4.5-2.4 9" fill="none" stroke="white" stroke-width="1.6" stroke-linecap="round" />
        </svg>
        <Coins v-else-if="method === 'crypto'" class="payment-logo crypto" :stroke-width="1.7" aria-hidden="true" />
        <CreditCard v-else-if="method === 'card'" class="payment-logo card" :stroke-width="1.7" aria-hidden="true" />
        <svg v-else viewBox="0 0 24 24" class="payment-logo paypal" aria-hidden="true">
          <path d="m7 3-3 18h4l1-6h3c10 0 12-12 2-12Z" fill="#87baff" /><path d="m10 8-2 13h4l1-5h2c7 0 9-8 2-8Z" fill="#4d92e2" />
        </svg>
        <span :translate="method === 'paypal' ? 'no' : undefined">{{ t(`needtoken.paymentMarks.${method}`) }}</span>
      </span>
    </div>
    <p>{{ t('needtoken.paymentMarks.checkoutNote') }}</p>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { Coins, CreditCard } from 'lucide-vue-next'
import { useAppStore } from '../../../stores/app'
import { paymentMarks } from '../lib/paymentMarks'

const { t } = useI18n()
const appStore = useAppStore()
const methods = computed(() => appStore.config?.wallet_only_payment ? [] : paymentMarks(appStore.config?.payment_channels))
</script>

<style scoped>
.payment-method-strip { margin-top: 20px; text-align: center; }
.payment-methods { display: flex; align-items: center; justify-content: center; flex-wrap: wrap; gap: 8px 18px; }
.payment-method { display: inline-flex; align-items: center; gap: 7px; color: var(--ui-text-secondary); font-size: 11px; line-height: 1.5; }
.payment-logo { width: 21px; height: 21px; flex: none; }
.alipay { color: #429afa; }
.wechat { color: #42d27b; }
.unionpay { width: 27px; }
.crypto { color: #e8c273; }
.card { color: #b8c5d9; }
.payment-method-strip p { margin-top: 9px; color: var(--ui-text-muted); font-size: 10px; line-height: 1.6; }
@media (max-width: 640px) { .payment-method-strip { margin-top: 16px; } .payment-methods { gap: 9px 14px; } }
</style>
