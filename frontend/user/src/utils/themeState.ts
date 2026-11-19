export const THEME_STORAGE_KEY = 'dujiao_theme'

export type ResolvedTheme = 'light' | 'dark'
export type ThemePreference = 'system' | ResolvedTheme

export function parseStoredTheme(value: string | null | undefined): ThemePreference {
  return value === 'light' || value === 'dark' ? value : 'system'
}

export function resolveTheme(preference: ThemePreference, systemDark: boolean): ResolvedTheme {
  if (preference === 'light' || preference === 'dark') return preference
  return systemDark ? 'dark' : 'light'
}

/**
 * Two-state toggle semantics:
 * - Target the opposite of the currently rendered theme.
 * - If the target matches the current system theme, clear the explicit override
 *   ('system') so future OS theme changes are followed again.
 * - Otherwise store the target literally as an explicit override.
 */
export function nextThemePreference(target: ResolvedTheme, systemDark: boolean): ThemePreference {
  const systemTheme: ResolvedTheme = systemDark ? 'dark' : 'light'
  return target === systemTheme ? 'system' : target
}

export function colorSchemeMetaValue(preference: ThemePreference): 'light dark' | ResolvedTheme {
  return preference === 'system' ? 'light dark' : preference
}
