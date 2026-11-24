<script lang="ts">
interface TurnstileRenderOptions {
  sitekey: string
  theme?: 'auto' | 'light' | 'dark'
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
import ImageCaptcha from './ImageCaptcha.vue'

const props = withDefaults(defineProps<{
  modelValue?: string
  siteKey: string
  disabled?: boolean
  scriptTimeoutMs?: number
  slowHintMs?: number
}>(), {
  modelValue: '',
  disabled: false,
  scriptTimeoutMs: 4500,
  slowHintMs: 6500,
})

const emit = defineEmits<{
  (event: 'update:modelValue', value: string): void
}>()

const { t } = useI18n()

const containerRef = ref<HTMLElement | null>(null)
const fallbackImageRef = ref<InstanceType<typeof ImageCaptcha> | null>(null)
const widgetID = ref<string>('')
const currentToken = ref<string>(props.modelValue || '')
const mode = ref<'loading' | 'turnstile' | 'fallback'>('loading')
const showSlowSwitch = ref(false)
const fallbackPayload = ref<CaptchaPayload>({})

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

const switchToFallback = () => {
  clearSlowTimer()
  showSlowSwitch.value = false
  if (window.turnstile && widgetID.value) {
    try {
      window.turnstile.remove(widgetID.value)
    } catch {
      // ignore
    }
    widgetID.value = ''
  }
  syncToken('')
  mode.value = 'fallback'
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
  showSlowSwitch.value = false
  fallbackPayload.value = {}

  if (!props.siteKey) {
    syncToken('')
    mode.value = 'turnstile'
    return
  }

  mode.value = 'loading'
  try {
    await loadScript(props.scriptTimeoutMs)
  } catch {
    if (epoch !== renderEpoch) return
    switchToFallback()
    return
  }

  if (epoch !== renderEpoch) return
  if (!window.turnstile || !containerRef.value) {
    switchToFallback()
    return
  }

  mode.value = 'turnstile'
  if (widgetID.value) {
    try {
      window.turnstile.remove(widgetID.value)
    } catch {
      // ignore
    }
    widgetID.value = ''
  }

  containerRef.value.innerHTML = ''
  try {
    widgetID.value = window.turnstile.render(containerRef.value, {
      sitekey: props.siteKey,
      theme: 'auto',
      callback: (token: string) => {
        clearSlowTimer()
        showSlowSwitch.value = false
        syncToken(token || '')
      },
      'expired-callback': () => syncToken(''),
      'error-callback': () => switchToFallback(),
    })
  } catch {
    switchToFallback()
    return
  }

  slowTimer = setTimeout(() => {
    if (epoch === renderEpoch && mode.value === 'turnstile' && !currentToken.value) {
      showSlowSwitch.value = true
    }
  }, props.slowHintMs)
}

const retryTurnstile = () => {
  clearTurnstileScriptCache()
  void renderWidget()
}

const reset = () => {
  if (mode.value === 'fallback') {
    fallbackPayload.value = {}
    fallbackImageRef.value?.refresh()
    return
  }
  if (window.turnstile && widgetID.value) {
    try {
      window.turnstile.reset(widgetID.value)
    } catch {
      // ignore
    }
  }
  syncToken('')
}

const getPayload = (): CaptchaPayload => {
  if (mode.value === 'fallback') {
    return {
      captcha_id: fallbackPayload.value.captcha_id || '',
      captcha_code: fallbackPayload.value.captcha_code || '',
    }
  }
  return {
    turnstile_token: currentToken.value || '',
  }
}

const hasPayload = (): boolean => {
  const payload = getPayload()
  return Boolean(payload.turnstile_token || (payload.captcha_id && payload.captcha_code))
}

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
  if (window.turnstile && widgetID.value) {
    try {
      window.turnstile.remove(widgetID.value)
    } catch {
      // ignore
    }
  }
})

defineExpose({
  reset,
  getPayload,
  hasPayload,
  switchToFallback,
  retryTurnstile,
})
</script>

<template>
  <div class="space-y-2">
    <template v-if="mode === 'fallback'">
      <div class="flex items-center justify-between gap-2 text-xs text-muted-foreground">
        <span>{{ t('admin.login.turnstileFallbackNotice') }}</span>
        <button
          type="button"
          class="shrink-0 underline underline-offset-2 hover:text-foreground"
          :disabled="disabled"
          @click="retryTurnstile"
        >
          {{ t('admin.login.turnstileRetry') }}
        </button>
      </div>
      <ImageCaptcha
        ref="fallbackImageRef"
        v-model="fallbackPayload"
        :disabled="disabled"
      />
    </template>
    <template v-else>
      <div v-if="mode === 'loading'" class="text-xs text-muted-foreground">
        {{ t('admin.login.turnstileLoading') }}
      </div>
      <div ref="containerRef" class="min-h-[65px]"></div>
      <div v-if="showSlowSwitch" class="flex items-center justify-between gap-2 text-xs text-muted-foreground">
        <span>{{ t('admin.login.turnstileSlowHint') }}</span>
        <button
          type="button"
          class="shrink-0 underline underline-offset-2 hover:text-foreground"
          :disabled="disabled"
          @click="switchToFallback"
        >
          {{ t('admin.login.turnstileUseImage') }}
        </button>
      </div>
    </template>
  </div>
</template>
