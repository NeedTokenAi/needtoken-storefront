/** Short-lived page memory only. Checkout always revalidates native prices. */
export function createCatalogRequestCache(ttl = 30_000, now: () => number = Date.now) {
  let activeScope = ''
  let epoch = 0
  const entries = new Map<string, { value?: unknown; expires: number; pending?: Promise<unknown> }>()

  const activate = (scope: string) => {
    if (scope === activeScope) return
    activeScope = scope
    epoch++
    entries.clear()
  }
  const peek = <T>(scope: string, key: string): T | undefined => {
    activate(scope)
    const entry = entries.get(key)
    return entry && entry.expires > now() ? entry.value as T : undefined
  }
  const read = async <T>(scope: string, key: string, load: () => Promise<T>, force = false): Promise<T> => {
    activate(scope)
    const previous = entries.get(key)
    if (!force && previous?.pending) return previous.pending as Promise<T>
    if (!force && previous && previous.expires > now()) return previous.value as T
    const ticket = epoch
    const entry: { value?: T; expires: number; pending?: Promise<T> } = { expires: 0 }
    const pending = load().then(value => {
      if (ticket === epoch && entries.get(key) === entry) {
        entry.value = value
        entry.expires = now() + ttl
        delete entry.pending
      }
      return value
    }, error => {
      if (entries.get(key) === entry) entries.delete(key)
      throw error
    })
    entry.pending = pending
    entries.delete(key)
    entries.set(key, entry)
    // Bound memory while retaining request deduplication for active requests.
    if (entries.size > 24) {
      const oldest = [...entries].find(([, row]) => !row.pending)
      if (oldest) entries.delete(oldest[0])
    }
    return pending
  }
  return { activate, peek, read }
}
