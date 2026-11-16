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
  id?: string
  name: string
  status: string
  impact: string
  updatedAt: string | null
  createdAt: string | null
  startedAt: string | null
  resolvedAt: string | null
  url: string | null
  components: PublicIncidentComponent[]
  latestUpdate: PublicIncidentUpdate | null
  updates: PublicIncidentUpdate[]
}

export interface PublicIncidentComponent {
  id?: string
  name: string
}

export interface PublicIncidentUpdate {
  body: string
  status: string
  updatedAt: string | null
}

export interface PublicStatusHistory {
  state: 'ready' | 'unavailable'
  coverage: 'partial'
  sourceUrl: string
  incidents: PublicStatusIncident[]
}

export interface PublicHistoryDay {
  date: string
  incidents: string[]
  records: PublicStatusIncident[]
  tone: PublicComponentStatus
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

const incidentUrl = (value: unknown, sourceUrl: string): string | null => {
  if (typeof value !== 'string') return null
  try {
    const url = new URL(value)
    return url.protocol === 'https:' && url.origin === new URL(sourceUrl).origin && !url.username && !url.password ? url.toString() : null
  } catch { return null }
}

const incidentComponents = (value: unknown): PublicIncidentComponent[] => {
  if (!Array.isArray(value)) return []
  return value.flatMap((row): PublicIncidentComponent[] => {
    const item = record(row)
    const id = shortText(item?.id ?? item?.code, 80)
    const name = shortText(item?.name, 80)
    return id || name ? [{ ...(id ? { id } : {}), name }] : []
  })
}

const parseIncident = (row: unknown, sourceUrl: string): PublicStatusIncident | null => {
  const item = record(row)
  const name = shortText(item?.name, 500)
  if (!item || !name) return null
  const id = shortText(item.id, 80)
  const updates = Array.isArray(item.incident_updates) ? item.incident_updates.flatMap(row => {
    const update = record(row)
    if (!update) return []
    return [{
      body: typeof update.body === 'string' ? update.body.replace(/\r\n?/g, '\n').trim().slice(0, 16_384) : '',
      status: shortText(update.status, 40) || 'unknown',
      updatedAt: timestamp(update.display_at) ?? timestamp(update.updated_at) ?? timestamp(update.created_at),
      components: incidentComponents(update.affected_components),
    }]
  }) : []
  updates.sort((a, b) => (Date.parse(b.updatedAt || '') || 0) - (Date.parse(a.updatedAt || '') || 0))
  const latest = updates.find(update => update.body) ?? updates[0]
  const components = incidentComponents(item.components)
  for (const component of updates.flatMap(update => update.components)) {
    if (!components.some(known => component.id ? component.id === known.id : component.name === known.name)) components.push(component)
  }
  return {
    ...(id ? { id } : {}), name,
    status: shortText(item.status, 40) || 'unknown',
    impact: shortText(item.impact, 40) || 'unknown',
    updatedAt: timestamp(item.updated_at),
    createdAt: timestamp(item.created_at),
    startedAt: timestamp(item.started_at),
    resolvedAt: timestamp(item.resolved_at),
    url: incidentUrl(item.url ?? item.shortlink, sourceUrl),
    components,
    latestUpdate: latest ? { body: latest.body, status: latest.status, updatedAt: latest.updatedAt } : null,
    updates: updates.map(({ body, status, updatedAt }) => ({ body, status, updatedAt })),
  }
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
      const incident = parseIncident(row, fallbackUrl)
      return incident ? [incident] : []
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
    const incident = parseIncident(row, shortText(record(data.page)?.url, 200))
    return incident ? [incident] : []
  })
}

/** The public feeds are bounded recent records, never a complete uptime series. */
export function parsePublicStatusHistory(payload: unknown, sourceUrl: string): PublicStatusHistory {
  const data = record(payload)
  const base: PublicStatusHistory = { state: 'unavailable', coverage: 'partial', sourceUrl, incidents: [] }
  if (!data || data.state !== 'ready' || data.coverage !== 'partial' || !Array.isArray(data.incidents)) return base
  return {
    state: 'ready', coverage: 'partial',
    sourceUrl: incidentUrl(data.source_url, sourceUrl) ?? sourceUrl,
    incidents: data.incidents.slice(0, 50).flatMap(row => {
      const incident = parseIncident(row, sourceUrl)
      return incident ? [incident] : []
    }),
  }
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

// Compatibility snapshots may flatten product groups and repeat names
// such as "Login". These IDs were verified against the ChatGPT group on the
// official status page and /api/v2/components.json on 2026-11-14. Match both ID
// and name only when native group metadata is absent. Renamed or unfamiliar
// components stay outside the compatibility fallback.
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
    // Real product boundaries take precedence over the compatibility fallback.
    if (groups.length || component.groupId) return false
    if (/^chatgpt(?:$|[\s:/-])/i.test(component.name.trim())) return true
    return !!component.id && CHATGPT_COMPONENTS[component.id] === component.name
  })
}

/** Incident titles alone never prove product impact. Preserve official linkage. */
export function serviceStatusIncidents(incidents: PublicStatusIncident[], components: PublicStatusComponent[], service: PublicStatusService): PublicStatusIncident[] {
  const scoped = serviceStatusComponents(components, service)
  const scopeIds = new Set(scoped.flatMap(component => component.id ? [component.id] : []))
  const nativeGroups = service === 'gpt' && components.some(component => component.isGroup && /^chatgpt$/i.test(component.name.trim()))
  if (service === 'gpt') for (const group of components.filter(component => component.isGroup && /^chatgpt$/i.test(component.name.trim()))) if (group.id) scopeIds.add(group.id)
  return incidents.filter(incident => incident.components.some(component => {
    if (component.id && scopeIds.has(component.id)) return true
    if (nativeGroups) return false
    return serviceStatusComponents([{ ...component, status: 'unknown' }], service).length > 0
  })).sort((a, b) => {
    const activeDifference = Number(incidentActive(b)) - Number(incidentActive(a))
    return activeDifference || incidentTime(b) - incidentTime(a)
  })
}

/** A later optional history fetch may be cached older than the current summary. */
export function mergePublicStatusIncidents(current: PublicStatusIncident[], history: PublicStatusIncident[]): PublicStatusIncident[] {
  const key = (incident: PublicStatusIncident) => incident.id || incident.url || incident.name
  const currentKeys = new Set(current.map(key))
  const records = new Map<string, PublicStatusIncident>()
  for (const row of [...history, ...current]) {
    // The current summary lists unresolved incidents. A cached historical
    // "active" record absent there cannot prove an impact continuing today.
    const incident = incidentActive(row) && !currentKeys.has(key(row)) ? { ...row, status: 'unknown' } : row
    const known = records.get(key(incident))
    if (!known || incidentTime(incident) >= incidentTime(known)) records.set(key(incident), incident)
  }
  return [...records.values()]
}

const incidentTime = (incident: PublicStatusIncident): number => {
  const value = Date.parse(incident.latestUpdate?.updatedAt || incident.updatedAt || incident.startedAt || incident.createdAt || '')
  return Number.isFinite(value) ? value : 0
}

export function incidentActive(incident: PublicStatusIncident): boolean {
  return incident.status === 'investigating' || incident.status === 'identified' || incident.status === 'monitoring'
}

export function incidentPhase(incident: PublicStatusIncident): number | null {
  const phases: Record<string, number> = { investigating: 0, identified: 1, monitoring: 2, resolved: 3, postmortem: 3 }
  return phases[incident.status] ?? null
}

export function incidentElapsedMinutes(incident: PublicStatusIncident, now: number): number | null {
  const start = Date.parse(incident.startedAt || '')
  const end = incidentActive(incident) ? now : Date.parse(incident.resolvedAt || '')
  return Number.isFinite(start) && Number.isFinite(end) && start <= end && end <= now + 30_000 ? Math.floor((end - start) / 60_000) : null
}

/** Known scoped incidents may contradict an all-operational component snapshot. */
export function serviceStatusTone(components: PublicStatusComponent[], incidents: PublicStatusIncident[], fresh: boolean): PublicComponentStatus {
  const componentTone = publicStatusTone(components, fresh)
  if (!fresh || !incidents.some(incidentActive)) return componentTone
  const incidentComponents = incidents.filter(incidentActive).map(incident => ({ name: incident.name, status: publicIncidentTone(incident) }))
  return publicStatusTone([...components, ...incidentComponents], fresh)
}

export function publicIncidentTone(incident: Pick<PublicStatusIncident, 'impact'>): PublicComponentStatus {
  const impacts: Record<string, PublicComponentStatus> = { minor: 'degraded_performance', major: 'partial_outage', critical: 'major_outage', maintenance: 'under_maintenance' }
  return impacts[incident.impact] ?? 'unknown'
}

export type PublicStatusColor = 'green' | 'yellow' | 'red' | 'gray'

/** The same severity colors apply to current checks and documented day records. */
export function publicStatusColor(status: PublicComponentStatus): PublicStatusColor {
  if (status === 'operational') return 'green'
  if (status === 'degraded_performance' || status === 'under_maintenance') return 'yellow'
  if (status === 'partial_outage' || status === 'major_outage') return 'red'
  return 'gray'
}

const incidentRecordBounds = (incident: PublicStatusIncident, now: number): [number, number] | null => {
  const start = Date.parse(incident.startedAt || incident.createdAt || incident.updatedAt || '')
  if (!Number.isFinite(start) || start > now) return null
  const resolved = Date.parse(incident.resolvedAt || '')
  // Unknown resolution time proves a record on its start day only.
  const end = incidentActive(incident) ? now : Number.isFinite(resolved) && resolved >= start && resolved <= now ? resolved : start
  return [start, end]
}

export function publicHistoryIncidents(incidents: PublicStatusIncident[], now: number): PublicStatusIncident[] {
  const dayMs = 24 * 60 * 60 * 1000
  const firstDay = Math.floor(now / dayMs) * dayMs - 89 * dayMs
  return incidents.filter(incident => {
    const bounds = incidentRecordBounds(incident, now)
    return !!bounds && bounds[1] >= firstDay
  })
}

/** UTC days with official scoped incident records; empty days remain unknown. */
export function publicHistoryDays(incidents: PublicStatusIncident[], now: number): PublicHistoryDay[] {
  const dayMs = 24 * 60 * 60 * 1000
  const today = Math.floor(now / dayMs) * dayMs
  const days = Array.from({ length: 90 }, (_, index): PublicHistoryDay => ({ date: new Date(today - (89 - index) * dayMs).toISOString().slice(0, 10), incidents: [], records: [], tone: 'unknown' }))
  for (const incident of incidents) {
    const bounds = incidentRecordBounds(incident, now)
    if (!bounds) continue
    const [start, end] = bounds
    const firstDay = Math.floor(start / dayMs) * dayMs
    const lastDay = Math.floor(end / dayMs) * dayMs
    for (const day of days) {
      const date = Date.parse(`${day.date}T00:00:00Z`)
      if (date >= firstDay && date <= lastDay) {
        if (!day.incidents.includes(incident.name)) day.incidents.push(incident.name)
        if (!day.records.some(record => record.id && incident.id ? record.id === incident.id : record === incident)) day.records.push(incident)
        day.tone = publicStatusTone(day.records.map(record => ({ name: record.name, status: publicIncidentTone(record) })), true)
      }
    }
  }
  return days
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
