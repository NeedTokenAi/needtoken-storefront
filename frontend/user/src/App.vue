<template>
  <!-- html keeps the scrollbar gutter; body locks must not reserve it twice. -->
  <ConfigProvider :scroll-body="{ padding: 0, margin: 0 }">
  <div id="app" class="min-h-screen bg-background text-foreground flex flex-col">
    <NeedtokenStartup v-if="startupPhase !== 'ready'" :branded="startupBranded" :failed="startupPhase === 'error'" :slow="startupSlow" @retry="retryPage" />
    <template v-else>
      <!-- vault 模板：自带顶栏/页脚的外壳包裹页面（控制台仍走下方分支） -->
      <VaultLayout v-if="activeTemplate === 'vault' && !isResellerConsole">
        <ErrorBoundary>
          <RouterView v-slot="{ Component }">
            <Transition name="page-fade" mode="out-in">
              <component :is="Component" />
            </Transition>
          </RouterView>
        </ErrorBoundary>
      </VaultLayout>

      <!-- needtoken 模板（needtoken.io 官网）：自带深色外壳 -->
      <NeedtokenLayout v-else-if="isNeedtoken && !isResellerConsole">
        <div v-if="navigationPending" class="nt-route-progress" role="status" :aria-label="t('needtoken.startup.routeLoading')"><span /><span class="sr-only">{{ t('needtoken.startup.routeLoading') }}</span></div>
        <div v-if="routeLoadFailed" class="nt-route-error" role="alert"><span>{{ t('needtoken.startup.routeFailed') }}</span><button type="button" @click="retryPage">{{ t('needtoken.startup.retry') }}</button></div>
        <ErrorBoundary>
          <RouterView v-slot="{ Component }">
            <div v-if="!Component" :class="{ 'nt-home-route-placeholder': route.path === '/' }">
              <NeedtokenSkeleton :kind="route.path === '/' ? 'prices' : 'page'" :count="route.path === '/' ? 4 : 3" />
            </div>
            <Transition name="nt-page" mode="out-in">
              <component :is="Component" v-if="Component" :key="route.path" />
            </Transition>
          </RouterView>
        </ErrorBoundary>
      </NeedtokenLayout>

      <!-- atelier 模板：Prism 精密科技外壳（悬浮导航 + 点阵画布 + 靛蓝令牌） -->
      <AtelierLayout v-else-if="activeTemplate === 'atelier' && !isResellerConsole">
        <ErrorBoundary>
          <RouterView v-slot="{ Component }">
            <Transition name="page-fade" mode="out-in">
              <component :is="Component" />
            </Transition>
          </RouterView>
        </ErrorBoundary>
      </AtelierLayout>

      <!-- classic 模板 / 分销控制台（保持原有结构不变） -->
      <template v-else>
        <Navbar v-if="!isResellerConsole" />
        <main class="flex-1" :class="isResellerConsole ? '' : 'pb-14 lg:pb-0'">
          <ErrorBoundary>
            <RouterView v-slot="{ Component }">
              <Transition name="page-fade" mode="out-in">
                <component :is="Component" />
              </Transition>
            </RouterView>
          </ErrorBoundary>
        </main>
        <Footer v-if="!isResellerConsole" />
        <BackToTop v-if="!isResellerConsole" />
        <MobileBottomNav v-if="!isResellerConsole" />
      </template>
    </template>

    <Loading v-if="!isNeedtoken && startupPhase === 'ready'" :loading="appStore.loading" />
    <Toast />
    <ConfirmDialog />
  </div>
  </ConfigProvider>
</template>

<script setup lang="ts">
import { computed, defineAsyncComponent, h, onUnmounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { ConfigProvider } from 'reka-ui'
import { useAppStore } from './stores/app'
import { getActiveTemplate, rememberTemplate } from './templates/registry'
import { shouldBrandNeedtokenStartup, storefrontStartupPhase } from './utils/storefrontStartup'
import { isChunkLoadError, lazyImportWithRetry, recoverFromChunkError } from './utils/chunkRecovery'
import NeedtokenStartup from './templates/needtoken/components/NeedtokenStartup.vue'
import NeedtokenSkeleton from './templates/needtoken/components/NeedtokenSkeleton.vue'
import Navbar from './components/Navbar.vue'
import Footer from './components/Footer.vue'
import Loading from './components/Loading.vue'
import Toast from './components/Toast.vue'
import ConfirmDialog from './components/ConfirmDialog.vue'
import ErrorBoundary from './components/ErrorBoundary.vue'
import BackToTop from './components/BackToTop.vue'
import MobileBottomNav from './components/MobileBottomNav.vue'

// 独立模板外壳按需加载，classic 用户不会拉取其 chunk/样式
const VaultLayout = defineAsyncComponent(lazyImportWithRetry(() => import('./templates/vault/layout/VaultLayout.vue')))
const AtelierLayout = defineAsyncComponent(lazyImportWithRetry(() => import('./templates/atelier/layout/AtelierLayout.vue')))
const NeedtokenLayout = defineAsyncComponent({
  loader: lazyImportWithRetry(() => import('./templates/needtoken/layout/NeedtokenLayout.vue')),
  loadingComponent: {
    setup() {
      const store = useAppStore()
      return () => h(NeedtokenStartup, { branded: !store.isResellerTenant })
    },
  },
  errorComponent: {
    setup() {
      const store = useAppStore()
      return () => h(NeedtokenStartup, { branded: !store.isResellerTenant, failed: true, onRetry: () => window.location.reload() })
    },
  },
  delay: 0,
  timeout: 20000,
})

// config 由 router.beforeEach 统一加载，无需在此重复调用
const appStore = useAppStore()
const route = useRoute()
const router = useRouter()
const { t } = useI18n()
const isResellerConsole = computed(() => route.meta.resellerConsole === true)
// getActiveTemplate 读取 appStore.config（响应式），config 加载后会重新计算
const activeTemplate = computed(() => getActiveTemplate())
const isNeedtoken = computed(() => activeTemplate.value === 'needtoken')

const initialRouteReady = ref(false)
const initialRouteFailed = ref(false)
const navigationPending = ref(false)
const routeLoadFailed = ref(false)
const failedDestination = ref('')
const startupSlow = ref(false)
const startupPhase = computed(() => storefrontStartupPhase({ hasConfig: !!appStore.config, routeReady: initialRouteReady.value, routeFailed: initialRouteFailed.value }))
let verifiedMainHint = false
try { verifiedMainHint = localStorage.getItem('dj-storefront-boot-main') === '1' } catch { /* Config determines the brand when storage is unavailable. */ }
const startupBranded = computed(() => appStore.config
  ? isNeedtoken.value && !appStore.isResellerTenant
  : shouldBrandNeedtokenStartup(getActiveTemplate(), verifiedMainHint, window.location.hostname))
const slowTimer = window.setTimeout(() => { if (startupPhase.value === 'pending') startupSlow.value = true }, 8000)
watch(startupPhase, phase => { if (phase !== 'pending') window.clearTimeout(slowTimer) })
watch(() => appStore.config, config => {
  if (!config) return
  try {
    if (config.storefront_template === 'needtoken' && !appStore.isResellerTenant) localStorage.setItem('dj-storefront-boot-main', '1')
    else localStorage.removeItem('dj-storefront-boot-main')
  } catch { /* Only a non-sensitive brand hint is stored. */ }
}, { immediate: true })
void router.isReady().then(() => { initialRouteReady.value = true }).catch(() => { initialRouteFailed.value = true })
const removeBeforeGuard = router.beforeEach(() => { navigationPending.value = true; routeLoadFailed.value = false })
const removeAfterGuard = router.afterEach(() => { navigationPending.value = false })
const removeErrorHandler = router.onError((error, destination) => {
  navigationPending.value = false
  failedDestination.value = destination.fullPath
  if (isChunkLoadError(error) && recoverFromChunkError(destination.fullPath)) {
    return
  }
  if (!initialRouteReady.value) initialRouteFailed.value = true
  else routeLoadFailed.value = true
})
function retryPage() {
  const destination = failedDestination.value
  if (destination.startsWith('/') && !destination.startsWith('//')) window.location.assign(destination)
  else window.location.reload()
}
onUnmounted(() => {
  window.clearTimeout(slowTimer)
  removeBeforeGuard()
  removeAfterGuard()
  removeErrorHandler()
})
watch(() => appStore.config?.storefront_template, (value) => rememberTemplate(value), { immediate: true })
</script>

<style>
.page-fade-enter-active,
.page-fade-leave-active {
  transition: opacity 200ms ease;
}

.page-fade-enter-from,
.page-fade-leave-to {
  opacity: 0;
}
.nt-page-enter-active {
  transition: opacity 220ms var(--nt-ease-spring-soft, cubic-bezier(0.22, 1.15, 0.36, 1)), transform 260ms var(--nt-ease-spring-soft, cubic-bezier(0.22, 1.15, 0.36, 1));
}
.nt-page-leave-active {
  transition: opacity 110ms ease-in, transform 110ms ease-in;
  pointer-events: none;
}
.nt-page-enter-from {
  opacity: 0;
  transform: translateY(6px);
}
.nt-page-leave-to {
  opacity: 0;
  transform: translateY(-3px);
}
.nt-route-progress { position: fixed; inset: 64px 0 auto; z-index: 45; height: 2px; overflow: hidden; pointer-events: none; background: color-mix(in srgb, var(--ui-accent, #2dd4a7) 10%, transparent); }
.nt-route-progress > span:first-child { display: block; width: 35%; height: 100%; background: var(--ui-accent, #2dd4a7); animation: nt-route-travel 1.1s ease-in-out infinite; }
.nt-route-error { display: flex; justify-content: center; align-items: center; gap: 12px; padding: 12px 20px; border-bottom: 1px solid var(--nt-warning-border, #e5b96940); color: var(--nt-warning-ink, #e5b969); font-size: 12px; line-height: 1.7; }
.nt-route-error button { flex-shrink: 0; border-bottom: 1px solid currentColor; cursor: pointer; }
.nt-home-route-placeholder { max-width: var(--nt-content-max-width, 1200px); min-height: 480px; margin: 0 auto; padding: 48px 24px; }
@media (max-width: 640px) { .nt-home-route-placeholder { padding: 32px 20px; } }
@keyframes nt-route-travel { from { transform: translateX(-105%); } to { transform: translateX(390%); } }
@media (prefers-reduced-motion: reduce) { .page-fade-enter-active, .page-fade-leave-active, .nt-page-enter-active, .nt-page-leave-active { transition: none; } .nt-page-enter-from, .nt-page-leave-to { opacity: 1; transform: none; } .nt-route-progress > span:first-child { animation: none; width: 100%; opacity: .6; } }
</style>
