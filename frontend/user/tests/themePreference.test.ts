import assert from 'node:assert/strict'
import test from 'node:test'
import {
  colorSchemeMetaValue,
  nextThemePreference,
  parseStoredTheme,
  resolveTheme,
} from '../src/utils/themeState.ts'

test('unset or malformed storage values default to following the system scheme', () => {
  assert.equal(parseStoredTheme(null), 'system')
  assert.equal(parseStoredTheme(undefined), 'system')
  assert.equal(parseStoredTheme(''), 'system')
  assert.equal(parseStoredTheme('auto'), 'system')
  assert.equal(parseStoredTheme('light'), 'light')
  assert.equal(parseStoredTheme('dark'), 'dark')
})

test('system preference resolves to the current OS color scheme and reflects light dark meta', () => {
  assert.equal(resolveTheme('system', false), 'light')
  assert.equal(resolveTheme('system', true), 'dark')
  assert.equal(colorSchemeMetaValue('system'), 'light dark')
})

test('explicit overrides pin light or dark regardless of later OS scheme changes', () => {
  assert.equal(resolveTheme('dark', false), 'dark')
  assert.equal(resolveTheme('dark', true), 'dark')
  assert.equal(resolveTheme('light', true), 'light')
  assert.equal(resolveTheme('light', false), 'light')
  assert.equal(colorSchemeMetaValue('dark'), 'dark')
  assert.equal(colorSchemeMetaValue('light'), 'light')
})

test('two-state toggling stores an explicit override when diverging from OS and clears it when matching OS', () => {
  // OS is light: switching to dark stores explicit override; switching back to light returns to system.
  assert.equal(nextThemePreference('dark', false), 'dark')
  assert.equal(nextThemePreference('light', false), 'system')

  // OS is dark: switching to light stores explicit override; switching back to dark returns to system.
  assert.equal(nextThemePreference('light', true), 'light')
  assert.equal(nextThemePreference('dark', true), 'system')
})
