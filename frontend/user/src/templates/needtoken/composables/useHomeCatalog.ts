import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { categoryAPI, productAPI } from '../../../api'
import type { PublicCategory } from '../../../utils/category'
import { sortHomeProductsByPrice } from '../lib/homeCardPresentation'
import {
  homeServiceSelectionId,
  filterServiceProducts,
  resolveServiceScope,
  serviceCardProducts,
  type ServiceKind,
  type ServiceCardProduct,
  type ServiceScope,
} from '../lib/serviceCatalog'
import { useCatalogRequests } from './useCatalogRequests'

export function useHomeCatalog() {
  const categories = ref<PublicCategory[]>([])
  const products = ref<ServiceCardProduct[]>([])
  const requests = useCatalogRequests()
  const categoriesLoading = ref(true)
  const productsLoading = ref(false)
  const loading = computed(() => categoriesLoading.value || productsLoading.value)
  const error = ref(false)
  const kind = ref<ServiceKind>('recharge')
  const selectedId = ref(0)
  let generation = 0
  let categoryGeneration = 0
  let disposed = false

  const scope = computed(() => resolveServiceScope(categories.value, kind.value))
  const childCategories = computed(() => scope.value?.categories ?? [])

  const applySelection = (preserve = false) => {
    selectedId.value = homeServiceSelectionId(scope.value, preserve ? selectedId.value : 0)
  }

  const requestProducts = (id: number, force = false) => requests.read(`products:${id}:1:12`, async () => {
    const response = await productAPI.list({ category_id: id, page: 1, page_size: 12 })
    if (!Array.isArray(response.data.data)) throw new Error('invalid public product response')
    return response
  }, force)

  const prefetchCategory = (id: number) => {
    if (categoriesLoading.value || !scope.value?.categories.some(row => row.id === id)) return
    void requestProducts(id).catch(() => {})
  }

  const loadProducts = async (force = false) => {
    // A service tab can change before the first category request returns.
    // Keep the catalogue skeleton until that request can resolve its scope.
    if (categoriesLoading.value) return
    const ticket = ++generation
    const current = scope.value
    products.value = []
    error.value = false
    productsLoading.value = true
    if (!current?.root || !selectedId.value) {
      productsLoading.value = false
      return
    }
    const selected = selectedId.value === current.root.id
      ? current.root
      : current.categories.find(row => row.id === selectedId.value) || null
    if (!selected) {
      productsLoading.value = false
      return
    }
    try {
      const response = await requestProducts(selected.id, force)
      if (disposed || ticket !== generation) return
      if (!Array.isArray(response.data.data)) throw new Error('invalid public product response')
      const scoped: ServiceScope = { ...current, selected }
      products.value = sortHomeProductsByPrice(serviceCardProducts(filterServiceProducts(response.data.data, scoped), kind.value))
    } catch {
      if (!disposed && ticket === generation) error.value = true
    } finally {
      if (!disposed && ticket === generation) productsLoading.value = false
    }
  }

  const load = async (force = false) => {
    const ticket = ++categoryGeneration
    generation += 1
    categoriesLoading.value = true
    productsLoading.value = false
    products.value = []
    error.value = false
    try {
      const response = await requests.read('categories', async () => {
        const response = await categoryAPI.list()
        if (!Array.isArray(response.data.data)) throw new Error('invalid public category response')
        return response
      }, force)
      if (disposed || ticket !== categoryGeneration) return
      if (!Array.isArray(response.data.data)) throw new Error('invalid public category response')
      categories.value = response.data.data
      applySelection(true)
      categoriesLoading.value = false
      await loadProducts(force)
    } catch {
      if (!disposed && ticket === categoryGeneration) {
        error.value = true
      }
    } finally {
      if (!disposed && ticket === categoryGeneration) categoriesLoading.value = false
    }
  }

  const setKind = (next: ServiceKind) => {
    if (kind.value === next) return
    kind.value = next
    applySelection()
    void loadProducts()
  }

  const selectCategory = (id: number) => {
    if (categoriesLoading.value || !scope.value?.root) return
    const allowed = id === scope.value.root.id || scope.value.categories.some(row => row.id === id)
    if (!allowed || selectedId.value === id) return
    selectedId.value = id
    void loadProducts()
  }

  onMounted(() => { void load() })
  watch(requests.scopeKey, () => { if (!disposed) void load() })
  onUnmounted(() => {
    disposed = true
    generation += 1
    categoryGeneration += 1
  })

  return {
    kind,
    scope,
    childCategories,
    selectedId,
    products,
    categoriesLoading,
    loading,
    error,
    setKind,
    selectCategory,
    prefetchCategory,
    reload: () => load(true),
  }
}
