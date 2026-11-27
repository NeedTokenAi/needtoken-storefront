<template>
  <div class="space-y-5">
    <!-- 顶部我的钱包总览卡 -->
    <div class="rounded-2xl border bg-card p-5 shadow-xs sm:p-6">
      <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div class="flex items-center gap-2">
          <h2 class="text-lg font-bold tracking-tight text-foreground">
            {{ t('needtoken.memberConsole.assets.title') }}
          </h2>
          <div ref="helpAnchorEl" class="relative">
            <button
              type="button"
              data-testid="wallet-help-trigger"
              class="grid h-6 w-6 place-items-center rounded-full border border-border bg-muted/50 text-muted-foreground transition-colors hover:border-primary/40 hover:bg-primary/10 hover:text-primary"
              :aria-label="t('needtoken.memberConsole.assets.helpAriaLabel')"
              :aria-expanded="helpOpen"
              @click="helpOpen = !helpOpen"
            >
              <HelpCircle class="h-3.5 w-3.5" />
            </button>
            <div
              v-if="helpOpen"
              role="note"
              data-testid="wallet-help-popover"
              class="absolute left-0 top-[calc(100%+8px)] z-30 w-[min(320px,calc(100vw-48px))] rounded-xl border border-border bg-card p-3.5 text-xs leading-relaxed text-muted-foreground shadow-lg"
            >
              <p class="font-semibold text-foreground">{{ t('needtoken.memberConsole.assets.helpTitle') }}</p>
              <ul class="mt-2 list-disc space-y-1.5 pl-4">
                <li>{{ t('needtoken.memberConsole.assets.helpWalletRule') }}</li>
                <li v-if="showGatewayFeatures">{{ t('needtoken.memberConsole.assets.helpPlanRule') }}</li>
                <li>{{ t('needtoken.memberConsole.assets.helpGiftCardRule') }}</li>
              </ul>
            </div>
          </div>
        </div>

        <!-- 醒目主动作组：兑换礼品卡 + 余额充值 位于选购套餐左侧 -->
        <div class="flex flex-wrap items-center gap-2.5">
          <Button
            type="button"
            data-testid="open-redeem-modal-btn"
            class="h-10 rounded-xl px-4 text-sm font-semibold shadow-xs"
            @click="openRedeemModal"
          >
            <Gift class="mr-1.5 h-4 w-4" />
            {{ t('needtoken.memberConsole.assets.redeemGiftCardBtn') }}
          </Button>
          <Button
            type="button"
            variant="outline"
            data-testid="open-recharge-modal-btn"
            class="h-10 rounded-xl border-primary/40 bg-primary/8 px-4 text-sm font-semibold text-primary hover:bg-primary/15 hover:text-primary"
            @click="openRechargeModal"
          >
            <CreditCard class="mr-1.5 h-4 w-4" />
            {{ t('needtoken.memberConsole.assets.rechargeBalanceBtn') }}
          </Button>
          <Button
            v-if="showGatewayFeatures"
            as-child
            variant="secondary"
            class="h-10 rounded-xl border border-foreground/15 bg-foreground px-4 text-sm font-semibold text-background hover:bg-foreground/90"
          >
            <RouterLink to="/plans">
              <Sparkles class="mr-1.5 h-4 w-4" />
              {{ t('needtoken.memberConsole.quick.buyPlans') }}
            </RouterLink>
          </Button>
        </div>
      </div>

      <!-- 3 张简洁核心资产卡 -->
      <div class="mt-4 grid grid-cols-1 gap-3.5" :class="showGatewayFeatures ? 'md:grid-cols-3' : 'md:grid-cols-2'">
        <div class="rounded-xl border border-primary/25 bg-primary/5 p-4">
          <p class="text-xs font-medium text-muted-foreground">{{ t('needtoken.memberConsole.overview.storeWallet') }}</p>
          <p class="mt-1.5 font-mono text-2xl font-black tabular-nums text-primary">
            {{ userProfileStore.loadingWallet ? '—' : storeWalletDisplay }}
          </p>
          <p v-if="showGatewayFeatures && apiQuotaDisplay" class="mt-1 font-mono text-[11.5px] text-muted-foreground">
            {{ t('needtoken.memberConsole.assets.apiUsdEquivalent', { usd: apiQuotaDisplay }) }}
          </p>
        </div>

        <div v-if="showGatewayFeatures" class="rounded-xl border bg-muted/20 p-4">
          <div class="flex items-center justify-between gap-2">
            <p class="text-xs font-medium text-muted-foreground">{{ t('needtoken.memberConsole.overview.apiPlanQuota') }}</p>
            <button
              type="button"
              class="text-[11px] font-semibold text-primary hover:underline"
              @click="$emit('navigate', 'plans')"
            >
              {{ t('needtoken.memberConsole.plans.viewAllPlans') }}
            </button>
          </div>
          <p class="mt-1.5 font-mono text-2xl font-black tabular-nums text-foreground">
            {{ gatewayLoading && !hasApiSummary ? '—' : planQuotaDisplay }}
          </p>
          <p class="mt-1 text-[11.5px] text-muted-foreground">
            {{ t('needtoken.memberConsole.subscriptions.activeCount', { count: activePlanCount }) }}
          </p>
        </div>

        <div class="rounded-xl border bg-muted/20 p-4">
          <div class="flex items-center justify-between gap-2">
            <p class="text-xs font-medium text-muted-foreground">{{ t('needtoken.memberConsole.overview.affiliateCommission') }}</p>
            <button
              type="button"
              class="text-[11px] font-semibold text-primary hover:underline"
              @click="$emit('navigate', 'affiliate')"
            >
              {{ t('needtoken.memberConsole.assets.goWithdraw') }}
            </button>
          </div>
          <p class="mt-1.5 font-mono text-2xl font-black tabular-nums text-foreground">
            {{ affiliateLoading ? '—' : affiliateAvailableDisplay }}
          </p>
          <p class="mt-1 text-[11.5px] text-muted-foreground">{{ affiliateHintText }}</p>
        </div>
      </div>
    </div>

    <!-- 下方合并记录列表：钱包流水 + 礼品卡兑换记录同表展示 -->
    <div class="rounded-2xl border bg-card p-5 shadow-xs sm:p-6">
      <div class="mb-4 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div class="flex items-center gap-2">
          <ReceiptText class="h-4.5 w-4.5 text-primary" />
          <h3 class="text-base font-bold text-foreground">
            {{ t('needtoken.memberConsole.assets.recordsTitle') }}
          </h3>
        </div>
        <div class="flex flex-wrap items-center gap-2">
          <div class="inline-flex flex-wrap items-center gap-1 rounded-xl border bg-muted/40 p-1" role="group" :aria-label="t('needtoken.memberConsole.assets.recordsTitle')">
            <button
              v-for="tab in recordFilterTabs"
              :key="tab.key"
              type="button"
              :data-record-filter="tab.key"
              class="rounded-lg px-2.5 py-1 text-xs font-semibold transition-colors"
              :class="recordFilter === tab.key ? 'bg-card text-primary shadow-2xs' : 'text-muted-foreground hover:text-foreground'"
              @click="recordFilter = tab.key"
            >
              {{ t(tab.labelKey) }}
            </button>
          </div>
          <Button type="button" variant="outline" size="sm" :disabled="txLoading" @click="refreshRecords">
            <RefreshCw class="mr-1.5 h-3.5 w-3.5" :class="txLoading ? 'animate-spin' : ''" />
            {{ t('needtoken.common.refresh') }}
          </Button>
        </div>
      </div>

      <Alert v-if="txError" class="mb-4" variant="destructive">
        <AlertDescription>{{ txError }}</AlertDescription>
      </Alert>

      <div v-if="txLoading" class="space-y-2.5">
        <div v-for="idx in 4" :key="idx" class="h-12 animate-pulse rounded-xl border bg-muted" />
      </div>

      <div v-else-if="filteredTransactions.length === 0" class="rounded-xl border border-dashed px-4 py-10 text-center text-sm text-muted-foreground">
        {{ t('needtoken.memberConsole.assets.recordsEmpty') }}
      </div>

      <div v-else class="overflow-x-auto rounded-xl border">
        <Table>
          <TableHeader>
            <TableRow class="bg-muted/50">
              <TableHead class="px-4">{{ t('personalCenter.wallet.table.createdAt') }}</TableHead>
              <TableHead class="px-4">{{ t('personalCenter.wallet.table.type') }}</TableHead>
              <TableHead class="px-4">{{ t('personalCenter.wallet.table.direction') }}</TableHead>
              <TableHead class="px-4">{{ t('personalCenter.wallet.table.amount') }}</TableHead>
              <TableHead class="px-4">{{ t('personalCenter.wallet.table.balanceAfter') }}</TableHead>
              <TableHead class="px-4">{{ t('personalCenter.wallet.table.remark') }}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            <TableRow v-for="item in filteredTransactions" :key="item.id">
              <TableCell class="whitespace-nowrap px-4 text-xs text-muted-foreground">{{ formatDate(item.created_at) }}</TableCell>
              <TableCell class="whitespace-nowrap px-4">
                <Badge size="sm" :variant="item.type === 'gift_card_redeem' ? 'accent' : 'secondary'">
                  {{ transactionTypeLabel(item.type) }}
                </Badge>
              </TableCell>
              <TableCell class="whitespace-nowrap px-4">
                <Badge size="sm" :variant="item.direction === 'in' ? 'success' : 'destructive'">
                  {{ item.direction === 'in' ? t('personalCenter.wallet.directionIn') : t('personalCenter.wallet.directionOut') }}
                </Badge>
              </TableCell>
              <TableCell class="whitespace-nowrap px-4 font-mono text-sm font-semibold tabular-nums" :class="item.direction === 'in' ? 'text-success' : 'text-destructive'">
                {{ signedAmount(item.direction, item.amount, item.currency) }}
              </TableCell>
              <TableCell class="whitespace-nowrap px-4 font-mono text-sm tabular-nums text-foreground">
                {{ formatMoney(item.balance_after, item.currency) }}
              </TableCell>
              <TableCell class="px-4 text-xs text-muted-foreground">{{ item.remark || '—' }}</TableCell>
            </TableRow>
          </TableBody>
        </Table>
      </div>

      <div v-if="pagination.total_page > 1" class="mt-5 flex flex-wrap items-center justify-center gap-3">
        <Button variant="outline" size="sm" :disabled="pagination.page <= 1 || txLoading" @click="changePage(pagination.page - 1)">
          {{ t('orders.prevPage') }}
        </Button>
        <span class="rounded-full border px-4 py-1.5 text-xs text-muted-foreground">
          {{ t('orders.pageInfo', { page: pagination.page, total: pagination.total_page }) }}
        </span>
        <Button variant="outline" size="sm" :disabled="pagination.page >= pagination.total_page || txLoading" @click="changePage(pagination.page + 1)">
          {{ t('orders.nextPage') }}
        </Button>
      </div>
    </div>

    <!-- 弹窗 1：兑换礼品卡（保留卡密 + 验证码输入，去掉重复余额横条） -->
    <dialog
      ref="redeemDialogRef"
      class="nt-modal-dialog m-auto w-[min(520px,calc(100vw-24px))] rounded-2xl border border-border bg-card p-0 text-foreground shadow-2xl"
      aria-labelledby="nt-redeem-dialog-title"
      @cancel.prevent="closeRedeemModal"
      @click="onBackdropClick($event, 'redeem')"
    >
      <div class="flex items-center justify-between gap-3 border-b border-border px-5 py-4">
        <div class="flex items-center gap-2.5">
          <span class="grid h-8 w-8 place-items-center rounded-xl bg-primary/10 text-primary">
            <Gift class="h-4 w-4" />
          </span>
          <h3 id="nt-redeem-dialog-title" class="text-base font-bold text-foreground">
            {{ t('needtoken.memberConsole.assets.redeemModalTitle') }}
          </h3>
        </div>
        <button
          type="button"
          class="grid h-8 w-8 place-items-center rounded-lg text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
          :aria-label="t('needtoken.memberConsole.assets.closeModal')"
          :disabled="redeemSubmitting"
          @click="closeRedeemModal"
        >
          <X class="h-4 w-4" />
        </button>
      </div>

      <div class="max-h-[calc(100dvh-130px)] overflow-y-auto px-5 py-5">
        <Alert
          v-if="redeemAlert"
          class="mb-4"
          :variant="pageAlertVariant(redeemAlert.level)"
          :class="pageAlertToneClass(redeemAlert.level)"
        >
          <AlertDescription>{{ redeemAlert.message }}</AlertDescription>
        </Alert>

        <div
          v-if="lastRedeem"
          class="mb-4 rounded-xl border border-success/25 bg-success/10 p-3.5 text-xs"
        >
          <div class="flex items-center gap-2 font-semibold text-success">
            <Check class="h-4 w-4" />
            <span>{{ t('personalCenter.giftCard.successTitle') }}</span>
          </div>
          <div class="mt-2 grid grid-cols-1 gap-2 text-success/90 sm:grid-cols-3">
            <div>
              <div class="opacity-75">{{ t('personalCenter.giftCard.successCode') }}</div>
              <div class="mt-0.5 font-mono">{{ String(lastRedeem.gift_card?.code || '-').toUpperCase() }}</div>
            </div>
            <div>
              <div class="opacity-75">{{ t('personalCenter.giftCard.successAmount') }}</div>
              <div class="mt-0.5 font-semibold">{{ redeemedAmountText }}</div>
            </div>
            <div>
              <div class="opacity-75">{{ t('personalCenter.giftCard.successBalance') }}</div>
              <div class="mt-0.5 font-semibold">{{ redeemedBalanceText }}</div>
            </div>
          </div>
        </div>

        <form class="space-y-4" @submit.prevent="submitRedeem">
          <div>
            <Label class="mb-2 block">{{ t('personalCenter.giftCard.codeLabel') }}</Label>
            <Input
              v-model="redeemForm.code"
              type="text"
              maxlength="80"
              autocomplete="off"
              :placeholder="t('personalCenter.giftCard.codePlaceholder')"
              class="h-11 font-mono uppercase tracking-[0.08em]"
            />
          </div>

          <div v-if="redeemOpen && redeemCaptchaEnabled" class="rounded-xl border px-4 py-3">
            <p class="text-xs font-semibold text-muted-foreground">{{ t('auth.common.captchaLabel') }}</p>
            <div class="mt-2">
              <ImageCaptcha
                v-if="captchaProvider === 'image'"
                ref="imageCaptchaRef"
                v-model="captchaPayload"
                :disabled="redeemSubmitting"
                @config-stale="handleCaptchaConfigStale"
              />
              <TurnstileCaptcha
                v-else-if="captchaProvider === 'turnstile'"
                ref="turnstileRef"
                v-model="turnstileToken"
                :site-key="turnstileSiteKey"
              />
            </div>
          </div>

          <div class="flex flex-wrap items-center justify-end gap-2.5 pt-2">
            <Button type="button" variant="outline" :disabled="redeemSubmitting" @click="resetRedeemForm">
              {{ t('personalCenter.giftCard.resetButton') }}
            </Button>
            <Button type="submit" :disabled="redeemSubmitting" class="min-w-[110px] font-semibold">
              {{ redeemSubmitting ? t('personalCenter.giftCard.redeeming') : t('personalCenter.giftCard.redeemButton') }}
            </Button>
          </div>
        </form>
      </div>
    </dialog>

    <!-- 弹窗 2：余额充值（保留金额 / 支付渠道 / 手续费 / 备注） -->
    <dialog
      ref="rechargeDialogRef"
      class="nt-modal-dialog m-auto w-[min(520px,calc(100vw-24px))] rounded-2xl border border-border bg-card p-0 text-foreground shadow-2xl"
      aria-labelledby="nt-recharge-dialog-title"
      @cancel.prevent="closeRechargeModal"
      @click="onBackdropClick($event, 'recharge')"
    >
      <div class="flex items-center justify-between gap-3 border-b border-border px-5 py-4">
        <div class="flex items-center gap-2.5">
          <span class="grid h-8 w-8 place-items-center rounded-xl bg-primary/10 text-primary">
            <CreditCard class="h-4 w-4" />
          </span>
          <h3 id="nt-recharge-dialog-title" class="text-base font-bold text-foreground">
            {{ t('personalCenter.wallet.rechargeTitle') }}
          </h3>
        </div>
        <button
          type="button"
          class="grid h-8 w-8 place-items-center rounded-lg text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
          :aria-label="t('needtoken.memberConsole.assets.closeModal')"
          :disabled="recharging"
          @click="closeRechargeModal"
        >
          <X class="h-4 w-4" />
        </button>
      </div>

      <div class="max-h-[calc(100dvh-130px)] overflow-y-auto px-5 py-5">
        <Alert
          v-if="rechargeAlert"
          class="mb-4"
          :variant="pageAlertVariant(rechargeAlert.level)"
          :class="pageAlertToneClass(rechargeAlert.level)"
        >
          <AlertDescription>{{ rechargeAlert.message }}</AlertDescription>
        </Alert>

        <form class="space-y-4" @submit.prevent="handleRecharge">
          <div>
            <Label class="mb-2 block">{{ t('personalCenter.wallet.amountLabel') }}</Label>
            <Input
              v-model="rechargeForm.amount"
              type="text"
              inputmode="decimal"
              :placeholder="t('personalCenter.wallet.amountPlaceholder')"
              class="h-11 font-mono"
            />
          </div>

          <div>
            <Label class="mb-2 block">{{ t('personalCenter.wallet.channelLabel') }}</Label>
            <select
              v-model.number="rechargeForm.channelId"
              :disabled="!hasChannels || channelLoading || recharging"
              class="h-11 w-full rounded-xl border border-input bg-background px-3 text-sm text-foreground outline-none transition-colors focus:border-primary disabled:opacity-60"
            >
              <option :value="0">{{ t('personalCenter.wallet.channelPlaceholder') }}</option>
              <option v-for="channel in channels" :key="channel.id" :value="channel.id">
                {{ channel.name }}
              </option>
            </select>
            <p v-if="channelLoading" class="mt-1.5 text-xs text-muted-foreground">{{ t('common.loading') }}</p>
            <p v-else-if="!hasChannels" class="mt-1.5 text-xs text-amber-600">{{ t('payment.channelEmpty') }}</p>
          </div>

          <div v-if="selectedChannel?.fee_policy === 'customer_surcharge'" class="grid grid-cols-3 gap-2.5 text-xs">
            <div class="rounded-xl border border-warning/40 p-3">
              <div class="text-warning">{{ t('payment.feeRateLabel') }}</div>
              <div class="mt-1 font-semibold text-foreground">{{ selectedChannelFeeRateDisplay }}</div>
            </div>
            <div class="rounded-xl border border-warning/40 p-3">
              <div class="text-warning">{{ t('payment.fixedFeeLabel') }}</div>
              <div class="mt-1 font-semibold text-foreground">{{ selectedChannelFixedFeeDisplay }}</div>
            </div>
            <div class="rounded-xl border border-warning/40 p-3">
              <div class="text-warning">{{ t('payment.feeAmountLabel') }}</div>
              <div class="mt-1 font-semibold text-foreground">{{ selectedChannelFeeAmountDisplay }}</div>
            </div>
          </div>

          <div>
            <Label class="mb-2 block">{{ t('personalCenter.wallet.remarkLabel') }}</Label>
            <Input
              v-model="rechargeForm.remark"
              type="text"
              :placeholder="t('personalCenter.wallet.remarkPlaceholder')"
              class="h-11"
            />
          </div>

          <div class="flex flex-wrap items-center justify-end gap-2.5 pt-2">
            <Button type="button" variant="outline" :disabled="recharging" @click="closeRechargeModal">
              {{ t('needtoken.memberConsole.keys.cancel') }}
            </Button>
            <Button
              type="submit"
              :disabled="recharging || channelLoading || !hasChannels"
              class="min-w-[110px] font-semibold"
            >
              {{ recharging ? t('personalCenter.wallet.recharging') : t('personalCenter.wallet.rechargeSubmit') }}
            </Button>
          </div>
        </form>
      </div>
    </dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { Check, CreditCard, Gift, HelpCircle, ReceiptText, RefreshCw, Sparkles, X } from 'lucide-vue-next'
import { giftCardAPI, walletAPI, type CaptchaPayload, type GiftCardRedeemResult } from '../../../../api'
import { useAppStore } from '../../../../stores/app'
import { useUserProfileStore } from '../../../../stores/userProfile'
import { useDisplayCurrency } from '../../composables/useDisplayCurrency'
import { pageAlertToneClass, pageAlertVariant, type PageAlert } from '../../../../utils/alerts'
import { amountToCents, basisPointsToPercent, calculateFeeCents, centsToAmount, rateToBasisPoints } from '../../../../utils/money'
import ImageCaptcha from '../../../../components/captcha/ImageCaptcha.vue'
import TurnstileCaptcha from '../../../../components/captcha/TurnstileCaptcha.vue'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { listGatewayLogs } from '../../lib/gatewayConsoleApi'
import {
  buildApiConsumeWalletRows,
  matchesWalletRecordCategory,
  type WalletRecordCategoryFilter,
  type WalletTransactionRow,
} from '../../lib/gatewayConsoleHelpers'

const props = defineProps<{
  showGatewayFeatures: boolean
  storeWalletDisplay: string
  apiQuotaDisplay: string
  hasApiSummary: boolean
  gatewayLoading: boolean
  planQuotaDisplay: string
  activePlanCount: number
  affiliateLoading: boolean
  affiliateAvailableDisplay: string
  affiliateHintText: string
  initialModal?: 'redeem' | 'recharge' | null
}>()

const emit = defineEmits<{
  changed: []
  navigate: [section: string]
}>()

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const appStore = useAppStore()
const userProfileStore = useUserProfileStore()
const { settlementCurrency, rate: usdCnyRate } = useDisplayCurrency()

// --- Help popover ---
const helpOpen = ref(false)
const helpAnchorEl = ref<HTMLElement | null>(null)

// --- Unified wallet + gift card + API pay-as-you-go transactions ---
const txLoading = ref(true)
const txError = ref('')
const transactions = ref<WalletTransactionRow[]>([])
const pagination = ref({
  page: 1,
  page_size: 20,
  total: 0,
  total_page: 1,
})
const recordFilter = ref<WalletRecordCategoryFilter>('all')
const recordFilterTabs: Array<{ key: WalletRecordCategoryFilter; labelKey: string }> = [
  { key: 'all', labelKey: 'needtoken.memberConsole.assets.filterAll' },
  { key: 'recharge', labelKey: 'needtoken.memberConsole.assets.filterRecharge' },
  { key: 'gift_card', labelKey: 'needtoken.memberConsole.assets.filterGiftCard' },
  { key: 'spend', labelKey: 'needtoken.memberConsole.assets.filterSpend' },
]

const filteredTransactions = computed(() =>
  transactions.value.filter(item => matchesWalletRecordCategory(item.type, recordFilter.value)),
)

const formatMoney = (amount?: string, currency?: string) => {
  if (amount === null || amount === undefined || amount === '') return '—'
  if (currency === null || currency === undefined || currency === '') return String(amount)
  return `${amount} ${currency}`
}

const formatDate = (raw?: string) => {
  if (!raw) return '—'
  const date = new Date(raw)
  if (Number.isNaN(date.getTime())) return raw
  return date.toLocaleString()
}

const transactionTypeLabel = (type?: string) => {
  const key = `personalCenter.wallet.types.${type || ''}`
  const translated = t(key)
  return translated === key ? (type || '—') : translated
}

const signedAmount = (direction: string, amount?: string, currency?: string) => {
  const base = formatMoney(amount, currency)
  if (base === '—') return base
  return direction === 'out' ? `-${base}` : `+${base}`
}

const loadTransactions = async (page = 1) => {
  txLoading.value = true
  txError.value = ''
  try {
    const [response, logsRes] = await Promise.all([
      walletAPI.transactions({
        page,
        page_size: pagination.value.page_size,
      }),
      props.showGatewayFeatures && page === 1
        ? listGatewayLogs({ p: 1, page_size: 50, type: 2 }).catch(() => null)
        : Promise.resolve(null),
    ])
    const baseRows: WalletTransactionRow[] = Array.isArray(response.data.data) ? response.data.data : []
    const apiRows = logsRes
      ? buildApiConsumeWalletRows({
          logs: Array.isArray(logsRes.items) ? logsRes.items : [],
          quotaPerUnit: logsRes.quota_per_unit || 500_000,
          siteCurrency: settlementCurrency.value,
          usdCnyRate: Number(usdCnyRate.value) || null,
        })
      : []
    const merged = [...baseRows, ...apiRows].sort((a, b) => {
      const ta = Date.parse(a.created_at) || 0
      const tb = Date.parse(b.created_at) || 0
      return tb - ta
    })
    transactions.value = merged
    pagination.value = response.data.pagination || pagination.value
  } catch (err: any) {
    transactions.value = []
    txError.value = err?.message || t('personalCenter.wallet.errors.loadFailed')
  } finally {
    txLoading.value = false
  }
}

const changePage = (page: number) => {
  if (page < 1 || page > pagination.value.total_page) return
  void loadTransactions(page)
}

const refreshRecords = async () => {
  await Promise.all([
    userProfileStore.loadWalletAccount(),
    loadTransactions(pagination.value.page),
  ])
  emit('changed')
}

// --- Modal 1: Redeem Gift Card ---
const redeemDialogRef = ref<HTMLDialogElement | null>(null)
const redeemOpen = ref(false)
const redeemForm = reactive({ code: '' })
const redeemSubmitting = ref(false)
const redeemAlert = ref<PageAlert | null>(null)
const lastRedeem = ref<GiftCardRedeemResult | null>(null)
const captchaPayload = ref<CaptchaPayload>({})
const turnstileToken = ref('')
const imageCaptchaRef = ref<InstanceType<typeof ImageCaptcha> | null>(null)
const turnstileRef = ref<InstanceType<typeof TurnstileCaptcha> | null>(null)

const captchaConfig = computed(() => appStore.config?.captcha || null)
const captchaProvider = computed(() => String(captchaConfig.value?.provider || 'none'))
const redeemCaptchaEnabled = computed(() => !!captchaConfig.value?.scenes?.gift_card_redeem && captchaProvider.value !== 'none')
const turnstileSiteKey = computed(() => String(captchaConfig.value?.turnstile?.site_key || ''))

const redeemedAmountText = computed(() => {
  const rawAmount = String(lastRedeem.value?.wallet_delta || lastRedeem.value?.gift_card?.amount || '').trim()
  const currency = String(lastRedeem.value?.gift_card?.currency || appStore.config?.currency || 'CNY').trim()
  return rawAmount ? `${rawAmount} ${currency}` : '—'
})

const redeemedBalanceText = computed(() => {
  const balance = String(lastRedeem.value?.wallet?.balance || userProfileStore.walletBalance || '').trim()
  const currency = String(lastRedeem.value?.gift_card?.currency || userProfileStore.walletAccount?.currency || appStore.config?.currency || 'CNY').trim()
  return balance ? `${balance} ${currency}` : '—'
})

const getCaptchaPayload = (): CaptchaPayload | undefined => {
  if (!redeemCaptchaEnabled.value) return undefined
  if (captchaProvider.value === 'image') {
    return {
      captcha_id: captchaPayload.value.captcha_id || '',
      captcha_code: captchaPayload.value.captcha_code || '',
    }
  }
  if (captchaProvider.value === 'turnstile') {
    return turnstileRef.value?.getPayload() ?? { turnstile_token: turnstileToken.value || '' }
  }
  return undefined
}

const resetCaptcha = () => {
  captchaPayload.value = {}
  turnstileToken.value = ''
  imageCaptchaRef.value?.refresh()
  turnstileRef.value?.reset()
}

const resetRedeemForm = () => {
  redeemForm.code = ''
  redeemAlert.value = null
  lastRedeem.value = null
  resetCaptcha()
}

const handleCaptchaConfigStale = async () => {
  await appStore.loadConfig(true)
  captchaPayload.value = {}
  turnstileToken.value = ''
}

const ensureCaptchaPassed = () => {
  if (!redeemCaptchaEnabled.value) return true
  if (captchaProvider.value === 'image') return Boolean(captchaPayload.value.captcha_id && captchaPayload.value.captcha_code)
  if (captchaProvider.value === 'turnstile') return Boolean(turnstileRef.value?.hasPayload() ?? turnstileToken.value)
  return false
}

const openRedeemModal = async () => {
  helpOpen.value = false
  redeemAlert.value = null
  lastRedeem.value = null
  redeemOpen.value = true
  await nextTick()
  if (redeemDialogRef.value && !redeemDialogRef.value.open) {
    redeemDialogRef.value.showModal()
  }
}

const closeRedeemModal = () => {
  if (redeemSubmitting.value) return
  if (redeemDialogRef.value?.open) redeemDialogRef.value.close()
  redeemOpen.value = false
}

const submitRedeem = async () => {
  redeemAlert.value = null
  const code = String(redeemForm.code || '').trim().toUpperCase()
  if (!code) {
    redeemAlert.value = { level: 'warning', message: t('personalCenter.giftCard.errors.codeRequired') }
    return
  }
  if (!ensureCaptchaPassed()) {
    redeemAlert.value = { level: 'warning', message: t('auth.common.captchaRequired') }
    return
  }
  redeemSubmitting.value = true
  try {
    const response = await giftCardAPI.redeem({
      code,
      captcha_payload: getCaptchaPayload(),
    })
    const payload = response.data.data || ({} as GiftCardRedeemResult)
    lastRedeem.value = payload
    if (payload.wallet) {
      userProfileStore.syncWalletAccount({
        balance: String(payload.wallet.balance ?? ''),
        currency: String(payload.gift_card?.currency || appStore.config?.currency || 'CNY'),
      })
    } else {
      void userProfileStore.loadWalletAccount()
    }
    redeemAlert.value = {
      level: 'success',
      message: t('personalCenter.giftCard.redeemSuccess', {
        amount: String(payload.wallet_delta || payload.gift_card?.amount || ''),
        currency: String(payload.gift_card?.currency || appStore.config?.currency || 'CNY'),
      }),
    }
    redeemForm.code = ''
    resetCaptcha()
    emit('changed')
    void loadTransactions(1)
  } catch (err: any) {
    redeemAlert.value = {
      level: 'error',
      message: err?.message || t('personalCenter.giftCard.errors.redeemFailed'),
    }
    if (captchaProvider.value === 'image') imageCaptchaRef.value?.refresh()
    if (captchaProvider.value === 'turnstile') {
      turnstileRef.value?.reset()
      turnstileToken.value = ''
    }
  } finally {
    redeemSubmitting.value = false
  }
}

// --- Modal 2: Wallet Recharge ---
const rechargeDialogRef = ref<HTMLDialogElement | null>(null)
const recharging = ref(false)
const rechargeAlert = ref<PageAlert | null>(null)
const channels = ref<any[]>([])
const channelFetchTimer = ref<number | null>(null)
const channelFetchSeq = ref(0)
const channelLoading = ref(false)
const channelsResolvedAmount = ref('')
const rechargeForm = reactive({
  amount: '',
  channelId: 0,
  remark: '',
})

const EPAY_ALLOWED_CHANNEL_TYPES = new Set(['wechat', 'wxpay', 'alipay', 'qqpay'])
const selectedChannelCurrency = computed(() => String(appStore.config?.currency || 'CNY'))
const selectedChannel = computed(() => channels.value.find((item: any) => item.id === rechargeForm.channelId) || null)

const hasChannels = computed(() => {
  const amount = rechargeForm.amount.trim()
  const amountCents = amountToCents(amount)
  if (!amount || amountCents === null || amountCents <= 0) return true
  if (channelLoading.value) return true
  if (channelsResolvedAmount.value !== amount) return true
  return channels.value.length > 0
})

const channelLimitMeta = (channel?: any) => {
  const minCents = amountToCents(String(channel?.min_amount ?? ''))
  const maxCents = amountToCents(String(channel?.max_amount ?? ''))
  return {
    minCents,
    maxCents,
    hasMin: minCents !== null && minCents > 0,
    hasMax: maxCents !== null && maxCents > 0,
  }
}

const isChannelOutOfRange = (channel: any, targetAmountCents: number) => {
  const meta = channelLimitMeta(channel)
  if (!meta.hasMin && !meta.hasMax) return false
  return Boolean((meta.hasMin && meta.minCents !== null && targetAmountCents < meta.minCents)
    || (meta.hasMax && meta.maxCents !== null && targetAmountCents > meta.maxCents))
}

const normalizeChannels = (list: any[], targetAmountCents: number) => {
  const allowedIds = appStore.config?.wallet_recharge_channel_ids
  const allowedIdSet = Array.isArray(allowedIds) && allowedIds.length > 0 ? new Set(allowedIds.map(Number)) : null
  return list
    .filter((ch: any) => String(ch?.provider_type || '').toLowerCase() !== 'epay' || EPAY_ALLOWED_CHANNEL_TYPES.has(String(ch?.channel_type || '').toLowerCase()))
    .filter((ch: any) => !(Boolean(ch?.hide_amount_out_range) && isChannelOutOfRange(ch, targetAmountCents)))
    .filter((ch: any) => !allowedIdSet || allowedIdSet.has(Number(ch?.id)))
    .map((ch: any) => ({
      id: Number(ch.id),
      name: String(ch.name || ch.channel_type || ch.id),
      channel_type: String(ch.channel_type || ''),
      fee_policy: String(ch.fee_policy || ''),
      fee_rate: String(ch.fee_rate ?? '0'),
      fixed_fee: String(ch.fixed_fee ?? '0'),
      min_amount: String(ch.min_amount ?? '0'),
      max_amount: String(ch.max_amount ?? '0'),
      hide_amount_out_range: Boolean(ch.hide_amount_out_range),
    }))
    .filter((ch: any) => Number.isFinite(ch.id) && ch.id > 0)
}

const selectedChannelAmountHint = computed(() => {
  const channel = selectedChannel.value
  if (!channel) return ''
  const amountCents = amountToCents(rechargeForm.amount.trim())
  if (amountCents === null || amountCents <= 0 || !isChannelOutOfRange(channel, amountCents)) return ''
  const meta = channelLimitMeta(channel)
  if (meta.hasMin && meta.hasMax && meta.minCents !== null && meta.maxCents !== null) {
    return t('payment.channelAmountLimitHint', {
      min: formatMoney(centsToAmount(meta.minCents), selectedChannelCurrency.value),
      max: formatMoney(centsToAmount(meta.maxCents), selectedChannelCurrency.value),
    })
  }
  if (meta.hasMin && meta.minCents !== null) {
    return t('payment.channelAmountMinHint', { min: formatMoney(centsToAmount(meta.minCents), selectedChannelCurrency.value) })
  }
  if (meta.hasMax && meta.maxCents !== null) {
    return t('payment.channelAmountMaxHint', { max: formatMoney(centsToAmount(meta.maxCents), selectedChannelCurrency.value) })
  }
  return ''
})

const selectedChannelFeeRateDisplay = computed(() => {
  const rate = rateToBasisPoints(selectedChannel.value?.fee_rate)
  return rate === null ? '0.00%' : `${basisPointsToPercent(rate)}%`
})
const selectedChannelFixedFeeDisplay = computed(() => formatMoney(String(selectedChannel.value?.fixed_fee ?? '0.00'), selectedChannelCurrency.value))
const selectedChannelFeeAmountDisplay = computed(() => {
  const amountCents = amountToCents(rechargeForm.amount)
  if (amountCents === null || amountCents <= 0) return formatMoney('0.00', selectedChannelCurrency.value)
  const rate = rateToBasisPoints(selectedChannel.value?.fee_rate) || 0
  const fixedFeeCents = amountToCents(selectedChannel.value?.fixed_fee) || 0
  const variableFeeCents = calculateFeeCents(amountCents, rate) || 0
  return formatMoney(centsToAmount(variableFeeCents + fixedFeeCents), selectedChannelCurrency.value)
})

const loadPaymentChannels = async (seq: number, amount: string) => {
  if (seq !== channelFetchSeq.value) return
  const amountCents = amountToCents(amount)
  if (!amount || amountCents === null || amountCents <= 0) {
    channels.value = []
    channelsResolvedAmount.value = ''
    return
  }
  try {
    const response = await walletAPI.getPaymentChannels(amount)
    if (seq !== channelFetchSeq.value) return
    const list = Array.isArray(response.data.data) ? response.data.data : []
    channels.value = normalizeChannels(list, amountCents)
  } catch {
    if (seq !== channelFetchSeq.value) return
    channels.value = []
  } finally {
    if (seq === channelFetchSeq.value) {
      channelsResolvedAmount.value = amount
      channelLoading.value = false
    }
  }
}

const scheduleLoadPaymentChannels = () => {
  const amount = rechargeForm.amount.trim()
  const amountCents = amountToCents(amount)
  if (!amount || amountCents === null || amountCents <= 0) {
    channelFetchSeq.value += 1
    if (channelFetchTimer.value) {
      window.clearTimeout(channelFetchTimer.value)
      channelFetchTimer.value = null
    }
    channelLoading.value = false
    channelsResolvedAmount.value = ''
    channels.value = []
    return
  }
  const seq = channelFetchSeq.value + 1
  channelFetchSeq.value = seq
  channelLoading.value = true
  if (channelFetchTimer.value) {
    window.clearTimeout(channelFetchTimer.value)
    channelFetchTimer.value = null
  }
  channelFetchTimer.value = window.setTimeout(() => {
    channelFetchTimer.value = null
    void loadPaymentChannels(seq, amount)
  }, 260)
}

watch(() => rechargeForm.amount, () => { scheduleLoadPaymentChannels() })
watch(channels, (list) => {
  if (!list.length) {
    rechargeForm.channelId = 0
    return
  }
  if (!list.some((item: any) => item.id === rechargeForm.channelId)) {
    rechargeForm.channelId = list[0]?.id ?? 0
  }
}, { immediate: true })

const openRechargeModal = async () => {
  helpOpen.value = false
  rechargeAlert.value = null
  await nextTick()
  if (rechargeDialogRef.value && !rechargeDialogRef.value.open) {
    rechargeDialogRef.value.showModal()
  }
}

const closeRechargeModal = () => {
  if (recharging.value) return
  if (rechargeDialogRef.value?.open) rechargeDialogRef.value.close()
}

const handleRecharge = async () => {
  rechargeAlert.value = null
  const amount = rechargeForm.amount.trim()
  const amountCents = amountToCents(amount)
  if (!amount || amountCents === null || amountCents <= 0) {
    rechargeAlert.value = { level: 'warning', message: t('personalCenter.wallet.errors.invalidAmount') }
    return
  }
  if (!Number.isFinite(rechargeForm.channelId) || rechargeForm.channelId <= 0) {
    rechargeAlert.value = { level: 'warning', message: t('personalCenter.wallet.errors.channelRequired') }
    return
  }
  if (selectedChannelAmountHint.value) {
    rechargeAlert.value = { level: 'warning', message: selectedChannelAmountHint.value }
    return
  }
  recharging.value = true
  try {
    const response = await walletAPI.recharge({
      amount,
      channel_id: rechargeForm.channelId,
      remark: rechargeForm.remark.trim() || undefined,
    })
    const payload = response.data.data || {}
    const rechargeNo = payload?.recharge?.recharge_no || payload?.recharge_no || ''
    rechargeForm.amount = ''
    rechargeForm.remark = ''
    if (rechargeNo) {
      closeRechargeModal()
      void router.push(`/recharge-orders/${encodeURIComponent(rechargeNo)}`)
    } else {
      rechargeAlert.value = { level: 'success', message: t('personalCenter.wallet.createPaymentSuccess') }
      emit('changed')
      void loadTransactions(1)
    }
  } catch (err: any) {
    rechargeAlert.value = {
      level: 'error',
      message: err?.message || t('personalCenter.wallet.errors.rechargeFailed'),
    }
  } finally {
    recharging.value = false
  }
}

const onBackdropClick = (event: MouseEvent, kind: 'redeem' | 'recharge') => {
  const dialog = kind === 'redeem' ? redeemDialogRef.value : rechargeDialogRef.value
  if (dialog && event.target === dialog) {
    if (kind === 'redeem') closeRedeemModal()
    else closeRechargeModal()
  }
}

const onDocClick = (event: MouseEvent) => {
  if (helpOpen.value && helpAnchorEl.value && !helpAnchorEl.value.contains(event.target as Node)) {
    helpOpen.value = false
  }
}

const onDocKeydown = (event: KeyboardEvent) => {
  if (event.key === 'Escape') {
    helpOpen.value = false
  }
}

const redirectRechargeReturn = () => {
  const query = route.query as Record<string, unknown>
  const rechargeNo = String(query.recharge_no || '').trim()
  const orderNo = String(query.order_no || '').trim()
  const targetNo = rechargeNo || (/^WR/i.test(orderNo) ? orderNo : '')
  if (targetNo) {
    void router.replace(`/recharge-orders/${encodeURIComponent(targetNo)}${window.location.search}`)
  }
}

onMounted(async () => {
  document.addEventListener('click', onDocClick)
  document.addEventListener('keydown', onDocKeydown)
  if (!appStore.config) {
    await appStore.loadConfig()
  }
  await Promise.all([
    userProfileStore.loadWalletAccount(),
    loadTransactions(1),
  ])
  redirectRechargeReturn()
  if (props.initialModal === 'redeem') {
    await openRedeemModal()
  } else if (props.initialModal === 'recharge') {
    await openRechargeModal()
  }
})

onUnmounted(() => {
  document.removeEventListener('click', onDocClick)
  document.removeEventListener('keydown', onDocKeydown)
  channelFetchSeq.value += 1
  if (channelFetchTimer.value) {
    window.clearTimeout(channelFetchTimer.value)
    channelFetchTimer.value = null
  }
})

defineExpose({
  openRedeemModal,
  openRechargeModal,
})
</script>
