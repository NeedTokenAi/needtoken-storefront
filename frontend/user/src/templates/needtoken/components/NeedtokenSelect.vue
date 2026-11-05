<template>
  <div v-bind="$attrs" class="nt-theme-select" :class="{ 'nt-theme-select--compact': compact }">
    <Select :model-value="encodedValue" :disabled="disabled" @update:model-value="onChange">
      <SelectTrigger :id="id" class="nt-theme-select__trigger" :aria-label="label">
        <span v-if="$slots.leading" class="nt-theme-select__leading" aria-hidden="true"><slot name="leading" /></span>
        <SelectValue class="nt-theme-select__value" />
      </SelectTrigger>
      <SelectContent class="nt-theme-select__menu" position="popper" :align="align" :side-offset="6" :collision-padding="12">
        <SelectItem v-for="option in options" :key="option.value" class="nt-theme-select__option" :value="encode(option.value)" :text-value="option.label" :disabled="option.disabled">
          {{ option.label }}
        </SelectItem>
      </SelectContent>
    </Select>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '../../../components/ui/select'

defineOptions({ inheritAttrs: false })
const props = withDefaults(defineProps<{
  modelValue: string
  options: readonly { value: string; label: string; disabled?: boolean }[]
  label: string
  id?: string
  disabled?: boolean
  compact?: boolean
  align?: 'start' | 'center' | 'end'
}>(), { disabled: false, compact: false, align: 'end' })
const emit = defineEmits<{ 'update:modelValue': [value: string] }>()
// Encode all values so an empty "all" filter is a real selectable option in Reka.
const encode = (value: string) => `nt:${value}`
const encodedValue = computed(() => encode(props.modelValue))
const onChange = (value: unknown) => {
  if (typeof value !== 'string' || !value.startsWith('nt:')) return
  const decoded = value.slice(3)
  if (props.options.some(option => option.value === decoded && !option.disabled)) emit('update:modelValue', decoded)
}
</script>

<style>
/* Namespaced styles also reach the Reka menu portaled to body.nt-tokens. */
.nt-theme-select { min-width: 0; max-width: 100%; }
.nt-theme-select__trigger { gap: 9px; min-width: 0; max-width: 100%; height: auto; min-height: 40px; padding: 9px 12px; border: 1px solid var(--ui-border); border-radius: 11px; background: var(--ui-bg-elevated); color: var(--ui-text-primary); font: inherit; font-size: 12px; font-weight: 550; line-height: 1.5; box-shadow: inset 0 1px 0 #ffffff05; cursor: pointer; transition: border-color 140ms ease, background-color 140ms ease, box-shadow 140ms ease; }
.nt-theme-select__trigger:hover:not(:disabled), .nt-theme-select__trigger[data-state="open"] { border-color: #2dd4a75c; background: var(--ui-bg-soft); }
.nt-theme-select__trigger:focus-visible { outline: 2px solid var(--ui-focus-ring); outline-offset: 3px; box-shadow: none; }
.nt-theme-select__trigger:disabled { cursor: default; opacity: .7; }
.nt-theme-select__trigger > span:not(.nt-theme-select__leading) { min-width: 0; flex: 1; text-align: start; overflow: hidden; white-space: nowrap; text-overflow: ellipsis; }
.nt-theme-select__leading { display: inline-flex; flex: none; color: var(--ui-accent); }
.nt-theme-select__trigger > svg { width: 13px; height: 13px; color: var(--ui-text-muted); opacity: 1; transition: transform 140ms ease; }
.nt-theme-select__trigger[data-state="open"] > svg { transform: rotate(180deg); }
.nt-theme-select--compact .nt-theme-select__trigger { min-height: 34px; padding: 7px 10px; gap: 8px; font-size: 11px; }
.nt-theme-select__menu { z-index: 90; min-width: max(160px, var(--reka-select-trigger-width)); max-width: min(360px, calc(100vw - 24px)); max-height: min(320px, var(--reka-select-content-available-height)); border: 1px solid var(--ui-border-strong); border-radius: 12px; background: var(--ui-bg-elevated); color: var(--ui-text-primary); box-shadow: 0 14px 40px #0005, inset 0 1px 0 #ffffff08; animation-duration: 140ms; }
.nt-theme-select__option { min-height: 34px; padding: 8px 31px 8px 11px; border-radius: 8px; color: var(--ui-text-secondary); font-family: inherit; font-size: 12px; line-height: 1.5; white-space: normal; overflow-wrap: anywhere; cursor: pointer; transition: background-color 120ms ease, color 120ms ease; }
.nt-theme-select__option[data-state="checked"] { color: var(--ui-accent); background: #2dd4a70d; }
.nt-theme-select__option[data-highlighted] { color: var(--ui-text-primary); background: var(--ui-accent-soft); }
.nt-theme-select__option > span:first-child { color: var(--ui-accent); }
.nt-theme-select__option[data-disabled] { opacity: .45; cursor: default; }
@media (prefers-reduced-motion: reduce) { .nt-theme-select__trigger, .nt-theme-select__trigger > svg, .nt-theme-select__option { transition: none; } .nt-theme-select__menu { animation: none !important; } }
</style>
