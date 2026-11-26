<template>
  <div class="space-y-5">
    <!-- 接口地址快捷条 + 顶部动作 -->
    <div class="rounded-2xl border bg-card p-5 shadow-sm">
      <div class="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h2 class="text-lg font-bold text-foreground">{{ t('needtoken.memberConsole.keys.title') }}</h2>
          <p class="mt-1 text-xs text-muted-foreground">{{ t('needtoken.memberConsole.keys.subtitle') }}</p>
        </div>
        <div class="flex flex-wrap items-center gap-2">
          <Button variant="outline" size="sm" class="border-primary/40 text-primary hover:bg-primary/5" @click="emit('open-quick-use')">
            <Zap class="mr-1.5 h-3.5 w-3.5" />
            {{ t('needtoken.memberConsole.keys.quickUse') }}
          </Button>
          <Button variant="outline" size="sm" :disabled="loading" @click="loadTokens">
            <RefreshCw class="mr-1.5 h-3.5 w-3.5" :class="loading ? 'animate-spin' : ''" />
            {{ t('needtoken.common.refresh') }}
          </Button>
          <Button size="sm" @click="openCreateModal">
            <Plus class="mr-1.5 h-4 w-4" />
            {{ t('needtoken.memberConsole.keys.createButton') }}
          </Button>
        </div>
      </div>

      <div class="mt-4 flex flex-wrap items-center justify-between gap-2 rounded-xl border bg-muted/40 px-3.5 py-2.5 text-xs">
        <div class="flex min-w-0 items-center gap-2">
          <span class="shrink-0 font-semibold text-muted-foreground">Base URL</span>
          <code class="truncate font-mono font-semibold text-foreground">{{ API_BASE_URL }}</code>
        </div>
        <div class="flex shrink-0 items-center gap-1.5">
          <button
            type="button"
            class="inline-flex items-center gap-1 rounded-md border bg-background px-2.5 py-1 text-xs font-medium text-foreground transition-colors hover:bg-accent"
            @click="copyText(API_BASE_URL, 'endpoint')"
          >
            <Check v-if="copiedKey === 'endpoint'" class="h-3.5 w-3.5 text-primary" />
            <Copy v-else class="h-3.5 w-3.5" />
            <span>{{ copiedKey === 'endpoint' ? t('needtoken.common.copied') : t('needtoken.common.copy') }}</span>
          </button>
        </div>
      </div>
    </div>

    <!-- 新建密钥一次性高亮提示条 -->
    <div v-if="newlyCreatedKey" class="rounded-2xl border border-primary/35 bg-primary/5 p-4 shadow-sm">
      <div class="flex flex-wrap items-center justify-between gap-3">
        <div class="min-w-0">
          <p class="text-xs font-semibold text-primary">{{ t('needtoken.memberConsole.keys.createdNotice') }}</p>
          <code class="mt-1 block break-all font-mono text-sm font-semibold text-foreground">{{ newlyCreatedKey }}</code>
        </div>
        <div class="flex items-center gap-2">
          <Button variant="outline" size="sm" @click="emit('open-quick-use', newlyCreatedTokenId ?? undefined)">
            <Zap class="mr-1.5 h-3.5 w-3.5 text-primary" />
            {{ t('needtoken.memberConsole.keys.quickUse') }}
          </Button>
          <Button size="sm" @click="copyText(newlyCreatedKey, 'new-key')">
            <Check v-if="copiedKey === 'new-key'" class="mr-1.5 h-3.5 w-3.5" />
            <Copy v-else class="mr-1.5 h-3.5 w-3.5" />
            {{ copiedKey === 'new-key' ? t('needtoken.common.copied') : t('needtoken.common.copy') }}
          </Button>
          <Button variant="ghost" size="sm" @click="newlyCreatedKey = ''; newlyCreatedTokenId = null">
            <X class="h-4 w-4" />
          </Button>
        </div>
      </div>
    </div>

    <!-- 错误提示 -->
    <div v-if="errorMsg" class="rounded-xl border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive">
      {{ errorMsg }}
    </div>

    <!-- 密钥列表 -->
    <div class="rounded-2xl border bg-card shadow-sm">
      <div class="flex flex-wrap items-center justify-between gap-3 border-b px-5 py-3.5">
        <div class="flex flex-wrap items-center gap-2">
          <div class="relative w-full min-w-[220px] sm:w-64">
            <Search class="pointer-events-none absolute left-3 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-muted-foreground" />
            <input
              v-model="keyword"
              type="text"
              :placeholder="t('needtoken.memberConsole.keys.searchPlaceholder')"
              class="h-9 w-full rounded-xl border bg-background pl-8 pr-3 text-xs text-foreground outline-none focus:border-primary"
            />
          </div>
          <NeedtokenSelect
            v-if="groupFilterOptions.length > 1"
            v-model="filterGroup"
            :options="groupFilterOptions"
            :label="t('needtoken.memberConsole.keys.allGroupsFilter')"
            compact
            align="start"
            class="min-w-[180px]"
          />
        </div>
        <span class="text-xs text-muted-foreground">
          {{ t('needtoken.memberConsole.keys.countSummary', { n: filteredTokens.length }) }}
        </span>
      </div>

      <div v-if="loading && !tokens.length" class="space-y-3 p-5">
        <div v-for="idx in 3" :key="idx" class="h-14 animate-pulse rounded-xl bg-muted" />
      </div>

      <div v-else-if="!filteredTokens.length" class="px-6 py-12 text-center">
        <KeyRound class="mx-auto h-9 w-9 text-muted-foreground/50" />
        <p class="mt-3 text-sm font-semibold text-foreground">{{ t('needtoken.memberConsole.keys.emptyTitle') }}</p>
        <p class="mt-1 text-xs text-muted-foreground">{{ t('needtoken.memberConsole.keys.emptyHint') }}</p>
        <Button class="mt-4" size="sm" @click="openCreateModal">
          <Plus class="mr-1.5 h-4 w-4" />
          {{ t('needtoken.memberConsole.keys.createButton') }}
        </Button>
      </div>

      <div v-else class="divide-y">
        <div
          v-for="tok in filteredTokens"
          :key="tok.id"
          class="flex flex-col gap-3 px-5 py-4 transition-colors hover:bg-muted/20 lg:flex-row lg:items-center lg:justify-between"
        >
          <div class="min-w-0 flex-1 space-y-1.5">
            <div class="flex flex-wrap items-center gap-1.5">
              <span class="truncate text-sm font-bold text-foreground">{{ tok.name || `Key #${tok.id}` }}</span>
              <Badge :variant="tok.status === 1 ? 'success' : 'secondary'" size="sm">
                {{ tokenStatusLabel(tok.status) }}
              </Badge>
              <span
                v-for="badge in tokenGroupBadges(tok)"
                :key="badge"
                class="rounded-md bg-muted px-2 py-0.5 text-[11px] font-medium text-foreground/80"
              >
                {{ badge }}
              </span>
              <span
                v-if="tok.model_limits_enabled && countTokenModels(tok) > 0"
                class="rounded-md border border-amber-500/30 bg-amber-500/10 px-2 py-0.5 text-[11px] font-medium text-amber-700 dark:text-amber-300"
              >
                {{ t('needtoken.memberConsole.keys.modelLockedCount', { n: countTokenModels(tok) }) }}
              </span>
              <span
                v-else
                class="rounded-md border border-primary/25 bg-primary/5 px-2 py-0.5 text-[11px] font-medium text-primary"
              >
                {{ t('needtoken.memberConsole.keys.modelAutoFollow') }}
              </span>
            </div>

            <div class="flex flex-wrap items-center gap-2 font-mono text-xs text-muted-foreground">
              <span class="max-w-full truncate rounded bg-muted/60 px-2 py-0.5 text-foreground">
                {{ revealedIds.has(tok.id) ? tok.key : tok.masked_key }}
              </span>
              <button
                type="button"
                class="inline-flex items-center gap-1 rounded px-1.5 py-0.5 text-xs hover:bg-accent hover:text-foreground"
                :aria-label="revealedIds.has(tok.id) ? t('needtoken.memberConsole.keys.hideKey') : t('needtoken.memberConsole.keys.showKey')"
                @click="toggleReveal(tok.id)"
              >
                <EyeOff v-if="revealedIds.has(tok.id)" class="h-3.5 w-3.5" />
                <Eye v-else class="h-3.5 w-3.5" />
              </button>
              <button
                type="button"
                class="inline-flex items-center gap-1 rounded px-1.5 py-0.5 text-xs hover:bg-accent hover:text-foreground"
                @click="copyText(tok.key, `tok-${tok.id}`)"
              >
                <Check v-if="copiedKey === `tok-${tok.id}`" class="h-3.5 w-3.5 text-primary" />
                <Copy v-else class="h-3.5 w-3.5" />
                <span>{{ copiedKey === `tok-${tok.id}` ? t('needtoken.common.copied') : t('needtoken.common.copy') }}</span>
              </button>
            </div>

            <div class="flex flex-wrap items-center gap-x-4 gap-y-1 text-[11.5px] text-muted-foreground">
              <span>
                {{ t('needtoken.memberConsole.keys.usedLabel') }}:
                <strong class="font-mono text-foreground">{{ formatApiQuotaUsd(tok.used_quota) }}</strong>
              </span>
              <span>
                {{ t('needtoken.memberConsole.keys.quotaLabel') }}:
                <strong class="font-mono text-foreground">
                  {{ tok.unlimited_quota ? t('needtoken.memberConsole.keys.unlimited') : formatApiQuotaUsd(tok.remain_quota) }}
                </strong>
              </span>
              <span>
                {{ t('needtoken.memberConsole.keys.expiryLabel') }}:
                {{ tok.expired_time > 0 ? formatUnixDate(tok.expired_time) : t('needtoken.memberConsole.keys.neverExpire') }}
              </span>
            </div>
          </div>

          <div class="flex shrink-0 flex-wrap items-center gap-2">
            <Button
              v-if="tok.status === 1"
              variant="outline"
              size="sm"
              class="border-primary/30 text-primary hover:bg-primary/5"
              @click="emit('open-quick-use', tok.id)"
            >
              <Zap class="mr-1 h-3.5 w-3.5" />
              {{ t('needtoken.memberConsole.keys.quickUse') }}
            </Button>
            <Button
              variant="outline"
              size="sm"
              :disabled="busyId === tok.id"
              @click="toggleStatus(tok)"
            >
              {{ tok.status === 1 ? t('needtoken.memberConsole.keys.disable') : t('needtoken.memberConsole.keys.enable') }}
            </Button>
            <Button variant="outline" size="sm" :disabled="busyId === tok.id" @click="openEditModal(tok)">
              <Pencil class="mr-1 h-3.5 w-3.5" />
              {{ t('needtoken.memberConsole.keys.edit') }}
            </Button>
            <Button
              v-if="confirmDeleteId !== tok.id"
              variant="ghost"
              size="sm"
              class="text-destructive hover:bg-destructive/10 hover:text-destructive"
              :disabled="busyId === tok.id"
              @click="confirmDeleteId = tok.id"
            >
              <Trash2 class="h-3.5 w-3.5" />
            </Button>
            <div v-else class="flex items-center gap-1">
              <Button variant="destructive" size="sm" :disabled="busyId === tok.id" @click="confirmDelete(tok)">
                {{ t('needtoken.memberConsole.keys.confirmDelete') }}
              </Button>
              <Button variant="ghost" size="sm" @click="confirmDeleteId = null">
                {{ t('needtoken.memberConsole.keys.cancel') }}
              </Button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 创建 / 编辑弹窗 -->
    <div
      v-if="modalOpen"
      class="fixed inset-0 z-[80] flex items-center justify-center bg-black/50 p-4 backdrop-blur-xs"
      role="dialog"
      aria-modal="true"
      @click.self="closeModal"
    >
      <form class="flex max-h-[90dvh] w-full max-w-2xl flex-col overflow-hidden rounded-2xl border bg-card shadow-xl" @submit.prevent="submitModal">
        <div class="flex shrink-0 items-center justify-between border-b px-6 py-4">
          <h3 class="text-base font-bold text-foreground">
            {{ editingToken ? t('needtoken.memberConsole.keys.editTitle') : t('needtoken.memberConsole.keys.createTitle') }}
          </h3>
          <button type="button" class="rounded-md p-1 text-muted-foreground hover:bg-accent" @click="closeModal">
            <X class="h-4 w-4" />
          </button>
        </div>

        <div class="flex-1 space-y-4 overflow-y-auto px-6 py-5">
          <div v-if="formError" class="rounded-xl border border-destructive/30 bg-destructive/5 px-3.5 py-2.5 text-xs text-destructive">
            {{ formError }}
          </div>

          <label class="block space-y-1.5 text-xs">
            <span class="font-semibold text-foreground">{{ t('needtoken.memberConsole.keys.nameLabel') }} *</span>
            <input
              v-model="form.name"
              type="text"
              maxlength="50"
              required
              :placeholder="t('needtoken.memberConsole.keys.namePlaceholder')"
              class="h-10 w-full rounded-xl border bg-background px-3 text-sm text-foreground outline-none focus:border-primary"
            />
          </label>

          <!-- 绑定渠道分组（支持单选或多选，隐藏无活跃渠道的空分组） -->
          <div class="space-y-2 text-xs">
            <div class="flex items-center justify-between gap-2">
              <span class="font-semibold text-foreground">{{ t('needtoken.memberConsole.keys.groupLabel') }} *</span>
              <button
                v-if="modalGroupOptions.length > 1"
                type="button"
                class="text-[11px] font-semibold text-primary hover:underline"
                @click="selectAllModalGroups"
              >
                {{ t('needtoken.memberConsole.keys.groupSelectAll') }}
              </button>
            </div>
            <p class="text-[11px] text-muted-foreground">
              {{ modalGroupOptions.length <= 1 ? t('needtoken.memberConsole.keys.singleGroupHint') : t('needtoken.memberConsole.keys.groupSelectHint') }}
            </p>
            <div class="grid grid-cols-1 gap-2 sm:grid-cols-2">
              <button
                v-for="opt in modalGroupOptions"
                :key="opt.code"
                type="button"
                class="flex items-center justify-between gap-2 rounded-xl border px-3 py-2.5 text-left transition-colors"
                :class="form.selectedGroups.includes(opt.code)
                  ? 'border-primary bg-primary/8 text-foreground'
                  : 'border-border bg-background text-muted-foreground hover:border-primary/40 hover:text-foreground'"
                @click="toggleGroupSelection(opt.code)"
              >
                <div class="flex min-w-0 items-center gap-2">
                  <span
                    class="grid h-4 w-4 shrink-0 place-items-center rounded border text-[10px]"
                    :class="form.selectedGroups.includes(opt.code)
                      ? 'border-primary bg-primary text-primary-foreground'
                      : 'border-muted-foreground/40 bg-background'"
                  >
                    <Check v-if="form.selectedGroups.includes(opt.code)" class="h-3 w-3" />
                  </span>
                  <span class="truncate text-xs font-semibold">{{ opt.label }}</span>
                </div>
                <div class="flex shrink-0 items-center gap-1">
                  <span v-if="!opt.hasActiveChannels" class="rounded bg-muted px-1.5 py-0.5 text-[10px] text-muted-foreground">
                    {{ t('needtoken.memberConsole.keys.groupInactiveTag') }}
                  </span>
                  <span v-else-if="opt.models.length > 0" class="rounded bg-muted/80 px-1.5 py-0.5 font-mono text-[10px] text-muted-foreground">
                    {{ t('needtoken.memberConsole.keys.groupModelCount', { n: opt.models.length }) }}
                  </span>
                </div>
              </button>
            </div>
          </div>

          <!-- 额度与有效期 -->
          <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
            <div class="space-y-2 rounded-xl border bg-muted/25 p-3.5 text-xs">
              <label class="flex cursor-pointer items-center gap-2 font-semibold text-foreground">
                <input v-model="form.unlimitedQuota" type="checkbox" class="h-4 w-4 accent-primary" />
                <span>{{ t('needtoken.memberConsole.keys.unlimitedToggle') }}</span>
              </label>
              <div v-if="!form.unlimitedQuota" class="pt-1">
                <label class="block space-y-1">
                  <span class="text-muted-foreground">{{ t('needtoken.memberConsole.keys.quotaUsdLabel') }}</span>
                  <input
                    v-model="form.quotaUsd"
                    type="number"
                    min="0"
                    step="0.01"
                    placeholder="10.00"
                    class="h-9 w-full rounded-lg border bg-background px-3 text-sm text-foreground outline-none focus:border-primary"
                  />
                </label>
              </div>
            </div>

            <div class="space-y-1.5 rounded-xl border bg-muted/25 p-3.5 text-xs">
              <span class="block font-semibold text-foreground">{{ t('needtoken.memberConsole.keys.expireDaysLabel') }}</span>
              <NeedtokenSelect
                v-model="form.expirePreset"
                :options="expirePresetOptions"
                :label="t('needtoken.memberConsole.keys.expireDaysLabel')"
                align="start"
                class="w-full"
              />
            </div>
          </div>

          <!-- 可用模型策略：跟随分组自动追加 vs 锁定列表 -->
          <div class="space-y-3 rounded-xl border bg-muted/20 p-4 text-xs">
            <span class="block font-semibold text-foreground">{{ t('needtoken.memberConsole.keys.modelStrategyLabel') }}</span>
            <div class="grid grid-cols-1 gap-2 sm:grid-cols-2">
              <button
                type="button"
                class="rounded-xl border p-3 text-left transition-colors"
                :class="form.modelStrategy === 'follow'
                  ? 'border-primary bg-primary/8'
                  : 'border-border bg-background hover:border-primary/40'"
                @click="form.modelStrategy = 'follow'"
              >
                <div class="flex items-center gap-2 font-semibold text-foreground">
                  <span
                    class="h-3.5 w-3.5 shrink-0 rounded-full border-4"
                    :class="form.modelStrategy === 'follow' ? 'border-primary bg-background' : 'border-muted-foreground/35 bg-transparent'"
                  />
                  <span>{{ t('needtoken.memberConsole.keys.modelStrategyFollowTitle') }}</span>
                </div>
                <p class="mt-1 pl-5 text-[11px] leading-relaxed text-muted-foreground">
                  {{ t('needtoken.memberConsole.keys.modelStrategyFollowDesc') }}
                </p>
              </button>

              <button
                type="button"
                class="rounded-xl border p-3 text-left transition-colors"
                :class="form.modelStrategy === 'lock'
                  ? 'border-primary bg-primary/8'
                  : 'border-border bg-background hover:border-primary/40'"
                @click="form.modelStrategy = 'lock'"
              >
                <div class="flex items-center gap-2 font-semibold text-foreground">
                  <span
                    class="h-3.5 w-3.5 shrink-0 rounded-full border-4"
                    :class="form.modelStrategy === 'lock' ? 'border-primary bg-background' : 'border-muted-foreground/35 bg-transparent'"
                  />
                  <span>{{ t('needtoken.memberConsole.keys.modelStrategyLockTitle') }}</span>
                </div>
                <p class="mt-1 pl-5 text-[11px] leading-relaxed text-muted-foreground">
                  {{ t('needtoken.memberConsole.keys.modelStrategyLockDesc') }}
                </p>
              </button>
            </div>

            <!-- 锁定模式：动态模型目录多选面板 -->
            <div v-if="form.modelStrategy === 'lock'" class="space-y-3 border-t pt-3">
              <div class="flex flex-wrap items-center justify-between gap-2">
                <div class="relative min-w-[180px] flex-1">
                  <Search class="pointer-events-none absolute left-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-muted-foreground" />
                  <input
                    v-model.trim="modelSearchQuery"
                    type="text"
                    :placeholder="t('needtoken.memberConsole.keys.modelSearchPlaceholder')"
                    class="h-8 w-full rounded-lg border bg-background pl-8 pr-3 font-mono text-xs text-foreground outline-none focus:border-primary"
                  />
                </div>
                <div class="flex items-center gap-1.5 text-[11px]">
                  <span class="font-mono text-muted-foreground">
                    {{ t('needtoken.memberConsole.keys.modelSelectedCount', { selected: form.selectedModels.length, total: availableCatalogModels.length }) }}
                  </span>
                  <button
                    type="button"
                    class="rounded-md border bg-background px-2 py-1 font-medium text-foreground hover:border-primary/40 hover:text-primary"
                    @click="selectAllVisibleModels"
                  >
                    {{ t('needtoken.memberConsole.keys.modelSelectAllVisible') }}
                  </button>
                  <button
                    type="button"
                    class="rounded-md border bg-background px-2 py-1 font-medium text-muted-foreground hover:text-foreground"
                    @click="form.selectedModels = []"
                  >
                    {{ t('needtoken.memberConsole.keys.modelClearSelected') }}
                  </button>
                </div>
              </div>

              <!-- 厂商快速筛选胶囊 -->
              <div v-if="modalVendorTabs.length > 2" class="flex flex-wrap gap-1.5">
                <button
                  v-for="tab in modalVendorTabs"
                  :key="tab.key"
                  type="button"
                  class="rounded-full border px-2.5 py-0.5 text-[11px] font-semibold transition-colors"
                  :class="modelVendorFilter === tab.key
                    ? 'border-primary bg-primary/10 text-primary'
                    : 'border-border bg-background text-muted-foreground hover:text-foreground'"
                  @click="modelVendorFilter = tab.key"
                >
                  {{ tab.label }}
                </button>
              </div>

              <!-- 多分组时按渠道分组快速筛选 -->
              <div v-if="form.selectedGroups.length > 1" class="flex flex-wrap gap-1.5">
                <button
                  type="button"
                  class="rounded-full border px-2.5 py-0.5 text-[11px] font-medium transition-colors"
                  :class="modelGroupFilter === 'all'
                    ? 'border-primary bg-primary/10 text-primary'
                    : 'border-border bg-background text-muted-foreground hover:text-foreground'"
                  @click="modelGroupFilter = 'all'"
                >
                  {{ t('needtoken.memberConsole.keys.modelAllSelectedGroups') }}
                </button>
                <button
                  v-for="grp in form.selectedGroups"
                  :key="grp"
                  type="button"
                  class="rounded-full border px-2.5 py-0.5 text-[11px] font-medium transition-colors"
                  :class="modelGroupFilter === grp
                    ? 'border-primary bg-primary/10 text-primary'
                    : 'border-border bg-background text-muted-foreground hover:text-foreground'"
                  @click="modelGroupFilter = grp"
                >
                  {{ formatGroup(grp) }}
                </button>
              </div>

              <!-- 模型标签勾选区 -->
              <div v-if="displayedModelChoices.length" class="max-h-52 overflow-y-auto rounded-xl border bg-background p-2.5">
                <div class="flex flex-wrap gap-1.5">
                  <button
                    v-for="item in displayedModelChoices"
                    :key="item.model"
                    type="button"
                    class="inline-flex items-center gap-1 rounded-lg border px-2.5 py-1 font-mono text-[11.5px] transition-colors"
                    :class="form.selectedModels.includes(item.model)
                      ? 'border-primary bg-primary/12 font-semibold text-primary'
                      : 'border-border bg-muted/25 text-foreground/85 hover:border-primary/40'"
                    @click="toggleModelSelection(item.model)"
                  >
                    <Check v-if="form.selectedModels.includes(item.model)" class="h-3 w-3 shrink-0" />
                    <span>{{ item.model }}</span>
                  </button>
                </div>
              </div>
              <div v-else class="rounded-xl border border-dashed bg-background px-4 py-5 text-center text-xs text-muted-foreground">
                {{ t('needtoken.memberConsole.keys.modelEmptyCatalog') }}
              </div>

              <!-- 可选手动追加自定义模型 ID -->
              <div class="flex items-center gap-2">
                <input
                  v-model.trim="customModelDraft"
                  type="text"
                  :placeholder="t('needtoken.memberConsole.keys.modelCustomInputPlaceholder')"
                  class="h-8 flex-1 rounded-lg border bg-background px-2.5 font-mono text-xs text-foreground outline-none focus:border-primary"
                  @keydown.enter.prevent="addCustomModel"
                />
                <Button type="button" variant="outline" size="sm" class="h-8 px-2.5 text-xs" :disabled="!customModelDraft" @click="addCustomModel">
                  <Plus class="h-3.5 w-3.5" />
                </Button>
              </div>
            </div>
          </div>
        </div>

        <div class="flex shrink-0 justify-end gap-2 border-t px-6 py-4">
          <Button type="button" variant="outline" size="sm" :disabled="saving" @click="closeModal">
            {{ t('needtoken.memberConsole.keys.cancel') }}
          </Button>
          <Button type="submit" size="sm" :disabled="saving">
            {{ saving ? t('needtoken.common.loading') : (editingToken ? t('needtoken.memberConsole.keys.save') : t('needtoken.memberConsole.keys.createButton')) }}
          </Button>
        </div>
      </form>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { Check, Copy, Eye, EyeOff, KeyRound, Pencil, Plus, RefreshCw, Search, Trash2, X, Zap } from 'lucide-vue-next'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import NeedtokenSelect from '../NeedtokenSelect.vue'
import { API_BASE_URL } from '../../config'
import { formatApiQuotaUsd } from '../../lib/apiAccountLinkPolicy'
import {
  buildSelectableGroupOptions,
  classifyConsoleModelVendor,
  collectModelsForConsoleGroups,
  formatConsoleGroupLabel,
  formatTokenGroupsSummary,
  type ConsoleModelVendorKey,
} from '../../lib/gatewayConsoleHelpers'
import {
  createGatewayToken,
  deleteGatewayToken,
  getGatewayModels,
  listGatewayTokens,
  rawQuotaToUsdNumber,
  updateGatewayToken,
  usdToRawQuota,
  type GatewayGroupMetaItem,
  type GatewayModelPricingItem,
  type GatewayTokenItem,
} from '../../lib/gatewayConsoleApi'

const emit = defineEmits<{
  (e: 'changed'): void
  (e: 'open-quick-use', tokenId?: number): void
}>()

const { t, locale } = useI18n()
const loading = ref(false)
const saving = ref(false)
const busyId = ref<number | null>(null)
const errorMsg = ref('')
const formError = ref('')
const keyword = ref('')
const filterGroup = ref('')

const tokens = ref<GatewayTokenItem[]>([])
const usableGroups = ref<Record<string, string>>({})
const groupRatio = ref<Record<string, number>>({})
const groupMeta = ref<Record<string, GatewayGroupMetaItem>>({})
const pricingModels = ref<GatewayModelPricingItem[]>([])
const defaultGroup = ref('default')
const userAutoGroups = ref<string[]>([])

const revealedIds = ref<Set<number>>(new Set())
const copiedKey = ref('')
const newlyCreatedKey = ref('')
const newlyCreatedTokenId = ref<number | null>(null)
const confirmDeleteId = ref<number | null>(null)
const modalOpen = ref(false)
const editingToken = ref<GatewayTokenItem | null>(null)

const modelSearchQuery = ref('')
const modelVendorFilter = ref<'all' | ConsoleModelVendorKey>('all')
const modelGroupFilter = ref('all')
const customModelDraft = ref('')

const form = reactive({
  name: '',
  selectedGroups: [] as string[],
  unlimitedQuota: true,
  quotaUsd: '10',
  expirePreset: 'never',
  modelStrategy: 'follow' as 'follow' | 'lock',
  selectedModels: [] as string[],
})

const formatGroup = (code: string) =>
  formatConsoleGroupLabel(code, groupMeta.value, usableGroups.value, groupRatio.value)

const baseSelectableGroups = computed(() =>
  buildSelectableGroupOptions({
    usableGroups: usableGroups.value,
    groupMeta: groupMeta.value,
    groupRatio: groupRatio.value,
    pricingModels: pricingModels.value,
    defaultGroup: defaultGroup.value,
  }),
)

const editingIncludedGroups = computed(() => {
  const tok = editingToken.value
  if (!tok) return []
  if (tok.group === 'auto') {
    return Array.isArray(tok.auto_groups) ? tok.auto_groups.filter(Boolean) : []
  }
  return tok.group ? [tok.group] : []
})

const modalGroupOptions = computed(() =>
  buildSelectableGroupOptions({
    usableGroups: usableGroups.value,
    groupMeta: groupMeta.value,
    groupRatio: groupRatio.value,
    pricingModels: pricingModels.value,
    defaultGroup: defaultGroup.value,
    includeCodes: editingIncludedGroups.value,
  }),
)

const groupFilterOptions = computed(() => [
  { value: '', label: t('needtoken.memberConsole.keys.allGroupsFilter') },
  ...baseSelectableGroups.value.map(g => ({ value: g.code, label: g.label })),
])

const expirePresetOptions = computed(() => {
  const opts = [
    { value: 'never', label: t('needtoken.memberConsole.keys.neverExpire') },
    { value: '7', label: `7 ${t('needtoken.memberConsole.keys.days')}` },
    { value: '30', label: `30 ${t('needtoken.memberConsole.keys.days')}` },
    { value: '90', label: `90 ${t('needtoken.memberConsole.keys.days')}` },
    { value: '365', label: `365 ${t('needtoken.memberConsole.keys.days')}` },
  ]
  if (editingToken.value && editingToken.value.expired_time > 0) {
    return [
      { value: 'keep', label: `${t('needtoken.memberConsole.keys.expireKeepExisting')} (${formatUnixDate(editingToken.value.expired_time)})` },
      ...opts,
    ]
  }
  return opts
})

const availableCatalogModels = computed(() =>
  collectModelsForConsoleGroups(form.selectedGroups, groupMeta.value, pricingModels.value),
)

const modalVendorTabs = computed(() => {
  const counts = new Map<ConsoleModelVendorKey, number>()
  for (const item of availableCatalogModels.value) {
    counts.set(item.vendor, (counts.get(item.vendor) || 0) + 1)
  }
  const order: Array<{ key: ConsoleModelVendorKey; label: string }> = [
    { key: 'openai', label: 'OpenAI' },
    { key: 'claude', label: 'Claude' },
    { key: 'gemini', label: 'Gemini' },
    { key: 'grok', label: 'Grok' },
    { key: 'other', label: t('needtoken.memberConsole.keys.modelVendorOther') },
  ]
  const tabs: Array<{ key: 'all' | ConsoleModelVendorKey; label: string }> = [
    { key: 'all', label: t('needtoken.memberConsole.keys.modelAllVendors') },
  ]
  for (const entry of order) {
    const c = counts.get(entry.key) || 0
    if (c > 0) tabs.push({ key: entry.key, label: `${entry.label} (${c})` })
  }
  return tabs
})

const displayedModelChoices = computed(() => {
  const q = modelSearchQuery.value.toLowerCase()
  const base = availableCatalogModels.value.filter((item) => {
    if (modelVendorFilter.value !== 'all' && item.vendor !== modelVendorFilter.value) return false
    if (modelGroupFilter.value !== 'all' && !item.groups.includes(modelGroupFilter.value)) return false
    if (q && !item.model.toLowerCase().includes(q)) return false
    return true
  })
  const baseSet = new Set(base.map(i => i.model))
  const customExtras = form.selectedModels
    .filter(m => !baseSet.has(m))
    .filter((m) => {
      if (modelVendorFilter.value !== 'all' && classifyConsoleModelVendor(m) !== modelVendorFilter.value) return false
      if (q && !m.toLowerCase().includes(q)) return false
      return true
    })
    .map(m => ({ model: m, vendor: classifyConsoleModelVendor(m), groups: [] as string[] }))
  return [...customExtras, ...base]
})

watch(() => form.selectedGroups, (groups) => {
  if (modelGroupFilter.value !== 'all' && !groups.includes(modelGroupFilter.value)) {
    modelGroupFilter.value = 'all'
  }
}, { deep: true })

const tokenGroupCodes = (tok: GatewayTokenItem): string[] => {
  const grp = String(tok.group || '').trim()
  if (grp === 'auto') {
    const subs = Array.isArray(tok.auto_groups) ? tok.auto_groups.map(s => String(s || '').trim()).filter(Boolean) : []
    if (subs.length) return subs
    if (userAutoGroups.value.length) return userAutoGroups.value
    return ['auto']
  }
  return [grp || defaultGroup.value || 'default']
}

const tokenGroupBadges = (tok: GatewayTokenItem): string[] =>
  tokenGroupCodes(tok).map(code => (code === 'auto' ? 'auto' : formatGroup(code)))

const countTokenModels = (tok: GatewayTokenItem): number =>
  String(tok.model_limits || '').split(/[\n,]+/).map(s => s.trim()).filter(Boolean).length

const filteredTokens = computed(() => {
  const q = keyword.value.trim().toLowerCase()
  const grpFilter = filterGroup.value
  return tokens.value.filter((item) => {
    const codes = tokenGroupCodes(item)
    if (grpFilter && !codes.includes(grpFilter)) return false
    if (!q) return true
    const summary = formatTokenGroupsSummary(
      item,
      defaultGroup.value,
      groupMeta.value,
      usableGroups.value,
      groupRatio.value,
    ).toLowerCase()
    return (
      item.name.toLowerCase().includes(q)
      || item.group.toLowerCase().includes(q)
      || codes.some(c => c.toLowerCase().includes(q))
      || summary.includes(q)
      || item.masked_key.toLowerCase().includes(q)
    )
  })
})

const tokenStatusLabel = (status: number) => {
  if (status === 1) return t('needtoken.memberConsole.keys.statusEnabled')
  if (status === 3) return t('needtoken.memberConsole.keys.statusExpired')
  if (status === 4) return t('needtoken.memberConsole.keys.statusExhausted')
  return t('needtoken.memberConsole.keys.statusDisabled')
}

const formatUnixDate = (ts: number) => {
  try {
    return new Intl.DateTimeFormat(locale.value || 'zh-CN', {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
    }).format(new Date(ts * 1000))
  } catch {
    return new Date(ts * 1000).toISOString().slice(0, 10)
  }
}

let copyTimer: ReturnType<typeof setTimeout> | undefined
const copyText = async (text: string, tag: string) => {
  if (copyTimer) clearTimeout(copyTimer)
  try {
    await navigator.clipboard.writeText(text)
    copiedKey.value = tag
    copyTimer = setTimeout(() => { copiedKey.value = '' }, 1800)
  } catch {
    copiedKey.value = ''
  }
}

const toggleReveal = (id: number) => {
  const next = new Set(revealedIds.value)
  if (next.has(id)) next.delete(id)
  else next.add(id)
  revealedIds.value = next
}

const toggleGroupSelection = (code: string) => {
  formError.value = ''
  const idx = form.selectedGroups.indexOf(code)
  if (idx >= 0) {
    if (form.selectedGroups.length <= 1) return
    form.selectedGroups.splice(idx, 1)
  } else {
    form.selectedGroups.push(code)
  }
}

const selectAllModalGroups = () => {
  formError.value = ''
  const activeOnly = modalGroupOptions.value.filter(o => o.hasActiveChannels).map(o => o.code)
  form.selectedGroups = activeOnly.length ? activeOnly : modalGroupOptions.value.map(o => o.code)
}

const toggleModelSelection = (model: string) => {
  formError.value = ''
  const idx = form.selectedModels.indexOf(model)
  if (idx >= 0) form.selectedModels.splice(idx, 1)
  else form.selectedModels.push(model)
}

const selectAllVisibleModels = () => {
  formError.value = ''
  const set = new Set(form.selectedModels)
  for (const item of displayedModelChoices.value) {
    set.add(item.model)
  }
  form.selectedModels = [...set]
}

const addCustomModel = () => {
  const clean = customModelDraft.value.trim()
  if (!clean) return
  if (!form.selectedModels.includes(clean)) {
    form.selectedModels.push(clean)
  }
  customModelDraft.value = ''
}

const loadTokens = async () => {
  loading.value = true
  errorMsg.value = ''
  try {
    const [tokRes, modRes] = await Promise.all([
      listGatewayTokens(),
      getGatewayModels().catch(() => null),
    ])
    tokens.value = Array.isArray(tokRes.items) ? tokRes.items : []
    usableGroups.value = { ...(modRes?.usable_groups || {}), ...(tokRes.usable_groups || {}) }
    groupRatio.value = { ...(modRes?.group_ratio || {}), ...(tokRes.group_ratio || {}) }
    groupMeta.value = { ...(modRes?.group_meta || {}), ...(tokRes.group_meta || {}) }
    pricingModels.value = Array.isArray(modRes?.models) ? modRes.models : []
    defaultGroup.value = tokRes.default_group || modRes?.default_group || 'default'
    userAutoGroups.value = Array.isArray(tokRes.auto_groups)
      ? tokRes.auto_groups
      : (Array.isArray(modRes?.auto_groups) ? modRes.auto_groups : [])
  } catch (err: any) {
    errorMsg.value = err?.response?.data?.msg || err?.message || t('needtoken.memberConsole.loadFailed')
  } finally {
    loading.value = false
  }
}

const resetModalFilters = () => {
  formError.value = ''
  modelSearchQuery.value = ''
  modelVendorFilter.value = 'all'
  modelGroupFilter.value = 'all'
  customModelDraft.value = ''
}

const openCreateModal = () => {
  editingToken.value = null
  resetModalFilters()
  form.name = ''
  const firstActive = baseSelectableGroups.value[0]?.code || defaultGroup.value || 'default'
  form.selectedGroups = firstActive && firstActive !== 'auto' ? [firstActive] : []
  form.unlimitedQuota = true
  form.quotaUsd = '10'
  form.expirePreset = 'never'
  form.modelStrategy = 'follow'
  form.selectedModels = []
  modalOpen.value = true
}

const openEditModal = (tok: GatewayTokenItem) => {
  editingToken.value = tok
  resetModalFilters()
  form.name = tok.name
  const existingGroups = tokenGroupCodes(tok).filter(g => g && g !== 'auto')
  form.selectedGroups = existingGroups.length
    ? [...existingGroups]
    : (modalGroupOptions.value[0]?.code ? [modalGroupOptions.value[0].code] : [])
  form.unlimitedQuota = tok.unlimited_quota
  form.quotaUsd = tok.unlimited_quota ? '10' : rawQuotaToUsdNumber(tok.remain_quota).toFixed(2)
  form.expirePreset = tok.expired_time > 0 ? 'keep' : 'never'
  const parsedModels = String(tok.model_limits || '').split(/[\n,]+/).map(s => s.trim()).filter(Boolean)
  form.modelStrategy = tok.model_limits_enabled && parsedModels.length > 0 ? 'lock' : 'follow'
  form.selectedModels = parsedModels
  modalOpen.value = true
}

const closeModal = () => {
  modalOpen.value = false
  editingToken.value = null
  formError.value = ''
}

const resolveExpiredTimestamp = (preset: string, keepExisting?: number): number => {
  if (preset === 'keep') {
    return keepExisting && keepExisting > 0 ? keepExisting : -1
  }
  if (preset === 'never') {
    return -1
  }
  const days = Number(preset)
  if (!Number.isFinite(days) || days <= 0) return -1
  return Math.floor(Date.now() / 1000) + days * 86400
}

const submitModal = async () => {
  const name = form.name.trim()
  if (!name) return
  const selectedGroups = form.selectedGroups.map(s => s.trim()).filter(g => g && g !== 'auto')
  if (!selectedGroups.length) {
    formError.value = t('needtoken.memberConsole.keys.groupRequiredError')
    return
  }
  if (form.modelStrategy === 'lock' && form.selectedModels.length === 0) {
    formError.value = t('needtoken.memberConsole.keys.modelLockRequiredError')
    return
  }

  saving.value = true
  errorMsg.value = ''
  formError.value = ''
  try {
    const isMultiGroup = selectedGroups.length > 1
    const lockModels = form.modelStrategy === 'lock'
    const payload = {
      name,
      group: isMultiGroup ? 'auto' : selectedGroups[0],
      auto_groups: isMultiGroup ? selectedGroups : [],
      cross_group_retry: isMultiGroup,
      unlimited_quota: form.unlimitedQuota,
      remain_quota: form.unlimitedQuota ? 0 : usdToRawQuota(form.quotaUsd),
      expired_time: resolveExpiredTimestamp(form.expirePreset, editingToken.value ? editingToken.value.expired_time : -1),
      model_limits_enabled: lockModels,
      model_limits: lockModels ? form.selectedModels.join(',') : '',
    }
    if (editingToken.value) {
      await updateGatewayToken(editingToken.value.id, payload)
    } else {
      const created = await createGatewayToken(payload)
      if (created?.key) {
        newlyCreatedKey.value = created.key
        newlyCreatedTokenId.value = created?.token?.id ?? null
      }
    }
    closeModal()
    await loadTokens()
    emit('changed')
  } catch (err: any) {
    formError.value = err?.response?.data?.msg || err?.message || t('needtoken.memberConsole.actionFailed')
  } finally {
    saving.value = false
  }
}

const toggleStatus = async (tok: GatewayTokenItem) => {
  busyId.value = tok.id
  errorMsg.value = ''
  try {
    await updateGatewayToken(tok.id, { status: tok.status === 1 ? 2 : 1 })
    await loadTokens()
    emit('changed')
  } catch (err: any) {
    errorMsg.value = err?.response?.data?.msg || err?.message || t('needtoken.memberConsole.actionFailed')
  } finally {
    busyId.value = null
  }
}

const confirmDelete = async (tok: GatewayTokenItem) => {
  busyId.value = tok.id
  errorMsg.value = ''
  try {
    await deleteGatewayToken(tok.id)
    confirmDeleteId.value = null
    await loadTokens()
    emit('changed')
  } catch (err: any) {
    errorMsg.value = err?.response?.data?.msg || err?.message || t('needtoken.memberConsole.actionFailed')
  } finally {
    busyId.value = null
  }
}

onMounted(() => {
  void loadTokens()
})

defineExpose({ openCreateModal, loadTokens })
</script>
