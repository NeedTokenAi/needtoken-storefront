<template>
  <ClassicProducts v-if="!categoriesLoading && !categoriesError && !serviceKind" />
  <div v-else class="nt-service-catalog mx-auto w-full max-w-[1200px] px-4 sm:px-6">
    <header class="nt-service-catalog__intro">
      <h1>{{ t(`needtoken.catalogServices.${serviceKind || 'recharge'}.titlePrefix`) }}<span class="nt-service-catalog__title-subject">{{ t(`needtoken.catalogServices.${serviceKind || 'recharge'}.titleSubject`) }}</span>{{ t(`needtoken.catalogServices.${serviceKind || 'recharge'}.titleSuffix`) }}</h1>
      <ul class="nt-service-catalog__promises" :aria-label="t('needtoken.catalogServices.serviceNotes')">
        <li v-for="(note, index) in serviceNotes" :key="note"><component :is="noteIcons[index]" :size="15" aria-hidden="true" />{{ t(`needtoken.catalogServices.${note}`) }}</li>
        <li class="nt-service-catalog__rate"><StandardExchangeRate placement="below" /></li>
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
        <CatalogueTools v-model="searchQuery" :search-label="t('needtoken.catalogServices.search')" :count="!loading && !categoriesLoading && !loadError ? totalProducts : null" :loading="categoriesLoading || loading" @search="submitSearch" />
      </div>

      <div v-if="categoriesLoading || loading" class="nt-service-catalog__grid" :aria-busy="true">
        <NeedtokenSkeleton kind="prices" :count="4" />
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
        <div class="nt-service-catalog__grid is-ready">
          <HomePriceCard v-for="(product, index) in products" :key="serviceCardKey(product)" :product="product" :kind="serviceKind!" :style="{ '--card-index': index }" />
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
import { ArrowUpRight, BookOpen, Building2, FileText, PackageCheck, PackageSearch, ShieldCheck, Undo2 } from 'lucide-vue-next'
import ClassicProducts from '../../views/Products.vue'
import PaginationNav from '../../components/PaginationNav.vue'
import { useLocalized } from '../../composables/useProduct'
import { usePageSeo } from '../../composables/usePageSeo'
import HomePriceCard from './components/HomePriceCard.vue'
import NeedtokenSkeleton from './components/NeedtokenSkeleton.vue'
import CatalogueTools from './components/CatalogueTools.vue'
import StandardExchangeRate from './components/StandardExchangeRate.vue'
import { useServiceCatalog } from './composables/useServiceCatalog'
import { serviceCardKey } from './lib/serviceCatalog'

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
usePageSeo({
  title: () => serviceKind.value ? t(`needtoken.catalogServices.${serviceKind.value}.label`) : t('nav.products'),
  description: () => serviceKind.value ? t(`needtoken.catalogServices.${serviceKind.value}.subtitle`) : undefined,
  canonicalPath: () => route.path,
})
</script>

<style scoped>
.nt-service-catalog { padding-top: 28px; padding-bottom: 40px; }
.nt-service-catalog__intro { text-align: center; margin-bottom: 20px; }
h1 { font-size: clamp(27px, 2.5vw, 34px); font-weight: 750; letter-spacing: -.045em; line-height: 1.25; text-wrap: balance; }
.nt-service-catalog__title-subject { color: var(--ui-accent); }
.nt-service-catalog__promises { display: grid; grid-template-columns: repeat(5, minmax(0, 1fr)); gap: 12px; margin-top: 16px; font-size: 12px; }
.nt-service-catalog__promises li { min-width: 0; display: flex; align-items: center; justify-content: center; gap: 8px; max-width: 100%; min-height: 44px; padding: 10px 12px; border: 1px solid var(--nt-benefit-border); border-radius: 12px; background: var(--nt-benefit-bg); color: var(--nt-benefit-fg); font-weight: 600; line-height: 1.5; text-align: center; box-shadow: var(--nt-benefit-shadow); transition: border-color var(--nt-motion-fast, 120ms), background var(--nt-motion-fast, 120ms), box-shadow var(--nt-motion-enter, 180ms), transform var(--nt-duration-rebound, 480ms) var(--nt-ease-spring); }
.nt-service-catalog__promises li:hover { border-color: var(--nt-benefit-border-hover); background: var(--nt-benefit-bg-hover); box-shadow: var(--nt-benefit-shadow-hover); transform: translateY(-2px); }
.nt-service-catalog__promises svg, .nt-service-catalog__promises :deep(.nt-standard-rate__trigger > svg) { flex: none; color: var(--nt-benefit-icon); opacity: 1; }
.nt-service-catalog__rate { position: relative; z-index: 3; }
.nt-service-catalog__toolbar { display: flex; align-items: center; justify-content: space-between; gap: 12px; padding: 14px 0 18px; border-top: 1px solid var(--ui-border); }
.nt-service-catalog__filters { min-width: 0; flex: 1; display: flex; align-items: center; gap: 8px; overflow-x: auto; padding: 4px; margin: -4px; scrollbar-width: thin; }
.nt-service-catalog__filter { min-height: 36px; flex-shrink: 0; border: 1px solid transparent; border-radius: 999px; background: transparent; padding: 8px 16px; color: var(--ui-text-secondary); font-size: 13px; font-weight: 500; white-space: nowrap; cursor: pointer; transition: color var(--nt-motion-fast, 120ms), background-color var(--nt-motion-fast, 120ms), border-color var(--nt-motion-fast, 120ms), transform var(--nt-duration-rebound, 480ms) var(--nt-ease-spring); }
.nt-service-catalog__filter:hover { color: var(--ui-text-primary); background: var(--ui-bg-soft); }
.nt-service-catalog__filter:active { transform: scale(0.965); transition-duration: var(--nt-duration-press, 80ms); }
.nt-service-catalog__filter.is-active { color: var(--ui-accent); border-color: var(--nt-accent-border); background: var(--ui-accent-soft); font-weight: 600; }
.nt-service-catalog__filter:focus-visible { outline: 2px solid var(--ui-accent); outline-offset: 3px; }
.nt-service-catalog__grid { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); min-height: 156px; gap: 16px; }
.nt-service-catalog__grid > :deep(.nt-page-skeleton--prices) { grid-column: 1 / -1; padding: 0; }
.nt-service-catalog__grid.is-ready > :deep(.home-price-card) { animation: nt-catalog-card-rise 320ms var(--nt-ease-spring-soft) calc(min(var(--card-index, 0), 10) * 38ms) backwards; }
@keyframes nt-catalog-card-rise { from { opacity: 0; transform: translateY(10px); } to { opacity: 1; transform: translateY(0); } }
.nt-service-catalog__state { min-height: 240px; display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 14px; text-align: center; border: 1px dashed var(--ui-border-strong); border-radius: 20px; background: var(--nt-surface-translucent); color: var(--ui-text-muted); }
.nt-service-catalog__state h2 { font-weight: 600; font-size: 18px; color: var(--ui-text-primary); }
.nt-service-catalog__state p { padding: 0 18px; font-size: 13px; }
.nt-service-catalog__state button { padding: 8px 16px; border: 1px solid var(--ui-border); border-radius: 10px; font-size: 12px; color: var(--ui-accent); }
.nt-service-catalog__help { display: flex; align-items: center; justify-content: center; flex-wrap: wrap; gap: 17px; margin-top: 35px; padding-top: 24px; border-top: 1px solid var(--ui-border); font-size: 11px; color: var(--ui-text-muted); }
.nt-service-catalog__help > div, .nt-service-catalog__help a { display: inline-flex; align-items: center; gap: 7px; }
.nt-service-catalog__help a:hover { color: var(--ui-accent); }
@media (min-width: 1440px) { .nt-service-catalog__grid { grid-template-columns: repeat(5, minmax(0, 1fr)); min-height: 132px; } }
@media (max-width: 1100px) { .nt-service-catalog__grid { grid-template-columns: repeat(2, minmax(0, 1fr)); } }
@media (max-width: 1023px) { .nt-service-catalog__promises { grid-template-columns: repeat(2, minmax(0, 1fr)); } .nt-service-catalog__rate { grid-column: 1 / -1; } }
@media (max-width: 640px) { .nt-service-catalog { padding-top: 22px; } .nt-service-catalog__promises { gap: 10px; } .nt-service-catalog__promises li { min-height: 40px; padding: 9px 10px; gap: 6px; font-size: 11px; } .nt-service-catalog__promises svg { width: 14px; height: 14px; } .nt-service-catalog__filter { padding-inline: 12px; font-size: 12px; } .nt-service-catalog__grid { grid-template-columns: minmax(0, 1fr); } .nt-service-catalog__help > div { width: 100%; justify-content: center; } }
@media (prefers-reduced-motion: reduce) {
  .nt-service-catalog__filter, .nt-service-catalog__promises li { transition: none; }
  .nt-service-catalog__promises li:hover, .nt-service-catalog__filter:active { transform: none; }
  .nt-service-catalog__grid.is-ready > :deep(.home-price-card) { animation: none; }
}
</style>
