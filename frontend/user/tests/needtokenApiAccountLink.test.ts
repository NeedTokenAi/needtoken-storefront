import test from 'node:test'
import assert from 'node:assert/strict'
import {
  isSafeAuthorizeUrl,
  parseAccountLinkFragment,
  resolveReturnRoute,
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
  assert.equal(sanitizeLinkNextPath('/checkout'), '/checkout')
  assert.equal(sanitizeLinkNextPath('/orders/NT-20261120'), '/orders/NT-20261120')
  assert.equal(sanitizeLinkNextPath('/me'), '/profile')
  assert.equal(sanitizeLinkNextPath('/me/orders'), '/orders')
  assert.equal(sanitizeLinkNextPath('https://evil.example/phish'), '/plans')
  assert.equal(sanitizeLinkNextPath('//evil.example/phish'), '/plans')
  assert.equal(sanitizeLinkNextPath('/orders/../../admin'), '/plans')
})

test('resolveReturnRoute maps safe paths and rejects external URLs', () => {
  assert.equal(resolveReturnRoute('/plans'), '/plans')
  assert.equal(resolveReturnRoute('/checkout'), '/checkout')
  assert.equal(resolveReturnRoute('/profile'), '/me')
  assert.equal(resolveReturnRoute('/orders'), '/me/orders')
  assert.equal(resolveReturnRoute('/orders/ORD_99'), '/orders/ORD_99')
  assert.equal(resolveReturnRoute('https://evil.example'), '/me/orders')
  assert.equal(resolveReturnRoute('javascript:alert(1)'), '/me/orders')
})

test('isSafeAuthorizeUrl accepts only credential-free http(s) URLs', () => {
  assert.equal(isSafeAuthorizeUrl('https://api.needtoken.ai/account-link?challenge=v1.a.b'), true)
  assert.equal(isSafeAuthorizeUrl('http://localhost:3000/account-link?challenge=v1.a.b'), true)
  assert.equal(isSafeAuthorizeUrl('javascript:alert(1)'), false)
  assert.equal(isSafeAuthorizeUrl('https://user:pass@api.needtoken.ai/account-link'), false)
  assert.equal(isSafeAuthorizeUrl(''), false)
})
