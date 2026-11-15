import test from 'node:test'
import assert from 'node:assert/strict'
import { extractPlatformCredential, validateSubscriptionSession } from '../src/utils/subscriptionAuthorization.ts'

test('subscription session rejects expired, redirected or malformed credential endpoints', () => {
  const now = Date.parse('2026-11-15T08:00:00Z')
  const session = { authorization_id: 'auth_order_1', consent_ref: 'auth_order_1', session_token: 'a'.repeat(43), credential_url: 'https://integration.example.test/api/authorization/v1/credential', expires_at: '2026-11-15T08:20:00Z', execution_scope: 'inspection_only' }
  assert.ok(validateSubscriptionSession(session, now))
  for (const credential_url of ['http://integration.example.test/api/authorization/v1/credential', 'https://user:secret@integration.example.test/api/authorization/v1/credential', 'https://integration.example.test/api/authorization/v1/credential?token=secret', 'https://integration.example.test/other']) {
    assert.equal(validateSubscriptionSession({ ...session, credential_url }, now), null)
  }
  assert.equal(validateSubscriptionSession({ ...session, execution_scope: 'subscription_checkout' }, now), null)
  assert.equal(validateSubscriptionSession({ ...session, expires_at: '2026-11-15T09:00:00Z' }, now), null)
  assert.equal(validateSubscriptionSession(session, Date.parse('2026-11-15T08:21:00Z')), null)
})

test('only platform accessToken is extracted from supplied session JSON', () => {
  assert.equal(extractPlatformCredential('{"accessToken":"short-lived-token","user":{"email":"private@example.test"}}'), 'short-lived-token')
  assert.equal(extractPlatformCredential('short-lived-token'), 'short-lived-token')
  assert.equal(extractPlatformCredential('{"user":{"email":"private@example.test"}}'), null)
  assert.equal(extractPlatformCredential('abc'), null)
  assert.equal(extractPlatformCredential('token\nwith-newline'), null)
})
