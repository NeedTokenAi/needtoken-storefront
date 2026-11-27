import test from 'node:test'
import assert from 'node:assert/strict'
import {
  deriveSubscriptionProgressSteps,
  extractPlatformCredential,
  shouldPollFulfillmentOrder,
  validateSubscriptionSession,
} from '../src/utils/subscriptionAuthorization.ts'

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

test('deriveSubscriptionProgressSteps tracks paid, authorized, and delivered lifecycle', () => {
  assert.deepEqual(
    deriveSubscriptionProgressSteps('paid', false).map(s => `${s.key}:${s.state}`),
    ['paid:done', 'authorize:active', 'checkout:pending', 'verified:pending'],
  )
  assert.deepEqual(
    deriveSubscriptionProgressSteps('processing', true).map(s => `${s.key}:${s.state}`),
    ['paid:done', 'authorize:done', 'checkout:active', 'verified:pending'],
  )
  assert.deepEqual(
    deriveSubscriptionProgressSteps('delivered', false).map(s => `${s.key}:${s.state}`),
    ['paid:done', 'authorize:done', 'checkout:done', 'verified:done'],
  )
})

test('R16: server-side fulfillment progress prevents refresh regression and stops polling on terminal states', () => {
  // After page refresh, in-memory `submitted` is false, but server progress `queued` keeps step 3 active
  assert.deepEqual(
    deriveSubscriptionProgressSteps('processing', false, {
      stage: 'queued',
      authorization_submitted: true,
      next_action: 'wait',
      terminal: false,
    }).map(s => `${s.key}:${s.state}`),
    ['paid:done', 'authorize:done', 'checkout:active', 'verified:pending'],
  )
  // Expired credential returns user to authorization step even if local memory wasn't reset
  assert.deepEqual(
    deriveSubscriptionProgressSteps('processing', true, {
      stage: 'credential_expired',
      authorization_submitted: false,
      next_action: 'resubmit_authorization',
      terminal: false,
    }).map(s => `${s.key}:${s.state}`),
    ['paid:done', 'authorize:active', 'checkout:pending', 'verified:pending'],
  )
  // payment_unknown must never appear as verified/completed
  const unkSteps = deriveSubscriptionProgressSteps('processing', false, {
    stage: 'payment_unknown',
    authorization_submitted: true,
    next_action: 'wait',
    terminal: false,
  })
  assert.equal(unkSteps.find(s => s.key === 'verified')?.state, 'pending')

  assert.equal(shouldPollFulfillmentOrder('processing', { stage: 'executing', authorization_submitted: true, next_action: 'wait', terminal: false }), true)
  assert.equal(shouldPollFulfillmentOrder('processing', { stage: 'payment_unknown', authorization_submitted: true, next_action: 'wait', terminal: false }), true)
  assert.equal(shouldPollFulfillmentOrder('delivered', null), false)
  assert.equal(shouldPollFulfillmentOrder('canceled', null), false)
  assert.equal(shouldPollFulfillmentOrder('processing', { stage: 'completed', authorization_submitted: true, next_action: 'none', terminal: true }), false)
})
