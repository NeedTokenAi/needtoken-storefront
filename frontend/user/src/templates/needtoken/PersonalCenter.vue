<template>
  <div class="flex min-h-dvh flex-col bg-background text-foreground lg:h-dvh lg:flex-row lg:overflow-hidden">
    <!-- 桌面端左侧固定可折叠侧边栏 -->
    <aside
      class="hidden shrink-0 flex-col border-r border-border bg-card/85 backdrop-blur transition-[width] duration-200 ease-out lg:flex"
      :class="sidebarCollapsed ? 'w-[68px]' : 'w-[232px]'"
    >
      <!-- 顶部品牌与折叠按钮 -->
      <div
        class="flex h-14 shrink-0 items-center border-b border-border"
        :class="sidebarCollapsed ? 'justify-center px-2' : 'justify-between gap-2 px-3.5'"
      >
        <RouterLink
          v-if="!sidebarCollapsed"
          to="/me"
          class="nt-brand flex min-w-0 items-center gap-2 text-[15px] font-bold tracking-tight text-foreground"
        >
          <BrandMark v-if="!appStore.isResellerTenant" class="h-6 w-6 shrink-0" />
          <img v-else-if="resellerLogo" :src="resellerLogo" alt="" class="h-6 w-6 shrink-0 rounded-md object-contain" />
          <span class="truncate">{{ brandName }}</span>
          <span class="shrink-0 rounded-md border border-primary/25 bg-primary/10 px-1.5 py-0.5 text-[10px] font-semibold leading-none text-primary">
            {{ t('needtoken.memberConsole.nav.consoleBadge') }}
          </span>
        </RouterLink>

        <button
          type="button"
          class="grid h-8 w-8 shrink-0 place-items-center rounded-lg text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
          :aria-label="sidebarCollapsed ? t('needtoken.memberConsole.nav.expandSidebar') : t('needtoken.memberConsole.nav.collapseSidebar')"
          :title="sidebarCollapsed ? t('needtoken.memberConsole.nav.expandSidebar') : t('needtoken.memberConsole.nav.collapseSidebar')"
          @click="toggleSidebarCollapse"
        >
          <PanelLeftOpen v-if="sidebarCollapsed" class="h-4 w-4" />
          <PanelLeftClose v-else class="h-4 w-4" />
        </button>
      </div>

      <!-- 中部导航菜单 -->
      <nav
        class="flex-1 space-y-4 overflow-y-auto px-2.5 py-3.5"
        :aria-label="t('personalCenter.title')"
      >
        <div v-for="group in navGroups" :key="group.key" class="space-y-0.5">
          <p
            v-if="!sidebarCollapsed"
            class="px-2.5 pb-1 text-[10.5px] font-semibold uppercase tracking-wider text-muted-foreground/80"
          >
            {{ t(group.labelKey) }}
          </p>
          <template v-for="item in group.items" :key="item.key">
            <!-- 可展开/收缩的二级菜单组（推广与分销） -->
            <div v-if="item.children?.length" class="space-y-0.5">
              <button
                type="button"
                data-testid="nav-group-growth-toggle"
                class="group relative flex w-full items-center rounded-xl py-2 text-left text-[13.5px] font-semibold transition-colors"
                :class="[
                  sidebarCollapsed ? 'justify-center px-0' : 'justify-between gap-2 pl-3.5 pr-2.5',
                  isGrowthSectionActive
                    ? 'bg-primary/10 text-primary'
                    : 'text-muted-foreground hover:bg-accent hover:text-foreground',
                ]"
                :aria-expanded="!sidebarCollapsed && growthExpanded"
                :title="sidebarCollapsed ? t(item.labelKey) : undefined"
                :aria-label="sidebarCollapsed ? t(item.labelKey) : undefined"
                @click="handleGrowthGroupClick"
              >
                <span
                  class="absolute left-0 top-1/2 h-4 w-1 -translate-y-1/2 rounded-r-full transition-all"
                  :class="isGrowthSectionActive ? 'bg-primary' : 'bg-transparent'"
                />
                <span class="flex min-w-0 items-center gap-2.5">
                  <component :is="item.icon" class="h-4 w-4 shrink-0" />
                  <span v-if="!sidebarCollapsed" class="truncate">{{ t(item.labelKey) }}</span>
                </span>
                <ChevronDown
                  v-if="!sidebarCollapsed"
                  class="h-3.5 w-3.5 shrink-0 transition-transform duration-150"
                  :class="growthExpanded ? 'rotate-180' : ''"
                />
              </button>
              <div v-if="!sidebarCollapsed && growthExpanded" class="space-y-0.5 pl-3.5">
                <button
                  v-for="child in item.children"
                  :key="child.key"
                  type="button"
                  :data-testid="`nav-item-${child.key}`"
                  class="group relative flex w-full items-center gap-2 rounded-xl py-1.5 pl-3.5 pr-2.5 text-left text-[13px] font-medium transition-colors"
                  :class="activeSection === child.key
                    ? 'bg-primary/10 font-semibold text-primary'
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
              class="group relative flex w-full items-center rounded-xl py-2 text-left text-[13.5px] font-medium transition-colors"
              :class="[
                sidebarCollapsed ? 'justify-center px-0' : 'gap-2.5 pl-3.5 pr-2.5',
                activeSection === item.key
                  ? 'bg-primary/10 font-semibold text-primary'
                  : 'text-muted-foreground hover:bg-accent hover:text-foreground',
              ]"
              :title="sidebarCollapsed ? t(item.labelKey) : undefined"
              :aria-label="sidebarCollapsed ? t(item.labelKey) : undefined"
              @click="navigateToSection(item.key)"
            >
              <span
                class="absolute left-0 top-1/2 h-4 w-1 -translate-y-1/2 rounded-r-full transition-all"
                :class="activeSection === item.key ? 'bg-primary' : 'bg-transparent'"
              />
              <component :is="item.icon" class="h-4 w-4 shrink-0" />
              <span v-if="!sidebarCollapsed" class="truncate">{{ t(item.labelKey) }}</span>
            </button>
          </template>
        </div>
      </nav>
    </aside>

    <!-- 移动端抽屉导航 -->
    <Teleport to="body">
      <Transition name="nt-drawer-fade">
        <div
          v-if="mobileDrawerOpen"
          class="fixed inset-0 z-50 flex lg:hidden"
          role="dialog"
          aria-modal="true"
          :aria-label="t('personalCenter.title')"
        >
          <div
            class="fixed inset-0 bg-black/45 backdrop-blur-[2px]"
            aria-hidden="true"
            @click="mobileDrawerOpen = false"
          />
          <aside class="relative z-10 flex h-full w-[268px] max-w-[82vw] flex-col border-r border-border bg-card shadow-xl">
            <div class="flex h-14 shrink-0 items-center justify-between border-b border-border px-4">
              <RouterLink to="/me" class="nt-brand flex min-w-0 items-center gap-2 text-[15px] font-bold text-foreground" @click="mobileDrawerOpen = false">
                <BrandMark v-if="!appStore.isResellerTenant" class="h-6 w-6 shrink-0" />
                <img v-else-if="resellerLogo" :src="resellerLogo" alt="" class="h-6 w-6 shrink-0 rounded-md object-contain" />
                <span class="truncate">{{ brandName }}</span>
                <span class="shrink-0 rounded-md border border-primary/25 bg-primary/10 px-1.5 py-0.5 text-[10px] font-semibold leading-none text-primary">
                  {{ t('needtoken.memberConsole.nav.consoleBadge') }}
                </span>
              </RouterLink>
              <button
                type="button"
                class="grid h-8 w-8 place-items-center rounded-lg text-muted-foreground hover:bg-accent hover:text-foreground"
                :aria-label="t('needtoken.memberConsole.nav.closeMenu')"
                @click="mobileDrawerOpen = false"
              >
                <X class="h-4 w-4" />
              </button>
            </div>

            <nav class="flex-1 space-y-4 overflow-y-auto px-3 py-4" :aria-label="t('personalCenter.title')">
              <div v-for="group in navGroups" :key="`m-${group.key}`" class="space-y-1">
                <p class="px-2.5 text-[10.5px] font-semibold uppercase tracking-wider text-muted-foreground/80">
                  {{ t(group.labelKey) }}
                </p>
                <template v-for="item in group.items" :key="`m-${item.key}`">
                  <div v-if="item.children?.length" class="space-y-1">
                    <button
                      type="button"
                      class="flex w-full items-center justify-between gap-2 rounded-xl px-3 py-2 text-left text-sm font-semibold transition-colors"
                      :class="isGrowthSectionActive ? 'bg-primary/10 text-primary' : 'text-muted-foreground hover:bg-accent hover:text-foreground'"
                      @click="growthExpanded = !growthExpanded"
                    >
                      <span class="flex items-center gap-2.5">
                        <component :is="item.icon" class="h-4 w-4 shrink-0" />
                        <span>{{ t(item.labelKey) }}</span>
                      </span>
                      <ChevronDown class="h-4 w-4 transition-transform" :class="growthExpanded ? 'rotate-180' : ''" />
                    </button>
                    <div v-if="growthExpanded" class="space-y-0.5 pl-4">
                      <button
                        v-for="child in item.children"
                        :key="`m-${child.key}`"
                        type="button"
                        class="flex w-full items-center gap-2 rounded-xl px-3 py-2 text-left text-[13px] font-medium transition-colors"
                        :class="activeSection === child.key ? 'bg-primary/10 font-semibold text-primary' : 'text-muted-foreground hover:bg-accent hover:text-foreground'"
                        @click="navigateToSection(child.key)"
                      >
                        <component :is="child.icon" class="h-3.5 w-3.5 shrink-0" />
                        <span>{{ t(child.labelKey) }}</span>
                      </button>
                    </div>
                  </div>
                  <button
                    v-else
                    type="button"
                    class="flex w-full items-center gap-2.5 rounded-xl px-3 py-2 text-left text-sm font-medium transition-colors"
                    :class="activeSection === item.key ? 'bg-primary/10 font-semibold text-primary' : 'text-muted-foreground hover:bg-accent hover:text-foreground'"
                    @click="navigateToSection(item.key)"
                  >
                    <component :is="item.icon" class="h-4 w-4 shrink-0" />
                    <span>{{ t(item.labelKey) }}</span>
                  </button>
                </template>
              </div>
            </nav>

            <!-- 移动端主题与语言快捷设置 -->
            <div class="space-y-2.5 border-t border-border p-3">
              <div class="flex items-center justify-between gap-2">
                <span class="text-xs text-muted-foreground">{{ t('needtoken.theme.label') }}</span>
                <ThemeToggle variant="menu" />
              </div>
              <div class="flex flex-wrap gap-1.5">
                <button
                  v-for="lang in languages"
                  :key="`ml-${lang.code}`"
                  type="button"
                  class="rounded-lg border px-2.5 py-1 text-xs font-medium transition-colors"
                  :class="appStore.locale === lang.code ? 'border-primary/40 bg-primary/10 text-primary' : 'border-border text-muted-foreground hover:text-foreground'"
                  @click="changeLanguage(lang.code)"
                >
                  {{ lang.name }}
                </button>
              </div>
            </div>
          </aside>
        </div>
      </Transition>
    </Teleport>

    <!-- 右侧工作区（顶部固定工作台顶栏 + 独立滚动主画布） -->
    <div class="flex min-w-0 flex-1 flex-col lg:h-dvh lg:overflow-hidden">
      <header class="sticky top-0 z-30 flex h-14 shrink-0 items-center justify-between gap-3 border-b border-border bg-card/90 px-4 backdrop-blur lg:px-6">
        <!-- 左侧：移动端菜单按钮 + 当前模块标题 -->
        <div class="flex min-w-0 items-center gap-2.5">
          <button
            type="button"
            class="grid h-9 w-9 shrink-0 place-items-center rounded-xl border border-border bg-background text-foreground transition-colors hover:border-primary/40 lg:hidden"
            :aria-label="t('needtoken.memberConsole.nav.openMenu')"
            @click="mobileDrawerOpen = true"
          >
            <Menu class="h-4 w-4" />
          </button>
          <h1 class="truncate text-[15px] font-bold tracking-tight text-foreground sm:text-base">
            {{ currentSectionTitle }}
          </h1>
        </div>

        <!-- 右侧：API Endpoint 复制 + 唯一返回店面入口 + 语言 + 主题 + 个人信息头像组件 -->
        <div class="flex shrink-0 items-center gap-2">
          <div
            v-if="showGatewayFeatures"
            class="hidden items-center gap-2 rounded-xl border border-border bg-muted/40 px-2.5 py-1 text-xs xl:inline-flex"
          >
            <span class="font-semibold text-muted-foreground">API</span>
            <code class="max-w-[190px] truncate font-mono font-semibold text-foreground">
              {{ API_BASE_URL }}
            </code>
            <button
              type="button"
              class="inline-flex items-center gap-1 rounded-md border border-border bg-background px-2 py-0.5 text-[11px] font-semibold text-foreground transition-colors hover:border-primary/40 hover:text-primary"
              @click="copyEndpoint"
            >
              <Check v-if="copiedEndpoint" class="h-3 w-3 text-primary" />
              <Copy v-else class="h-3 w-3" />
              <span>{{ copiedEndpoint ? t('needtoken.common.copied') : t('needtoken.common.copy') }}</span>
            </button>
          </div>

          <!-- 唯一的返回店面入口 -->
          <RouterLink
            to="/"
            class="inline-flex h-9 items-center gap-1.5 rounded-xl border border-border bg-background px-2.5 text-xs font-semibold text-foreground transition-colors hover:border-primary/40 hover:text-primary sm:px-3"
          >
            <Store class="h-3.5 w-3.5 text-primary" />
            <span class="hidden sm:inline">{{ t('needtoken.memberConsole.nav.backToStore') }}</span>
            <ArrowUpRight class="h-3.5 w-3.5 opacity-60 sm:hidden" />
          </RouterLink>

          <!-- 语言切换 -->
          <div ref="langEl" class="relative hidden sm:block" @focusout="onLanguageFocusOut">
            <button
              type="button"
              class="grid h-9 w-9 place-items-center rounded-xl border border-border bg-background text-muted-foreground transition-colors hover:border-primary/40 hover:text-foreground"
              :aria-label="t('navbar.selectLanguage')"
              :aria-expanded="langOpen"
              aria-controls="nt-console-language-options"
              @click="langOpen = !langOpen; profileMenuOpen = false"
            >
              <Languages class="h-4 w-4" />
            </button>
            <div
              v-if="langOpen"
              id="nt-console-language-options"
              class="absolute right-0 top-[calc(100%+8px)] z-50 flex min-w-[144px] flex-col rounded-xl border border-border bg-card p-1.5 shadow-lg"
            >
              <button
                v-for="lang in languages"
                :key="lang.code"
                type="button"
                class="rounded-lg px-3 py-1.5 text-left text-xs font-medium transition-colors hover:bg-accent"
                :class="appStore.locale === lang.code ? 'bg-primary/10 font-semibold text-primary' : 'text-muted-foreground'"
                @click="changeLanguage(lang.code)"
              >
                {{ lang.name }}
              </button>
            </div>
          </div>

          <!-- 明暗主题切换 -->
          <ThemeToggle class="hidden sm:inline-flex" />

          <!-- 顶栏最右侧：个人信息头像组件（点击展开紧凑资料卡与账号安全/退出） -->
          <div ref="profileMenuEl" class="relative" @focusout="onProfileFocusOut">
            <button
              type="button"
              data-testid="topbar-profile-trigger"
              class="inline-flex h-9 max-w-[200px] items-center gap-2 rounded-full border border-border bg-background pl-1 pr-2.5 text-xs font-semibold text-foreground transition-colors hover:border-primary/45 hover:text-primary"
              :aria-expanded="profileMenuOpen"
              aria-controls="nt-console-profile-menu"
              @click="profileMenuOpen = !profileMenuOpen; langOpen = false"
            >
              <span class="grid h-7 w-7 shrink-0 place-items-center rounded-full bg-primary text-[11px] font-black text-primary-foreground">
                {{ displayInitial }}
              </span>
              <span class="hidden max-w-[100px] truncate sm:inline">
                {{ userProfileStore.displayName }}
              </span>
              <span
                v-if="showGatewayFeatures && displayNativeUserId"
                class="hidden rounded-full border border-primary/25 bg-primary/10 px-1.5 py-0.5 font-mono text-[10px] font-semibold leading-none text-primary md:inline-flex"
              >
                #{{ displayNativeUserId }}
              </span>
              <ChevronDown
                class="h-3.5 w-3.5 shrink-0 opacity-65 transition-transform duration-150"
                :class="profileMenuOpen ? 'rotate-180' : ''"
              />
            </button>

            <div
              v-if="profileMenuOpen"
              id="nt-console-profile-menu"
              class="absolute right-0 top-[calc(100%+8px)] z-50 w-[280px] rounded-2xl border border-border bg-card p-2 shadow-xl"
            >
              <button
                type="button"
                data-testid="sidebar-profile-card"
                class="group flex w-full items-center gap-3 rounded-xl border p-2.5 text-left transition-all duration-150"
                :class="activeSection === 'security'
                  ? 'border-primary/45 bg-primary/10'
                  : 'border-border/70 bg-muted/25 hover:border-primary/40 hover:bg-primary/5'"
                :title="t('needtoken.memberConsole.nav.profileCardHint')"
                @click="navigateToSection('security')"
              >
                <div
                  class="flex h-9 w-9 shrink-0 items-center justify-center rounded-xl text-sm font-black transition-colors"
                  :class="activeSection === 'security'
                    ? 'bg-primary text-primary-foreground'
                    : 'bg-primary/10 text-primary group-hover:bg-primary group-hover:text-primary-foreground'"
                >
                  {{ displayInitial }}
                </div>
                <div class="min-w-0 flex-1">
                  <div class="flex items-center gap-1.5">
                    <span class="truncate text-xs font-bold text-foreground group-hover:text-primary">
                      {{ userProfileStore.displayName }}
                    </span>
                    <span
                      v-if="showGatewayFeatures && displayNativeUserId"
                      class="shrink-0 rounded-full border border-primary/25 bg-primary/10 px-1.5 py-0.5 font-mono text-[10px] font-semibold leading-none text-primary"
                    >
                      #{{ displayNativeUserId }}
                    </span>
                  </div>
                  <p class="mt-0.5 truncate text-[11px] text-muted-foreground">
                    {{ userProfileStore.profile?.email || t('personalCenter.subtitle') }}
                  </p>
                  <div class="mt-1 flex flex-wrap items-center gap-1">
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
                <ChevronRight class="h-4 w-4 shrink-0 text-muted-foreground/70 transition-transform group-hover:translate-x-0.5 group-hover:text-primary" />
              </button>

              <div class="mt-1.5 space-y-0.5 border-t border-border pt-1.5">
                <button
                  type="button"
                  class="flex w-full items-center gap-2.5 rounded-xl px-3 py-2 text-left text-xs font-semibold text-foreground transition-colors hover:bg-accent hover:text-primary"
                  @click="navigateToSection('security')"
                >
                  <ShieldCheck class="h-4 w-4 shrink-0 text-primary" />
                  <span>{{ t('needtoken.memberConsole.nav.security') }}</span>
                </button>
                <button
                  type="button"
                  class="flex w-full items-center gap-2.5 rounded-xl px-3 py-2 text-left text-xs font-semibold text-red-600 transition-colors hover:bg-red-500/10 dark:text-red-400"
                  @click="handleLogout"
                >
                  <LogOut class="h-4 w-4 shrink-0" />
                  <span>{{ t('needtoken.accountMenu.signOut') }}</span>
                </button>
              </div>
            </div>
          </div>
        </div>
      </header>

      <!-- 主内容画布（独立滚动） -->
      <main ref="mainScrollEl" class="flex-1 overflow-y-auto">
        <div class="mx-auto w-full max-w-[1440px] space-y-5 px-4 py-5 lg:px-6">
          <Alert
            v-if="globalAlert"
            :variant="pageAlertVariant(globalAlert.level)"
            :class="pageAlertToneClass(globalAlert.level)"
          >
            <AlertDescription>{{ globalAlert.message }}</AlertDescription>
          </Alert>

          <!-- 菜单切换仅更新右侧主内容区，固定侧栏与顶栏保持不动 -->
          <Transition name="nt-console-panel" mode="out-in" @before-enter="resetMainScroll">
            <div :key="activeSection" class="space-y-5">
              <!-- 1. 控制台总览（Bento 仪表盘布局） -->
          <template v-if="activeSection === 'overview'">
            <!-- 仅在存在未完成的旧账号手动绑定或待激活订单时提示 -->
            <ApiAccountLinkBanner v-if="showGatewayFeatures" next-path="/me" only-when-action-needed />

            <!-- 第一行：四格核心 KPI 指标卡 -->
            <div class="grid grid-cols-1 gap-3.5 sm:grid-cols-2 xl:grid-cols-4">
              <button
                type="button"
                class="rounded-2xl border bg-card p-4 text-left shadow-xs transition-all hover:border-primary/40 hover:shadow-sm"
                @click="navigateToSection('assets')"
              >
                <div class="flex items-center justify-between text-xs font-medium text-muted-foreground">
                  <span>{{ t('needtoken.memberConsole.overview.storeWallet') }}</span>
                  <Wallet class="h-4 w-4 text-primary" />
                </div>
                <p class="mt-2 font-mono text-2xl font-black text-primary">
                  {{ userProfileStore.loadingWallet ? '—' : storeWalletDisplay }}
                </p>
                <p class="mt-1 text-[11.5px] text-muted-foreground">
                  {{
                    showGatewayFeatures && apiSummary
                      ? t('needtoken.memberConsole.overview.unifiedBalanceApiHint', {
                        usd: formatApiQuotaUsd(apiSummary.quota ?? 0),
                      })
                      : t('needtoken.memberConsole.overview.storeWalletHint')
                  }}
                </p>
              </button>

              <button
                v-if="showGatewayFeatures"
                type="button"
                class="rounded-2xl border bg-card p-4 text-left shadow-xs transition-all hover:border-primary/40 hover:shadow-sm"
                @click="navigateToSection('plans')"
              >
                <div class="flex items-center justify-between text-xs font-medium text-muted-foreground">
                  <span>{{ t('needtoken.memberConsole.overview.apiPlanQuota') }}</span>
                  <Cpu class="h-4 w-4 text-primary" />
                </div>
                <p class="mt-2 font-mono text-2xl font-black text-foreground">
                  {{ gatewayLoading && !apiSummary ? '—' : formatApiQuotaUsd(subscriptionRemainingQuota) }}
                </p>
                <p class="mt-1 text-[11.5px] text-muted-foreground">
                  {{
                    t('needtoken.memberConsole.overview.apiPlanQuotaHint', {
                      count: activeSubscriptions.length,
                    })
                  }}
                </p>
              </button>

              <button
                type="button"
                class="rounded-2xl border bg-card p-4 text-left shadow-xs transition-all hover:border-primary/40 hover:shadow-sm"
                @click="navigateToSection('affiliate')"
              >
                <div class="flex items-center justify-between text-xs font-medium text-muted-foreground">
                  <span>{{ t('needtoken.memberConsole.overview.affiliateCommission') }}</span>
                  <Megaphone class="h-4 w-4 text-primary" />
                </div>
                <p class="mt-2 font-mono text-2xl font-black text-foreground">
                  {{ affiliateLoading ? '—' : affiliateAvailableDisplay }}
                </p>
                <p class="mt-1 text-[11.5px] text-muted-foreground">
                  {{ affiliateHintText }}
                </p>
              </button>

              <button
                v-if="showGatewayFeatures"
                type="button"
                class="rounded-2xl border bg-card p-4 text-left shadow-xs transition-all hover:border-primary/40 hover:shadow-sm"
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
                <p class="mt-1 text-[11.5px] text-muted-foreground">
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
                class="rounded-2xl border bg-card p-4 text-left shadow-xs transition-all hover:border-primary/40 hover:shadow-sm"
                @click="navigateToSection('orders')"
              >
                <div class="flex items-center justify-between text-xs font-medium text-muted-foreground">
                  <span>{{ t('personalCenter.tabs.orders') }}</span>
                  <ShoppingBag class="h-4 w-4 text-primary" />
                </div>
                <p class="mt-2 font-mono text-2xl font-black text-foreground">
                  {{ userProfileStore.loadingOrders ? '—' : userProfileStore.ordersTotal }}
                </p>
                <p class="mt-1 text-[11.5px] text-muted-foreground">
                  {{ t('personalCenter.overview.viewAllOrders') }}
                </p>
              </button>
            </div>

            <!-- 第二行：双栏 Bento 主控网格 -->
            <div class="grid grid-cols-1 items-start gap-4 xl:grid-cols-12">
              <!-- 左 8 列：趋势与生效套餐（非网关模式下展示最近订单） -->
              <div class="space-y-4 xl:col-span-8">
                <template v-if="showGatewayFeatures">
                  <!-- 近 7 天网关调用趋势 -->
                  <div class="rounded-2xl border bg-card p-5 shadow-xs">
                    <div class="mb-4 flex flex-wrap items-center justify-between gap-2">
                      <div>
                        <h2 class="text-sm font-bold text-foreground sm:text-base">{{ t('needtoken.memberConsole.overview.trendTitle') }}</h2>
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
                  <div class="rounded-2xl border bg-card p-5 shadow-xs">
                    <div class="mb-4 flex flex-wrap items-center justify-between gap-2">
                      <div>
                        <h2 class="text-sm font-bold text-foreground sm:text-base">{{ t('needtoken.memberConsole.plans.title') }}</h2>
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

                    <div v-else-if="activeSubscriptions.length === 0" class="flex flex-col items-start justify-between gap-3 rounded-xl border border-dashed p-4 sm:flex-row sm:items-center">
                      <div>
                        <p class="text-sm font-semibold text-foreground">{{ t('needtoken.memberConsole.plans.emptyTitle') }}</p>
                        <p class="mt-0.5 text-xs text-muted-foreground">{{ t('needtoken.memberConsole.plans.emptyHint') }}</p>
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
                                class="rounded-md border bg-background px-1.5 py-0.5 text-[10.5px] font-medium text-muted-foreground"
                              >
                                {{ formatOverviewGroup(grp) }}
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
                </template>

                <!-- 非网关模式（分销独立站）左栏直接展示最近订单 -->
                <div v-else class="rounded-2xl border bg-card p-5 shadow-xs">
                  <div class="mb-4 flex flex-wrap items-center justify-between gap-2">
                    <h2 class="text-sm font-bold text-foreground sm:text-base">{{ t('personalCenter.overview.recentOrdersTitle') }}</h2>
                    <Button variant="ghost" size="sm" class="rounded-full" @click="navigateToSection('orders')">
                      {{ t('personalCenter.overview.viewAllOrders') }}
                    </Button>
                  </div>

                  <div v-if="userProfileStore.loadingOrders" class="space-y-3">
                    <div v-for="idx in 3" :key="idx" class="h-14 animate-pulse rounded-xl border bg-muted" />
                  </div>

                  <div v-else-if="userProfileStore.recentOrders.length === 0" class="rounded-xl border border-dashed px-4 py-5 text-sm text-muted-foreground">
                    {{ t('personalCenter.overview.emptyOrders') }}
                  </div>

                  <div v-else class="space-y-2.5">
                    <div
                      v-for="order in userProfileStore.recentOrders"
                      :key="order.order_no"
                      class="rounded-xl border bg-card px-4 py-3 transition-all hover:border-primary/30 hover:shadow-xs"
                    >
                      <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                        <div>
                          <div class="font-mono text-xs text-muted-foreground">{{ t('orders.orderNo') }}：{{ order.order_no }}</div>
                          <div class="mt-1 text-sm font-semibold text-foreground">
                            {{ formatMoney(order.total_amount, order.currency) }}
                          </div>
                          <div class="mt-0.5 text-xs text-muted-foreground">{{ formatDate(order.created_at) }}</div>
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
              </div>

              <!-- 右 4 列：快捷工作台 + 最近订单摘要 -->
              <div class="space-y-4 xl:col-span-4">
                <!-- 快捷工作台 -->
                <div class="rounded-2xl border bg-card p-4 shadow-xs">
                  <h2 class="mb-3 text-sm font-bold text-foreground">{{ t('needtoken.memberConsole.quick.title') }}</h2>
                  <button
                    v-if="showGatewayFeatures"
                    type="button"
                    class="mb-2.5 flex w-full items-center justify-between gap-2 rounded-xl border border-primary/35 bg-primary/8 px-3.5 py-2.5 text-left transition-colors hover:border-primary/60 hover:bg-primary/12"
                    @click="openQuickUseDrawer()"
                  >
                    <div class="flex min-w-0 items-center gap-2.5">
                      <span class="grid h-8 w-8 shrink-0 place-items-center rounded-lg bg-primary text-primary-foreground">
                        <Zap class="h-4 w-4" />
                      </span>
                      <div class="min-w-0">
                        <p class="truncate text-xs font-bold text-foreground">{{ t('needtoken.memberConsole.quick.quickUse') }}</p>
                        <p class="truncate text-[11px] text-muted-foreground">{{ t('needtoken.memberConsole.quick.quickUseHint') }}</p>
                      </div>
                    </div>
                    <ChevronRight class="h-4 w-4 shrink-0 text-primary" />
                  </button>
                  <div class="grid grid-cols-2 gap-2">
                    <Button v-if="showGatewayFeatures" size="sm" class="h-9 justify-start px-3 text-xs" @click="navigateToSection('keys')">
                      <KeyRound class="mr-1.5 h-3.5 w-3.5 shrink-0" />
                      <span class="truncate">{{ t('needtoken.memberConsole.quick.manageKeys') }}</span>
                    </Button>
                    <Button v-if="showGatewayFeatures" variant="outline" size="sm" class="h-9 justify-start px-3 text-xs" @click="navigateToSection('playground')">
                      <FlaskConical class="mr-1.5 h-3.5 w-3.5 shrink-0 text-primary" />
                      <span class="truncate">{{ t('needtoken.memberConsole.nav.playground') }}</span>
                    </Button>
                    <Button v-if="showGatewayFeatures" as-child variant="outline" size="sm" class="h-9 justify-start px-3 text-xs">
                      <RouterLink to="/plans">
                        <Sparkles class="mr-1.5 h-3.5 w-3.5 shrink-0 text-primary" />
                        <span class="truncate">{{ t('needtoken.memberConsole.quick.buyPlans') }}</span>
                      </RouterLink>
                    </Button>
                    <Button v-if="showGatewayFeatures" variant="outline" size="sm" class="h-9 justify-start px-3 text-xs" @click="navigateToSection('usage')">
                      <Activity class="mr-1.5 h-3.5 w-3.5 shrink-0" />
                      <span class="truncate">{{ t('needtoken.memberConsole.quick.viewUsage') }}</span>
                    </Button>
                    <Button variant="outline" size="sm" class="h-9 justify-start px-3 text-xs" @click="openAssetsSubTab('giftCard')">
                      <Gift class="mr-1.5 h-3.5 w-3.5 shrink-0" />
                      <span class="truncate">{{ t('needtoken.memberConsole.quick.redeemGiftCard') }}</span>
                    </Button>
                    <Button v-if="canAccessResellerConsole" variant="outline" size="sm" class="h-9 justify-start px-3 text-xs" @click="navigateToSection('reseller')">
                      <Store class="mr-1.5 h-3.5 w-3.5 shrink-0" />
                      <span class="truncate">{{ t('needtoken.memberConsole.nav.resellerSystem') }}</span>
                    </Button>
                  </div>

                  <!-- 小于 xl 屏幕时在此处展示 API 接入地址复制栏 -->
                  <div v-if="showGatewayFeatures" class="mt-3 flex items-center justify-between gap-2 rounded-xl border bg-muted/40 px-3 py-2 text-xs xl:hidden">
                    <div class="flex min-w-0 items-center gap-2">
                      <span class="shrink-0 font-semibold text-muted-foreground">{{ t('needtoken.memberConsole.quick.apiEndpoint') }}</span>
                      <code class="truncate font-mono font-semibold text-foreground">{{ API_BASE_URL }}</code>
                    </div>
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

                <!-- 网关模式下的最近订单紧凑卡片 -->
                <div v-if="showGatewayFeatures" class="rounded-2xl border bg-card p-4 shadow-xs">
                  <div class="mb-3 flex items-center justify-between gap-2">
                    <h2 class="text-sm font-bold text-foreground">{{ t('personalCenter.overview.recentOrdersTitle') }}</h2>
                    <Button variant="ghost" size="sm" class="h-7 rounded-full px-2.5 text-xs" @click="navigateToSection('orders')">
                      {{ t('personalCenter.overview.viewAllOrders') }}
                    </Button>
                  </div>

                  <div v-if="userProfileStore.loadingOrders" class="space-y-2.5">
                    <div v-for="idx in 2" :key="idx" class="h-14 animate-pulse rounded-xl border bg-muted" />
                  </div>

                  <div v-else-if="userProfileStore.recentOrders.length === 0" class="rounded-xl border border-dashed px-3.5 py-4 text-xs text-muted-foreground">
                    {{ t('personalCenter.overview.emptyOrders') }}
                  </div>

                  <div v-else class="space-y-2.5">
                    <div
                      v-for="order in userProfileStore.recentOrders"
                      :key="order.order_no"
                      class="rounded-xl border bg-muted/15 p-3 transition-all hover:border-primary/30"
                    >
                      <div class="flex items-start justify-between gap-2">
                        <div class="min-w-0">
                          <div class="truncate font-mono text-[11px] text-muted-foreground">{{ order.order_no }}</div>
                          <div class="mt-0.5 text-sm font-bold text-foreground">
                            {{ formatMoney(order.total_amount, order.currency) }}
                          </div>
                          <div class="mt-0.5 text-[11px] text-muted-foreground">{{ formatDate(order.created_at) }}</div>
                        </div>
                        <Badge :variant="statusVariant(order.status)" size="sm" class="shrink-0">
                          {{ statusLabel(order.status) }}
                        </Badge>
                      </div>
                      <div class="mt-2.5 flex flex-wrap items-center gap-1.5">
                        <Button as-child variant="outline" size="sm" class="h-7 px-2.5 text-xs">
                          <RouterLink :to="`/orders/${order.order_no}`">{{ t('orders.viewDetails') }}</RouterLink>
                        </Button>
                        <Button v-if="order.status === 'pending_payment'" as-child size="sm" class="h-7 px-2.5 text-xs">
                          <RouterLink :to="`/pay?order_no=${order.order_no}`">{{ t('orders.payNow') }}</RouterLink>
                        </Button>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </template>

          <!-- 2. API 密钥管理 -->
          <GatewayKeysPanel
            v-else-if="activeSection === 'keys' && showGatewayFeatures"
            ref="gatewayKeysPanelRef"
            @changed="refreshGatewayData"
            @open-quick-use="openQuickUseDrawer"
          />

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

          <!-- 8. 我的钱包（统一钱包余额、套餐额度、弹窗充值与礼品卡兑换、合并资金与兑换记录） -->
          <MemberWalletPanel
            v-else-if="activeSection === 'assets'"
            ref="memberWalletPanelRef"
            :show-gateway-features="showGatewayFeatures"
            :store-wallet-display="storeWalletDisplay"
            :api-quota-display="apiSummary ? formatApiQuotaUsd(apiSummary.quota ?? 0) : ''"
            :has-api-summary="!!apiSummary"
            :gateway-loading="gatewayLoading"
            :plan-quota-display="formatApiQuotaUsd(subscriptionRemainingQuota)"
            :active-plan-count="activeSubscriptions.length"
            :affiliate-loading="affiliateLoading"
            :affiliate-available-display="affiliateAvailableDisplay"
            :affiliate-hint-text="affiliateHintText"
            :initial-modal="pendingWalletModal"
            @changed="handleAssetsChanged"
            @navigate="navigateToSection"
          />

          <!-- 9. 推广拉新 -->
          <AffiliatePanel v-else-if="activeSection === 'affiliate'" />

          <!-- 10. 分销系统（全面纳管重构，统一 NeedToken 视觉与路由） -->
          <ResellerWorkspacePanel v-else-if="activeSection === 'reseller' && canAccessResellerConsole" />

          <!-- 11. 账号与安全（点击顶部头像菜单或侧栏菜单进入） -->
          <template v-else-if="activeSection === 'security'">
            <ProfilePanel />
            <SecurityPanel />
          </template>
            </div>
          </Transition>
        </div>
      </main>
    </div>

    <!-- 快捷使用 70% × 100% 抽屉（自动注入会员专属密钥与模型配置） -->
    <MemberQuickUseDrawer
      v-if="showGatewayFeatures"
      ref="quickUseDrawerRef"
      @create-key="handleQuickUseCreateKey"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch, type Component } from 'vue'
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
  Languages,
  LayoutDashboard,
  LogOut,
  Megaphone,
  Menu,
  PanelLeftClose,
  PanelLeftOpen,
  ShieldCheck,
  ShoppingBag,
  Sparkles,
  Store,
  UserPlus,
  Wallet,
  X,
  Zap,
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
import AffiliatePanel from '../../views/personal/AffiliatePanel.vue'
import ApiAccountLinkBanner from './components/ApiAccountLinkBanner.vue'
import BrandMark from './components/BrandMark.vue'
import ThemeToggle from './components/ThemeToggle.vue'
import GatewayKeysPanel from './components/member/GatewayKeysPanel.vue'
import GatewayModelsPanel from './components/member/GatewayModelsPanel.vue'
import GatewayPlansPanel from './components/member/GatewayPlansPanel.vue'
import GatewayPlaygroundPanel from './components/member/GatewayPlaygroundPanel.vue'
import GatewayUsagePanel from './components/member/GatewayUsagePanel.vue'
import MemberQuickUseDrawer from './components/member/MemberQuickUseDrawer.vue'
import MemberWalletPanel from './components/member/MemberWalletPanel.vue'
import ResellerWorkspacePanel from './components/member/ResellerWorkspacePanel.vue'
import { API_BASE_URL, NEEDTOKEN_BRAND_NAME } from './config'
import { useApiAccountLink } from './composables/useApiAccountLink'
import {
  computeSubscriptionRemainingQuota,
  computeSubscriptionUsagePercent,
  formatApiQuotaUsd,
  formatSubscriptionExpiry,
} from './lib/apiAccountLinkPolicy'
import {
  formatConsoleGroupLabel,
  normalizeNeedtokenMemberSection,
  sumSubscriptionRemainingQuota,
  type NeedtokenMemberSection,
} from './lib/gatewayConsoleHelpers'
import {
  getGatewayUsage,
  listGatewayTokens,
  type GatewayGroupMetaItem,
  type GatewayUsageResponse,
} from './lib/gatewayConsoleApi'

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

const brandName = computed(() => (appStore.isResellerTenant
  ? String(appStore.config?.brand?.site_name || '').trim() || NEEDTOKEN_BRAND_NAME
  : NEEDTOKEN_BRAND_NAME))
const resellerLogo = computed(() => {
  const logo = String(appStore.config?.brand?.site_logo || '').trim()
  return logo ? getImageUrl(logo) : ''
})

const GATEWAY_ONLY_SECTIONS = new Set<NeedtokenMemberSection>(['keys', 'models', 'playground', 'usage', 'plans'])

const activeSection = computed<NeedtokenMemberSection>(() => {
  const normalized = normalizeNeedtokenMemberSection(props.section)
  if (!showGatewayFeatures.value && GATEWAY_ONLY_SECTIONS.has(normalized)) {
    return 'overview'
  }
  return normalized
})

const SIDEBAR_COLLAPSE_KEY = 'nt-member-sidebar-collapsed'
const readSidebarCollapsed = () => {
  if (typeof window === 'undefined') return false
  try {
    return localStorage.getItem(SIDEBAR_COLLAPSE_KEY) === '1'
  } catch {
    return false
  }
}
const sidebarCollapsed = ref(readSidebarCollapsed())
const toggleSidebarCollapse = () => {
  sidebarCollapsed.value = !sidebarCollapsed.value
  try {
    localStorage.setItem(SIDEBAR_COLLAPSE_KEY, sidebarCollapsed.value ? '1' : '0')
  } catch {
    /* ignore storage unavailable */
  }
}

const mobileDrawerOpen = ref(false)
const mainScrollEl = ref<HTMLElement | null>(null)
const profileMenuOpen = ref(false)
const profileMenuEl = ref<HTMLElement | null>(null)
const langOpen = ref(false)
const langEl = ref<HTMLElement | null>(null)

const languages = [
  { code: 'zh-CN', name: '简体中文' },
  { code: 'zh-TW', name: '繁體中文' },
  { code: 'en-US', name: 'English' },
]

const changeLanguage = (code: string) => {
  appStore.setLocale(code)
  langOpen.value = false
}

const onLanguageFocusOut = (event: FocusEvent) => {
  if (!(event.relatedTarget instanceof Node) || !langEl.value?.contains(event.relatedTarget)) {
    langOpen.value = false
  }
}

const onProfileFocusOut = (event: FocusEvent) => {
  if (!(event.relatedTarget instanceof Node) || !profileMenuEl.value?.contains(event.relatedTarget)) {
    profileMenuOpen.value = false
  }
}

const memberWalletPanelRef = ref<InstanceType<typeof MemberWalletPanel> | null>(null)
const gatewayKeysPanelRef = ref<InstanceType<typeof GatewayKeysPanel> | null>(null)
const quickUseDrawerRef = ref<InstanceType<typeof MemberQuickUseDrawer> | null>(null)
const openQuickUseDrawer = (tokenId?: number) => {
  quickUseDrawerRef.value?.openDrawer(tokenId)
}
const handleQuickUseCreateKey = async () => {
  navigateToSection('keys')
  await nextTick()
  setTimeout(() => {
    gatewayKeysPanelRef.value?.openCreateModal()
  }, 120)
}
const pendingWalletModal = ref<'redeem' | 'recharge' | null>(props.section === 'giftCard' ? 'redeem' : null)
watch(
  () => props.section,
  (next) => {
    if (next === 'giftCard') {
      pendingWalletModal.value = 'redeem'
      if (activeSection.value === 'assets') {
        void memberWalletPanelRef.value?.openRedeemModal()
      }
    } else if (next !== 'assets') {
      pendingWalletModal.value = null
    }
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

const handleGrowthGroupClick = () => {
  if (sidebarCollapsed.value) {
    sidebarCollapsed.value = false
    growthExpanded.value = true
    try {
      localStorage.setItem(SIDEBAR_COLLAPSE_KEY, '0')
    } catch {
      /* ignore */
    }
    return
  }
  growthExpanded.value = !growthExpanded.value
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
  groups.push({
    key: 'account',
    labelKey: 'needtoken.memberConsole.nav.groupAccount',
    items: [
      { key: 'security', labelKey: 'needtoken.memberConsole.nav.security', icon: ShieldCheck },
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

const currentSectionTitle = computed(() => {
  const hit = flatNavItems.value.find(item => item.key === activeSection.value)
  return hit ? t(hit.labelKey) : t('needtoken.memberConsole.nav.overview')
})

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
  mobileDrawerOpen.value = false
  profileMenuOpen.value = false
  const key = section as NeedtokenMemberSection
  if (sectionRoutes[key]) {
    void router.push(sectionRoutes[key])
  }
}

const openAssetsSubTab = (tab: 'giftCard' | 'wallet') => {
  pendingWalletModal.value = tab === 'giftCard' ? 'redeem' : 'recharge'
  if (activeSection.value === 'assets') {
    if (tab === 'giftCard') void memberWalletPanelRef.value?.openRedeemModal()
    else void memberWalletPanelRef.value?.openRechargeModal()
    return
  }
  navigateToSection('assets')
}

const gatewayLoading = ref(false)
const activeKeyCount = ref<number | null>(null)
const overviewUsage = ref<GatewayUsageResponse | null>(null)
const overviewGroupMeta = ref<Record<string, GatewayGroupMetaItem>>({})
const overviewUsableGroups = ref<Record<string, string>>({})
const overviewGroupRatio = ref<Record<string, number>>({})
const affiliateLoading = ref(false)
const affiliateDashboard = ref<AffiliateDashboardData | null>(null)

const formatOverviewGroup = (code: string) =>
  formatConsoleGroupLabel(code, overviewGroupMeta.value, overviewUsableGroups.value, overviewGroupRatio.value)

const overviewBarPercent = (quota: number) => {
  const daily = overviewUsage.value?.daily || []
  const maxQuota = Math.max(0, ...daily.map(item => item.quota))
  if (maxQuota <= 0 || quota <= 0) return 6
  return Math.max(10, Math.round((quota / maxQuota) * 100))
}

const displayNativeUserId = computed(() => apiStatus.value?.native_user_id || apiSummary.value?.native_user_id || 0)
const activeSubscriptions = computed(() => apiSummary.value?.active_subscriptions ?? [])
const subscriptionRemainingQuota = computed(() => sumSubscriptionRemainingQuota(activeSubscriptions.value))

const handleAssetsChanged = async () => {
  const tasks: Array<Promise<unknown>> = [userProfileStore.loadWalletAccount()]
  if (showGatewayFeatures.value && userAuth.isAuthenticated) {
    tasks.push(loadApiSummary(true))
  }
  await Promise.all(tasks)
}

const resetMainScroll = () => {
  mainScrollEl.value?.scrollTo({ top: 0 })
}

watch(activeSection, (next) => {
  mobileDrawerOpen.value = false
  profileMenuOpen.value = false
  langOpen.value = false
  if (next === 'assets' || next === 'overview') {
    void handleAssetsChanged()
  }
})

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
      overviewGroupMeta.value = tokenRes.group_meta || {}
      overviewUsableGroups.value = tokenRes.usable_groups || {}
      overviewGroupRatio.value = tokenRes.group_ratio || {}
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
  profileMenuOpen.value = false
  mobileDrawerOpen.value = false
  await userAuth.logout('/')
}

const onDocClick = (event: MouseEvent) => {
  const target = event.target as Node
  if (profileMenuOpen.value && profileMenuEl.value && !profileMenuEl.value.contains(target)) {
    profileMenuOpen.value = false
  }
  if (langOpen.value && langEl.value && !langEl.value.contains(target)) {
    langOpen.value = false
  }
}

const onDocKeydown = (event: KeyboardEvent) => {
  if (event.key !== 'Escape') return
  profileMenuOpen.value = false
  langOpen.value = false
  mobileDrawerOpen.value = false
}

onMounted(() => {
  document.addEventListener('click', onDocClick)
  document.addEventListener('keydown', onDocKeydown)
  void Promise.all([refreshGatewayData(), loadAffiliateOverview()])
})

onUnmounted(() => {
  if (copyTimer) clearTimeout(copyTimer)
  document.removeEventListener('click', onDocClick)
  document.removeEventListener('keydown', onDocKeydown)
})
</script>

<style scoped>
.nt-drawer-fade-enter-active,
.nt-drawer-fade-leave-active {
  transition: opacity 180ms ease;
}
.nt-drawer-fade-enter-from,
.nt-drawer-fade-leave-to {
  opacity: 0;
}
@media (prefers-reduced-motion: no-preference) {
  .nt-console-panel-enter-active {
    transition: opacity 160ms ease-out, transform 180ms cubic-bezier(0.22, 1, 0.36, 1);
  }
  .nt-console-panel-leave-active {
    transition: opacity 90ms ease-in, transform 90ms ease-in;
    pointer-events: none;
  }
  .nt-console-panel-enter-from {
    opacity: 0;
    transform: translateY(4px);
  }
  .nt-console-panel-leave-to {
    opacity: 0;
    transform: translateY(-2px);
  }
}
</style>
