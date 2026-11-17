<template>
  <div class="nt-page-skeleton" :class="`nt-page-skeleton--${kind}`" role="status" :aria-label="t('needtoken.startup.loading')" aria-busy="true">
    <div class="nt-skeleton-heading" aria-hidden="true"><i /><i /><i /></div>
    <div class="nt-skeleton-cards" aria-hidden="true">
      <article v-for="index in count + (kind === 'prices' && count === 4 ? 1 : 0)" :key="index" :class="{ 'nt-skeleton-price-extra': kind === 'prices' && count === 4 && index === 5 }">
        <template v-if="kind === 'prices'">
          <div class="nt-skeleton-price-heading"><i class="nt-skeleton-price-mark" /><i class="nt-skeleton-price-name" /><i class="nt-skeleton-price-invoice" /></div>
          <i class="nt-skeleton-price-chip" />
          <div class="nt-skeleton-price-bottom"><i class="nt-skeleton-price-value" /><i class="nt-skeleton-price-art" /></div>
        </template>
        <template v-else><i /><i /><i /><i /></template>
      </article>
    </div>
    <span class="sr-only">{{ t('needtoken.startup.loading') }}</span>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
withDefaults(defineProps<{ kind?: 'page' | 'cards' | 'section' | 'prices'; count?: number }>(), { kind: 'page', count: 3 })
const { t } = useI18n()
</script>

<style scoped>
.nt-page-skeleton { box-sizing: border-box; min-width: 0; width: 100%; max-width: 1200px; margin: 0 auto; padding: 48px 24px; }
.nt-skeleton-heading { display: grid; justify-items: center; gap: 16px; margin: 0 auto 34px; }
.nt-page-skeleton i { display: block; position: relative; overflow: hidden; border-radius: 8px; background: color-mix(in srgb, var(--ui-text-muted, #7d8a99) 12%, transparent); }
.nt-page-skeleton i::after { position: absolute; inset: 0; content: ''; background: linear-gradient(100deg, transparent 20%, #ffffff09 45%, transparent 70%); transform: translateX(-100%); animation: nt-skeleton-shimmer 2s ease-in-out infinite; }
.nt-skeleton-heading i:first-child { width: 100px; height: 12px; }
.nt-skeleton-heading i:nth-child(2) { width: min(70%, 520px); height: 37px; }
.nt-skeleton-heading i:nth-child(3) { width: min(55%, 390px); height: 14px; }
.nt-skeleton-cards { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 20px; }
.nt-skeleton-cards article { box-sizing: border-box; min-width: 0; min-height: 260px; padding: 24px; border: 1px solid var(--ui-border, #7d8a9929); border-radius: 22px; background: color-mix(in srgb, var(--ui-bg-elevated, #0e1218) 55%, transparent); }
.nt-page-skeleton:not(.nt-page-skeleton--prices) article > i:first-child { width: 38px; height: 38px; margin-bottom: 28px; border-radius: 12px; }
.nt-page-skeleton:not(.nt-page-skeleton--prices) article > i:nth-child(2) { width: 65%; height: 24px; margin-bottom: 16px; }
.nt-page-skeleton:not(.nt-page-skeleton--prices) article > i:nth-child(3) { width: 88%; height: 40px; margin-bottom: 32px; }
.nt-page-skeleton:not(.nt-page-skeleton--prices) article > i:nth-child(4) { width: 100%; height: 40px; margin-top: auto; }
.nt-page-skeleton--cards .nt-skeleton-heading { display: none; }
.nt-page-skeleton--section { padding: 28px 0; }
.nt-page-skeleton--section .nt-skeleton-heading { justify-items: start; }
.nt-page-skeleton--section .nt-skeleton-cards article { min-height: 180px; }
.nt-page-skeleton--prices { grid-column: 1 / -1; max-width: none; padding: 0; }
.nt-page-skeleton--prices .nt-skeleton-heading { display: none; }
.nt-page-skeleton--prices .nt-skeleton-cards { gap: 16px; }
.nt-page-skeleton--prices article { position: relative; display: flex; flex-direction: column; min-height: 180px; padding: 20px; border-color: #dde5e0; border-radius: 20px; background: #f8fbf9; }
.nt-page-skeleton--prices .nt-skeleton-price-extra { display: none; }
.nt-page-skeleton--prices article i { flex: none; margin: 0; border-radius: 6px; background: #d7e1db; }
.nt-page-skeleton--prices article i::after { background: linear-gradient(100deg, transparent 20%, #ffffff70 45%, transparent 70%); }
.nt-skeleton-price-heading { display: flex; align-items: center; gap: 8px; }
.nt-skeleton-price-mark { width: 24px; height: 24px; }
.nt-page-skeleton--prices .nt-skeleton-price-name { flex: 1; min-width: 32px; max-width: 160px; height: 16px; }
.nt-skeleton-price-invoice { width: 52px; height: 12px; }
.nt-page-skeleton--prices .nt-skeleton-price-chip { width: 108px; height: 20px; margin-top: 12px; border-radius: 10px; }
.nt-skeleton-price-bottom { display: flex; align-items: flex-end; margin-top: auto; padding-top: 24px; padding-right: 76px; }
.nt-skeleton-price-value { width: min(112px, 100%); height: 32px; }
.nt-page-skeleton--prices .nt-skeleton-price-art { position: absolute; right: 20px; bottom: 20px; width: 68px; height: 68px; border-radius: 12px; }
@media (min-width: 1440px) {
  .nt-page-skeleton--prices .nt-skeleton-cards { grid-template-columns: repeat(5, minmax(0, 1fr)); }
  .nt-page-skeleton--prices article { min-height: 184px; padding: 12px; border-radius: 16px; }
  .nt-page-skeleton--prices .nt-skeleton-price-extra { display: flex; }
  .nt-page-skeleton--prices .nt-skeleton-price-heading { display: grid; grid-template-columns: 24px minmax(0, 1fr); gap: 4px 8px; }
  .nt-page-skeleton--prices .nt-skeleton-price-invoice { grid-column: 2; }
  .nt-page-skeleton--prices .nt-skeleton-price-bottom { min-height: 76px; padding-right: 56px; }
  .nt-page-skeleton--prices .nt-skeleton-price-art { right: 8px; bottom: 12px; width: 52px; height: 48px; border-radius: 8px; }
}
@media (max-width: 1100px) { .nt-page-skeleton--prices .nt-skeleton-cards { grid-template-columns: repeat(2, minmax(0, 1fr)); } }
@keyframes nt-skeleton-shimmer { to { transform: translateX(100%); } }
@media (max-width: 640px) {
  .nt-page-skeleton { padding: 32px 20px; }
  .nt-skeleton-cards { grid-template-columns: 1fr; gap: 12px; }
  .nt-skeleton-cards article { min-height: 118px; padding: 20px; }
  .nt-page-skeleton:not(.nt-page-skeleton--prices) article > i:first-child { width: 32px; height: 32px; margin-bottom: 16px; }
  .nt-page-skeleton:not(.nt-page-skeleton--prices) article > i:nth-child(3) { height: 20px; margin-bottom: 16px; }
  .nt-skeleton-heading i:nth-child(2) { width: 80%; height: 32px; }
  .nt-page-skeleton--prices { padding: 0; }
  .nt-page-skeleton--prices .nt-skeleton-cards { grid-template-columns: 1fr; gap: 16px; }
  .nt-page-skeleton--prices article { min-height: 176px; }
}
@media (max-width: 359px) { .nt-page-skeleton--prices article { min-height: 204px; } }
@media (prefers-reduced-motion: reduce) { .nt-page-skeleton i::after { animation: none; } }
</style>
