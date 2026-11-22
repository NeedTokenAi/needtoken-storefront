<template>
  <div class="nt-link-cb mx-auto flex min-h-[70vh] w-full max-w-[540px] items-center px-4 py-16">
    <section class="nt-link-cb__card w-full" :aria-busy="state === 'loading'">
      <div class="nt-link-cb__icon" :class="`is-${state}`" aria-hidden="true">
        <LoaderCircle v-if="state === 'loading'" :size="28" class="nt-link-cb__spin" />
        <CheckCircle2 v-else-if="state === 'success'" :size="28" />
        <AlertCircle v-else :size="28" />
      </div>

      <template v-if="state === 'loading'">
        <h1>{{ t('needtoken.apiAccountLink.callbackLoadingTitle') }}</h1>
        <p>{{ t('needtoken.apiAccountLink.callbackLoadingDesc') }}</p>
      </template>

      <template v-else-if="state === 'success'">
        <h1>{{ t('needtoken.apiAccountLink.callbackSuccessTitle') }}</h1>
        <p>
          {{
            t('needtoken.apiAccountLink.callbackSuccessDesc', {
              id: result?.native_user_id ? `#${result.native_user_id}` : '',
            })
          }}
        </p>
        <p v-if="result && result.awakened_orders > 0" class="nt-link-cb__highlight" role="status">
          {{ t('needtoken.apiAccountLink.callbackAwakened', { count: result.awakened_orders }) }}
        </p>
        <div class="nt-link-cb__actions">
          <button type="button" class="nt-link-cb__primary" @click="goNext">
            <span>{{ t('needtoken.apiAccountLink.callbackContinue') }}</span>
            <ArrowRight :size="15" aria-hidden="true" />
          </button>
          <RouterLink to="/me/keys" class="nt-link-cb__secondary">
            <span>{{ t('needtoken.memberConsole.nav.keys') }}</span>
            <ArrowUpRight :size="14" aria-hidden="true" />
          </RouterLink>
        </div>
      </template>

      <template v-else>
        <h1>{{ t('needtoken.apiAccountLink.callbackFailedTitle') }}</h1>
        <p role="alert">{{ failureReason }}</p>
        <div class="nt-link-cb__actions">
          <button type="button" class="nt-link-cb__primary" :disabled="retrying" @click="restart">
            <LoaderCircle v-if="retrying" :size="15" class="nt-link-cb__spin" aria-hidden="true" />
            <span>{{ t('needtoken.apiAccountLink.retryBind') }}</span>
          </button>
          <RouterLink to="/me/orders" class="nt-link-cb__secondary">
            <span>{{ t('needtoken.apiCatalog.myOrders') }}</span>
          </RouterLink>
        </div>
      </template>
    </section>
  </div>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { AlertCircle, ArrowRight, ArrowUpRight, CheckCircle2, LoaderCircle } from 'lucide-vue-next'
import {
  completeApiAccountLinkRequest,
  startApiAccountLinkRequest,
  type ApiAccountLinkCompleteResult,
} from './composables/useApiAccountLink'
import { parseAccountLinkFragment, resolveReturnRoute } from './lib/apiAccountLinkPolicy'

const { t } = useI18n()
const router = useRouter()

const state = ref<'loading' | 'success' | 'error'>('loading')
const result = ref<ApiAccountLinkCompleteResult | null>(null)
const failureReason = ref('')
const retrying = ref(false)
let redirectTimer: ReturnType<typeof setTimeout> | undefined

const goNext = () => {
  if (redirectTimer) clearTimeout(redirectTimer)
  const target = resolveReturnRoute(result.value?.next_path)
  void router.replace(target)
}

const restart = async () => {
  if (retrying.value) return
  retrying.value = true
  try {
    const ch = await startApiAccountLinkRequest('/plans')
    window.location.assign(ch.authorize_url)
  } catch (err: any) {
    failureReason.value = err?.message || t('needtoken.apiAccountLink.startFailed')
    retrying.value = false
  }
}

onMounted(async () => {
  const payload = parseAccountLinkFragment(window.location.hash)
  // Scrub approval token from URL hash immediately so it never stays in browser history.
  if (window.location.hash) {
    window.history.replaceState(null, '', window.location.pathname)
  }
  if (!payload) {
    state.value = 'error'
    failureReason.value = t('needtoken.apiAccountLink.callbackMissingParams')
    return
  }
  try {
    const completed = await completeApiAccountLinkRequest(payload.challengeId, payload.approval)
    result.value = completed
    state.value = 'success'
    redirectTimer = setTimeout(goNext, 2600)
  } catch (err: any) {
    state.value = 'error'
    failureReason.value = err?.message || t('needtoken.apiAccountLink.callbackFailedDesc')
  }
})

onBeforeUnmount(() => {
  if (redirectTimer) clearTimeout(redirectTimer)
})
</script>

<style scoped>
.nt-link-cb__card {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 14px;
  padding: 36px 28px;
  border: 1px solid var(--ui-border, hsl(var(--border)));
  border-radius: 20px;
  background: var(--ui-bg-elevated, hsl(var(--card)));
  text-align: center;
  box-shadow: var(--nt-card-shadow, none);
}
.nt-link-cb__icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 56px;
  height: 56px;
  border-radius: 16px;
  background: var(--nt-accent-subtle, hsl(var(--muted)));
  color: var(--ui-accent, hsl(var(--primary)));
}
.nt-link-cb__icon.is-success {
  background: rgba(16, 185, 129, 0.12);
  color: #059669;
}
.nt-link-cb__icon.is-error {
  background: rgba(239, 68, 68, 0.12);
  color: #dc2626;
}
.nt-link-cb__card h1 {
  font-size: 20px;
  font-weight: 700;
  color: var(--ui-text-primary, hsl(var(--foreground)));
}
.nt-link-cb__card p {
  max-width: 44ch;
  font-size: 13px;
  line-height: 1.7;
  color: var(--ui-text-secondary, hsl(var(--muted-foreground)));
}
.nt-link-cb__highlight {
  padding: 10px 14px;
  border-radius: 10px;
  border: 1px solid rgba(16, 185, 129, 0.35);
  background: rgba(16, 185, 129, 0.08);
  color: #059669 !important;
  font-weight: 600;
}
.nt-link-cb__actions {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: 10px;
  margin-top: 8px;
}
.nt-link-cb__primary {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-height: 40px;
  padding: 0 18px;
  border: 0;
  border-radius: 10px;
  background: var(--ui-accent, hsl(var(--primary)));
  color: #fff;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
}
.nt-link-cb__secondary {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  min-height: 40px;
  padding: 0 16px;
  border: 1px solid var(--ui-border, hsl(var(--border)));
  border-radius: 10px;
  color: var(--ui-text-primary, hsl(var(--foreground)));
  font-size: 13px;
  font-weight: 500;
}
.nt-link-cb__spin {
  animation: nt-link-cb-spin 1s linear infinite;
}
@keyframes nt-link-cb-spin {
  to { transform: rotate(360deg); }
}
</style>
