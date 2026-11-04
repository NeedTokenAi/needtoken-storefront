import { describe, expect, it } from 'vitest'
import { mergeSiteSettingPatch, normalizeNeedtokenResources, normalizeNeedtokenSupport } from './needtokenSettings'

describe('NeedToken public settings', () => {
  it('preserves unrelated site config when saving extensions or the basic form', () => {
    const latest = { brand: { site_name: 'NeedToken.AI' }, scripts: [{ code: 'x' }], languages: ['en-US'], needtoken_support: { enabled: false } }
    expect(mergeSiteSettingPatch(latest, { needtoken_resources: { apple_ids: [] } })).toEqual({ ...latest, needtoken_resources: { apple_ids: [] } })
    const savedBasic = mergeSiteSettingPatch(latest, { brand: { site_name: 'Updated' } })
    expect(savedBasic.needtoken_support).toEqual({ enabled: false })
    expect(savedBasic.languages).toEqual(['en-US'])
    expect(latest.brand.site_name).toBe('NeedToken.AI')
  })
  it('does not invent QR codes, working hours or contact links', () => {
    expect(normalizeNeedtokenSupport(null)).toEqual({ enabled: true, business_email: 'business@needtoken.ai', service_hours: '', contact_url: '', wecom_qr: '', qq_qr: '' })
    expect(normalizeNeedtokenSupport({ enabled: false, business_email: '' })).toMatchObject({ enabled: false, business_email: '' })
  })
  it('keeps draft rows and drops unknown resource fields', () => {
    const entries = normalizeNeedtokenResources({ apple_ids: [{ id: 'draft', enabled: false, title: { 'zh-CN': '示例' }, password: 'private', token: 'private' }] })
    expect(entries[0]).toEqual({ id: 'draft', enabled: false, title: { 'zh-CN': '示例', 'zh-TW': '', 'en-US': '' }, description: { 'zh-CN': '', 'zh-TW': '', 'en-US': '' }, region: '', url: '', updated_at: '' })
  })
})
