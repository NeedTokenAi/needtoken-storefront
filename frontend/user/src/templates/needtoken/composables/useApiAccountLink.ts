import { ref } from 'vue'
import { userApi } from '../../../api/client'
import { useAppStore } from '../../../stores/app'
import { getActiveTemplate } from '../../registry'
import {
  isSafeAuthorizeUrl,
  sanitizeConsoleNextPath,
  sanitizeConsoleTargetPath,
  sanitizeLinkNextPath,
  type ApiConsoleTargetPath,
} from '../lib/apiAccountLinkPolicy'

export interface ApiAccountLinkStatus {
  enabled: boolean
  linked: boolean
  status: string
  native_user_id?: number
  pending_orders: number
  updated_at?: string
}

export interface ApiAccountLinkStartResult {
  challenge_id: string
  authorize_url: string
  expires_at: string
}

export interface ApiAccountLinkCompleteResult {
  linked: boolean
  status: string
  native_user_id?: number
  next_path?: string
  awakened_orders: number
}

export interface ApiConsoleSSOResult {
  mode: 'sso' | 'bind_challenge' | string
  redirect_url: string
  linked: boolean
  native_user_id?: number
  auto_created?: boolean
  awakened_orders?: number
}

export interface ApiAccountEnsureResult {
  linked: boolean
  status: string
  native_user_id?: number
  awakened_orders?: number
}

export interface ApiSubscriptionSummary {
  id: number
  plan_id: number
  plan_title?: string
  status: string
  amount_total: number
  amount_used: number
  start_time: number
  end_time: number
  allow_wallet_overflow?: boolean
}

export interface ApiAccountSummary {
  linked: boolean
  native_user_id?: number
  quota?: number
  used_quota?: number
  request_count?: number
  group?: string
  active_subscriptions?: ApiSubscriptionSummary[]
}

const sharedStatus = ref<ApiAccountLinkStatus | null>(null)
const sharedLoaded = ref(false)
const sharedSummary = ref<ApiAccountSummary | null>(null)
const sharedSummaryLoaded = ref(false)
const sharedEnsureResult = ref<ApiAccountEnsureResult | null>(null)
const sharedEnsureAttempted = ref(false)

let statusInflight: Promise<ApiAccountLinkStatus | null> | null = null
let ensureInflight: Promise<ApiAccountEnsureResult | null> | null = null
let summaryInflight: Promise<ApiAccountSummary | null> | null = null

export async function fetchApiAccountLinkStatus(force = false): Promise<ApiAccountLinkStatus | null> {
  if (!force && sharedLoaded.value) {
    return sharedStatus.value
  }
  if (statusInflight) {
    return statusInflight
  }
  statusInflight = userApi
    .get('/me/api-account-link', { silentBusinessError: true })
    .then(res => {
      const data = (res?.data?.data || null) as ApiAccountLinkStatus | null
      sharedStatus.value = data
      sharedLoaded.value = true
      return data
    })
    .catch(() => {
      sharedStatus.value = null
      return null
    })
    .finally(() => {
      statusInflight = null
    })
  return statusInflight
}

export async function fetchApiAccountSummary(force = false): Promise<ApiAccountSummary | null> {
  if (!force && sharedSummaryLoaded.value) {
    return sharedSummary.value
  }
  if (summaryInflight) {
    return summaryInflight
  }
  summaryInflight = userApi
    .get('/me/api-account-summary', { silentBusinessError: true })
    .then(res => {
      const data = (res?.data?.data || null) as ApiAccountSummary | null
      sharedSummary.value = data
      sharedSummaryLoaded.value = true
      return data
    })
    .catch(() => null)
    .finally(() => {
      summaryInflight = null
    })
  return summaryInflight
}

export async function ensureApiAccountLink(options?: { force?: boolean }): Promise<ApiAccountEnsureResult | null> {
  const force = Boolean(options?.force)
  if (!force && sharedEnsureAttempted.value) {
    return sharedEnsureResult.value
  }
  if (ensureInflight) {
    return ensureInflight
  }
  ensureInflight = userApi
    .post('/me/api-account-link/ensure', {}, { silentBusinessError: true })
    .then(res => {
      const data = (res?.data?.data || null) as ApiAccountEnsureResult | null
      sharedEnsureAttempted.value = true
      sharedEnsureResult.value = data
      if (data?.linked) {
        sharedStatus.value = {
          enabled: true,
          linked: true,
          status: data.status || 'self_verified',
          native_user_id: data.native_user_id,
          pending_orders: 0,
        }
        sharedLoaded.value = true
        void fetchApiAccountSummary(true)
      }
      return data
    })
    .catch(() => {
      sharedEnsureAttempted.value = true
      return null
    })
    .finally(() => {
      ensureInflight = null
    })
  return ensureInflight
}

/**
 * Non-blocking, fail-silent trigger called after login, registration, 2FA,
 * Google OAuth, and verified email binding. Never throws or delays auth navigation.
 */
export function triggerSilentApiAccountLinkEnsure(options?: { force?: boolean }): void {
  try {
    if (getActiveTemplate() !== 'needtoken') return
    const appStore = useAppStore()
    if (appStore.isResellerTenant) return
    void ensureApiAccountLink(options).catch(() => null)
  } catch {
    // Silent degradation by design.
  }
}

export function resetApiAccountLinkCache(): void {
  sharedStatus.value = null
  sharedLoaded.value = false
  sharedSummary.value = null
  sharedSummaryLoaded.value = false
  sharedEnsureResult.value = null
  sharedEnsureAttempted.value = false
  statusInflight = null
  ensureInflight = null
  summaryInflight = null
}

export async function startApiAccountLinkRequest(nextPath?: string): Promise<ApiAccountLinkStartResult> {
  const safeNext = sanitizeLinkNextPath(nextPath)
  const res = await userApi.post(
    '/me/api-account-link/start',
    { next_path: safeNext },
    { silentBusinessError: true },
  )
  const data = res?.data?.data as ApiAccountLinkStartResult | undefined
  if (!data || !data.challenge_id || !isSafeAuthorizeUrl(data.authorize_url)) {
    throw new Error('invalid_authorize_url')
  }
  return data
}

export async function requestApiConsoleSSO(
  targetPath: ApiConsoleTargetPath | string = '/dashboard',
  nextPath = '/me',
): Promise<ApiConsoleSSOResult> {
  const safeTarget = sanitizeConsoleTargetPath(targetPath)
  const safeNext = sanitizeConsoleNextPath(nextPath)
  const res = await userApi.post(
    '/me/api-console-sso',
    { target_path: safeTarget, next_path: safeNext },
    { silentBusinessError: true },
  )
  const data = res?.data?.data as ApiConsoleSSOResult | undefined
  if (!data || !isSafeAuthorizeUrl(data.redirect_url)) {
    throw new Error('invalid_sso_redirect_url')
  }
  if (data.linked) {
    sharedStatus.value = {
      enabled: true,
      linked: true,
      status: 'self_verified',
      native_user_id: data.native_user_id,
      pending_orders: 0,
    }
    sharedLoaded.value = true
  }
  return data
}

export async function completeApiAccountLinkRequest(
  challengeId: string,
  approval: string,
): Promise<ApiAccountLinkCompleteResult> {
  const res = await userApi.post(
    '/me/api-account-link/complete',
    { challenge_id: challengeId, approval },
    { silentBusinessError: true },
  )
  const data = res?.data?.data as ApiAccountLinkCompleteResult | undefined
  if (!data || !data.linked) {
    throw new Error('link_complete_failed')
  }
  sharedStatus.value = {
    enabled: true,
    linked: true,
    status: data.status || 'self_verified',
    native_user_id: data.native_user_id,
    pending_orders: 0,
  }
  sharedLoaded.value = true
  sharedEnsureResult.value = {
    linked: true,
    status: data.status || 'self_verified',
    native_user_id: data.native_user_id,
    awakened_orders: data.awakened_orders,
  }
  void fetchApiAccountSummary(true)
  return data
}

export function useApiAccountLink() {
  const loading = ref(false)
  const ensuring = ref(false)
  const summaryLoading = ref(false)
  const starting = ref(false)
  const ssoLoading = ref(false)
  const ssoTarget = ref<string>('')
  const errorMessage = ref('')

  const load = async (force = false) => {
    loading.value = true
    try {
      return await fetchApiAccountLinkStatus(force)
    } finally {
      loading.value = false
    }
  }

  const ensure = async (force = false) => {
    ensuring.value = true
    try {
      return await ensureApiAccountLink({ force })
    } finally {
      ensuring.value = false
    }
  }

  const loadSummary = async (force = false) => {
    summaryLoading.value = true
    try {
      return await fetchApiAccountSummary(force)
    } finally {
      summaryLoading.value = false
    }
  }

  const beginLinking = async (nextPath?: string) => {
    if (starting.value || ssoLoading.value) return
    starting.value = true
    errorMessage.value = ''
    try {
      const challenge = await startApiAccountLinkRequest(nextPath)
      window.location.assign(challenge.authorize_url)
    } catch (err: any) {
      errorMessage.value = typeof err?.message === 'string' ? err.message : ''
      starting.value = false
    }
  }

  const openApiConsoleSSO = async (targetPath: ApiConsoleTargetPath | string = '/dashboard', nextPath = '/me') => {
    if (ssoLoading.value || starting.value) return
    ssoLoading.value = true
    ssoTarget.value = sanitizeConsoleTargetPath(targetPath)
    errorMessage.value = ''
    try {
      const result = await requestApiConsoleSSO(targetPath, nextPath)
      window.location.assign(result.redirect_url)
    } catch (err: any) {
      errorMessage.value = typeof err?.message === 'string' ? err.message : ''
      ssoLoading.value = false
      ssoTarget.value = ''
    }
  }

  return {
    status: sharedStatus,
    summary: sharedSummary,
    ensureResult: sharedEnsureResult,
    loading,
    ensuring,
    summaryLoading,
    starting,
    ssoLoading,
    ssoTarget,
    errorMessage,
    load,
    ensure,
    loadSummary,
    beginLinking,
    openApiConsoleSSO,
  }
}
