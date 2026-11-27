import test from 'node:test'
import assert from 'node:assert/strict'
import {
  getCachedPaymentRestorePolicy,
  getPaymentResetPolicy,
  isCustomerSurchargePayment,
  isRedirectPaymentInteractionMode,
  resolvePaymentInteractionLabelKey,
  resolvePaymentLinkNavigationTarget,
  resolvePaymentPresentationMode,
  resolvePaymentResultTitleKey,
  resolveRechargeBalanceSyncState,
  shouldAutoOpenPaymentLink,
  shouldContinueRechargePolling,
} from '../src/utils/paymentResumePolicy.ts'

test('manual payment method changes do not auto resume the latest payment', () => {
  assert.deepEqual(getPaymentResetPolicy('change_payment_method'), {
    resumeLatestPayment: false,
    clearSelectedChannel: true,
    stopActivePaymentWatch: true,
  })
})

test('route changes keep the normal latest payment resume behavior', () => {
  assert.deepEqual(getPaymentResetPolicy('route_change'), {
    resumeLatestPayment: true,
    clearSelectedChannel: false,
    stopActivePaymentWatch: false,
  })
})

test('cached payment restore resumes status watching without opening a cashier', () => {
  assert.deepEqual(getCachedPaymentRestorePolicy(), {
    startActivePaymentWatch: true,
    autoOpenPayLink: false,
  })
})

test('Alipay page and WAP modes use redirect presentation while QR stays scannable', () => {
  assert.equal(resolvePaymentPresentationMode('qr'), 'qr')
  assert.equal(resolvePaymentPresentationMode('redirect'), 'redirect')
  assert.equal(resolvePaymentPresentationMode('wap'), 'redirect')
  assert.equal(resolvePaymentPresentationMode('page'), 'redirect')
  assert.equal(isRedirectPaymentInteractionMode(' WAP '), true)
  assert.equal(isRedirectPaymentInteractionMode('qr'), false)
  assert.equal(resolvePaymentInteractionLabelKey('qr'), 'payment.modeQr')
  assert.equal(resolvePaymentInteractionLabelKey('wap'), 'payment.modeWap')
  assert.equal(resolvePaymentInteractionLabelKey('page'), 'payment.modePage')
  assert.equal(resolvePaymentResultTitleKey('qr'), 'payment.resultTitle')
  assert.equal(resolvePaymentResultTitleKey('wap'), 'payment.modeWap')
  assert.equal(resolvePaymentResultTitleKey('page'), 'payment.modePage')
})

test('redirect-style payments auto open unless a customer fee needs confirmation', () => {
  assert.equal(
    shouldAutoOpenPaymentLink({ interaction_mode: 'redirect', pay_url: 'https://pay.example.com' }),
    true,
  )
  assert.equal(
    shouldAutoOpenPaymentLink({ interaction_mode: 'wap', pay_url: 'https://pay.example.com/alipay-wap' }),
    true,
  )
  assert.equal(
    shouldAutoOpenPaymentLink({ interaction_mode: 'page', pay_url: 'https://pay.example.com/alipay-page' }),
    true,
  )
  assert.equal(
    shouldAutoOpenPaymentLink({ interaction_mode: 'qr', pay_url: 'https://pay.example.com' }),
    false,
  )
  assert.equal(
    shouldAutoOpenPaymentLink({ interaction_mode: 'redirect', pay_url: '   ' }),
    false,
  )
  assert.equal(
    shouldAutoOpenPaymentLink({ interaction_mode: 'redirect', pay_url: 'https://pay.example.com', fee_policy: 'customer_surcharge' }),
    false,
  )
  assert.equal(
    shouldAutoOpenPaymentLink({ interaction_mode: 'redirect', pay_url: 'https://pay.example.com', fee_policy: 'legacy_customer_surcharge' }),
    false,
  )
  assert.equal(isCustomerSurchargePayment({ fee_policy: 'merchant_absorbed' }), false)
  assert.equal(isCustomerSurchargePayment({ fee_policy: 'legacy_customer_surcharge' }), true)
})

test('automatic cashier navigation uses the current tab to avoid popup blocking', () => {
  assert.equal(resolvePaymentLinkNavigationTarget(true), 'current-tab')
  assert.equal(resolvePaymentLinkNavigationTarget(false), 'new-window')
})

test('recharge balance sync keeps polling while syncing or errored and stops once credited', () => {
  assert.equal(resolveRechargeBalanceSyncState({ status: 'pending' }, null), 'none')
  assert.equal(shouldContinueRechargePolling({ status: 'pending' }, null), true)
  assert.equal(shouldContinueRechargePolling({ status: 'initiated' }, null), true)
  assert.equal(shouldContinueRechargePolling({ status: 'failed' }, null), false)

  assert.equal(
    resolveRechargeBalanceSyncState({ status: 'success' }, { status: 'syncing', attempts: 1 }),
    'syncing',
  )
  assert.equal(
    shouldContinueRechargePolling({ status: 'success' }, { status: 'syncing' }),
    true,
  )

  assert.equal(
    resolveRechargeBalanceSyncState(
      { status: 'success' },
      { status: 'sync_error', last_error: 'upstream timeout' },
    ),
    'sync_error',
  )
  assert.equal(
    shouldContinueRechargePolling(
      { status: 'success' },
      { status: 'sync_error', last_error: 'upstream timeout' },
    ),
    true,
  )

  assert.equal(
    resolveRechargeBalanceSyncState({ status: 'success' }, { status: 'credited' }),
    'credited',
  )
  assert.equal(
    shouldContinueRechargePolling({ status: 'success' }, { status: 'credited' }),
    false,
  )
  // Legacy non-unified wallet without balance_sync keeps existing behavior
  assert.equal(resolveRechargeBalanceSyncState({ status: 'success' }, null), 'credited')
  assert.equal(shouldContinueRechargePolling({ status: 'success' }, null), false)
})
