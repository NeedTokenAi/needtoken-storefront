import test from 'node:test'
import assert from 'node:assert/strict'
import { authReturnPath } from '../src/utils/authReturnPath.ts'

test('native product SKU, quantity, catalogue and checkout continuations retain their query and fragment', () => {
  for (const path of [
    '/products/chatgpt-plus?sku=1202&quantity=2#purchase',
    '/plans?kind=enterprise&group=team',
    '/checkout?source=quick-buy&sku=1202',
    '/products/%E6%B5%8B%E8%AF%95?sku=1202',
    '/cart?note=space%20in%20query&next=https%3A%2F%2Fexample.test',
    '/',
  ]) assert.equal(authReturnPath(path), path)
  assert.equal(authReturnPath('/products/../checkout?sku=1202'), '/checkout?sku=1202')
})

test('missing, malformed and non-string continuations use the caller’s local fallback', () => {
  for (const value of [undefined, null, '', false, 1, {}, [], ['/checkout'], 'checkout', '#purchase', '?sku=1202', '/products/%ZZ']) {
    assert.equal(authReturnPath(value), '/me/orders', String(value))
    assert.equal(authReturnPath(value, '/plans'), '/plans', String(value))
  }
})

test('absolute URLs, schemes and protocol-relative destinations never become auth continuations', () => {
  for (const value of [
    'https://evil.example/checkout', 'http://evil.example/',
    'https://storefront.invalid/checkout', '//evil.example/checkout', '///evil.example',
    '//storefront.invalid/checkout', 'javascript:alert(1)', 'data:text/html,hello',
    '/a/..//evil.example/checkout', '/..//evil.example', '/%2f%2fevil.example',
  ]) assert.equal(authReturnPath(value), '/me/orders', value)
})

test('backslashes, whitespace and control characters cannot be normalized into another navigation target', () => {
  for (const value of [
    '/\\evil.example', '/path\\..\\checkout', '\\evil.example', '/%5cevil.example',
    '/checkout\n', '/checkout\r?sku=1', '/\t/evil.example', '/\u0000checkout',
    '/checkout\u007f', ' /checkout', '/checkout ', '/%0a/evil.example', '/%09checkout',
  ]) assert.equal(authReturnPath(value, '/'), '/', JSON.stringify(value))
})

test('auth continuations cannot loop into auth through case, encoded separators or dot segments', () => {
  for (const value of [
    '/auth', '/auth?redirect=/checkout', '/auth/login', '/auth/register?redirect=/plans',
    '/AUTH/login', '/Auth/forgot#form', '/products/../auth/login',
    '/products/%2e%2e/auth/login', '/%61uth/login', '/auth%2flogin',
  ]) assert.equal(authReturnPath(value), '/me/orders', value)
  assert.equal(authReturnPath('/author-guide?sku=1202'), '/author-guide?sku=1202')
})

test('accepted results remain a single-slash URL on the storefront origin', () => {
  for (const value of ['/checkout', '/a/../products/pro?sku=1202', '/products/%E6%B5%8B%E8%AF%95', '/cart?next=%2F%2Fevil.example']) {
    const result = authReturnPath(value)
    assert.ok(result.startsWith('/') && !result.startsWith('//'))
    assert.equal(new URL(result, 'https://needtoken.ai').origin, 'https://needtoken.ai')
  }
})
