<template>
  <div class="radar-record-text">
    <p class="record-preview">{{ text }}</p>
    <details>
      <summary>{{ t('needtoken.radarLayout.fullText') }}<ChevronDown :size="14" aria-hidden="true" /></summary>
      <p class="record-full">{{ text }}</p>
    </details>
  </div>
</template>

<script setup lang="ts">
import { ChevronDown } from 'lucide-vue-next'
import { useI18n } from 'vue-i18n'
defineProps<{ text: string }>()
const { t } = useI18n()
</script>

<style scoped>
.radar-record-text { min-width: 0; }
p { color: var(--ui-text-secondary); font-size: 14px; line-height: 1.7; overflow-wrap: anywhere; white-space: pre-line; }
.record-preview { display: -webkit-box; -webkit-line-clamp: 2; -webkit-box-orient: vertical; overflow: hidden; }
.radar-record-text:has(details[open]) > .record-preview { display: none; }
summary { display: inline-flex; align-items: center; gap: 4px; min-height: 32px; color: var(--ui-text-muted); font-size: 12px; cursor: pointer; list-style: none; transition: color 160ms var(--nt-motion-ease, ease-out); }
summary::-webkit-details-marker { display: none; }
summary:hover, details[open] > summary { color: var(--ui-text-primary); }
summary svg { transition: transform 160ms var(--nt-motion-ease, ease-out); }
details[open] summary svg { transform: rotate(180deg); }
summary:focus-visible { outline: 2px solid var(--ui-accent); outline-offset: 2px; border-radius: 4px; }
@media (prefers-reduced-motion: reduce) { summary, summary svg { transition: none; } }
</style>
