<template>
  <div class="account-tool">
    <header class="tool-heading">
      <h1>{{ t('needtoken.codexCheck.heroTitle') }}</h1>
      <p>{{ t('needtoken.codexCheck.heroIntro') }}</p>
    </header>

    <div class="account-layout" :class="{ 'has-credential': hasCredential }">
      <section class="account-input-panel" aria-labelledby="session-input-title">
        <div class="panel-heading">
          <h2 id="session-input-title"><ProductBrandMark brand="gpt" />ChatGPT Session</h2>
          <span class="credential-memory"><LockKeyhole v-if="hasCredential" :size="16" aria-hidden="true" /><Clipboard v-else :size="16" aria-hidden="true" />{{ t(hasCredential ? 'needtoken.codexCheck.memoryOnly' : 'needtoken.codexCheck.waitingPaste') }}</span>
        </div>
        <form @submit.prevent="detect">
          <div v-if="!hasCredential" class="session-paste-body">
            <h3>{{ t('needtoken.codexCheck.inputTitle') }} <kbd>{{ pasteShortcut }}</kbd></h3>
            <ol class="session-steps">
              <li><span>{{ t('needtoken.codexCheck.openStep') }}</span><a href="https://chatgpt.com/api/auth/session" target="_blank" rel="noopener noreferrer"><span translate="no">chatgpt.com/api/auth/session</span><ArrowUpRight :size="15" aria-hidden="true" /></a><button type="button" class="copy-session-link" :aria-label="t('needtoken.codexCheck.copySessionLink')" @click="copySessionLink"><Check v-if="linkCopied" :size="15" aria-hidden="true" /><Copy v-else :size="15" aria-hidden="true" /></button></li>
              <li>{{ t('needtoken.codexCheck.copyStep') }}</li>
            </ol>
            <template v-if="manualInput">
              <label class="session-label" for="codex-session">{{ t('needtoken.codexCheck.inputLabel') }}</label>
              <textarea id="codex-session" ref="inputElement" v-model="sessionInput" :maxlength="MAX_CODEX_SESSION_INPUT" :placeholder="t('needtoken.codexCheck.inputPlaceholder')" :disabled="busy" :aria-invalid="!!inputError" :aria-describedby="inputError ? 'session-error session-privacy' : 'session-privacy'" autocomplete="off" autocapitalize="off" autocorrect="off" spellcheck="false" data-1p-ignore data-lpignore="true" @input="handleInput" />
            </template>
            <p v-if="clipboardUnavailable" class="clipboard-note" role="status">{{ t('needtoken.codexCheck.manualHint') }}</p>
            <p v-if="inputError" id="session-error" class="inline-error" role="alert">{{ t(`needtoken.codexCheck.inputErrors.${inputError}`) }}</p>
            <div class="form-actions">
              <button v-if="manualInput && sessionInput.trim()" type="submit" class="primary-action" :disabled="busy"><ScanLine :size="19" aria-hidden="true" />{{ t('needtoken.codexCheck.detect') }}</button>
              <button v-else type="button" class="primary-action" :disabled="readingClipboard" :aria-busy="readingClipboard" @click="pasteFromClipboard"><LoaderCircle v-if="readingClipboard" :size="19" class="loading-icon" aria-hidden="true" /><Clipboard v-else :size="19" aria-hidden="true" />{{ t(readingClipboard ? 'needtoken.common.loading' : 'needtoken.codexCheck.pasteAction') }}</button>
              <button v-if="readingClipboard" type="button" class="secondary-action" @click="clear">{{ t('needtoken.codexCheck.cancel') }}</button>
            </div>
            <button v-if="!manualInput" type="button" class="manual-input-action" @click="showManualInput">{{ t('needtoken.codexCheck.manualPaste') }}</button>
          </div>
          <div v-else class="credential-body">
            <p class="credential-ready"><ShieldCheck :size="18" :stroke-width="1.5" aria-hidden="true" />{{ t('needtoken.codexCheck.credentialReady') }}</p>
            <div class="form-actions">
              <button type="button" class="primary-action" :disabled="busy" :aria-busy="busy" @click="requestCheck()"><LoaderCircle v-if="busy" :size="16" class="loading-icon" aria-hidden="true" /><RefreshCw v-else :size="16" aria-hidden="true" />{{ t(busy ? 'needtoken.codexCheck.loading' : 'needtoken.codexCheck.detectAgain') }}</button>
              <button type="button" class="secondary-action" @click="clear"><X v-if="busy" :size="16" aria-hidden="true" />{{ t(busy ? 'needtoken.codexCheck.cancel' : 'needtoken.codexCheck.clear') }}</button>
            </div>
          </div>
        </form>
      </section>
      <div id="session-privacy" class="privacy-note"><p><LockKeyhole :size="15" aria-hidden="true" />{{ t('needtoken.codexCheck.privacyShort') }}</p><p>{{ t('needtoken.codexCheck.privacy') }}</p></div>

      <section v-if="hasCredential || busy || result || requestError" class="account-results" aria-live="polite" :aria-busy="busy" :aria-labelledby="'account-results-title'">
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
import { computed, nextTick, onMounted, onUnmounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { ArrowUpRight, Check, CircleAlert, Clipboard, Copy, LoaderCircle, LockKeyhole, RefreshCw, ScanLine, ShieldCheck, X } from 'lucide-vue-next'
import ProductBrandMark from './components/ProductBrandMark.vue'
import { usePageSeo } from '../../composables/usePageSeo'
import { checkCodexAccount, CodexCheckError, MAX_CODEX_SESSION_INPUT, parseCodexSessionInput, type CodexAccountResult, type CodexCheckErrorCode, type SessionInputError } from './lib/codexAccountCheck'

const { t, locale } = useI18n()
const sessionInput = ref('')
const inputElement = ref<HTMLTextAreaElement | null>(null)
const manualInput = ref(false)
const clipboardUnavailable = ref(false)
const readingClipboard = ref(false)
const linkCopied = ref(false)
const pasteShortcut = ref('Ctrl V')
let copyTimer: ReturnType<typeof setTimeout> | undefined
let clipboardGeneration = 0
let copyGeneration = 0
function invalidateClipboardRead() { clipboardGeneration++; readingClipboard.value = false }
function handleInput() { invalidateClipboardRead(); inputError.value = null }
async function showManualInput() {
  invalidateClipboardRead()
  manualInput.value = true
  await nextTick()
  if (!disposed && manualInput.value) inputElement.value?.focus()
}
async function pasteFromClipboard() {
  if (disposed || readingClipboard.value || hasCredential.value || busy.value) return
  const ticket = ++clipboardGeneration
  readingClipboard.value = true
  try {
    const text = await navigator.clipboard.readText()
    if (disposed || ticket !== clipboardGeneration) return
    sessionInput.value = text
    if (!sessionInput.value.trim()) { clipboardUnavailable.value = true; await showManualInput(); return }
    detect()
    if (inputError.value) await showManualInput()
  } catch {
    if (!disposed && ticket === clipboardGeneration) { clipboardUnavailable.value = true; await showManualInput() }
  } finally {
    if (ticket === clipboardGeneration) readingClipboard.value = false
  }
}
function handlePaste(event: ClipboardEvent) {
  if (hasCredential.value || busy.value) return
  const text = event.clipboardData?.getData('text')
  if (!text) return
  event.preventDefault()
  invalidateClipboardRead()
  sessionInput.value = text
  detect()
  if (inputError.value) void showManualInput()
}
async function copySessionLink() {
  const ticket = ++copyGeneration
  try {
    await navigator.clipboard.writeText('https://chatgpt.com/api/auth/session')
    if (disposed || ticket !== copyGeneration) return
    linkCopied.value = true
    if (copyTimer !== undefined) clearTimeout(copyTimer)
    copyTimer = setTimeout(() => { if (ticket === copyGeneration) linkCopied.value = false }, 1800)
  } catch { /* The adjacent official link remains available when clipboard access is denied. */ }
}
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
  invalidateClipboardRead()
  copyGeneration++
  if (copyTimer !== undefined) clearTimeout(copyTimer)
  copyTimer = undefined
  linkCopied.value = false
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
  manualInput.value = false
  clipboardUnavailable.value = false
}
function detect() {
  if (disposed || hasCredential.value || busy.value) return
  invalidateClipboardRead()
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
onMounted(() => { pasteShortcut.value = /Mac|iPhone|iPad/.test(navigator.platform) ? '⌘ V' : 'Ctrl V'; window.addEventListener('pagehide', clear); window.addEventListener('paste', handlePaste) })
onUnmounted(() => { disposed = true; window.removeEventListener('pagehide', clear); window.removeEventListener('paste', handlePaste); clear() })
</script>

<style scoped>
.account-tool { width: 100%; max-width: 1024px; margin: 0 auto; padding: 56px 24px 72px; }
.tool-heading { max-width: 680px; margin: 0 auto 40px; text-align: center; }
.tool-heading h1 { font-size: clamp(40px, 6vw, 72px); font-weight: 400; line-height: 1.15; letter-spacing: -.045em; }
.tool-heading > p { margin-top: 20px; color: var(--ui-text-secondary); font-size: clamp(16px, 2vw, 21px); line-height: 1.8; text-wrap: balance; }
.account-layout { display: flex; flex-direction: column; gap: 24px; }
.account-input-panel, .account-results { width: 100%; min-width: 0; border: 1px solid var(--ui-border); border-radius: 24px; background: var(--ui-bg-elevated); }
.account-input-panel { overflow: hidden; }
.panel-heading { display: flex; flex-wrap: wrap; align-items: center; justify-content: space-between; gap: 12px; }
.account-input-panel > .panel-heading { padding: 24px 28px; border-bottom: 1px solid var(--ui-border); }
.panel-heading h2 { display: flex; align-items: center; gap: 10px; font-size: 16px; font-weight: 500; }
.panel-heading :deep(.product-brand-mark) { --product-brand-ink: var(--ui-text-primary); }
.credential-memory { display: flex; align-items: center; gap: 8px; color: var(--ui-text-secondary); font-size: 12px; }
.session-paste-body { padding: 40px 28px 24px; text-align: center; }
.session-paste-body h3 { display: flex; align-items: center; justify-content: center; gap: 12px; font-size: 21px; font-weight: 500; }
kbd { display: inline-flex; align-items: center; justify-content: center; min-height: 28px; padding: 3px 8px; border: 1px solid var(--ui-border-strong); border-radius: 6px; color: var(--ui-text-muted); font-family: var(--nt-font-mono); font-size: 12px; font-weight: 400; }
.session-steps { display: grid; justify-content: center; gap: 16px; width: max-content; max-width: 100%; margin: 24px auto 36px; padding-left: 24px; color: var(--ui-text-secondary); font-size: 15px; line-height: 1.8; list-style: decimal; text-align: left; }
.session-steps > li { padding-left: 8px; }
.session-steps li::marker { color: var(--ui-text-muted); font-family: var(--nt-font-mono); font-size: 12px; }
.session-steps a { display: inline-flex; align-items: center; gap: 6px; max-width: 100%; margin-left: 8px; padding: 4px 8px; border-radius: 8px; background: var(--ui-bg-soft); color: var(--ui-text-primary); vertical-align: middle; }
.session-steps a > span { min-width: 0; overflow-wrap: anywhere; font-family: var(--nt-font-mono); font-size: 14px; }
.copy-session-link { display: inline-flex; align-items: center; justify-content: center; width: 32px; height: 32px; margin-left: 4px; border-radius: 6px; color: var(--ui-text-muted); vertical-align: middle; }
.copy-session-link:hover { color: var(--ui-accent); background: var(--ui-bg-soft); }
.session-label { display: block; margin-top: 24px; color: var(--ui-text-secondary); font-size: 13px; text-align: left; }
textarea { display: block; width: 100%; min-height: 128px; margin-top: 12px; padding: 16px; resize: vertical; border: 1px solid var(--ui-border-strong); border-radius: 12px; background: var(--ui-bg-page); color: var(--ui-text-primary); font-family: var(--nt-font-mono); font-size: 12px; line-height: 1.7; overflow-wrap: anywhere; transition: border-color 160ms ease; }
textarea:hover:not(:disabled):not(:focus-visible) { border-color: var(--ui-text-muted); }
textarea::placeholder { color: var(--ui-text-muted); }
textarea[aria-invalid="true"] { border-color: #c9a36b; }
.privacy-note { text-align: center; color: var(--ui-text-muted); }
.privacy-note p:first-child { display: flex; align-items: center; justify-content: center; gap: 8px; color: var(--ui-text-secondary); font-size: 13px; }
.privacy-note p + p { max-width: 680px; margin: 10px auto 0; font-size: 12px; line-height: 1.8; }
.privacy-note svg { flex: none; }
.form-actions { display: flex; justify-content: center; flex-wrap: wrap; gap: 12px; margin-top: 24px; }
.primary-action, .secondary-action { display: inline-flex; justify-content: center; align-items: center; gap: 10px; min-height: 48px; padding: 12px 28px; border: 1px solid var(--ui-border-strong); border-radius: 999px; font-size: 15px; font-weight: 500; transition: color 160ms ease, border-color 160ms ease, background-color 160ms ease, transform 160ms ease; }
.session-paste-body .primary-action { min-width: 240px; }
.primary-action { color: var(--ui-text-on-accent); background: var(--ui-accent); border-color: transparent; }
.primary-action:hover:not(:disabled) { background: var(--ui-accent-hover); transform: translateY(-1px); }
.primary-action:active:not(:disabled), .secondary-action:active:not(:disabled) { transform: scale(.98); }
.secondary-action { color: var(--ui-text-secondary); background: transparent; font-size: 13px; }
.secondary-action:hover:not(:disabled) { color: var(--ui-text-primary); background: var(--ui-bg-soft); }
.manual-input-action { display: inline-flex; justify-content: center; min-height: 36px; margin-top: 12px; padding: 8px 12px; border-radius: 8px; color: var(--ui-text-muted); font-size: 12px; }
.manual-input-action:hover { color: var(--ui-text-primary); background: var(--ui-bg-soft); }
button:disabled { cursor: not-allowed; opacity: .5; }
.credential-body { padding: 24px 28px; }
.credential-ready { display: flex; align-items: flex-start; gap: 8px; color: var(--ui-text-secondary); font-size: 13px; line-height: 1.8; }
.credential-ready svg { flex: none; margin-top: 3px; color: var(--ui-accent); }
.inline-error { margin-top: 12px; color: #e5bb83; font-size: 12px; line-height: 1.7; }
.clipboard-note { margin-top: 12px; color: var(--ui-text-muted); font-size: 12px; line-height: 1.7; }
.account-results { padding: 28px; }
.account-results { min-height: 320px; }
.has-credential .account-results { min-height: 0; }
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
@media (max-width: 640px) { .account-tool { padding: 40px 20px 48px; } .tool-heading { margin-bottom: 28px; } .account-input-panel > .panel-heading { padding: 20px; } .session-paste-body { padding: 32px 20px 20px; } .session-steps { font-size: 13px; padding-left: 18px; } .session-steps a { margin-left: 0; margin-top: 6px; } .session-steps a > span { font-size: 12px; } .copy-session-link { display: none; } .session-paste-body .primary-action { width: 100%; min-width: 0; } .privacy-note p:first-child { align-items: flex-start; font-size: 12px; } .privacy-note svg { margin-top: 2px; } }
@media (max-width: 640px) { .usage-windows { grid-template-columns: 1fr; } }
@media (max-width: 480px) { .account-results { padding: 20px; } .account-facts > div { flex-direction: column; align-items: flex-start; gap: 8px; } .account-facts dd { text-align: left; } .usage-heading { gap: 12px; } }
@media (prefers-reduced-motion: reduce) {
  .account-tool :deep(*) { transition: none !important; animation: none !important; }
  .account-tool :is(.primary-action, .secondary-action, .usage-action):active:not(:disabled),
  .account-tool .primary-action:hover:not(:disabled) > svg:not(.loading-icon),
  .account-tool .primary-action:focus-visible > svg:not(.loading-icon),
  .account-tool .usage-action:hover:not(:disabled) > svg,
  .account-tool .usage-action:focus-visible > svg,
  .account-tool a:hover > svg,
  .account-tool a:focus-visible > svg,
  .account-tool .primary-action:hover:not(:disabled) { transform: none; }
}
</style>
