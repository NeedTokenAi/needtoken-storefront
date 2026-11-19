<template>
  <div class="reset-radar">
    <ToolNavigation />
    <section class="radar-hero" :aria-busy="loading">
      <div class="hero-topline">
        <div v-if="usableFeed" class="radar-freshness" :data-stale="stale" role="status">
          <span class="freshness-mark" aria-hidden="true" />
          <span>{{ t(stale ? 'needtoken.radarLive.stale' : 'needtoken.radarLayout.scope') }}</span>
          <span class="poll-note">{{ t('needtoken.radarLayout.polling') }}</span>
        </div>
        <span v-else class="tool-capability">{{ t('needtoken.radarLayout.scope') }}</span>
        <button type="button" class="radar-action" data-radar-refresh :aria-disabled="loading" :aria-busy="loading" @click="!loading && loadFeed()">
          <RefreshCw :size="16" :stroke-width="1.5" aria-hidden="true" />{{ t(loading ? 'needtoken.common.loading' : 'needtoken.radarLive.refresh') }}
        </button>
      </div>
      <header class="radar-heading">
        <h1>{{ t('needtoken.radar.title') }}</h1>
        <p>{{ t('needtoken.radarLayout.description') }}</p>
      </header>
      <div v-if="loading && !feed" class="radar-loading" role="status" :aria-label="t('needtoken.common.loading')">
        <div><span class="skeleton-label" /><span class="skeleton-timer" /><span class="skeleton-copy" /></div>
        <div class="skeleton-calendar"><span v-for="cell in 30" :key="cell" /></div>
      </div>
      <div v-else-if="!usableFeed" class="radar-load-error" role="status">
        <Info :size="20" aria-hidden="true" />
        <div><h2>{{ t('needtoken.radarLive.unavailableTitle') }}</h2><p>{{ t(expired ? 'needtoken.radarLive.expired' : 'needtoken.radarLive.unavailable') }}</p>
          <p v-if="feed" class="last-success">{{ t('needtoken.radarLive.lastSuccess', { date: dateLabel(feed.fetched_at) }) }}</p>
          <button type="button" class="radar-action" :aria-disabled="loading" :aria-busy="loading" @click="!loading && loadFeed()">{{ t(loading ? 'needtoken.common.loading' : 'needtoken.radarLive.retry') }}</button>
        </div>
      </div>
      <template v-else>
        <p v-if="stale" class="stale-note" role="status">{{ t('needtoken.radarLive.staleDetail', { date: dateLabel(usableFeed.fetched_at) }) }}</p>
        <div class="radar-overview">
          <div class="overview-main">
            <div v-if="latest" class="elapsed-block" :data-radar-latest-id="latest.id" :aria-label="t('needtoken.radarLive.latestRecord')">
              <p>{{ t('needtoken.radarLayout.elapsed') }}</p>
              <div class="elapsed-time" :aria-label="t('needtoken.radarLive.elapsedValue', elapsed)">
                <span><strong>{{ elapsed.days }}</strong><small>{{ t('needtoken.radarLive.units.days') }}</small></span>
                <span><strong>{{ elapsed.hours }}</strong><small>{{ t('needtoken.radarLive.units.hours') }}</small></span>
                <span class="elapsed-minutes"><strong>{{ elapsed.minutes }}</strong><small>{{ t('needtoken.radarLive.units.minutes') }}</small></span>
              </div>
              <div class="latest-date"><Clock3 :size="14" aria-hidden="true" /><time :datetime="latest.announced_at">{{ t('needtoken.radarLayout.latestAt', { date: dateLabel(latest.announced_at) }) }}</time></div>
            </div>
            <div v-else class="radar-empty" role="status"><h2>{{ t('needtoken.radarLive.noRecords') }}</h2><p>{{ t('needtoken.radarLive.noRecordsDetail') }}</p></div>
            <article v-if="latest" class="latest-record" :data-record-id="latest.id" :data-type="latest.type">
              <div class="record-label">
                <span class="record-kind" :data-type="latest.type">{{ t(`needtoken.radarLive.types.${latest.type}`) }}</span>
                <span class="record-origin">{{ t(`needtoken.radarLive.sources.${latest.source.type}`) }}<span v-if="latest.source.author"> · @{{ latest.source.author }}</span></span>
                <a v-if="latest.source.url" :href="latest.source.url" target="_blank" rel="noopener noreferrer">{{ t(latest.source.type === 'x_post' ? 'needtoken.radarLive.originalPost' : 'needtoken.radarLive.viewSource') }}<ArrowUpRight :size="14" aria-hidden="true" /></a>
              </div>
              <RadarRecordText :text="latest.text" />
              <span v-if="!latest.source.url" class="missing-source">{{ t('needtoken.radarLive.noSourceLink') }}</span>
            </article>
          </div>
          <aside class="radar-activity" :aria-label="t('needtoken.radarLayout.activityTitle')">
            <div class="activity-heading"><span><CalendarDays :size="16" :stroke-width="1.5" aria-hidden="true" />{{ t('needtoken.radarLayout.activityTitle') }}</span><strong>{{ activityCount }}</strong></div>
            <ol class="activity-grid">
              <li v-for="day in activityDays" :key="day.date" :data-date="day.date" :data-count="day.regular + day.banked" :data-active="day.regular + day.banked > 0" :title="t('needtoken.radarLayout.dayLabel', { date: dayLabel(day.date), regular: day.regular, banked: day.banked })">
                <time :datetime="day.date">{{ dayNumber(day.date) }}</time>
                <span class="day-marks" aria-hidden="true"><i v-if="day.regular" class="regular-mark" /><span v-if="day.regular + day.banked">{{ day.regular + day.banked }}</span><i v-if="day.banked" class="banked-mark" /></span>
                <span class="sr-only">{{ t('needtoken.radarLayout.dayLabel', { date: dayLabel(day.date), regular: day.regular, banked: day.banked }) }}</span>
              </li>
            </ol>
            <div class="activity-range"><span>{{ activityDays[0] ? dayLabel(activityDays[0].date) : '' }}</span><span>{{ t('needtoken.radarLayout.today') }}</span></div>
            <div class="activity-legend"><span><i class="regular-mark" />{{ t('needtoken.radarLayout.regularCount') }}</span><span><i class="banked-mark" />{{ t('needtoken.radarLayout.bankedCount') }}</span></div>
            <p class="activity-scope">{{ t('needtoken.radarLayout.activityScope') }}</p>
          </aside>
        </div>
      </template>
    </section>

    <dl v-if="usableFeed" class="radar-metrics">
      <div><dt>{{ t('needtoken.radarLayout.loadedCount') }}</dt><dd>{{ usableFeed.records.length }}</dd></div>
      <div><dt><i class="regular-mark" />{{ t('needtoken.radarLayout.regularCount') }}</dt><dd>{{ regularCount }}</dd></div>
      <div><dt><i class="banked-mark" />{{ t('needtoken.radarLayout.bankedCount') }}</dt><dd>{{ bankedCount }}</dd></div>
    </dl>
    <p class="radar-source-note">{{ t('needtoken.radarLayout.sourceShort') }}</p>

    <section class="radar-events" aria-labelledby="radar-events-title">
      <div class="section-topline"><div><h2 id="radar-events-title">{{ t('needtoken.radarLayout.history') }}</h2><span v-if="usableFeed">{{ t('needtoken.radarLayout.recordCount', { count: usableFeed.records.length }) }}</span></div><span>{{ t('needtoken.radarLive.timezone') }}</span></div>
      <template v-if="usableFeed">
        <div class="radar-filters" role="group" :aria-label="t('needtoken.radar.filterLabel')">
          <button v-for="kind in filters" :key="kind" type="button" :data-radar-filter="kind" :data-radar-type="kind" :aria-pressed="filter === kind" @click="filter = kind">{{ t(`needtoken.radarLive.filters.${kind}`) }}</button>
        </div>
        <ol v-if="visibleRecords.length" class="radar-timeline">
          <li v-for="entry in visibleRecords" :key="entry.id" :data-record-id="entry.id" :data-type="entry.type">
            <div class="record-date"><time :datetime="entry.announced_at">{{ dayLabel(entry.announced_at) }}</time><span>{{ timeLabel(entry.announced_at) }}</span></div>
            <div class="record-copy">
              <div class="record-label"><span class="record-kind" :data-type="entry.type">{{ t(`needtoken.radarLive.types.${entry.type}`) }}</span><span class="record-origin">{{ t(`needtoken.radarLive.sources.${entry.source.type}`) }}<span v-if="entry.source.author"> · @{{ entry.source.author }}</span></span><a v-if="entry.source.url" :href="entry.source.url" target="_blank" rel="noopener noreferrer">{{ t(entry.source.type === 'x_post' ? 'needtoken.radarLive.originalPost' : 'needtoken.radarLive.viewSource') }}<ArrowUpRight :size="14" aria-hidden="true" /></a></div>
              <RadarRecordText :text="entry.text" />
              <span v-if="!entry.source.url" class="missing-source">{{ t('needtoken.radarLive.noSourceLink') }}</span>
            </div>
          </li>
        </ol>
        <p v-else class="updates-state">{{ t('needtoken.radarLive.noFilteredRecords') }}</p>
        <div class="history-bottom"><span>{{ t('needtoken.radarLayout.visibleCount', { shown: visibleRecords.length, count: filteredRecords.length }) }}</span><button v-if="visibleRecords.length < filteredRecords.length" type="button" class="radar-action" data-radar-more @click="recordLimit += RESET_RECORD_PAGE_SIZE">{{ t('needtoken.radarLayout.showMore') }}<ChevronDown :size="14" aria-hidden="true" /></button></div>
        <p class="history-note">{{ t('needtoken.radarLayout.coverage', { count: usableFeed.records.length }) }}<span v-if="usableFeed.has_more"> · {{ t('needtoken.radarLive.moreHistory', { count: usableFeed.records.length }) }}</span></p>
      </template>
      <p v-else class="updates-state">{{ t('needtoken.radarLive.historyUnavailable') }}</p>
    </section>

    <section class="radar-resources" :aria-label="t('needtoken.radarLayout.rules')">
      <details v-if="usableFeed" class="resource-details" data-radar-statistics :open="statsOpen" @toggle="handleStatsToggle">
        <summary>{{ t('needtoken.radarLayout.statistics') }}<ChevronDown :size="16" aria-hidden="true" /></summary>
        <ResetHistoryOverview v-if="statsOpen" :records="usableFeed.records" :has-more="usableFeed.has_more" :now="now" />
      </details>
      <details class="resource-details" data-radar-rules>
        <summary>{{ t('needtoken.radarLayout.rules') }}<ChevronDown :size="16" aria-hidden="true" /></summary>
        <div class="guide-items"><div v-for="kind in ['banked', 'automatic', 'paid']" :key="kind"><h3>{{ t(`needtoken.radar.guide.${kind}.title`) }}</h3><p>{{ t(`needtoken.radar.guide.${kind}.body`) }}</p></div></div>
        <div class="resource-links"><a href="https://help.openai.com/en/articles/20001498-how-resets-work-in-codex" target="_blank" rel="noopener noreferrer">{{ t('needtoken.radar.resetGuide') }}<ArrowUpRight :size="14" aria-hidden="true" /></a><a href="https://help.openai.com/en/articles/20001507-paid-weekly-work-and-codex-rate-limit-resets" target="_blank" rel="noopener noreferrer">{{ t('needtoken.radar.paidGuide') }}<ArrowUpRight :size="14" aria-hidden="true" /></a></div>
      </details>
      <details class="resource-details" data-radar-source>
        <summary>{{ t('needtoken.radarLayout.dataSource') }}<ChevronDown :size="16" aria-hidden="true" /></summary>
        <p>{{ t('needtoken.radarLive.sourceDetail') }}</p>
        <p v-if="usableFeed">{{ t('needtoken.radarLayout.fetchedAt', { date: dateLabel(usableFeed.fetched_at) }) }} · {{ t('needtoken.radarLayout.pollingDetail') }}</p>
        <p class="source-credit">{{ t('needtoken.radarLive.attribution') }} · codex-resets.com</p>
      </details>
      <details class="resource-details site-updates-details" @toggle="handleUpdatesToggle">
        <summary>{{ t('needtoken.radar.siteUpdates') }}<ChevronDown :size="16" aria-hidden="true" /></summary>
        <div class="site-updates-heading"><p>{{ t('needtoken.radarLive.siteUpdatesDetail') }}</p><button type="button" :aria-disabled="updatesLoading" :aria-busy="updatesLoading" @click="!updatesLoading && loadUpdates()"><RefreshCw :size="14" aria-hidden="true" />{{ t(updatesLoading ? 'needtoken.common.loading' : 'needtoken.radar.refreshUpdates') }}</button></div>
        <div v-if="updatesLoading" class="updates-skeleton" role="status" :aria-label="t('needtoken.common.loading')"><span /><span /></div>
        <p v-else-if="updatesError" class="updates-state" role="status">{{ t('needtoken.radar.updatesError') }}</p>
        <ul v-else-if="updates.length" class="site-updates"><li v-for="post in updates" :key="post.id"><RouterLink :to="`/blog/${encodeURIComponent(post.slug)}`"><time v-if="post.published_at" :datetime="post.published_at">{{ dateLabel(post.published_at) }}</time><span>{{ getLocalizedText(post.title) }}</span><ArrowUpRight :size="14" aria-hidden="true" /></RouterLink></li></ul>
        <p v-else class="updates-state" role="status">{{ t('needtoken.radar.noSiteUpdates') }}</p>
      </details>
      <div class="usage-row"><div><h3>{{ t('needtoken.radarLayout.usage') }}</h3><p>{{ t('needtoken.radar.usageBody') }}</p></div><a class="radar-action" href="https://chatgpt.com/codex/settings/usage" target="_blank" rel="noopener noreferrer">{{ t('needtoken.radar.openUsage') }}<ArrowUpRight :size="16" aria-hidden="true" /></a></div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { ArrowUpRight, CalendarDays, ChevronDown, Clock3, Info, RefreshCw } from 'lucide-vue-next'
import { postAPI } from '../../api'
import { useLocalized } from '../../composables/useProduct'
import ToolNavigation from './components/ToolNavigation.vue'
import RadarRecordText from './components/RadarRecordText.vue'
import ResetHistoryOverview from './components/ResetHistoryOverview.vue'
import { usePageSeo } from '../../composables/usePageSeo'
import { fetchResetFeed, RESET_FEED_MAX_AGE_MS, RESET_FEED_POLL_MS, RESET_RECORD_PAGE_SIZE, resetRecordPage, resetActivityDays, resetElapsedTime, resetFeedIsExpired, resetUpdates, type ResetFeed } from './lib/resetRadar'

const { t, locale } = useI18n()
const { getLocalizedText } = useLocalized()
const filters = ['all', 'regular', 'banked'] as const
const filter = ref<typeof filters[number]>('all')
const feed = ref<ResetFeed | null>(null)
const loading = ref(true)
const loadError = ref(false)
const now = ref(Date.now())
const expired = computed(() => Boolean(feed.value && resetFeedIsExpired(feed.value, now.value)))
const usableFeed = computed(() => expired.value ? null : feed.value)
const stale = computed(() => Boolean(usableFeed.value && (usableFeed.value.stale || loadError.value)))
const latest = computed(() => usableFeed.value?.records[0])
const recordLimit = ref(RESET_RECORD_PAGE_SIZE)
const filteredRecords = computed(() => usableFeed.value?.records.filter(row => filter.value === 'all' || row.type === filter.value) || [])
const recordPage = computed(() => resetRecordPage(usableFeed.value?.records || [], filter.value, recordLimit.value))
const visibleRecords = computed(() => recordPage.value.records)
watch(filter, () => { recordLimit.value = RESET_RECORD_PAGE_SIZE })
const regularCount = computed(() => usableFeed.value?.records.filter(row => row.type === 'regular').length || 0)
const bankedCount = computed(() => usableFeed.value?.records.filter(row => row.type === 'banked').length || 0)
const activityDays = computed(() => resetActivityDays(usableFeed.value?.records || [], now.value))
const activityCount = computed(() => activityDays.value.reduce((sum, day) => sum + day.regular + day.banked, 0))
const elapsed = computed(() => resetElapsedTime(latest.value?.announced_at || '', now.value))
const dateLabel = (value: string) => new Intl.DateTimeFormat(locale.value, { year: 'numeric', month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit', hour12: false, timeZone: 'Asia/Shanghai' }).format(new Date(value))

const dayLabel = (value: string) => new Intl.DateTimeFormat(locale.value, { year: 'numeric', month: 'short', day: 'numeric', timeZone: 'Asia/Shanghai' }).format(new Date(value))
const dayNumber = (value: string) => new Intl.DateTimeFormat(locale.value, { day: 'numeric', timeZone: 'Asia/Shanghai' }).format(new Date(value))
const timeLabel = (value: string) => new Intl.DateTimeFormat(locale.value, { hour: '2-digit', minute: '2-digit', hour12: false, timeZone: 'Asia/Shanghai' }).format(new Date(value))

let generation = 0
let disposed = false
let lastAttemptAt = 0
let controller: AbortController | null = null
let pollTimer: ReturnType<typeof setTimeout> | undefined
let expiryTimer: ReturnType<typeof setTimeout> | undefined
function clearPoll() { if (pollTimer !== undefined) clearTimeout(pollTimer); pollTimer = undefined }
function scheduleExpiry(snapshot: ResetFeed) {
  if (expiryTimer !== undefined) clearTimeout(expiryTimer)
  expiryTimer = setTimeout(() => { now.value = Date.now() }, Math.max(0, Date.parse(snapshot.fetched_at) + RESET_FEED_MAX_AGE_MS - Date.now()))
}
function scheduleNext() {
  clearPoll()
  if (disposed || document.hidden) return
  pollTimer = setTimeout(() => { void loadFeed() }, Math.max(0, RESET_FEED_POLL_MS - (Date.now() - lastAttemptAt)))
}
async function loadFeed() {
  if (disposed || document.hidden) return
  clearPoll()
  controller?.abort()
  const request = ++generation
  const activeController = new AbortController()
  controller = activeController
  lastAttemptAt = Date.now()
  now.value = lastAttemptAt
  loading.value = true
  const timeout = setTimeout(() => activeController.abort(), 15_000)
  try {
    const result = await fetchResetFeed(activeController.signal)
    if (!disposed && request === generation) { feed.value = result; loadError.value = false; scheduleExpiry(result) }
  } catch {
    if (!disposed && request === generation) loadError.value = true
  } finally {
    clearTimeout(timeout)
    if (!disposed && request === generation) {
      controller = null
      now.value = Date.now()
      loading.value = false
      scheduleNext()
    }
  }
}
function handleVisibilityChange() {
  now.value = Date.now()
  if (document.hidden) clearPoll()
  else if (!loading.value && (!feed.value || expired.value || Date.now() - lastAttemptAt >= RESET_FEED_POLL_MS)) void loadFeed()
  else if (!loading.value) scheduleNext()
  // A page initially mounted in a hidden tab has not started a request yet.
  else if (lastAttemptAt === 0) void loadFeed()
}

const statsOpen = ref(false)
function handleStatsToggle(event: Event) { statsOpen.value = (event.target as HTMLDetailsElement).open }

const updatesLoading = ref(false)
const updatesError = ref(false)
const updatesLoaded = ref(false)
const updates = ref<ReturnType<typeof resetUpdates>>([])
let updatesGeneration = 0
async function loadUpdates() {
  const request = ++updatesGeneration
  updatesLoading.value = true
  updatesError.value = false
  try {
    const response = await postAPI.list({ category_slug: 'codex-resets', type: 'blog', page: 1, page_size: 10 })
    if (!disposed && request === updatesGeneration) { updates.value = resetUpdates(response.data.data); updatesLoaded.value = true }
  } catch {
    if (!disposed && request === updatesGeneration) updatesError.value = true
  } finally {
    if (!disposed && request === updatesGeneration) updatesLoading.value = false
  }
}
function handleUpdatesToggle(event: Event) {
  if ((event.target as HTMLDetailsElement).open && !updatesLoaded.value && !updatesLoading.value) void loadUpdates()
}
usePageSeo({ title: () => t('needtoken.radar.title'), description: () => t('needtoken.radarLive.description'), canonicalPath: () => '/tools/radar' })
onMounted(() => { document.addEventListener('visibilitychange', handleVisibilityChange); void loadFeed() })
onUnmounted(() => { disposed = true; generation++; updatesGeneration++; controller?.abort(); clearPoll(); if (expiryTimer !== undefined) clearTimeout(expiryTimer); document.removeEventListener('visibilitychange', handleVisibilityChange) })
</script>

<style scoped>
.reset-radar { width: 100%; max-width: 1184px; margin: 0 auto; padding: 32px 32px 64px; }
.radar-hero { padding: 24px 0 32px; border-bottom: 1px solid var(--ui-border); }
.hero-topline, .radar-freshness, .record-label, .section-topline, .history-bottom, .radar-source-note { display: flex; align-items: center; gap: 12px; }
.hero-topline { justify-content: space-between; min-height: 40px; margin-bottom: 16px; }
.tool-capability, .radar-freshness { color: var(--ui-text-secondary); font-size: 12px; }
.radar-freshness { flex-wrap: wrap; gap: 8px; }
.freshness-mark { width: 6px; height: 6px; flex: none; border-radius: 50%; background: var(--ui-accent); }
.poll-note { padding-left: 8px; border-left: 1px solid var(--ui-border); color: var(--ui-text-muted); }
.radar-freshness[data-stale="true"] { color: var(--nt-warning-ink); }
.radar-freshness[data-stale="true"] .freshness-mark { background: var(--nt-warning-ink); }
.radar-heading h1 { color: var(--ui-text-primary); font-size: clamp(28px, 3.2vw, 40px); line-height: 1.2; font-weight: 500; }
.radar-heading > p { margin-top: 12px; color: var(--ui-text-secondary); font-size: 14px; line-height: 1.6; }
.radar-action { display: inline-flex; flex: none; align-items: center; justify-content: center; gap: 8px; min-height: 40px; padding: 8px 12px; border: 1px solid var(--ui-border-strong); border-radius: 8px; color: var(--ui-text-primary); font-size: 12px; line-height: 20px; transition: background-color 160ms var(--nt-motion-ease, ease-out), border-color 160ms var(--nt-motion-ease, ease-out); }
.radar-action:hover:not([aria-disabled="true"]) { background: var(--ui-bg-soft); border-color: var(--ui-text-muted); }
button[aria-disabled="true"] { opacity: .5; cursor: wait; }
.radar-overview { display: grid; grid-template-columns: minmax(0, 1fr) 348px; gap: 64px; align-items: start; margin-top: 32px; }
.overview-main { min-width: 0; }
.elapsed-block > p { color: var(--ui-text-secondary); font-size: 14px; }
.elapsed-time { display: flex; flex-wrap: wrap; align-items: baseline; gap: 20px; margin: 12px 0 16px; font-variant-numeric: tabular-nums; }
.elapsed-time > span { display: inline-flex; align-items: baseline; gap: 8px; }
.elapsed-time strong { color: var(--ui-text-primary); font-size: clamp(48px, 6.8vw, 88px); font-weight: 500; line-height: 1; letter-spacing: -.02em; }
.elapsed-time small { color: var(--ui-text-secondary); font-size: 18px; }
.elapsed-time .elapsed-minutes strong { font-size: 32px; color: var(--ui-text-secondary); letter-spacing: 0; }
.elapsed-time .elapsed-minutes small { font-size: 13px; }
.latest-date { display: flex; align-items: center; gap: 8px; color: var(--ui-text-muted); font-size: 12px; line-height: 20px; font-variant-numeric: tabular-nums; }
.latest-date svg { flex: none; }
.latest-record { margin-top: 24px; padding: 16px 20px 12px; border: 1px solid var(--ui-border); border-radius: 12px; background: var(--ui-bg-elevated); }
.record-label { flex-wrap: wrap; gap: 8px 12px; margin-bottom: 8px; }
.record-kind { display: inline-flex; align-items: center; min-height: 24px; padding: 0 8px; border-radius: 4px; color: var(--ui-accent); background: color-mix(in srgb, var(--ui-accent) 8%, transparent); font-size: 12px; line-height: 20px; }
.record-kind[data-type="banked"] { color: var(--nt-warning-ink); background: var(--nt-warning-bg); }
.record-origin, .missing-source { color: var(--ui-text-muted); font-size: 12px; overflow-wrap: anywhere; }
.record-label > a { margin-left: auto; }
a:not(.radar-action) { display: inline-flex; align-items: center; gap: 4px; color: var(--ui-text-secondary); font-size: 12px; line-height: 20px; transition: color 160ms var(--nt-motion-ease, ease-out); }
a:not(.radar-action):hover { color: var(--ui-accent); }
a svg { flex: none; }
.missing-source { display: inline-block; margin-top: 4px; line-height: 20px; }
.radar-activity { min-width: 0; padding: 20px; border: 1px solid var(--ui-border); border-radius: 16px; background: var(--ui-bg-elevated); }
.activity-heading { display: flex; align-items: center; justify-content: space-between; gap: 12px; margin-bottom: 16px; }
.activity-heading > span { display: flex; align-items: center; gap: 8px; color: var(--ui-text-secondary); font-size: 13px; }
.activity-heading svg { flex: none; color: var(--ui-text-muted); }
.activity-heading strong { color: var(--ui-text-primary); font-size: 24px; font-weight: 500; font-variant-numeric: tabular-nums; }
.activity-grid { display: grid; grid-template-columns: repeat(6, minmax(0, 1fr)); gap: 4px; list-style: none; padding: 0; margin: 0; }
.activity-grid > li { display: flex; min-width: 0; min-height: 40px; flex-direction: column; align-items: center; justify-content: center; gap: 2px; border: 1px solid transparent; border-radius: 4px; background: var(--ui-bg-soft); color: var(--ui-text-muted); font-size: 11px; line-height: 14px; font-variant-numeric: tabular-nums; transition: border-color 160ms var(--nt-motion-ease, ease-out), background-color 160ms var(--nt-motion-ease, ease-out); }
.activity-grid > li[data-active="true"] { border-color: color-mix(in srgb, var(--ui-accent) 20%, transparent); color: var(--ui-text-primary); background: color-mix(in srgb, var(--ui-accent) 5%, var(--ui-bg-elevated)); }
.activity-grid > li:hover { border-color: var(--ui-text-muted); }
.day-marks { display: flex; align-items: center; justify-content: center; gap: 4px; height: 12px; font-size: 11px; }
.regular-mark, .banked-mark { display: inline-block; flex: none; width: 5px; height: 5px; border-radius: 50%; background: var(--ui-accent); }
.banked-mark { background: var(--nt-warning-ink); }
.activity-range { display: flex; justify-content: space-between; gap: 12px; margin-top: 8px; color: var(--ui-text-muted); font-size: 11px; line-height: 16px; }
.activity-legend { display: flex; flex-wrap: wrap; align-items: center; gap: 16px; margin-top: 16px; }
.activity-legend > span { display: inline-flex; align-items: center; gap: 6px; color: var(--ui-text-secondary); font-size: 12px; }
.activity-scope { margin-top: 8px; color: var(--ui-text-muted); font-size: 11px; line-height: 16px; }
.radar-metrics { display: grid; grid-template-columns: 1.25fr 1fr 1fr; margin: 0; padding: 24px 0; border-bottom: 1px solid var(--ui-border); }
.radar-metrics > div { min-width: 0; padding: 0 32px; border-left: 1px solid var(--ui-border); }
.radar-metrics > div:first-child { padding-left: 0; border-left: 0; }
.radar-metrics dt { display: flex; align-items: center; gap: 8px; color: var(--ui-text-secondary); font-size: 12px; line-height: 20px; }
.radar-metrics dd { margin-top: 8px; color: var(--ui-text-primary); font-size: 32px; line-height: 1.1; font-weight: 500; font-variant-numeric: tabular-nums; }
.radar-source-note { justify-content: space-between; align-items: baseline; flex-wrap: wrap; margin: 16px 0 40px; color: var(--ui-text-muted); font-size: 12px; line-height: 20px; }
.radar-events { min-width: 0; }
h2 { color: var(--ui-text-primary); font-size: 20px; line-height: 28px; font-weight: 500; }
.section-topline { justify-content: space-between; align-items: baseline; }
.section-topline > div { display: flex; align-items: baseline; gap: 12px; }
.section-topline span { color: var(--ui-text-muted); font-size: 12px; line-height: 20px; }
.radar-filters { display: flex; flex-wrap: wrap; gap: 4px; width: fit-content; max-width: 100%; margin: 20px 0 8px; padding: 4px; border: 1px solid var(--ui-border); border-radius: 8px; background: var(--ui-bg-elevated); }
.radar-filters button { min-height: 36px; padding: 8px 12px; border: 1px solid transparent; border-radius: 4px; color: var(--ui-text-secondary); font-size: 12px; line-height: 20px; transition: background-color 160ms var(--nt-motion-ease, ease-out), color 160ms var(--nt-motion-ease, ease-out); }
.radar-filters button:hover { color: var(--ui-text-primary); background: var(--ui-bg-soft); }
.radar-filters [aria-pressed="true"] { color: var(--ui-text-primary); background: var(--ui-bg-soft); border-color: var(--ui-border); }
.radar-timeline { list-style: none; padding: 0; margin: 0; }
.radar-timeline > li { display: grid; grid-template-columns: 128px minmax(0, 1fr); gap: 32px; padding: 20px 0; border-bottom: 1px solid var(--ui-border); }
.record-date { display: flex; flex-direction: column; gap: 4px; padding-top: 2px; color: var(--ui-text-secondary); font-size: 12px; line-height: 20px; font-variant-numeric: tabular-nums; }
.record-date > span { color: var(--ui-text-muted); }
.record-copy { min-width: 0; }
.history-bottom { justify-content: space-between; min-height: 48px; margin-top: 12px; }
.history-bottom > span, .history-note { color: var(--ui-text-muted); font-size: 12px; line-height: 20px; }
.history-note { margin-top: 4px; }
.radar-resources { margin-top: 40px; border-top: 1px solid var(--ui-border); }
.resource-details { padding: 0 4px; border-bottom: 1px solid var(--ui-border); }
.resource-details > summary { display: flex; align-items: center; justify-content: space-between; gap: 16px; min-height: 56px; color: var(--ui-text-primary); font-size: 14px; cursor: pointer; list-style: none; }
summary::-webkit-details-marker { display: none; }
.resource-details > summary svg { flex: none; color: var(--ui-text-muted); transition: transform 160ms var(--nt-motion-ease, ease-out); }
.resource-details[open] > summary svg { transform: rotate(180deg); }
.resource-details > p { margin-bottom: 16px; max-width: 80ch; color: var(--ui-text-secondary); font-size: 13px; line-height: 1.7; }
.resource-details > a { margin-bottom: 20px; }
.guide-items { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 32px; padding: 8px 0 20px; }
h3 { color: var(--ui-text-primary); font-size: 14px; line-height: 22px; font-weight: 500; }
.guide-items p, .usage-row p { margin-top: 8px; color: var(--ui-text-secondary); font-size: 13px; line-height: 1.7; }
.resource-links { display: flex; flex-wrap: wrap; gap: 12px 24px; margin-bottom: 24px; }
.usage-row { display: flex; align-items: center; justify-content: space-between; gap: 32px; padding: 24px 4px; }
.usage-row p { max-width: 64ch; }
.usage-row .radar-action { font-size: 12px; }
.site-updates-heading { display: flex; align-items: center; justify-content: space-between; gap: 16px; padding: 8px 0 16px; }
.site-updates-heading p { color: var(--ui-text-muted); font-size: 12px; line-height: 20px; }
.site-updates-heading button { display: inline-flex; flex: none; align-items: center; gap: 8px; min-height: 36px; color: var(--ui-text-secondary); font-size: 12px; }
.updates-state { margin: 16px 0 24px; color: var(--ui-text-muted); font-size: 13px; line-height: 1.7; }
.site-updates { list-style: none; padding: 0; margin: 0 0 16px; }
.site-updates a { display: flex; gap: 16px; min-height: 40px; padding: 8px 0; }
.site-updates a > span { min-width: 0; overflow-wrap: anywhere; }
.site-updates a > svg { margin-left: auto; }
.site-updates time { color: var(--ui-text-muted); font-variant-numeric: tabular-nums; }
.updates-skeleton { display: grid; gap: 12px; margin-bottom: 24px; }
.updates-skeleton span { width: 80%; height: 16px; border-radius: 4px; background: var(--ui-bg-soft); }
.updates-skeleton span + span { width: 60%; }
.stale-note { margin-top: 20px; padding: 12px 16px; border-left: 2px solid var(--nt-warning-ink); color: var(--ui-text-secondary); background: var(--nt-warning-bg); font-size: 13px; line-height: 1.7; }
.radar-load-error { display: flex; gap: 16px; padding: 24px 0; margin-top: 24px; }
.radar-load-error > svg { flex: none; margin-top: 4px; color: var(--ui-text-muted); }
.radar-load-error h2, .radar-empty h2 { font-size: 16px; line-height: 24px; }
.radar-load-error p, .radar-empty p { margin-top: 12px; color: var(--ui-text-secondary); font-size: 14px; line-height: 1.7; }
.radar-load-error .last-success { color: var(--ui-text-muted); font-size: 12px; }
.radar-load-error button { margin-top: 20px; }
.radar-empty { display: flex; flex-direction: column; justify-content: center; min-height: 200px; }
.radar-loading { display: grid; grid-template-columns: minmax(0, 1fr) 348px; align-items: start; gap: 64px; min-height: 352px; padding-top: 32px; }
.radar-loading > div > span, .skeleton-calendar span { display: block; background: var(--ui-bg-soft); border-radius: 4px; }
.skeleton-label { width: 128px; height: 16px; }
.skeleton-timer { width: 80%; height: 88px; margin-top: 16px; }
.skeleton-copy { width: 100%; height: 108px; margin-top: 24px; }
.skeleton-calendar { display: grid; grid-template-columns: repeat(6, minmax(0, 1fr)); gap: 4px; padding: 20px; border: 1px solid var(--ui-border); border-radius: 16px; }
.skeleton-calendar span { height: 40px; }
a:focus-visible, button:focus-visible, summary:focus-visible { outline: 2px solid var(--ui-accent); outline-offset: 4px; }
.sr-only { position: absolute; width: 1px; height: 1px; padding: 0; overflow: hidden; clip: rect(0, 0, 0, 0); white-space: nowrap; border: 0; }
@media (max-width: 1024px) {
  .radar-overview, .radar-loading { grid-template-columns: minmax(0, 1fr) 300px; gap: 32px; }
  .elapsed-time { gap: 12px; }
  .elapsed-time strong { font-size: 64px; }
  .elapsed-time .elapsed-minutes { flex-basis: 100%; }
}
@media (max-width: 760px) {
  .reset-radar { padding: 24px 20px 48px; }
  .radar-hero { padding-top: 8px; }
  .radar-overview { grid-template-columns: minmax(0, 1fr); gap: 24px; margin-top: 24px; }
  .radar-heading h1 { font-size: 30px; }
  .elapsed-time { flex-wrap: wrap; gap: 16px; }
  .elapsed-time strong { font-size: 64px; }
  .elapsed-time small { font-size: 16px; }
  .elapsed-time .elapsed-minutes { flex-basis: auto; }
  .elapsed-time .elapsed-minutes strong { font-size: 28px; }
  .elapsed-time .elapsed-minutes small { font-size: 12px; }
  .latest-record { padding: 16px 16px 8px; }
  .radar-activity { padding: 16px; }
  .activity-grid { grid-template-columns: repeat(10, minmax(0, 1fr)); }
  .activity-grid > li { min-height: 28px; }
  .day-marks { height: 6px; gap: 2px; }
  .day-marks > span { display: none; }
  .activity-scope { margin-top: 4px; }
  .activity-legend { margin-top: 12px; }
  .radar-metrics { grid-template-columns: 1.2fr 1fr 1fr; padding: 20px 0; }
  .radar-metrics > div { padding: 0 16px; }
  .radar-metrics dt { font-size: 11px; gap: 4px; flex-wrap: wrap; }
  .radar-metrics dd { font-size: 28px; }
  .radar-source-note { margin-bottom: 32px; gap: 8px; }
  .radar-timeline > li { grid-template-columns: 96px minmax(0, 1fr); gap: 20px; }
  .record-label > a { margin-left: 0; }
  .guide-items { grid-template-columns: 1fr; gap: 20px; }
  .usage-row { flex-direction: column; align-items: flex-start; gap: 16px; }
  .radar-loading { grid-template-columns: 1fr; gap: 24px; }
  .skeleton-timer { max-width: 320px; height: 64px; }
  .skeleton-calendar { grid-template-columns: repeat(10, minmax(0, 1fr)); padding: 16px; }
  .skeleton-calendar span { height: 36px; }
}
@media (max-width: 480px) {
  .skeleton-calendar span { height: 28px; }
  .hero-topline { align-items: flex-start; }
  .radar-freshness { gap: 6px; flex-direction: column; align-items: flex-start; padding-left: 12px; position: relative; }
  .freshness-mark { position: absolute; left: 0; top: 6px; }
  .poll-note { padding: 0; border: 0; }
  .elapsed-time { gap: 12px; }
  .elapsed-time > span { gap: 4px; }
  .elapsed-time strong { font-size: 56px; }
  .elapsed-time .elapsed-minutes strong { font-size: 24px; }
  .latest-date { align-items: flex-start; }
  .latest-date svg { margin-top: 3px; }
  .section-topline { align-items: flex-start; gap: 8px; flex-wrap: wrap; }
  .section-topline > div { gap: 8px; }
  .radar-timeline > li { grid-template-columns: minmax(0, 1fr); gap: 12px; }
  .record-date { flex-direction: row; justify-content: space-between; padding: 0; }
  .radar-metrics > div { padding: 0 12px; }
  .radar-metrics dt { display: block; }
  .radar-metrics dt > i { margin-right: 4px; }
  .site-updates-heading { flex-wrap: wrap; gap: 8px; }
  .site-updates a { align-items: flex-start; flex-wrap: wrap; gap: 4px 12px; }
  .site-updates time { flex-basis: 100%; }
}
@media (prefers-reduced-motion: reduce) { *, *::before, *::after { transition: none !important; } }
</style>
