import test from 'node:test'
import assert from 'node:assert/strict'
import {
  hasWalletRechargeConfig,
  shouldShowCheckoutWalletBox,
  shouldShowWalletSidebarTab,
} from '../src/templates/needtoken/lib/walletVisibility.ts'

test('needtoken template hides redundant wallet sidebar tab when wallet recharge is unconfigured', () => {
  assert.equal(hasWalletRechargeConfig({}), false)
  assert.equal(shouldShowWalletSidebarTab('needtoken', {}), false)
  assert.equal(shouldShowWalletSidebarTab('needtoken', { wallet_recharge_channel_ids: [] }), false)
  assert.equal(shouldShowWalletSidebarTab('needtoken', { wallet_recharge_channel_ids: [3] }), true)
  assert.equal(shouldShowWalletSidebarTab('needtoken', { wallet_only_payment: true }), true)
  assert.equal(shouldShowWalletSidebarTab('classic', {}), true)
})

test('checkout wallet balance box stays hidden for zero balance unless wallet recharge is enabled', () => {
  assert.equal(shouldShowCheckoutWalletBox({}, '0.00'), false)
  assert.equal(shouldShowCheckoutWalletBox({}, ''), false)
  assert.equal(shouldShowCheckoutWalletBox({}, '12.50'), true)
  assert.equal(shouldShowCheckoutWalletBox({ wallet_recharge_channel_ids: [1] }, '0.00'), true)
  assert.equal(shouldShowCheckoutWalletBox({ wallet_only_payment: true }, '0.00'), true)
})
