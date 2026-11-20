<template>
  <div
    ref="rootEl"
    class="status-pills"
    :class="{ 'is-open': !!opened }"
    :aria-label="t('needtoken.statusStrip.label')"
    @mouseleave="onStripLeave"
    @focusout="onFocusOut"
    @keydown.esc.stop.prevent="close(true)"
  >
    <div class="status-pill-row">
      <button
        v-for="provider in providers"
        :key="provider.id"
        type="button"
        class="status-pill"
        :data-provider="provider.id"
        :data-color="publicStatusColor(provider.tone)"
        :aria-expanded="opened === provider.id"
        :aria-controls="opened === provider.id ? `nt-status-${provider.id}` : undefined"
        :aria-label="`${t(`needtoken.statusStrip.${provider.id}`)} · ${t(`needtoken.statusStrip.component.${provider.tone}`)}`"
        @mouseenter="hoverOpen(provider.id)"
        @focus="focusOpen(provider.id)"
        @click="toggle(provider.id)"
      >
        <span
          :key="`${provider.id}-${provider.source.sequence}`"
          class="status-bars"
          :class="[`is-${provider.tone}`, { 'is-checking': provider.source.refreshing && pageVisible, 'is-received': provider.fresh && provider.source.sequence > 0 }]"
          aria-hidden="true"
        >
          <i v-for="index in 5" :key="index" :style="{ '--bar-index': index - 1 }"></i>
        </span>
        <span>{{ t(`needtoken.statusStrip.${provider.id}`) }}</span>
      </button>
    </div>

    <Transition name="status-popover" mode="out-in" @before-enter="showEnteringPanel" @before-leave="hideLeavingPanel">
      <section
        v-if="activeProvider"
        :id="`nt-status-${activeProvider.id}`"
        :key="activeProvider.id"
        class="status-popover"
        :style="{ '--popover-origin': popoverOrigin }"
        :aria-label="t(`needtoken.statusStrip.${activeProvider.id}`)"
      >
        <header class="status-overview">
          <span class="status-badge-icon is-lg" :class="`is-${activeProvider.tone}`" aria-hidden="true">
            <Check v-if="activeProvider.tone === 'operational'" :size="12" :stroke-width="3" />
            <AlertTriangle v-else-if="activeProvider.tone === 'degraded_performance' || activeProvider.tone === 'under_maintenance'" :size="11" :stroke-width="2.6" />
            <AlertCircle v-else-if="activeProvider.tone === 'partial_outage' || activeProvider.tone === 'major_outage'" :size="12" :stroke-width="2.6" />
            <span v-else class="status-badge-dot"></span>
          </span>
          <h2>{{ headlineText(activeProvider) }}</h2>
        </header>

        <a
          v-if="activeProvider.activeIncident"
          class="status-active-banner"
          :class="`is-${activeProvider.tone}`"
          :href="activeProvider.activeIncident.url || activeProvider.source.href"
          target="_blank"
          rel="noopener noreferrer"
        >
          <span class="status-active-banner-text">{{ activeProvider.activeIncident.name }}</span>
          <ArrowUpRight :size="13" aria-hidden="true" />
        </a>

        <div v-if="activeProvider.series.length" class="status-series-list">
          <div
            v-for="(item, index) in activeProvider.series"
            :key="`${activeProvider.id}-${item.key}`"
            class="status-series-item"
          >
            <div class="status-series-head">
              <span class="status-badge-icon is-sm" :class="`is-${activeProvider.fresh ? item.status : 'unknown'}`" aria-hidden="true">
                <Check v-if="activeProvider.fresh && item.status === 'operational'" :size="10" :stroke-width="3" />
                <AlertTriangle v-else-if="activeProvider.fresh && (item.status === 'degraded_performance' || item.status === 'under_maintenance')" :size="10" :stroke-width="2.6" />
                <AlertCircle v-else-if="activeProvider.fresh && (item.status === 'partial_outage' || item.status === 'major_outage')" :size="10" :stroke-width="2.6" />
                <span v-else class="status-badge-dot"></span>
              </span>
              <span class="status-series-name">{{ item.label }}</span>
              <span v-if="seriesSubtitle(item)" class="status-series-meta">{{ seriesSubtitle(item) }}</span>
              <span class="status-series-uptime">
                {{ item.uptimePercent !== null ? t('needtoken.statusStrip.series.uptime', { value: formatPercent(item.uptimePercent) }) : t(`needtoken.statusStrip.component.${activeProvider.fresh ? item.status : 'unknown'}`) }}
              </span>
            </div>

            <StatusHistoryChart
              :days="item.days"
              :product-label="item.label"
              :source-url="activeProvider.source.href"
              :row-index="index"
            />
          </div>

          <div class="status-series-axis">
            <span>{{ t('needtoken.statusStrip.series.daysAgo') }}</span>
            <span>{{ t('needtoken.statusStrip.series.today') }}</span>
          </div>
        </div>

        <p v-else-if="activeProvider.source.state === 'loading'" class="status-empty-note" role="status">
          {{ t('needtoken.statusStrip.loading') }}
        </p>
        <p v-else class="status-empty-note" role="status">
          {{ t(activeProvider.source.error === 'timeout' ? 'needtoken.statusStrip.timeout' : 'needtoken.statusStrip.unavailable') }}
        </p>

        <footer class="status-footer">
          <a :href="activeProvider.source.href" target="_blank" rel="noopener noreferrer">
            {{ sourceHost(activeProvider.source.href) }}
            <ArrowUpRight :size="12" aria-hidden="true" />
          </a>
          <time
            v-if="activeProvider.source.fetchedAt"
            :datetime="activeProvider.source.fetchedAt"
            :title="t('needtoken.statusStrip.checked', { time: renderTime(activeProvider.source.fetchedAt) })"
          >
            {{ updatedAge(activeProvider.source.fetchedAt) }}
          </time>
        </footer>
      </section>
    </Transition>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { AlertCircle, AlertTriangle, ArrowUpRight, Check } from 'lucide-vue-next'
import StatusHistoryChart from './StatusHistoryChart.vue'
import {
  PUBLIC_STATUS_MAX_AGE_MS,
  fallbackSeriesFromSummary,
  incidentActive,
  parseCachedPublicStatus,
  parsePublicStatus,
  parsePublicStatusSeries,
  parseRetryAfterMs,
  publicStatusColor,
  publicStatusFresh,
  serializeCachedPublicStatus,
  seriesOverallTone,
  serviceStatusComponents,
  serviceStatusIncidents,
  serviceStatusTone,
  shouldFetchStatus,
  statusRefreshDelayMs,
  type PublicComponentStatus,
  type PublicStatusIncident,
  type PublicStatusSeries,
  type PublicStatusService,
  type PublicStatusSnapshot,
} from '../lib/publicStatus'

type Provider = PublicStatusService

const STATUS_CACHE_KEY_PREFIX = 'nt-public-status-v2:'

interface StatusSource {
  id: 'openai' | 'claude'
  href: string
  state: 'loading' | 'ready' | 'unavailable'
  snapshot: PublicStatusSnapshot | null
  series: PublicStatusSeries[]
  fetchedAt: string | null
  stale: boolean
  refreshing: boolean
  error: 'timeout' | 'unavailable' | null
  failures: number
  nextRetryAt: number
  sequence: number
}

interface ProviderView {
  id: Provider
  source: StatusSource
  fresh: boolean
  tone: PublicComponentStatus
  series: PublicStatusSeries[]
  activeIncident: PublicStatusIncident | null
}

const { t, locale } = useI18n()

const sources = ref<StatusSource[]>([
  {
    id: 'openai',
    href: 'https://status.openai.com/',
    state: 'loading',
    snapshot: null,
    series: [],
    fetchedAt: null,
    stale: true,
    refreshing: false,
    error: null,
    failures: 0,
    nextRetryAt: 0,
    sequence: 0,
  },
  {
    id: 'claude',
    href: 'https://status.claude.com/',
    state: 'loading',
    snapshot: null,
    series: [],
    fetchedAt: null,
    stale: true,
    refreshing: false,
    error: null,
    failures: 0,
    nextRetryAt: 0,
    sequence: 0,
  },
])

const rootEl = ref<HTMLElement | null>(null)
const opened = ref<Provider | null>(null)
const pinned = ref(false)
const finePointer = ref(false)
const pageVisible = ref(true)
const now = ref(Date.now())

let controller: AbortController | undefined
let pointerQuery: MediaQueryList | undefined
let timeout: number | undefined
let refreshTimer: number | undefined
let expiryTimer: number | undefined
let hoverCloseTimer: number | undefined
let inFlight = false
let currentRun = 0
let disposed = false

const hideLeavingPanel = (element: Element) => {
  element.setAttribute('inert', '')
  element.setAttribute('aria-hidden', 'true')
}
const showEnteringPanel = (element: Element) => {
  element.removeAttribute('inert')
  element.removeAttribute('aria-hidden')
}

const providers = computed<ProviderView[]>(() =>
  sources.value.map(source => {
    const id: Provider = source.id === 'openai' ? 'gpt' : 'claude'
    const fresh = source.state === 'ready' && publicStatusFresh(source.fetchedAt, now.value)
    const series = source.series.length
      ? source.series
      : fallbackSeriesFromSummary(source.snapshot, id, now.value)

    const scopedIncidents = serviceStatusIncidents(
      source.snapshot?.incidents ?? [],
      source.snapshot?.components ?? [],
      id,
    )
    const activeIncident = fresh
      ? (scopedIncidents.find(incidentActive) ?? source.snapshot?.incidents.find(incidentActive) ?? null)
      : null

    const tone: PublicComponentStatus = series.length
      ? seriesOverallTone(series, fresh)
      : serviceStatusTone(
        serviceStatusComponents(source.snapshot?.components ?? [], id),
        scopedIncidents,
        fresh,
      )

    return {
      id,
      source,
      fresh,
      tone,
      series,
      activeIncident,
    }
  }),
)

const activeProvider = computed(() => providers.value.find(item => item.id === opened.value) ?? null)
const popoverOrigin = computed(() => (opened.value === 'gpt' ? 'calc(100% - 148px) 100%' : 'calc(100% - 48px) 100%'))

const headlineText = (provider: ProviderView) => {
  if (provider.source.state === 'loading' && !provider.source.snapshot) {
    return t('needtoken.statusStrip.headline.loading')
  }
  if (provider.source.state === 'unavailable') {
    return t('needtoken.statusStrip.headline.unavailable')
  }
  if (!provider.fresh) {
    return t('needtoken.statusStrip.headline.outdated')
  }
  const prefix = provider.id === 'gpt' ? 'gpt' : 'claude'
  if (provider.tone === 'operational') {
    return t(`needtoken.statusStrip.headline.${prefix}Operational`)
  }
  if (provider.tone === 'degraded_performance' || provider.tone === 'under_maintenance') {
    return t(`needtoken.statusStrip.headline.${prefix}Degraded`)
  }
  return t(`needtoken.statusStrip.headline.${prefix}Outage`)
}

const seriesSubtitle = (item: PublicStatusSeries): string => {
  if (item.componentCount > 1) {
    return t('needtoken.statusStrip.series.serviceCount', { count: item.componentCount })
  }
  const subtitleKey = `needtoken.statusStrip.series.subtitle.${item.key}`
  const translated = t(subtitleKey)
  return translated !== subtitleKey ? translated : ''
}

const formatPercent = (value: number) => value.toFixed(2)

const sourceHost = (href: string) => {
  try {
    return new URL(href).host
  } catch {
    return href
  }
}

const renderTime = (value: string) => new Date(value).toLocaleString(locale.value)

const updatedAge = (value: string) => {
  const minutes = Math.max(0, Math.floor((now.value - Date.parse(value)) / 60_000))
  return minutes
    ? t('needtoken.statusStrip.series.updatedMinutesAgo', { count: minutes })
    : t('needtoken.statusStrip.series.updatedJustNow')
}

function clearHoverCloseTimer() {
  window.clearTimeout(hoverCloseTimer)
  hoverCloseTimer = undefined
}

function close(restoreFocus = false) {
  clearHoverCloseTimer()
  const current = opened.value
  if (restoreFocus && current) {
    rootEl.value?.querySelector<HTMLButtonElement>(`[data-provider="${current}"]`)?.focus()
  }
  opened.value = null
  pinned.value = false
}

function hoverOpen(id: Provider) {
  if (!finePointer.value) return
  clearHoverCloseTimer()
  if (opened.value !== id) pinned.value = false
  opened.value = id
}

function onStripLeave() {
  if (!finePointer.value || pinned.value) return
  clearHoverCloseTimer()
  hoverCloseTimer = window.setTimeout(() => {
    if (!pinned.value && !rootEl.value?.contains(document.activeElement)) close()
  }, 120)
}

function focusOpen(id: Provider) {
  clearHoverCloseTimer()
  if (opened.value !== id) pinned.value = false
  opened.value = id
}

function toggle(id: Provider) {
  clearHoverCloseTimer()
  if (opened.value === id && pinned.value) {
    close()
    return
  }
  opened.value = id
  pinned.value = true
}

function onFocusOut(event: FocusEvent) {
  if (event.relatedTarget instanceof Node && rootEl.value?.contains(event.relatedTarget)) return
  if (opened.value) close()
}

function onOutsidePointer(event: PointerEvent) {
  if (event.target instanceof Node && !rootEl.value?.contains(event.target)) close()
}

function onPointerChange() {
  finePointer.value = pointerQuery?.matches ?? false
}

function cacheKey(id: StatusSource['id']) {
  return `${STATUS_CACHE_KEY_PREFIX}${id}`
}

function hydrateSourceFromCache(source: StatusSource, currentTime: number): boolean {
  try {
    const cached = parseCachedPublicStatus(window.localStorage.getItem(cacheKey(source.id)), source.id, source.href, currentTime)
    if (!cached) return false
    const existingTs = Date.parse(source.fetchedAt || '')
    const cachedTs = Date.parse(cached.fetchedAt)
    if (Number.isFinite(existingTs) && existingTs >= cachedTs && source.snapshot) return true
    source.snapshot = cached.snapshot
    if (cached.series.length || !source.series.length) source.series = cached.series
    source.fetchedAt = cached.fetchedAt
    source.stale = cached.stale
    source.state = 'ready'
    source.error = null
    source.failures = 0
    source.nextRetryAt = 0
    return true
  } catch {
    return false
  }
}

function persistSourceCache(source: StatusSource, rawSummary: unknown, rawSeries: unknown, fetchedAt: string, stale: boolean) {
  try {
    window.localStorage.setItem(cacheKey(source.id), serializeCachedPublicStatus({
      provider: source.id,
      fetchedAt,
      stale,
      summary: rawSummary,
      series: rawSeries,
    }))
  } catch {
    /* localStorage 不可用（隐私模式或容量满）时静默降级为内存态 */
  }
}

function scheduleNextRefresh() {
  window.clearTimeout(refreshTimer)
  if (disposed || !pageVisible.value) return
  const currentTime = Date.now()
  const delays = sources.value.map(source => {
    if (source.nextRetryAt > currentTime) return source.nextRetryAt - currentTime
    return statusRefreshDelayMs({
      fetchedAt: source.fetchedAt,
      failures: source.failures,
      now: currentTime,
    })
  })
  const delay = Math.max(10_000, Math.min(...delays))
  refreshTimer = window.setTimeout(() => {
    void refresh()
  }, delay)
}

function scheduleExpiry() {
  window.clearTimeout(expiryTimer)
  const expiries = sources.value.flatMap(source => {
    const expiry = Date.parse(source.fetchedAt || '') + PUBLIC_STATUS_MAX_AGE_MS
    return Number.isFinite(expiry) && expiry > Date.now() ? [expiry] : []
  })
  if (!expiries.length || disposed || !pageVisible.value) return
  expiryTimer = window.setTimeout(() => {
    now.value = Date.now()
    scheduleExpiry()
  }, Math.max(0, Math.min(...expiries) - Date.now()))
}

async function loadSource(source: StatusSource, signal: AbortSignal, run: number) {
  let retryAfterMs: number | null = null
  try {
    const response = await fetch(`/api/v1/public/tools/service-status/${source.id}`, {
      signal,
      headers: { accept: 'application/json' },
      credentials: 'omit',
    })
    if (response.status === 429 || response.status === 503) {
      retryAfterMs = parseRetryAfterMs(response.headers.get('retry-after'), Date.now())
    }
    if (!response.ok) throw new Error('status unavailable')
    const body = await response.json()
    const data = body?.status_code === 0 ? body.data : null
    if (
      !data
      || data.provider !== source.id
      || typeof data.fetched_at !== 'string'
      || !Number.isFinite(Date.parse(data.fetched_at))
      || typeof data.stale !== 'boolean'
    ) {
      throw new Error('invalid status response')
    }
    const snapshot = parsePublicStatus(data.summary, source.href)
    if (!snapshot.ok) throw new Error('invalid status payload')
    if (disposed || run !== currentRun || !pageVisible.value) return
    const parsedSeries = parsePublicStatusSeries(data.series, source.href)
    source.snapshot = snapshot
    // 若后端临时未返回 90 天序列，保留内存中尚未过期的既有序列避免图表闪烁清空
    if (parsedSeries.length || !publicStatusFresh(source.fetchedAt, Date.now())) {
      source.series = parsedSeries
    }
    source.fetchedAt = data.fetched_at
    source.stale = data.stale
    source.state = 'ready'
    source.error = null
    source.failures = 0
    source.nextRetryAt = 0
    source.sequence++
    persistSourceCache(source, data.summary, parsedSeries.length ? data.series : [], data.fetched_at, data.stale)
  } catch {
    if (disposed || run !== currentRun || !pageVisible.value) return
    const currentTime = Date.now()
    source.failures = Math.min(source.failures + 1, 6)
    const backoffMs = statusRefreshDelayMs({
      fetchedAt: source.fetchedAt,
      failures: source.failures,
      retryAfterMs,
      now: currentTime,
    })
    source.nextRetryAt = currentTime + backoffMs
    // 已有仍在有效期内的快照时，遇到 429 或临时超时不清空状态，仅记录退避时间等待平滑恢复
    const hasUsableSnapshot = source.snapshot !== null && publicStatusFresh(source.fetchedAt, currentTime)
    if (!hasUsableSnapshot) {
      source.state = 'unavailable'
      source.stale = true
      source.error = signal.aborted ? 'timeout' : 'unavailable'
      source.sequence++
    }
  } finally {
    if (run === currentRun) source.refreshing = false
  }
}

async function refresh() {
  if (disposed || !pageVisible.value || inFlight) return
  const currentTime = Date.now()
  now.value = currentTime
  sources.value.forEach(source => {
    hydrateSourceFromCache(source, currentTime)
  })
  const targets = sources.value.filter(source => shouldFetchStatus(source.fetchedAt, source.nextRetryAt, currentTime))
  if (!targets.length) {
    scheduleExpiry()
    scheduleNextRefresh()
    return
  }
  window.clearTimeout(refreshTimer)
  inFlight = true
  const run = ++currentRun
  controller = new AbortController()
  const activeController = controller
  targets.forEach(source => {
    source.refreshing = true
  })
  timeout = window.setTimeout(() => activeController.abort(), 8000)
  try {
    await Promise.all(targets.map(source => loadSource(source, activeController.signal, run)))
  } finally {
    if (run !== currentRun) return
    window.clearTimeout(timeout)
    inFlight = false
    controller = undefined
    now.value = Date.now()
    scheduleExpiry()
    scheduleNextRefresh()
  }
}

function onStorageSync(event: StorageEvent) {
  if (disposed || !event.key || !event.key.startsWith(STATUS_CACHE_KEY_PREFIX)) return
  const providerId = event.key.slice(STATUS_CACHE_KEY_PREFIX.length)
  const target = sources.value.find(source => source.id === providerId)
  if (!target) return
  const currentTime = Date.now()
  now.value = currentTime
  if (hydrateSourceFromCache(target, currentTime)) {
    scheduleExpiry()
    scheduleNextRefresh()
  }
}

function cancelRefresh() {
  currentRun++
  controller?.abort()
  controller = undefined
  inFlight = false
  sources.value.forEach(source => {
    source.refreshing = false
  })
  window.clearTimeout(timeout)
  window.clearTimeout(refreshTimer)
  window.clearTimeout(expiryTimer)
}

function onVisibilityChange() {
  pageVisible.value = !document.hidden
  now.value = Date.now()
  if (pageVisible.value) void refresh()
  else cancelRefresh()
}

onMounted(() => {
  pointerQuery = window.matchMedia('(hover: hover) and (pointer: fine)')
  onPointerChange()
  pointerQuery.addEventListener('change', onPointerChange)
  document.addEventListener('pointerdown', onOutsidePointer)
  document.addEventListener('visibilitychange', onVisibilityChange)
  window.addEventListener('storage', onStorageSync)
  onVisibilityChange()
})

onUnmounted(() => {
  disposed = true
  clearHoverCloseTimer()
  cancelRefresh()
  pointerQuery?.removeEventListener('change', onPointerChange)
  document.removeEventListener('pointerdown', onOutsidePointer)
  document.removeEventListener('visibilitychange', onVisibilityChange)
  window.removeEventListener('storage', onStorageSync)
})
</script>

<style scoped>
.status-pills {
  position: relative;
  display: inline-flex;
  align-items: center;
  --st-green: #22c58b;
  --st-yellow: #f2b830;
  --st-red: #ec5f54;
  --st-gray: #8a9792;
}

:global(.dark) .status-pills {
  --st-green: #34d399;
  --st-yellow: #eab338;
  --st-red: #f06a5d;
  --st-gray: #6f7d89;
}

/* Invisible hover bridge between pills and popover */
.status-pills::before {
  content: '';
  position: absolute;
  bottom: 100%;
  right: 0;
  width: 100%;
  height: 14px;
  pointer-events: none;
}

.status-pills.is-open::before {
  pointer-events: auto;
}

.status-pill-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: flex-end;
  gap: 8px;
}

.status-pill {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-height: 32px;
  padding: 5px 12px;
  border: 1px solid var(--ui-border);
  border-radius: 999px;
  background: var(--nt-surface-subtle);
  color: var(--ui-text-secondary);
  font-size: 11.5px;
  font-weight: 500;
  line-height: 1.4;
  cursor: pointer;
  transition:
    color 160ms ease,
    background-color 160ms ease,
    border-color 160ms ease,
    box-shadow 180ms ease;
}

@media (prefers-reduced-motion: no-preference) {
  .status-pill {
    transition:
      color 160ms ease,
      background-color 160ms ease,
      border-color 160ms ease,
      box-shadow 180ms ease,
      transform var(--nt-duration-rebound, 480ms) var(--nt-ease-spring, cubic-bezier(0.34, 1.4, 0.64, 1));
  }
  .status-pill:active {
    transform: scale(0.96);
    transition-duration: var(--nt-duration-press, 80ms);
  }
}

.status-pill:hover,
.status-pill[aria-expanded="true"] {
  color: var(--ui-text-primary);
  border-color: var(--ui-border-strong);
  background: var(--ui-bg-soft);
}

.status-pill[aria-expanded="true"] {
  border-color: var(--nt-accent-border-strong, var(--ui-border-strong));
  box-shadow: 0 2px 10px -4px var(--ui-accent-soft);
}

.status-pill:focus-visible,
.status-popover a:focus-visible {
  outline: 2px solid var(--ui-accent);
  outline-offset: 3px;
}

.status-bars {
  display: inline-flex;
  align-items: center;
  gap: 2px;
  height: 13px;
  color: var(--st-gray);
}

.status-bars i {
  display: block;
  width: 2px;
  height: 12px;
  border-radius: 1px;
  background: currentColor;
  transform-origin: center bottom;
}

.status-bars.is-operational { color: var(--st-green); }
.status-bars.is-degraded_performance,
.status-bars.is-under_maintenance { color: var(--st-yellow); }
.status-bars.is-partial_outage,
.status-bars.is-major_outage { color: var(--st-red); }

@media (prefers-reduced-motion: no-preference) {
  .status-bars.is-received i {
    animation: status-received 620ms var(--nt-motion-ease, ease-out) calc(var(--bar-index) * 55ms) 1;
  }
  .status-bars.is-checking i {
    animation: status-checking 700ms ease-in-out calc(var(--bar-index) * -100ms) infinite;
  }
}

@keyframes status-checking {
  0%, 100% { transform: scaleY(0.45); opacity: 0.5; }
  50% { transform: scaleY(1); opacity: 1; }
}

@keyframes status-received {
  0% { transform: scaleY(0.4); opacity: 0.4; }
  100% { transform: scaleY(1); opacity: 1; }
}

.status-popover {
  position: absolute;
  right: 0;
  bottom: calc(100% + 10px);
  z-index: 50;
  width: min(444px, calc(100vw - 32px));
  padding: 20px 22px 16px;
  border: 1px solid var(--ui-border-strong);
  border-radius: 18px;
  background: var(--ui-bg-elevated);
  box-shadow: var(--nt-popover-shadow);
  color: var(--ui-text-secondary);
  text-align: left;
  transform-origin: var(--popover-origin, bottom right);
}

.status-popover-enter-active {
  transition: opacity 180ms ease-out;
}
.status-popover-leave-active {
  transition: opacity 110ms ease-in;
  pointer-events: none;
}
.status-popover-enter-from,
.status-popover-leave-to {
  opacity: 0;
}

@media (prefers-reduced-motion: no-preference) {
  .status-popover-enter-active {
    transition:
      opacity 180ms ease-out,
      transform 280ms cubic-bezier(0.22, 1, 0.36, 1);
    transition:
      opacity 180ms ease-out,
      transform 320ms linear(0, 0.45 16%, 1.04 42%, 0.99 66%, 1.005 84%, 1);
  }
  .status-popover-leave-active {
    transition:
      opacity 120ms ease-in,
      transform 120ms ease-in;
  }
  .status-popover-enter-from {
    opacity: 0;
    transform: translateY(10px) scale(0.95);
  }
  .status-popover-leave-to {
    opacity: 0;
    transform: translateY(6px) scale(0.97);
  }
}

.status-overview {
  display: flex;
  align-items: center;
  gap: 10px;
}

.status-overview h2 {
  min-width: 0;
  color: var(--ui-text-primary);
  font-size: 15px;
  line-height: 1.35;
  font-weight: 600;
  letter-spacing: -0.01em;
}

.status-badge-icon {
  display: inline-grid;
  place-items: center;
  flex: none;
  border-radius: 999px;
  color: #ffffff;
  background: var(--st-gray);
}

.status-badge-icon.is-lg {
  width: 20px;
  height: 20px;
}

.status-badge-icon.is-sm {
  width: 17px;
  height: 17px;
}

.status-badge-icon.is-operational { background: var(--st-green); }
.status-badge-icon.is-degraded_performance,
.status-badge-icon.is-under_maintenance { background: var(--st-yellow); }
.status-badge-icon.is-partial_outage,
.status-badge-icon.is-major_outage { background: var(--st-red); }

.status-badge-dot {
  width: 6px;
  height: 6px;
  border-radius: 999px;
  background: #ffffff;
  opacity: 0.85;
}

.status-active-banner {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin-top: 12px;
  padding: 8px 11px;
  border-radius: 10px;
  border: 1px solid var(--nt-warning-border);
  background: var(--nt-warning-bg);
  color: var(--nt-warning-ink);
  font-size: 12px;
  line-height: 1.4;
}

.status-active-banner.is-partial_outage,
.status-active-banner.is-major_outage {
  border-color: var(--nt-danger-border);
  background: var(--nt-danger-bg);
  color: var(--nt-danger-ink);
}

.status-active-banner-text {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.status-active-banner svg {
  flex: none;
}

.status-series-list {
  display: grid;
  gap: 16px;
  margin-top: 18px;
}

.status-series-head {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.status-series-name {
  color: var(--ui-text-primary);
  font-size: 14px;
  font-weight: 600;
  line-height: 1.3;
  letter-spacing: -0.01em;
}

.status-series-meta {
  color: var(--ui-text-muted);
  font-size: 12px;
  line-height: 1.3;
  white-space: nowrap;
}

.status-series-uptime {
  margin-left: auto;
  color: var(--ui-text-muted);
  font-size: 12px;
  line-height: 1.3;
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}

.status-series-axis {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: -6px;
  color: var(--ui-text-muted);
  font-size: 11.5px;
  line-height: 1.3;
}

.status-empty-note {
  margin-top: 14px;
  font-size: 12.5px;
  line-height: 1.5;
  color: var(--ui-text-muted);
}

.status-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-top: 16px;
  padding-top: 13px;
  border-top: 1px solid var(--ui-border);
  color: var(--ui-text-muted);
  font-size: 12px;
  line-height: 1.35;
}

.status-footer a {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  color: var(--ui-text-secondary);
}

.status-footer a:hover {
  color: var(--ui-accent);
}

.status-footer time {
  white-space: nowrap;
  font-variant-numeric: tabular-nums;
}

@media (max-width: 640px) {
  .status-pills {
    width: 100%;
  }
  .status-pill-row {
    justify-content: flex-start;
    width: 100%;
  }
  .status-pill {
    min-height: 36px;
  }
  .status-popover {
    left: 0;
    right: auto;
    width: min(444px, 100%);
    padding: 16px 16px 14px;
    border-radius: 16px;
    transform-origin: bottom left;
  }
}
</style>
