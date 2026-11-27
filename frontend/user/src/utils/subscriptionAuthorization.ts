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

export type FulfillmentProgressStage =
  | 'awaiting_authorization'
  | 'queued'
  | 'executing'
  | 'credential_expired'
  | 'payment_unknown'
  | 'manual_review'
  | 'completed'
  | 'canceled'
  | 'refunded'
  | string

export interface FulfillmentProgress {
  stage: FulfillmentProgressStage
  authorization_submitted: boolean
  next_action: string
  updated_at?: string
  terminal: boolean
}

export interface SubscriptionProgressStep {
  key: 'paid' | 'authorize' | 'checkout' | 'verified'
  state: SubscriptionProgressStepState
}

const TERMINAL_ORDER_STATUSES = new Set(['delivered', 'completed', 'canceled', 'refunded', 'expired'])

export function shouldPollFulfillmentOrder(
  orderStatus: string | undefined,
  progress?: FulfillmentProgress | null,
): boolean {
  if (progress?.terminal) return false
  const st = String(orderStatus || '').trim().toLowerCase()
  if (st && TERMINAL_ORDER_STATUSES.has(st)) return false
  return true
}

export function deriveSubscriptionProgressSteps(
  orderStatus: string | undefined,
  submitted: boolean,
  progress?: FulfillmentProgress | null,
): SubscriptionProgressStep[] {
  const stage = String(progress?.stage || '').trim().toLowerCase()
  const status = String(orderStatus || '').trim().toLowerCase()
  if (stage === 'completed' || status === 'delivered' || status === 'completed') {
    return [
      { key: 'paid', state: 'done' },
      { key: 'authorize', state: 'done' },
      { key: 'checkout', state: 'done' },
      { key: 'verified', state: 'done' },
    ]
  }
  if (stage === 'executing' || stage === 'payment_unknown' || stage === 'manual_review') {
    return [
      { key: 'paid', state: 'done' },
      { key: 'authorize', state: 'done' },
      { key: 'checkout', state: 'active' },
      { key: 'verified', state: 'pending' },
    ]
  }
  if (stage === 'queued' || (submitted && stage !== 'credential_expired')) {
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
