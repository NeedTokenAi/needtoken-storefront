<template>
  <div ref="rootEl" class="nt-model-combobox" :class="{ 'nt-model-combobox--compact': compact }">
    <button
      :id="id"
      type="button"
      class="nt-theme-select__trigger w-full"
      :aria-label="label"
      :aria-expanded="open"
      :aria-controls="listId"
      :disabled="disabled"
      @click="toggleOpen"
    >
      <span class="nt-model-combobox__value font-mono">
        {{ modelValue || placeholder || label }}
      </span>
      <ChevronDown class="h-3.5 w-3.5 shrink-0 transition-transform duration-150" :class="open ? 'rotate-180' : ''" />
    </button>

    <div
      v-if="open"
      :id="listId"
      class="nt-model-combobox__panel"
      @keydown.esc.stop.prevent="closePanel"
    >
      <div class="nt-model-combobox__search">
        <Search class="pointer-events-none h-3.5 w-3.5 shrink-0 text-muted-foreground" />
        <input
          ref="searchInputEl"
          v-model="query"
          type="text"
          class="nt-model-combobox__input font-mono"
          :placeholder="searchPlaceholder || placeholder || label"
          @keydown.down.prevent="moveHighlight(1)"
          @keydown.up.prevent="moveHighlight(-1)"
          @keydown.enter.prevent="confirmCurrent"
        />
        <button
          v-if="query"
          type="button"
          class="rounded p-0.5 text-muted-foreground hover:text-foreground"
          @click="query = ''"
        >
          <X class="h-3 w-3" />
        </button>
      </div>

      <div v-if="vendorTabs.length > 2" class="nt-model-combobox__vendors">
        <button
          v-for="tab in vendorTabs"
          :key="tab.key"
          type="button"
          class="nt-model-combobox__vendor-pill"
          :class="{ 'is-active': activeVendor === tab.key }"
          @click="activeVendor = tab.key"
        >
          {{ tab.label }}
        </button>
      </div>

      <div class="nt-model-combobox__list">
        <button
          v-if="allowCustom && canUseCustomQuery"
          type="button"
          class="nt-model-combobox__option nt-model-combobox__option--custom"
          @click="selectModel(query.trim())"
        >
          <span class="truncate font-mono">{{ query.trim() }}</span>
          <span class="shrink-0 text-[10px] font-semibold opacity-75">Enter</span>
        </button>

        <button
          v-for="(item, idx) in filteredModels"
          :key="item"
          type="button"
          class="nt-model-combobox__option"
          :class="{
            'is-selected': item === modelValue,
            'is-highlighted': idx === highlightedIndex,
          }"
          @mouseenter="highlightedIndex = idx"
          @click="selectModel(item)"
        >
          <span class="truncate font-mono">{{ item }}</span>
          <Check v-if="item === modelValue" class="h-3.5 w-3.5 shrink-0 text-primary" />
        </button>

        <div v-if="!filteredModels.length && !(allowCustom && canUseCustomQuery)" class="px-3 py-4 text-center text-xs text-muted-foreground">
          {{ emptyText }}
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { Check, ChevronDown, Search, X } from 'lucide-vue-next'
import { classifyConsoleModelVendor, type ConsoleModelVendorKey } from '../../lib/gatewayConsoleHelpers'

const props = withDefaults(defineProps<{
  modelValue: string
  models: readonly string[]
  label: string
  id?: string
  placeholder?: string
  searchPlaceholder?: string
  emptyText?: string
  disabled?: boolean
  allowCustom?: boolean
  compact?: boolean
}>(), {
  disabled: false,
  allowCustom: true,
  compact: false,
})

const emit = defineEmits<{ 'update:modelValue': [value: string] }>()
const { t } = useI18n()

const rootEl = ref<HTMLElement | null>(null)
const searchInputEl = ref<HTMLInputElement | null>(null)
const open = ref(false)
const query = ref('')
const activeVendor = ref<'all' | ConsoleModelVendorKey>('all')
const highlightedIndex = ref(0)
const listId = `nt-model-combo-${Math.random().toString(36).slice(2, 9)}`

const uniqueModels = computed(() => {
  const seen = new Set<string>()
  const out: string[] = []
  for (const raw of props.models || []) {
    const clean = String(raw || '').trim()
    if (clean && !seen.has(clean)) {
      seen.add(clean)
      out.push(clean)
    }
  }
  return out
})

const vendorTabs = computed(() => {
  const counts = new Map<ConsoleModelVendorKey, number>()
  for (const m of uniqueModels.value) {
    const v = classifyConsoleModelVendor(m)
    counts.set(v, (counts.get(v) || 0) + 1)
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
    if ((counts.get(entry.key) || 0) > 0) {
      tabs.push({ key: entry.key, label: `${entry.label} (${counts.get(entry.key)})` })
    }
  }
  return tabs
})

const filteredModels = computed(() => {
  const q = query.value.trim().toLowerCase()
  return uniqueModels.value.filter((item) => {
    if (activeVendor.value !== 'all' && classifyConsoleModelVendor(item) !== activeVendor.value) {
      return false
    }
    if (!q) return true
    return item.toLowerCase().includes(q)
  })
})

const canUseCustomQuery = computed(() => {
  const q = query.value.trim()
  if (!q) return false
  return !uniqueModels.value.includes(q)
})

const emptyText = computed(() => props.emptyText || t('needtoken.memberConsole.playground.noMatchingModels'))

watch([query, activeVendor], () => {
  highlightedIndex.value = 0
})

const openPanel = async () => {
  if (props.disabled) return
  open.value = true
  query.value = ''
  activeVendor.value = 'all'
  highlightedIndex.value = Math.max(0, uniqueModels.value.indexOf(props.modelValue))
  await nextTick()
  searchInputEl.value?.focus()
}

const closePanel = () => {
  open.value = false
}

const toggleOpen = () => {
  if (open.value) closePanel()
  else void openPanel()
}

const selectModel = (value: string) => {
  const clean = value.trim()
  if (!clean) return
  emit('update:modelValue', clean)
  closePanel()
}

const moveHighlight = (step: number) => {
  const len = filteredModels.value.length
  if (!len) return
  highlightedIndex.value = (highlightedIndex.value + step + len) % len
}

const confirmCurrent = () => {
  const current = filteredModels.value[highlightedIndex.value]
  if (current) {
    selectModel(current)
    return
  }
  if (props.allowCustom && query.value.trim()) {
    selectModel(query.value.trim())
  }
}

const onPointerDown = (event: MouseEvent) => {
  if (!open.value || !rootEl.value) return
  const target = event.target as Node | null
  if (target && !rootEl.value.contains(target)) {
    closePanel()
  }
}

onMounted(() => {
  document.addEventListener('mousedown', onPointerDown)
})

onUnmounted(() => {
  document.removeEventListener('mousedown', onPointerDown)
})
</script>

<style scoped>
.nt-model-combobox {
  position: relative;
  min-width: 0;
  width: 100%;
}
.nt-model-combobox--compact .nt-theme-select__trigger {
  min-height: 34px;
  padding: 6px 10px;
  font-size: 11.5px;
}
.nt-model-combobox__value {
  min-width: 0;
  flex: 1;
  text-align: start;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}
.nt-model-combobox__panel {
  position: absolute;
  left: 0;
  right: 0;
  top: calc(100% + 6px);
  z-index: 60;
  border: 1px solid var(--ui-border-strong);
  border-radius: 12px;
  background: var(--ui-bg-elevated);
  color: var(--ui-text-primary);
  box-shadow: var(--nt-popover-shadow);
  padding: 8px;
}
.nt-model-combobox__search {
  display: flex;
  align-items: center;
  gap: 6px;
  border: 1px solid var(--ui-border);
  border-radius: 9px;
  background: var(--nt-surface-input);
  padding: 0 9px;
  height: 34px;
}
.nt-model-combobox__search:focus-within {
  border-color: var(--ui-accent);
}
.nt-model-combobox__input {
  min-width: 0;
  flex: 1;
  background: transparent;
  border: 0;
  outline: none;
  font-size: 12px;
  color: var(--ui-text-primary);
}
.nt-model-combobox__vendors {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  margin-top: 7px;
}
.nt-model-combobox__vendor-pill {
  border: 1px solid var(--ui-border);
  border-radius: 999px;
  background: var(--ui-bg-soft);
  color: var(--ui-text-secondary);
  padding: 2px 8px;
  font-size: 11px;
  font-weight: 600;
  cursor: pointer;
}
.nt-model-combobox__vendor-pill:hover {
  border-color: var(--nt-accent-border-strong);
  color: var(--ui-text-primary);
}
.nt-model-combobox__vendor-pill.is-active {
  border-color: var(--nt-accent-border-strong);
  background: var(--nt-accent-subtle);
  color: var(--ui-accent);
}
.nt-model-combobox__list {
  margin-top: 7px;
  max-height: 240px;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.nt-model-combobox__option {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  width: 100%;
  min-height: 32px;
  padding: 6px 10px;
  border-radius: 8px;
  border: 0;
  background: transparent;
  color: var(--ui-text-secondary);
  font-size: 12px;
  text-align: left;
  cursor: pointer;
}
.nt-model-combobox__option.is-highlighted,
.nt-model-combobox__option:hover {
  background: var(--ui-accent-soft);
  color: var(--ui-text-primary);
}
.nt-model-combobox__option.is-selected {
  background: var(--nt-accent-subtle);
  color: var(--ui-accent);
  font-weight: 600;
}
.nt-model-combobox__option--custom {
  border: 1px dashed var(--nt-accent-border);
  color: var(--ui-accent);
}
</style>
