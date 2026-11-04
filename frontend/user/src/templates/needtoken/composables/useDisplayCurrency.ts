import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useAppStore } from '../../../stores/app'
import { useLocalized } from '../../../composables/useProduct'
import { formatServiceMoney } from '../lib/displayMoney'
import { convertDisplayAmount, displayCurrencyOptions, isDisplayCurrency, normalizeUsdCnyRate, resolveDisplayCurrency } from '../lib/currencyDisplay'

const PREFERENCE_KEY = 'needtoken-display-currency'
const preference = ref<string | null>(null)
let subscribers = 0
function readPreference() {
  try {
    const value = localStorage.getItem(PREFERENCE_KEY)
    preference.value = isDisplayCurrency(value) ? value : null
  } catch { /* Browsing and checkout remain available when storage is disabled. */ }
}
function onStorage(event: StorageEvent) {
  if (event.key === PREFERENCE_KEY || event.key === null) readPreference()
}

export function useDisplayCurrency() {
  const appStore = useAppStore()
  const { siteCurrency } = useLocalized()
  const settlementCurrency = computed(() => siteCurrency.value)
  const pricing = computed(() => appStore.isResellerTenant ? undefined : appStore.config?.needtoken_pricing)
  const rate = computed(() => normalizeUsdCnyRate(pricing.value?.usd_cny_rate))
  const rateUpdatedAt = computed(() => {
    const value = pricing.value?.rate_updated_at
    return typeof value === 'string' && Number.isFinite(Date.parse(value)) ? value : ''
  })
  const availableCurrencies = computed(() => displayCurrencyOptions(settlementCurrency.value, rate.value))
  const canSwitch = computed(() => availableCurrencies.value.length > 1)
  const currency = computed(() => resolveDisplayCurrency(settlementCurrency.value, rate.value, pricing.value?.default_display_currency, preference.value))
  const isConverted = computed(() => currency.value !== settlementCurrency.value)
  const selectCurrency = (code: string) => {
    if (!availableCurrencies.value.includes(code) || !isDisplayCurrency(code)) return
    preference.value = code
    try { localStorage.setItem(PREFERENCE_KEY, code) } catch { /* Keep the choice in this tab. */ }
  }
  const format = (amount: unknown) => {
    const converted = convertDisplayAmount(amount, settlementCurrency.value, currency.value, rate.value)
    return converted === null ? null : formatServiceMoney(converted, currency.value, appStore.locale)
  }
  const formatSettlement = (amount: unknown) => formatServiceMoney(amount, settlementCurrency.value, appStore.locale)
  onMounted(() => {
    if (subscribers++ === 0) {
      readPreference()
      window.addEventListener('storage', onStorage)
    }
  })
  onUnmounted(() => {
    if (--subscribers === 0) window.removeEventListener('storage', onStorage)
  })
  return { currency, settlementCurrency, availableCurrencies, canSwitch, rate, rateUpdatedAt, isConverted, selectCurrency, format, formatSettlement }
}
