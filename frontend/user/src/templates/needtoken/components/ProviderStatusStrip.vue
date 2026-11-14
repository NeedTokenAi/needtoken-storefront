<template>
  <div ref="rootEl" class="status-pills" :aria-label="t('needtoken.statusStrip.label')" @keydown.esc.stop.prevent="close(true)">
    <div v-for="provider in providers" :key="provider.id" class="status-pill-wrap" @mouseenter="hoverOpen(provider.id)" @mouseleave="hoverClose(provider.id)" @focusin="focusOpen(provider.id)" @focusout="onFocusOut($event, provider.id)">
      <button type="button" class="status-pill" :data-provider="provider.id" :aria-expanded="opened === provider.id" :aria-controls="`nt-status-${provider.id}`" :aria-label="`${t(`needtoken.statusStrip.${provider.id}`)} · ${t(`needtoken.statusStrip.component.${provider.tone}`)}`" @click="toggle(provider.id)">
        <span :key="`${provider.id}-${provider.source.sequence}`" class="status-bars" :class="[`is-${provider.tone}`, { 'is-checking': provider.source.refreshing && pageVisible, 'is-received': provider.fresh && provider.source.sequence > 0 }]" aria-hidden="true"><i v-for="index in 5" :key="index" :style="{ '--bar-index': index - 1 }"></i></span>
        {{ t(`needtoken.statusStrip.${provider.id}`) }}
      </button>
      <Transition name="status-popover" @before-enter="showEnteringPanel" @before-leave="hideLeavingPanel">
      <section v-if="opened === provider.id" :id="`nt-status-${provider.id}`" class="status-popover" :aria-label="t(`needtoken.statusStrip.${provider.id}`)">
        <header class="status-card-head">
          <span class="status-dot" :class="`is-${provider.tone}`" aria-hidden="true"></span>
          <h2>{{ t(`needtoken.statusStrip.scopeTitle.${provider.id}`) }}</h2>
          <a :href="provider.source.href" target="_blank" rel="noopener noreferrer">{{ sourceHost(provider.source.href) }}<ArrowUpRight :size="12" aria-hidden="true" /></a>
        </header>
        <p class="status-scope">{{ t(`needtoken.statusStrip.scope.${provider.id}`) }}</p>
        <p v-if="provider.source.state === 'loading'" role="status">{{ t('needtoken.statusStrip.loading') }}</p>
        <p v-else-if="provider.source.state === 'unavailable'" class="status-warning" role="status">{{ t(provider.source.error === 'timeout' ? 'needtoken.statusStrip.timeout' : 'needtoken.statusStrip.unavailable') }}</p>
        <p v-else-if="!provider.fresh" class="status-warning" role="status">{{ t('needtoken.statusStrip.outdated') }}</p>
        <template v-if="provider.source.snapshot">
          <p v-if="!provider.components.length" role="status">{{ t('needtoken.statusStrip.scopeEmpty') }}</p>
          <p v-else class="status-count">{{ t(provider.fresh ? 'needtoken.statusStrip.currentCount' : 'needtoken.statusStrip.previousCount', operationalCount(provider.components)) }}</p>
          <ul v-if="provider.components.length" class="status-components" :class="{ 'is-old': !provider.fresh }">
            <li v-for="component in provider.components" :key="component.id || component.name">
              <span class="status-dot" :class="`is-${provider.fresh ? component.status : 'unknown'}`" aria-hidden="true"></span>
              <span>{{ component.name }}</span>
              <span>{{ t(`needtoken.statusStrip.component.${component.status}`) }}</span>
            </li>
          </ul>
          <p class="status-history">{{ t('needtoken.statusStrip.noHistory') }}</p>
          <p v-if="provider.source.fetchedAt" class="status-time">{{ t('needtoken.statusStrip.checked', { time: renderTime(provider.source.fetchedAt) }) }}</p>
          <p v-if="provider.source.snapshot.updatedAt" class="status-time">{{ t('needtoken.statusStrip.sourceChanged', { time: renderTime(provider.source.snapshot.updatedAt) }) }}</p>
        </template>
        <div class="status-refresh-row"><span>{{ t(provider.source.refreshing ? 'needtoken.statusStrip.refreshing' : 'needtoken.statusStrip.refreshPolicy') }}</span><button type="button" :aria-disabled="provider.source.refreshing" :aria-busy="provider.source.refreshing" @click="refresh">{{ t('needtoken.statusStrip.refreshAction') }}</button></div>
      </section>
      </Transition>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { ArrowUpRight } from 'lucide-vue-next'
import { operationalCount, parsePublicStatus, publicStatusFresh, publicStatusTone, serviceStatusComponents, PUBLIC_STATUS_MAX_AGE_MS, type PublicStatusService, type PublicStatusSnapshot } from '../lib/publicStatus'

type Provider = PublicStatusService
interface StatusSource {
  id: 'openai' | 'claude'
  href: string
  state: 'loading' | 'ready' | 'unavailable'
  snapshot: PublicStatusSnapshot | null
  fetchedAt: string | null
  stale: boolean
  refreshing: boolean
  error: 'timeout' | 'unavailable' | null
  sequence: number
}
const { t, locale } = useI18n()
const sources = ref<StatusSource[]>([
  { id: 'openai', href: 'https://status.openai.com/', state: 'loading', snapshot: null, fetchedAt: null, stale: true, refreshing: false, error: null, sequence: 0 },
  { id: 'claude', href: 'https://status.claude.com/', state: 'loading', snapshot: null, fetchedAt: null, stale: true, refreshing: false, error: null, sequence: 0 },
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
const providers = computed(() => sources.value.map(source => {
  const id: Provider = source.id === 'openai' ? 'gpt' : 'claude'
  const components = serviceStatusComponents(source.snapshot?.components ?? [], id)
  const fresh = source.state === 'ready' && !source.stale && publicStatusFresh(source.fetchedAt, now.value)
  return { id, source, components, fresh, tone: publicStatusTone(components, fresh) }
}))
const sourceHost = (href: string) => { try { return new URL(href).host } catch { return href } }
const renderTime = (value: string) => new Date(value).toLocaleString(locale.value)
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
.status-pill { display: inline-flex; align-items: center; gap: 8px; min-height: 32px; padding: 5px 11px; border: 1px solid rgba(168,188,185,.14); border-radius: 999px; background: rgba(168,188,185,.045); color: var(--ui-text-secondary); font-size: 11px; font-weight: 500; line-height: 1.4; transition: color 160ms ease, background 160ms ease; }
.status-pill:hover, .status-pill[aria-expanded="true"] { color: var(--ui-text-primary); background: rgba(168,188,185,.09); }
.status-pill:focus-visible { outline: 2px solid var(--ui-accent); outline-offset: 3px; }
.status-bars { display: inline-flex; align-items: center; gap: 2px; height: 13px; color: #77848c; }
.status-bars i { display: block; width: 2px; height: 12px; border-radius: 1px; background: currentColor; transform-origin: center bottom; }
.status-bars.is-received i { animation: status-received 800ms var(--nt-motion-ease, ease-out) calc(var(--bar-index) * 60ms) 1; }
.status-bars.is-checking i { animation: status-checking 700ms var(--nt-motion-ease, ease-in-out) calc(var(--bar-index) * -100ms) infinite; }
@keyframes status-checking { 0%, 100% { transform: scaleY(.45); opacity: .5; } 50% { transform: scaleY(1); opacity: 1; } }
@keyframes status-received { 0% { transform: scaleY(.4); opacity: .4; } 100% { transform: scaleY(1); opacity: 1; } }
.status-scope { color: var(--ui-text-muted); }
.status-popover .status-warning { color: #c7ad70; }
.status-refresh-row { display: flex; justify-content: space-between; align-items: center; gap: 12px; margin-top: 16px; padding-top: 12px; border-top: 1px solid var(--ui-border); color: var(--ui-text-muted); font-size: 11px; line-height: 1.6; }
.status-refresh-row button { color: var(--ui-accent); padding: 4px 0; }
.status-refresh-row button[aria-disabled="true"] { opacity: .5; }
.status-refresh-row button:focus-visible { outline: 2px solid var(--ui-accent); outline-offset: 3px; }
.status-components.is-old { opacity: .65; }
.status-popover { position: absolute; right: 0; bottom: calc(100% + 8px); z-index: 50; width: min(420px, calc(100vw - 72px)); max-height: min(460px, 70vh); overflow-y: auto; padding: 20px; border: 1px solid var(--ui-border-strong); border-radius: 16px; background: var(--ui-bg-elevated); box-shadow: 0 16px 40px rgba(0,0,0,.35); color: var(--ui-text-secondary); text-align: left; }
.status-popover-enter-active { transition: opacity var(--nt-motion-enter) var(--nt-motion-ease), transform var(--nt-motion-enter) var(--nt-motion-ease); }
.status-popover-leave-active { transition: opacity var(--nt-motion-fast) var(--nt-motion-ease); pointer-events: none; }
.status-popover-enter-from, .status-popover-leave-to { opacity: 0; }
.status-popover-enter-from { transform: translateY(4px); }
.status-pill-wrap::before { content: ''; position: absolute; bottom: 100%; right: 0; width: 100%; height: 12px; pointer-events: none; }
.status-pill-wrap:has(.status-popover)::before { pointer-events: auto; }
.status-card-head { display: flex; align-items: center; gap: 8px; min-width: 0; }
.status-card-head h2 { min-width: 0; color: var(--ui-text-primary); font-size: 14px; font-weight: 600; }
.status-card-head a { margin-left: auto; display: inline-flex; align-items: center; gap: 4px; color: var(--ui-text-muted); font-size: 10px; }
.status-card-head a:hover { color: var(--ui-accent); }
.status-popover p { margin-top: 10px; font-size: 12px; line-height: 1.7; }
.status-popover .status-count { color: var(--ui-text-secondary); }
.status-popover .status-history, .status-popover .status-time { color: var(--ui-text-muted); font-size: 11px; }
.status-components { display: grid; gap: 9px; margin-block: 15px; }
.status-components li { display: grid; grid-template-columns: 6px minmax(0, 1fr) auto; align-items: center; gap: 8px; font-size: 11px; }
.status-components li span:nth-child(2) { min-width: 0; overflow-wrap: anywhere; }
.status-components li span:last-child { color: var(--ui-text-muted); }
.status-dot { width: 6px; height: 6px; flex: none; border-radius: 50%; background: #77848c; }
.status-dot.is-operational { background: #3bcba0; }
.status-bars.is-operational { color: #3bcba0; }
.status-dot.is-degraded_performance, .status-dot.is-under_maintenance { background: #e2b340; }
.status-bars.is-degraded_performance, .status-bars.is-under_maintenance { color: #e2b340; }
.status-dot.is-partial_outage, .status-dot.is-major_outage { background: #e06a5c; }
.status-bars.is-partial_outage, .status-bars.is-major_outage { color: #e06a5c; }
@media (max-width: 640px) { .status-pills { justify-content: flex-start; width: 100%; }.status-pill { min-height: 36px; }.status-popover { left: 0; right: auto; padding: 16px; } }
@media (prefers-reduced-motion: reduce) { .status-bars:is(.is-checking, .is-received) i { animation: none; }.status-pill { transition: none; }.status-popover-enter-from { transform: none; } }
</style>
