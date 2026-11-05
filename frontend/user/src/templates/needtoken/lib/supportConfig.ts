export interface NeedtokenSupport {
  enabled: boolean
  businessEmail: string
  serviceHours: string
  contactUrl: string
  wecomQr: string
  qqQr: string
}

export function safeSupportLink(value: unknown): string {
  if (typeof value !== 'string') return ''
  const text = value.trim()
  if (!text || text.length > 2048 || /[\u0000-\u0020\u007f]/.test(text)) return ''
  try {
    const url = new URL(text)
    if (!['https:', 'http:'].includes(url.protocol) || url.username || url.password) return ''
    return url.href
  } catch {
    return ''
  }
}

export function safeSupportImage(value: unknown): string {
  if (typeof value !== 'string') return ''
  const text = value.trim()
  if (text.length > 2048) return ''
  if (text.startsWith('/uploads/') && !/[\\\u0000-\u0020\u007f]/.test(text) && !/(?:\.\.|%2e|%2f|%5c)/i.test(text)) return text
  return safeSupportLink(text)
}

export function normalizeNeedtokenSupport(raw: unknown, useBrandDefault = true): NeedtokenSupport {
  const config = raw && typeof raw === 'object' && !Array.isArray(raw) ? raw as Record<string, unknown> : {}
  const candidate = typeof config.business_email === 'string' ? config.business_email.trim() : ''
  const email = candidate.length <= 254 && /^[a-z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-z0-9-]+(?:\.[a-z0-9-]+)+$/i.test(candidate) ? candidate : ''
  return {
    enabled: config.enabled !== false && useBrandDefault,
    businessEmail: email || (useBrandDefault ? 'business@needtoken.ai' : ''),
    serviceHours: typeof config.service_hours === 'string' ? config.service_hours.trim().slice(0, 160) : '',
    contactUrl: safeSupportLink(config.contact_url),
    wecomQr: safeSupportImage(config.wecom_qr),
    qqQr: safeSupportImage(config.qq_qr),
  }
}

export function supportEmailHref(email: string): string {
  return email ? `mailto:${encodeURIComponent(email)}` : ''
}
