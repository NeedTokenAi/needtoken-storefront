import { describe, expect, it } from 'vitest'
import { changedDefaultSKUDisplayMeta, isValidOfficialUSDPrice, isValidSKUFeatureLines, normalizeSKUDisplayMeta } from './skuDisplayMeta'

describe('SKU display metadata editing', () => {
  it('copies all translated presentation fields without leaking internal metadata', () => {
    const source = { label: { 'zh-CN': '年度', 'en-US': 'Annual' }, title: { 'zh-CN': 'Plus 年度' }, description: { 'zh-CN': '整年使用' }, feature_lines: { 'zh-CN': '功能一\n功能二' }, price_suffix: { 'zh-CN': '/年' }, official_price_usd: '200.00', source_token: 'private' }
    const meta = normalizeSKUDisplayMeta(source)
    expect(meta.official_price_usd).toBe('200.00')
    expect(meta.feature_lines['zh-CN']).toBe('功能一\n功能二')
    expect(meta).not.toHaveProperty('source_token')
    expect(meta.label['zh-TW']).toBe('')
    meta.label['zh-CN'] = '修改'
    expect(source.label['zh-CN']).toBe('年度')
  })
  it('omits unchanged DEFAULT display metadata and keeps explicit clearing', () => {
    const original = normalizeSKUDisplayMeta({ title: { 'zh-CN': '旧标题' } })
    expect(changedDefaultSKUDisplayMeta(normalizeSKUDisplayMeta(original), original)).toBeUndefined()
    expect(changedDefaultSKUDisplayMeta(normalizeSKUDisplayMeta(null), original)?.title['zh-CN']).toBe('')
    expect(changedDefaultSKUDisplayMeta(normalizeSKUDisplayMeta({ official_price_usd: '20' }), normalizeSKUDisplayMeta(null))?.official_price_usd).toBe('20')
  })
  it('prefills legacy USD references only before display metadata has been configured', () => {
    expect(normalizeSKUDisplayMeta(null, { official_usd: '500', source: 'private' }).official_price_usd).toBe('500')
    expect(normalizeSKUDisplayMeta({ official_price_usd: '' }, { official_usd: '500' }).official_price_usd).toBe('')
    expect(normalizeSKUDisplayMeta(null, { official_usd: 'NaN' }).official_price_usd).toBe('')
  })
  it('clears inherited variant presentation when collapsing to the blank single editor', () => {
    const blank = normalizeSKUDisplayMeta(null)
    expect(changedDefaultSKUDisplayMeta(blank, blank)).toBeUndefined()
    const cleared = changedDefaultSKUDisplayMeta(blank, blank, true)
    expect(cleared).toEqual(blank)
    expect(cleared?.official_price_usd).toBe('')
    const edited = normalizeSKUDisplayMeta({ title: { 'zh-CN': '新的单规格标题' } })
    expect(changedDefaultSKUDisplayMeta(edited, blank, true)?.title['zh-CN']).toBe('新的单规格标题')
  })
  it('validates feature count and Unicode line length across all languages', () => {
    expect(isValidSKUFeatureLines({ 'zh-CN': '功能一\n\n功能二', 'en-US': '🙂'.repeat(160) })).toBe(true)
    expect(isValidSKUFeatureLines({ 'zh-CN': Array(9).fill('功能').join('\n') })).toBe(false)
    expect(isValidSKUFeatureLines({ 'zh-TW': '🙂'.repeat(161) })).toBe(false)
  })
  it('accepts a blank reference or an exact positive two-decimal USD value', () => {
    for (const value of ['', '20', '0.01', '500.00']) expect(isValidOfficialUSDPrice(value)).toBe(true)
    for (const value of ['0', '-1', '5e2', '5.123', 'NaN', '500 USD', '100000000']) expect(isValidOfficialUSDPrice(value)).toBe(false)
  })
})
