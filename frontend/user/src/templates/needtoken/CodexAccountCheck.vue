<template>
  <div class="account-tool">
    <ToolNavigation />
    <header class="tool-heading">
      <span class="tool-capability"><ShieldCheck :size="16" :stroke-width="1.5" aria-hidden="true" />{{ t('needtoken.toolsPolish.accountScope') }}</span>
      <h1>{{ t('needtoken.codexCheck.title') }}</h1>
      <p>{{ t('needtoken.codexCheck.intro') }}</p>
    </header>

    <div class="account-layout" :class="{ 'has-credential': hasCredential }">
      <section class="account-input-panel" :aria-labelledby="'session-input-title'">
        <div class="panel-heading">
          <h2 id="session-input-title">{{ t('needtoken.codexCheck.inputTitle') }}</h2>
          <span v-if="hasCredential" class="credential-memory"><LockKeyhole :size="14" :stroke-width="1.5" aria-hidden="true" />{{ t('needtoken.codexCheck.memoryOnly') }}</span>
        </div>
        <form @submit.prevent="detect">
          <template v-if="!hasCredential">
            <label class="session-label" for="codex-session">{{ t('needtoken.codexCheck.inputLabel') }}</label>
            <textarea id="codex-session" v-model="sessionInput" :maxlength="MAX_CODEX_SESSION_INPUT" :placeholder="t('needtoken.codexCheck.inputPlaceholder')" :disabled="busy" :aria-invalid="!!inputError" :aria-describedby="inputError ? 'session-error session-privacy' : 'session-privacy'" autocomplete="off" autocapitalize="off" autocorrect="off" spellcheck="false" data-1p-ignore data-lpignore="true" @input="inputError = null" />
          </template>
          <p v-else class="credential-ready"><ShieldCheck :size="18" :stroke-width="1.5" aria-hidden="true" />{{ t('needtoken.codexCheck.credentialReady') }}</p>
          <p v-if="inputError" id="session-error" class="inline-error" role="alert">{{ t(`needtoken.codexCheck.inputErrors.${inputError}`) }}</p>
          <div class="form-actions">
            <button v-if="!hasCredential" type="submit" class="primary-action" :disabled="busy || !sessionInput.trim()"><ScanLine :size="16" :stroke-width="1.5" aria-hidden="true" />{{ t('needtoken.codexCheck.detect') }}</button>
            <button v-else type="button" class="primary-action" :disabled="busy" :aria-busy="busy" @click="requestCheck()"><LoaderCircle v-if="busy" :size="16" :stroke-width="1.5" class="loading-icon" aria-hidden="true" /><RefreshCw v-else :size="16" :stroke-width="1.5" aria-hidden="true" />{{ t(busy ? 'needtoken.codexCheck.loading' : 'needtoken.codexCheck.detectAgain') }}</button>
            <button v-if="busy" type="button" class="secondary-action" @click="clear"><X :size="16" :stroke-width="1.5" aria-hidden="true" />{{ t('needtoken.codexCheck.cancel') }}</button>
            <button v-else-if="sessionInput || hasCredential || result || requestError" type="button" class="secondary-action" @click="clear">{{ t('needtoken.codexCheck.clear') }}</button>
          </div>
          <p id="session-privacy" class="privacy-copy">{{ t('needtoken.codexCheck.privacy') }}</p>
        </form>
        <details class="session-help">
          <summary>{{ t('needtoken.codexCheck.howTo') }}<ChevronDown :size="16" :stroke-width="1.5" aria-hidden="true" /></summary>
          <ol>
            <li><a href="https://chatgpt.com/" target="_blank" rel="noopener noreferrer">{{ t('needtoken.codexCheck.signIn') }}<ArrowUpRight :size="13" aria-hidden="true" /></a></li>
            <li><a href="https://chatgpt.com/api/auth/session" target="_blank" rel="noopener noreferrer">{{ t('needtoken.codexCheck.openSession') }}<ArrowUpRight :size="13" aria-hidden="true" /></a></li>
            <li>{{ t('needtoken.codexCheck.pasteSession') }}</li>
          </ol>
        </details>
      </section>

      <section class="account-results" aria-live="polite" :aria-busy="busy" :aria-labelledby="'account-results-title'">
        <div class="panel-heading"><h2 id="account-results-title">{{ t('needtoken.codexCheck.board') }}</h2><span v-if="result" class="verified-label"><span class="result-dot" />{{ t('needtoken.codexCheck.verified') }}</span></div>
        <div v-if="busy" class="result-loading" role="status">
          <p class="loading-label"><LoaderCircle :size="18" :stroke-width="1.5" class="loading-icon" aria-hidden="true" />{{ t('needtoken.codexCheck.loading') }}</p>
          <div class="result-skeleton" aria-hidden="true"><span class="skeleton-select" /><div><span /><span /></div><div><span /><span /></div><span class="skeleton-usage" /></div>
        </div>
        <div v-if="requestError" class="request-error" role="alert"><CircleAlert :size="18" :stroke-width="1.5" aria-hidden="true" /><div><h3>{{ t('needtoken.codexCheck.checkFailed') }}</h3><p>{{ t(`needtoken.codexCheck.errors.${requestError}`) }}</p></div></div>
        <div v-if="!result && !busy && !requestError" class="results-empty"><ScanLine :size="28" :stroke-width="1.5" aria-hidden="true" /><h3>{{ t('needtoken.codexCheck.emptyTitle') }}</h3><p>{{ t('needtoken.codexCheck.emptyBody') }}</p></div>
        <template v-if="result">
          <div class="account-selection">
            <label for="codex-account">{{ t('needtoken.codexCheck.accountLabel', { count: result.accounts.length }) }}</label>
            <select id="codex-account" v-model="selectedAccountId" :disabled="busy">
              <option value="" disabled>{{ t('needtoken.codexCheck.chooseAccount') }}</option>
              <option v-for="account in result.accounts" :key="account.id" :value="account.id">{{ planLabel(account.plan) }} · {{ account.id }}{{ account.isDefault ? ` · ${t('needtoken.codexCheck.defaultAccount')}` : '' }}</option>
            </select>
          </div>
          <template v-if="selectedAccount">
            <dl class="account-facts">
              <div class="plan-fact"><dt>{{ t('needtoken.codexCheck.subscription') }}</dt><dd>{{ planLabel(selectedAccount.plan) }}</dd></div>
              <div><dt>{{ t('needtoken.codexCheck.expiresAt') }}</dt><dd>{{ selectedAccount.expiresAt ? formatDate(selectedAccount.expiresAt) : t('needtoken.codexCheck.notProvided') }}</dd></div>
              <div><dt>{{ t('needtoken.codexCheck.accountId') }}</dt><dd class="account-id" translate="no">{{ selectedAccount.id }}</dd></div>
            </dl>
            <div class="usage-heading"><h3>{{ t('needtoken.codexCheck.usageTitle') }}</h3><button type="button" class="usage-action" :disabled="busy || !hasCredential" @click="requestCheck(selectedAccountId)"><RefreshCw :size="14" :stroke-width="1.5" aria-hidden="true" />{{ t('needtoken.codexCheck.readUsage') }}</button></div>
            <template v-if="selectedUsageStatus === 'ready' && result.usage">
              <div class="usage-windows">
                <div v-for="window in usageWindows" :key="window.id" class="usage-window">
                  <div><span>{{ t(`needtoken.codexCheck.windows.${window.id}`) }}</span><strong>{{ t('needtoken.codexCheck.usedPercent', { percent: formatPercent(window.value.usedPercent) }) }}</strong></div>
                  <progress :value="window.value.usedPercent" max="100" :aria-label="t(`needtoken.codexCheck.windows.${window.id}`)" />
                  <p>{{ windowLabel(window.value.windowSeconds) }}</p>
                  <p>{{ t('needtoken.codexCheck.resetsAt') }} <span>{{ window.value.resetAt ? formatDate(window.value.resetAt) : t('needtoken.codexCheck.notProvided') }}</span></p>
                </div>
              </div>
            </template>
            <p v-else class="usage-note">{{ t(selectedUsageStatus === 'unavailable' ? 'needtoken.codexCheck.usageUnavailable' : 'needtoken.codexCheck.usagePrompt') }}</p>
            <p class="eligibility-note">{{ t('needtoken.codexCheck.eligibilityNote') }} <a href="https://chatgpt.com/#pricing" target="_blank" rel="noopener noreferrer">{{ t('needtoken.codexCheck.checkPlans') }}<ArrowUpRight :size="12" aria-hidden="true" /></a></p>
          </template>
          <p v-else class="usage-note">{{ t('needtoken.codexCheck.chooseAccountHint') }}</p>
          <div class="result-source"><span>{{ t('needtoken.codexCheck.checkedAt', { time: formatDate(result.checkedAt) }) }}</span><a href="https://chatgpt.com/" target="_blank" rel="noopener noreferrer">{{ t('needtoken.codexCheck.source') }}<ArrowUpRight :size="12" aria-hidden="true" /></a></div>
        </template>
      </section>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { ArrowUpRight, ChevronDown, CircleAlert, LoaderCircle, LockKeyhole, RefreshCw, ScanLine, ShieldCheck, X } from 'lucide-vue-next'
import ToolNavigation from './components/ToolNavigation.vue'
import { usePageSeo } from '../../composables/usePageSeo'
import { checkCodexAccount, CodexCheckError, MAX_CODEX_SESSION_INPUT, parseCodexSessionInput, type CodexAccountResult, type CodexCheckErrorCode, type SessionInputError } from './lib/codexAccountCheck'

const { t, locale } = useI18n()
const sessionInput = ref('')
const inputError = ref<SessionInputError | null>(null)
const requestError = ref<CodexCheckErrorCode | null>(null)
const busy = ref(false)
const result = ref<CodexAccountResult | null>(null)
const selectedAccountId = ref('')
let accessToken = ''
const hasCredential = ref(false)
let controller: AbortController | null = null
let generation = 0
let disposed = false
const selectedAccount = computed(() => result.value?.accounts.find(account => account.id === selectedAccountId.value) ?? null)
const selectedUsageStatus = computed(() => result.value?.selectedAccountId === selectedAccountId.value ? result.value.usageStatus : 'not_requested')
const usageWindows = computed(() => {
  if (selectedUsageStatus.value !== 'ready' || !result.value?.usage) return []
  return (['primary', 'secondary'] as const).flatMap(id => {
    const value = result.value?.usage?.[id]
    return value ? [{ id, value }] : []
  })
})
const planLabel = (plan: string) => ['free', 'go', 'plus', 'pro', 'team', 'business', 'enterprise', 'edu', 'unknown'].includes(plan) ? t(`needtoken.codexCheck.plans.${plan}`) : plan
const formatDate = (value: string) => new Date(value).toLocaleString(locale.value, { year: 'numeric', month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' })
const formatPercent = (value: number) => new Intl.NumberFormat(locale.value, { maximumFractionDigits: 1 }).format(value)
const windowLabel = (seconds: number) => seconds % 86400 === 0 ? t('needtoken.codexCheck.windowDays', { count: seconds / 86400 }) : seconds % 3600 === 0 ? t('needtoken.codexCheck.windowHours', { count: seconds / 3600 }) : t('needtoken.codexCheck.windowMinutes', { count: Math.round(seconds / 60) })
usePageSeo({ title: () => t('needtoken.codexCheck.title'), description: () => t('needtoken.codexCheck.intro'), canonicalPath: () => '/tools/codex-account' })

function clear() {
  generation++
  controller?.abort()
  controller = null
  accessToken = ''
  hasCredential.value = false
  sessionInput.value = ''
  inputError.value = null
  requestError.value = null
  selectedAccountId.value = ''
  result.value = null
  busy.value = false
}
function detect() {
  const parsed = parseCodexSessionInput(sessionInput.value)
  if (!parsed.ok) { inputError.value = parsed.error; return }
  accessToken = parsed.accessToken
  hasCredential.value = true
  sessionInput.value = ''
  inputError.value = null
  result.value = null
  selectedAccountId.value = ''
  void requestCheck()
}
async function requestCheck(accountId?: string) {
  if (!accessToken || busy.value) return
  const ticket = ++generation
  controller?.abort()
  controller = new AbortController()
  busy.value = true
  requestError.value = null
  result.value = null
  try {
    const next = await checkCodexAccount({ accessToken, accountId, locale: locale.value, signal: controller.signal, baseUrl: import.meta.env.VITE_API_BASE_URL || '' })
    if (disposed || ticket !== generation) return
    result.value = next
    if (!next.accounts.some(account => account.id === selectedAccountId.value)) selectedAccountId.value = next.accounts.length === 1 ? next.accounts[0]!.id : ''
  } catch (error) {
    if (disposed || ticket !== generation || (error instanceof DOMException && error.name === 'AbortError')) return
    requestError.value = error instanceof CodexCheckError ? error.code : 'unavailable'
    selectedAccountId.value = ''
    if (requestError.value === 'unauthorized' || requestError.value === 'invalid_input') { accessToken = ''; hasCredential.value = false }
  } finally { if (!disposed && ticket === generation) { busy.value = false; controller = null } }
}
onMounted(() => window.addEventListener('pagehide', clear))
onUnmounted(() => { disposed = true; window.removeEventListener('pagehide', clear); clear() })
</script>

<style scoped>
.account-tool { width: 100%; max-width: 1120px; margin: 0 auto; padding: 32px 24px 64px; }
.tool-heading { margin-bottom: 32px; }
.tool-capability { display: inline-flex; align-items: center; gap: 8px; min-height: 24px; margin-bottom: 12px; color: var(--ui-text-muted); font-size: 12px; line-height: 20px; }
.tool-capability svg { transition: color var(--nt-motion-fast, 120ms) var(--nt-motion-ease, ease-out), transform var(--nt-motion-fast, 120ms) var(--nt-motion-ease, ease-out); }
.tool-heading:hover .tool-capability svg { color: var(--ui-accent); transform: translateY(-1px); }
.tool-heading h1 { font-size: clamp(28px, 3vw, 36px); font-weight: 500; line-height: 1.2; }
.tool-heading > p { margin-top: 12px; color: var(--ui-text-secondary); font-size: 14px; line-height: 1.7; }
.account-layout { display: grid; grid-template-columns: minmax(0, .9fr) minmax(0, 1.1fr); align-items: start; gap: 24px; }
.account-layout.has-credential { grid-template-columns: minmax(240px, .65fr) minmax(0, 1.35fr); }
.account-input-panel, .account-results { min-width: 0; padding: 24px; border: 1px solid var(--ui-border); border-radius: 16px; background: var(--ui-bg-elevated); }
.panel-heading { display: flex; flex-wrap: wrap; align-items: center; justify-content: space-between; gap: 12px; }
.panel-heading h2 { font-size: 16px; font-weight: 500; }
.credential-memory { display: flex; align-items: center; gap: 6px; color: var(--ui-text-muted); font-size: 12px; }
.session-label { display: block; margin-top: 24px; color: var(--ui-text-secondary); font-size: 13px; }
textarea { display: block; width: 100%; min-height: 160px; margin-top: 12px; padding: 16px; resize: vertical; border: 1px solid var(--ui-border-strong); border-radius: 8px; background: var(--ui-bg-page); color: var(--ui-text-primary); font-family: var(--nt-font-mono); font-size: 12px; line-height: 1.7; overflow-wrap: anywhere; transition: border-color var(--nt-motion-fast, 120ms) var(--nt-motion-ease, ease-out), background-color var(--nt-motion-fast, 120ms) var(--nt-motion-ease, ease-out); }
textarea:hover:not(:disabled):not(:focus-visible) { border-color: var(--ui-text-muted); }
textarea:focus-visible { background: var(--ui-bg-elevated); }
textarea::placeholder { color: var(--ui-text-muted); }
textarea[aria-invalid="true"] { border-color: #c9a36b; }
.privacy-copy { margin-top: 16px; color: var(--ui-text-muted); font-size: 12px; line-height: 1.8; }
.form-actions { display: flex; flex-wrap: wrap; gap: 12px; margin-top: 20px; }
.form-actions .primary-action { flex: 1; }
.primary-action, .secondary-action { display: inline-flex; justify-content: center; align-items: center; gap: 8px; min-height: 44px; padding: 12px 16px; border: 1px solid var(--ui-border-strong); border-radius: 8px; font-size: 13px; font-weight: 500; transition: color var(--nt-motion-fast, 120ms) var(--nt-motion-ease, ease-out), border-color var(--nt-motion-fast, 120ms) var(--nt-motion-ease, ease-out), background-color var(--nt-motion-fast, 120ms) var(--nt-motion-ease, ease-out), transform var(--nt-motion-fast, 120ms) var(--nt-motion-ease, ease-out); }
.primary-action { color: var(--ui-text-on-accent); background: var(--ui-accent); border-color: transparent; }
.primary-action:hover:not(:disabled) { background: var(--ui-accent-hover); }
.secondary-action { color: var(--ui-text-secondary); background: transparent; }
.secondary-action:hover:not(:disabled) { color: var(--ui-text-primary); background: var(--ui-bg-soft); }
.primary-action:active:not(:disabled), .secondary-action:active:not(:disabled) { transform: translateY(1px); }
.primary-action > svg, .usage-action > svg { flex: none; transition: transform var(--nt-motion-fast, 120ms) var(--nt-motion-ease, ease-out); }
.primary-action:hover:not(:disabled) > svg:not(.loading-icon), .primary-action:focus-visible > svg:not(.loading-icon), .usage-action:hover:not(:disabled) > svg, .usage-action:focus-visible > svg { transform: translateX(2px); }
button:disabled { cursor: not-allowed; opacity: .5; }
.credential-ready { display: flex; align-items: flex-start; gap: 8px; margin-top: 24px; color: var(--ui-text-secondary); font-size: 13px; line-height: 1.8; }
.credential-ready svg { flex: none; margin-top: 4px; color: var(--ui-accent); }
.inline-error { margin-top: 12px; color: #e5bb83; font-size: 12px; line-height: 1.7; }
.session-help { margin-top: 24px; padding-top: 24px; border-top: 1px solid var(--ui-border); }
.session-help summary { display: flex; justify-content: space-between; align-items: center; gap: 12px; color: var(--ui-text-secondary); font-size: 13px; cursor: pointer; list-style: none; }
.session-help summary::-webkit-details-marker { display: none; }
.session-help summary > svg { flex: none; color: var(--ui-text-muted); transition: transform var(--nt-motion-fast, 120ms) var(--nt-motion-ease, ease-out), color var(--nt-motion-fast, 120ms) var(--nt-motion-ease, ease-out); }
.session-help summary:hover, .session-help summary:focus-visible { color: var(--ui-text-primary); }
.session-help[open] summary > svg { transform: rotate(180deg); }
.session-help ol { display: grid; gap: 12px; margin: 16px 0 0 20px; color: var(--ui-text-muted); font-size: 12px; line-height: 1.8; list-style: decimal; }
.session-help a { display: inline-flex; align-items: center; gap: 4px; color: var(--ui-text-secondary); }
.session-help a:hover { color: var(--ui-accent); }
.account-results { min-height: 320px; }
.has-credential .account-results { min-height: 0; }
.has-credential .account-input-panel { padding: 20px; }
.has-credential .credential-ready { margin-top: 16px; }
.has-credential .session-help { margin-top: 20px; padding-top: 20px; }
.has-credential .form-actions { gap: 8px; margin-top: 20px; }
.has-credential .form-actions button { flex: 1; padding-inline: 12px; }
.verified-label { display: flex; align-items: center; gap: 8px; color: var(--ui-text-secondary); font-size: 12px; }
.result-dot { width: 6px; height: 6px; border-radius: 50%; background: var(--ui-accent); }
.result-loading { min-height: 224px; padding-top: 24px; color: var(--ui-text-muted); }
.loading-label { display: flex; align-items: center; gap: 8px; }
.results-empty { display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 16px; min-height: 224px; color: var(--ui-text-muted); text-align: center; }
.result-skeleton { display: grid; gap: 20px; margin-top: 24px; }
.result-skeleton > span, .result-skeleton > div > span { display: block; height: 12px; border-radius: 4px; background: var(--ui-bg-soft); animation: account-pending 1.4s ease-in-out infinite; }
.result-skeleton .skeleton-select { width: 100%; height: 40px; border-radius: 8px; }
.result-skeleton > div { display: flex; justify-content: space-between; gap: 24px; }
.result-skeleton > div > span { width: 28%; }
.result-skeleton > div > span:last-child { width: 40%; }
.result-skeleton .skeleton-usage { width: 72%; }
.results-empty h3 { color: var(--ui-text-secondary); font-size: 16px; font-weight: 400; }
.results-empty p, .result-loading p { max-width: 32ch; font-size: 13px; line-height: 1.8; }
.loading-icon { animation: check-spin .8s linear infinite; }
.request-error { display: flex; align-items: flex-start; gap: 12px; margin-top: 24px; padding: 16px; border: 1px solid #a27d4340; border-radius: 8px; color: #e5bb83; background: #a27d430a; }
.request-error > svg { flex: none; margin-top: 2px; }
.request-error h3 { font-size: 14px; font-weight: 500; }
.request-error p { margin-top: 8px; font-size: 13px; line-height: 1.7; }
.account-selection { margin-top: 24px; }
.account-selection label { display: block; color: var(--ui-text-muted); font-size: 12px; }
.account-selection select { width: 100%; min-height: 44px; margin-top: 8px; padding: 12px; border: 1px solid var(--ui-border-strong); border-radius: 8px; color: var(--ui-text-primary); background: var(--ui-bg-soft); font-size: 12px; text-overflow: ellipsis; transition: border-color var(--nt-motion-fast, 120ms) var(--nt-motion-ease, ease-out); }
.account-selection select:hover:not(:disabled) { border-color: var(--ui-text-muted); }
.account-facts { margin-top: 24px; }
.account-facts > div { display: flex; justify-content: space-between; align-items: baseline; gap: 16px; padding-block: 12px; font-size: 13px; }
.account-facts dt { flex: none; color: var(--ui-text-muted); }
.account-facts dd { min-width: 0; color: var(--ui-text-secondary); text-align: right; overflow-wrap: anywhere; }
.account-facts .plan-fact { padding: 16px; margin-bottom: 12px; border-radius: 8px; background: var(--ui-bg-soft); }
.plan-fact dd { color: var(--ui-text-primary); font-size: 28px; font-weight: 500; }
.account-id { font-family: var(--nt-font-mono); font-size: 12px; }
.usage-heading { display: flex; align-items: center; justify-content: space-between; gap: 16px; margin-top: 24px; padding-top: 24px; border-top: 1px solid var(--ui-border); }
.usage-heading h3 { font-size: 14px; font-weight: 500; }
.usage-action { display: inline-flex; align-items: center; gap: 8px; min-height: 36px; padding: 8px; margin-right: -8px; border-radius: 8px; color: var(--ui-accent); font-size: 12px; transition: color var(--nt-motion-fast, 120ms) var(--nt-motion-ease, ease-out), background-color var(--nt-motion-fast, 120ms) var(--nt-motion-ease, ease-out), transform var(--nt-motion-fast, 120ms) var(--nt-motion-ease, ease-out); }
.usage-action:hover:not(:disabled) { background: var(--ui-bg-soft); }
.usage-action:active:not(:disabled) { transform: translateY(1px); }
.usage-windows { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 12px; margin-top: 16px; }
.usage-window { min-width: 0; padding: 16px; border: 1px solid var(--ui-border); border-radius: 8px; }
.usage-window > div { display: flex; flex-wrap: wrap; justify-content: space-between; gap: 8px; font-size: 13px; }
.usage-window strong { font-weight: 500; font-variant-numeric: tabular-nums; }
progress { display: block; width: 100%; height: 4px; margin: 12px 0; appearance: none; border: 0; border-radius: 2px; background: var(--ui-bg-soft); accent-color: var(--ui-accent); }
progress::-webkit-progress-bar { border-radius: 2px; background: var(--ui-bg-soft); }
progress::-webkit-progress-value { border-radius: 2px; background: var(--ui-accent); }
progress::-moz-progress-bar { border-radius: 2px; background: var(--ui-accent); }
.usage-window p { margin-top: 8px; color: var(--ui-text-muted); font-size: 12px; line-height: 1.7; }
.usage-window p span { color: var(--ui-text-secondary); }
.usage-note, .eligibility-note { margin-top: 16px; color: var(--ui-text-muted); font-size: 12px; line-height: 1.8; }
.eligibility-note a { display: inline-flex; align-items: center; gap: 4px; color: var(--ui-text-secondary); }
.result-source { display: flex; flex-wrap: wrap; justify-content: space-between; gap: 12px; margin-top: 24px; padding-top: 16px; border-top: 1px solid var(--ui-border); color: var(--ui-text-muted); font-size: 11px; line-height: 1.8; }
.result-source a { display: inline-flex; align-items: center; gap: 4px; }
.account-tool a { transition: color var(--nt-motion-fast, 120ms) var(--nt-motion-ease, ease-out); }
.account-tool a:hover { color: var(--ui-accent); }
.account-tool a > svg { flex: none; transition: transform var(--nt-motion-fast, 120ms) var(--nt-motion-ease, ease-out); }
.account-tool a:hover > svg, .account-tool a:focus-visible > svg { transform: translate(1px, -1px); }
.account-tool :is(a, button, select, textarea, summary):focus-visible { outline: 2px solid var(--ui-accent); outline-offset: 4px; }
@keyframes check-spin { to { transform: rotate(360deg); } }
@keyframes account-pending { 50% { opacity: .45; } }
@media (max-width: 800px) { .account-layout, .account-layout.has-credential { grid-template-columns: 1fr; } }
@media (max-width: 640px) { .account-tool { padding: 24px 20px 48px; } .tool-heading { margin-bottom: 24px; } }
@media (max-width: 640px) { .usage-windows { grid-template-columns: 1fr; } }
@media (max-width: 480px) { .account-input-panel, .account-results { padding: 20px; } .account-facts > div { flex-direction: column; align-items: flex-start; gap: 8px; } .account-facts dd { text-align: left; } .usage-heading { gap: 12px; } }
@media (prefers-reduced-motion: reduce) {
  .account-tool :deep(*) { transition: none !important; animation: none !important; }
  .account-tool :is(.primary-action, .secondary-action, .usage-action):active:not(:disabled),
  .account-tool .primary-action:hover:not(:disabled) > svg:not(.loading-icon),
  .account-tool .primary-action:focus-visible > svg:not(.loading-icon),
  .account-tool .usage-action:hover:not(:disabled) > svg,
  .account-tool .usage-action:focus-visible > svg,
  .account-tool a:hover > svg,
  .account-tool a:focus-visible > svg,
  .tool-heading:hover .tool-capability svg { transform: none; }
  .session-help[open] summary > svg { transform: rotate(180deg); }
}
</style>
