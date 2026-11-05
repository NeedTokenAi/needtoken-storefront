import test from 'node:test'
import assert from 'node:assert/strict'
import { apiPlanAudience, apiPlanMemberLoginPath, filterAPIPlanProducts, resolveAPIPlanScope } from '../src/templates/needtoken/lib/apiPlanCatalog.ts'

const categories = [
  { id: 1, slug: 'plans', parent_id: 0 }, { id: 2, slug: 'plans-personal', parent_id: 1 },
  { id: 3, slug: 'plans-enterprise', parent_id: 1 }, { id: 4, slug: 'team-dedicated', parent_id: 3 },
  { id: 10, slug: 'recharge', parent_id: 0 }, { id: 11, slug: 'chatgpt-recharge', parent_id: 10 },
]

test('API plans stay in the assigned personal or enterprise category branch', () => {
  assert.deepEqual(resolveAPIPlanScope(categories, 'personal').categoryIds, [2])
  assert.deepEqual(resolveAPIPlanScope(categories, 'enterprise').categoryIds, [3, 4])
  const rows = [2, 3, 4, 10].map(id => ({ id, slug: String(id), category_id: id }))
  assert.deepEqual(filterAPIPlanProducts(rows, resolveAPIPlanScope(categories, 'enterprise')).map(row => row.id), [3, 4])
})

test('missing API roots or audience categories never expand to every product', () => {
  assert.deepEqual(resolveAPIPlanScope(categories.filter(row => row.id !== 1), 'personal').categoryIds, [])
  assert.deepEqual(resolveAPIPlanScope(categories.filter(row => row.id !== 2), 'personal').categoryIds, [])
  assert.equal(apiPlanAudience('unexpected'), 'personal')
  assert.equal(apiPlanAudience(['enterprise']), 'personal')
  assert.equal(apiPlanAudience('enterprise'), 'enterprise')
})

test('member login is storefront-local and returns to the selected SKU', () => {
  const path = apiPlanMemberLoginPath('api-pro', 27)
  assert.equal(path, '/auth/login?redirect=%2Fproducts%2Fapi-pro%3Fsku%3D27')
  assert.ok(!path.includes('api.needtoken.ai'))
})
