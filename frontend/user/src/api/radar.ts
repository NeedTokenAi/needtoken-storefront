import { userApi } from './client'

export type RadarSubscriptionChannel = 'codex' | 'claude' | 'gemini' | 'grok'

export interface RadarSubscriptionPreferences {
  email: string
  email_bound: boolean
  channels: Record<RadarSubscriptionChannel, boolean>
  subscribed_count: number
}

export const radarSubscriptionAPI = {
  getPreferences: () =>
    userApi.get('/me/radar-subscriptions'),
  updatePreferences: (channels: Partial<Record<RadarSubscriptionChannel, boolean>>) =>
    userApi.put('/me/radar-subscriptions', { channels }),
}
