<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { notifyError, notifySuccess } from '@/utils/notify'
import { mergeSiteSettingPatch } from '@/utils/needtokenSettings'
import { normalizeNeedtokenPricing, pricingPatchForSave } from '@/utils/needtokenPricing'

const { t } = useI18n()
const busy = ref(false)
const loaded = ref(false)
const settlementCurrency = ref('CNY')
const pricing = reactive(normalizeNeedtokenPricing(undefined))
const textCurrency = (raw: unknown) => typeof raw === 'string' && /^[A-Z]{3}$/.test(raw.trim().toUpperCase()) ? raw.trim().toUpperCase() : 'CNY'

async function load() {
  busy.value = true
  try {
    const response = await adminAPI.getSettings({ key: 'site_config' })
    const value = response.data.data as Record<string, unknown>
    settlementCurrency.value = textCurrency(value?.currency)
    Object.assign(pricing, normalizeNeedtokenPricing(value?.needtoken_pricing, settlementCurrency.value))
    loaded.value = true
  } catch (error) {
    if (!(error as { __notified?: boolean })?.__notified) notifyError(t('needtokenPricing.loadFailed'))
  } finally { busy.value = false }
}

async function save() {
  if (!loaded.value || busy.value) return
  busy.value = true
  try {
    const latest = await adminAPI.getSettings({ key: 'site_config' })
    const value = latest.data.data as Record<string, unknown>
    const previous = normalizeNeedtokenPricing(value?.needtoken_pricing, textCurrency(value?.currency))
    const update = pricingPatchForSave(pricing, previous, new Date().toISOString())
    const payload = mergeSiteSettingPatch(value, { needtoken_pricing: update })
    const response = await adminAPI.updateSettings({ key: 'site_config', value: payload })
    const saved = response.data.data as Record<string, unknown>
    settlementCurrency.value = textCurrency(saved?.currency)
    Object.assign(pricing, normalizeNeedtokenPricing(saved?.needtoken_pricing, settlementCurrency.value))
    notifySuccess(t('needtokenPricing.saved'))
  } catch (error) {
    if (!(error as { __notified?: boolean })?.__notified) notifyError((error as Error)?.message || t('needtokenPricing.saveFailed'))
  } finally { busy.value = false }
}
onMounted(load)
defineExpose({ save })
</script>

<template>
  <div class="space-y-6" :aria-busy="busy">
    <div class="rounded-xl border border-border bg-card">
      <header class="border-b border-border bg-muted/40 px-6 py-4">
        <h2 class="text-lg font-semibold">{{ t('needtokenPricing.title') }}</h2>
        <p class="mt-1 max-w-3xl text-xs leading-relaxed text-muted-foreground">{{ t('needtokenPricing.description') }}</p>
      </header>
      <fieldset :disabled="busy || !loaded" class="space-y-6 p-6">
        <div class="rounded-lg border border-border bg-muted/30 p-4 text-sm">
          <p class="font-medium">{{ t('needtokenPricing.settlement', { currency: settlementCurrency }) }}</p>
          <p class="mt-2 text-xs leading-relaxed text-muted-foreground">{{ t('needtokenPricing.settlementHint') }}</p>
        </div>
        <div class="grid gap-6 md:grid-cols-2">
          <div class="space-y-2">
            <Label for="needtoken-display-currency">{{ t('needtokenPricing.defaultCurrency') }}</Label>
            <Select v-model="pricing.default_display_currency">
              <SelectTrigger id="needtoken-display-currency" class="h-10 w-full"><SelectValue /></SelectTrigger>
              <SelectContent>
                <SelectItem value="CNY">{{ t('needtokenPricing.cny') }}</SelectItem>
                <SelectItem value="USD">{{ t('needtokenPricing.usd') }}</SelectItem>
              </SelectContent>
            </Select>
            <p class="text-xs leading-relaxed text-muted-foreground">{{ t('needtokenPricing.defaultHint') }}</p>
          </div>
          <div class="space-y-2">
            <Label for="needtoken-usd-cny-rate">{{ t('needtokenPricing.rate') }}</Label>
            <div class="flex items-center gap-3">
              <span class="shrink-0 text-sm text-muted-foreground">1 USD =</span>
              <Input id="needtoken-usd-cny-rate" v-model="pricing.usd_cny_rate" type="text" inputmode="decimal" maxlength="12" autocomplete="off" :placeholder="t('needtokenPricing.ratePlaceholder')" />
              <span class="shrink-0 text-sm text-muted-foreground">CNY</span>
            </div>
            <p class="text-xs leading-relaxed text-muted-foreground">{{ t('needtokenPricing.rateHint') }}</p>
          </div>
        </div>
        <p class="text-xs leading-relaxed text-muted-foreground">{{ pricing.rate_updated_at ? t('needtokenPricing.updated', { time: pricing.rate_updated_at }) : t('needtokenPricing.notConfigured') }}</p>
        <div class="rounded-lg border border-border p-4 text-xs leading-relaxed text-muted-foreground">{{ t('needtokenPricing.sourceHint') }}</div>
      </fieldset>
      <div v-if="!loaded && !busy" class="border-t border-border px-6 py-4"><Button type="button" variant="outline" @click="load">{{ t('needtokenPricing.retry') }}</Button></div>
    </div>
  </div>
</template>
