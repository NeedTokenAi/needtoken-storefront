<template>
  <div class="email-entry theme-auth-page">
    <div class="entry-shell">
      <section class="entry-card" aria-labelledby="entry-title">
        <router-link to="/" class="entry-brand"><BrandMark v-if="!isReseller" /><span>{{ brand }}</span></router-link>

        <div
          v-if="showModeTabs"
          class="mode-tabs"
          role="tablist"
          :aria-label="t('needtoken.emailAuth.title')"
        >
          <button
            type="button"
            role="tab"
            :aria-selected="mode === 'login'"
            :class="{ active: mode === 'login' }"
            @click="switchMode('login')"
          >{{ t('needtoken.emailAuth.tabLogin') }}</button>
          <button
            type="button"
            role="tab"
            :aria-selected="mode === 'register'"
            :class="{ active: mode === 'register' }"
            @click="switchMode('register')"
          >{{ t('needtoken.emailAuth.tabRegister') }}</button>
        </div>

        <header>
          <h1 id="entry-title">{{ t(titleKey) }}</h1>
          <p>{{ subtitleText }}</p>
        </header>

        <div v-if="stage === 'register_complete'" class="email-chip">
          <Mail :size="16" />
          <span>{{ email }}</span>
          <button type="button" @click="backToRegisterEmail">{{ t('needtoken.emailAuth.change') }}</button>
        </div>

        <div v-if="message" role="alert" class="entry-error">
          <p>{{ message }}</p>
          <button
            v-if="emailExistsHint"
            type="button"
            class="entry-inline-action"
            @click="switchMode('login')"
          >{{ t('needtoken.emailAuth.emailExistsSwitch') }}</button>
        </div>
        <p v-if="login.info" role="status" class="entry-info">{{ login.info }}</p>

        <!-- 1. 密码登录（默认主入口：受信任环境一次直接登录；环境变更时自动触发邮箱验证） -->
        <form v-if="stage === 'login'" @submit.prevent="submitLogin">
          <div class="field-group">
            <label for="entry-login-email">{{ t('auth.login.emailLabel') }}</label>
            <input
              id="entry-login-email"
              v-model="email"
              type="email"
              autocomplete="username"
              autocorrect="off"
              autocapitalize="none"
              spellcheck="false"
              maxlength="254"
              required
              :placeholder="t('auth.login.emailPlaceholder')"
              :disabled="login.userAuthStore.loading"
            />
          </div>

          <div class="field-group">
            <label for="entry-password">{{ t('auth.login.passwordLabel') }}</label>
            <div class="password-field">
              <input
                id="entry-password"
                v-model="login.password"
                :type="login.showPassword ? 'text' : 'password'"
                autocomplete="current-password"
                required
                :placeholder="t('auth.login.passwordPlaceholder')"
                :disabled="login.userAuthStore.loading"
              />
              <button
                type="button"
                :aria-pressed="login.showPassword"
                :aria-label="t(login.showPassword ? 'auth.common.hidePassword' : 'auth.common.showPassword')"
                @click="login.showPassword = !login.showPassword"
              ><EyeOff v-if="login.showPassword" :size="18" /><Eye v-else :size="18" /></button>
            </div>
          </div>

          <div v-if="login.loginCaptchaEnabled" class="captcha-field">
            <ImageCaptcha
              v-if="login.captchaProvider === 'image'"
              ref="loginImageRef"
              v-model="login.captchaPayload"
              :disabled="login.userAuthStore.loading"
              @config-stale="login.handleCaptchaConfigStale"
            />
            <TurnstileCaptcha
              v-else-if="login.captchaProvider === 'turnstile'"
              ref="loginTurnstileRef"
              v-model="login.turnstileToken"
              :site-key="login.turnstileSiteKey"
              :disabled="login.userAuthStore.loading"
            />
          </div>

          <div class="form-options">
            <label class="check-label"><input v-model="login.rememberMe" type="checkbox" />{{ t('auth.login.rememberMe') }}</label>
            <router-link v-if="login.emailVerificationEnabled" to="/auth/forgot">{{ t('auth.login.forgot') }}</router-link>
          </div>

          <button class="entry-primary" type="submit" :disabled="login.userAuthStore.loading">
            {{ login.userAuthStore.loading ? t('auth.login.submitting') : t('auth.login.submit') }}<ArrowRight :size="18" />
          </button>
        </form>

        <!-- 2. 登录环境变更：邮箱安全验证码核验（验证通过后记住当前设备 30 天并完成登录） -->
        <form v-else-if="stage === 'login_email_verify'" @submit.prevent="login.handleVerifyLoginEmail">
          <p class="field-hint">{{ t('auth.login.totp.countdown', { seconds: login.challengeRemainingSeconds }) }}</p>
          <div class="field-group">
            <label for="entry-env-code">{{ t('auth.login.emailVerify.codeLabel') }}</label>
            <input
              id="entry-env-code"
              v-model="login.emailVerifyCode"
              class="code-input"
              inputmode="numeric"
              pattern="[0-9]{4,10}"
              autocomplete="one-time-code"
              autocorrect="off"
              autocapitalize="none"
              spellcheck="false"
              maxlength="10"
              required
              :placeholder="t('auth.login.emailVerify.codePlaceholder')"
              :disabled="login.userAuthStore.loading"
            />
          </div>
          <p class="field-hint">{{ t('auth.login.emailVerify.deviceTrustHint') }}</p>
          <button class="entry-primary" type="submit" :disabled="login.userAuthStore.loading">
            {{ login.userAuthStore.loading ? t('auth.login.emailVerify.verifying') : t('auth.login.emailVerify.submit') }}<ArrowRight :size="18" />
          </button>
          <button
            type="button"
            class="entry-text"
            :disabled="login.emailVerifyResending || login.emailVerifyResendCountdown > 0"
            @click="login.handleResendLoginEmail"
          >{{ login.emailVerifyResendCountdown > 0 ? t('auth.login.emailVerify.resendCountdown', { seconds: login.emailVerifyResendCountdown }) : t('auth.login.emailVerify.resend') }}</button>
          <button type="button" class="entry-text" @click="cancelSecondaryStep">{{ t('auth.login.emailVerify.cancel') }}</button>
        </form>

        <!-- 3. 两步验证 TOTP（已启用 2FA 的账号直接走 TOTP，通过后同样记住当前设备） -->
        <form v-else-if="stage === 'totp'" @submit.prevent="login.handleVerify2FA">
          <p class="field-hint">{{ t('auth.login.totp.countdown', { seconds: login.challengeRemainingSeconds }) }}</p>
          <div class="field-group">
            <label for="entry-totp">{{ t(login.totpMode === 'code' ? 'auth.login.totp.codeLabel' : 'auth.login.totp.recoveryLabel') }}</label>
            <input
              v-if="login.totpMode === 'code'"
              id="entry-totp"
              v-model="login.totpCode"
              class="code-input"
              inputmode="numeric"
              autocomplete="one-time-code"
              autocorrect="off"
              autocapitalize="none"
              spellcheck="false"
              maxlength="6"
              required
              :placeholder="t('auth.login.totp.codePlaceholder')"
            />
            <input
              v-else
              id="entry-totp"
              v-model="login.recoveryCode"
              autocomplete="off"
              autocorrect="off"
              autocapitalize="none"
              spellcheck="false"
              required
              :placeholder="t('auth.login.totp.recoveryPlaceholder')"
            />
          </div>
          <button class="entry-primary" type="submit" :disabled="login.userAuthStore.loading">
            {{ login.userAuthStore.loading ? t('auth.login.totp.verifying') : t('auth.login.totp.submit') }}
          </button>
          <button type="button" class="entry-text" @click="login.totpMode = login.totpMode === 'code' ? 'recovery' : 'code'">
            {{ t(login.totpMode === 'code' ? 'auth.login.totp.useRecovery' : 'auth.login.totp.useCode') }}
          </button>
          <button type="button" class="entry-text" @click="cancelSecondaryStep">{{ t('auth.login.totp.cancel') }}</button>
        </form>

        <!-- 4. 新用户注册 Step 1：验证邮箱（发送验证码） -->
        <form v-else-if="stage === 'register_email'" @submit.prevent="sendRegisterCode">
          <div class="field-group">
            <label for="entry-register-email">{{ t('auth.login.emailLabel') }}</label>
            <input
              id="entry-register-email"
              v-model="email"
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
          </div>
          <div v-if="registration.inviteOnlyEnabled" class="field-group">
            <label for="entry-register-invite">{{ t('auth.invite.codeLabel') }}</label>
            <input
              id="entry-register-invite"
              v-model="registration.inviteCode"
              autocomplete="off"
              autocorrect="off"
              autocapitalize="none"
              spellcheck="false"
              required
              :placeholder="t('auth.invite.codePlaceholder')"
              :disabled="busy"
            />
          </div>
          <div v-if="sendCaptchaEnabled" class="captcha-field">
            <ImageCaptcha v-if="provider === 'image'" ref="sendImageRef" v-model="sendCaptcha" :disabled="busy" @config-stale="captchaStale" />
            <TurnstileCaptcha v-else-if="provider === 'turnstile'" ref="sendTurnstileRef" v-model="sendTurnstileToken" :site-key="sendSiteKey" :disabled="busy" />
          </div>
          <button class="entry-primary" type="submit" :disabled="busy || countdown > 0 || !registration.registrationEnabled">
            {{ busy ? t('needtoken.emailAuth.sending') : countdown > 0 ? t('auth.common.countdown', { seconds: countdown }) : t('needtoken.emailAuth.sendRegisterCode') }}<ArrowRight :size="18" />
          </button>
        </form>

        <!-- 5. 新用户注册 Step 2：输入邮箱验证码 + 设置登录密码，一键注册并直接登录 -->
        <form v-else-if="stage === 'register_complete'" @submit.prevent="submitRegister">
          <div class="field-group">
            <label for="entry-reg-code">{{ t('auth.register.codeLabel') }}</label>
            <input
              id="entry-reg-code"
              v-model="registration.code"
              class="code-input"
              inputmode="numeric"
              pattern="[0-9]{4,10}"
              autocomplete="one-time-code"
              autocorrect="off"
              autocapitalize="none"
              spellcheck="false"
              maxlength="10"
              required
              :placeholder="t('needtoken.emailAuth.codePlaceholder')"
              :disabled="registration.userAuthStore.loading"
            />
            <p class="field-hint">{{ t('needtoken.emailAuth.codeHint') }}</p>
          </div>

          <div class="field-group">
            <label for="entry-new-password">{{ t('needtoken.emailAuth.newPassword') }}</label>
            <div class="password-field">
              <input
                id="entry-new-password"
                v-model="registration.password"
                :type="registration.showPassword ? 'text' : 'password'"
                autocomplete="new-password"
                minlength="6"
                required
                :placeholder="t('auth.register.passwordPlaceholder')"
                :disabled="registration.userAuthStore.loading"
              />
              <button
                type="button"
                :aria-pressed="registration.showPassword"
                :aria-label="t(registration.showPassword ? 'auth.common.hidePassword' : 'auth.common.showPassword')"
                @click="registration.showPassword = !registration.showPassword"
              ><EyeOff v-if="registration.showPassword" :size="18" /><Eye v-else :size="18" /></button>
            </div>
            <p v-if="registration.formValidation.getError('password')" class="field-error">{{ registration.formValidation.getError('password') }}</p>
            <p v-else-if="registration.password" class="field-hint">{{ t(`formValidation.passwordStrength.${registration.passwordStrength}`) }}</p>
          </div>

          <div v-if="registration.inviteOnlyEnabled" class="field-group">
            <label for="entry-invite">{{ t('auth.invite.codeLabel') }}</label>
            <input id="entry-invite" v-model="registration.inviteCode" autocomplete="off" autocorrect="off" autocapitalize="none" spellcheck="false" required :placeholder="t('auth.invite.codePlaceholder')" />
          </div>

          <label class="check-label setup-choice">
            <input v-model="setupTwoFactor" type="checkbox" />
            <span>{{ t('needtoken.emailAuth.setupTwoFactor') }}<small>{{ t('needtoken.emailAuth.setupTwoFactorHint') }}</small></span>
            <ShieldCheck :size="20" />
          </label>
          <label class="check-label agreement">
            <input v-model="registration.agreed" type="checkbox" required />
            <span>{{ t('auth.register.agreementPrefix') }} <router-link to="/privacy" target="_blank" rel="noopener noreferrer">{{ t('footer.privacy') }}</router-link> {{ t('auth.register.agreementAnd') }} <router-link to="/terms" target="_blank" rel="noopener noreferrer">{{ t('footer.terms') }}</router-link></span>
          </label>

          <button class="entry-primary" type="submit" :disabled="registration.userAuthStore.loading || !registration.agreed || !registration.registrationEnabled">
            {{ registration.userAuthStore.loading ? t('auth.register.creating') : t('needtoken.emailAuth.submitRegister') }}<ArrowRight :size="18" />
          </button>

          <div v-if="sendCaptchaEnabled && countdown === 0" class="captcha-field">
            <ImageCaptcha v-if="provider === 'image'" ref="sendImageRef" v-model="sendCaptcha" :disabled="busy" @config-stale="captchaStale" />
            <TurnstileCaptcha v-else-if="provider === 'turnstile'" ref="sendTurnstileRef" v-model="sendTurnstileToken" :site-key="sendSiteKey" :disabled="busy" />
          </div>
          <button type="button" class="entry-text" :disabled="busy || countdown > 0" @click="sendRegisterCode">
            {{ countdown > 0 ? t('auth.common.countdown', { seconds: countdown }) : t('needtoken.emailAuth.resend') }}
          </button>
        </form>

        <!-- 6. 未开启邮箱验证时的直接注册 -->
        <form v-else-if="stage === 'register_direct'" @submit.prevent="submitRegister">
          <div class="field-group">
            <label for="entry-direct-email">{{ t('auth.login.emailLabel') }}</label>
            <input
              id="entry-direct-email"
              v-model="email"
              type="email"
              autocomplete="email"
              autocorrect="off"
              autocapitalize="none"
              spellcheck="false"
              maxlength="254"
              required
              :placeholder="t('auth.login.emailPlaceholder')"
              :disabled="registration.userAuthStore.loading"
            />
          </div>
          <div class="field-group">
            <label for="entry-direct-password">{{ t('needtoken.emailAuth.newPassword') }}</label>
            <div class="password-field">
              <input
                id="entry-direct-password"
                v-model="registration.password"
                :type="registration.showPassword ? 'text' : 'password'"
                autocomplete="new-password"
                minlength="6"
                required
                :placeholder="t('auth.register.passwordPlaceholder')"
                :disabled="registration.userAuthStore.loading"
              />
              <button
                type="button"
                :aria-pressed="registration.showPassword"
                :aria-label="t(registration.showPassword ? 'auth.common.hidePassword' : 'auth.common.showPassword')"
                @click="registration.showPassword = !registration.showPassword"
              ><EyeOff v-if="registration.showPassword" :size="18" /><Eye v-else :size="18" /></button>
            </div>
          </div>
          <div v-if="registration.inviteOnlyEnabled" class="field-group">
            <label for="entry-direct-invite">{{ t('auth.invite.codeLabel') }}</label>
            <input id="entry-direct-invite" v-model="registration.inviteCode" autocomplete="off" autocorrect="off" autocapitalize="none" spellcheck="false" required :placeholder="t('auth.invite.codePlaceholder')" />
          </div>
          <div v-if="sendCaptchaEnabled" class="captcha-field">
            <ImageCaptcha v-if="provider === 'image'" ref="sendImageRef" v-model="sendCaptcha" :disabled="registration.userAuthStore.loading" @config-stale="captchaStale" />
            <TurnstileCaptcha v-else-if="provider === 'turnstile'" ref="sendTurnstileRef" v-model="sendTurnstileToken" :site-key="sendSiteKey" :disabled="registration.userAuthStore.loading" />
          </div>
          <label class="check-label setup-choice">
            <input v-model="setupTwoFactor" type="checkbox" />
            <span>{{ t('needtoken.emailAuth.setupTwoFactor') }}<small>{{ t('needtoken.emailAuth.setupTwoFactorHint') }}</small></span>
            <ShieldCheck :size="20" />
          </label>
          <label class="check-label agreement">
            <input v-model="registration.agreed" type="checkbox" required />
            <span>{{ t('auth.register.agreementPrefix') }} <router-link to="/privacy" target="_blank" rel="noopener noreferrer">{{ t('footer.privacy') }}</router-link> {{ t('auth.register.agreementAnd') }} <router-link to="/terms" target="_blank" rel="noopener noreferrer">{{ t('footer.terms') }}</router-link></span>
          </label>
          <button class="entry-primary" type="submit" :disabled="registration.userAuthStore.loading || !registration.agreed || !registration.registrationEnabled">
            {{ registration.userAuthStore.loading ? t('auth.register.creating') : t('needtoken.emailAuth.submitRegister') }}<ArrowRight :size="18" />
          </button>
        </form>

        <div v-show="stage === 'login' || stage === 'register_email' || stage === 'register_direct'" class="social-area">
          <template v-if="login.showThirdPartyLogin">
            <div class="social-divider"><span>{{ t('auth.login.socialOr') }}</span></div>
            <div v-if="login.showTelegramWidget"><div ref="telegramWidgetRef" class="telegram-widget"></div><p class="field-hint">{{ t('auth.login.telegramHint') }}</p></div>
            <button v-else-if="login.showTelegramOidc" type="button" class="entry-secondary" @click="login.startTelegramOidc">{{ t('auth.login.telegramOidcButton') }}</button>
            <p v-else-if="login.showMiniAppLoginHint" class="field-hint">{{ t(login.attemptingMiniAppLogin ? 'auth.login.telegramMiniAppLoggingIn' : 'auth.login.telegramMiniAppHint') }}</p>
            <GoogleIdentityButton
              v-if="login.showGoogleLogin"
              :client-id="login.googleClientID"
              :locale="login.googleButtonLocale"
              :ux-mode="login.googleIdentityUXMode"
              :login-uri="login.googleRedirectLoginURI"
              :prepare-redirect="login.prepareGoogleRedirectLogin"
              :disabled="login.userAuthStore.loading"
              :loading-label="t('auth.login.googleLoading')"
              @credential="login.handleGoogleCredential"
              @error="login.handleGoogleScriptError"
            />
          </template>
          <button v-if="login.showTelegramMiniAppEntry" type="button" class="entry-secondary" @click="login.openTelegramMiniAppEntry">{{ t('auth.login.telegramMiniAppEntryAction') }}</button>
        </div>

        <p class="entry-security"><LockKeyhole :size="13" />{{ t('needtoken.emailAuth.deviceSecurityHint') }}</p>
      </section>
    </div>
    <InviteCodeDialog @success="login.continueAfterLogin" />
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { ArrowRight, Mail, Eye, EyeOff, ShieldCheck, LockKeyhole } from 'lucide-vue-next'
import ImageCaptcha from '../../../components/captcha/ImageCaptcha.vue'
import TurnstileCaptcha from '../../../components/captcha/TurnstileCaptcha.vue'
import GoogleIdentityButton from '../../../components/auth/GoogleIdentityButton.vue'
import InviteCodeDialog from '../../../components/InviteCodeDialog.vue'
import BrandMark from './BrandMark.vue'
import { useEmailEntry, type AuthEntryMode } from '../composables/useEmailEntry'

const props = withDefaults(defineProps<{ initialMode?: AuthEntryMode }>(), {
  initialMode: 'login',
})

const { t } = useI18n()
const {
  app, mode, stage, email, message, emailExistsHint, busy, countdown, setupTwoFactor, login, registration,
  titleKey, subtitleText, sendCaptcha, sendTurnstileToken, sendImageRef, sendTurnstileRef, provider,
  sendCaptchaEnabled, sendSiteKey, switchMode, backToRegisterEmail, submitLogin, sendRegisterCode,
  submitRegister, cancelSecondaryStep, captchaStale, telegramWidgetRef, loginImageRef, loginTurnstileRef,
} = useEmailEntry(props.initialMode)

const isReseller = computed(() => app.isResellerTenant)
const brand = computed(() => isReseller.value ? login.brandSiteName : 'NeedToken.AI')
const showModeTabs = computed(() =>
  registration.registrationEnabled && stage.value !== 'totp' && stage.value !== 'login_email_verify',
)

watch(stage, async () => {
  await nextTick()
  document.querySelector<HTMLInputElement>('.entry-card form input:not([type="checkbox"])')?.focus()
})

void sendImageRef
void sendTurnstileRef
void telegramWidgetRef
void loginImageRef
void loginTurnstileRef
</script>

<style scoped>
.email-entry{min-height:calc(100dvh - 52px);display:grid;place-items:center;padding:14px 16px 20px;background:transparent;color:var(--ui-text-primary)}
.entry-shell{width:100%;max-width:420px}
.entry-card{padding:22px 26px;border:1px solid var(--nt-card-border);border-radius:20px;background:var(--ui-bg-elevated);box-shadow:var(--nt-popover-shadow)}
.entry-brand{display:flex;align-items:center;gap:8px;width:fit-content;color:var(--ui-text-primary);font-size:16px;font-weight:750;line-height:1.2}
.entry-brand :is(img,svg){width:26px;height:26px;flex-shrink:0}
.mode-tabs{display:grid;grid-template-columns:1fr 1fr;gap:4px;padding:3px;margin-top:14px;border-radius:11px;background:var(--ui-bg-soft);border:1px solid var(--ui-border)}
.mode-tabs button{min-height:34px;border-radius:8px;font-size:13px;font-weight:650;color:var(--ui-text-secondary);transition:background-color .16s,color .16s,box-shadow .16s}
.mode-tabs button:hover{color:var(--ui-text-primary)}
.mode-tabs button.active{background:var(--ui-bg-elevated);color:var(--ui-text-primary);box-shadow:0 2px 8px rgba(0,0,0,.18)}
header{margin:14px 0}
h1{font-size:21px;font-weight:730;line-height:1.3;letter-spacing:-.03em}
header p{margin-top:4px;color:var(--ui-text-secondary);font-size:12.5px;line-height:1.5}
.entry-card form{display:flex;flex-direction:column;gap:11px}
.field-group{display:flex;flex-direction:column;gap:5px}
.field-group>label{font-size:12.5px;color:var(--ui-text-secondary);font-weight:600;line-height:1.35}
.entry-card input:not([type=checkbox]){box-sizing:border-box;width:100%;height:42px;border:1px solid var(--ui-border-strong);border-radius:10px;padding:0 12px;color:var(--ui-text-primary);background:var(--nt-surface-input);outline:none;font-size:14px}
.entry-card input:focus-visible{border-color:var(--ui-accent);box-shadow:0 0 0 3px var(--ui-focus-ring)}
.entry-card input::placeholder{color:var(--ui-text-muted);font-size:13px}
.entry-card input:autofill{box-shadow:0 0 0 100px var(--ui-bg-soft) inset;-webkit-text-fill-color:var(--ui-text-primary)}
.entry-primary,.entry-secondary{width:100%;min-height:42px;display:flex;justify-content:center;align-items:center;gap:8px;border-radius:10px;font-size:14px;font-weight:700;transition:transform .18s,box-shadow .18s,background-color .18s,border-color .18s}
.entry-primary{background:var(--nt-purchase-bg);color:var(--nt-purchase-fg);border:1px solid var(--nt-purchase-border);margin-top:4px;box-shadow:inset 0 1px 0 rgba(255,255,255,.16),0 6px 18px var(--ui-accent-soft)}
.entry-secondary{background:var(--ui-bg-soft);border:1px solid var(--ui-border-strong);color:var(--ui-text-primary);min-height:40px;margin-top:8px}
.entry-primary:hover:not(:disabled){background:var(--nt-purchase-bg-hover);border-color:var(--nt-purchase-bg-hover);transform:translateY(-1px)}
.entry-secondary:hover:not(:disabled){border-color:var(--nt-accent-border-strong);transform:translateY(-1px)}
.entry-primary:active:not(:disabled){transform:translateY(1px)}
button:disabled{opacity:.45;cursor:not-allowed}
button:focus-visible,a:focus-visible{outline:2px solid var(--ui-accent);outline-offset:3px}
.entry-text{color:var(--ui-accent);align-self:center;font-size:12px;padding:3px 6px;min-height:30px}
.field-hint{font-size:12px;line-height:1.5;color:var(--ui-text-muted)}
.field-error{font-size:12px;color:var(--nt-danger-ink)}
.email-chip{display:flex;align-items:center;gap:9px;border:1px solid var(--ui-border);background:var(--ui-bg-soft);padding:9px 11px;border-radius:10px;margin-bottom:12px;font-size:13px;min-width:0}
.email-chip>span{flex:1;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}
.email-chip>button{font-size:12px;color:var(--ui-accent);padding:3px;flex-shrink:0}
.entry-error,.entry-info{font-size:12.5px;line-height:1.5;padding:10px 12px;border-radius:9px;margin-bottom:12px;overflow-wrap:anywhere}
.entry-error{color:var(--nt-danger-ink);background:var(--nt-danger-bg);border:1px solid var(--nt-danger-border)}
.entry-inline-action{margin-top:6px;display:inline-flex;align-items:center;font-size:12px;font-weight:650;color:var(--ui-accent);text-decoration:underline;text-underline-offset:3px}
.entry-info{color:var(--ui-accent);background:var(--ui-accent-soft)}
.password-field{position:relative}
.password-field input{padding-right:42px!important}
.password-field button{position:absolute;right:8px;top:0;height:42px;padding:0 6px;color:var(--ui-text-muted)}
.form-options{display:flex;align-items:center;justify-content:space-between;gap:10px;color:var(--ui-text-muted);font-size:12px}
.form-options a,.agreement a{color:var(--ui-accent)}
.check-label{display:flex;gap:8px;align-items:center;font-size:12px!important;font-weight:400!important;line-height:1.5}
.check-label input{accent-color:var(--ui-accent);width:15px;height:15px;flex-shrink:0}
.setup-choice{padding:10px 12px;border:1px solid var(--nt-accent-border);background:var(--nt-accent-subtle);border-radius:10px;margin-top:2px}
.setup-choice small{display:block;color:var(--ui-text-muted);font-size:11px;font-weight:400}
.setup-choice svg{margin-left:auto;flex-shrink:0;color:var(--ui-accent)}
.agreement{align-items:start}
.agreement input{margin-top:3px}
.code-input{text-align:center;letter-spacing:.4em;font-size:18px!important}
.social-divider{border-top:1px solid var(--ui-border);text-align:center;margin:16px 0 12px}
.social-divider span{position:relative;top:-10px;background:var(--ui-bg-elevated);padding:0 12px;font-size:11px;color:var(--ui-text-muted)}
.telegram-widget{display:flex;justify-content:center}
.entry-security{display:flex;align-items:center;justify-content:center;gap:6px;font-size:11px;line-height:1.45;color:var(--ui-text-muted);margin-top:14px}
.captcha-field{margin:2px 0;width:100%;display:flex;justify-content:center}
@media(pointer:coarse){
  .entry-card input:not([type=checkbox]){min-height:46px;font-size:16px}
  .password-field button{height:46px}
  .entry-primary,.entry-secondary{min-height:46px}
}
@media(max-width:480px){
  .email-entry{padding:12px 12px 18px}
  .entry-card{padding:18px 16px;border-radius:18px}
  .entry-card header{margin:12px 0}
  h1{font-size:19px}
}
@media(prefers-reduced-motion:reduce){button{transition:none!important;transform:none!important}}
</style>
