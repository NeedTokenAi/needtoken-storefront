import { ref } from 'vue'
import { userApi } from '../../../api/client'
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

const sharedStatus = ref<ApiAccountLinkStatus | null>(null)
const sharedLoaded = ref(false)
let inflight: Promise<ApiAccountLinkStatus | null> | null = null

export async function fetchApiAccountLinkStatus(force = false): Promise<ApiAccountLinkStatus | null> {
  if (!force && sharedLoaded.value) {
    return sharedStatus.value
  }
  if (inflight) {
    return inflight
  }
  inflight = userApi
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
      inflight = null
    })
  return inflight
}

export function resetApiAccountLinkCache(): void {
  sharedStatus.value = null
  sharedLoaded.value = false
  inflight = null
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
  return data
}

export function useApiAccountLink() {
  const loading = ref(false)
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
    loading,
    starting,
    ssoLoading,
    ssoTarget,
    errorMessage,
    load,
    beginLinking,
    openApiConsoleSSO,
  }
}
