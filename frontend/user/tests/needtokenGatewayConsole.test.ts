import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import {
  buildSelectableGroupOptions,
  classifyConsoleModelVendor,
  collectModelsForConsoleGroups,
  computeCombinedApiQuota,
  formatConsoleGroupLabel,
  formatTokenGroupsSummary,
  isNeedtokenMemberConsolePath,
  mapLegacyResellerPath,
  matchesWalletRecordCategory,
  normalizeNeedtokenMemberSection,
  rawQuotaToUsdNumber,
  renderQuickToolTemplate,
  resolveNeedtokenRouteTransitionKey,
  resolveResellerViewFromPath,
  resolveTokenAvailableModels,
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

test('matchesWalletRecordCategory classifies wallet top-ups, gift cards, and spend/refunds consistently', () => {
  assert.equal(matchesWalletRecordCategory('gift_card_redeem', 'all'), true)
  assert.equal(matchesWalletRecordCategory('gift_card_redeem', 'gift_card'), true)
  assert.equal(matchesWalletRecordCategory('gift_card_redeem', 'recharge'), false)
  assert.equal(matchesWalletRecordCategory('recharge', 'recharge'), true)
  assert.equal(matchesWalletRecordCategory('admin_adjust', 'recharge'), true)
  assert.equal(matchesWalletRecordCategory('order_pay', 'spend'), true)
  assert.equal(matchesWalletRecordCategory('order_refund', 'spend'), true)
  assert.equal(matchesWalletRecordCategory('order_pay', 'gift_card'), false)
})

test('member console locales keep Wallet Balance, Plan Quota, and My Wallet labels unified across zh-CN, zh-TW, and en-US', () => {
  const here = dirname(fileURLToPath(import.meta.url))
  const loadLocale = (file: string) => JSON.parse(readFileSync(resolve(here, '../src/i18n/locales', file), 'utf8'))
  const zhCN = loadLocale('zh-CN.json').needtoken.memberConsole
  const zhTW = loadLocale('zh-TW.json').needtoken.memberConsole
  const enUS = loadLocale('en-US.json').needtoken.memberConsole

  assert.equal(zhCN.nav.assets, '我的钱包')
  assert.equal(zhCN.assets.title, '我的钱包')
  assert.equal(zhCN.overview.storeWallet, '钱包余额')
  assert.equal(zhCN.subscriptions.walletQuota, '钱包余额')
  assert.equal(zhCN.overview.apiPlanQuota, '套餐额度')
  assert.equal(zhCN.subscriptions.activeRemaining, '套餐额度')

  assert.equal(zhTW.nav.assets, '我的錢包')
  assert.equal(zhTW.assets.title, '我的錢包')
  assert.equal(zhTW.overview.storeWallet, '錢包餘額')
  assert.equal(zhTW.subscriptions.walletQuota, '錢包餘額')
  assert.equal(zhTW.overview.apiPlanQuota, '套餐額度')
  assert.equal(zhTW.subscriptions.activeRemaining, '套餐額度')

  assert.equal(enUS.nav.assets, 'My Wallet')
  assert.equal(enUS.assets.title, 'My Wallet')
  assert.equal(enUS.overview.storeWallet, 'Wallet Balance')
  assert.equal(enUS.subscriptions.walletQuota, 'Wallet Balance')
  assert.equal(enUS.overview.apiPlanQuota, 'Plan Quota')
  assert.equal(enUS.subscriptions.activeRemaining, 'Plan Quota')

  for (const mc of [zhCN, zhTW, enUS]) {
    assert.ok(mc.assets.redeemGiftCardBtn)
    assert.ok(mc.assets.rechargeBalanceBtn)
    assert.ok(mc.assets.helpWalletRule)
    assert.ok(mc.assets.recordsTitle)
    assert.ok(mc.assets.filterGiftCard)
    assert.ok(mc.quick.quickUse)
    assert.ok(mc.quickUse.title)
    assert.ok(mc.quickUse.copyAiPrompt)
    assert.ok(mc.quickUse.autoAllModelsLabel)
    assert.ok(mc.quickUse.deeplinkNotInstalledHint)
    assert.equal(mc.keys.quickConnect, undefined)
    assert.ok(mc.keys.modelStrategyFollowTitle)
    assert.ok(mc.keys.modelStrategyLockTitle)
  }
})

test('console group helpers format code-channel-ratio labels and hide empty ghost groups', () => {
  const groupMeta = {
    SSVIP: {
      code: 'SSVIP',
      display_name: 'needtoken-pool',
      description: '自用 ChatGPT',
      ratio: 1,
      enabled_channel_count: 1,
      models: ['gpt-5.4', 'gpt-6.2-sol'],
    },
    claude_pro: {
      code: 'claude_pro',
      display_name: 'Anthropic Pool',
      ratio: 1.5,
      enabled_channel_count: 2,
      models: ['claude-sonnet-4-6'],
    },
    default: {
      code: 'default',
      description: 'Default group',
      ratio: 1,
      enabled_channel_count: 0,
      models: [],
    },
  }
  const usableGroups = { SSVIP: '自用 ChatGPT', claude_pro: 'Claude', default: 'Default group' }

  assert.equal(formatConsoleGroupLabel('SSVIP', groupMeta, usableGroups), 'SSVIP - needtoken-pool - ×1.00')
  assert.equal(formatConsoleGroupLabel('claude_pro', groupMeta, usableGroups), 'claude_pro - Anthropic Pool - ×1.50')
  assert.equal(formatConsoleGroupLabel('default', groupMeta, usableGroups), 'default - ×1.00')

  assert.equal(
    formatTokenGroupsSummary({ group: 'auto', auto_groups: ['SSVIP', 'claude_pro'] }, 'default', groupMeta, usableGroups),
    'SSVIP - needtoken-pool - ×1.00 / claude_pro - Anthropic Pool - ×1.50',
  )

  const selectable = buildSelectableGroupOptions({
    usableGroups,
    groupMeta,
    defaultGroup: 'default',
  })
  assert.deepEqual(selectable.map(o => o.code), ['claude_pro', 'SSVIP'])

  const editingWithLegacy = buildSelectableGroupOptions({
    usableGroups,
    groupMeta,
    defaultGroup: 'default',
    includeCodes: ['default'],
  })
  assert.equal(editingWithLegacy.some(o => o.code === 'default' && !o.hasActiveChannels), true)
})

test('dynamic model aggregation and token model resolution respect follow vs lock policies', () => {
  const groupMeta = {
    SSVIP: { code: 'SSVIP', enabled_channel_count: 1, models: ['gpt-5.4', 'gpt-6.2-sol'] },
    claude_pro: { code: 'claude_pro', enabled_channel_count: 1, models: ['claude-sonnet-4-6'] },
  }

  const combined = collectModelsForConsoleGroups(['SSVIP', 'claude_pro'], groupMeta, [])
  assert.deepEqual(combined.map(c => c.model), ['claude-sonnet-4-6', 'gpt-5.4', 'gpt-6.2-sol'])
  assert.equal(classifyConsoleModelVendor('gpt-6.2-sol'), 'openai')
  assert.equal(classifyConsoleModelVendor('claude-sonnet-4-6'), 'claude')
  assert.equal(classifyConsoleModelVendor('gemini-2.5-pro'), 'gemini')

  const followModels = resolveTokenAvailableModels({
    token: { group: 'auto', auto_groups: ['SSVIP', 'claude_pro'], model_limits_enabled: false, model_limits: '' },
    groupMeta,
  })
  assert.deepEqual(followModels, ['claude-sonnet-4-6', 'gpt-5.4', 'gpt-6.2-sol'])

  const lockedModels = resolveTokenAvailableModels({
    token: { group: 'SSVIP', model_limits_enabled: true, model_limits: 'gpt-6.2-sol' },
    groupMeta,
  })
  assert.deepEqual(lockedModels, ['gpt-6.2-sol'])
})

test('renderQuickToolTemplate injects member credentials, dynamic model lists, and supports URL encoding', () => {
  const vars = {
    apiKey: 'sk-test+abc/123',
    baseUrl: 'https://api.needtoken.ai',
    baseUrlV1: 'https://api.needtoken.ai/v1',
    model: 'gpt-6.2-sol',
    allModels: ['gpt-5.4', 'gpt-6.2-sol'],
    tokenName: 'My Key',
    group: 'SSVIP - needtoken-pool - ×1.00',
  }

  assert.equal(
    renderQuickToolTemplate('endpoint="{{MODELS_ENDPOINT}}" models={{ALL_MODELS_JSON}} key="{{API_KEY}}"', vars),
    'endpoint="https://api.needtoken.ai/v1/models" models=["gpt-5.4","gpt-6.2-sol"] key="sk-test+abc/123"',
  )

  const openCodeMap = renderQuickToolTemplate('{"models": {{ALL_MODELS_MAP_JSON}}}', vars)
  assert.deepEqual(JSON.parse(openCodeMap), {
    models: {
      'gpt-5.4': { name: 'gpt-5.4' },
      'gpt-6.2-sol': { name: 'gpt-6.2-sol' },
    },
  })

  const deepLink = renderQuickToolTemplate(
    'ccswitch://v1/import?endpoint={{BASE_URL}}&apiKey={{API_KEY}}',
    vars,
    true,
  )
  assert.equal(
    deepLink,
    'ccswitch://v1/import?endpoint=https%3A%2F%2Fapi.needtoken.ai&apiKey=sk-test%2Babc%2F123',
  )
})
