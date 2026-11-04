<template>
  <div class="nt-display-currency" :class="{ 'nt-display-currency--compact': compact }">
    <label :for="selectId">{{ t('needtoken.displayPricing.selectorLabel') }}</label>
    <div class="nt-display-currency__control">
      <Coins :size="15" aria-hidden="true" />
      <select :id="selectId" :value="currency" :disabled="!canSwitch" @change="onChange">
        <option v-for="code in availableCurrencies" :key="code" :value="code">{{ currencyLabel(code) }}</option>
      </select>
      <ChevronDown v-if="canSwitch" :size="13" class="nt-display-currency__chevron" aria-hidden="true" />
    </div>
    <span v-if="isConverted" class="nt-display-currency__settlement">{{ t('needtoken.displayPricing.settlementHint', { currency: settlementCurrency }) }}</span>
  </div>
</template>

<script setup lang="ts">
import { useId } from 'vue'
import { useI18n } from 'vue-i18n'
import { ChevronDown, Coins } from 'lucide-vue-next'
import { useDisplayCurrency } from '../composables/useDisplayCurrency'

withDefaults(defineProps<{ compact?: boolean }>(), { compact: false })
const { t } = useI18n()
const selectId = `nt-display-currency-${useId()}`
const { currency, settlementCurrency, availableCurrencies, canSwitch, isConverted, selectCurrency } = useDisplayCurrency()
const currencyLabel = (code: string) => {
  const name = ['CNY', 'USD'].includes(code) ? t(`needtoken.displayPricing.currencyNames.${code}`) : code
  return code !== settlementCurrency.value ? t('needtoken.displayPricing.referenceOption', { name }) : name
}
const onChange = (event: Event) => selectCurrency((event.target as HTMLSelectElement).value)
</script>

<style scoped>
.nt-display-currency { display: flex; align-items: center; justify-content: flex-end; flex-wrap: wrap; gap: 8px 10px; min-width: 0; color: var(--ui-text-muted); font-size: 12px; line-height: 1.5; }
.nt-display-currency > label { flex: none; }
.nt-display-currency__control { position: relative; display: inline-flex; align-items: center; gap: 8px; max-width: 100%; min-height: 36px; padding-left: 11px; border: 1px solid var(--ui-border); border-radius: 10px; background: var(--ui-bg-elevated); color: var(--ui-text-secondary); }
.nt-display-currency__control > svg { flex: none; color: var(--ui-accent); pointer-events: none; }
.nt-display-currency select { min-width: 0; max-width: 100%; appearance: none; padding: 8px 29px 8px 0; border-radius: 10px; outline-offset: 3px; background: transparent; color: var(--ui-text-primary); font: inherit; font-weight: 550; cursor: pointer; }
.nt-display-currency select:disabled { padding-right: 12px; cursor: default; opacity: .8; }
.nt-display-currency select option { background: var(--ui-bg-elevated); color: var(--ui-text-primary); }
.nt-display-currency__control > .nt-display-currency__chevron { position: absolute; right: 10px; color: var(--ui-text-muted); }
.nt-display-currency__settlement { flex-basis: 100%; text-align: right; font-size: 11px; overflow-wrap: anywhere; }
.nt-display-currency--compact { gap: 6px 9px; font-size: 11px; }
.nt-display-currency--compact .nt-display-currency__control { min-height: 32px; }
.nt-display-currency--compact select { padding-block: 6px; }
@media (max-width: 640px) { .nt-display-currency__settlement { text-align: left; } }
</style>
