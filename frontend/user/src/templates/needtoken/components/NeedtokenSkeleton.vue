<template>
  <div class="nt-page-skeleton" :class="`nt-page-skeleton--${kind}`" role="status" :aria-label="t('needtoken.startup.loading')" aria-busy="true">
    <div class="nt-skeleton-heading" aria-hidden="true"><i /><i /><i /></div>
    <div class="nt-skeleton-cards" aria-hidden="true"><article v-for="index in count" :key="index"><i /><i /><i /><i /></article></div>
    <span class="sr-only">{{ t('needtoken.startup.loading') }}</span>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
withDefaults(defineProps<{ kind?: 'page' | 'cards' | 'section'; count?: number }>(), { kind: 'page', count: 3 })
const { t } = useI18n()
</script>

<style scoped>
.nt-page-skeleton { width: 100%; max-width: 1200px; margin: 0 auto; padding: 48px 24px; }
.nt-skeleton-heading { display: grid; justify-items: center; gap: 16px; margin: 0 auto 34px; }
.nt-page-skeleton i { display: block; position: relative; overflow: hidden; border-radius: 8px; background: color-mix(in srgb, var(--ui-text-muted, #7d8a99) 12%, transparent); }
.nt-page-skeleton i::after { position: absolute; inset: 0; content: ''; background: linear-gradient(100deg, transparent 20%, #ffffff09 45%, transparent 70%); transform: translateX(-100%); animation: nt-skeleton-shimmer 2s ease-in-out infinite; }
.nt-skeleton-heading i:first-child { width: 100px; height: 12px; }
.nt-skeleton-heading i:nth-child(2) { width: min(70%, 520px); height: 37px; }
.nt-skeleton-heading i:nth-child(3) { width: min(55%, 390px); height: 14px; }
.nt-skeleton-cards { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 20px; }
.nt-skeleton-cards article { min-height: 260px; padding: 24px; border: 1px solid var(--ui-border, #7d8a9929); border-radius: 22px; background: color-mix(in srgb, var(--ui-bg-elevated, #0e1218) 55%, transparent); }
.nt-skeleton-cards article i:first-child { width: 38px; height: 38px; margin-bottom: 27px; border-radius: 12px; }
.nt-skeleton-cards article i:nth-child(2) { width: 65%; height: 23px; margin-bottom: 15px; }
.nt-skeleton-cards article i:nth-child(3) { width: 88%; height: 42px; margin-bottom: 30px; }
.nt-skeleton-cards article i:nth-child(4) { width: 100%; height: 40px; margin-top: auto; }
.nt-page-skeleton--cards .nt-skeleton-heading { display: none; }
.nt-page-skeleton--section { padding: 28px 0; }
.nt-page-skeleton--section .nt-skeleton-heading { justify-items: start; }
.nt-page-skeleton--section .nt-skeleton-cards article { min-height: 180px; }
@keyframes nt-skeleton-shimmer { to { transform: translateX(100%); } }
@media (max-width: 640px) { .nt-page-skeleton { padding: 32px 20px; } .nt-skeleton-cards { grid-template-columns: 1fr; gap: 12px; } .nt-skeleton-cards article { min-height: 118px; padding: 18px; } .nt-skeleton-cards article i:first-child { width: 30px; height: 30px; margin-bottom: 14px; } .nt-skeleton-cards article i:nth-child(3) { height: 20px; margin-bottom: 16px; } .nt-skeleton-heading i:nth-child(2) { width: 80%; height: 32px; } }
@media (prefers-reduced-motion: reduce) { .nt-page-skeleton i::after { animation: none; } }
</style>
