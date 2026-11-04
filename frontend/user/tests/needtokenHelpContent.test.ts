import { test } from 'node:test'
import assert from 'node:assert/strict'
import { faqCategories, publicAppleResources, publicResourceUrl } from '../src/templates/needtoken/lib/helpContent.ts'

test('FAQ scope requires the dedicated root and keeps unrelated article categories out', () => {
  const categories = [
    { id: 1, parent_id: 0, slug: 'guides', name: {} },
    { id: 2, parent_id: 0, slug: 'faq', name: {} },
    { id: 3, parent_id: 2, slug: 'faq-payments', name: {}, sort_order: 2 },
    { id: 4, parent_id: 2, slug: 'faq-api', name: {}, sort_order: 1 },
    { id: 5, parent_id: 1, slug: 'other', name: {} },
  ]
  assert.deepEqual(faqCategories(categories).children.map((item) => item.id), [4, 3])
  assert.equal(faqCategories(categories).root?.id, 2)
  assert.deepEqual(faqCategories(categories.filter((item) => item.id !== 2)), { children: [] })
  assert.deepEqual(faqCategories(null), { children: [] })
})

test('public resources reject unsafe schemes and embedded credentials', () => {
  for (const url of ['javascript:alert(1)', 'data:text/html,test', '//example.com', '/uploads/id.txt', 'https://user:secret@example.com']) assert.equal(publicResourceUrl(url), '')
  assert.equal(publicResourceUrl('https://example.com/shared-guide'), 'https://example.com/shared-guide')
})

test('free ID publication keeps only explicitly enabled unique entries with valid resource links', () => {
  const base = { id: 'public-guide', title: { 'zh-CN': 'Guide' }, description: {}, region: 'US', url: 'https://example.com/guide', enabled: true }
  const resources = publicAppleResources([base, { ...base }, { ...base, id: 'disabled', enabled: false }, { ...base, id: 'draft', enabled: undefined }, { ...base, id: 'unsafe', url: 'javascript:bad' }])
  assert.equal(resources.length, 1)
  assert.equal(resources[0]?.id, 'public-guide')
  assert.deepEqual(publicAppleResources(undefined), [])
})
