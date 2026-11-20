<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { Check, Copy, Eye, KeyRound, ShieldCheck } from 'lucide-vue-next'
import { guestOrderAPI, userOrderAPI } from '../api/order'

interface AssetClaimVerification {
  source: string
  observed_at: string
  plan_type?: string
  subscription_end?: string
  evidence_sha256?: string
}

interface AssetClaimStatus {
  eligible: boolean
  asset_id?: string
  sku_code?: string
  platform?: string
  sale_mode?: 'buyout' | 'rental' | string
  masked_identifier?: string
  delivered_at?: string
  expires_at?: string
  rental_expires_at?: string
  verification?: AssetClaimVerification
  max_reveals: number
  reveal_count: number
  remaining_reveals: number
  first_revealed_at?: string
  last_revealed_at?: string
}

interface RevealedCredential {
  account_identifier: string
  secret_payload: string
  delivery_notes?: string
}

const props = defineProps<{
  order: { order_no: string; children?: any[] }
  guestAuth?: { email: string; order_password: string }
}>()

const { t } = useI18n()
const status = ref<AssetClaimStatus | null>(null)
const revealed = ref<RevealedCredential | null>(null)
const busy = ref(false)
const copied = ref(false)
const errorMsg = ref('')
let generation = 0
let copyTimer: ReturnType<typeof setTimeout> | undefined

const reset = () => {
  generation++
  revealed.value = null
  errorMsg.value = ''
  busy.value = false
  copied.value = false
  if (copyTimer) clearTimeout(copyTimer)
}

watch(
  () => [props.order.order_no, props.guestAuth?.email, props.guestAuth?.order_password],
  async () => {
    reset()
    status.value = null
    if (props.order.children?.length) return
    const current = generation
    try {
      const res = props.guestAuth
        ? await guestOrderAPI.assetClaimStatus(props.order.order_no, props.guestAuth)
        : await userOrderAPI.assetClaimStatus(props.order.order_no)
      if (current === generation && res?.data?.data?.eligible) {
        status.value = res.data.data as AssetClaimStatus
      }
    } catch {
      /* Non-vault orders or transient failures silently keep panel hidden. */
    }
  },
  { immediate: true },
)

const isRental = computed(() => status.value?.sale_mode === 'rental')
const canReveal = computed(() => Boolean(status.value?.eligible && (status.value.remaining_reveals ?? 0) > 0 && !revealed.value))

const reveal = async () => {
  if (!canReveal.value || busy.value) return
  const current = generation
  busy.value = true
  errorMsg.value = ''
  try {
    const res = props.guestAuth
      ? await guestOrderAPI.revealAssetClaim(props.order.order_no, props.guestAuth)
      : await userOrderAPI.revealAssetClaim(props.order.order_no)
    if (current !== generation) return
    const data = res?.data?.data
    if (!data?.secret_payload) {
      throw new Error('empty_secret')
    }
    revealed.value = {
      account_identifier: data.account_identifier || '',
      secret_payload: data.secret_payload || '',
      delivery_notes: data.delivery_notes || '',
    }
    status.value = {
      ...(status.value || { eligible: true, max_reveals: 1, reveal_count: 1, remaining_reveals: 0 }),
      ...data,
      eligible: true,
    }
  } catch (err: any) {
    if (current === generation) {
      errorMsg.value = err?.message || t('assetVaultClaim.failed')
    }
  } finally {
    if (current === generation) busy.value = false
  }
}

const copyAll = async () => {
  if (!revealed.value) return
  const lines = [
    `${t('assetVaultClaim.accountLabel')}: ${revealed.value.account_identifier}`,
    `${t('assetVaultClaim.secretLabel')}: ${revealed.value.secret_payload}`,
  ]
  if (revealed.value.delivery_notes) {
    lines.push(`${t('assetVaultClaim.notesLabel')}: ${revealed.value.delivery_notes}`)
  }
  try {
    await navigator.clipboard.writeText(lines.join('\n'))
    copied.value = true
    if (copyTimer) clearTimeout(copyTimer)
    copyTimer = setTimeout(() => { copied.value = false }, 2200)
  } catch {
    /* Clipboard unavailable */
  }
}

const formatTime = (iso?: string) => {
  if (!iso) return ''
  const d = new Date(iso)
  return Number.isNaN(d.getTime()) ? iso : d.toLocaleString()
}

// Always clear decrypted secrets from memory when leaving the order page.
onBeforeUnmount(reset)
</script>

<template>
  <template v-if="order.children?.length">
    <AssetVaultClaimPanel
      v-for="child in order.children"
      :key="child.order_no"
      :order="child"
      :guest-auth="guestAuth"
    />
  </template>

  <section
    v-else-if="status?.eligible"
    class="my-4 rounded-2xl border border-emerald-500/30 bg-card p-6 shadow-sm"
    :aria-label="t('assetVaultClaim.title')"
  >
    <div class="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
      <div class="flex items-start gap-3">
        <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-emerald-500/10 text-emerald-600 dark:text-emerald-400">
          <KeyRound class="h-5 w-5" />
        </div>
        <div>
          <h2 class="text-base font-bold text-foreground">{{ t('assetVaultClaim.title') }}</h2>
          <p class="mt-1 text-xs text-muted-foreground">{{ t('assetVaultClaim.subtitle') }}</p>
        </div>
      </div>
      <span class="inline-flex w-fit items-center gap-1.5 rounded-full border border-emerald-500/30 bg-emerald-500/10 px-3 py-1 text-xs font-semibold text-emerald-700 dark:text-emerald-300">
        <ShieldCheck class="h-3.5 w-3.5" />
        {{ isRental ? t('assetVaultClaim.modeRental') : t('assetVaultClaim.modeBuyout') }}
      </span>
    </div>

    <div class="mt-4 grid grid-cols-1 gap-3 rounded-xl border bg-muted/30 p-4 text-xs sm:grid-cols-2 lg:grid-cols-3">
      <div>
        <div class="text-muted-foreground">{{ t('assetVaultClaim.maskedLabel') }}</div>
        <div class="mt-1 font-mono font-semibold text-foreground">{{ status.masked_identifier }}</div>
      </div>
      <div>
        <div class="text-muted-foreground">{{ t('assetVaultClaim.remainingLabel') }}</div>
        <div class="mt-1 font-semibold text-foreground">
          {{ status.remaining_reveals }} / {{ status.max_reveals }}
        </div>
      </div>
      <div v-if="status.verification?.plan_type">
        <div class="text-muted-foreground">{{ t('assetVaultClaim.verifiedPlanLabel') }}</div>
        <div class="mt-1 font-semibold text-foreground">
          {{ status.verification.plan_type }}
          <span v-if="status.verification.observed_at" class="font-normal text-muted-foreground">
            ({{ formatTime(status.verification.observed_at) }})
          </span>
        </div>
      </div>
      <div v-if="isRental && status.rental_expires_at">
        <div class="text-muted-foreground">{{ t('assetVaultClaim.rentalExpiresLabel') }}</div>
        <div class="mt-1 font-semibold text-amber-600 dark:text-amber-400">{{ formatTime(status.rental_expires_at) }}</div>
      </div>
    </div>

    <div v-if="revealed" class="mt-4 space-y-3 rounded-xl border border-amber-400/40 bg-amber-50/60 p-4 dark:bg-amber-950/25">
      <div class="flex flex-wrap items-center justify-between gap-2">
        <p class="text-xs font-semibold text-amber-800 dark:text-amber-300">{{ t('assetVaultClaim.saveNowWarning') }}</p>
        <button
          type="button"
          class="inline-flex items-center gap-1.5 rounded-lg bg-emerald-600 px-3 py-1.5 text-xs font-semibold text-white transition hover:bg-emerald-700"
          @click="copyAll"
        >
          <Check v-if="copied" class="h-3.5 w-3.5" />
          <Copy v-else class="h-3.5 w-3.5" />
          <span>{{ copied ? t('assetVaultClaim.copied') : t('assetVaultClaim.copyAll') }}</span>
        </button>
      </div>
      <div class="space-y-2 rounded-lg border bg-background p-3 font-mono text-xs break-all">
        <div><span class="text-muted-foreground">{{ t('assetVaultClaim.accountLabel') }}：</span>{{ revealed.account_identifier }}</div>
        <div class="whitespace-pre-wrap"><span class="text-muted-foreground">{{ t('assetVaultClaim.secretLabel') }}：</span>{{ revealed.secret_payload }}</div>
        <div v-if="revealed.delivery_notes" class="pt-1 font-sans text-muted-foreground">
          {{ t('assetVaultClaim.notesLabel') }}：{{ revealed.delivery_notes }}
        </div>
      </div>
    </div>

    <div v-else-if="canReveal" class="mt-4 flex flex-wrap items-center justify-between gap-3">
      <p class="text-xs text-muted-foreground">{{ t('assetVaultClaim.beforeRevealHint') }}</p>
      <button
        type="button"
        class="inline-flex items-center gap-2 rounded-xl bg-emerald-600 px-4 py-2.5 text-sm font-semibold text-white shadow-sm transition hover:bg-emerald-700 disabled:opacity-60"
        :disabled="busy"
        @click="reveal"
      >
        <Eye class="h-4 w-4" />
        <span>{{ busy ? t('assetVaultClaim.revealing') : t('assetVaultClaim.revealAction') }}</span>
      </button>
    </div>

    <div v-else class="mt-4 rounded-xl border border-dashed p-3.5 text-xs text-muted-foreground">
      {{ t('assetVaultClaim.exhaustedHint', { time: formatTime(status.last_revealed_at || status.first_revealed_at) }) }}
    </div>

    <p v-if="errorMsg" role="alert" class="mt-3 text-xs font-medium text-destructive">{{ errorMsg }}</p>
  </section>
</template>
