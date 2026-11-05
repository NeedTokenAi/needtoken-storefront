import assert from 'node:assert/strict'
import test from 'node:test'
import { shouldBrandNeedtokenStartup, storefrontStartupPhase } from '../src/utils/storefrontStartup.ts'

test('boot skeleton hands over as soon as configuration and route are ready', () => {
  assert.equal(storefrontStartupPhase({ hasConfig: false, routeReady: false, routeFailed: false }), 'pending')
  assert.equal(storefrontStartupPhase({ hasConfig: true, routeReady: false, routeFailed: false }), 'pending')
  assert.equal(storefrontStartupPhase({ hasConfig: true, routeReady: true, routeFailed: false }), 'ready')
})
test('failed startup is retryable instead of an indefinite loading state', () => {
  assert.equal(storefrontStartupPhase({ hasConfig: false, routeReady: true, routeFailed: false }), 'error')
  assert.equal(storefrontStartupPhase({ hasConfig: true, routeReady: false, routeFailed: true }), 'error')
})
test('unknown and reseller domains do not inherit NeedToken boot branding from template alone', () => {
  assert.equal(shouldBrandNeedtokenStartup(null), false)
  assert.equal(shouldBrandNeedtokenStartup('needtoken'), false)
  assert.equal(shouldBrandNeedtokenStartup('classic', true), false)
  assert.equal(shouldBrandNeedtokenStartup('vault', true), false)
  assert.equal(shouldBrandNeedtokenStartup('needtoken', true), true)
})
test('first visits to the exact public main hosts show the NeedToken logo without cached configuration', () => {
  assert.equal(shouldBrandNeedtokenStartup(null, false, 'needtoken.ai'), true)
  assert.equal(shouldBrandNeedtokenStartup(null, false, 'www.needtoken.ai'), true)
})
test('host branding does not extend to subdomains, suffix matches, or reseller hosts', () => {
  for (const hostname of ['ops.needtoken.ai', 'api.needtoken.ai', 'store.needtoken.ai', 'needtoken.ai.example.com', 'example.needtoken.ai', 'needtoken.app', 'needtoken.io', 'reseller.example.com', 'localhost', '']) {
    assert.equal(shouldBrandNeedtokenStartup('needtoken', false, hostname), false, hostname)
  }
})
