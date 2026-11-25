<template>
  <section class="api-hero-wrap" aria-labelledby="api-services-title">
    <div class="api-hero">
      <div class="api-hero-copy">
        <p class="api-hero-label">{{ t('needtoken.apiConnect.eyebrow') }}</p>
        <h1 id="api-services-title">
          {{ t('needtoken.apiConnect.heroTitle') }}
          <span>{{ t('needtoken.apiConnect.heroAccent') }}</span>
        </h1>
        <p class="api-hero-intro">{{ t('needtoken.apiConnect.heroIntro') }}</p>

        <!-- 直显可一键复制的 API 接口地址 -->
        <div class="api-endpoint-bar">
          <span class="api-endpoint-tag">API Base URL</span>
          <code class="api-endpoint-url">{{ API_BASE_URL }}</code>
          <button
            type="button"
            class="api-endpoint-copy"
            :aria-label="t('needtoken.apiConnect.copyEndpoint')"
            @click="copyBaseUrl"
          >
            <Check v-if="copiedEndpoint" :size="15" aria-hidden="true" />
            <Copy v-else :size="15" aria-hidden="true" />
            <span>{{ copiedEndpoint ? t('needtoken.common.copied') : t('needtoken.common.copy') }}</span>
          </button>
        </div>

        <div class="api-hero-actions">
          <button
            type="button"
            class="primary"
            @click="handleApiKeyAction"
          >
            <KeyRound :size="16" aria-hidden="true" />
            <span>{{ userAuth.isAuthenticated ? t('needtoken.apiConnect.viewApiKeyAction') : t('needtoken.apiConnect.createApiKeyAction') }}</span>
            <ArrowRight :size="15" aria-hidden="true" />
          </button>
          <button
            type="button"
            class="console-btn"
            @click="handleOpenConsole"
          >
            <LayoutDashboard :size="16" aria-hidden="true" />
            <span>{{ t('needtoken.apiConnect.goDashboardAction') }}</span>
            <ArrowRight :size="15" aria-hidden="true" />
          </button>
          <a href="#gateway-plans" class="ghost-link">
            <span>{{ t('needtoken.apiConnect.browsePlansAnchor') }}</span>
            <ArrowDown :size="15" aria-hidden="true" />
          </a>
        </div>
        <p class="api-hero-footnote">{{ t('needtoken.apiConnect.heroNote') }}</p>
      </div>

      <div class="api-preview" :aria-label="t('needtoken.apiConnect.previewLabel')">
        <div class="preview-header">
          <span><Braces :size="16" aria-hidden="true" />{{ t('needtoken.apiConnect.previewTitle') }}</span>
          <span class="preview-pill">{{ activeProtocolMeta.badge }}</span>
        </div>
        <ol class="api-route" :aria-label="t('needtoken.apiConnect.routeLabel')">
          <li>
            <Terminal :size="18" aria-hidden="true" />
            <span>{{ t('needtoken.apiConnect.client') }}<small>{{ t('needtoken.apiConnect.clientDetail') }}</small></span>
          </li>
          <li class="route-arrow" aria-hidden="true"><ArrowRight :size="16" /></li>
          <li class="route-gateway">
            <img src="/needtoken-icon.svg" alt="" />
            <span>NeedToken<small>{{ t('needtoken.apiConnect.gatewayDetail') }}</small></span>
          </li>
          <li class="route-arrow" aria-hidden="true"><ArrowRight :size="16" /></li>
          <li>
            <Cpu :size="18" aria-hidden="true" />
            <span>{{ t('needtoken.apiConnect.model') }}<small>{{ t('needtoken.apiConnect.modelDetail') }}</small></span>
          </li>
        </ol>
        <div class="preview-endpoint">
          <span>POST</span>
          <code>{{ activeProtocolMeta.endpoint }}</code>
        </div>
        <div class="preview-code-head">
          <div role="group" :aria-label="t('needtoken.apiConnect.previewLanguage')">
            <button
              v-for="tab in previewTabs"
              :key="tab.id"
              type="button"
              :aria-pressed="previewKind === tab.id"
              @click="previewKind = tab.id"
            >
              {{ tab.label }}
            </button>
          </div>
          <div class="preview-code-meta">
            <span>{{ t('needtoken.apiConnect.requestExample') }}</span>
            <button type="button" class="preview-copy-btn" @click="copySnippet">
              <Check v-if="copiedSnippet" :size="13" aria-hidden="true" />
              <Copy v-else :size="13" aria-hidden="true" />
              <span>{{ copiedSnippet ? t('needtoken.common.copied') : t('needtoken.common.copy') }}</span>
            </button>
          </div>
        </div>
        <pre :key="previewKind" class="preview-code" tabindex="0" :aria-label="t('needtoken.apiConnect.requestExample')"><code>{{ previewSnippet }}</code></pre>
        <div class="preview-footer">
          <KeyRound :size="14" aria-hidden="true" />
          <code>NEEDTOKEN_API_KEY</code>
          <span>{{ t('needtoken.apiConnect.environmentKey') }}</span>
        </div>
      </div>
    </div>

    <div v-if="$slots['before-metrics']" class="api-hero-before-metrics">
      <slot name="before-metrics" />
    </div>

    <!-- 实时网关指标条 -->
    <dl class="api-metrics-strip" :aria-label="t('needtoken.apiConnect.metrics.label')">
      <div class="metric-cell">
        <dt>{{ t('needtoken.apiConnect.metrics.modelsLabel') }}</dt>
        <dd>
          <strong>{{ modelCountDisplay }}</strong>
          <span>{{ t('needtoken.apiConnect.metrics.modelsSub') }}</span>
        </dd>
      </div>
      <div class="metric-cell">
        <dt>{{ t('needtoken.apiConnect.metrics.vendorsLabel') }}</dt>
        <dd>
          <strong>{{ vendorCountDisplay }}</strong>
          <span>{{ t('needtoken.apiConnect.metrics.vendorsSub') }}</span>
        </dd>
      </div>
      <div class="metric-cell">
        <dt>{{ t('needtoken.apiConnect.metrics.protocolsLabel') }}</dt>
        <dd>
          <strong>4</strong>
          <span>{{ t('needtoken.apiConnect.metrics.protocolsSub') }}</span>
        </dd>
      </div>
      <div class="metric-cell">
        <dt>{{ t('needtoken.apiConnect.metrics.poolLabel') }}</dt>
        <dd>
          <strong class="metric-status">
            <i class="status-dot" :class="`is-${poolSummary.overall}`" aria-hidden="true" />
            {{ poolStatusDisplay }}
          </strong>
          <span>{{ poolStatusSub }}</span>
        </dd>
      </div>
    </dl>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { ArrowDown, ArrowRight, Braces, Check, Copy, Cpu, KeyRound, LayoutDashboard, Terminal } from 'lucide-vue-next'
import { useUserAuthStore } from '../../../stores/userAuth'
import { useAppStore } from '../../../stores/app'
import { API_BASE_URL, CONSOLE_URL, DEFAULT_MODELS } from '../config'
import { apiPreviewCode } from '../lib/apiIntegrations'
import { usePricing } from '../lib/usePricing'
import { summarizePoolStatus } from '../lib/poolStatus'
import { usePoolStatus } from '../lib/usePoolStatus'

type HeroPreviewKind = 'responses' | 'curl' | 'anthropic' | 'python'

const { t } = useI18n()
const router = useRouter()
const userAuth = useUserAuthStore()
const appStore = useAppStore()
const pricing = usePricing()
const pool = usePoolStatus()

const previewTabs: readonly { id: HeroPreviewKind; label: string }[] = [
  { id: 'responses', label: 'Responses' },
  { id: 'curl', label: 'Chat' },
  { id: 'anthropic', label: 'Messages' },
  { id: 'python', label: 'Python' },
]
const previewKind = ref<HeroPreviewKind>('responses')

const rootOrigin = computed(() => API_BASE_URL.replace(/\/v1$/, ''))
const primaryModel = computed(() => DEFAULT_MODELS[0])
const claudeModel = computed(() => DEFAULT_MODELS.find(m => m.startsWith('claude')) || 'claude-sonnet-4-6')

const activeProtocolMeta = computed(() => {
  switch (previewKind.value) {
    case 'responses':
      return { endpoint: '/v1/responses', badge: 'OpenAI Responses' }
    case 'anthropic':
      return { endpoint: '/v1/messages', badge: 'Anthropic Native' }
    default:
      return { endpoint: '/v1/chat/completions', badge: 'OpenAI Compatible' }
  }
})

const previewSnippet = computed(() => {
  if (previewKind.value === 'responses') {
    return `curl '${API_BASE_URL}/responses' \\\n  -H "Authorization: Bearer $NEEDTOKEN_API_KEY" \\\n  -H "Content-Type: application/json" \\\n  -d '{\n  "model": "${primaryModel.value}",\n  "input": "Hello from NeedToken Gateway"\n}'`
  }
  if (previewKind.value === 'anthropic') {
    return `curl '${rootOrigin.value}/v1/messages' \\\n  -H "x-api-key: $NEEDTOKEN_API_KEY" \\\n  -H "anthropic-version: 2023-06-01" \\\n  -H "content-type: application/json" \\\n  -d '{\n  "model": "${claudeModel.value}",\n  "max_tokens": 1024,\n  "messages": [{"role": "user", "content": "Hello"}]\n}'`
  }
  return apiPreviewCode(previewKind.value, API_BASE_URL, primaryModel.value)
})

const uniqueModelCount = computed(() => new Set(pricing.offers.value.map(o => o.modelId)).size)
const uniqueVendorCount = computed(() => new Set(pricing.offers.value.map(o => o.vendor)).size)
const modelCountDisplay = computed(() => (uniqueModelCount.value > 0 ? String(uniqueModelCount.value) : `${DEFAULT_MODELS.length}+`))
const vendorCountDisplay = computed(() => (uniqueVendorCount.value > 0 ? String(uniqueVendorCount.value) : '5+'))

const poolSummary = computed(() => summarizePoolStatus(pool.view.value))
const poolStatusDisplay = computed(() => {
  if (!poolSummary.value.enabled) return t('needtoken.apiConnect.metrics.poolStandby')
  return `${poolSummary.value.healthyGroups} / ${poolSummary.value.totalGroups}`
})
const poolStatusSub = computed(() => {
  if (!poolSummary.value.enabled) return t('needtoken.apiConnect.metrics.poolStandbySub')
  return t('needtoken.apiConnect.metrics.poolOnlineSub')
})

const copiedEndpoint = ref(false)
const copiedSnippet = ref(false)
let copyEndpointTimer: ReturnType<typeof setTimeout> | undefined
let copySnippetTimer: ReturnType<typeof setTimeout> | undefined

const copyBaseUrl = async () => {
  if (copyEndpointTimer) clearTimeout(copyEndpointTimer)
  try {
    await navigator.clipboard.writeText(API_BASE_URL)
    copiedEndpoint.value = true
    copyEndpointTimer = setTimeout(() => { copiedEndpoint.value = false }, 1800)
  } catch {
    copiedEndpoint.value = false
  }
}

const copySnippet = async () => {
  if (copySnippetTimer) clearTimeout(copySnippetTimer)
  try {
    await navigator.clipboard.writeText(previewSnippet.value)
    copiedSnippet.value = true
    copySnippetTimer = setTimeout(() => { copiedSnippet.value = false }, 1800)
  } catch {
    copiedSnippet.value = false
  }
}

const handleApiKeyAction = () => {
  if (appStore.isResellerTenant) {
    window.open(`${CONSOLE_URL}/keys`, '_blank', 'noopener,noreferrer')
    return
  }
  if (!userAuth.isAuthenticated) {
    void router.push({ path: '/auth/login', query: { redirect: '/me/keys' } })
    return
  }
  void router.push('/me/keys')
}

const handleOpenConsole = () => {
  if (appStore.isResellerTenant) {
    window.open(CONSOLE_URL, '_blank', 'noopener,noreferrer')
    return
  }
  if (!userAuth.isAuthenticated) {
    void router.push({ path: '/auth/login', query: { redirect: '/me' } })
    return
  }
  void router.push('/me')
}

onMounted(() => {
  void pricing.ensureLoaded()
  void pool.ensureLoaded()
})

onUnmounted(() => {
  if (copyEndpointTimer) clearTimeout(copyEndpointTimer)
  if (copySnippetTimer) clearTimeout(copySnippetTimer)
})
</script>

<style scoped>
.api-hero-wrap { padding-top: 48px; padding-bottom: 44px; animation: api-enter 400ms cubic-bezier(.22,1,.36,1) both; }
.api-hero { display: grid; grid-template-columns: minmax(0, .95fr) minmax(0, 1.05fr); align-items: center; gap: 52px; }
.api-hero-copy { min-width: 0; }
.api-hero-label { color: var(--ui-accent); font-size: 12.5px; font-weight: 600; letter-spacing: .08em; text-transform: uppercase; }
.api-hero h1 { margin-top: 16px; font-size: clamp(34px, 4vw, 52px); line-height: 1.15; letter-spacing: -.035em; font-weight: 650; text-wrap: balance; }
.api-hero h1 span { display: block; margin-top: 8px; color: var(--ui-accent); }
.api-hero-intro { margin-top: 18px; max-width: 38em; color: var(--ui-text-secondary); font-size: 15.5px; line-height: 1.8; text-wrap: pretty; }
.api-endpoint-bar { display: flex; align-items: center; gap: 10px; margin-top: 22px; padding: 6px 8px 6px 12px; border: 1px solid var(--ui-border-strong); border-radius: 12px; background: var(--ui-bg-elevated); max-width: 520px; }
.api-endpoint-tag { flex: none; padding: 2px 8px; border-radius: 6px; background: var(--ui-accent-soft, rgba(16, 185, 129, 0.1)); color: var(--ui-accent); font: 600 11px var(--nt-font-mono); }
.api-endpoint-url { flex: 1; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font: 600 13px var(--nt-font-mono); color: var(--ui-text-primary); }
.api-endpoint-copy { display: inline-flex; flex: none; align-items: center; gap: 5px; height: 32px; padding: 0 10px; border-radius: 8px; border: 1px solid var(--ui-border); background: var(--ui-bg-soft); color: var(--ui-text-secondary); font-size: 12px; font-weight: 600; cursor: pointer; }
.api-endpoint-copy:hover { border-color: var(--ui-accent); color: var(--ui-accent); }
.api-hero-actions { display: flex; flex-wrap: wrap; align-items: center; gap: 10px; margin-top: 18px; }
.api-hero-actions a,
.api-hero-actions button { display: inline-flex; align-items: center; justify-content: center; gap: 9px; min-height: 46px; padding: 11px 18px; border: 1px solid var(--ui-border-strong); border-radius: 11px; color: var(--ui-text-primary); background: var(--ui-bg-elevated); font-size: 13.5px; font-weight: 600; cursor: pointer; transition: color 160ms, background 160ms, border-color 160ms, transform 160ms; }
.api-hero-actions a:hover,
.api-hero-actions button:hover:not(:disabled) { border-color: var(--nt-accent-border-strong, var(--ui-accent)); background: var(--ui-bg-soft); }
.api-hero-actions a.primary,
.api-hero-actions button.primary { color: var(--ui-text-on-accent); border-color: var(--ui-accent); background: var(--ui-accent); }
.api-hero-actions a.primary:hover,
.api-hero-actions button.primary:hover:not(:disabled) { background: var(--ui-accent-hover); }
.preview-code-meta { display: inline-flex; align-items: center; gap: 8px; }
.preview-copy-btn { display: inline-flex; align-items: center; gap: 4px; padding: 3px 8px; border-radius: 6px; border: 1px solid var(--ui-border); background: var(--ui-bg-soft); color: var(--ui-text-secondary); font-size: 11px; font-weight: 500; cursor: pointer; }
.preview-copy-btn:hover { color: var(--ui-accent); border-color: var(--ui-accent); }
.api-hero-actions .console-btn { border-color: var(--nt-accent-border, var(--ui-border-strong)); color: var(--ui-accent); background: var(--nt-accent-subtle, var(--ui-bg-elevated)); }
.api-hero-actions .ghost-link { border-color: transparent; background: transparent; color: var(--ui-text-secondary); padding-inline: 12px; }
.api-hero-actions button:disabled { opacity: .65; cursor: not-allowed; }
.api-hero-footnote { margin-top: 18px; color: var(--ui-text-muted); font-size: 12px; line-height: 1.75; }

.api-preview { min-width: 0; overflow: hidden; border: 1px solid var(--ui-border-strong); border-radius: 20px; background: var(--ui-bg-elevated); }
.preview-header, .preview-footer { display: flex; align-items: center; justify-content: space-between; gap: 12px; padding: 14px 20px; color: var(--ui-text-muted); font-size: 12px; }
.preview-header { border-bottom: 1px solid var(--ui-border); }
.preview-header > span:first-child { display: inline-flex; align-items: center; gap: 8px; color: var(--ui-text-secondary); font-weight: 500; }
.preview-pill { padding: 2px 8px; border-radius: 999px; border: 1px solid var(--nt-accent-border, var(--ui-border)); color: var(--ui-accent); font: 11px var(--nt-font-mono); }
.api-route { display: flex; align-items: center; justify-content: space-between; gap: 12px; margin: 18px 20px; }
.api-route li:not(.route-arrow) { display: flex; align-items: center; gap: 8px; min-width: 0; color: var(--ui-text-secondary); font-size: 13px; }
.api-route li > svg { flex: none; color: var(--ui-text-muted); }
.api-route li > span { min-width: 0; }
.api-route small { display: block; margin-top: 3px; color: var(--ui-text-muted); font-size: 11px; line-height: 1.5; }
.api-route img { width: 28px; height: 28px; flex: none; }
.api-route .route-gateway > span { color: var(--ui-accent); font-weight: 600; }
.route-arrow { flex: none; color: var(--ui-text-muted); }
.preview-endpoint { display: flex; align-items: center; gap: 12px; padding: 10px 20px; border-block: 1px solid var(--ui-border); background: var(--ui-bg-soft); font-size: 12px; }
.preview-endpoint span { color: var(--ui-accent); font-family: var(--nt-font-mono); font-weight: 600; }
.preview-endpoint code { min-width: 0; overflow-wrap: anywhere; color: var(--ui-text-secondary); font-family: var(--nt-font-mono); }
.preview-code-head { display: flex; align-items: center; justify-content: space-between; gap: 8px; padding: 10px 20px 0; color: var(--ui-text-muted); font-size: 11px; }
.preview-code-head > div { display: flex; flex-wrap: wrap; gap: 4px; }
.preview-code-head button { padding: 5px 9px; border-radius: 6px; color: var(--ui-text-muted); font-size: 12px; transition: color 160ms, background 160ms; }
.preview-code-head button[aria-pressed=true] { color: var(--ui-accent); background: var(--ui-accent-soft, var(--ui-bg-soft)); font-weight: 600; }
.preview-code { height: 184px; overflow: auto; padding: 14px 20px; color: var(--ui-text-secondary); font: 12px/1.8 var(--nt-font-mono); tab-size: 2; }
.preview-footer { justify-content: flex-start; flex-wrap: wrap; gap: 8px; border-top: 1px solid var(--ui-border); background: var(--ui-bg-soft); font-size: 11px; }
.preview-footer > span { margin-left: auto; }

.api-hero-before-metrics { margin-top: 36px; }
.api-hero-before-metrics :deep(.routing-showcase) { padding-block: 0; border-top: 0; }

.api-metrics-strip { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 12px; margin-top: 24px; }
.metric-cell { padding: 16px 18px; border: 1px solid var(--ui-border); border-radius: 16px; background: var(--ui-bg-elevated); }
.metric-cell dt { font-size: 11.5px; color: var(--ui-text-muted); font-weight: 500; }
.metric-cell dd { margin-top: 6px; display: flex; flex-direction: column; gap: 3px; }
.metric-cell strong { font: 700 22px/1.2 var(--nt-font-mono); color: var(--ui-text-primary); letter-spacing: -.03em; }
.metric-cell dd > span { font-size: 11.5px; color: var(--ui-text-secondary); }
.metric-status { display: inline-flex; align-items: center; gap: 8px; font-family: inherit !important; font-size: 17px !important; }
.status-dot { width: 8px; height: 8px; border-radius: 999px; background: var(--ui-text-muted); }
.status-dot.is-available { background: #10b981; box-shadow: 0 0 0 4px rgba(16, 185, 129, 0.16); }
.status-dot.is-busy { background: #f59e0b; box-shadow: 0 0 0 4px rgba(245, 158, 11, 0.16); }
.status-dot.is-unavailable { background: #ef4444; }

.spin { animation: hero-spin 1s linear infinite; }
@keyframes hero-spin { to { transform: rotate(360deg); } }
@keyframes api-enter { from { opacity: 0; transform: translateY(8px); } to { opacity: 1; transform: none; } }

.api-hero-wrap :is(a,button,pre):focus-visible { outline: 2px solid var(--ui-accent); outline-offset: 3px; }
@media (max-width: 1024px) {
  .api-hero { gap: 32px; }
  .api-metrics-strip { grid-template-columns: repeat(2, minmax(0, 1fr)); }
}
@media (max-width: 850px) {
  .api-hero { grid-template-columns: minmax(0, 1fr); gap: 36px; }
  .api-hero-copy { max-width: 640px; }
  .api-preview { width: 100%; max-width: 680px; }
}
@media (max-width: 480px) {
  .api-hero-wrap { padding-top: 28px; padding-bottom: 28px; }
  .api-hero h1 { font-size: 32px; }
  .api-hero-actions a, .api-hero-actions button { min-height: 42px; padding: 10px 14px; font-size: 13px; }
  .api-route small { display: none; }
  .api-metrics-strip { grid-template-columns: 1fr 1fr; gap: 8px; }
  .metric-cell { padding: 13px 14px; }
  .metric-cell strong { font-size: 18px; }
}
@media (prefers-reduced-motion: reduce) { .api-hero-wrap { animation: none; } .api-hero-wrap :is(a,button) { transition: none; } }
</style>
