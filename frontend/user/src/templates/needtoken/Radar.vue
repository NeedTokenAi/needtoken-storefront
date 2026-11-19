<template>
  <div class="reset-radar">
    <!-- Radar Channel Tabs (Codex / Claude / Grok; extensible for Gemini) -->
    <div
      ref="tabRailEl"
      class="radar-tabs"
      role="tablist"
      :aria-label="t('needtoken.radarV2.tabListLabel')"
      @keydown="onTabKeydown"
    >
      <span
        class="nt-sliding-thumb radar-tab-thumb"
        :class="{ 'is-ready': tabIndicatorReady }"
        :style="tabIndicatorStyle"
        aria-hidden="true"
      />
      <button
        v-for="(channel, idx) in channels"
        :id="`radar-tab-${channel.id}`"
        :key="channel.id"
        :ref="el => setTabEl(el, idx)"
        type="button"
        role="tab"
        class="radar-tab"
        :data-radar-tab-active="activeChannel === channel.id ? 'true' : undefined"
        :aria-selected="activeChannel === channel.id"
        :tabindex="activeChannel === channel.id ? 0 : -1"
        @click="selectChannel(channel.id)"
      >
        <ProductBrandMark :brand="channel.brand" class="radar-tab-brand" />
        <span>{{ t(`needtoken.radarV2.tabs.${channel.id}`) }}</span>
      </button>
    </div>

    <!-- Codex Live Radar Panel -->
    <div
      v-if="activeChannel === 'codex'"
      role="tabpanel"
      aria-labelledby="radar-tab-codex"
      class="radar-panel"
    >
      <section class="radar-hero" :aria-busy="loading">
        <div class="hero-topline">
          <div v-if="usableFeed" class="radar-freshness" :data-stale="stale" role="status">
            <span class="freshness-mark" aria-hidden="true" />
            <span>{{ t(stale ? 'needtoken.radarLive.stale' : 'needtoken.radarLayout.scope') }}</span>
            <span class="poll-note">{{ t('needtoken.radarLayout.polling') }}</span>
          </div>
          <span v-else class="tool-capability">{{ t('needtoken.radarLayout.scope') }}</span>
        </div>

        <header class="radar-heading">
          <h1>{{ t('needtoken.radar.title') }}</h1>
          <p>{{ t('needtoken.radarLayout.description') }}</p>
        </header>

        <div v-if="loading && !feed" class="radar-loading" role="status" :aria-label="t('needtoken.common.loading')">
          <div class="skeleton-col">
            <span class="skeleton-label" />
            <span class="skeleton-timer" />
            <span class="skeleton-date" />
          </div>
          <div class="skeleton-card" />
        </div>

        <div v-else-if="!usableFeed" class="radar-load-error" role="status">
          <Info :size="20" aria-hidden="true" />
          <div>
            <h2>{{ t('needtoken.radarLive.unavailableTitle') }}</h2>
            <p>{{ t(expired ? 'needtoken.radarLive.expired' : 'needtoken.radarLive.unavailable') }}</p>
            <p v-if="feed" class="last-success">{{ t('needtoken.radarLive.lastSuccess', { date: slashDateTime(feed.fetched_at) }) }}</p>
            <button type="button" class="radar-action" :aria-disabled="loading" :aria-busy="loading" @click="!loading && loadFeed()">
              {{ t(loading ? 'needtoken.common.loading' : 'needtoken.radarLive.retry') }}
            </button>
          </div>
        </div>

        <template v-else>
          <p v-if="stale" class="stale-note" role="status">
            {{ t('needtoken.radarLive.staleDetail', { date: slashDateTime(usableFeed.fetched_at) }) }}
          </p>

          <div class="radar-overview">
            <div class="overview-main">
              <div v-if="latest" class="elapsed-block" :data-radar-latest-id="latest.id" :aria-label="t('needtoken.radarLive.latestRecord')">
                <p class="elapsed-label">{{ t('needtoken.radarV2.elapsedTitle') }}</p>
                <div class="elapsed-time" :aria-label="t('needtoken.radarLive.elapsedValue', elapsed)">
                  <span><strong>{{ elapsed.days }}</strong><small>{{ t('needtoken.radarLive.units.days') }}</small></span>
                  <span><strong>{{ elapsed.hours }}</strong><small>{{ t('needtoken.radarLive.units.hours') }}</small></span>
                  <span class="elapsed-minutes"><strong>{{ elapsed.minutes }}</strong><small>{{ t('needtoken.radarLive.units.minutes') }}</small></span>
                </div>
                <div class="latest-date">
                  <Clock3 :size="14" aria-hidden="true" />
                  <time :datetime="latest.announced_at">
                    <strong>{{ t('needtoken.radarV2.lastReset', { date: slashDateTime(latest.announced_at) }) }}</strong>
                  </time>
                </div>
              </div>
              <div v-else class="radar-empty" role="status">
                <h2>{{ t('needtoken.radarLive.noRecords') }}</h2>
                <p>{{ t('needtoken.radarLive.noRecordsDetail') }}</p>
              </div>
            </div>

            <!-- Latest reset announcement card moved to top-right above the full-width calendar -->
            <article v-if="latest" class="latest-record-card" :data-record-id="latest.id" :data-type="latest.type">
              <div class="record-label">
                <span class="record-kind" :data-type="latest.type">{{ t(`needtoken.radarV2.types.${latest.type}`) }}</span>
                <span class="record-origin">
                  {{ t(`needtoken.radarLive.sources.${latest.source.type}`) }}<span v-if="latest.source.author"> · @{{ latest.source.author }}</span>
                </span>
                <a v-if="latest.source.url" :href="latest.source.url" target="_blank" rel="noopener noreferrer">
                  {{ t(latest.source.type === 'x_post' ? 'needtoken.radarLive.originalPost' : 'needtoken.radarLive.viewSource') }}
                  <ArrowUpRight :size="14" aria-hidden="true" />
                </a>
              </div>
              <RadarRecordText :text="latest.text" />
              <span v-if="!latest.source.url" class="missing-source">{{ t('needtoken.radarLive.noSourceLink') }}</span>
            </article>
          </div>
        </template>
      </section>

      <!-- Full-width Reset Calendar -->
      <section v-if="usableFeed" class="radar-calendar-section">
        <RadarCalendar :records="usableFeed.records" :now="now" />
      </section>

      <!-- Two-column Reset Records + Reading Guide -->
      <section v-if="usableFeed" class="radar-records-layout" aria-labelledby="radar-records-heading">
        <div class="records-card">
          <div class="records-card-head">
            <div>
              <h2 id="radar-records-heading">{{ t('needtoken.radarV2.recordsTitle') }}</h2>
              <p>{{ t('needtoken.radarV2.recordsSubtitle') }}</p>
            </div>
            <div class="radar-filters" role="group" :aria-label="t('needtoken.radar.filterLabel')">
              <button
                v-for="kind in filters"
                :key="kind"
                type="button"
                :data-radar-filter="kind"
                :aria-pressed="filter === kind"
                @click="filter = kind"
              >
                {{ filterLabel(kind) }}
              </button>
            </div>
          </div>

          <ol v-if="visibleRecords.length" class="records-timeline">
            <li
              v-for="(entry, idx) in visibleRecords"
              :key="entry.id"
              class="timeline-item"
              :data-record-id="entry.id"
              :data-type="entry.type"
            >
              <div class="timeline-rail" aria-hidden="true">
                <span class="timeline-node" :data-type="entry.type">
                  <Ticket v-if="entry.type === 'banked'" :size="14" :stroke-width="2" />
                  <RotateCw v-else :size="13" :stroke-width="2.1" />
                </span>
                <span v-if="idx < visibleRecords.length - 1" class="timeline-line" />
              </div>

              <div class="timeline-body">
                <div class="timeline-meta">
                  <span class="record-kind" :data-type="entry.type">{{ t(`needtoken.radarV2.types.${entry.type}`) }}</span>
                  <span class="timeline-when">
                    {{ relativeAge(entry.announced_at) }} · <time :datetime="entry.announced_at">{{ slashDateTime(entry.announced_at) }}</time>
                  </span>
                </div>

                <RadarRecordText :text="entry.text" />

                <div class="timeline-foot">
                  <a v-if="entry.source.url" :href="entry.source.url" target="_blank" rel="noopener noreferrer">
                    {{ t(entry.source.type === 'x_post' ? 'needtoken.radarLive.originalPost' : 'needtoken.radarLive.viewSource') }}
                    <ArrowUpRight :size="13" aria-hidden="true" />
                  </a>
                  <span v-else class="timeline-no-link">{{ t('needtoken.radarLive.noSourceLink') }}</span>
                  <span v-if="gapLabel(entry, idx)" class="timeline-gap">{{ gapLabel(entry, idx) }}</span>
                </div>
              </div>
            </li>
          </ol>
          <p v-else class="records-empty">{{ t('needtoken.radarLive.noFilteredRecords') }}</p>

          <div v-if="remainingCount > 0" class="records-more-row">
            <button type="button" class="radar-more-btn" data-radar-more @click="recordLimit += RESET_RECORD_PAGE_SIZE">
              <span>{{ t('needtoken.radarV2.showMore', { count: remainingCount }) }}</span>
              <ChevronDown :size="15" aria-hidden="true" />
            </button>
          </div>
        </div>

        <aside class="guide-card" aria-labelledby="radar-guide-heading">
          <div class="guide-head">
            <h2 id="radar-guide-heading">{{ t('needtoken.radarV2.guideTitle') }}</h2>
            <Info :size="16" aria-hidden="true" />
          </div>
          <ul class="guide-list">
            <li>
              <p class="guide-term"><i class="dot-regular" aria-hidden="true" />{{ t('needtoken.radarV2.guide.regularTitle') }}</p>
              <p class="guide-desc">{{ t('needtoken.radarV2.guide.regularBody') }}</p>
            </li>
            <li>
              <p class="guide-term"><i class="dot-banked" aria-hidden="true" />{{ t('needtoken.radarV2.guide.bankedTitle') }}</p>
              <p class="guide-desc">{{ t('needtoken.radarV2.guide.bankedBody') }}</p>
            </li>
            <li>
              <p class="guide-term"><i class="dot-time" aria-hidden="true" />{{ t('needtoken.radarV2.guide.timeTitle') }}</p>
              <p class="guide-desc">{{ t('needtoken.radarV2.guide.timeBody') }}</p>
            </li>
          </ul>
        </aside>
      </section>
    </div>

    <!-- Placeholder Radar Panels (Claude / Grok) -->
    <div
      v-else
      role="tabpanel"
      :aria-labelledby="`radar-tab-${activeChannel}`"
      class="radar-panel"
    >
      <section class="placeholder-card">
        <span class="placeholder-badge">
          <Clock3 :size="13" aria-hidden="true" />
          {{ t('needtoken.radarV2.comingSoonBadge') }}
        </span>
        <header class="placeholder-head">
          <ProductBrandMark :brand="activeChannelMeta.brand" class="placeholder-brand" />
          <div>
            <h1>{{ t(`needtoken.radarV2.placeholder.${activeChannel}.title`) }}</h1>
            <p>{{ t(`needtoken.radarV2.placeholder.${activeChannel}.subtitle`) }}</p>
          </div>
        </header>

        <div class="placeholder-rules">
          <h2>{{ t(`needtoken.radarV2.placeholder.${activeChannel}.rulesTitle`) }}</h2>
          <div class="placeholder-grid">
            <article v-for="n in [1, 2, 3]" :key="n" class="placeholder-rule">
              <h3>{{ t(`needtoken.radarV2.placeholder.${activeChannel}.rule${n}Title`) }}</h3>
              <p>{{ t(`needtoken.radarV2.placeholder.${activeChannel}.rule${n}Body`) }}</p>
            </article>
          </div>
        </div>
      </section>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch, type ComponentPublicInstance } from 'vue'
import { useI18n } from 'vue-i18n'
import { ArrowUpRight, ChevronDown, Clock3, Info, RotateCw, Ticket } from 'lucide-vue-next'
import ProductBrandMark from './components/ProductBrandMark.vue'
import RadarCalendar from './components/RadarCalendar.vue'
import RadarRecordText from './components/RadarRecordText.vue'
import { useSlidingIndicator } from './composables/useSlidingIndicator'
import { usePageSeo } from '../../composables/usePageSeo'
import {
  fetchResetFeed,
  RESET_FEED_MAX_AGE_MS,
  RESET_FEED_POLL_MS,
  RESET_RECORD_PAGE_SIZE,
  resetElapsedTime,
  resetFeedIsExpired,
  resetGapDays,
  resetRecordPage,
  type ResetFeed,
  type ResetRecord,
} from './lib/resetRadar'

type RadarChannelId = 'codex' | 'claude' | 'grok'

const channels: ReadonlyArray<{ id: RadarChannelId; brand: 'gpt' | 'claude' | 'grok' }> = [
  { id: 'codex', brand: 'gpt' },
  { id: 'claude', brand: 'claude' },
  { id: 'grok', brand: 'grok' },
]

const { t, locale } = useI18n()
const activeChannel = ref<RadarChannelId>('codex')
const activeChannelMeta = computed(() => channels.find(c => c.id === activeChannel.value) ?? channels[0]!)

const tabRailEl = ref<HTMLElement | null>(null)
const tabEls: Array<HTMLButtonElement | null> = []
function setTabEl(el: Element | ComponentPublicInstance | null, idx: number) {
  tabEls[idx] = el instanceof HTMLButtonElement ? el : null
}

const { indicatorStyle: tabIndicatorStyle, indicatorReady: tabIndicatorReady } = useSlidingIndicator(
  tabRailEl,
  '[data-radar-tab-active="true"]',
  [() => activeChannel.value, () => locale.value],
)

function selectChannel(id: RadarChannelId) {
  activeChannel.value = id
  const idx = channels.findIndex(c => c.id === id)
  tabEls[idx]?.scrollIntoView({ block: 'nearest', inline: 'nearest' })
}

function onTabKeydown(event: KeyboardEvent) {
  const idx = channels.findIndex(c => c.id === activeChannel.value)
  let next: number | null = null
  if (event.key === 'ArrowRight') next = (idx + 1) % channels.length
  else if (event.key === 'ArrowLeft') next = (idx - 1 + channels.length) % channels.length
  else if (event.key === 'Home') next = 0
  else if (event.key === 'End') next = channels.length - 1
  if (next === null) return
  event.preventDefault()
  selectChannel(channels[next]!.id)
  tabEls[next]?.focus()
}

const filters = ['all', 'regular', 'banked'] as const
const filter = ref<typeof filters[number]>('all')
const feed = ref<ResetFeed | null>(null)
const loading = ref(true)
const loadError = ref(false)
const now = ref(Date.now())
const expired = computed(() => Boolean(feed.value && resetFeedIsExpired(feed.value, now.value)))
const usableFeed = computed(() => (expired.value ? null : feed.value))
const stale = computed(() => Boolean(usableFeed.value && (usableFeed.value.stale || loadError.value)))
const latest = computed(() => usableFeed.value?.records[0])
const recordLimit = ref(RESET_RECORD_PAGE_SIZE)
const filteredRecords = computed(() =>
  usableFeed.value?.records.filter(row => filter.value === 'all' || row.type === filter.value) || [],
)
const recordPage = computed(() => resetRecordPage(usableFeed.value?.records || [], filter.value, recordLimit.value))
const visibleRecords = computed(() => recordPage.value.records)
const remainingCount = computed(() => recordPage.value.remaining)
watch(filter, () => { recordLimit.value = RESET_RECORD_PAGE_SIZE })

const elapsed = computed(() => resetElapsedTime(latest.value?.announced_at || '', now.value))

function filterLabel(kind: typeof filters[number]) {
  if (kind === 'all') return t('needtoken.radarLive.filters.all')
  return t(`needtoken.radarV2.types.${kind}`)
}

function slashDateTime(value: string) {
  const date = new Date(value)
  if (!Number.isFinite(date.getTime())) return ''
  const parts = new Intl.DateTimeFormat('en-CA', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
    timeZone: 'Asia/Shanghai',
  }).formatToParts(date)
  const get = (type: string) => parts.find(p => p.type === type)?.value || ''
  return `${get('year')}/${get('month')}/${get('day')} ${get('hour')}:${get('minute')}`
}

function relativeAge(announcedAt: string) {
  const diffMs = Math.max(0, now.value - Date.parse(announcedAt))
  const hours = Math.floor(diffMs / 3_600_000)
  if (hours < 1) return t('needtoken.radarV2.justNow')
  if (hours < 24) return t('needtoken.radarV2.hoursAgo', { count: hours })
  return t('needtoken.radarV2.daysAgo', { count: Math.floor(hours / 24) })
}

function gapLabel(entry: ResetRecord, idx: number) {
  const older = filteredRecords.value[idx + 1]
  if (!older) return ''
  const gap = resetGapDays(entry.announced_at, older.announced_at)
  if (gap === null) return ''
  if (gap === 0) return t('needtoken.radarV2.gapSameDay')
  return t('needtoken.radarV2.gapDays', { count: gap })
}

let generation = 0
let disposed = false
let lastAttemptAt = 0
let controller: AbortController | null = null
let pollTimer: ReturnType<typeof setTimeout> | undefined
let expiryTimer: ReturnType<typeof setTimeout> | undefined

function clearPoll() {
  if (pollTimer !== undefined) clearTimeout(pollTimer)
  pollTimer = undefined
}

function scheduleExpiry(snapshot: ResetFeed) {
  if (expiryTimer !== undefined) clearTimeout(expiryTimer)
  expiryTimer = setTimeout(() => {
    now.value = Date.now()
  }, Math.max(0, Date.parse(snapshot.fetched_at) + RESET_FEED_MAX_AGE_MS - Date.now()))
}

function scheduleNext() {
  clearPoll()
  if (disposed || document.hidden) return
  pollTimer = setTimeout(() => {
    void loadFeed()
  }, Math.max(0, RESET_FEED_POLL_MS - (Date.now() - lastAttemptAt)))
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
    if (!disposed && request === generation) {
      feed.value = result
      loadError.value = false
      scheduleExpiry(result)
    }
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
  else if (lastAttemptAt === 0) void loadFeed()
}

usePageSeo({
  title: () => t('needtoken.radar.title'),
  description: () => t('needtoken.radarLive.description'),
  canonicalPath: () => '/tools/radar',
})

onMounted(() => {
  document.addEventListener('visibilitychange', handleVisibilityChange)
  void loadFeed()
})

onUnmounted(() => {
  disposed = true
  generation++
  controller?.abort()
  clearPoll()
  if (expiryTimer !== undefined) clearTimeout(expiryTimer)
  document.removeEventListener('visibilitychange', handleVisibilityChange)
})
</script>

<style scoped>
.reset-radar {
  width: 100%;
  max-width: var(--nt-content-max-width, 1200px);
  margin: 0 auto;
  padding: 28px 24px 64px;
  --rd-regular: #15803d;
  --rd-regular-bg: rgba(21, 128, 61, 0.1);
  --rd-banked: #c97a16;
  --rd-banked-bg: rgba(201, 122, 22, 0.12);
}

:global(.dark) .reset-radar {
  --rd-regular: #34d399;
  --rd-regular-bg: rgba(52, 211, 153, 0.12);
  --rd-banked: #eab338;
  --rd-banked-bg: rgba(234, 179, 56, 0.13);
}

/* Top radar channel switcher */
.radar-tabs {
  position: relative;
  display: inline-flex;
  align-items: center;
  gap: 4px;
  max-width: 100%;
  margin-bottom: 24px;
  padding: 4px;
  border: 1px solid var(--ui-border);
  border-radius: 14px;
  background: var(--nt-nav-rail-bg, var(--ui-bg-elevated));
  overflow-x: auto;
  scrollbar-width: none;
  scroll-snap-type: inline proximity;
}
.radar-tabs::-webkit-scrollbar { display: none; }

.radar-tab-thumb {
  border-radius: 10px;
  background: var(--nt-nav-thumb-bg, var(--ui-bg-elevated));
  box-shadow: var(--nt-nav-thumb-shadow);
}

.radar-tab {
  position: relative;
  z-index: 1;
  flex: none;
  scroll-snap-align: start;
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-height: 38px;
  padding: 7px 16px;
  border: 1px solid transparent;
  border-radius: 10px;
  color: var(--ui-text-muted);
  font-size: 13.5px;
  font-weight: 500;
  line-height: 1.35;
  white-space: nowrap;
  cursor: pointer;
  transition: color 160ms var(--nt-motion-ease, ease-out);
}

.radar-tab:hover,
.radar-tab[aria-selected="true"] {
  color: var(--ui-text-primary);
}

.radar-tab[aria-selected="true"]::after {
  content: '';
  position: absolute;
  inset: auto 16px 2px;
  border-block-end: 2px solid var(--ui-accent);
  border-radius: 999px;
}

.radar-tab-brand {
  width: 16px;
  height: 16px;
  opacity: 0.72;
  transition: opacity 160ms ease, transform 360ms var(--nt-ease-spring, ease-out);
}
.radar-tab-brand :deep(.product-brand-symbol) {
  width: 16px;
  height: 16px;
}
.radar-tab:hover .radar-tab-brand,
.radar-tab[aria-selected="true"] .radar-tab-brand {
  opacity: 1;
}

@media (prefers-reduced-motion: no-preference) {
  .radar-tab:hover .radar-tab-brand { transform: scale(1.1); }
  .radar-tab:active { transform: scale(0.97); transition-duration: var(--nt-duration-press, 80ms); }
}

.radar-tab:focus-visible {
  outline: 2px solid var(--ui-accent);
  outline-offset: 2px;
}

/* Hero area */
.radar-hero {
  padding-bottom: 28px;
}

.hero-topline {
  display: flex;
  align-items: center;
  margin-bottom: 12px;
}

.tool-capability,
.radar-freshness {
  display: inline-flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  color: var(--ui-text-secondary);
  font-size: 12px;
}

.freshness-mark {
  width: 7px;
  height: 7px;
  flex: none;
  border-radius: 50%;
  background: var(--ui-accent);
}

.poll-note {
  padding-left: 8px;
  border-left: 1px solid var(--ui-border);
  color: var(--ui-text-muted);
}

.radar-freshness[data-stale="true"] { color: var(--nt-warning-ink); }
.radar-freshness[data-stale="true"] .freshness-mark { background: var(--nt-warning-ink); }

.radar-heading h1 {
  color: var(--ui-text-primary);
  font-size: clamp(26px, 3vw, 36px);
  line-height: 1.2;
  font-weight: 600;
  letter-spacing: -0.025em;
}

.radar-heading > p {
  margin-top: 8px;
  color: var(--ui-text-secondary);
  font-size: 14px;
  line-height: 1.6;
}

.radar-overview {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(320px, 470px);
  gap: 32px 48px;
  align-items: end;
  margin-top: 28px;
}

.overview-main { min-width: 0; }

.elapsed-label {
  color: var(--ui-text-secondary);
  font-size: 13.5px;
  font-weight: 500;
}

.elapsed-time {
  display: flex;
  flex-wrap: wrap;
  align-items: baseline;
  gap: 18px;
  margin: 10px 0 14px;
  font-variant-numeric: tabular-nums;
}

.elapsed-time > span {
  display: inline-flex;
  align-items: baseline;
  gap: 6px;
}

.elapsed-time strong {
  color: var(--ui-text-primary);
  font-size: clamp(46px, 6vw, 76px);
  font-weight: 600;
  line-height: 1;
  letter-spacing: -0.03em;
}

.elapsed-time small {
  color: var(--ui-text-secondary);
  font-size: 17px;
}

.elapsed-time .elapsed-minutes strong {
  font-size: clamp(26px, 3vw, 34px);
  color: var(--ui-text-secondary);
  letter-spacing: -0.01em;
}

.elapsed-time .elapsed-minutes small {
  font-size: 13px;
}

.latest-date {
  display: flex;
  align-items: center;
  gap: 7px;
  color: var(--ui-text-primary);
  font-size: 13.5px;
  line-height: 1.5;
  font-variant-numeric: tabular-nums;
}

.latest-date svg {
  flex: none;
  color: var(--ui-text-muted);
}

.latest-date strong {
  font-weight: 600;
}

.latest-record-card {
  min-width: 0;
  padding: 18px 20px 14px;
  border: 1px solid var(--ui-border);
  border-radius: 16px;
  background: var(--ui-bg-elevated);
  box-shadow: var(--nt-card-shadow);
}

.record-label {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px 10px;
  margin-bottom: 10px;
}

.record-kind {
  display: inline-flex;
  align-items: center;
  padding: 2px 9px;
  border-radius: 999px;
  font-size: 12px;
  font-weight: 600;
  line-height: 1.5;
  color: var(--rd-regular);
  background: var(--rd-regular-bg);
}

.record-kind[data-type="banked"] {
  color: var(--rd-banked);
  background: var(--rd-banked-bg);
}

.record-origin,
.missing-source {
  color: var(--ui-text-muted);
  font-size: 12px;
  overflow-wrap: anywhere;
}

.record-label > a {
  margin-left: auto;
  display: inline-flex;
  align-items: center;
  gap: 4px;
  color: var(--ui-text-secondary);
  font-size: 12px;
}

.record-label > a:hover {
  color: var(--ui-accent);
}

/* Full-width Calendar */
.radar-calendar-section {
  margin-bottom: 24px;
}

/* Two-column Records + Reading Guide */
.radar-records-layout {
  display: grid;
  grid-template-columns: minmax(0, 1.85fr) minmax(260px, 0.9fr);
  gap: 20px;
  align-items: start;
}

.records-card,
.guide-card {
  min-width: 0;
  padding: 24px 26px;
  border: 1px solid var(--ui-border);
  border-radius: 20px;
  background: var(--ui-bg-elevated);
  box-shadow: var(--nt-card-shadow);
}

.records-card-head {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-start;
  justify-content: space-between;
  gap: 14px;
  margin-bottom: 22px;
}

.records-card-head h2,
.guide-head h2 {
  color: var(--ui-text-primary);
  font-size: 16px;
  font-weight: 600;
  line-height: 1.4;
}

.records-card-head p {
  margin-top: 4px;
  color: var(--ui-text-muted);
  font-size: 13px;
}

.radar-filters {
  display: inline-flex;
  gap: 3px;
  padding: 3px;
  border: 1px solid var(--ui-border);
  border-radius: 10px;
  background: var(--nt-surface-subtle);
}

.radar-filters button {
  padding: 5px 11px;
  border: 1px solid transparent;
  border-radius: 7px;
  color: var(--ui-text-muted);
  font-size: 12px;
  font-weight: 500;
  cursor: pointer;
}

.radar-filters button:hover {
  color: var(--ui-text-primary);
}

.radar-filters button[aria-pressed="true"] {
  color: var(--ui-text-primary);
  background: var(--ui-bg-elevated);
  border-color: var(--ui-border);
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.06);
}

.records-timeline {
  list-style: none;
  margin: 0;
  padding: 0;
}

.timeline-item {
  display: grid;
  grid-template-columns: 28px minmax(0, 1fr);
  gap: 14px;
}

.timeline-rail {
  display: flex;
  flex-direction: column;
  align-items: center;
}

.timeline-node {
  display: grid;
  place-items: center;
  width: 28px;
  height: 28px;
  flex: none;
  border-radius: 999px;
  border: 1px solid color-mix(in srgb, var(--rd-regular) 28%, transparent);
  background: var(--rd-regular-bg);
  color: var(--rd-regular);
}

.timeline-node[data-type="banked"] {
  border-color: color-mix(in srgb, var(--rd-banked) 30%, transparent);
  background: var(--rd-banked-bg);
  color: var(--rd-banked);
}

.timeline-line {
  width: 1px;
  flex: 1;
  min-height: 24px;
  margin: 6px 0;
  background: var(--ui-border);
}

.timeline-body {
  min-width: 0;
  padding-bottom: 24px;
}

.timeline-item:last-child .timeline-body {
  padding-bottom: 4px;
}

.timeline-meta {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px 10px;
  margin-bottom: 8px;
}

.timeline-when {
  color: var(--ui-text-muted);
  font-size: 12.5px;
  font-variant-numeric: tabular-nums;
}

.timeline-foot {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 14px;
  margin-top: 8px;
  font-size: 12px;
  color: var(--ui-text-muted);
}

.timeline-foot a {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  color: var(--ui-text-secondary);
}

.timeline-foot a:hover {
  color: var(--ui-accent);
}

.timeline-gap {
  font-variant-numeric: tabular-nums;
}

.records-empty {
  padding: 20px 0;
  color: var(--ui-text-muted);
  font-size: 13px;
}

.records-more-row {
  margin-top: 18px;
  padding-top: 16px;
  border-top: 1px solid var(--ui-border);
  display: flex;
  justify-content: center;
}

.radar-more-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-height: 38px;
  padding: 8px 18px;
  border: 1px solid var(--ui-border-strong);
  border-radius: 999px;
  background: var(--nt-surface-subtle);
  color: var(--ui-text-primary);
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
}

.radar-more-btn:hover {
  border-color: var(--nt-accent-border-strong);
  background: var(--ui-bg-soft);
  color: var(--ui-accent);
}

/* Right sticky guide card */
.guide-card {
  position: sticky;
  top: 84px;
}

.guide-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  color: var(--ui-text-muted);
}

.guide-list {
  display: grid;
  gap: 18px;
  margin-top: 18px;
  list-style: none;
  padding: 0;
}

.guide-term {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  color: var(--ui-text-primary);
  font-size: 13.5px;
  font-weight: 600;
}

.guide-term i {
  width: 7px;
  height: 7px;
  border-radius: 999px;
}
.dot-regular { background: var(--rd-regular); }
.dot-banked { background: var(--rd-banked); }
.dot-time { background: var(--ui-text-muted); }

.guide-desc {
  margin-top: 6px;
  padding-left: 15px;
  color: var(--ui-text-secondary);
  font-size: 13px;
  line-height: 1.65;
}

/* Placeholder panels (Claude / Grok) */
.placeholder-card {
  padding: 32px;
  border: 1px solid var(--ui-border);
  border-radius: 22px;
  background: var(--ui-bg-elevated);
  box-shadow: var(--nt-card-shadow);
}

.placeholder-badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 12px;
  border-radius: 999px;
  border: 1px solid var(--nt-accent-border);
  background: var(--ui-accent-soft);
  color: var(--ui-accent);
  font-size: 12px;
  font-weight: 600;
}

.placeholder-head {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-top: 18px;
}

.placeholder-brand {
  width: 44px;
  height: 44px;
  border-radius: 14px;
  border: 1px solid var(--ui-border);
  background: var(--nt-surface-subtle);
}

.placeholder-head h1 {
  color: var(--ui-text-primary);
  font-size: clamp(22px, 2.5vw, 28px);
  font-weight: 600;
}

.placeholder-head p {
  margin-top: 4px;
  color: var(--ui-text-secondary);
  font-size: 14px;
}

.placeholder-rules {
  margin-top: 28px;
  padding-top: 24px;
  border-top: 1px solid var(--ui-border);
}

.placeholder-rules h2 {
  color: var(--ui-text-primary);
  font-size: 15px;
  font-weight: 600;
  margin-bottom: 16px;
}

.placeholder-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 16px;
}

.placeholder-rule {
  padding: 18px 20px;
  border: 1px solid var(--ui-border);
  border-radius: 14px;
  background: var(--nt-surface-subtle);
}

.placeholder-rule h3 {
  color: var(--ui-text-primary);
  font-size: 14px;
  font-weight: 600;
}

.placeholder-rule p {
  margin-top: 8px;
  color: var(--ui-text-secondary);
  font-size: 13px;
  line-height: 1.65;
}

/* Loading & error states */
.stale-note {
  margin-top: 16px;
  padding: 10px 14px;
  border-left: 2px solid var(--nt-warning-ink);
  border-radius: 0 8px 8px 0;
  color: var(--ui-text-secondary);
  background: var(--nt-warning-bg);
  font-size: 13px;
}

.radar-load-error {
  display: flex;
  gap: 14px;
  padding: 24px;
  margin-top: 20px;
  border: 1px solid var(--ui-border);
  border-radius: 16px;
  background: var(--ui-bg-elevated);
}

.radar-load-error h2 {
  color: var(--ui-text-primary);
  font-size: 16px;
  font-weight: 600;
}

.radar-load-error p {
  margin-top: 6px;
  color: var(--ui-text-secondary);
  font-size: 13.5px;
}

.radar-action {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  margin-top: 14px;
  min-height: 36px;
  padding: 6px 14px;
  border: 1px solid var(--ui-border-strong);
  border-radius: 8px;
  color: var(--ui-text-primary);
  font-size: 12.5px;
}

.radar-loading {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(300px, 440px);
  gap: 32px;
  margin-top: 28px;
}

.skeleton-col { display: grid; gap: 12px; }
.skeleton-label { width: 110px; height: 14px; border-radius: 4px; background: var(--ui-bg-soft); }
.skeleton-timer { width: 70%; height: 68px; border-radius: 10px; background: var(--ui-bg-soft); }
.skeleton-date { width: 180px; height: 16px; border-radius: 4px; background: var(--ui-bg-soft); }
.skeleton-card { height: 140px; border-radius: 16px; background: var(--ui-bg-soft); }

@media (max-width: 960px) {
  .radar-overview {
    grid-template-columns: 1fr;
    gap: 20px;
  }
  .radar-records-layout {
    grid-template-columns: 1fr;
  }
  .guide-card {
    position: static;
  }
  .placeholder-grid {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 640px) {
  .reset-radar {
    padding: 20px 16px 48px;
  }
  .records-card,
  .guide-card,
  .placeholder-card {
    padding: 18px 16px;
    border-radius: 16px;
  }
  .radar-loading {
    grid-template-columns: 1fr;
  }
}
</style>
