<template>
  <div class="nt-display-currency" :class="{ 'nt-display-currency--compact': compact, 'nt-display-currency--mobile-code': mobileCode }">
    <label :for="selectId" :class="{ 'sr-only': hideLabel }">{{ t('needtoken.displayPricing.selectorLabel') }}</label>
    <NeedtokenSelect :id="selectId" :model-value="currency" :options="currencyOptions" :label="mobileCode ? `${t('needtoken.displayPricing.selectorLabel')}: ${currency}` : t('needtoken.displayPricing.selectorLabel')" :disabled="!canSwitch" :compact="compact" @update:model-value="selectCurrency">
      <template #leading><Coins class="nt-display-currency__leading-icon" :size="15" /><span v-if="mobileCode" class="nt-display-currency__mobile-code">{{ currency }}</span></template>
    </NeedtokenSelect>
    <span v-if="isConverted" class="nt-display-currency__settlement">{{ t('needtoken.displayPricing.settlementHint', { currency: settlementCurrency }) }}</span>
  </div>
</template>

<script setup lang="ts">
import { computed, useId } from 'vue'
import { useI18n } from 'vue-i18n'
import { Coins } from 'lucide-vue-next'
import { useDisplayCurrency } from '../composables/useDisplayCurrency'
import NeedtokenSelect from './NeedtokenSelect.vue'

withDefaults(defineProps<{ compact?: boolean; hideLabel?: boolean; mobileCode?: boolean }>(), { compact: false, hideLabel: false, mobileCode: false })
const { t } = useI18n()
const selectId = `nt-display-currency-${useId()}`
const { currency, settlementCurrency, availableCurrencies, canSwitch, isConverted, selectCurrency } = useDisplayCurrency()
const currencyLabel = (code: string) => {
  const name = ['CNY', 'USD'].includes(code) ? t(`needtoken.displayPricing.currencyNames.${code}`) : code
  return code !== settlementCurrency.value ? t('needtoken.displayPricing.referenceOption', { name }) : name
}
const currencyOptions = computed(() => availableCurrencies.value.map(code => ({ value: code, label: currencyLabel(code) })))
</script>

<style scoped>
.nt-display-currency { display: flex; align-items: center; justify-content: flex-end; flex-wrap: wrap; gap: 8px; min-width: 0; color: var(--ui-text-muted); font-size: 12px; line-height: 1.5; }
.nt-display-currency > label { flex: none; }
.nt-display-currency > .nt-theme-select { flex: 0 1 auto; }
.nt-display-currency__settlement { flex-basis: 100%; text-align: right; font-size: 11px; overflow-wrap: anywhere; }
.nt-display-currency--compact { gap: 8px; font-size: 11px; }
.nt-display-currency__mobile-code { display: none; }
.nt-display-currency :deep(.nt-theme-select__trigger) { border-radius: 10px; box-shadow: none; font-weight: 500; font-variant-numeric: tabular-nums; transition: color var(--nt-motion-fast, 120ms), background-color var(--nt-motion-fast, 120ms), border-color var(--nt-motion-fast, 120ms), transform var(--nt-motion-fast, 120ms) var(--nt-motion-ease, ease-out); }
.nt-display-currency :deep(.nt-theme-select__trigger):active:not(:disabled) { transform: translateY(1px); }
@media (max-width: 640px) {
  .nt-display-currency__settlement { text-align: left; }
  .nt-display-currency--mobile-code :deep(.nt-theme-select__leading) { color: var(--ui-text-primary); }
  .nt-display-currency--mobile-code .nt-display-currency__leading-icon { display: none; }
  .nt-display-currency--mobile-code .nt-display-currency__mobile-code { display: inline; }
  .nt-display-currency--mobile-code :deep(.nt-theme-select__value) { position: absolute; width: 1px; height: 1px; padding: 0; margin: -1px; overflow: hidden; clip: rect(0, 0, 0, 0); white-space: nowrap; border: 0; }
  .nt-display-currency--mobile-code :deep(.nt-theme-select__trigger) { min-width: 64px; min-height: 36px; padding: 8px; gap: 8px; }
}
@media (prefers-reduced-motion: reduce) { .nt-display-currency :deep(.nt-theme-select__trigger) { transition: none; } .nt-display-currency :deep(.nt-theme-select__trigger):active:not(:disabled) { transform: none; } }
</style>
