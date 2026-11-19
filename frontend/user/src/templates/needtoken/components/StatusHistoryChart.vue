<template>
  <div
    ref="widgetEl"
    class="status-history-widget"
    :style="{ '--row-offset': `${(rowIndex ?? 0) * 55}ms` }"
    @mouseleave="onWidgetLeave"
    @focusout="onFocusOut"
    @keydown="onWidgetKeydown"
  >
    <p :id="helpId" class="sr-only">{{ t('needtoken.statusStrip.daily.keyboardHelp') }}</p>
    <div
      class="status-history-bars"
      role="group"
      :aria-label="`${productLabel} · ${t('needtoken.statusStrip.daily.chooseDay')}`"
      :aria-describedby="helpId"
    >
      <button
        v-for="(day, index) in days"
        :key="day.date"
        :ref="element => setButton(element, index)"
        type="button"
        class="status-history-day"
        :data-date="day.date"
        :data-status="day.status"
        :data-color="publicStatusColor(day.status)"
        :style="supportsSiblingIndex ? undefined : { '--sibling-index': index + 1 }"
        :tabindex="day.date === activeDate ? 0 : -1"
        :aria-label="`${formatDate(day.date)} · ${t(`needtoken.statusStrip.component.${day.status}`)}`"
        :aria-controls="detailId"
        :aria-expanded="opened && day.date === activeDate"
        @mouseenter="showDay(index)"
        @focus="showDay(index)"
        @click="pinDay(index)"
        @keydown="onDayKeydown($event, index)"
      >
        <i aria-hidden="true"></i>
      </button>
    </div>

    <span v-if="opened" class="status-day-bridge" :style="bridgeStyle" aria-hidden="true"></span>

    <Transition name="status-day" @before-enter="showEnteringPanel" @before-leave="hideLeavingPanel">
      <section
        v-if="opened && selectedDay"
        :id="detailId"
        ref="detailEl"
        class="status-day-detail"
        :data-date="selectedDay.date"
        :data-side="detailSide"
        :style="detailStyle"
        role="region"
        :aria-label="t('needtoken.statusStrip.daily.detailLabel', { date: formatDate(selectedDay.date), product: productLabel })"
      >
        <header class="status-day-head">
          <time :datetime="selectedDay.date">{{ formatDate(selectedDay.date) }}</time>
          <span class="status-day-badge" :data-color="publicStatusColor(selectedDay.status)">
            <i aria-hidden="true"></i>
            {{ t(`needtoken.statusStrip.component.${selectedDay.status}`) }}
          </span>
        </header>

        <ul v-if="selectedDay.events.length" class="status-day-records">
          <li v-for="(event, idx) in selectedDay.events.slice(0, 3)" :key="event.id || `${event.name}-${idx}`">
            <i class="status-day-fault" :data-color="publicStatusColor(selectedDay.status)" aria-hidden="true"></i>
            <a
              v-if="event.url"
              :href="event.url"
              target="_blank"
              rel="noopener noreferrer"
            >
              {{ event.name }}<ArrowUpRight :size="11" aria-hidden="true" />
            </a>
            <span v-else>{{ event.name }}</span>
          </li>
        </ul>
        <p v-else class="status-day-empty">
          {{ t(selectedDay.status === 'unknown' ? 'needtoken.statusStrip.series.unknownDayShort' : 'needtoken.statusStrip.series.noIncidentDay') }}
        </p>
      </section>
    </Transition>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, useId, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { ArrowUpRight } from 'lucide-vue-next'
import { publicStatusColor, type PublicStatusSeriesDay } from '../lib/publicStatus'

const props = defineProps<{
  days: PublicStatusSeriesDay[]
  productLabel: string
  sourceUrl: string
  rowIndex?: number
}>()

const { t, locale } = useI18n()
const id = useId().replace(/[^a-zA-Z0-9_-]/g, '')
const detailId = `status-day-${id}`
const helpId = `status-day-help-${id}`
const widgetEl = ref<HTMLElement | null>(null)
const detailEl = ref<HTMLElement | null>(null)
const buttons: HTMLButtonElement[] = []
const activeDate = ref(props.days[props.days.length - 1]?.date || '')
const opened = ref(false)
const pinned = ref(false)
const detailSide = ref<'above' | 'below'>('above')
const detailStyle = ref({ left: '0px', top: '0px', visibility: 'hidden' as 'visible' | 'hidden' })
const bridgeStyle = ref({ left: '0px', top: '0px', width: '0px', height: '0px' })

const supportsSiblingIndex = typeof CSS !== 'undefined'
  && typeof CSS.supports === 'function'
  && CSS.supports('animation-delay: calc(sibling-index() * 0.1s)')

const activeIndex = computed(() => Math.max(0, props.days.findIndex(day => day.date === activeDate.value)))
const selectedDay = computed(() => props.days.find(day => day.date === activeDate.value) ?? null)

const formatDate = (value: string) =>
  new Date(`${value}T00:00:00Z`).toLocaleDateString(locale.value, {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    timeZone: 'UTC',
  })

const setButton = (element: unknown, index: number) => {
  if (element instanceof HTMLButtonElement) buttons[index] = element
}

const hideLeavingPanel = (element: Element) => {
  element.setAttribute('inert', '')
  element.setAttribute('aria-hidden', 'true')
}
const showEnteringPanel = (element: Element) => {
  element.removeAttribute('inert')
  element.removeAttribute('aria-hidden')
}

function positionDetail() {
  if (!opened.value || !widgetEl.value || !detailEl.value) return
  const widget = widgetEl.value.getBoundingClientRect()
  const track = widgetEl.value.querySelector('.status-history-bars')?.getBoundingClientRect()
  const panel = widgetEl.value.closest('.status-popover')?.getBoundingClientRect()
  if (!track || !panel) return

  const detail = detailEl.value.getBoundingClientRect()
  const anchor = track.left + ((activeIndex.value + 0.5) / Math.max(1, props.days.length)) * track.width
  const left = Math.max(panel.left + 12, Math.min(anchor - detail.width / 2, panel.right - detail.width - 12))
  const above = track.top - detail.height - 8
  const below = track.bottom + 8
  detailSide.value = above >= panel.top + 8 || below + detail.height > panel.bottom - 8 ? 'above' : 'below'
  const preferredTop = detailSide.value === 'above' ? above : below
  const top = Math.max(panel.top + 8, Math.min(preferredTop, panel.bottom - detail.height - 8))

  detailStyle.value = {
    left: `${left - widget.left}px`,
    top: `${top - widget.top}px`,
    visibility: 'visible',
  }
  bridgeStyle.value = {
    left: `${left - widget.left}px`,
    width: `${detail.width}px`,
    top: `${(detailSide.value === 'above' ? top + detail.height : track.bottom) - widget.top}px`,
    height: `${Math.max(4, detailSide.value === 'above' ? track.top - top - detail.height + 4 : top - track.bottom + 4)}px`,
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
  if (opened.value && pinned.value && props.days[index]?.date === activeDate.value) {
    closeDetail()
    return
  }
  showDay(index)
  pinned.value = true
}

function closeDetail(restoreFocus = false) {
  if (restoreFocus) buttons[activeIndex.value]?.focus({ preventScroll: true })
  opened.value = false
  pinned.value = false
  detailStyle.value.visibility = 'hidden'
}

function onDayKeydown(event: KeyboardEvent, index: number) {
  const next = event.key === 'ArrowRight' ? index + 1
    : event.key === 'ArrowLeft' ? index - 1
      : event.key === 'Home' ? 0
        : event.key === 'End' ? props.days.length - 1
          : event.key === 'PageUp' ? index - 7
            : event.key === 'PageDown' ? index + 7
              : null
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
  if (!props.days.some(day => day.date === activeDate.value)) {
    activeDate.value = props.days[props.days.length - 1]?.date || ''
    closeDetail()
  }
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
.status-history-widget {
  position: relative;
  margin-top: 8px;
  --bar-green: #22c58b;
  --bar-yellow: #f2b830;
  --bar-red: #ec5f54;
  --bar-gray: rgba(15, 23, 20, 0.14);
}

:global(.dark) .status-history-widget {
  --bar-green: #34d399;
  --bar-yellow: #eab338;
  --bar-red: #f06a5d;
  --bar-gray: rgba(148, 163, 184, 0.22);
}

.status-history-bars {
  display: grid;
  grid-template-columns: repeat(90, minmax(0, 1fr));
  gap: 2px;
  height: 28px;
  align-items: center;
}

.status-history-day {
  display: flex;
  align-items: center;
  min-width: 0;
  height: 28px;
  padding: 2px 0;
  border: 0;
  border-radius: 2px;
  background: transparent;
  cursor: pointer;
  --day-color: var(--bar-gray);
}

.status-history-day i {
  display: block;
  width: 100%;
  height: 24px;
  border-radius: 1.5px;
  background: var(--day-color);
  transition: transform 140ms var(--nt-motion-ease, ease-out), filter 140ms ease;
}

.status-history-day[data-color="green"],
[data-color="green"] {
  --day-color: var(--bar-green);
}
.status-history-day[data-color="yellow"],
[data-color="yellow"] {
  --day-color: var(--bar-yellow);
}
.status-history-day[data-color="red"],
[data-color="red"] {
  --day-color: var(--bar-red);
}
.status-history-day[data-color="gray"],
[data-color="gray"] {
  --day-color: var(--bar-gray);
}

.status-history-day:hover i,
.status-history-day[aria-expanded="true"] i {
  transform: scaleY(1.14);
  filter: brightness(1.08);
}

.status-history-day:focus-visible {
  outline: 2px solid var(--ui-accent);
  outline-offset: 1px;
}

@media (prefers-reduced-motion: no-preference) {
  .status-history-day {
    --bar-step: 2.2ms;
    transform-origin: center bottom;
    animation: status-bar-rise 260ms cubic-bezier(0.22, 1, 0.36, 1) both;
    animation-delay: calc(var(--row-offset, 0ms) + var(--sibling-index, 1) * var(--bar-step));
  }

  @supports (animation-delay: calc(sibling-index() * 0.1s)) {
    .status-history-day {
      animation-delay: calc(var(--row-offset, 0ms) + sibling-index() * var(--bar-step));
    }
  }
}

@keyframes status-bar-rise {
  from {
    opacity: 0;
    transform: scaleY(0.25);
  }
  to {
    opacity: 1;
    transform: scaleY(1);
  }
}

.status-day-bridge {
  position: absolute;
  z-index: 6;
}

.status-day-detail {
  position: absolute;
  z-index: 8;
  width: min(272px, calc(100vw - 88px));
  padding: 12px 14px;
  border: 1px solid var(--ui-border-strong);
  border-radius: 12px;
  background: var(--ui-bg-elevated);
  box-shadow: var(--nt-popover-shadow);
  color: var(--ui-text-secondary);
}

.status-day-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}

.status-day-head time {
  color: var(--ui-text-primary);
  font-size: 12px;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
}

.status-day-badge {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  font-size: 11px;
  font-weight: 500;
  color: var(--ui-text-secondary);
  white-space: nowrap;
}

.status-day-badge i {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--day-color);
}

.status-day-records {
  display: grid;
  gap: 7px;
  margin-top: 9px;
  padding-top: 9px;
  border-top: 1px solid var(--ui-border);
}

.status-day-records li {
  display: flex;
  align-items: stretch;
  gap: 7px;
  min-width: 0;
  font-size: 11.5px;
  line-height: 1.45;
}

.status-day-fault {
  display: block;
  width: 2.5px;
  flex: none;
  border-radius: 2px;
  background: var(--day-color);
}

.status-day-records a,
.status-day-records span {
  min-width: 0;
  color: var(--ui-text-secondary);
  overflow-wrap: anywhere;
}

.status-day-records a {
  display: inline;
}

.status-day-records a:hover {
  color: var(--ui-accent);
}

.status-day-records a svg {
  display: inline;
  margin-left: 3px;
  vertical-align: -1px;
}

.status-day-empty {
  margin-top: 6px;
  font-size: 11.5px;
  line-height: 1.45;
  color: var(--ui-text-muted);
}

.status-day-enter-active,
.status-day-leave-active {
  transition: opacity 130ms ease-out;
}
.status-day-leave-active {
  pointer-events: none;
}
.status-day-enter-from,
.status-day-leave-to {
  opacity: 0;
}

@media (prefers-reduced-motion: no-preference) {
  .status-day-enter-active,
  .status-day-leave-active {
    transition: opacity 140ms ease-out, transform 160ms var(--nt-motion-ease, ease-out);
  }
  .status-day-enter-from,
  .status-day-leave-to {
    opacity: 0;
    transform: translateY(4px) scale(0.98);
  }
}

.sr-only {
  position: absolute;
  width: 1px;
  height: 1px;
  padding: 0;
  margin: -1px;
  overflow: hidden;
  clip: rect(0, 0, 0, 0);
  white-space: nowrap;
  border: 0;
}

@media (max-width: 640px) {
  .status-history-bars {
    gap: 1px;
  }
}
</style>
