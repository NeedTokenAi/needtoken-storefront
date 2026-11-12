<template>
  <div class="tool-board">
    <header>
      <p>{{ t('needtoken.codexCheck.eyebrow') }}</p>
      <h1>{{ t('needtoken.codexCheck.title') }}</h1>
      <p>{{ t('needtoken.codexCheck.intro') }}</p>
    </header>
    <section class="tool-panel" aria-live="polite">
      <div class="tool-panel-head">
        <h2>{{ t('needtoken.codexCheck.board') }}</h2>
        <span>{{ t('needtoken.codexCheck.unavailableTitle') }}</span>
      </div>
      <p role="status">{{ t('needtoken.codexCheck.unavailableBody') }}</p>
      <table>
        <caption class="sr-only">{{ t('needtoken.codexCheck.board') }}</caption>
        <thead><tr><th>{{ t('needtoken.codexCheck.item') }}</th><th>{{ t('needtoken.codexCheck.result') }}</th></tr></thead>
        <tbody>
          <tr v-for="row in rows" :key="row">
            <td>{{ t(`needtoken.codexCheck.rows.${row}`) }}</td>
            <td>{{ t('needtoken.codexCheck.notChecked') }}</td>
          </tr>
        </tbody>
      </table>
    </section>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { usePageSeo } from '../../composables/usePageSeo'

const { t } = useI18n()
const rows = ['subscription', 'open', 'upgrade']
usePageSeo({ title: () => t('needtoken.codexCheck.title'), description: () => t('needtoken.codexCheck.intro'), canonicalPath: () => '/tools/codex-account' })
</script>

<style scoped>
.tool-board { width: min(920px, calc(100% - 32px)); margin: 28px auto 12px; }
.tool-board header p:first-child { color: var(--ui-accent); font-size: 12px; font-weight: 650; letter-spacing: .04em; }
.tool-board h1 { margin-top: 8px; font-size: clamp(28px, 4vw, 40px); font-weight: 750; letter-spacing: -.04em; }
.tool-board header p:last-child { max-width: 68ch; margin-top: 12px; color: var(--ui-text-secondary); font-size: 14px; line-height: 1.8; }
.tool-panel { margin-top: 22px; padding: 18px; border: 1px solid var(--ui-border); border-radius: 18px; background: rgba(255,255,255,.03); animation: tool-in .28s ease; }
.tool-panel-head { display: flex; align-items: center; justify-content: space-between; gap: 12px; }
.tool-panel-head h2 { font-size: 16px; font-weight: 700; }
.tool-panel-head span { color: #e2b340; font-size: 12px; font-weight: 650; }
.tool-panel > p { margin-top: 12px; color: var(--ui-text-muted); font-size: 13px; line-height: 1.7; }
.tool-panel table { width: 100%; margin-top: 14px; border-collapse: collapse; font-size: 13px; }
.tool-panel th, .tool-panel td { padding: 10px 8px; border-bottom: 1px solid var(--ui-border); text-align: left; }
.tool-panel th { color: var(--ui-text-muted); font-size: 11px; font-weight: 650; }
.sr-only { position: absolute; width: 1px; height: 1px; padding: 0; margin: -1px; overflow: hidden; clip: rect(0,0,0,0); white-space: nowrap; border: 0; }
@keyframes tool-in { from { opacity: 0; transform: translateY(8px); } to { opacity: 1; transform: none; } }
@media (prefers-reduced-motion: reduce) { .tool-panel { animation: none; } }
@media (max-width: 640px) { .tool-board { width: min(100% - 28px, 920px); margin-top: 18px; } .tool-panel-head { align-items: flex-start; flex-direction: column; } }
</style>
