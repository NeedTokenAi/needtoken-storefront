<template>
  <div v-if="support.enabled" ref="widgetEl" class="nt-support-widget" :class="{ 'nt-support-widget--raised': raised }">
    <section v-if="open" :id="panelId" ref="panelEl" class="nt-support-panel" role="dialog" aria-modal="false" :aria-labelledby="`${panelId}-title`" @keydown="onPanelKeydown">
      <header class="nt-support-heading">
        <div class="nt-support-heading-icon"><Headset :size="22" /></div>
        <div class="min-w-0 flex-1">
          <h2 :id="`${panelId}-title`" class="text-base font-semibold">{{ t('needtoken.support.title') }}</h2>
          <p class="mt-1 text-xs leading-relaxed text-muted-foreground">{{ t('needtoken.support.intro') }}</p>
        </div>
        <button type="button" class="nt-support-close" :aria-label="t('needtoken.support.close')" @click="close(true)"><X :size="18" /></button>
      </header>

      <div class="nt-support-tabs" role="tablist" :aria-label="t('needtoken.support.tabsLabel')">
        <button v-for="item in tabs" :id="`${panelId}-tab-${item}`" :key="item" type="button" role="tab" :aria-selected="tab === item" :aria-controls="`${panelId}-content-${item}`" :tabindex="tab === item ? 0 : -1" @click="tab = item" @keydown="onTabKeydown($event, item)">
          <MessageCircle v-if="item === 'contact'" :size="16" /><Star v-else :size="16" />{{ t(`needtoken.support.${item}`) }}
        </button>
      </div>

      <div v-if="tab === 'contact'" :id="`${panelId}-content-contact`" class="nt-support-content" role="tabpanel" :aria-labelledby="`${panelId}-tab-contact`" tabindex="0">
        <div v-if="qrCards.length" class="nt-support-qrs">
          <figure v-for="card in qrCards" :key="card.key">
            <div class="nt-support-qr"><img :src="getImageUrl(card.image)" :alt="card.label" width="160" height="160" loading="lazy" decoding="async" referrerpolicy="no-referrer" /></div>
            <figcaption>{{ card.label }}</figcaption>
          </figure>
        </div>
        <p v-else class="nt-support-empty">{{ t('needtoken.support.emailIntro') }}</p>
        <p v-if="support.serviceHours" class="nt-support-hours"><Clock3 :size="14" />{{ t('needtoken.support.hours', { hours: support.serviceHours }) }}</p>
        <a v-if="support.contactUrl" :href="support.contactUrl" target="_blank" rel="noopener noreferrer" class="nt-support-action"><MessageCircle :size="17" />{{ t('needtoken.support.openContact') }}<ArrowUpRight :size="16" /></a>
        <a v-else-if="support.businessEmail" :href="emailHref" class="nt-support-action"><Mail :size="17" />{{ t('needtoken.support.sendEmail') }}</a>
        <a v-if="support.businessEmail" :href="emailHref" class="nt-support-email">{{ support.businessEmail }}</a>
        <p class="nt-support-order-note">{{ t('needtoken.support.orderNote') }}</p>
      </div>

      <div v-else :id="`${panelId}-content-bookmark`" class="nt-support-content" role="tabpanel" :aria-labelledby="`${panelId}-tab-bookmark`" tabindex="0">
        <h3 class="text-sm font-medium text-foreground">{{ t('needtoken.support.bookmarkTitle') }}</h3>
        <p class="mt-2 text-xs leading-relaxed text-muted-foreground">{{ t('needtoken.support.bookmarkIntro') }}</p>
        <dl class="nt-support-shortcuts">
          <div><dt>{{ t('needtoken.support.windows') }}</dt><dd><kbd>Ctrl + D</kbd></dd></div>
          <div><dt>{{ t('needtoken.support.mac') }}</dt><dd><kbd>⌘ + D</kbd></dd></div>
          <div><dt>{{ t('needtoken.support.mobile') }}</dt><dd>{{ t('needtoken.support.mobileHelp') }}</dd></div>
        </dl>
      </div>
    </section>

    <button ref="triggerEl" type="button" class="nt-support-trigger" :aria-label="open ? t('needtoken.support.close') : t('needtoken.support.open')" :aria-expanded="open" :aria-controls="panelId" aria-haspopup="dialog" @click="toggle">
      <X v-if="open" :size="23" />
      <span v-else class="nt-support-eyes" aria-hidden="true"><span v-for="eye in 2" :key="eye" class="nt-support-eye"><span class="nt-support-pupil" /></span></span>
    </button>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, useId, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { ArrowUpRight, Clock3, Headset, Mail, MessageCircle, Star, X } from 'lucide-vue-next'
import { getImageUrl } from '../../../utils/image'
import { supportEmailHref, type NeedtokenSupport } from '../lib/supportConfig'

const props = defineProps<{ support: NeedtokenSupport; raised?: boolean }>()
const { t } = useI18n()
const panelId = `nt-support-${useId()}`
const open = ref(false)
const tab = ref<'contact' | 'bookmark'>('contact')
const tabs = ['contact', 'bookmark'] as const
const widgetEl = ref<HTMLElement | null>(null)
const panelEl = ref<HTMLElement | null>(null)
const triggerEl = ref<HTMLButtonElement | null>(null)
let eyeFrame = 0
let eyeTracking = false
let eyeVisible = true
let eyePointerX = 0, eyePointerY = 0
let lastEyeX = 0, lastEyeY = 0
let motionQuery: MediaQueryList | undefined
let pointerQuery: MediaQueryList | undefined
let eyeObserver: IntersectionObserver | undefined
const emailHref = computed(() => supportEmailHref(props.support.businessEmail))
const qrCards = computed(() => [
  { key: 'wecom', label: t('needtoken.support.wecom'), image: props.support.wecomQr },
  { key: 'qq', label: t('needtoken.support.qq'), image: props.support.qqQr },
].filter(card => !!card.image))

function close(restoreFocus = false) {
  open.value = false
  if (restoreFocus) void nextTick(() => triggerEl.value?.focus())
}
async function toggle() {
  if (open.value) return close(true)
  open.value = true
  tab.value = 'contact'
  await nextTick()
  panelEl.value?.querySelector<HTMLButtonElement>('.nt-support-close')?.focus()
}
function onPanelKeydown(event: KeyboardEvent) {
  if (event.key !== 'Escape') return
  event.preventDefault()
  event.stopPropagation()
  close(true)
}
function onTabKeydown(event: KeyboardEvent, current: typeof tab.value) {
  if (!['ArrowLeft', 'ArrowRight', 'Home', 'End'].includes(event.key)) return
  event.preventDefault()
  tab.value = event.key === 'Home' ? 'contact' : event.key === 'End' ? 'bookmark' : current === 'contact' ? 'bookmark' : 'contact'
  void nextTick(() => document.getElementById(`${panelId}-tab-${tab.value}`)?.focus())
}
function onDocumentPointer(event: PointerEvent) {
  if (event.target instanceof Node && !widgetEl.value?.contains(event.target)) close()
}
function onDocumentFocus(event: FocusEvent) {
  if (event.target instanceof Node && !widgetEl.value?.contains(event.target)) close()
}
function resetEyes() {
  if (eyeFrame) window.cancelAnimationFrame(eyeFrame)
  eyeFrame = 0; lastEyeX = 0; lastEyeY = 0
  triggerEl.value?.style.removeProperty('--nt-eye-x')
  triggerEl.value?.style.removeProperty('--nt-eye-y')
}
function syncEyeTracking() {
  const enabled = !!triggerEl.value && props.support.enabled && !open.value && eyeVisible && !document.hidden && !motionQuery?.matches && !!pointerQuery?.matches
  if (enabled === eyeTracking) return
  eyeTracking = enabled
  if (enabled) window.addEventListener('pointermove', onEyePointer, { passive: true })
  else { window.removeEventListener('pointermove', onEyePointer); resetEyes() }
}
function updateEyes() {
  eyeFrame = 0
  const trigger = triggerEl.value
  if (!eyeTracking || !trigger) return
  const bounds = trigger.getBoundingClientRect()
  if (bounds.bottom <= 0 || bounds.right <= 0 || bounds.top >= window.innerHeight || bounds.left >= window.innerWidth) return
  const dx = eyePointerX - (bounds.left + bounds.width / 2), dy = eyePointerY - (bounds.top + bounds.height / 2)
  const distance = Math.hypot(dx, dy)
  const scale = distance ? Math.min(1, distance / 100) * 2.5 / distance : 0
  const x = Math.round(dx * scale * 10) / 10, y = Math.round(dy * scale * 10) / 10
  if (x === lastEyeX && y === lastEyeY) return
  lastEyeX = x; lastEyeY = y
  trigger.style.setProperty('--nt-eye-x', `${x}px`)
  trigger.style.setProperty('--nt-eye-y', `${y}px`)
}
function onEyePointer(event: PointerEvent) {
  if (event.pointerType === 'touch') return
  eyePointerX = event.clientX; eyePointerY = event.clientY
  if (!eyeFrame) eyeFrame = window.requestAnimationFrame(updateEyes)
}
watch(() => props.support.enabled, enabled => { if (!enabled) close() })
watch([triggerEl, open], ([trigger], [previousTrigger]) => {
  if (previousTrigger !== trigger) {
    if (previousTrigger) eyeObserver?.unobserve(previousTrigger)
    if (trigger) eyeObserver?.observe(trigger)
    eyeVisible = !!trigger
  }
  syncEyeTracking()
})
onMounted(() => {
  document.addEventListener('pointerdown', onDocumentPointer)
  document.addEventListener('focusin', onDocumentFocus)
  motionQuery = window.matchMedia('(prefers-reduced-motion: reduce)')
  pointerQuery = window.matchMedia('(hover: hover) and (pointer: fine)')
  motionQuery.addEventListener('change', syncEyeTracking)
  pointerQuery.addEventListener('change', syncEyeTracking)
  document.addEventListener('visibilitychange', syncEyeTracking)
  if (typeof IntersectionObserver !== 'undefined') {
    eyeObserver = new IntersectionObserver(entries => {
      for (const entry of entries) if (entry.target === triggerEl.value) eyeVisible = entry.isIntersecting
      syncEyeTracking()
    })
    if (triggerEl.value) eyeObserver.observe(triggerEl.value)
  }
  syncEyeTracking()
})
onUnmounted(() => {
  document.removeEventListener('pointerdown', onDocumentPointer)
  document.removeEventListener('focusin', onDocumentFocus)
  window.removeEventListener('pointermove', onEyePointer)
  document.removeEventListener('visibilitychange', syncEyeTracking)
  motionQuery?.removeEventListener('change', syncEyeTracking)
  pointerQuery?.removeEventListener('change', syncEyeTracking)
  eyeObserver?.disconnect()
  resetEyes()
})
</script>

<style scoped>
.nt-support-widget { position: fixed; right: max(22px, env(safe-area-inset-right)); bottom: calc(22px + env(safe-area-inset-bottom)); z-index: 55; }
.nt-support-trigger { width: 54px; height: 54px; display: grid; place-items: center; margin-left: auto; border: 1px solid rgba(163, 255, 224, .28); border-radius: 50%; background: #2dd4a7; color: #04110d; box-shadow: 0 6px 24px #0005, inset 0 1px 0 #ffffff4d; transition: background-color 160ms, transform 160ms; }
.nt-support-trigger:hover { background: #5eead4; transform: translateY(-2px); }
.nt-support-eyes { display: flex; align-items: center; justify-content: center; gap: 4px; pointer-events: none; }
.nt-support-eye { display: grid; place-items: center; width: 15px; height: 17px; border-radius: 50%; background: #fff; box-shadow: inset 0 -1px 0 #052f2514; transform-origin: center; }
.nt-support-pupil { display: block; width: 6px; height: 7px; border-radius: 50%; background: #123f34; transform: translate(var(--nt-eye-x, 0), var(--nt-eye-y, 0)); transition: transform 100ms ease-out; }
.nt-support-trigger:hover .nt-support-eye, .nt-support-trigger:focus-visible .nt-support-eye { animation: nt-support-blink 300ms ease-in-out; }
@keyframes nt-support-blink { 0%, 100% { transform: scaleY(1); } 42% { transform: scaleY(.22); } 72% { transform: scaleY(1.06); } }
.nt-support-panel { position: absolute; right: 0; bottom: 68px; width: 364px; max-width: calc(100vw - 32px); max-height: min(640px, calc(100dvh - 120px)); overflow: auto; border: 1px solid rgba(94, 234, 192, .25); border-radius: 20px; background: #0e1218; box-shadow: 0 24px 80px #0009; }
.nt-support-heading { display: flex; align-items: flex-start; gap: 12px; padding: 22px 20px 18px; border-bottom: 1px solid var(--ui-border); background: linear-gradient(115deg, #2dd4a711, transparent); }
.nt-support-heading-icon { display: grid; flex-shrink: 0; place-items: center; width: 42px; height: 42px; border-radius: 12px; background: #2dd4a71a; color: var(--ui-accent); }
.nt-support-close { display: grid; place-items: center; flex-shrink: 0; width: 28px; height: 28px; border-radius: 50%; color: var(--ui-text-secondary); }
.nt-support-close:hover { background: var(--ui-bg-soft); }
.nt-support-tabs { display: grid; grid-template-columns: 1fr 1fr; gap: 4px; margin: 18px 20px 0; padding: 4px; border: 1px solid var(--ui-border); border-radius: 12px; background: #07090d; }
.nt-support-tabs button { display: flex; justify-content: center; align-items: center; gap: 8px; padding: 9px 6px; border-radius: 8px; color: var(--ui-text-muted); font-size: 13px; font-weight: 600; }
.nt-support-tabs button[aria-selected="true"] { background: var(--ui-bg-soft); color: var(--ui-accent); box-shadow: inset 0 1px 0 #ffffff0d; }
.nt-support-content { padding: 22px 20px 18px; }
.nt-support-qrs { display: flex; justify-content: center; gap: 14px; }
.nt-support-qrs figure { flex: 1; min-width: 0; max-width: 192px; text-align: center; }
.nt-support-qr { border: 1px solid #ffffff26; padding: 8px; border-radius: 12px; background: #fff; }
.nt-support-qr img { display: block; width: 100%; aspect-ratio: 1; height: auto; object-fit: contain; }
.nt-support-qrs figcaption { margin-top: 10px; color: var(--ui-text-secondary); font-size: 12px; }
.nt-support-empty { padding: 4px 0 18px; text-align: center; font-size: 13px; line-height: 1.8; color: var(--ui-text-secondary); }
.nt-support-hours { display: flex; justify-content: center; align-items: center; gap: 6px; margin: 18px 0 14px; font-size: 12px; color: var(--ui-accent); line-height: 1.6; overflow-wrap: anywhere; }
.nt-support-action { display: flex; justify-content: center; align-items: center; gap: 8px; width: 100%; min-height: 44px; margin-top: 18px; padding: 10px 12px; border: 1px solid #a3ffe033; border-radius: 10px; color: #04110d; background: var(--ui-accent); font-size: 13px; font-weight: 700; text-align: center; }
.nt-support-action:hover { background: #5eead4; }
.nt-support-email { display: block; margin-top: 14px; color: var(--ui-text-secondary); font-size: 12px; text-align: center; overflow-wrap: anywhere; }
.nt-support-order-note { margin-top: 14px; font-size: 11px; line-height: 1.7; text-align: center; color: var(--ui-text-muted); }
.nt-support-shortcuts { display: grid; gap: 10px; margin-top: 18px; }
.nt-support-shortcuts div { display: flex; align-items: center; justify-content: space-between; gap: 12px; min-height: 50px; padding: 12px; border: 1px solid var(--ui-border); border-radius: 10px; background: #07090d80; color: var(--ui-text-secondary); font-size: 12px; }
.nt-support-shortcuts dd { text-align: right; max-width: 60%; }
.nt-support-shortcuts kbd { white-space: nowrap; padding: 4px 6px; border: 1px solid var(--ui-border); border-bottom-width: 2px; border-radius: 5px; color: var(--ui-text-primary); font-family: var(--nt-font-mono); }
@media (max-width: 767px) {
  .nt-support-widget { right: max(16px, env(safe-area-inset-right)); bottom: calc(18px + env(safe-area-inset-bottom)); }
  .nt-support-widget--raised { bottom: calc(98px + env(safe-area-inset-bottom)); }
  .nt-support-widget--raised .nt-support-panel { max-height: calc(100dvh - 202px); }
  .nt-support-trigger { width: 48px; height: 48px; }
  .nt-support-panel { bottom: 62px; max-height: calc(100dvh - 106px); }
}
@media (prefers-reduced-motion: reduce) { .nt-support-trigger, .nt-support-pupil { transition: none; } .nt-support-trigger:hover { transform: none; } .nt-support-trigger .nt-support-eye { animation: none; } }
</style>
