<template>
  <div class="nt-claude-env">
    <ToolNavigation />

    <header class="env-heading">
      <div>
        <h1>{{ t('needtoken.checkCenter.claude.title') }}</h1>
        <p>{{ t('needtoken.checkCenter.claude.subtitle') }}</p>
      </div>
      <div class="env-actions">
        <label class="rtc-toggle">
          <input v-model="includeWebRtc" type="checkbox" :disabled="running" />
          <span>{{ t('needtoken.checkCenter.claude.includeRtc') }}</span>
        </label>
        <button v-if="running" type="button" class="env-btn secondary" @click="cancel">
          <Square :size="14" aria-hidden="true" />
          {{ t('needtoken.claudeEnvironment.cancel') }}
        </button>
        <button type="button" class="env-btn primary" :disabled="running" :aria-busy="running" @click="run">
          <LoaderCircle v-if="running" class="env-spin" :size="15" aria-hidden="true" />
          <RefreshCw v-else :size="15" aria-hidden="true" />
          {{ t(running ? 'needtoken.claudeEnvironment.running' : 'needtoken.checkCenter.claude.recheck') }}
        </button>
      </div>
    </header>

    <div class="env-grid">
      <!-- Left: 10-Signal Local China Environment Fingerprint Score Card -->
      <section class="env-card env-fingerprint-card" aria-labelledby="fp-score-heading">
        <div class="card-head">
          <div>
            <h2 id="fp-score-heading">{{ t('needtoken.checkCenter.claude.scoreTitle') }}</h2>
            <p>{{ t('needtoken.checkCenter.claude.scoreSub') }}</p>
          </div>
          <span class="triggered-pill" :data-band="fingerprint.band">
            {{ t('needtoken.checkCenter.claude.triggeredSummary', { count: fingerprint.triggeredCount }) }}
          </span>
        </div>

        <div class="score-hero">
          <div class="score-gauge" :data-band="fingerprint.band" aria-hidden="true">
            <svg viewBox="0 0 120 120">
              <circle class="gauge-track" cx="60" cy="60" r="50" />
              <circle
                class="gauge-fill"
                cx="60"
                cy="60"
                r="50"
                :style="{
                  strokeDasharray: `${RING_CIRCUMFERENCE}px`,
                  strokeDashoffset: `${RING_CIRCUMFERENCE * (1 - fingerprint.totalScore / 100)}px`,
                }"
              />
            </svg>
            <div class="gauge-center">
              <strong>{{ fingerprint.totalScore }}</strong>
              <span>/ 100</span>
            </div>
          </div>

          <div class="score-summary">
            <span class="band-badge" :data-band="fingerprint.band">
              <ShieldCheck v-if="fingerprint.band === 'low'" :size="14" aria-hidden="true" />
              <AlertTriangle v-else-if="fingerprint.band === 'medium'" :size="14" aria-hidden="true" />
              <ShieldAlert v-else :size="14" aria-hidden="true" />
              {{ t(`needtoken.checkCenter.claude.band.${fingerprint.band}`) }}
            </span>
            <p class="band-desc">{{ t(`needtoken.checkCenter.claude.bandDesc.${fingerprint.band}`) }}</p>
          </div>
        </div>

        <ul class="signal-list" :aria-label="t('needtoken.checkCenter.claude.scoreTitle')">
          <li
            v-for="sig in fingerprint.signals"
            :key="sig.id"
            class="signal-row"
            :data-status="sig.status"
          >
            <div class="signal-main">
              <span class="signal-dot" aria-hidden="true" />
              <div class="signal-text">
                <div class="signal-title-line">
                  <strong>{{ t(`needtoken.checkCenter.claude.signals.${sig.id}`) }}</strong>
                  <span v-if="sig.claudeUsed" class="claude-used-tag">
                    {{ t('needtoken.checkCenter.claude.claudeUsedBadge') }}
                  </span>
                </div>
                <span class="signal-raw" translate="no">{{ formatSignalRaw(sig) }}</span>
              </div>
            </div>
            <span class="signal-pts" :data-status="sig.status">
              <template v-if="sig.points > 0">
                {{ t('needtoken.checkCenter.claude.weightPts', { points: sig.points, weight: sig.weight }) }}
              </template>
              <template v-else-if="sig.status === 'unknown'">
                {{ t('needtoken.checkCenter.claude.notChecked') }}
              </template>
              <template v-else>
                {{ t('needtoken.checkCenter.claude.safeTag') }}
              </template>
            </span>
          </li>
        </ul>

        <div class="fixes-box">
          <h3>{{ t('needtoken.checkCenter.claude.fixesTitle') }}</h3>
          <ul v-if="priorityFixes.length" class="fixes-list">
            <li v-for="sig in priorityFixes" :key="sig.id">
              <strong>{{ t(`needtoken.checkCenter.claude.signals.${sig.id}`) }}</strong>
              <span>{{ t(`needtoken.checkCenter.claude.fixes.${sig.id}`) }}</span>
            </li>
          </ul>
          <p v-else class="fixes-clean">
            <CheckCircle2 :size="15" aria-hidden="true" />
            <span>{{ t('needtoken.checkCenter.claude.fixesClean') }}</span>
          </p>
        </div>
      </section>

      <!-- Right: Independent Exit Network + Anthropic Connectivity -->
      <div class="env-side-col">
        <section class="env-card" data-network-check="exit" :aria-busy="pending.has('exit')">
          <div class="card-head">
            <h2><Network :size="16" aria-hidden="true" />{{ t('needtoken.checkCenter.claude.exitCardTitle') }}</h2>
            <span class="region-badge" :data-region="regionSupport">
              {{ t(`needtoken.checkCenter.claude.region.${regionSupport}`) }}
            </span>
          </div>

          <p class="exit-ip" :class="{ 'is-ipv6': (exit?.ip || '').includes(':') }" translate="no">
            <span v-if="pending.has('exit')" class="skeleton-line ip-skeleton" aria-hidden="true" />
            <template v-else>{{ exit?.ip || '—' }}</template>
          </p>

          <dl class="exit-facts">
            <div>
              <dt>{{ t('needtoken.claudeEnvironment.country') }}</dt>
              <dd>
                <span v-if="pending.has('exit')" class="skeleton-line" aria-hidden="true" />
                <template v-else>{{ exit?.country ? `${exit.country}${exit.countryCode ? ` (${exit.countryCode})` : ''}` : unknown }}</template>
              </dd>
            </div>
            <div>
              <dt>{{ t('needtoken.claudeEnvironment.location') }}</dt>
              <dd>
                <span v-if="pending.has('exit')" class="skeleton-line" aria-hidden="true" />
                <template v-else>{{ [exit?.region, exit?.city].filter(Boolean).join(' · ') || unknown }}</template>
              </dd>
            </div>
            <div>
              <dt translate="no">ASN</dt>
              <dd>
                <span v-if="pending.has('exit')" class="skeleton-line" aria-hidden="true" />
                <template v-else>{{ exit?.asn || unknown }}</template>
              </dd>
            </div>
            <div>
              <dt>{{ t('needtoken.claudeEnvironment.organization') }}</dt>
              <dd>
                <span v-if="pending.has('exit')" class="skeleton-line" aria-hidden="true" />
                <template v-else>{{ exit?.organization || unknown }}</template>
              </dd>
            </div>
          </dl>

          <div class="exit-checks">
            <div v-if="exit?.timezone && browserTimezone" class="exit-check-pill" :data-tone="timeMismatch ? 'warn' : 'ok'">
              <AlertTriangle v-if="timeMismatch" :size="14" aria-hidden="true" />
              <CheckCircle2 v-else :size="14" aria-hidden="true" />
              <span>
                {{ timeMismatch
                  ? t('needtoken.checkCenter.claude.timezoneMismatch', { browser: browserTimezone, exit: exit.timezone })
                  : t('needtoken.checkCenter.claude.timezoneMatch') }}
              </span>
            </div>

            <div v-if="rtcIncluded && rtc" class="exit-check-pill" :data-tone="rtc.status === 'warning' ? 'risk' : rtc.status === 'ok' ? 'ok' : 'muted'">
              <ShieldAlert v-if="rtc.status === 'warning'" :size="14" aria-hidden="true" />
              <CheckCircle2 v-else-if="rtc.status === 'ok'" :size="14" aria-hidden="true" />
              <Info v-else :size="14" aria-hidden="true" />
              <span>
                {{ rtc.status === 'warning'
                  ? t('needtoken.checkCenter.claude.webrtcLeak')
                  : rtc.status === 'ok' || rtc.detail === 'localOnly' || rtc.detail === 'noCandidates'
                    ? t('needtoken.checkCenter.claude.webrtcClean')
                    : t(`needtoken.environmentV6.rtcStates.${rtc.detail}`) }}
              </span>
            </div>

            <div v-if="rtc?.publicAddresses.length" class="rtc-ips">
              <code v-for="address in rtc.publicAddresses" :key="address">{{ address }}</code>
            </div>
          </div>

          <p v-if="exit?.detail === 'lookupUnavailable'" class="card-note">{{ t('needtoken.environmentV6.exitUnavailable') }}</p>
          <p v-else-if="exit?.detail === 'ipOnly'" class="card-note">{{ t('needtoken.environmentV6.exitPartial') }}</p>
        </section>

        <section class="env-card" :aria-label="t('needtoken.checkCenter.claude.connectCardTitle')" :aria-busy="running">
          <div class="card-head">
            <h2><Globe2 :size="16" aria-hidden="true" />{{ t('needtoken.checkCenter.claude.connectCardTitle') }}</h2>
            <a href="https://status.claude.com/" target="_blank" rel="noopener noreferrer" class="status-ext-link">
              {{ t('needtoken.toolsPolish.providerStatus') }}
              <ArrowUpRight :size="12" aria-hidden="true" />
            </a>
          </div>

          <ul class="service-list">
            <li v-for="id in CLAUDE_SERVICES" :key="id" :data-network-check="id">
              <div class="service-name">
                <Code2 v-if="id === 'anthropic'" :size="15" aria-hidden="true" />
                <Globe2 v-else :size="15" aria-hidden="true" />
                <div>
                  <strong>{{ t(`needtoken.environmentV6.targets.${id}`) }}</strong>
                  <span translate="no">{{ serviceHost(id) }}</span>
                </div>
              </div>
              <span class="service-state" :class="serviceClass(id)">
                <LoaderCircle v-if="pending.has(id)" :size="13" class="env-spin" aria-hidden="true" />
                <i v-else aria-hidden="true" />
                {{ serviceLabel(id) }}
              </span>
              <span class="service-latency" translate="no">
                {{ services[id]?.latencyMs !== undefined ? `${services[id]!.latencyMs} ms` : '—' }}
              </span>
            </li>
          </ul>
          <p class="card-note">{{ t('needtoken.environmentV6.browserScope') }}</p>
        </section>
      </div>
    </div>

    <!-- Collapsible CLI Verification & Redacted Report -->
    <div class="env-advanced">
      <details id="network-terminal" class="adv-details">
        <summary>
          <Terminal :size="15" aria-hidden="true" />
          <span>{{ t('needtoken.environmentV6.terminalTitle') }}</span>
          <span class="adv-meta">{{ cli ? t(`needtoken.claudeEnvironment.cliStates.${cli.curl.status}`) : t('needtoken.claudeEnvironment.notImported') }}</span>
          <ChevronDown :size="15" aria-hidden="true" />
        </summary>
        <div class="adv-content">
          <p class="adv-copy">{{ t('needtoken.environmentV6.terminalIntro') }}</p>
          <ol class="cli-steps">
            <li>
              {{ t('needtoken.claudeEnvironment.cliStep1') }}
              <a href="/needtoken-claude-network-check.mjs" download>{{ t('needtoken.claudeEnvironment.downloadScript') }}<Download :size="12" /></a>
              <a href="/needtoken-claude-network-check.mjs" target="_blank" rel="noopener noreferrer">{{ t('needtoken.claudeEnvironment.reviewScript') }}<ArrowUpRight :size="12" /></a>
            </li>
            <li>
              {{ t('needtoken.claudeEnvironment.cliStep2') }}
              <code class="cli-cmd">node needtoken-claude-network-check.mjs &gt; claude-network-report.json</code>
            </li>
            <li>{{ t('needtoken.claudeEnvironment.cliStep3') }}</li>
          </ol>
          <label class="cli-import">
            <Upload :size="15" />
            <span>{{ t('needtoken.claudeEnvironment.importReport') }}</span>
            <input type="file" accept="application/json,.json" @change="importCliReport" />
          </label>
          <p v-if="importMessage" class="card-note" role="status">{{ t(`needtoken.claudeEnvironment.${importMessage}`) }}</p>
          <template v-if="cli">
            <dl class="exit-facts cli-facts">
              <div><dt>{{ t('needtoken.claudeEnvironment.cliDns') }}</dt><dd>{{ t(`needtoken.claudeEnvironment.cliStates.${cli.dns.status}`) }}</dd></div>
              <div><dt>{{ t('needtoken.claudeEnvironment.cliCurl') }}</dt><dd>{{ t(`needtoken.claudeEnvironment.cliStates.${cli.curl.status}`) }}<span v-if="cli.curl.http_status"> · HTTP {{ cli.curl.http_status }}</span></dd></div>
              <div><dt>{{ t('needtoken.claudeEnvironment.cliTls') }}</dt><dd>{{ cli.curl.ssl_verify_result === 0 && cli.curl.http_status ? t('needtoken.claudeEnvironment.cliTlsVerified') : unknown }}</dd></div>
              <div><dt>{{ t('needtoken.claudeEnvironment.cliProxy') }}</dt><dd>{{ t(hasCliProxy ? 'needtoken.claudeEnvironment.proxyPresent' : 'needtoken.claudeEnvironment.proxyAbsent') }}</dd></div>
            </dl>
          </template>
        </div>
      </details>

      <details v-if="measuredAt || cli" class="adv-details">
        <summary>
          <FileJson :size="15" aria-hidden="true" />
          <span>{{ t('needtoken.claudeEnvironment.reportTitle') }}</span>
          <ChevronDown :size="15" aria-hidden="true" />
        </summary>
        <div class="adv-content">
          <p class="adv-copy">{{ t('needtoken.claudeEnvironment.reportHint') }}</p>
          <button type="button" class="env-btn secondary" :disabled="running" @click="copyReport">
            <Copy :size="14" aria-hidden="true" />
            {{ t('needtoken.claudeEnvironment.copyReport') }}
          </button>
          <p v-if="copyMessage" class="card-note" role="status">{{ t(`needtoken.claudeEnvironment.${copyMessage}`) }}</p>
          <textarea class="report-textarea" :value="report" readonly :aria-label="t('needtoken.claudeEnvironment.reportTitle')" />
        </div>
      </details>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  AlertTriangle,
  ArrowUpRight,
  CheckCircle2,
  ChevronDown,
  Code2,
  Copy,
  Download,
  FileJson,
  Globe2,
  Info,
  LoaderCircle,
  Network,
  RefreshCw,
  ShieldAlert,
  ShieldCheck,
  Square,
  Terminal,
  Upload,
} from 'lucide-vue-next'
import ToolNavigation from './components/ToolNavigation.vue'
import { usePageSeo } from '../../composables/usePageSeo'
import {
  CLAUDE_PROBES,
  CLAUDE_SERVICES,
  claudeNetworkReport,
  collectClaudeLocalChecks,
  parseCliReport,
  runClaudeNetworkChecks,
  timezoneDiffers,
  type AnthropicConnection,
  type ClaudeLocalCheck,
  type ClaudeServiceId,
  type CliReport,
  type ExitNetwork,
  type NetworkCheckId,
  type WebRtcObservation,
} from './lib/claudeNetwork'
import {
  classifyAnthropicRegion,
  collectBrowserFingerprintInput,
  evaluateChinaFingerprint,
  type FingerprintInput,
  type FingerprintReport,
  type FingerprintSignalResult,
} from './lib/claudeFingerprint'
import type { ConnectionDiagnostic } from './lib/browserDiagnostics'

const RING_RADIUS = 50
const RING_CIRCUMFERENCE = 2 * Math.PI * RING_RADIUS

const { t } = useI18n()
usePageSeo({
  title: () => t('needtoken.checkCenter.claude.title'),
  description: () => t('needtoken.checkCenter.claude.subtitle'),
  canonicalPath: () => '/tools/environment',
})

const unknown = computed(() => t('needtoken.environmentV6.notAvailable'))
const exit = ref<ExitNetwork | null>(null)
const services = ref<Partial<Record<ClaudeServiceId, AnthropicConnection>>>({})
const rtc = ref<WebRtcObservation | null>(null)
const cli = ref<CliReport | null>(null)
const running = ref(false)
const measuredAt = ref('')
const includeWebRtc = ref(true)
const copyMessage = ref('')
const importMessage = ref('')
const cancelled = ref(false)
const localChecks = ref<ClaudeLocalCheck[]>([])
const storefront = ref<ConnectionDiagnostic | null>(null)
const pending = ref(new Set<NetworkCheckId>())
const rtcIncluded = ref(true)

const baseFingerprintInput = ref<FingerprintInput>({
  timezone: '',
  timezoneOffsetMinutes: 0,
  languages: [],
  intlLocale: '',
  userAgent: '',
  scFontHits: [],
  tcFontHits: [],
  vendorFontHits: [],
})

const fingerprint = computed<FingerprintReport>(() =>
  evaluateChinaFingerprint({
    ...baseFingerprintInput.value,
    webrtcLeakDetail: rtcIncluded.value ? rtc.value?.detail : 'notRequested',
    webrtcPublicIps: rtc.value?.publicAddresses ?? [],
  }),
)

const priorityFixes = computed(() =>
  [...fingerprint.value.signals]
    .filter(sig => sig.points > 0)
    .sort((a, b) => b.points - a.points)
    .slice(0, 4),
)

const browserTimezone = computed(() => baseFingerprintInput.value.timezone)
const timeMismatch = computed(() => timezoneDiffers(browserTimezone.value, exit.value))
const regionSupport = computed(() => classifyAnthropicRegion(exit.value?.countryCode))
const hasCliProxy = computed(() => !!cli.value && (cli.value.proxy.http || cli.value.proxy.https || cli.value.proxy.all))

const serviceHost = (id: ClaudeServiceId) => new URL(CLAUDE_PROBES[id]).hostname
const serviceLabel = (id: ClaudeServiceId) =>
  pending.value.has(id)
    ? t('needtoken.environmentV5.pending')
    : cancelled.value && !services.value[id]
      ? t('needtoken.environmentV5.stopped')
      : t(`needtoken.environmentV6.states.${services.value[id]?.detail || 'idle'}`)
const serviceClass = (id: ClaudeServiceId) =>
  services.value[id]?.detail === 'opaqueResponse'
    ? 'is-observed'
    : services.value[id]?.status === 'warning'
      ? 'is-warning'
      : services.value[id]?.detail === 'readableResponse'
        ? 'is-readable'
        : ''

function formatSignalRaw(sig: FingerprintSignalResult): string {
  if (sig.id === 'webrtcLeak') {
    if (sig.status === 'unknown') return t('needtoken.checkCenter.claude.notChecked')
    if (sig.status === 'safe') return t('needtoken.checkCenter.claude.webrtcClean')
    return sig.raw || t('needtoken.checkCenter.claude.webrtcLeak')
  }
  return sig.raw || t('needtoken.checkCenter.claude.notDetected')
}

const report = computed(() =>
  claudeNetworkReport({
    exit: exit.value,
    anthropic: services.value.anthropic || null,
    services: services.value,
    rtc: rtc.value,
    cli: cli.value,
    measuredAt: measuredAt.value,
    local: localChecks.value,
    storefront: storefront.value,
  }),
)

let active: AbortController | null = null
let generation = 0
let importGeneration = 0
let disposed = false

async function run() {
  if (running.value) return
  const current = ++generation
  const controller = typeof AbortController === 'function' ? new AbortController() : null
  active = controller
  running.value = true
  cancelled.value = false
  copyMessage.value = ''
  measuredAt.value = new Date().toISOString()
  exit.value = null
  services.value = {}
  storefront.value = null
  rtcIncluded.value = includeWebRtc.value
  rtc.value = includeWebRtc.value
    ? null
    : { status: 'unknown', detail: 'notRequested', publicAddresses: [], localCount: 0, mdnsCount: 0 }
  pending.value = new Set<NetworkCheckId>(['storefront', 'exit', ...CLAUDE_SERVICES, ...(rtcIncluded.value ? ['rtc' as const] : [])])
  localChecks.value = collectClaudeLocalChecks()

  const fpPromise = collectBrowserFingerprintInput()
    .then(input => {
      if (!disposed && generation === current) baseFingerprintInput.value = input
    })
    .catch(() => {})

  try {
    await Promise.all([
      fpPromise,
      runClaudeNetworkChecks({
        signal: controller?.signal,
        includeWebRtc: rtcIncluded.value,
        onResult: event => {
          if (disposed || generation !== current) return
          if (event.id === 'storefront') storefront.value = event.result
          else if (event.id === 'exit') exit.value = event.result
          else if (event.id === 'rtc') rtc.value = event.result
          else services.value = { ...services.value, [event.id]: event.result }
          const remaining = new Set(pending.value)
          remaining.delete(event.id)
          pending.value = remaining
        },
      }),
    ])
  } catch { /* Cancelled runs keep completed observations. */ }
  finally {
    if (!disposed && generation === current) {
      running.value = false
      pending.value = new Set()
      active = null
    }
  }
}

function cancel() {
  generation++
  active?.abort()
  active = null
  running.value = false
  cancelled.value = true
  pending.value = new Set()
}

async function copyReport() {
  try {
    await navigator.clipboard.writeText(report.value)
    copyMessage.value = 'copied'
  } catch {
    copyMessage.value = 'copyFailed'
  }
}

async function importCliReport(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  const current = ++importGeneration
  input.value = ''
  importMessage.value = ''
  if (!file) return
  if (file.size > 65536) {
    importMessage.value = 'invalidReport'
    return
  }
  try {
    const parsed = parseCliReport(await file.text())
    if (disposed || current !== importGeneration) return
    if (!parsed) {
      importMessage.value = 'invalidReport'
      return
    }
    cli.value = parsed
    importMessage.value = 'imported'
  } catch {
    if (!disposed && current === importGeneration) importMessage.value = 'invalidReport'
  }
}

onMounted(() => {
  void run()
})

onBeforeUnmount(() => {
  disposed = true
  generation++
  importGeneration++
  active?.abort()
})
</script>

<style scoped>
.nt-claude-env {
  width: 100%;
  max-width: var(--nt-content-max-width, 1200px);
  margin: 0 auto;
  padding: 28px 24px 64px;
  --band-low: #15803d;
  --band-low-bg: rgba(21, 128, 61, 0.1);
  --band-med: #c97a16;
  --band-med-bg: rgba(201, 122, 22, 0.12);
  --band-high: #d1433b;
  --band-high-bg: rgba(209, 67, 59, 0.11);
}

:global(.dark) .nt-claude-env {
  --band-low: #34d399;
  --band-low-bg: rgba(52, 211, 153, 0.12);
  --band-med: #eab338;
  --band-med-bg: rgba(234, 179, 56, 0.13);
  --band-high: #f2746b;
  --band-high-bg: rgba(242, 116, 107, 0.14);
}

.env-heading {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-end;
  justify-content: space-between;
  gap: 16px 24px;
  margin-bottom: 22px;
}

.env-heading h1 {
  color: var(--ui-text-primary);
  font-size: clamp(24px, 2.6vw, 32px);
  font-weight: 600;
  line-height: 1.25;
  letter-spacing: -0.025em;
}

.env-heading p {
  max-width: 60ch;
  margin-top: 8px;
  color: var(--ui-text-secondary);
  font-size: 14px;
  line-height: 1.65;
}

.env-actions {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 12px;
}

.rtc-toggle {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  color: var(--ui-text-secondary);
  font-size: 12.5px;
  cursor: pointer;
}

.rtc-toggle input {
  width: 15px;
  height: 15px;
  accent-color: var(--ui-accent);
}

.env-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 7px;
  min-height: 38px;
  padding: 8px 18px;
  border: 1px solid transparent;
  border-radius: 999px;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  transition: background-color 160ms ease, border-color 160ms ease, transform var(--nt-duration-rebound, 480ms) var(--nt-ease-spring, ease-out);
}

.env-btn.primary {
  background: var(--ui-accent);
  color: var(--ui-text-on-accent);
}
.env-btn.primary:hover:not(:disabled) {
  background: var(--ui-accent-hover);
  transform: translateY(-1px);
}
.env-btn.secondary {
  background: var(--nt-surface-subtle);
  border-color: var(--ui-border-strong);
  color: var(--ui-text-primary);
  font-weight: 500;
}
.env-btn.secondary:hover:not(:disabled) {
  background: var(--ui-bg-soft);
}
.env-btn:disabled {
  opacity: 0.55;
  cursor: progress;
}
.env-btn:active:not(:disabled) {
  transform: scale(0.97);
  transition-duration: var(--nt-duration-press, 80ms);
}

.env-grid {
  display: grid;
  grid-template-columns: minmax(0, 1.35fr) minmax(300px, 1fr);
  gap: 20px;
  align-items: start;
}

.env-side-col {
  display: grid;
  gap: 20px;
}

.env-card {
  min-width: 0;
  padding: 22px 24px;
  border: 1px solid var(--ui-border);
  border-radius: 18px;
  background: var(--ui-bg-elevated);
  box-shadow: var(--nt-card-shadow);
}

.card-head {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-start;
  justify-content: space-between;
  gap: 10px 14px;
}

.card-head h2 {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  color: var(--ui-text-primary);
  font-size: 15.5px;
  font-weight: 600;
  line-height: 1.4;
}

.card-head p {
  margin-top: 3px;
  color: var(--ui-text-muted);
  font-size: 12px;
  line-height: 1.5;
}

.triggered-pill,
.region-badge {
  display: inline-flex;
  align-items: center;
  padding: 3px 10px;
  border-radius: 999px;
  font-size: 11.5px;
  font-weight: 600;
  white-space: nowrap;
}

.triggered-pill[data-band="low"],
.region-badge[data-region="supported"] {
  color: var(--band-low);
  background: var(--band-low-bg);
}

.triggered-pill[data-band="medium"] {
  color: var(--band-med);
  background: var(--band-med-bg);
}

.triggered-pill[data-band="high"],
.region-badge[data-region="restricted"] {
  color: var(--band-high);
  background: var(--band-high-bg);
}

.region-badge[data-region="unknown"] {
  color: var(--ui-text-muted);
  background: var(--ui-bg-soft);
}

/* Score ring hero */
.score-hero {
  display: flex;
  align-items: center;
  gap: 22px;
  margin-top: 18px;
  padding: 18px 20px;
  border: 1px solid var(--ui-border);
  border-radius: 14px;
  background: var(--nt-surface-subtle);
}

.score-gauge {
  position: relative;
  width: 102px;
  height: 102px;
  flex: none;
  --ring-color: var(--band-low);
}

.score-gauge[data-band="medium"] { --ring-color: var(--band-med); }
.score-gauge[data-band="high"] { --ring-color: var(--band-high); }

.score-gauge svg {
  width: 100%;
  height: 100%;
  transform: rotate(-90deg);
}

.gauge-track {
  fill: none;
  stroke: var(--ui-border);
  stroke-width: 9;
}

.gauge-fill {
  fill: none;
  stroke: var(--ring-color);
  stroke-width: 9;
  stroke-linecap: round;
  transition: stroke-dashoffset 520ms cubic-bezier(0.22, 1, 0.36, 1), stroke 220ms ease;
}

.gauge-center {
  position: absolute;
  inset: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  font-variant-numeric: tabular-nums;
}

.gauge-center strong {
  color: var(--ui-text-primary);
  font-size: 28px;
  font-weight: 700;
  line-height: 1;
}

.gauge-center span {
  margin-top: 2px;
  color: var(--ui-text-muted);
  font-size: 11px;
}

.score-summary {
  min-width: 0;
  flex: 1;
}

.band-badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 11px;
  border-radius: 999px;
  font-size: 12.5px;
  font-weight: 600;
}

.band-badge[data-band="low"] { color: var(--band-low); background: var(--band-low-bg); }
.band-badge[data-band="medium"] { color: var(--band-med); background: var(--band-med-bg); }
.band-badge[data-band="high"] { color: var(--band-high); background: var(--band-high-bg); }

.band-desc {
  margin-top: 9px;
  color: var(--ui-text-secondary);
  font-size: 13px;
  line-height: 1.6;
}

/* 10 signals breakdown */
.signal-list {
  list-style: none;
  margin: 16px 0 0;
  padding: 0;
}

.signal-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
  padding: 10px 0;
  border-top: 1px solid var(--ui-border);
}

.signal-row:first-child {
  border-top: 0;
}

.signal-main {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  min-width: 0;
}

.signal-dot {
  width: 7px;
  height: 7px;
  margin-top: 6px;
  flex: none;
  border-radius: 999px;
  background: var(--band-low);
}

.signal-row[data-status="warn"] .signal-dot { background: var(--band-med); }
.signal-row[data-status="risk"] .signal-dot { background: var(--band-high); }
.signal-row[data-status="unknown"] .signal-dot { background: var(--ui-text-muted); }

.signal-text {
  min-width: 0;
}

.signal-title-line {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px;
}

.signal-title-line strong {
  color: var(--ui-text-primary);
  font-size: 13px;
  font-weight: 500;
}

.claude-used-tag {
  padding: 1px 7px;
  border-radius: 999px;
  background: var(--ui-accent-soft);
  color: var(--ui-accent);
  font-size: 10.5px;
  font-weight: 600;
}

.signal-raw {
  display: block;
  margin-top: 2px;
  color: var(--ui-text-muted);
  font-family: var(--nt-font-mono);
  font-size: 11.5px;
  line-height: 1.45;
  overflow-wrap: anywhere;
}

.signal-pts {
  flex: none;
  padding: 2px 9px;
  border-radius: 999px;
  font-size: 11.5px;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}

.signal-pts[data-status="safe"] { color: var(--band-low); background: var(--band-low-bg); }
.signal-pts[data-status="warn"] { color: var(--band-med); background: var(--band-med-bg); }
.signal-pts[data-status="risk"] { color: var(--band-high); background: var(--band-high-bg); }
.signal-pts[data-status="unknown"] { color: var(--ui-text-muted); background: var(--ui-bg-soft); }

/* Priority Fixes */
.fixes-box {
  margin-top: 16px;
  padding: 16px 18px;
  border: 1px solid var(--ui-border);
  border-radius: 14px;
  background: var(--nt-surface-subtle);
}

.fixes-box h3 {
  color: var(--ui-text-primary);
  font-size: 13.5px;
  font-weight: 600;
}

.fixes-list {
  display: grid;
  gap: 10px;
  margin-top: 10px;
  padding: 0;
  list-style: none;
}

.fixes-list li {
  font-size: 12.5px;
  line-height: 1.6;
  color: var(--ui-text-secondary);
}

.fixes-list strong {
  color: var(--ui-text-primary);
  font-weight: 600;
  margin-right: 6px;
}

.fixes-clean {
  display: flex;
  align-items: center;
  gap: 7px;
  margin-top: 8px;
  color: var(--band-low);
  font-size: 12.5px;
}

/* Exit IP panel */
.exit-ip {
  margin: 16px 0 18px;
  color: var(--ui-text-primary);
  font-family: var(--nt-font-mono);
  font-size: clamp(20px, 2.2vw, 25px);
  font-weight: 600;
  letter-spacing: -0.02em;
  overflow-wrap: anywhere;
}

.exit-ip.is-ipv6 {
  font-size: clamp(14px, 1.35vw, 17px);
  letter-spacing: -0.01em;
}

.exit-facts {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 14px;
}

.exit-facts dt {
  color: var(--ui-text-muted);
  font-size: 11.5px;
}

.exit-facts dd {
  margin-top: 3px;
  color: var(--ui-text-secondary);
  font-size: 13px;
  font-weight: 500;
  overflow-wrap: anywhere;
}

.exit-checks {
  display: grid;
  gap: 8px;
  margin-top: 16px;
  padding-top: 14px;
  border-top: 1px solid var(--ui-border);
}

.exit-check-pill {
  display: flex;
  align-items: center;
  gap: 7px;
  padding: 8px 11px;
  border-radius: 10px;
  font-size: 12px;
  line-height: 1.45;
}

.exit-check-pill[data-tone="ok"] { color: var(--band-low); background: var(--band-low-bg); }
.exit-check-pill[data-tone="warn"] { color: var(--band-med); background: var(--band-med-bg); }
.exit-check-pill[data-tone="risk"] { color: var(--band-high); background: var(--band-high-bg); }
.exit-check-pill[data-tone="muted"] { color: var(--ui-text-muted); background: var(--ui-bg-soft); }
.exit-check-pill svg { flex: none; }

.rtc-ips {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.rtc-ips code {
  padding: 2px 8px;
  border-radius: 6px;
  background: var(--ui-bg-soft);
  color: var(--ui-text-primary);
  font-size: 11.5px;
}

/* Services list */
.status-ext-link {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  color: var(--ui-text-muted);
  font-size: 12px;
}
.status-ext-link:hover { color: var(--ui-accent); }

.service-list {
  list-style: none;
  margin: 14px 0 0;
  padding: 0;
}

.service-list li {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto 58px;
  gap: 10px;
  align-items: center;
  padding: 12px 0;
  border-top: 1px solid var(--ui-border);
}

.service-list li:first-child {
  border-top: 0;
  padding-top: 4px;
}

.service-name {
  display: flex;
  align-items: center;
  gap: 9px;
  min-width: 0;
}

.service-name svg {
  flex: none;
  color: var(--ui-text-muted);
}

.service-name strong {
  display: block;
  color: var(--ui-text-primary);
  font-size: 13px;
  font-weight: 500;
}

.service-name span {
  display: block;
  color: var(--ui-text-muted);
  font-size: 11.5px;
  overflow-wrap: anywhere;
}

.service-state {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  color: var(--ui-text-muted);
  font-size: 12px;
  white-space: nowrap;
}

.service-state i {
  width: 6px;
  height: 6px;
  border-radius: 999px;
  background: currentColor;
}

.service-state.is-readable { color: var(--band-low); }
.service-state.is-observed { color: var(--ui-text-secondary); }
.service-state.is-warning { color: var(--band-med); }

.service-latency {
  text-align: right;
  color: var(--ui-text-secondary);
  font-family: var(--nt-font-mono);
  font-size: 12px;
  font-variant-numeric: tabular-nums;
}

.card-note {
  margin-top: 12px;
  color: var(--ui-text-muted);
  font-size: 12px;
  line-height: 1.55;
}

/* Collapsible advanced section */
.env-advanced {
  display: grid;
  gap: 12px;
  margin-top: 20px;
}

.adv-details {
  border: 1px solid var(--ui-border);
  border-radius: 14px;
  background: var(--ui-bg-elevated);
}

.adv-details > summary {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 15px 20px;
  color: var(--ui-text-secondary);
  font-size: 13.5px;
  font-weight: 500;
  cursor: pointer;
  list-style: none;
}
.adv-details > summary::-webkit-details-marker { display: none; }
.adv-details > summary > svg:last-child { margin-left: auto; color: var(--ui-text-muted); }
.adv-details[open] > summary > svg:last-child { transform: rotate(180deg); }
.adv-meta { margin-left: auto; margin-right: 6px; color: var(--ui-text-muted); font-size: 12px; font-weight: 400; }

.adv-content {
  padding: 0 20px 20px;
}

.adv-copy {
  color: var(--ui-text-secondary);
  font-size: 12.5px;
  line-height: 1.6;
}

.cli-steps {
  display: grid;
  gap: 10px;
  margin-top: 14px;
  padding-left: 18px;
  list-style: decimal;
  color: var(--ui-text-secondary);
  font-size: 12.5px;
}

.cli-steps a {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  margin-left: 10px;
  color: var(--ui-accent);
}

.cli-cmd {
  display: block;
  margin-top: 6px;
  padding: 10px 12px;
  border: 1px solid var(--ui-border);
  border-radius: 8px;
  background: var(--ui-bg-page);
  font-size: 12px;
  overflow-wrap: anywhere;
}

.cli-import {
  position: relative;
  display: inline-flex;
  align-items: center;
  gap: 7px;
  margin-top: 14px;
  padding: 8px 14px;
  border: 1px solid var(--ui-border-strong);
  border-radius: 999px;
  color: var(--ui-text-primary);
  font-size: 12.5px;
  cursor: pointer;
}

.cli-import input {
  position: absolute;
  inset: 0;
  opacity: 0;
  cursor: pointer;
}

.cli-facts {
  margin-top: 16px;
}

.report-textarea {
  display: block;
  width: 100%;
  min-height: 180px;
  margin-top: 12px;
  padding: 12px 14px;
  border: 1px solid var(--ui-border);
  border-radius: 10px;
  background: var(--ui-bg-page);
  color: var(--ui-text-secondary);
  font-family: var(--nt-font-mono);
  font-size: 11.5px;
  line-height: 1.6;
}

.skeleton-line {
  display: block;
  width: 75%;
  height: 13px;
  border-radius: 4px;
  background: var(--ui-bg-soft);
  animation: env-pulse 1.4s ease-in-out infinite;
}
.ip-skeleton {
  height: 26px;
  width: 65%;
}

.env-spin {
  animation: env-spin 0.9s linear infinite;
}

@keyframes env-spin { to { transform: rotate(360deg); } }
@keyframes env-pulse { 50% { opacity: 0.45; } }

@media (max-width: 960px) {
  .env-grid {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 640px) {
  .nt-claude-env { padding: 20px 16px 48px; }
  .env-card { padding: 18px 16px; border-radius: 16px; }
  .score-hero { flex-direction: column; align-items: flex-start; gap: 14px; }
  .exit-facts { grid-template-columns: 1fr; }
}

@media (prefers-reduced-motion: reduce) {
  .env-spin,
  .skeleton-line { animation: none; }
  .gauge-fill,
  .env-btn { transition: none; }
}
</style>
