import { test } from 'node:test'
import assert from 'node:assert/strict'
import { normalizeNeedtokenSupport, safeSupportImage, safeSupportLink, supportEmailHref } from '../src/templates/needtoken/lib/supportConfig.ts'

test('NeedToken support uses the merchant email until real channels are configured', () => {
  const support = normalizeNeedtokenSupport(undefined)
  assert.equal(support.businessEmail, 'business@needtoken.ai')
  assert.equal(support.enabled, true)
  assert.equal(support.wecomQr, '')
  assert.equal(support.qqQr, '')
  assert.equal(support.serviceHours, '')
  assert.equal(support.contactUrl, '')
  assert.equal(normalizeNeedtokenSupport({ enabled: false }).enabled, false)
})

test('support URLs reject executable schemes, embedded credentials and unsafe upload paths', () => {
  for (const value of ['javascript:alert(1)', 'data:text/html,hello', '//untrusted.test/qr.png', 'https://user:pass@example.com/']) {
    assert.equal(safeSupportLink(value), '')
    assert.equal(safeSupportImage(value), '')
  }
  for (const value of ['/uploads/../config.yml', '/uploads/%2e%2e/private', '/uploads/a%5cb.png', '/uploads/a\\b.png']) assert.equal(safeSupportImage(value), '')
  assert.equal(safeSupportImage('/uploads/images/contact.png'), '/uploads/images/contact.png')
  assert.equal(safeSupportImage('https://example.com/qr.png'), 'https://example.com/qr.png')
  assert.equal(safeSupportLink('https://example.com/help?id=1'), 'https://example.com/help?id=1')
})

test('reseller templates cannot inherit NeedToken contact identities', () => {
  const support = normalizeNeedtokenSupport(undefined, false)
  assert.equal(support.enabled, false)
  assert.equal(support.businessEmail, '')
  assert.equal(support.wecomQr, '')
})

test('email links encode characters that otherwise become mailto query parameters', () => {
  assert.equal(supportEmailHref('service+help@example.com'), 'mailto:service%2Bhelp%40example.com')
  assert.equal(supportEmailHref(''), '')
  assert.equal(normalizeNeedtokenSupport({ business_email: 'bad\n@example.com' }).businessEmail, 'business@needtoken.ai')
})
