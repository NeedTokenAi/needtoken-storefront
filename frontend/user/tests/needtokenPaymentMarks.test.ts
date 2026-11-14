import test from 'node:test'
import assert from 'node:assert/strict'
import { paymentDisplayState, publicPaymentMethods, paymentMarks, paymentNetworks } from '../src/templates/needtoken/lib/paymentMarks.ts'

test('payment logos require recognized public channel metadata, not display-name guesses', () => {
  for (const rows of [null, undefined, {}, [], [{ name: 'Alipay UnionPay crypto credit card' }]]) assert.deepEqual(paymentMarks(rows), [])
  assert.deepEqual(paymentMarks([{ channel_type: 'unknown' }, { channel_type: 'balance' }, null]), [])
})

test('aliases are deduplicated and disabled channels are not advertised', () => {
  assert.deepEqual(paymentMarks([
    { channel_type: 'wxpay' }, { channel_type: 'WECHAT' },
    { channel_type: 'stripe' }, { channel_type: 'alipay', is_active: false },
  ]), ['wechat', 'card'])
})

test('configured UnionPay, crypto, and cards stay distinct', () => {
  assert.deepEqual(paymentMarks([
    { channel_type: 'unionpay' }, { channel_type: 'usdt-trc20' },
    { channel_type: 'usdc-trc20' }, { channel_type: 'stripe' }, { channel_type: 'alipay' },
  ]), ['alipay', 'unionpay', 'usdt', 'crypto', 'card'])
})

test('crypto gateway names do not imply USDT or a supported network', () => {
  const channels = [{ channel_type: 'bepusdt', name: 'USDT BEP20 · TRC20' }, { channel_type: 'epusdt' }]
  assert.deepEqual(paymentMarks(channels), ['crypto'])
  assert.deepEqual(paymentNetworks(channels), [])
  assert.deepEqual(paymentNetworks([{ name: 'USDT TRC20' }, { channel_type: 'usdt' }]), [])
})

test('network labels require explicit public channel types and exclude disabled channels', () => {
  assert.deepEqual(paymentNetworks([
    { channel_type: 'usdt-trc20' }, { channel_type: 'USDC-TRC20' },
    { channel_type: 'usdt-bep20', is_active: false },
  ]), ['trc20'])
  assert.deepEqual(paymentNetworks([{ channel_type: 'usdt-bep20' }, { channel_type: 'usdt-trc20' }]), ['bep20', 'trc20'])
})

test('actual public config DTO and DujiaoPay token IDs render without an is_active field', () => {
  const channels = [
    { id: 1, name: 'AliPay', provider_type: 'official', channel_type: 'alipay', interaction_mode: 'qr', min_amount: '0', max_amount: '0', hide_amount_out_range: false },
    { id: 2, name: 'Tether on Tron', provider_type: 'dujiaopay', channel_type: 'tron-usdt', interaction_mode: 'redirect', min_amount: '0', max_amount: '0', hide_amount_out_range: false },
    { id: 3, name: 'Tether on BSC', provider_type: 'dujiaopay', channel_type: 'bsc-usdt', interaction_mode: 'redirect', min_amount: '0', max_amount: '0', hide_amount_out_range: false },
  ]
  assert.deepEqual(paymentMarks(channels), ['alipay', 'usdt'])
  assert.deepEqual(paymentNetworks(channels), ['bep20', 'trc20'])
  assert.equal(paymentDisplayState({ payment_channels: channels }), 'ready')
  assert.deepEqual(paymentMarks([{ channel_type: 'base-usdc' }]), ['crypto'])
  assert.deepEqual(paymentNetworks([{ channel_type: 'base-usdc' }]), [])
})

test('an unfamiliar real channel keeps its name without inventing a payment brand', () => {
  const channels = [
    { id: 9, name: 'Custom gateway', channel_type: 'future-token' },
    { id: 10, name: 'Custom gateway', channel_type: 'future-token' },
    { name: 'Alipay USDT', channel_type: 'unknown' },
    { id: 11, name: 'Disabled channel', channel_type: 'unknown', is_active: false },
    { id: 12, name: 'Wallet', channel_type: 'balance' },
  ]
  assert.deepEqual(publicPaymentMethods(channels), [{ key: 'channel-9', mark: null, label: 'Custom gateway' }])
  assert.deepEqual(paymentMarks(channels), [])
  assert.deepEqual(paymentNetworks(channels), [])
  assert.equal(paymentDisplayState({ payment_channels: channels }), 'ready')
})

test('missing, empty, loading and wallet-only public configuration remain distinguishable', () => {
  for (const config of [null, undefined, {}, { payment_channels: null }, { payment_channels: {} }]) assert.equal(paymentDisplayState(config), 'unavailable')
  assert.equal(paymentDisplayState(null, true), 'loading')
  assert.equal(paymentDisplayState({ payment_channels: [] }), 'empty')
  assert.equal(paymentDisplayState({ wallet_only_payment: true, payment_channels: [{ channel_type: 'alipay' }] }), 'wallet')
  assert.equal(paymentDisplayState({ payment_channels: [{ id: 1, name: 'Future method', channel_type: 'future' }] }), 'ready')
  assert.equal(paymentDisplayState({ payment_channels: [{ channel_type: 'alipay' }] }, true), 'loading')
})
