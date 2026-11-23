<template>
  <div class="relative min-h-screen bg-background pb-16 pt-20 text-foreground sm:pt-24">
    <div class="mx-auto w-full max-w-[1240px] px-4 sm:px-6">
      <div class="grid grid-cols-1 gap-6 lg:grid-cols-12">
        <!-- 左侧精简导航（桌面粘性竖栏 + 移动横向切换条） -->
        <aside class="lg:col-span-3">
          <div class="rounded-2xl border bg-card p-3.5 shadow-sm lg:sticky lg:top-24">
            <!-- 收敛后的紧凑个人资料卡（悬停样式反馈，点击直达账号与安全 /me/security） -->
            <button
              type="button"
              data-testid="sidebar-profile-card"
              class="group mb-3.5 flex w-full items-center gap-3 rounded-xl border p-3 text-left transition-all duration-150"
              :class="activeSection === 'security'
                ? 'border-primary/45 bg-primary/10 shadow-xs'
                : 'border-border/70 bg-muted/25 hover:border-primary/40 hover:bg-primary/5 hover:shadow-xs'"
              :title="t('needtoken.memberConsole.nav.profileCardHint')"
              @click="navigateToSection('security')"
            >
              <div
                class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl text-base font-black transition-colors"
                :class="activeSection === 'security'
                  ? 'bg-primary text-primary-foreground'
                  : 'bg-primary/10 text-primary group-hover:bg-primary group-hover:text-primary-foreground'"
              >
                {{ displayInitial }}
              </div>
              <div class="min-w-0 flex-1">
                <div class="flex items-center gap-1.5">
                  <span class="truncate text-sm font-bold text-foreground transition-colors group-hover:text-primary">
                    {{ userProfileStore.displayName }}
                  </span>
                  <span
                    v-if="showGatewayFeatures && displayNativeUserId"
                    class="shrink-0 rounded-full border border-primary/25 bg-primary/10 px-1.5 py-0.5 font-mono text-[10px] font-semibold leading-none text-primary"
                  >
                    #{{ displayNativeUserId }}
                  </span>
                </div>
                <p class="mt-0.5 truncate text-[11.5px] text-muted-foreground">
                  {{ userProfileStore.profile?.email || t('personalCenter.subtitle') }}
                </p>
                <div class="mt-1 flex flex-wrap items-center gap-1.5">
                  <Badge :variant="emailVerifiedVariant" size="sm" class="h-4 px-1.5 text-[10px]">
                    {{ emailVerifiedLabel }}
                  </Badge>
                  <span
                    v-if="userProfileStore.currentLevel"
                    class="inline-flex items-center gap-0.5 rounded-full border border-primary/25 bg-primary/10 px-1.5 py-0.5 text-[10px] font-semibold leading-none text-primary"
                  >
                    <img
                      v-if="isImagePath(userProfileStore.currentLevel?.icon)"
                      :src="getImageUrl(userProfileStore.currentLevel!.icon)"
                      class="h-2.5 w-2.5 object-contain"
                      alt=""
                    />
                    <Crown v-else class="h-2.5 w-2.5" />
                    {{ levelName(userProfileStore.currentLevel) }}
                  </span>
                </div>
              </div>
              <ChevronRight
                class="h-4 w-4 shrink-0 transition-transform duration-150"
                :class="activeSection === 'security'
                  ? 'translate-x-0.5 text-primary'
                  : 'text-muted-foreground/70 group-hover:translate-x-0.5 group-hover:text-primary'"
              />
            </button>

            <!-- 桌面端分组导航 -->
            <nav class="hidden flex-col gap-4 lg:flex" :aria-label="t('personalCenter.title')">
              <div v-for="group in navGroups" :key="group.key" class="space-y-1">
                <p class="px-3 text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">
                  {{ t(group.labelKey) }}
                </p>
                <template v-for="item in group.items" :key="item.key">
                  <!-- 可展开/收缩的二级菜单组（推广与分销） -->
                  <div v-if="item.children?.length" class="space-y-1">
                    <button
                      type="button"
                      data-testid="nav-group-growth-toggle"
                      class="group relative flex w-full items-center justify-between gap-2 rounded-xl py-2.5 pl-4 pr-3 text-left text-sm font-semibold transition-colors"
                      :class="isGrowthSectionActive
                        ? 'bg-primary/10 text-primary'
                        : 'text-muted-foreground hover:bg-accent hover:text-foreground'"
                      :aria-expanded="growthExpanded"
                      @click="growthExpanded = !growthExpanded"
                    >
                      <span
                        class="absolute left-0 top-1/2 h-5 w-1 -translate-y-1/2 rounded-r-full transition-all"
                        :class="isGrowthSectionActive ? 'bg-primary' : 'bg-transparent'"
                      />
                      <span class="flex min-w-0 items-center gap-2.5">
                        <component :is="item.icon" class="h-4 w-4 shrink-0" />
                        <span class="truncate">{{ t(item.labelKey) }}</span>
                      </span>
                      <ChevronDown
                        class="h-4 w-4 shrink-0 transition-transform duration-150"
                        :class="growthExpanded ? 'rotate-180' : ''"
                      />
                    </button>
                    <div v-if="growthExpanded" class="space-y-1 pl-4">
                      <button
                        v-for="child in item.children"
                        :key="child.key"
                        type="button"
                        :data-testid="`nav-item-${child.key}`"
                        class="group relative flex w-full items-center gap-2.5 rounded-xl py-2 pl-4 pr-3 text-left text-[13px] font-semibold transition-colors"
                        :class="activeSection === child.key
                          ? 'bg-primary/10 text-primary'
                          : 'text-muted-foreground hover:bg-accent hover:text-foreground'"
                        @click="navigateToSection(child.key)"
                      >
                        <span
                          class="h-1.5 w-1.5 shrink-0 rounded-full transition-colors"
                          :class="activeSection === child.key ? 'bg-primary' : 'bg-muted-foreground/40 group-hover:bg-foreground/60'"
                        />
                        <component :is="child.icon" class="h-3.5 w-3.5 shrink-0" />
                        <span class="truncate">{{ t(child.labelKey) }}</span>
                      </button>
                    </div>
                  </div>

                  <!-- 普通一级菜单项 -->
                  <button
                    v-else
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
                </template>
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

            <!-- 快捷操作与 API 接口条 -->
            <div class="flex flex-col gap-3 rounded-2xl border bg-card p-4 shadow-sm xl:flex-row xl:items-center xl:justify-between">
              <div class="flex flex-wrap items-center gap-2">
                <Button v-if="showGatewayFeatures" size="sm" @click="navigateToSection('keys')">
                  <KeyRound class="mr-1.5 h-4 w-4" />
                  {{ t('needtoken.memberConsole.quick.manageKeys') }}
                </Button>
                <Button v-if="showGatewayFeatures" variant="outline" size="sm" @click="navigateToSection('playground')">
                  <FlaskConical class="mr-1.5 h-4 w-4 text-primary" />
                  {{ t('needtoken.memberConsole.nav.playground') }}
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
                <Button v-if="canAccessResellerConsole" variant="outline" size="sm" @click="navigateToSection('reseller')">
                  <Store class="mr-1.5 h-4 w-4" />
                  {{ t('needtoken.memberConsole.nav.resellerSystem') }}
                </Button>
              </div>

              <div v-if="showGatewayFeatures" class="flex items-center gap-2 rounded-xl border bg-muted/40 px-3 py-1.5 text-xs">
                <span class="shrink-0 font-semibold text-muted-foreground">API</span>
                <code class="max-w-[200px] truncate font-mono font-semibold text-foreground sm:max-w-[240px]">
                  {{ API_BASE_URL }}
                </code>
                <button
                  type="button"
                  class="inline-flex shrink-0 items-center gap-1 rounded-md border bg-background px-2 py-0.5 text-[11px] font-semibold text-foreground transition-colors hover:border-primary/40 hover:text-primary"
                  @click="copyEndpoint"
                >
                  <Check v-if="copiedEndpoint" class="h-3 w-3 text-primary" />
                  <Copy v-else class="h-3 w-3" />
                  <span>{{ copiedEndpoint ? t('needtoken.common.copied') : t('needtoken.common.copy') }}</span>
                </button>
              </div>
            </div>

            <!-- 近 7 天网关调用趋势 -->
            <div v-if="showGatewayFeatures" class="rounded-2xl border bg-card p-6 shadow-sm">
              <div class="mb-4 flex flex-wrap items-center justify-between gap-2">
                <div>
                  <h2 class="text-base font-bold text-foreground">{{ t('needtoken.memberConsole.overview.trendTitle') }}</h2>
                  <p class="mt-0.5 text-xs text-muted-foreground">
                    {{
                      t('needtoken.memberConsole.overview.trendSubtitle', {
                        spend: formatApiQuotaUsd(overviewUsage?.total_quota ?? 0),
                        requests: formatInteger(overviewUsage?.total_requests ?? 0),
                      })
                    }}
                  </p>
                </div>
                <Button variant="ghost" size="sm" class="rounded-full" @click="navigateToSection('usage')">
                  {{ t('needtoken.memberConsole.overview.viewUsageDetails') }}
                </Button>
              </div>

              <div v-if="gatewayLoading && !overviewUsage" class="h-24 animate-pulse rounded-xl bg-muted" />
              <div v-else-if="!overviewUsage?.daily?.length" class="rounded-xl border border-dashed px-4 py-5 text-xs text-muted-foreground">
                {{ t('needtoken.memberConsole.overview.trendEmpty') }}
              </div>
              <div v-else class="flex h-24 items-end gap-2">
                <div
                  v-for="d in overviewUsage.daily"
                  :key="d.date"
                  class="group flex flex-1 flex-col items-center gap-1"
                  :title="`${d.date} · ${formatApiQuotaUsd(d.quota)} · ${formatInteger(d.requests)} ${t('needtoken.memberConsole.usage.reqUnit')}`"
                >
                  <div class="flex h-16 w-full items-end justify-center rounded-md bg-muted/30 px-1 pt-1">
                    <div
                      class="w-full max-w-[28px] rounded-t-md bg-primary/80 transition-all group-hover:bg-primary"
                      :style="{ height: `${overviewBarPercent(d.quota)}%` }"
                    />
                  </div>
                  <span class="font-mono text-[10px] text-muted-foreground">{{ d.date.slice(5) }}</span>
                </div>
              </div>
            </div>

            <!-- 生效中的 Need 网关套餐 -->
            <div v-if="showGatewayFeatures" class="rounded-2xl border bg-card p-6 shadow-sm">
              <div class="mb-4 flex flex-wrap items-center justify-between gap-2">
                <div>
                  <h2 class="text-base font-bold text-foreground">{{ t('needtoken.memberConsole.plans.title') }}</h2>
                  <p class="mt-0.5 text-xs text-muted-foreground">{{ t('needtoken.memberConsole.plans.subtitle') }}</p>
                </div>
                <div class="flex items-center gap-2">
                  <Button variant="ghost" size="sm" class="rounded-full" @click="navigateToSection('plans')">
                    {{ t('needtoken.memberConsole.plans.viewAllPlans') }}
                  </Button>
                  <Button as-child variant="outline" size="sm">
                    <RouterLink to="/plans">
                      {{ t('needtoken.memberConsole.plans.browseMore') }}
                      <ArrowUpRight class="ml-1 h-3.5 w-3.5" />
                    </RouterLink>
                  </Button>
                </div>
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

          <!-- 3. 模型与计费（模型目录 + 分组实时价格 + 调用分析） -->
          <GatewayModelsPanel v-else-if="activeSection === 'models' && showGatewayFeatures" />

          <!-- 4. 在线调试（Playground） -->
          <GatewayPlaygroundPanel v-else-if="activeSection === 'playground' && showGatewayFeatures" />

          <!-- 5. 用量与调用日志（趋势图 + 调用日志 / 异步任务 / 绘图任务） -->
          <GatewayUsagePanel v-else-if="activeSection === 'usage' && showGatewayFeatures" />

          <!-- 6. 我的套餐（生效中 + 历史订阅全量明细） -->
          <GatewayPlansPanel v-else-if="activeSection === 'plans' && showGatewayFeatures" />

          <!-- 7. 我的订单 -->
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

          <!-- 6. 推广拉新 -->
          <AffiliatePanel v-else-if="activeSection === 'affiliate'" />

          <!-- 7. 分销系统（全面纳管重构，统一 NeedToken 视觉与路由） -->
          <ResellerWorkspacePanel v-else-if="activeSection === 'reseller' && canAccessResellerConsole" />

          <!-- 8. 账号与安全（点击左侧顶部资料卡进入） -->
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
  Box,
  Check,
  ChevronDown,
  ChevronRight,
  Copy,
  Cpu,
  CreditCard,
  Crown,
  FlaskConical,
  Gift,
  KeyRound,
  LayoutDashboard,
  LogOut,
  Megaphone,
  ShoppingBag,
  Sparkles,
  Store,
  UserPlus,
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
import GatewayModelsPanel from './components/member/GatewayModelsPanel.vue'
import GatewayPlansPanel from './components/member/GatewayPlansPanel.vue'
import GatewayPlaygroundPanel from './components/member/GatewayPlaygroundPanel.vue'
import GatewayUsagePanel from './components/member/GatewayUsagePanel.vue'
import ResellerWorkspacePanel from './components/member/ResellerWorkspacePanel.vue'
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
import { getGatewayUsage, listGatewayTokens, type GatewayUsageResponse } from './lib/gatewayConsoleApi'

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

const GATEWAY_ONLY_SECTIONS = new Set<NeedtokenMemberSection>(['keys', 'models', 'playground', 'usage', 'plans'])

const activeSection = computed<NeedtokenMemberSection>(() => {
  const normalized = normalizeNeedtokenMemberSection(props.section)
  if (!showGatewayFeatures.value && GATEWAY_ONLY_SECTIONS.has(normalized)) {
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

interface NavLeafItem {
  key: NeedtokenMemberSection
  labelKey: string
  icon: Component
}
interface NavItem {
  key: string
  labelKey: string
  icon: Component
  children?: NavLeafItem[]
}
interface NavGroup {
  key: string
  labelKey: string
  items: NavItem[]
}

const isGrowthSectionActive = computed(() => activeSection.value === 'affiliate' || activeSection.value === 'reseller')
const growthExpanded = ref(isGrowthSectionActive.value)
watch(isGrowthSectionActive, (active) => {
  if (active) growthExpanded.value = true
})

const navGroups = computed<NavGroup[]>(() => {
  const groups: NavGroup[] = []
  if (showGatewayFeatures.value) {
    groups.push({
      key: 'gateway',
      labelKey: 'needtoken.memberConsole.nav.groupGateway',
      items: [
        { key: 'overview', labelKey: 'needtoken.memberConsole.nav.overview', icon: LayoutDashboard },
        { key: 'keys', labelKey: 'needtoken.memberConsole.nav.keys', icon: KeyRound },
        { key: 'models', labelKey: 'needtoken.memberConsole.nav.models', icon: Box },
        { key: 'playground', labelKey: 'needtoken.memberConsole.nav.playground', icon: FlaskConical },
        { key: 'usage', labelKey: 'needtoken.memberConsole.nav.usage', icon: Activity },
        { key: 'plans', labelKey: 'needtoken.memberConsole.nav.plans', icon: CreditCard },
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
  const growthNavItem: NavItem = canAccessResellerConsole.value
    ? {
        key: 'growth',
        labelKey: 'needtoken.memberConsole.nav.affiliate',
        icon: Megaphone,
        children: [
          { key: 'affiliate', labelKey: 'needtoken.memberConsole.nav.affiliateReferral', icon: UserPlus },
          { key: 'reseller', labelKey: 'needtoken.memberConsole.nav.resellerSystem', icon: Store },
        ],
      }
    : {
        key: 'affiliate',
        labelKey: 'needtoken.memberConsole.nav.affiliateReferral',
        icon: UserPlus,
      }
  groups.push({
    key: 'assets',
    labelKey: 'needtoken.memberConsole.nav.groupAssets',
    items: [
      { key: 'orders', labelKey: 'needtoken.memberConsole.nav.orders', icon: ShoppingBag },
      { key: 'assets', labelKey: 'needtoken.memberConsole.nav.assets', icon: Wallet },
      growthNavItem,
    ],
  })
  return groups
})

const flatNavItems = computed<NavLeafItem[]>(() =>
  navGroups.value.flatMap(group =>
    group.items.flatMap(item =>
      item.children?.length
        ? item.children
        : [{ key: item.key as NeedtokenMemberSection, labelKey: item.labelKey, icon: item.icon }],
    ),
  ),
)

const sectionRoutes: Record<NeedtokenMemberSection, string> = {
  overview: '/me',
  keys: '/me/keys',
  models: '/me/models',
  playground: '/me/playground',
  usage: '/me/usage',
  plans: '/me/plans',
  orders: '/me/orders',
  assets: '/me/assets',
  affiliate: '/me/affiliate',
  reseller: '/me/reseller',
  security: '/me/security',
}

const navigateToSection = (section: string) => {
  const key = section as NeedtokenMemberSection
  if (sectionRoutes[key]) {
    void router.push(sectionRoutes[key])
  }
}

const openAssetsSubTab = (tab: 'giftCard' | 'wallet') => {
  assetsSubTab.value = tab
  navigateToSection('assets')
}

const gatewayLoading = ref(false)
const activeKeyCount = ref<number | null>(null)
const overviewUsage = ref<GatewayUsageResponse | null>(null)
const affiliateLoading = ref(false)
const affiliateDashboard = ref<AffiliateDashboardData | null>(null)

const overviewBarPercent = (quota: number) => {
  const daily = overviewUsage.value?.daily || []
  const maxQuota = Math.max(0, ...daily.map(item => item.quota))
  if (maxQuota <= 0 || quota <= 0) return 6
  return Math.max(10, Math.round((quota / maxQuota) * 100))
}

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
    const [tokenRes, usageRes] = await Promise.all([
      listGatewayTokens().catch(() => null),
      getGatewayUsage(7).catch(() => null),
    ])
    if (tokenRes) {
      const items = Array.isArray(tokenRes.items) ? tokenRes.items : []
      activeKeyCount.value = items.filter(tok => tok.status === 1).length
    } else {
      activeKeyCount.value = null
    }
    if (usageRes) {
      overviewUsage.value = usageRes
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
