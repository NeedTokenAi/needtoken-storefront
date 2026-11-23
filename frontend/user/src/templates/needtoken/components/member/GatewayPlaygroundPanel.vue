<template>
  <div class="space-y-5">
    <!-- 顶部调试配置栏 -->
    <div class="rounded-2xl border bg-card p-5 shadow-sm">
      <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h2 class="text-lg font-bold text-foreground">{{ t('needtoken.memberConsole.playground.title') }}</h2>
          <p class="mt-1 text-xs text-muted-foreground">{{ t('needtoken.memberConsole.playground.subtitle') }}</p>
        </div>
        <div class="flex items-center gap-2">
          <Button variant="outline" size="sm" :disabled="streaming || messages.length === 0" @click="clearConversation">
            <Trash2 class="mr-1.5 h-3.5 w-3.5" />
            {{ t('needtoken.memberConsole.playground.clearChat') }}
          </Button>
          <Button variant="outline" size="sm" @click="showAdvanced = !showAdvanced">
            <SlidersHorizontal class="mr-1.5 h-3.5 w-3.5" />
            {{ t('needtoken.memberConsole.playground.parameters') }}
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

      <div v-else class="mt-4 grid grid-cols-1 gap-3 border-t pt-4 md:grid-cols-2">
        <div>
          <label class="mb-1 block text-xs font-semibold text-foreground">
            {{ t('needtoken.memberConsole.playground.selectKey') }}
          </label>
          <select
            v-model="selectedTokenKey"
            class="h-9 w-full rounded-xl border bg-background px-3 text-xs font-medium text-foreground outline-none focus:border-primary"
            :disabled="streaming"
          >
            <option v-for="tok in activeTokens" :key="tok.id" :value="tok.key">
              {{ tok.name || `Key #${tok.id}` }} ({{ tok.group || 'default' }} · {{ tok.masked_key }})
            </option>
          </select>
        </div>

        <div>
          <label class="mb-1 block text-xs font-semibold text-foreground">
            {{ t('needtoken.memberConsole.playground.selectModel') }}
          </label>
          <div class="flex gap-2">
            <input
              v-model.trim="selectedModel"
              list="needtoken-playground-models"
              type="text"
              :placeholder="t('needtoken.memberConsole.playground.modelPlaceholder')"
              class="h-9 flex-1 rounded-xl border bg-background px-3 font-mono text-xs text-foreground outline-none focus:border-primary"
              :disabled="streaming"
            />
            <datalist id="needtoken-playground-models">
              <option v-for="m in modelNames" :key="m" :value="m" />
            </datalist>
          </div>
        </div>
      </div>

      <!-- 高级参数（System Prompt + Temperature） -->
      <div v-if="showAdvanced" class="mt-3 grid grid-cols-1 gap-3 rounded-xl border bg-muted/20 p-3.5 md:grid-cols-3">
        <div class="md:col-span-2">
          <label class="mb-1 block text-xs font-semibold text-foreground">System Prompt</label>
          <textarea
            v-model="systemPrompt"
            rows="2"
            :placeholder="t('needtoken.memberConsole.playground.systemPromptPlaceholder')"
            class="w-full rounded-lg border bg-background px-3 py-2 text-xs text-foreground outline-none focus:border-primary"
            :disabled="streaming"
          />
        </div>
        <div>
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
      </div>
    </div>

    <!-- 对话消息流 -->
    <div class="flex min-h-[420px] flex-col rounded-2xl border bg-card shadow-sm">
      <div ref="messagesBox" class="flex-1 space-y-4 overflow-y-auto p-5" style="max-height: 520px;">
        <div v-if="!messages.length" class="flex h-full min-h-[260px] flex-col items-center justify-center text-center">
          <FlaskConical class="h-10 w-10 text-primary/40" />
          <p class="mt-3 text-sm font-semibold text-foreground">{{ t('needtoken.memberConsole.playground.emptyTitle') }}</p>
          <p class="mt-1 max-w-md text-xs text-muted-foreground">{{ t('needtoken.memberConsole.playground.emptyHint') }}</p>
        </div>

        <div
          v-for="(msg, idx) in messages"
          :key="idx"
          class="flex flex-col gap-1.5"
          :class="msg.role === 'user' ? 'items-end' : 'items-start'"
        >
          <span class="text-[11px] font-semibold uppercase tracking-wide text-muted-foreground">
            {{ msg.role === 'user' ? t('needtoken.memberConsole.playground.roleUser') : (msg.model || selectedModel) }}
          </span>
          <div
            class="max-w-[90%] rounded-2xl px-4 py-3 text-sm leading-relaxed sm:max-w-[82%]"
            :class="msg.role === 'user'
              ? 'bg-primary text-primary-foreground'
              : 'border bg-muted/30 text-foreground'"
          >
            <pre class="whitespace-pre-wrap break-words font-sans text-[13.5px]">{{ msg.content || (streaming && idx === messages.length - 1 ? '...' : '') }}</pre>
          </div>
          <div v-if="msg.role === 'assistant' && (msg.ttftMs || msg.totalMs || msg.totalTokens || msg.content)" class="flex flex-wrap items-center gap-2.5 text-[11px] text-muted-foreground">
            <span v-if="msg.ttftMs" class="font-mono">TTFT {{ msg.ttftMs }}ms</span>
            <span v-if="msg.totalMs" class="font-mono">{{ (msg.totalMs / 1000).toFixed(2) }}s</span>
            <span v-if="msg.promptTokens || msg.completionTokens" class="font-mono">
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

      <div v-if="errorMsg" class="mx-5 mb-3 rounded-xl border border-destructive/30 bg-destructive/5 px-3.5 py-2 text-xs text-destructive">
        {{ errorMsg }}
      </div>

      <!-- 底部输入条 -->
      <form class="border-t p-4" @submit.prevent="sendPrompt">
        <div class="flex flex-col gap-2 sm:flex-row sm:items-end">
          <textarea
            v-model="promptInput"
            rows="2"
            :placeholder="t('needtoken.memberConsole.playground.inputPlaceholder')"
            class="min-h-[44px] flex-1 resize-none rounded-xl border bg-background px-3.5 py-2.5 text-sm text-foreground outline-none focus:border-primary"
            :disabled="streaming || !selectedTokenKey"
            @keydown.enter.exact.prevent="sendPrompt"
          />
          <div class="flex shrink-0 items-center gap-2">
            <Button v-if="streaming" type="button" variant="destructive" size="sm" class="h-10 px-4" @click="stopStreaming">
              <Square class="mr-1.5 h-3.5 w-3.5" />
              {{ t('needtoken.memberConsole.playground.stop') }}
            </Button>
            <Button v-else type="submit" size="sm" class="h-10 px-4" :disabled="!promptInput.trim() || !selectedTokenKey || !selectedModel">
              <Send class="mr-1.5 h-3.5 w-3.5" />
              {{ t('needtoken.memberConsole.playground.send') }}
            </Button>
          </div>
        </div>
      </form>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import { Check, Copy, FlaskConical, KeyRound, Send, SlidersHorizontal, Square, Trash2 } from 'lucide-vue-next'
import { Button } from '@/components/ui/button'
import { API_BASE_URL, DEFAULT_MODELS } from '../../config'
import {
  getGatewayModels,
  listGatewayTokens,
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
const modelNames = ref<string[]>([...DEFAULT_MODELS])
const selectedTokenKey = ref('')
const selectedModel = ref(String(route.query.model || '').trim() || DEFAULT_MODELS[0] || 'gpt-4o-mini')
const systemPrompt = ref('')
const temperature = ref(0.7)
const showAdvanced = ref(false)
const promptInput = ref('')
const messages = ref<ChatBubble[]>([])
const streaming = ref(false)
const errorMsg = ref('')
const messagesBox = ref<HTMLElement | null>(null)
const copiedIndex = ref<number | null>(null)

let abortController: AbortController | null = null
let copyTimer: ReturnType<typeof setTimeout> | undefined

const activeTokens = computed(() => tokens.value.filter(tok => tok.status === 1 && tok.key))

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

const clearConversation = () => {
  messages.value = []
  errorMsg.value = ''
}

const stopStreaming = () => {
  abortController?.abort()
  abortController = null
  streaming.value = false
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

  try {
    const resp = await fetch(`${API_BASE_URL}/chat/completions`, {
      method: 'POST',
      headers: {
        Authorization: `Bearer ${selectedTokenKey.value}`,
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        model: selectedModel.value,
        messages: reqMessages,
        temperature: temperature.value,
        stream: true,
        stream_options: { include_usage: true },
      }),
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
    if (tokRes?.items) {
      tokens.value = tokRes.items
      const firstEnabled = tokRes.items.find(tok => tok.status === 1 && tok.key)
      if (firstEnabled) {
        selectedTokenKey.value = firstEnabled.key
      }
    }
    if (modRes?.models?.length) {
      const names = modRes.models.map(m => m.model_name).filter(Boolean)
      if (names.length) {
        modelNames.value = names
        if (!route.query.model && !names.includes(selectedModel.value)) {
          selectedModel.value = names[0]!
        }
      }
    }
  } finally {
    initLoading.value = false
  }
})

onUnmounted(() => {
  stopStreaming()
  if (copyTimer) clearTimeout(copyTimer)
})
</script>
