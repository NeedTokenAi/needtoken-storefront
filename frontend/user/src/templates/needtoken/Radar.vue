<template>
  <div class="reset-radar">
    <header class="radar-heading">
      <div>
        <h1>{{ t('needtoken.radar.title') }}</h1>
        <p>{{ t('needtoken.radarLive.description') }}</p>
      </div>
    </header>

    <section v-if="loading && !feed" class="radar-loading" role="status" :aria-label="t('needtoken.common.loading')">
      <span class="skeleton-label" /><span class="skeleton-timer" /><span class="skeleton-copy" /><span class="skeleton-copy short" />
      <div class="skeleton-metrics"><span /><span /><span /></div>
    </section>
    <section v-else-if="!usableFeed" class="radar-load-error" role="status">
      <Info :size="20" aria-hidden="true" />
      <div><h2>{{ t('needtoken.radarLive.unavailableTitle') }}</h2><p>{{ t(expired ? 'needtoken.radarLive.expired' : 'needtoken.radarLive.unavailable') }}</p>
        <p v-if="feed" class="last-success">{{ t('needtoken.radarLive.lastSuccess', { date: dateLabel(feed.fetched_at) }) }}</p>
        <button type="button" class="radar-action" :disabled="loading" @click="loadFeed">{{ t(loading ? 'needtoken.common.loading' : 'needtoken.radarLive.retry') }}</button>
      </div>
    </section>

    <template v-else>
      <p v-if="stale" class="stale-note">{{ t('needtoken.radarLive.staleDetail', { date: dateLabel(usableFeed.fetched_at) }) }}</p>
      <section v-if="latestReset" class="radar-summary" :aria-label="t('needtoken.radarLive.latestRecord')" :aria-busy="loading">
        <div class="elapsed-block">
          <p>{{ t('needtoken.radarLive.elapsed') }}</p>
          <div class="elapsed-time" :aria-label="t('needtoken.radarLive.elapsedValue', elapsed)">
            <span><strong>{{ elapsed.days }}</strong><small>{{ t('needtoken.radarLive.units.days') }}</small></span>
            <span><strong>{{ elapsed.hours }}</strong><small>{{ t('needtoken.radarLive.units.hours') }}</small></span>
            <span><strong>{{ elapsed.minutes }}</strong><small>{{ t('needtoken.radarLive.units.minutes') }}</small></span>
          </div>
          <time :datetime="latestReset.announced_at">{{ dateLabel(latestReset.announced_at) }}</time>
        </div>
        <div v-if="latest" class="latest-record">
          <div class="record-label"><h2 class="reset-badge" :data-type="latest.type">{{ t(`needtoken.radarLive.types.${latest.type}`) }}</h2><span>{{ t(`needtoken.radarLive.sources.${latest.source.type}`) }}</span></div>
          <p class="latest-copy">{{ latest.text }}</p>
          <div class="record-source">
            <span v-if="latest.source.author">@{{ latest.source.author }}</span>
            <a v-if="latest.source.url" :href="latest.source.url" target="_blank" rel="noopener noreferrer">{{ t(latest.source.type === 'x_post' ? 'needtoken.radarLive.originalPost' : 'needtoken.radarLive.viewSource') }}<ArrowUpRight :size="14" aria-hidden="true" /></a>
            <span v-else>{{ t('needtoken.radarLive.noSourceLink') }}</span>
          </div>
        </div>
      </section>
      <section v-else-if="!latest" class="radar-empty" role="status"><h2>{{ t('needtoken.radarLive.noRecords') }}</h2><p>{{ t('needtoken.radarLive.noRecordsDetail') }}</p></section>
      <ResetHistoryOverview :records="usableFeed.records" :has-more="usableFeed.has_more" :now="now" />
    </template>

    <div class="radar-layout">
      <section class="radar-events" aria-labelledby="radar-events-title">
        <div class="section-topline"><h2 id="radar-events-title">{{ t('needtoken.radarLive.history') }}</h2><a :href="RESET_FEED_SOURCE" target="_blank" rel="noopener noreferrer">codex-resets.com<ArrowUpRight :size="12" aria-hidden="true" /></a><span>{{ t('needtoken.radarLive.timezone') }}</span></div>
        <template v-if="usableFeed">
          <div class="radar-filters" role="group" :aria-label="t('needtoken.radar.filterLabel')">
            <button v-for="kind in filters" :key="kind" type="button" :data-radar-filter="kind" :data-radar-type="kind" :aria-pressed="filter === kind" @click="selectFilter(kind)">{{ t(`needtoken.radarLive.filters.${kind}`) }}</button>
          </div>
          <ol v-if="visibleRecords.length" class="radar-timeline">
            <li v-for="entry in visibleRecords" :key="entry.id" :data-record-id="entry.id">
              <div class="record-date"><time :datetime="entry.announced_at">{{ dateLabel(entry.announced_at) }}</time><span class="reset-badge" :data-type="entry.type">{{ t(`needtoken.radarLive.types.${entry.type}`) }}</span></div>
              <div class="record-copy"><span class="record-origin">{{ t(`needtoken.radarLive.sources.${entry.source.type}`) }}<span v-if="entry.source.author"> · @{{ entry.source.author }}</span></span><p>{{ entry.text }}</p>
                <a v-if="entry.source.url" :href="entry.source.url" target="_blank" rel="noopener noreferrer">{{ t(entry.source.type === 'x_post' ? 'needtoken.radarLive.originalPost' : 'needtoken.radarLive.viewSource') }}<ArrowUpRight :size="14" aria-hidden="true" /></a>
                <span v-else class="missing-source">{{ t('needtoken.radarLive.noSourceLink') }}</span>
              </div>
            </li>
          </ol>
          <p v-else class="updates-state">{{ t('needtoken.radarLive.noFilteredRecords') }}</p>
          <button v-if="recordPage.remaining" class="load-more" type="button" data-radar-more @click="visibleCount += RESET_RECORD_PAGE_SIZE">{{ t('needtoken.radarLive.showMore', { count: Math.min(RESET_RECORD_PAGE_SIZE, recordPage.remaining) }) }}<ChevronDown :size="16" aria-hidden="true" /></button>
          <p class="history-note">{{ t(usableFeed.has_more ? 'needtoken.radarLive.moreHistory' : 'needtoken.radarLive.historyCoverage', { count: usableFeed.records.length }) }}</p>
        </template>
        <p v-else class="updates-state">{{ t('needtoken.radarLive.historyUnavailable') }}</p>

        <details class="site-updates-details" @toggle="handleUpdatesToggle">
          <summary>{{ t('needtoken.radar.siteUpdates') }}</summary>
          <div class="site-updates-heading"><p>{{ t('needtoken.radarLive.siteUpdatesDetail') }}</p><button type="button" :disabled="updatesLoading" @click="loadUpdates"><RefreshCw :size="14" aria-hidden="true" />{{ t(updatesLoading ? 'needtoken.common.loading' : 'needtoken.radar.refreshUpdates') }}</button></div>
          <div v-if="updatesLoading" class="updates-skeleton" role="status" :aria-label="t('needtoken.common.loading')"><span /><span /></div>
          <p v-else-if="updatesError" class="updates-state" role="status">{{ t('needtoken.radar.updatesError') }}</p>
          <ul v-else-if="updates.length" class="site-updates"><li v-for="post in updates" :key="post.id"><RouterLink :to="`/blog/${encodeURIComponent(post.slug)}`"><time v-if="post.published_at" :datetime="post.published_at">{{ dateLabel(post.published_at) }}</time><span>{{ getLocalizedText(post.title) }}</span><ArrowUpRight :size="14" aria-hidden="true" /></RouterLink></li></ul>
          <p v-else class="updates-state" role="status">{{ t('needtoken.radar.noSiteUpdates') }}</p>
        </details>
      </section>
      <aside class="radar-guide" :aria-label="t('needtoken.radar.guideTitle')">
        <h2>{{ t('needtoken.radar.guideTitle') }}</h2>
        <div v-for="kind in ['banked', 'automatic', 'paid']" :key="kind" class="guide-item"><h3>{{ t(`needtoken.radar.guide.${kind}.title`) }}</h3><p>{{ t(`needtoken.radar.guide.${kind}.body`) }}</p></div>
        <a href="https://help.openai.com/en/articles/20001498-how-resets-work-in-codex" target="_blank" rel="noopener noreferrer">{{ t('needtoken.radar.resetGuide') }}<ArrowUpRight :size="14" aria-hidden="true" /></a>
        <a href="https://help.openai.com/en/articles/20001507-paid-weekly-work-and-codex-rate-limit-resets" target="_blank" rel="noopener noreferrer">{{ t('needtoken.radar.paidGuide') }}<ArrowUpRight :size="14" aria-hidden="true" /></a>
        <div class="usage-guide"><h3>{{ t('needtoken.radar.usageTitle') }}</h3><p>{{ t('needtoken.radar.usageBody') }}</p><a href="https://chatgpt.com/codex/settings/usage" target="_blank" rel="noopener noreferrer">{{ t('needtoken.radar.openUsage') }}<ArrowUpRight :size="14" aria-hidden="true" /></a></div>
      </aside>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { ArrowUpRight, ChevronDown, Info, RefreshCw } from 'lucide-vue-next'
import { postAPI } from '../../api'
import { useLocalized } from '../../composables/useProduct'
import ResetHistoryOverview from './components/ResetHistoryOverview.vue'
import { usePageSeo } from '../../composables/usePageSeo'
import { fetchResetFeed, RESET_FEED_MAX_AGE_MS, RESET_FEED_POLL_MS, RESET_FEED_SOURCE, resetElapsedTime, resetFeedIsExpired, resetRecordPage, RESET_RECORD_PAGE_SIZE, resetUpdates, type ResetFeed } from './lib/resetRadar'

const { t, locale } = useI18n()
const { getLocalizedText } = useLocalized()
const filters = ['all', 'regular', 'banked'] as const
const filter = ref<typeof filters[number]>('all')
const visibleCount = ref(RESET_RECORD_PAGE_SIZE)
function selectFilter(kind: typeof filters[number]) { filter.value = kind; visibleCount.value = RESET_RECORD_PAGE_SIZE }
const feed = ref<ResetFeed | null>(null)
const loading = ref(true)
const loadError = ref(false)
const now = ref(Date.now())
const expired = computed(() => Boolean(feed.value && resetFeedIsExpired(feed.value, now.value)))
const usableFeed = computed(() => expired.value ? null : feed.value)
const stale = computed(() => Boolean(usableFeed.value && (usableFeed.value.stale || loadError.value)))
const latest = computed(() => usableFeed.value?.records[0])
const latestReset = computed(() => usableFeed.value?.records.find(record => record.type === 'regular' && Date.parse(record.announced_at) <= now.value))
const recordPage = computed(() => resetRecordPage(usableFeed.value?.records || [], filter.value, visibleCount.value))
const visibleRecords = computed(() => recordPage.value.records)
const elapsed = computed(() => resetElapsedTime(latestReset.value?.announced_at || '', now.value))
const dateLabel = (value: string) => new Intl.DateTimeFormat(locale.value, { year: 'numeric', month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit', hour12: false, timeZone: 'Asia/Shanghai' }).format(new Date(value))

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
.reset-radar { width: 100%; max-width: 1120px; margin: 0 auto; padding: 32px 24px 64px; }
.tool-capability { display: inline-flex; align-items: center; margin-bottom: 12px; color: var(--ui-text-muted); font-size: 12px; line-height: 20px; }
.radar-heading { display: flex; align-items: center; justify-content: space-between; gap: 24px; margin-bottom: 32px; }
.radar-heading h1 { color: var(--ui-text-primary); font-size: clamp(28px, 3vw, 36px); line-height: 1.2; font-weight: 500; }
.radar-heading p { margin-top: 12px; max-width: 62ch; color: var(--ui-text-secondary); font-size: 14px; line-height: 1.7; }
.radar-action { display: inline-flex; flex: none; align-items: center; justify-content: center; gap: 8px; min-height: 44px; padding: 12px 16px; border: 1px solid var(--ui-border-strong); border-radius: 8px; color: var(--ui-text-primary); font-size: 13px; transition: background-color 160ms ease, border-color 160ms ease; }
.radar-action:hover:not(:disabled) { background: var(--ui-bg-soft); border-color: var(--ui-text-muted); }
button:disabled { opacity: .5; cursor: wait; }
.stale-note { margin: -8px 0 20px; color: var(--ui-text-secondary); font-size: 13px; line-height: 1.7; }
.radar-summary { display: grid; grid-template-columns: minmax(0, .9fr) minmax(0, 1.1fr); gap: 40px; align-items: center; padding: 32px; border: 1px solid var(--ui-border); border-radius: 16px; background: var(--ui-bg-elevated); }
.elapsed-block > p { color: var(--ui-text-secondary); font-size: 13px; }
.elapsed-time { display: flex; align-items: baseline; flex-wrap: wrap; gap: 16px; margin: 16px 0; }
.elapsed-time > span { display: flex; align-items: baseline; gap: 8px; }
.elapsed-time strong { color: var(--ui-accent); font-size: clamp(32px, 4.8vw, 64px); line-height: 1.1; font-weight: 500; font-variant-numeric: tabular-nums; letter-spacing: -.03em; }
.elapsed-time small { color: var(--ui-text-muted); font-size: 13px; }
.elapsed-block > time { color: var(--ui-text-muted); font-size: 12px; font-variant-numeric: tabular-nums; }
.record-label { display: flex; align-items: baseline; flex-wrap: wrap; gap: 8px 16px; }
h2 { color: var(--ui-text-primary); font-size: 16px; line-height: 1.5; font-weight: 500; }
.record-label > span, .record-origin { color: var(--ui-text-muted); font-size: 12px; }
.latest-copy { display: -webkit-box; -webkit-line-clamp: 4; -webkit-box-orient: vertical; overflow: hidden; margin-top: 12px; color: var(--ui-text-secondary); font-size: 15px; line-height: 1.8; overflow-wrap: anywhere; white-space: pre-line; }
.record-source { display: flex; align-items: center; flex-wrap: wrap; gap: 12px; margin-top: 16px; color: var(--ui-text-muted); font-size: 12px; }
.record-source a, .record-copy a, .radar-guide a { display: inline-flex; align-items: center; gap: 4px; color: var(--ui-accent); font-size: 12px; }
.radar-layout { margin-top: 32px; display: grid; grid-template-columns: minmax(0, 1fr) 272px; gap: 32px; align-items: start; }
.radar-events { min-width: 0; }
.section-topline, .site-updates-heading { display: flex; align-items: center; justify-content: space-between; gap: 16px; flex-wrap: wrap; }
.section-topline > a { display: inline-flex; align-items: center; gap: 4px; margin-left: auto; color: var(--ui-text-muted); font-size: 12px; }
.section-topline > a:hover { color: var(--ui-accent); }
.section-topline > span { color: var(--ui-text-muted); font-size: 12px; }
.radar-filters { display: flex; flex-wrap: wrap; gap: 8px; margin: 20px 0 4px; }
.radar-filters button { min-height: 36px; padding: 8px 12px; border: 1px solid transparent; border-radius: 8px; color: var(--ui-text-secondary); font-size: 13px; transition: background-color 160ms ease, color 160ms ease; }
.radar-filters button:hover { background: var(--ui-bg-soft); }
.radar-filters [aria-pressed="true"] { color: var(--ui-text-primary); background: var(--ui-bg-soft); border-color: var(--ui-border); }
.radar-timeline { padding: 0; margin: 0; list-style: none; }
.radar-timeline li { display: grid; grid-template-columns: 132px minmax(0, 1fr); gap: 24px; padding: 24px 0; border-bottom: 1px solid var(--ui-border); }
.record-date { display: flex; flex-direction: column; gap: 8px; color: var(--ui-text-secondary); font-size: 12px; line-height: 1.7; font-variant-numeric: tabular-nums; }

.record-copy { min-width: 0; }
.record-copy p { margin-top: 8px; color: var(--ui-text-secondary); font-size: 14px; line-height: 1.8; white-space: pre-line; overflow-wrap: anywhere; }
.record-copy a, .missing-source { display: inline-flex; margin-top: 12px; }
.missing-source, .history-note { color: var(--ui-text-muted); font-size: 12px; line-height: 1.7; }
.history-note { margin-top: 16px; }
.load-more { display: flex; align-items: center; justify-content: center; gap: 8px; width: 100%; min-height: 44px; margin-top: 20px; border: 1px solid var(--ui-border); border-radius: 999px; background: var(--ui-bg-soft); color: var(--ui-text-primary); font-size: 13px; transition: background-color 160ms ease, border-color 160ms ease; }
.load-more:hover { background: var(--ui-bg-elevated); border-color: var(--ui-border-strong); }
.reset-badge { display: inline-flex; align-items: center; align-self: flex-start; gap: 6px; padding: 3px 9px; border-radius: 999px; font-size: 12px; line-height: 1.6; white-space: nowrap; }
.reset-badge::before { content: ''; width: 5px; height: 5px; border-radius: 50%; background: currentColor; }
.reset-badge[data-type="regular"] { color: var(--ui-accent); background: color-mix(in srgb, var(--ui-accent) 10%, transparent); }
.reset-badge[data-type="banked"] { color: #dca342; background: #dca34214; }
.radar-guide { position: sticky; top: 88px; padding: 24px; border: 1px solid var(--ui-border); border-radius: 16px; background: var(--ui-bg-elevated); }
.guide-item { margin-top: 24px; }
.guide-item h3, .usage-guide h3 { color: var(--ui-text-primary); font-size: 14px; font-weight: 500; }
.guide-item p, .usage-guide p { margin-top: 8px; color: var(--ui-text-secondary); font-size: 13px; line-height: 1.8; }
.radar-guide > a { display: flex; margin-top: 16px; }
.usage-guide { margin-top: 24px; padding-top: 24px; border-top: 1px solid var(--ui-border); }
.usage-guide a { margin-top: 16px; }
.site-updates-details { margin-top: 32px; padding-top: 24px; border-top: 1px solid var(--ui-border); }
.site-updates-details > summary { min-height: 32px; color: var(--ui-text-primary); font-size: 14px; line-height: 24px; cursor: pointer; }
.site-updates-heading { margin-top: 12px; }
.site-updates-heading p { color: var(--ui-text-muted); font-size: 12px; }
.site-updates-heading button { display: inline-flex; align-items: center; gap: 8px; min-height: 36px; color: var(--ui-text-secondary); font-size: 12px; }
.updates-state { margin-top: 16px; color: var(--ui-text-muted); font-size: 13px; line-height: 1.7; }
.updates-skeleton { display: grid; gap: 12px; margin-top: 16px; }
.updates-skeleton span { height: 16px; max-width: 80%; border-radius: 4px; background: var(--ui-bg-soft); }
.updates-skeleton span + span { max-width: 60%; }
.site-updates { list-style: none; padding: 0; margin-top: 12px; }
.site-updates a { display: flex; align-items: center; gap: 16px; padding: 16px 0; color: var(--ui-text-primary); font-size: 13px; }
.site-updates time { color: var(--ui-text-muted); font-size: 12px; flex: none; }
.site-updates a > span { min-width: 0; overflow-wrap: anywhere; }
.site-updates a > svg { margin-left: auto; flex: none; }
.radar-load-error, .radar-empty { display: flex; gap: 16px; padding: 32px; border: 1px solid var(--ui-border); border-radius: 16px; background: var(--ui-bg-elevated); }
.radar-load-error > svg { flex: none; margin-top: 4px; color: var(--ui-text-muted); }
.radar-load-error p, .radar-empty p { margin-top: 12px; color: var(--ui-text-secondary); font-size: 14px; line-height: 1.8; }
.radar-load-error .last-success { color: var(--ui-text-muted); font-size: 12px; }
.radar-load-error button { margin-top: 20px; }
.radar-empty { display: block; }
.radar-loading { padding: 32px; border: 1px solid var(--ui-border); border-radius: 16px; }
.radar-loading > span, .skeleton-metrics span { display: block; background: var(--ui-bg-soft); border-radius: 4px; }
.skeleton-label { width: 144px; height: 16px; }.skeleton-timer { width: 60%; height: 64px; margin-top: 20px; }.skeleton-copy { width: 80%; height: 16px; margin-top: 20px; }.skeleton-copy.short { width: 48%; margin-top: 12px; }
.skeleton-metrics { display: grid; grid-template-columns: repeat(3, 1fr); gap: 24px; margin-top: 32px; }.skeleton-metrics span { height: 48px; }
a:focus-visible, button:focus-visible, summary:focus-visible { outline: 2px solid var(--ui-accent); outline-offset: 4px; }
@media (max-width: 900px) { .radar-summary { gap: 24px; padding: 24px; }.elapsed-time { gap: 12px; }.radar-layout { grid-template-columns: minmax(0, 1fr); gap: 32px; }.radar-guide { position: static; display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 24px; }.radar-guide h2 { grid-column: 1 / -1; }.guide-item { margin-top: 0; }.radar-guide > a { margin-top: 0; }.usage-guide { grid-column: 1 / -1; margin-top: 0; } }
@media (max-width: 640px) { .reset-radar { padding: 24px 20px 48px; }.radar-heading { align-items: flex-start; flex-direction: column; gap: 20px; margin-bottom: 24px; }.radar-summary { grid-template-columns: 1fr; gap: 24px; padding: 24px; }.elapsed-time strong { font-size: 44px; }.latest-record { padding-top: 24px; border-top: 1px solid var(--ui-border); }.radar-timeline li { grid-template-columns: 1fr; gap: 12px; }.record-date { flex-direction: row; flex-wrap: wrap; gap: 8px 16px; }.radar-guide { display: block; padding: 24px; }.guide-item { margin-top: 24px; }.radar-guide > a { margin-top: 16px; }.usage-guide { margin-top: 24px; }.site-updates a { align-items: flex-start; flex-wrap: wrap; gap: 8px 12px; }.site-updates time { flex-basis: 100%; }.radar-load-error { padding: 24px; }.radar-loading { padding: 24px; }.skeleton-timer { width: 88%; }.skeleton-metrics { gap: 12px; } }
@media (prefers-reduced-motion: reduce) { *, *::before, *::after { transition: none !important; } }
</style>
