<template>
  <div ref="scopeEl" class="nt-scope" @pointerover.passive="prefetchLink" @focusin="prefetchLink" @touchstart.passive="prefetchLink">
    <header class="nt-site-header sticky top-0 z-50 border-b border-border backdrop-blur">
      <div class="nt-header-inner mx-auto flex h-16 w-full max-w-[1320px] items-center gap-2 sm:gap-4">
        <RouterLink to="/" class="nt-brand flex min-w-0 items-center gap-2 text-[15px] font-bold tracking-tight text-foreground sm:shrink-0 sm:text-[17px]" :title="brandName" :aria-label="brandName">
          <BrandMark v-if="!appStore.isResellerTenant" class="h-7 w-7 shrink-0" />
          <img v-else-if="resellerLogo" :src="resellerLogo" alt="" class="h-7 w-7 rounded-md object-contain" />
          <span class="truncate">{{ brandName }}</span>
        </RouterLink>

        <nav ref="navRailEl" class="nt-desktop-navigation ml-4 flex items-center gap-0.5 max-[1239px]:hidden" :aria-label="t('needtoken.nav.label')">
          <span class="nt-sliding-thumb nt-nav-thumb" :class="{ 'is-ready': navIndicatorReady }" :style="navIndicatorStyle" aria-hidden="true" />
          <template v-for="node in navNodes" :key="node.key">
            <RouterLink v-if="node.kind === 'link'" :to="node.path" class="nt-nav-link inline-flex items-center gap-1.5 whitespace-nowrap rounded-full px-3.5 py-1.5 text-[14px] font-medium text-muted-foreground hover:text-foreground" :class="isActive(node) ? 'is-active !text-primary' : ''" :data-nav-active="isActive(node) ? 'true' : undefined" :aria-current="isActive(node) ? 'page' : undefined" @click="triggerIconBurst(node.key)">
              <NavAnimatedIcon :name="node.icon" :active="isActive(node)" :burst="iconBursts[node.key]" />
              <span>{{ node.label }}</span>
            </RouterLink>
            <div v-else :ref="el => setGroupEl(node.key, el)" class="relative" @focusout="onGroupFocusOut($event, node.key)" @keydown="onGroupKeydown($event, node.key)">
              <button :id="`nt-nav-${node.key}`" type="button" class="nt-nav-group-button" :class="node.items.some(isActive) ? 'is-active text-primary' : 'text-muted-foreground'" :data-nav-active="node.items.some(isActive) ? 'true' : undefined" :aria-expanded="openGroup === node.key" :aria-controls="`nt-nav-panel-${node.key}`" @click="toggleGroup(node.key)">
                <NavAnimatedIcon :name="node.icon" :active="node.items.some(isActive) || openGroup === node.key" :burst="iconBursts[node.key]" />
                <span>{{ node.label }}</span>
                <ChevronDown class="h-3.5 w-3.5 transition-transform" :class="openGroup === node.key ? 'rotate-180' : ''" />
              </button>
              <Transition name="nt-menu" @before-enter="showEnteringPanel" @before-leave="hideLeavingPanel">
              <div v-if="openGroup === node.key" :id="`nt-nav-panel-${node.key}`" class="nt-nav-dropdown" :aria-labelledby="`nt-nav-${node.key}`">
                <template v-for="item in node.items" :key="item.key">
                  <span v-if="item.disabled" class="nt-nav-dropdown-link is-disabled" aria-disabled="true">{{ item.label }}<em>{{ item.badge }}</em></span>
                  <RouterLink v-else :to="item.path" class="nt-nav-dropdown-link" :class="isActive(item) ? 'text-primary bg-primary/5' : 'text-muted-foreground'" :aria-current="isActive(item) ? 'page' : undefined" @click="closeMenus">{{ item.label }}<ArrowUpRight class="h-3.5 w-3.5 opacity-50" /></RouterLink>
                </template>
              </div>
              </Transition>
            </div>
          </template>
        </nav>

        <div class="ml-auto flex shrink-0 items-center gap-2">
          <RouterLink to="/guest/orders" class="grid h-9 w-9 place-items-center rounded-md text-muted-foreground hover:bg-secondary hover:text-foreground max-[419px]:hidden" :aria-label="t('navbar.guestOrders')" :title="t('navbar.guestOrders')">
            <ClipboardList class="h-[18px] w-[18px]" />
          </RouterLink>
          <div ref="langEl" class="relative max-[1239px]:hidden" @focusout="onLanguageFocusOut">
            <button type="button" class="grid h-9 w-9 place-items-center rounded-md text-muted-foreground hover:bg-secondary hover:text-foreground" :aria-label="t('navbar.selectLanguage')" :aria-expanded="langOpen" aria-controls="nt-language-options" @click="langOpen = !langOpen; openGroup = null">
              <Languages class="h-[18px] w-[18px]" />
            </button>
            <Transition name="nt-menu" @before-enter="showEnteringPanel" @before-leave="hideLeavingPanel">
            <div v-if="langOpen" id="nt-language-options" class="nt-language-options absolute right-0 top-[calc(100%+8px)] z-[60] flex min-w-[150px] flex-col rounded-md border border-border bg-card p-2">
              <button v-for="lang in languages" :key="lang.code" type="button" class="rounded px-3 py-2 text-left text-sm hover:bg-secondary" :class="appStore.locale === lang.code ? 'text-primary' : 'text-muted-foreground'" @click="changeLanguage(lang.code)">{{ lang.name }}</button>
            </div>
            </Transition>
          </div>
          <ThemeToggle class="max-[479px]:hidden" />
          <RouterLink v-if="authStore.isAuthenticated" to="/me" class="rounded-md bg-primary px-3 py-1.5 text-[13px] font-semibold text-primary-foreground hover:bg-primary/90">{{ t('navbar.personalCenter') }}</RouterLink>
          <RouterLink v-else :to="loginTarget" class="nt-auth-entry inline-flex min-h-9 items-center whitespace-nowrap rounded-md bg-primary px-3 py-1.5 text-[13px] font-semibold text-primary-foreground hover:bg-primary/90">{{ t('needtoken.nav.authEntry') }}</RouterLink>

          <div ref="moreEl" class="relative hidden max-[1239px]:block" @focusout="onMoreFocusOut">
            <button ref="moreButtonEl" type="button" class="grid h-9 w-9 place-items-center rounded-md text-muted-foreground hover:bg-secondary" :aria-label="t('navbar.more')" :aria-expanded="moreOpen" aria-controls="nt-mobile-navigation" @click="moreOpen = !moreOpen; openGroup = null">
              <Menu v-if="!moreOpen" class="pointer-events-none h-[18px] w-[18px]" />
              <X v-else class="pointer-events-none h-[18px] w-[18px]" />
            </button>
            <Transition name="nt-menu" @before-enter="showEnteringPanel" @before-leave="hideLeavingPanel">
            <nav v-if="moreOpen" id="nt-mobile-navigation" class="nt-mobile-navigation absolute right-0 top-[calc(100%+8px)] z-[60] flex min-w-[230px] flex-col rounded-xl border border-border bg-card p-2" :aria-label="t('needtoken.nav.label')">
              <template v-for="node in navNodes" :key="`m-${node.key}`">
                <RouterLink v-if="node.kind === 'link'" :to="node.path" class="flex items-center gap-2 rounded px-3 py-2 text-sm text-muted-foreground hover:bg-secondary hover:text-foreground" :class="isActive(node) ? '!text-primary' : ''" :aria-current="isActive(node) ? 'page' : undefined" @click="triggerIconBurst(node.key); moreOpen = false">
                  <NavAnimatedIcon :name="node.icon" :active="isActive(node)" :burst="iconBursts[node.key]" />
                  <span>{{ node.label }}</span>
                </RouterLink>
                <div v-else class="mt-1 border-t border-border pt-1">
                  <button type="button" class="flex w-full items-center justify-between rounded px-3 py-2 text-left text-sm text-muted-foreground hover:bg-secondary hover:text-foreground" :aria-expanded="mobileGroup === node.key" :aria-controls="`nt-mobile-${node.key}`" @click="triggerIconBurst(node.key); mobileGroup = mobileGroup === node.key ? null : node.key">
                    <span class="inline-flex items-center gap-2"><NavAnimatedIcon :name="node.icon" :active="node.items.some(isActive) || mobileGroup === node.key" :burst="iconBursts[node.key]" /><span>{{ node.label }}</span></span>
                    <ChevronDown class="h-3.5 w-3.5 transition-transform" :class="mobileGroup === node.key ? 'rotate-180' : ''" />
                  </button>
                  <div v-if="mobileGroup === node.key" :id="`nt-mobile-${node.key}`" class="ml-3 border-l border-border pl-2">
                    <template v-for="item in node.items" :key="`m-${item.key}`">
                      <span v-if="item.disabled" class="nt-mobile-disabled flex items-center justify-between rounded px-3 py-2 text-sm text-muted-foreground" aria-disabled="true">{{ item.label }}<em class="text-[11px] not-italic">{{ item.badge }}</em></span>
                      <RouterLink v-else :to="item.path" class="block rounded px-3 py-2 text-sm hover:bg-secondary" :class="isActive(item) ? 'text-primary' : 'text-muted-foreground'" :aria-current="isActive(item) ? 'page' : undefined" @click="closeMenus">{{ item.label }}</RouterLink>
                    </template>
                  </div>
                </div>
              </template>
              <RouterLink to="/guest/orders" class="rounded px-3 py-2 text-sm text-muted-foreground hover:bg-secondary hover:text-foreground" @click="moreOpen = false">{{ t('navbar.guestOrders') }}</RouterLink>
              <RouterLink v-if="authStore.isAuthenticated" to="/me" class="rounded px-3 py-2 text-sm text-muted-foreground hover:bg-secondary hover:text-foreground" @click="closeMenus">{{ t('navbar.personalCenter') }}</RouterLink>
              <span class="px-3 pb-1 pt-2 text-xs text-muted-foreground">{{ t('needtoken.theme.label') }}</span>
              <div class="px-2 pb-1"><ThemeToggle variant="menu" /></div>
              <span class="px-3 pb-1 pt-2 text-xs text-muted-foreground">{{ t('navbar.selectLanguage') }}</span>
              <button v-for="lang in languages" :key="`ml-${lang.code}`" type="button" class="rounded px-3 py-2 text-left text-sm hover:bg-secondary" :class="appStore.locale === lang.code ? 'text-primary' : 'text-muted-foreground'" @click="changeLanguage(lang.code); moreOpen = false">{{ lang.name }}</button>
            </nav>
            </Transition>
          </div>
        </div>
      </div>
    </header>

    <main class="relative isolate flex-1">
      <GhostFibersBackground v-if="route.path !== '/' && !route.path.startsWith('/tools/')" variant="subtle" />
      <div class="nt-page-content relative z-[1]"><slot /></div>
    </main>

    <NeedtokenFooter :brand-name="brandName" :reseller-logo="resellerLogo" :support="support" />
    <SupportWidget :support="support" :raised="raiseSupport" />
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, reactive, ref, watch, type ComponentPublicInstance } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import { ArrowUpRight, ChevronDown, ClipboardList, Languages, Menu, X } from 'lucide-vue-next'
import { useAppStore } from '../../../stores/app'
import { useUserAuthStore } from '../../../stores/userAuth'
import { getImageUrl } from '../../../utils/image'
import { authReturnPath as safeAuthReturnPath } from '../../../utils/authReturnPath'
import { API_PLANS_PATH, API_SERVICES_PATH, CATEGORY_SLUGS, NEEDTOKEN_BRAND_NAME, NEEDTOKEN_BRAND_ICON, NEEDTOKEN_TOUCH_ICON } from '../config'
import BrandMark from '../components/BrandMark.vue'
import NavAnimatedIcon, { type NavIconName } from '../components/NavAnimatedIcon.vue'
import NeedtokenFooter from '../components/NeedtokenFooter.vue'
import GhostFibersBackground from '../components/GhostFibersBackground.vue'
import SupportWidget from '../components/SupportWidget.vue'
import ThemeToggle from '../components/ThemeToggle.vue'
import { useNeedtokenEffects } from '../composables/useNeedtokenEffects'
import { useSlidingIndicator } from '../composables/useSlidingIndicator'
import { prefetchStorefrontRoute } from '../../../router'
import { normalizeNeedtokenSupport } from '../lib/supportConfig'
import { useTheme } from '../../../utils/theme'
import '../styles/needtoken.css'

const { t } = useI18n()
const route = useRoute()
const appStore = useAppStore()
const authStore = useUserAuthStore()
const authReturnPath = computed(() => safeAuthReturnPath(route.path.startsWith('/auth/') ? route.query.redirect : route.fullPath, '/'))
const loginTarget = computed(() => ({ path: '/auth/login', query: { redirect: authReturnPath.value } }))
const scopeEl = ref<HTMLElement | null>(null)
const navRailEl = ref<HTMLElement | null>(null)
const iconBursts = reactive<Record<string, number>>({})
function triggerIconBurst(key: string) {
  iconBursts[key] = (iconBursts[key] || 0) + 1
}
useNeedtokenEffects(scopeEl)
const { indicatorStyle: navIndicatorStyle, indicatorReady: navIndicatorReady } = useSlidingIndicator(
  navRailEl,
  '[data-nav-active="true"]',
  [() => route.path, () => appStore.locale],
)

function prefetchLink(event: Event) {
  const link = event.target instanceof Element ? event.target.closest<HTMLAnchorElement>('a[href]') : null
  if (link && !link.hasAttribute('download') && link.target !== '_blank') prefetchStorefrontRoute(link.href)
}

const langOpen = ref(false)
const moreOpen = ref(false)
const langEl = ref<HTMLElement | null>(null)
const moreEl = ref<HTMLElement | null>(null)
const moreButtonEl = ref<HTMLButtonElement | null>(null)
const openGroup = ref<string | null>(null)
const mobileGroup = ref<string | null>(null)
const groupEls = new Map<string, HTMLElement>()
const hideLeavingPanel = (element: Element) => { element.setAttribute('inert', ''); element.setAttribute('aria-hidden', 'true') }
const showEnteringPanel = (element: Element) => { element.removeAttribute('inert'); element.removeAttribute('aria-hidden') }

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

interface NavLink { kind: 'link'; key: string; path: string; label: string; icon?: NavIconName; disabled?: boolean; badge?: string }
interface NavTopLink extends NavLink { icon: NavIconName }
interface NavGroup { kind: 'group'; key: string; label: string; icon: NavIconName; items: NavLink[] }

const link = (key: string, path: string, label: string, extra: Partial<NavLink> = {}): NavLink => ({ kind: 'link', key, path, label, ...extra })
const navNodes = computed<(NavTopLink | NavGroup)[]>(() => [
  { kind: 'link', key: 'home', path: '/', label: t('needtoken.nav.home'), icon: 'home' },
  { kind: 'group', key: 'personal', label: t('needtoken.nav.personal'), icon: 'personal', items: [
    link('recharge', `/categories/${CATEGORY_SLUGS.recharge}`, t('needtoken.nav.recharge')),
    link('rental', `/categories/${CATEGORY_SLUGS.rental}`, t('needtoken.nav.rental')),
    link('workstation', '', t('needtoken.nav.workstation'), { disabled: true, badge: t('needtoken.nav.comingSoon') }),
  ] },
  { kind: 'group', key: 'gateway', label: t('needtoken.nav.gateway'), icon: 'gateway', items: [
    link('plans', API_SERVICES_PATH, t('needtoken.nav.plans')),
    link('models', '/models', t('needtoken.nav.models')),
    link('docs', '/docs', t('needtoken.nav.docs')),
  ] },
  { kind: 'link', key: 'radar', path: '/tools/radar', label: t('needtoken.nav.radar'), icon: 'radar' },
  { kind: 'link', key: 'accountCheck', path: '/tools/codex-account', label: t('needtoken.nav.accountCheck'), icon: 'accountCheck' },
  { kind: 'group', key: 'help', label: t('needtoken.nav.help'), icon: 'help', items: [
    link('blog', '/blog', t('needtoken.nav.blog')),
    link('faq', '/faq', t('needtoken.nav.faq')),
  ] },
])

const isActive = (item: { key: string; path?: string; disabled?: boolean }) => {
  if (item.disabled || !item.path) return false
  if (item.key === 'plans') return [API_PLANS_PATH, API_SERVICES_PATH].some(path => route.path === path || route.path.startsWith(`${path}/`))
  if (item.key === 'accountCheck') return ['/tools/codex-account', '/tools/environment'].some(path => route.path === path || route.path.startsWith(`${path}/`))
  return item.path === '/' ? route.path === '/' : route.path === item.path || route.path.startsWith(`${item.path}/`)
}

function setGroupEl(key: string, el: Element | ComponentPublicInstance | null) {
  if (el instanceof HTMLElement) groupEls.set(key, el)
  else groupEls.delete(key)
}
function toggleGroup(key: string) {
  triggerIconBurst(key)
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
  if (!(event.relatedTarget instanceof Node) || !groupEls.get(key)?.contains(event.relatedTarget)) openGroup.value = null
}
function onMoreFocusOut(event: FocusEvent) {
  if (!(event.relatedTarget instanceof Node) || !moreEl.value?.contains(event.relatedTarget)) {
    moreOpen.value = false
    mobileGroup.value = null
  }
}
function onLanguageFocusOut(event: FocusEvent) {
  if (!(event.relatedTarget instanceof Node) || !langEl.value?.contains(event.relatedTarget)) langOpen.value = false
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

// 跟随系统明暗并支持用户手动切换；挂载/卸载时仅维护 body.nt-tokens 令牌作用域
useTheme()
onMounted(() => {
  document.addEventListener('click', onDocClick)
  document.addEventListener('keydown', onDocumentKeydown)
  document.body.classList.add('nt-tokens')
})
onUnmounted(() => {
  appStore.setStorefrontBrandOverride(previousBrandOverride)
  document.removeEventListener('click', onDocClick)
  document.removeEventListener('keydown', onDocumentKeydown)
  document.body.classList.remove('nt-tokens')
})
</script>

<style scoped>
.nt-site-header { padding-inline: 24px; background: var(--nt-header-bg); }
@media (min-width: 1240px) { .nt-header-inner { padding-inline: 24px; } }
@media (max-width: 640px) { .nt-site-header { padding-inline: 14px; } }
.nt-site-header :where(a, button) { transition: color var(--nt-motion-fast) var(--nt-motion-ease), background-color var(--nt-motion-fast) var(--nt-motion-ease), transform var(--nt-duration-rebound) var(--nt-ease-spring); }
.nt-desktop-navigation { position: relative; padding: 4px; border: 1px solid var(--ui-border); border-radius: 999px; background: var(--nt-nav-rail-bg); }
.nt-nav-thumb { border-radius: 999px; background: var(--nt-nav-thumb-bg); box-shadow: var(--nt-nav-thumb-shadow); }
.nt-nav-link { position: relative; z-index: 1; }
.nt-nav-link.is-active, .nt-nav-group-button.is-active { font-weight: 600; }
.nt-nav-group-button { position: relative; z-index: 1; display: flex; align-items: center; gap: 6px; border-radius: 999px; padding: 6px 14px; white-space: nowrap; font-size: 14px; font-weight: 500; }
.nt-nav-group-button:hover, .nt-nav-group-button[aria-expanded="true"] { color: var(--ui-text-primary); }
.nt-nav-group-button:not(.is-active):hover, .nt-nav-group-button:not(.is-active)[aria-expanded="true"] { background: var(--nt-surface-subtle); }
.nt-nav-dropdown { position: absolute; top: calc(100% + 10px); left: 0; z-index: 60; width: max-content; min-width: 192px; max-width: min(320px, 70vw); padding: 8px; border: 1px solid var(--ui-border); border-radius: 16px; background: var(--ui-bg-elevated); box-shadow: var(--nt-popover-shadow); transform-origin: top left; }
.nt-nav-dropdown-link { display: flex; justify-content: space-between; align-items: center; gap: 16px; min-height: 40px; padding: 8px 12px; border-radius: 10px; font-size: 13px; }
.nt-nav-dropdown-link:hover { color: var(--ui-accent); background: var(--ui-bg-soft); }
.nt-nav-dropdown-link.is-disabled { color: var(--ui-text-muted); cursor: default; }
.nt-nav-dropdown-link.is-disabled em, .nt-mobile-disabled em { font-style: normal; font-size: 11px; color: var(--nt-warning-ink); }
.nt-language-options, .nt-mobile-navigation { box-shadow: var(--nt-popover-shadow); transform-origin: top right; }
.nt-mobile-navigation { max-height: calc(100dvh - 90px); overflow-y: auto; overscroll-behavior: contain; }
.nt-auth-entry:active { transform: scale(0.97); transition-duration: var(--nt-duration-press); }
.nt-menu-enter-active { transition: opacity 180ms var(--nt-ease-emphasized), transform 220ms var(--nt-ease-spring-soft); }
.nt-menu-leave-active { transition: opacity 110ms var(--nt-motion-ease), transform 110ms var(--nt-motion-ease); pointer-events: none; }
.nt-menu-enter-from, .nt-menu-leave-to { opacity: 0; transform: translateY(-6px) scale(0.96); }
@media (prefers-reduced-motion: reduce) {
  .nt-menu-enter-from, .nt-menu-leave-to { transform: none; }
  .nt-auth-entry:active { transform: none; }
}
</style>
