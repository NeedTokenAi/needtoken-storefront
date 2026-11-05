<template>
  <div class="nt-display-currency" :class="{ 'nt-display-currency--compact': compact }">
    <label :for="selectId">{{ t('needtoken.displayPricing.selectorLabel') }}</label>
    <NeedtokenSelect :id="selectId" :model-value="currency" :options="currencyOptions" :label="t('needtoken.displayPricing.selectorLabel')" :disabled="!canSwitch" :compact="compact" @update:model-value="selectCurrency">
      <template #leading><Coins :size="15" /></template>
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

withDefaults(defineProps<{ compact?: boolean }>(), { compact: false })
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
.nt-display-currency { display: flex; align-items: center; justify-content: flex-end; flex-wrap: wrap; gap: 8px 10px; min-width: 0; color: var(--ui-text-muted); font-size: 12px; line-height: 1.5; }
.nt-display-currency > label { flex: none; }
.nt-display-currency > .nt-theme-select { flex: 0 1 auto; }
.nt-display-currency__settlement { flex-basis: 100%; text-align: right; font-size: 11px; overflow-wrap: anywhere; }
.nt-display-currency--compact { gap: 6px 9px; font-size: 11px; }
@media (max-width: 640px) { .nt-display-currency__settlement { text-align: left; } }
</style>
