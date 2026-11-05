export const needtokenLanguages = ['zh-CN', 'zh-TW', 'en-US'] as const
export type NeedtokenLanguage = typeof needtokenLanguages[number]
export type NeedtokenLocalized = Record<NeedtokenLanguage, string>
export interface NeedtokenSupportSettings {
  enabled: boolean
  business_email: string
  service_hours: string
  contact_url: string
  wecom_qr: string
  qq_qr: string
}
export interface NeedtokenPublicResource {
  id: string
  title: NeedtokenLocalized
  region: string
  description: NeedtokenLocalized
  url: string
  enabled: boolean
  updated_at: string
}
const record = (raw: unknown): Record<string, unknown> => raw && typeof raw === 'object' && !Array.isArray(raw) ? raw as Record<string, unknown> : {}
const text = (raw: unknown) => typeof raw === 'string' ? raw : ''
const localized = (raw: unknown): NeedtokenLocalized => {
  const value = record(raw)
  return { 'zh-CN': text(value['zh-CN']), 'zh-TW': text(value['zh-TW']), 'en-US': text(value['en-US']) }
}
export function normalizeNeedtokenSupport(raw: unknown): NeedtokenSupportSettings {
  const value = record(raw)
  return {
    enabled: value.enabled !== false,
    business_email: typeof value.business_email === 'string' ? value.business_email : 'business@needtoken.ai',
    service_hours: text(value.service_hours),
    contact_url: text(value.contact_url),
    wecom_qr: text(value.wecom_qr),
    qq_qr: text(value.qq_qr),
  }
}
export function normalizeNeedtokenResources(raw: unknown): NeedtokenPublicResource[] {
  const entries = record(raw).apple_ids
  if (!Array.isArray(entries)) return []
  return entries.map((rawItem) => {
    const item = record(rawItem)
    return {
      id: text(item.id), title: localized(item.title), region: text(item.region),
      description: localized(item.description), url: text(item.url), enabled: item.enabled === true,
      updated_at: text(item.updated_at),
    }
  })
}
// The generic settings API replaces the complete value. Preserve every field
// fetched immediately before saving, including extensions from other modules.
export function mergeSiteSettingPatch(latest: unknown, patch: Record<string, unknown>): Record<string, unknown> {
  return { ...record(latest), ...patch }
}
