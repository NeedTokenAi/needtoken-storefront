import test from 'node:test'
import assert from 'node:assert/strict'
import { invoiceContactEmail, invoiceRequestError, normalizeInvoiceRequest, type InvoiceRequest } from '../src/utils/invoiceRequest.ts'

const personal: InvoiceRequest = { type: 'ordinary', title_type: 'personal', title: ' Example Customer ', email: ' customer@example.test ' }
test('personal invoice normalizes contact and removes stale company and bank details', () => {
  const input = { ...personal, tax_id: '123456789012345678', bank_account: '1234567890123456', bank_name: 'Private bank' }
  assert.equal(invoiceRequestError(input), null)
  assert.deepEqual(normalizeInvoiceRequest(input), { type: 'ordinary', title_type: 'personal', title: 'Example Customer', email: 'customer@example.test' })
})
test('special VAT requires a company tax ID without forcing optional bank information', () => {
  const input: InvoiceRequest = { ...personal, type: 'special_vat', title_type: 'company', title: 'Example Company', tax_id: ' 12345678901234567x ' }
  assert.equal(invoiceRequestError(input), null)
  assert.equal(normalizeInvoiceRequest(input).tax_id, '12345678901234567X')
  assert.equal(invoiceRequestError({ ...input, tax_id: '' }), 'taxInvalid')
  assert.equal(invoiceRequestError({ ...input, company_phone: 'not-a-phone' }), 'extraInvalid')
})
test('billing details reject invalid recipients, controls and excessive values', () => {
  assert.equal(invoiceRequestError({ ...personal, email: 'Name <customer@example.test>' }), 'emailInvalid')
  assert.equal(invoiceRequestError({ ...personal, title: 'x'.repeat(121) }), 'titleInvalid')
  assert.equal(invoiceRequestError({ ...personal, title: 'Example\nCompany' }), 'titleInvalid')
  assert.equal(invoiceRequestError({ ...personal, type: 'other' as InvoiceRequest['type'] }), 'invalid')
  assert.equal(invoiceContactEmail('not-an-email'), '')
})
test('bank normalization is confined to special VAT and matches server validation', () => {
  const input: InvoiceRequest = { ...personal, type: 'special_vat', title_type: 'company', tax_id: '123456789012345678', bank_account: '1234 5678-9012', company_phone: '+86 (010) 12345678' }
  assert.equal(invoiceRequestError(input), null)
  assert.equal(normalizeInvoiceRequest(input).bank_account, '123456789012')
  assert.equal(invoiceRequestError({ ...input, bank_account: 'bad account' }), 'extraInvalid')
  assert.equal(normalizeInvoiceRequest({ ...input, type: 'ordinary' }).bank_account, undefined)
})
