<template>
  <PopoverRoot v-model:open="open">
    <PopoverAnchor as-child>
      <div ref="triggersEl" class="nt-catalogue-tools">
        <button type="button" class="nt-catalogue-tools__trigger" :class="{ active: open && mode === 'search' || !!modelValue.trim() }" :aria-label="searchButtonLabel" :title="searchButtonLabel" :aria-expanded="open && mode === 'search'" :aria-controls="panelId" aria-haspopup="dialog" @click="toggle($event, 'search')"><Search :size="18" aria-hidden="true" /><span v-if="modelValue.trim()" class="nt-catalogue-tools__dot" aria-hidden="true" /></button>
        <button type="button" class="nt-catalogue-tools__trigger nt-catalogue-tools__currency" :class="{ active: open && mode === 'currency' }" :aria-label="currencyButtonLabel" :title="currencyButtonLabel" :aria-expanded="open && mode === 'currency'" :aria-controls="panelId" aria-haspopup="dialog" @click="toggle($event, 'currency')"><span translate="no" aria-hidden="true">{{ currencySymbol }}</span></button>
      </div>
    </PopoverAnchor>
    <PopoverPortal>
      <PopoverContent :id="panelId" class="nt-catalogue-tools__panel" align="end" :side-offset="8" :collision-padding="12" role="dialog" :aria-labelledby="`${panelId}-title`" @open-auto-focus="onOpenAutoFocus" @close-auto-focus="onCloseAutoFocus" @interact-outside="onInteractOutside" @escape-key-down="restoreFocus = true">
        <header class="nt-catalogue-tools__heading"><div><h2 :id="`${panelId}-title`">{{ t('needtoken.catalogToolbar.title') }}</h2><span v-if="loading" role="status">{{ t('needtoken.common.loading') }}</span><span v-else-if="count !== null" aria-live="polite">{{ t('needtoken.catalogServices.resultCount', { count }) }}</span></div><button type="button" class="nt-catalogue-tools__close" :aria-label="t('needtoken.catalogToolbar.close')" @click="close"><X :size="17" aria-hidden="true" /></button></header>
        <form class="nt-catalogue-tools__search" role="search" @submit.prevent="submit">
          <label :for="`${panelId}-search`">{{ t('needtoken.catalogToolbar.searchLabel') }}</label>
          <div><Search :size="16" aria-hidden="true" /><input :id="`${panelId}-search`" ref="searchEl" type="search" :value="modelValue" :placeholder="searchLabel" :aria-label="searchLabel" @input="emit('update:modelValue', ($event.target as HTMLInputElement).value)" /><button type="submit" :aria-label="t('needtoken.catalogServices.searchSubmit')"><ArrowRight :size="17" aria-hidden="true" /></button></div>
        </form>
        <div ref="currencyEl" class="nt-catalogue-tools__pricing"><DisplayCurrencySelector compact /></div>
        <div v-if="$slots.default" class="nt-catalogue-tools__extra"><slot /></div>
      </PopoverContent>
    </PopoverPortal>
  </PopoverRoot>
</template>

<script setup lang="ts">
import { computed, nextTick, ref, useId } from 'vue'
import { useI18n } from 'vue-i18n'
import { ArrowRight, Search, X } from 'lucide-vue-next'
import { PopoverAnchor, PopoverContent, PopoverPortal, PopoverRoot } from 'reka-ui'
import { useDisplayCurrency } from '../composables/useDisplayCurrency'
import DisplayCurrencySelector from './DisplayCurrencySelector.vue'

const props = withDefaults(defineProps<{ modelValue: string; searchLabel: string; count?: number | null; loading?: boolean }>(), { count: null, loading: false })
const emit = defineEmits<{ 'update:modelValue': [value: string]; search: [] }>()
const { t } = useI18n()
const { currency } = useDisplayCurrency()
const open = ref(false)
const mode = ref<'search' | 'currency'>('search')
const triggersEl = ref<HTMLElement | null>(null)
const searchEl = ref<HTMLInputElement | null>(null)
const currencyEl = ref<HTMLElement | null>(null)
const panelId = `nt-catalogue-tools-${useId()}`
let activeTrigger: HTMLButtonElement | null = null
let restoreFocus = true
const searchButtonLabel = computed(() => props.modelValue.trim() ? t('needtoken.catalogToolbar.searchActive', { query: props.modelValue.trim() }) : t('needtoken.catalogToolbar.searchButton'))
const currencyButtonLabel = computed(() => t('needtoken.catalogToolbar.currencyButton', { currency: currency.value }))
const currencySymbol = computed(() => {
  try { return new Intl.NumberFormat('en-US', { style: 'currency', currency: currency.value, currencyDisplay: 'narrowSymbol' }).formatToParts(0).find(part => part.type === 'currency')?.value || currency.value }
  catch { return currency.value }
})
function focusControl() {
  if (mode.value === 'search') { searchEl.value?.focus(); searchEl.value?.select() }
  else currencyEl.value?.querySelector<HTMLElement>('[role="combobox"],button')?.focus()
}
function toggle(event: MouseEvent, next: typeof mode.value) {
  activeTrigger = event.currentTarget as HTMLButtonElement
  restoreFocus = true
  if (open.value && mode.value === next) return close()
  mode.value = next
  open.value = true
  void nextTick(focusControl)
}
function close() { restoreFocus = true; open.value = false }
function submit() { emit('search'); close() }
function onOpenAutoFocus(event: Event) { event.preventDefault(); void nextTick(focusControl) }
function onCloseAutoFocus(event: Event) {
  event.preventDefault()
  if (restoreFocus) void nextTick(() => activeTrigger?.focus())
}
function onInteractOutside(event: Event) {
  const original = (event as CustomEvent<{ originalEvent: Event }>).detail?.originalEvent
  if (original?.target instanceof Node && triggersEl.value?.contains(original.target)) event.preventDefault()
  else restoreFocus = false
}
</script>

<style>
.nt-catalogue-tools { display: flex; align-items: center; flex: none; gap: 8px; }
.nt-catalogue-tools__trigger { position: relative; display: grid; place-items: center; width: 40px; height: 40px; flex: none; border: 1px solid var(--ui-border); border-radius: 10px; background: var(--ui-bg-elevated); color: var(--ui-text-secondary); transition: border-color .15s, color .15s, background .15s; }
.nt-catalogue-tools__trigger:hover, .nt-catalogue-tools__trigger.active { color: var(--ui-accent); border-color: #2dd4a766; background: #2dd4a70d; }
.nt-catalogue-tools__trigger:focus-visible, .nt-catalogue-tools__close:focus-visible { outline: 2px solid var(--ui-accent); outline-offset: 3px; }
.nt-catalogue-tools__currency { font-family: var(--nt-font-mono); font-size: 19px; font-weight: 600; }
.nt-catalogue-tools__dot { position: absolute; right: 7px; top: 7px; width: 5px; height: 5px; border-radius: 50%; background: var(--ui-accent); }
.nt-catalogue-tools__panel { z-index: 80; width: 330px; max-width: calc(100vw - 24px); max-height: min(480px, calc(100dvh - 100px)); overflow-y: auto; padding: 16px; border: 1px solid #2dd4a73d; border-radius: 16px; background: var(--ui-bg-elevated); color: var(--ui-text-primary); box-shadow: 0 18px 60px #0008, inset 0 1px 0 #ffffff08; outline: none; }
.nt-catalogue-tools__heading { display: flex; align-items: flex-start; justify-content: space-between; gap: 16px; }
.nt-catalogue-tools__heading h2 { font-size: 14px; font-weight: 650; line-height: 1.5; }
.nt-catalogue-tools__heading span { display: block; margin-top: 4px; font-size: 12px; line-height: 1.5; color: var(--ui-text-muted); }
.nt-catalogue-tools__close { display: grid; place-items: center; flex: none; width: 32px; height: 32px; margin: -4px -4px 0 0; border-radius: 8px; color: var(--ui-text-muted); }
.nt-catalogue-tools__close:hover { color: var(--ui-text-primary); background: var(--ui-bg-soft); }
.nt-catalogue-tools__search { margin-top: 16px; }
.nt-catalogue-tools__search > label { display: block; margin-bottom: 8px; color: var(--ui-text-secondary); font-size: 12px; }
.nt-catalogue-tools__search > div { display: flex; align-items: center; gap: 9px; min-height: 42px; padding-left: 12px; border: 1px solid var(--ui-border); border-radius: 10px; background: var(--ui-bg-soft); color: var(--ui-text-muted); }
.nt-catalogue-tools__search > div:focus-within { outline: 2px solid var(--ui-accent); outline-offset: 2px; }
.nt-catalogue-tools__search input { min-width: 0; width: 100%; flex: 1; padding: 9px 0; background: transparent; outline: none; color: var(--ui-text-primary); font-size: 13px; }
.nt-catalogue-tools__search button { display: grid; place-items: center; width: 40px; min-height: 40px; flex: none; color: var(--ui-accent); }
.nt-catalogue-tools__pricing { margin-top: 16px; padding-top: 14px; border-top: 1px solid var(--ui-border); }
.nt-catalogue-tools__pricing .nt-display-currency { justify-content: space-between; font-size: 12px; }
.nt-catalogue-tools__extra { margin-top: 14px; padding-top: 12px; border-top: 1px solid var(--ui-border); }
@media (max-width: 640px), (pointer: coarse) { .nt-catalogue-tools__trigger { width: 44px; height: 44px; } .nt-catalogue-tools__search > div, .nt-catalogue-tools__search button { min-height: 44px; } }
@media (prefers-reduced-motion: reduce) { .nt-catalogue-tools__trigger { transition: none; } }
</style>
