<template>
  <section class="reset-overview" :aria-label="t('needtoken.radarStats.title')">
    <div class="reset-stats">
      <article class="reset-stat interval-stat">
        <div class="stat-heading"><h2>{{ t(stats.averageRemainingDays === null ? 'needtoken.radarStats.regularElapsed' : stats.averageRemainingDays > 0 ? 'needtoken.radarStats.averageRemaining' : 'needtoken.radarStats.averagePassed') }}</h2><span v-if="stats.averageIntervalDays !== null">{{ t('needtoken.radarStats.averageEvery', { days: number(stats.averageIntervalDays) }) }}</span></div>
        <p class="stat-value"><strong>{{ intervalValue }}</strong><span>{{ t('needtoken.radarLive.units.days') }}</span></p>
        <div class="interval-track" :title="t('needtoken.radarStats.averageNote')"><span :style="{ width: `${intervalProgress}%` }" /><i aria-hidden="true" /></div>
        <p class="stat-note">{{ stats.elapsedDays === null ? t('needtoken.radarStats.noRegular') : t('needtoken.radarStats.elapsedDays', { days: number(stats.elapsedDays) }) }}</p>
        <p class="stat-disclaimer">{{ t('needtoken.radarStats.averageNote') }}</p>
      </article>
      <article class="reset-stat">
        <div class="stat-heading"><h2>{{ t('needtoken.radarStats.recorded') }}</h2></div>
        <p class="stat-value"><strong>{{ stats.total }}</strong><span>{{ t('needtoken.radarStats.times') }}</span></p>
        <div class="type-track" :aria-label="t('needtoken.radarStats.typeCounts', { regular: stats.regular, banked: stats.banked })"><span class="regular" :style="{ flex: stats.regular }" /><span class="banked" :style="{ flex: stats.banked }" /></div>
        <div class="stat-legend"><span><i class="regular" />{{ t('needtoken.radarStats.regularCount', { count: stats.regular }) }}</span><span><i class="banked" />{{ t('needtoken.radarStats.bankedCount', { count: stats.banked }) }}</span></div>
      </article>
      <article class="reset-stat">
        <div class="stat-heading"><h2>{{ t('needtoken.radarStats.recent30') }}</h2></div>
        <p class="stat-value"><strong>{{ stats.recent30 }}</strong><span>{{ t('needtoken.radarStats.times') }}</span></p>
        <p class="stat-note recent-comparison">{{ t('needtoken.radarStats.previous30', { count: stats.previous30 }) }}</p>
      </article>
    </div>
    <section class="reset-calendar" aria-labelledby="reset-calendar-title">
      <div class="calendar-heading"><div><h2 id="reset-calendar-title">{{ t('needtoken.radarStats.calendar') }}</h2><p>{{ t('needtoken.radarStats.calendarIntro') }}</p></div><span>{{ t('needtoken.radarLive.timezone') }}</span></div>
      <div ref="calendarScroll" class="calendar-scroll">
        <div class="calendar-months" aria-hidden="true"><span v-for="(week, index) in weeks" :key="week[0]!.date" :style="{ gridColumn: index + 1 }">{{ monthLabel(week, index) }}</span></div>
        <div class="calendar-weeks" role="group" :aria-label="t('needtoken.radarStats.calendarKeyboard')" @keydown="moveFocus">
          <div v-for="(week, weekIndex) in weeks" :key="week[0]!.date" class="calendar-week">
            <div v-for="(day, dayIndex) in week" :key="day.date" class="calendar-cell" :class="{ 'align-left': weekIndex < 5, 'align-right': weekIndex > weeks.length - 6 }">
              <button :ref="element => setDayElement(element, weekIndex * 7 + dayIndex)" type="button" :tabindex="!day.future && focusedDay === weekIndex * 7 + dayIndex ? 0 : -1" :disabled="day.future" :data-state="dayState(day)" :aria-label="dayLabel(day)" @focus="focusedDay = weekIndex * 7 + dayIndex"><span class="sr-only">{{ dayLabel(day) }}</span></button>
              <span v-if="!day.future" class="calendar-tooltip" role="tooltip">{{ dayLabel(day) }}</span>
            </div>
          </div>
        </div>
      </div>
      <div class="calendar-footer"><p>{{ t(hasMore ? 'needtoken.radarStats.partialCoverage' : 'needtoken.radarStats.coverage', { count: records.length }) }}</p><div class="stat-legend"><span><i class="regular" />{{ t('needtoken.radarLive.types.regular') }}</span><span><i class="banked" />{{ t('needtoken.radarLive.types.banked') }}</span><span><i class="no-record" />{{ t('needtoken.radarStats.noRecord') }}</span></div></div>
    </section>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, type ComponentPublicInstance } from 'vue'
import { useI18n } from 'vue-i18n'
import { resetCalendar, resetStatistics, type ResetCalendarDay, type ResetRecord } from '../lib/resetRadar'
const props = defineProps<{ records: ResetRecord[]; hasMore: boolean; now: number }>()
const { t, locale } = useI18n()
const stats = computed(() => resetStatistics(props.records, props.now))
const weeks = computed(() => resetCalendar(props.records, props.now))
const calendarScroll = ref<HTMLDivElement | null>(null)
onMounted(() => {
  // Narrow screens start at the newest dates without moving keyboard focus.
  // Polling leaves the operator's chosen scroll position untouched.
  if (calendarScroll.value) calendarScroll.value.scrollLeft = calendarScroll.value.scrollWidth
})
const intervalValue = computed(() => stats.value.averageRemainingDays === null ? stats.value.elapsedDays === null ? '—' : number(stats.value.elapsedDays) : stats.value.averageRemainingDays > 0 ? number(stats.value.averageRemainingDays) : number(Math.max(0, (stats.value.elapsedDays ?? 0) - (stats.value.averageIntervalDays ?? 0))))
const intervalProgress = computed(() => stats.value.averageIntervalDays && stats.value.elapsedDays !== null ? Math.min(100, stats.value.elapsedDays / stats.value.averageIntervalDays * 65) : 0)
const number = (value: number) => new Intl.NumberFormat(locale.value, { minimumFractionDigits: 1, maximumFractionDigits: 1 }).format(value)
function dayState(day: ResetCalendarDay) { return day.future ? 'future' : day.regular && day.banked ? 'both' : day.banked ? 'banked' : day.regular ? 'regular' : day.inCoverage ? 'empty' : 'unknown' }
function dayLabel(day: ResetCalendarDay) {
  const date = new Intl.DateTimeFormat(locale.value, { year: 'numeric', month: 'short', day: 'numeric', timeZone: 'UTC' }).format(new Date(`${day.date}T00:00:00Z`))
  return `${date} · ${day.regular || day.banked ? t('needtoken.radarStats.typeCounts', { regular: day.regular, banked: day.banked }) : t(day.inCoverage ? 'needtoken.radarStats.noRecord' : 'needtoken.radarStats.outsideCoverage')}`
}
function monthLabel(week: ResetCalendarDay[], index: number) {
  const day = index === 0 ? week[0] : week.find(value => value.date.endsWith('-01'))
  return day ? new Intl.DateTimeFormat(locale.value, { month: 'short', timeZone: 'UTC' }).format(new Date(`${day.date}T00:00:00Z`)) : ''
}
const lastDay = computed(() => weeks.value.flat().filter(day => !day.future).length - 1)
const focusedDay = ref(lastDay.value)
const dayElements: (HTMLButtonElement | null)[] = []
function setDayElement(element: Element | ComponentPublicInstance | null, index: number) { dayElements[index] = element instanceof HTMLButtonElement ? element : null }
function moveFocus(event: KeyboardEvent) {
  const step = { ArrowUp: -1, ArrowDown: 1, ArrowLeft: -7, ArrowRight: 7 }[event.key]
  if (step === undefined) return
  event.preventDefault()
  const target = Math.max(0, Math.min(lastDay.value, focusedDay.value + step))
  dayElements[target]?.focus()
}
</script>

<style scoped>
.reset-overview { margin-top: 24px; --reset-regular: var(--ui-accent); --reset-banked: var(--nt-warning-ink); }
.reset-stats { display: grid; grid-template-columns: 1.5fr 1fr 1fr; gap: 16px; }
.reset-stat, .reset-calendar { min-width: 0; padding: 24px; border: 1px solid var(--ui-border); border-radius: 16px; background: var(--ui-bg-elevated); }
.stat-heading { display: flex; justify-content: space-between; align-items: baseline; flex-wrap: wrap; gap: 8px; }
h2 { font-size: 14px; font-weight: 500; line-height: 1.6; color: var(--ui-text-secondary); }
.stat-heading > span { color: var(--ui-text-muted); font-size: 12px; }
.stat-value { display: flex; align-items: baseline; gap: 8px; margin-top: 12px; }
.stat-value strong { font-size: 40px; font-weight: 500; line-height: 1.15; letter-spacing: -.035em; font-variant-numeric: tabular-nums; }
.stat-value > span { color: var(--ui-text-muted); font-size: 12px; }
.interval-track, .type-track { display: flex; height: 8px; margin-top: 24px; border-radius: 999px; background: var(--ui-bg-soft); }
.interval-track { position: relative; }
.interval-track > span { border-radius: inherit; background: var(--reset-regular); }
.interval-track > i { position: absolute; height: 16px; width: 2px; top: -4px; left: 65%; border-radius: 1px; background: var(--ui-text-muted); }
.type-track { gap: 2px; overflow: hidden; }
.type-track > span { min-width: 0; }
.regular { background: var(--reset-regular); }.banked { background: var(--reset-banked); }.no-record { background: var(--ui-border-strong); }
.stat-note { margin-top: 20px; color: var(--ui-text-secondary); font-size: 12px; line-height: 1.7; }
.stat-disclaimer { margin-top: 4px; color: var(--ui-text-muted); font-size: 12px; line-height: 1.6; }
.recent-comparison { margin-top: 52px; }
.stat-legend { display: flex; flex-wrap: wrap; gap: 8px 16px; margin-top: 16px; color: var(--ui-text-secondary); font-size: 12px; line-height: 1.6; }
.stat-legend > span { display: inline-flex; align-items: center; gap: 6px; }.stat-legend i { flex: none; display: block; width: 8px; height: 8px; border-radius: 2px; }
.reset-calendar { margin-top: 16px; }
.calendar-heading { display: flex; align-items: flex-start; justify-content: space-between; gap: 16px; }.calendar-heading h2 { color: var(--ui-text-primary); font-size: 16px; }.calendar-heading p, .calendar-heading > span { margin-top: 4px; font-size: 12px; line-height: 1.7; color: var(--ui-text-muted); }
.calendar-scroll { overflow-x: auto; overflow-y: hidden; padding: 44px 0 8px; }
.calendar-months, .calendar-weeks { display: grid; grid-template-columns: repeat(v-bind('weeks.length'), minmax(10px, 1fr)); gap: 3px; min-width: 650px; }
.calendar-months { margin-bottom: 8px; color: var(--ui-text-muted); font-size: 11px; }.calendar-months > span { white-space: nowrap; }
.calendar-week { display: grid; grid-template-rows: repeat(7, 1fr); gap: 3px; }.calendar-cell { position: relative; aspect-ratio: 1; }
.calendar-cell > button { display: block; width: 100%; height: 100%; border-radius: 3px; background: var(--ui-bg-soft); transition: transform 140ms ease, outline-color 140ms ease; }
.calendar-cell > button[data-state="regular"] { background: var(--reset-regular); }.calendar-cell > button[data-state="banked"] { background: var(--reset-banked); }.calendar-cell > button[data-state="both"] { background: linear-gradient(135deg, var(--reset-regular) 50%, var(--reset-banked) 50%); }.calendar-cell > button[data-state="unknown"] { background: color-mix(in srgb, var(--ui-bg-soft) 45%, transparent); }.calendar-cell > button[data-state="future"] { visibility: hidden; }
.calendar-cell > button:hover, .calendar-cell > button:focus-visible { outline: 2px solid var(--ui-text-secondary); outline-offset: 2px; transform: scale(1.08); }
.calendar-tooltip { display: none; position: absolute; z-index: 2; bottom: calc(100% + 8px); left: 50%; transform: translateX(-50%); width: max-content; max-width: 260px; padding: 8px 10px; border: 1px solid var(--ui-border-strong); border-radius: 8px; color: var(--ui-text-primary); background: var(--ui-bg-page); font-size: 11px; line-height: 1.6; pointer-events: none; }
.calendar-cell:hover > .calendar-tooltip, .calendar-cell:focus-within > .calendar-tooltip { display: block; }.calendar-cell.align-left > .calendar-tooltip { left: 0; transform: none; }.calendar-cell.align-right > .calendar-tooltip { left: auto; right: 0; transform: none; }
.calendar-footer { display: flex; align-items: baseline; justify-content: space-between; gap: 16px; margin-top: 12px; }.calendar-footer > p { color: var(--ui-text-muted); font-size: 12px; line-height: 1.7; }.calendar-footer .stat-legend { margin-top: 0; }
@media (max-width: 850px) { .reset-stats { grid-template-columns: 1fr 1fr; }.interval-stat { grid-column: 1 / -1; }.calendar-footer { align-items: flex-start; flex-direction: column; gap: 8px; } }
@media (max-width: 640px) { .reset-stat, .reset-calendar { padding: 20px; }.reset-stats { gap: 12px; }.reset-stat:not(.interval-stat) .stat-heading { min-height: 44px; }.stat-value strong { font-size: 36px; }.stat-legend { gap: 8px; }.calendar-heading { flex-direction: column; gap: 4px; }.calendar-heading > span { display: none; } }
@media (prefers-reduced-motion: reduce) { .calendar-cell > button { transition: none; } }
</style>
