import test from 'node:test'
import assert from 'node:assert/strict'
import {
  canRenderResellerConsoleModule,
  getResellerConsoleState,
  resellerConsolePath,
  resolveResellerConsoleModule,
} from '../src/utils/resellerConsole.ts'

test('reseller console route gate blocks active-only modules until profile is active', () => {
  const pendingState = getResellerConsoleState({
    opened: true,
    can_apply: false,
    profile: { status: 'pending_review' },
    domains: [],
  } as any)
  const applyAllowedState = getResellerConsoleState({
    opened: false,
    can_apply: true,
    profile: null,
    domains: [],
  } as any)
  const activeState = getResellerConsoleState({
    opened: true,
    can_apply: false,
    profile: { status: 'active' },
    domains: [],
  } as any)

  assert.equal(pendingState.modules.apply.enabled, false)
  assert.equal(applyAllowedState.modules.apply.enabled, true)
  assert.equal(activeState.modules.apply.enabled, false)
  assert.equal(resolveResellerConsoleModule('/reseller/products'), 'products')
  assert.equal(canRenderResellerConsoleModule('/reseller/apply', pendingState), true)
  assert.equal(canRenderResellerConsoleModule('/reseller', pendingState), true)
  assert.equal(canRenderResellerConsoleModule('/reseller/products', pendingState), false)
  assert.equal(canRenderResellerConsoleModule('/reseller/products', activeState), true)

  assert.equal(resellerConsolePath('/me/reseller'), '/me/reseller')
  assert.equal(resellerConsolePath('/me/reseller/orders', 'orders/ORD-1'), '/me/reseller/orders/ORD-1')
  assert.equal(resellerConsolePath('/reseller/finance', 'withdraws'), '/reseller/withdraws')
})
