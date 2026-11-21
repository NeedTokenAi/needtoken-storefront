<template>
  <section id="api-connect" class="api-connect" aria-labelledby="api-connect-title">
    <header class="connect-heading">
      <div><p>{{ t('needtoken.apiConnect.setupLabel') }}</p><h2 id="api-connect-title">{{ t('needtoken.apiConnect.setupTitle') }}</h2><p class="connect-intro">{{ t('needtoken.apiConnect.setupIntro') }}</p></div>
      <button v-if="userAuth.isAuthenticated && !appStore.isResellerTenant" type="button" class="connect-key-btn" :disabled="ssoLoading" @click="openConsoleKeys">{{ t('needtoken.apiConnect.getKey') }}<ArrowUpRight :size="16" aria-hidden="true" /></button>
      <a v-else :href="CONSOLE_URL" target="_blank" rel="noopener noreferrer">{{ t('needtoken.apiConnect.getKey') }}<ArrowUpRight :size="16" aria-hidden="true" /></a>
    </header>

    <div class="connect-toolbar">
    <div class="scenario-tabs" role="tablist" :aria-label="t('needtoken.apiConnect.scenariosLabel')">
      <button v-for="(item, index) in scenarios" :id="`api-scenario-${item.id}`" :key="item.id" :data-api-scenario="item.id" type="button" role="tab" :aria-selected="scenario === item.id" aria-controls="api-scenario-panel" :tabindex="scenario === item.id ? 0 : -1" @click="selectScenario(item.id)" @keydown="onScenarioKey($event, index)">
        <component :is="item.icon" :size="17" :stroke-width="1.5" aria-hidden="true" /><span>{{ t(`needtoken.apiConnect.scenarios.${item.id}.title`) }}</span>
      </button>
    </div>
    <a class="connect-reference" :href="CC_SWITCH_REFERENCE_URL" target="_blank" rel="noopener noreferrer">{{ t('needtoken.apiConnect.reference') }}<ArrowUpRight :size="13" aria-hidden="true" /></a>
    </div>

    <div id="api-scenario-panel" class="integration-workspace" role="tabpanel" :aria-labelledby="`api-scenario-${scenario}`">
      <ul class="integration-options" :aria-label="t('needtoken.apiConnect.chooseTool')">
        <li v-for="tool in scenarioTools" :key="tool.id" :class="{ 'is-selected': selected.id === tool.id }">
          <span class="integration-icon" aria-hidden="true"><component :is="toolIcons[tool.id]" :size="22" :stroke-width="1.5" /></span>
          <div class="integration-summary">
            <a :href="tool.get || tool.docs" target="_blank" rel="noopener noreferrer"><strong translate="no">{{ tool.name }}</strong><ArrowUpRight :size="13" aria-hidden="true" /></a>
            <p>{{ t(`needtoken.apiConnect.tools.${tool.id}.summary`) }}</p>
          </div>
          <button :data-api-tool="tool.id" type="button" :aria-pressed="selected.id === tool.id" aria-controls="api-integration-detail" :aria-label="t('needtoken.apiConnect.configureTool', { name: tool.name })" @click="selectedId = tool.id"><Settings2 :size="15" aria-hidden="true" /><span>{{ t('needtoken.apiConnect.configure') }}</span></button>
        </li>
      </ul>

      <article id="api-integration-detail" class="integration-detail" :aria-label="selected.name">
        <header class="integration-heading">
          <div><p>{{ t(`needtoken.apiConnect.tools.${selected.id}.kind`) }}</p><h3 translate="no">{{ selected.name }}</h3></div>
          <div class="integration-links">
            <a v-if="selected.get" :href="selected.get" target="_blank" rel="noopener noreferrer">{{ t(`needtoken.apiConnect.tools.${selected.id}.get`) }}<ArrowUpRight :size="14" aria-hidden="true" /></a>
            <a v-if="selected.docs" :href="selected.docs" target="_blank" rel="noopener noreferrer">{{ t('needtoken.apiConnect.docs') }}<ArrowUpRight :size="14" aria-hidden="true" /></a>
          </div>
        </header>
        <p class="integration-description">{{ t(`needtoken.apiConnect.tools.${selected.id}.description`) }}</p>
        <div v-if="selected.id === 'ccswitch'" class="quick-config">
          <div><strong>{{ t('needtoken.apiConnect.quickConfigTitle') }}</strong><p>{{ t('needtoken.apiConnect.quickConfigHint') }}</p></div>
          <a :href="CONSOLE_URL" target="_blank" rel="noopener noreferrer" data-api-quick-config><Zap :size="15" aria-hidden="true" />{{ t('needtoken.apiConnect.quickConfig') }}<ArrowUpRight :size="14" aria-hidden="true" /></a>
        </div>
        <ol class="integration-steps">
          <li v-for="step in 3" :key="step"><span aria-hidden="true">{{ step }}</span><p>{{ t(`needtoken.apiConnect.tools.${selected.id}.step${step}`) }}</p></li>
        </ol>

        <div class="integration-fields">
          <label for="api-integration-endpoint">{{ t('needtoken.apiConnect.endpoint') }}</label>
          <div class="endpoint-control"><input id="api-integration-endpoint" :value="endpoint" readonly spellcheck="false" @focus="($event.target as HTMLInputElement).select()" /><button data-api-copy-endpoint type="button" :aria-label="t('needtoken.apiConnect.copyEndpoint')" @click="copyEndpoint"><Check v-if="copied" :size="16" aria-hidden="true" /><Copy v-else :size="16" aria-hidden="true" /></button></div>
          <p v-if="copyFailed" class="copy-feedback" role="status">{{ t('needtoken.apiConnect.copyFailed') }}</p>
          <label v-if="selected.id !== 'cloudflare'" for="api-integration-model">{{ t('needtoken.apiConnect.modelId') }}</label>
          <input v-if="selected.id !== 'cloudflare'" id="api-integration-model" v-model="model" type="text" maxlength="200" spellcheck="false" :aria-invalid="!validModel" aria-describedby="api-model-hint" />
          <p v-if="selected.id !== 'cloudflare'" id="api-model-hint" class="field-hint" :class="{ 'is-invalid': !validModel }">{{ t(validModel ? 'needtoken.apiConnect.modelHint' : 'needtoken.apiConnect.modelInvalid') }} <RouterLink to="/models">{{ t('needtoken.nav.models') }}<ArrowUpRight :size="12" aria-hidden="true" /></RouterLink></p>
        </div>

        <template v-if="template">
          <div class="template-heading"><h4>{{ t(selected.filename ? 'needtoken.apiConnect.configTemplate' : 'needtoken.apiConnect.configFields') }}</h4><button v-if="selected.filename" data-api-download type="button" @click="downloadTemplate"><Download :size="14" aria-hidden="true" />{{ t('needtoken.apiConnect.downloadConfig') }}</button></div>
          <div data-api-template><CodeBlock :title="selected.filename || 'NeedToken'" :code="template" /></div>
        </template>
        <p v-if="selected.id === 'opencode' || selected.id === 'openclaw'" class="environment-hint"><code>NEEDTOKEN_API_KEY=&lt;YOUR_NEEDTOKEN_API_KEY&gt;</code><span>{{ t('needtoken.apiConnect.environmentHint') }}</span></p>
        <p class="integration-note">{{ t(`needtoken.apiConnect.tools.${selected.id}.note`) }}</p>
      </article>
    </div>
    <p class="connect-compatibility">{{ t('needtoken.apiConnect.compatibility') }}</p>
  </section>
</template>

<script setup lang="ts">
import { computed, nextTick, onUnmounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { ArrowUpRight, Bot, Check, Cloud, Code2, Copy, Download, Network, Server, Settings2, Zap } from 'lucide-vue-next'
import { API_BASE_URL, CONSOLE_URL, DEFAULT_MODELS } from '../config'
import { API_INTEGRATIONS, CC_SWITCH_REFERENCE_URL, integrationEndpoint, integrationModel, integrationTemplate, type IntegrationId, type IntegrationScenario } from '../lib/apiIntegrations'
import { useUserAuthStore } from '../../../stores/userAuth'
import { useAppStore } from '../../../stores/app'
import { useApiAccountLink } from '../composables/useApiAccountLink'
import CodeBlock from './CodeBlock.vue'

const { t } = useI18n()
const userAuth = useUserAuthStore()
const appStore = useAppStore()
const { ssoLoading, openApiConsoleSSO } = useApiAccountLink()
const openConsoleKeys = () => {
  void openApiConsoleSSO('/keys', '/api-services')
}
const scenarios = [
  { id: 'coding', icon: Code2 },
  { id: 'local', icon: Network },
  { id: 'cloud', icon: Cloud },
  { id: 'agent', icon: Bot },
] as const
const toolIcons = { opencode: Code2, ccswitch: Network, sub2api: Server, cloudflare: Cloud, openclaw: Bot }
const scenario = ref<IntegrationScenario>('coding')
const selectedId = ref<IntegrationId>('opencode')
const model = ref<string>(DEFAULT_MODELS[0])
const validModel = computed(() => integrationModel(model.value))
const scenarioTools = computed(() => API_INTEGRATIONS.filter(tool => tool.scenario === scenario.value))
const selected = computed(() => scenarioTools.value.find(tool => tool.id === selectedId.value) || scenarioTools.value[0]!)
const endpoint = computed(() => integrationEndpoint(selected.value.id, API_BASE_URL))
const template = computed(() => integrationTemplate(selected.value.id, API_BASE_URL, model.value))
const copiedEndpoint = ref(''), copyFailed = ref(false)
const copied = computed(() => copiedEndpoint.value === endpoint.value)
let disposed = false
let copyTimer: ReturnType<typeof setTimeout> | undefined

function selectScenario(value: IntegrationScenario) {
  scenario.value = value
  selectedId.value = API_INTEGRATIONS.find(tool => tool.scenario === value)!.id
}
async function onScenarioKey(event: KeyboardEvent, index: number) {
  const next = event.key === 'ArrowRight' ? (index + 1) % scenarios.length : event.key === 'ArrowLeft' ? (index + scenarios.length - 1) % scenarios.length : event.key === 'Home' ? 0 : event.key === 'End' ? scenarios.length - 1 : null
  if (next === null) return
  event.preventDefault()
  const id = scenarios[next]!.id
  selectScenario(id)
  await nextTick()
  document.getElementById(`api-scenario-${id}`)?.focus()
}
async function copyEndpoint() {
  if (copyTimer) clearTimeout(copyTimer)
  const value = endpoint.value
  try {
    await navigator.clipboard.writeText(value)
    if (disposed) return
    copiedEndpoint.value = value
    copyFailed.value = false
    copyTimer = setTimeout(() => { copiedEndpoint.value = '' }, 1800)
  } catch { if (!disposed) copyFailed.value = true }
}
function downloadTemplate() {
  const content = template.value, filename = selected.value.filename
  if (!content || !filename) return
  const url = URL.createObjectURL(new Blob([content + '\n'], { type: 'application/json;charset=utf-8' }))
  const link = document.createElement('a')
  link.href = url
  link.download = filename
  link.click()
  setTimeout(() => URL.revokeObjectURL(url), 1000)
}
onUnmounted(() => { disposed = true; if (copyTimer) clearTimeout(copyTimer) })
</script>

<style scoped>
.api-connect { padding-block: 56px; border-top: 1px solid var(--ui-border); scroll-margin-top: 96px; }
.connect-heading { display: flex; align-items: flex-end; justify-content: space-between; gap: 24px; }
.connect-heading > div > p:first-child { color: var(--ui-accent); font-size: 12px; }
.connect-heading h2 { margin-top: 12px; font-size: 32px; line-height: 1.3; font-weight: 500; letter-spacing: -.02em; }
.connect-intro { margin-top: 12px; color: var(--ui-text-secondary); font-size: 14px; line-height: 1.8; }
.connect-heading > a, .connect-key-btn { display: inline-flex; align-items: center; gap: 8px; flex: none; min-height: 40px; color: var(--ui-accent); font-size: 13px; font-weight: 600; cursor: pointer; }
.connect-toolbar { display: flex; align-items: center; justify-content: space-between; flex-wrap: wrap; gap: 12px; margin-top: 28px; }
.scenario-tabs { display: inline-flex; gap: 4px; max-width: 100%; padding: 4px; border: 1px solid var(--ui-border); border-radius: 13px; background: var(--ui-bg-soft); }
.scenario-tabs button { display: inline-flex; align-items: center; justify-content: center; gap: 8px; min-width: 0; min-height: 40px; padding: 8px 16px; border: 1px solid transparent; border-radius: 9px; color: var(--ui-text-secondary); white-space: nowrap; font-size: 14px; transition: color 160ms, background 160ms, border-color 160ms; }
.scenario-tabs button > svg { color: var(--ui-text-muted); flex: none; }
.scenario-tabs button:hover { color: var(--ui-text-primary); }
.scenario-tabs button[aria-selected=true] { color: var(--ui-text-primary); border-color: var(--ui-border-strong); background: var(--ui-bg-elevated); }
.scenario-tabs button[aria-selected=true] > svg { color: var(--ui-accent); }
.connect-reference { display: inline-flex; align-items: center; gap: 4px; min-height: 36px; color: var(--ui-text-muted); font-size: 12px; }
.connect-reference:hover { color: var(--ui-accent); }
.integration-workspace { display: grid; gap: 20px; margin-top: 20px; }
.integration-options { display: grid; gap: 10px; min-width: 0; padding: 0; list-style: none; }
.integration-options li { display: flex; align-items: center; gap: 16px; min-width: 0; min-height: 84px; padding: 16px 20px; border: 1px solid var(--ui-border); border-radius: 16px; background: var(--ui-bg-elevated); transition: background 160ms, border-color 160ms; }
.integration-options li.is-selected { border-color: color-mix(in srgb, var(--ui-accent) 55%, var(--ui-border)); background: color-mix(in srgb, var(--ui-accent) 5%, var(--ui-bg-elevated)); }
.integration-icon { display: grid; place-items: center; flex: none; width: 44px; height: 44px; border: 1px solid var(--ui-border); border-radius: 12px; color: var(--ui-text-secondary); background: var(--ui-bg-soft); }
.integration-options li.is-selected .integration-icon { color: var(--ui-accent); }
.integration-summary { flex: 1; min-width: 0; }
.integration-summary > a { display: inline-flex; align-items: center; gap: 6px; max-width: 100%; min-height: 24px; color: var(--ui-text-primary); }
.integration-summary > a:hover { color: var(--ui-accent); }
.integration-summary strong { font-size: 15px; font-weight: 500; }
.integration-summary > a svg { flex: none; color: var(--ui-text-muted); }
.integration-summary > p { margin-top: 4px; color: var(--ui-text-muted); font-size: 12px; line-height: 1.6; }
.integration-options button { display: inline-flex; align-items: center; justify-content: center; gap: 6px; flex: none; min-height: 36px; padding: 6px 12px; border: 1px solid var(--ui-border); border-radius: 9px; color: var(--ui-text-secondary); font-size: 12px; transition: color 160ms, background 160ms, border-color 160ms; }
.integration-options button:hover { color: var(--ui-accent); background: var(--ui-bg-soft); border-color: var(--ui-border-strong); }
.integration-options button[aria-pressed=true] { color: var(--ui-accent); border-color: color-mix(in srgb, var(--ui-accent) 35%, var(--ui-border)); }
.integration-detail { min-width: 0; padding: 28px; border: 1px solid var(--ui-border); border-radius: 20px; background: var(--ui-bg-elevated); }
.integration-heading { display: flex; align-items: flex-start; justify-content: space-between; gap: 20px; }
.integration-heading > div:first-child > p { color: var(--ui-text-muted); font-size: 12px; }
.integration-heading h3 { margin-top: 8px; color: var(--ui-text-primary); font-size: 24px; line-height: 1.3; font-weight: 500; }
.integration-links { display: flex; align-items: center; flex-wrap: wrap; justify-content: flex-end; gap: 8px 16px; }
.integration-links a { display: inline-flex; align-items: center; gap: 4px; min-height: 32px; color: var(--ui-accent); font-size: 12px; }
.integration-description { max-width: 64ch; margin-top: 16px; color: var(--ui-text-secondary); font-size: 14px; line-height: 1.8; }
.quick-config { display: flex; align-items: center; justify-content: space-between; gap: 20px; margin-top: 20px; padding: 16px; border: 1px solid var(--ui-border); border-radius: 12px; background: var(--ui-bg-soft); }
.quick-config strong { color: var(--ui-text-primary); font-size: 13px; font-weight: 500; }
.quick-config p { max-width: 65ch; margin-top: 6px; color: var(--ui-text-muted); font-size: 12px; line-height: 1.8; }
.quick-config > a { display: inline-flex; align-items: center; justify-content: center; gap: 6px; flex: none; min-height: 40px; padding: 8px 14px; border: 1px solid var(--ui-border-strong); border-radius: 10px; color: var(--ui-accent); font-size: 13px; transition: background 160ms; }
.quick-config > a:hover { background: var(--ui-bg-elevated); }
.integration-steps { display: grid; gap: 16px; margin-top: 24px; padding-bottom: 24px; border-bottom: 1px solid var(--ui-border); }
.integration-steps li { display: flex; align-items: flex-start; gap: 12px; color: var(--ui-text-secondary); font-size: 13px; line-height: 1.8; }
.integration-steps li > span { display: grid; place-items: center; width: 24px; height: 24px; flex: none; border: 1px solid var(--ui-border-strong); border-radius: 6px; color: var(--ui-text-muted); font: 12px var(--nt-font-mono); }
.integration-fields { display: grid; gap: 8px; margin-top: 24px; }
.integration-fields > label { color: var(--ui-text-secondary); font-size: 12px; }
.integration-fields > label:not(:first-child) { margin-top: 12px; }
.endpoint-control { display: flex; min-width: 0; border: 1px solid var(--ui-border-strong); border-radius: 10px; overflow: hidden; background: var(--ui-bg-soft); }
.endpoint-control input, .integration-fields > input { width: 100%; min-width: 0; min-height: 44px; padding: 10px 12px; color: var(--ui-text-primary); font: 12px var(--nt-font-mono); background: var(--ui-bg-soft); }
.endpoint-control input { border: 0; }
.endpoint-control button { display: grid; place-items: center; min-width: 44px; color: var(--ui-text-secondary); border-left: 1px solid var(--ui-border); }
.endpoint-control button:hover { color: var(--ui-accent); }
.integration-fields > input { max-width: 420px; border: 1px solid var(--ui-border-strong); border-radius: 10px; }
.field-hint, .copy-feedback { color: var(--ui-text-muted); font-size: 12px; line-height: 1.75; }
.field-hint a { display: inline-flex; align-items: center; gap: 4px; color: var(--ui-accent); }
.field-hint.is-invalid, .copy-feedback { color: var(--nt-warning-ink); }
.template-heading { display: flex; align-items: center; justify-content: space-between; gap: 12px; margin-top: 28px; margin-bottom: 12px; }
.template-heading h4 { color: var(--ui-text-secondary); font-size: 13px; font-weight: 500; }
.template-heading button { display: inline-flex; align-items: center; gap: 8px; min-height: 36px; padding: 8px 12px; border: 1px solid var(--ui-border-strong); border-radius: 8px; color: var(--ui-text-secondary); font-size: 12px; transition: color 160ms, background 160ms; }
.template-heading button:hover { color: var(--ui-text-primary); background: var(--ui-bg-soft); }
.integration-detail :deep(.nt-code) { margin-top: 0; border-radius: 12px; }
.integration-detail :deep(.nt-code-bar i) { display: none; }
.integration-detail :deep(.nt-code-bar > span) { margin-left: 0; }
.integration-detail :deep(.nt-code pre) { max-height: 360px; overflow: auto; font-size: 12px; line-height: 1.8; }
.environment-hint { display: grid; gap: 8px; margin-top: 16px; color: var(--ui-text-muted); font-size: 12px; line-height: 1.7; }
.environment-hint code { color: var(--ui-text-secondary); font-size: 11px; overflow-wrap: anywhere; }
.integration-note { margin-top: 20px; padding-top: 16px; border-top: 1px solid var(--ui-border); color: var(--ui-text-muted); font-size: 12px; line-height: 1.8; }
.connect-compatibility { max-width: 92ch; margin-top: 24px; color: var(--ui-text-muted); font-size: 12px; line-height: 1.8; }
.api-connect :is(a,button,input):focus-visible { outline: 2px solid var(--ui-accent); outline-offset: 3px; }
@media (max-width: 1000px) { .integration-detail { padding: 24px; } }
@media (max-width: 760px) { .api-connect { padding-block: 40px; } .connect-heading { align-items: flex-start; flex-direction: column; gap: 12px; } .connect-heading h2 { font-size: 28px; } .connect-toolbar { margin-top: 20px; } .scenario-tabs button { gap: 6px; padding: 8px 10px; font-size: 12px; } .integration-workspace { gap: 16px; } .integration-heading { flex-direction: column; gap: 12px; } .integration-links { justify-content: flex-start; } .quick-config { align-items: flex-start; flex-direction: column; gap: 12px; } }
@media (max-width: 480px) { .scenario-tabs { width: 100%; } .scenario-tabs button { flex: 1; gap: 4px; padding: 8px 6px; } .scenario-tabs button > svg { width: 15px; } .integration-options li { gap: 12px; padding: 12px; min-height: 76px; border-radius: 13px; } .integration-icon { width: 36px; height: 36px; border-radius: 10px; } .integration-summary strong { font-size: 13px; } .integration-summary > p { font-size: 12px; } .integration-options button { min-width: 36px; padding: 8px; } .integration-options button > span { display: none; } .integration-detail { padding: 20px; border-radius: 16px; } .integration-heading h3 { font-size: 22px; } .integration-description { font-size: 13px; } .integration-steps { gap: 16px; } .integration-steps li { gap: 8px; font-size: 12px; } .template-heading { flex-wrap: wrap; } .environment-hint code { font-size: 12px; } }
@media (prefers-reduced-motion: reduce) { .api-connect :is(a,button,li) { transition: none; } }
</style>
