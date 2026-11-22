<template>
  <section id="api-connect" class="api-connect" aria-labelledby="api-connect-title">
    <header class="connect-heading">
      <div>
        <p class="connect-eyebrow">{{ t('needtoken.apiConnect.setupLabel') }}</p>
        <h2 id="api-connect-title">{{ t('needtoken.apiConnect.setupTitle') }}</h2>
        <p class="connect-intro">{{ t('needtoken.apiConnect.setupIntro') }}</p>
      </div>
      <div class="connect-head-actions">
        <a class="connect-reference" :href="CC_SWITCH_REFERENCE_URL" target="_blank" rel="noopener noreferrer">
          {{ t('needtoken.apiConnect.reference') }}
          <ArrowUpRight :size="13" aria-hidden="true" />
        </a>
        <button
          v-if="userAuth.isAuthenticated && !appStore.isResellerTenant"
          type="button"
          class="connect-key-btn"
          :disabled="ssoLoading"
          @click="openConsoleKeys"
        >
          <span>{{ t('needtoken.apiConnect.getKey') }}</span>
          <ArrowUpRight :size="15" aria-hidden="true" />
        </button>
        <RouterLink v-else to="/auth/login?redirect=/api-services" class="connect-key-btn">
          <span>{{ t('needtoken.apiConnect.getKey') }}</span>
          <ArrowUpRight :size="15" aria-hidden="true" />
        </RouterLink>
      </div>
    </header>

    <!-- CC Switch 风格双栏桌面工作台（参考图 5） -->
    <div class="cc-workbench">
      <!-- 左侧导航边栏 -->
      <aside class="cc-sidebar" :aria-label="t('needtoken.apiConnect.chooseTool')">
        <div class="cc-window-dots" aria-hidden="true">
          <i class="dot-red" /><i class="dot-amber" /><i class="dot-green" />
          <span class="cc-window-brand">NeedToken Switch</span>
        </div>

        <div class="cc-nav-scroll">
          <div v-for="group in groupedTools" :key="group.id" class="cc-nav-group">
            <p class="cc-nav-group-label">{{ t(`needtoken.apiConnect.scenarios.${group.id}.title`) }}</p>
            <button
              v-for="tool in group.tools"
              :key="tool.id"
              :data-api-tool="tool.id"
              type="button"
              :class="['cc-nav-item', { 'is-active': selected.id === tool.id }]"
              :aria-pressed="selected.id === tool.id"
              @click="selectTool(tool.id)"
            >
              <span class="cc-nav-icon" aria-hidden="true">
                <component :is="toolIcons[tool.id]" :size="15" :stroke-width="1.75" />
              </span>
              <span class="cc-nav-name" translate="no">{{ tool.name }}</span>
              <span v-if="tool.badge" class="cc-nav-badge">{{ tool.badge }}</span>
            </button>
          </div>
        </div>

        <div class="cc-sidebar-foot">
          <RouterLink to="/models" class="cc-foot-link">
            <Layers :size="13" aria-hidden="true" />
            <span>{{ t('needtoken.nav.models') }}</span>
          </RouterLink>
          <RouterLink to="/docs" class="cc-foot-link">
            <BookOpen :size="13" aria-hidden="true" />
            <span>{{ t('needtoken.nav.docs') }}</span>
          </RouterLink>
        </div>
      </aside>

      <!-- 右侧配置主面板 -->
      <article id="api-integration-detail" class="cc-main" :aria-label="selected.name">
        <header class="cc-main-head">
          <div class="cc-main-title">
            <span class="cc-main-icon" aria-hidden="true">
              <component :is="toolIcons[selected.id]" :size="18" />
            </span>
            <div>
              <div class="cc-title-row">
                <h3 translate="no">{{ selected.name }}</h3>
                <span class="cc-status-pill">
                  <i aria-hidden="true" />
                  <span>NeedToken Gateway</span>
                </span>
              </div>
              <p class="cc-main-desc">{{ t(`needtoken.apiConnect.tools.${selected.id}.description`) }}</p>
            </div>
          </div>
          <div class="cc-main-links">
            <a v-if="selected.get" :href="selected.get" target="_blank" rel="noopener noreferrer">
              {{ t(`needtoken.apiConnect.tools.${selected.id}.get`) }}
              <ArrowUpRight :size="13" aria-hidden="true" />
            </a>
            <a v-if="selected.docs" :href="selected.docs" target="_blank" rel="noopener noreferrer">
              {{ t('needtoken.apiConnect.docs') }}
              <ArrowUpRight :size="13" aria-hidden="true" />
            </a>
          </div>
        </header>

        <div v-if="selected.id === 'ccswitch'" class="cc-quick-banner">
          <div>
            <strong>{{ t('needtoken.apiConnect.quickConfigTitle') }}</strong>
            <p>{{ t('needtoken.apiConnect.quickConfigHint') }}</p>
          </div>
          <a :href="CONSOLE_URL" target="_blank" rel="noopener noreferrer" data-api-quick-config>
            <Zap :size="14" aria-hidden="true" />
            <span>{{ t('needtoken.apiConnect.quickConfig') }}</span>
            <ArrowUpRight :size="13" aria-hidden="true" />
          </a>
        </div>

        <!-- 接入步骤 -->
        <ol class="cc-steps">
          <li v-for="step in 3" :key="step">
            <span class="cc-step-num" aria-hidden="true">{{ step }}</span>
            <p>{{ t(`needtoken.apiConnect.tools.${selected.id}.step${step}`) }}</p>
          </li>
        </ol>

        <!-- Base URL & 模型自定义 -->
        <div class="cc-fields-grid">
          <div class="cc-field">
            <label for="api-integration-endpoint">{{ t('needtoken.apiConnect.endpoint') }}</label>
            <div class="cc-endpoint-box">
              <input
                id="api-integration-endpoint"
                :value="endpoint"
                readonly
                spellcheck="false"
                @focus="($event.target as HTMLInputElement).select()"
              />
              <button
                data-api-copy-endpoint
                type="button"
                :aria-label="t('needtoken.apiConnect.copyEndpoint')"
                @click="copyEndpoint"
              >
                <Check v-if="copied" :size="15" aria-hidden="true" />
                <Copy v-else :size="15" aria-hidden="true" />
              </button>
            </div>
            <p v-if="copyFailed" class="cc-field-warn" role="status">{{ t('needtoken.apiConnect.copyFailed') }}</p>
          </div>

          <div v-if="selected.id !== 'cloudflare'" class="cc-field">
            <label for="api-integration-model">{{ t('needtoken.apiConnect.modelId') }}</label>
            <input
              id="api-integration-model"
              v-model="model"
              type="text"
              maxlength="200"
              spellcheck="false"
              :aria-invalid="!validModel"
              aria-describedby="api-model-hint"
            />
            <p id="api-model-hint" class="cc-field-hint" :class="{ 'is-invalid': !validModel }">
              {{ t(validModel ? 'needtoken.apiConnect.modelHint' : 'needtoken.apiConnect.modelInvalid') }}
              <RouterLink to="/models">{{ t('needtoken.nav.models') }}<ArrowUpRight :size="11" aria-hidden="true" /></RouterLink>
            </p>
          </div>
        </div>

        <!-- 配置模板 -->
        <template v-if="template">
          <div class="cc-template-head">
            <h4>{{ t(selected.filename ? 'needtoken.apiConnect.configTemplate' : 'needtoken.apiConnect.configFields') }}</h4>
            <div class="cc-template-actions">
              <button v-if="selected.filename" data-api-download type="button" @click="downloadTemplate">
                <Download :size="13" aria-hidden="true" />
                <span>{{ t('needtoken.apiConnect.downloadConfig') }}</span>
              </button>
            </div>
          </div>
          <div data-api-template class="cc-code-wrap">
            <CodeBlock :title="selected.filename || selected.name" :code="template" />
          </div>
        </template>

        <p v-if="selected.id === 'opencode' || selected.id === 'openclaw' || selected.id === 'codex'" class="cc-env-hint">
          <code>NEEDTOKEN_API_KEY=&lt;YOUR_NEEDTOKEN_API_KEY&gt;</code>
          <span>{{ t('needtoken.apiConnect.environmentHint') }}</span>
        </p>
        <p class="cc-note">{{ t(`needtoken.apiConnect.tools.${selected.id}.note`) }}</p>
      </article>
    </div>

    <p class="connect-compatibility">{{ t('needtoken.apiConnect.compatibility') }}</p>
  </section>
</template>

<script setup lang="ts">
import { computed, onUnmounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  ArrowUpRight,
  BookOpen,
  Bot,
  Check,
  Cloud,
  Code2,
  Copy,
  Download,
  Layers,
  Network,
  Server,
  Sparkles,
  Terminal,
  Zap,
} from 'lucide-vue-next'
import { API_BASE_URL, CONSOLE_URL, DEFAULT_MODELS } from '../config'
import {
  API_INTEGRATIONS,
  CC_SWITCH_REFERENCE_URL,
  integrationEndpoint,
  integrationModel,
  integrationTemplate,
  type IntegrationId,
  type IntegrationScenario,
} from '../lib/apiIntegrations'
import { useUserAuthStore } from '../../../stores/userAuth'
import { useAppStore } from '../../../stores/app'
import { useApiAccountLink } from '../composables/useApiAccountLink'
import CodeBlock from './CodeBlock.vue'

const props = defineProps<{
  initialModel?: string
}>()

const { t } = useI18n()
const userAuth = useUserAuthStore()
const appStore = useAppStore()
const { ssoLoading, openApiConsoleSSO } = useApiAccountLink()
const openConsoleKeys = () => {
  void openApiConsoleSSO('/keys', '/api-services')
}

const scenarios: readonly IntegrationScenario[] = ['coding', 'local', 'cloud', 'agent']
const toolIcons: Record<IntegrationId, any> = {
  claudecode: Terminal,
  codex: Code2,
  geminicli: Sparkles,
  opencode: Code2,
  cursor: Layers,
  ccswitch: Network,
  sub2api: Server,
  cloudflare: Cloud,
  openclaw: Bot,
  cherrystudio: Sparkles,
}

const groupedTools = computed(() =>
  scenarios
    .map(id => ({ id, tools: API_INTEGRATIONS.filter(tool => tool.scenario === id) }))
    .filter(group => group.tools.length > 0),
)

const selectedId = ref<IntegrationId>('claudecode')
const model = ref<string>(DEFAULT_MODELS.find(m => m.startsWith('claude')) || DEFAULT_MODELS[0] || 'claude-sonnet-4-6')

watch(() => props.initialModel, (nextModel) => {
  if (!nextModel) return
  model.value = nextModel
  const lower = nextModel.toLowerCase()
  if (lower.startsWith('claude')) selectedId.value = 'claudecode'
  else if (lower.startsWith('gpt') || lower.startsWith('o1') || lower.startsWith('o3') || lower.startsWith('o4')) selectedId.value = 'codex'
  else if (lower.startsWith('gemini')) selectedId.value = 'geminicli'
})

const validModel = computed(() => integrationModel(model.value))
const selected = computed(() => API_INTEGRATIONS.find(tool => tool.id === selectedId.value) || API_INTEGRATIONS[0]!)
const endpoint = computed(() => integrationEndpoint(selected.value.id, API_BASE_URL))
const template = computed(() => integrationTemplate(selected.value.id, API_BASE_URL, model.value))
const copiedEndpoint = ref('')
const copyFailed = ref(false)
const copied = computed(() => copiedEndpoint.value === endpoint.value)
let disposed = false
let copyTimer: ReturnType<typeof setTimeout> | undefined

function selectTool(id: IntegrationId) {
  selectedId.value = id
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
  } catch {
    if (!disposed) copyFailed.value = true
  }
}

function downloadTemplate() {
  const content = template.value
  const filename = selected.value.filename
  if (!content || !filename) return
  const mime = filename.endsWith('.json') ? 'application/json;charset=utf-8' : 'text/plain;charset=utf-8'
  const url = URL.createObjectURL(new Blob([content + '\n'], { type: mime }))
  const link = document.createElement('a')
  link.href = url
  link.download = filename
  link.click()
  setTimeout(() => URL.revokeObjectURL(url), 1000)
}

onUnmounted(() => {
  disposed = true
  if (copyTimer) clearTimeout(copyTimer)
})
</script>

<style scoped>
.api-connect { padding-block: 52px; border-top: 1px solid var(--ui-border); scroll-margin-top: 96px; }
.connect-heading { display: flex; align-items: flex-end; justify-content: space-between; flex-wrap: wrap; gap: 16px; }
.connect-eyebrow { color: var(--ui-accent); font-size: 11.5px; font-weight: 700; letter-spacing: .12em; text-transform: uppercase; }
.connect-heading h2 { margin-top: 8px; font-size: clamp(24px, 2.8vw, 32px); font-weight: 700; letter-spacing: -.025em; color: var(--ui-text-primary); }
.connect-intro { margin-top: 8px; color: var(--ui-text-secondary); font-size: 14px; line-height: 1.75; }
.connect-head-actions { display: flex; align-items: center; gap: 12px; flex-wrap: wrap; }
.connect-reference { display: inline-flex; align-items: center; gap: 4px; font-size: 12px; color: var(--ui-text-muted); }
.connect-reference:hover { color: var(--ui-accent); }
.connect-key-btn { display: inline-flex; align-items: center; gap: 6px; min-height: 38px; padding: 0 14px; border: 1px solid var(--nt-accent-border, var(--ui-border-strong)); border-radius: 10px; background: var(--nt-accent-subtle, var(--ui-bg-elevated)); color: var(--ui-accent); font-size: 13px; font-weight: 600; cursor: pointer; }
.connect-key-btn:hover:not(:disabled) { border-color: var(--ui-accent); }

/* CC Switch 双栏深色工作台窗口 */
.cc-workbench {
  display: grid;
  grid-template-columns: 248px minmax(0, 1fr);
  margin-top: 24px;
  border: 1px solid #262b36;
  border-radius: 20px;
  overflow: hidden;
  background: #14171f;
  color: #e5e7eb;
  box-shadow: 0 22px 50px rgba(15, 23, 42, 0.16);
}
.cc-sidebar {
  display: flex;
  flex-direction: column;
  border-right: 1px solid #232834;
  background: #10131a;
}
.cc-window-dots {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 14px 16px;
  border-bottom: 1px solid #1e2330;
}
.cc-window-dots i { width: 10px; height: 10px; border-radius: 999px; display: block; }
.dot-red { background: #ef4444; }
.dot-amber { background: #f59e0b; }
.dot-green { background: #22c55e; }
.cc-window-brand { margin-left: 8px; font: 600 12px var(--nt-font-mono); color: #9ca3af; }

.cc-nav-scroll { flex: 1; padding: 12px 10px; display: flex; flex-direction: column; gap: 14px; overflow-y: auto; max-height: 620px; }
.cc-nav-group { display: flex; flex-direction: column; gap: 3px; }
.cc-nav-group-label { padding: 4px 10px; font-size: 10.5px; font-weight: 700; letter-spacing: .08em; text-transform: uppercase; color: #6b7280; }
.cc-nav-item {
  display: flex;
  align-items: center;
  gap: 9px;
  width: 100%;
  min-height: 36px;
  padding: 6px 10px;
  border-radius: 9px;
  color: #cbd5e1;
  font-size: 13px;
  font-weight: 500;
  text-align: left;
  cursor: pointer;
  transition: background 140ms, color 140ms;
}
.cc-nav-item:hover { background: rgba(255, 255, 255, 0.06); color: #fff; }
.cc-nav-item.is-active { background: rgba(16, 185, 129, 0.16); color: #34d399; font-weight: 600; }
.cc-nav-icon { display: inline-grid; place-items: center; flex: none; opacity: .85; }
.cc-nav-name { flex: 1; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.cc-nav-badge {
  flex: none;
  padding: 1px 6px;
  border-radius: 5px;
  background: rgba(255, 255, 255, 0.08);
  color: #94a3b8;
  font-size: 10px;
  font-weight: 600;
}
.cc-nav-item.is-active .cc-nav-badge { background: rgba(16, 185, 129, 0.22); color: #6ee7b7; }

.cc-sidebar-foot {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 10px;
  border-top: 1px solid #1e2330;
}
.cc-foot-link {
  flex: 1;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 5px;
  height: 32px;
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.04);
  color: #9ca3af;
  font-size: 12px;
}
.cc-foot-link:hover { background: rgba(255, 255, 255, 0.09); color: #fff; }

/* 右侧详情区 */
.cc-main { padding: 26px 28px; min-width: 0; display: flex; flex-direction: column; gap: 18px; }
.cc-main-head { display: flex; align-items: flex-start; justify-content: space-between; gap: 16px; flex-wrap: wrap; }
.cc-main-title { display: flex; align-items: flex-start; gap: 12px; min-width: 0; }
.cc-main-icon {
  display: grid;
  place-items: center;
  flex: none;
  width: 40px;
  height: 40px;
  border-radius: 11px;
  border: 1px solid #2b3344;
  background: #1b202c;
  color: #34d399;
}
.cc-title-row { display: flex; align-items: center; flex-wrap: wrap; gap: 10px; }
.cc-title-row h3 { font-size: 20px; font-weight: 700; color: #f8fafc; line-height: 1.2; }
.cc-status-pill {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 3px 10px;
  border-radius: 999px;
  background: rgba(16, 185, 129, 0.14);
  border: 1px solid rgba(52, 211, 153, 0.3);
  color: #6ee7b7;
  font-size: 11px;
  font-weight: 600;
}
.cc-status-pill i { width: 6px; height: 6px; border-radius: 999px; background: #10b981; }
.cc-main-desc { margin-top: 6px; font-size: 13px; line-height: 1.65; color: #9ca3af; max-width: 64ch; }
.cc-main-links { display: flex; align-items: center; gap: 10px; }
.cc-main-links a {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 6px 11px;
  border-radius: 8px;
  border: 1px solid #2b3344;
  background: #1b202c;
  color: #cbd5e1;
  font-size: 12px;
  font-weight: 500;
}
.cc-main-links a:hover { border-color: #34d399; color: #34d399; }

.cc-quick-banner {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
  padding: 14px 16px;
  border: 1px solid rgba(52, 211, 153, 0.28);
  border-radius: 12px;
  background: rgba(16, 185, 129, 0.08);
}
.cc-quick-banner strong { font-size: 13px; color: #ecfdf5; }
.cc-quick-banner p { margin-top: 4px; font-size: 12px; color: #9ca3af; }
.cc-quick-banner a {
  display: inline-flex;
  flex: none;
  align-items: center;
  gap: 5px;
  padding: 8px 13px;
  border-radius: 9px;
  background: #10b981;
  color: #052e16;
  font-size: 12.5px;
  font-weight: 700;
}

.cc-steps { display: grid; gap: 10px; padding: 14px 16px; border: 1px solid #232834; border-radius: 12px; background: #181c26; list-style: none; margin: 0; }
.cc-steps li { display: flex; align-items: flex-start; gap: 10px; font-size: 13px; line-height: 1.65; color: #cbd5e1; }
.cc-step-num { display: grid; place-items: center; flex: none; width: 22px; height: 22px; border-radius: 6px; background: rgba(16, 185, 129, 0.16); color: #34d399; font: 700 11.5px var(--nt-font-mono); margin-top: 1px; }

.cc-fields-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 14px; }
.cc-field { display: flex; flex-direction: column; gap: 6px; min-width: 0; }
.cc-field label { font-size: 11.5px; font-weight: 600; color: #9ca3af; }
.cc-endpoint-box { display: flex; border: 1px solid #2b3344; border-radius: 9px; overflow: hidden; background: #0f1219; }
.cc-endpoint-box input,
.cc-field > input {
  width: 100%;
  min-width: 0;
  height: 38px;
  padding: 0 11px;
  background: #0f1219;
  color: #f8fafc;
  font: 12px var(--nt-font-mono);
  outline: none;
}
.cc-endpoint-box input { border: 0; }
.cc-endpoint-box button { display: grid; place-items: center; min-width: 38px; border-left: 1px solid #2b3344; color: #9ca3af; cursor: pointer; }
.cc-endpoint-box button:hover { color: #34d399; background: rgba(255, 255, 255, 0.04); }
.cc-field > input { border: 1px solid #2b3344; border-radius: 9px; }
.cc-field > input:focus { border-color: #34d399; }
.cc-field-hint { font-size: 11.5px; color: #94a3b8; display: inline-flex; align-items: center; gap: 6px; flex-wrap: wrap; }
.cc-field-hint a { display: inline-flex; align-items: center; gap: 3px; color: #34d399; }
.cc-field-hint.is-invalid, .cc-field-warn { font-size: 11.5px; color: #fbbf24; }

.cc-template-head { display: flex; align-items: center; justify-content: space-between; gap: 10px; }
.cc-template-head h4 { font-size: 12.5px; font-weight: 600; color: #cbd5e1; }
.cc-template-actions button {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  height: 30px;
  padding: 0 10px;
  border: 1px solid #2b3344;
  border-radius: 7px;
  background: #1b202c;
  color: #cbd5e1;
  font-size: 12px;
  cursor: pointer;
}
.cc-template-actions button:hover { border-color: #34d399; color: #34d399; }

.cc-code-wrap :deep(.nt-code) { margin-top: 0; border-color: #262c3a; background: #0d1017; }
.cc-code-wrap :deep(.nt-code-bar) { background: #161b26; border-bottom-color: #232834; color: #9ca3af; }
.cc-code-wrap :deep(.nt-code pre) { max-height: 300px; overflow: auto; color: #e2e8f0; }

.cc-env-hint { display: flex; align-items: center; flex-wrap: wrap; gap: 8px; font-size: 12px; color: #94a3b8; }
.cc-env-hint code { padding: 3px 8px; border-radius: 6px; background: #0f1219; border: 1px solid #232834; color: #6ee7b7; font-size: 11.5px; }
.cc-note { padding-top: 12px; border-top: 1px solid #232834; font-size: 12px; line-height: 1.7; color: #94a3b8; }
.connect-compatibility { margin-top: 16px; font-size: 12px; line-height: 1.75; color: var(--ui-text-muted); }

@media (max-width: 900px) {
  .cc-workbench { grid-template-columns: minmax(0, 1fr); }
  .cc-sidebar { border-right: 0; border-bottom: 1px solid #232834; }
  .cc-nav-scroll { max-height: none; flex-direction: row; flex-wrap: nowrap; overflow-x: auto; padding: 10px 12px; gap: 12px; }
  .cc-nav-group { flex-direction: row; align-items: center; gap: 4px; }
  .cc-nav-group-label { display: none; }
  .cc-nav-item { width: auto; white-space: nowrap; }
  .cc-sidebar-foot { display: none; }
  .cc-fields-grid { grid-template-columns: minmax(0, 1fr); }
  .cc-main { padding: 20px 16px; }
  .cc-quick-banner { flex-direction: column; align-items: flex-start; }
}
</style>
