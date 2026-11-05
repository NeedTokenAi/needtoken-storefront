<template>
  <div class="nt-faq mx-auto w-full max-w-[1200px] px-4 sm:px-6">
    <header class="nt-help-heading">
      <span class="nt-help-eyebrow"><CircleHelp :size="16" aria-hidden="true" />{{ t('needtoken.faq.eyebrow') }}</span>
      <h1>{{ t('needtoken.faq.title') }}</h1>
      <p>{{ t('needtoken.faq.subtitle') }}</p>
      <form class="nt-faq-search" role="search" @submit.prevent="submitSearch">
        <Search :size="19" aria-hidden="true" />
        <input v-model="searchInput" type="search" :aria-label="t('needtoken.faq.search')" :placeholder="t('needtoken.faq.search')" />
        <button type="submit">{{ t('needtoken.faq.searchAction') }}<ArrowRight :size="16" aria-hidden="true" /></button>
      </form>
    </header>

    <div class="nt-faq-layout">
      <aside class="nt-faq-sidebar">
        <nav :aria-label="t('needtoken.faq.categories')">
          <h2>{{ t('needtoken.faq.categories') }}</h2>
          <button type="button" :aria-pressed="selectedSlug === 'faq'" :class="{ 'is-active': selectedSlug === 'faq' }" @click="selectCategory('faq')"><Layers :size="16" aria-hidden="true" />{{ t('needtoken.faq.all') }}<ChevronRight :size="15" aria-hidden="true" /></button>
          <button v-for="category in scope.children" :key="category.id" type="button" :aria-pressed="selectedSlug === category.slug" :class="{ 'is-active': selectedSlug === category.slug }" @click="selectCategory(category.slug)"><BookOpen :size="16" aria-hidden="true" />{{ localized(category.name) }}<ChevronRight :size="15" aria-hidden="true" /></button>
        </nav>
        <div class="nt-faq-contact">
          <MessageCircle :size="22" aria-hidden="true" />
          <h2>{{ t('needtoken.faq.needHelp') }}</h2>
          <p>{{ t('needtoken.faq.contactDescription') }}</p>
          <RouterLink to="/guest/orders">{{ t('needtoken.faq.orderHelp') }}<ArrowUpRight :size="15" aria-hidden="true" /></RouterLink>
        </div>
      </aside>

      <section class="nt-faq-main" :aria-label="t('needtoken.faq.questions')" :aria-busy="loading">
        <div class="nt-faq-result"><h2>{{ selectedName }}</h2><span v-if="!loading && !failed" aria-live="polite">{{ t('needtoken.faq.count', { count: total }) }}</span></div>
        <div v-if="loading" class="nt-help-state" role="status"><LoaderCircle class="animate-spin" :size="25" aria-hidden="true" /><p>{{ t('needtoken.common.loading') }}</p></div>
        <div v-else-if="failed" class="nt-help-state" role="alert"><CircleAlert :size="28" aria-hidden="true" /><h3>{{ t('needtoken.faq.loadError') }}</h3><p>{{ t('needtoken.faq.loadErrorDescription') }}</p><button type="button" @click="retry">{{ t('needtoken.common.retry') }}</button></div>
        <div v-else-if="!posts.length" class="nt-help-state" role="status"><SearchX :size="30" aria-hidden="true" /><h3>{{ t(searchTerm ? 'needtoken.faq.noResults' : 'needtoken.faq.empty') }}</h3><p>{{ t(searchTerm ? 'needtoken.faq.noResultsDescription' : 'needtoken.faq.emptyDescription') }}</p><button v-if="searchTerm" type="button" @click="clearSearch">{{ t('needtoken.faq.clear') }}</button></div>
        <template v-else>
          <div class="nt-faq-answers">
            <details v-for="post in posts" :key="post.id" class="nt-faq-answer" name="needtoken-faq-answers">
              <summary><span>{{ localized(post.title) }}</span><Plus :size="19" aria-hidden="true" /></summary>
              <div class="nt-faq-answer-body">
                <div class="prose prose-sm dark:prose-invert max-w-none theme-prose" v-html="processHtmlForDisplay(localized(post.content) || localized(post.summary))"></div>
                <RouterLink :to="`/blog/${encodeURIComponent(post.slug)}`">{{ t('needtoken.faq.readArticle') }}<ArrowUpRight :size="14" aria-hidden="true" /></RouterLink>
              </div>
            </details>
          </div>
          <PaginationNav :current-page="page" :total-pages="totalPages" :loading="loading" :scroll-top="false" compact @change-page="changePage" />
        </template>
      </section>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { ArrowRight, ArrowUpRight, BookOpen, ChevronRight, CircleAlert, CircleHelp, Layers, LoaderCircle, MessageCircle, Plus, Search, SearchX } from 'lucide-vue-next'
import { postAPI } from '../../api'
import PaginationNav from '../../components/PaginationNav.vue'
import { usePageSeo } from '../../composables/usePageSeo'
import { useAppStore } from '../../stores/app'
import { processHtmlForDisplay } from '../../utils/content'
import { getLocalizedText } from '../../utils/resellerSiteConfig'
import { faqCategories } from './lib/helpContent'

interface HelpPost { id: number; slug: string; title: Record<string, unknown>; summary: Record<string, unknown>; content: Record<string, unknown> }
const { t } = useI18n()
const appStore = useAppStore()
const localized = (value?: Record<string, unknown>) => getLocalizedText(value, appStore.locale)
const categories = ref<unknown>([])
const scope = computed(() => faqCategories(categories.value))
const selectedSlug = ref('faq')
const selectedName = computed(() => selectedSlug.value === 'faq' ? t('needtoken.faq.all') : localized(scope.value.children.find((item) => item.slug === selectedSlug.value)?.name))
const searchInput = ref('')
const searchTerm = ref('')
const posts = ref<HelpPost[]>([])
const page = ref(1)
const total = ref(0)
const totalPages = computed(() => Math.max(1, Math.ceil(total.value / 12)))
const loading = ref(true)
const failed = ref(false)
let generation = 0
let disposed = false
let categoriesReady = false
usePageSeo({ title: () => t('needtoken.faq.title'), description: () => t('needtoken.faq.subtitle'), canonicalPath: () => '/faq' })

async function loadPosts() {
  const current = ++generation
  loading.value = true
  failed.value = false
  posts.value = []
  total.value = 0
  if (!scope.value.root) { loading.value = false; return }
  try {
    const response = await postAPI.list({ type: 'blog', category_slug: selectedSlug.value, search: searchTerm.value, page: page.value, page_size: 12 })
    if (disposed || current !== generation) return
    posts.value = Array.isArray(response.data.data) ? response.data.data : []
    total.value = Number(response.data.pagination?.total) || 0
  } catch {
    if (!disposed && current === generation) failed.value = true
  } finally {
    if (!disposed && current === generation) loading.value = false
  }
}

async function initialize() {
  const current = ++generation
  loading.value = true
  failed.value = false
  try {
    const response = await postAPI.categories()
    if (disposed || current !== generation) return
    categories.value = response.data.data
    categoriesReady = true
    await loadPosts()
  } catch {
    if (!disposed && current === generation) { failed.value = true; loading.value = false }
  }
}
function selectCategory(slug: string) {
  if (loading.value || selectedSlug.value === slug) return
  if (slug !== 'faq' && !scope.value.children.some((item) => item.slug === slug)) return
  selectedSlug.value = slug
  page.value = 1
  void loadPosts()
}
function submitSearch() { searchTerm.value = searchInput.value.trim(); page.value = 1; if (categoriesReady) void loadPosts() }
function clearSearch() { searchInput.value = ''; submitSearch() }
function changePage(value: number) { if (loading.value || value < 1 || value > totalPages.value) return; page.value = value; void loadPosts() }
function retry() { if (categoriesReady) void loadPosts(); else void initialize() }
onMounted(initialize)
onBeforeUnmount(() => { disposed = true; generation++ })
</script>

<style scoped>
.nt-faq { padding-top: 58px; padding-bottom: 80px; }
.nt-help-heading { max-width: 760px; margin: 0 auto 52px; text-align: center; }
.nt-help-eyebrow { display: inline-flex; align-items: center; gap: 8px; font-size: 13px; color: var(--ui-accent); }
.nt-help-heading h1 { margin: 16px 0; font-size: clamp(32px, 4.5vw, 52px); font-weight: 750; letter-spacing: -.04em; }
.nt-help-heading p { color: var(--ui-text-muted); font-size: 15px; line-height: 1.8; }
.nt-faq-search { display: flex; align-items: center; gap: 12px; margin: 28px auto 0; padding: 7px 7px 7px 18px; border: 1px solid var(--ui-border); border-radius: 15px; background: color-mix(in srgb, var(--ui-bg-elevated) 85%, transparent); color: var(--ui-text-muted); }
.nt-faq-search:focus-within { border-color: color-mix(in srgb, var(--ui-accent) 65%, transparent); }
.nt-faq-search input { min-width: 0; flex: 1; background: transparent; color: var(--ui-text-primary); font-size: 14px; outline: none; }
.nt-faq-search button, .nt-help-state button { display: inline-flex; align-items: center; gap: 8px; padding: 12px 16px; border-radius: 10px; background: var(--ui-accent); color: var(--ui-text-on-accent); font-size: 13px; font-weight: 650; white-space: nowrap; }
.nt-faq-layout { display: grid; grid-template-columns: 245px minmax(0, 1fr); gap: 44px; align-items: start; }
.nt-faq-sidebar nav { padding: 18px 12px; border: 1px solid var(--ui-border); border-radius: 17px; background: color-mix(in srgb, var(--ui-bg-elevated) 80%, transparent); }
.nt-faq-sidebar nav h2 { padding: 2px 12px 14px; color: var(--ui-text-muted); font-size: 12px; font-weight: 600; letter-spacing: .05em; }
.nt-faq-sidebar nav button { display: flex; align-items: center; gap: 10px; width: 100%; padding: 13px 12px; border: 1px solid transparent; border-radius: 10px; color: var(--ui-text-muted); font-size: 14px; text-align: left; }
.nt-faq-sidebar nav button > svg:last-child { margin-left: auto; }
.nt-faq-sidebar nav button.is-active { border-color: color-mix(in srgb, var(--ui-accent) 22%, transparent); background: color-mix(in srgb, var(--ui-accent) 8%, transparent); color: var(--ui-accent); }
.nt-faq-sidebar nav button:hover { color: var(--ui-text-primary); }
.nt-faq-contact { margin-top: 20px; padding: 25px; border: 1px solid color-mix(in srgb, var(--ui-accent) 16%, transparent); border-radius: 17px; background: linear-gradient(135deg, color-mix(in srgb, var(--ui-accent) 6%, transparent), transparent); }
.nt-faq-contact > svg { color: var(--ui-accent); }
.nt-faq-contact h2 { margin: 14px 0 8px; font-size: 15px; font-weight: 650; }
.nt-faq-contact p { color: var(--ui-text-muted); font-size: 13px; line-height: 1.9; }
.nt-faq-contact a, .nt-faq-answer-body > a { display: inline-flex; gap: 6px; align-items: center; margin-top: 17px; color: var(--ui-accent); font-size: 13px; }
.nt-faq-result { display: flex; align-items: center; justify-content: space-between; gap: 14px; min-height: 35px; margin-bottom: 20px; }
.nt-faq-result h2 { font-size: 20px; font-weight: 650; }
.nt-faq-result > span { color: var(--ui-text-muted); font-size: 12px; }
.nt-faq-answers { display: grid; gap: 12px; }
.nt-faq-answer { border: 1px solid var(--ui-border); border-radius: 14px; background: color-mix(in srgb, var(--ui-bg-elevated) 75%, transparent); }
.nt-faq-answer[open] { border-color: color-mix(in srgb, var(--ui-accent) 30%, transparent); }
.nt-faq-answer summary { display: flex; align-items: center; justify-content: space-between; gap: 18px; padding: 21px 23px; cursor: pointer; list-style: none; font-size: 15px; font-weight: 550; line-height: 1.6; }
.nt-faq-answer summary::-webkit-details-marker { display: none; }
.nt-faq-answer summary svg { flex-shrink: 0; color: var(--ui-accent); }
.nt-faq-answer[open] summary svg { transform: rotate(45deg); }
.nt-faq-answer-body { padding: 0 23px 23px; overflow-wrap: anywhere; }
.nt-faq-answer-body :deep(.prose) { color: var(--ui-text-muted); line-height: 1.9; }
.nt-faq-answer-body :deep(img) { max-width: 100%; height: auto; border-radius: 10px; }
.nt-faq-answer-body :deep(pre) { overflow-x: auto; }
.nt-help-state { display: grid; justify-items: center; align-content: center; gap: 14px; min-height: 300px; padding: 35px 25px; border: 1px dashed var(--ui-border); border-radius: 17px; text-align: center; }
.nt-help-state > svg { color: var(--ui-accent); }
.nt-help-state h3 { font-weight: 600; }
.nt-help-state p { max-width: 45ch; color: var(--ui-text-muted); font-size: 14px; line-height: 1.8; }
@media (max-width: 760px) { .nt-faq { padding-top: 36px; padding-bottom: 52px; } .nt-help-heading { margin-bottom: 32px; } .nt-faq-layout { grid-template-columns: 1fr; gap: 24px; } .nt-faq-sidebar nav { display: flex; flex-wrap: wrap; gap: 6px; padding: 12px; } .nt-faq-sidebar nav h2 { width: 100%; padding: 3px 6px 5px; } .nt-faq-sidebar nav button { width: auto; padding: 10px; font-size: 13px; } .nt-faq-sidebar nav button > svg:last-child { display: none; } .nt-faq-contact { display: none; } .nt-faq-answer summary { padding: 18px; } .nt-faq-answer-body { padding: 0 18px 18px; } }
</style>
