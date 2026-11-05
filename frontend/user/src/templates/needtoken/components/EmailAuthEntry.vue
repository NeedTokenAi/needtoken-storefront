<template>
  <div class="email-entry theme-auth-page">
    <div class="entry-shell">
      <router-link to="/" class="back-link"><ArrowLeft :size="16" />{{ t('auth.login.backHome') }}</router-link>
      <section class="entry-card" aria-labelledby="entry-title">
        <router-link to="/" class="entry-brand"><BrandMark v-if="!isReseller" /><span>{{ brand }}</span></router-link>
        <header><h1 id="entry-title">{{ t(titleKey) }}</h1><p>{{ t(stage === 'email' ? 'needtoken.emailAuth.subtitle' : stage === 'totp' ? 'auth.login.totp.subtitle' : 'needtoken.emailAuth.continueHint') }}</p></header>
        <div v-if="stage !== 'email' && stage !== 'totp'" class="email-chip"><Mail :size="16" /><span>{{ email }}</span><button type="button" @click="changeEmail">{{ t('needtoken.emailAuth.change') }}</button></div>
        <p v-if="message" role="alert" class="entry-error">{{ message }}</p>
        <p v-if="login.info" role="status" class="entry-info">{{ login.info }}</p>

        <form v-if="stage === 'email'" @submit.prevent="sendCode">
          <label for="entry-email">{{ t('auth.login.emailLabel') }}</label>
          <input id="entry-email" v-model="email" type="email" autocomplete="username" maxlength="254" required :placeholder="t('auth.login.emailPlaceholder')" :disabled="busy" />
          <p class="field-hint">{{ t('needtoken.emailAuth.privacyHint') }}</p>
          <div v-if="sendCaptchaEnabled" class="captcha-field">
            <ImageCaptcha v-if="provider === 'image'" ref="sendImageRef" v-model="sendCaptcha" :disabled="busy" @config-stale="captchaStale" />
            <TurnstileCaptcha v-else-if="provider === 'turnstile'" ref="sendTurnstileRef" v-model="sendTurnstileToken" :site-key="sendSiteKey" />
          </div>
          <button class="entry-primary" type="submit" :disabled="busy || countdown > 0">{{ busy ? t('needtoken.emailAuth.sending') : countdown > 0 ? t('auth.common.countdown', { seconds: countdown }) : t('needtoken.emailAuth.continue') }}<ArrowRight :size="18" /></button>
          <button type="button" class="entry-text" @click="passwordShortcut">{{ t('needtoken.emailAuth.passwordShortcut') }}</button>
        </form>

        <form v-else-if="stage === 'code'" @submit.prevent="verifyCode">
          <label for="entry-code">{{ t('auth.register.codeLabel') }}</label>
          <input id="entry-code" v-model="inboxCode" class="code-input" inputmode="numeric" pattern="[0-9]{6,10}" autocomplete="one-time-code" maxlength="10" required :placeholder="t('needtoken.emailAuth.codePlaceholder')" />
          <p class="field-hint">{{ t('needtoken.emailAuth.codeHint') }}</p>
          <button class="entry-primary" type="submit" :disabled="busy">{{ busy ? t('auth.login.totp.verifying') : t('needtoken.emailAuth.verify') }}<ArrowRight :size="18" /></button>
          <div v-if="sendCaptchaEnabled && countdown === 0" class="captcha-field">
            <ImageCaptcha v-if="provider === 'image'" ref="sendImageRef" v-model="sendCaptcha" :disabled="busy" @config-stale="captchaStale" />
            <TurnstileCaptcha v-else-if="provider === 'turnstile'" ref="sendTurnstileRef" v-model="sendTurnstileToken" :site-key="sendSiteKey" />
          </div>
          <button type="button" class="entry-text" :disabled="busy || countdown > 0" @click="sendCode">{{ countdown > 0 ? t('auth.common.countdown', { seconds: countdown }) : t('needtoken.emailAuth.resend') }}</button>
        </form>

        <form v-else-if="stage === 'password'" @submit.prevent="login.handleLogin">
          <label for="entry-password">{{ t('auth.login.passwordLabel') }}</label>
          <div class="password-field"><input id="entry-password" v-model="login.password" :type="login.showPassword ? 'text' : 'password'" autocomplete="current-password" required :placeholder="t('auth.login.passwordPlaceholder')" /><button type="button" :aria-label="t(login.showPassword ? 'auth.common.hidePassword' : 'auth.common.showPassword')" @click="login.showPassword = !login.showPassword"><EyeOff v-if="login.showPassword" :size="18" /><Eye v-else :size="18" /></button></div>
          <div v-if="login.loginCaptchaEnabled" class="captcha-field">
            <ImageCaptcha v-if="login.captchaProvider === 'image'" ref="loginImageRef" v-model="login.captchaPayload" :disabled="login.userAuthStore.loading" @config-stale="login.handleCaptchaConfigStale" />
            <TurnstileCaptcha v-else-if="login.captchaProvider === 'turnstile'" ref="loginTurnstileRef" v-model="login.turnstileToken" :site-key="login.turnstileSiteKey" />
          </div>
          <div class="form-options"><label class="check-label"><input v-model="login.rememberMe" type="checkbox" />{{ t('auth.login.rememberMe') }}</label><router-link v-if="login.emailVerificationEnabled" to="/auth/forgot">{{ t('auth.login.forgot') }}</router-link></div>
          <button class="entry-primary" type="submit" :disabled="login.userAuthStore.loading">{{ login.userAuthStore.loading ? t('auth.login.submitting') : t('auth.login.submit') }}<ArrowRight :size="18" /></button>
          <button type="button" class="entry-text" @click="changeEmail">{{ t('needtoken.emailAuth.useEmailCode') }}</button>
        </form>

        <form v-else-if="stage === 'register'" @submit.prevent="register">
          <label for="entry-new-password">{{ t('needtoken.emailAuth.newPassword') }}</label>
          <div class="password-field"><input id="entry-new-password" v-model="registration.password" :type="registration.showPassword ? 'text' : 'password'" autocomplete="new-password" minlength="6" required :placeholder="t('auth.register.passwordPlaceholder')" /><button type="button" :aria-label="t(registration.showPassword ? 'auth.common.hidePassword' : 'auth.common.showPassword')" @click="registration.showPassword = !registration.showPassword"><EyeOff v-if="registration.showPassword" :size="18" /><Eye v-else :size="18" /></button></div>
          <p v-if="registration.formValidation.getError('password')" class="field-error">{{ registration.formValidation.getError('password') }}</p>
          <p v-else-if="registration.password" class="field-hint">{{ t(`formValidation.passwordStrength.${registration.passwordStrength}`) }}</p>
          <template v-if="registration.inviteOnlyEnabled"><label for="entry-invite">{{ t('auth.invite.codeLabel') }}</label><input id="entry-invite" v-model="registration.inviteCode" autocomplete="off" required :placeholder="t('auth.invite.codePlaceholder')" /></template>
          <label class="check-label setup-choice"><input v-model="setupTwoFactor" type="checkbox" /><span>{{ t('needtoken.emailAuth.setupTwoFactor') }}<small>{{ t('needtoken.emailAuth.setupTwoFactorHint') }}</small></span><ShieldCheck :size="20" /></label>
          <label class="check-label agreement"><input v-model="registration.agreed" type="checkbox" required /><span>{{ t('auth.register.agreementPrefix') }} <router-link to="/privacy" target="_blank" rel="noopener noreferrer">{{ t('footer.privacy') }}</router-link> {{ t('auth.register.agreementAnd') }} <router-link to="/terms" target="_blank" rel="noopener noreferrer">{{ t('footer.terms') }}</router-link></span></label>
          <button class="entry-primary" type="submit" :disabled="registration.userAuthStore.loading || !registration.agreed || !registration.registrationEnabled">{{ registration.userAuthStore.loading ? t('auth.register.creating') : t('auth.register.create') }}<ArrowRight :size="18" /></button>
        </form>

        <form v-else-if="stage === 'totp'" @submit.prevent="login.handleVerify2FA">
          <p class="field-hint">{{ t('auth.login.totp.countdown', { seconds: login.challengeRemainingSeconds }) }}</p>
          <label for="entry-totp">{{ t(login.totpMode === 'code' ? 'auth.login.totp.codeLabel' : 'auth.login.totp.recoveryLabel') }}</label>
          <input v-if="login.totpMode === 'code'" id="entry-totp" v-model="login.totpCode" class="code-input" inputmode="numeric" autocomplete="one-time-code" maxlength="6" required :placeholder="t('auth.login.totp.codePlaceholder')" />
          <input v-else id="entry-totp" v-model="login.recoveryCode" autocomplete="off" required :placeholder="t('auth.login.totp.recoveryPlaceholder')" />
          <button class="entry-primary" type="submit" :disabled="login.userAuthStore.loading">{{ login.userAuthStore.loading ? t('auth.login.totp.verifying') : t('auth.login.totp.submit') }}</button>
          <button type="button" class="entry-text" @click="login.totpMode = login.totpMode === 'code' ? 'recovery' : 'code'">{{ t(login.totpMode === 'code' ? 'auth.login.totp.useRecovery' : 'auth.login.totp.useCode') }}</button>
          <button type="button" class="entry-text" @click="changeEmail">{{ t('auth.login.totp.cancel') }}</button>
        </form>

        <div v-show="stage !== 'totp'" class="social-area">
          <template v-if="login.showThirdPartyLogin">
            <div class="social-divider"><span>{{ t('auth.login.socialOr') }}</span></div>
            <div v-if="login.showTelegramWidget"><div ref="telegramWidgetRef" class="telegram-widget"></div><p class="field-hint">{{ t('auth.login.telegramHint') }}</p></div>
            <button v-else-if="login.showTelegramOidc" type="button" class="entry-secondary" @click="login.startTelegramOidc">{{ t('auth.login.telegramOidcButton') }}</button>
            <p v-else-if="login.showMiniAppLoginHint" class="field-hint">{{ t(login.attemptingMiniAppLogin ? 'auth.login.telegramMiniAppLoggingIn' : 'auth.login.telegramMiniAppHint') }}</p>
            <GoogleIdentityButton v-if="login.showGoogleLogin" :client-id="login.googleClientID" :locale="login.googleButtonLocale" :ux-mode="login.googleIdentityUXMode" :login-uri="login.googleRedirectLoginURI" :prepare-redirect="login.prepareGoogleRedirectLogin" :disabled="login.userAuthStore.loading" :loading-label="t('auth.login.googleLoading')" @credential="login.handleGoogleCredential" @error="login.handleGoogleScriptError" />
          </template>
          <button v-if="login.showTelegramMiniAppEntry" type="button" class="entry-secondary" @click="login.openTelegramMiniAppEntry">{{ t('auth.login.telegramMiniAppEntryAction') }}</button>
        </div>
        <p class="entry-security"><LockKeyhole :size="13" />{{ t('needtoken.emailAuth.securityHint') }}</p>
      </section>
    </div>
    <InviteCodeDialog @success="login.continueAfterLogin" />
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { ArrowLeft, ArrowRight, Mail, Eye, EyeOff, ShieldCheck, LockKeyhole } from 'lucide-vue-next'
import ImageCaptcha from '../../../components/captcha/ImageCaptcha.vue'
import TurnstileCaptcha from '../../../components/captcha/TurnstileCaptcha.vue'
import GoogleIdentityButton from '../../../components/auth/GoogleIdentityButton.vue'
import InviteCodeDialog from '../../../components/InviteCodeDialog.vue'
import BrandMark from './BrandMark.vue'
import { useEmailEntry } from '../composables/useEmailEntry'
const { t } = useI18n()
const { app, stage, email, inboxCode, message, busy, countdown, setupTwoFactor, login, registration, titleKey,
  sendCaptcha, sendTurnstileToken, sendImageRef, sendTurnstileRef, provider, sendCaptchaEnabled, sendSiteKey,
  sendCode, verifyCode, passwordShortcut, changeEmail, register, captchaStale, telegramWidgetRef, loginImageRef, loginTurnstileRef } = useEmailEntry()
const isReseller = computed(() => app.isResellerTenant)
const brand = computed(() => isReseller.value ? login.brandSiteName : 'NeedToken.AI')
watch(stage, async () => { await nextTick(); document.querySelector<HTMLInputElement>('.entry-card form input:not([type="checkbox"])')?.focus() })
void sendImageRef; void sendTurnstileRef; void telegramWidgetRef; void loginImageRef; void loginTurnstileRef
</script>

<style scoped>
.email-entry{min-height:100svh;display:grid;place-items:center;padding:36px 20px 60px;background:radial-gradient(ellipse at 50% 0%,#11342f80,transparent 54%),#080c10;color:#e8eef3}.entry-shell{width:100%;max-width:440px}.back-link{display:inline-flex;gap:8px;align-items:center;color:#a8b4c1;font-size:13px;margin:0 0 20px 4px}.entry-card{padding:32px;border:1px solid #ffffff1f;border-radius:24px;background:linear-gradient(145deg,#172129ed,#0e151bee);box-shadow:0 24px 80px #0005}.entry-brand{display:flex;align-items:center;gap:10px;width:fit-content;color:#eff7f5;font-size:19px;font-weight:750}.entry-brand img{width:34px;height:34px}header{margin:28px 0 24px}h1{font-size:27px;font-weight:730;line-height:1.35;letter-spacing:-.04em}header p{margin-top:9px;color:#99a8b6;font-size:13px;line-height:1.7}.entry-card form{display:flex;flex-direction:column;gap:12px}.entry-card form>label{font-size:13px;color:#c1ced7;font-weight:600}.entry-card input:not([type=checkbox]){box-sizing:border-box;width:100%;height:48px;border:1px solid #ffffff29;border-radius:11px;padding:0 13px;color:#f3f8fa;background:#080f17b3;outline:none;font-size:15px}.entry-card input:focus-visible{border-color:#2dd4bf;box-shadow:0 0 0 3px #2dd4bf24}.entry-card input::placeholder{color:#778997;font-size:13px}.entry-card input:autofill{box-shadow:0 0 0 100px #172932 inset;-webkit-text-fill-color:#eef7f5}.entry-primary,.entry-secondary{width:100%;min-height:48px;display:flex;justify-content:center;align-items:center;gap:10px;border-radius:11px;font-size:14px;font-weight:700;transition:transform .18s,box-shadow .18s,filter .18s}.entry-primary{background:linear-gradient(135deg,#45e3c4,#20cba7);color:#06251e;border:1px solid #a6ffdf8c;margin-top:8px;box-shadow:inset 0 1px 0 #e0fff078,0 6px 18px #15bf941c}.entry-secondary{background:#ffffff09;border:1px solid #ffffff26;color:#dbe9ed;margin-top:12px}.entry-primary:hover:not(:disabled),.entry-secondary:hover{transform:translateY(-1px);box-shadow:0 6px 24px #22d3a52c,inset 0 1px #fff7}.entry-primary:active:not(:disabled){transform:translateY(1px)}button:disabled{opacity:.45;cursor:not-allowed}button:focus-visible,a:focus-visible{outline:2px solid #5eead4;outline-offset:3px}.entry-text{color:#8edccc;align-self:center;font-size:12px;padding:4px 6px;min-height:32px}.field-hint{font-size:12px;line-height:1.7;color:#96a8b5}.field-error{font-size:12px;color:#ffb2ad}.email-chip{display:flex;align-items:center;gap:9px;border:1px solid #ffffff19;background:#ffffff05;padding:11px;border-radius:10px;margin-bottom:22px;font-size:13px;min-width:0}.email-chip>span{flex:1;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}.email-chip>button{font-size:12px;color:#5ed8bd;padding:3px;flex-shrink:0}.entry-error,.entry-info{font-size:13px;line-height:1.6;padding:12px;border-radius:9px;margin-bottom:16px;overflow-wrap:anywhere}.entry-error{color:#fecaca;background:#ef444414;border:1px solid #ef444435}.entry-info{color:#99f6e4;background:#14b8a614}.password-field{position:relative}.password-field input{padding-right:44px!important}.password-field button{position:absolute;right:10px;top:0;height:48px;padding:0 5px;color:#91a6b3}.form-options{display:flex;align-items:center;justify-content:space-between;gap:10px;color:#9eafbc;font-size:12px}.form-options a,.agreement a{color:#7de0c9}.check-label{display:flex;gap:9px;align-items:center;font-size:12px!important;font-weight:400!important;line-height:1.7}.check-label input{accent-color:#2dd4bf;width:15px;height:15px;flex-shrink:0}.setup-choice{padding:13px;border:1px solid #2dd4bf38;background:#2dd4bf08;border-radius:10px;margin-top:8px}.setup-choice small{display:block;color:#8ca4b0;font-size:11px;font-weight:400}.setup-choice svg{margin-left:auto;flex-shrink:0;color:#5cceba}.agreement{align-items:start}.agreement input{margin-top:4px}.code-input{text-align:center;letter-spacing:.4em;font-size:20px!important}.social-divider{border-top:1px solid #ffffff1b;text-align:center;margin:26px 0 20px}.social-divider span{position:relative;top:-10px;background:#141d25;padding:0 12px;font-size:11px;color:#8da0ad}.telegram-widget{display:flex;justify-content:center}.entry-security{display:flex;align-items:center;justify-content:center;gap:7px;font-size:11px;color:#78938f;margin-top:25px}.captcha-field{margin:4px 0}@media(max-width:480px){.email-entry{padding:24px 16px 40px}.entry-card{padding:26px 22px}.entry-card header{margin-top:24px}h1{font-size:25px}}@media(prefers-reduced-motion:reduce){button{transition:none!important;transform:none!important}}
</style>
