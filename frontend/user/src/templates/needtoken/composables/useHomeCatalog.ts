import { computed, onMounted, onUnmounted, ref } from 'vue'
import { categoryAPI, productAPI } from '../../../api'
import type { PublicCategory } from '../../../utils/category'
import { sortHomeProductsByPrice } from '../lib/homeCardPresentation'
import {
  defaultServiceChild,
  filterServiceProducts,
  resolveServiceScope,
  type ServiceKind,
  type ServiceProduct,
  type ServiceScope,
} from '../lib/serviceCatalog'

export function useHomeCatalog() {
  const categories = ref<PublicCategory[]>([])
  const products = ref<ServiceProduct[]>([])
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

  const applyDefaultSelection = () => {
    const current = scope.value
    if (!current?.root) {
      selectedId.value = 0
      return
    }
    const child = defaultServiceChild({ ...current, selected: current.root }) || current.categories[0]
    selectedId.value = child?.id || current.root.id
  }

  const loadProducts = async () => {
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
      const response = await productAPI.list({ category_id: selected.id, page: 1, page_size: 12 })
      if (disposed || ticket !== generation) return
      if (!Array.isArray(response.data.data)) throw new Error('invalid public product response')
      const scoped: ServiceScope = { ...current, selected }
      products.value = sortHomeProductsByPrice(filterServiceProducts(response.data.data, scoped))
    } catch {
      if (!disposed && ticket === generation) error.value = true
    } finally {
      if (!disposed && ticket === generation) productsLoading.value = false
    }
  }

  const load = async () => {
    const ticket = ++categoryGeneration
    generation += 1
    categoriesLoading.value = true
    productsLoading.value = false
    products.value = []
    error.value = false
    try {
      const response = await categoryAPI.list()
      if (disposed || ticket !== categoryGeneration) return
      if (!Array.isArray(response.data.data)) throw new Error('invalid public category response')
      categories.value = response.data.data
      applyDefaultSelection()
      categoriesLoading.value = false
      await loadProducts()
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
    applyDefaultSelection()
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
    reload: load,
  }
}
