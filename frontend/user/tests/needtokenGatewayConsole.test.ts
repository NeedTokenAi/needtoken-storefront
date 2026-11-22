import test from 'node:test'
import assert from 'node:assert/strict'
import {
  computeCombinedApiQuota,
  normalizeNeedtokenMemberSection,
  rawQuotaToUsdNumber,
  sumSubscriptionRemainingQuota,
  usdToRawQuota,
} from '../src/templates/needtoken/lib/gatewayConsoleHelpers.ts'

test('normalizeNeedtokenMemberSection maps legacy and new member routes cleanly', () => {
  assert.equal(normalizeNeedtokenMemberSection('overview'), 'overview')
  assert.equal(normalizeNeedtokenMemberSection('keys'), 'keys')
  assert.equal(normalizeNeedtokenMemberSection('api'), 'keys')
  assert.equal(normalizeNeedtokenMemberSection('usage'), 'usage')
  assert.equal(normalizeNeedtokenMemberSection('orders'), 'orders')
  assert.equal(normalizeNeedtokenMemberSection('assets'), 'assets')
  assert.equal(normalizeNeedtokenMemberSection('wallet'), 'assets')
  assert.equal(normalizeNeedtokenMemberSection('giftCard'), 'assets')
  assert.equal(normalizeNeedtokenMemberSection('affiliate'), 'affiliate')
  assert.equal(normalizeNeedtokenMemberSection('reseller'), 'affiliate')
  assert.equal(normalizeNeedtokenMemberSection('security'), 'security')
  assert.equal(normalizeNeedtokenMemberSection('profile'), 'security')
  assert.equal(normalizeNeedtokenMemberSection('unknown'), 'overview')
})

test('quota helpers convert USD safely and combine wallet quota with active subscriptions', () => {
  assert.equal(usdToRawQuota(2.5), 1_250_000)
  assert.equal(usdToRawQuota(-5), 0)
  assert.equal(rawQuotaToUsdNumber(1_000_000), 2)

  const subs = [
    { status: 'active', amount_total: 10_000_000, amount_used: 2_500_000 },
    { status: 'active', amount_total: 5_000_000, amount_used: 6_000_000 },
    { status: 'expired', amount_total: 5_000_000, amount_used: 0 },
  ]
  assert.equal(sumSubscriptionRemainingQuota(subs), 7_500_000)
  assert.equal(computeCombinedApiQuota(1_500_000, subs), 9_000_000)
})
