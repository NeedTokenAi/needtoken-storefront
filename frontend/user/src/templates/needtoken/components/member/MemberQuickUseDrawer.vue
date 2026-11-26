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

      <!-- 会员专属上下文配置条（密钥 + 动态模型策略 + 真实密钥开关） -->
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

        <div class="grid grid-cols-1 items-end gap-3 md:grid-cols-12">
          <!-- 会员自己的 API Key（就地渲染，不依赖 body Portal） -->
          <div ref="keyMenuRootEl" class="relative md:col-span-5">
            <label class="mb-1.5 block text-[11px] font-semibold text-muted-foreground">
              {{ t('needtoken.memberConsole.quickUse.selectKey') }}
            </label>
            <button
              type="button"
              class="nt-theme-select__trigger inline-flex w-full items-center justify-between"
              :data-state="keyMenuOpen ? 'open' : 'closed'"
              :aria-expanded="keyMenuOpen"
              :disabled="activeTokens.length === 0"
              @click="keyMenuOpen = !keyMenuOpen"
            >
              <span class="min-w-0 flex-1 truncate text-left text-xs font-medium">
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

          <!-- 模型模式：默认自动同步网关全部最新模型，可选指定默认模型 -->
          <div class="md:col-span-5">
            <label class="mb-1.5 block text-[11px] font-semibold text-muted-foreground">
              {{ t('needtoken.memberConsole.quickUse.modelModeLabel') }}
            </label>
            <NeedtokenModelCombobox
              v-model="selectedModel"
              :models="availableModels"
              :label="t('needtoken.memberConsole.quickUse.modelModeLabel')"
              :auto-option-label="t('needtoken.memberConsole.quickUse.autoAllModelsLabel', { n: availableModels.length })"
              :search-placeholder="t('needtoken.memberConsole.playground.searchModelPlaceholder')"
            />
          </div>

          <!-- 明文 Key / 占位符切换（与下拉框严密齐平 40px） -->
          <div class="md:col-span-2">
            <button
              type="button"
              class="inline-flex h-10 w-full items-center justify-center gap-1.5 rounded-[11px] border px-3 text-xs font-semibold transition-colors"
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

          <div class="flex items-center gap-3.5 rounded-2xl border bg-muted/20 p-4">
            <QuickToolLogo
              :name="activeGuideTool.name"
              :slug="activeGuideTool.slug"
              :icon-key="activeGuideTool.icon_key"
              :logo-url="activeGuideTool.logo_url"
            />
            <div class="min-w-0">
              <h2 class="text-base font-bold text-foreground">{{ activeGuideTool.name }}</h2>
              <p v-if="activeGuideTool.summary" class="mt-0.5 text-xs text-muted-foreground">{{ activeGuideTool.summary }}</p>
            </div>
          </div>

          <!-- AI 一句话自动配置（放首位方便立即使用） -->
          <div v-if="activeGuideTool.ai_prompt_template" class="space-y-2 rounded-xl border border-primary/30 bg-primary/5 p-4">
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
                <QuickToolLogo
                  :name="tool.name"
                  :slug="tool.slug"
                  :icon-key="tool.icon_key"
                  :logo-url="tool.logo_url"
                />
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

              <!-- 深链未唤起提示条（例如未安装 CC Switch） -->
              <div
                v-if="deeplinkMissedToolId === tool.id"
                class="rounded-xl border border-amber-500/35 bg-amber-500/10 p-3 text-[11.5px] leading-relaxed text-amber-900 dark:text-amber-200"
                role="status"
              >
                <div class="flex items-start gap-2">
                  <AlertCircle class="mt-0.5 h-3.5 w-3.5 shrink-0" />
                  <div class="min-w-0 flex-1 space-y-1.5">
                    <p>{{ t('needtoken.memberConsole.quickUse.deeplinkNotInstalledHint', { name: tool.name }) }}</p>
                    <div class="flex flex-wrap items-center gap-2">
                      <button
                        v-if="tool.ai_prompt_template"
                        type="button"
                        class="inline-flex items-center gap-1 rounded-md bg-primary px-2.5 py-1 text-[11px] font-semibold text-primary-foreground hover:opacity-90"
                        @click="copyAiPrompt(tool)"
                      >
                        <Sparkles class="h-3 w-3" />
                        <span>{{ t('needtoken.memberConsole.quickUse.useAiPromptInstead') }}</span>
                      </button>
                      <a
                        v-if="tool.docs_url && /^https?:\/\//i.test(tool.docs_url)"
                        :href="tool.docs_url"
                        target="_blank"
                        rel="noopener noreferrer"
                        class="inline-flex items-center gap-1 font-semibold underline"
                      >
                        <span>{{ t('needtoken.memberConsole.quickUse.downloadClient') }}</span>
                        <ExternalLink class="h-3 w-3" />
                      </a>
                    </div>
                  </div>
                </div>
              </div>

              <!-- Logo 下方操作按钮组（AI 配置优先；深链工具提供一键导入 + 未安装探测） -->
              <div class="flex flex-col gap-2 border-t border-border/70 pt-3">
                <div class="flex flex-wrap items-center gap-2">
                  <!-- 场景 A：原生深链工具（如 CC Switch）：复制给 AI 配置（第一推荐） + 一键导入 -->
                  <template v-if="isDeeplinkTool(tool)">
                    <Button
                      v-if="tool.ai_prompt_template"
                      size="sm"
                      class="h-8 flex-1 text-xs"
                      @click="copyAiPrompt(tool)"
                    >
                      <Check v-if="copiedTag === `ai:${tool.id}`" class="mr-1.5 h-3.5 w-3.5" />
                      <Sparkles v-else class="mr-1.5 h-3.5 w-3.5" />
                      <span class="truncate">
                        {{ copiedTag === `ai:${tool.id}` ? t('needtoken.common.copied') : t('needtoken.memberConsole.quickUse.copyAiPrompt') }}
                      </span>
                    </Button>
                    <Button
                      variant="outline"
                      size="sm"
                      class="h-8 flex-1 text-xs"
                      :disabled="launchingToolId === tool.id"
                      @click="runDeeplinkImport(tool)"
                    >
                      <Check v-if="copiedTag === `imp:${tool.id}`" class="mr-1.5 h-3.5 w-3.5 text-primary" />
                      <ArrowUpRight v-else class="mr-1.5 h-3.5 w-3.5" />
                      <span class="truncate">
                        {{
                          launchingToolId === tool.id
                            ? t('needtoken.memberConsole.quickUse.launchingClient')
                            : (copiedTag === `imp:${tool.id}`
                              ? t('needtoken.memberConsole.quickUse.importTriggered')
                              : (tool.import_label || t('needtoken.memberConsole.quickUse.oneClickImport')))
                        }}
                      </span>
                    </Button>
                  </template>

                  <!-- 场景 B：普通 CLI / IDE / 对话客户端：第一主按钮为「复制给 AI 配置」 -->
                  <template v-else>
                    <Button
                      v-if="tool.ai_prompt_template"
                      size="sm"
                      class="h-8 flex-1 text-xs"
                      @click="copyAiPrompt(tool)"
                    >
                      <Check v-if="copiedTag === `ai:${tool.id}`" class="mr-1.5 h-3.5 w-3.5" />
                      <Sparkles v-else class="mr-1.5 h-3.5 w-3.5" />
                      <span class="truncate">
                        {{ copiedTag === `ai:${tool.id}` ? t('needtoken.common.copied') : t('needtoken.memberConsole.quickUse.copyAiPrompt') }}
                      </span>
                    </Button>
                    <Button
                      v-if="tool.config_template"
                      :variant="tool.ai_prompt_template ? 'outline' : 'default'"
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
                  </template>
                </div>

                <!-- 次级按钮栏：深链工具补「下载配置」，全部工具提供「手动配置教程」 -->
                <div class="flex flex-wrap items-center gap-1.5">
                  <button
                    v-if="isDeeplinkTool(tool) && tool.config_template"
                    type="button"
                    class="inline-flex h-7 flex-1 items-center justify-center gap-1 rounded-lg border border-border bg-muted/30 px-2.5 text-[11px] font-medium text-foreground transition-colors hover:border-primary/40 hover:text-primary"
                    @click="downloadConfig(tool)"
                  >
                    <Check v-if="copiedTag === `dl:${tool.id}`" class="h-3 w-3 shrink-0 text-primary" />
                    <Download v-else class="h-3 w-3 shrink-0" />
                    <span class="truncate">
                      {{ copiedTag === `dl:${tool.id}` ? t('needtoken.memberConsole.quickUse.downloaded') : t('needtoken.memberConsole.quickUse.downloadConfig') }}
                    </span>
                  </button>
                  <button
                    v-if="tool.manual_guide || tool.config_template"
                    type="button"
                    class="inline-flex h-7 flex-1 items-center justify-center gap-1 rounded-lg border border-border bg-muted/30 px-2.5 text-[11px] font-medium text-foreground transition-colors hover:border-primary/40 hover:text-primary"
                    @click="activeGuideTool = tool"
                  >
                    <BookOpen class="h-3 w-3 shrink-0" />
                    <span class="truncate">{{ t('needtoken.memberConsole.quickUse.manualGuide') }}</span>
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
  AlertCircle,
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
  X,
  Zap,
} from 'lucide-vue-next'
import { Button } from '@/components/ui/button'
import NeedtokenModelCombobox from './NeedtokenModelCombobox.vue'
import QuickToolLogo from './QuickToolLogo.vue'
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
const launchingToolId = ref('')
const deeplinkMissedToolId = ref('')

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
// 空字符串表示「自动跟随网关全部模型（动态同步 /v1/models）」，不写死单一模型
const selectedModel = ref<string>('')

const FALLBACK_CATEGORIES: QuickToolCategory[] = [
  { id: 'cat_coding_cli', name: '编程 CLI 工具', slug: 'coding-cli', sort_order: 10, enabled: true },
  { id: 'cat_switch_proxy', name: '切换与配置管理', slug: 'switch-proxy', sort_order: 20, enabled: true },
  { id: 'cat_ide_plugins', name: 'IDE 与编辑器插件', slug: 'ide-plugins', sort_order: 30, enabled: true },
  { id: 'cat_chat_clients', name: 'AI 聊天客户端', slug: 'chat-clients', sort_order: 40, enabled: true },
]

const FALLBACK_TOOLS: QuickToolItem[] = [
  {
    id: 'tool_codex_cli',
    category_id: 'cat_coding_cli',
    name: 'Codex CLI',
    slug: 'codex',
    icon_key: 'codex',
    summary: 'OpenAI 官方终端编程助手，动态接入全部授权模型（含新同步版本）',
    tags: ['codex', 'openai', 'cli', '编程工具'],
    import_type: 'none',
    config_filename: 'config.toml',
    config_template: '# 保存为 ~/.codex/config.toml，并在 shell 配置文件持久化设置 NEEDTOKEN_API_KEY={{API_KEY}}\n# 不写死单一模型，后台同步新模型（如 GPT-6.2）后自动可用\nmodel_provider = "needtoken"\n\n[model_providers.needtoken]\nname = "NeedToken"\nbase_url = "{{BASE_URL_V1}}"\nwire_api = "responses"\nenv_key = "NEEDTOKEN_API_KEY"\n',
    manual_guide: '1. 将下载的 config.toml 保存至 ~/.codex/config.toml。\n2. 在 ~/.zshrc 或 ~/.bashrc 永久写入 export NEEDTOKEN_API_KEY="{{API_KEY}}" 并 source。\n3. 启动 codex 后可随时通过 /model 或 {{MODELS_ENDPOINT}} 切换全部最新模型。',
    ai_prompt_template: '请帮我配置本机 OpenAI Codex CLI 接入 NeedToken 网关：更新 ~/.codex/config.toml 设置 model_provider="needtoken"、base_url="{{BASE_URL_V1}}"、wire_api="responses"、env_key="NEEDTOKEN_API_KEY"（不写死单一旧模型），并在 shell 配置文件持久化写入 export NEEDTOKEN_API_KEY="{{API_KEY}}"，完成后请求 GET {{MODELS_ENDPOINT}} 校验可用模型列表。',
    docs_url: 'https://github.com/openai/codex',
    sort_order: 10,
    enabled: true,
  },
  {
    id: 'tool_claude_code',
    category_id: 'cat_coding_cli',
    name: 'Claude Code',
    slug: 'claude-code',
    icon_key: 'claude',
    summary: 'Anthropic 官方终端编码 Agent，自动同步网关最新模型列表',
    tags: ['claude', 'anthropic', 'cli', '编程工具'],
    import_type: 'none',
    config_filename: 'claude-settings.json',
    config_template: '{\n  "env": {\n    "ANTHROPIC_BASE_URL": "{{BASE_URL}}",\n    "ANTHROPIC_AUTH_TOKEN": "{{API_KEY}}"\n  }\n}\n',
    manual_guide: '1. 将配置合并到 ~/.claude/settings.json（未写死单一模型）。\n2. 重启终端并运行 claude，在会话中用 /model 随时切换最新模型。',
    ai_prompt_template: '请帮我配置本机 Claude Code（~/.claude/settings.json），合并设置 env.ANTHROPIC_BASE_URL="{{BASE_URL}}" 与 env.ANTHROPIC_AUTH_TOKEN="{{API_KEY}}"，不要写死单一模型以便自动跟随网关新增模型，并请求 GET {{MODELS_ENDPOINT}} 核对可用模型。',
    docs_url: 'https://docs.anthropic.com/en/docs/claude-code/overview',
    sort_order: 20,
    enabled: true,
  },
  {
    id: 'tool_gemini_cli',
    category_id: 'cat_coding_cli',
    name: 'Gemini CLI',
    slug: 'gemini',
    icon_key: 'gemini',
    summary: 'Google Gemini 终端 AI 助手，动态跟随网关全量可用模型',
    tags: ['gemini', 'google', 'cli', '编程工具'],
    import_type: 'none',
    config_filename: 'gemini.env',
    config_template: 'GOOGLE_GEMINI_BASE_URL={{BASE_URL}}\nGEMINI_API_KEY={{API_KEY}}\n',
    manual_guide: '1. 将配置写入 ~/.gemini/.env（未锁定单一模型）。\n2. 运行 gemini 即可按需使用全部最新模型。',
    ai_prompt_template: '请帮我配置本机 Gemini CLI（~/.gemini/.env），设置 GOOGLE_GEMINI_BASE_URL={{BASE_URL}} 与 GEMINI_API_KEY={{API_KEY}}，不写死单一模型，并通过 GET {{MODELS_ENDPOINT}} 核对可用模型列表。',
    docs_url: 'https://github.com/google-gemini/gemini-cli',
    sort_order: 30,
    enabled: true,
  },
  {
    id: 'tool_opencode',
    category_id: 'cat_coding_cli',
    name: 'OpenCode',
    slug: 'opencode',
    icon_key: 'opencode',
    summary: '开源多模型终端编程代理，自动注入当前密钥全量可用模型',
    tags: ['opencode', 'cli', '编程工具'],
    import_type: 'none',
    config_filename: 'opencode.json',
    config_template: '{\n  "$schema": "https://opencode.ai/config.json",\n  "provider": {\n    "needtoken": {\n      "npm": "@ai-sdk/openai-compatible",\n      "name": "NeedToken",\n      "options": {\n        "baseURL": "{{BASE_URL_V1}}",\n        "apiKey": "{{API_KEY}}"\n      },\n      "models": {{ALL_MODELS_MAP_JSON}}\n    }\n  }\n}\n',
    manual_guide: '1. 下载 opencode.json 放入 ~/.config/opencode/opencode.json，已包含你当前密钥的全部模型。\n2. 后台新增模型后重新下载或让 AI 从 {{MODELS_ENDPOINT}} 刷新 models 字段即可。',
    ai_prompt_template: '请帮我配置并动态同步本机 OpenCode（~/.config/opencode/opencode.json）：先请求 GET {{MODELS_ENDPOINT}}（Bearer {{API_KEY}}）获取全部最新模型，将 NeedToken（baseURL={{BASE_URL_V1}}、apiKey={{API_KEY}}）及全量模型写入 provider.needtoken.models。',
    docs_url: 'https://opencode.ai/docs/providers/',
    sort_order: 40,
    enabled: true,
  },
  {
    id: 'tool_cc_switch',
    category_id: 'cat_switch_proxy',
    name: 'CC Switch',
    slug: 'cc-switch',
    icon_key: 'ccswitch',
    summary: 'Claude Code / Codex / Gemini 桌面配置切换器，支持协议深链一键导入',
    tags: ['cc switch', 'cc-switch', 'claude', 'codex'],
    import_type: 'deeplink',
    import_label: '一键导入 CC Switch',
    import_template: 'ccswitch://v1/import?resource=provider&app=claude&name=NeedToken&endpoint={{BASE_URL}}&apiKey={{API_KEY}}',
    config_filename: 'cc-switch-needtoken.json',
    config_template: '{\n  "name": "NeedToken ({{GROUP}})",\n  "endpoint": "{{BASE_URL}}",\n  "openaiBaseUrl": "{{BASE_URL_V1}}",\n  "modelsEndpoint": "{{MODELS_ENDPOINT}}",\n  "apiKey": "{{API_KEY}}"\n}\n',
    manual_guide: '1. 已安装 CC Switch 时点击「一键导入」直接唤起应用。\n2. 未安装时建议优先使用「复制给 AI 配置」或先下载安装 CC Switch。',
    ai_prompt_template: '请帮我在本机配置 NeedToken 网关（如已安装 CC Switch 则新增 Provider，否则直接帮我配置本机 CLI 工具）：Anthropic 地址 {{BASE_URL}}，OpenAI 地址 {{BASE_URL_V1}}，API Key {{API_KEY}}，通过 GET {{MODELS_ENDPOINT}} 自动同步全部可用模型。',
    docs_url: 'https://github.com/farion1231/cc-switch/releases/latest',
    sort_order: 10,
    enabled: true,
  },
  {
    id: 'tool_cursor',
    category_id: 'cat_ide_plugins',
    name: 'Cursor',
    slug: 'cursor',
    icon_key: 'cursor',
    summary: 'AI 原生代码编辑器，通过自定义 OpenAI Base URL 动态拉取全部模型',
    tags: ['cursor', 'ide', '编程工具'],
    import_type: 'none',
    config_filename: 'cursor-needtoken.json',
    config_template: '{\n  "openai_api_key": "{{API_KEY}}",\n  "override_openai_base_url": "{{BASE_URL_V1}}",\n  "models_endpoint": "{{MODELS_ENDPOINT}}",\n  "available_models": {{ALL_MODELS_JSON}}\n}\n',
    manual_guide: '1. 打开 Cursor Settings → Models，填入 API Key 与 Override OpenAI Base URL {{BASE_URL_V1}}。\n2. 验证后按 {{MODELS_ENDPOINT}} 返回的最新模型列表启用所需模型。',
    ai_prompt_template: '请帮我配置 Cursor 接入 NeedToken 网关：Override OpenAI Base URL={{BASE_URL_V1}}，API Key={{API_KEY}}，并请求 GET {{MODELS_ENDPOINT}} 获取全部最新模型列表。',
    docs_url: 'https://docs.cursor.com/',
    sort_order: 10,
    enabled: true,
  },
  {
    id: 'tool_cline',
    category_id: 'cat_ide_plugins',
    name: 'Cline / Roo Code',
    slug: 'cline',
    icon_key: 'cline',
    summary: 'VS Code 自主编码插件，通过 OpenAI Compatible 动态同步网关全部模型',
    tags: ['cline', 'roo code', 'vscode', 'ide'],
    import_type: 'none',
    config_filename: 'cline-needtoken-profile.json',
    config_template: '{\n  "apiProvider": "openai",\n  "openAiBaseUrl": "{{BASE_URL_V1}}",\n  "openAiApiKey": "{{API_KEY}}",\n  "modelsEndpoint": "{{MODELS_ENDPOINT}}",\n  "availableModels": {{ALL_MODELS_JSON}}\n}\n',
    manual_guide: '1. 在 VS Code 打开 Cline / Roo Code 设置，选择 OpenAI Compatible。\n2. 填入 Base URL {{BASE_URL_V1}} 与 API Key，点击刷新模型列表即可同步全部最新模型。',
    ai_prompt_template: '请帮我配置 VS Code 的 Cline / Roo Code 使用 NeedToken 网关（OpenAI Compatible）：Base URL={{BASE_URL_V1}}，API Key={{API_KEY}}，通过 GET {{MODELS_ENDPOINT}} 动态加载全部最新模型。',
    docs_url: 'https://docs.cline.bot/',
    sort_order: 20,
    enabled: true,
  },
  {
    id: 'tool_t3_chat',
    category_id: 'cat_chat_clients',
    name: 'T3 Chat / Cherry Studio',
    slug: 't3',
    icon_key: 't3',
    summary: '高速 AI 聊天与多模型对比客户端，支持自动拉取网关最新模型列表',
    tags: ['t3', 'cherry studio', 'chat'],
    import_type: 'none',
    config_filename: 't3-needtoken-provider.json',
    config_template: '{\n  "providerName": "NeedToken",\n  "apiType": "openai-compatible",\n  "baseURL": "{{BASE_URL_V1}}",\n  "modelsEndpoint": "{{MODELS_ENDPOINT}}",\n  "apiKey": "{{API_KEY}}",\n  "availableModels": {{ALL_MODELS_JSON}}\n}\n',
    manual_guide: '1. 在客户端新增 OpenAI 兼容服务商 NeedToken，填入 {{BASE_URL_V1}} 与 API Key。\n2. 点击「获取/刷新模型列表」即可自动同步全部后台最新模型。',
    ai_prompt_template: '请帮我配置当前 AI 对话客户端接入 NeedToken（OpenAI Compatible）：Base URL={{BASE_URL_V1}}，API Key={{API_KEY}}，并从 GET {{MODELS_ENDPOINT}} 自动拉取全部最新可用模型。',
    docs_url: 'https://docs.cherry-ai.com/',
    sort_order: 10,
    enabled: true,
  },
]

const effectiveBaseUrl = computed(() => (gatewayBaseUrl.value || API_BASE_URL).replace(/\/+$/, '').replace(/\/v1$/, ''))
const effectiveBaseUrlV1 = computed(() => `${effectiveBaseUrl.value}/v1`)
const effectiveModelsEndpoint = computed(() => `${effectiveBaseUrlV1.value}/models`)

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
  if (selectedModel.value && availableModels.value.length && !availableModels.value.includes(selectedModel.value)) {
    selectedModel.value = ''
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

const isDeeplinkTool = (tool: QuickToolItem) =>
  tool.import_type === 'deeplink' && Boolean(String(tool.import_template || '').trim())

const resolveModelForTool = (tool: QuickToolItem): string => {
  if (selectedModel.value) return selectedModel.value
  const hint = String(tool.default_model_hint || '').trim()
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
    modelsEndpoint: effectiveModelsEndpoint.value,
    model: resolveModelForTool(tool),
    allModels: availableModels.value,
    tokenName: tok?.name || 'NeedToken',
    group: tok ? formatTokenGroups(tok) : defaultGroup.value,
  }
}

const renderToolText = (tool: QuickToolItem, template?: string, encodeForUrl = false) =>
  renderQuickToolTemplate(template || '', buildRenderVars(tool), encodeForUrl)

let feedbackTimer: ReturnType<typeof setTimeout> | undefined
let deeplinkCleanup: (() => void) | null = null

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

const clearDeeplinkProbe = () => {
  deeplinkCleanup?.()
  deeplinkCleanup = null
  launchingToolId.value = ''
}

const runDeeplinkImport = (tool: QuickToolItem) => {
  if (!tool.import_template) return
  clearDeeplinkProbe()
  deeplinkMissedToolId.value = ''

  const uri = renderToolText(tool, tool.import_template, true)
  if (!uri || /^javascript:/i.test(uri)) return

  launchingToolId.value = tool.id
  let appFocusedAway = false

  const onAppDetected = () => {
    appFocusedAway = true
    clearDeeplinkProbe()
    deeplinkMissedToolId.value = ''
    setFeedback(`imp:${tool.id}`)
  }

  const onVisibilityChange = () => {
    if (document.hidden) onAppDetected()
  }

  window.addEventListener('blur', onAppDetected)
  document.addEventListener('visibilitychange', onVisibilityChange)

  const probeTimer = window.setTimeout(() => {
    clearDeeplinkProbe()
    if (!appFocusedAway && !document.hidden) {
      deeplinkMissedToolId.value = tool.id
    }
  }, 1500)

  deeplinkCleanup = () => {
    window.clearTimeout(probeTimer)
    window.removeEventListener('blur', onAppDetected)
    document.removeEventListener('visibilitychange', onVisibilityChange)
  }

  window.location.href = uri
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
  deeplinkMissedToolId.value = ''
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
  deeplinkMissedToolId.value = ''
  clearDeeplinkProbe()
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
  clearDeeplinkProbe()
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
  clearDeeplinkProbe()
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
</style>
