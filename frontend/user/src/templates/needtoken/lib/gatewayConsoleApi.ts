import { userApi } from '../../../api/client'
import { API_QUOTA_PER_USD } from './apiAccountLinkPolicy.ts'
export {
  computeCombinedApiQuota,
  normalizeNeedtokenMemberSection,
  rawQuotaToUsdNumber,
  sumSubscriptionRemainingQuota,
  usdToRawQuota,
  type NeedtokenMemberSection,
} from './gatewayConsoleHelpers.ts'

export interface GatewayTokenItem {
  id: number
  user_id: number
  name: string
  key: string
  masked_key: string
  status: number
  group: string
  remain_quota: number
  used_quota: number
  unlimited_quota: boolean
  expired_time: number
  created_time: number
  accessed_time: number
  model_limits_enabled: boolean
  model_limits: string
  allow_ips?: string | null
  cross_group_retry: boolean
  auto_groups?: string[] | null
}

export interface GatewayTokensResponse {
  items: GatewayTokenItem[]
  total: number
  usable_groups: Record<string, string>
  default_group: string
  default_auto_group: boolean
  quota_per_unit: number
  active_subscriptions?: Array<{
    id: number
    plan_id: number
    plan_title?: string
    allowed_groups?: string
    status: string
    amount_total: number
    amount_used: number
    start_time: number
    end_time: number
    allow_wallet_overflow?: boolean
  }>
}

export interface GatewayTokenUpsertPayload {
  name?: string
  status?: number
  group?: string
  remain_quota?: number
  unlimited_quota?: boolean
  expired_time?: number
  model_limits_enabled?: boolean
  model_limits?: string
  cross_group_retry?: boolean
  auto_groups?: string[]
}

export interface GatewayLogItem {
  id: number
  created_at: number
  type: number
  model_name: string
  token_name: string
  group: string
  prompt_tokens: number
  completion_tokens: number
  quota: number
  use_time: number
  is_stream: boolean
  content: string
}

export interface GatewayLogsResponse {
  items: GatewayLogItem[]
  total: number
  page: number
  page_size: number
  quota_per_unit: number
}

export interface GatewayUsageDailyItem {
  date: string
  quota: number
  requests: number
  token_used: number
}

export interface GatewayUsageModelItem {
  model_name: string
  quota: number
  requests: number
  token_used: number
}

export interface GatewayUsageResponse {
  days: number
  start_ts: number
  end_ts: number
  total_quota: number
  total_requests: number
  total_tokens: number
  quota_per_unit: number
  daily: GatewayUsageDailyItem[]
  models: GatewayUsageModelItem[]
}

export async function listGatewayTokens(): Promise<GatewayTokensResponse> {
  const res = await userApi.get('/me/gateway/tokens', { silentBusinessError: true })
  return (res?.data?.data || { items: [], total: 0, usable_groups: {}, default_group: 'default', default_auto_group: false, quota_per_unit: API_QUOTA_PER_USD }) as GatewayTokensResponse
}

export async function createGatewayToken(payload: GatewayTokenUpsertPayload): Promise<{ key: string; token: GatewayTokenItem }> {
  const res = await userApi.post('/me/gateway/tokens', payload)
  return res?.data?.data as { key: string; token: GatewayTokenItem }
}

export async function updateGatewayToken(tokenId: number, payload: GatewayTokenUpsertPayload): Promise<GatewayTokenItem> {
  const res = await userApi.put(`/me/gateway/tokens/${tokenId}`, payload)
  return res?.data?.data as GatewayTokenItem
}

export async function deleteGatewayToken(tokenId: number): Promise<void> {
  await userApi.delete(`/me/gateway/tokens/${tokenId}`)
}

export async function listGatewayLogs(params: {
  p?: number
  page_size?: number
  model_name?: string
  token_name?: string
  type?: number
  start_timestamp?: number
  end_timestamp?: number
} = {}): Promise<GatewayLogsResponse> {
  const res = await userApi.get('/me/gateway/logs', { params, silentBusinessError: true })
  return (res?.data?.data || { items: [], total: 0, page: 1, page_size: 20, quota_per_unit: API_QUOTA_PER_USD }) as GatewayLogsResponse
}

export async function getGatewayUsage(days = 7): Promise<GatewayUsageResponse> {
  const res = await userApi.get('/me/gateway/usage', { params: { days }, silentBusinessError: true })
  return (res?.data?.data || { days, start_ts: 0, end_ts: 0, total_quota: 0, total_requests: 0, total_tokens: 0, quota_per_unit: API_QUOTA_PER_USD, daily: [], models: [] }) as GatewayUsageResponse
}
