<template>
  <ClassicProducts v-if="!categoriesLoading && !categoriesError && !serviceKind" />
  <div v-else class="nt-service-catalog mx-auto w-full max-w-[1200px] px-4 sm:px-6">
    <header class="nt-service-catalog__intro">
      <span class="nt-service-catalog__eyebrow">{{ t(`needtoken.catalogServices.${serviceKind || 'recharge'}.label`) }}</span>
      <h1>{{ t(`needtoken.catalogServices.${serviceKind || 'recharge'}.title`) }}</h1>
      <p class="nt-service-catalog__subtitle">{{ t(`needtoken.catalogServices.${serviceKind || 'recharge'}.subtitle`) }}</p>
      <ul class="nt-service-catalog__promises" :aria-label="t('needtoken.catalogServices.serviceNotes')">
        <li v-for="(note, index) in serviceNotes" :key="note"><component :is="noteIcons[index]" :size="15" aria-hidden="true" />{{ t(`needtoken.catalogServices.${note}`) }}</li>
      </ul>
    </header>

    <div v-if="categoriesError" role="alert" class="nt-service-catalog__state">
      <PackageSearch :size="30" aria-hidden="true" />
      <h2>{{ t('needtoken.catalogServices.loadError') }}</h2>
      <button type="button" @click="loadCategories">{{ t('needtoken.common.retry') }}</button>
    </div>
    <template v-else>
      <div v-if="scope?.root" class="nt-service-catalog__toolbar">
        <nav class="nt-service-catalog__filters" :aria-label="t('needtoken.catalogServices.filterLabel')">
          <button type="button" class="nt-service-catalog__filter" :class="{ 'is-active': scope.selected?.id === scope.root.id }" :aria-pressed="scope.selected?.id === scope.root.id" @click="selectCategory(null)">{{ t('needtoken.catalogServices.all') }}</button>
          <button v-for="category in scope.categories" :key="category.id" type="button" class="nt-service-catalog__filter" :class="{ 'is-active': scope.selected?.id === category.id }" :aria-pressed="scope.selected?.id === category.id" @click="selectCategory(category)">{{ getLocalizedText(category.name) }}</button>
        </nav>
        <form class="nt-service-catalog__search" role="search" @submit.prevent="submitSearch">
          <Search :size="16" aria-hidden="true" />
          <input v-model="searchQuery" type="search" :placeholder="t('needtoken.catalogServices.search')" :aria-label="t('needtoken.catalogServices.search')" />
          <button type="submit" :aria-label="t('needtoken.catalogServices.searchSubmit')"><ArrowRight :size="16" aria-hidden="true" /></button>
        </form>
      </div>

      <div v-if="categoriesLoading || loading" class="nt-service-catalog__grid" role="status" :aria-label="t('needtoken.common.loading')" aria-busy="true">
        <div v-for="index in 6" :key="index" class="nt-service-catalog__skeleton"><div /><div /><div /><div /></div>
      </div>
      <div v-else-if="loadError" role="alert" class="nt-service-catalog__state">
        <PackageSearch :size="30" aria-hidden="true" />
        <h2>{{ t('needtoken.catalogServices.loadError') }}</h2>
        <button type="button" @click="loadProducts">{{ t('needtoken.common.retry') }}</button>
      </div>
      <div v-else-if="!scope?.root || !products.length" class="nt-service-catalog__state" role="status">
        <PackageSearch :size="30" aria-hidden="true" />
        <h2>{{ t(searchQuery ? 'needtoken.catalogServices.emptySearch' : 'needtoken.catalogServices.empty') }}</h2>
        <p>{{ t('needtoken.catalogServices.emptyDescription') }}</p>
        <button v-if="searchQuery" type="button" @click="clearSearch">{{ t('products.clearFilters') }}</button>
      </div>
      <template v-else>
        <div class="nt-service-catalog__result" aria-live="polite">
          <span>{{ t(`needtoken.catalogServices.${serviceKind}.section`) }}</span>
          <span>{{ t('needtoken.catalogServices.resultCount', { count: totalProducts }) }}</span>
        </div>
        <div class="nt-service-catalog__grid">
          <ServiceProductCard v-for="product in products" :key="product.id" :product="product" :kind="serviceKind!" :category-label="categoryLabel(product.category_id)" />
        </div>
        <PaginationNav :current-page="page" :total-pages="totalPages" :loading="loading" compact @change-page="changePage" />
      </template>

      <div class="nt-service-catalog__help">
        <div><BookOpen :size="18" aria-hidden="true" /><span>{{ t('needtoken.catalogServices.help') }}</span></div>
        <RouterLink to="/blog">{{ t('needtoken.catalogServices.guides') }}<ArrowUpRight :size="14" aria-hidden="true" /></RouterLink>
        <RouterLink to="/notice">{{ t('needtoken.catalogServices.notices') }}<ArrowUpRight :size="14" aria-hidden="true" /></RouterLink>
        <RouterLink to="/guest/orders">{{ t('needtoken.catalogServices.orders') }}<ArrowUpRight :size="14" aria-hidden="true" /></RouterLink>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { ArrowRight, ArrowUpRight, BookOpen, Building2, FileText, PackageCheck, PackageSearch, Search, ShieldCheck, Undo2 } from 'lucide-vue-next'
import ClassicProducts from '../../views/Products.vue'
import PaginationNav from '../../components/PaginationNav.vue'
import { useLocalized } from '../../composables/useProduct'
import { usePageSeo } from '../../composables/usePageSeo'
import ServiceProductCard from './components/ServiceProductCard.vue'
import { useServiceCatalog } from './composables/useServiceCatalog'

const { t } = useI18n()
const route = useRoute()
const { getLocalizedText } = useLocalized()
const {
  scope, serviceKind, categoriesLoading, categoriesError, products, loading, loadError,
  page, totalPages, totalProducts, searchQuery, loadCategories, loadProducts,
  selectCategory, changePage, submitSearch, clearSearch,
} = useServiceCatalog()
const serviceNotes = computed(() => serviceKind.value === 'rental'
  ? ['deliveryNote', 'afterSaleNote', 'invoiceNote', 'businessTransferNote']
  : ['warrantyNote', 'refundNote', 'invoiceNote', 'businessTransferNote'])
const noteIcons = computed(() => serviceKind.value === 'rental'
  ? [PackageCheck, ShieldCheck, FileText, Building2]
  : [ShieldCheck, Undo2, FileText, Building2])
const categoryLabel = (id: number) => getLocalizedText([scope.value?.root, ...(scope.value?.categories || [])].find(row => row?.id === id)?.name)
usePageSeo({
  title: () => serviceKind.value ? t(`needtoken.catalogServices.${serviceKind.value}.label`) : t('nav.products'),
  description: () => serviceKind.value ? t(`needtoken.catalogServices.${serviceKind.value}.subtitle`) : undefined,
  canonicalPath: () => route.path,
})
</script>

<style scoped>
.nt-service-catalog { padding-top: 48px; padding-bottom: 32px; }
.nt-service-catalog__intro { text-align: center; margin-bottom: 34px; }
.nt-service-catalog__eyebrow { display: inline-flex; align-items: center; gap: 7px; color: var(--ui-accent); font-size: 11px; font-weight: 650; letter-spacing: .12em; }
.nt-service-catalog__eyebrow::before { content: ''; width: 5px; height: 5px; border-radius: 50%; background: currentColor; }
h1 { margin-top: 12px; font-size: clamp(28px, 3.5vw, 42px); font-weight: 750; letter-spacing: -.055em; line-height: 1.25; }
.nt-service-catalog__subtitle { color: var(--ui-text-secondary); margin-top: 14px; font-size: 15px; }
.nt-service-catalog__promises { display: flex; justify-content: center; flex-wrap: wrap; gap: 10px 23px; margin-top: 22px; color: var(--ui-text-muted); font-size: 12px; }
.nt-service-catalog__promises li { display: inline-flex; align-items: center; gap: 7px; }
.nt-service-catalog__promises svg { color: var(--ui-accent); opacity: .75; }
.nt-service-catalog__toolbar { display: flex; align-items: center; justify-content: space-between; gap: 20px; padding: 20px 0; border-top: 1px solid var(--ui-border); }
.nt-service-catalog__filters { min-width: 0; display: flex; gap: 8px; overflow-x: auto; padding: 4px; margin: -4px; scrollbar-width: thin; }
.nt-service-catalog__filter { min-height: 40px; flex-shrink: 0; border: 1px solid rgba(204, 219, 235, .17); border-radius: 12px; background: linear-gradient(135deg, rgba(255,255,255,.075), rgba(255,255,255,.018)); box-shadow: inset 0 1px 0 rgba(255,255,255,.09), 0 4px 16px rgba(0,0,0,.08); backdrop-filter: blur(18px); padding: 8px 17px; color: var(--ui-text-secondary); font-size: 12px; white-space: nowrap; cursor: pointer; transition: .15s; }
.nt-service-catalog__filter:hover { color: var(--ui-text-primary); border-color: rgba(204,219,235,.35); }
.nt-service-catalog__filter.is-active { color: var(--ui-accent); border-color: rgba(45,212,167,.44); background: linear-gradient(135deg, rgba(45,212,167,.15), rgba(45,212,167,.04)); box-shadow: inset 0 1px 0 rgba(153,255,223,.13), 0 0 18px rgba(45,212,167,.035); }
.nt-service-catalog__filter:focus-visible, .nt-service-catalog__search:focus-within { outline: 2px solid var(--ui-accent); outline-offset: 3px; }
.nt-service-catalog__search { display: flex; align-items: center; gap: 8px; width: 220px; flex-shrink: 0; min-height: 40px; padding: 0 11px; border: 1px solid var(--ui-border); border-radius: 12px; background: rgba(255,255,255,.025); color: var(--ui-text-muted); }
.nt-service-catalog__search input { min-width: 0; flex: 1; width: 100%; outline: none; background: transparent; font-size: 12px; color: var(--ui-text-primary); }
.nt-service-catalog__search button { display: grid; width: 24px; height: 28px; place-items: center; }
.nt-service-catalog__result { display: flex; justify-content: space-between; gap: 12px; margin: 9px 0 18px; color: var(--ui-text-muted); font-size: 11px; }
.nt-service-catalog__result > span:first-child { color: var(--ui-text-primary); font-size: 13px; font-weight: 600; }
.nt-service-catalog__grid { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 16px; }
.nt-service-catalog__skeleton { min-height: 406px; border: 1px solid var(--ui-border); border-radius: 24px; padding: 24px; }
.nt-service-catalog__skeleton > div { width: 80%; height: 20px; margin-bottom: 20px; border-radius: 7px; background: rgba(255,255,255,.045); }
.nt-service-catalog__skeleton > div:first-child { width: 42px; height: 42px; }
.nt-service-catalog__skeleton > div:nth-child(3) { width: 100%; height: 70px; }
.nt-service-catalog__state { min-height: 320px; display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 14px; text-align: center; border: 1px solid var(--ui-border); border-radius: 24px; color: var(--ui-text-muted); }
.nt-service-catalog__state h2 { font-weight: 600; font-size: 18px; color: var(--ui-text-primary); }
.nt-service-catalog__state p { padding: 0 18px; font-size: 13px; }
.nt-service-catalog__state button { padding: 8px 16px; border: 1px solid var(--ui-border); border-radius: 10px; font-size: 12px; }
.nt-service-catalog__help { display: flex; align-items: center; justify-content: center; flex-wrap: wrap; gap: 17px; margin-top: 35px; padding-top: 24px; border-top: 1px solid var(--ui-border); font-size: 11px; color: var(--ui-text-muted); }
.nt-service-catalog__help > div, .nt-service-catalog__help a { display: inline-flex; align-items: center; gap: 7px; }
.nt-service-catalog__help a:hover { color: var(--ui-accent); }
@media (max-width: 1023px) { .nt-service-catalog__grid { grid-template-columns: repeat(2, minmax(0, 1fr)); } }
@media (min-width: 1024px) {
  .nt-service-catalog { padding-top: 28px; }
  .nt-service-catalog__intro { margin-bottom: 24px; }
  h1 { margin-top: 9px; font-size: 38px; }
  .nt-service-catalog__subtitle { margin-top: 10px; }
  .nt-service-catalog__promises { margin-top: 15px; }
  .nt-service-catalog__toolbar { padding-top: 16px; padding-bottom: 16px; }
  .nt-service-catalog__result { margin-top: 6px; margin-bottom: 12px; }
  .nt-service-catalog__skeleton { min-height: 280px; padding: 18px; border-radius: 20px; }
}
@media (max-width: 640px) { .nt-service-catalog { padding-top: 32px; } .nt-service-catalog__intro { margin-bottom: 27px; } .nt-service-catalog__promises { gap: 10px 15px; font-size: 10px; } .nt-service-catalog__toolbar { align-items: stretch; flex-direction: column; gap: 15px; } .nt-service-catalog__search { width: 100%; } .nt-service-catalog__grid { grid-template-columns: minmax(0, 1fr); } h1 { font-size: 29px; } .nt-service-catalog__help > div { width: 100%; justify-content: center; } }
@media (prefers-reduced-motion: reduce) { .nt-service-catalog__filter { transition: none; } }
</style>
