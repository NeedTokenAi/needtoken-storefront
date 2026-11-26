<template>
  <dialog
    ref="dialogRef"
    class="nt-quick-drawer"
    aria-labelledby="nt-quick-use-title"
    @cancel.prevent="closeDrawer"
    @pointerdown="onPointerDown"
    @pointerup="onPointerUp"
  >
    <div class="nt-quick-drawer__sheet flex h-full flex-col bg-card text-foreground">
      <!-- 抽屉顶栏 -->
      <div class="flex shrink-0 items-center justify-between gap-3 border-b border-border px-6 py-4">
        <div class="flex min-w-0 items-center gap-3">
          <span class="grid h-9 w-9 shrink-0 place-items-center rounded-xl bg-primary/12 text-primary">
            <Zap class="h-4.5 w-4.5" />
          </span>
          <div class="min-w-0">
            <h1 id="nt-quick-use-title" class="truncate text-base font-bold text-foreground">
              {{ t('needtoken.memberConsole.quickUse.title') }}
            </h1>
            <p class="mt-0.5 truncate text-xs text-muted-foreground">
              {{ t('needtoken.memberConsole.quickUse.subtitle') }}
            </p>
          </div>
        </div>
        <button
          type="button"
          autofocus
          class="grid h-8 w-8 shrink-0 place-items-center rounded-lg border border-transparent text-muted-foreground transition-colors hover:border-border hover:bg-accent hover:text-foreground"
          :aria-label="t('needtoken.memberConsole.quickUse.closeDrawer')"
          @click="closeDrawer"
        >
          <X class="h-4 w-4" />
        </button>
      </div>

      <!-- 会员专属上下文配置条（密钥 + 模型 + 真实密钥开关） -->
      <div class="shrink-0 space-y-3 border-b border-border bg-muted/25 px-6 py-3.5">
        <div
          v-if="!loading && activeTokens.length === 0"
          class="flex flex-wrap items-center justify-between gap-2 rounded-xl border border-dashed border-primary/35 bg-primary/5 px-3.5 py-2.5 text-xs"
        >
          <span class="text-foreground">{{ t('needtoken.memberConsole.quickUse.noKeyBanner') }}</span>
          <Button size="sm" class="h-7 px-3 text-xs" @click="goCreateKey">
            <KeyRound class="mr-1 h-3.5 w-3.5" />
            {{ t('needtoken.memberConsole.quickUse.createKeyNow') }}
          </Button>
        </div>

        <div class="grid grid-cols-1 gap-3 md:grid-cols-12">
          <!-- 会员自己的 API Key（就地渲染，不依赖 body Portal） -->
          <div ref="keyMenuRootEl" class="relative md:col-span-5">
            <label class="mb-1 block text-[11px] font-semibold text-muted-foreground">
              {{ t('needtoken.memberConsole.quickUse.selectKey') }}
            </label>
            <button
              type="button"
              class="nt-theme-select__trigger w-full"
              :aria-expanded="keyMenuOpen"
              :disabled="activeTokens.length === 0"
              @click="keyMenuOpen = !keyMenuOpen"
            >
              <span class="truncate text-left text-xs font-medium">
                {{ selectedTokenLabel }}
              </span>
              <ChevronDown class="h-3.5 w-3.5 shrink-0 transition-transform" :class="keyMenuOpen ? 'rotate-180' : ''" />
            </button>
            <div
              v-if="keyMenuOpen && activeTokens.length > 0"
              class="nt-quick-drawer__dropdown"
            >
              <button
                v-for="tok in activeTokens"
                :key="tok.id"
                type="button"
                class="nt-quick-drawer__dropdown-item"
                :class="{ 'is-selected': tok.id === selectedTokenId }"
                @click="selectToken(tok.id)"
              >
                <div class="min-w-0 flex-1">
                  <div class="truncate text-xs font-semibold text-foreground">{{ tok.name || `Key #${tok.id}` }}</div>
                  <div class="truncate font-mono text-[11px] text-muted-foreground">
                    {{ formatTokenGroups(tok) }} · {{ tok.masked_key }}
                  </div>
                </div>
                <Check v-if="tok.id === selectedTokenId" class="h-3.5 w-3.5 shrink-0 text-primary" />
              </button>
            </div>
          </div>

          <!-- 联动所选密钥的可用模型 -->
          <div class="md:col-span-5">
            <label class="mb-1 block text-[11px] font-semibold text-muted-foreground">
              {{ t('needtoken.memberConsole.quickUse.selectModel') }}
            </label>
            <NeedtokenModelCombobox
              v-model="selectedModel"
              :models="availableModels"
              :label="t('needtoken.memberConsole.quickUse.selectModel')"
              :placeholder="t('needtoken.memberConsole.playground.modelPlaceholder')"
              :search-placeholder="t('needtoken.memberConsole.playground.searchModelPlaceholder')"
            />
          </div>

          <!-- 明文 Key / 占位符切换 -->
          <div class="flex flex-col justify-end md:col-span-2">
            <button
              type="button"
              class="inline-flex h-10 w-full items-center justify-center gap-1.5 rounded-xl border px-3 text-xs font-semibold transition-colors"
              :class="includeRealKey && activeTokens.length > 0
                ? 'border-primary/40 bg-primary/10 text-primary'
                : 'border-border bg-background text-muted-foreground hover:text-foreground'"
              :disabled="activeTokens.length === 0"
              @click="includeRealKey = !includeRealKey"
            >
              <Eye v-if="includeRealKey && activeTokens.length > 0" class="h-3.5 w-3.5 shrink-0" />
              <EyeOff v-else class="h-3.5 w-3.5 shrink-0" />
              <span class="truncate">
                {{ includeRealKey && activeTokens.length > 0
                  ? t('needtoken.memberConsole.quickUse.includeRealKey')
                  : t('needtoken.memberConsole.quickUse.placeholderKeyMode') }}
              </span>
            </button>
          </div>
        </div>

        <div class="flex flex-wrap items-center justify-between gap-2 text-[11px] text-muted-foreground">
          <div class="flex min-w-0 items-center gap-1.5">
            <span class="font-semibold">{{ t('needtoken.memberConsole.quickUse.gatewayEndpoint') }}:</span>
            <code class="truncate rounded bg-background px-1.5 py-0.5 font-mono text-foreground">{{ effectiveBaseUrlV1 }}</code>
          </div>
          <button
            type="button"
            class="inline-flex items-center gap-1 font-medium text-primary hover:underline"
            @click="copyText(effectiveBaseUrlV1, 'drawer-base-url')"
          >
            <Check v-if="copiedTag === 'drawer-base-url'" class="h-3 w-3" />
            <Copy v-else class="h-3 w-3" />
            <span>{{ copiedTag === 'drawer-base-url' ? t('needtoken.common.copied') : t('needtoken.common.copy') }}</span>
          </button>
        </div>
      </div>

      <!-- 主滚动体：工具网格列表 或 单个工具的手动教程/配置预览详情 -->
      <div class="flex-1 overflow-y-auto px-6 py-5">
        <!-- 详情子视图：手动配置教程 & 配置文件预览 -->
        <div v-if="activeGuideTool" class="space-y-4">
          <div class="flex flex-wrap items-center justify-between gap-2">
            <button
              type="button"
              class="inline-flex items-center gap-1.5 rounded-lg border border-border bg-background px-3 py-1.5 text-xs font-semibold text-foreground hover:border-primary/40 hover:text-primary"
              @click="activeGuideTool = null"
            >
              <ArrowLeft class="h-3.5 w-3.5" />
              <span>{{ t('needtoken.memberConsole.quickUse.backToList') }}</span>
            </button>
            <div class="flex flex-wrap items-center gap-2">
              <a
                v-if="activeGuideTool.docs_url && /^https?:\/\//i.test(activeGuideTool.docs_url)"
                :href="activeGuideTool.docs_url"
                target="_blank"
                rel="noopener noreferrer"
                class="inline-flex items-center gap-1 rounded-lg border border-border bg-background px-3 py-1.5 text-xs font-semibold text-foreground hover:border-primary/40 hover:text-primary"
              >
                <span>{{ t('needtoken.memberConsole.quickUse.officialDocs') }}</span>
                <ExternalLink class="h-3 w-3" />
              </a>
              <Button v-if="activeGuideTool.config_template" size="sm" variant="outline" @click="downloadConfig(activeGuideTool)">
                <Download class="mr-1.5 h-3.5 w-3.5" />
                {{ t('needtoken.memberConsole.quickUse.downloadConfig') }}
              </Button>
              <Button v-if="activeGuideTool.ai_prompt_template" size="sm" @click="copyAiPrompt(activeGuideTool)">
                <Check v-if="copiedTag === `ai:${activeGuideTool.id}`" class="mr-1.5 h-3.5 w-3.5" />
                <Sparkles v-else class="mr-1.5 h-3.5 w-3.5" />
                {{ copiedTag === `ai:${activeGuideTool.id}` ? t('needtoken.common.copied') : t('needtoken.memberConsole.quickUse.copyAiPrompt') }}
              </Button>
            </div>
          </div>

          <div class="flex items-center gap-3 rounded-2xl border bg-muted/20 p-4">
            <div class="nt-quick-tool-logo h-12 w-12 shrink-0" :style="logoBadgeStyle(activeGuideTool)">
              <img
                v-if="activeGuideTool.logo_url && !brokenLogos.has(activeGuideTool.id)"
                :src="activeGuideTool.logo_url"
                :alt="activeGuideTool.name"
                class="h-8 w-8 object-contain"
                @error="markLogoBroken(activeGuideTool.id)"
              />
              <span v-else class="text-sm font-black tracking-tight">{{ logoGlyph(activeGuideTool) }}</span>
            </div>
            <div class="min-w-0">
              <h2 class="text-base font-bold text-foreground">{{ activeGuideTool.name }}</h2>
              <p v-if="activeGuideTool.summary" class="mt-0.5 text-xs text-muted-foreground">{{ activeGuideTool.summary }}</p>
            </div>
          </div>

          <!-- 配置文件预览 -->
          <div v-if="activeGuideTool.config_template" class="space-y-2">
            <div class="flex items-center justify-between gap-2">
              <span class="text-xs font-bold text-foreground">
                {{ t('needtoken.memberConsole.quickUse.configPreview') }}
                <code v-if="activeGuideTool.config_filename" class="ml-1.5 rounded bg-muted px-1.5 py-0.5 font-mono text-[11px] font-medium text-muted-foreground">
                  {{ activeGuideTool.config_filename }}
                </code>
              </span>
              <button
                type="button"
                class="inline-flex items-center gap-1 rounded-md border bg-background px-2.5 py-1 text-xs font-medium text-foreground hover:border-primary/40 hover:text-primary"
                @click="copyText(renderToolText(activeGuideTool, activeGuideTool.config_template), `cfg:${activeGuideTool.id}`)"
              >
                <Check v-if="copiedTag === `cfg:${activeGuideTool.id}`" class="h-3.5 w-3.5 text-primary" />
                <Copy v-else class="h-3.5 w-3.5" />
                <span>{{ copiedTag === `cfg:${activeGuideTool.id}` ? t('needtoken.common.copied') : t('needtoken.common.copy') }}</span>
              </button>
            </div>
            <pre class="max-h-72 overflow-auto rounded-xl border bg-muted/35 p-4 font-mono text-xs leading-relaxed text-foreground">{{ renderToolText(activeGuideTool, activeGuideTool.config_template) }}</pre>
          </div>

          <!-- 手动配置步骤 -->
          <div v-if="activeGuideTool.manual_guide" class="space-y-2">
            <span class="block text-xs font-bold text-foreground">{{ t('needtoken.memberConsole.quickUse.guideTitle') }}</span>
            <pre class="whitespace-pre-wrap break-words rounded-xl border bg-background p-4 font-sans text-xs leading-relaxed text-foreground">{{ renderToolText(activeGuideTool, activeGuideTool.manual_guide) }}</pre>
          </div>

          <!-- AI 一句话自动配置预览 -->
          <div v-if="activeGuideTool.ai_prompt_template" class="space-y-2 rounded-xl border border-primary/25 bg-primary/5 p-4">
            <div class="flex items-center justify-between gap-2">
              <span class="text-xs font-bold text-primary">{{ t('needtoken.memberConsole.quickUse.copyAiPrompt') }}</span>
              <button
                type="button"
                class="inline-flex items-center gap-1 rounded-md border border-primary/35 bg-background px-2.5 py-1 text-xs font-semibold text-primary hover:bg-primary/10"
                @click="copyAiPrompt(activeGuideTool)"
              >
                <Check v-if="copiedTag === `ai:${activeGuideTool.id}`" class="h-3.5 w-3.5" />
                <Copy v-else class="h-3.5 w-3.5" />
                <span>{{ copiedTag === `ai:${activeGuideTool.id}` ? t('needtoken.common.copied') : t('needtoken.common.copy') }}</span>
              </button>
            </div>
            <p class="text-[11px] text-muted-foreground">{{ t('needtoken.memberConsole.quickUse.aiPromptHint') }}</p>
            <pre class="whitespace-pre-wrap break-words rounded-lg border bg-background p-3 font-sans text-xs leading-relaxed text-foreground">{{ renderToolText(activeGuideTool, activeGuideTool.ai_prompt_template) }}</pre>
          </div>
        </div>

        <!-- 默认列表视图：搜索 + 分类胶囊 + Logo List 卡片 -->
        <div v-else class="space-y-4">
          <div class="space-y-3">
            <div class="relative">
              <Search class="pointer-events-none absolute left-3.5 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
              <input
                v-model.trim="searchQuery"
                type="search"
                :placeholder="t('needtoken.memberConsole.quickUse.searchPlaceholder')"
                class="h-10 w-full rounded-xl border bg-background pl-10 pr-4 text-xs text-foreground outline-none focus:border-primary"
              />
            </div>

            <div class="flex flex-wrap gap-1.5">
              <button
                type="button"
                class="rounded-full border px-3 py-1 text-xs font-semibold transition-colors"
                :class="activeCategoryId === 'all'
                  ? 'border-primary bg-primary text-primary-foreground'
                  : 'border-border bg-background text-muted-foreground hover:border-primary/40 hover:text-foreground'"
                @click="activeCategoryId = 'all'"
              >
                {{ t('needtoken.memberConsole.quickUse.allCategories') }}
                <span class="ml-1 opacity-75">({{ enabledTools.length }})</span>
              </button>
              <button
                v-for="cat in enabledCategories"
                :key="cat.id"
                type="button"
                class="rounded-full border px-3 py-1 text-xs font-semibold transition-colors"
                :class="activeCategoryId === cat.id
                  ? 'border-primary bg-primary text-primary-foreground'
                  : 'border-border bg-background text-muted-foreground hover:border-primary/40 hover:text-foreground'"
                @click="activeCategoryId = cat.id"
              >
                {{ cat.name }}
              </button>
            </div>
          </div>

          <div v-if="loading && !enabledTools.length" class="grid grid-cols-1 gap-3.5 md:grid-cols-2">
            <div v-for="idx in 6" :key="idx" class="h-44 animate-pulse rounded-2xl bg-muted" />
          </div>

          <div v-else-if="!filteredTools.length" class="rounded-2xl border border-dashed px-6 py-14 text-center">
            <Sparkles class="mx-auto h-9 w-9 text-muted-foreground/50" />
            <p class="mt-3 text-sm font-semibold text-foreground">{{ t('needtoken.memberConsole.quickUse.emptyTitle') }}</p>
            <p class="mt-1 text-xs text-muted-foreground">{{ t('needtoken.memberConsole.quickUse.emptyHint') }}</p>
          </div>

          <div v-else class="grid grid-cols-1 gap-3.5 md:grid-cols-2">
            <article
              v-for="tool in filteredTools"
              :key="tool.id"
              class="flex flex-col justify-between gap-3.5 rounded-2xl border bg-background p-4 shadow-xs transition-all hover:border-primary/40 hover:shadow-sm"
            >
              <!-- Logo + 标题与摘要 -->
              <div class="flex items-start gap-3.5">
                <div class="nt-quick-tool-logo h-12 w-12 shrink-0" :style="logoBadgeStyle(tool)">
                  <img
                    v-if="tool.logo_url && !brokenLogos.has(tool.id)"
                    :src="tool.logo_url"
                    :alt="tool.name"
                    class="h-8 w-8 object-contain"
                    @error="markLogoBroken(tool.id)"
                  />
                  <span v-else class="text-sm font-black tracking-tight">{{ logoGlyph(tool) }}</span>
                </div>
                <div class="min-w-0 flex-1">
                  <div class="flex flex-wrap items-center gap-1.5">
                    <h3 class="truncate text-sm font-bold text-foreground">{{ tool.name }}</h3>
                    <span
                      v-if="categoryNameMap.get(tool.category_id)"
                      class="rounded-md bg-muted px-1.5 py-0.5 text-[10px] font-medium text-muted-foreground"
                    >
                      {{ categoryNameMap.get(tool.category_id) }}
                    </span>
                  </div>
                  <p v-if="tool.summary" class="mt-1 line-clamp-2 text-xs leading-relaxed text-muted-foreground">
                    {{ tool.summary }}
                  </p>
                  <div v-if="tool.tags?.length" class="mt-2 flex flex-wrap gap-1">
                    <span
                      v-for="tag in tool.tags.slice(0, 4)"
                      :key="tag"
                      class="rounded border border-border/70 bg-muted/35 px-1.5 py-0.5 text-[10px] text-muted-foreground"
                    >
                      {{ tag }}
                    </span>
                  </div>
                </div>
              </div>

              <!-- Logo 下方操作按钮组 -->
              <div class="flex flex-wrap items-center gap-2 border-t border-border/70 pt-3">
                <!-- 主按钮 1：一键导入（深链或终端命令） -->
                <Button
                  v-if="tool.import_type && tool.import_type !== 'none' && tool.import_template"
                  size="sm"
                  class="h-8 flex-1 text-xs"
                  @click="runImportAction(tool)"
                >
                  <Check v-if="copiedTag === `imp:${tool.id}`" class="mr-1.5 h-3.5 w-3.5" />
                  <Terminal v-else-if="tool.import_type === 'command'" class="mr-1.5 h-3.5 w-3.5" />
                  <ArrowUpRight v-else class="mr-1.5 h-3.5 w-3.5" />
                  <span class="truncate">
                    {{
                      copiedTag === `imp:${tool.id}`
                        ? (tool.import_type === 'deeplink'
                          ? t('needtoken.memberConsole.quickUse.importTriggered')
                          : t('needtoken.common.copied'))
                        : (tool.import_label || (tool.import_type === 'command'
                          ? t('needtoken.memberConsole.quickUse.copyCommand')
                          : t('needtoken.memberConsole.quickUse.oneClickImport')))
                    }}
                  </span>
                </Button>

                <!-- 主按钮 2：下载配置 -->
                <Button
                  v-if="tool.config_template"
                  :variant="tool.import_type && tool.import_type !== 'none' && tool.import_template ? 'outline' : 'default'"
                  size="sm"
                  class="h-8 flex-1 text-xs"
                  @click="downloadConfig(tool)"
                >
                  <Check v-if="copiedTag === `dl:${tool.id}`" class="mr-1.5 h-3.5 w-3.5 text-primary" />
                  <Download v-else class="mr-1.5 h-3.5 w-3.5" />
                  <span class="truncate">
                    {{ copiedTag === `dl:${tool.id}` ? t('needtoken.memberConsole.quickUse.downloaded') : t('needtoken.memberConsole.quickUse.downloadConfig') }}
                  </span>
                </Button>

                <!-- 次级快捷操作：手动教程 / 复制给 AI 配置的一句话 -->
                <div class="flex w-full flex-wrap items-center gap-1.5">
                  <button
                    v-if="tool.manual_guide || tool.config_template"
                    type="button"
                    class="inline-flex h-7 flex-1 items-center justify-center gap-1 rounded-lg border border-border bg-muted/30 px-2.5 text-[11px] font-medium text-foreground transition-colors hover:border-primary/40 hover:text-primary"
                    @click="activeGuideTool = tool"
                  >
                    <BookOpen class="h-3 w-3 shrink-0" />
                    <span class="truncate">{{ t('needtoken.memberConsole.quickUse.manualGuide') }}</span>
                  </button>
                  <button
                    v-if="tool.ai_prompt_template"
                    type="button"
                    class="inline-flex h-7 flex-1 items-center justify-center gap-1 rounded-lg border border-primary/25 bg-primary/5 px-2.5 text-[11px] font-semibold text-primary transition-colors hover:bg-primary/12"
                    @click="copyAiPrompt(tool)"
                  >
                    <Check v-if="copiedTag === `ai:${tool.id}`" class="h-3 w-3 shrink-0" />
                    <Sparkles v-else class="h-3 w-3 shrink-0" />
                    <span class="truncate">
                      {{ copiedTag === `ai:${tool.id}` ? t('needtoken.common.copied') : t('needtoken.memberConsole.quickUse.copyAiPrompt') }}
                    </span>
                  </button>
                </div>
              </div>
            </article>
          </div>
        </div>
      </div>
    </div>
  </dialog>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  ArrowLeft,
  ArrowUpRight,
  BookOpen,
  Check,
  ChevronDown,
  Copy,
  Download,
  ExternalLink,
  Eye,
  EyeOff,
  KeyRound,
  Search,
  Sparkles,
  Terminal,
  X,
  Zap,
} from 'lucide-vue-next'
import { Button } from '@/components/ui/button'
import NeedtokenModelCombobox from './NeedtokenModelCombobox.vue'
import { API_BASE_URL, DEFAULT_MODELS } from '../../config'
import {
  formatTokenGroupsSummary,
  renderQuickToolTemplate,
  resolveTokenAvailableModels,
} from '../../lib/gatewayConsoleHelpers'
import {
  getGatewayModels,
  getGatewayQuickTools,
  listGatewayTokens,
  type GatewayGroupMetaItem,
  type GatewayModelPricingItem,
  type GatewayTokenItem,
  type QuickToolCategory,
  type QuickToolItem,
} from '../../lib/gatewayConsoleApi'

const emit = defineEmits<{
  (e: 'create-key'): void
}>()

const { t } = useI18n()

const dialogRef = ref<HTMLDialogElement | null>(null)
const keyMenuRootEl = ref<HTMLElement | null>(null)
const loading = ref(false)
const keyMenuOpen = ref(false)
const includeRealKey = ref(true)
const searchQuery = ref('')
const activeCategoryId = ref('all')
const activeGuideTool = ref<QuickToolItem | null>(null)
const copiedTag = ref('')
const brokenLogos = ref<Set<string>>(new Set())

const gatewayBaseUrl = ref(API_BASE_URL.replace(/\/v1\/?$/, ''))
const categories = ref<QuickToolCategory[]>([])
const tools = ref<QuickToolItem[]>([])

const tokens = ref<GatewayTokenItem[]>([])
const usableGroups = ref<Record<string, string>>({})
const groupRatio = ref<Record<string, number>>({})
const groupMeta = ref<Record<string, GatewayGroupMetaItem>>({})
const pricingModels = ref<GatewayModelPricingItem[]>([])
const defaultGroup = ref('default')
const userAutoGroups = ref<string[]>([])

const selectedTokenId = ref<number | null>(null)
const selectedModel = ref<string>(DEFAULT_MODELS[0] || 'gpt-5.4')

const FALLBACK_CATEGORIES: QuickToolCategory[] = [
  { id: 'cat-cli', name: 'CLI 编程工具', slug: 'cli', sort_order: 10, enabled: true },
  { id: 'cat-ide', name: 'IDE 与插件', slug: 'ide', sort_order: 20, enabled: true },
  { id: 'cat-router', name: '聚合与路由切换', slug: 'router', sort_order: 30, enabled: true },
  { id: 'cat-chat', name: '对话客户端', slug: 'chat', sort_order: 40, enabled: true },
]

const FALLBACK_TOOLS: QuickToolItem[] = [
  {
    id: 'tool-codex',
    category_id: 'cat-cli',
    name: 'Codex CLI',
    slug: 'codex',
    icon_key: 'codex',
    summary: 'OpenAI 官方终端编程助手，使用 Responses 协议直连 Need 网关。',
    tags: ['codex', 'openai', 'cli', '编程工具'],
    default_model_hint: 'gpt-5.4',
    import_type: 'command',
    import_label: '复制配置命令',
    import_template: 'export OPENAI_BASE_URL="{{BASE_URL_V1}}"\nexport OPENAI_API_KEY="{{API_KEY}}"',
    config_filename: 'config.toml',
    config_template: 'model_provider = "needtoken"\nmodel = "{{MODEL}}"\n\n[model_providers.needtoken]\nname = "NeedToken"\nbase_url = "{{BASE_URL_V1}}"\nwire_api = "responses"\nenv_key = "NEEDTOKEN_API_KEY"\n',
    manual_guide: '1. 将配置文件保存至 ~/.codex/config.toml\n2. 写入环境变量 NEEDTOKEN_API_KEY="{{API_KEY}}"\n3. 终端执行 codex 即可开始使用。',
    ai_prompt_template: '请帮我配置本地 Codex CLI（~/.codex/config.toml 与环境变量 NEEDTOKEN_API_KEY），使用 NeedToken 网关：Base URL 为 {{BASE_URL_V1}}，wire_api 为 responses，默认模型 {{MODEL}}，API Key 为 {{API_KEY}}。配置完成后请帮我验证连通性。',
    docs_url: 'https://github.com/openai/codex',
    sort_order: 10,
    enabled: true,
  },
  {
    id: 'tool-claude-code',
    category_id: 'cat-cli',
    name: 'Claude Code',
    slug: 'claude-code',
    icon_key: 'claude',
    summary: 'Anthropic 官方终端编程 Agent，支持一键写入环境变量与 settings.json。',
    tags: ['claude', 'anthropic', 'cli', '编程工具'],
    default_model_hint: 'claude-sonnet-4-6',
    import_type: 'command',
    import_label: '复制配置命令',
    import_template: 'export ANTHROPIC_BASE_URL="{{BASE_URL}}"\nexport ANTHROPIC_AUTH_TOKEN="{{API_KEY}}"\nexport ANTHROPIC_MODEL="{{MODEL}}"',
    config_filename: 'claude-code-settings.json',
    config_template: '{\n  "env": {\n    "ANTHROPIC_BASE_URL": "{{BASE_URL}}",\n    "ANTHROPIC_AUTH_TOKEN": "{{API_KEY}}",\n    "ANTHROPIC_MODEL": "{{MODEL}}"\n  }\n}\n',
    manual_guide: '1. 下载配置并合并至 ~/.claude/settings.json\n2. 重启终端并运行 claude 即可生效。',
    ai_prompt_template: '请帮我配置 Claude Code（~/.claude/settings.json），将 ANTHROPIC_BASE_URL 设为 {{BASE_URL}}，ANTHROPIC_AUTH_TOKEN 设为 {{API_KEY}}，ANTHROPIC_MODEL 设为 {{MODEL}}。',
    docs_url: 'https://docs.anthropic.com/en/docs/claude-code/overview',
    sort_order: 20,
    enabled: true,
  },
  {
    id: 'tool-gemini-cli',
    category_id: 'cat-cli',
    name: 'Gemini CLI',
    slug: 'gemini-cli',
    icon_key: 'gemini',
    summary: 'Google Gemini 终端编程工具，通过自定义 Base URL 无缝接入 Need 网关。',
    tags: ['gemini', 'google', 'cli', '编程工具'],
    default_model_hint: 'gemini-2.5-pro',
    import_type: 'command',
    import_label: '复制配置命令',
    import_template: 'export GOOGLE_GEMINI_BASE_URL="{{BASE_URL}}"\nexport GEMINI_API_KEY="{{API_KEY}}"\nexport GEMINI_MODEL="{{MODEL}}"',
    config_filename: 'gemini-cli.env',
    config_template: 'GOOGLE_GEMINI_BASE_URL={{BASE_URL}}\nGEMINI_API_KEY={{API_KEY}}\nGEMINI_MODEL={{MODEL}}\n',
    manual_guide: '1. 将上述变量追加至 ~/.zshrc 或 ~/.bashrc\n2. 执行 source 后运行 gemini。',
    ai_prompt_template: '请帮我在当前环境配置 Gemini CLI，设置 GOOGLE_GEMINI_BASE_URL={{BASE_URL}}、GEMINI_API_KEY={{API_KEY}}、GEMINI_MODEL={{MODEL}}。',
    docs_url: 'https://github.com/google-gemini/gemini-cli',
    sort_order: 30,
    enabled: true,
  },
  {
    id: 'tool-opencode',
    category_id: 'cat-cli',
    name: 'OpenCode',
    slug: 'opencode',
    icon_key: 'opencode',
    summary: '开源多模型终端编码助手，支持导入 opencode.json 自定义 NeedToken Provider。',
    tags: ['opencode', 'cli', '编程工具'],
    default_model_hint: 'claude-sonnet-4-6',
    import_type: 'none',
    config_filename: 'opencode.json',
    config_template: '{\n  "$schema": "https://opencode.ai/config.json",\n  "model": "needtoken/{{MODEL}}",\n  "providers": {\n    "needtoken": {\n      "name": "NeedToken",\n      "package": "@opencode/ai/providers/openai-compatible",\n      "settings": {\n        "baseURL": "{{BASE_URL_V1}}",\n        "apiKey": "{{API_KEY}}"\n      },\n      "models": {\n        "{{MODEL}}": { "name": "{{MODEL}}" }\n      }\n    }\n  }\n}\n',
    manual_guide: '1. 下载 opencode.json 放入项目根目录或 ~/.config/opencode/\n2. 运行 opencode 即可调用 {{MODEL}}。',
    ai_prompt_template: '请帮我生成并写入 OpenCode 配置文件 opencode.json，使用 NeedToken OpenAI-Compatible Provider，baseURL={{BASE_URL_V1}}，apiKey={{API_KEY}}，默认模型 {{MODEL}}。',
    docs_url: 'https://opencode.ai/docs/providers/',
    sort_order: 40,
    enabled: true,
  },
  {
    id: 'tool-cc-switch',
    category_id: 'cat-router',
    name: 'CC Switch',
    slug: 'cc-switch',
    icon_key: 'ccswitch',
    summary: '桌面端一键切换 Claude / Codex / Gemini 供应商配置，支持深度链接直接导入。',
    tags: ['cc switch', 'ccswitch', 'router', '编程工具'],
    default_model_hint: 'claude-sonnet-4-6',
    import_type: 'deeplink',
    import_label: '一键导入 CC Switch',
    import_template: 'ccswitch://v1/import?resource=provider&app=claude&name=NeedToken&endpoint={{BASE_URL}}&apiKey={{API_KEY}}&model={{MODEL}}',
    config_filename: 'cc-switch-needtoken.json',
    config_template: '{\n  "name": "NeedToken",\n  "endpoint": "{{BASE_URL}}",\n  "apiKey": "{{API_KEY}}",\n  "model": "{{MODEL}}"\n}\n',
    manual_guide: '1. 安装 CC Switch 后点击「一键导入」直接唤起应用\n2. 如未唤起，可下载 JSON 在应用内手动新增供应商。',
    ai_prompt_template: '请在 CC Switch 中新增 NeedToken 供应商配置：Endpoint={{BASE_URL}}，API Key={{API_KEY}}，默认模型={{MODEL}}。',
    docs_url: 'https://ccswitch.io/',
    sort_order: 50,
    enabled: true,
  },
  {
    id: 'tool-cursor',
    category_id: 'cat-ide',
    name: 'Cursor / Cline',
    slug: 'cursor',
    icon_key: 'cursor',
    summary: '在 Cursor、Cline 或 Roo Code 中覆盖 OpenAI Base URL，一键解锁全系模型。',
    tags: ['cursor', 'cline', 'ide', '编程工具'],
    default_model_hint: 'claude-sonnet-4-6',
    import_type: 'none',
    config_filename: 'cursor-needtoken.txt',
    config_template: 'Provider: OpenAI Compatible\nBase URL: {{BASE_URL_V1}}\nAPI Key: {{API_KEY}}\nModel: {{MODEL}}\n',
    manual_guide: '1. 打开 Cursor Settings → Models → 勾选 Override OpenAI Base URL\n2. 填入 {{BASE_URL_V1}} 与您的 API Key\n3. 添加模型 {{MODEL}} 即可使用。',
    ai_prompt_template: '请指导或帮我将当前 IDE 插件（Cline / Roo Code / Continue）的 OpenAI Compatible Provider 配置为 Base URL={{BASE_URL_V1}}、API Key={{API_KEY}}、Model={{MODEL}}。',
    docs_url: 'https://docs.cursor.com/',
    sort_order: 60,
    enabled: true,
  },
  {
    id: 'tool-t3-chat',
    category_id: 'cat-chat',
    name: 'T3 Chat / Cherry Studio',
    slug: 't3',
    icon_key: 't3',
    summary: '多模型高速对话客户端，支持 OpenAI Compatible 自定义接口快速导入。',
    tags: ['t3', 'cherry studio', 'chat', '对话'],
    default_model_hint: 'gpt-5.4',
    import_type: 'none',
    config_filename: 'needtoken-openai-compatible.json',
    config_template: '{\n  "provider": "NeedToken",\n  "base_url": "{{BASE_URL_V1}}",\n  "api_key": "{{API_KEY}}",\n  "default_model": "{{MODEL}}"\n}\n',
    manual_guide: '1. 打开客户端设置 → 自定义 OpenAI 接口\n2. 填写接口地址 {{BASE_URL_V1}} 与密钥 {{API_KEY}}\n3. 获取模型列表后即可直接对话。',
    ai_prompt_template: '请帮我用 OpenAI Compatible 方式配置对话客户端：接口地址 {{BASE_URL_V1}}，API Key {{API_KEY}}，默认模型 {{MODEL}}。',
    docs_url: 'https://docs.cherry-ai.com/',
    sort_order: 70,
    enabled: true,
  },
]

const effectiveBaseUrl = computed(() => (gatewayBaseUrl.value || API_BASE_URL).replace(/\/+$/, '').replace(/\/v1$/, ''))
const effectiveBaseUrlV1 = computed(() => `${effectiveBaseUrl.value}/v1`)

const activeTokens = computed(() => tokens.value.filter(tok => tok.status === 1 && tok.key))
const selectedToken = computed(() => activeTokens.value.find(tok => tok.id === selectedTokenId.value) || activeTokens.value[0] || null)

const formatTokenGroups = (tok: GatewayTokenItem) =>
  formatTokenGroupsSummary(tok, defaultGroup.value, groupMeta.value, usableGroups.value, groupRatio.value)

const selectedTokenLabel = computed(() => {
  if (!selectedToken.value) return t('needtoken.memberConsole.keys.emptyTitle')
  const tok = selectedToken.value
  return `${tok.name || `Key #${tok.id}`} (${formatTokenGroups(tok)} · ${tok.masked_key})`
})

const availableModels = computed(() =>
  resolveTokenAvailableModels({
    token: selectedToken.value,
    defaultGroup: defaultGroup.value,
    userAutoGroups: userAutoGroups.value,
    groupMeta: groupMeta.value,
    pricingModels: pricingModels.value,
    fallbackModels: [...DEFAULT_MODELS],
  }),
)

watch([selectedTokenId, availableModels], () => {
  const list = availableModels.value
  if (!list.length) return
  if (!selectedModel.value || !list.includes(selectedModel.value)) {
    selectedModel.value = list[0]!
  }
})

const enabledCategories = computed(() => {
  const list = categories.value.length ? categories.value : FALLBACK_CATEGORIES
  return list.filter(c => c.enabled !== false).sort((a, b) => a.sort_order - b.sort_order)
})

const categoryNameMap = computed(() => {
  const m = new Map<string, string>()
  for (const cat of enabledCategories.value) {
    m.set(cat.id, cat.name)
  }
  return m
})

const enabledTools = computed(() => {
  const list = tools.value.length ? tools.value : FALLBACK_TOOLS
  return list.filter(item => item.enabled !== false).sort((a, b) => a.sort_order - b.sort_order)
})

const filteredTools = computed(() => {
  const q = searchQuery.value.trim().toLowerCase()
  return enabledTools.value.filter((item) => {
    if (activeCategoryId.value !== 'all' && item.category_id !== activeCategoryId.value) {
      return false
    }
    if (!q) return true
    const catName = categoryNameMap.value.get(item.category_id) || ''
    const haystack = [
      item.name,
      item.slug,
      item.summary || '',
      catName,
      ...(item.tags || []),
    ].join(' ').toLowerCase()
    return haystack.includes(q)
  })
})

const resolveModelForTool = (tool: QuickToolItem): string => {
  const hint = String(tool.default_model_hint || '').trim()
  if (selectedModel.value) return selectedModel.value
  if (hint && availableModels.value.includes(hint)) return hint
  return availableModels.value[0] || hint || 'gpt-5.4'
}

const buildRenderVars = (tool: QuickToolItem) => {
  const tok = selectedToken.value
  const apiKey = includeRealKey.value && tok?.key ? tok.key : '<YOUR_NEEDTOKEN_API_KEY>'
  return {
    apiKey,
    baseUrl: effectiveBaseUrl.value,
    baseUrlV1: effectiveBaseUrlV1.value,
    model: resolveModelForTool(tool),
    tokenName: tok?.name || 'NeedToken',
    group: tok ? formatTokenGroups(tok) : defaultGroup.value,
  }
}

const renderToolText = (tool: QuickToolItem, template?: string, encodeForUrl = false) =>
  renderQuickToolTemplate(template || '', buildRenderVars(tool), encodeForUrl)

const logoGlyph = (tool: QuickToolItem): string => {
  const key = (tool.icon_key || tool.slug || tool.name || '').toLowerCase()
  if (key.includes('codex') || key.includes('openai')) return 'CX'
  if (key.includes('claude')) return 'CC'
  if (key.includes('gemini')) return 'GM'
  if (key.includes('opencode')) return 'OC'
  if (key.includes('ccswitch') || key.includes('cc-switch')) return 'SW'
  if (key.includes('cursor')) return 'CR'
  if (key.includes('cline') || key.includes('roo')) return 'CL'
  if (key.includes('t3') || key.includes('cherry')) return 'T3'
  return (tool.name || 'AI').slice(0, 2).toUpperCase()
}

const logoBadgeStyle = (tool: QuickToolItem): Record<string, string> => {
  const key = (tool.icon_key || tool.slug || '').toLowerCase()
  if (key.includes('codex') || key.includes('openai')) return { background: 'rgba(16, 163, 127, 0.14)', color: '#0d9473' }
  if (key.includes('claude')) return { background: 'rgba(194, 93, 63, 0.14)', color: '#c25d3f' }
  if (key.includes('gemini')) return { background: 'rgba(59, 126, 240, 0.14)', color: '#3b7ef0' }
  if (key.includes('opencode')) return { background: 'rgba(124, 77, 147, 0.14)', color: '#7c4d93' }
  if (key.includes('ccswitch') || key.includes('cc-switch')) return { background: 'rgba(13, 148, 115, 0.16)', color: '#0a7a5e' }
  if (key.includes('cursor')) return { background: 'rgba(71, 91, 116, 0.16)', color: '#475b74' }
  if (key.includes('t3') || key.includes('cherry')) return { background: 'rgba(180, 118, 18, 0.14)', color: '#9a6510' }
  return { background: 'var(--ui-accent-soft)', color: 'var(--ui-accent)' }
}

const markLogoBroken = (toolId: string) => {
  const next = new Set(brokenLogos.value)
  next.add(toolId)
  brokenLogos.value = next
}

let feedbackTimer: ReturnType<typeof setTimeout> | undefined
const setFeedback = (tag: string) => {
  if (feedbackTimer) clearTimeout(feedbackTimer)
  copiedTag.value = tag
  feedbackTimer = setTimeout(() => { copiedTag.value = '' }, 1800)
}

const copyText = async (text: string, tag: string) => {
  try {
    await navigator.clipboard.writeText(text)
    setFeedback(tag)
  } catch {
    copiedTag.value = ''
  }
}

const runImportAction = async (tool: QuickToolItem) => {
  if (!tool.import_template) return
  if (tool.import_type === 'deeplink') {
    const uri = renderToolText(tool, tool.import_template, true)
    if (uri && !/^javascript:/i.test(uri)) {
      window.location.href = uri
      setFeedback(`imp:${tool.id}`)
    }
    return
  }
  const cmd = renderToolText(tool, tool.import_template, false)
  await copyText(cmd, `imp:${tool.id}`)
}

const downloadConfig = (tool: QuickToolItem) => {
  if (!tool.config_template) return
  const content = renderToolText(tool, tool.config_template, false)
  const filename = (tool.config_filename || `${tool.slug || 'needtoken'}-config.txt`).trim()
  const blob = new Blob([content], { type: 'text/plain;charset=utf-8' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  document.body.appendChild(a)
  a.click()
  a.remove()
  setTimeout(() => URL.revokeObjectURL(url), 1000)
  setFeedback(`dl:${tool.id}`)
}

const copyAiPrompt = async (tool: QuickToolItem) => {
  if (!tool.ai_prompt_template) return
  const prompt = renderToolText(tool, tool.ai_prompt_template, false)
  await copyText(prompt, `ai:${tool.id}`)
}

const selectToken = (id: number) => {
  selectedTokenId.value = id
  keyMenuOpen.value = false
}

const goCreateKey = () => {
  closeDrawer()
  emit('create-key')
}

const loadAll = async (preferredTokenId?: number) => {
  loading.value = true
  try {
    const [quickRes, tokRes, modRes] = await Promise.all([
      getGatewayQuickTools().catch(() => null),
      listGatewayTokens().catch(() => null),
      getGatewayModels().catch(() => null),
    ])
    if (quickRes) {
      if (quickRes.gateway_base_url) gatewayBaseUrl.value = quickRes.gateway_base_url
      categories.value = Array.isArray(quickRes.categories) ? quickRes.categories : []
      tools.value = Array.isArray(quickRes.tools) ? quickRes.tools : []
    }
    if (modRes) {
      pricingModels.value = Array.isArray(modRes.models) ? modRes.models : []
      usableGroups.value = { ...(modRes.usable_groups || {}) }
      groupRatio.value = { ...(modRes.group_ratio || {}) }
      groupMeta.value = { ...(modRes.group_meta || {}) }
      defaultGroup.value = modRes.default_group || defaultGroup.value
      userAutoGroups.value = Array.isArray(modRes.auto_groups) ? modRes.auto_groups : []
    }
    if (tokRes) {
      tokens.value = Array.isArray(tokRes.items) ? tokRes.items : []
      usableGroups.value = { ...usableGroups.value, ...(tokRes.usable_groups || {}) }
      groupRatio.value = { ...groupRatio.value, ...(tokRes.group_ratio || {}) }
      groupMeta.value = { ...groupMeta.value, ...(tokRes.group_meta || {}) }
      defaultGroup.value = tokRes.default_group || defaultGroup.value
      if (Array.isArray(tokRes.auto_groups) && tokRes.auto_groups.length) {
        userAutoGroups.value = tokRes.auto_groups
      }
      const enabled = activeTokens.value
      if (preferredTokenId && enabled.some(t => t.id === preferredTokenId)) {
        selectedTokenId.value = preferredTokenId
      } else if (!selectedTokenId.value || !enabled.some(t => t.id === selectedTokenId.value)) {
        selectedTokenId.value = enabled[0]?.id ?? null
      }
    }
  } finally {
    loading.value = false
  }
}

const openDrawer = (preferredTokenId?: number) => {
  activeGuideTool.value = null
  keyMenuOpen.value = false
  if (preferredTokenId) {
    selectedTokenId.value = preferredTokenId
  }
  const el = dialogRef.value
  if (el && !el.open) {
    el.showModal()
  }
  void loadAll(preferredTokenId)
}

const closeDrawer = () => {
  keyMenuOpen.value = false
  const el = dialogRef.value
  if (el?.open) {
    el.close()
  }
}

let pressedOutside = false
const isOutsideDialog = (e: PointerEvent, dialog: HTMLDialogElement) => {
  const r = dialog.getBoundingClientRect()
  return e.clientX < r.left || e.clientX > r.right || e.clientY < r.top || e.clientY > r.bottom
}
const onPointerDown = (e: PointerEvent) => {
  if (!dialogRef.value) return
  pressedOutside = isOutsideDialog(e, dialogRef.value)
}
const onPointerUp = (e: PointerEvent) => {
  if (!dialogRef.value) return
  if (pressedOutside && isOutsideDialog(e, dialogRef.value)) {
    closeDrawer()
  }
  pressedOutside = false
}

const onDocMouseDown = (event: MouseEvent) => {
  if (!keyMenuOpen.value || !keyMenuRootEl.value) return
  const target = event.target as Node | null
  if (target && !keyMenuRootEl.value.contains(target)) {
    keyMenuOpen.value = false
  }
}

onMounted(() => {
  document.addEventListener('mousedown', onDocMouseDown)
})

onUnmounted(() => {
  if (feedbackTimer) clearTimeout(feedbackTimer)
  document.removeEventListener('mousedown', onDocMouseDown)
})

defineExpose({ openDrawer, closeDrawer })
</script>

<style scoped>
.nt-quick-drawer__dropdown {
  position: absolute;
  left: 0;
  right: 0;
  top: calc(100% + 6px);
  z-index: 60;
  max-height: 240px;
  overflow-y: auto;
  border: 1px solid var(--ui-border-strong);
  border-radius: 12px;
  background: var(--ui-bg-elevated);
  box-shadow: var(--nt-popover-shadow);
  padding: 6px;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.nt-quick-drawer__dropdown-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  width: 100%;
  border: 0;
  border-radius: 8px;
  background: transparent;
  padding: 7px 9px;
  text-align: left;
  cursor: pointer;
}
.nt-quick-drawer__dropdown-item:hover {
  background: var(--ui-accent-soft);
}
.nt-quick-drawer__dropdown-item.is-selected {
  background: var(--nt-accent-subtle);
}
.nt-quick-tool-logo {
  display: grid;
  place-items: center;
  border-radius: 14px;
  border: 1px solid var(--ui-border);
}
</style>
