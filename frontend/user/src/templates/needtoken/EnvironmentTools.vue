<template>
  <div class="nt-claude-network mx-auto w-full max-w-[1200px] px-4 pb-16 pt-11 sm:px-6 sm:pt-14">
    <header class="network-heading">
      <span class="network-eyebrow"><Activity :size="15" aria-hidden="true" />{{ t('needtoken.claudeEnvironment.eyebrow') }}</span>
      <h1>{{ t('needtoken.claudeEnvironment.title') }}</h1>
      <p>{{ t('needtoken.claudeEnvironment.subtitle') }}</p>
      <nav class="network-anchors" :aria-label="t('needtoken.claudeEnvironment.sections')">
        <a href="#network-overview">{{ t('needtoken.claudeEnvironment.overview') }}</a><a href="#network-browser">{{ t('needtoken.claudeEnvironment.browser') }}</a><a href="#network-terminal">{{ t('needtoken.claudeEnvironment.terminal') }}</a><a href="#network-rules">{{ t('needtoken.claudeEnvironment.rules') }}</a>
      </nav>
    </header>

    <section class="network-start" :aria-busy="running">
      <div class="network-start-top">
        <div><h2>{{ t('needtoken.claudeEnvironment.startTitle') }}</h2><p role="status" aria-live="polite">{{ running ? t('needtoken.claudeEnvironment.runningHint') : measuredAt ? t('needtoken.claudeEnvironment.checkedAt', { time: dateTime(measuredAt) }) : t('needtoken.claudeEnvironment.idleHint') }}</p></div>
        <div class="network-actions"><button v-if="running" type="button" class="network-button secondary" @click="cancel"><Square :size="14" />{{ t('needtoken.claudeEnvironment.cancel') }}</button><button type="button" class="network-button primary" :disabled="running" @click="run"><LoaderCircle v-if="running" class="network-spin" :size="17" /><Play v-else :size="16" />{{ t(running ? 'needtoken.claudeEnvironment.running' : measuredAt ? 'needtoken.claudeEnvironment.runAgain' : 'needtoken.claudeEnvironment.run') }}</button></div>
      </div>
      <label class="network-rtc-choice"><input v-model="includeWebRtc" type="checkbox" :disabled="running" /><span>{{ t('needtoken.claudeEnvironment.rtcChoice') }}<small>{{ t('needtoken.claudeEnvironment.rtcChoiceHint') }}</small></span></label>
      <p class="network-disclosure"><ShieldCheck :size="16" aria-hidden="true" /><span>{{ t('needtoken.claudeEnvironment.disclosure') }}</span></p>
      <div v-if="measuredAt" class="network-summary" aria-live="polite"><span><Check :size="13" />{{ t('needtoken.claudeEnvironment.confirmedCount', { count: summary.ok }) }}</span><span class="review"><TriangleAlert :size="13" />{{ t('needtoken.claudeEnvironment.reviewCount', { count: summary.warning }) }}</span><span><CircleHelp :size="13" />{{ t('needtoken.claudeEnvironment.unknownCount', { count: summary.unknown }) }}</span></div>
    </section>

    <div class="network-layout">
      <div class="network-main">
        <section id="network-overview" class="network-panel" :aria-label="t('needtoken.claudeEnvironment.overview')">
          <div class="network-panel-heading"><h2><Globe2 :size="19" />{{ t('needtoken.claudeEnvironment.overview') }}</h2><span class="network-status" :class="`is-${exit?.status || 'idle'}`">{{ statusLabel(exit?.status) }}</span></div>
          <div class="network-exit-ip"><span>{{ t('needtoken.claudeEnvironment.exitIp') }}</span><strong>{{ exit?.ip || t('needtoken.claudeEnvironment.unmeasured') }}</strong><small v-if="exit?.source">{{ t('needtoken.claudeEnvironment.source', { source: exit.source }) }}</small></div>
          <dl class="network-facts"><div><dt>{{ t('needtoken.claudeEnvironment.country') }}</dt><dd>{{ exit?.country || unknown }}</dd></div><div><dt>{{ t('needtoken.claudeEnvironment.location') }}</dt><dd>{{ [exit?.region, exit?.city].filter(Boolean).join(' · ') || unknown }}</dd></div><div><dt>ASN</dt><dd>{{ exit?.asn || unknown }}</dd></div><div><dt>{{ t('needtoken.claudeEnvironment.organization') }}</dt><dd>{{ exit?.organization || unknown }}</dd></div><div><dt>{{ t('needtoken.claudeEnvironment.exitTimezone') }}</dt><dd>{{ exit?.timezone || unknown }}</dd></div><div><dt>{{ t('needtoken.claudeEnvironment.reputation') }}</dt><dd>{{ t('needtoken.claudeEnvironment.reputationUnknown') }}</dd></div></dl>
          <p class="network-note">{{ t(`needtoken.claudeEnvironment.exitDetails.${exit?.detail || 'notChecked'}`) }}</p>
          <p v-if="exit" class="network-timestamp">{{ t('needtoken.claudeEnvironment.observedAt', { time: dateTime(exit.measuredAt) }) }}</p>
        </section>

        <section id="network-browser" class="network-browser-grid" :aria-label="t('needtoken.claudeEnvironment.browser')">
          <article class="network-panel"><div class="network-panel-heading"><h2><Radio :size="19" />{{ t('needtoken.claudeEnvironment.anthropic') }}</h2><span class="network-status" :class="`is-${anthropic?.status || 'idle'}`">{{ statusLabel(anthropic?.status) }}</span></div><p class="network-target" translate="no">api.anthropic.com</p><p class="network-copy">{{ t(`needtoken.claudeEnvironment.connectionDetails.${anthropic?.detail || 'notChecked'}`) }}</p><div v-if="anthropic?.latencyMs !== undefined" class="network-metric">{{ anthropic.latencyMs }}<small>ms</small></div><p v-if="anthropic?.httpStatus" class="network-note">HTTP {{ anthropic.httpStatus }}</p><p class="network-note">{{ t('needtoken.claudeEnvironment.browserScope') }}</p></article>
          <article class="network-panel"><div class="network-panel-heading"><h2><Waypoints :size="19" />WebRTC</h2><span class="network-status" :class="`is-${rtc?.status || 'idle'}`">{{ statusLabel(rtc?.status) }}</span></div><p class="network-copy">{{ t(`needtoken.claudeEnvironment.rtcDetails.${rtc?.detail || 'notChecked'}`) }}</p><div v-if="rtc?.publicAddresses.length" class="network-addresses"><code v-for="address in rtc.publicAddresses" :key="address">{{ address }}</code></div><p v-if="rtc && rtc.detail !== 'notRequested'" class="network-note">{{ t('needtoken.claudeEnvironment.rtcCounts', { local: rtc.localCount, mdns: rtc.mdnsCount }) }}</p><p class="network-note">{{ t('needtoken.claudeEnvironment.rtcScope') }}</p></article>
          <article class="network-panel network-local"><div class="network-panel-heading"><h2><Monitor :size="19" />{{ t('needtoken.claudeEnvironment.localEnvironment') }}</h2><span v-if="timeMismatch" class="network-status is-warning">{{ t('needtoken.claudeEnvironment.compareReview') }}</span></div><dl class="network-facts"><div><dt>{{ t('needtoken.claudeEnvironment.browserTimezone') }}</dt><dd>{{ localTimezone || unknown }}</dd></div><div><dt>{{ t('needtoken.claudeEnvironment.browserLanguage') }}</dt><dd>{{ localLanguage || unknown }}</dd></div></dl><p class="network-note">{{ timeMismatch ? t('needtoken.claudeEnvironment.timeMismatch') : t('needtoken.claudeEnvironment.localScope') }}</p></article>
        </section>

        <section id="network-terminal" class="network-panel" :aria-label="t('needtoken.claudeEnvironment.terminal')">
          <div class="network-panel-heading"><h2><Terminal :size="19" />{{ t('needtoken.claudeEnvironment.terminal') }}</h2><span class="network-status" :class="`is-${cliStatus || 'idle'}`">{{ cli ? statusLabel(cliStatus) : t('needtoken.claudeEnvironment.notImported') }}</span></div>
          <p class="network-copy">{{ t('needtoken.claudeEnvironment.cliIntro') }}</p>
          <ol class="network-cli-steps"><li>{{ t('needtoken.claudeEnvironment.cliStep1') }} <a href="/needtoken-claude-network-check.mjs" download>{{ t('needtoken.claudeEnvironment.downloadScript') }}<Download :size="13" /></a><a href="/needtoken-claude-network-check.mjs" target="_blank" rel="noopener noreferrer">{{ t('needtoken.claudeEnvironment.reviewScript') }}<ArrowUpRight :size="13" /></a></li><li>{{ t('needtoken.claudeEnvironment.cliStep2') }}<code class="network-command">node needtoken-claude-network-check.mjs &gt; claude-network-report.json</code></li><li>{{ t('needtoken.claudeEnvironment.cliStep3') }}</li></ol>
          <label class="network-import"><Upload :size="16" />{{ t('needtoken.claudeEnvironment.importReport') }}<input type="file" accept="application/json,.json" @change="importCliReport" /></label><p v-if="importMessage" class="network-note" role="status">{{ t(`needtoken.claudeEnvironment.${importMessage}`) }}</p>
          <template v-if="cli"><p class="network-import-label">{{ t('needtoken.claudeEnvironment.importedAt', { time: dateTime(cli.measured_at) }) }}</p><dl class="network-facts"><div><dt>{{ t('needtoken.claudeEnvironment.cliDns') }}</dt><dd>{{ t(`needtoken.claudeEnvironment.cliStates.${cli.dns.status}`) }}</dd></div><div><dt>{{ t('needtoken.claudeEnvironment.cliCurl') }}</dt><dd>{{ t(`needtoken.claudeEnvironment.cliStates.${cli.curl.status}`) }}<span v-if="cli.curl.http_status"> · HTTP {{ cli.curl.http_status }}</span><span v-if="cli.curl.exit_code >= 0"> · {{ t('needtoken.claudeEnvironment.curlExit', { code: cli.curl.exit_code }) }}</span></dd></div><div><dt>{{ t('needtoken.claudeEnvironment.cliTls') }}</dt><dd>{{ cli.curl.ssl_verify_result === 0 && cli.curl.http_status ? t('needtoken.claudeEnvironment.cliTlsVerified') : cli.curl.ssl_verify_result ? t('needtoken.claudeEnvironment.tlsErrorCode', { code: cli.curl.ssl_verify_result }) : unknown }}</dd></div><div><dt>{{ t('needtoken.claudeEnvironment.cliProxy') }}</dt><dd>{{ t(hasCliProxy ? 'needtoken.claudeEnvironment.proxyPresent' : 'needtoken.claudeEnvironment.proxyAbsent') }}</dd></div></dl><p v-if="cli.curl.status === 'error' && [5, 6, 7, 28, 35, 60].includes(cli.curl.exit_code)" class="network-note">{{ t(`needtoken.claudeEnvironment.curlErrors.${cli.curl.exit_code}`) }}</p><p v-if="cli.proxy.custom_base_url" class="network-review-note"><Info :size="15" />{{ t('needtoken.claudeEnvironment.customBaseUrl') }}</p></template>
          <p class="network-note">{{ t('needtoken.claudeEnvironment.cliScope') }}</p>
        </section>

        <section class="network-panel" :aria-label="t('needtoken.claudeEnvironment.reportTitle')"><div class="network-panel-heading"><h2><FileJson :size="18" />{{ t('needtoken.claudeEnvironment.reportTitle') }}</h2><button type="button" class="network-button secondary" :disabled="(!measuredAt && !cli) || running" @click="copyReport"><Copy :size="14" />{{ t('needtoken.claudeEnvironment.copyReport') }}</button></div><p class="network-note">{{ t('needtoken.claudeEnvironment.reportHint') }}</p><p v-if="copyMessage" class="network-note" role="status">{{ t(`needtoken.claudeEnvironment.${copyMessage}`) }}</p><details v-if="measuredAt || cli"><summary>{{ t('needtoken.claudeEnvironment.viewReport') }}</summary><textarea :value="report" readonly :aria-label="t('needtoken.claudeEnvironment.reportTitle')" /></details></section>
      </div>

      <aside id="network-rules" class="network-rules" :aria-label="t('needtoken.claudeEnvironment.rules')"><section class="network-panel"><div class="network-panel-heading"><h2><ShieldCheck :size="19" />{{ t('needtoken.claudeEnvironment.rules') }}</h2></div><p class="network-copy">{{ t('needtoken.claudeEnvironment.rulesIntro') }}</p><ul class="network-rule-list"><li v-for="key in ['region','account','consistency','limits']" :key="key"><span><Check :size="13" /></span><p>{{ t(`needtoken.claudeEnvironment.staticRules.${key}`) }}</p></li></ul><a class="network-link" href="https://www.anthropic.com/supported-countries" target="_blank" rel="noopener noreferrer">{{ t('needtoken.claudeEnvironment.officialRegions') }}<ArrowUpRight :size="13" /></a><a class="network-link" href="https://www.anthropic.com/legal/aup" target="_blank" rel="noopener noreferrer">{{ t('needtoken.claudeEnvironment.officialPolicy') }}<ArrowUpRight :size="13" /></a></section>
        <section class="network-panel network-operator"><h2>{{ t('needtoken.claudeEnvironment.operatorUpdates') }}</h2><p v-if="rulesLoading" class="network-note" role="status">{{ t('needtoken.claudeEnvironment.rulesLoading') }}</p><div v-else-if="rulesFailed" class="network-empty"><p>{{ t('needtoken.claudeEnvironment.rulesFailed') }}</p><button type="button" @click="loadRules">{{ t('needtoken.common.retry') }}</button></div><p v-else-if="!rules.length" class="network-note">{{ t('needtoken.claudeEnvironment.rulesEmpty') }}</p><article v-for="rule in rules" v-else :key="rule.id" class="network-rule-article"><h3>{{ localized(rule.title) }}</h3><p>{{ plain(localized(rule.summary)) }}</p><RouterLink :to="`/blog/${encodeURIComponent(rule.slug)}`">{{ t('needtoken.claudeEnvironment.readUpdate') }}<ArrowUpRight :size="12" /></RouterLink></article></section>
        <p class="network-reference">{{ t('needtoken.claudeEnvironment.reference') }} <a href="https://github.com/LinXiaoTao/FuckClaude" target="_blank" rel="noopener noreferrer">LinXiaoTao/FuckClaude</a></p>
      </aside>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { Activity, ArrowUpRight, Check, CircleHelp, Copy, Download, FileJson, Globe2, Info, LoaderCircle, Monitor, Play, Radio, ShieldCheck, Square, Terminal, TriangleAlert, Upload, Waypoints } from 'lucide-vue-next'
import { postAPI } from '../../api'
import { usePageSeo } from '../../composables/usePageSeo'
import { useAppStore } from '../../stores/app'
import { getLocalizedText } from '../../utils/resellerSiteConfig'
import { serviceDescriptionText } from './lib/serviceCatalog'
import { classifyWebRtc, claudeNetworkReport, claudeRuleCategoryIds, lookupExitNetwork, parseCliReport, probeAnthropic, probeWebRtc, timezoneDiffers } from './lib/claudeNetwork'
import type { AnthropicConnection, CliReport, ExitNetwork, NetworkStatus, WebRtcObservation } from './lib/claudeNetwork'

const { t, locale } = useI18n()
const appStore = useAppStore()
usePageSeo({ title: () => t('needtoken.claudeEnvironment.title'), description: () => t('needtoken.claudeEnvironment.subtitle') })
const unknown = computed(() => t('needtoken.claudeEnvironment.unknown'))
const localized = (value?: Record<string, unknown>) => getLocalizedText(value, appStore.locale)
const plain = serviceDescriptionText
const exit = ref<ExitNetwork | null>(null), anthropic = ref<AnthropicConnection | null>(null), rtc = ref<WebRtcObservation | null>(null), cli = ref<CliReport | null>(null)
const running = ref(false), measuredAt = ref(''), localTimezone = ref(''), localLanguage = ref(''), includeWebRtc = ref(false), copyMessage = ref(''), importMessage = ref('')
const rulesLoading = ref(true), rulesFailed = ref(false), rules = ref<Array<{ id: number; slug: string; title: Record<string, unknown>; summary: Record<string, unknown> }>>([])
let active: AbortController | null = null, generation = 0, importGeneration = 0, rulesGeneration = 0, disposed = false
const dateTime = (value: string) => Number.isFinite(Date.parse(value)) ? new Date(value).toLocaleString(locale.value) : unknown.value
const statusLabel = (status?: NetworkStatus) => t(`needtoken.claudeEnvironment.status.${status || 'idle'}`)
const timeMismatch = computed(() => timezoneDiffers(localTimezone.value, exit.value))
const hasCliProxy = computed(() => !!cli.value && (cli.value.proxy.http || cli.value.proxy.https || cli.value.proxy.all))
const cliStatus = computed<NetworkStatus | undefined>(() => !cli.value ? undefined : cli.value.curl.status === 'unsupported' ? 'unknown' : cli.value.curl.status === 'ok' && ![403, 429].includes(cli.value.curl.http_status || 0) && (cli.value.curl.http_status || 0) < 500 ? 'ok' : 'warning')
const summary = computed(() => {
  const rows = [exit.value, anthropic.value, ...(rtc.value?.detail !== 'notRequested' ? [rtc.value] : [])].filter(Boolean)
  return { ok: rows.filter(row => row?.status === 'ok').length, warning: rows.filter(row => row?.status === 'warning').length, unknown: rows.filter(row => row?.status === 'unknown').length }
})
const report = computed(() => claudeNetworkReport({ exit: exit.value, anthropic: anthropic.value, rtc: rtc.value, cli: cli.value, measuredAt: measuredAt.value }))

async function run() {
  if (running.value) return
  const current = ++generation
  const controller = new AbortController(); active = controller
  running.value = true; measuredAt.value = ''; exit.value = null; anthropic.value = null; rtc.value = null; copyMessage.value = ''
  localTimezone.value = Intl.DateTimeFormat().resolvedOptions().timeZone || ''
  localLanguage.value = navigator.languages?.join(', ') || navigator.language || ''
  const exitTask = lookupExitNetwork({ signal: controller.signal }).then(value => { if (!disposed && generation === current) exit.value = value; return value })
  const connectionTask = probeAnthropic({ signal: controller.signal }).then(value => { if (!disposed && generation === current) anthropic.value = value })
  const rtcTask = includeWebRtc.value ? probeWebRtc({ signal: controller.signal }) : null
  const results = await Promise.allSettled([exitTask, connectionTask, rtcTask] as const)
  if (disposed || generation !== current) return
  const rtcResult = results[2]
  const expectedIp = results[0].status === 'fulfilled' ? results[0].value.ip : undefined
  if (!rtcTask) rtc.value = { status: 'unknown', detail: 'notRequested', publicAddresses: [], localCount: 0, mdnsCount: 0 }
  else if (rtcResult?.status === 'fulfilled' && rtcResult.value && 'supported' in rtcResult.value) rtc.value = rtcResult.value.supported ? classifyWebRtc(rtcResult.value.addresses, rtcResult.value.mdnsCount, expectedIp) : { status: 'unknown', detail: 'unsupported', publicAddresses: [], localCount: 0, mdnsCount: 0 }
  measuredAt.value = new Date().toISOString(); running.value = false; active = null
}
function cancel() { generation++; active?.abort(); active = null; running.value = false; measuredAt.value = '' }
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
async function loadRules() {
  const current = ++rulesGeneration; rulesLoading.value = true; rulesFailed.value = false
  try {
    const categoryResponse = await postAPI.categories()
    if (disposed || current !== rulesGeneration) return
    if (!Array.isArray(categoryResponse.data.data)) throw new Error('invalid public categories')
    const allowed = new Set(claudeRuleCategoryIds(categoryResponse.data.data))
    if (!allowed.size) { rules.value = []; return }
    const response = await postAPI.list({ type: 'blog', category_slug: 'claude-code-rules', page: 1, page_size: 5 })
    if (disposed || current !== rulesGeneration) return
    rules.value = Array.isArray(response.data.data) ? response.data.data.filter((row: any) => row && typeof row.slug === 'string' && row.type === 'blog' && allowed.has(Number(row.category_id))).slice(0, 5) : []
  } catch { if (!disposed && current === rulesGeneration) rulesFailed.value = true }
  finally { if (!disposed && current === rulesGeneration) rulesLoading.value = false }
}
onMounted(loadRules)
onBeforeUnmount(() => { disposed = true; generation++; importGeneration++; rulesGeneration++; active?.abort() })
</script>

<style scoped>
.network-heading { max-width: 820px; margin-bottom: 30px; }
.network-eyebrow { display: inline-flex; align-items: center; gap: 8px; color: var(--ui-accent); font-size: 12px; letter-spacing: .06em; }
.network-heading h1 { margin: 15px 0; font-size: clamp(29px, 4.2vw, 46px); font-weight: 750; letter-spacing: -.04em; line-height: 1.25; }
.network-heading > p { color: var(--ui-text-secondary); font-size: 14px; line-height: 1.9; }
.network-anchors { display: flex; flex-wrap: wrap; gap: 9px; margin-top: 22px; }
.network-anchors a { padding: 7px 12px; border: 1px solid var(--ui-border); border-radius: 999px; color: var(--ui-text-muted); font-size: 12px; }
.network-anchors a:hover { color: var(--ui-accent); border-color: #2dd4a745; }
.network-start { border: 1px solid #2dd4a733; border-radius: 20px; background: linear-gradient(115deg, #2dd4a709, transparent), var(--ui-bg-elevated); margin-bottom: 28px; padding: 23px; }
.network-start-top { display: flex; flex-wrap: wrap; gap: 20px; align-items: center; justify-content: space-between; }
.network-start h2 { font-size: 17px; font-weight: 650; }
.network-start-top p { margin-top: 6px; font-size: 12px; color: var(--ui-text-muted); line-height: 1.8; }
.network-actions { display: flex; flex-wrap: wrap; gap: 9px; }
.network-button { display: inline-flex; align-items: center; justify-content: center; gap: 8px; min-height: 40px; padding: 9px 14px; border: 1px solid transparent; border-radius: 10px; font-size: 12px; font-weight: 650; }
.network-button.primary { background: var(--ui-accent); color: var(--ui-text-on-accent); }
.network-button.secondary { background: var(--ui-bg-soft); border-color: var(--ui-border); color: var(--ui-text-secondary); }
.network-button:disabled { cursor: default; opacity: .5; }
.network-rtc-choice { display: flex; gap: 10px; align-items: flex-start; margin-top: 20px; padding-top: 17px; border-top: 1px solid var(--ui-border); font-size: 12px; color: var(--ui-text-secondary); }
.network-rtc-choice input { margin-top: 4px; accent-color: var(--ui-accent); }
.network-rtc-choice small { display: block; margin-top: 4px; font-size: 11px; color: var(--ui-text-muted); line-height: 1.7; }
.network-disclosure { display: flex; align-items: flex-start; gap: 8px; margin-top: 14px; color: var(--ui-text-muted); font-size: 11px; line-height: 1.8; }
.network-disclosure svg { flex: none; margin-top: 3px; color: var(--ui-accent); }
.network-summary { display: flex; flex-wrap: wrap; gap: 12px 22px; margin-top: 19px; font-size: 11px; }
.network-summary > span { display: inline-flex; align-items: center; gap: 6px; color: var(--ui-text-secondary); }
.network-summary .review { color: #e5b969; }
.network-layout { display: grid; grid-template-columns: minmax(0, 1fr) 290px; gap: 24px; align-items: start; }
.network-main { display: grid; gap: 22px; min-width: 0; }
.network-panel { min-width: 0; padding: 23px; border: 1px solid var(--ui-border); border-radius: 18px; background: color-mix(in srgb, var(--ui-bg-elevated) 88%, transparent); scroll-margin-top: 88px; }
.network-panel-heading { display: flex; flex-wrap: wrap; align-items: center; justify-content: space-between; gap: 12px; margin-bottom: 17px; }
.network-panel h2 { display: flex; align-items: center; gap: 9px; font-size: 16px; font-weight: 650; }
.network-panel h2 > svg { flex: none; color: var(--ui-accent); }
.network-status { display: inline-flex; flex: none; padding: 4px 8px; border: 1px solid var(--ui-border); border-radius: 999px; font-size: 10px; color: var(--ui-text-muted); }
.network-status.is-ok { color: var(--ui-accent); border-color: #2dd4a733; background: #2dd4a709; }
.network-status.is-warning { color: #e5b969; border-color: #e5b96940; background: #e5b9690a; }
.network-exit-ip { display: flex; flex-wrap: wrap; align-items: baseline; gap: 9px 15px; margin-bottom: 24px; }
.network-exit-ip > span { flex-basis: 100%; color: var(--ui-text-muted); font-size: 11px; }
.network-exit-ip > strong { font-family: var(--nt-font-mono); font-size: clamp(21px, 3vw, 30px); color: var(--ui-accent); overflow-wrap: anywhere; }
.network-exit-ip small { color: var(--ui-text-muted); font-size: 10px; }
.network-facts { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 18px; }
.network-facts dt { color: var(--ui-text-muted); font-size: 11px; }
.network-facts dd { margin-top: 5px; color: var(--ui-text-secondary); font-size: 12px; line-height: 1.65; overflow-wrap: anywhere; }
.network-note { margin-top: 15px; color: var(--ui-text-muted); font-size: 11px; line-height: 1.9; overflow-wrap: anywhere; }
.network-timestamp { margin-top: 12px; color: var(--ui-text-muted); font-size: 10px; }
.network-browser-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 20px; scroll-margin-top: 88px; }
.network-copy { color: var(--ui-text-secondary); font-size: 12px; line-height: 1.9; }
.network-target { margin: 0 0 12px; font-family: var(--nt-font-mono); font-size: 13px; color: var(--ui-accent); }
.network-metric { margin-top: 15px; font-family: var(--nt-font-mono); font-size: 27px; color: var(--ui-text-primary); }
.network-metric small { font-size: 12px; color: var(--ui-text-muted); margin-left: 5px; }
.network-addresses { display: grid; gap: 6px; margin-top: 15px; }
.network-addresses code { color: var(--ui-text-secondary); font-size: 12px; overflow-wrap: anywhere; }
.network-local { grid-column: 1 / -1; }
.network-cli-steps { display: grid; gap: 14px; list-style: decimal; margin: 18px 0 20px; padding-left: 18px; color: var(--ui-text-secondary); font-size: 12px; line-height: 1.9; }
.network-cli-steps a, .network-link, .network-rule-article a { display: inline-flex; align-items: center; gap: 4px; margin: 5px 9px 0 0; color: var(--ui-accent); font-size: 11px; }
.network-command { display: block; max-width: 100%; margin-top: 9px; padding: 12px; border: 1px solid var(--ui-border); border-radius: 9px; background: var(--ui-bg-page); color: var(--ui-text-primary); font-size: 11px; overflow-wrap: anywhere; white-space: normal; }
.network-import { display: inline-flex; align-items: center; gap: 8px; padding: 10px 13px; border: 1px solid var(--ui-border-strong); border-radius: 10px; font-size: 12px; color: var(--ui-text-secondary); cursor: pointer; }
.network-import input { position: absolute; width: 1px; height: 1px; opacity: 0; }
.network-import:has(input:focus-visible) { outline: 2px solid var(--ui-accent); outline-offset: 4px; }
.network-import-label { margin: 17px 0; padding: 10px 12px; border: 1px solid #e5b96933; border-radius: 8px; color: #e5b969; font-size: 11px; line-height: 1.8; }
.network-review-note { display: flex; gap: 7px; margin-top: 15px; color: #e5b969; font-size: 11px; line-height: 1.8; }
.network-review-note svg { flex: none; margin-top: 3px; }
.network-panel details { margin-top: 17px; }
.network-panel summary { padding: 4px 0; color: var(--ui-accent); font-size: 12px; cursor: pointer; }
.network-panel textarea { width: 100%; height: 190px; resize: vertical; margin-top: 12px; padding: 12px; border: 1px solid var(--ui-border); border-radius: 10px; background: var(--ui-bg-page); color: var(--ui-text-secondary); font-family: var(--nt-font-mono); font-size: 10px; line-height: 1.8; }
.network-rules { display: grid; gap: 20px; min-width: 0; scroll-margin-top: 88px; }
.network-rule-list { display: grid; gap: 18px; margin-top: 22px; }
.network-rule-list > li { display: flex; gap: 9px; align-items: flex-start; color: var(--ui-text-secondary); font-size: 12px; line-height: 1.85; }
.network-rule-list > li > span { display: grid; flex: none; place-items: center; width: 18px; height: 18px; margin-top: 3px; border: 1px solid #2dd4a72b; border-radius: 50%; color: var(--ui-accent); }
.network-operator h2 { margin-bottom: 16px; }
.network-rule-article + .network-rule-article { border-top: 1px solid var(--ui-border); padding-top: 16px; margin-top: 16px; }
.network-rule-article h3 { font-size: 13px; font-weight: 600; overflow-wrap: anywhere; }
.network-rule-article p { margin-top: 7px; color: var(--ui-text-muted); font-size: 11px; line-height: 1.85; overflow-wrap: anywhere; }
.network-reference { color: var(--ui-text-muted); font-size: 10px; line-height: 1.85; }
.network-reference a { color: var(--ui-text-secondary); overflow-wrap: anywhere; }
.network-empty { color: var(--ui-text-muted); font-size: 11px; line-height: 1.8; }
.network-empty button { margin-top: 8px; color: var(--ui-accent); }
.network-spin { animation: network-spin 1s linear infinite; }
@keyframes network-spin { to { transform: rotate(360deg); } }
@media (max-width: 1000px) { .network-layout { grid-template-columns: minmax(0, 1fr); } .network-rules { grid-template-columns: repeat(2, minmax(0, 1fr)); } .network-reference { grid-column: 1 / -1; } }
@media (max-width: 600px) { .network-start, .network-panel { padding: 19px; } .network-browser-grid, .network-rules { grid-template-columns: minmax(0, 1fr); } .network-facts { gap: 15px 12px; } .network-heading > p { font-size: 13px; } .network-panel-heading h2 { font-size: 15px; } .network-start-top { gap: 16px; } .network-actions { width: 100%; } .network-actions > .primary { flex: 1; } }
@media (prefers-reduced-motion: reduce) { .network-spin { animation: none; } }
</style>
