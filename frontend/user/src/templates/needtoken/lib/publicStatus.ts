/**
 * Public status-page summaries only. This parser never calculates an uptime
 * percentage: the public payload does not include a historical availability series.
 */

export type PublicComponentStatus =
  | 'operational'
  | 'degraded_performance'
  | 'partial_outage'
  | 'major_outage'
  | 'under_maintenance'
  | 'unknown'

export interface PublicStatusComponent {
  id?: string
  name: string
  status: PublicComponentStatus
  groupId?: string
  isGroup?: boolean
  childIds?: string[]
}

export interface PublicStatusIncident {
  name: string
  status: string
  impact: string
  updatedAt: string | null
}

export interface PublicStatusSnapshot {
  ok: true
  name: string
  url: string
  description: string
  indicator: 'none' | 'minor' | 'major' | 'critical' | 'unknown'
  updatedAt: string | null
  components: PublicStatusComponent[]
  incidents: PublicStatusIncident[]
}

export type PublicStatusView = PublicStatusSnapshot | { ok: false }

const COMPONENT_STATUSES = new Set<PublicComponentStatus>([
  'operational',
  'degraded_performance',
  'partial_outage',
  'major_outage',
  'under_maintenance',
])

const INDICATORS = new Set(['none', 'minor', 'major', 'critical'])

const record = (value: unknown): Record<string, unknown> | null =>
  value && typeof value === 'object' && !Array.isArray(value) ? value as Record<string, unknown> : null

const shortText = (value: unknown, max: number): string => {
  if (typeof value !== 'string') return ''
  const text = value.replace(/\s+/g, ' ').trim()
  return text.length > max ? text.slice(0, max) : text
}

const httpsUrl = (value: unknown, fallback: string): string => {
  if (typeof value !== 'string') return fallback
  try {
    const url = new URL(value)
    return url.protocol === 'https:' ? url.toString() : fallback
  } catch {
    return fallback
  }
}

const timestamp = (value: unknown): string | null => {
  if (typeof value !== 'string' || !value.trim()) return null
  const time = new Date(value)
  return Number.isNaN(time.getTime()) ? null : time.toISOString()
}

export function parsePublicStatus(payload: unknown, fallbackUrl: string): PublicStatusView {
  const data = record(payload)
  const page = record(data?.page)
  const status = record(data?.status)
  if (!data || !page || !status) return { ok: false }

  const components = Array.isArray(data.components)
    ? data.components.flatMap((row): PublicStatusComponent[] => {
      const item = record(row)
      const name = shortText(item?.name, 80)
      if (!name) return []
      const raw = shortText(item?.status, 40)
      const componentStatus: PublicComponentStatus = COMPONENT_STATUSES.has(raw as PublicComponentStatus)
        ? raw as PublicComponentStatus
        : 'unknown'
      const id = shortText(item?.id, 80)
      const groupId = shortText(item?.group_id, 80)
      return [{
        ...(id ? { id } : {}),
        name,
        status: componentStatus,
        ...(groupId ? { groupId } : {}),
        ...(item?.group === true ? { isGroup: true } : {}),
        ...(Array.isArray(item?.components) ? { childIds: item.components.flatMap(value => typeof value === 'string' && value.trim() ? [value.trim()] : []) } : {}),
      }]
    })
    : []

  const incidents = Array.isArray(data.incidents)
    ? data.incidents.flatMap((row): PublicStatusIncident[] => {
      const item = record(row)
      const name = shortText(item?.name, 160)
      if (!name) return []
      return [{
        name,
        status: shortText(item?.status, 40) || 'unknown',
        impact: shortText(item?.impact, 40) || 'unknown',
        updatedAt: timestamp(item?.updated_at),
      }]
    })
    : []

  const description = shortText(status.description, 160)
  if (!description && !components.length && !incidents.length) return { ok: false }
  const indicator = shortText(status.indicator, 20)
  return {
    ok: true,
    name: shortText(page.name, 80) || 'status',
    url: httpsUrl(page.url, fallbackUrl),
    description: description || 'unknown',
    indicator: INDICATORS.has(indicator) ? indicator as PublicStatusSnapshot['indicator'] : 'unknown',
    updatedAt: timestamp(page.updated_at),
    components,
    incidents,
  }
}

/** Incident feeds publish page + incidents and omit the summary status object. */
export function parsePublicIncidents(payload: unknown): PublicStatusIncident[] | null {
  const data = record(payload)
  if (!data || !Array.isArray(data.incidents)) return null
  return data.incidents.flatMap((row): PublicStatusIncident[] => {
    const item = record(row)
    const name = shortText(item?.name, 160)
    if (!name) return []
    return [{
      name,
      status: shortText(item?.status, 40) || 'unknown',
      impact: shortText(item?.impact, 40) || 'unknown',
      updatedAt: timestamp(item?.updated_at),
    }]
  })
}

export function operationalCount(components: PublicStatusComponent[]): { ok: number; total: number } {
  return {
    ok: components.filter(component => component.status === 'operational').length,
    total: components.length,
  }
}

export function matchesTerms(value: string, terms: string[]): boolean {
  const haystack = value.toLowerCase()
  return terms.some(term => haystack.includes(term.toLowerCase()))
}

export type PublicStatusService = 'gpt' | 'claude'

// OpenAI's compatibility API currently flattens product groups and repeats names
// such as "Login". These IDs were verified against the ChatGPT group on the
// official status page and /api/v2/components.json on 2026-11-14. Match both ID
// and name; renamed or unfamiliar components stay outside this scoped summary.
// https://status.openai.com/ (ChatGPT group, excluding Codex in ChatGPT Desktop)
const CHATGPT_COMPONENTS: Record<string, string> = {
  '01JMXBNJXGV1T5GT2M9XA83XNG': 'Conversations',
  '01JMXBNJXG1S2D9V65P1ZZTD94': 'Login',
  '01KX45G1SH21AX5DT93D4HMF0P': 'ChatGPT Work',
  '01JNKS9D9S72PMP1938PVFFQN4': 'Compliance API',
  '01JMXBNJXGKKP51D4DEJ2HZJ8Q': 'Search',
  '01JMXBNJXG1YMQPPCPCQX3MPA2': 'File uploads',
  '01JMXBNJXGGT5SR5DB9J7GYY48': 'Voice mode',
  '01JSFK5QX36ZRW0TW0ZV0ZYFXQ': 'GPTs',
  '01JQ7EKW990MSPSWVXC7VPV2ZJ': 'Image Generation',
  '01JSYVYQSWMJ9QG35XHP08BHA7': 'Deep Research',
  '01JSG1XMJ9RVJJQ0E85NVSJ2AZ': 'Agent',
  '01M3Q6NERHVE2WPPAQRHBAC6XB': 'dots',
  '01M3Q6NERHS4BG0HQ85VAR1SRH': 'Space',
  '01KX45G1SHQQ9DTAX9S4W7FV8G': 'Sites',
  '01K6TVGGGDCP0PPGCHXAG3AQX8': 'Connectors/Apps',
}

/** Consumer product status only; unrelated API and Codex failures cannot leak in. */
export function serviceStatusComponents(components: PublicStatusComponent[], service: PublicStatusService): PublicStatusComponent[] {
  if (service === 'claude') return components.filter(component => !component.isGroup && component.name.toLowerCase().trim() === 'claude.ai')
  const groups = components.filter(component => component.isGroup && /^chatgpt$/i.test(component.name.trim()))
  const groupIds = new Set(groups.flatMap(component => component.id ? [component.id] : []))
  const childIds = new Set(groups.flatMap(component => component.childIds ?? []))
  return components.filter(component => {
    if (component.isGroup || /codex/i.test(component.name)) return false
    if (component.groupId && groupIds.has(component.groupId)) return true
    if (component.id && childIds.has(component.id)) return true
    if (/^chatgpt(?:$|[\s:/-])/i.test(component.name.trim())) return true
    return !!component.id && CHATGPT_COMPONENTS[component.id] === component.name
  })
}

export const PUBLIC_STATUS_MAX_AGE_MS = 2 * 60 * 1000

/** Retrieval time is freshness; a provider's unchanged component date is not. */
export function publicStatusFresh(fetchedAt: string | null, now: number): boolean {
  if (!fetchedAt) return false
  const fetched = Date.parse(fetchedAt)
  return Number.isFinite(fetched) && fetched <= now + 30_000 && now - fetched < PUBLIC_STATUS_MAX_AGE_MS
}

export function publicStatusTone(components: PublicStatusComponent[], fresh: boolean): PublicComponentStatus {
  if (!fresh || !components.length) return 'unknown'
  for (const status of ['major_outage', 'partial_outage', 'degraded_performance', 'under_maintenance', 'unknown'] as const) {
    if (components.some(component => component.status === status)) return status
  }
  return 'operational'
}
