import { computed, reactive, ref } from 'vue'
import { RANKINGS_URL } from '../config'
import {
  EMPTY_RANKINGS_SNAPSHOT,
  normalizeRankingPeriod,
  normalizeRankingsResponse,
  type RankingPeriod,
  type RankingsApiResponse,
  type RankingsSnapshot,
} from './rankings'

type LoadState = 'idle' | 'loading' | 'ready' | 'error'

interface PeriodEntry {
  state: LoadState
  snapshot: RankingsSnapshot
  loadedAt: Date | null
  isStale: boolean
}

const createEntry = (): PeriodEntry => ({
  state: 'idle',
  snapshot: { ...EMPTY_RANKINGS_SNAPSHOT },
  loadedAt: null,
  isStale: false,
})

const entries = reactive<Record<RankingPeriod, PeriodEntry>>({
  today: createEntry(),
  week: createEntry(),
  month: createEntry(),
  year: createEntry(),
})

const inflightByPeriod = new Map<RankingPeriod, Promise<void>>()

async function fetchPeriod(period: RankingPeriod): Promise<void> {
  const entry = entries[period]
  entry.state = 'loading'
  try {
    const url = `${RANKINGS_URL}?period=${encodeURIComponent(period)}`
    const res = await fetch(url, { headers: { Accept: 'application/json' }, credentials: 'omit' })
    if (!res.ok) throw new Error(`rankings http ${res.status}`)
    const staleHeader = res.headers.get('X-Storefront-Stale') === '1'
    const body = (await res.json()) as RankingsApiResponse
    if (body.success === false) throw new Error(body.error_code || body.message || 'rankings unavailable')
    entry.snapshot = normalizeRankingsResponse(body)
    entry.loadedAt = new Date()
    entry.isStale = staleHeader
    entry.state = 'ready'
  } catch {
    entry.state = 'error'
  }
}

function loadPeriod(period: RankingPeriod, force = false): Promise<void> {
  const normalized = normalizeRankingPeriod(period)
  const entry = entries[normalized]
  if (!force && entry.state === 'ready') {
    return Promise.resolve()
  }
  const existing = inflightByPeriod.get(normalized)
  if (existing) {
    return existing
  }
  const promise = fetchPeriod(normalized).finally(() => {
    inflightByPeriod.delete(normalized)
  })
  inflightByPeriod.set(normalized, promise)
  return promise
}

export function useRankings(initialPeriod: RankingPeriod = 'week') {
  const period = ref<RankingPeriod>(normalizeRankingPeriod(initialPeriod))

  const current = computed(() => entries[period.value])
  const snapshot = computed(() => current.value.snapshot)
  const state = computed(() => current.value.state)
  const loadedAt = computed(() => current.value.loadedAt)
  const isStale = computed(() => current.value.isStale)
  const isLoading = computed(() => {
    const e = current.value
    const hasData = e.snapshot.models.length > 0 || e.snapshot.vendors.length > 0
    return (e.state === 'loading' || e.state === 'idle') && !hasData
  })
  const isRefreshing = computed(() => current.value.state === 'loading')

  const setPeriod = (next: RankingPeriod) => {
    const normalized = normalizeRankingPeriod(next)
    period.value = normalized
    return loadPeriod(normalized, false)
  }

  const ensureLoaded = (target?: RankingPeriod) => {
    const p = target ? normalizeRankingPeriod(target) : period.value
    if (target) period.value = p
    return loadPeriod(p, false)
  }

  const reload = (target?: RankingPeriod) => {
    const p = target ? normalizeRankingPeriod(target) : period.value
    return loadPeriod(p, true)
  }

  return {
    period,
    snapshot,
    state,
    loadedAt,
    isStale,
    isLoading,
    isRefreshing,
    setPeriod,
    ensureLoaded,
    reload,
  }
}
