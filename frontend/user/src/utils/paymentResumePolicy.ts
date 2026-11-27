export type PaymentResetReason = 'generic' | 'route_change' | 'change_payment_method'

export interface PaymentResetPolicy {
  resumeLatestPayment: boolean
  clearSelectedChannel: boolean
  stopActivePaymentWatch: boolean
}

export interface CachedPaymentRestorePolicy {
  startActivePaymentWatch: boolean
  autoOpenPayLink: boolean
}

export type PaymentPresentationMode = 'qr' | 'redirect'
export type PaymentLinkNavigationTarget = 'current-tab' | 'new-window'
export type PaymentInteractionLabelKey =
  | 'payment.modeQr'
  | 'payment.modeRedirect'
  | 'payment.modeWap'
  | 'payment.modePage'
export type PaymentResultTitleKey =
  | 'payment.resultTitle'
  | 'payment.resultRedirectTitle'
  | 'payment.modeWap'
  | 'payment.modePage'

const REDIRECT_PAYMENT_INTERACTION_MODES = new Set(['redirect', 'wap', 'page'])
const normalizeInteractionMode = (mode: unknown) => String(mode || '').trim().toLowerCase()

export const isCustomerSurchargePayment = (payment?: { fee_policy?: unknown } | null) => {
  const policy = String(payment?.fee_policy || '').trim().toLowerCase()
  return policy === 'customer_surcharge' || policy === 'legacy_customer_surcharge'
}

export const isRedirectPaymentInteractionMode = (mode: unknown) => {
  return REDIRECT_PAYMENT_INTERACTION_MODES.has(normalizeInteractionMode(mode))
}

export const resolvePaymentPresentationMode = (mode: unknown): PaymentPresentationMode => {
  return isRedirectPaymentInteractionMode(mode) ? 'redirect' : 'qr'
}

export const resolvePaymentLinkNavigationTarget = (automatic: boolean): PaymentLinkNavigationTarget => {
  return automatic ? 'current-tab' : 'new-window'
}

export const resolvePaymentInteractionLabelKey = (mode: unknown): PaymentInteractionLabelKey | null => {
  switch (normalizeInteractionMode(mode)) {
    case 'qr':
      return 'payment.modeQr'
    case 'redirect':
      return 'payment.modeRedirect'
    case 'wap':
      return 'payment.modeWap'
    case 'page':
      return 'payment.modePage'
    default:
      return null
  }
}

export const resolvePaymentResultTitleKey = (mode: unknown): PaymentResultTitleKey => {
  switch (normalizeInteractionMode(mode)) {
    case 'wap':
      return 'payment.modeWap'
    case 'page':
      return 'payment.modePage'
    case 'redirect':
      return 'payment.resultRedirectTitle'
    default:
      return 'payment.resultTitle'
  }
}

export const getPaymentResetPolicy = (reason: PaymentResetReason = 'generic'): PaymentResetPolicy => {
  if (reason === 'change_payment_method') {
    return {
      resumeLatestPayment: false,
      clearSelectedChannel: true,
      stopActivePaymentWatch: true,
    }
  }

  return {
    resumeLatestPayment: true,
    clearSelectedChannel: false,
    stopActivePaymentWatch: false,
  }
}

export const getCachedPaymentRestorePolicy = (): CachedPaymentRestorePolicy => ({
  startActivePaymentWatch: true,
  autoOpenPayLink: false,
})

export const shouldAutoOpenPaymentLink = (payment?: { interaction_mode?: unknown; pay_url?: unknown; fee_policy?: unknown } | null) => {
  if (!payment || isCustomerSurchargePayment(payment)) return false
  const payURL = String(payment.pay_url || '').trim()
  return isRedirectPaymentInteractionMode(payment.interaction_mode) && payURL !== ''
}

export type RechargeBalanceSyncState = 'none' | 'syncing' | 'sync_error' | 'credited'

export const resolveRechargeBalanceSyncState = (
  recharge?: { status?: unknown } | null,
  balanceSync?: { status?: unknown; last_error?: unknown } | null,
): RechargeBalanceSyncState => {
  const orderStatus = String(recharge?.status || '').trim().toLowerCase()
  if (orderStatus !== 'success') {
    return 'none'
  }
  const syncStatus = String(balanceSync?.status || '').trim().toLowerCase()
  if (syncStatus === 'sync_error' || (syncStatus === 'credit_pending' && String(balanceSync?.last_error || '').trim() !== '')) {
    return 'sync_error'
  }
  if (syncStatus === 'syncing' || syncStatus === 'credit_pending') {
    return 'syncing'
  }
  if (syncStatus === 'credited' || syncStatus === '') {
    return 'credited'
  }
  return 'syncing'
}

export const shouldContinueRechargePolling = (
  recharge?: { status?: unknown } | null,
  balanceSync?: { status?: unknown; last_error?: unknown } | null,
): boolean => {
  const orderStatus = String(recharge?.status || '').trim().toLowerCase()
  if (orderStatus === 'pending' || orderStatus === 'initiated') {
    return true
  }
  if (orderStatus === 'success') {
    const syncState = resolveRechargeBalanceSyncState(recharge, balanceSync)
    return syncState === 'syncing' || syncState === 'sync_error'
  }
  return false
}
