<template>
  <div class="space-y-5">
    <!-- 顶部标题与双视图切换 -->
    <div class="rounded-2xl border bg-card p-5 shadow-xs">
      <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h2 class="text-lg font-bold text-foreground">{{ t('needtoken.memberConsole.models.title') }}</h2>
          <p class="mt-1 text-xs text-muted-foreground">{{ t('needtoken.memberConsole.models.subtitle') }}</p>
        </div>
        <div class="flex flex-wrap items-center gap-2">
          <span
            v-if="activePlanInfo"
            class="inline-flex items-center gap-1.5 rounded-full border border-emerald-500/35 bg-emerald-500/10 px-3 py-1 text-[11.5px] font-semibold text-emerald-700 dark:text-emerald-300"
          >
            <span>{{ t('needtoken.memberConsole.models.planActiveBadge') }}: {{ activePlanInfo.title }}</span>
            <span v-if="activePlanInfo.multiplier > 1.01" class="font-mono">({{ Number(activePlanInfo.multiplier.toFixed(2)) }}x)</span>
          </span>
          <div class="flex items-center gap-1.5 rounded-xl border bg-muted/40 p-1">
            <button
              type="button"
              class="inline-flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-xs font-semibold transition-colors"
              :class="activeTab === 'catalog' ? 'bg-background text-primary shadow-xs' : 'text-muted-foreground hover:text-foreground'"
              @click="activeTab = 'catalog'"
            >
              <Box class="h-3.5 w-3.5" />
              <span>{{ t('needtoken.memberConsole.models.tabCatalog') }}</span>
            </button>
            <button
              type="button"
              class="inline-flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-xs font-semibold transition-colors"
              :class="activeTab === 'analytics' ? 'bg-background text-primary shadow-xs' : 'text-muted-foreground hover:text-foreground'"
              @click="switchToAnalytics"
            >
              <BarChart3 class="h-3.5 w-3.5" />
              <span>{{ t('needtoken.memberConsole.models.tabAnalytics') }}</span>
            </button>
          </div>
        </div>
      </div>

      <!-- Tab 1 筛选栏：对齐店面模型广场 UI（厂商胶囊 + 搜索 + 分组 + 排序 + 动态套餐视角 + 视图切换） -->
      <div v-if="activeTab === 'catalog'" class="mt-4 space-y-3 border-t pt-4">
        <!-- 厂商筛选胶囊栏 -->
        <nav v-if="vendorStats.length > 0" class="flex items-center gap-2 overflow-x-auto pb-1" :aria-label="t('needtoken.models.vendor')">
          <button
            type="button"
            class="inline-flex min-h-8 shrink-0 items-center gap-1.5 rounded-full border px-3.5 py-1 text-xs font-semibold transition-colors"
            :class="selectedVendor === '' ? 'border-primary bg-primary text-primary-foreground' : 'border-border bg-background text-muted-foreground hover:text-foreground'"
            @click="selectedVendor = ''"
          >
            <span>{{ t('needtoken.models.allVendors') }}</span>
            <span class="font-mono text-[11px] opacity-80">{{ groupScopedModels.length }}</span>
          </button>
          <button
            v-for="item in vendorStats"
            :key="item.name"
            type="button"
            class="inline-flex min-h-8 shrink-0 items-center gap-1.5 rounded-full border px-3.5 py-1 text-xs font-semibold transition-colors"
            :class="selectedVendor === item.name ? 'border-primary bg-primary text-primary-foreground' : 'border-border bg-background text-muted-foreground hover:text-foreground'"
            @click="selectedVendor = selectedVendor === item.name ? '' : item.name"
          >
            <span>{{ item.name }}</span>
            <span class="font-mono text-[11px] opacity-80">{{ item.count }}</span>
          </button>
        </nav>

        <!-- 搜索 + 分组 + 排序 + 动态套餐折扣 + 视图切换 -->
        <div class="flex flex-wrap items-center gap-2.5">
          <div class="relative min-w-[200px] flex-1 sm:max-w-[260px]">
            <Search class="pointer-events-none absolute left-3 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-muted-foreground" />
            <input
              v-model.trim="searchKeyword"
              type="search"
              :placeholder="t('needtoken.memberConsole.models.searchPlaceholder')"
              class="h-9 w-full rounded-xl border bg-background pl-8 pr-3 text-xs text-foreground outline-none focus:border-primary"
            />
          </div>

          <NeedtokenSelect
            v-if="groupSelectOptions.length > 1"
            v-model="selectedGroup"
            :options="groupSelectOptions"
            :label="t('needtoken.memberConsole.models.colGroups')"
            compact
            align="start"
            class="min-w-[200px]"
          />

          <NeedtokenSelect
            v-if="hasPerCallModels"
            v-model="billingFilter"
            :options="billingSelectOptions"
            :label="t('needtoken.memberConsole.models.colBilling')"
            compact
            class="min-w-[130px]"
          />

          <NeedtokenSelect
            v-model="sortOrder"
            :options="sortSelectOptions"
            :label="t('needtoken.models.sort')"
            compact
            class="min-w-[140px]"
          />

          <div class="ml-auto flex flex-wrap items-center gap-2">
            <!-- 动态加载生效订阅套餐折扣切换 -->
            <div v-if="planMultiplierOptions.length > 0" class="inline-flex flex-wrap rounded-xl border bg-muted/30 p-1">
              <button
                type="button"
                class="rounded-lg px-2.5 py-1 text-xs font-semibold transition-colors"
                :class="priceMultiplier === 1 ? 'bg-background text-primary shadow-xs' : 'text-muted-foreground hover:text-foreground'"
                @click="priceMultiplier = 1"
              >
                {{ t('needtoken.memberConsole.models.paygMode') }}
              </button>
              <button
                v-for="tier in planMultiplierOptions"
                :key="tier.key"
                type="button"
                class="rounded-lg px-2.5 py-1 text-xs font-semibold transition-colors"
                :class="priceMultiplier === tier.multiplier ? 'bg-background text-primary shadow-xs' : 'text-muted-foreground hover:text-foreground'"
                @click="priceMultiplier = tier.multiplier"
              >
                {{ tier.label }}
              </button>
            </div>

            <!-- 比价表 / 卡片视图切换 -->
            <div class="inline-flex rounded-xl border bg-muted/30 p-1">
              <button
                type="button"
                class="rounded-lg px-2.5 py-1 text-xs font-semibold transition-colors"
                :class="viewMode === 'table' ? 'bg-background text-primary shadow-xs' : 'text-muted-foreground hover:text-foreground'"
                @click="viewMode = 'table'"
              >
                {{ t('needtoken.models.viewTable') }}
              </button>
              <button
                type="button"
                class="rounded-lg px-2.5 py-1 text-xs font-semibold transition-colors"
                :class="viewMode === 'cards' ? 'bg-background text-primary shadow-xs' : 'text-muted-foreground hover:text-foreground'"
                @click="viewMode = 'cards'"
              >
                {{ t('needtoken.models.viewCards') }}
              </button>
            </div>

            <Button variant="outline" size="sm" class="h-9" :disabled="modelsLoading" @click="loadModels">
              <RefreshCw class="h-3.5 w-3.5" :class="modelsLoading ? 'animate-spin' : ''" />
            </Button>
          </div>
        </div>

        <div class="flex flex-wrap items-center justify-between gap-2 text-xs text-muted-foreground">
          <span>{{ t('needtoken.memberConsole.models.activeGroupLabel', { group: activeGroupDisplay }) }}</span>
          <span class="font-mono">{{ t('needtoken.memberConsole.models.totalCount', { count: filteredModels.length }) }}</span>
        </div>
      </div>

      <!-- Tab 2 筛选栏：统计周期 -->
      <div v-else class="mt-4 flex flex-wrap items-center justify-between gap-3 border-t pt-4">
        <p class="text-xs text-muted-foreground">{{ t('needtoken.memberConsole.models.analyticsHint') }}</p>
        <div class="flex items-center gap-1 rounded-xl border bg-muted/40 p-1">
          <button
            v-for="opt in dayOptions"
            :key="opt.days"
            type="button"
            class="rounded-lg px-3 py-1 text-xs font-semibold transition-colors"
            :class="analyticsDays === opt.days ? 'bg-background text-primary shadow-xs' : 'text-muted-foreground hover:text-foreground'"
            @click="changeAnalyticsDays(opt.days)"
          >
            {{ opt.label }}
          </button>
        </div>
      </div>
    </div>

    <div v-if="errorMsg" class="rounded-xl border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive">
      {{ errorMsg }}
    </div>

    <!-- Tab 1：可用模型（店面风格比价表 / 卡片） -->
    <template v-if="activeTab === 'catalog'">
      <div v-if="modelsLoading && !models.length" class="space-y-3 rounded-2xl border bg-card p-5">
        <div v-for="idx in 6" :key="idx" class="h-12 animate-pulse rounded-xl bg-muted" />
      </div>

      <div v-else-if="!filteredModels.length" class="rounded-2xl border bg-card px-6 py-12 text-center">
        <Box class="mx-auto h-9 w-9 text-muted-foreground/50" />
        <p class="mt-3 text-sm font-semibold text-foreground">{{ t('needtoken.memberConsole.models.emptyModels') }}</p>
      </div>

      <!-- 比价表视图（7 列：Model / Context / 实付单价与官网价 / 供应线（可用分组）/ Uptime (SLA) / 是否可用 / 操作） -->
      <div v-else-if="viewMode === 'table'" class="overflow-hidden rounded-2xl border bg-card shadow-xs">
        <div class="overflow-x-auto">
          <table class="w-full min-w-[940px] border-collapse text-left text-xs">
            <thead class="border-b bg-muted/35 text-[11px] font-semibold text-muted-foreground">
              <tr>
                <th class="px-5 py-3.5">{{ t('needtoken.memberConsole.models.colModel') }}</th>
                <th class="px-3 py-3.5">{{ t('needtoken.memberConsole.models.colContext') }}</th>
                <th class="bg-emerald-500/8 px-4 py-3.5 font-bold text-foreground">{{ t('needtoken.memberConsole.models.colPriceCompare') }}</th>
                <th class="px-3 py-3.5">{{ t('needtoken.memberConsole.models.colGroups') }}</th>
                <th class="px-3 py-3.5">{{ t('needtoken.memberConsole.models.colUptime') }}</th>
                <th class="px-3 py-3.5">{{ t('needtoken.memberConsole.models.colAvailability') }}</th>
                <th class="px-5 py-3.5 text-right">{{ t('needtoken.memberConsole.models.colActions') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="row in pagedModels"
                :key="row.modelName"
                class="border-b border-dashed border-border/80 transition-colors last:border-b-0 hover:bg-muted/20"
              >
                <!-- 1. Model -->
                <td class="px-5 py-3.5">
                  <div class="flex items-center gap-1.5">
                    <strong class="font-mono text-[13px] font-bold text-foreground">{{ row.modelName }}</strong>
                    <button
                      type="button"
                      class="inline-flex h-6 w-6 shrink-0 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
                      :aria-label="t('needtoken.models.copyModelId', { model: row.modelName })"
                      @click="copyModelId(row.modelName)"
                    >
                      <Check v-if="copiedModel === row.modelName" class="h-3.5 w-3.5 text-emerald-500" />
                      <Copy v-else class="h-3.5 w-3.5" />
                    </button>
                  </div>
                  <div class="mt-0.5 text-[11px] text-muted-foreground">{{ row.vendor }}</div>
                </td>

                <!-- 2. Context -->
                <td class="px-3 py-3.5 font-mono text-xs text-muted-foreground">
                  {{ row.contextLength || '—' }}
                </td>

                <!-- 3. NeedToken 实付单价 / 官网价（输入/输出）合并列 -->
                <td class="bg-emerald-500/8 px-4 py-3.5 font-mono">
                  <div v-if="row.isUnpriced" class="inline-flex items-center rounded-md border border-border bg-background/80 px-2 py-1 text-[11px] font-medium text-muted-foreground">
                    {{ t('needtoken.memberConsole.models.unpricedPending') }}
                  </div>
                  <template v-else-if="row.isPerCall">
                    <div class="flex flex-wrap items-center gap-1.5">
                      <strong class="text-[13.5px] font-bold text-foreground">{{ formatUsd(row.needInputPrice) }}</strong>
                      <span class="text-[11px] text-muted-foreground">/ {{ t('needtoken.memberConsole.models.perCall') }}</span>
                      <span v-if="row.discountPercent !== null" class="rounded bg-foreground px-1.5 py-0.5 text-[10px] font-bold text-background">
                        {{ row.discountPercent }}% off
                      </span>
                    </div>
                    <div v-if="row.officialInputPrice !== null" class="mt-1 text-[11px] text-muted-foreground">
                      {{ t('needtoken.memberConsole.models.officialPriceShort') }}:
                      <span class="line-through">{{ formatUsd(row.officialInputPrice) }} / {{ t('needtoken.memberConsole.models.perCall') }}</span>
                    </div>
                  </template>
                  <template v-else>
                    <!-- 上方：NeedToken 输入/输出实付单价 + 折扣徽章 -->
                    <div class="flex flex-wrap items-center gap-1.5">
                      <span class="text-[13px] font-bold text-foreground">
                        {{ formatUsd(row.needInputPrice) }}
                        <span class="mx-0.5 text-muted-foreground">/</span>
                        {{ formatUsd(row.needOutputPrice) }}
                      </span>
                      <span class="text-[10.5px] text-muted-foreground">/1M</span>
                      <span v-if="row.discountPercent !== null" class="rounded bg-foreground px-1.5 py-0.5 text-[10px] font-bold text-background">
                        {{ row.discountPercent }}% off
                      </span>
                    </div>
                    <!-- 下方：官网输入/输出价对比 -->
                    <div v-if="row.officialInputPrice !== null || row.officialOutputPrice !== null" class="mt-1 text-[11px] text-muted-foreground">
                      {{ t('needtoken.memberConsole.models.officialPriceShort') }}:
                      <span class="line-through">
                        {{ formatUsd(row.officialInputPrice) }} / {{ formatUsd(row.officialOutputPrice) }}
                      </span>
                    </div>
                  </template>
                </td>

                <!-- 4. 供应线（可用分组） -->
                <td class="px-3 py-3.5">
                  <div class="flex flex-wrap gap-1">
                    <span
                      v-for="grp in row.groups.slice(0, 3)"
                      :key="grp"
                      class="rounded-full border border-border bg-muted/40 px-2 py-0.5 font-mono text-[10.5px] font-medium text-muted-foreground"
                    >
                      {{ formatGroupShort(grp) }}
                    </span>
                  </div>
                </td>

                <!-- 5. Uptime (SLA) -->
                <td class="px-3 py-3.5">
                  <div v-if="row.uptimePct !== null" class="flex min-w-[105px] flex-col gap-1">
                    <span class="font-mono text-xs font-bold" :class="row.uptimePct >= 99 ? 'text-emerald-600 dark:text-emerald-400' : 'text-amber-600'">
                      {{ row.uptimePct.toFixed(2) }}%
                    </span>
                    <div class="flex h-2.5 items-center gap-[2px]" aria-hidden="true">
                      <i
                        v-for="(slot, idx) in row.uptimeHistory.slice(-20)"
                        :key="idx"
                        class="block h-full w-[3px] rounded-sm"
                        :class="slot === 'available' ? 'bg-emerald-500' : slot === 'busy' ? 'bg-amber-500' : slot === 'unavailable' ? 'bg-red-500' : 'bg-muted-foreground/30'"
                      />
                    </div>
                  </div>
                  <span v-else class="inline-flex items-center gap-1.5 text-[11px] text-muted-foreground">
                    <span class="h-1.5 w-1.5 rounded-full" :class="row.dotClass" aria-hidden="true" />
                    <span>{{ t('needtoken.apiServices.pricing.slaMonitoring') }}</span>
                  </span>
                </td>

                <!-- 6. 是否可用 -->
                <td class="px-3 py-3.5 whitespace-nowrap">
                  <span
                    class="inline-flex items-center gap-1.5 rounded-full border px-2.5 py-0.5 text-[11px] font-semibold"
                    :class="row.availabilityClass"
                  >
                    <span class="h-1.5 w-1.5 rounded-full" :class="row.dotClass" aria-hidden="true" />
                    <span>{{ row.availabilityLabel }}</span>
                  </span>
                </td>

                <!-- 7. 操作：在线调试 -->
                <td class="px-5 py-3.5 text-right whitespace-nowrap">
                  <Button
                    variant="outline"
                    size="sm"
                    class="h-7.5 px-3 text-xs font-semibold text-primary hover:border-primary/50 hover:bg-primary/5"
                    @click="tryInPlayground(row.modelName)"
                  >
                    <FlaskConical class="mr-1.5 h-3 w-3" />
                    {{ t('needtoken.memberConsole.models.tryPlayground') }}
                  </Button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <div v-if="totalModelPages > 1" class="flex items-center justify-between border-t px-5 py-3 text-xs text-muted-foreground">
          <span>{{ t('needtoken.memberConsole.usage.pageInfo', { page: modelPage, totalPages: totalModelPages, total: filteredModels.length }) }}</span>
          <div class="flex items-center gap-2">
            <Button variant="outline" size="sm" :disabled="modelPage <= 1" @click="modelPage--">
              {{ t('needtoken.memberConsole.usage.prevPage') }}
            </Button>
            <Button variant="outline" size="sm" :disabled="modelPage >= totalModelPages" @click="modelPage++">
              {{ t('needtoken.memberConsole.usage.nextPage') }}
            </Button>
          </div>
        </div>
      </div>

      <!-- 卡片视图（对齐店面卡片样式） -->
      <div v-else class="space-y-4">
        <div class="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
          <article
            v-for="row in pagedModels"
            :key="`card-${row.modelName}`"
            class="flex flex-col rounded-2xl border bg-card p-5 transition-colors hover:border-primary/45"
          >
            <div class="flex items-start justify-between gap-2">
              <div class="min-w-0 flex-1">
                <div class="flex items-center gap-1.5">
                  <h3 class="truncate font-mono text-sm font-bold text-foreground" :title="row.modelName">{{ row.modelName }}</h3>
                  <button
                    type="button"
                    class="inline-flex h-6 w-6 shrink-0 items-center justify-center rounded text-muted-foreground hover:bg-muted hover:text-foreground"
                    @click="copyModelId(row.modelName)"
                  >
                    <Check v-if="copiedModel === row.modelName" class="h-3.5 w-3.5 text-emerald-500" />
                    <Copy v-else class="h-3.5 w-3.5" />
                  </button>
                </div>
                <div class="mt-1 flex flex-wrap items-center gap-2 text-[11.5px] text-muted-foreground">
                  <span class="font-medium text-foreground/85">{{ row.vendor }}</span>
                  <span v-if="row.contextLength">· {{ row.contextLength }}</span>
                </div>
              </div>
              <span v-if="row.discountPercent !== null" class="shrink-0 rounded bg-foreground px-1.5 py-0.5 font-mono text-[10.5px] font-bold text-background">
                {{ row.discountPercent }}% off
              </span>
            </div>

            <div class="mt-3 flex flex-wrap items-center gap-1.5 text-[11px]">
              <span class="inline-flex items-center gap-1.5 rounded-full border px-2 py-0.5 font-medium" :class="row.availabilityClass">
                <span class="h-1.5 w-1.5 rounded-full" :class="row.dotClass" />
                <span>{{ row.availabilityLabel }}</span>
              </span>
              <span v-if="row.uptimePct !== null" class="rounded-full border bg-muted/40 px-2 py-0.5 font-mono text-emerald-600 dark:text-emerald-400">
                SLA {{ row.uptimePct.toFixed(2) }}%
              </span>
            </div>

            <div class="mt-4 rounded-xl border border-emerald-500/20 bg-emerald-500/8 p-3 font-mono text-xs">
              <div v-if="row.isUnpriced" class="text-muted-foreground">
                {{ t('needtoken.memberConsole.models.unpricedPending') }}
              </div>
              <template v-else-if="row.isPerCall">
                <div class="text-sm font-bold text-foreground">
                  {{ formatUsd(row.needInputPrice) }}
                  <span class="text-[11px] font-normal text-muted-foreground">/ {{ t('needtoken.memberConsole.models.perCall') }}</span>
                </div>
                <div v-if="row.officialInputPrice !== null" class="mt-1 text-[11px] text-muted-foreground line-through">
                  {{ t('needtoken.memberConsole.models.officialPriceShort') }} {{ formatUsd(row.officialInputPrice) }}
                </div>
              </template>
              <template v-else>
                <div class="grid grid-cols-2 gap-2">
                  <div>
                    <div class="text-[10.5px] text-muted-foreground">{{ t('needtoken.memberConsole.models.inputLabel') }}</div>
                    <div class="mt-0.5 text-sm font-bold text-foreground">{{ formatUsd(row.needInputPrice) }}</div>
                    <div v-if="row.officialInputPrice !== null" class="text-[10.5px] text-muted-foreground line-through">
                      {{ formatUsd(row.officialInputPrice) }}
                    </div>
                  </div>
                  <div>
                    <div class="text-[10.5px] text-muted-foreground">{{ t('needtoken.memberConsole.models.outputLabel') }}</div>
                    <div class="mt-0.5 text-sm font-bold text-foreground">{{ formatUsd(row.needOutputPrice) }}</div>
                    <div v-if="row.officialOutputPrice !== null" class="text-[10.5px] text-muted-foreground line-through">
                      {{ formatUsd(row.officialOutputPrice) }}
                    </div>
                  </div>
                </div>
              </template>
            </div>

            <div class="mt-auto flex items-center justify-between gap-2 pt-4">
              <div class="flex flex-wrap gap-1">
                <span
                  v-for="grp in row.groups.slice(0, 2)"
                  :key="grp"
                  class="rounded border bg-muted/40 px-1.5 py-0.5 font-mono text-[10px] text-muted-foreground"
                >
                  {{ formatGroupShort(grp) }}
                </span>
              </div>
              <Button size="sm" class="h-8 px-3 text-xs font-semibold" @click="tryInPlayground(row.modelName)">
                <FlaskConical class="mr-1.5 h-3.5 w-3.5" />
                {{ t('needtoken.memberConsole.models.tryPlayground') }}
              </Button>
            </div>
          </article>
        </div>

        <div v-if="totalModelPages > 1" class="flex items-center justify-between rounded-2xl border bg-card px-5 py-3 text-xs text-muted-foreground">
          <span>{{ t('needtoken.memberConsole.usage.pageInfo', { page: modelPage, totalPages: totalModelPages, total: filteredModels.length }) }}</span>
          <div class="flex items-center gap-2">
            <Button variant="outline" size="sm" :disabled="modelPage <= 1" @click="modelPage--">
              {{ t('needtoken.memberConsole.usage.prevPage') }}
            </Button>
            <Button variant="outline" size="sm" :disabled="modelPage >= totalModelPages" @click="modelPage++">
              {{ t('needtoken.memberConsole.usage.nextPage') }}
            </Button>
          </div>
        </div>
      </div>
    </template>

    <!-- Tab 2：模型调用分析 -->
    <div v-else class="space-y-4">
      <div class="grid grid-cols-1 gap-3 sm:grid-cols-3">
        <div class="rounded-2xl border bg-card p-5 shadow-xs">
          <p class="text-xs text-muted-foreground">{{ t('needtoken.memberConsole.usage.totalSpend') }}</p>
          <p class="mt-1.5 font-mono text-2xl font-black text-primary" :title="formatApiQuotaUsdExact(usage.total_quota)">
            {{ formatApiQuotaUsd(usage.total_quota, { compact: true, locale }) }}
          </p>
        </div>
        <div class="rounded-2xl border bg-card p-5 shadow-xs">
          <p class="text-xs text-muted-foreground">{{ t('needtoken.memberConsole.usage.totalRequests') }}</p>
          <p class="mt-1.5 font-mono text-2xl font-black text-foreground" :title="formatExactInteger(usage.total_requests)">
            {{ formatLargeCount(usage.total_requests, locale) }}
          </p>
        </div>
        <div class="rounded-2xl border bg-card p-5 shadow-xs">
          <p class="text-xs text-muted-foreground">{{ t('needtoken.memberConsole.usage.totalTokens') }}</p>
          <p class="mt-1.5 font-mono text-2xl font-black text-foreground" :title="formatExactInteger(usage.total_tokens)">
            {{ formatLargeCount(usage.total_tokens, locale) }}
          </p>
        </div>
      </div>

      <div class="rounded-2xl border bg-card shadow-xs">
        <div v-if="usageLoading && !usage.models.length" class="space-y-3 p-5">
          <div v-for="idx in 4" :key="idx" class="h-12 animate-pulse rounded-xl bg-muted" />
        </div>
        <div v-else-if="!usage.models.length" class="px-6 py-12 text-center text-sm text-muted-foreground">
          {{ t('needtoken.memberConsole.usage.emptyLogs') }}
        </div>
        <div v-else class="overflow-x-auto">
          <table class="w-full min-w-[640px] text-left text-xs">
            <thead class="border-b bg-muted/30 text-[11px] font-semibold uppercase tracking-wide text-muted-foreground">
              <tr>
                <th class="px-5 py-3">{{ t('needtoken.memberConsole.models.colModel') }}</th>
                <th class="px-4 py-3 text-right">{{ t('needtoken.memberConsole.usage.totalRequests') }}</th>
                <th class="px-4 py-3 text-right">{{ t('needtoken.memberConsole.usage.totalTokens') }}</th>
                <th class="px-4 py-3 text-right">{{ t('needtoken.memberConsole.usage.colCost') }}</th>
                <th class="px-5 py-3">{{ t('needtoken.memberConsole.models.colShare') }}</th>
              </tr>
            </thead>
            <tbody class="divide-y">
              <tr v-for="item in usage.models" :key="item.model_name" class="hover:bg-muted/20">
                <td class="px-5 py-3 font-mono font-semibold text-foreground">{{ item.model_name }}</td>
                <td class="px-4 py-3 text-right font-mono">{{ formatExactInteger(item.requests) }}</td>
                <td class="px-4 py-3 text-right font-mono text-muted-foreground" :title="formatExactInteger(item.token_used)">
                  {{ formatLargeCount(item.token_used, locale) }}
                </td>
                <td class="px-4 py-3 text-right font-mono font-bold text-primary" :title="formatApiQuotaUsdExact(item.quota)">
                  {{ formatApiQuotaUsd(item.quota, { compact: true, locale }) }}
                </td>
                <td class="px-5 py-3">
                  <div class="flex items-center gap-2">
                    <div class="h-2 flex-1 overflow-hidden rounded-full bg-muted">
                      <div class="h-full rounded-full bg-primary" :style="{ width: `${sharePercent(item.quota)}%` }" />
                    </div>
                    <span class="w-10 text-right font-mono text-[11px] text-muted-foreground">{{ sharePercent(item.quota) }}%</span>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { BarChart3, Box, Check, Copy, FlaskConical, RefreshCw, Search } from 'lucide-vue-next'
import { Button } from '@/components/ui/button'
import NeedtokenSelect from '../NeedtokenSelect.vue'
import {
  formatApiQuotaUsd,
  formatApiQuotaUsdExact,
  formatExactInteger,
  formatLargeCount,
} from '../../lib/apiAccountLinkPolicy'
import {
  buildSelectableGroupOptions,
  formatConsoleGroupLabel,
} from '../../lib/gatewayConsoleHelpers'
import {
  getGatewayModels,
  getGatewaySubscriptions,
  getGatewayUsage,
  type GatewayGroupMetaItem,
  type GatewayModelPricingItem,
  type GatewaySubscriptionItem,
  type GatewayUsageResponse,
  type GatewayVendorItem,
} from '../../lib/gatewayConsoleApi'
import {
  buildDynamicPlanMultiplierOptions,
  effectiveDiscountPercent,
  effectivePlanOfferPrice,
  formatUsd,
  inferContextLength,
  inferVendorFromModelName,
  UNCONFIGURED_MODEL_RATIO,
} from '../../lib/pricing'
import { usePricing } from '../../lib/usePricing'
import { historyFor, levelFor, uptimeFor, type PoolLevel } from '../../lib/poolStatus'
import { usePoolStatus } from '../../lib/usePoolStatus'

const { t, locale } = useI18n()
const router = useRouter()
const pricing = usePricing()
const pool = usePoolStatus()

const activeTab = ref<'catalog' | 'analytics'>('catalog')
const modelsLoading = ref(false)
const usageLoading = ref(false)
const errorMsg = ref('')

const models = ref<GatewayModelPricingItem[]>([])
const vendors = ref<GatewayVendorItem[]>([])
const usableGroups = ref<Record<string, string>>({})
const groupRatio = ref<Record<string, number>>({})
const groupMeta = ref<Record<string, GatewayGroupMetaItem>>({})
const defaultGroup = ref('default')
const activeSubs = ref<GatewaySubscriptionItem[]>([])

const searchKeyword = ref('')
const selectedGroup = ref('')
const selectedVendor = ref('')
const billingFilter = ref('all')
const sortOrder = ref<'default' | 'price_asc' | 'price_desc'>('default')
const viewMode = ref<'table' | 'cards'>('table')
const priceMultiplier = ref<number>(1)
const modelPage = ref(1)
const modelPageSize = 20

const analyticsDays = ref(7)
const usage = ref<GatewayUsageResponse>({
  days: 7,
  start_ts: 0,
  end_ts: 0,
  total_quota: 0,
  total_requests: 0,
  total_tokens: 0,
  quota_per_unit: 500_000,
  daily: [],
  models: [],
})

const dayOptions = computed(() => [
  { days: 1, label: t('needtoken.memberConsole.usage.range1d') },
  { days: 7, label: t('needtoken.memberConsole.usage.range7d') },
  { days: 30, label: t('needtoken.memberConsole.usage.range30d') },
])

const selectableGroups = computed(() =>
  buildSelectableGroupOptions({
    usableGroups: usableGroups.value,
    groupMeta: groupMeta.value,
    groupRatio: groupRatio.value,
    pricingModels: models.value,
    defaultGroup: defaultGroup.value,
  }),
)

const allowedGroupCodeSet = computed(() => new Set(selectableGroups.value.map(g => g.code)))

const groupSelectOptions = computed(() => {
  if (!selectableGroups.value.length) {
    const code = defaultGroup.value || 'default'
    return [{ value: code, label: formatConsoleGroupLabel(code, groupMeta.value, usableGroups.value, groupRatio.value) }]
  }
  return selectableGroups.value.map(g => ({ value: g.code, label: g.label }))
})

const billingSelectOptions = computed(() => [
  { value: 'all', label: t('needtoken.memberConsole.models.billingAll') },
  { value: 'token', label: t('needtoken.memberConsole.models.billingToken') },
  { value: 'fixed', label: t('needtoken.memberConsole.models.billingFixed') },
])

const sortSelectOptions = computed(() => [
  { value: 'default', label: t('needtoken.models.sortDefault') },
  { value: 'price_asc', label: t('needtoken.models.sortPriceAsc') },
  { value: 'price_desc', label: t('needtoken.models.sortPriceDesc') },
])

const planMultiplierOptions = computed(() => buildDynamicPlanMultiplierOptions(pricing.activePlans.value))

const activePlanInfo = computed(() => {
  if (!activeSubs.value.length) return null
  const firstSub = activeSubs.value[0]!
  const matchedPlan = pricing.activePlans.value.find(p => p.id === firstSub.plan_id)
  const mult = matchedPlan && Number.isFinite(matchedPlan.multiplier) ? matchedPlan.multiplier : 1
  return {
    title: firstSub.plan_title || matchedPlan?.title || `Plan #${firstSub.plan_id}`,
    multiplier: mult,
  }
})

const formatGroup = (code: string) =>
  formatConsoleGroupLabel(code, groupMeta.value, usableGroups.value, groupRatio.value)

const formatGroupShort = (code: string) => {
  const meta = groupMeta.value[code]
  const ch = meta?.display_name || meta?.channel_names?.[0]
  return ch ? `${code} · ${ch}` : code
}

const activeGroupCode = computed(() => selectedGroup.value || selectableGroups.value[0]?.code || defaultGroup.value || 'default')
const activeGroupDisplay = computed(() => formatGroup(activeGroupCode.value))
const activeGroupRatio = computed(() => {
  const key = activeGroupCode.value
  const metaRatio = groupMeta.value[key]?.ratio
  if (typeof metaRatio === 'number' && Number.isFinite(metaRatio) && metaRatio > 0) return metaRatio
  const val = groupRatio.value[key]
  return typeof val === 'number' && Number.isFinite(val) && val > 0 ? val : 1
})

const vendorMap = computed(() => {
  const m = new Map<number, string>()
  for (const v of vendors.value) m.set(v.id, v.name)
  return m
})

const resolveVendorName = (item: GatewayModelPricingItem): string => {
  if (item.vendor_id && vendorMap.value.has(item.vendor_id)) {
    const mapped = vendorMap.value.get(item.vendor_id)
    if (mapped) return mapped
  }
  return inferVendorFromModelName(item.model_name) || 'Other'
}

const visibleModelGroups = (item: GatewayModelPricingItem): string[] => {
  const egs = Array.isArray(item.enable_groups) ? item.enable_groups : []
  if (egs.includes('all')) {
    return activeGroupCode.value ? [activeGroupCode.value] : []
  }
  const authorized = egs.filter(g => allowedGroupCodeSet.value.has(g))
  if (authorized.length) return authorized
  return activeGroupCode.value ? [activeGroupCode.value] : egs
}

const groupScopedModels = computed(() => {
  const grp = activeGroupCode.value
  const metaModels = new Set(groupMeta.value[grp]?.models || [])
  return models.value.filter((item) => {
    if (!grp) return true
    const egs = Array.isArray(item.enable_groups) ? item.enable_groups : []
    const inPricing = egs.length === 0 || egs.includes('all') || egs.includes(grp)
    const inMeta = metaModels.size > 0 && metaModels.has(item.model_name)
    return inPricing || inMeta
  })
})

const hasPerCallModels = computed(() => groupScopedModels.value.some(m => m.quota_type === 1))

const PREFERRED_VENDOR_ORDER = ['OpenAI', 'Anthropic', 'Google', 'DeepSeek', 'xAI', 'Qwen', 'Zhipu', 'Moonshot', 'Meta', 'Mistral']

const vendorStats = computed(() => {
  const counts = new Map<string, number>()
  for (const item of groupScopedModels.value) {
    const v = resolveVendorName(item)
    counts.set(v, (counts.get(v) || 0) + 1)
  }
  return [...counts.entries()]
    .map(([name, count]) => ({ name, count }))
    .sort((a, b) => {
      const rankA = PREFERRED_VENDOR_ORDER.indexOf(a.name)
      const rankB = PREFERRED_VENDOR_ORDER.indexOf(b.name)
      if (rankA !== -1 && rankB !== -1) return rankA - rankB
      if (rankA !== -1) return -1
      if (rankB !== -1) return 1
      return b.count - a.count || a.name.localeCompare(b.name)
    })
})

const publicOfferMap = computed(() => {
  const map = new Map<string, { officialInputPrice: number | null; officialOutputPrice: number | null; contextLength: string | null }>()
  for (const o of pricing.offers.value) {
    if (!map.has(o.modelId)) {
      map.set(o.modelId, {
        officialInputPrice: o.officialInputPrice,
        officialOutputPrice: o.officialOutputPrice,
        contextLength: o.contextLength,
      })
    }
  }
  return map
})

const dotClassMap: Record<PoolLevel, string> = {
  available: 'bg-emerald-500',
  busy: 'bg-amber-500',
  unavailable: 'bg-red-500',
  unknown: 'bg-muted-foreground/45',
}

const badgeClassMap: Record<PoolLevel, string> = {
  available: 'border-emerald-500/40 bg-emerald-500/10 text-emerald-700 dark:text-emerald-300',
  busy: 'border-amber-500/45 bg-amber-500/10 text-amber-700 dark:text-amber-300',
  unavailable: 'border-red-500/45 bg-red-500/10 text-red-700 dark:text-red-300',
  unknown: 'border-border bg-muted/40 text-muted-foreground',
}

interface ConsoleModelRow {
  modelName: string
  vendor: string
  contextLength: string | null
  isUnpriced: boolean
  isPerCall: boolean
  needInputPrice: number | null
  needOutputPrice: number | null
  officialInputPrice: number | null
  officialOutputPrice: number | null
  discountPercent: number | null
  groups: string[]
  uptimePct: number | null
  uptimeHistory: PoolLevel[]
  availabilityLabel: string
  availabilityClass: string
  dotClass: string
}

const enrichedModels = computed<ConsoleModelRow[]>(() => {
  const grp = activeGroupCode.value
  const grpMeta = groupMeta.value[grp]
  const hasEnabledChannel = grpMeta ? grpMeta.enabled_channel_count > 0 : true
  const poolLevel = levelFor(pool.view.value, grp)
  const uptimePct = uptimeFor(pool.view.value, grp)
  const uptimeHistory = historyFor(pool.view.value, grp)
  const overrides = pricing.meta.value?.official_overrides || {}

  return groupScopedModels.value.map((item) => {
    const isPerCall = item.quota_type === 1
    const isUnpriced = !isPerCall && (!Number.isFinite(item.model_ratio) || item.model_ratio <= 0 || item.model_ratio === UNCONFIGURED_MODEL_RATIO)

    const baseInput = isUnpriced ? null : (isPerCall ? item.model_price * activeGroupRatio.value : item.model_ratio * 2 * activeGroupRatio.value)
    const baseOutput = (isUnpriced || isPerCall || baseInput === null) ? null : baseInput * (item.completion_ratio || 1)
    const needInputPrice = effectivePlanOfferPrice(baseInput, priceMultiplier.value)
    const needOutputPrice = effectivePlanOfferPrice(baseOutput, priceMultiplier.value)

    const ov = overrides[item.model_name]
    const pub = publicOfferMap.value.get(item.model_name)
    let officialInputPrice: number | null = null
    let officialOutputPrice: number | null = null
    if (!isUnpriced) {
      if (typeof ov?.input_usd === 'number' && Number.isFinite(ov.input_usd) && ov.input_usd > 0) {
        officialInputPrice = ov.input_usd
      } else if (pub?.officialInputPrice != null) {
        officialInputPrice = pub.officialInputPrice
      } else if (activeGroupRatio.value < 1 || priceMultiplier.value > 1) {
        officialInputPrice = isPerCall ? item.model_price : item.model_ratio * 2
      }

      if (!isPerCall) {
        if (typeof ov?.output_usd === 'number' && Number.isFinite(ov.output_usd) && ov.output_usd > 0) {
          officialOutputPrice = ov.output_usd
        } else if (pub?.officialOutputPrice != null) {
          officialOutputPrice = pub.officialOutputPrice
        } else if (officialInputPrice !== null) {
          officialOutputPrice = officialInputPrice * (item.completion_ratio || 1)
        }
      }
    }

    const discountPercent = effectiveDiscountPercent(officialInputPrice, needInputPrice)
    const contextLength = (ov?.context_length && ov.context_length.trim()) || pub?.contextLength || inferContextLength(item.model_name)

    let effectiveLevel: PoolLevel = 'available'
    let availabilityLabel = t('needtoken.memberConsole.models.availableStatus')
    if (!hasEnabledChannel || poolLevel === 'unavailable') {
      effectiveLevel = 'unavailable'
      availabilityLabel = t('needtoken.memberConsole.models.unavailableStatus')
    } else if (poolLevel === 'busy') {
      effectiveLevel = 'busy'
      availabilityLabel = t('needtoken.memberConsole.models.busyStatus')
    }

    return {
      modelName: item.model_name,
      vendor: resolveVendorName(item),
      contextLength,
      isUnpriced,
      isPerCall,
      needInputPrice,
      needOutputPrice,
      officialInputPrice,
      officialOutputPrice,
      discountPercent,
      groups: visibleModelGroups(item),
      uptimePct,
      uptimeHistory,
      availabilityLabel,
      availabilityClass: badgeClassMap[effectiveLevel],
      dotClass: dotClassMap[effectiveLevel],
    }
  })
})

const filteredModels = computed(() => {
  const kw = searchKeyword.value.toLowerCase()
  const list = enrichedModels.value.filter((row) => {
    if (selectedVendor.value && row.vendor !== selectedVendor.value) return false
    if (billingFilter.value === 'token' && row.isPerCall) return false
    if (billingFilter.value === 'fixed' && !row.isPerCall) return false
    if (!kw) return true
    const grpLabels = row.groups.map(formatGroupShort).join(' ').toLowerCase()
    return (
      row.modelName.toLowerCase().includes(kw)
      || row.vendor.toLowerCase().includes(kw)
      || grpLabels.includes(kw)
    )
  })

  if (sortOrder.value === 'default') return list
  return [...list].sort((a, b) => {
    if (a.isUnpriced !== b.isUnpriced) return a.isUnpriced ? 1 : -1
    const pa = a.needInputPrice ?? Number.MAX_SAFE_INTEGER
    const pb = b.needInputPrice ?? Number.MAX_SAFE_INTEGER
    return sortOrder.value === 'price_asc' ? pa - pb : pb - pa
  })
})

watch([searchKeyword, selectedGroup, selectedVendor, billingFilter, sortOrder], () => {
  modelPage.value = 1
})

const totalModelPages = computed(() => Math.max(1, Math.ceil(filteredModels.value.length / modelPageSize)))
const pagedModels = computed(() => {
  const start = (modelPage.value - 1) * modelPageSize
  return filteredModels.value.slice(start, start + modelPageSize)
})

const sharePercent = (quota: number) => {
  if (!usage.value.total_quota || usage.value.total_quota <= 0) return 0
  return Math.max(1, Math.min(100, Math.round((quota / usage.value.total_quota) * 100)))
}

const copiedModel = ref('')
let copyTimer: ReturnType<typeof setTimeout> | undefined
const copyModelId = async (modelId: string) => {
  if (copyTimer) clearTimeout(copyTimer)
  try {
    await navigator.clipboard.writeText(modelId)
    copiedModel.value = modelId
    copyTimer = setTimeout(() => { copiedModel.value = '' }, 1500)
  } catch {
    copiedModel.value = ''
  }
}

const tryInPlayground = (modelId: string) => {
  void router.push({ path: '/me/playground', query: { model: modelId } })
}

const loadModels = async () => {
  modelsLoading.value = true
  errorMsg.value = ''
  try {
    const [res, subsRes] = await Promise.all([
      getGatewayModels(),
      getGatewaySubscriptions().catch(() => null),
      pricing.ensureLoaded().catch(() => null),
      pool.ensureLoaded().catch(() => null),
    ])
    models.value = Array.isArray(res.models) ? res.models : []
    vendors.value = Array.isArray(res.vendors) ? res.vendors : []
    usableGroups.value = res.usable_groups || {}
    groupRatio.value = res.group_ratio || {}
    groupMeta.value = res.group_meta || {}
    defaultGroup.value = res.default_group || 'default'
    if (subsRes && Array.isArray(subsRes.active_subscriptions)) {
      activeSubs.value = subsRes.active_subscriptions
      const activePlanMatch = planMultiplierOptions.value.find(p => subsRes.active_subscriptions.some(s => s.plan_id === p.planId))
      if (activePlanMatch && priceMultiplier.value === 1) {
        priceMultiplier.value = activePlanMatch.multiplier
      }
    }
    const available = selectableGroups.value
    if (!selectedGroup.value || !available.some(g => g.code === selectedGroup.value)) {
      selectedGroup.value = available[0]?.code || defaultGroup.value
    }
  } catch (err: any) {
    errorMsg.value = err?.response?.data?.msg || t('needtoken.memberConsole.loadFailed')
  } finally {
    modelsLoading.value = false
  }
}

const loadAnalytics = async () => {
  usageLoading.value = true
  try {
    usage.value = await getGatewayUsage(analyticsDays.value)
  } catch {
    // keep previous data
  } finally {
    usageLoading.value = false
  }
}

const switchToAnalytics = () => {
  activeTab.value = 'analytics'
  if (!usage.value.models.length) {
    void loadAnalytics()
  }
}

const changeAnalyticsDays = (days: number) => {
  analyticsDays.value = days
  void loadAnalytics()
}

onMounted(() => {
  void loadModels()
})

onUnmounted(() => {
  if (copyTimer) clearTimeout(copyTimer)
})
</script>
