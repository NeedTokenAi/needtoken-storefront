import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '../stores/app'
import { resolveFallbackSiteName } from '../utils/storefrontStartup'

/**
 * 404 页共享逻辑（classic + vault 双模板共用）。
 * 完整保留原 views/NotFound.vue 的行为，仅抽离为 composable。
 */
export function useNotFound() {
  const router = useRouter()
  const appStore = useAppStore()
  const { t } = useI18n()

  const brandSiteName = computed(() => resolveFallbackSiteName(appStore.config?.brand?.site_name, {
    brandOverrideName: appStore.storefrontBrandOverride?.siteName,
    fallbackLabel: t('common.siteName'),
    hostname: typeof window === 'undefined' ? '' : window.location.hostname,
    isReseller: appStore.isResellerTenant,
  }))

  const goBack = () => {
    if (window.history.length > 1) {
      router.back()
      return
    }
    router.push('/')
  }

  return {
    brandSiteName,
    goBack,
  }
}
