import test from 'node:test'
import assert from 'node:assert/strict'
import {
  computeSubscriptionRemainingQuota,
  computeSubscriptionUsagePercent,
  formatApiQuotaUsd,
  formatSubscriptionExpiry,
  isSafeAuthorizeUrl,
  parseAccountLinkFragment,
  resolveReturnRoute,
  sanitizeConsoleNextPath,
  sanitizeConsoleTargetPath,
  sanitizeLinkNextPath,
} from '../src/templates/needtoken/lib/apiAccountLinkPolicy.ts'

test('parseAccountLinkFragment extracts valid challenge_id and approval', () => {
  const parsed = parseAccountLinkFragment('#challenge_id=lnk_abc123&approval=v1.payload.sig')
  assert.deepEqual(parsed, {
    challengeId: 'lnk_abc123',
    approval: 'v1.payload.sig',
  })
  assert.equal(parseAccountLinkFragment(''), null)
  assert.equal(parseAccountLinkFragment('#challenge_id=only_id'), null)
  assert.equal(parseAccountLinkFragment('#approval=only_token'), null)
  assert.equal(parseAccountLinkFragment(`#challenge_id=${'a'.repeat(129)}&approval=ok`), null)
})

test('sanitizeLinkNextPath prevents open redirect and maps member paths', () => {
  assert.equal(sanitizeLinkNextPath(''), '/plans')
  assert.equal(sanitizeLinkNextPath('/plans'), '/plans')
  assert.equal(sanitizeLinkNextPath('/api-services'), '/api-services')
  assert.equal(sanitizeLinkNextPath('/models'), '/models')
  assert.equal(sanitizeLinkNextPath('/checkout'), '/checkout')
  assert.equal(sanitizeLinkNextPath('/orders/NT-20261120'), '/orders/NT-20261120')
  assert.equal(sanitizeLinkNextPath('/me'), '/profile')
  assert.equal(sanitizeLinkNextPath('/me/orders'), '/orders')
  assert.equal(sanitizeLinkNextPath('https://evil.example/phish'), '/plans')
  assert.equal(sanitizeLinkNextPath('//evil.example/phish'), '/plans')
  assert.equal(sanitizeLinkNextPath('/orders/../../admin'), '/plans')
})

test('sanitizeConsoleTargetPath and sanitizeConsoleNextPath enforce console SSO allowlists', () => {
  assert.equal(sanitizeConsoleTargetPath(''), '/dashboard')
  assert.equal(sanitizeConsoleTargetPath('/keys'), '/keys')
  assert.equal(sanitizeConsoleTargetPath('/subscriptions'), '/subscriptions')
  assert.equal(sanitizeConsoleTargetPath('/playground'), '/playground')
  assert.equal(sanitizeConsoleTargetPath('/admin'), '/dashboard')
  assert.equal(sanitizeConsoleTargetPath('https://evil.example'), '/dashboard')

  assert.equal(sanitizeConsoleNextPath(''), '/me')
  assert.equal(sanitizeConsoleNextPath('/me'), '/me')
  assert.equal(sanitizeConsoleNextPath('/api-services'), '/api-services')
  assert.equal(sanitizeConsoleNextPath('https://evil.example'), '/me')
})

test('resolveReturnRoute maps safe paths and rejects external URLs', () => {
  assert.equal(resolveReturnRoute('/plans'), '/plans')
  assert.equal(resolveReturnRoute('/checkout'), '/checkout')
  assert.equal(resolveReturnRoute('/profile'), '/me')
  assert.equal(resolveReturnRoute('/me'), '/me')
  assert.equal(resolveReturnRoute('/orders'), '/me/orders')
  assert.equal(resolveReturnRoute('/orders/ORD_99'), '/orders/ORD_99')
  assert.equal(resolveReturnRoute('https://evil.example'), '/me/orders')
  assert.equal(resolveReturnRoute('javascript:alert(1)'), '/me/orders')
})

test('isSafeAuthorizeUrl accepts only credential-free http(s) URLs', () => {
  assert.equal(isSafeAuthorizeUrl('https://api.needtoken.ai/account-link?challenge=v1.a.b'), true)
  assert.equal(isSafeAuthorizeUrl('https://api.needtoken.ai/oauth?redirect=%2Fdashboard#needtoken_ticket=t1'), true)
  assert.equal(isSafeAuthorizeUrl('http://localhost:3000/account-link?challenge=v1.a.b'), true)
  assert.equal(isSafeAuthorizeUrl('javascript:alert(1)'), false)
  assert.equal(isSafeAuthorizeUrl('https://user:pass@api.needtoken.ai/account-link'), false)
  assert.equal(isSafeAuthorizeUrl(''), false)
})

test('API quota and subscription summary formatters handle raw quota units and timestamps', () => {
  assert.equal(formatApiQuotaUsd(0), '$0.00')
  assert.equal(formatApiQuotaUsd(500_000), '$1.00')
  assert.equal(formatApiQuotaUsd(5_000_000), '$10.00')
  assert.equal(formatApiQuotaUsd(125_000), '$0.25')
  assert.equal(formatApiQuotaUsd(1_000), '$0.002')
  assert.equal(formatApiQuotaUsd(-100), '$0.00')

  assert.equal(computeSubscriptionRemainingQuota(10_000_000, 2_500_000), 7_500_000)
  assert.equal(computeSubscriptionRemainingQuota(1_000_000, 1_500_000), 0)
  assert.equal(computeSubscriptionUsagePercent(10_000_000, 2_500_000), 25)
  assert.equal(computeSubscriptionUsagePercent(0, 100), 0)
  assert.equal(computeSubscriptionUsagePercent(1_000, 2_000), 100)

  assert.equal(formatSubscriptionExpiry(0, 'en-US'), null)
  assert.equal(formatSubscriptionExpiry(null, 'en-US'), null)
  const formatted = formatSubscriptionExpiry(1_702_592_000, 'en-US')
  assert.ok(formatted && formatted.includes('2023'))
})
