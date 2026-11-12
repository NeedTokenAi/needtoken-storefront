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
  name: string
  status: PublicComponentStatus
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
      return [{ name, status: componentStatus }]
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
