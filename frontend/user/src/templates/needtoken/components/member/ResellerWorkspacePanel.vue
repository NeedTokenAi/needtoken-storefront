<template>
  <div class="space-y-5">
    <!-- 统一 NeedToken 风格分销头卡 -->
    <div class="rounded-2xl border bg-card p-5 shadow-sm sm:p-6">
      <div class="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
        <div class="min-w-0">
          <div class="flex flex-wrap items-center gap-2">
            <span class="inline-flex h-8 w-8 items-center justify-center rounded-xl bg-primary/10 text-primary">
              <Store class="h-4 w-4" />
            </span>
            <h2 class="text-lg font-bold text-foreground">
              {{ t('needtoken.memberConsole.reseller.title') }}
            </h2>
            <Badge v-if="profileReady && statusBadgeLabel" :variant="statusBadgeVariant" size="sm">
              {{ statusBadgeLabel }}
            </Badge>
          </div>
          <p class="mt-1.5 text-xs leading-relaxed text-muted-foreground">
            {{ isActive ? t('needtoken.memberConsole.reseller.activeSubtitle') : t('needtoken.memberConsole.reseller.inactiveSubtitle') }}
          </p>
        </div>

        <div v-if="isActive && primaryDomainHref" class="flex shrink-0 items-center gap-2">
          <a
            :href="primaryDomainHref"
            target="_blank"
            rel="noopener noreferrer"
            class="inline-flex items-center gap-1.5 rounded-xl border bg-muted/40 px-3 py-2 text-xs font-semibold text-foreground transition-colors hover:border-primary/40 hover:text-primary"
          >
            <Globe class="h-3.5 w-3.5 text-primary" />
            <span class="max-w-[180px] truncate font-mono">{{ primaryDomainLabel }}</span>
            <ArrowUpRight class="h-3.5 w-3.5" />
          </a>
        </div>
      </div>

      <!-- 已开通分销：5 大聚合二级导航 -->
      <div v-if="profileReady && isActive" class="mt-5 flex flex-wrap gap-1.5 border-t pt-4">
        <button
          v-for="tab in mainTabs"
          :key="tab.key"
          type="button"
          class="inline-flex items-center gap-1.5 rounded-xl px-3.5 py-2 text-xs font-semibold transition-colors"
          :class="resolvedView.mainTab === tab.key
            ? 'bg-primary text-primary-foreground shadow-xs'
            : 'bg-muted/60 text-muted-foreground hover:bg-accent hover:text-foreground'"
          @click="switchMainTab(tab.key)"
        >
          <component :is="tab.icon" class="h-3.5 w-3.5" />
          <span>{{ t(tab.labelKey) }}</span>
        </button>
      </div>

      <!-- 财务与提现子切换条 -->
      <div v-if="profileReady && isActive && resolvedView.mainTab === 'finance'" class="mt-3 flex flex-wrap items-center gap-1.5">
        <button
          v-for="sub in financeSubTabs"
          :key="sub.key"
          type="button"
          class="rounded-lg border px-3 py-1.5 text-xs font-semibold transition-colors"
          :class="resolvedView.financeSubTab === sub.key
            ? 'border-primary/40 bg-primary/10 text-primary'
            : 'border-transparent bg-muted/40 text-muted-foreground hover:text-foreground'"
          @click="router.push(`/me/reseller/${sub.key}`)"
        >
          {{ t(sub.labelKey) }}
        </button>
      </div>

      <!-- 独立站与域名子切换条 -->
      <div v-if="profileReady && isActive && resolvedView.mainTab === 'site'" class="mt-3 flex flex-wrap items-center gap-1.5">
        <button
          v-for="sub in siteSubTabs"
          :key="sub.key"
          type="button"
          class="rounded-lg border px-3 py-1.5 text-xs font-semibold transition-colors"
          :class="resolvedView.siteSubTab === sub.key
            ? 'border-primary/40 bg-primary/10 text-primary'
            : 'border-transparent bg-muted/40 text-muted-foreground hover:text-foreground'"
          @click="router.push(`/me/reseller/${sub.key}`)"
        >
          {{ t(sub.labelKey) }}
        </button>
      </div>
    </div>

    <!-- 加载骨架 -->
    <div v-if="!profileReady || profileLoading" class="space-y-3">
      <div class="h-28 animate-pulse rounded-2xl border bg-card" />
      <div class="h-48 animate-pulse rounded-2xl border bg-card" />
    </div>

    <!-- 未开通 / 审核中 / 未激活：直接纳管入驻申请 -->
    <div v-else-if="!isActive" class="needtoken-reseller-shell">
      <ResellerApply />
    </div>

    <!-- 已激活：按二级 Tab 渲染重构纳管的模块 -->
    <div v-else class="needtoken-reseller-shell">
      <ResellerDashboard v-if="resolvedView.mainTab === 'dashboard'" />
      <ResellerProducts v-else-if="resolvedView.mainTab === 'products'" />
      <template v-else-if="resolvedView.mainTab === 'orders'">
        <ResellerOrderDetail v-if="resolvedView.isOrderDetail" />
        <ResellerOrders v-else />
      </template>
      <template v-else-if="resolvedView.mainTab === 'finance'">
        <ResellerLedger v-if="resolvedView.financeSubTab === 'ledger'" />
        <ResellerWithdraws v-else-if="resolvedView.financeSubTab === 'withdraws'" />
        <ResellerFinance v-else />
      </template>
      <template v-else-if="resolvedView.mainTab === 'site'">
        <ResellerSiteConfig v-if="resolvedView.siteSubTab === 'site'" />
        <ResellerApply v-else-if="resolvedView.siteSubTab === 'apply'" />
        <ResellerDomains v-else />
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, type Component } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import {
  ArrowUpRight,
  Banknote,
  Globe,
  LayoutGrid,
  ShoppingBag,
  Store,
  Tag,
} from 'lucide-vue-next'
import { Badge } from '@/components/ui/badge'
import { useResellerProfile } from '@/composables/reseller/useResellerProfile'
import ResellerApply from '@/views/reseller/ResellerApply.vue'
import ResellerDashboard from '@/views/reseller/ResellerDashboard.vue'
import ResellerDomains from '@/views/reseller/ResellerDomains.vue'
import ResellerFinance from '@/views/reseller/ResellerFinance.vue'
import ResellerLedger from '@/views/reseller/ResellerLedger.vue'
import ResellerOrderDetail from '@/views/reseller/ResellerOrderDetail.vue'
import ResellerOrders from '@/views/reseller/ResellerOrders.vue'
import ResellerProducts from '@/views/reseller/ResellerProducts.vue'
import ResellerSiteConfig from '@/views/reseller/ResellerSiteConfig.vue'
import ResellerWithdraws from '@/views/reseller/ResellerWithdraws.vue'
import {
  resolveResellerViewFromPath,
  type ResellerFinanceSubTab,
  type ResellerSiteSubTab,
  type ResellerWorkspaceMainTab,
} from '../../lib/gatewayConsoleHelpers'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()

const {
  loading: profileLoading,
  snapshot,
  state: profileState,
  load: loadProfile,
} = useResellerProfile()
const profileReady = ref(false)

const isActive = computed(() => profileState.value.profileStatus === 'active')
const resolvedView = computed(() => resolveResellerViewFromPath(route.path))

const statusBadgeLabel = computed(() => {
  const status = profileState.value.profileStatus
  if (status === 'active') return t('needtoken.memberConsole.reseller.statusActive')
  if (status === 'pending_review') return t('needtoken.memberConsole.reseller.statusPending')
  if (status === 'rejected') return t('needtoken.memberConsole.reseller.statusRejected')
  if (status === 'disabled') return t('needtoken.memberConsole.reseller.statusDisabled')
  return ''
})

const statusBadgeVariant = computed(() => {
  const status = profileState.value.profileStatus
  if (status === 'active') return 'success'
  if (status === 'pending_review') return 'warning'
  if (status === 'rejected' || status === 'disabled') return 'destructive'
  return 'secondary'
})

const primaryDomainLabel = computed(() => {
  const domains = snapshot.value?.domains ?? []
  const primary = domains.find((d) => d.is_primary) || domains.find((d) => d.status === 'active') || domains[0]
  const raw = String(primary?.domain || '').trim()
  return raw.replace(/^https?:\/\//i, '').replace(/\/+$/, '')
})

const primaryDomainHref = computed(() => {
  if (!primaryDomainLabel.value) return ''
  return `https://${primaryDomainLabel.value}`
})

const mainTabs: Array<{ key: ResellerWorkspaceMainTab; labelKey: string; icon: Component }> = [
  { key: 'dashboard', labelKey: 'needtoken.memberConsole.reseller.tabDashboard', icon: LayoutGrid },
  { key: 'products', labelKey: 'needtoken.memberConsole.reseller.tabProducts', icon: Tag },
  { key: 'orders', labelKey: 'needtoken.memberConsole.reseller.tabOrders', icon: ShoppingBag },
  { key: 'finance', labelKey: 'needtoken.memberConsole.reseller.tabFinance', icon: Banknote },
  { key: 'site', labelKey: 'needtoken.memberConsole.reseller.tabSite', icon: Globe },
]

const financeSubTabs: Array<{ key: ResellerFinanceSubTab; labelKey: string }> = [
  { key: 'finance', labelKey: 'resellerConsole.nav.finance' },
  { key: 'ledger', labelKey: 'resellerConsole.nav.ledger' },
  { key: 'withdraws', labelKey: 'resellerConsole.nav.withdraws' },
]

const siteSubTabs: Array<{ key: ResellerSiteSubTab; labelKey: string }> = [
  { key: 'domains', labelKey: 'resellerConsole.nav.domains' },
  { key: 'site', labelKey: 'resellerConsole.nav.site' },
  { key: 'apply', labelKey: 'needtoken.memberConsole.reseller.subTabProfile' },
]

const switchMainTab = (tab: ResellerWorkspaceMainTab) => {
  if (tab === 'dashboard') {
    void router.push('/me/reseller')
    return
  }
  if (tab === 'site') {
    void router.push('/me/reseller/domains')
    return
  }
  void router.push(`/me/reseller/${tab}`)
}

onMounted(async () => {
  await loadProfile()
  profileReady.value = true
})
</script>

<style scoped>
.needtoken-reseller-shell :deep(.rounded-xl) {
  border-radius: 1rem;
}
</style>
