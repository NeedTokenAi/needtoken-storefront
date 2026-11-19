<template>
  <div ref="rootEl" class="status-pills" :aria-label="t('needtoken.statusStrip.label')" @keydown.esc.stop.prevent="close(true)">
    <div v-for="provider in providers" :key="provider.id" class="status-pill-wrap" @mouseenter="hoverOpen(provider.id)" @mouseleave="hoverClose(provider.id)" @focusin="focusOpen(provider.id)" @focusout="onFocusOut($event, provider.id)">
      <button type="button" class="status-pill" :data-provider="provider.id" :data-color="publicStatusColor(provider.tone)" :aria-expanded="opened === provider.id" :aria-controls="`nt-status-${provider.id}`" :aria-label="`${t(`needtoken.statusStrip.${provider.id}`)} · ${t(`needtoken.statusStrip.component.${provider.tone}`)}`" @click="toggle(provider.id)">
        <span :key="`${provider.id}-${provider.source.sequence}`" class="status-bars" :class="[`is-${provider.tone}`, { 'is-checking': provider.source.refreshing && pageVisible, 'is-received': provider.fresh && provider.source.sequence > 0 }]" aria-hidden="true"><i v-for="index in 5" :key="index" :style="{ '--bar-index': index - 1 }"></i></span>
        {{ t(`needtoken.statusStrip.${provider.id}`) }}
      </button>
      <Transition name="status-popover" @before-enter="showEnteringPanel" @before-leave="hideLeavingPanel">
      <section v-if="opened === provider.id" :id="`nt-status-${provider.id}`" class="status-popover" :aria-label="t(`needtoken.statusStrip.${provider.id}`)">
        <header class="status-card-head">
          <span class="status-dot" :class="`is-${provider.tone}`" aria-hidden="true"></span>
          <h2>{{ t(`needtoken.statusStrip.scopeTitle.${provider.id}`) }}</h2>
          <span class="status-health">{{ t(`needtoken.statusStrip.component.${provider.tone}`) }}</span>
        </header>
        <div class="status-scope-row"><p class="status-scope">{{ t(`needtoken.statusStrip.compact.scope.${provider.id}`) }}</p><span v-if="provider.components.length" class="status-count">{{ t('needtoken.statusStrip.compact.componentCount', { count: provider.components.length }) }}</span></div>
        <p v-if="provider.source.state === 'loading'" class="status-warning" role="status">{{ t('needtoken.statusStrip.loading') }}</p>
        <p v-else-if="provider.source.state === 'unavailable'" class="status-warning" role="status">{{ t(provider.source.error === 'timeout' ? 'needtoken.statusStrip.timeout' : 'needtoken.statusStrip.unavailable') }}</p>
        <p v-else-if="!provider.fresh" class="status-warning" role="status">{{ t('needtoken.statusStrip.outdated') }}</p>
        <article v-if="provider.source.snapshot" class="status-incident" :class="{ 'is-old': !provider.fresh }" :data-phase="provider.phase === null ? 'unknown' : phases[provider.phase]">
          <div class="status-incident-meta"><span>{{ t(provider.fresh ? provider.incident ? incidentActive(provider.incident) ? 'needtoken.statusStrip.compact.activeIncident' : 'needtoken.statusStrip.compact.recentIncident' : 'needtoken.statusStrip.compact.currentSummary' : 'needtoken.statusStrip.compact.previousSnapshot') }}</span><time v-if="provider.incident && (provider.incident.startedAt || provider.incident.createdAt)" :datetime="provider.incident.startedAt || provider.incident.createdAt || undefined">{{ renderDate(provider.incident.startedAt || provider.incident.createdAt || '') }}</time></div>
          <h3 class="status-incident-title">{{ provider.incident?.name || t(provider.tone === 'operational' ? 'needtoken.statusStrip.compact.noIncidentTitle' : provider.tone === 'unknown' ? 'needtoken.statusStrip.compact.unknownSummary' : 'needtoken.statusStrip.compact.componentSummary') }}</h3>
          <p class="status-incident-update">{{ provider.incident ? provider.incident.latestUpdate?.body || t('needtoken.statusStrip.compact.noUpdate') : provider.affected.length ? t('needtoken.statusStrip.compact.affectedComponents', { names: provider.affected.map(component => component.name).join(' · ') }) : t(provider.components.length ? 'needtoken.statusStrip.compact.noIncidentBody' : 'needtoken.statusStrip.scopeEmpty') }}</p>
          <p v-if="provider.incident && provider.incidentComponents.length" class="status-affected">{{ t('needtoken.statusStrip.compact.affectedComponents', { names: provider.incidentComponents.map(component => component.name).join(' · ') }) }}</p>
          <ol v-if="provider.incident" class="status-phases" :aria-label="t('needtoken.statusStrip.compact.phaseLabel')">
            <li v-for="(phase, index) in phases" :key="phase" class="status-phase" :class="{ 'is-reached': provider.fresh && provider.phase !== null && index <= provider.phase, 'is-current': provider.fresh && index === provider.phase }" :aria-current="provider.fresh && index === provider.phase ? 'step' : undefined" :style="{ '--phase-index': index }"><span class="status-phase-track" aria-hidden="true"><i></i></span><span>{{ t(`needtoken.statusStrip.compact.phases.${phase}`) }}</span></li>
          </ol>
          <div class="status-incident-footer"><span>{{ provider.incident ? incidentTiming(provider.incident) : provider.affected.length ? t('needtoken.statusStrip.compact.noLinkedIncident') : t('needtoken.statusStrip.compact.noActiveIncident') }}</span><a :href="provider.incident?.url || provider.source.href" target="_blank" rel="noopener noreferrer">{{ t('needtoken.statusStrip.compact.officialProgress') }}<ArrowUpRight :size="12" aria-hidden="true" /></a></div>
          <details v-if="provider.incident && provider.incident.updates.length > 1" class="status-update-history">
            <summary>{{ t('needtoken.statusStrip.compact.allUpdates', { count: provider.incident.updates.length }) }}</summary>
            <ol><li v-for="(update, index) in provider.incident.updates" :key="`${update.updatedAt}-${index}`"><div><span>{{ phaseName(update.status) }}</span><time v-if="update.updatedAt" :datetime="update.updatedAt">{{ renderTime(update.updatedAt) }}</time></div><p>{{ update.body || t('needtoken.statusStrip.compact.noUpdate') }}</p></li></ol>
          </details>
          <details v-if="provider.otherIncidents.length" class="status-update-history">
            <summary>{{ t('needtoken.statusStrip.compact.otherIncidents', { count: provider.otherIncidents.length }) }}</summary>
            <ul><li v-for="incident in provider.otherIncidents" :key="incident.id || incident.name"><a :href="incident.url || provider.source.href" target="_blank" rel="noopener noreferrer">{{ incident.name }}<ArrowUpRight :size="12" aria-hidden="true" /></a><p>{{ incident.latestUpdate?.body }}</p></li></ul>
          </details>
        </article>
        <section class="status-history-region" :aria-label="t('needtoken.statusStrip.compact.historyTitle')">
          <div class="status-history-head"><h3>{{ t('needtoken.statusStrip.compact.historyTitle') }}</h3><span>{{ t(provider.historyReady ? 'needtoken.statusStrip.compact.incidentCount' : 'needtoken.statusStrip.compact.historyUnknown', { count: provider.historyIncidents.length }) }}</span></div>
          <StatusHistoryChart :days="provider.historyDays" :product-label="t(`needtoken.statusStrip.scopeTitle.${provider.id}`)" :history-ready="provider.historyReady" :source-url="provider.source.history.sourceUrl" :current-tone="provider.tone" :current-fresh="provider.fresh" :checked-at="provider.source.fetchedAt" />
          <div class="status-history-axis"><time :datetime="provider.historyDays[0]?.date">{{ provider.historyDays[0]?.date }}</time><span>UTC</span><time :datetime="provider.historyDays[89]?.date">{{ t('needtoken.statusStrip.compact.today') }}</time></div>
          <p class="status-history-note">{{ t(provider.historyReady ? provider.historyIncidents.length ? 'needtoken.statusStrip.compact.partialHistory' : 'needtoken.statusStrip.compact.noScopedHistory' : 'needtoken.statusStrip.compact.historyUnavailable') }}</p>
          <details v-if="provider.historyIncidents.length" class="status-history-events"><summary>{{ t('needtoken.statusStrip.compact.recentRecords') }}</summary><ul><li v-for="incident in provider.historyIncidents.slice(0, 3)" :key="incident.id || incident.name"><time v-if="incident.startedAt || incident.createdAt" :datetime="incident.startedAt || incident.createdAt || undefined">{{ renderDate(incident.startedAt || incident.createdAt || '') }}</time><a :href="incident.url || provider.source.history.sourceUrl" target="_blank" rel="noopener noreferrer">{{ incident.name }}<ArrowUpRight :size="12" aria-hidden="true" /></a></li></ul></details>
        </section>
        <footer class="status-refresh-row"><a :href="provider.source.href" target="_blank" rel="noopener noreferrer">{{ sourceHost(provider.source.href) }}<ArrowUpRight :size="12" aria-hidden="true" /></a><time v-if="provider.source.fetchedAt" :datetime="provider.source.fetchedAt" :title="t('needtoken.statusStrip.checked', { time: renderTime(provider.source.fetchedAt) })">{{ checkedAge(provider.source.fetchedAt) }}</time><button type="button" :aria-disabled="provider.source.refreshing" :aria-busy="provider.source.refreshing" @click="refresh">{{ t(provider.source.refreshing ? 'needtoken.statusStrip.compact.refreshing' : 'needtoken.statusStrip.refreshAction') }}</button></footer>
      </section>
      </Transition>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { ArrowUpRight } from 'lucide-vue-next'
import StatusHistoryChart from './StatusHistoryChart.vue'
import { incidentActive, incidentElapsedMinutes, incidentPhase, mergePublicStatusIncidents, parsePublicStatus, parsePublicStatusHistory, publicHistoryDays, publicHistoryIncidents, publicStatusColor, publicStatusFresh, serviceStatusComponents, serviceStatusIncidents, serviceStatusTone, PUBLIC_STATUS_MAX_AGE_MS, type PublicStatusHistory, type PublicStatusIncident, type PublicStatusService, type PublicStatusSnapshot } from '../lib/publicStatus'

type Provider = PublicStatusService
interface StatusSource {
  id: 'openai' | 'claude'
  href: string
  state: 'loading' | 'ready' | 'unavailable'
  snapshot: PublicStatusSnapshot | null
  history: PublicStatusHistory
  fetchedAt: string | null
  stale: boolean
  refreshing: boolean
  error: 'timeout' | 'unavailable' | null
  sequence: number
}
const { t, locale } = useI18n()
const sources = ref<StatusSource[]>([
  { id: 'openai', href: 'https://status.openai.com/', history: { state: 'unavailable', coverage: 'partial', sourceUrl: 'https://status.openai.com/', incidents: [] }, state: 'loading', snapshot: null, fetchedAt: null, stale: true, refreshing: false, error: null, sequence: 0 },
  { id: 'claude', href: 'https://status.claude.com/', history: { state: 'unavailable', coverage: 'partial', sourceUrl: 'https://status.claude.com/', incidents: [] }, state: 'loading', snapshot: null, fetchedAt: null, stale: true, refreshing: false, error: null, sequence: 0 },
])
const rootEl = ref<HTMLElement | null>(null)
const hideLeavingPanel = (element: Element) => { element.setAttribute('inert', ''); element.setAttribute('aria-hidden', 'true') }
const showEnteringPanel = (element: Element) => { element.removeAttribute('inert'); element.removeAttribute('aria-hidden') }
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
let inFlight = false
let currentRun = 0
let disposed = false
const phases = ['investigating', 'identified', 'monitoring', 'resolved'] as const
const providers = computed(() => sources.value.map(source => {
  const id: Provider = source.id === 'openai' ? 'gpt' : 'claude'
  const components = serviceStatusComponents(source.snapshot?.components ?? [], id)
  const fresh = source.state === 'ready' && !source.stale && publicStatusFresh(source.fetchedAt, now.value)
  const incidents = serviceStatusIncidents(source.snapshot?.incidents ?? [], source.snapshot?.components ?? [], id)
  const historyReady = fresh && source.history.state === 'ready'
  const historyIncidents = historyReady ? publicHistoryIncidents(serviceStatusIncidents(mergePublicStatusIncidents(source.snapshot?.incidents ?? [], source.history.incidents), source.snapshot?.components ?? [], id), now.value) : []
  const incident = incidents.find(incidentActive) ?? null
  return {
    id, source, components, fresh, incident,
    otherIncidents: incidents.filter(item => incidentActive(item) && item !== incident),
    incidentComponents: components.filter(component => incident?.components.some(part => component.id && part.id === component.id || !part.id && part.name === component.name)),
    phase: incident ? incidentPhase(incident) : null,
    affected: components.filter(component => component.status !== 'operational'),
    tone: serviceStatusTone(components, incidents, fresh),
    historyReady, historyIncidents,
    historyDays: publicHistoryDays(historyIncidents, now.value),
  }
}))
const phaseName = (status: string) => t(phases.includes(status as typeof phases[number]) ? `needtoken.statusStrip.compact.phases.${status}` : 'needtoken.statusStrip.component.unknown')
const sourceHost = (href: string) => { try { return new URL(href).host } catch { return href } }
const renderTime = (value: string) => new Date(value).toLocaleString(locale.value)
const renderDate = (value: string) => new Date(value).toLocaleDateString(locale.value, { month: 'short', day: 'numeric' })
const checkedAge = (value: string) => {
  const minutes = Math.max(0, Math.floor((now.value - Date.parse(value)) / 60_000))
  return t(minutes ? 'needtoken.statusStrip.compact.checkedMinutesAgo' : 'needtoken.statusStrip.compact.justChecked', { count: minutes })
}
const incidentTiming = (incident: PublicStatusIncident) => {
  const minutes = incidentElapsedMinutes(incident, now.value)
  if (minutes === null) return t('needtoken.statusStrip.compact.timeUnknown')
  const duration = minutes >= 1440 ? t('needtoken.statusStrip.compact.durationDays', { count: Math.floor(minutes / 1440) })
    : minutes >= 60 ? t('needtoken.statusStrip.compact.durationHours', { count: Math.floor(minutes / 60), minutes: minutes % 60 })
    : t('needtoken.statusStrip.compact.durationMinutes', { count: minutes })
  return t(incidentActive(incident) ? 'needtoken.statusStrip.compact.ongoingFor' : 'needtoken.statusStrip.compact.lastedFor', { duration })
}
function close(restoreFocus = false) {
  const current = opened.value
  if (restoreFocus && current) rootEl.value?.querySelector<HTMLButtonElement>(`[data-provider="${current}"]`)?.focus()
  opened.value = null
  pinned.value = false
}
function hoverOpen(id: Provider) {
  if (!finePointer.value) return
  if (opened.value !== id) pinned.value = false
  opened.value = id
}
function hoverClose(id: Provider) {
  if (finePointer.value && !pinned.value && opened.value === id && !rootEl.value?.querySelector(`#nt-status-${id}`)?.contains(document.activeElement)) close()
}
function focusOpen(id: Provider) {
  if (opened.value !== id) pinned.value = false
  opened.value = id
}
function toggle(id: Provider) {
  if (opened.value === id && pinned.value) { close(); return }
  opened.value = id
  pinned.value = true
}
function onFocusOut(event: FocusEvent, id: Provider) {
  if (event.relatedTarget instanceof Node && (event.currentTarget as HTMLElement).contains(event.relatedTarget)) return
  if (opened.value === id) close()
}
function onOutsidePointer(event: PointerEvent) {
  if (event.target instanceof Node && !rootEl.value?.contains(event.target)) close()
}
function onPointerChange() { finePointer.value = pointerQuery?.matches ?? false }
function scheduleExpiry() {
  window.clearTimeout(expiryTimer)
  const expiries = sources.value.flatMap(source => {
    const expiry = Date.parse(source.fetchedAt || '') + PUBLIC_STATUS_MAX_AGE_MS
    return Number.isFinite(expiry) && expiry > Date.now() ? [expiry] : []
  })
  if (!expiries.length || disposed || !pageVisible.value) return
  expiryTimer = window.setTimeout(() => { now.value = Date.now(); scheduleExpiry() }, Math.max(0, Math.min(...expiries) - Date.now()))
}
async function loadSource(source: StatusSource, signal: AbortSignal, run: number) {
  try {
    const response = await fetch(`/api/v1/public/tools/service-status/${source.id}`, { signal, headers: { accept: 'application/json' }, credentials: 'omit' })
    if (!response.ok) throw new Error('status unavailable')
    const body = await response.json()
    const data = body?.status_code === 0 ? body.data : null
    if (!data || data.provider !== source.id || typeof data.fetched_at !== 'string' || !Number.isFinite(Date.parse(data.fetched_at)) || typeof data.stale !== 'boolean') throw new Error('invalid status response')
    const snapshot = parsePublicStatus(data.summary, source.href)
    if (!snapshot.ok) throw new Error('invalid status payload')
    if (disposed || run !== currentRun || !pageVisible.value) return
    source.snapshot = snapshot
    source.history = parsePublicStatusHistory(data.history, source.href)
    source.fetchedAt = data.fetched_at
    source.stale = data.stale
    source.state = 'ready'
    source.error = null
    source.sequence++
  } catch {
    if (disposed || run !== currentRun || !pageVisible.value) return
    source.state = 'unavailable'
    source.stale = true
    source.error = signal.aborted ? 'timeout' : 'unavailable'
    source.sequence++
  } finally {
    if (run === currentRun) source.refreshing = false
  }
}
async function refresh() {
  if (disposed || !pageVisible.value || inFlight) return
  window.clearTimeout(refreshTimer)
  inFlight = true
  const run = ++currentRun
  controller = new AbortController()
  const activeController = controller
  sources.value.forEach(source => { source.refreshing = true })
  timeout = window.setTimeout(() => activeController.abort(), 8000)
  try { await Promise.all(sources.value.map(source => loadSource(source, activeController.signal, run))) }
  finally {
    if (run !== currentRun) return
    window.clearTimeout(timeout)
    inFlight = false
    controller = undefined
    now.value = Date.now()
    scheduleExpiry()
    if (!disposed && pageVisible.value) refreshTimer = window.setTimeout(refresh, 60_000)
  }
}
function cancelRefresh() {
  currentRun++
  controller?.abort()
  controller = undefined
  inFlight = false
  sources.value.forEach(source => { source.refreshing = false })
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
  onVisibilityChange()
})
onUnmounted(() => {
  disposed = true
  cancelRefresh()
  pointerQuery?.removeEventListener('change', onPointerChange)
  document.removeEventListener('pointerdown', onOutsidePointer)
  document.removeEventListener('visibilitychange', onVisibilityChange)
})
</script>


<style scoped>
.status-pills { position: relative; display: flex; flex-wrap: wrap; align-items: center; justify-content: flex-end; gap: 8px; }
.status-pill-wrap { position: static; }
.status-pill { display: inline-flex; align-items: center; gap: 8px; min-height: 32px; padding: 5px 11px; border: 1px solid var(--ui-border); border-radius: 999px; background: var(--nt-surface-subtle); color: var(--ui-text-secondary); font-size: 11px; font-weight: 500; line-height: 1.4; transition: color 160ms ease, background 160ms ease, border-color 160ms ease; }
.status-pill:hover, .status-pill[aria-expanded="true"] { color: var(--ui-text-primary); border-color: var(--ui-border-strong); background: var(--ui-bg-soft); }
.status-pill:focus-visible, .status-popover a:focus-visible, .status-popover button:focus-visible, .status-popover summary:focus-visible { outline: 2px solid var(--ui-accent); outline-offset: 3px; }
.status-bars { display: inline-flex; align-items: center; gap: 2px; height: 13px; color: var(--nt-status-gray, #77848c); }
.status-bars i { display: block; width: 2px; height: 12px; border-radius: 1px; background: currentColor; transform-origin: center bottom; }
.status-bars.is-received i { animation: status-received 800ms var(--nt-motion-ease, ease-out) calc(var(--bar-index) * 60ms) 1; }
.status-bars.is-checking i { animation: status-checking 700ms var(--nt-motion-ease, ease-in-out) calc(var(--bar-index) * -100ms) infinite; }
@keyframes status-checking { 0%, 100% { transform: scaleY(.45); opacity: .5; } 50% { transform: scaleY(1); opacity: 1; } }
@keyframes status-received { 0% { transform: scaleY(.4); opacity: .4; } 100% { transform: scaleY(1); opacity: 1; } }
.status-popover { position: absolute; right: 0; bottom: calc(100% + 8px); z-index: 50; width: min(424px, calc(100vw - 72px)); max-height: min(620px, calc(100dvh - 80px)); overflow-y: auto; padding: 20px; border: 1px solid var(--ui-border-strong); border-radius: 20px; background: var(--ui-bg-elevated); box-shadow: var(--nt-popover-shadow); color: var(--ui-text-secondary); text-align: left; }
.status-popover-enter-active { transition: opacity var(--nt-motion-enter) var(--nt-motion-ease), transform var(--nt-motion-enter) var(--nt-motion-ease); }
.status-popover-leave-active { transition: opacity var(--nt-motion-fast) var(--nt-motion-ease); pointer-events: none; }
.status-popover-enter-from, .status-popover-leave-to { opacity: 0; }
.status-popover-enter-from { transform: translateY(4px); }
.status-pill-wrap::before { content: ''; position: absolute; bottom: 100%; right: 0; width: 100%; height: 12px; pointer-events: none; }
.status-pill-wrap:has(.status-popover)::before { pointer-events: auto; }
.status-card-head { display: flex; align-items: center; gap: 8px; min-width: 0; }
.status-card-head h2 { min-width: 0; color: var(--ui-text-primary); font-size: 15px; line-height: 20px; font-weight: 600; }
.status-health { margin-left: auto; font-size: 12px; line-height: 20px; }
.status-scope-row { display: flex; align-items: baseline; justify-content: space-between; gap: 12px; margin-top: 4px; color: var(--ui-text-muted); font-size: 11px; line-height: 18px; }
.status-count { white-space: nowrap; font-variant-numeric: tabular-nums; }
.status-warning { margin-top: 12px; color: var(--nt-warning-ink); font-size: 12px; line-height: 20px; }
.status-incident { margin-top: 16px; padding-top: 16px; border-top: 1px solid var(--ui-border); }
.status-incident.is-old { opacity: .65; }
.status-incident-meta { display: flex; align-items: center; justify-content: space-between; gap: 12px; color: var(--ui-text-muted); font-size: 11px; line-height: 16px; }
.status-incident-meta time { white-space: nowrap; }
.status-incident-title { margin-top: 8px; color: var(--ui-text-primary); font-size: 16px; line-height: 24px; font-weight: 500; overflow-wrap: anywhere; }
.status-incident-update { white-space: pre-line; margin-top: 8px; font-size: 12px; line-height: 20px; overflow-wrap: anywhere; }
.status-affected { margin-top: 10px; font-size: 11px; line-height: 18px; color: var(--ui-text-muted); overflow-wrap: anywhere; }
.status-update-history { margin-top: 14px; font-size: 12px; line-height: 20px; }
.status-update-history summary { width: fit-content; cursor: pointer; color: var(--ui-accent); }
.status-update-history ol, .status-update-history ul { display: grid; gap: 14px; margin-top: 12px; padding-left: 12px; border-left: 1px solid var(--ui-border); }
.status-update-history li > div { display: flex; flex-wrap: wrap; justify-content: space-between; gap: 4px 12px; color: var(--ui-text-muted); font-size: 11px; }
.status-update-history li p { margin-top: 5px; white-space: pre-line; overflow-wrap: anywhere; }
.status-update-history li > a { display: inline-flex; align-items: baseline; gap: 4px; color: var(--ui-text-primary); }
.status-phases { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 8px; margin-top: 16px; }
.status-phase { min-width: 0; color: var(--ui-text-muted); font-size: 11px; line-height: 16px; }
.status-phase > span:last-child { display: block; overflow-wrap: anywhere; }
.status-phase-track { display: block; height: 4px; margin-bottom: 8px; overflow: hidden; border-radius: 2px; background: var(--ui-border-strong); }
.status-phase-track i { display: block; width: 100%; height: 100%; transform: scaleX(0); transform-origin: left center; background: var(--nt-status-yellow, #c5a65e); }
.status-phase.is-reached .status-phase-track i { transform: scaleX(1); animation: status-phase-reveal 400ms var(--nt-motion-ease, ease-out) calc(var(--phase-index) * 60ms) 1 both; }
.status-phase.is-current { color: var(--ui-text-primary); }
.status-incident[data-phase="resolved"] .status-phase-track i { background: var(--nt-status-green, #3bcba0); }
@keyframes status-phase-reveal { from { transform: scaleX(0); } to { transform: scaleX(1); } }
.status-incident-footer { display: flex; align-items: baseline; justify-content: space-between; gap: 12px; margin-top: 12px; color: var(--ui-text-muted); font-size: 11px; line-height: 18px; }
.status-popover a { display: inline-flex; align-items: center; gap: 4px; }
.status-incident-footer a { flex-shrink: 0; }
.status-popover a:hover { color: var(--ui-accent); }
.status-history-region { margin-top: 16px; padding-top: 16px; border-top: 1px solid var(--ui-border); }
.status-history-head { display: flex; justify-content: space-between; align-items: baseline; gap: 12px; font-size: 11px; line-height: 18px; }
.status-history-head h3 { color: var(--ui-text-secondary); font-weight: 400; }
.status-history-head > span { color: var(--ui-text-muted); font-variant-numeric: tabular-nums; white-space: nowrap; }
.status-history-axis { display: flex; justify-content: space-between; gap: 8px; margin-top: 8px; color: var(--ui-text-muted); font-size: 10px; line-height: 16px; font-variant-numeric: tabular-nums; }
.status-history-note { margin-top: 8px; color: var(--ui-text-muted); font-size: 11px; line-height: 18px; }
.status-history-events { margin-top: 12px; color: var(--ui-text-muted); font-size: 11px; line-height: 18px; }
.status-history-events summary { cursor: pointer; width: fit-content; }
.status-history-events ul { display: grid; gap: 8px; margin-top: 12px; }
.status-history-events li { display: flex; align-items: baseline; gap: 8px; }
.status-history-events li time { flex-shrink: 0; }
.status-history-events li a { color: var(--ui-text-secondary); overflow-wrap: anywhere; }
.status-history-events li a svg { flex-shrink: 0; }
.status-refresh-row { display: flex; flex-wrap: wrap; align-items: center; gap: 12px; margin-top: 16px; padding-top: 12px; border-top: 1px solid var(--ui-border); color: var(--ui-text-muted); font-size: 11px; line-height: 18px; }
.status-refresh-row > a { margin-right: auto; }
.status-refresh-row > time { white-space: nowrap; }
.status-refresh-row button { color: var(--ui-accent); padding: 4px 0; white-space: nowrap; }
.status-refresh-row button[aria-disabled="true"] { opacity: .5; }
.status-dot { width: 6px; height: 6px; flex: none; border-radius: 50%; background: var(--nt-status-gray, #77848c); }
.status-dot.is-operational { background: var(--nt-status-green, #3bcba0); }
.status-bars.is-operational { color: var(--nt-status-green, #3bcba0); }
.status-dot.is-degraded_performance, .status-dot.is-under_maintenance { background: var(--nt-status-yellow, #c5a65e); }
.status-bars.is-degraded_performance, .status-bars.is-under_maintenance { color: var(--nt-status-yellow, #c5a65e); }
.status-dot.is-partial_outage, .status-dot.is-major_outage { background: var(--nt-status-red, #e06a5c); }
.status-bars.is-partial_outage, .status-bars.is-major_outage { color: var(--nt-status-red, #e06a5c); }
@media (max-width: 640px) { .status-pills { justify-content: flex-start; width: 100%; }.status-pill { min-height: 36px; }.status-popover { left: 0; right: auto; padding: 16px; border-radius: 16px; }.status-refresh-row { gap: 8px; } }
@media (prefers-reduced-motion: reduce) { .status-bars:is(.is-checking, .is-received) i, .status-phase.is-reached .status-phase-track i { animation: none; }.status-pill { transition: none; }.status-popover-enter-from { transform: none; } }
</style>
