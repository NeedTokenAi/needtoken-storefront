<template>
  <div ref="scopeEl" class="nt-scope">
    <header class="nt-site-header sticky top-0 z-50 border-b border-border bg-background/90 backdrop-blur">
      <div class="nt-header-inner mx-auto flex h-16 w-full max-w-[1320px] items-center gap-2 sm:gap-4">
        <RouterLink to="/" class="nt-brand flex min-w-0 items-center gap-2 text-[15px] font-bold tracking-tight text-foreground sm:shrink-0 sm:text-[17px]" :title="brandName" :aria-label="brandName">
          <BrandMark v-if="!appStore.isResellerTenant" class="h-7 w-7 shrink-0" />
          <img v-else-if="resellerLogo" :src="resellerLogo" alt="" class="h-7 w-7 rounded-md object-contain" />
          <span class="truncate">{{ brandName }}</span>
        </RouterLink>

        <nav class="nt-desktop-navigation ml-4 flex items-center gap-1 max-[1239px]:hidden" :aria-label="t('needtoken.nav.label')">
          <RouterLink
            v-for="item in menuItems"
            :key="item.key"
            :to="item.path"
            class="whitespace-nowrap rounded-md px-3 py-2 text-[14px] font-medium text-muted-foreground transition-colors hover:bg-secondary hover:text-foreground"
            :class="isActive(item) ? '!text-primary' : ''"
            :aria-current="isActive(item) ? 'page' : undefined"
          >{{ item.label }}</RouterLink>
          <div v-for="group in navGroups" :key="group.key" :ref="el => setGroupEl(group.key, el)" class="relative" @focusout="onGroupFocusOut($event, group.key)" @keydown="onGroupKeydown($event, group.key)">
            <button :id="`nt-nav-${group.key}`" type="button" class="nt-nav-group-button" :class="group.items.some(isActive) ? 'text-primary' : 'text-muted-foreground'" :aria-expanded="openGroup === group.key" :aria-controls="`nt-nav-panel-${group.key}`" @click="toggleGroup(group.key)">
              {{ group.label }}<ChevronDown class="h-3.5 w-3.5 transition-transform" :class="openGroup === group.key ? 'rotate-180' : ''" />
            </button>
            <div v-if="openGroup === group.key" :id="`nt-nav-panel-${group.key}`" class="nt-nav-dropdown" :aria-labelledby="`nt-nav-${group.key}`">
              <RouterLink v-for="item in group.items" :key="item.key" :to="item.path" class="nt-nav-dropdown-link" :class="isActive(item) ? 'text-primary bg-primary/5' : 'text-muted-foreground'" :aria-current="isActive(item) ? 'page' : undefined" @click="closeMenus">{{ item.label }}<ArrowUpRight class="h-3.5 w-3.5 opacity-50" /></RouterLink>
            </div>
          </div>
        </nav>

        <div class="ml-auto flex shrink-0 items-center gap-2">
          <RouterLink to="/guest/orders" class="grid h-9 w-9 place-items-center rounded-md text-muted-foreground hover:bg-secondary hover:text-foreground max-[419px]:hidden" :aria-label="t('navbar.guestOrders')" :title="t('navbar.guestOrders')">
            <ClipboardList class="h-[18px] w-[18px]" />
          </RouterLink>
          <RouterLink to="/cart" class="relative grid h-9 w-9 place-items-center rounded-md text-muted-foreground hover:bg-secondary hover:text-foreground" :aria-label="t('navbar.cart')">
            <ShoppingCart class="h-[18px] w-[18px]" />
            <span v-if="cartCount > 0" class="absolute -right-1 -top-1 grid h-[18px] min-w-[18px] place-items-center rounded-full bg-primary px-1 text-[11px] font-bold text-primary-foreground">{{ cartCount }}</span>
          </RouterLink>
          <div ref="langEl" class="relative max-[1239px]:hidden" @focusout="onLanguageFocusOut">
            <button type="button" class="grid h-9 w-9 place-items-center rounded-md text-muted-foreground hover:bg-secondary hover:text-foreground" :aria-label="t('navbar.selectLanguage')" :aria-expanded="langOpen" aria-controls="nt-language-options" @click="langOpen = !langOpen; openGroup = null">
              <Languages class="h-[18px] w-[18px]" />
            </button>
            <div v-if="langOpen" id="nt-language-options" class="absolute right-0 top-[calc(100%+8px)] z-[60] flex min-w-[150px] flex-col rounded-md border border-border bg-card p-1.5">
              <button v-for="lang in languages" :key="lang.code" type="button" class="rounded px-3 py-2 text-left text-sm hover:bg-secondary" :class="appStore.locale === lang.code ? 'text-primary' : 'text-muted-foreground'" @click="changeLanguage(lang.code)">{{ lang.name }}</button>
            </div>
          </div>
          <RouterLink v-if="authStore.isAuthenticated" to="/me" class="rounded-md bg-primary px-3 py-1.5 text-[13px] font-semibold text-primary-foreground hover:bg-primary/90">{{ t('navbar.personalCenter') }}</RouterLink>
          <RouterLink v-else :to="loginTarget" class="nt-auth-entry inline-flex min-h-9 items-center whitespace-nowrap rounded-md bg-primary px-3 py-1.5 text-[13px] font-semibold text-primary-foreground hover:bg-primary/90">{{ t('needtoken.nav.authEntry') }}</RouterLink>

          <div ref="moreEl" class="relative hidden max-[1239px]:block" @focusout="onMoreFocusOut">
            <button ref="moreButtonEl" type="button" class="grid h-9 w-9 place-items-center rounded-md text-muted-foreground hover:bg-secondary" :aria-label="t('navbar.more')" :aria-expanded="moreOpen" aria-controls="nt-mobile-navigation" @click="moreOpen = !moreOpen; openGroup = null">
              <Menu v-if="!moreOpen" class="pointer-events-none h-[18px] w-[18px]" />
              <X v-else class="pointer-events-none h-[18px] w-[18px]" />
            </button>
            <nav v-if="moreOpen" id="nt-mobile-navigation" class="nt-mobile-navigation absolute right-0 top-[calc(100%+8px)] z-[60] flex min-w-[230px] flex-col rounded-xl border border-border bg-card p-2" :aria-label="t('needtoken.nav.label')">
              <RouterLink v-for="item in menuItems" :key="`m-${item.key}`" :to="item.path" class="rounded px-3 py-2 text-sm text-muted-foreground hover:bg-secondary hover:text-foreground" :class="isActive(item) ? '!text-primary' : ''" :aria-current="isActive(item) ? 'page' : undefined" @click="moreOpen = false">{{ item.label }}</RouterLink>
              <div v-for="group in navGroups" :key="`m-${group.key}`" class="mt-1 border-t border-border pt-1">
                <button type="button" class="flex w-full items-center justify-between rounded px-3 py-2 text-left text-sm text-muted-foreground hover:bg-secondary hover:text-foreground" :aria-expanded="mobileGroup === group.key" :aria-controls="`nt-mobile-${group.key}`" @click="mobileGroup = mobileGroup === group.key ? null : group.key">{{ group.label }}<ChevronDown class="h-3.5 w-3.5 transition-transform" :class="mobileGroup === group.key ? 'rotate-180' : ''" /></button>
                <div v-if="mobileGroup === group.key" :id="`nt-mobile-${group.key}`" class="ml-3 border-l border-border pl-2">
                  <RouterLink v-for="item in group.items" :key="`m-${item.key}`" :to="item.path" class="block rounded px-3 py-2 text-sm hover:bg-secondary" :class="isActive(item) ? 'text-primary' : 'text-muted-foreground'" :aria-current="isActive(item) ? 'page' : undefined" @click="closeMenus">{{ item.label }}</RouterLink>
                </div>
              </div>
              <RouterLink to="/guest/orders" class="rounded px-3 py-2 text-sm text-muted-foreground hover:bg-secondary hover:text-foreground" @click="moreOpen = false">{{ t('navbar.guestOrders') }}</RouterLink>
              <RouterLink v-if="authStore.isAuthenticated" to="/me" class="rounded px-3 py-2 text-sm text-muted-foreground hover:bg-secondary hover:text-foreground" @click="closeMenus">{{ t('navbar.personalCenter') }}</RouterLink>
              <span class="px-3 pb-1 pt-2 text-[11px] uppercase tracking-wider text-muted-foreground">{{ t('navbar.selectLanguage') }}</span>
              <button v-for="lang in languages" :key="`ml-${lang.code}`" type="button" class="rounded px-3 py-2 text-left text-sm hover:bg-secondary" :class="appStore.locale === lang.code ? 'text-primary' : 'text-muted-foreground'" @click="changeLanguage(lang.code); moreOpen = false">{{ lang.name }}</button>
            </nav>
          </div>
        </div>
      </div>
    </header>

    <main class="relative isolate flex-1">
      <GhostFibersBackground v-if="route.path !== '/'" variant="subtle" />
      <div class="relative z-[1]"><slot /></div>
    </main>

    <NeedtokenFooter :brand-name="brandName" :reseller-logo="resellerLogo" :support="support" />
    <SupportWidget :support="support" :raised="raiseSupport" />
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch, type ComponentPublicInstance } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import { ArrowUpRight, ChevronDown, ClipboardList, Languages, Menu, ShoppingCart, X } from 'lucide-vue-next'
import { useAppStore } from '../../../stores/app'
import { useCartStore } from '../../../stores/cart'
import { useUserAuthStore } from '../../../stores/userAuth'
import { getImageUrl } from '../../../utils/image'
import { authReturnPath as safeAuthReturnPath } from '../../../utils/authReturnPath'
import { API_PLANS_PATH, API_SERVICES_PATH, CATEGORY_SLUGS, NEEDTOKEN_BRAND_NAME, NEEDTOKEN_BRAND_ICON, NEEDTOKEN_TOUCH_ICON } from '../config'
import BrandMark from '../components/BrandMark.vue'
import NeedtokenFooter from '../components/NeedtokenFooter.vue'
import GhostFibersBackground from '../components/GhostFibersBackground.vue'
import SupportWidget from '../components/SupportWidget.vue'
import { useNeedtokenEffects } from '../composables/useNeedtokenEffects'
import { normalizeNeedtokenSupport } from '../lib/supportConfig'
import '../styles/needtoken.css'

const { t } = useI18n()
const route = useRoute()
const appStore = useAppStore()
const cartStore = useCartStore()
const authStore = useUserAuthStore()
const authReturnPath = computed(() => safeAuthReturnPath(route.path.startsWith('/auth/') ? route.query.redirect : route.fullPath, '/'))
const loginTarget = computed(() => ({ path: '/auth/login', query: { redirect: authReturnPath.value } }))
const scopeEl = ref<HTMLElement | null>(null)
useNeedtokenEffects(scopeEl)

const langOpen = ref(false)
const moreOpen = ref(false)
const langEl = ref<HTMLElement | null>(null)
const moreEl = ref<HTMLElement | null>(null)
const moreButtonEl = ref<HTMLButtonElement | null>(null)
const openGroup = ref<string | null>(null)
const mobileGroup = ref<string | null>(null)
const groupEls = new Map<string, HTMLElement>()

const brandName = computed(() => appStore.isResellerTenant
  ? String(appStore.config?.brand?.site_name || '').trim() || NEEDTOKEN_BRAND_NAME
  : NEEDTOKEN_BRAND_NAME)
const resellerLogo = computed(() => {
  const logo = String(appStore.config?.brand?.site_logo || '').trim()
  return logo ? getImageUrl(logo) : ''
})
const previousBrandOverride = appStore.storefrontBrandOverride
watch(() => appStore.isResellerTenant, (isReseller) => {
  appStore.setStorefrontBrandOverride(isReseller ? null : {
    siteName: NEEDTOKEN_BRAND_NAME,
    siteIcon: NEEDTOKEN_BRAND_ICON,
    touchIcon: NEEDTOKEN_TOUCH_ICON,
  })
}, { immediate: true })
const support = computed(() => normalizeNeedtokenSupport(appStore.isResellerTenant ? undefined : appStore.config?.needtoken_support, !appStore.isResellerTenant))
const raiseSupport = computed(() => /^\/(?:products\/|checkout(?:\/|$)|pay(?:ment)?(?:\/|$))/.test(route.path))
const cartCount = computed(() => cartStore.totalItems)

interface MenuItem { key: string; path: string; label: string }

const menuItems = computed<MenuItem[]>(() => [
  { key: 'home', path: '/', label: t('needtoken.nav.home') },
  { key: 'recharge', path: `/categories/${CATEGORY_SLUGS.recharge}`, label: t('needtoken.nav.recharge') },
  { key: 'rental', path: `/categories/${CATEGORY_SLUGS.rental}`, label: t('needtoken.nav.rental') },
  { key: 'plans', path: API_SERVICES_PATH, label: t('needtoken.nav.plans') },
  { key: 'models', path: '/models', label: t('needtoken.nav.models') },
])
const navGroups = computed(() => [
  { key: 'help', label: t('needtoken.nav.help'), items: [
    { key: 'blog', path: '/blog', label: t('needtoken.nav.blog') },
    { key: 'docs', path: '/docs', label: t('needtoken.nav.docs') },
    { key: 'faq', path: '/faq', label: t('needtoken.nav.faq') },
  ] },
  { key: 'tools', label: t('needtoken.nav.tools'), items: [
    { key: 'environment', path: '/tools/environment', label: t('needtoken.nav.environment') },
    { key: 'appleIds', path: '/free-apple-id', label: t('needtoken.nav.appleIds') },
  ] },
])

const isActive = (item: MenuItem) => item.key === 'plans'
  ? [API_PLANS_PATH, API_SERVICES_PATH].some(path => route.path === path || route.path.startsWith(`${path}/`))
  : item.path === '/' ? route.path === '/' : route.path === item.path || route.path.startsWith(`${item.path}/`)

function setGroupEl(key: string, el: Element | ComponentPublicInstance | null) {
  if (el instanceof HTMLElement) groupEls.set(key, el)
  else groupEls.delete(key)
}
function toggleGroup(key: string) {
  openGroup.value = openGroup.value === key ? null : key
  langOpen.value = false
}
function closeMenus() {
  moreOpen.value = false
  openGroup.value = null
  mobileGroup.value = null
  langOpen.value = false
}
function onGroupFocusOut(event: FocusEvent, key: string) {
  if (event.relatedTarget instanceof Node && !groupEls.get(key)?.contains(event.relatedTarget)) openGroup.value = null
}
function onMoreFocusOut(event: FocusEvent) {
  if (event.relatedTarget instanceof Node && !moreEl.value?.contains(event.relatedTarget)) {
    moreOpen.value = false
    mobileGroup.value = null
  }
}
function onLanguageFocusOut(event: FocusEvent) {
  if (event.relatedTarget instanceof Node && !langEl.value?.contains(event.relatedTarget)) langOpen.value = false
}
function onGroupKeydown(event: KeyboardEvent, key: string) {
  if (event.key === 'Escape') {
    event.preventDefault()
    event.stopPropagation()
    openGroup.value = null
    groupEls.get(key)?.querySelector<HTMLButtonElement>('button')?.focus()
    return
  }
  if (!['ArrowDown', 'ArrowUp', 'Home', 'End'].includes(event.key)) return
  event.preventDefault()
  openGroup.value = key
  void nextTick(() => {
    const links = [...(groupEls.get(key)?.querySelectorAll<HTMLAnchorElement>('a') || [])]
    if (!links.length) return
    const index = links.indexOf(document.activeElement as HTMLAnchorElement)
    const next = event.key === 'Home' ? 0 : event.key === 'End' ? links.length - 1 : event.key === 'ArrowDown' ? (index + 1) % links.length : index <= 0 ? links.length - 1 : index - 1
    links[next]?.focus()
  })
}
function onDocumentKeydown(event: KeyboardEvent) {
  if (event.key !== 'Escape') return
  if (moreOpen.value) moreButtonEl.value?.focus()
  else if (langOpen.value) langEl.value?.querySelector<HTMLButtonElement>('button')?.focus()
  closeMenus()
}
watch(() => route.fullPath, closeMenus)

const languages = [
  { code: 'zh-CN', name: '简体中文' },
  { code: 'zh-TW', name: '繁體中文' },
  { code: 'en-US', name: 'English' },
]

const changeLanguage = (code: string) => {
  appStore.setLocale(code)
  langOpen.value = false
}

const onDocClick = (e: MouseEvent) => {
  const target = e.target as Node
  if (langOpen.value && langEl.value && !langEl.value.contains(target)) langOpen.value = false
  if (moreOpen.value && moreEl.value && !moreEl.value.contains(target)) moreOpen.value = false
  if (openGroup.value && !groupEls.get(openGroup.value)?.contains(target)) openGroup.value = null
}

// 深色是本模板的固定风格：挂载时打上 .dark，卸载（切回其他模板）时恢复原状，不写入用户主题偏好
let hadDark = false
onMounted(() => {
  document.addEventListener('click', onDocClick)
  document.addEventListener('keydown', onDocumentKeydown)
  hadDark = document.documentElement.classList.contains('dark')
  document.documentElement.classList.add('dark')
  document.body.classList.add('nt-tokens')
})
onUnmounted(() => {
  appStore.setStorefrontBrandOverride(previousBrandOverride)
  document.removeEventListener('click', onDocClick)
  document.removeEventListener('keydown', onDocumentKeydown)
  if (!hadDark) document.documentElement.classList.remove('dark')
  document.body.classList.remove('nt-tokens')
})
</script>

<style scoped>
.nt-site-header { padding-inline: 24px; }
@media (max-width: 640px) { .nt-site-header { padding-inline: 14px; } }
.nt-nav-group-button { display: flex; align-items: center; gap: 5px; border-radius: 6px; padding: 8px 10px; white-space: nowrap; font-size: 14px; font-weight: 500; }
.nt-nav-group-button:hover { background: var(--ui-bg-soft); color: var(--ui-text-primary); }
.nt-nav-dropdown { position: absolute; top: calc(100% + 10px); left: 0; z-index: 60; width: 186px; padding: 7px; border: 1px solid var(--ui-border); border-radius: 12px; background: #0e1218; box-shadow: 0 12px 32px #0005; }
.nt-nav-dropdown-link { display: flex; justify-content: space-between; align-items: center; gap: 16px; padding: 10px 12px; border-radius: 6px; font-size: 13px; }
.nt-nav-dropdown-link:hover { color: var(--ui-accent); background: var(--ui-bg-soft); }
.nt-mobile-navigation { max-height: calc(100dvh - 90px); overflow-y: auto; overscroll-behavior: contain; }
</style>
