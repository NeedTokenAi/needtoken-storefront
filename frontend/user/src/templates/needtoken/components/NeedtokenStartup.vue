<template>
  <section class="nt-startup" :class="{ 'nt-startup--branded': branded }" :aria-busy="!failed">
    <header class="nt-startup-header" aria-hidden="true"><div class="nt-startup-identity"><img v-if="branded" src="/needtoken-icon.svg" alt="" width="30" height="30" /><span v-if="branded" translate="no">NeedToken.AI</span><i v-else /></div><div class="nt-startup-nav"><i /><i /><i /></div></header>
    <div class="nt-startup-scene" :role="failed ? 'alert' : 'status'" aria-live="polite">
      <svg v-if="branded" class="nt-startup-logo" viewBox="0 0 64 64" fill="none" aria-hidden="true"><rect x="1" y="1" width="62" height="62" rx="18" fill="#10241f" stroke="#2dd4a7" stroke-opacity=".3" stroke-width="2" /><path class="nt-startup-stroke" d="M19 45V19l26 26V19" stroke="#5eeac0" stroke-width="7" stroke-linecap="round" stroke-linejoin="round" pathLength="1" /><circle class="nt-startup-signal" cx="45" cy="19" r="4.5" fill="#cefff0" /></svg>
      <span v-else class="nt-startup-neutral" aria-hidden="true"><i /><i /><i /></span>
      <p class="nt-startup-message">{{ t(failed ? 'needtoken.startup.failed' : 'needtoken.startup.loading') }}</p>
      <p v-if="failed || slow" class="nt-startup-help">{{ t(failed ? 'needtoken.startup.failedHint' : 'needtoken.startup.slowHint') }}</p>
      <button v-if="failed || slow" type="button" class="nt-startup-retry" @click="$emit('retry')">{{ t('needtoken.startup.retry') }}</button>
    </div>
    <NeedtokenSkeleton v-if="!failed" kind="cards" />
  </section>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import NeedtokenSkeleton from './NeedtokenSkeleton.vue'
withDefaults(defineProps<{ branded?: boolean; failed?: boolean; slow?: boolean }>(), { branded: false, failed: false, slow: false })
defineEmits<{ retry: [] }>()
const { t } = useI18n()
</script>

<style scoped>
.nt-startup { min-height: 100svh; color: var(--ui-text-primary, #1d1d1f); background: var(--ui-bg-page, #f5f5f7); }
.nt-startup--branded { --ui-bg-page: #07090d; --ui-bg-elevated: #0e1218; --ui-text-primary: #e8edf3; --ui-text-muted: #7d8a99; --ui-border: #94a3b829; background: radial-gradient(ellipse at 50% 0, #2dd4a70b, transparent 55%), #07090d; color: #e8edf3; }
.nt-startup-header { display: flex; align-items: center; justify-content: space-between; gap: 24px; width: 100%; max-width: 1200px; height: 64px; padding: 0 24px; margin: auto; }
.nt-startup-identity { display: flex; align-items: center; gap: 10px; font: 700 17px/1.5 ui-sans-serif, system-ui, sans-serif; letter-spacing: -.035em; }
.nt-startup-identity > i { width: 135px; height: 24px; border-radius: 8px; background: color-mix(in srgb, var(--ui-text-muted, #7d8a99) 12%, transparent); }
.nt-startup-nav { display: flex; gap: 20px; }
.nt-startup-nav i { width: 60px; height: 12px; border-radius: 6px; background: color-mix(in srgb, var(--ui-text-muted, #7d8a99) 10%, transparent); }
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
@media (max-width: 640px) { .nt-startup-header { padding: 0 20px; } .nt-startup-nav { gap: 10px; }.nt-startup-nav i { width: 25px; }.nt-startup-scene { min-height: 186px; padding-top: 24px; gap: 14px; }.nt-startup-logo { width: 48px; height: 48px; } }
@media (prefers-reduced-motion: reduce) { .nt-startup-stroke, .nt-startup-signal, .nt-startup-neutral i { animation: none; } }
</style>
