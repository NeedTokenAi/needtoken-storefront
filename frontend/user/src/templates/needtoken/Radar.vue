<template>
  <div class="tool-board">
    <header>
      <p>{{ t('needtoken.radar.eyebrow') }}</p>
      <h1>{{ t('needtoken.radar.title') }}</h1>
      <p>{{ t('needtoken.radar.intro') }}</p>
    </header>

    <section class="tool-panel" aria-live="polite">
      <div class="tool-panel-head">
        <h2>{{ t('needtoken.radar.board') }}</h2>
        <a href="https://status.openai.com/" target="_blank" rel="noopener noreferrer">status.openai.com<ArrowUpRight :size="13" aria-hidden="true" /></a>
      </div>
      <p v-if="state === 'loading'" role="status">{{ t('needtoken.statusStrip.loading') }}</p>
      <p v-else-if="state === 'unavailable'" role="status">{{ t('needtoken.radar.unavailable') }}</p>
      <template v-else>
        <p v-if="updatedAt" class="tool-meta">{{ t('needtoken.statusStrip.updated', { time: formatTime(updatedAt) }) }}</p>
        <p v-if="incidentsUnavailable" role="status">{{ t('needtoken.radar.incidentsUnavailable') }}</p>
        <table v-if="rows.length">
          <caption class="sr-only">{{ t('needtoken.radar.board') }}</caption>
          <thead><tr><th>{{ t('needtoken.radar.item') }}</th><th>{{ t('needtoken.radar.kind') }}</th><th>{{ t('needtoken.radar.state') }}</th></tr></thead>
          <tbody>
            <tr v-for="row in rows" :key="row.key">
              <td>{{ row.name }}</td>
              <td>{{ t(`needtoken.radar.kinds.${row.kind}`) }}</td>
              <td>{{ row.state }}</td>
            </tr>
          </tbody>
        </table>
        <p v-else role="status">{{ t('needtoken.radar.empty') }}</p>
        <p class="tool-note">{{ t('needtoken.radar.note') }}</p>
      </template>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { ArrowUpRight } from 'lucide-vue-next'
import { usePageSeo } from '../../composables/usePageSeo'
import { matchesTerms, parsePublicIncidents, parsePublicStatus, type PublicStatusIncident, type PublicStatusComponent } from './lib/publicStatus'

const { t, locale } = useI18n()
usePageSeo({ title: () => t('needtoken.radar.title'), description: () => t('needtoken.radar.intro'), canonicalPath: () => '/tools/radar' })

const state = ref<'loading' | 'ready' | 'unavailable'>('loading')
const updatedAt = ref<string | null>(null)
const components = ref<PublicStatusComponent[]>([])
const incidents = ref<PublicStatusIncident[]>([])
const incidentsUnavailable = ref(false)
let controller: AbortController | undefined

const formatTime = (value: string) => new Date(value).toLocaleString(locale.value)
const rows = computed(() => [
  ...components.value.filter(row => matchesTerms(row.name, ['codex'])).map(row => ({
    key: `component-${row.name}`,
    name: row.name,
    kind: 'component' as const,
    state: t(`needtoken.statusStrip.component.${row.status}`),
  })),
  ...incidents.value.filter(row => {
    const text = `${row.name} ${row.impact}`
    return matchesTerms(text, ['codex']) || (matchesTerms(text, ['reset']) && matchesTerms(text, ['quota']))
  }).map(row => ({
    key: `incident-${row.name}`,
    name: row.name,
    kind: 'incident' as const,
    state: row.status,
  })),
])

async function load() {
  controller = new AbortController()
  const timer = window.setTimeout(() => controller?.abort(), 8000)
  try {
    const [summaryResponse, incidentResponse] = await Promise.all([
      fetch('https://status.openai.com/api/v2/summary.json', { signal: controller.signal, headers: { accept: 'application/json' } }),
      fetch('https://status.openai.com/api/v2/incidents.json', { signal: controller.signal, headers: { accept: 'application/json' } }).catch(() => null),
    ])
    if (!summaryResponse.ok) throw new Error('status source failed')
    const summary = parsePublicStatus(await summaryResponse.json(), 'https://status.openai.com/')
    if (!summary.ok) throw new Error('invalid status payload')
    components.value = summary.components
    updatedAt.value = summary.updatedAt
    const incidentPayload = incidentResponse?.ok ? await incidentResponse.json() : null
    const incidentRows = incidentPayload ? parsePublicIncidents(incidentPayload) : null
    incidents.value = incidentRows ?? []
    incidentsUnavailable.value = incidentRows === null
    state.value = 'ready'
  } catch {
    if (!controller.signal.aborted || state.value === 'loading') state.value = 'unavailable'
  } finally {
    window.clearTimeout(timer)
  }
}

onMounted(() => { void load() })
onUnmounted(() => controller?.abort())
</script>

<style scoped>
.tool-board { width: min(920px, calc(100% - 32px)); margin: 28px auto 12px; }
.tool-board header p:first-child { color: var(--ui-accent); font-size: 12px; font-weight: 650; letter-spacing: .04em; }
.tool-board h1 { margin-top: 8px; font-size: clamp(28px, 4vw, 40px); font-weight: 750; letter-spacing: -.04em; }
.tool-board header p:last-child { max-width: 68ch; margin-top: 12px; color: var(--ui-text-secondary); font-size: 14px; line-height: 1.8; }
.tool-panel { margin-top: 22px; padding: 18px; border: 1px solid var(--ui-border); border-radius: 18px; background: rgba(255,255,255,.03); animation: tool-in .28s ease; }
.tool-panel-head { display: flex; align-items: center; justify-content: space-between; gap: 12px; }
.tool-panel-head h2 { font-size: 16px; font-weight: 700; }
.tool-panel-head a { display: inline-flex; align-items: center; gap: 4px; color: var(--ui-accent); font-size: 12px; }
.tool-panel > p, .tool-note, .tool-meta { margin-top: 12px; color: var(--ui-text-muted); font-size: 13px; line-height: 1.7; }
.tool-panel table { width: 100%; margin-top: 14px; border-collapse: collapse; font-size: 13px; }
.tool-panel th, .tool-panel td { padding: 10px 8px; border-bottom: 1px solid var(--ui-border); text-align: left; vertical-align: top; }
.tool-panel th { color: var(--ui-text-muted); font-size: 11px; font-weight: 650; }
.sr-only { position: absolute; width: 1px; height: 1px; padding: 0; margin: -1px; overflow: hidden; clip: rect(0,0,0,0); white-space: nowrap; border: 0; }
@keyframes tool-in { from { opacity: 0; transform: translateY(8px); } to { opacity: 1; transform: none; } }
@media (prefers-reduced-motion: reduce) { .tool-panel { animation: none; } }
@media (max-width: 640px) { .tool-board { width: min(100% - 28px, 920px); margin-top: 18px; } .tool-panel { padding: 14px; } .tool-panel table { display: block; overflow-x: auto; } }
</style>
