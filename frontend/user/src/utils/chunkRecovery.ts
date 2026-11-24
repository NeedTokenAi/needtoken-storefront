const CHUNK_RELOAD_STORAGE_KEY = 'dj-chunk-reload-ts'
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

export function clearChunkReloadGuard(): void {
  if (typeof window === 'undefined') return
  try {
    window.sessionStorage.removeItem(CHUNK_RELOAD_STORAGE_KEY)
  } catch {
    // ignore storage errors
  }
}

/**
 * 检测到发版后旧 chunk 失效时，在冷却窗口内自动整页刷新一次以拉取最新 index.html。
 * 返回 true 表示已触发跳转/刷新；返回 false 表示冷却期内已刷新过，交由 UI 提示用户手动重试。
 */
export function recoverFromChunkError(targetPath?: string): boolean {
  if (typeof window === 'undefined') return false
  if (!canTriggerChunkReload()) return false
  if (targetPath && targetPath.startsWith('/') && !targetPath.startsWith('//')) {
    window.location.assign(targetPath)
    return true
  }
  window.location.reload()
  return true
}

/**
 * 包装异步组件加载器：遇到动态模块加载失败先自动刷新恢复，避免整页落入死循环错误边界。
 */
export function lazyImportWithRetry<T>(loader: () => Promise<T>): () => Promise<T> {
  return () =>
    loader().catch((err) => {
      if (isChunkLoadError(err) && recoverFromChunkError()) {
        return new Promise<T>(() => {
          // 页面正在整页重载，保持挂起避免闪现原始错误卡片
        })
      }
      throw err
    })
}

let preloadHandlerInstalled = false

/**
 * 监听 Vite 预加载失败事件（vite:preloadError），发版后自动拉取新版入口。
 */
export function installVitePreloadErrorHandler(): void {
  if (typeof window === 'undefined' || preloadHandlerInstalled) return
  preloadHandlerInstalled = true
  window.addEventListener('vite:preloadError', (event) => {
    event.preventDefault()
    recoverFromChunkError()
  })
}
