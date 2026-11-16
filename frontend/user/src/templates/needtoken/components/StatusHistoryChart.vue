<template>
  <div ref="widgetEl" class="status-history-widget" @mouseleave="onWidgetLeave" @focusout="onFocusOut" @keydown="onWidgetKeydown">
    <p :id="helpId" class="sr-only">{{ t('needtoken.statusStrip.daily.keyboardHelp') }}</p>
    <div class="status-history-bars" role="group" :aria-label="t('needtoken.statusStrip.daily.chooseDay')" :aria-describedby="helpId">
      <button v-for="(day, index) in days" :key="day.date" :ref="element => setButton(element, index)" type="button" class="status-history-day" :data-date="day.date" :data-status="day.tone" :data-color="publicStatusColor(day.tone)" :tabindex="day.date === activeDate ? 0 : -1" :aria-label="`${formatDate(day.date)} UTC · ${t(`needtoken.statusStrip.component.${day.tone}`)}`" :aria-controls="detailId" :aria-expanded="opened && day.date === activeDate" @mouseenter="showDay(index)" @focus="showDay(index)" @click="pinDay(index)" @keydown="onDayKeydown($event, index)"><i :class="{ 'has-incident': day.records.length > 0 }" aria-hidden="true"></i></button>
    </div>
    <span v-if="opened" class="status-day-bridge" :style="bridgeStyle" aria-hidden="true"></span>
    <Transition name="status-day" @before-enter="showEnteringPanel" @before-leave="hideLeavingPanel">
      <section v-if="opened && selectedDay" :id="detailId" ref="detailEl" class="status-day-detail" :data-date="selectedDay.date" :data-side="detailSide" :style="detailStyle" role="region" :aria-label="t('needtoken.statusStrip.daily.detailLabel', { date: formatDate(selectedDay.date), product: productLabel })">
        <header><time :datetime="selectedDay.date" :aria-label="`${formatDate(selectedDay.date)} UTC`"><strong>{{ formatDay(selectedDay.date) }}</strong><span class="status-day-date-meta"><span>{{ formatMonth(selectedDay.date) }}</span><span>{{ formatWeekday(selectedDay.date) }}</span><small>UTC</small></span></time><button type="button" class="status-day-close" :aria-label="t('needtoken.statusStrip.daily.close')" @click="closeDetail(true)"><X :size="14" aria-hidden="true" /></button></header>
        <div class="status-day-product"><strong>{{ productLabel }}</strong><span :data-color="publicStatusColor(selectedDay.tone)"><i aria-hidden="true"></i>{{ t(`needtoken.statusStrip.component.${selectedDay.tone}`) }}</span></div>
        <ul v-if="selectedDay.records.length" class="status-day-records"><li v-for="record in selectedDay.records.slice(0, 3)" :key="record.id || record.name" :data-color="publicStatusColor(publicIncidentTone(record))"><i class="status-day-fault" aria-hidden="true"></i><a :href="record.url || sourceUrl" target="_blank" rel="noopener noreferrer">{{ record.name }}<ArrowUpRight :size="12" aria-hidden="true" /></a></li></ul>
        <p v-else class="status-day-empty">{{ t(historyReady ? 'needtoken.statusStrip.daily.noFaultRecords' : 'needtoken.statusStrip.daily.unknownDay') }}</p>
        <p class="status-day-coverage">{{ t(historyReady ? 'needtoken.statusStrip.daily.partialCoverage' : 'needtoken.statusStrip.daily.historyUnavailable') }}</p>
        <div v-if="selectedDay.date === days[days.length - 1]?.date && currentFresh && checkedAt" class="status-day-current" :data-status="currentTone" :data-color="publicStatusColor(currentTone)"><i aria-hidden="true"></i><span>{{ t('needtoken.statusStrip.daily.currentCheck', { time: formatCheckTime(checkedAt), status: t(`needtoken.statusStrip.component.${currentTone}`) }) }}</span></div>
        <nav class="status-day-navigation" :aria-label="t('needtoken.statusStrip.daily.navigate')"><button type="button" :aria-disabled="activeIndex <= 0" :aria-label="t('needtoken.statusStrip.daily.previous')" @click="moveDay(-1)"><ChevronLeft :size="16" aria-hidden="true" /></button><span>{{ t('needtoken.statusStrip.daily.dayPosition', { index: activeIndex + 1, total: days.length }) }}</span><button type="button" :aria-disabled="activeIndex >= days.length - 1" :aria-label="t('needtoken.statusStrip.daily.next')" @click="moveDay(1)"><ChevronRight :size="16" aria-hidden="true" /></button></nav>
      </section>
    </Transition>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, useId, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { ArrowUpRight, ChevronLeft, ChevronRight, X } from 'lucide-vue-next'
import { publicIncidentTone, publicStatusColor, type PublicComponentStatus, type PublicHistoryDay } from '../lib/publicStatus'

const props = defineProps<{
  days: PublicHistoryDay[]
  productLabel: string
  historyReady: boolean
  sourceUrl: string
  currentTone: PublicComponentStatus
  currentFresh: boolean
  checkedAt: string | null
}>()
const { t, locale } = useI18n()
const id = useId().replace(/[^a-zA-Z0-9_-]/g, '')
const detailId = `status-day-${id}`, helpId = `status-day-help-${id}`
const widgetEl = ref<HTMLElement | null>(null), detailEl = ref<HTMLElement | null>(null)
const buttons: HTMLButtonElement[] = []
const activeDate = ref(props.days[props.days.length - 1]?.date || '')
const opened = ref(false), pinned = ref(false)
const detailSide = ref<'above' | 'below'>('above')
const detailStyle = ref({ left: '0px', top: '0px', maxHeight: '320px', visibility: 'hidden' as 'visible' | 'hidden' })
const bridgeStyle = ref({ left: '0px', top: '0px', width: '0px', height: '0px' })
const activeIndex = computed(() => Math.max(0, props.days.findIndex(day => day.date === activeDate.value)))
const selectedDay = computed(() => props.days.find(day => day.date === activeDate.value) ?? null)
const formatDate = (value: string) => new Date(`${value}T00:00:00Z`).toLocaleDateString(locale.value, { year: 'numeric', month: 'short', day: 'numeric', timeZone: 'UTC' })
const formatDay = (value: string) => new Intl.DateTimeFormat(locale.value, { day: 'numeric', timeZone: 'UTC' }).formatToParts(new Date(`${value}T00:00:00Z`)).find(part => part.type === 'day')?.value || ''
const formatMonth = (value: string) => new Date(`${value}T00:00:00Z`).toLocaleDateString(locale.value, { month: 'short', timeZone: 'UTC' })
const formatWeekday = (value: string) => new Date(`${value}T00:00:00Z`).toLocaleDateString(locale.value, { weekday: 'short', timeZone: 'UTC' })
const formatCheckTime = (value: string) => `${new Date(value).toLocaleTimeString(locale.value, { hour: '2-digit', minute: '2-digit', timeZone: 'UTC' })} UTC`
const setButton = (element: unknown, index: number) => { if (element instanceof HTMLButtonElement) buttons[index] = element }
const hideLeavingPanel = (element: Element) => { element.setAttribute('inert', ''); element.setAttribute('aria-hidden', 'true') }
const showEnteringPanel = (element: Element) => { element.removeAttribute('inert'); element.removeAttribute('aria-hidden') }

function positionDetail() {
  if (!opened.value || !widgetEl.value || !detailEl.value) return
  const widget = widgetEl.value.getBoundingClientRect()
  const track = widgetEl.value.querySelector('.status-history-bars')?.getBoundingClientRect()
  const panel = widgetEl.value.closest('.status-popover')?.getBoundingClientRect()
  if (!track || !panel) return
  const detail = detailEl.value.getBoundingClientRect()
  const maxHeight = Math.max(80, Math.min(320, panel.height - 24, innerHeight - 32))
  const height = Math.min(detail.height, maxHeight)
  const anchor = track.left + (activeIndex.value + .5) / Math.max(1, props.days.length) * track.width
  const left = Math.max(panel.left + 12, Math.min(anchor - detail.width / 2, panel.right - detail.width - 12))
  const above = track.top - height - 8
  const below = track.bottom + 8
  detailSide.value = above >= panel.top + 12 || below + height > panel.bottom - 12 ? 'above' : 'below'
  const preferredTop = detailSide.value === 'above' ? above : below
  const top = Math.max(panel.top + 12, Math.min(preferredTop, panel.bottom - height - 12))
  detailStyle.value = { left: `${left - widget.left}px`, top: `${top - widget.top}px`, maxHeight: `${maxHeight}px`, visibility: 'visible' }
  bridgeStyle.value = {
    left: `${left - widget.left}px`, width: `${detail.width}px`,
    top: `${(detailSide.value === 'above' ? top + height : track.bottom) - widget.top}px`,
    height: `${Math.max(2, detailSide.value === 'above' ? track.top - top - height + 2 : top - track.bottom + 2)}px`,
  }
}
function showDay(index: number) {
  const day = props.days[index]
  if (!day) return
  if (day.date !== activeDate.value) pinned.value = false
  activeDate.value = day.date
  opened.value = true
  void nextTick(positionDetail)
}
function pinDay(index: number) {
  if (opened.value && pinned.value && props.days[index]?.date === activeDate.value) { closeDetail(); return }
  showDay(index)
  pinned.value = true
}
function closeDetail(restoreFocus = false) {
  if (restoreFocus) buttons[activeIndex.value]?.focus({ preventScroll: true })
  opened.value = false
  pinned.value = false
  detailStyle.value.visibility = 'hidden'
}
function moveDay(step: number) {
  const index = activeIndex.value + step
  if (index < 0 || index >= props.days.length) return
  showDay(index)
  pinned.value = true
}
function onDayKeydown(event: KeyboardEvent, index: number) {
  const next = event.key === 'ArrowRight' ? index + 1 : event.key === 'ArrowLeft' ? index - 1
    : event.key === 'Home' ? 0 : event.key === 'End' ? props.days.length - 1
      : event.key === 'PageUp' ? index - 7 : event.key === 'PageDown' ? index + 7 : null
  if (next === null) return
  event.preventDefault()
  event.stopPropagation()
  const target = Math.max(0, Math.min(props.days.length - 1, next))
  showDay(target)
  buttons[target]?.focus({ preventScroll: true })
}
function onWidgetKeydown(event: KeyboardEvent) {
  if (event.key !== 'Escape' || !opened.value) return
  event.preventDefault()
  event.stopPropagation()
  closeDetail(true)
}
function onWidgetLeave() {
  if (!pinned.value && !widgetEl.value?.contains(document.activeElement)) closeDetail()
}
function onFocusOut(event: FocusEvent) {
  if (event.relatedTarget instanceof Node && widgetEl.value?.contains(event.relatedTarget)) return
  closeDetail()
}
function onOutsidePointer(event: PointerEvent) {
  if (event.target instanceof Node && !widgetEl.value?.contains(event.target)) closeDetail()
}
watch(() => props.days, () => {
  if (!props.days.some(day => day.date === activeDate.value)) { activeDate.value = props.days[props.days.length - 1]?.date || ''; closeDetail() }
  if (opened.value) void nextTick(positionDetail)
})
onMounted(() => {
  document.addEventListener('pointerdown', onOutsidePointer)
  window.addEventListener('resize', positionDetail)
  window.addEventListener('scroll', positionDetail, true)
})
onUnmounted(() => {
  document.removeEventListener('pointerdown', onOutsidePointer)
  window.removeEventListener('resize', positionDetail)
  window.removeEventListener('scroll', positionDetail, true)
})
</script>

<style scoped>
.status-history-widget { position: relative; margin-top: 8px; --status-green: #3bcba0; --status-yellow: #c5a65e; --status-red: #e06a5c; --status-gray: #67737c; }
.status-history-bars { display: grid; grid-template-columns: repeat(90, minmax(0, 1fr)); gap: 2px; min-height: 36px; }
.status-history-day { display: flex; align-items: center; min-width: 0; height: 36px; padding: 4px 0; border: 0; border-radius: 2px; background: transparent; --day-color: var(--status-gray); }
.status-history-day i { display: block; width: 100%; height: 28px; border-radius: 1px; background: var(--day-color); opacity: .5; transition: opacity 120ms ease; }
.status-history-day[data-color="green"], [data-color="green"] { --day-color: var(--status-green); }
.status-history-day[data-color="yellow"], [data-color="yellow"] { --day-color: var(--status-yellow); }
.status-history-day[data-color="red"], [data-color="red"] { --day-color: var(--status-red); }
.status-history-day[data-color="gray"], [data-color="gray"] { --day-color: var(--status-gray); }
.status-history-day[data-color="green"] i, .status-history-day[data-color="yellow"] i, .status-history-day[data-color="red"] i, .status-history-day:hover i, .status-history-day[aria-expanded="true"] i { opacity: 1; }
.status-history-day:focus-visible { outline: 2px solid var(--ui-accent); outline-offset: 2px; }
.status-day-bridge { position: absolute; z-index: 4; }
.status-day-detail { position: absolute; z-index: 5; width: min(300px, calc(100vw - 96px)); overflow-y: auto; padding: 16px; border: 1px solid var(--ui-border-strong); border-radius: 12px; background: var(--ui-bg-elevated); box-shadow: 0 8px 28px rgba(0,0,0,.3); color: var(--ui-text-secondary); }
.status-day-detail header { display: flex; align-items: center; justify-content: space-between; gap: 12px; }
.status-day-detail header time { display: flex; align-items: center; gap: 12px; color: var(--ui-text-primary); }
.status-day-detail header time > strong { font-size: 44px; line-height: 1; font-weight: 500; font-variant-numeric: tabular-nums; }
.status-day-date-meta { display: grid; gap: 0; color: var(--ui-text-secondary); font-size: 12px; line-height: 16px; }
.status-day-date-meta small { color: var(--ui-text-muted); font-size: 11px; }
.status-day-close { display: grid; flex: none; width: 28px; height: 28px; place-items: center; color: var(--ui-text-muted); }
.status-day-product { display: flex; align-items: baseline; justify-content: space-between; gap: 12px; margin-top: 12px; font-size: 12px; line-height: 20px; }
.status-day-product strong { color: var(--ui-text-primary); font-weight: 500; }
.status-day-product > span { display: inline-flex; align-items: center; gap: 6px; color: var(--ui-text-muted); white-space: nowrap; }
.status-day-product i, .status-day-current > i { width: 6px; height: 6px; flex: none; border-radius: 50%; background: var(--day-color); }
.status-day-records { display: grid; gap: 12px; margin-top: 16px; }
.status-day-records li { display: flex; align-items: stretch; gap: 8px; min-width: 0; font-size: 12px; line-height: 20px; }
.status-day-fault { display: block; width: 3px; flex: none; border-radius: 2px; background: var(--day-color); }
.status-day-records a { display: inline; min-width: 0; color: var(--ui-text-secondary); overflow-wrap: anywhere; }
.status-day-records a svg { display: inline; margin-left: 4px; }
.status-day-empty { margin-top: 16px; color: var(--ui-text-secondary); font-size: 12px; line-height: 20px; }
.status-day-coverage { margin-top: 8px; color: var(--ui-text-muted); font-size: 11px; line-height: 18px; }
.status-day-current { display: flex; align-items: center; gap: 8px; margin-top: 12px; padding-top: 12px; border-top: 1px solid var(--ui-border); color: var(--ui-text-secondary); font-size: 11px; line-height: 18px; }
.status-day-navigation { display: flex; align-items: center; justify-content: space-between; gap: 12px; margin-top: 12px; color: var(--ui-text-muted); font-size: 11px; }
.status-day-navigation button { display: grid; width: 32px; height: 32px; place-items: center; border: 1px solid var(--ui-border); border-radius: 6px; color: var(--ui-text-secondary); }
.status-day-navigation button[aria-disabled="true"] { opacity: .35; }
.status-day-detail button:focus-visible, .status-day-detail a:focus-visible { outline: 2px solid var(--ui-accent); outline-offset: 2px; }
.status-day-detail a:hover, .status-day-detail button:hover { color: var(--ui-accent); }
.status-day-enter-active, .status-day-leave-active { transition: opacity 140ms ease, transform 140ms ease; }
.status-day-leave-active { pointer-events: none; }
.status-day-enter-from, .status-day-leave-to { opacity: 0; transform: translateY(3px); }
.sr-only { position: absolute; width: 1px; height: 1px; padding: 0; margin: -1px; overflow: hidden; clip: rect(0,0,0,0); white-space: nowrap; border: 0; }
@media (max-width: 640px) { .status-history-bars { gap: 1px; } }
@media (prefers-reduced-motion: reduce) { .status-history-day i, .status-day-enter-active, .status-day-leave-active { transition: none; }.status-day-enter-from, .status-day-leave-to { transform: none; } }
</style>
