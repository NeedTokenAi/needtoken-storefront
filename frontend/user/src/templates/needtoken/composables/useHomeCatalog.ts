import { computed, onMounted, onUnmounted, ref } from 'vue'
import { categoryAPI, productAPI } from '../../../api'
import type { PublicCategory } from '../../../utils/category'
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
  const loading = ref(true)
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
    const child = defaultServiceChild({ ...current, selected: current.root })
    selectedId.value = child?.id || current.root.id
  }

  const loadProducts = async () => {
    const ticket = ++generation
    const current = scope.value
    products.value = []
    error.value = false
    if (!current?.root || !selectedId.value) {
      loading.value = false
      return
    }
    const selected = selectedId.value === current.root.id
      ? current.root
      : current.categories.find(row => row.id === selectedId.value) || null
    if (!selected) {
      loading.value = false
      return
    }
    loading.value = true
    try {
      const response = await productAPI.list({ category_id: selected.id, page: 1, page_size: 12 })
      if (disposed || ticket !== generation) return
      if (!Array.isArray(response.data.data)) throw new Error('invalid public product response')
      const scoped: ServiceScope = { ...current, selected }
      products.value = filterServiceProducts(response.data.data, scoped)
    } catch {
      if (!disposed && ticket === generation) error.value = true
    } finally {
      if (!disposed && ticket === generation) loading.value = false
    }
  }

  const load = async () => {
    const ticket = ++categoryGeneration
    loading.value = true
    error.value = false
    try {
      const response = await categoryAPI.list()
      if (disposed || ticket !== categoryGeneration) return
      if (!Array.isArray(response.data.data)) throw new Error('invalid public category response')
      categories.value = response.data.data
      applyDefaultSelection()
      await loadProducts()
    } catch {
      if (!disposed && ticket === categoryGeneration) {
        error.value = true
        loading.value = false
      }
    }
  }

  const setKind = (next: ServiceKind) => {
    if (kind.value === next) return
    kind.value = next
    applyDefaultSelection()
    void loadProducts()
  }

  const selectCategory = (id: number) => {
    if (!scope.value?.root) return
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
    loading,
    error,
    setKind,
    selectCategory,
    reload: load,
  }
}
