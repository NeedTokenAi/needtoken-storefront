import { computed, onUnmounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { userAuthAPI, type CaptchaPayload } from '../../../api'
import { useAppStore } from '../../../stores/app'
import { useLogin } from '../../../composables/useLogin'
import { useRegister } from '../../../composables/useRegister'
import { authReturnPath } from '../../../utils/authReturnPath'
import ImageCaptcha from '../../../components/captcha/ImageCaptcha.vue'
import TurnstileCaptcha from '../../../components/captcha/TurnstileCaptcha.vue'

export type AuthEntryMode = 'login' | 'register'
export type AuthEntryStage = 'login' | 'login_email_verify' | 'totp' | 'register_email' | 'register_complete' | 'register_direct'

export function useEmailEntry(initialMode: AuthEntryMode = 'login') {
  const app = useAppStore(), route = useRoute(), router = useRouter(), { t } = useI18n()
  const mode = ref<AuthEntryMode>(initialMode)
  const registerCodeSent = ref(false)
  const email = ref('')
  const error = ref('')
  const emailExistsHint = ref(false)
  const busy = ref(false)
  const countdown = ref(0)
  const setupTwoFactor = ref(false)
  let epoch = 0, disposed = false

  const sendCaptcha = ref<CaptchaPayload>({})
  const sendTurnstileToken = ref('')
  const sendImageRef = ref<InstanceType<typeof ImageCaptcha> | null>(null)
  const sendTurnstileRef = ref<InstanceType<typeof TurnstileCaptcha> | null>(null)

  const destination = () => authReturnPath(route.query.redirect, '/me')
  const nativeLogin = useLogin({ onAuthenticated: () => router.push(destination()) })
  const nativeRegister = useRegister({
    onRegistered: () => router.push(setupTwoFactor.value
      ? { path: '/me/security', query: { setup_2fa: '1', return_to: destination() } }
      : destination()),
  })
  const login = reactive(nativeLogin)
  const registration = reactive(nativeRegister)

  const provider = computed(() => String(app.config?.captcha?.provider || 'none'))
  const sendCaptchaEnabled = computed(() => !!app.config?.captcha?.scenes?.register_send_code && provider.value !== 'none')
  const sendSiteKey = computed(() => String(app.config?.captcha?.turnstile?.site_key || ''))
  const normalizedEmail = computed(() => email.value.trim().toLowerCase())
  const validEmail = computed(() => normalizedEmail.value.length <= 254 && /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(normalizedEmail.value))

  const stage = computed<AuthEntryStage>(() => {
    if (login.step === 'totp') return 'totp'
    if (login.step === 'email_verify') return 'login_email_verify'
    if (mode.value === 'login') return 'login'
    if (!registration.emailVerificationEnabled) return 'register_direct'
    return registerCodeSent.value ? 'register_complete' : 'register_email'
  })

  const message = computed(() => error.value || login.error || registration.error)

  const titleKey = computed(() => {
    switch (stage.value) {
      case 'totp': return 'needtoken.emailAuth.totpTitle'
      case 'login_email_verify': return 'auth.login.emailVerify.title'
      case 'register_email':
      case 'register_complete':
      case 'register_direct':
        return 'needtoken.emailAuth.registerTitle'
      default:
        return 'needtoken.emailAuth.passwordTitle'
    }
  })

  const subtitleText = computed(() => {
    switch (stage.value) {
      case 'totp':
        return t('auth.login.totp.subtitle')
      case 'login_email_verify':
        return t('auth.login.emailVerify.subtitle', { email: login.emailVerifyMaskedEmail })
      case 'register_complete':
        return t('needtoken.emailAuth.registerCodeSubtitle')
      case 'register_email':
      case 'register_direct':
        return t('needtoken.emailAuth.registerSubtitle')
      default:
        return t('needtoken.emailAuth.loginSubtitle')
    }
  })

  const timer = setInterval(() => {
    if (countdown.value > 0) countdown.value--
  }, 1000)

  const clearMessages = () => {
    error.value = ''
    emailExistsHint.value = false
    login.error = ''
    login.info = ''
    registration.error = ''
  }

  const resetSendCaptcha = () => {
    sendImageRef.value?.refresh()
    sendTurnstileRef.value?.reset()
    sendCaptcha.value = {}
    sendTurnstileToken.value = ''
  }

  const resolveSendCaptchaPayload = (): CaptchaPayload | undefined => {
    if (!sendCaptchaEnabled.value) return undefined
    if (provider.value === 'image') {
      return {
        captcha_id: sendCaptcha.value.captcha_id || '',
        captcha_code: sendCaptcha.value.captcha_code || '',
      }
    }
    if (provider.value === 'turnstile') {
      return sendTurnstileRef.value?.getPayload() ?? { turnstile_token: sendTurnstileToken.value }
    }
    return undefined
  }

  const hasValidSendCaptcha = (): boolean => {
    if (!sendCaptchaEnabled.value) return true
    const payload = resolveSendCaptchaPayload()
    if (provider.value === 'image') {
      return Boolean(payload?.captcha_id && payload?.captcha_code)
    }
    if (provider.value === 'turnstile') {
      return Boolean(payload?.turnstile_token || (payload?.captcha_id && payload?.captcha_code))
    }
    return true
  }

  function syncBoundEmail() {
    const addr = normalizedEmail.value
    login.email = addr
    registration.email = addr
    const at = addr.lastIndexOf('@')
    if (at > 0) {
      registration.emailLocalPart = addr.slice(0, at)
      registration.selectedEmailDomain = addr.slice(at + 1)
    }
  }

  function switchMode(nextMode: AuthEntryMode) {
    if (nextMode === 'register' && !registration.registrationEnabled) return
    epoch++
    busy.value = false
    mode.value = nextMode
    registerCodeSent.value = false
    clearMessages()
    login.cancel2FA()
    login.cancelEmailVerify()
    registration.code = ''
    registration.password = ''
    registration.showPassword = false
    setupTwoFactor.value = false
    syncBoundEmail()
    resetSendCaptcha()
    const targetPath = nextMode === 'register' ? '/auth/register' : '/auth/login'
    if (route.path !== targetPath) {
      void router.replace({ path: targetPath, query: route.query })
    }
  }

  function backToRegisterEmail() {
    epoch++
    busy.value = false
    registerCodeSent.value = false
    registration.code = ''
    clearMessages()
    resetSendCaptcha()
  }

  async function submitLogin() {
    clearMessages()
    syncBoundEmail()
    await login.handleLogin()
  }

  async function sendRegisterCode() {
    clearMessages()
    if (!registration.registrationEnabled) {
      error.value = t('auth.register.disabled')
      return
    }
    if (!validEmail.value) {
      error.value = t('error.email_invalid')
      return
    }
    if (registration.inviteOnlyEnabled && !registration.inviteCode.trim()) {
      error.value = t('auth.register.errors.inviteCodeRequired')
      return
    }
    if (busy.value || countdown.value > 0) return
    if (!hasValidSendCaptcha()) {
      error.value = t('auth.common.captchaRequired')
      return
    }

    const current = ++epoch
    const address = normalizedEmail.value
    busy.value = true
    try {
      await userAuthAPI.sendVerifyCode({
        email: address,
        purpose: 'register',
        invite_code: registration.inviteOnlyEnabled ? registration.inviteCode.trim() : undefined,
        captcha_payload: resolveSendCaptchaPayload(),
      })
      if (disposed || current !== epoch) return
      syncBoundEmail()
      registerCodeSent.value = true
      countdown.value = 60
      login.info = t('needtoken.emailAuth.codeSentHint')
    } catch (err: any) {
      if (!disposed && current === epoch) {
        const msg = err?.message || t('auth.register.errors.sendCodeFailed')
        error.value = msg
        const lower = String(msg).toLowerCase()
        if (lower.includes('已存在') || lower.includes('已註冊') || lower.includes('已注册') || lower.includes('already') || lower.includes('exists')) {
          emailExistsHint.value = true
        }
      }
    } finally {
      if (!disposed && current === epoch) {
        busy.value = false
        resetSendCaptcha()
      }
    }
  }

  async function submitRegister() {
    clearMessages()
    syncBoundEmail()
    if (!validEmail.value) {
      error.value = t('error.email_invalid')
      return
    }
    if (registration.emailVerificationEnabled && !/^\d{4,10}$/.test(registration.code.trim())) {
      error.value = t('needtoken.emailAuth.codeInvalid')
      return
    }
    if (!registration.emailVerificationEnabled && sendCaptchaEnabled.value) {
      if (!hasValidSendCaptcha()) {
        error.value = t('auth.common.captchaRequired')
        return
      }
      const payload = resolveSendCaptchaPayload()
      if (provider.value === 'image' && payload) {
        registration.captchaPayload = payload
      } else if (provider.value === 'turnstile' && payload) {
        if (payload.turnstile_token) {
          registration.turnstileToken = payload.turnstile_token
        } else {
          // 降级图片验证码场景：同步传递到 register API
          registration.captchaPayload = payload
        }
      }
    }
    if (!registration.emailVerificationEnabled && sendCaptchaEnabled.value && provider.value === 'turnstile') {
      // 直接带降级/正常 payload 注册
      if (!registration.formValidation.validateAll({ email: normalizedEmail.value, password: registration.password })) return
      if (registration.inviteOnlyEnabled && !registration.inviteCode.trim()) {
        error.value = t('auth.register.errors.inviteCodeRequired')
        return
      }
      if (!registration.agreed) {
        error.value = t('auth.register.errors.agreementRequired')
        return
      }
      try {
        await registration.userAuthStore.register({
          email: normalizedEmail.value,
          password: registration.password,
          code: '',
          agreement_accepted: registration.agreed,
          invite_code: registration.inviteOnlyEnabled ? registration.inviteCode.trim() : undefined,
          captcha_payload: resolveSendCaptchaPayload(),
        })
        await router.push(setupTwoFactor.value
          ? { path: '/me/security', query: { setup_2fa: '1', return_to: destination() } }
          : destination())
      } catch (err: any) {
        error.value = err?.message || t('auth.register.errors.registerFailed')
        resetSendCaptcha()
      }
      return
    }
    await registration.handleRegister()
  }

  function cancelSecondaryStep() {
    clearMessages()
    login.cancel2FA()
    login.cancelEmailVerify()
  }

  async function captchaStale() {
    await app.loadConfig(true)
    resetSendCaptcha()
  }

  watch(email, () => {
    syncBoundEmail()
  })

  watch(
    () => route.path,
    (path) => {
      if (path === '/auth/register' && mode.value !== 'register') {
        mode.value = 'register'
        registerCodeSent.value = false
        clearMessages()
      } else if (path === '/auth/login' && mode.value !== 'login') {
        mode.value = 'login'
        registerCodeSent.value = false
        clearMessages()
      }
    },
  )

  onUnmounted(() => {
    disposed = true
    epoch++
    clearInterval(timer)
    login.password = ''
    registration.password = ''
    registration.code = ''
  })

  return {
    app,
    mode,
    stage,
    email,
    message,
    emailExistsHint,
    busy,
    countdown,
    setupTwoFactor,
    login,
    registration,
    titleKey,
    subtitleText,
    sendCaptcha,
    sendTurnstileToken,
    sendImageRef,
    sendTurnstileRef,
    provider,
    sendCaptchaEnabled,
    sendSiteKey,
    switchMode,
    backToRegisterEmail,
    submitLogin,
    sendRegisterCode,
    submitRegister,
    cancelSecondaryStep,
    captchaStale,
    telegramWidgetRef: nativeLogin.telegramWidgetRef,
    loginImageRef: nativeLogin.imageCaptchaRef,
    loginTurnstileRef: nativeLogin.turnstileRef,
  }
}
