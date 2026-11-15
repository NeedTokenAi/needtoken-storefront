<template>
  <div class="nt-claude-network">
    <header class="network-heading">
      <div><span class="tool-capability"><Network :size="16" :stroke-width="1.5" aria-hidden="true" />{{ t('needtoken.toolsPolish.networkScope') }}</span><h1>{{ t('needtoken.environmentV6.title') }}</h1><p>{{ t('needtoken.environmentV6.subtitle') }}</p></div>
      <div class="network-actions">
        <a href="https://status.claude.com/" target="_blank" rel="noopener noreferrer" class="network-status-link">{{ t('needtoken.toolsPolish.providerStatus') }}<ArrowUpRight :size="14" aria-hidden="true" /></a>
        <button v-if="running" type="button" class="network-button secondary" @click="cancel"><Square :size="16" aria-hidden="true" />{{ t('needtoken.claudeEnvironment.cancel') }}</button>
        <button type="button" class="network-button primary" :disabled="running" :aria-busy="running" @click="run"><LoaderCircle v-if="running" class="network-spin" :size="16" aria-hidden="true" /><Play v-else :size="16" aria-hidden="true" />{{ t(running ? 'needtoken.claudeEnvironment.running' : measuredAt ? 'needtoken.claudeEnvironment.runAgain' : 'needtoken.claudeEnvironment.run') }}</button>
      </div>
    </header>
    <div class="network-options"><label><input v-model="includeWebRtc" type="checkbox" :disabled="running" />{{ t('needtoken.claudeEnvironment.rtcChoice') }}</label><p>{{ t('needtoken.environmentV6.privacy') }}</p></div>
    <p class="network-run-state" :class="{ 'is-running': running }" role="status" aria-live="polite">{{ running ? t('needtoken.environmentV6.running', { count: pending.size }) : cancelled ? t('needtoken.environmentV5.cancelledHint') : measuredAt ? t('needtoken.claudeEnvironment.checkedAt', { time: dateTime(measuredAt) }) : t('needtoken.environmentV6.idle') }}</p>
    <div class="network-primary">
      <section class="network-panel network-services" :aria-label="t('needtoken.environmentV6.services')" :aria-busy="running">
        <div class="network-panel-heading"><h2><Globe2 :size="16" aria-hidden="true" />{{ t('needtoken.environmentV6.services') }}</h2><span class="network-meta">{{ serviceCompleted }} / {{ CLAUDE_SERVICES.length }}</span></div>
        <progress v-if="running" class="network-progress" :value="serviceCompleted" :max="CLAUDE_SERVICES.length" :aria-label="t('needtoken.environmentV6.services')" />
        <ul class="network-service-list">
          <li v-for="id in CLAUDE_SERVICES" :key="id" :data-network-check="id" :class="{ 'is-pending': pending.has(id) }">
            <div class="network-service-name"><span class="network-service-icon" aria-hidden="true"><Code2 v-if="id === 'anthropic'" :size="16" /><Globe2 v-else :size="16" /></span><strong>{{ t(`needtoken.environmentV6.targets.${id}`) }}</strong><span translate="no">{{ serviceHost(id) }}</span></div>
            <span class="network-service-state" :class="serviceClass(id)"><LoaderCircle v-if="pending.has(id)" :size="14" class="network-spin" aria-hidden="true" /><span v-else class="network-status-dot" aria-hidden="true" />{{ serviceLabel(id) }}</span>
            <span class="network-latency" translate="no"><span v-if="pending.has(id)" class="network-skeleton-line latency-skeleton" aria-hidden="true" /><template v-else>{{ services[id]?.latencyMs !== undefined ? `${services[id]!.latencyMs} ms` : '—' }}</template></span>
            <details v-if="services[id]" class="network-service-detail"><summary>{{ t('needtoken.toolsPolish.responseDetails') }}<span v-if="services[id]!.httpStatus" translate="no">HTTP {{ services[id]!.httpStatus }}</span><ChevronDown :size="14" aria-hidden="true" /></summary><p>{{ t(`needtoken.environmentV6.details.${services[id]!.detail}`) }}</p></details>
          </li>
        </ul>
        <p v-if="adviceKeys.includes('opaque')" class="network-observed-note"><Info :size="14" aria-hidden="true" />{{ t('needtoken.environmentV6.details.opaqueResponse') }}</p>
        <details class="network-scope"><summary>{{ t('needtoken.toolsPolish.scopeDetails') }}<ChevronDown :size="14" aria-hidden="true" /></summary><p>{{ t('needtoken.environmentV6.browserScope') }}</p></details>
      </section>
      <section class="network-panel network-exit" :aria-label="t('needtoken.claudeEnvironment.overview')" data-network-check="exit" :aria-busy="pending.has('exit')">
        <div class="network-panel-heading"><h2><Network :size="16" aria-hidden="true" />{{ t('needtoken.claudeEnvironment.overview') }}</h2><span class="network-meta">{{ pending.has('exit') ? t('needtoken.environmentV5.pending') : exit?.source || '—' }}</span></div>
        <p class="network-exit-ip" translate="no"><span v-if="pending.has('exit')" class="network-skeleton-line exit-skeleton" aria-hidden="true" /><template v-else>{{ exit?.ip || '—' }}</template></p>
        <dl class="network-facts"><div><dt>{{ t('needtoken.claudeEnvironment.country') }}</dt><dd><span v-if="pending.has('exit')" class="network-skeleton-line" aria-hidden="true" /><template v-else>{{ exit?.country || unknown }}</template></dd></div><div><dt>{{ t('needtoken.claudeEnvironment.location') }}</dt><dd><span v-if="pending.has('exit')" class="network-skeleton-line" aria-hidden="true" /><template v-else>{{ [exit?.region, exit?.city].filter(Boolean).join(' · ') || unknown }}</template></dd></div><div><dt translate="no">ASN</dt><dd><span v-if="pending.has('exit')" class="network-skeleton-line" aria-hidden="true" /><template v-else>{{ exit?.asn || unknown }}</template></dd></div><div><dt>{{ t('needtoken.claudeEnvironment.organization') }}</dt><dd><span v-if="pending.has('exit')" class="network-skeleton-line" aria-hidden="true" /><template v-else>{{ exit?.organization || unknown }}</template></dd></div></dl>
        <p v-if="exit?.detail === 'lookupUnavailable'" class="network-note">{{ t('needtoken.environmentV6.exitUnavailable') }}</p><p v-else-if="exit?.detail === 'ipOnly'" class="network-note">{{ t('needtoken.environmentV6.exitPartial') }}</p>
      </section>
    </div>
    <section v-if="measuredAt && !running && !cancelled && adviceKeys.length" class="network-advice" :aria-label="t('needtoken.environmentV6.nextSteps')"><h2>{{ t('needtoken.environmentV6.nextSteps') }}</h2><ul><li v-for="key in adviceKeys" :key="key">{{ t(`needtoken.environmentV6.advice.${key}`) }}</li></ul></section>
    <section v-if="rtcIncluded" class="network-panel network-rtc" data-network-check="rtc">
      <div class="network-panel-heading"><h2>{{ t('needtoken.environmentV6.webrtc') }}</h2><span class="network-service-state" :class="rtc?.status === 'warning' ? 'is-warning' : rtc?.status === 'ok' ? 'is-readable' : ''">{{ pending.has('rtc') ? t('needtoken.environmentV5.pending') : rtc ? t(`needtoken.environmentV6.rtcStates.${rtc.detail}`) : '—' }}</span></div>
      <p class="network-copy">{{ t(`needtoken.claudeEnvironment.rtcDetails.${rtc?.detail || 'notChecked'}`) }}</p><div v-if="rtc?.publicAddresses.length" class="network-addresses"><code v-for="address in rtc.publicAddresses" :key="address">{{ address }}</code></div><p v-if="rtc" class="network-note">{{ t('needtoken.claudeEnvironment.rtcCounts', { local: rtc.localCount, mdns: rtc.mdnsCount }) }}</p>
    </section>
    <details class="network-details"><summary><Monitor :size="16" class="network-detail-symbol" aria-hidden="true" />{{ t('needtoken.environmentV6.localDetails') }}<ChevronDown :size="16" aria-hidden="true" /></summary><div class="network-details-content">
      <p v-if="!measuredAt" class="network-note">{{ t('needtoken.environmentV6.localIdle') }}</p>
      <dl v-else class="network-local-facts"><div v-for="check in localChecks" :key="check.id"><dt>{{ localLabel(check.id) }}</dt><dd>{{ check.value || statusLabel(check.status) }}</dd></div><div><dt>{{ t('needtoken.claudeEnvironment.exitTimezone') }}</dt><dd>{{ exit?.timezone || unknown }}</dd></div><div><dt>{{ t('needtoken.environmentV5.healthTitle') }}</dt><dd>{{ storefront?.latencyMs !== undefined ? `${storefront.latencyMs} ms` : statusLabel(storefront?.status) }}</dd></div></dl><p v-if="timeMismatch" class="network-note">{{ t('needtoken.environmentV6.timezoneNote') }}</p>
    </div></details>
        <details id="network-terminal" class="network-details">
          <summary><Terminal :size="16" class="network-detail-symbol" aria-hidden="true" />{{ t('needtoken.environmentV6.terminalTitle') }}<span class="network-meta">{{ cli ? t(`needtoken.claudeEnvironment.cliStates.${cli.curl.status}`) : t('needtoken.claudeEnvironment.notImported') }}</span><ChevronDown :size="16" aria-hidden="true" /></summary><div class="network-details-content">
          <p class="network-copy">{{ t('needtoken.environmentV6.terminalIntro') }}</p>
          <ol class="network-cli-steps"><li>{{ t('needtoken.claudeEnvironment.cliStep1') }} <a href="/needtoken-claude-network-check.mjs" download>{{ t('needtoken.claudeEnvironment.downloadScript') }}<Download :size="13" /></a><a href="/needtoken-claude-network-check.mjs" target="_blank" rel="noopener noreferrer">{{ t('needtoken.claudeEnvironment.reviewScript') }}<ArrowUpRight :size="13" /></a></li><li>{{ t('needtoken.claudeEnvironment.cliStep2') }}<code class="network-command">node needtoken-claude-network-check.mjs &gt; claude-network-report.json</code></li><li>{{ t('needtoken.claudeEnvironment.cliStep3') }}</li></ol>
          <label class="network-import"><Upload :size="16" />{{ t('needtoken.claudeEnvironment.importReport') }}<input type="file" accept="application/json,.json" @change="importCliReport" /></label><p v-if="importMessage" class="network-note" role="status">{{ t(`needtoken.claudeEnvironment.${importMessage}`) }}</p>
          <template v-if="cli"><p class="network-import-label">{{ t('needtoken.claudeEnvironment.importedAt', { time: dateTime(cli.measured_at) }) }}</p><dl class="network-facts"><div><dt>{{ t('needtoken.claudeEnvironment.cliDns') }}</dt><dd>{{ t(`needtoken.claudeEnvironment.cliStates.${cli.dns.status}`) }}</dd></div><div><dt>{{ t('needtoken.claudeEnvironment.cliCurl') }}</dt><dd>{{ t(`needtoken.claudeEnvironment.cliStates.${cli.curl.status}`) }}<span v-if="cli.curl.http_status"> · HTTP {{ cli.curl.http_status }}</span><span v-if="cli.curl.exit_code >= 0"> · {{ t('needtoken.claudeEnvironment.curlExit', { code: cli.curl.exit_code }) }}</span></dd></div><div><dt>{{ t('needtoken.claudeEnvironment.cliTls') }}</dt><dd>{{ cli.curl.ssl_verify_result === 0 && cli.curl.http_status ? t('needtoken.claudeEnvironment.cliTlsVerified') : cli.curl.ssl_verify_result ? t('needtoken.claudeEnvironment.tlsErrorCode', { code: cli.curl.ssl_verify_result }) : unknown }}</dd></div><div><dt>{{ t('needtoken.claudeEnvironment.cliProxy') }}</dt><dd>{{ t(hasCliProxy ? 'needtoken.claudeEnvironment.proxyPresent' : 'needtoken.claudeEnvironment.proxyAbsent') }}</dd></div></dl><p v-if="cli.curl.status === 'error' && [5, 6, 7, 28, 35, 60].includes(cli.curl.exit_code)" class="network-note">{{ t(`needtoken.claudeEnvironment.curlErrors.${cli.curl.exit_code}`) }}</p><p v-if="cli.proxy.custom_base_url" class="network-review-note"><Info :size="15" />{{ t('needtoken.claudeEnvironment.customBaseUrl') }}</p></template>

        </div></details>
    <details v-if="measuredAt || cli" class="network-details network-report"><summary><FileJson :size="16" class="network-detail-symbol" aria-hidden="true" />{{ t('needtoken.claudeEnvironment.reportTitle') }}<ChevronDown :size="16" aria-hidden="true" /></summary><div class="network-details-content"><p class="network-copy">{{ t('needtoken.claudeEnvironment.reportHint') }}</p><button type="button" class="network-button secondary" :disabled="running" @click="copyReport"><Copy :size="16" aria-hidden="true" />{{ t('needtoken.claudeEnvironment.copyReport') }}</button><p v-if="copyMessage" class="network-note" role="status">{{ t(`needtoken.claudeEnvironment.${copyMessage}`) }}</p><textarea :value="report" readonly :aria-label="t('needtoken.claudeEnvironment.reportTitle')" /></div></details>
  </div>
</template>
<script setup lang="ts">
import { computed, onBeforeUnmount, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { ArrowUpRight, ChevronDown, Code2, Copy, Download, FileJson, Globe2, Info, LoaderCircle, Monitor, Network, Play, Square, Terminal, Upload } from 'lucide-vue-next'
import { usePageSeo } from '../../composables/usePageSeo'
import { CLAUDE_PROBES, CLAUDE_SERVICES, claudeNetworkReport, collectClaudeLocalChecks, parseCliReport, runClaudeNetworkChecks, timezoneDiffers } from './lib/claudeNetwork'
import type { ConnectionDiagnostic } from './lib/browserDiagnostics'
import type { AnthropicConnection, ClaudeLocalCheck, ClaudeServiceId, CliReport, ExitNetwork, LocalCheckId, NetworkCheckId, NetworkStatus, WebRtcObservation } from './lib/claudeNetwork'

const { t, locale } = useI18n()
usePageSeo({ title: () => t('needtoken.claudeEnvironment.title'), description: () => t('needtoken.environmentV6.subtitle') })
const unknown = computed(() => t('needtoken.environmentV6.notAvailable'))
const exit = ref<ExitNetwork | null>(null), services = ref<Partial<Record<ClaudeServiceId, AnthropicConnection>>>({}), rtc = ref<WebRtcObservation | null>(null), cli = ref<CliReport | null>(null)
const running = ref(false), measuredAt = ref(''), includeWebRtc = ref(false), copyMessage = ref(''), importMessage = ref(''), cancelled = ref(false)
const localChecks = ref<ClaudeLocalCheck[]>([]), storefront = ref<ConnectionDiagnostic | null>(null), pending = ref(new Set<NetworkCheckId>()), rtcIncluded = ref(false)
let active: AbortController | null = null, generation = 0, importGeneration = 0, disposed = false
const dateTime = (value: string) => { try { return Number.isFinite(Date.parse(value)) ? new Date(value).toLocaleString(locale.value) : unknown.value } catch { return unknown.value } }
const statusLabel = (status?: NetworkStatus) => t(`needtoken.claudeEnvironment.status.${status || 'idle'}`)
const timeMismatch = computed(() => timezoneDiffers(localChecks.value.find(check => check.id === 'timezone')?.value || '', exit.value))
const hasCliProxy = computed(() => !!cli.value && (cli.value.proxy.http || cli.value.proxy.https || cli.value.proxy.all))
const serviceCompleted = computed(() => CLAUDE_SERVICES.filter(id => !!services.value[id]).length)
const serviceHost = (id: ClaudeServiceId) => new URL(CLAUDE_PROBES[id]).hostname
const serviceLabel = (id: ClaudeServiceId) => pending.value.has(id) ? t('needtoken.environmentV5.pending') : cancelled.value && !services.value[id] ? t('needtoken.environmentV5.stopped') : t(`needtoken.environmentV6.states.${services.value[id]?.detail || 'idle'}`)
const serviceClass = (id: ClaudeServiceId) => services.value[id]?.detail === 'opaqueResponse' ? 'is-observed' : services.value[id]?.status === 'warning' ? 'is-warning' : services.value[id]?.detail === 'readableResponse' ? 'is-readable' : ''
const oldIds = new Set(['secure', 'online', 'cookies', 'storage', 'crypto', 'wasm', 'language', 'timezone'])
const localLabel = (id: LocalCheckId) => t(`${oldIds.has(id) ? 'needtoken.environment' : 'needtoken.environmentV5'}.labels.${id}`)
const adviceKeys = computed(() => {
  const rows = Object.values(services.value), keys: string[] = []
  if (localChecks.value.some(check => check.id === 'online' && check.status === 'warning')) keys.push('offline')
  if (rows.some(row => row?.detail === 'timeout' || row?.detail === 'requestUnconfirmed')) keys.push('connection')
  if (rows.some(row => row?.httpStatus === 403)) keys.push('restricted')
  if (rows.some(row => row?.httpStatus === 429)) keys.push('rateLimit')
  if (rows.some(row => (row?.httpStatus || 0) >= 500)) keys.push('upstream')
  if (rows.some(row => row?.detail === 'opaqueResponse')) keys.push('opaque')
  return keys
})
const report = computed(() => claudeNetworkReport({ exit: exit.value, anthropic: services.value.anthropic || null, services: services.value, rtc: rtc.value, cli: cli.value, measuredAt: measuredAt.value, local: localChecks.value, storefront: storefront.value }))
async function run() {
  if (running.value) return
  const current = ++generation, controller = typeof AbortController === 'function' ? new AbortController() : null
  active = controller; running.value = true; cancelled.value = false; copyMessage.value = ''
  measuredAt.value = new Date().toISOString(); exit.value = null; services.value = {}; storefront.value = null
  rtcIncluded.value = includeWebRtc.value
  rtc.value = includeWebRtc.value ? null : { status: 'unknown', detail: 'notRequested', publicAddresses: [], localCount: 0, mdnsCount: 0 }
  pending.value = new Set<NetworkCheckId>(['storefront', 'exit', ...CLAUDE_SERVICES, ...(rtcIncluded.value ? ['rtc' as const] : [])])
  localChecks.value = collectClaudeLocalChecks()
  try {
    await runClaudeNetworkChecks({ signal: controller?.signal, includeWebRtc: rtcIncluded.value, onResult: event => {
      if (disposed || generation !== current) return
      if (event.id === 'storefront') storefront.value = event.result
      else if (event.id === 'exit') exit.value = event.result
      else if (event.id === 'rtc') rtc.value = event.result
      else services.value = { ...services.value, [event.id]: event.result }
      const remaining = new Set(pending.value); remaining.delete(event.id); pending.value = remaining
    } })
  } catch { /* Cancelled runs keep completed observations. */ }
  finally { if (!disposed && generation === current) { running.value = false; pending.value = new Set(); active = null } }
}
function cancel() { generation++; active?.abort(); active = null; running.value = false; cancelled.value = true; pending.value = new Set() }
async function copyReport() { try { await navigator.clipboard.writeText(report.value); copyMessage.value = 'copied' } catch { copyMessage.value = 'copyFailed' } }
async function importCliReport(event: Event) {
  const input = event.target as HTMLInputElement, file = input.files?.[0], current = ++importGeneration
  input.value = ''; importMessage.value = ''
  if (!file) return
  if (file.size > 65536) { importMessage.value = 'invalidReport'; return }
  try {
    const parsed = parseCliReport(await file.text())
    if (disposed || current !== importGeneration) return
    if (!parsed) { importMessage.value = 'invalidReport'; return }
    cli.value = parsed; importMessage.value = 'imported'
  } catch { if (!disposed && current === importGeneration) importMessage.value = 'invalidReport' }
}
onBeforeUnmount(() => { disposed = true; generation++; importGeneration++; active?.abort() })
</script>
<style scoped>
.nt-claude-network { width:100%; max-width:1120px; margin:0 auto; padding:32px 24px 64px; }
.tool-capability { display:inline-flex; min-height:24px; align-items:center; margin-bottom:12px; color:var(--ui-text-muted); font-size:12px; line-height:20px; }
.network-heading { display:flex; align-items:center; justify-content:space-between; gap:24px; }
.network-heading h1 { font-size:clamp(28px,3vw,36px); line-height:1.2; font-weight:500; }
.network-heading p { max-width:56ch; margin-top:12px; color:var(--ui-text-secondary); font-size:14px; line-height:1.7; }
.network-actions { display:flex; flex:none; flex-wrap:wrap; align-items:center; justify-content:flex-end; gap:8px 16px; }
.network-status-link { display:inline-flex; min-height:44px; align-items:center; gap:4px; color:var(--ui-text-muted); font-size:12px; line-height:20px; }
.network-status-link:hover { color:var(--ui-text-primary); }
.network-button { display:inline-flex; align-items:center; justify-content:center; gap:8px; min-height:44px; padding:12px 16px; border:1px solid transparent; border-radius:8px; font-size:13px; line-height:20px; font-weight:500; transition:color var(--nt-motion-fast,120ms) var(--nt-motion-ease,ease-out),background-color var(--nt-motion-fast,120ms) var(--nt-motion-ease,ease-out),border-color var(--nt-motion-fast,120ms) var(--nt-motion-ease,ease-out),transform var(--nt-motion-fast,120ms) var(--nt-motion-ease,ease-out); }
.network-button.primary { background:var(--ui-accent); color:var(--ui-text-on-accent); }
.network-button.secondary { background:var(--ui-bg-soft); border-color:var(--ui-border); color:var(--ui-text-primary); }
.network-button:hover:not(:disabled) { background:var(--ui-bg-muted); color:var(--ui-text-primary); }
.network-button.primary:hover:not(:disabled) { background:var(--ui-accent-hover); color:var(--ui-text-on-accent); }
.network-button:disabled { cursor:wait; opacity:.6; }
.network-options { display:flex; flex-wrap:wrap; align-items:center; gap:12px 24px; margin-top:20px; }
.network-options label { display:inline-flex; flex:none; align-items:center; gap:8px; color:var(--ui-text-secondary); font-size:13px; min-height:32px; }
.network-options input { accent-color:var(--ui-accent); }
.network-options p { max-width:64ch; color:var(--ui-text-muted); font-size:12px; line-height:20px; }
.network-run-state { min-height:20px; margin:12px 0 20px; color:var(--ui-text-muted); font-size:12px; line-height:20px; }
.network-primary { display:grid; grid-template-columns:minmax(0,1.5fr) minmax(280px,1fr); gap:24px; align-items:start; }
.network-panel { min-width:0; padding:24px; border:1px solid var(--ui-border); border-radius:16px; background:var(--ui-bg-elevated); }
.network-panel-heading { display:flex; align-items:center; justify-content:space-between; gap:16px; margin-bottom:24px; }
.network-panel-heading h2,.network-advice h2 { font-size:16px; line-height:24px; font-weight:500; }
.network-panel-heading h2 { display:flex; align-items:center; gap:8px; }
.network-meta { margin-left:auto; color:var(--ui-text-muted); font-size:12px; line-height:20px; font-variant-numeric:tabular-nums; }
.network-service-list { display:grid; }
.network-service-list>li { display:grid; grid-template-columns:minmax(0,1fr) minmax(100px,auto) 64px; gap:8px 16px; padding:16px 0; border-top:1px solid var(--ui-border); align-items:center; }
.network-service-list>li:first-child { padding-top:0; border-top:0; }
.network-service-name { display:grid; grid-template-columns:20px minmax(0,1fr); gap:4px 8px; min-width:0; }
.network-service-name strong,.network-service-name>span:not(.network-service-icon) { grid-column:2; }
.network-service-icon { display:grid; grid-row:1/3; place-items:center; align-self:center; color:var(--ui-text-muted); transition:color var(--nt-motion-fast,120ms) var(--nt-motion-ease,ease-out),transform var(--nt-motion-fast,120ms) var(--nt-motion-ease,ease-out); }
.network-service-list>li:hover .network-service-icon { color:var(--ui-text-primary); transform:translateY(-1px); }
.network-service-name strong { font-size:14px; line-height:20px; font-weight:500; }
.network-service-name>span { font-size:12px; line-height:20px; color:var(--ui-text-muted); overflow-wrap:anywhere; }
.network-service-state { display:inline-flex; align-items:center; gap:8px; color:var(--ui-text-muted); font-size:12px; line-height:20px; }
.network-service-state.is-observed { color:var(--ui-text-secondary); }
.network-service-state.is-readable { color:var(--ui-accent); }
.network-service-state.is-warning { color:#e5b969; }
.network-latency { justify-self:end; color:var(--ui-text-secondary); font-family:var(--nt-font-mono); font-size:12px; line-height:20px; font-variant-numeric:tabular-nums; }
.network-service-detail { grid-column:1/-1; color:var(--ui-text-muted); font-size:12px; line-height:20px; }
.network-service-detail summary, .network-scope summary { display:flex; width:fit-content; max-width:100%; align-items:center; gap:8px; cursor:pointer; list-style:none; }
.network-service-detail summary::-webkit-details-marker, .network-scope summary::-webkit-details-marker { display:none; }
.network-service-detail summary > span { color:var(--ui-text-secondary); }
.network-service-detail[open] summary > svg, .network-scope[open] summary > svg { transform:rotate(180deg); }
.network-service-detail p { margin-top:8px; }
.network-scope { margin-top:20px; padding-top:16px; border-top:1px solid var(--ui-border); color:var(--ui-text-muted); font-size:12px; line-height:20px; }
.network-scope p { margin-top:8px; }
.network-exit { align-self:stretch; }
.network-exit-ip { margin-bottom:24px; color:var(--ui-text-primary); font-family:var(--nt-font-mono); font-size:clamp(20px,2.5vw,28px); line-height:1.4; font-variant-numeric:tabular-nums; overflow-wrap:anywhere; }
.network-facts { display:grid; grid-template-columns:repeat(2,minmax(0,1fr)); gap:20px 16px; }
.network-facts dt { color:var(--ui-text-muted); font-size:12px; line-height:20px; }
.network-facts dd { margin-top:4px; color:var(--ui-text-secondary); font-size:13px; line-height:20px; overflow-wrap:anywhere; }
.network-note { margin-top:16px; color:var(--ui-text-muted); font-size:12px; line-height:20px; }
.network-copy { color:var(--ui-text-secondary); font-size:13px; line-height:20px; }
.network-advice { margin:24px 0; padding:0 4px; }
.network-advice ul { display:grid; gap:8px; margin-top:12px; padding-left:16px; list-style:disc; color:var(--ui-text-secondary); font-size:13px; line-height:20px; }
.network-rtc { margin-top:24px; }
.network-addresses { display:flex; flex-wrap:wrap; gap:8px 16px; margin-top:16px; font-size:13px; }
.network-details { margin-top:16px; border:1px solid var(--ui-border); border-radius:12px; background:var(--ui-bg-elevated); }
.network-details summary { display:flex; align-items:center; gap:12px; padding:20px 24px; color:var(--ui-text-secondary); font-size:14px; line-height:20px; cursor:pointer; list-style:none; }
.network-details summary::-webkit-details-marker { display:none; }
.network-details summary>svg:last-child { flex:none; margin-left:auto; color:var(--ui-text-muted); transition:transform var(--nt-motion-fast,120ms) var(--nt-motion-ease,ease-out),color var(--nt-motion-fast,120ms) var(--nt-motion-ease,ease-out); }
.network-details[open] summary>svg:last-child { transform:rotate(180deg); }
.network-details-content { padding:0 24px 24px; }
.network-local-facts { display:grid; }
.network-local-facts>div { display:grid; grid-template-columns:minmax(0,1fr) minmax(0,1.4fr); gap:16px; padding:12px 0; border-bottom:1px solid var(--ui-border); font-size:13px; line-height:20px; }
.network-local-facts dt { color:var(--ui-text-muted); }
.network-local-facts dd { color:var(--ui-text-secondary); text-align:right; overflow-wrap:anywhere; }
.network-cli-steps { display:grid; gap:16px; margin-top:20px; padding-left:20px; list-style:decimal; color:var(--ui-text-secondary); font-size:13px; line-height:20px; }
.network-cli-steps a { display:inline-flex; align-items:center; gap:4px; margin-left:12px; color:var(--ui-accent); }
.network-command { display:block; padding:16px; margin-top:12px; border:1px solid var(--ui-border); border-radius:8px; background:var(--ui-bg-page); color:var(--ui-text-secondary); font-size:12px; line-height:20px; overflow-wrap:anywhere; }
.network-import { position:relative; display:inline-flex; align-items:center; gap:8px; min-height:44px; padding:12px 16px; margin-top:20px; border:1px solid var(--ui-border); border-radius:8px; color:var(--ui-text-secondary); font-size:13px; line-height:20px; cursor:pointer; }
.network-import input { position:absolute; inset:0; width:100%; opacity:0; cursor:pointer; }
#network-terminal .network-facts { margin-top:24px; }
.network-review-note { margin-top:16px; color:#e5b969; font-size:13px; line-height:20px; }
.network-review-note>svg { display:inline; vertical-align:middle; margin-right:4px; }
.network-import-label { margin:20px 0; color:var(--ui-text-muted); font-size:12px; line-height:20px; }
.network-report .network-button { margin-top:16px; }
.network-report textarea { display:block; width:100%; min-height:240px; margin-top:16px; padding:16px; border:1px solid var(--ui-border); border-radius:8px; background:var(--ui-bg-page); color:var(--ui-text-secondary); font-family:var(--nt-font-mono); font-size:12px; line-height:20px; resize:vertical; }
.nt-claude-network :is(button,a,input,summary,textarea):focus-visible,.network-import:focus-within { outline:2px solid var(--ui-accent); outline-offset:4px; }
@keyframes network-spin { to { transform:rotate(360deg); } }
.network-spin { animation:network-spin 1s linear infinite; }

.tool-capability { gap:8px; }
.tool-capability svg { transition:color var(--nt-motion-fast,120ms) var(--nt-motion-ease,ease-out),transform var(--nt-motion-fast,120ms) var(--nt-motion-ease,ease-out); }
.network-heading:hover .tool-capability svg { color:var(--ui-accent); transform:translateY(-1px); }
.network-button:active:not(:disabled) { transform:translateY(1px); }
.network-button>svg { flex:none; transition:transform var(--nt-motion-fast,120ms) var(--nt-motion-ease,ease-out); }
.network-button:hover:not(:disabled)>svg:not(.network-spin),.network-button:focus-visible>svg:not(.network-spin) { transform:translateX(2px); }
.network-button:disabled { cursor:not-allowed; }
.network-button[aria-busy="true"] { cursor:progress; }
.network-options input { width:16px; height:16px; }
.network-run-state.is-running { color:var(--ui-text-secondary); }
.network-status-dot { width:6px; height:6px; flex:none; border-radius:50%; background:currentColor; }
.network-progress { display:block; width:100%; height:3px; margin:-8px 0 20px; border:0; border-radius:2px; appearance:none; background:var(--ui-bg-soft); accent-color:var(--ui-accent); }
.network-progress::-webkit-progress-bar { background:var(--ui-bg-soft); border-radius:2px; }
.network-progress::-webkit-progress-value { background:var(--ui-accent); border-radius:2px; }
.network-progress::-moz-progress-bar { background:var(--ui-accent); border-radius:2px; }
.network-observed-note { display:flex; align-items:flex-start; gap:8px; margin-top:8px; color:var(--ui-text-muted); font-size:12px; line-height:20px; }
.network-observed-note>svg { flex:none; margin-top:3px; }
.network-skeleton-line { display:block; width:80%; height:12px; min-height:12px; border-radius:4px; background:var(--ui-bg-soft); animation:network-pending 1.4s ease-in-out infinite; }
.latency-skeleton { width:40px; }
.exit-skeleton { width:72%; height:28px; }
.network-facts .network-skeleton-line { margin-top:8px; }
.network-detail-symbol { flex:none; color:var(--ui-text-muted); transition:color var(--nt-motion-fast,120ms) var(--nt-motion-ease,ease-out); }
.network-details>summary { border-radius:11px; transition:color var(--nt-motion-fast,120ms) var(--nt-motion-ease,ease-out),background-color var(--nt-motion-fast,120ms) var(--nt-motion-ease,ease-out); }
.network-details>summary:hover,.network-details>summary:focus-visible { color:var(--ui-text-primary); background:var(--ui-bg-soft); }
.network-details>summary:hover .network-detail-symbol,.network-details>summary:focus-visible .network-detail-symbol { color:var(--ui-text-primary); }
.network-details[open]>.network-details-content { animation:network-details-in var(--nt-motion-enter,180ms) var(--nt-motion-ease,ease-out); }
.network-service-detail summary>svg,.network-scope summary>svg { flex:none; transition:transform var(--nt-motion-fast,120ms) var(--nt-motion-ease,ease-out); }
.network-service-detail summary:hover,.network-scope summary:hover { color:var(--ui-text-primary); }
.nt-claude-network a { transition:color var(--nt-motion-fast,120ms) var(--nt-motion-ease,ease-out); }
.nt-claude-network a:hover { color:var(--ui-accent); }
.nt-claude-network a>svg { flex:none; transition:transform var(--nt-motion-fast,120ms) var(--nt-motion-ease,ease-out); }
.nt-claude-network a:hover>svg,.nt-claude-network a:focus-visible>svg { transform:translate(1px,-1px); }
.network-import { transition:color var(--nt-motion-fast,120ms) var(--nt-motion-ease,ease-out),border-color var(--nt-motion-fast,120ms) var(--nt-motion-ease,ease-out),background-color var(--nt-motion-fast,120ms) var(--nt-motion-ease,ease-out); }
.network-import:hover { color:var(--ui-text-primary); border-color:var(--ui-border-strong); background:var(--ui-bg-soft); }
.network-rtc .network-panel-heading { margin-bottom:16px; }
@keyframes network-pending { 50% { opacity:.45; } }
@keyframes network-details-in { from { opacity:.5; transform:translateY(4px); } to { opacity:1; transform:none; } }

@media(max-width:900px) { .network-primary { grid-template-columns:1fr; } }
@media(max-width:640px) {
 .nt-claude-network { padding:24px 20px 48px; }
 .network-heading { align-items:flex-start; flex-direction:column; gap:20px; }
 .network-actions { width:100%; justify-content:space-between; gap:8px; }
 .network-actions .primary { order:-2; flex:1; }
 .network-actions .secondary { order:-1; }
 .network-actions .network-button { padding-inline:12px; }
 .network-heading h1 { font-size:28px; }
 .network-panel { padding:20px; }
 .network-primary { gap:16px; }
 .network-service-list>li { grid-template-columns:minmax(0,1fr) auto; gap:8px 12px; }
 .network-service-state { justify-self:end; max-width:140px; text-align:right; }
 .network-latency { grid-column:2; grid-row:2; }
 .network-service-detail { grid-column:1; grid-row:2; }
 .network-details summary { padding:20px; }
 .network-details-content { padding:0 20px 20px; }
 .network-local-facts>div { grid-template-columns:1fr; gap:4px; }
 .network-local-facts dd { text-align:left; }
}
@media(prefers-reduced-motion:reduce) {
  .nt-claude-network :deep(*) { transition:none !important; animation:none !important; }
  .network-button:active:not(:disabled),
  .network-button:hover:not(:disabled)>svg:not(.network-spin),
  .network-button:focus-visible>svg:not(.network-spin),
  .network-heading:hover .tool-capability svg,
  .network-service-list>li:hover .network-service-icon,
  .nt-claude-network a:hover>svg,
  .nt-claude-network a:focus-visible>svg { transform:none; }
}
</style>
