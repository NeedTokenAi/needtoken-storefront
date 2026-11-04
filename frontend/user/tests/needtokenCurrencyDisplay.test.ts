import assert from 'node:assert/strict'
import test from 'node:test'
import { convertDisplayAmount, displayCurrencyOptions, normalizeUsdCnyRate, resolveDisplayCurrency } from '../src/templates/needtoken/lib/currencyDisplay.ts'

test('FX is explicit CNY per USD with bounded decimal precision and no invented fallback', () => {
  assert.equal(normalizeUsdCnyRate('007.200000'), '7.2')
  assert.equal(normalizeUsdCnyRate('1000.000000'), '1000')
  for (const value of ['', null, 0, -7, 'Infinity', 'NaN', '1e2', '7.1234567', '1000.000001', '0.000000']) assert.equal(normalizeUsdCnyRate(value), null)
})

test('CNY/USD quotes convert in both directions and round half up at cents', () => {
  assert.equal(convertDisplayAmount('720', 'CNY', 'USD', '7.2'), '100.00')
  assert.equal(convertDisplayAmount('100', 'USD', 'CNY', '7.2'), '720.00')
  assert.equal(convertDisplayAmount('179', 'CNY', 'USD', '7.2'), '24.86')
  assert.equal(convertDisplayAmount('0.01', 'CNY', 'USD', '2'), '0.01')
  assert.equal(convertDisplayAmount('1.01', 'USD', 'CNY', '7.123456'), '7.19')
  assert.equal(convertDisplayAmount('0', 'USD', 'CNY', '7.2'), '0.00')
})

test('missing rates, unsafe amounts and unsupported conversion never produce a false quote', () => {
  for (const amount of ['', null, -1, 'NaN', '9007199254740992']) assert.equal(convertDisplayAmount(amount, 'CNY', 'USD', '7'), null)
  assert.equal(convertDisplayAmount('12', 'CNY', 'USD', ''), null)
  assert.equal(convertDisplayAmount('12', 'EUR', 'USD', '7'), null)
  assert.equal(convertDisplayAmount('90071992547400', 'CNY', 'USD', '0.000001'), null)
  assert.equal(convertDisplayAmount('12.34', 'EUR', 'EUR', ''), '12.34')
})

test('merchant default and remembered choice require a usable rate and settlement pair', () => {
  assert.equal(resolveDisplayCurrency('CNY', '7.2', 'USD', null), 'USD')
  assert.equal(resolveDisplayCurrency('CNY', '7.2', 'USD', 'CNY'), 'CNY')
  assert.equal(resolveDisplayCurrency('CNY', '', 'USD', 'USD'), 'CNY')
  assert.equal(resolveDisplayCurrency('EUR', '7.2', 'USD', 'USD'), 'EUR')
  assert.deepEqual(displayCurrencyOptions('CNY', ''), ['CNY'])
  assert.deepEqual(displayCurrencyOptions('USD', '7.2'), ['CNY', 'USD'])
})
