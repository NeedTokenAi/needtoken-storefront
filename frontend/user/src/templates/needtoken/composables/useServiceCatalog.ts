import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { categoryAPI, productAPI } from '../../../api'
import type { PublicCategory } from '../../../utils/category'
import { sortHomeProductsByPrice } from '../lib/homeCardPresentation'
import {
  defaultServiceChild,
  filterServiceProducts,
  resolveServiceScope,
  serviceKindFromSlug,
  serviceCardProducts,
  type ServiceCardProduct,
} from '../lib/serviceCatalog'
import { useCatalogRequests } from './useCatalogRequests'

const firstQueryValue = (value: unknown) => String(Array.isArray(value) ? value[0] || '' : value || '').trim()
const readPage = (value: unknown) => {
  const text = firstQueryValue(value)
  const number = Number(text)
  return /^\d+$/.test(text) && Number.isSafeInteger(number) && number > 0 ? number : 1
}

export function useServiceCatalog() {
  const route = useRoute()
  const router = useRouter()
  const requests = useCatalogRequests()
  const categories = ref<PublicCategory[]>([])
  const categoriesLoading = ref(true)
  const categoriesError = ref(false)
  const products = ref<ServiceCardProduct[]>([])
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
  let categoryGeneration = 0
  let disposed = false
  // Set when the visitor clicks ALL, so the root URL stays put instead of bouncing to ChatGPT.
  let explicitRootSlug: string | null = null

  const queryWithoutPage = () => {
    const query = { ...route.query }
    delete query.page
    return query
  }

  const redirectRootToDefaultChild = async () => {
    const child = defaultServiceChild(scope.value, explicitRootSlug)
    if (!child?.slug || child.slug === slug.value) return false
    await router.replace({ path: `/categories/${child.slug}`, query: queryWithoutPage() }).catch(() => {})
    return slug.value === child.slug
  }

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
      const params = {
        category_id: currentScope.selected.id,
        page: page.value,
        page_size: 15,
        search: firstQueryValue(route.query.q) || undefined,
      }
      const response = await requests.read(`products:${JSON.stringify(params)}`, async () => {
        const response = await productAPI.list(params)
        if (!Array.isArray(response.data.data)) throw new Error('Invalid public product response')
        return response
      })
      if (disposed || ticket !== generation) return
      if (!Array.isArray(response.data.data)) throw new Error('Invalid public product response')
      const rows = response.data.data
      products.value = sortHomeProductsByPrice(serviceCardProducts(filterServiceProducts(rows, currentScope), currentScope.kind))
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
    const ticket = ++categoryGeneration
    generation++
    categoriesLoading.value = true
    categoriesError.value = false
    try {
      const response = await requests.read('categories', async () => {
        const response = await categoryAPI.list()
        if (!Array.isArray(response.data.data)) throw new Error('Invalid public category response')
        return response
      })
      if (disposed || ticket !== categoryGeneration) return
      if (!Array.isArray(response.data.data)) throw new Error('Invalid public category response')
      const rows = response.data.data as PublicCategory[]
      const child = defaultServiceChild(resolveServiceScope(rows, slug.value), explicitRootSlug)
      if (child?.slug && child.slug !== slug.value) {
        await router.replace({ path: `/categories/${child.slug}`, query: queryWithoutPage() }).catch(() => {})
        if (disposed || ticket !== categoryGeneration) return
      }
      categories.value = rows
      categoriesLoading.value = false
      await loadProducts()
    } catch {
      if (!disposed && ticket === categoryGeneration) categoriesError.value = true
    } finally {
      if (!disposed && ticket === categoryGeneration) categoriesLoading.value = false
    }
  }

  const selectCategory = (category: PublicCategory | null) => {
    if (!scope.value?.root) return
    const target = category || scope.value.root
    explicitRootSlug = target.id === scope.value.root.id ? String(target.slug || '') : null
    return router.push({ path: `/categories/${target.slug}`, query: queryWithoutPage() })
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
    if (categoriesLoading.value || categoriesError.value) return
    void (async () => {
      if (await redirectRootToDefaultChild()) return
      if (!disposed) await loadProducts()
    })()
  })
  onMounted(() => { void loadCategories() })
  watch(requests.scopeKey, () => {
    generation++
    if (!disposed) void loadCategories()
  })
  onUnmounted(() => { disposed = true; generation += 1; categoryGeneration += 1 })

  return {
    scope, serviceKind, categoriesLoading, categoriesError, products, loading, loadError,
    page, totalPages, totalProducts, searchQuery, loadCategories, loadProducts,
    selectCategory, changePage, submitSearch, clearSearch,
  }
}
