<template>
  <div class="relative min-h-screen bg-background pb-16 pt-20 text-foreground sm:pt-24">
    <div class="mx-auto w-full max-w-[1240px] px-4 sm:px-6">
      <!-- 顶部一体化会员控制台头 -->
      <header class="mb-6 overflow-hidden rounded-3xl border bg-card shadow-sm">
        <div class="flex flex-col gap-5 p-5 sm:p-6 lg:flex-row lg:items-center lg:justify-between">
          <div class="flex min-w-0 items-center gap-4">
            <div class="flex h-14 w-14 shrink-0 items-center justify-center rounded-2xl bg-primary/10 text-xl font-black text-primary sm:h-16 sm:w-16 sm:text-2xl">
              {{ displayInitial }}
            </div>
            <div class="min-w-0">
              <div class="flex flex-wrap items-center gap-2">
                <p class="text-[11px] font-semibold uppercase tracking-[0.2em] text-primary">
                  {{ t('needtoken.memberConsole.eyebrow') }}
                </p>
                <span
                  v-if="showGatewayFeatures && displayNativeUserId"
                  class="rounded-full border border-primary/25 bg-primary/10 px-2 py-0.5 font-mono text-[11px] font-semibold text-primary"
                >
                  API #{{ displayNativeUserId }}
                </span>
                <span
                  v-if="showGatewayFeatures && apiSummary?.group"
                  class="rounded-full border bg-muted/60 px-2 py-0.5 font-mono text-[11px] font-medium text-muted-foreground"
                >
                  {{ apiSummary.group }}
                </span>
              </div>
              <h1 class="mt-1 truncate text-xl font-black text-foreground sm:text-2xl">
                {{ userProfileStore.displayName }}
              </h1>
              <div class="mt-1 flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
                <span class="truncate">{{ userProfileStore.profile?.email || t('personalCenter.subtitle') }}</span>
                <Badge :variant="emailVerifiedVariant" size="sm">{{ emailVerifiedLabel }}</Badge>
                <span
                  v-if="userProfileStore.currentLevel"
                  class="inline-flex items-center gap-1 rounded-full border border-primary/25 bg-primary/10 px-2.5 py-0.5 text-[11px] font-semibold text-primary"
                >
                  <img
                    v-if="isImagePath(userProfileStore.currentLevel?.icon)"
                    :src="getImageUrl(userProfileStore.currentLevel!.icon)"
                    class="h-3 w-3 object-contain"
                    alt=""
                  />
                  <Crown v-else class="h-3 w-3" />
                  {{ levelName(userProfileStore.currentLevel) }}
                </span>
              </div>
            </div>
          </div>

          <div v-if="showGatewayFeatures" class="flex flex-col gap-2 sm:flex-row sm:items-center">
            <div class="flex items-center gap-2 rounded-xl border bg-muted/40 px-3 py-2 text-xs">
              <span class="shrink-0 font-semibold text-muted-foreground">API</span>
              <code class="max-w-[220px] truncate font-mono font-semibold text-foreground sm:max-w-[260px]">
                {{ API_BASE_URL }}
              </code>
              <button
                type="button"
                class="inline-flex shrink-0 items-center gap-1 rounded-md border bg-background px-2 py-1 text-[11px] font-semibold text-foreground transition-colors hover:border-primary/40 hover:text-primary"
                @click="copyEndpoint"
              >
                <Check v-if="copiedEndpoint" class="h-3 w-3 text-primary" />
                <Copy v-else class="h-3 w-3" />
                <span>{{ copiedEndpoint ? t('needtoken.common.copied') : t('needtoken.common.copy') }}</span>
              </button>
            </div>
          </div>
        </div>
      </header>

      <div class="grid grid-cols-1 gap-6 lg:grid-cols-12">
        <!-- 左侧精简导航（桌面粘性竖栏 + 移动横向切换条） -->
        <aside class="lg:col-span-3">
          <div class="rounded-2xl border bg-card p-3.5 shadow-sm lg:sticky lg:top-24">
            <!-- 桌面端分组导航 -->
            <nav class="hidden flex-col gap-4 lg:flex" :aria-label="t('personalCenter.title')">
              <div v-for="group in navGroups" :key="group.key" class="space-y-1">
                <p class="px-3 text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">
                  {{ t(group.labelKey) }}
                </p>
                <button
                  v-for="item in group.items"
                  :key="item.key"
                  type="button"
                  class="group relative flex w-full items-center gap-2.5 rounded-xl py-2.5 pl-4 pr-3 text-left text-sm font-semibold transition-colors"
                  :class="activeSection === item.key
                    ? 'bg-primary/10 text-primary'
                    : 'text-muted-foreground hover:bg-accent hover:text-foreground'"
                  @click="navigateToSection(item.key)"
                >
                  <span
                    class="absolute left-0 top-1/2 h-5 w-1 -translate-y-1/2 rounded-r-full transition-all"
                    :class="activeSection === item.key ? 'bg-primary' : 'bg-transparent'"
                  />
                  <component :is="item.icon" class="h-4 w-4 shrink-0" />
                  <span class="truncate">{{ t(item.labelKey) }}</span>
                </button>
              </div>

              <div class="border-t pt-3">
                <button
                  type="button"
                  class="flex w-full items-center gap-2.5 rounded-xl px-4 py-2.5 text-left text-sm font-semibold text-red-600 transition-colors hover:bg-red-500/10 dark:text-red-400"
                  @click="handleLogout"
                >
                  <LogOut class="h-4 w-4 shrink-0" />
                  <span>{{ t('needtoken.accountMenu.signOut') }}</span>
                </button>
              </div>
            </nav>

            <!-- 移动端横向切换栏 -->
            <div class="lg:hidden">
              <div class="flex gap-1.5 overflow-x-auto pb-1">
                <button
                  v-for="item in flatNavItems"
                  :key="item.key"
                  type="button"
                  class="shrink-0 rounded-xl px-3.5 py-2 text-xs font-semibold transition-colors"
                  :class="activeSection === item.key
                    ? 'bg-primary/10 text-primary'
                    : 'text-muted-foreground hover:bg-accent hover:text-foreground'"
                  @click="navigateToSection(item.key)"
                >
                  <span class="flex items-center gap-1.5">
                    <component :is="item.icon" class="h-3.5 w-3.5" />
                    <span>{{ t(item.labelKey) }}</span>
                  </span>
                </button>
                <button
                  type="button"
                  class="shrink-0 rounded-xl px-3.5 py-2 text-xs font-semibold text-red-600 hover:bg-red-500/10 dark:text-red-400"
                  @click="handleLogout"
                >
                  <span class="flex items-center gap-1.5">
                    <LogOut class="h-3.5 w-3.5" />
                    <span>{{ t('needtoken.accountMenu.signOut') }}</span>
                  </span>
                </button>
              </div>
            </div>
          </div>
        </aside>

        <!-- 右侧核心内容区 -->
        <section class="min-w-0 space-y-6 lg:col-span-9">
          <Alert
            v-if="globalAlert"
            :variant="pageAlertVariant(globalAlert.level)"
            :class="pageAlertToneClass(globalAlert.level)"
          >
            <AlertDescription>{{ globalAlert.message }}</AlertDescription>
          </Alert>

          <!-- 1. 控制台总览 -->
          <template v-if="activeSection === 'overview'">
            <!-- 仅在存在未完成的旧账号手动绑定或待激活订单时提示 -->
            <ApiAccountLinkBanner v-if="showGatewayFeatures" next-path="/me" only-when-action-needed />

            <!-- 四格一体化资产与网关聚合卡 -->
            <div class="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
              <button
                v-if="showGatewayFeatures"
                type="button"
                class="rounded-2xl border bg-card p-5 text-left shadow-sm transition-all hover:border-primary/40 hover:shadow-md"
                @click="navigateToSection('assets')"
              >
                <div class="flex items-center justify-between text-xs font-medium text-muted-foreground">
                  <span>{{ t('needtoken.memberConsole.overview.apiTotalQuota') }}</span>
                  <Cpu class="h-4 w-4 text-primary" />
                </div>
                <p class="mt-2 font-mono text-2xl font-black text-primary">
                  {{ gatewayLoading && !apiSummary ? '—' : formatApiQuotaUsd(combinedApiQuota) }}
                </p>
                <p class="mt-1.5 text-[11.5px] text-muted-foreground">
                  {{
                    t('needtoken.memberConsole.overview.apiQuotaBreakdown', {
                      balance: formatApiQuotaUsd(apiSummary?.quota ?? 0),
                      plan: formatApiQuotaUsd(subscriptionRemainingQuota),
                    })
                  }}
                </p>
              </button>

              <button
                type="button"
                class="rounded-2xl border bg-card p-5 text-left shadow-sm transition-all hover:border-primary/40 hover:shadow-md"
                @click="navigateToSection('assets')"
              >
                <div class="flex items-center justify-between text-xs font-medium text-muted-foreground">
                  <span>{{ t('needtoken.memberConsole.overview.storeWallet') }}</span>
                  <Wallet class="h-4 w-4 text-primary" />
                </div>
                <p class="mt-2 font-mono text-2xl font-black text-foreground">
                  {{ userProfileStore.loadingWallet ? '—' : storeWalletDisplay }}
                </p>
                <p class="mt-1.5 text-[11.5px] text-muted-foreground">
                  {{ t('needtoken.memberConsole.overview.storeWalletHint') }}
                </p>
              </button>

              <button
                type="button"
                class="rounded-2xl border bg-card p-5 text-left shadow-sm transition-all hover:border-primary/40 hover:shadow-md"
                @click="navigateToSection('affiliate')"
              >
                <div class="flex items-center justify-between text-xs font-medium text-muted-foreground">
                  <span>{{ t('needtoken.memberConsole.overview.affiliateCommission') }}</span>
                  <Megaphone class="h-4 w-4 text-primary" />
                </div>
                <p class="mt-2 font-mono text-2xl font-black text-foreground">
                  {{ affiliateLoading ? '—' : affiliateAvailableDisplay }}
                </p>
                <p class="mt-1.5 text-[11.5px] text-muted-foreground">
                  {{ affiliateHintText }}
                </p>
              </button>

              <button
                v-if="showGatewayFeatures"
                type="button"
                class="rounded-2xl border bg-card p-5 text-left shadow-sm transition-all hover:border-primary/40 hover:shadow-md"
                @click="navigateToSection('keys')"
              >
                <div class="flex items-center justify-between text-xs font-medium text-muted-foreground">
                  <span>{{ t('needtoken.memberConsole.overview.keysAndRequests') }}</span>
                  <KeyRound class="h-4 w-4 text-primary" />
                </div>
                <p class="mt-2 font-mono text-2xl font-black text-foreground">
                  {{ activeKeyCount === null ? '—' : activeKeyCount }}
                  <span class="ml-1 text-xs font-semibold text-muted-foreground">
                    {{ t('needtoken.memberConsole.overview.activeKeysUnit') }}
                  </span>
                </p>
                <p class="mt-1.5 text-[11.5px] text-muted-foreground">
                  {{
                    t('needtoken.memberConsole.overview.totalRequestsHint', {
                      count: formatInteger(apiSummary?.request_count ?? 0),
                    })
                  }}
                </p>
              </button>

              <button
                v-else
                type="button"
                class="rounded-2xl border bg-card p-5 text-left shadow-sm transition-all hover:border-primary/40 hover:shadow-md"
                @click="navigateToSection('orders')"
              >
                <div class="flex items-center justify-between text-xs font-medium text-muted-foreground">
                  <span>{{ t('personalCenter.tabs.orders') }}</span>
                  <ShoppingBag class="h-4 w-4 text-primary" />
                </div>
                <p class="mt-2 font-mono text-2xl font-black text-foreground">
                  {{ userProfileStore.loadingOrders ? '—' : userProfileStore.ordersTotal }}
                </p>
                <p class="mt-1.5 text-[11.5px] text-muted-foreground">
                  {{ t('personalCenter.overview.viewAllOrders') }}
                </p>
              </button>
            </div>

            <!-- 快捷操作条 -->
            <div class="flex flex-wrap items-center gap-2.5 rounded-2xl border bg-card p-4 shadow-sm">
              <Button v-if="showGatewayFeatures" size="sm" @click="navigateToSection('keys')">
                <KeyRound class="mr-1.5 h-4 w-4" />
                {{ t('needtoken.memberConsole.quick.manageKeys') }}
              </Button>
              <Button v-if="showGatewayFeatures" as-child variant="outline" size="sm">
                <RouterLink to="/plans">
                  <Sparkles class="mr-1.5 h-4 w-4 text-primary" />
                  {{ t('needtoken.memberConsole.quick.buyPlans') }}
                </RouterLink>
              </Button>
              <Button v-if="showGatewayFeatures" variant="outline" size="sm" @click="navigateToSection('usage')">
                <Activity class="mr-1.5 h-4 w-4" />
                {{ t('needtoken.memberConsole.quick.viewUsage') }}
              </Button>
              <Button variant="outline" size="sm" @click="openAssetsSubTab('giftCard')">
                <Gift class="mr-1.5 h-4 w-4" />
                {{ t('needtoken.memberConsole.quick.redeemGiftCard') }}
              </Button>
              <Button v-if="canAccessResellerConsole" as-child variant="outline" size="sm">
                <RouterLink to="/reseller">
                  <Banknote class="mr-1.5 h-4 w-4" />
                  {{ t('resellerConsole.title') }}
                </RouterLink>
              </Button>
            </div>

            <!-- 生效中的 Need 网关套餐 -->
            <div v-if="showGatewayFeatures" class="rounded-2xl border bg-card p-6 shadow-sm">
              <div class="mb-4 flex flex-wrap items-center justify-between gap-2">
                <div>
                  <h2 class="text-base font-bold text-foreground">{{ t('needtoken.memberConsole.plans.title') }}</h2>
                  <p class="mt-0.5 text-xs text-muted-foreground">{{ t('needtoken.memberConsole.plans.subtitle') }}</p>
                </div>
                <Button as-child variant="outline" size="sm">
                  <RouterLink to="/plans">
                    {{ t('needtoken.memberConsole.plans.browseMore') }}
                    <ArrowUpRight class="ml-1 h-3.5 w-3.5" />
                  </RouterLink>
                </Button>
              </div>

              <div v-if="gatewayLoading && !apiSummary" class="space-y-3">
                <div v-for="idx in 2" :key="idx" class="h-20 animate-pulse rounded-xl bg-muted" />
              </div>

              <div v-else-if="activeSubscriptions.length === 0" class="flex flex-col items-start justify-between gap-3 rounded-xl border border-dashed p-5 sm:flex-row sm:items-center">
                <div>
                  <p class="text-sm font-semibold text-foreground">{{ t('needtoken.memberConsole.plans.emptyTitle') }}</p>
                  <p class="mt-1 text-xs text-muted-foreground">{{ t('needtoken.memberConsole.plans.emptyHint') }}</p>
                </div>
                <Button as-child size="sm" class="shrink-0">
                  <RouterLink to="/plans">{{ t('needtoken.memberConsole.quick.buyPlans') }}</RouterLink>
                </Button>
              </div>

              <div v-else class="grid grid-cols-1 gap-3.5 md:grid-cols-2">
                <div
                  v-for="sub in activeSubscriptions"
                  :key="sub.id"
                  class="rounded-xl border bg-muted/20 p-4"
                >
                  <div class="flex items-start justify-between gap-2">
                    <div class="min-w-0">
                      <p class="truncate text-sm font-bold text-foreground">
                        {{ sub.plan_title || t('needtoken.apiAccountLink.subscriptionFallbackName', { id: sub.plan_id }) }}
                      </p>
                      <div v-if="splitGroups(sub.allowed_groups).length" class="mt-1.5 flex flex-wrap gap-1">
                        <span
                          v-for="grp in splitGroups(sub.allowed_groups)"
                          :key="grp"
                          class="rounded-md border bg-background px-1.5 py-0.5 font-mono text-[10.5px] font-medium text-muted-foreground"
                        >
                          {{ grp }}
                        </span>
                      </div>
                    </div>
                    <span class="shrink-0 text-[11px] text-muted-foreground">
                      {{ subscriptionExpiryText(sub.end_time) }}
                    </span>
                  </div>

                  <div class="mt-3 h-2 w-full overflow-hidden rounded-full bg-muted">
                    <div
                      class="h-full rounded-full bg-primary transition-all"
                      :style="{ width: `${computeSubscriptionUsagePercent(sub.amount_total, sub.amount_used)}%` }"
                    />
                  </div>

                  <div class="mt-2 flex flex-wrap items-center justify-between gap-2 text-xs">
                    <span class="font-mono font-semibold text-foreground">
                      {{
                        t('needtoken.apiAccountLink.subscriptionRemaining', {
                          remaining: formatApiQuotaUsd(computeSubscriptionRemainingQuota(sub.amount_total, sub.amount_used)),
                          total: formatApiQuotaUsd(sub.amount_total),
                        })
                      }}
                    </span>
                    <span v-if="sub.allow_wallet_overflow" class="rounded-full bg-primary/10 px-2 py-0.5 text-[10.5px] font-semibold text-primary">
                      {{ t('needtoken.apiAccountLink.walletOverflowEnabled') }}
                    </span>
                  </div>
                </div>
              </div>
            </div>

            <!-- 最近订单 -->
            <div class="rounded-2xl border bg-card p-6 shadow-sm">
              <div class="mb-4 flex flex-wrap items-center justify-between gap-2">
                <h2 class="text-base font-bold text-foreground">{{ t('personalCenter.overview.recentOrdersTitle') }}</h2>
                <Button variant="ghost" size="sm" class="rounded-full" @click="navigateToSection('orders')">
                  {{ t('personalCenter.overview.viewAllOrders') }}
                </Button>
              </div>

              <div v-if="userProfileStore.loadingOrders" class="space-y-3">
                <div v-for="idx in 3" :key="idx" class="h-16 animate-pulse rounded-xl border bg-muted" />
              </div>

              <div v-else-if="userProfileStore.recentOrders.length === 0" class="rounded-xl border border-dashed px-4 py-5 text-sm text-muted-foreground">
                {{ t('personalCenter.overview.emptyOrders') }}
              </div>

              <div v-else class="space-y-3">
                <div
                  v-for="order in userProfileStore.recentOrders"
                  :key="order.order_no"
                  class="rounded-xl border bg-card px-4 py-3 transition-all hover:border-primary/30 hover:shadow-sm"
                >
                  <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                    <div>
                      <div class="font-mono text-xs text-muted-foreground">{{ t('orders.orderNo') }}：{{ order.order_no }}</div>
                      <div class="mt-1 text-sm font-semibold text-foreground">
                        {{ formatMoney(order.total_amount, order.currency) }}
                      </div>
                      <div class="mt-1 text-xs text-muted-foreground">{{ formatDate(order.created_at) }}</div>
                    </div>
                    <div class="flex flex-wrap items-center gap-2">
                      <Badge :variant="statusVariant(order.status)" size="sm">
                        {{ statusLabel(order.status) }}
                      </Badge>
                      <Button as-child variant="outline" size="sm">
                        <RouterLink :to="`/orders/${order.order_no}`">{{ t('orders.viewDetails') }}</RouterLink>
                      </Button>
                      <Button v-if="order.status === 'pending_payment'" as-child size="sm">
                        <RouterLink :to="`/pay?order_no=${order.order_no}`">{{ t('orders.payNow') }}</RouterLink>
                      </Button>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </template>

          <!-- 2. API 密钥管理 -->
          <GatewayKeysPanel v-else-if="activeSection === 'keys' && showGatewayFeatures" @changed="refreshGatewayData" />

          <!-- 3. 用量与调用日志 -->
          <GatewayUsagePanel v-else-if="activeSection === 'usage' && showGatewayFeatures" />

          <!-- 4. 我的订单 -->
          <OrdersPanel v-else-if="activeSection === 'orders'" />

          <!-- 5. 资产与礼品卡（集合展示 API 额度 + 店面钱包 + 推广佣金 + 礼品卡兑换 + 钱包流水） -->
          <template v-else-if="activeSection === 'assets'">
            <div class="rounded-2xl border bg-card p-6 shadow-sm">
              <div class="mb-4 flex flex-wrap items-center justify-between gap-2">
                <div>
                  <h2 class="text-lg font-bold text-foreground">{{ t('needtoken.memberConsole.assets.title') }}</h2>
                  <p class="mt-1 text-xs text-muted-foreground">{{ t('needtoken.memberConsole.assets.subtitle') }}</p>
                </div>
                <Button v-if="showGatewayFeatures" as-child size="sm">
                  <RouterLink to="/plans">{{ t('needtoken.memberConsole.quick.buyPlans') }}</RouterLink>
                </Button>
              </div>

              <div class="grid grid-cols-1 gap-3.5 md:grid-cols-3">
                <div v-if="showGatewayFeatures" class="rounded-xl border border-primary/25 bg-primary/5 p-4">
                  <p class="text-xs font-medium text-muted-foreground">{{ t('needtoken.memberConsole.overview.apiTotalQuota') }}</p>
                  <p class="mt-1.5 font-mono text-xl font-black text-primary">
                    {{ formatApiQuotaUsd(combinedApiQuota) }}
                  </p>
                  <p class="mt-1 text-[11.5px] text-muted-foreground">
                    {{
                      t('needtoken.memberConsole.overview.apiQuotaBreakdown', {
                        balance: formatApiQuotaUsd(apiSummary?.quota ?? 0),
                        plan: formatApiQuotaUsd(subscriptionRemainingQuota),
                      })
                    }}
                  </p>
                </div>

                <div class="rounded-xl border bg-muted/20 p-4">
                  <p class="text-xs font-medium text-muted-foreground">{{ t('needtoken.memberConsole.overview.storeWallet') }}</p>
                  <p class="mt-1.5 font-mono text-xl font-black text-foreground">
                    {{ userProfileStore.loadingWallet ? '—' : storeWalletDisplay }}
                  </p>
                  <p class="mt-1 text-[11.5px] text-muted-foreground">
                    {{ t('needtoken.memberConsole.assets.walletUsageNote') }}
                  </p>
                </div>

                <div class="rounded-xl border bg-muted/20 p-4">
                  <div class="flex items-center justify-between gap-2">
                    <p class="text-xs font-medium text-muted-foreground">{{ t('needtoken.memberConsole.overview.affiliateCommission') }}</p>
                    <button
                      type="button"
                      class="text-[11px] font-semibold text-primary hover:underline"
                      @click="navigateToSection('affiliate')"
                    >
                      {{ t('needtoken.memberConsole.assets.goWithdraw') }}
                    </button>
                  </div>
                  <p class="mt-1.5 font-mono text-xl font-black text-foreground">
                    {{ affiliateLoading ? '—' : affiliateAvailableDisplay }}
                  </p>
                  <p class="mt-1 text-[11.5px] text-muted-foreground">{{ affiliateHintText }}</p>
                </div>
              </div>

              <!-- 礼品卡兑换 / 钱包充值与流水 轻量切换 -->
              <div class="mt-5 flex items-center gap-2 border-t pt-4">
                <button
                  type="button"
                  class="inline-flex items-center gap-1.5 rounded-xl px-4 py-2 text-xs font-semibold transition-colors"
                  :class="assetsSubTab === 'giftCard' ? 'bg-primary text-primary-foreground' : 'bg-muted/60 text-muted-foreground hover:text-foreground'"
                  @click="assetsSubTab = 'giftCard'"
                >
                  <Gift class="h-3.5 w-3.5" />
                  <span>{{ t('needtoken.memberConsole.assets.tabGiftCard') }}</span>
                </button>
                <button
                  type="button"
                  class="inline-flex items-center gap-1.5 rounded-xl px-4 py-2 text-xs font-semibold transition-colors"
                  :class="assetsSubTab === 'wallet' ? 'bg-primary text-primary-foreground' : 'bg-muted/60 text-muted-foreground hover:text-foreground'"
                  @click="assetsSubTab = 'wallet'"
                >
                  <Wallet class="h-3.5 w-3.5" />
                  <span>{{ t('needtoken.memberConsole.assets.tabWallet') }}</span>
                </button>
              </div>
            </div>

            <GiftCardPanel v-if="assetsSubTab === 'giftCard'" />
            <WalletPanel v-else />
          </template>

          <!-- 6. 推广与分销 -->
          <template v-else-if="activeSection === 'affiliate'">
            <div v-if="canAccessResellerConsole" class="flex flex-col justify-between gap-4 rounded-2xl border border-primary/25 bg-primary/5 p-6 shadow-sm sm:flex-row sm:items-center">
              <div>
                <div class="flex items-center gap-2">
                  <Banknote class="h-5 w-5 text-primary" />
                  <h2 class="text-base font-bold text-foreground">{{ t('resellerConsole.title') }}</h2>
                </div>
                <p class="mt-1 text-xs text-muted-foreground">{{ t('resellerConsole.dashboard.description') }}</p>
              </div>
              <Button as-child size="sm" class="shrink-0">
                <RouterLink to="/reseller">
                  {{ t('resellerConsole.nav.dashboard') }}
                  <ArrowUpRight class="ml-1 h-3.5 w-3.5" />
                </RouterLink>
              </Button>
            </div>
            <AffiliatePanel />
          </template>

          <!-- 7. 账号与安全（合并资料 + 安全设置） -->
          <template v-else-if="activeSection === 'security'">
            <ProfilePanel />
            <SecurityPanel />
          </template>
        </section>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch, type Component } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import {
  Activity,
  ArrowUpRight,
  Banknote,
  Check,
  Copy,
  Cpu,
  Crown,
  Gift,
  KeyRound,
  LayoutDashboard,
  LogOut,
  Megaphone,
  ShieldCheck,
  ShoppingBag,
  Sparkles,
  Wallet,
} from 'lucide-vue-next'
import { affiliateAPI, type AffiliateDashboardData } from '../../api'
import { useAppStore } from '../../stores/app'
import { useUserAuthStore } from '../../stores/userAuth'
import { usePersonalCenter, type PersonalSection } from '../../composables/usePersonalCenter'
import { getImageUrl } from '../../utils/image'
import { pageAlertToneClass, pageAlertVariant } from '../../utils/alerts'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import ProfilePanel from '../../views/personal/ProfilePanel.vue'
import SecurityPanel from '../../views/personal/SecurityPanel.vue'
import OrdersPanel from '../../views/personal/OrdersPanel.vue'
import WalletPanel from '../../views/personal/WalletPanel.vue'
import GiftCardPanel from '../../views/personal/GiftCardPanel.vue'
import AffiliatePanel from '../../views/personal/AffiliatePanel.vue'
import ApiAccountLinkBanner from './components/ApiAccountLinkBanner.vue'
import GatewayKeysPanel from './components/member/GatewayKeysPanel.vue'
import GatewayUsagePanel from './components/member/GatewayUsagePanel.vue'
import { API_BASE_URL } from './config'
import { useApiAccountLink } from './composables/useApiAccountLink'
import {
  computeSubscriptionRemainingQuota,
  computeSubscriptionUsagePercent,
  formatApiQuotaUsd,
  formatSubscriptionExpiry,
} from './lib/apiAccountLinkPolicy'
import {
  computeCombinedApiQuota,
  normalizeNeedtokenMemberSection,
  sumSubscriptionRemainingQuota,
  type NeedtokenMemberSection,
} from './lib/gatewayConsoleHelpers'
import { listGatewayTokens } from './lib/gatewayConsoleApi'

const props = withDefaults(defineProps<{ section?: PersonalSection }>(), {
  section: 'overview',
})

const { t, locale } = useI18n()
const router = useRouter()
const appStore = useAppStore()
const userAuth = useUserAuthStore()

const {
  userProfileStore,
  canAccessResellerConsole,
  globalAlert,
  displayInitial,
  statusLabel,
  statusVariant,
  formatMoney,
  formatDate,
  emailVerifiedLabel,
  emailVerifiedVariant,
  isImagePath,
  levelName,
} = usePersonalCenter(() => props.section)

const {
  status: apiStatus,
  summary: apiSummary,
  load: loadApiStatus,
  ensure: ensureApiLink,
  loadSummary: loadApiSummary,
} = useApiAccountLink()

const showGatewayFeatures = computed(() => !appStore.isResellerTenant)

const activeSection = computed<NeedtokenMemberSection>(() => {
  const normalized = normalizeNeedtokenMemberSection(props.section)
  if (!showGatewayFeatures.value && (normalized === 'keys' || normalized === 'usage')) {
    return 'overview'
  }
  return normalized
})

const assetsSubTab = ref<'giftCard' | 'wallet'>(props.section === 'wallet' ? 'wallet' : 'giftCard')
watch(
  () => props.section,
  (next) => {
    if (next === 'wallet') assetsSubTab.value = 'wallet'
    else if (next === 'giftCard') assetsSubTab.value = 'giftCard'
  },
)

interface NavItem {
  key: NeedtokenMemberSection
  labelKey: string
  icon: Component
}
interface NavGroup {
  key: string
  labelKey: string
  items: NavItem[]
}

const navGroups = computed<NavGroup[]>(() => {
  const groups: NavGroup[] = []
  if (showGatewayFeatures.value) {
    groups.push({
      key: 'gateway',
      labelKey: 'needtoken.memberConsole.nav.groupGateway',
      items: [
        { key: 'overview', labelKey: 'needtoken.memberConsole.nav.overview', icon: LayoutDashboard },
        { key: 'keys', labelKey: 'needtoken.memberConsole.nav.keys', icon: KeyRound },
        { key: 'usage', labelKey: 'needtoken.memberConsole.nav.usage', icon: Activity },
      ],
    })
  } else {
    groups.push({
      key: 'main',
      labelKey: 'needtoken.memberConsole.nav.groupAccount',
      items: [
        { key: 'overview', labelKey: 'needtoken.memberConsole.nav.overview', icon: LayoutDashboard },
      ],
    })
  }
  groups.push({
    key: 'assets',
    labelKey: 'needtoken.memberConsole.nav.groupAssets',
    items: [
      { key: 'orders', labelKey: 'needtoken.memberConsole.nav.orders', icon: ShoppingBag },
      { key: 'assets', labelKey: 'needtoken.memberConsole.nav.assets', icon: Wallet },
      { key: 'affiliate', labelKey: 'needtoken.memberConsole.nav.affiliate', icon: Megaphone },
    ],
  })
  groups.push({
    key: 'account',
    labelKey: 'needtoken.memberConsole.nav.groupAccount',
    items: [
      { key: 'security', labelKey: 'needtoken.memberConsole.nav.security', icon: ShieldCheck },
    ],
  })
  return groups
})

const flatNavItems = computed(() => navGroups.value.flatMap(group => group.items))

const sectionRoutes: Record<NeedtokenMemberSection, string> = {
  overview: '/me',
  keys: '/me/keys',
  usage: '/me/usage',
  orders: '/me/orders',
  assets: '/me/assets',
  affiliate: '/me/affiliate',
  security: '/me/security',
}

const navigateToSection = (section: NeedtokenMemberSection) => {
  void router.push(sectionRoutes[section])
}

const openAssetsSubTab = (tab: 'giftCard' | 'wallet') => {
  assetsSubTab.value = tab
  navigateToSection('assets')
}

const gatewayLoading = ref(false)
const activeKeyCount = ref<number | null>(null)
const affiliateLoading = ref(false)
const affiliateDashboard = ref<AffiliateDashboardData | null>(null)

const displayNativeUserId = computed(() => apiStatus.value?.native_user_id || apiSummary.value?.native_user_id || 0)
const activeSubscriptions = computed(() => apiSummary.value?.active_subscriptions ?? [])
const subscriptionRemainingQuota = computed(() => sumSubscriptionRemainingQuota(activeSubscriptions.value))
const combinedApiQuota = computed(() => computeCombinedApiQuota(apiSummary.value?.quota, activeSubscriptions.value))

const storeWalletDisplay = computed(() => {
  const bal = userProfileStore.walletBalance || '0.00'
  const cur = userProfileStore.walletAccount?.currency || appStore.config?.currency || ''
  return cur ? `${bal} ${cur}` : bal
})

const affiliateAvailableDisplay = computed(() => {
  const raw = affiliateDashboard.value?.available_commission
  const cur = appStore.config?.currency || ''
  if (!raw) return cur ? `0.00 ${cur}` : '0.00'
  return cur ? `${raw} ${cur}` : raw
})

const affiliateHintText = computed(() => {
  if (!affiliateDashboard.value?.opened) {
    return t('needtoken.memberConsole.overview.affiliateNotOpened')
  }
  return t('needtoken.memberConsole.overview.affiliatePendingHint', {
    pending: affiliateDashboard.value.pending_commission || '0.00',
  })
})

const splitGroups = (raw?: string | null) => {
  if (!raw) return []
  return raw.split(',').map(item => item.trim()).filter(Boolean)
}

const formatInteger = (value?: number | null) => {
  const n = typeof value === 'number' && Number.isFinite(value) ? Math.max(0, Math.trunc(value)) : 0
  return n.toLocaleString('en-US')
}

const subscriptionExpiryText = (endTime?: number | null) => {
  const formatted = formatSubscriptionExpiry(endTime, String(locale.value || 'zh-CN'))
  if (!formatted) return t('needtoken.apiAccountLink.subscriptionPermanent')
  return t('needtoken.apiAccountLink.subscriptionValidUntil', { date: formatted })
}

const refreshGatewayData = async () => {
  if (!showGatewayFeatures.value || !userAuth.isAuthenticated) return
  gatewayLoading.value = true
  try {
    const status = await loadApiStatus(true)
    if (status?.enabled && !status.linked) {
      await ensureApiLink()
    }
    await loadApiSummary(true)
    try {
      const tokenRes = await listGatewayTokens()
      const items = Array.isArray(tokenRes.items) ? tokenRes.items : []
      activeKeyCount.value = items.filter(tok => tok.status === 1).length
    } catch {
      activeKeyCount.value = null
    }
  } finally {
    gatewayLoading.value = false
  }
}

const loadAffiliateOverview = async () => {
  if (!userAuth.isAuthenticated) return
  affiliateLoading.value = true
  try {
    const res = await affiliateAPI.dashboard()
    affiliateDashboard.value = (res.data?.data || null) as AffiliateDashboardData | null
  } catch {
    affiliateDashboard.value = null
  } finally {
    affiliateLoading.value = false
  }
}

const copiedEndpoint = ref(false)
let copyTimer: ReturnType<typeof setTimeout> | undefined
const copyEndpoint = async () => {
  if (copyTimer) clearTimeout(copyTimer)
  try {
    await navigator.clipboard.writeText(API_BASE_URL)
    copiedEndpoint.value = true
    copyTimer = setTimeout(() => { copiedEndpoint.value = false }, 1800)
  } catch {
    copiedEndpoint.value = false
  }
}

const handleLogout = async () => {
  await userAuth.logout('/')
}

onMounted(() => {
  void Promise.all([refreshGatewayData(), loadAffiliateOverview()])
})

onUnmounted(() => {
  if (copyTimer) clearTimeout(copyTimer)
})
</script>
