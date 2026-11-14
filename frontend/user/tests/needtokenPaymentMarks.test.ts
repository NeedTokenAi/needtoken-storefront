import test from 'node:test'
import assert from 'node:assert/strict'
import { paymentMarks, paymentNetworks } from '../src/templates/needtoken/lib/paymentMarks.ts'

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
