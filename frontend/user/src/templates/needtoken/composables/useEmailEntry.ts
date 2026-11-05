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

export function useEmailEntry() {
  const app = useAppStore(), route = useRoute(), router = useRouter(), { t } = useI18n()
  const phase = ref<'email' | 'code' | 'password' | 'register'>('email')
  const email = ref(''), inboxCode = ref(''), error = ref(''), busy = ref(false), countdown = ref(0)
  const setupTwoFactor = ref(false), ticketExpiresAt = ref(0)
  let epoch = 0, disposed = false
  const sendCaptcha = ref<CaptchaPayload>({}), sendTurnstileToken = ref('')
  const sendImageRef = ref<InstanceType<typeof ImageCaptcha> | null>(null)
  const sendTurnstileRef = ref<InstanceType<typeof TurnstileCaptcha> | null>(null)
  const destination = () => authReturnPath(route.query.redirect)
  const nativeLogin = useLogin({ onAuthenticated: () => router.push(destination()) })
  const nativeRegister = useRegister({ requireEmailProof: true, onRegistered: () => router.push(setupTwoFactor.value
    ? { path: '/me/security', query: { setup_2fa: '1', return_to: destination() } }
    : destination()) })
  const login = reactive(nativeLogin), registration = reactive(nativeRegister)
  const stage = computed(() => login.step === 'totp' ? 'totp' : phase.value)
  const provider = computed(() => String(app.config?.captcha?.provider || 'none'))
  const sendCaptchaEnabled = computed(() => !!app.config?.captcha?.scenes?.register_send_code && provider.value !== 'none')
  const sendSiteKey = computed(() => String(app.config?.captcha?.turnstile?.site_key || ''))
  const normalizedEmail = computed(() => email.value.trim().toLowerCase())
  const validEmail = computed(() => normalizedEmail.value.length <= 254 && /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(normalizedEmail.value))
  const message = computed(() => error.value || login.error || registration.error)
  const titleKey = computed(() => `needtoken.emailAuth.${({ email: 'title', code: 'codeTitle', password: 'passwordTitle', register: 'registerTitle', totp: 'totpTitle' } as const)[stage.value]}`)
  const timer = setInterval(() => { if (countdown.value > 0) countdown.value-- }, 1000)
  const resetCaptcha = () => {
    sendImageRef.value?.refresh(); sendTurnstileRef.value?.reset()
    sendCaptcha.value = {}; sendTurnstileToken.value = ''
  }
  const clearMessages = () => { error.value = ''; login.error = ''; registration.error = '' }
  const captchaPayload = (): CaptchaPayload | undefined => !sendCaptchaEnabled.value ? undefined : provider.value === 'image'
    ? sendCaptcha.value : { turnstile_token: sendTurnstileToken.value }
  function bindEmail() {
    login.email = normalizedEmail.value
    registration.email = normalizedEmail.value
    const at = normalizedEmail.value.lastIndexOf('@')
    registration.emailLocalPart = normalizedEmail.value.slice(0, at)
    registration.selectedEmailDomain = normalizedEmail.value.slice(at + 1)
  }
  async function sendCode() {
    clearMessages()
    if (!validEmail.value) { error.value = t('error.email_invalid'); return }
    if (busy.value || countdown.value > 0) return
    if (sendCaptchaEnabled.value && (provider.value === 'image'
      ? !sendCaptcha.value.captcha_id || !sendCaptcha.value.captcha_code : !sendTurnstileToken.value)) {
      error.value = t('auth.common.captchaRequired'); return
    }
    const current = ++epoch, address = normalizedEmail.value
    busy.value = true
    try {
      await userAuthAPI.sendVerifyCode({ email: address, purpose: 'email_auth', captcha_payload: captchaPayload() })
      if (disposed || current !== epoch || address !== normalizedEmail.value) return
      phase.value = 'code'; inboxCode.value = ''; countdown.value = 60
      bindEmail()
    } catch (err: any) { if (!disposed && current === epoch) error.value = err?.message || t('auth.register.errors.sendCodeFailed') }
    finally { if (!disposed && current === epoch) { busy.value = false; resetCaptcha() } }
  }
  async function verifyCode() {
    clearMessages()
    if (!validEmail.value || !/^\d{6,10}$/.test(inboxCode.value.trim())) { error.value = t('needtoken.emailAuth.codeInvalid'); return }
    if (busy.value) return
    const current = ++epoch, address = normalizedEmail.value
    busy.value = true
    try {
      const response = await userAuthAPI.verifyEmailEntry({ email: address, code: inboxCode.value.trim() })
      if (disposed || current !== epoch || address !== normalizedEmail.value) return
      const result = response.data.data
      bindEmail(); inboxCode.value = ''
      if (result?.next_step === 'password') { phase.value = 'password'; return }
      const expiry = Date.parse(result?.expires_at || '')
      if (result?.next_step !== 'register' || typeof result?.registration_code !== 'string' || !result.registration_code.startsWith('eauth_') || !Number.isFinite(expiry) || expiry <= Date.now()) throw new Error(t('needtoken.emailAuth.codeInvalid'))
      registration.code = result.registration_code; ticketExpiresAt.value = expiry; phase.value = 'register'
    } catch (err: any) { if (!disposed && current === epoch) error.value = err?.message || t('needtoken.emailAuth.codeInvalid') }
    finally { if (!disposed && current === epoch) busy.value = false }
  }
  function passwordShortcut() {
    clearMessages()
    if (!validEmail.value) { error.value = t('error.email_invalid'); return }
    epoch++; busy.value = false; inboxCode.value = ''; registration.code = ''; ticketExpiresAt.value = 0
    bindEmail(); phase.value = 'password'; resetCaptcha()
  }
  function changeEmail() {
    epoch++; busy.value = false; phase.value = 'email'; clearMessages()
    login.cancel2FA(); login.password = ''; registration.password = ''; registration.code = ''
    login.showPassword = false; registration.showPassword = false
    registration.agreed = false; setupTwoFactor.value = false; inboxCode.value = ''; ticketExpiresAt.value = 0; resetCaptcha()
  }
  async function register() {
    clearMessages()
    if (Date.now() >= ticketExpiresAt.value || !registration.code) { changeEmail(); error.value = t('needtoken.emailAuth.expired'); return }
    await registration.handleRegister()
  }
  async function captchaStale() { await app.loadConfig(true); resetCaptcha() }
  watch(email, (value, old) => { if (value !== old && phase.value !== 'email') changeEmail() })
  onUnmounted(() => { disposed = true; epoch++; clearInterval(timer); login.password = ''; registration.password = ''; registration.code = ''; inboxCode.value = '' })
  return { app, phase, stage, email, inboxCode, message, busy, countdown, setupTwoFactor, login, registration,
    titleKey, sendCaptcha, sendTurnstileToken, sendImageRef, sendTurnstileRef, provider, sendCaptchaEnabled,
    sendSiteKey, sendCode, verifyCode, passwordShortcut, changeEmail, register, captchaStale,
    telegramWidgetRef: nativeLogin.telegramWidgetRef, loginImageRef: nativeLogin.imageCaptchaRef, loginTurnstileRef: nativeLogin.turnstileRef }
}
