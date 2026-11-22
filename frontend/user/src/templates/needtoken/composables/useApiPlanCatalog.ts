import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRoute, useRouter, type LocationQueryRaw } from 'vue-router'
import { categoryAPI, productAPI } from '../../../api'
import type { PublicCategory } from '../../../utils/category'
import type { ServiceProduct } from '../lib/serviceCatalog'
import { filterAPIPlanProducts, resolveUnifiedAPIPlanScope } from '../lib/apiPlanCatalog'
import { CATEGORY_SLUGS } from '../config'

const queryText = (value: unknown) => typeof value === 'string' ? value.trim() : ''
const queryPage = (value: unknown) => {
  const raw = queryText(value)
  const page = Number(raw)
  return /^\d+$/.test(raw) && Number.isSafeInteger(page) && page > 0 ? page : 1
}

export function useApiPlanCatalog(options?: { pageSize?: number; syncRoute?: boolean }) {
  const pageSize = options?.pageSize ?? 12
  const syncRoute = options?.syncRoute !== false
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
  const localCategory = ref('')
  const localPage = ref(1)
  const searchQuery = ref(syncRoute ? queryText(route.query.q) : '')

  const selectedCategorySlug = computed(() => (syncRoute ? queryText(route.query.cat) : localCategory.value))
  const scope = computed(() => resolveUnifiedAPIPlanScope(categories.value, selectedCategorySlug.value, CATEGORY_SLUGS.plans))
  const subcategories = computed(() => scope.value.subcategories)
  const page = computed(() => (syncRoute ? queryPage(route.query.page) : localPage.value))
  let request = 0
  let disposed = false

  const loadProducts = async () => {
    const ticket = ++request
    products.value = []
    totalPages.value = 0
    totalProducts.value = 0
    loadError.value = false
    const requestedScope = scope.value
    const targetCategory = requestedScope.selected || requestedScope.root
    // A missing root category must never expand to the entire storefront.
    if (!targetCategory) { loading.value = false; return }
    loading.value = true
    try {
      const searchParam = syncRoute ? queryText(route.query.q) : searchQuery.value.trim()
      const response = await productAPI.list({
        category_id: targetCategory.id,
        page: page.value,
        page_size: pageSize,
        search: searchParam || undefined,
      })
      if (disposed || ticket !== request) return
      if (!Array.isArray(response.data.data)) throw new Error('Invalid API plan catalog response')
      products.value = filterAPIPlanProducts(response.data.data, requestedScope)
      const pages = Number(response.data.pagination?.total_page)
      const count = Number(response.data.pagination?.total)
      totalPages.value = Number.isSafeInteger(pages) && pages >= 0 ? pages : 0
      totalProducts.value = Number.isSafeInteger(count) && count >= 0 ? count : products.value.length
      if (syncRoute && totalPages.value && page.value > totalPages.value) {
        await router.replace({ query: { ...route.query, page: String(totalPages.value) } })
      }
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

  const selectCategory = (slug: string) => {
    const clean = String(slug || '').trim()
    if (clean === selectedCategorySlug.value) return
    if (!syncRoute) {
      localCategory.value = clean
      localPage.value = 1
      void loadProducts()
      return
    }
    const query: LocationQueryRaw = { ...route.query }
    delete query.page
    delete query.kind
    if (clean) query.cat = clean
    else delete query.cat
    return router.push({ path: '/plans', query, hash: '#api-plan-catalog' })
  }

  const changePage = (next: number) => {
    if (!Number.isSafeInteger(next) || next < 1 || next > totalPages.value) return
    if (!syncRoute) {
      localPage.value = next
      void loadProducts()
      return
    }
    return router.push({ query: { ...route.query, page: String(next) }, hash: '#api-plan-catalog' })
  }

  const submitSearch = () => {
    if (!syncRoute) {
      localPage.value = 1
      void loadProducts()
      return
    }
    const query = { ...route.query }
    delete query.page
    if (searchQuery.value.trim()) query.q = searchQuery.value.trim()
    else delete query.q
    return router.push({ query, hash: '#api-plan-catalog' })
  }

  const clearSearch = () => { searchQuery.value = ''; return submitSearch() }

  if (syncRoute) {
    watch(() => [selectedCategorySlug.value, route.query.page, route.query.q], () => {
      searchQuery.value = queryText(route.query.q)
      if (!categoriesLoading.value && !categoriesError.value) void loadProducts()
    })
  }
  onMounted(loadCategories)
  onUnmounted(() => { disposed = true; request++ })
  return {
    selectedCategorySlug,
    subcategories,
    scope,
    categoriesLoading,
    categoriesError,
    products,
    loading,
    loadError,
    totalPages,
    totalProducts,
    page,
    searchQuery,
    loadCategories,
    loadProducts,
    selectCategory,
    changePage,
    submitSearch,
    clearSearch,
  }
}
