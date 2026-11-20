export interface SubscriptionSession {
  authorization_id: string
  session_token: string
  credential_url: string
  expires_at: string
  execution_scope: 'inspection_only'
  consent_ref: string
}

// Caller credentials remain in component memory and go only to the endpoint
// supplied by the authenticated order service, never to the storefront API.
export function validateSubscriptionSession(value: unknown, now = Date.now()): SubscriptionSession | null {
  if (!value || typeof value !== 'object') return null
  const session = value as Record<string, unknown>
  if (session.execution_scope !== 'inspection_only' || typeof session.authorization_id !== 'string' ||
      typeof session.session_token !== 'string' || !/^[A-Za-z0-9_-]{43}$/.test(session.session_token) ||
      typeof session.credential_url !== 'string' || typeof session.expires_at !== 'string' ||
      session.consent_ref !== session.authorization_id) return null
  try {
    const endpoint = new URL(session.credential_url)
    const expires = Date.parse(session.expires_at)
    if (endpoint.protocol !== 'https:' || !endpoint.hostname || endpoint.username || endpoint.password ||
        endpoint.pathname !== '/api/authorization/v1/credential' || endpoint.search || endpoint.hash ||
        !Number.isFinite(expires) || expires <= now || expires > now + 30 * 60 * 1000) return null
  } catch { return null }
  return session as unknown as SubscriptionSession
}

export function extractPlatformCredential(input: string): string | null {
  if (input.length > 32768) return null
  let credential = input.trim()
  if (credential.startsWith('{')) {
    try {
      const session = JSON.parse(credential)
      if (typeof session.accessToken !== 'string') return null
      credential = session.accessToken.trim()
    } catch { return null }
  }
  return credential.length >= 8 && credential.length <= 16384 && !/[\s\x00]/.test(credential) ? credential : null
}

export type SubscriptionProgressStepState = 'done' | 'active' | 'pending'

export interface SubscriptionProgressStep {
  key: 'paid' | 'authorize' | 'checkout' | 'verified'
  state: SubscriptionProgressStepState
}

export function deriveSubscriptionProgressSteps(orderStatus: string | undefined, submitted: boolean): SubscriptionProgressStep[] {
  const status = String(orderStatus || '').trim().toLowerCase()
  const finished = status === 'delivered' || status === 'completed'
  if (finished) {
    return [
      { key: 'paid', state: 'done' },
      { key: 'authorize', state: 'done' },
      { key: 'checkout', state: 'done' },
      { key: 'verified', state: 'done' },
    ]
  }
  if (submitted) {
    return [
      { key: 'paid', state: 'done' },
      { key: 'authorize', state: 'done' },
      { key: 'checkout', state: 'active' },
      { key: 'verified', state: 'pending' },
    ]
  }
  return [
    { key: 'paid', state: 'done' },
    { key: 'authorize', state: 'active' },
    { key: 'checkout', state: 'pending' },
    { key: 'verified', state: 'pending' },
  ]
}
