<template>
  <div class="reset-radar">
    <!-- Radar Channel Tabs (Codex / Claude / Gemini / Grok) -->
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

    <!-- Unified Live Radar Panel for Codex / Claude / Gemini / Grok -->
    <div
      role="tabpanel"
      :aria-labelledby="`radar-tab-${activeChannel}`"
      class="radar-panel"
    >
      <section class="radar-hero" :aria-busy="loading">
        <div class="hero-topline">
          <div v-if="usableFeed" class="radar-freshness" :data-stale="stale" role="status">
            <span class="freshness-mark" aria-hidden="true" />
            <span>{{ deliveryLabel }}</span>
            <span class="poll-note">{{ t('needtoken.radarV2.delivery.cacheNote') }}</span>
          </div>
          <span v-else class="tool-capability">{{ t('needtoken.radarLayout.scope') }}</span>

          <div ref="subscribeBoxEl" class="hero-source-actions">
            <div class="radar-subscribe-wrap">
              <button
                type="button"
                class="radar-subscribe-btn"
                :class="{ 'is-subscribed': isActiveChannelSubscribed }"
                :aria-expanded="subscribePanelOpen"
                aria-haspopup="dialog"
                @click="handleSubscribeButtonClick"
              >
                <BellRing v-if="isActiveChannelSubscribed" :size="13" aria-hidden="true" />
                <Bell v-else :size="13" aria-hidden="true" />
                <span>{{ isActiveChannelSubscribed ? t('needtoken.radarV2.subscribe.subscribedAction') : t('needtoken.radarV2.subscribe.action') }}</span>
                <span v-if="subscribedTotalCount > 0" class="radar-sub-count">{{ subscribedTotalCount }}</span>
              </button>

              <Transition name="cal-tip">
                <div
                  v-if="subscribePanelOpen"
                  class="radar-subscribe-popover"
                  role="dialog"
                  :aria-label="t('needtoken.radarV2.subscribe.panelTitle')"
                >
                  <div class="radar-sub-head">
                    <strong>{{ t('needtoken.radarV2.subscribe.panelTitle') }}</strong>
                    <p>{{ t('needtoken.radarV2.subscribe.panelDesc') }}</p>
                  </div>

                  <div v-if="subLoading && !subPrefs" class="radar-sub-state">
                    {{ t('needtoken.common.loading') }}
                  </div>

                  <template v-else-if="subPrefs">
                    <div v-if="!subPrefs.email_bound" class="radar-sub-warning" role="alert">
                      <p>{{ t('needtoken.radarV2.subscribe.bindEmailHint') }}</p>
                      <RouterLink to="/me/security" class="radar-sub-bind-link" @click="subscribePanelOpen = false">
                        {{ t('needtoken.radarV2.subscribe.bindEmailAction') }}
                      </RouterLink>
                    </div>
                    <p v-else class="radar-sub-email">
                      <span>{{ t('needtoken.radarV2.subscribe.emailLabel') }}</span>
                      <strong>{{ subPrefs.email }}</strong>
                    </p>

                    <ul class="radar-sub-list">
                      <li v-for="ch in channels" :key="ch.id" class="radar-sub-item" :class="{ 'is-current': ch.id === activeChannel }">
                        <span class="radar-sub-item-info">
                          <ProductBrandMark :brand="ch.brand" class="radar-sub-brand" />
                          <span>{{ t(`needtoken.radarV2.tabs.${ch.id}`) }}</span>
                        </span>
                        <button
                          type="button"
                          role="switch"
                          class="radar-sub-switch"
                          :class="{ 'is-on': Boolean(subPrefs.channels[ch.id]) }"
                          :aria-checked="Boolean(subPrefs.channels[ch.id])"
                          :disabled="subSavingChannel === ch.id || (!subPrefs.email_bound && !subPrefs.channels[ch.id])"
                          @click="toggleChannelSubscription(ch.id)"
                        >
                          <span class="radar-sub-switch-thumb" />
                        </button>
                      </li>
                    </ul>
                  </template>

                  <p v-if="subError" class="radar-sub-error" role="alert">{{ subError }}</p>
                  <p v-else-if="subSuccess" class="radar-sub-success" role="status">{{ subSuccess }}</p>
                </div>
              </Transition>
            </div>

            <button
              type="button"
              class="radar-refresh-btn"
              :disabled="loading"
              :aria-busy="loading"
              @click="loadChannel(activeChannel, { force: true })"
            >
              <RotateCw :size="13" :class="{ 'is-spinning': loading }" aria-hidden="true" />
              <span>{{ t('needtoken.radarV2.delivery.refresh') }}</span>
            </button>
          </div>
        </div>

        <header class="radar-heading">
          <div class="radar-heading-title">
            <ProductBrandMark :brand="activeChannelMeta.brand" class="radar-heading-brand" />
            <h1>{{ channelHeadingTitle }}</h1>
          </div>
          <p>{{ channelHeadingSubtitle }}</p>
        </header>

        <!-- Gemini / AI Studio Pure Client-Side Daily RPD Countdown (0 network requests) -->
        <div v-if="activeChannel === 'gemini'" class="gemini-daily-strip" role="status">
          <div class="gemini-daily-left">
            <span class="gemini-daily-badge">
              <Clock3 :size="13" aria-hidden="true" />
              {{ t('needtoken.radarV2.geminiDaily.badge') }}
            </span>
            <strong class="gemini-daily-countdown">
              {{ t('needtoken.radarV2.geminiDaily.countdown', { hours: geminiDaily.hoursLeft, minutes: geminiDaily.minutesLeft }) }}
            </strong>
          </div>
          <p class="gemini-daily-rule">
            {{ t(geminiDaily.isPacificDst ? 'needtoken.radarV2.geminiDaily.dstRule' : 'needtoken.radarV2.geminiDaily.stdRule', { time: geminiDaily.resetHourBeijing }) }}
          </p>
        </div>

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
            <button type="button" class="radar-action" :aria-disabled="loading" :aria-busy="loading" @click="!loading && loadChannel(activeChannel, { force: true })">
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

            <!-- Latest reset announcement card -->
            <article v-if="latest" class="latest-record-card" :data-record-id="latest.id" :data-type="latest.type">
              <div class="record-label">
                <span class="record-kind" :data-type="latest.type">{{ recordTypeLabel(latest.type) }}</span>
                <span v-if="localizedScope(latest)" class="record-scope">{{ localizedScope(latest) }}</span>
                <span class="record-origin">
                  {{ t(`needtoken.radarLive.sources.${latest.source.type}`) }}<span v-if="latest.source.author"> · @{{ latest.source.author }}</span>
                </span>
                <a v-if="latest.source.url" :href="latest.source.url" target="_blank" rel="noopener noreferrer">
                  {{ t(latest.source.type === 'x_post' ? 'needtoken.radarLive.originalPost' : 'needtoken.radarLive.viewSource') }}
                  <ArrowUpRight :size="14" aria-hidden="true" />
                </a>
              </div>
              <RadarRecordText :text="localizedText(latest)" />
              <span v-if="!latest.source.url" class="missing-source">{{ t('needtoken.radarLive.noSourceLink') }}</span>
            </article>
          </div>
        </template>
      </section>

      <!-- Full-width Reset Calendar -->
      <section v-if="usableFeed" class="radar-calendar-section">
        <RadarCalendar :records="usableFeed.records" :now="now" :banked-label="activeChannel === 'gemini' ? t('needtoken.radarV2.types.preview') : undefined" />
      </section>

      <!-- Two-column Reset Records + Channel-Aware Reading Guide -->
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
                  <span class="record-kind" :data-type="entry.type">{{ recordTypeLabel(entry.type) }}</span>
                  <span v-if="localizedScope(entry)" class="record-scope">{{ localizedScope(entry) }}</span>
                  <span class="timeline-when">
                    {{ relativeAge(entry.announced_at) }} · <time :datetime="entry.announced_at">{{ slashDateTime(entry.announced_at) }}</time>
                  </span>
                </div>

                <RadarRecordText :text="localizedText(entry)" />

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
            <h2 id="radar-guide-heading">
              {{ activeChannel === 'codex' ? t('needtoken.radarV2.guideTitle') : t(`needtoken.radarV2.placeholder.${activeChannel}.rulesTitle`) }}
            </h2>
            <Info :size="16" aria-hidden="true" />
          </div>
          <ul v-if="activeChannel === 'codex'" class="guide-list">
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
          <ul v-else class="guide-list">
            <li v-for="(n, idx) in [1, 2, 3]" :key="n">
              <p class="guide-term">
                <i :class="idx === 0 ? 'dot-regular' : idx === 1 ? 'dot-banked' : 'dot-time'" aria-hidden="true" />
                {{ t(`needtoken.radarV2.placeholder.${activeChannel}.rule${n}Title`) }}
              </p>
              <p class="guide-desc">{{ t(`needtoken.radarV2.placeholder.${activeChannel}.rule${n}Body`) }}</p>
            </li>
          </ul>
        </aside>
      </section>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive, ref, watch, type ComponentPublicInstance } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import { ArrowUpRight, Bell, BellRing, ChevronDown, Clock3, Info, RotateCw, Ticket } from 'lucide-vue-next'
import ProductBrandMark from './components/ProductBrandMark.vue'
import RadarCalendar from './components/RadarCalendar.vue'
import RadarRecordText from './components/RadarRecordText.vue'
import { useSlidingIndicator } from './composables/useSlidingIndicator'
import { usePageSeo } from '../../composables/usePageSeo'
import { useUserAuthStore } from '../../stores/userAuth'
import { radarSubscriptionAPI, type RadarSubscriptionPreferences } from '../../api/radar'
import {
  fetchChannelResetFeed,
  geminiDailyResetSchedule,
  RADAR_CHANNELS,
  readCachedChannelFeed,
  RESET_CLIENT_CACHE_TTL_MS,
  RESET_RECORD_PAGE_SIZE,
  resetElapsedTime,
  resetFeedIsExpired,
  resetGapDays,
  resetRecordPage,
  type RadarChannelId,
  type ResetFeed,
  type ResetRecord,
} from './lib/resetRadar'

const channels = RADAR_CHANNELS
const { t, locale } = useI18n()
const route = useRoute()
const router = useRouter()
const userAuth = useUserAuthStore()
const initialChannelQuery = typeof route.query.channel === 'string' ? route.query.channel.toLowerCase() : ''
const initialChannel: RadarChannelId = channels.some(c => c.id === initialChannelQuery)
  ? (initialChannelQuery as RadarChannelId)
  : 'codex'
const activeChannel = ref<RadarChannelId>(initialChannel)
const activeChannelMeta = computed(() => channels.find(c => c.id === activeChannel.value) ?? channels[0]!)

const subscribeBoxEl = ref<HTMLElement | null>(null)
const subscribePanelOpen = ref(false)
const subLoading = ref(false)
const subSavingChannel = ref<RadarChannelId | null>(null)
const subPrefs = ref<RadarSubscriptionPreferences | null>(null)
const subError = ref('')
const subSuccess = ref('')

const isActiveChannelSubscribed = computed(() => Boolean(subPrefs.value?.channels?.[activeChannel.value]))
const subscribedTotalCount = computed(() => subPrefs.value?.subscribed_count ?? 0)

async function loadSubscriptionPreferences() {
  if (!userAuth.isAuthenticated) {
    subPrefs.value = null
    return
  }
  subLoading.value = true
  subError.value = ''
  try {
    const res = await radarSubscriptionAPI.getPreferences()
    subPrefs.value = res.data?.data ?? null
  } catch (err: any) {
    subError.value = err?.response?.data?.msg || t('needtoken.radarV2.subscribe.loadFailed')
  } finally {
    subLoading.value = false
  }
}

async function handleSubscribeButtonClick() {
  if (!userAuth.isAuthenticated) {
    void router.push({
      path: '/auth/login',
      query: { redirect: `/tools/radar?channel=${activeChannel.value}` },
    })
    return
  }
  subscribePanelOpen.value = !subscribePanelOpen.value
  subSuccess.value = ''
  subError.value = ''
  if (subscribePanelOpen.value) {
    await loadSubscriptionPreferences()
  }
}

async function toggleChannelSubscription(channel: RadarChannelId) {
  if (!subPrefs.value || subSavingChannel.value) return
  const nextEnabled = !subPrefs.value.channels[channel]
  subSavingChannel.value = channel
  subError.value = ''
  subSuccess.value = ''
  try {
    const res = await radarSubscriptionAPI.updatePreferences({ [channel]: nextEnabled })
    if (res.data?.data) {
      subPrefs.value = res.data.data
    }
    subSuccess.value = nextEnabled
      ? t('needtoken.radarV2.subscribe.savedEnabled', { channel: t(`needtoken.radarV2.tabs.${channel}`) })
      : t('needtoken.radarV2.subscribe.savedDisabled', { channel: t(`needtoken.radarV2.tabs.${channel}`) })
  } catch (err: any) {
    subError.value = err?.response?.data?.msg || t('needtoken.radarV2.subscribe.saveFailed')
  } finally {
    subSavingChannel.value = null
  }
}

function handleDocumentPointerDown(event: MouseEvent) {
  if (!subscribePanelOpen.value) return
  if (event.target instanceof Node && subscribeBoxEl.value?.contains(event.target)) return
  subscribePanelOpen.value = false
}

function handleDocumentKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape' && subscribePanelOpen.value) {
    subscribePanelOpen.value = false
  }
}

watch(() => userAuth.isAuthenticated, (authed) => {
  if (authed) {
    void loadSubscriptionPreferences()
  } else {
    subPrefs.value = null
    subscribePanelOpen.value = false
  }
})

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

const filters = ['all', 'regular', 'banked'] as const
const filter = ref<typeof filters[number]>('all')
const recordLimit = ref(RESET_RECORD_PAGE_SIZE)
const now = ref(Date.now())

const feedsByChannel = reactive<Record<RadarChannelId, ResetFeed | null>>({
  codex: null,
  claude: null,
  gemini: null,
  grok: null,
})
const loadingByChannel = reactive<Record<RadarChannelId, boolean>>({
  codex: false,
  claude: false,
  gemini: false,
  grok: false,
})
const errorByChannel = reactive<Record<RadarChannelId, boolean>>({
  codex: false,
  claude: false,
  gemini: false,
  grok: false,
})

const feed = computed(() => feedsByChannel[activeChannel.value])
const loading = computed(() => loadingByChannel[activeChannel.value])
const loadError = computed(() => errorByChannel[activeChannel.value])
const expired = computed(() => Boolean(feed.value && resetFeedIsExpired(feed.value, now.value)))
const usableFeed = computed(() => (expired.value ? null : feed.value))
const stale = computed(() => Boolean(usableFeed.value && (usableFeed.value.stale || loadError.value)))
const latest = computed(() => usableFeed.value?.records[0])
const filteredRecords = computed(() =>
  usableFeed.value?.records.filter(row => filter.value === 'all' || row.type === filter.value) || [],
)
const recordPage = computed(() => resetRecordPage(usableFeed.value?.records || [], filter.value, recordLimit.value))
const visibleRecords = computed(() => recordPage.value.records)
const remainingCount = computed(() => recordPage.value.remaining)
const elapsed = computed(() => resetElapsedTime(latest.value?.announced_at || '', now.value))
const geminiDaily = computed(() => geminiDailyResetSchedule(now.value))

const channelHeadingTitle = computed(() =>
  activeChannel.value === 'codex'
    ? t('needtoken.radar.title')
    : t(`needtoken.radarV2.channels.${activeChannel.value}.title`),
)
const channelHeadingSubtitle = computed(() =>
  activeChannel.value === 'codex'
    ? t('needtoken.radarLayout.description')
    : t(`needtoken.radarV2.channels.${activeChannel.value}.subtitle`),
)
const deliveryLabel = computed(() => {
  if (stale.value) return t('needtoken.radarLive.stale')
  return t('needtoken.radarV2.delivery.cached_proxy')
})

watch(filter, () => { recordLimit.value = RESET_RECORD_PAGE_SIZE })
watch(activeChannel, () => {
  filter.value = 'all'
  recordLimit.value = RESET_RECORD_PAGE_SIZE
})

function recordTypeLabel(kind: ResetRecord['type']) {
  if (activeChannel.value === 'gemini' && kind === 'banked') return t('needtoken.radarV2.types.preview')
  return t(`needtoken.radarV2.types.${kind}`)
}

function filterLabel(kind: typeof filters[number]) {
  if (kind === 'all') return t('needtoken.radarLive.filters.all')
  return recordTypeLabel(kind)
}

function localizedText(entry: ResetRecord) {
  return locale.value.startsWith('zh') && entry.text_zh ? entry.text_zh : entry.text
}

function localizedScope(entry: ResetRecord) {
  return locale.value.startsWith('zh') ? (entry.scope_zh || entry.scope || '') : (entry.scope || entry.scope_zh || '')
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

let disposed = false
const generations: Record<RadarChannelId, number> = { codex: 0, claude: 0, gemini: 0, grok: 0 }
const controllers: Partial<Record<RadarChannelId, AbortController>> = {}
let clockTimer: ReturnType<typeof setInterval> | undefined

async function loadChannel(channel: RadarChannelId, options: { force?: boolean } = {}) {
  if (disposed) return
  const currentNow = Date.now()
  now.value = currentNow

  if (!options.force) {
    const cached = readCachedChannelFeed(channel, currentNow)
    if (cached) {
      feedsByChannel[channel] = cached
      errorByChannel[channel] = false
      loadingByChannel[channel] = false
      return
    }
  }

  controllers[channel]?.abort()
  const requestGen = ++generations[channel]
  const ctrl = new AbortController()
  controllers[channel] = ctrl
  loadingByChannel[channel] = true

  try {
    const result = await fetchChannelResetFeed(channel, ctrl.signal, { force: options.force })
    if (!disposed && requestGen === generations[channel]) {
      feedsByChannel[channel] = result
      errorByChannel[channel] = false
      // Sync sibling caches populated by shared direct fetch (e.g. Claude + Grok).
      for (const ch of channels) {
        if (ch.id === channel) continue
        const siblingCached = readCachedChannelFeed(ch.id, Date.now())
        if (siblingCached && !feedsByChannel[ch.id]) {
          feedsByChannel[ch.id] = siblingCached
        }
      }
    }
  } catch {
    if (!disposed && requestGen === generations[channel]) {
      errorByChannel[channel] = true
    }
  } finally {
    if (!disposed && requestGen === generations[channel]) {
      controllers[channel] = undefined
      now.value = Date.now()
      loadingByChannel[channel] = false
    }
  }
}

function selectChannel(id: RadarChannelId) {
  activeChannel.value = id
  const idx = channels.findIndex(c => c.id === id)
  tabEls[idx]?.scrollIntoView({ block: 'nearest', inline: 'nearest' })
  void loadChannel(id)
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

function handleVisibilityChange() {
  now.value = Date.now()
  if (document.hidden) return
  const currentFeed = feedsByChannel[activeChannel.value]
  if (!loadingByChannel[activeChannel.value] && (!currentFeed || now.value - Date.parse(currentFeed.fetched_at) >= RESET_CLIENT_CACHE_TTL_MS)) {
    void loadChannel(activeChannel.value)
  }
}

usePageSeo({
  title: () => t('needtoken.radar.title'),
  description: () => t('needtoken.radarLive.description'),
  canonicalPath: () => '/tools/radar',
})

onMounted(() => {
  document.addEventListener('visibilitychange', handleVisibilityChange)
  document.addEventListener('mousedown', handleDocumentPointerDown)
  document.addEventListener('keydown', handleDocumentKeydown)
  // Local clock tick only (0 network requests) to keep elapsed & daily countdown fresh.
  clockTimer = setInterval(() => {
    if (!document.hidden) now.value = Date.now()
  }, 30_000)
  void loadChannel(activeChannel.value)
  if (userAuth.isAuthenticated) {
    void loadSubscriptionPreferences()
  }
})

onUnmounted(() => {
  disposed = true
  for (const ch of channels) {
    generations[ch.id]++
    controllers[ch.id]?.abort()
  }
  if (clockTimer !== undefined) clearInterval(clockTimer)
  document.removeEventListener('visibilitychange', handleVisibilityChange)
  document.removeEventListener('mousedown', handleDocumentPointerDown)
  document.removeEventListener('keydown', handleDocumentKeydown)
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
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 10px 16px;
  margin-bottom: 14px;
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

.hero-source-actions {
  display: inline-flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  font-size: 12px;
}

.radar-refresh-btn {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 4px 10px;
  border: 1px solid var(--ui-border);
  border-radius: 999px;
  background: var(--ui-bg-elevated);
  color: var(--ui-text-secondary);
  font-size: 12px;
  cursor: pointer;
}
.radar-refresh-btn:hover:not(:disabled) {
  border-color: var(--nt-accent-border-strong);
  color: var(--ui-accent);
}
.radar-refresh-btn:disabled {
  opacity: 0.6;
  cursor: default;
}

.radar-subscribe-wrap {
  position: relative;
}

.radar-subscribe-btn {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 4px 11px;
  border: 1px solid var(--ui-border);
  border-radius: 999px;
  background: var(--ui-bg-elevated);
  color: var(--ui-text-secondary);
  font-size: 12px;
  font-weight: 500;
  cursor: pointer;
  transition: border-color 150ms ease, color 150ms ease, background 150ms ease;
}
.radar-subscribe-btn:hover {
  border-color: var(--nt-accent-border-strong);
  color: var(--ui-accent);
}
.radar-subscribe-btn.is-subscribed {
  border-color: var(--nt-accent-border-strong, var(--ui-accent));
  background: var(--nt-accent-subtle, rgba(16, 185, 129, 0.1));
  color: var(--ui-accent);
}
.radar-sub-count {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 16px;
  height: 16px;
  padding: 0 4px;
  border-radius: 999px;
  background: var(--ui-accent);
  color: var(--ui-text-on-accent);
  font: 700 10px var(--nt-font-mono);
}

.radar-subscribe-popover {
  position: absolute;
  top: calc(100% + 8px);
  right: 0;
  z-index: 30;
  width: min(320px, calc(100vw - 32px));
  padding: 16px;
  border: 1px solid var(--ui-border-strong);
  border-radius: 16px;
  background: var(--ui-bg-elevated);
  box-shadow: var(--nt-popover-shadow, 0 18px 42px rgba(0, 0, 0, 0.28));
}

.radar-sub-head strong {
  display: block;
  color: var(--ui-text-primary);
  font-size: 14px;
  font-weight: 650;
}
.radar-sub-head p {
  margin-top: 4px;
  color: var(--ui-text-muted);
  font-size: 12px;
  line-height: 1.55;
}

.radar-sub-state {
  margin-top: 12px;
  color: var(--ui-text-muted);
  font-size: 12.5px;
}

.radar-sub-email {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin-top: 12px;
  padding: 8px 10px;
  border-radius: 10px;
  border: 1px solid var(--ui-border);
  background: var(--ui-bg-soft);
  font-size: 12px;
  color: var(--ui-text-muted);
}
.radar-sub-email strong {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--ui-text-primary);
  font-weight: 600;
}

.radar-sub-warning {
  margin-top: 12px;
  padding: 10px 12px;
  border-radius: 10px;
  border: 1px solid var(--nt-warning-border, rgba(245, 158, 11, 0.35));
  background: color-mix(in srgb, var(--nt-warning-ink, #f59e0b) 8%, transparent);
  color: var(--nt-warning-ink, #d97706);
  font-size: 12px;
  line-height: 1.55;
}
.radar-sub-bind-link {
  display: inline-block;
  margin-top: 6px;
  color: var(--ui-accent);
  font-weight: 600;
  text-decoration: underline;
}

.radar-sub-list {
  display: grid;
  gap: 8px;
  margin: 12px 0 0;
  padding: 0;
  list-style: none;
}
.radar-sub-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 8px 10px;
  border-radius: 10px;
  border: 1px solid var(--ui-border);
  background: var(--ui-bg-soft);
}
.radar-sub-item.is-current {
  border-color: var(--nt-accent-border, var(--ui-accent));
}
.radar-sub-item-info {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  color: var(--ui-text-primary);
  font-size: 12.5px;
  font-weight: 500;
}
.radar-sub-brand {
  width: 15px;
  height: 15px;
}
.radar-sub-brand :deep(.product-brand-symbol) {
  width: 15px;
  height: 15px;
}

.radar-sub-switch {
  position: relative;
  width: 34px;
  height: 20px;
  padding: 0;
  border: 1px solid var(--ui-border-strong);
  border-radius: 999px;
  background: var(--ui-bg-elevated);
  cursor: pointer;
  transition: background 150ms ease, border-color 150ms ease;
}
.radar-sub-switch.is-on {
  border-color: var(--ui-accent);
  background: var(--ui-accent);
}
.radar-sub-switch:disabled {
  opacity: 0.45;
  cursor: not-allowed;
}
.radar-sub-switch-thumb {
  position: absolute;
  top: 2px;
  left: 2px;
  width: 14px;
  height: 14px;
  border-radius: 50%;
  background: var(--ui-text-secondary);
  transition: transform 150ms ease, background 150ms ease;
}
.radar-sub-switch.is-on .radar-sub-switch-thumb {
  transform: translateX(14px);
  background: var(--ui-text-on-accent, #04130f);
}

.radar-sub-error {
  margin-top: 10px;
  color: var(--ui-danger, #ef4444);
  font-size: 12px;
}
.radar-sub-success {
  margin-top: 10px;
  color: var(--ui-accent);
  font-size: 12px;
}
.is-spinning {
  animation: nt-radar-spin 0.9s linear infinite;
}
@keyframes nt-radar-spin {
  to { transform: rotate(360deg); }
}

.radar-heading-title {
  display: flex;
  align-items: center;
  gap: 12px;
}

.radar-heading-brand {
  width: 30px;
  height: 30px;
}
.radar-heading-brand :deep(.product-brand-symbol) {
  width: 26px;
  height: 26px;
}

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

/* Gemini daily RPD countdown strip */
.gemini-daily-strip {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 10px 20px;
  margin-top: 18px;
  padding: 12px 16px;
  border: 1px solid color-mix(in srgb, var(--nt-brand-gemini, #3b7ef0) 26%, var(--ui-border));
  border-radius: 14px;
  background: color-mix(in srgb, var(--nt-brand-gemini, #3b7ef0) 6%, var(--ui-bg-elevated));
}

.gemini-daily-left {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px 14px;
}

.gemini-daily-badge {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 2px 9px;
  border-radius: 999px;
  background: color-mix(in srgb, var(--nt-brand-gemini, #3b7ef0) 14%, transparent);
  color: var(--nt-brand-gemini, #3b7ef0);
  font-size: 12px;
  font-weight: 600;
}

.gemini-daily-countdown {
  color: var(--ui-text-primary);
  font-size: 14px;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
}

.gemini-daily-rule {
  color: var(--ui-text-secondary);
  font-size: 12.5px;
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

.record-scope {
  display: inline-flex;
  align-items: center;
  padding: 2px 8px;
  border-radius: 999px;
  border: 1px solid var(--ui-border);
  background: var(--nt-surface-subtle);
  color: var(--ui-text-secondary);
  font-size: 11.5px;
  line-height: 1.45;
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
}

@media (max-width: 640px) {
  .reset-radar {
    padding: 20px 16px 48px;
  }
  .records-card,
  .guide-card {
    padding: 18px 16px;
    border-radius: 16px;
  }
  .radar-loading {
    grid-template-columns: 1fr;
  }
}
</style>
