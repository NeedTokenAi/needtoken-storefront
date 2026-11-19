import { computed, ref, watch } from 'vue'
import {
    THEME_STORAGE_KEY,
    colorSchemeMetaValue,
    nextThemePreference,
    parseStoredTheme,
    resolveTheme,
    type ResolvedTheme,
    type ThemePreference,
} from './themeState'

export {
    THEME_STORAGE_KEY,
    colorSchemeMetaValue,
    nextThemePreference,
    parseStoredTheme,
    resolveTheme,
    type ResolvedTheme,
    type ThemePreference,
}

const isBrowser = typeof window !== 'undefined' && typeof document !== 'undefined'

function readStoredPreference(): ThemePreference {
    if (!isBrowser) return 'system'
    try {
        return parseStoredTheme(window.localStorage.getItem(THEME_STORAGE_KEY))
    } catch {
        return 'system'
    }
}

function writeStoredPreference(next: ThemePreference): void {
    if (!isBrowser) return
    try {
        if (next === 'system') {
            window.localStorage.removeItem(THEME_STORAGE_KEY)
        } else {
            window.localStorage.setItem(THEME_STORAGE_KEY, next)
        }
    } catch {
        /* Storage may be blocked in restricted contexts; theme still applies in memory. */
    }
}

const mediaQuery = isBrowser && typeof window.matchMedia === 'function'
    ? window.matchMedia('(prefers-color-scheme: dark)')
    : null

const systemDark = ref<boolean>(mediaQuery?.matches ?? false)
const preference = ref<ThemePreference>(readStoredPreference())
const theme = computed<ResolvedTheme>(() => resolveTheme(preference.value, systemDark.value))

function syncDocumentTheme(pref: ThemePreference, resolved: ResolvedTheme): void {
    if (!isBrowser) return
    const root = document.documentElement
    root.classList.toggle('dark', resolved === 'dark')
    let meta = document.querySelector<HTMLMetaElement>('meta[name="color-scheme"]')
    if (!meta) {
        meta = document.createElement('meta')
        meta.name = 'color-scheme'
        document.head.appendChild(meta)
    }
    meta.content = colorSchemeMetaValue(pref)
}

if (isBrowser) {
    // Keep system preference reactive without retroactively clearing explicit user overrides.
    const onSystemChange = (event: MediaQueryListEvent) => {
        systemDark.value = event.matches
    }
    if (mediaQuery) {
        if (typeof mediaQuery.addEventListener === 'function') {
            mediaQuery.addEventListener('change', onSystemChange)
        } else {
            mediaQuery.addListener(onSystemChange)
        }
    }

    window.addEventListener('storage', (event: StorageEvent) => {
        if (event.key === THEME_STORAGE_KEY || event.key === null) {
            preference.value = parseStoredTheme(event.newValue)
        }
    })

    watch([preference, theme], ([pref, resolved]) => {
        syncDocumentTheme(pref, resolved)
    }, { immediate: true })
}

export const useTheme = () => {
    const setTheme = (next: ThemePreference) => {
        const resolvedNext = next === 'system'
            ? 'system'
            : nextThemePreference(next, systemDark.value)
        preference.value = resolvedNext
        writeStoredPreference(resolvedNext)
    }

    const toggleTheme = () => {
        const target: ResolvedTheme = theme.value === 'dark' ? 'light' : 'dark'
        setTheme(target)
    }

    return {
        theme,
        preference,
        setTheme,
        toggleTheme,
    }
}
