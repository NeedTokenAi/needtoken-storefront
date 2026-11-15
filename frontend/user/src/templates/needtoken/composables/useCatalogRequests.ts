import { computed, watch } from 'vue'
import { useAppStore } from '../../../stores/app'
import { useUserAuthStore } from '../../../stores/userAuth'
import { createCatalogRequestCache } from '../lib/catalogRequestCache'

const requests = createCatalogRequestCache()

export function useCatalogRequests() {
  const app = useAppStore()
  const auth = useUserAuthStore()
  // A different identity, member profile, locale or tenant/configuration must
  // never reuse another visitor's presentation or prices. Nothing is persisted.
  const scopeKey = computed(() => JSON.stringify([auth.token, auth.user, app.locale, app.config]))
  watch(scopeKey, value => requests.activate(value), { immediate: true, flush: 'sync' })
  return {
    scopeKey,
    read: <T>(key: string, load: () => Promise<T>, force = false) => requests.read(scopeKey.value, key, load, force),
    peek: <T>(key: string) => requests.peek<T>(scopeKey.value, key),
  }
}
