<template>
  <div class="nt-scope">
    <header class="sticky top-0 z-50 border-b border-border bg-background/90 backdrop-blur">
      <div class="mx-auto flex h-16 w-full max-w-[1200px] items-center gap-2 px-4 sm:gap-4 sm:px-6">
        <RouterLink to="/" class="nt-brand flex shrink-0 items-center gap-2 text-[15px] font-bold tracking-tight text-foreground sm:text-[17px]" :title="brandName">
          <BrandMark v-if="!appStore.isResellerTenant" class="h-7 w-7" />
          <img v-else-if="resellerLogo" :src="resellerLogo" alt="" class="h-7 w-7 rounded-md object-contain" />
          <span>{{ brandName }}</span>
        </RouterLink>

        <nav class="ml-4 flex items-center gap-1 max-[1279px]:hidden" :aria-label="t('needtoken.nav.label')">
          <RouterLink
            v-for="item in menuItems"
            :key="item.key"
            :to="item.path"
            class="whitespace-nowrap rounded-md px-3 py-2 text-[14px] font-medium text-muted-foreground transition-colors hover:bg-secondary hover:text-foreground"
            :class="isActive(item) ? '!text-primary' : ''"
          >{{ item.label }}</RouterLink>
        </nav>

        <div class="ml-auto flex items-center gap-2">
          <RouterLink to="/guest/orders" class="grid h-9 w-9 place-items-center rounded-md text-muted-foreground hover:bg-secondary hover:text-foreground max-[419px]:hidden" :aria-label="t('navbar.guestOrders')" :title="t('navbar.guestOrders')">
            <ClipboardList class="h-[18px] w-[18px]" />
          </RouterLink>
          <RouterLink to="/cart" class="relative grid h-9 w-9 place-items-center rounded-md text-muted-foreground hover:bg-secondary hover:text-foreground" :aria-label="t('navbar.cart')">
            <ShoppingCart class="h-[18px] w-[18px]" />
            <span v-if="cartCount > 0" class="absolute -right-1 -top-1 grid h-[18px] min-w-[18px] place-items-center rounded-full bg-primary px-1 text-[11px] font-bold text-primary-foreground">{{ cartCount }}</span>
          </RouterLink>
          <div ref="langEl" class="relative max-[1279px]:hidden">
            <button type="button" class="grid h-9 w-9 place-items-center rounded-md text-muted-foreground hover:bg-secondary hover:text-foreground" :aria-label="t('navbar.selectLanguage')" @click="langOpen = !langOpen">
              <Languages class="h-[18px] w-[18px]" />
            </button>
            <div v-if="langOpen" class="absolute right-0 top-[calc(100%+8px)] z-[60] flex min-w-[150px] flex-col rounded-md border border-border bg-card p-1.5">
              <button v-for="lang in languages" :key="lang.code" type="button" class="rounded px-3 py-2 text-left text-sm hover:bg-secondary" :class="appStore.locale === lang.code ? 'text-primary' : 'text-muted-foreground'" @click="changeLanguage(lang.code)">{{ lang.name }}</button>
            </div>
          </div>
          <!-- 注册 / 登录统一进 new-api 控制台（跨域读不到登录态，固定显示） -->
          <a :href="CONSOLE_LOGIN_URL" class="rounded-md px-3 py-1.5 text-[13px] font-semibold text-foreground hover:bg-secondary max-[640px]:hidden" rel="noopener">{{ t('needtoken.nav.login') }}</a>
          <a :href="CONSOLE_REGISTER_URL" class="rounded-md bg-primary px-3 py-1.5 text-[13px] font-semibold text-primary-foreground hover:bg-primary/90" rel="noopener">{{ t('needtoken.nav.register') }}</a>

          <div ref="moreEl" class="relative hidden max-[1279px]:block">
            <button type="button" class="grid h-9 w-9 place-items-center rounded-md text-muted-foreground hover:bg-secondary" :aria-label="t('navbar.more')" :aria-expanded="moreOpen" @click="moreOpen = !moreOpen">
              <Menu v-if="!moreOpen" class="pointer-events-none h-[18px] w-[18px]" />
              <X v-else class="pointer-events-none h-[18px] w-[18px]" />
            </button>
            <div v-if="moreOpen" class="absolute right-0 top-[calc(100%+8px)] z-[60] flex min-w-[190px] flex-col rounded-md border border-border bg-card p-1.5">
              <RouterLink v-for="item in menuItems" :key="`m-${item.key}`" :to="item.path" class="rounded px-3 py-2 text-sm text-muted-foreground hover:bg-secondary hover:text-foreground" @click="moreOpen = false">{{ item.label }}</RouterLink>
              <RouterLink to="/guest/orders" class="rounded px-3 py-2 text-sm text-muted-foreground hover:bg-secondary hover:text-foreground" @click="moreOpen = false">{{ t('navbar.guestOrders') }}</RouterLink>
              <a :href="CONSOLE_LOGIN_URL" class="rounded px-3 py-2 text-sm text-muted-foreground hover:bg-secondary hover:text-foreground" rel="noopener">{{ t('needtoken.nav.login') }}</a>
              <span class="px-3 pb-1 pt-2 text-[11px] uppercase tracking-wider text-muted-foreground">{{ t('navbar.selectLanguage') }}</span>
              <button v-for="lang in languages" :key="`ml-${lang.code}`" type="button" class="rounded px-3 py-2 text-left text-sm hover:bg-secondary" :class="appStore.locale === lang.code ? 'text-primary' : 'text-muted-foreground'" @click="changeLanguage(lang.code); moreOpen = false">{{ lang.name }}</button>
            </div>
          </div>
        </div>
      </div>
    </header>

    <main class="flex-1">
      <slot />
    </main>

    <footer class="mt-20 border-t border-border">
      <div class="mx-auto grid w-full max-w-[1200px] grid-cols-2 gap-8 px-6 py-12 md:grid-cols-4">
        <div class="col-span-2 md:col-span-1">
          <div class="nt-brand flex items-center gap-2 text-[15px] font-bold text-foreground">
            <BrandMark v-if="!appStore.isResellerTenant" class="h-7 w-7" />
            <span>{{ brandName }}</span>
          </div>
          <p class="mt-3 max-w-[32ch] text-[13px] text-muted-foreground">{{ t('needtoken.footer.tagline') }}</p>
        </div>
        <div>
          <h4 class="mb-3 text-[13px] font-semibold text-foreground">{{ t('needtoken.footer.products') }}</h4>
          <RouterLink v-for="item in menuItems.slice(1, 5)" :key="`f-${item.key}`" :to="item.path" class="block py-1 text-[13px] text-muted-foreground hover:text-primary">{{ item.label }}</RouterLink>
        </div>
        <div>
          <h4 class="mb-3 text-[13px] font-semibold text-foreground">{{ t('needtoken.footer.developers') }}</h4>
          <RouterLink to="/docs" class="block py-1 text-[13px] text-muted-foreground hover:text-primary">{{ t('needtoken.nav.docs') }}</RouterLink>
          <a :href="CONSOLE_URL" class="block py-1 text-[13px] text-muted-foreground hover:text-primary" rel="noopener">{{ t('needtoken.footer.console') }}</a>
          <RouterLink to="/blog" class="block py-1 text-[13px] text-muted-foreground hover:text-primary">{{ t('needtoken.nav.blog') }}</RouterLink>
        </div>
        <div>
          <h4 class="mb-3 text-[13px] font-semibold text-foreground">{{ t('needtoken.footer.support') }}</h4>
          <RouterLink to="/guest/orders" class="block py-1 text-[13px] text-muted-foreground hover:text-primary">{{ t('navbar.guestOrders') }}</RouterLink>
          <RouterLink to="/terms" class="block py-1 text-[13px] text-muted-foreground hover:text-primary">{{ t('footer.terms') }}</RouterLink>
          <RouterLink to="/privacy" class="block py-1 text-[13px] text-muted-foreground hover:text-primary">{{ t('footer.privacy') }}</RouterLink>
          <a v-if="contact?.email" :href="`mailto:${contact.email}`" class="block py-1 text-[13px] text-muted-foreground hover:text-primary">{{ t('footer.email') }}</a>
        </div>
      </div>
      <div class="mx-auto flex w-full max-w-[1200px] flex-wrap justify-between gap-3 border-t border-border px-6 py-5 text-[12px] text-muted-foreground">
        <span>© {{ year }} {{ brandName }}</span>
      </div>
    </footer>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import { ClipboardList, Languages, Menu, ShoppingCart, X } from 'lucide-vue-next'
import { useAppStore } from '../../../stores/app'
import { useCartStore } from '../../../stores/cart'
import { getImageUrl } from '../../../utils/image'
import { CATEGORY_SLUGS, CONSOLE_LOGIN_URL, CONSOLE_REGISTER_URL, CONSOLE_URL, NEEDTOKEN_BRAND_NAME, NEEDTOKEN_BRAND_ICON, NEEDTOKEN_TOUCH_ICON } from '../config'
import BrandMark from '../components/BrandMark.vue'
import '../styles/needtoken.css'

const { t } = useI18n()
const route = useRoute()
const appStore = useAppStore()
const cartStore = useCartStore()

const langOpen = ref(false)
const moreOpen = ref(false)
const langEl = ref<HTMLElement | null>(null)
const moreEl = ref<HTMLElement | null>(null)
const year = new Date().getFullYear()

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
const contact = computed(() => appStore.config?.contact as { email?: string } | undefined)
const cartCount = computed(() => cartStore.totalItems)

interface MenuItem { key: string; path: string; label: string }

const menuItems = computed<MenuItem[]>(() => [
  { key: 'home', path: '/', label: t('needtoken.nav.home') },
  { key: 'recharge', path: `/categories/${CATEGORY_SLUGS.recharge}`, label: t('needtoken.nav.recharge') },
  { key: 'rental', path: `/categories/${CATEGORY_SLUGS.rental}`, label: t('needtoken.nav.rental') },
  { key: 'plans', path: '/plans', label: t('needtoken.nav.plans') },
  { key: 'models', path: '/models', label: t('needtoken.nav.models') },
  { key: 'docs', path: '/docs', label: t('needtoken.nav.docs') },
  { key: 'blog', path: '/blog', label: t('needtoken.nav.blog') },
])

const isActive = (item: MenuItem) => item.path === '/' ? route.path === '/' : route.path.startsWith(item.path)

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
}

// 深色是本模板的固定风格：挂载时打上 .dark，卸载（切回其他模板）时恢复原状，不写入用户主题偏好
let hadDark = false
onMounted(() => {
  document.addEventListener('click', onDocClick)
  hadDark = document.documentElement.classList.contains('dark')
  document.documentElement.classList.add('dark')
  document.body.classList.add('nt-tokens')
})
onUnmounted(() => {
  appStore.setStorefrontBrandOverride(previousBrandOverride)
  document.removeEventListener('click', onDocClick)
  if (!hadDark) document.documentElement.classList.remove('dark')
  document.body.classList.remove('nt-tokens')
})
</script>
