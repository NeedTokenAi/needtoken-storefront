<script lang="ts">
interface TurnstileRenderOptions {
  sitekey: string
  theme?: 'auto' | 'light' | 'dark'
  size?: 'normal' | 'compact' | 'flexible'
  callback?: (token: string) => void
  'expired-callback'?: () => void
  'error-callback'?: () => void
}

interface TurnstileAPI {
  render: (container: HTMLElement, options: TurnstileRenderOptions) => string
  reset: (widgetID?: string) => void
  remove: (widgetID: string) => void
}

declare global {
  interface Window {
    turnstile?: TurnstileAPI
  }
}

let turnstileScriptPromise: Promise<void> | null = null

const clearTurnstileScriptCache = () => {
  turnstileScriptPromise = null
  const existing = document.querySelector<HTMLScriptElement>('script[data-turnstile="1"]')
  if (existing && !window.turnstile) {
    existing.remove()
  }
}
</script>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { CaptchaPayload } from '@/api/admin'

const props = withDefaults(defineProps<{
  modelValue?: string
  siteKey: string
  disabled?: boolean
  scriptTimeoutMs?: number
  slowHintMs?: number
}>(), {
  modelValue: '',
  disabled: false,
  scriptTimeoutMs: 12000,
  slowHintMs: 5000,
})

const emit = defineEmits<{
  (event: 'update:modelValue', value: string): void
}>()

const { t } = useI18n()

const containerRef = ref<HTMLElement | null>(null)
const widgetID = ref<string>('')
const currentToken = ref<string>(props.modelValue || '')
const mode = ref<'loading' | 'ready' | 'error'>('loading')
const errorType = ref<'load' | 'expired' | ''>('')
const showSlowReload = ref(false)

let slowTimer: ReturnType<typeof setTimeout> | null = null
let renderEpoch = 0

const clearSlowTimer = () => {
  if (slowTimer) {
    clearTimeout(slowTimer)
    slowTimer = null
  }
}

const syncToken = (token: string) => {
  currentToken.value = token
  emit('update:modelValue', token)
}

const removeWidget = () => {
  if (window.turnstile && widgetID.value) {
    try {
      window.turnstile.remove(widgetID.value)
    } catch {
      // ignore
    }
    widgetID.value = ''
  }
}

const scheduleSlowHint = (epoch: number) => {
  clearSlowTimer()
  showSlowReload.value = false
  slowTimer = setTimeout(() => {
    if (epoch === renderEpoch && !currentToken.value && mode.value !== 'error') {
      showSlowReload.value = true
    }
  }, props.slowHintMs)
}

const enterErrorMode = (type: 'load' | 'expired') => {
  clearSlowTimer()
  showSlowReload.value = false
  removeWidget()
  syncToken('')
  errorType.value = type
  mode.value = 'error'
}

const loadScript = async (timeoutMs: number) => {
  if (window.turnstile) return
  if (!turnstileScriptPromise) {
    turnstileScriptPromise = new Promise<void>((resolve, reject) => {
      const existing = document.querySelector<HTMLScriptElement>('script[data-turnstile="1"]')
      const script = existing ?? document.createElement('script')
      let settled = false
      const timer = window.setTimeout(() => {
        if (settled) return
        settled = true
        clearTurnstileScriptCache()
        reject(new Error('turnstile script load timeout'))
      }, timeoutMs)

      const finish = (err?: Error) => {
        if (settled) return
        settled = true
        window.clearTimeout(timer)
        if (err) {
          clearTurnstileScriptCache()
          reject(err)
          return
        }
        resolve()
      }

      script.addEventListener('load', () => finish(), { once: true })
      script.addEventListener('error', () => finish(new Error('failed to load turnstile script')), { once: true })

      if (!existing) {
        script.src = 'https://challenges.cloudflare.com/turnstile/v0/api.js?render=explicit'
        script.async = true
        script.defer = true
        script.dataset.turnstile = '1'
        document.head.appendChild(script)
      }
    })
  }
  await turnstileScriptPromise
}

const renderWidget = async () => {
  const epoch = ++renderEpoch
  clearSlowTimer()
  showSlowReload.value = false
  errorType.value = ''
  syncToken('')

  if (!props.siteKey) {
    mode.value = 'error'
    errorType.value = 'load'
    return
  }

  mode.value = 'loading'
  scheduleSlowHint(epoch)
  try {
    await loadScript(props.scriptTimeoutMs)
  } catch {
    if (epoch !== renderEpoch) return
    enterErrorMode('load')
    return
  }

  if (epoch !== renderEpoch) return
  if (!window.turnstile || !containerRef.value) {
    enterErrorMode('load')
    return
  }

  mode.value = 'ready'
  removeWidget()

  containerRef.value.innerHTML = ''
  const theme: 'light' | 'dark' = document.documentElement.classList.contains('dark') ? 'dark' : 'light'
  try {
    widgetID.value = window.turnstile.render(containerRef.value, {
      sitekey: props.siteKey,
      theme,
      size: 'flexible',
      callback: (token: string) => {
        clearSlowTimer()
        showSlowReload.value = false
        errorType.value = ''
        syncToken(token || '')
      },
      'expired-callback': () => {
        syncToken('')
        if (window.turnstile && widgetID.value) {
          try {
            window.turnstile.reset(widgetID.value)
            scheduleSlowHint(epoch)
            return
          } catch {
            // fall through
          }
        }
        enterErrorMode('expired')
      },
      'error-callback': () => {
        if (epoch !== renderEpoch) return
        enterErrorMode('load')
      },
    })
  } catch {
    enterErrorMode('load')
    return
  }
}

const retryTurnstile = () => {
  clearSlowTimer()
  removeWidget()
  clearTurnstileScriptCache()
  void renderWidget()
}

const reset = () => {
  if (mode.value === 'error') {
    retryTurnstile()
    return
  }
  if (window.turnstile && widgetID.value) {
    try {
      window.turnstile.reset(widgetID.value)
      scheduleSlowHint(renderEpoch)
    } catch {
      enterErrorMode('load')
      return
    }
  }
  syncToken('')
}

const getPayload = (): CaptchaPayload => ({
  turnstile_token: currentToken.value || '',
})

const hasPayload = (): boolean => Boolean(currentToken.value)

watch(
  () => props.siteKey,
  () => {
    void renderWidget()
  },
)

onMounted(() => {
  void renderWidget()
})

onBeforeUnmount(() => {
  renderEpoch++
  clearSlowTimer()
  removeWidget()
})

defineExpose({
  reset,
  getPayload,
  hasPayload,
  retryTurnstile,
})
</script>

<template>
  <div class="w-full space-y-1.5">
    <div
      v-if="mode === 'error'"
      class="flex min-h-[65px] items-center justify-between gap-3 rounded-lg border border-border bg-muted/40 px-3.5 py-2.5 text-xs text-muted-foreground"
      role="alert"
    >
      <span>{{ t(errorType === 'expired' ? 'admin.login.turnstileExpired' : 'admin.login.turnstileLoadFailed') }}</span>
      <button
        type="button"
        class="inline-flex shrink-0 items-center gap-1 rounded-md border border-border bg-background px-2.5 py-1.5 text-xs font-medium text-foreground transition-colors hover:bg-secondary disabled:cursor-not-allowed disabled:opacity-50"
        :disabled="disabled"
        @click="retryTurnstile"
      >
        <svg class="h-3.5 w-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
          <path d="M21 12a9 9 0 1 1-9-9c2.52 0 4.93 1 6.74 2.74L21 8" />
          <path d="M21 3v5h-5" />
        </svg>
        <span>{{ t('admin.login.turnstileRetry') }}</span>
      </button>
    </div>
    <template v-else>
      <div class="relative w-full min-h-[65px]">
        <div
          v-if="mode === 'loading'"
          class="pointer-events-none absolute inset-0 flex items-center justify-center rounded-lg border border-border/70 bg-muted/30 px-3 text-xs text-muted-foreground"
          role="status"
        >
          {{ t('admin.login.turnstileLoading') }}
        </div>
        <div ref="containerRef" class="turnstile-widget-host w-full min-h-[65px]"></div>
      </div>
      <div v-if="showSlowReload && !currentToken" class="flex items-center justify-between gap-2 text-xs text-muted-foreground">
        <span>{{ t('admin.login.turnstileSlowHint') }}</span>
        <button
          type="button"
          class="inline-flex shrink-0 items-center gap-1 font-medium text-foreground underline underline-offset-2 hover:opacity-80 disabled:cursor-not-allowed disabled:opacity-50"
          :disabled="disabled"
          @click="retryTurnstile"
        >
          {{ t('admin.login.turnstileRetry') }}
        </button>
      </div>
    </template>
  </div>
</template>

<style scoped>
.turnstile-widget-host :deep(> div),
.turnstile-widget-host :deep(iframe) {
  width: 100% !important;
  max-width: 100%;
  display: block;
}
</style>
