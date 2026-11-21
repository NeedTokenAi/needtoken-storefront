export interface CallbackFragmentPayload {
  challengeId: string
  approval: string
}

export const ALLOWED_CONSOLE_TARGET_PATHS = [
  '/dashboard',
  '/keys',
  '/subscriptions',
  '/wallet',
  '/usage-logs',
  '/playground',
] as const

export type ApiConsoleTargetPath = (typeof ALLOWED_CONSOLE_TARGET_PATHS)[number]

const SAFE_NEXT_PATH_RE = /^\/(|me|plans|api-services|models|checkout|profile|orders|orders\/[A-Za-z0-9_-]{1,64})$/

/**
 * Extract challenge_id and approval from URL fragment (#challenge_id=...&approval=...).
 * Tokens travel only in fragment so they stay out of proxy logs and Referer headers.
 */
export function parseAccountLinkFragment(rawHash: string): CallbackFragmentPayload | null {
  const cleaned = rawHash.startsWith('#') ? rawHash.slice(1) : rawHash
  if (!cleaned) return null
  const params = new URLSearchParams(cleaned)
  const challengeId = (params.get('challenge_id') || '').trim()
  const approval = (params.get('approval') || '').trim()
  if (!challengeId || !approval || challengeId.length > 128 || approval.length > 2048) {
    return null
  }
  return { challengeId, approval }
}

/**
 * Normalize an internal storefront next_path before sending it to the backend.
 * Any disallowed or external value falls back to '/plans'.
 */
export function sanitizeLinkNextPath(raw?: string | null): string {
  const trimmed = (raw || '').trim()
  if (!trimmed) return '/plans'
  if (trimmed === '/me' || trimmed === '/me/profile') return '/profile'
  if (trimmed === '/me/orders') return '/orders'
  return SAFE_NEXT_PATH_RE.test(trimmed) ? trimmed : '/plans'
}

/**
 * Normalize target_path for API console SSO against the explicit allowlist.
 */
export function sanitizeConsoleTargetPath(raw?: string | null): ApiConsoleTargetPath {
  const trimmed = (raw || '').trim() as ApiConsoleTargetPath
  return (ALLOWED_CONSOLE_TARGET_PATHS as readonly string[]).includes(trimmed)
    ? trimmed
    : '/dashboard'
}

/**
 * Normalize return next_path for API console SSO, defaulting to '/me'.
 */
export function sanitizeConsoleNextPath(raw?: string | null): string {
  const trimmed = (raw || '').trim()
  if (!trimmed || trimmed === '/me/profile') return '/me'
  if (trimmed === '/me/orders') return '/orders'
  return SAFE_NEXT_PATH_RE.test(trimmed) ? trimmed : '/me'
}

/**
 * Convert the backend-returned safe next_path back into a storefront vue-router path.
 */
export function resolveReturnRoute(nextPath?: string | null): string {
  const trimmed = (nextPath || '').trim()
  if (!SAFE_NEXT_PATH_RE.test(trimmed)) return '/me/orders'
  if (trimmed === '' || trimmed === '/') return '/plans'
  if (trimmed === '/profile' || trimmed === '/me') return '/me'
  if (trimmed === '/orders') return '/me/orders'
  return trimmed
}

/**
 * Reject non-http(s) or credential-bearing authorize URLs before navigating.
 */
export function isSafeAuthorizeUrl(raw?: string | null): boolean {
  const trimmed = (raw || '').trim()
  if (!trimmed) return false
  try {
    const parsed = new URL(trimmed)
    return (parsed.protocol === 'https:' || parsed.protocol === 'http:') && !parsed.username && !parsed.password
  } catch {
    return false
  }
}
