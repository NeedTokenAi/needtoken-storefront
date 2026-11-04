<template>
  <div class="nt-environment mx-auto w-full max-w-[1160px] px-5 pb-20 pt-12 sm:px-6 sm:pt-16">
    <header class="max-w-[780px]">
      <span class="nt-tool-eyebrow"><Activity :size="15" aria-hidden="true" />{{ t('needtoken.environment.eyebrow') }}</span>
      <h1 class="mt-5 text-[34px] font-bold leading-tight tracking-tight text-foreground sm:text-[48px]">{{ t('needtoken.environment.title') }}</h1>
      <p class="mt-4 max-w-[65ch] text-[15px] leading-7 text-muted-foreground sm:text-[16px]">{{ t('needtoken.environment.subtitle') }}</p>
    </header>

    <div class="nt-tool-privacy mt-7 flex items-start gap-3 rounded-2xl border p-4 text-[13px] leading-6 sm:p-5">
      <ShieldCheck :size="20" class="mt-0.5 shrink-0 text-primary" aria-hidden="true" />
      <p>{{ t('needtoken.environment.privacy') }}</p>
    </div>

    <section class="mt-8 rounded-[24px] border border-border bg-card/75" :aria-label="t('needtoken.environment.checks')" :aria-busy="running">
      <div class="flex flex-col gap-5 border-b border-border p-5 sm:flex-row sm:items-center sm:justify-between sm:p-6">
        <div>
          <h2 class="text-[20px] font-semibold text-foreground">{{ t('needtoken.environment.checks') }}</h2>
          <p class="mt-1 text-[13px] leading-6 text-muted-foreground" role="status" aria-live="polite">
            {{ running ? t('needtoken.environment.runningHint') : measuredAt ? t('needtoken.environment.checkedAt', { time: displayTime }) : t('needtoken.environment.idleHint') }}
          </p>
        </div>
        <div class="flex shrink-0 flex-wrap gap-2">
          <button v-if="running" type="button" class="nt-tool-button border border-border bg-secondary text-foreground" @click="cancelChecks">
            <Square :size="14" aria-hidden="true" />{{ t('needtoken.environment.cancel') }}
          </button>
          <button type="button" class="nt-tool-button bg-primary text-primary-foreground disabled:cursor-wait disabled:opacity-60" :disabled="running" @click="runChecks">
            <LoaderCircle v-if="running" :size="17" class="nt-tool-spinner" aria-hidden="true" /><Play v-else :size="16" aria-hidden="true" />
            {{ running ? t('needtoken.environment.running') : measuredAt ? t('needtoken.environment.runAgain') : t('needtoken.environment.run') }}
          </button>
        </div>
      </div>

      <div class="grid gap-px overflow-hidden rounded-b-[24px] bg-border md:grid-cols-2">
        <article v-for="id in localIds" :key="id" class="nt-tool-row flex gap-3 bg-card p-5 sm:p-6">
          <component :is="icons[id]" :size="19" class="mt-1 shrink-0 text-muted-foreground" aria-hidden="true" />
          <div class="min-w-0 flex-1">
            <div class="flex flex-wrap items-center justify-between gap-2">
              <h3 class="text-[14px] font-semibold text-foreground">{{ t(`needtoken.environment.labels.${id}`) }}</h3>
              <span class="nt-tool-status" :class="`is-${resultFor(id)?.status || 'idle'}`">
                <component :is="statusIcon(resultFor(id)?.status)" :size="12" aria-hidden="true" />
                {{ t(`needtoken.environment.status.${resultFor(id)?.status || 'idle'}`) }}
              </span>
            </div>
            <p v-if="resultFor(id)?.value" class="mt-2 break-words font-mono text-[13px] text-foreground">{{ resultFor(id)?.value }}</p>
            <p class="mt-2 text-[13px] leading-6 text-muted-foreground">{{ t(`needtoken.environment.details.${resultFor(id)?.detail || 'notChecked'}`) }}</p>
          </div>
        </article>
      </div>
    </section>

    <section class="mt-5 rounded-[24px] border border-border bg-card/75 p-5 sm:p-6" :aria-label="t('needtoken.environment.connectionTitle')">
      <div class="flex flex-wrap items-center justify-between gap-3">
        <h2 class="flex items-center gap-3 text-[18px] font-semibold text-foreground"><Radio :size="20" class="text-primary" aria-hidden="true" />{{ t('needtoken.environment.connectionTitle') }}</h2>
        <span class="nt-tool-status" :class="running ? 'is-running' : `is-${connection?.status || 'idle'}`">
          <LoaderCircle v-if="running" :size="12" class="nt-tool-spinner" aria-hidden="true" />
          <component v-else :is="statusIcon(connection?.status)" :size="12" aria-hidden="true" />
          {{ running ? t('needtoken.environment.running') : t(`needtoken.environment.status.${connection?.status || 'idle'}`) }}
        </span>
      </div>
      <p class="mt-3 text-[14px] leading-7 text-muted-foreground">{{ t(`needtoken.environment.connection.${connection?.detail || 'notChecked'}`) }}</p>
      <div v-if="connection?.latencyMs !== undefined" class="mt-4 inline-flex items-baseline gap-2 rounded-xl border border-primary/20 bg-primary/5 px-4 py-3">
        <span class="font-mono text-[24px] font-semibold text-primary">{{ connection.latencyMs }}<span class="ml-1 text-[13px]">ms</span></span>
        <span class="text-[12px] text-muted-foreground">{{ t('needtoken.environment.requestTime') }}</span>
      </div>
      <p v-if="connection?.httpStatus" class="mt-3 font-mono text-[13px] text-muted-foreground">{{ t('needtoken.environment.httpStatus', { status: connection.httpStatus }) }}</p>
      <p class="mt-4 text-[12px] leading-6 text-muted-foreground">{{ t('needtoken.environment.connectionScope') }}</p>
    </section>

    <div class="mt-8 grid gap-4 md:grid-cols-2">
      <section class="rounded-2xl border border-border bg-card/65 p-5 sm:p-6">
        <h2 class="flex items-center gap-2 text-[16px] font-semibold text-foreground"><CircleHelp :size="18" class="text-primary" aria-hidden="true" />{{ t('needtoken.environment.limitsTitle') }}</h2>
        <p class="mt-3 text-[13px] leading-7 text-muted-foreground">{{ t('needtoken.environment.limits') }}</p>
        <router-link to="/faq" class="mt-4 inline-flex items-center gap-2 text-[13px] font-medium text-primary hover:underline">{{ t('needtoken.environment.faq') }}<ArrowUpRight :size="14" aria-hidden="true" /></router-link>
      </section>
      <section class="rounded-2xl border border-border bg-card/65 p-5 sm:p-6">
        <h2 class="flex items-center gap-2 text-[16px] font-semibold text-foreground"><FileText :size="18" class="text-primary" aria-hidden="true" />{{ t('needtoken.environment.reportTitle') }}</h2>
        <p class="mt-3 text-[13px] leading-7 text-muted-foreground">{{ t('needtoken.environment.reportHint') }}</p>
        <button type="button" class="nt-tool-button mt-4 border border-border bg-secondary text-[13px] text-foreground disabled:cursor-not-allowed disabled:opacity-50" :disabled="!measuredAt || running" @click="copyReport"><Copy :size="14" aria-hidden="true" />{{ t('needtoken.environment.copyReport') }}</button>
        <p v-if="reportMessage" role="status" class="mt-2 text-[12px] leading-6 text-muted-foreground">{{ t(`needtoken.environment.${reportMessage}`) }}</p>
        <details v-if="measuredAt && !running" class="mt-4 text-[12px] text-muted-foreground">
          <summary class="cursor-pointer rounded-sm py-1 text-primary">{{ t('needtoken.environment.viewReport') }}</summary>
          <textarea :value="report" readonly :aria-label="t('needtoken.environment.reportTitle')" class="mt-3 h-[180px] w-full resize-y rounded-lg border border-border bg-background p-3 font-mono text-[11px] leading-5 text-muted-foreground" />
        </details>
      </section>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { Activity, ArrowUpRight, Check, CircleHelp, CircleMinus, Copy, Database, FileText, Globe, KeyRound, Languages, LoaderCircle, LockKeyhole, Play, Radio, ShieldCheck, Square, TriangleAlert, Cookie, Cpu } from 'lucide-vue-next'
import { usePageSeo } from '../../composables/usePageSeo'
import { collectLocalDiagnostics, probeStorefrontConnection, redactedDiagnosticReport } from './lib/browserDiagnostics'
import type { ConnectionDiagnostic, DiagnosticId, DiagnosticStatus, LocalDiagnostic } from './lib/browserDiagnostics'

const { t, locale } = useI18n()
usePageSeo({ title: () => t('needtoken.environment.title'), description: () => t('needtoken.environment.subtitle') })

const localIds: DiagnosticId[] = ['secure', 'online', 'cookies', 'storage', 'crypto', 'wasm', 'language', 'timezone']
const icons = { secure: LockKeyhole, online: Globe, cookies: Cookie, storage: Database, crypto: KeyRound, wasm: Cpu, language: Languages, timezone: Globe }
const localResults = ref<LocalDiagnostic[]>([])
const connection = ref<ConnectionDiagnostic | null>(null)
const measuredAt = ref('')
const running = ref(false)
const reportMessage = ref('')
let activeController: AbortController | null = null
let generation = 0
const displayTime = computed(() => measuredAt.value ? new Date(measuredAt.value).toLocaleTimeString(locale.value) : '')
const report = computed(() => redactedDiagnosticReport(localResults.value, connection.value, measuredAt.value))
const resultFor = (id: DiagnosticId) => localResults.value.find((row) => row.id === id)
const statusIcon = (status?: DiagnosticStatus) => status === 'ok' ? Check : status === 'warning' ? TriangleAlert : CircleMinus

async function runChecks() {
  if (running.value) return
  const current = ++generation
  const controller = new AbortController()
  activeController = controller
  running.value = true
  measuredAt.value = ''
  connection.value = null
  reportMessage.value = ''
  localResults.value = collectLocalDiagnostics({
    secure: window.isSecureContext,
    online: navigator.onLine,
    cookiesEnabled: navigator.cookieEnabled,
    cookieDocument: document,
    storage: () => window.localStorage,
    cryptoAvailable: !!window.crypto?.subtle,
    wasmAvailable: typeof WebAssembly === 'object' && typeof WebAssembly.validate === 'function',
    language: navigator.language || '',
    timezone: () => Intl.DateTimeFormat().resolvedOptions().timeZone || '',
  })
  try {
    const result = await probeStorefrontConnection({ signal: controller.signal })
    if (current !== generation) return
    connection.value = result
    measuredAt.value = new Date().toISOString()
  } catch {
    // Cancellation deliberately leaves no successful report or stale network result.
  } finally {
    if (current === generation) {
      running.value = false
      activeController = null
    }
  }
}

function cancelChecks() {
  generation++
  activeController?.abort()
  activeController = null
  running.value = false
  measuredAt.value = ''
  connection.value = null
  reportMessage.value = 'cancelled'
}

async function copyReport() {
  if (!measuredAt.value || running.value) return
  try {
    await navigator.clipboard.writeText(report.value)
    reportMessage.value = 'copied'
  } catch {
    reportMessage.value = 'copyFailed'
  }
}

onBeforeUnmount(() => {
  generation++
  activeController?.abort()
})
</script>

<style scoped>
.nt-tool-eyebrow { display: inline-flex; align-items: center; gap: 9px; color: var(--ui-accent); font-size: 12px; font-weight: 600; letter-spacing: .06em; }
.nt-tool-privacy { border-color: color-mix(in srgb, var(--ui-accent) 20%, transparent); background: color-mix(in srgb, var(--ui-accent) 5%, transparent); color: var(--ui-text-muted); }
.nt-tool-button { display: inline-flex; justify-content: center; align-items: center; gap: 8px; min-height: 42px; border-radius: 11px; padding: 10px 15px; font-weight: 600; font-size: 13px; transition: background-color .15s, border-color .15s; }
.nt-tool-status { display: inline-flex; align-items: center; gap: 5px; flex-shrink: 0; border: 1px solid var(--ui-border); border-radius: 999px; padding: 3px 8px; font-size: 11px; color: var(--ui-text-muted); background: color-mix(in srgb, var(--ui-text-muted) 6%, transparent); }
.nt-tool-status.is-ok { color: var(--ui-accent); border-color: color-mix(in srgb, var(--ui-accent) 25%, transparent); background: color-mix(in srgb, var(--ui-accent) 7%, transparent); }
.nt-tool-status.is-warning { color: #e5b969; border-color: #e5b96940; background: #e5b9690b; }
.nt-tool-status.is-running { color: var(--ui-accent); }
.nt-tool-spinner { animation: nt-diagnostic-spin 1s linear infinite; }
@keyframes nt-diagnostic-spin { to { transform: rotate(360deg); } }
@media (prefers-reduced-motion: reduce) { .nt-tool-spinner { animation: none; } .nt-tool-button { transition: none; } }
</style>
