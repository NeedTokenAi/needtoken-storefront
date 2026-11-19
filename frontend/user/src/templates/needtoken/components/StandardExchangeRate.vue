<template>
  <div ref="rootEl" class="nt-standard-rate" :class="`nt-standard-rate--${placement}`" @keydown.esc.stop="close(true)" @focusout="onFocusOut">
    <button ref="triggerEl" type="button" class="nt-standard-rate__trigger" :aria-expanded="open" :aria-controls="panelId" @click="open = !open">
      <ArrowLeftRight :size="15" aria-hidden="true" />
      <span>{{ t('needtoken.displayPricing.rateBadge') }}</span>
    </button>
    <Transition name="nt-rate-pop">
    <div v-if="open" :id="panelId" class="nt-standard-rate__panel" role="region" :aria-label="t('needtoken.displayPricing.rateTitle')">
      <div class="nt-standard-rate__heading"><h2>{{ t('needtoken.displayPricing.rateTitle') }}</h2><button type="button" :aria-label="t('needtoken.displayPricing.closeRate')" @click="close(true)"><X :size="15" aria-hidden="true" /></button></div>
      <template v-if="rate">
        <p class="nt-standard-rate__value" translate="no">1 USD = {{ rate }} CNY</p>
        <p v-if="updatedAt">{{ t('needtoken.displayPricing.rateUpdated', { time: updatedAt }) }}</p>
        <p>{{ t('needtoken.displayPricing.rateSource') }}</p>
      </template>
      <template v-else>
        <p class="nt-standard-rate__pending">{{ t('needtoken.displayPricing.ratePending') }}</p>
        <p>{{ t('needtoken.displayPricing.ratePendingHint', { currency: settlementCurrency }) }}</p>
      </template>
      <p class="nt-standard-rate__note">{{ t('needtoken.displayPricing.rateSettlementHint', { currency: settlementCurrency }) }}</p>
    </div>
    </Transition>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, useId } from 'vue'
import { useI18n } from 'vue-i18n'
import { ArrowLeftRight, X } from 'lucide-vue-next'
import { useDisplayCurrency } from '../composables/useDisplayCurrency'

withDefaults(defineProps<{ placement?: 'above' | 'below' }>(), { placement: 'above' })
const { t, locale } = useI18n()
const { rate, rateUpdatedAt, settlementCurrency } = useDisplayCurrency()
const panelId = `nt-standard-rate-${useId()}`
const rootEl = ref<HTMLElement | null>(null)
const triggerEl = ref<HTMLButtonElement | null>(null)
const open = ref(false)
const updatedAt = computed(() => {
  const date = new Date(rateUpdatedAt.value)
  return rateUpdatedAt.value && Number.isFinite(date.getTime()) ? date.toLocaleString(locale.value, { year: 'numeric', month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' }) : ''
})
function close(restoreFocus = false) { open.value = false; if (restoreFocus) void nextTick(() => triggerEl.value?.focus()) }
function onFocusOut(event: FocusEvent) { if (event.relatedTarget instanceof Node && !rootEl.value?.contains(event.relatedTarget)) close() }
function onPointerDown(event: PointerEvent) { if (event.target instanceof Node && !rootEl.value?.contains(event.target)) close() }
onMounted(() => document.addEventListener('pointerdown', onPointerDown))
onUnmounted(() => document.removeEventListener('pointerdown', onPointerDown))
</script>

<style scoped>
.nt-standard-rate { position: relative; width: 100%; height: 100%; min-width: 0; text-align: center; }
.nt-standard-rate__trigger { display: flex; align-items: center; justify-content: center; gap: 8px; width: 100%; min-height: 100%; color: inherit; cursor: pointer; font: inherit; text-align: center; }
.nt-standard-rate__trigger > svg { flex: none; opacity: .85; }
.nt-standard-rate__trigger > span { min-width: 0; overflow-wrap: anywhere; }
.nt-standard-rate__panel { position: absolute; right: -6px; bottom: calc(100% + 18px); z-index: 40; width: 330px; max-width: calc(100vw - 40px); padding: 18px; border: 1px solid var(--nt-accent-border); border-radius: 15px; background: var(--ui-bg-elevated); color: var(--ui-text-secondary); box-shadow: var(--nt-popover-shadow); font-size: 12px; font-weight: 400; line-height: 1.75; text-align: left; transform-origin: bottom right; }
.nt-standard-rate--below .nt-standard-rate__panel { top: calc(100% + 18px); bottom: auto; transform-origin: top right; }
.nt-rate-pop-enter-active { transition: opacity 180ms var(--nt-ease-emphasized), transform 240ms var(--nt-ease-spring-soft); }
.nt-rate-pop-leave-active { transition: opacity 110ms var(--nt-motion-ease), transform 110ms var(--nt-motion-ease); pointer-events: none; }
.nt-rate-pop-enter-from, .nt-rate-pop-leave-to { opacity: 0; transform: translateY(6px) scale(0.96); }
.nt-standard-rate--below .nt-rate-pop-enter-from, .nt-standard-rate--below .nt-rate-pop-leave-to { transform: translateY(-6px) scale(0.96); }
@media (prefers-reduced-motion: reduce) {
  .nt-rate-pop-enter-active, .nt-rate-pop-leave-active { transition: none; }
  .nt-rate-pop-enter-from, .nt-rate-pop-leave-to { transform: none; }
}
.nt-standard-rate__heading { display: flex; align-items: flex-start; justify-content: space-between; gap: 14px; }
.nt-standard-rate__heading h2 { color: var(--ui-text-primary); font-size: 14px; font-weight: 650; line-height: 1.5; }
.nt-standard-rate__heading button { display: grid; flex: none; place-items: center; width: 22px; height: 22px; margin: -2px -3px 0 0; border-radius: 6px; color: var(--ui-text-muted); }
.nt-standard-rate__heading button:hover { background: var(--ui-bg-soft); color: var(--ui-text-primary); }
.nt-standard-rate__panel p { margin-top: 8px; overflow-wrap: anywhere; }
.nt-standard-rate__panel .nt-standard-rate__value { margin: 13px 0 10px; color: var(--ui-accent); font-family: var(--nt-font-mono); font-size: 18px; font-weight: 600; line-height: 1.45; }
.nt-standard-rate__pending { color: var(--nt-warning-ink); }
.nt-standard-rate__panel .nt-standard-rate__note { margin-top: 13px; padding-top: 11px; border-top: 1px solid var(--ui-border); color: var(--ui-text-muted); font-size: 11px; }
</style>
