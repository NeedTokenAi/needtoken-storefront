<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { userOrderAPI, guestOrderAPI } from '../api/order'
import {
  deriveSubscriptionProgressSteps,
  extractPlatformCredential,
  validateSubscriptionSession,
  type SubscriptionSession,
} from '../utils/subscriptionAuthorization'

const props = defineProps<{ order: { order_no: string; status?: string; children?: any[] }; guestAuth?: { email: string; order_password: string } }>()
const { t } = useI18n()
const eligibility = ref<{ eligible: boolean; guest_proof_required: boolean; plan?: string } | null>(null)
const consent = ref(false)
const nonRefundableConsent = ref(false)
const credential = ref('')
const code = ref('')
const operationID = ref('')
const challengeSent = ref(false)
const cooldown = ref(false)
const busy = ref(false)
const submitted = ref(false)
const failure = ref('')
const progressSteps = computed(() => deriveSubscriptionProgressSteps(props.order.status, submitted.value))
let session: SubscriptionSession | null = null
let expiry: ReturnType<typeof setTimeout> | undefined
let resend: ReturnType<typeof setTimeout> | undefined
let request: AbortController | null = null
let generation = 0
const reset = () => {
  generation++
  request?.abort(); request = null
  if (expiry) clearTimeout(expiry)
  if (resend) clearTimeout(resend)
  session = null; credential.value = ''; code.value = ''; operationID.value = ''
  challengeSent.value = false; cooldown.value = false; consent.value = false; nonRefundableConsent.value = false; submitted.value = false; failure.value = ''; busy.value = false
}
watch(() => [props.order.order_no, props.guestAuth?.email, props.guestAuth?.order_password], async () => {
  reset(); eligibility.value = null
  if (props.order.children?.length) return
  const current = generation
  try {
    const result = props.guestAuth
      ? await guestOrderAPI.subscriptionAuthorization(props.order.order_no, props.guestAuth)
      : await userOrderAPI.subscriptionAuthorization(props.order.order_no)
    if (current === generation) eligibility.value = result.data.data
  } catch { /* No customer action is offered while eligibility is unknown. */ }
}, { immediate: true })

const ensureOperation = () => {
  if (!operationID.value) operationID.value = `auth_${crypto.randomUUID().replace(/-/g, '')}`
  return operationID.value
}
const sendCode = async () => {
  if (!props.guestAuth || busy.value || cooldown.value) return
  const current = generation
  busy.value = true; failure.value = ''
  try {
    if (challengeSent.value) { operationID.value = ''; session = null; code.value = '' }
    await guestOrderAPI.challengeSubscriptionAuthorization(props.order.order_no, props.guestAuth, ensureOperation())
    if (current === generation) {
      challengeSent.value = true; cooldown.value = true
      resend = setTimeout(() => { cooldown.value = false }, 60000)
    }
  } catch { if (current === generation) failure.value = t('subscriptionAuthorization.failed') }
  finally { if (current === generation) busy.value = false }
}
const submit = async () => {
  if (busy.value || !consent.value || !nonRefundableConsent.value || !eligibility.value?.eligible || submitted.value) return
  const platformCredential = extractPlatformCredential(credential.value)
  if (!platformCredential) { failure.value = t('subscriptionAuthorization.invalidCredential'); return }
  const current = generation
  busy.value = true; failure.value = ''
  try {
    if (!session) {
      const payload = { operation_id: ensureOperation(), code: code.value.trim(), consent_purchase: true, consent_cancel_renewal: true }
      const result = props.guestAuth
        ? await guestOrderAPI.openSubscriptionAuthorization(props.order.order_no, props.guestAuth, payload)
        : await userOrderAPI.openSubscriptionAuthorization(props.order.order_no, payload)
      if (current !== generation) return
      session = validateSubscriptionSession(result.data.data)
      if (!session) throw new Error('invalid_session')
      expiry = setTimeout(() => { reset(); failure.value = t('subscriptionAuthorization.expired') }, Date.parse(session.expires_at) - Date.now())
    }
    const controller = new AbortController()
    request = controller
    const timeout = setTimeout(() => controller.abort(), 15000)
    let response: Response
    try {
      response = await fetch(session.credential_url, {
        method: 'POST', credentials: 'omit', redirect: 'error', referrerPolicy: 'no-referrer', signal: controller.signal,
        headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${session.session_token}` },
        body: JSON.stringify({ credential: platformCredential }),
      })
    } finally { clearTimeout(timeout) }
    const result = await response.json()
    if (!response.ok || result.ok !== true || result.accepted !== true || result.authorization_id !== session.authorization_id ||
        result.authorization?.credential_present !== true || result.authorization?.execution_scope !== 'inspection_only') throw new Error('submission_unverified')
    if (current === generation) { submitted.value = true; session = null; code.value = ''; if (expiry) clearTimeout(expiry) }
  } catch { if (current === generation) failure.value = t('subscriptionAuthorization.failed') }
  finally { if (current === generation) { credential.value = ''; busy.value = false; request = null } }
}
onBeforeUnmount(reset)
</script>

<template>
  <template v-if="order.children?.length">
    <SubscriptionAuthorizationPanel v-for="child in order.children" :key="child.order_no" :order="child" :guest-auth="guestAuth" />
  </template>
  <section v-else-if="eligibility?.eligible" class="my-4 rounded-xl border bg-card p-5" :aria-label="t('subscriptionAuthorization.title')">
    <h2 class="text-base font-semibold">{{ t('subscriptionAuthorization.title') }}</h2>
    <p class="mt-2 text-sm text-muted-foreground">{{ t('subscriptionAuthorization.description', { order: order.order_no }) }}</p>
    <ol class="mt-4 grid gap-2 sm:grid-cols-4" :aria-label="t('subscriptionAuthorization.progressTitle')">
      <li
        v-for="(step, idx) in progressSteps"
        :key="step.key"
        class="rounded-lg border px-3 py-2 text-xs"
        :class="step.state === 'done' ? 'border-emerald-500/40 bg-emerald-500/5 text-emerald-700 dark:text-emerald-300' : step.state === 'active' ? 'border-primary/50 bg-primary/5 font-medium text-foreground' : 'border-border text-muted-foreground'"
      >
        <span class="mr-1 font-mono">{{ idx + 1 }}.</span>{{ t(`subscriptionAuthorization.steps.${step.key}`) }}
      </li>
    </ol>
    <div class="mt-4 rounded-lg border border-amber-500/40 bg-amber-500/5 px-3.5 py-2.5 text-xs text-amber-800 dark:text-amber-300">
      {{ t('subscriptionAuthorization.nonRefundableNotice') }}
    </div>
    <p v-if="submitted" role="status" class="mt-4 text-sm">{{ t('subscriptionAuthorization.submitted') }}</p>
    <form v-else class="mt-4 space-y-4" @submit.prevent="submit">
      <div v-if="eligibility.guest_proof_required" class="flex flex-wrap items-end gap-3">
        <label class="text-sm">{{ t('subscriptionAuthorization.emailCode') }}
          <input v-model="code" class="mt-1 block h-10 rounded-md border bg-background px-3" inputmode="numeric" autocomplete="one-time-code" maxlength="6" :disabled="busy" />
        </label>
        <button type="button" class="h-10 rounded-md border px-3 text-sm" :disabled="busy || cooldown" @click="sendCode">{{ t(cooldown ? 'subscriptionAuthorization.codeSent' : 'subscriptionAuthorization.sendCode') }}</button>
      </div>
      <label class="block text-sm">{{ t('subscriptionAuthorization.credential') }}
        <textarea v-model="credential" rows="3" maxlength="32768" class="mt-1 block w-full rounded-md border bg-background p-3 font-mono text-sm" autocomplete="off" autocapitalize="off" spellcheck="false" :disabled="busy" />
      </label>
      <label class="flex items-start gap-2 text-sm">
        <input v-model="consent" type="checkbox" class="mt-1" :disabled="busy" />
        <span>{{ t('subscriptionAuthorization.consent') }}</span>
      </label>
      <label class="flex items-start gap-2 text-sm">
        <input v-model="nonRefundableConsent" type="checkbox" class="mt-1" :disabled="busy" />
        <span>{{ t('subscriptionAuthorization.nonRefundableConsent') }}</span>
      </label>
      <p v-if="failure" role="alert" class="text-sm text-destructive">{{ failure }}</p>
      <button type="submit" class="h-10 rounded-md bg-primary px-4 text-sm text-primary-foreground disabled:opacity-50" :disabled="busy || !consent || !nonRefundableConsent || !credential || (eligibility.guest_proof_required && (!challengeSent || code.length !== 6))">{{ t(busy ? 'subscriptionAuthorization.submitting' : 'subscriptionAuthorization.submit') }}</button>
    </form>
  </section>
</template>
