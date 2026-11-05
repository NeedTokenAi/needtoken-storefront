<template>
  <div class="api-plans mx-auto w-full max-w-[1200px] px-4 pb-12 sm:px-6">
    <header class="api-plans-heading">
      <h1>{{ t('needtoken.apiCatalog.titlePrefix') }}<span>{{ t('needtoken.nav.plans') }}</span></h1>
      <ul class="api-plans-promises" :aria-label="t('needtoken.catalogServices.serviceNotes')">
        <li v-for="(note, index) in notes" :key="note"><component :is="noteIcons[index]" :size="15" aria-hidden="true" />{{ t(`needtoken.apiCatalog.${note}`) }}</li>
        <li class="api-plans-rate"><StandardExchangeRate placement="below" /></li>
      </ul>
    </header>

    <section id="api-plan-catalog" class="api-plan-catalog" :aria-label="t('needtoken.apiCatalog.catalogTitle')">
      <div class="api-plan-toolbar">
        <nav class="api-plan-tabs" :aria-label="t('needtoken.apiCatalog.audiences')">
          <button v-for="kind in audiences" :key="kind" type="button" :aria-pressed="audience === kind" :class="{ active: audience === kind }" @click="selectAudience(kind)"><component :is="kind === 'personal' ? UserRound : Building2" :size="16" aria-hidden="true" />{{ t(`needtoken.apiCatalog.${kind}`) }}</button>
        </nav>
        <CatalogueTools v-model="searchQuery" :search-label="t('needtoken.apiCatalog.search')" :count="!loading && !categoriesLoading && !loadError && !categoriesError ? totalProducts : null" :loading="categoriesLoading || loading" @search="submitSearch">
          <a :href="CONSOLE_URL" class="api-plan-console" rel="noopener">{{ t('needtoken.plans.console') }}<ArrowUpRight :size="14" aria-hidden="true" /></a>
        </CatalogueTools>
      </div>

      <div class="api-plan-member-note"><LockKeyhole :size="18" aria-hidden="true" /><details><summary><h2>{{ t('needtoken.apiCatalog.memberTitle') }}</h2><span>{{ t('needtoken.apiCatalog.step1Title') }}</span><ChevronDown :size="14" aria-hidden="true" /></summary><p>{{ t('needtoken.apiCatalog.memberDescription') }}</p></details><RouterLink v-if="!userAuth.isAuthenticated" :to="loginTarget">{{ t('needtoken.apiCatalog.login') }}<ArrowUpRight :size="14" aria-hidden="true" /></RouterLink><RouterLink v-else to="/me/orders">{{ t('needtoken.apiCatalog.myOrders') }}<ArrowUpRight :size="14" aria-hidden="true" /></RouterLink></div>

      <div v-if="categoriesLoading || loading" class="api-plan-grid" role="status" aria-busy="true" :aria-label="t('needtoken.common.loading')"><div v-for="index in 6" :key="index" class="api-plan-skeleton"><div /><div /><div /><div /></div></div>
      <div v-else-if="categoriesError || loadError" class="api-plan-state" role="alert"><PackageSearch :size="30" aria-hidden="true" /><h3>{{ t('needtoken.catalogServices.loadError') }}</h3><button type="button" @click="categoriesError ? loadCategories() : loadProducts()">{{ t('needtoken.common.retry') }}</button></div>
      <div v-else-if="!products.length" class="api-plan-state" role="status"><PackageSearch :size="30" aria-hidden="true" /><h3>{{ t(searchQuery ? 'needtoken.catalogServices.emptySearch' : 'needtoken.apiCatalog.empty') }}</h3><p>{{ t('needtoken.apiLanding.publicBrowse') }}</p><p>{{ t('needtoken.apiCatalog.emptyDescription') }}</p><button v-if="searchQuery" type="button" @click="clearSearch">{{ t('products.clearFilters') }}</button><a v-else :href="businessEmailHref">{{ t('needtoken.apiCatalog.businessContact') }}<ArrowUpRight :size="14" aria-hidden="true" /></a></div>
      <template v-else><div class="api-plan-grid"><ServiceProductCard v-for="product in products" :key="product.id" :product="product" kind="recharge" member-only :category-label="t(`needtoken.apiCatalog.${audience}`)" :cta-label="t(userAuth.isAuthenticated ? 'needtoken.apiCatalog.choose' : 'needtoken.apiCatalog.loginToChoose')" :delivery-text="t('needtoken.apiCatalog.memberDelivery')" :description-fallback="t('needtoken.apiCatalog.productFallback')" /></div><PaginationNav :current-page="page" :total-pages="totalPages" :loading="loading" compact @change-page="changePage" /></template>
    </section>

    <section class="api-plan-how" :aria-label="t('needtoken.apiCatalog.howTitle')"><h2>{{ t('needtoken.apiCatalog.howTitle') }}</h2><ol><li v-for="n in 3" :key="n"><span>0{{ n }}</span><div><h3>{{ t(`needtoken.apiCatalog.step${n}Title`) }}</h3><p>{{ t(`needtoken.apiCatalog.step${n}Description`) }}</p></div></li></ol></section>
    <section class="api-plan-connect"><div><h2>{{ t('needtoken.apiServices.connectTitle') }}</h2><p>{{ t('needtoken.apiCatalog.connectDescription') }}</p></div><RouterLink to="/docs">{{ t('needtoken.nav.docs') }}<ArrowUpRight :size="14" aria-hidden="true" /></RouterLink><CodeBlock :code="API_BASE_URL" title="baseURL" /></section>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { ArrowUpRight, Building2, ChevronDown, Clock3, FileText, KeyRound, LockKeyhole, PackageSearch, UserRound } from 'lucide-vue-next'
import { usePageSeo } from '../../composables/usePageSeo'
import { useUserAuthStore } from '../../stores/userAuth'
import { useAppStore } from '../../stores/app'
import PaginationNav from '../../components/PaginationNav.vue'
import { API_BASE_URL, CONSOLE_URL } from './config'
import CodeBlock from './components/CodeBlock.vue'
import ServiceProductCard from './components/ServiceProductCard.vue'
import CatalogueTools from './components/CatalogueTools.vue'
import StandardExchangeRate from './components/StandardExchangeRate.vue'
import { useApiPlanCatalog } from './composables/useApiPlanCatalog'
import { normalizeNeedtokenSupport, supportEmailHref } from './lib/supportConfig'

const { t } = useI18n()
const userAuth = useUserAuthStore()
const appStore = useAppStore()
const { audience, categoriesLoading, categoriesError, products, loading, loadError, totalPages, totalProducts, page, searchQuery, loadCategories, loadProducts, selectAudience, changePage, submitSearch, clearSearch } = useApiPlanCatalog()
const audiences = ['personal', 'enterprise'] as const
const notes = ['memberOnly', 'quotaNote', 'usageNote', 'enterpriseNote']
const noteIcons = [LockKeyhole, Clock3, KeyRound, FileText]
const loginTarget = computed(() => ({ path: '/auth/login', query: { redirect: `/plans?kind=${audience.value}#api-plan-catalog` } }))
const businessEmailHref = computed(() => supportEmailHref(normalizeNeedtokenSupport(appStore.config?.needtoken_support).businessEmail))
usePageSeo({ title: () => t('needtoken.plans.title'), description: () => t('needtoken.apiCatalog.subtitle'), canonicalPath: () => '/plans' })
</script>

<style scoped>
.api-plans { padding-top: 20px; }
.api-plans-heading { text-align: center; padding-bottom: 12px; }
.api-plans-heading h1 { font-size: clamp(27px, 2.5vw, 34px); font-weight: 750; letter-spacing: -.045em; line-height: 1.25; text-wrap: balance; }
.api-plans-heading h1 span { color: var(--ui-accent); }
.api-plans-promises { display: grid; grid-template-columns: repeat(5, minmax(0, 1fr)); gap: 8px; margin-top: 12px; font-size: 12px; }
.api-plans-promises li { display: flex; min-width: 0; min-height: 34px; justify-content: center; align-items: center; gap: 7px; padding: 6px 11px; border: 1px solid #ffe07e73; border-radius: 9px; background: linear-gradient(135deg, #f3cb57, #e8b83b); color: #172330; font-weight: 650; line-height: 1.5; }
.api-plans-promises svg { flex: none; opacity: .85; }
.api-plans-rate { position: relative; z-index: 3; }
.api-plan-catalog { scroll-margin-top: 85px; }
.api-plan-toolbar { display: flex; align-items: center; justify-content: space-between; gap: 12px; padding: 10px 0 12px; border-top: 1px solid var(--ui-border); }
.api-plan-console { display: inline-flex; align-items: center; gap: 6px; flex: none; min-height: 40px; color: var(--ui-accent); font-size: 12px; }
.api-plan-tabs { display: flex; flex: 1; gap: 9px; min-width: 0; overflow-x: auto; padding: 4px; margin: -4px; scrollbar-width: thin; }
.api-plan-tabs button { display: inline-flex; align-items: center; gap: 8px; min-height: 40px; padding: 8px 14px; border: 1px solid var(--ui-border); border-radius: 10px; background: linear-gradient(135deg, #ffffff11, #ffffff04); color: var(--ui-text-secondary); font-size: 13px; white-space: nowrap; }
.api-plan-tabs button.active { color: var(--ui-accent); border-color: #2dd4a770; background: linear-gradient(135deg, #2dd4a725, #2dd4a709); }
.api-plan-tabs button:focus-visible, .api-plan-console:focus-visible { outline: 2px solid var(--ui-accent); outline-offset: 3px; }
.api-plan-member-note { display: flex; align-items: center; gap: 10px; margin-bottom: 12px; padding: 8px 12px; border: 1px solid #2dd4a733; border-radius: 11px; background: #2dd4a709; }
.api-plan-member-note > svg { flex: none; color: var(--ui-accent); margin-top: 3px; }
.api-plan-member-note > details { min-width: 0; flex: 1; }
.api-plan-member-note summary { display: flex; align-items: center; flex-wrap: wrap; gap: 3px 12px; list-style: none; cursor: pointer; }
.api-plan-member-note summary::-webkit-details-marker { display: none; }
.api-plan-member-note summary:focus-visible { outline: 2px solid var(--ui-accent); outline-offset: 4px; border-radius: 3px; }
.api-plan-member-note summary > span { color: var(--ui-text-secondary); font-size: 12px; }
.api-plan-member-note summary > svg { flex: none; color: var(--ui-accent); }
.api-plan-member-note details[open] summary > svg { transform: rotate(180deg); }
.api-plan-member-note h2 { font-size: 13px; font-weight: 650; }
.api-plan-member-note p { margin-top: 8px; color: var(--ui-text-muted); font-size: 12px; line-height: 1.7; overflow-wrap: anywhere; }
.api-plan-member-note > a { display: inline-flex; flex: none; align-items: center; gap: 5px; color: var(--ui-accent); font-size: 12px; margin-top: 3px; }
.api-plan-grid { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 17px; }
.api-plan-skeleton { min-height: 330px; padding: 18px; border: 1px solid var(--ui-border); border-radius: 20px; }
.api-plan-skeleton div { height: 25px; width: 70%; margin-bottom: 22px; border-radius: 8px; background: #ffffff09; }
.api-plan-skeleton div:nth-child(3) { height: 85px; width: 100%; }
.api-plan-state { display: grid; align-content: center; justify-items: center; gap: 14px; min-height: 310px; padding: 25px; border: 1px solid var(--ui-border); border-radius: 22px; text-align: center; color: var(--ui-text-muted); }
.api-plan-state h3 { color: var(--ui-text-primary); font-size: 17px; font-weight: 650; }
.api-plan-state p { font-size: 13px; line-height: 1.8; max-width: 56ch; }
.api-plan-state button, .api-plan-state a { display: inline-flex; align-items: center; gap: 6px; padding: 8px 12px; border: 1px solid var(--ui-border); border-radius: 9px; font-size: 12px; color: var(--ui-accent); }
.api-plan-how { margin-top: 44px; }
.api-plan-how > h2, .api-plan-connect h2 { font-size: 20px; font-weight: 650; }
.api-plan-how ol { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); margin-top: 20px; border: 1px solid var(--ui-border); border-radius: 18px; overflow: hidden; background: var(--ui-bg-elevated); }
.api-plan-how li { display: flex; gap: 15px; min-width: 0; padding: 24px; }
.api-plan-how li + li { border-left: 1px solid var(--ui-border); }
.api-plan-how li > span { flex: none; color: var(--ui-accent); font-family: var(--nt-font-mono); font-size: 12px; margin-top: 3px; }
.api-plan-how h3 { font-size: 13px; font-weight: 650; }
.api-plan-how p { margin-top: 9px; font-size: 12px; line-height: 1.9; color: var(--ui-text-muted); overflow-wrap: anywhere; }
.api-plan-connect { display: grid; grid-template-columns: 1fr auto; align-items: center; gap: 20px; padding: 26px; margin-top: 38px; border: 1px solid var(--ui-border); border-radius: 18px; background: var(--ui-bg-elevated); }
.api-plan-connect > div:first-child { min-width: 0; }
.api-plan-connect p { margin-top: 9px; color: var(--ui-text-muted); font-size: 12px; line-height: 1.85; }
.api-plan-connect > a { display: inline-flex; align-items: center; gap: 6px; font-size: 12px; color: var(--ui-accent); }
.api-plan-connect > .nt-code { grid-column: 1 / -1; min-width: 0; }
@media (max-width: 1023px) { .api-plans-promises { grid-template-columns: repeat(2, minmax(0, 1fr)); } .api-plans-rate { grid-column: 1 / -1; } .api-plan-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); } .api-plan-member-note { align-items: flex-start; } }
@media (max-width: 640px) { .api-plans { padding-top: 22px; } .api-plans-promises li { padding: 7px 9px; gap: 6px; } .api-plan-grid { grid-template-columns: minmax(0, 1fr); } .api-plan-tabs button { flex: none; padding-inline: 10px; gap: 6px; } .api-plan-member-note { flex-wrap: wrap; padding: 12px; } .api-plan-member-note > details { flex-basis: calc(100% - 32px); } .api-plan-member-note > a { margin-left: 28px; } .api-plan-how ol { grid-template-columns: 1fr; } .api-plan-how li + li { border-left: 0; border-top: 1px solid var(--ui-border); } .api-plan-connect { grid-template-columns: 1fr; padding: 20px; } .api-plan-connect > a { width: fit-content; } }
</style>
