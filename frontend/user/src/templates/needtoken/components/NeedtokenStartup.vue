<template>
  <!-- 会员控制台 /me 专属启动骨架屏（与真实侧栏 + Bento 仪表盘 1:1 对应，刷新零闪屏） -->
  <section
    v-if="mode === 'console' && !failed"
    class="nt-startup nt-startup--console"
    :class="{ 'nt-startup--branded': branded }"
    role="status"
    aria-live="polite"
    aria-busy="true"
  >
    <aside class="nt-cs-aside" aria-hidden="true">
      <div class="nt-cs-aside-head">
        <BrandMark v-if="branded" width="24" height="24" />
        <span v-if="branded" class="nt-cs-brand" translate="no">NeedToken.AI</span>
        <i v-else class="nt-cs-brand-sk" />
      </div>
      <div class="nt-cs-nav">
        <i class="nt-cs-label" />
        <i v-for="n in 5" :key="`g1-${n}`" class="nt-cs-item" />
        <i class="nt-cs-label nt-cs-label--gap" />
        <i v-for="n in 3" :key="`g2-${n}`" class="nt-cs-item" />
      </div>
    </aside>

    <div class="nt-cs-main">
      <header class="nt-cs-topbar" aria-hidden="true">
        <i class="nt-cs-title" />
        <div class="nt-cs-topbar-right">
          <i class="nt-cs-chip" />
          <i class="nt-cs-avatar" />
        </div>
      </header>

      <div class="nt-cs-body" aria-hidden="true">
        <div class="nt-cs-kpis">
          <div v-for="idx in 4" :key="idx" class="nt-cs-kpi">
            <i class="nt-cs-kpi-label" />
            <i class="nt-cs-kpi-val" />
            <i class="nt-cs-kpi-sub" />
          </div>
        </div>
        <div class="nt-cs-bento">
          <div class="nt-cs-panel">
            <i class="nt-cs-panel-head" />
            <i class="nt-cs-panel-chart" />
            <i class="nt-cs-panel-line" />
          </div>
          <div class="nt-cs-panel">
            <i class="nt-cs-panel-head" />
            <i class="nt-cs-panel-row" />
            <i class="nt-cs-panel-row" />
            <i class="nt-cs-panel-row" />
          </div>
        </div>
      </div>

      <div v-if="slow" class="nt-startup-slow">
        <p class="nt-startup-help">{{ t('needtoken.startup.slowHint') }}</p>
        <button type="button" class="nt-startup-retry" @click="$emit('retry')">{{ t('needtoken.startup.retry') }}</button>
      </div>
      <span class="sr-only">{{ t('needtoken.startup.loading') }}</span>
    </div>
  </section>

  <!-- 店面默认启动骨架屏 / 失败提示态 -->
  <section v-else class="nt-startup" :class="{ 'nt-startup--branded': branded }" :aria-busy="!failed">
    <header class="nt-startup-header" aria-hidden="true"><div class="nt-startup-identity"><BrandMark v-if="branded" width="30" height="30" /><span v-if="branded" translate="no">NeedToken.AI</span><i v-else /></div><div class="nt-startup-nav"><i /><i /><i /></div></header>
    <div v-if="branded && !failed" class="nt-startup-catalogue" role="status" aria-live="polite">
      <div class="nt-startup-catalogue-heading" aria-hidden="true"><i /><i /></div>
      <div class="nt-startup-catalogue-toolbar" aria-hidden="true"><i /><i /></div>
      <div class="nt-startup-catalogue-filters" aria-hidden="true"><i /><i /></div>
      <NeedtokenSkeleton kind="prices" :count="4" />
      <span class="sr-only">{{ t('needtoken.startup.loading') }}</span>
      <div v-if="slow" class="nt-startup-slow"><p class="nt-startup-help">{{ t('needtoken.startup.slowHint') }}</p><button type="button" class="nt-startup-retry" @click="$emit('retry')">{{ t('needtoken.startup.retry') }}</button></div>
    </div>
    <div v-else class="nt-startup-scene" :role="failed ? 'alert' : 'status'" aria-live="polite">
      <svg v-if="branded" class="nt-startup-logo" viewBox="0 0 64 64" fill="none" aria-hidden="true"><rect x="1" y="1" width="62" height="62" rx="18" fill="#10241f" stroke="#2dd4a7" stroke-opacity=".3" stroke-width="2" /><path class="nt-startup-stroke" d="M19 45V19l26 26V19" stroke="#5eeac0" stroke-width="7" stroke-linecap="round" stroke-linejoin="round" pathLength="1" /><circle class="nt-startup-signal" cx="45" cy="19" r="4.5" fill="#cefff0" /></svg>
      <span v-else class="nt-startup-neutral" aria-hidden="true"><i /><i /><i /></span>
      <p class="nt-startup-message">{{ t(failed ? 'needtoken.startup.failed' : 'needtoken.startup.loading') }}</p>
      <p v-if="failed || slow" class="nt-startup-help">{{ t(failed ? 'needtoken.startup.failedHint' : 'needtoken.startup.slowHint') }}</p>
      <button v-if="failed || slow" type="button" class="nt-startup-retry" @click="$emit('retry')">{{ t('needtoken.startup.retry') }}</button>
    </div>
    <NeedtokenSkeleton v-if="!branded && !failed" kind="cards" />
  </section>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import BrandMark from './BrandMark.vue'
import NeedtokenSkeleton from './NeedtokenSkeleton.vue'
withDefaults(defineProps<{ branded?: boolean; failed?: boolean; slow?: boolean; mode?: 'storefront' | 'console' }>(), {
  branded: false,
  failed: false,
  slow: false,
  mode: 'storefront',
})
defineEmits<{ retry: [] }>()
const { t } = useI18n()
</script>

<style scoped>
.nt-startup { min-height: 100svh; color: var(--ui-text-primary, #0f1714); background: var(--ui-bg-page, #f5f7f6); }
.nt-startup--console { display: flex; min-height: 100svh; }
.nt-cs-aside { width: 232px; flex: none; border-right: 1px solid var(--ui-border, rgba(15, 23, 20, 0.1)); background: var(--ui-bg-elevated, #ffffff); display: flex; flex-direction: column; }
.nt-cs-aside-head { height: 56px; padding: 0 14px; border-bottom: 1px solid var(--ui-border, rgba(15, 23, 20, 0.08)); display: flex; align-items: center; gap: 8px; }
.nt-cs-brand { font: 700 15px/1.2 ui-sans-serif, system-ui, sans-serif; letter-spacing: -.03em; color: var(--ui-text-primary); }
.nt-cs-brand-sk { width: 112px; height: 18px; }
.nt-cs-nav { padding: 16px 12px; display: flex; flex-direction: column; gap: 8px; }
.nt-cs-label { width: 56px; height: 10px; margin: 2px 8px 4px; }
.nt-cs-label--gap { margin-top: 14px; }
.nt-cs-item { width: 100%; height: 34px; border-radius: 10px; }
.nt-cs-main { flex: 1; min-width: 0; display: flex; flex-direction: column; }
.nt-cs-topbar { height: 56px; padding: 0 24px; border-bottom: 1px solid var(--ui-border, rgba(15, 23, 20, 0.08)); background: var(--ui-bg-elevated, #ffffff); display: flex; align-items: center; justify-content: space-between; gap: 16px; }
.nt-cs-title { width: 120px; height: 18px; }
.nt-cs-topbar-right { display: flex; align-items: center; gap: 10px; }
.nt-cs-chip { width: 136px; height: 32px; border-radius: 10px; }
.nt-cs-avatar { width: 96px; height: 32px; border-radius: 999px; }
.nt-cs-body { width: 100%; max-width: 1440px; margin: 0 auto; padding: 20px 24px; display: flex; flex-direction: column; gap: 20px; box-sizing: border-box; }
.nt-cs-kpis { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 14px; }
.nt-cs-kpi, .nt-cs-panel { border: 1px solid var(--ui-border, rgba(15, 23, 20, 0.1)); border-radius: 16px; background: var(--ui-bg-elevated, #ffffff); padding: 18px; box-sizing: border-box; }
.nt-cs-kpi { min-height: 112px; display: flex; flex-direction: column; gap: 12px; }
.nt-cs-kpi-label { width: 48%; height: 12px; }
.nt-cs-kpi-val { width: 62%; height: 26px; }
.nt-cs-kpi-sub { width: 72%; height: 11px; }
.nt-cs-bento { display: grid; grid-template-columns: 8fr 4fr; gap: 16px; }
.nt-cs-panel { min-height: 240px; display: flex; flex-direction: column; gap: 14px; }
.nt-cs-panel-head { width: 38%; height: 16px; }
.nt-cs-panel-chart { width: 100%; height: 132px; border-radius: 12px; }
.nt-cs-panel-line { width: 60%; height: 13px; }
.nt-cs-panel-row { width: 100%; height: 48px; border-radius: 12px; }
.nt-startup--console i { position: relative; overflow: hidden; display: block; border-radius: 8px; background: var(--nt-skeleton-base, #e4ebe7); }
.nt-startup--console i::after { content: ''; position: absolute; inset: 0; background: linear-gradient(100deg, transparent 20%, var(--nt-skeleton-sheen, rgba(255, 255, 255, 0.72)) 45%, transparent 70%); transform: translateX(-100%); animation: nt-cs-shimmer 1.8s ease-in-out infinite; }
@keyframes nt-cs-shimmer { to { transform: translateX(100%); } }
@media (max-width: 1023px) { .nt-cs-aside { display: none; } .nt-cs-kpis { grid-template-columns: repeat(2, minmax(0, 1fr)); } .nt-cs-bento { grid-template-columns: 1fr; } .nt-cs-body { padding: 16px; } }
@media (max-width: 639px) { .nt-cs-kpis { grid-template-columns: 1fr; } }

.nt-startup-header { display: flex; align-items: center; justify-content: space-between; gap: 24px; width: 100%; max-width: 1320px; height: 64px; padding: 0 24px; margin: auto; }
.nt-startup-identity { display: flex; align-items: center; gap: 10px; font: 700 17px/1.5 ui-sans-serif, system-ui, sans-serif; letter-spacing: -.035em; }
.nt-startup-identity > i { width: 135px; height: 24px; border-radius: 8px; background: color-mix(in srgb, var(--ui-text-muted, #7d8a99) 12%, transparent); }
.nt-startup-nav { display: flex; gap: 20px; }
.nt-startup-nav i { width: 60px; height: 12px; border-radius: 6px; background: color-mix(in srgb, var(--ui-text-muted, #7d8a99) 10%, transparent); }
.nt-startup-catalogue { max-width: var(--nt-content-max-width, 1200px); margin: auto; padding: 48px 24px 24px; }
.nt-startup-catalogue i { display: block; border-radius: 8px; background: var(--nt-skeleton-base, #e4ebe7); }
.nt-startup-catalogue-heading, .nt-startup-catalogue-toolbar { display: flex; align-items: center; justify-content: space-between; gap: 24px; }
.nt-startup-catalogue-heading { min-height: 56px; }
.nt-startup-catalogue-heading i:first-child { width: 100px; height: 28px; }
.nt-startup-catalogue-heading i:last-child { width: min(40%, 300px); height: 40px; }
.nt-startup-catalogue-toolbar { margin-top: 24px; }
.nt-startup-catalogue-toolbar i:first-child { width: 388px; max-width: 48%; height: 50px; }
.nt-startup-catalogue-toolbar i:last-child { width: 440px; max-width: 48%; height: 34px; }
.nt-startup-catalogue-filters { display: flex; gap: 12px; margin: 20px 0 24px; }
.nt-startup-catalogue-filters i { width: 104px; height: 36px; border-radius: 18px; }
.nt-startup-slow { display: flex; align-items: center; justify-content: center; flex-wrap: wrap; gap: 16px; margin-top: 24px; }
.nt-startup-scene { display: flex; min-height: 230px; align-items: center; justify-content: center; flex-direction: column; gap: 17px; padding: 40px 24px 0; text-align: center; }
.nt-startup-logo { width: 60px; height: 60px; filter: drop-shadow(0 0 18px #2dd4a715); }
.nt-startup-stroke { stroke-dasharray: 1; stroke-dashoffset: 0; animation: nt-logo-draw 1.9s ease-in-out infinite; }
.nt-startup-signal { transform-origin: 45px 19px; animation: nt-logo-signal 1.9s ease-in-out infinite; }
.nt-startup-message { font-size: 13px; color: var(--ui-text-primary); letter-spacing: .025em; }
.nt-startup-help { max-width: 46ch; font-size: 12px; line-height: 1.8; color: var(--ui-text-muted); }
.nt-startup-retry { padding: 9px 16px; border: 1px solid #2dd4a740; border-radius: 9px; background: #2dd4a70d; color: inherit; font-size: 12px; cursor: pointer; }
.nt-startup-retry:focus-visible { outline: 2px solid #2dd4a7; outline-offset: 3px; }
.nt-startup-neutral { display: flex; gap: 6px; align-items: center; height: 42px; }
.nt-startup-neutral i { width: 7px; height: 7px; border-radius: 50%; background: currentColor; opacity: .25; animation: nt-neutral-pulse 1.3s ease-in-out infinite; }
.nt-startup-neutral i:nth-child(2) { animation-delay: .15s; }.nt-startup-neutral i:nth-child(3) { animation-delay: .3s; }
@keyframes nt-logo-draw { 0% { stroke-dashoffset: 1; opacity: .4; } 45%, 75% { stroke-dashoffset: 0; opacity: 1; } 100% { stroke-dashoffset: 0; opacity: .55; } }
@keyframes nt-logo-signal { 0%, 30%, 100% { opacity: .35; transform: scale(.8); } 55%, 75% { opacity: 1; transform: scale(1); } }
@keyframes nt-neutral-pulse { 50% { opacity: .75; } }
@media (max-width: 1100px) { .nt-startup-catalogue-toolbar { align-items: stretch; flex-direction: column; gap: 16px; } .nt-startup-catalogue-toolbar i:first-child { max-width: 100%; } .nt-startup-catalogue-toolbar i:last-child { align-self: flex-end; max-width: 100%; } }
@media (max-width: 640px) { .nt-startup-catalogue { padding: 32px 20px 24px; } .nt-startup-catalogue-heading { gap: 16px; } .nt-startup-catalogue-heading i:last-child { width: 60%; height: 28px; } .nt-startup-catalogue-toolbar { margin-top: 20px; } .nt-startup-catalogue-toolbar i:first-child { width: 100%; } .nt-startup-catalogue-toolbar i:last-child { width: 84%; } .nt-startup-catalogue-filters { gap: 4px; margin: 16px 0 20px; } .nt-startup-header { padding: 0 20px; } .nt-startup-nav { gap: 10px; }.nt-startup-nav i { width: 25px; }.nt-startup-scene { min-height: 186px; padding-top: 24px; gap: 14px; }.nt-startup-logo { width: 48px; height: 48px; } }
@media (prefers-reduced-motion: reduce) { .nt-startup-stroke, .nt-startup-signal, .nt-startup-neutral i, .nt-startup--console i::after { animation: none; } }
</style>

<style>
.nt-startup--branded {
  --ui-bg-page: #f5f7f6;
  --ui-bg-elevated: #ffffff;
  --ui-text-primary: #0f1714;
  --ui-text-muted: #667770;
  --ui-border: rgba(15, 23, 20, 0.1);
  --nt-skeleton-base: #e4ebe7;
  --nt-skeleton-sheen: rgba(255, 255, 255, 0.72);
  background: radial-gradient(ellipse at 50% 0, rgba(13, 148, 115, 0.08), transparent 55%), #f5f7f6;
  color: #0f1714;
}
.nt-startup--console.nt-startup--branded {
  background: #f5f7f6;
}
html.dark .nt-startup--branded {
  --ui-bg-page: #07090d;
  --ui-bg-elevated: #0e1218;
  --ui-text-primary: #e8edf3;
  --ui-text-muted: #7d8a99;
  --ui-border: rgba(148, 163, 184, 0.16);
  --nt-skeleton-base: #182028;
  --nt-skeleton-sheen: rgba(255, 255, 255, 0.08);
  background: radial-gradient(ellipse at 50% 0, #2dd4a70b, transparent 55%), #07090d;
  color: #e8edf3;
}
html.dark .nt-startup--console.nt-startup--branded {
  background: #07090d;
}
</style>
