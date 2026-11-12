<template>
  <section class="status-strip" :aria-label="t('needtoken.statusStrip.label')">
    <article v-for="card in cards" :key="card.id" class="status-card" :data-status-source="card.id">
      <header class="status-card-head">
        <span class="status-dot" :class="dotClass(card)" aria-hidden="true"></span>
        <h2>{{ t(`needtoken.statusStrip.sources.${card.id}`) }}</h2>
        <a :href="card.href" target="_blank" rel="noopener noreferrer">{{ sourceHost(card.href) }}<ArrowUpRight :size="12" aria-hidden="true" /></a>
      </header>
      <p v-if="card.state === 'loading'" role="status">{{ t('needtoken.statusStrip.loading') }}</p>
      <p v-else-if="card.state === 'unavailable'" role="status">{{ t('needtoken.statusStrip.unavailable') }}</p>
      <template v-else-if="card.snapshot">
        <p class="status-description">{{ card.snapshot.description }}</p>
        <p v-if="card.snapshot.components.length" class="status-count">{{ t('needtoken.statusStrip.currentCount', operationalCount(card.snapshot.components)) }}</p>
        <ul v-if="notable(card.snapshot).length" class="status-components">
          <li v-for="component in notable(card.snapshot)" :key="component.name">
            <span class="status-dot" :class="`is-${component.status}`" aria-hidden="true"></span>
            <span>{{ component.name }}</span>
            <span>{{ t(`needtoken.statusStrip.component.${component.status}`) }}</span>
          </li>
        </ul>
        <p class="status-history">{{ t('needtoken.statusStrip.noHistory') }}</p>
        <p v-if="card.snapshot.updatedAt" class="status-time">{{ freshness(card.snapshot.updatedAt) }}</p>
      </template>
    </article>
  </section>
</template>

<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { ArrowUpRight } from 'lucide-vue-next'
import { operationalCount, parsePublicStatus, type PublicStatusSnapshot, type PublicStatusComponent } from '../lib/publicStatus'

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
let controller: AbortController | undefined

const sourceHost = (href: string) => {
  try { return new URL(href).host } catch { return href }
}
const dotClass = (card: StatusCard) => {
  if (card.state !== 'ready' || !card.snapshot) return 'is-unknown'
  return card.snapshot.indicator === 'none' ? 'is-operational' : 'is-degraded_performance'
}
const notable = (snapshot: PublicStatusSnapshot): PublicStatusComponent[] => {
  const degraded = snapshot.components.filter(component => component.status !== 'operational')
  return (degraded.length ? degraded : snapshot.components).slice(0, 4)
}
const freshness = (value: string) => {
  const time = new Date(value)
  const rendered = time.toLocaleString(locale.value)
  const stale = Date.now() - time.getTime() > 24 * 60 * 60 * 1000
  return t(stale ? 'needtoken.statusStrip.stale' : 'needtoken.statusStrip.updated', { time: rendered })
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
  controller = new AbortController()
  const timer = window.setTimeout(() => controller?.abort(), 8000)
  void Promise.all(cards.value.map(loadCard)).finally(() => window.clearTimeout(timer))
})
onUnmounted(() => controller?.abort())
</script>

<style scoped>
.status-strip { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 14px; margin: 0 42px; padding: 22px 0 8px; }
.status-card { min-width: 0; padding: 16px 16px 14px; border: 1px solid rgba(160, 190, 185, .16); border-radius: 16px; background: rgba(255,255,255,.025); }
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
@media (max-width: 800px) { .status-strip { grid-template-columns: 1fr; margin-inline: 22px; } }
</style>
