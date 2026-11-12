<template>
  <div class="status-pills" :aria-label="t('needtoken.statusStrip.label')">
    <div class="status-pill-wrap" @mouseenter="hoverOpen('gpt')" @mouseleave="hoverClose('gpt')" @focusin="opened = 'gpt'" @focusout="onFocusOut($event, 'gpt')">
      <button type="button" class="status-pill" :aria-expanded="opened === 'gpt'" aria-controls="nt-status-gpt" @click="toggle('gpt')">
        <span class="status-bars" aria-hidden="true"><i v-for="(tone, index) in barTones(openaiComponents)" :key="index" :class="`is-${tone}`"></i></span>
        {{ t('needtoken.statusStrip.gpt') }}
      </button>
      <div v-if="opened === 'gpt'" id="nt-status-gpt" class="status-popover is-wide" role="region" :aria-label="t('needtoken.statusStrip.gpt')">
        <article v-for="card in cards" :key="card.id" class="status-card">
          <header class="status-card-head">
            <span class="status-dot" :class="dotClass(card)" aria-hidden="true"></span>
            <h2>{{ t(`needtoken.statusStrip.sources.${card.id}`) }}</h2>
            <a :href="card.href" target="_blank" rel="noopener noreferrer">{{ sourceHost(card.href) }}<ArrowUpRight :size="12" aria-hidden="true" /></a>
          </header>
          <p v-if="card.state === 'loading'" role="status">{{ t('needtoken.statusStrip.loading') }}</p>
          <p v-else-if="card.state !== 'ready' || !card.snapshot" role="status">{{ t('needtoken.statusStrip.unavailable') }}</p>
          <template v-else>
            <p class="status-description">{{ card.snapshot.description }}</p>
            <p v-if="card.snapshot.components.length" class="status-count">{{ t('needtoken.statusStrip.currentCount', operationalCount(card.snapshot.components)) }}</p>
            <ul v-if="notable(card.snapshot.components).length" class="status-components">
              <li v-for="component in notable(card.snapshot.components)" :key="component.name">
                <span class="status-dot" :class="`is-${component.status}`" aria-hidden="true"></span>
                <span>{{ component.name }}</span>
                <span>{{ t(`needtoken.statusStrip.component.${component.status}`) }}</span>
              </li>
            </ul>
            <p class="status-history">{{ t('needtoken.statusStrip.noHistory') }}</p>
            <p v-if="card.snapshot.updatedAt" class="status-time">{{ freshness(card.snapshot.updatedAt) }}</p>
          </template>
        </article>
      </div>
    </div>
    <div class="status-pill-wrap" @mouseenter="hoverOpen('codex')" @mouseleave="hoverClose('codex')" @focusin="opened = 'codex'" @focusout="onFocusOut($event, 'codex')">
      <button type="button" class="status-pill" :aria-expanded="opened === 'codex'" aria-controls="nt-status-codex" @click="toggle('codex')">
        <span class="status-bars" aria-hidden="true"><i v-for="(tone, index) in barTones(codexComponents)" :key="index" :class="`is-${tone}`"></i></span>
        {{ t('needtoken.statusStrip.codex') }}
      </button>
      <div v-if="opened === 'codex'" id="nt-status-codex" class="status-popover" role="region" :aria-label="t('needtoken.statusStrip.codex')">
        <article class="status-card">
          <header class="status-card-head">
            <span class="status-dot" :class="codexDot" aria-hidden="true"></span>
            <h2>{{ t('needtoken.statusStrip.sources.codex') }}</h2>
            <a href="https://status.openai.com/" target="_blank" rel="noopener noreferrer">status.openai.com<ArrowUpRight :size="12" aria-hidden="true" /></a>
          </header>
          <p v-if="openaiCard?.state === 'loading'" role="status">{{ t('needtoken.statusStrip.loading') }}</p>
          <p v-else-if="openaiCard?.state !== 'ready'" role="status">{{ t('needtoken.statusStrip.unavailable') }}</p>
          <template v-else>
            <p class="status-description">{{ t('needtoken.statusStrip.codexIntro') }}</p>
            <p v-if="codexComponents.length" class="status-count">{{ t('needtoken.statusStrip.currentCount', operationalCount(codexComponents)) }}</p>
            <p v-else role="status">{{ t('needtoken.statusStrip.codexEmpty') }}</p>
            <ul v-if="codexNotable.length" class="status-components">
              <li v-for="component in codexNotable" :key="component.name">
                <span class="status-dot" :class="`is-${component.status}`" aria-hidden="true"></span>
                <span>{{ component.name }}</span>
                <span>{{ t(`needtoken.statusStrip.component.${component.status}`) }}</span>
              </li>
            </ul>
            <p class="status-history">{{ t('needtoken.statusStrip.noHistory') }}</p>
            <p v-if="openaiCard.snapshot?.updatedAt" class="status-time">{{ freshness(openaiCard.snapshot.updatedAt) }}</p>
          </template>
        </article>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { ArrowUpRight } from 'lucide-vue-next'
import { operationalCount, parsePublicStatus, type PublicStatusComponent, type PublicStatusSnapshot } from '../lib/publicStatus'

interface StatusCard {
  id: 'openai' | 'claude'
  href: string
  endpoint: string
  state: 'loading' | 'ready' | 'unavailable'
  snapshot: PublicStatusSnapshot | null
}

const { t, locale } = useI18n()
const cards = ref<StatusCard[]>([
  { id: 'openai', href: 'https://status.openai.com/', endpoint: 'https://status.openai.com/api/v2/summary.json', state: 'loading', snapshot: null },
  { id: 'claude', href: 'https://status.claude.com/', endpoint: 'https://status.claude.com/api/v2/summary.json', state: 'loading', snapshot: null },
])
const opened = ref<'gpt' | 'codex' | null>(null)
const pinned = ref(false)
const finePointer = ref(false)
let controller: AbortController | undefined
let pointerQuery: MediaQueryList | undefined

const openaiCard = computed(() => cards.value.find(card => card.id === 'openai') || null)
const openaiComponents = computed(() => openaiCard.value?.snapshot?.components ?? [])
const codexComponents = computed(() => openaiComponents.value.filter(component => component.name.toLowerCase().includes('codex')))
const codexNotable = computed(() => notable(codexComponents.value))
const codexDot = computed(() => {
  if (openaiCard.value?.state !== 'ready') return 'is-unknown'
  if (!codexComponents.value.length) return 'is-unknown'
  if (codexComponents.value.some(component => component.status === 'major_outage' || component.status === 'partial_outage')) return 'is-major_outage'
  if (codexComponents.value.some(component => component.status !== 'operational')) return 'is-degraded_performance'
  return 'is-operational'
})

const sourceHost = (href: string) => {
  try { return new URL(href).host } catch { return href }
}
const notable = (components: PublicStatusComponent[]) => {
  const degraded = components.filter(component => component.status !== 'operational')
  return (degraded.length ? degraded : components).slice(0, 4)
}
const barTones = (components: PublicStatusComponent[]) => {
  const sample = components.slice(0, 5).map(component => component.status)
  if (!sample.length) return ['unknown', 'unknown', 'unknown', 'unknown', 'unknown']
  while (sample.length < 5) sample.push(sample[sample.length - 1]!)
  return sample
}
const freshness = (value: string) => {
  const time = new Date(value)
  const rendered = time.toLocaleString(locale.value)
  const stale = Date.now() - time.getTime() > 24 * 60 * 60 * 1000
  return t(stale ? 'needtoken.statusStrip.stale' : 'needtoken.statusStrip.updated', { time: rendered })
}
const dotClass = (card: StatusCard) => {
  if (card.state !== 'ready' || !card.snapshot) return 'is-unknown'
  return card.snapshot.indicator === 'none' ? 'is-operational' : 'is-degraded_performance'
}

function hoverOpen(id: 'gpt' | 'codex') {
  if (finePointer.value && !pinned.value) opened.value = id
}
function hoverClose(id: 'gpt' | 'codex') {
  if (finePointer.value && !pinned.value && opened.value === id) opened.value = null
}
function toggle(id: 'gpt' | 'codex') {
  if (opened.value === id && pinned.value) {
    opened.value = null
    pinned.value = false
    return
  }
  opened.value = id
  pinned.value = true
}
function onFocusOut(event: FocusEvent, id: 'gpt' | 'codex') {
  const next = event.relatedTarget
  if (next instanceof Node && (event.currentTarget as HTMLElement).contains(next)) return
  if (opened.value === id) opened.value = null
}

async function loadCard(card: StatusCard) {
  try {
    const response = await fetch(card.endpoint, { signal: controller?.signal, headers: { accept: 'application/json' } })
    if (!response.ok) throw new Error(`status ${response.status}`)
    const view = parsePublicStatus(await response.json(), card.href)
    if (!view.ok) throw new Error('invalid status payload')
    card.snapshot = view
    card.href = view.url
    card.state = 'ready'
  } catch {
    if (controller?.signal.aborted) return
    card.state = 'unavailable'
    card.snapshot = null
  }
}

onMounted(() => {
  pointerQuery = window.matchMedia('(hover: hover) and (pointer: fine)')
  finePointer.value = pointerQuery.matches
  pointerQuery.addEventListener('change', () => { finePointer.value = pointerQuery?.matches ?? false })
  controller = new AbortController()
  const timer = window.setTimeout(() => controller?.abort(), 8000)
  void Promise.all(cards.value.map(loadCard)).finally(() => window.clearTimeout(timer))
})
onUnmounted(() => controller?.abort())
</script>

<style scoped>
.status-pills { display: flex; flex-wrap: wrap; align-items: center; justify-content: flex-end; gap: 8px; }
.status-pill-wrap { position: relative; }
.status-pill { display: inline-flex; align-items: center; gap: 8px; min-height: 28px; padding: 4px 12px 4px 10px; border: 1px solid rgba(255,255,255,.14); border-radius: 999px; background: rgba(255,255,255,.06); color: var(--ui-text-primary); font-size: 12px; font-weight: 650; line-height: 1; }
.status-pill:hover, .status-pill[aria-expanded="true"] { background: rgba(255,255,255,.1); }
.status-bars { display: inline-flex; align-items: flex-end; gap: 2px; height: 14px; }
.status-bars i { display: block; width: 3px; border-radius: 2px; background: #8b97a3; }
.status-bars i:nth-child(1) { height: 8px; }
.status-bars i:nth-child(2) { height: 13px; }
.status-bars i:nth-child(3) { height: 10px; }
.status-bars i:nth-child(4) { height: 14px; }
.status-bars i:nth-child(5) { height: 11px; }
.status-bars i.is-operational { background: #3dd68c; }
.status-bars i.is-degraded_performance, .status-bars i.is-under_maintenance { background: #e2b340; }
.status-bars i.is-partial_outage, .status-bars i.is-major_outage { background: #e06a5c; }
.status-popover { position: absolute; right: 0; bottom: calc(100% + 8px); z-index: 30; display: grid; gap: 10px; width: min(360px, 86vw); padding: 0; }
.status-popover::before { content: ''; position: absolute; left: 0; right: 0; bottom: -12px; height: 12px; }
.status-popover.is-wide { width: min(720px, 86vw); grid-template-columns: repeat(2, minmax(0, 1fr)); }
.status-card { min-width: 0; padding: 14px; border: 1px solid rgba(160, 190, 185, .2); border-radius: 16px; background: #141a21; box-shadow: 0 16px 40px rgba(0,0,0,.35); }
.status-card-head { display: flex; align-items: center; gap: 8px; min-width: 0; }
.status-card-head h2 { min-width: 0; color: var(--ui-text-primary); font-size: 14px; font-weight: 700; }
.status-card-head a { margin-left: auto; display: inline-flex; align-items: center; gap: 4px; color: var(--ui-text-muted); font-size: 11px; }
.status-card p { margin-top: 8px; color: var(--ui-text-secondary); font-size: 12px; line-height: 1.6; }
.status-count, .status-history, .status-time { color: var(--ui-text-muted); }
.status-components { display: grid; gap: 6px; margin-top: 10px; }
.status-components li { display: grid; grid-template-columns: 8px minmax(0, 1fr) auto; align-items: center; gap: 8px; color: var(--ui-text-secondary); font-size: 12px; }
.status-components li span:nth-child(2) { min-width: 0; overflow-wrap: anywhere; }
.status-dot { width: 8px; height: 8px; flex: none; border-radius: 50%; background: #8b97a3; }
.status-dot.is-operational { background: #3dd68c; }
.status-dot.is-degraded_performance, .status-dot.is-under_maintenance { background: #e2b340; }
.status-dot.is-partial_outage, .status-dot.is-major_outage { background: #e06a5c; }
@media (max-width: 720px) { .status-popover, .status-popover.is-wide { grid-template-columns: 1fr; width: min(320px, 86vw); } }
</style>
