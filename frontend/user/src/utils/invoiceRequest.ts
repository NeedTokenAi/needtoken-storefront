export type InvoiceType = 'ordinary' | 'special_vat'
export type InvoiceTitleType = 'personal' | 'company'

export interface InvoiceRequest {
  type: InvoiceType
  title_type: InvoiceTitleType
  title: string
  tax_id?: string
  email: string
  company_address?: string
  company_phone?: string
  bank_name?: string
  bank_account?: string
}

export function normalizeInvoiceRequest(value: InvoiceRequest): InvoiceRequest {
  const result: InvoiceRequest = {
    type: value.type,
    title_type: value.type === 'special_vat' ? 'company' : value.title_type,
    title: String(value.title || '').trim(),
    email: String(value.email || '').trim(),
  }
  if (result.title_type === 'company') result.tax_id = String(value.tax_id || '').trim().toUpperCase()
  if (result.type === 'special_vat') {
    for (const field of ['company_address', 'company_phone', 'bank_name', 'bank_account'] as const) {
      const text = String(value[field] || '').trim()
      if (text) result[field] = field === 'bank_account' ? text.replace(/[ -]/g, '') : text
    }
  }
  return result
}

export function invoiceRequestError(value: InvoiceRequest): string | null {
  if (!['ordinary', 'special_vat'].includes(value.type) || !['personal', 'company'].includes(value.title_type)) return 'invalid'
  const row = normalizeInvoiceRequest(value)
  if (!row.title || Array.from(row.title).length > 120 || /[\u0000-\u001f\u007f]/.test(row.title)) return 'titleInvalid'
  if (row.email.length > 254 || !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(row.email)) return 'emailInvalid'
  if (row.title_type === 'company' && !/^[A-Z0-9]{15,20}$/.test(row.tax_id || '')) return 'taxInvalid'
  for (const field of ['company_address', 'company_phone', 'bank_name', 'bank_account'] as const) {
    const text = row[field] || ''
    const limit = ({ company_address: 300, company_phone: 32, bank_name: 120, bank_account: 40 })[field]
    if (Array.from(text).length > limit || /[\u0000-\u001f\u007f]/.test(text)) return 'extraInvalid'
  }
  if (row.company_phone && (!/^[0-9+() -]{6,32}$/.test(row.company_phone) || row.company_phone.replace(/[^0-9]/g, '').length < 6)) return 'extraInvalid'
  if (row.bank_account && !/^[0-9]{8,40}$/.test(row.bank_account)) return 'extraInvalid'
  return null
}

export function invoiceContactEmail(value: unknown): string {
  const text = typeof value === 'string' ? value.trim() : ''
  return text.length <= 254 && /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(text) ? text : ''
}
