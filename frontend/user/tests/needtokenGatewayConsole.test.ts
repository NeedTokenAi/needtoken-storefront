import test from 'node:test'
import assert from 'node:assert/strict'
import {
  computeCombinedApiQuota,
  isNeedtokenMemberConsolePath,
  mapLegacyResellerPath,
  normalizeNeedtokenMemberSection,
  rawQuotaToUsdNumber,
  resolveNeedtokenRouteTransitionKey,
  resolveResellerViewFromPath,
  sumSubscriptionRemainingQuota,
  usdToRawQuota,
} from '../src/templates/needtoken/lib/gatewayConsoleHelpers.ts'

test('normalizeNeedtokenMemberSection maps legacy and new member routes cleanly', () => {
  assert.equal(normalizeNeedtokenMemberSection('overview'), 'overview')
  assert.equal(normalizeNeedtokenMemberSection('keys'), 'keys')
  assert.equal(normalizeNeedtokenMemberSection('api'), 'keys')
  assert.equal(normalizeNeedtokenMemberSection('models'), 'models')
  assert.equal(normalizeNeedtokenMemberSection('playground'), 'playground')
  assert.equal(normalizeNeedtokenMemberSection('usage'), 'usage')
  assert.equal(normalizeNeedtokenMemberSection('plans'), 'plans')
  assert.equal(normalizeNeedtokenMemberSection('subscriptions'), 'plans')
  assert.equal(normalizeNeedtokenMemberSection('orders'), 'orders')
  assert.equal(normalizeNeedtokenMemberSection('assets'), 'assets')
  assert.equal(normalizeNeedtokenMemberSection('wallet'), 'assets')
  assert.equal(normalizeNeedtokenMemberSection('giftCard'), 'assets')
  assert.equal(normalizeNeedtokenMemberSection('affiliate'), 'affiliate')
  assert.equal(normalizeNeedtokenMemberSection('reseller'), 'reseller')
  assert.equal(normalizeNeedtokenMemberSection('security'), 'security')
  assert.equal(normalizeNeedtokenMemberSection('profile'), 'security')
  assert.equal(normalizeNeedtokenMemberSection('unknown'), 'overview')
})

test('reseller path helpers redirect legacy routes into member center and resolve nested tabs', () => {
  assert.equal(mapLegacyResellerPath('/reseller'), '/me/reseller')
  assert.equal(mapLegacyResellerPath('/reseller/products'), '/me/reseller/products')
  assert.equal(mapLegacyResellerPath('/reseller/orders/ORD-1001'), '/me/reseller/orders/ORD-1001')

  assert.deepEqual(resolveResellerViewFromPath('/me/reseller'), {
    mainTab: 'dashboard',
    financeSubTab: 'finance',
    siteSubTab: 'domains',
    isOrderDetail: false,
  })
  assert.deepEqual(resolveResellerViewFromPath('/me/reseller/products'), {
    mainTab: 'products',
    financeSubTab: 'finance',
    siteSubTab: 'domains',
    isOrderDetail: false,
  })
  assert.deepEqual(resolveResellerViewFromPath('/me/reseller/orders/ORD-1001'), {
    mainTab: 'orders',
    financeSubTab: 'finance',
    siteSubTab: 'domains',
    isOrderDetail: true,
  })
  assert.deepEqual(resolveResellerViewFromPath('/me/reseller/withdraws'), {
    mainTab: 'finance',
    financeSubTab: 'withdraws',
    siteSubTab: 'domains',
    isOrderDetail: false,
  })
  assert.deepEqual(resolveResellerViewFromPath('/me/reseller/site'), {
    mainTab: 'site',
    financeSubTab: 'finance',
    siteSubTab: 'site',
    isOrderDetail: false,
  })
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

test('member console paths share one stable shell transition key', () => {
  assert.equal(isNeedtokenMemberConsolePath('/me'), true)
  assert.equal(isNeedtokenMemberConsolePath('/me/keys'), true)
  assert.equal(isNeedtokenMemberConsolePath('/me/reseller/orders/ORD-1'), true)
  assert.equal(isNeedtokenMemberConsolePath('/'), false)
  assert.equal(isNeedtokenMemberConsolePath('/media'), false)
  assert.equal(isNeedtokenMemberConsolePath('/products/chatgpt'), false)
  assert.equal(isNeedtokenMemberConsolePath('/reseller'), false)

  assert.equal(resolveNeedtokenRouteTransitionKey('/me'), '/me')
  assert.equal(resolveNeedtokenRouteTransitionKey('/me/keys'), '/me')
  assert.equal(resolveNeedtokenRouteTransitionKey('/me/orders'), '/me')
  assert.equal(resolveNeedtokenRouteTransitionKey('/me/reseller/products'), '/me')
  assert.equal(resolveNeedtokenRouteTransitionKey('/products/a'), '/products/a')
  assert.equal(resolveNeedtokenRouteTransitionKey('/products/b'), '/products/b')
  assert.equal(resolveNeedtokenRouteTransitionKey('/media'), '/media')
})
