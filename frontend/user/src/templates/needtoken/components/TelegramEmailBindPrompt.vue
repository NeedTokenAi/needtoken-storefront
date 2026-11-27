<template>
  <Teleport to="body">
    <div
      v-if="visible"
      class="nt-tg-prompt-backdrop"
      role="dialog"
      aria-modal="true"
      aria-labelledby="nt-tg-prompt-title"
      @click.self="dismissPrompt"
    >
      <div class="nt-tg-prompt-card">
        <button
          type="button"
          class="nt-tg-prompt-close"
          :aria-label="t('common.close')"
          :disabled="busy"
          @click="dismissPrompt"
        >
          <X :size="16" aria-hidden="true" />
        </button>

        <template v-if="step === 'choice'">
          <div class="nt-tg-prompt-icon" aria-hidden="true">
            <Mail :size="20" />
          </div>
          <h2 id="nt-tg-prompt-title">{{ t('needtoken.tgEmailPrompt.title') }}</h2>
          <p class="nt-tg-prompt-desc">{{ t('needtoken.tgEmailPrompt.desc') }}</p>

          <p v-if="errorMsg" role="alert" class="nt-tg-prompt-error">{{ errorMsg }}</p>

          <div class="nt-tg-prompt-actions">
            <button
              type="button"
              class="nt-tg-prompt-primary"
              :disabled="busy"
              @click="openBindForm"
            >
              {{ t('needtoken.tgEmailPrompt.bindAction') }}
            </button>
            <button
              type="button"
              class="nt-tg-prompt-secondary"
              :disabled="busy"
              @click="skipAndProvision"
            >
              {{ skipping ? t('needtoken.tgEmailPrompt.provisioning') : t('needtoken.tgEmailPrompt.skipAction') }}
            </button>
          </div>
        </template>

        <template v-else>
          <h2 id="nt-tg-prompt-title" class="nt-tg-prompt-form-title">{{ t('needtoken.tgEmailPrompt.bindTitle') }}</h2>
          <p v-if="errorMsg" role="alert" class="nt-tg-prompt-error">{{ errorMsg }}</p>

          <form class="nt-tg-prompt-form" @submit.prevent="submitBindEmail">
            <label class="nt-tg-prompt-field">
              <span>{{ t('auth.login.emailLabel') }}</span>
              <input
                v-model="emailInput"
                type="email"
                autocomplete="email"
                autocorrect="off"
                autocapitalize="none"
                spellcheck="false"
                maxlength="254"
                required
                :placeholder="t('auth.login.emailPlaceholder')"
                :disabled="busy"
              />
            </label>

            <label class="nt-tg-prompt-field">
              <span>{{ t('auth.register.codeLabel') }}</span>
              <div class="nt-tg-prompt-code-row">
                <input
                  v-model="codeInput"
                  inputmode="numeric"
                  pattern="[0-9]{4,10}"
                  autocomplete="one-time-code"
                  autocorrect="off"
                  autocapitalize="none"
                  spellcheck="false"
                  maxlength="10"
                  required
                  :placeholder="t('auth.register.codePlaceholder')"
                  :disabled="busy"
                />
                <button
                  type="button"
                  class="nt-tg-prompt-send"
                  :disabled="busy || sendCooldown > 0 || !emailInput.trim()"
                  @click="sendCode"
                >
                  {{ sendCooldown > 0 ? t('auth.common.countdown', { seconds: sendCooldown }) : t('auth.common.sendCode') }}
                </button>
              </div>
            </label>

            <div class="nt-tg-prompt-actions">
              <button type="submit" class="nt-tg-prompt-primary" :disabled="busy">
                {{ binding ? t('needtoken.tgEmailPrompt.binding') : t('needtoken.tgEmailPrompt.confirmBind') }}
              </button>
              <button type="button" class="nt-tg-prompt-text" :disabled="busy" @click="backToChoice">
                {{ t('common.back') }}
              </button>
            </div>
          </form>
        </template>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { Mail, X } from 'lucide-vue-next'
import { useUserAuthStore } from '../../../stores/userAuth'
import { useUserProfileStore } from '../../../stores/userProfile'
import { ensureApiAccountLink, fetchApiAccountLinkStatus } from '../composables/useApiAccountLink'

const { t } = useI18n()
const authStore = useUserAuthStore()
const profileStore = useUserProfileStore()

const visible = ref(false)
const step = ref<'choice' | 'bind'>('choice')
const emailInput = ref('')
const codeInput = ref('')
const errorMsg = ref('')
const skipping = ref(false)
const binding = ref(false)
const sending = ref(false)
const sendCooldown = ref(0)
let cooldownTimer: ReturnType<typeof setInterval> | null = null
let checking = false

const busy = computed(() => skipping.value || binding.value || sending.value)
const userID = computed(() => Number(authStore.user?.id || 0))
const userEmail = computed(() => String(authStore.user?.email || '').trim().toLowerCase())
const needsEmailChoice = computed(() => {
  if (!authStore.isAuthenticated) return false
  if (userEmail.value.endsWith('@login.local')) return true
  return authStore.user?.email_change_mode === 'bind_only'
})

const dismissStorageKey = computed(() =>
  userID.value > 0 ? `nt_tg_email_prompt_dismissed_${userID.value}` : 'nt_tg_email_prompt_dismissed',
)

const isDismissed = (): boolean => {
  try {
    return sessionStorage.getItem(dismissStorageKey.value) === '1'
  } catch {
    return false
  }
}

const markDismissed = () => {
  try {
    sessionStorage.setItem(dismissStorageKey.value, '1')
  } catch {
    // ignore storage failures
  }
}

const stopCooldown = () => {
  if (cooldownTimer) {
    clearInterval(cooldownTimer)
    cooldownTimer = null
  }
}

const startCooldown = (seconds = 60) => {
  stopCooldown()
  sendCooldown.value = seconds
  cooldownTimer = setInterval(() => {
    if (sendCooldown.value <= 1) {
      sendCooldown.value = 0
      stopCooldown()
      return
    }
    sendCooldown.value -= 1
  }, 1000)
}

const checkEligibility = async () => {
  if (checking) return
  if (!needsEmailChoice.value || isDismissed()) {
    visible.value = false
    return
  }
  checking = true
  try {
    const status = await fetchApiAccountLinkStatus()
    if (!status || !status.enabled || status.linked || isDismissed() || !needsEmailChoice.value) {
      visible.value = false
      return
    }
    step.value = 'choice'
    errorMsg.value = ''
    visible.value = true
  } finally {
    checking = false
  }
}

const dismissPrompt = () => {
  if (busy.value) return
  markDismissed()
  visible.value = false
}

const openBindForm = () => {
  errorMsg.value = ''
  step.value = 'bind'
}

const backToChoice = () => {
  errorMsg.value = ''
  step.value = 'choice'
}

const skipAndProvision = async () => {
  if (busy.value) return
  errorMsg.value = ''
  skipping.value = true
  try {
    const result = await ensureApiAccountLink({ force: true, allowWithoutEmail: true })
    if (result?.linked) {
      markDismissed()
      visible.value = false
      return
    }
    errorMsg.value = t('needtoken.tgEmailPrompt.provisionFailed')
  } finally {
    skipping.value = false
  }
}

const sendCode = async () => {
  if (busy.value || sendCooldown.value > 0) return
  const target = emailInput.value.trim()
  if (!target) {
    errorMsg.value = t('error.email_invalid')
    return
  }
  errorMsg.value = ''
  sending.value = true
  try {
    const ok = await profileStore.sendChangeEmailCode({ kind: 'new', new_email: target })
    if (!ok) {
      errorMsg.value = profileStore.securityError || t('personalCenter.security.sendCodeFailed')
      return
    }
    startCooldown(60)
  } finally {
    sending.value = false
  }
}

const submitBindEmail = async () => {
  if (busy.value) return
  const target = emailInput.value.trim()
  const code = codeInput.value.trim()
  if (!target || !code) {
    errorMsg.value = t('personalCenter.security.bindEmailRequired')
    return
  }
  errorMsg.value = ''
  binding.value = true
  try {
    const ok = await profileStore.changeEmail({ new_email: target, new_code: code })
    if (!ok) {
      errorMsg.value = profileStore.securityError || t('personalCenter.security.changeEmailFailed')
      return
    }
    await ensureApiAccountLink({ force: true })
    markDismissed()
    visible.value = false
  } finally {
    binding.value = false
  }
}

const onKeydown = (e: KeyboardEvent) => {
  if (e.key === 'Escape' && visible.value) {
    dismissPrompt()
  }
}

watch([() => authStore.isAuthenticated, userID, userEmail], () => {
  void checkEligibility()
})

onMounted(() => {
  window.addEventListener('keydown', onKeydown)
  void checkEligibility()
})

onBeforeUnmount(() => {
  window.removeEventListener('keydown', onKeydown)
  stopCooldown()
})
</script>

<style scoped>
.nt-tg-prompt-backdrop {
  position: fixed;
  inset: 0;
  z-index: 90;
  display: grid;
  place-items: center;
  padding: 16px;
  background: var(--nt-overlay-scrim, rgba(15, 23, 20, 0.45));
  backdrop-filter: blur(4px);
}
.nt-tg-prompt-card {
  position: relative;
  width: 100%;
  max-width: 390px;
  padding: 24px 22px 20px;
  border-radius: 18px;
  border: 1px solid var(--nt-card-border, rgba(15, 23, 20, 0.12));
  background: var(--ui-bg-elevated, #ffffff);
  color: var(--ui-text-primary, #0f1714);
  box-shadow: var(--nt-popover-shadow, 0 18px 44px rgba(0, 0, 0, 0.2));
  text-align: center;
}
.nt-tg-prompt-close {
  position: absolute;
  top: 12px;
  right: 12px;
  width: 28px;
  height: 28px;
  display: grid;
  place-items: center;
  border-radius: 8px;
  color: var(--ui-text-muted, #667770);
}
.nt-tg-prompt-close:hover:not(:disabled) {
  background: var(--ui-bg-soft, #eef2f0);
  color: var(--ui-text-primary, #0f1714);
}
.nt-tg-prompt-icon {
  width: 42px;
  height: 42px;
  margin: 0 auto 12px;
  display: grid;
  place-items: center;
  border-radius: 12px;
  background: var(--nt-accent-subtle, rgba(13, 148, 115, 0.08));
  color: var(--ui-accent, #0d9473);
}
.nt-tg-prompt-card h2 {
  font-size: 17px;
  font-weight: 700;
  line-height: 1.3;
  text-wrap: balance;
}
.nt-tg-prompt-form-title {
  margin-bottom: 14px;
}
.nt-tg-prompt-desc {
  margin-top: 6px;
  font-size: 13px;
  line-height: 1.5;
  color: var(--ui-text-secondary, #43524c);
}
.nt-tg-prompt-error {
  margin-top: 12px;
  padding: 8px 10px;
  border-radius: 8px;
  font-size: 12.5px;
  line-height: 1.4;
  text-align: left;
  color: var(--nt-danger-ink, #c2362d);
  background: var(--nt-danger-bg, rgba(220, 38, 38, 0.08));
  border: 1px solid var(--nt-danger-border, rgba(220, 38, 38, 0.25));
}
.nt-tg-prompt-actions {
  margin-top: 16px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.nt-tg-prompt-primary,
.nt-tg-prompt-secondary {
  width: 100%;
  min-height: 40px;
  padding: 0 14px;
  border-radius: 9px;
  font-size: 13.5px;
  font-weight: 650;
  transition: background-color 0.16s, border-color 0.16s;
}
.nt-tg-prompt-primary {
  background: var(--nt-purchase-bg, #0d9473);
  color: var(--nt-purchase-fg, #ffffff);
  border: 1px solid var(--nt-purchase-border, #0b8567);
}
.nt-tg-prompt-primary:hover:not(:disabled) {
  background: var(--nt-purchase-bg-hover, #0a7a5e);
}
.nt-tg-prompt-secondary {
  background: var(--ui-bg-soft, #eef2f0);
  color: var(--ui-text-primary, #0f1714);
  border: 1px solid var(--ui-border-strong, rgba(15, 23, 20, 0.18));
}
.nt-tg-prompt-secondary:hover:not(:disabled) {
  border-color: var(--nt-accent-border-strong, rgba(13, 148, 115, 0.45));
}
.nt-tg-prompt-text {
  min-height: 32px;
  font-size: 12.5px;
  color: var(--ui-text-muted, #667770);
}
.nt-tg-prompt-text:hover:not(:disabled) {
  color: var(--ui-text-primary, #0f1714);
}
.nt-tg-prompt-form {
  display: flex;
  flex-direction: column;
  gap: 10px;
  text-align: left;
}
.nt-tg-prompt-field {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.nt-tg-prompt-field > span {
  font-size: 12px;
  font-weight: 600;
  color: var(--ui-text-secondary, #43524c);
}
.nt-tg-prompt-field input {
  box-sizing: border-box;
  width: 100%;
  height: 40px;
  padding: 0 11px;
  border-radius: 9px;
  border: 1px solid var(--ui-border-strong, rgba(15, 23, 20, 0.2));
  background: var(--nt-surface-input, #f8faf9);
  color: var(--ui-text-primary, #0f1714);
  font-size: 14px;
  outline: none;
}
.nt-tg-prompt-field input:focus-visible {
  border-color: var(--ui-accent, #0d9473);
  box-shadow: 0 0 0 3px var(--ui-focus-ring, rgba(13, 148, 115, 0.3));
}
.nt-tg-prompt-code-row {
  display: flex;
  gap: 8px;
}
.nt-tg-prompt-code-row input {
  flex: 1;
  min-width: 0;
}
.nt-tg-prompt-send {
  flex-shrink: 0;
  height: 40px;
  padding: 0 12px;
  border-radius: 9px;
  border: 1px solid var(--ui-border-strong, rgba(15, 23, 20, 0.2));
  background: var(--ui-bg-soft, #eef2f0);
  color: var(--ui-text-primary, #0f1714);
  font-size: 12.5px;
  font-weight: 600;
  white-space: nowrap;
}
button:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
@media (pointer: coarse) {
  .nt-tg-prompt-field input,
  .nt-tg-prompt-send,
  .nt-tg-prompt-primary,
  .nt-tg-prompt-secondary {
    min-height: 44px;
    font-size: 15px;
  }
}
</style>
