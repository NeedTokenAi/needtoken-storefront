const CHUNK_RELOAD_STORAGE_KEY = 'dj-admin-chunk-reload-ts'
const CHUNK_RELOAD_COOLDOWN_MS = 30_000

const CHUNK_ERROR_PATTERNS = [
  'failed to fetch dynamically imported module',
  'error loading dynamically imported module',
  'importing a module script failed',
  'loading chunk',
  'loading css chunk',
  'unable to preload css',
  'dynamically imported module',
]

export function isChunkLoadError(error: unknown): boolean {
  if (!error) return false
  const name = typeof (error as { name?: unknown }).name === 'string'
    ? (error as { name: string }).name.toLowerCase()
    : ''
  if (name === 'chunkloaderror') return true
  const message = typeof error === 'string'
    ? error.toLowerCase()
    : typeof (error as { message?: unknown }).message === 'string'
      ? (error as { message: string }).message.toLowerCase()
      : ''
  if (!message) return false
  return CHUNK_ERROR_PATTERNS.some((pattern) => message.includes(pattern))
}

function canTriggerChunkReload(now = Date.now()): boolean {
  if (typeof window === 'undefined') return false
  try {
    const raw = window.sessionStorage.getItem(CHUNK_RELOAD_STORAGE_KEY)
    const lastTs = raw ? Number(raw) : 0
    if (Number.isFinite(lastTs) && lastTs > 0 && now - lastTs < CHUNK_RELOAD_COOLDOWN_MS) {
      return false
    }
    window.sessionStorage.setItem(CHUNK_RELOAD_STORAGE_KEY, String(now))
    return true
  } catch {
    return false
  }
}

export function recoverFromChunkError(targetUrl?: string): boolean {
  if (typeof window === 'undefined') return false
  if (!canTriggerChunkReload()) return false
  if (targetUrl && targetUrl.startsWith('/') && !targetUrl.startsWith('//')) {
    window.location.assign(targetUrl)
    return true
  }
  window.location.reload()
  return true
}

let preloadHandlerInstalled = false

export function installVitePreloadErrorHandler(): void {
  if (typeof window === 'undefined' || preloadHandlerInstalled) return
  preloadHandlerInstalled = true
  window.addEventListener('vite:preloadError', (event) => {
    event.preventDefault()
    recoverFromChunkError()
  })
}
