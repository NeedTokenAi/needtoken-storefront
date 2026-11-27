<template>
  <div class="flex min-h-[calc(100dvh-9.5rem)] flex-col gap-4">
    <!-- 顶部调试工作台配置条 -->
    <div class="rounded-2xl border bg-card p-4 shadow-xs sm:p-5">
      <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div class="min-w-0">
          <div class="flex items-center gap-2">
            <span class="grid h-7 w-7 shrink-0 place-items-center rounded-lg bg-primary/10 text-primary">
              <FlaskConical class="h-4 w-4" />
            </span>
            <h2 class="truncate text-base font-bold text-foreground sm:text-lg">
              {{ t('needtoken.memberConsole.playground.title') }}
            </h2>
          </div>
          <p class="mt-1 text-xs text-muted-foreground">
            {{ t('needtoken.memberConsole.playground.subtitle') }}
          </p>
        </div>

        <div class="flex flex-wrap items-center gap-2">
          <Button
            variant="outline"
            size="sm"
            class="h-8 text-xs"
            :disabled="!selectedTokenKey || !selectedModel"
            @click="copyCurlSnippet"
          >
            <Check v-if="curlCopied" class="mr-1.5 h-3.5 w-3.5 text-primary" />
            <Terminal v-else class="mr-1.5 h-3.5 w-3.5" />
            {{ curlCopied ? t('needtoken.common.copied') : t('needtoken.memberConsole.playground.copyCurl') }}
          </Button>
          <Button
            variant="outline"
            size="sm"
            class="h-8 text-xs"
            :class="showAdvanced ? 'border-primary/40 bg-primary/8 text-primary' : ''"
            @click="showAdvanced = !showAdvanced"
          >
            <SlidersHorizontal class="mr-1.5 h-3.5 w-3.5" />
            {{ t('needtoken.memberConsole.playground.parameters') }}
          </Button>
          <Button
            variant="outline"
            size="sm"
            class="h-8 text-xs"
            :disabled="streaming || messages.length === 0"
            @click="clearConversation"
          >
            <Trash2 class="mr-1.5 h-3.5 w-3.5" />
            {{ t('needtoken.memberConsole.playground.clearChat') }}
          </Button>
        </div>
      </div>

      <!-- 缺少启用中的 API 密钥提示 -->
      <div
        v-if="!initLoading && activeTokens.length === 0"
        class="mt-4 flex flex-col items-start justify-between gap-3 rounded-xl border border-dashed border-primary/35 bg-primary/5 p-4 sm:flex-row sm:items-center"
      >
        <div>
          <p class="text-sm font-semibold text-foreground">{{ t('needtoken.memberConsole.playground.noKeyTitle') }}</p>
          <p class="mt-0.5 text-xs text-muted-foreground">{{ t('needtoken.memberConsole.playground.noKeyHint') }}</p>
        </div>
        <Button size="sm" @click="router.push('/me/keys')">
          <KeyRound class="mr-1.5 h-3.5 w-3.5" />
          {{ t('needtoken.memberConsole.quick.manageKeys') }}
        </Button>
      </div>

      <div v-else class="mt-4 grid grid-cols-1 gap-3 border-t pt-3.5 md:grid-cols-2">
        <div>
          <label class="mb-1.5 block text-[11.5px] font-semibold text-foreground">
            {{ t('needtoken.memberConsole.playground.selectKey') }}
          </label>
          <NeedtokenSelect
            v-model="selectedTokenKey"
            :options="tokenSelectOptions"
            :label="t('needtoken.memberConsole.playground.selectKey')"
            :disabled="streaming || tokenSelectOptions.length === 0"
            align="start"
            class="w-full"
          />
        </div>

        <div>
          <div class="mb-1.5 flex items-center justify-between gap-2">
            <label class="text-[11.5px] font-semibold text-foreground">
              {{ t('needtoken.memberConsole.playground.selectModel') }}
            </label>
            <span v-if="availableModelsForToken.length" class="font-mono text-[11px] text-muted-foreground">
              {{ t('needtoken.memberConsole.playground.supportedModelsCount', { n: availableModelsForToken.length }) }}
            </span>
          </div>
          <NeedtokenModelCombobox
            v-model="selectedModel"
            :models="availableModelsForToken"
            :label="t('needtoken.memberConsole.playground.selectModel')"
            :placeholder="t('needtoken.memberConsole.playground.modelPlaceholder')"
            :search-placeholder="t('needtoken.memberConsole.playground.searchModelPlaceholder')"
            :disabled="streaming"
          />
        </div>
      </div>

      <!-- 高级参数（System Prompt + Temperature + Max Tokens） -->
      <div v-if="showAdvanced" class="mt-3 grid grid-cols-1 gap-3 rounded-xl border bg-muted/20 p-3.5 md:grid-cols-12">
        <div class="md:col-span-6">
          <label class="mb-1 block text-xs font-semibold text-foreground">System Prompt</label>
          <textarea
            v-model="systemPrompt"
            rows="2"
            :placeholder="t('needtoken.memberConsole.playground.systemPromptPlaceholder')"
            class="w-full rounded-lg border bg-background px-3 py-2 text-xs text-foreground outline-none focus:border-primary"
            :disabled="streaming"
          />
        </div>
        <div class="md:col-span-4">
          <label class="mb-1 flex items-center justify-between text-xs font-semibold text-foreground">
            <span>Temperature</span>
            <span class="font-mono text-primary">{{ temperature.toFixed(1) }}</span>
          </label>
          <input
            v-model.number="temperature"
            type="range"
            min="0"
            max="2"
            step="0.1"
            class="mt-2 w-full accent-primary"
            :disabled="streaming"
          />
          <p class="mt-1 text-[11px] text-muted-foreground">{{ t('needtoken.memberConsole.playground.temperatureHint') }}</p>
        </div>
        <div class="md:col-span-2">
          <label class="mb-1 block text-xs font-semibold text-foreground">
            {{ t('needtoken.memberConsole.playground.maxTokens') }}
          </label>
          <input
            v-model.number="maxTokens"
            type="number"
            min="64"
            max="128000"
            step="256"
            placeholder="4096"
            class="h-9 w-full rounded-lg border bg-background px-2.5 font-mono text-xs text-foreground outline-none focus:border-primary"
            :disabled="streaming"
          />
        </div>
      </div>
    </div>

    <!-- 主对话区：自适应高度 + 底部一体式复合输入框 -->
    <div class="flex min-h-[460px] flex-1 flex-col overflow-hidden rounded-2xl border bg-card shadow-xs">
      <div ref="messagesBox" class="flex-1 space-y-4 overflow-y-auto p-4 sm:p-6">
        <!-- 空态：展示欢迎引导 + 3 个可点击快捷试测卡片 -->
        <div v-if="!messages.length" class="mx-auto flex h-full max-w-2xl flex-col items-center justify-center py-6 text-center">
          <div class="grid h-12 w-12 place-items-center rounded-2xl border border-primary/20 bg-primary/8 text-primary">
            <Sparkles class="h-6 w-6" />
          </div>
          <h3 class="mt-3.5 text-base font-bold text-foreground">{{ t('needtoken.memberConsole.playground.emptyTitle') }}</h3>
          <p class="mt-1 max-w-md text-xs leading-relaxed text-muted-foreground">
            {{ t('needtoken.memberConsole.playground.emptyHint') }}
          </p>

          <div class="mt-6 grid w-full grid-cols-1 gap-2.5 text-left sm:grid-cols-3">
            <button
              v-for="starter in starterPrompts"
              :key="starter.id"
              type="button"
              :disabled="!selectedTokenKey || !selectedModel || streaming"
              class="group flex flex-col justify-between rounded-xl border bg-background/80 p-3.5 transition-all hover:border-primary/45 hover:bg-primary/5 disabled:opacity-50"
              @click="runStarterPrompt(starter.body)"
            >
              <div>
                <div class="flex items-center justify-between gap-2">
                  <span class="text-xs font-bold text-foreground group-hover:text-primary">{{ starter.title }}</span>
                  <ArrowUpRight class="h-3.5 w-3.5 shrink-0 text-muted-foreground group-hover:text-primary" />
                </div>
                <p class="mt-1.5 line-clamp-2 text-[11.5px] leading-relaxed text-muted-foreground">{{ starter.body }}</p>
              </div>
            </button>
          </div>
        </div>

        <!-- 消息气泡流 -->
        <div
          v-for="(msg, idx) in messages"
          :key="idx"
          class="flex flex-col gap-1.5"
          :class="msg.role === 'user' ? 'items-end' : 'items-start'"
        >
          <div class="flex items-center gap-2 text-[11px] font-semibold text-muted-foreground">
            <span class="uppercase tracking-wide">
              {{ msg.role === 'user' ? t('needtoken.memberConsole.playground.roleUser') : (msg.model || selectedModel) }}
            </span>
          </div>
          <div
            class="max-w-[92%] rounded-2xl px-4 py-3 text-sm leading-relaxed sm:max-w-[82%]"
            :class="msg.role === 'user'
              ? 'bg-primary text-primary-foreground shadow-2xs'
              : 'border bg-muted/25 text-foreground'"
          >
            <pre class="whitespace-pre-wrap break-words font-sans text-[13.5px] leading-relaxed">{{ msg.content || (streaming && idx === messages.length - 1 ? '...' : '') }}</pre>
          </div>
          <div v-if="msg.role === 'assistant' && (msg.ttftMs || msg.totalMs || msg.totalTokens || msg.content)" class="flex flex-wrap items-center gap-2.5 text-[11px] text-muted-foreground">
            <span v-if="msg.ttftMs" class="rounded border bg-background px-1.5 py-0.5 font-mono">TTFT {{ msg.ttftMs }}ms</span>
            <span v-if="msg.totalMs" class="rounded border bg-background px-1.5 py-0.5 font-mono">{{ (msg.totalMs / 1000).toFixed(2) }}s</span>
            <span v-if="msg.promptTokens || msg.completionTokens" class="rounded border bg-background px-1.5 py-0.5 font-mono">
              Tokens: {{ msg.promptTokens ?? 0 }} / {{ msg.completionTokens ?? 0 }}
            </span>
            <button
              v-if="msg.content"
              type="button"
              class="inline-flex items-center gap-1 font-medium text-primary hover:underline"
              @click="copyReply(msg.content, idx)"
            >
              <Check v-if="copiedIndex === idx" class="h-3 w-3" />
              <Copy v-else class="h-3 w-3" />
              {{ copiedIndex === idx ? t('needtoken.common.copied') : t('needtoken.common.copy') }}
            </button>
          </div>
        </div>
      </div>

      <div v-if="errorMsg" class="mx-4 mb-2 rounded-xl border border-destructive/30 bg-destructive/5 px-3.5 py-2 text-xs text-destructive sm:mx-6">
        {{ errorMsg }}
      </div>

      <!-- 一体化圆角复合输入区 -->
      <form class="border-t bg-muted/15 p-3 sm:p-4" @submit.prevent="sendPrompt">
        <div class="rounded-2xl border bg-background p-2.5 shadow-2xs transition-colors focus-within:border-primary">
          <textarea
            v-model="promptInput"
            rows="2"
            :placeholder="t('needtoken.memberConsole.playground.inputPlaceholder')"
            class="max-h-40 min-h-[52px] w-full resize-none bg-transparent px-2 py-1 text-sm text-foreground outline-none placeholder:text-muted-foreground"
            :disabled="streaming || !selectedTokenKey"
            @keydown.enter.exact="handlePromptKeydown"
          />
          <div class="mt-2 flex flex-wrap items-center justify-between gap-2 border-t border-border/60 pt-2">
            <div class="flex flex-wrap items-center gap-2 px-1 text-[11px] text-muted-foreground">
              <span v-if="selectedModel" class="rounded-md bg-muted px-1.5 py-0.5 font-mono text-foreground">{{ selectedModel }}</span>
              <span class="font-mono">T={{ temperature.toFixed(1) }}</span>
              <span class="hidden sm:inline">· {{ t('needtoken.memberConsole.playground.shortcutHint') }}</span>
            </div>
            <div class="flex items-center gap-2">
              <Button v-if="streaming" type="button" variant="destructive" size="sm" class="h-8 rounded-xl px-3.5 text-xs" @click="stopStreaming">
                <Square class="mr-1.5 h-3 w-3" />
                {{ t('needtoken.memberConsole.playground.stop') }}
              </Button>
              <Button
                v-else
                type="submit"
                size="sm"
                class="h-8 rounded-xl px-4 text-xs font-semibold"
                :disabled="!promptInput.trim() || !selectedTokenKey || !selectedModel"
              >
                <Send class="mr-1.5 h-3.5 w-3.5" />
                {{ t('needtoken.memberConsole.playground.send') }}
              </Button>
            </div>
          </div>
        </div>
      </form>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import {
  ArrowUpRight,
  Check,
  Copy,
  FlaskConical,
  KeyRound,
  Send,
  SlidersHorizontal,
  Sparkles,
  Square,
  Terminal,
  Trash2,
} from 'lucide-vue-next'
import { Button } from '@/components/ui/button'
import NeedtokenSelect from '../NeedtokenSelect.vue'
import NeedtokenModelCombobox from './NeedtokenModelCombobox.vue'
import { API_BASE_URL, DEFAULT_MODELS } from '../../config'
import {
  formatTokenGroupsSummary,
  resolveTokenAvailableModels,
} from '../../lib/gatewayConsoleHelpers'
import {
  getGatewayModels,
  listGatewayTokens,
  type GatewayGroupMetaItem,
  type GatewayModelPricingItem,
  type GatewayTokenItem,
} from '../../lib/gatewayConsoleApi'

interface ChatBubble {
  role: 'user' | 'assistant'
  content: string
  model?: string
  ttftMs?: number
  totalMs?: number
  promptTokens?: number
  completionTokens?: number
  totalTokens?: number
}

const { t } = useI18n()
const route = useRoute()
const router = useRouter()

const initLoading = ref(false)
const tokens = ref<GatewayTokenItem[]>([])
const pricingModels = ref<GatewayModelPricingItem[]>([])
const usableGroups = ref<Record<string, string>>({})
const groupRatio = ref<Record<string, number>>({})
const groupMeta = ref<Record<string, GatewayGroupMetaItem>>({})
const defaultGroup = ref('default')
const userAutoGroups = ref<string[]>([])

const selectedTokenKey = ref('')
const selectedModel = ref(String(route.query.model || '').trim() || DEFAULT_MODELS[0] || 'gpt-4o-mini')
const systemPrompt = ref('')
const temperature = ref(0.7)
const maxTokens = ref<number | undefined>(undefined)
const showAdvanced = ref(false)
const promptInput = ref('')
const messages = ref<ChatBubble[]>([])
const streaming = ref(false)
const errorMsg = ref('')
const messagesBox = ref<HTMLElement | null>(null)
const copiedIndex = ref<number | null>(null)
const curlCopied = ref(false)

let abortController: AbortController | null = null
let copyTimer: ReturnType<typeof setTimeout> | undefined
let curlTimer: ReturnType<typeof setTimeout> | undefined

const starterPrompts = computed(() => [
  {
    id: 'ping',
    title: t('needtoken.memberConsole.playground.quickPrompt1Title'),
    body: t('needtoken.memberConsole.playground.quickPrompt1Body'),
  },
  {
    id: 'code',
    title: t('needtoken.memberConsole.playground.quickPrompt2Title'),
    body: t('needtoken.memberConsole.playground.quickPrompt2Body'),
  },
  {
    id: 'reason',
    title: t('needtoken.memberConsole.playground.quickPrompt3Title'),
    body: t('needtoken.memberConsole.playground.quickPrompt3Body'),
  },
])

const activeTokens = computed(() => tokens.value.filter(tok => tok.status === 1 && tok.key))

const selectedToken = computed(() =>
  activeTokens.value.find(tok => tok.key === selectedTokenKey.value) || null,
)

const tokenSelectOptions = computed(() =>
  activeTokens.value.map((tok) => {
    const grpSummary = formatTokenGroupsSummary(
      tok,
      defaultGroup.value,
      groupMeta.value,
      usableGroups.value,
      groupRatio.value,
    )
    return {
      value: tok.key,
      label: `${tok.name || `Key #${tok.id}`} (${grpSummary} · ${tok.masked_key})`,
    }
  }),
)

const availableModelsForToken = computed(() =>
  resolveTokenAvailableModels({
    token: selectedToken.value,
    defaultGroup: defaultGroup.value,
    userAutoGroups: userAutoGroups.value,
    groupMeta: groupMeta.value,
    pricingModels: pricingModels.value,
    fallbackModels: [...DEFAULT_MODELS],
  }),
)

watch([selectedTokenKey, availableModelsForToken], () => {
  const list = availableModelsForToken.value
  if (!list.length) return
  if (!selectedModel.value || !list.includes(selectedModel.value)) {
    const queryModel = String(route.query.model || '').trim()
    if (queryModel && list.includes(queryModel)) {
      selectedModel.value = queryModel
    } else {
      selectedModel.value = list[0]!
    }
  }
})

const scrollToBottom = async () => {
  await nextTick()
  if (messagesBox.value) {
    messagesBox.value.scrollTop = messagesBox.value.scrollHeight
  }
}

const copyReply = async (content: string, idx: number) => {
  if (copyTimer) clearTimeout(copyTimer)
  try {
    await navigator.clipboard.writeText(content)
    copiedIndex.value = idx
    copyTimer = setTimeout(() => { copiedIndex.value = null }, 1500)
  } catch {
    copiedIndex.value = null
  }
}

const copyCurlSnippet = async () => {
  if (!selectedTokenKey.value || !selectedModel.value) return
  if (curlTimer) clearTimeout(curlTimer)
  const payload = {
    model: selectedModel.value,
    messages: [{ role: 'user', content: promptInput.value.trim() || 'Hello' }],
    stream: true,
  }
  const snippet = [
    `curl -N ${API_BASE_URL}/chat/completions \\`,
    `  -H "Content-Type: application/json" \\`,
    `  -H "Authorization: Bearer ${selectedTokenKey.value}" \\`,
    `  -d '${JSON.stringify(payload)}'`,
  ].join('\n')
  try {
    await navigator.clipboard.writeText(snippet)
    curlCopied.value = true
    curlTimer = setTimeout(() => { curlCopied.value = false }, 1800)
  } catch {
    curlCopied.value = false
  }
}

const clearConversation = () => {
  messages.value = []
  errorMsg.value = ''
}

const stopStreaming = () => {
  abortController?.abort()
  abortController = null
  streaming.value = false
}

const handlePromptKeydown = (event: KeyboardEvent) => {
  if (event.isComposing || event.keyCode === 229) return
  event.preventDefault()
  void sendPrompt()
}

const runStarterPrompt = (body: string) => {
  if (streaming.value || !selectedTokenKey.value || !selectedModel.value) return
  promptInput.value = body
  void sendPrompt()
}

const sendPrompt = async () => {
  const text = promptInput.value.trim()
  if (!text || streaming.value || !selectedTokenKey.value || !selectedModel.value) return
  errorMsg.value = ''

  const history = messages.value.map(m => ({ role: m.role, content: m.content }))
  const reqMessages: Array<{ role: string; content: string }> = []
  if (systemPrompt.value.trim()) {
    reqMessages.push({ role: 'system', content: systemPrompt.value.trim() })
  }
  reqMessages.push(...history, { role: 'user', content: text })

  messages.value.push({ role: 'user', content: text })
  const assistantMsg: ChatBubble = { role: 'assistant', content: '', model: selectedModel.value }
  messages.value.push(assistantMsg)
  promptInput.value = ''
  await scrollToBottom()

  streaming.value = true
  abortController = new AbortController()
  const startedAt = performance.now()
  let firstChunkSeen = false

  const reqBody: Record<string, unknown> = {
    model: selectedModel.value,
    messages: reqMessages,
    temperature: temperature.value,
    stream: true,
    stream_options: { include_usage: true },
  }
  if (typeof maxTokens.value === 'number' && Number.isFinite(maxTokens.value) && maxTokens.value > 0) {
    reqBody.max_tokens = Math.round(maxTokens.value)
  }

  try {
    const resp = await fetch(`${API_BASE_URL}/chat/completions`, {
      method: 'POST',
      headers: {
        Authorization: `Bearer ${selectedTokenKey.value}`,
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(reqBody),
      signal: abortController.signal,
    })

    if (!resp.ok) {
      const errText = await resp.text().catch(() => '')
      let detail = errText
      try {
        const parsed = JSON.parse(errText)
        detail = parsed?.error?.message || parsed?.message || errText
      } catch {
        // keep raw text
      }
      throw new Error(detail || `HTTP ${resp.status}`)
    }

    const reader = resp.body?.getReader()
    if (!reader) {
      throw new Error('ReadableStream unavailable')
    }
    const decoder = new TextDecoder()
    let buffer = ''

    while (true) {
      const { value, done } = await reader.read()
      if (done) break
      buffer += decoder.decode(value, { stream: true })
      const lines = buffer.split('\n')
      buffer = lines.pop() ?? ''
      for (const rawLine of lines) {
        const line = rawLine.trim()
        if (!line.startsWith('data:')) continue
        const payload = line.slice(5).trim()
        if (!payload || payload === '[DONE]') continue
        try {
          const chunk = JSON.parse(payload)
          const delta = chunk?.choices?.[0]?.delta?.content
          if (typeof delta === 'string' && delta.length > 0) {
            if (!firstChunkSeen) {
              firstChunkSeen = true
              assistantMsg.ttftMs = Math.round(performance.now() - startedAt)
            }
            assistantMsg.content += delta
            await scrollToBottom()
          }
          if (chunk?.usage) {
            assistantMsg.promptTokens = chunk.usage.prompt_tokens
            assistantMsg.completionTokens = chunk.usage.completion_tokens
            assistantMsg.totalTokens = chunk.usage.total_tokens
          }
        } catch {
          // ignore malformed SSE chunk
        }
      }
    }
    assistantMsg.totalMs = Math.round(performance.now() - startedAt)
  } catch (err: any) {
    if (err?.name !== 'AbortError') {
      errorMsg.value = err?.message || t('needtoken.memberConsole.actionFailed')
      if (!assistantMsg.content) {
        messages.value.pop()
      }
    }
  } finally {
    streaming.value = false
    abortController = null
    await scrollToBottom()
  }
}

onMounted(async () => {
  initLoading.value = true
  try {
    const [tokRes, modRes] = await Promise.all([
      listGatewayTokens().catch(() => null),
      getGatewayModels().catch(() => null),
    ])
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
      const firstEnabled = tokens.value.find(tok => tok.status === 1 && tok.key)
      if (firstEnabled) {
        selectedTokenKey.value = firstEnabled.key
      }
    }
  } finally {
    initLoading.value = false
  }
})

onUnmounted(() => {
  stopStreaming()
  if (copyTimer) clearTimeout(copyTimer)
  if (curlTimer) clearTimeout(curlTimer)
})
</script>
