import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRoute, useRouter, type LocationQueryRaw } from 'vue-router'
import { categoryAPI, productAPI } from '../../../api'
import type { PublicCategory } from '../../../utils/category'
import type { ServiceProduct } from '../lib/serviceCatalog'
import { apiPlanAudience, filterAPIPlanProducts, resolveAPIPlanScope, type APIPlanAudience } from '../lib/apiPlanCatalog'
import { CATEGORY_SLUGS } from '../config'

const queryText = (value: unknown) => typeof value === 'string' ? value.trim() : ''
const queryPage = (value: unknown) => {
  const raw = queryText(value)
  const page = Number(raw)
  return /^\d+$/.test(raw) && Number.isSafeInteger(page) && page > 0 ? page : 1
}

export function useApiPlanCatalog() {
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
  const searchQuery = ref(queryText(route.query.q))
  const audience = computed(() => apiPlanAudience(route.query.kind))
  const scope = computed(() => resolveAPIPlanScope(categories.value, audience.value, CATEGORY_SLUGS.plans))
  const page = computed(() => queryPage(route.query.page))
  let request = 0
  let disposed = false

  const loadProducts = async () => {
    const ticket = ++request
    products.value = []
    totalPages.value = 0
    totalProducts.value = 0
    loadError.value = false
    // A missing root/selected category must never become the entire storefront.
    if (!scope.value.category) { loading.value = false; return }
    const requestedScope = scope.value
    loading.value = true
    try {
      const response = await productAPI.list({ category_id: requestedScope.category!.id, page: page.value, page_size: 6, search: queryText(route.query.q) || undefined })
      if (disposed || ticket !== request) return
      if (!Array.isArray(response.data.data)) throw new Error('Invalid API plan catalog response')
      products.value = filterAPIPlanProducts(response.data.data, requestedScope)
      const pages = Number(response.data.pagination?.total_page)
      const count = Number(response.data.pagination?.total)
      totalPages.value = Number.isSafeInteger(pages) && pages >= 0 ? pages : 0
      totalProducts.value = Number.isSafeInteger(count) && count >= 0 ? count : products.value.length
      if (totalPages.value && page.value > totalPages.value) await router.replace({ query: { ...route.query, page: String(totalPages.value) } })
    } catch {
      if (!disposed && ticket === request) loadError.value = true
    } finally {
      if (!disposed && ticket === request) loading.value = false
    }
  }
  const loadCategories = async () => {
    categoriesLoading.value = true
    categoriesError.value = false
    try {
      const response = await categoryAPI.list()
      if (disposed) return
      if (!Array.isArray(response.data.data)) throw new Error('Invalid API plan categories')
      categories.value = response.data.data
      categoriesLoading.value = false
      await loadProducts()
    } catch {
      if (!disposed) categoriesError.value = true
    } finally {
      if (!disposed) categoriesLoading.value = false
    }
  }
  const selectAudience = (next: APIPlanAudience) => {
    if (next === audience.value) return
    const query: LocationQueryRaw = { ...route.query, kind: next }
    delete query.page
    return router.push({ path: '/plans', query, hash: '#api-plan-catalog' })
  }
  const changePage = (next: number) => {
    if (!Number.isSafeInteger(next) || next < 1 || next > totalPages.value) return
    return router.push({ query: { ...route.query, page: String(next) }, hash: '#api-plan-catalog' })
  }
  const submitSearch = () => {
    const query = { ...route.query }
    delete query.page
    if (searchQuery.value.trim()) query.q = searchQuery.value.trim()
    else delete query.q
    return router.push({ query, hash: '#api-plan-catalog' })
  }
  const clearSearch = () => { searchQuery.value = ''; return submitSearch() }
  watch(() => [audience.value, route.query.page, route.query.q], () => {
    searchQuery.value = queryText(route.query.q)
    if (!categoriesLoading.value && !categoriesError.value) void loadProducts()
  })
  onMounted(loadCategories)
  onUnmounted(() => { disposed = true; request++ })
  return { audience, scope, categoriesLoading, categoriesError, products, loading, loadError, totalPages, totalProducts, page, searchQuery, loadCategories, loadProducts, selectAudience, changePage, submitSearch, clearSearch }
}
