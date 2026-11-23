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
          <Button variant="outline" size="sm" @click="showSnippets = !showSnippets">
            <Terminal class="mr-1.5 h-3.5 w-3.5 text-primary" />
            {{ t('needtoken.memberConsole.keys.quickConnect') }}
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
        <button
          type="button"
          class="inline-flex shrink-0 items-center gap-1 rounded-md border bg-background px-2.5 py-1 text-xs font-medium text-foreground transition-colors hover:bg-accent"
          @click="copyText(API_BASE_URL, 'endpoint')"
        >
          <Check v-if="copiedKey === 'endpoint'" class="h-3.5 w-3.5 text-primary" />
          <Copy v-else class="h-3.5 w-3.5" />
          <span>{{ copiedKey === 'endpoint' ? t('needtoken.common.copied') : t('needtoken.common.copy') }}</span>
        </button>
      </div>

      <!-- 一键接入代码片段（cURL / Python / Claude Code / Codex / OpenCode） -->
      <div v-if="showSnippets" class="mt-4 rounded-xl border bg-muted/20 p-4">
        <div class="flex flex-wrap items-center justify-between gap-2">
          <div class="flex flex-wrap items-center gap-1.5">
            <button
              v-for="tab in snippetTabs"
              :key="tab.key"
              type="button"
              class="rounded-lg px-2.5 py-1 text-xs font-semibold transition-colors"
              :class="activeSnippet === tab.key ? 'bg-primary text-primary-foreground' : 'bg-background text-muted-foreground hover:text-foreground'"
              @click="activeSnippet = tab.key"
            >
              {{ tab.label }}
            </button>
          </div>
          <Button variant="outline" size="sm" class="h-7 px-2.5 text-xs" @click="copyText(activeSnippetCode, `snippet:${activeSnippet}`)">
            <Check v-if="copiedKey === `snippet:${activeSnippet}`" class="mr-1 h-3 w-3 text-primary" />
            <Copy v-else class="mr-1 h-3 w-3" />
            {{ copiedKey === `snippet:${activeSnippet}` ? t('needtoken.common.copied') : t('needtoken.common.copy') }}
          </Button>
        </div>
        <pre class="mt-3 overflow-x-auto rounded-lg border bg-background p-3 font-mono text-[11.5px] leading-relaxed text-foreground">{{ activeSnippetCode }}</pre>
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
          <Button size="sm" @click="copyText(newlyCreatedKey, 'new-key')">
            <Check v-if="copiedKey === 'new-key'" class="mr-1.5 h-3.5 w-3.5" />
            <Copy v-else class="mr-1.5 h-3.5 w-3.5" />
            {{ copiedKey === 'new-key' ? t('needtoken.common.copied') : t('needtoken.common.copy') }}
          </Button>
          <Button variant="ghost" size="sm" @click="newlyCreatedKey = ''">
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
        <div class="relative w-full max-w-xs">
          <Search class="pointer-events-none absolute left-3 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-muted-foreground" />
          <input
            v-model="keyword"
            type="text"
            :placeholder="t('needtoken.memberConsole.keys.searchPlaceholder')"
            class="h-9 w-full rounded-lg border bg-background pl-8 pr-3 text-xs text-foreground outline-none focus:border-primary"
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
            <div class="flex flex-wrap items-center gap-2">
              <span class="truncate text-sm font-bold text-foreground">{{ tok.name || `Key #${tok.id}` }}</span>
              <Badge :variant="tok.status === 1 ? 'success' : 'secondary'" size="sm">
                {{ tokenStatusLabel(tok.status) }}
              </Badge>
              <span class="rounded-md bg-muted px-2 py-0.5 font-mono text-[11px] font-medium text-muted-foreground">
                {{ tok.group || defaultGroup || 'default' }}
              </span>
              <span v-if="tok.model_limits_enabled && tok.model_limits" class="rounded-md border px-2 py-0.5 text-[11px] text-muted-foreground">
                {{ t('needtoken.memberConsole.keys.modelRestricted') }}
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
      <form class="w-full max-w-lg rounded-2xl border bg-card p-6 shadow-xl" @submit.prevent="submitModal">
        <div class="flex items-center justify-between border-b pb-3">
          <h3 class="text-base font-bold text-foreground">
            {{ editingToken ? t('needtoken.memberConsole.keys.editTitle') : t('needtoken.memberConsole.keys.createTitle') }}
          </h3>
          <button type="button" class="rounded-md p-1 text-muted-foreground hover:bg-accent" @click="closeModal">
            <X class="h-4 w-4" />
          </button>
        </div>

        <div class="mt-4 space-y-4">
          <label class="block space-y-1.5 text-xs">
            <span class="font-semibold text-foreground">{{ t('needtoken.memberConsole.keys.nameLabel') }} *</span>
            <input
              v-model="form.name"
              type="text"
              maxlength="50"
              required
              :placeholder="t('needtoken.memberConsole.keys.namePlaceholder')"
              class="h-10 w-full rounded-lg border bg-background px-3 text-sm text-foreground outline-none focus:border-primary"
            />
          </label>

          <div class="block space-y-1.5 text-xs">
            <span class="font-semibold text-foreground">{{ t('needtoken.memberConsole.keys.groupLabel') }}</span>
            <div
              v-if="groupOptions.length <= 1"
              class="flex items-center justify-between gap-2 rounded-lg border bg-muted/30 px-3 py-2.5 text-sm"
            >
              <span class="font-mono font-semibold text-foreground">
                {{ groupOptions[0]?.name || defaultGroup }}
                <span v-if="groupOptions[0]?.desc && groupOptions[0]?.desc !== groupOptions[0]?.name" class="font-sans text-xs font-normal text-muted-foreground">
                  — {{ groupOptions[0].desc }}
                </span>
              </span>
              <span class="text-[11px] text-muted-foreground">{{ t('needtoken.memberConsole.keys.singleGroupHint') }}</span>
            </div>
            <select
              v-else
              v-model="form.group"
              class="h-10 w-full rounded-lg border bg-background px-3 text-sm text-foreground outline-none focus:border-primary"
            >
              <option v-for="g in groupOptions" :key="g.name" :value="g.name">
                {{ g.name }}{{ g.desc && g.desc !== g.name ? ` — ${g.desc}` : '' }}
              </option>
            </select>
          </div>

          <div class="space-y-2 rounded-xl border bg-muted/30 p-3.5 text-xs">
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

          <label class="block space-y-1.5 text-xs">
            <span class="font-semibold text-foreground">{{ t('needtoken.memberConsole.keys.expireDaysLabel') }}</span>
            <select
              v-model="form.expirePreset"
              class="h-10 w-full rounded-lg border bg-background px-3 text-sm text-foreground outline-none focus:border-primary"
            >
              <option value="never">{{ t('needtoken.memberConsole.keys.neverExpire') }}</option>
              <option value="7">7 {{ t('needtoken.memberConsole.keys.days') }}</option>
              <option value="30">30 {{ t('needtoken.memberConsole.keys.days') }}</option>
              <option value="90">90 {{ t('needtoken.memberConsole.keys.days') }}</option>
              <option value="365">365 {{ t('needtoken.memberConsole.keys.days') }}</option>
            </select>
          </label>

          <label class="block space-y-1.5 text-xs">
            <span class="font-semibold text-foreground">{{ t('needtoken.memberConsole.keys.modelLimitsLabel') }}</span>
            <input
              v-model="form.modelLimits"
              type="text"
              :placeholder="t('needtoken.memberConsole.keys.modelLimitsPlaceholder')"
              class="h-10 w-full rounded-lg border bg-background px-3 text-sm text-foreground outline-none focus:border-primary"
            />
          </label>
        </div>

        <div class="mt-6 flex justify-end gap-2 border-t pt-4">
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
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { Check, Copy, Eye, EyeOff, KeyRound, Pencil, Plus, RefreshCw, Search, Terminal, Trash2, X } from 'lucide-vue-next'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { API_BASE_URL, DEFAULT_MODELS } from '../../config'
import { apiPreviewCode, integrationTemplate } from '../../lib/apiIntegrations'
import { formatApiQuotaUsd } from '../../lib/apiAccountLinkPolicy'
import {
  createGatewayToken,
  deleteGatewayToken,
  listGatewayTokens,
  rawQuotaToUsdNumber,
  updateGatewayToken,
  usdToRawQuota,
  type GatewayTokenItem,
} from '../../lib/gatewayConsoleApi'

const emit = defineEmits<{ (e: 'changed'): void }>()

const { t, locale } = useI18n()
const loading = ref(false)
const saving = ref(false)
const busyId = ref<number | null>(null)
const errorMsg = ref('')
const keyword = ref('')
const tokens = ref<GatewayTokenItem[]>([])
const usableGroups = ref<Record<string, string>>({})
const defaultGroup = ref('default')
const defaultAutoGroup = ref(false)
const revealedIds = ref<Set<number>>(new Set())
const copiedKey = ref('')
const newlyCreatedKey = ref('')
const confirmDeleteId = ref<number | null>(null)
const modalOpen = ref(false)
const editingToken = ref<GatewayTokenItem | null>(null)
const showSnippets = ref(false)
const activeSnippet = ref<'curl' | 'python' | 'claude' | 'codex' | 'opencode'>('curl')

const snippetTabs: Array<{ key: 'curl' | 'python' | 'claude' | 'codex' | 'opencode'; label: string }> = [
  { key: 'curl', label: 'cURL' },
  { key: 'python', label: 'Python SDK' },
  { key: 'claude', label: 'Claude Code' },
  { key: 'codex', label: 'Codex CLI' },
  { key: 'opencode', label: 'OpenCode' },
]

const activeSnippetCode = computed(() => {
  const model = DEFAULT_MODELS[0] || 'claude-sonnet-4-6'
  if (activeSnippet.value === 'python') return apiPreviewCode('python', API_BASE_URL, model)
  if (activeSnippet.value === 'claude') return integrationTemplate('claudecode', API_BASE_URL, model) || `export ANTHROPIC_BASE_URL="${API_BASE_URL.replace(/\/v1$/, '')}"\nexport ANTHROPIC_AUTH_TOKEN="<YOUR_NEEDTOKEN_API_KEY>"`
  if (activeSnippet.value === 'codex') return integrationTemplate('codex', API_BASE_URL, model) || `export OPENAI_BASE_URL="${API_BASE_URL}"\nexport OPENAI_API_KEY="<YOUR_NEEDTOKEN_API_KEY>"`
  if (activeSnippet.value === 'opencode') return integrationTemplate('opencode', API_BASE_URL, model) || apiPreviewCode('curl', API_BASE_URL, model)
  return apiPreviewCode('curl', API_BASE_URL, model)
})

const form = reactive({
  name: '',
  group: 'default',
  unlimitedQuota: true,
  quotaUsd: '10',
  expirePreset: 'never',
  modelLimits: '',
})

const groupOptions = computed(() => {
  const entries = Object.entries(usableGroups.value)
  if (!entries.length) return [{ name: defaultGroup.value || 'default', desc: '' }]
  return entries.map(([name, desc]) => ({ name, desc }))
})

const filteredTokens = computed(() => {
  const q = keyword.value.trim().toLowerCase()
  if (!q) return tokens.value
  return tokens.value.filter(
    item =>
      item.name.toLowerCase().includes(q)
      || item.group.toLowerCase().includes(q)
      || item.masked_key.toLowerCase().includes(q),
  )
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

const loadTokens = async () => {
  loading.value = true
  errorMsg.value = ''
  try {
    const res = await listGatewayTokens()
    tokens.value = Array.isArray(res.items) ? res.items : []
    usableGroups.value = res.usable_groups || {}
    defaultGroup.value = res.default_group || 'default'
    defaultAutoGroup.value = Boolean(res.default_auto_group)
  } catch (err: any) {
    errorMsg.value = err?.response?.data?.msg || err?.message || t('needtoken.memberConsole.loadFailed')
  } finally {
    loading.value = false
  }
}

const openCreateModal = () => {
  editingToken.value = null
  form.name = ''
  const preferred = defaultAutoGroup.value && usableGroups.value.auto ? 'auto' : (defaultGroup.value || 'default')
  form.group = groupOptions.value.some(g => g.name === preferred)
    ? preferred
    : (groupOptions.value[0]?.name || preferred)
  form.unlimitedQuota = true
  form.quotaUsd = '10'
  form.expirePreset = 'never'
  form.modelLimits = ''
  modalOpen.value = true
}

const openEditModal = (tok: GatewayTokenItem) => {
  editingToken.value = tok
  form.name = tok.name
  form.group = tok.group || defaultGroup.value || 'default'
  form.unlimitedQuota = tok.unlimited_quota
  form.quotaUsd = tok.unlimited_quota ? '10' : rawQuotaToUsdNumber(tok.remain_quota).toFixed(2)
  form.expirePreset = 'never'
  form.modelLimits = tok.model_limits_enabled ? tok.model_limits : ''
  modalOpen.value = true
}

const closeModal = () => {
  modalOpen.value = false
  editingToken.value = null
}

const resolveExpiredTimestamp = (preset: string, keepExisting?: number): number => {
  if (preset === 'never') {
    return keepExisting && keepExisting > 0 ? keepExisting : -1
  }
  const days = Number(preset)
  if (!Number.isFinite(days) || days <= 0) return -1
  return Math.floor(Date.now() / 1000) + days * 86400
}

const submitModal = async () => {
  const name = form.name.trim()
  if (!name) return
  saving.value = true
  errorMsg.value = ''
  try {
    const limits = form.modelLimits.trim()
    const payload = {
      name,
      group: form.group,
      unlimited_quota: form.unlimitedQuota,
      remain_quota: form.unlimitedQuota ? 0 : usdToRawQuota(form.quotaUsd),
      expired_time: resolveExpiredTimestamp(form.expirePreset, editingToken.value ? editingToken.value.expired_time : -1),
      model_limits_enabled: limits.length > 0,
      model_limits: limits,
    }
    if (editingToken.value) {
      await updateGatewayToken(editingToken.value.id, payload)
    } else {
      const created = await createGatewayToken(payload)
      if (created?.key) {
        newlyCreatedKey.value = created.key
      }
    }
    closeModal()
    await loadTokens()
    emit('changed')
  } catch (err: any) {
    errorMsg.value = err?.response?.data?.msg || err?.message || t('needtoken.memberConsole.actionFailed')
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
