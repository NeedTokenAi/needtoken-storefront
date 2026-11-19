<template>
  <section class="radar-calendar-card" aria-labelledby="radar-calendar-heading">
    <header class="cal-head">
      <h2 id="radar-calendar-heading">{{ t('needtoken.radarV2.calendarTitle') }}</h2>
      <p>{{ t('needtoken.radarV2.calendarSubtitle') }}</p>
    </header>

    <div
      ref="canvasEl"
      class="cal-stage"
      @mouseleave="closeTooltip"
      @focusout="onFocusOut"
      @keydown.esc.stop.prevent="closeTooltip"
    >
      <div ref="scrollEl" class="cal-scroll">
        <div class="cal-grid-wrap" :style="{ '--weeks': weeks.length }">
          <div class="cal-months" aria-hidden="true">
            <span class="cal-weekday-spacer"></span>
            <div class="cal-months-track">
              <span
                v-for="(label, weekIdx) in monthLabels"
                :key="weekIdx"
                class="cal-month-label"
                :style="{ gridColumn: weekIdx + 1 }"
              >{{ label }}</span>
            </div>
          </div>

          <div class="cal-body">
            <div class="cal-weekdays" aria-hidden="true">
              <span>{{ t('needtoken.radarV2.weekdays.mon') }}</span>
              <span></span>
              <span>{{ t('needtoken.radarV2.weekdays.wed') }}</span>
              <span></span>
              <span>{{ t('needtoken.radarV2.weekdays.fri') }}</span>
              <span></span>
              <span></span>
            </div>

            <div
              class="cal-weeks"
              role="group"
              :aria-label="t('needtoken.radarStats.calendarKeyboard')"
              @keydown="moveFocus"
            >
              <div v-for="(week, weekIndex) in weeks" :key="week[0]!.date" class="cal-week">
                <button
                  v-for="(day, dayIndex) in week"
                  :key="day.date"
                  :ref="el => setCellEl(el, weekIndex * 7 + dayIndex)"
                  type="button"
                  class="cal-cell"
                  :class="{ 'is-today': day.isToday, 'is-active': activeDay?.date === day.date }"
                  :data-state="cellState(day)"
                  :disabled="day.future"
                  :tabindex="!day.future && focusedIndex === weekIndex * 7 + dayIndex ? 0 : -1"
                  :aria-label="cellAriaLabel(day)"
                  @mouseenter="openTooltip(day, $event.currentTarget as HTMLElement)"
                  @focus="focusedIndex = weekIndex * 7 + dayIndex; openTooltip(day, $event.currentTarget as HTMLElement)"
                  @click="openTooltip(day, $event.currentTarget as HTMLElement)"
                >
                  <span class="sr-only">{{ cellAriaLabel(day) }}</span>
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>

      <Transition name="cal-tip">
        <div
          v-if="activeDay"
          ref="tooltipEl"
          class="cal-tooltip"
          :style="tooltipStyle"
          role="tooltip"
        >
          <p class="cal-tip-date">{{ formatFullDay(activeDay.date) }}</p>
          <template v-if="activeDay.records.length">
            <div
              v-for="record in activeDay.records.slice(0, 2)"
              :key="record.id"
              class="cal-tip-record"
            >
              <p class="cal-tip-kind" :data-type="record.type">
                <i aria-hidden="true"></i>
                <span>{{ t(`needtoken.radarV2.types.${record.type}`) }}</span>
              </p>
              <p class="cal-tip-text">{{ record.text }}</p>
            </div>
          </template>
          <p v-else class="cal-tip-empty">{{ t('needtoken.radarV2.noResetDay') }}</p>
        </div>
      </Transition>
    </div>

    <footer class="cal-footer">
      <p class="cal-total">{{ t('needtoken.radarV2.calendarTotal', { count: periodTotal }) }}</p>
      <div class="cal-legend">
        <span><i class="legend-regular" aria-hidden="true"></i>{{ t('needtoken.radarV2.types.regular') }}</span>
        <span><i class="legend-banked" aria-hidden="true"></i>{{ t('needtoken.radarV2.types.banked') }}</span>
      </div>
    </footer>
  </section>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, ref, type ComponentPublicInstance } from 'vue'
import { useI18n } from 'vue-i18n'
import { resetCalendarWithRecords, type ResetCalendarRichDay, type ResetRecord } from '../lib/resetRadar'

const props = defineProps<{
  records: readonly ResetRecord[]
  now: number
}>()

const { t, locale } = useI18n()
const weeks = computed(() => resetCalendarWithRecords(props.records, props.now))
const periodTotal = computed(() =>
  weeks.value.flat().reduce((sum, day) => sum + day.regular + day.banked, 0),
)

const monthLabels = computed(() => {
  const formatter = new Intl.DateTimeFormat(locale.value, { month: 'short', timeZone: 'UTC' })
  let lastShownIdx = -4
  let lastMonth = -1
  return weeks.value.map((week, idx) => {
    const firstOfMonth = week.find(day => day.date.endsWith('-01'))
    if (!firstOfMonth) return ''
    const dateObj = new Date(`${firstOfMonth.date}T00:00:00Z`)
    const month = dateObj.getUTCMonth()
    if (month === lastMonth || idx - lastShownIdx < 3) return ''
    lastMonth = month
    lastShownIdx = idx
    return formatter.format(dateObj)
  })
})

const canvasEl = ref<HTMLElement | null>(null)
const scrollEl = ref<HTMLElement | null>(null)
const tooltipEl = ref<HTMLElement | null>(null)
const activeDay = ref<ResetCalendarRichDay | null>(null)
const tooltipStyle = ref<Record<string, string>>({ left: '0px', top: '0px', visibility: 'hidden' })

const lastValidIndex = computed(() => {
  const flat = weeks.value.flat()
  for (let i = flat.length - 1; i >= 0; i--) {
    if (!flat[i]!.future) return i
  }
  return 0
})
const focusedIndex = ref(lastValidIndex.value)
const cellEls: Array<HTMLButtonElement | null> = []

function setCellEl(el: Element | ComponentPublicInstance | null, index: number) {
  cellEls[index] = el instanceof HTMLButtonElement ? el : null
}

function cellState(day: ResetCalendarRichDay) {
  if (day.future) return 'future'
  if (day.regular && day.banked) return 'both'
  if (day.banked) return 'banked'
  if (day.regular) return 'regular'
  return 'empty'
}

function formatFullDay(dateStr: string) {
  const dateObj = new Date(`${dateStr}T00:00:00Z`)
  if (locale.value.startsWith('zh')) {
    const year = dateObj.getUTCFullYear()
    const month = dateObj.getUTCMonth() + 1
    const day = dateObj.getUTCDate()
    const weekday = new Intl.DateTimeFormat(locale.value, { weekday: 'short', timeZone: 'UTC' }).format(dateObj)
    return `${year}年${month}月${day}日${weekday}`
  }
  return new Intl.DateTimeFormat(locale.value, {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    weekday: 'short',
    timeZone: 'UTC',
  }).format(dateObj)
}

function cellAriaLabel(day: ResetCalendarRichDay) {
  const dateText = formatFullDay(day.date)
  if (!day.regular && !day.banked) return `${dateText} · ${t('needtoken.radarV2.noResetDay')}`
  const parts: string[] = []
  if (day.regular) parts.push(`${t('needtoken.radarV2.types.regular')} ${day.regular}`)
  if (day.banked) parts.push(`${t('needtoken.radarV2.types.banked')} ${day.banked}`)
  return `${dateText} · ${parts.join(' / ')}`
}

async function openTooltip(day: ResetCalendarRichDay, anchorEl: HTMLElement | null) {
  if (day.future || !anchorEl || !canvasEl.value) return
  activeDay.value = day
  await nextTick()
  if (!tooltipEl.value || !canvasEl.value) return
  const stage = canvasEl.value.getBoundingClientRect()
  const cell = anchorEl.getBoundingClientRect()
  const tip = tooltipEl.value.getBoundingClientRect()
  const centerX = cell.left - stage.left + cell.width / 2
  const left = Math.max(8, Math.min(centerX - tip.width / 2, stage.width - tip.width - 8))
  const aboveTop = cell.top - stage.top - tip.height - 10
  const belowTop = cell.bottom - stage.top + 10
  const top = aboveTop >= -16 ? aboveTop : belowTop
  tooltipStyle.value = {
    left: `${Math.round(left)}px`,
    top: `${Math.round(top)}px`,
    visibility: 'visible',
  }
}

function closeTooltip() {
  activeDay.value = null
  tooltipStyle.value = { left: '0px', top: '0px', visibility: 'hidden' }
}

function onFocusOut(event: FocusEvent) {
  if (event.relatedTarget instanceof Node && canvasEl.value?.contains(event.relatedTarget)) return
  closeTooltip()
}

function moveFocus(event: KeyboardEvent) {
  const stepMap: Record<string, number> = {
    ArrowUp: -1,
    ArrowDown: 1,
    ArrowLeft: -7,
    ArrowRight: 7,
  }
  const step = stepMap[event.key]
  if (step === undefined) return
  event.preventDefault()
  const target = Math.max(0, Math.min(lastValidIndex.value, focusedIndex.value + step))
  focusedIndex.value = target
  cellEls[target]?.focus()
}

onMounted(() => {
  if (scrollEl.value) {
    scrollEl.value.scrollLeft = scrollEl.value.scrollWidth
  }
})
</script>

<style scoped>
.radar-calendar-card {
  width: 100%;
  padding: 24px 28px 22px;
  border: 1px solid var(--ui-border);
  border-radius: 20px;
  background: var(--ui-bg-elevated);
  box-shadow: var(--nt-card-shadow);
  --cal-regular: #15803d;
  --cal-banked: #d98a1c;
  --cal-empty: rgba(15, 23, 20, 0.055);
  --cal-today-ring: rgba(15, 23, 20, 0.28);
}

:global(.dark) .radar-calendar-card {
  --cal-regular: #22c58b;
  --cal-banked: #eab338;
  --cal-empty: rgba(148, 163, 184, 0.16);
  --cal-empty-border: rgba(148, 163, 184, 0.09);
  --cal-today-ring: rgba(232, 237, 243, 0.5);
}

.cal-head h2 {
  color: var(--ui-text-primary);
  font-size: 16px;
  font-weight: 600;
  line-height: 1.4;
}

.cal-head p {
  margin-top: 4px;
  color: var(--ui-text-muted);
  font-size: 13px;
  line-height: 1.5;
}

.cal-stage {
  position: relative;
  margin-top: 18px;
}

.cal-scroll {
  overflow-x: auto;
  overflow-y: visible;
  padding-bottom: 6px;
}

.cal-grid-wrap {
  min-width: 720px;
  width: 100%;
}

.cal-months {
  display: grid;
  grid-template-columns: 24px minmax(0, 1fr);
  gap: 8px;
  align-items: end;
  margin-bottom: 6px;
}

.cal-months-track {
  display: grid;
  grid-template-columns: repeat(var(--weeks, 53), minmax(0, 1fr));
  gap: 4px;
}

.cal-month-label {
  color: var(--ui-text-muted);
  font-size: 11px;
  line-height: 1.3;
  white-space: nowrap;
}

.cal-body {
  display: grid;
  grid-template-columns: 24px minmax(0, 1fr);
  gap: 8px;
  align-items: stretch;
}

.cal-weekdays {
  display: grid;
  grid-template-rows: repeat(7, minmax(0, 1fr));
  gap: 4px;
  align-items: center;
  color: var(--ui-text-muted);
  font-size: 11px;
  line-height: 1;
}

.cal-weeks {
  display: grid;
  grid-template-columns: repeat(var(--weeks, 53), minmax(0, 1fr));
  gap: 4px;
}

.cal-week {
  display: grid;
  grid-template-rows: repeat(7, minmax(0, 1fr));
  gap: 4px;
}

.cal-cell {
  display: block;
  width: 100%;
  aspect-ratio: 1;
  border: 0;
  border-radius: 3.5px;
  background: var(--cal-empty);
  box-shadow: inset 0 0 0 1px var(--cal-empty-border, transparent);
  cursor: pointer;
  transition: transform 140ms var(--nt-motion-ease, ease-out), box-shadow 140ms ease;
}

.cal-cell[data-state="regular"] {
  background: var(--cal-regular);
}

.cal-cell[data-state="banked"] {
  background: var(--cal-banked);
}

.cal-cell[data-state="both"] {
  background: linear-gradient(135deg, var(--cal-regular) 50%, var(--cal-banked) 50%);
}

.cal-cell[data-state="future"] {
  visibility: hidden;
  cursor: default;
}

.cal-cell.is-today[data-state="empty"] {
  box-shadow: inset 0 0 0 1.5px var(--cal-today-ring);
}

.cal-cell:hover:not(:disabled),
.cal-cell.is-active:not(:disabled) {
  transform: scale(1.16);
  box-shadow: 0 0 0 2px var(--ui-bg-elevated), 0 0 0 3.5px var(--ui-text-secondary);
  z-index: 2;
}

.cal-cell:focus-visible {
  outline: 2px solid var(--ui-accent);
  outline-offset: 2px;
}

.cal-tooltip {
  position: absolute;
  z-index: 20;
  width: min(300px, calc(100% - 16px));
  padding: 14px 16px;
  border: 1px solid var(--ui-border-strong);
  border-radius: 14px;
  background: var(--ui-bg-elevated);
  box-shadow: var(--nt-popover-shadow);
  pointer-events: none;
}

.cal-tip-date {
  color: var(--ui-text-primary);
  font-size: 13px;
  font-weight: 600;
  line-height: 1.4;
}

.cal-tip-record {
  margin-top: 8px;
}

.cal-tip-record + .cal-tip-record {
  padding-top: 8px;
  border-top: 1px solid var(--ui-border);
}

.cal-tip-kind {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  font-weight: 600;
  color: var(--cal-regular);
}

.cal-tip-kind[data-type="banked"] {
  color: var(--cal-banked);
}

.cal-tip-kind i {
  width: 6px;
  height: 6px;
  border-radius: 999px;
  background: currentColor;
}

.cal-tip-text {
  margin-top: 6px;
  color: var(--ui-text-secondary);
  font-size: 12.5px;
  line-height: 1.55;
  display: -webkit-box;
  -webkit-line-clamp: 3;
  -webkit-box-orient: vertical;
  overflow: hidden;
  overflow-wrap: anywhere;
}

.cal-tip-empty {
  margin-top: 6px;
  color: var(--ui-text-muted);
  font-size: 12px;
  line-height: 1.5;
}

.cal-footer {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-top: 14px;
}

.cal-total {
  color: var(--ui-text-muted);
  font-size: 12.5px;
  font-variant-numeric: tabular-nums;
}

.cal-legend {
  display: inline-flex;
  align-items: center;
  gap: 16px;
  color: var(--ui-text-secondary);
  font-size: 12.5px;
}

.cal-legend > span {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.cal-legend i {
  display: block;
  width: 10px;
  height: 10px;
  border-radius: 2.5px;
}

.legend-regular {
  background: var(--cal-regular);
}

.legend-banked {
  background: var(--cal-banked);
}

.cal-tip-enter-active,
.cal-tip-leave-active {
  transition: opacity 130ms ease-out;
}
.cal-tip-enter-from,
.cal-tip-leave-to {
  opacity: 0;
}

@media (prefers-reduced-motion: no-preference) {
  .cal-tip-enter-active,
  .cal-tip-leave-active {
    transition: opacity 140ms ease-out, transform 160ms var(--nt-motion-ease, ease-out);
  }
  .cal-tip-enter-from,
  .cal-tip-leave-to {
    opacity: 0;
    transform: translateY(4px) scale(0.98);
  }
}

.sr-only {
  position: absolute;
  width: 1px;
  height: 1px;
  padding: 0;
  overflow: hidden;
  clip: rect(0, 0, 0, 0);
  white-space: nowrap;
  border: 0;
}

@media (max-width: 640px) {
  .radar-calendar-card {
    padding: 18px 16px;
    border-radius: 16px;
  }
}

@media (prefers-reduced-motion: reduce) {
  .cal-cell {
    transition: none;
  }
  .cal-cell:hover:not(:disabled),
  .cal-cell.is-active:not(:disabled) {
    transform: none;
  }
}
</style>
