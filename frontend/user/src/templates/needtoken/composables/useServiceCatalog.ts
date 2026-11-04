import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { categoryAPI, productAPI } from '../../../api'
import type { PublicCategory } from '../../../utils/category'
import {
  filterServiceProducts,
  resolveServiceScope,
  serviceKindFromSlug,
  type ServiceProduct,
} from '../lib/serviceCatalog'

const firstQueryValue = (value: unknown) => String(Array.isArray(value) ? value[0] || '' : value || '').trim()
const readPage = (value: unknown) => {
  const text = firstQueryValue(value)
  const number = Number(text)
  return /^\d+$/.test(text) && Number.isSafeInteger(number) && number > 0 ? number : 1
}

export function useServiceCatalog() {
  const route = useRoute()
  const router = useRouter()
  const categories = ref<PublicCategory[]>([])
  const categoriesLoading = ref(true)
  const categoriesError = ref(false)
  const products = ref<ServiceProduct[]>([])
  const loading = ref(false)
  const loadError = ref(false)
  const totalPages = ref(0)
  const totalProducts = ref(0)
  const searchQuery = ref(firstQueryValue(route.query.q))
  const slug = computed(() => String(route.params.slug || ''))
  const scope = computed(() => resolveServiceScope(categories.value, slug.value))
  const serviceKind = computed(() => scope.value?.kind || serviceKindFromSlug(slug.value))
  const page = computed(() => readPage(route.query.page))
  let generation = 0
  let disposed = false

  const loadProducts = async () => {
    const ticket = ++generation
    const currentScope = scope.value
    products.value = []
    totalPages.value = 0
    totalProducts.value = 0
    loadError.value = false
    // Never send an unscoped request when the configured service category is missing.
    if (!currentScope?.root || !currentScope.selected) {
      loading.value = false
      return
    }
    loading.value = true
    try {
      const response = await productAPI.list({
        category_id: currentScope.selected.id,
        page: page.value,
        page_size: 6,
        search: firstQueryValue(route.query.q) || undefined,
      })
      if (disposed || ticket !== generation) return
      if (!Array.isArray(response.data.data)) throw new Error('Invalid public product response')
      const rows = response.data.data
      products.value = filterServiceProducts(rows, currentScope)
      const pages = Number(response.data.pagination?.total_page)
      const total = Number(response.data.pagination?.total)
      totalPages.value = Number.isSafeInteger(pages) && pages >= 0 ? pages : 0
      totalProducts.value = Number.isSafeInteger(total) && total >= 0 ? total : products.value.length
      if (totalPages.value > 0 && page.value > totalPages.value) {
        await router.replace({ query: { ...route.query, page: String(totalPages.value) } })
      }
    } catch {
      if (!disposed && ticket === generation) loadError.value = true
    } finally {
      if (!disposed && ticket === generation) loading.value = false
    }
  }

  const loadCategories = async () => {
    categoriesLoading.value = true
    categoriesError.value = false
    try {
      const response = await categoryAPI.list()
      if (disposed) return
      if (!Array.isArray(response.data.data)) throw new Error('Invalid public category response')
      categories.value = response.data.data
      categoriesLoading.value = false
      await loadProducts()
    } catch {
      if (!disposed) categoriesError.value = true
    } finally {
      if (!disposed) categoriesLoading.value = false
    }
  }

  const selectCategory = (category: PublicCategory | null) => {
    if (!scope.value?.root) return
    const target = category || scope.value.root
    const query = { ...route.query }
    delete query.page
    return router.push({ path: `/categories/${target.slug}`, query })
  }
  const changePage = (nextPage: number) => {
    if (!Number.isSafeInteger(nextPage) || nextPage < 1 || nextPage > totalPages.value) return
    const query = { ...route.query }
    if (nextPage > 1) query.page = String(nextPage)
    else delete query.page
    return router.push({ query })
  }
  const submitSearch = () => {
    const query = { ...route.query }
    delete query.page
    const text = searchQuery.value.trim()
    if (text) query.q = text
    else delete query.q
    return router.push({ query })
  }
  const clearSearch = () => {
    searchQuery.value = ''
    return submitSearch()
  }

  watch(() => [slug.value, route.query.page, route.query.q], () => {
    searchQuery.value = firstQueryValue(route.query.q)
    if (!categoriesLoading.value && !categoriesError.value) void loadProducts()
  })
  onMounted(() => { void loadCategories() })
  onUnmounted(() => { disposed = true; generation += 1 })

  return {
    scope, serviceKind, categoriesLoading, categoriesError, products, loading, loadError,
    page, totalPages, totalProducts, searchQuery, loadCategories, loadProducts,
    selectCategory, changePage, submitSearch, clearSearch,
  }
}
