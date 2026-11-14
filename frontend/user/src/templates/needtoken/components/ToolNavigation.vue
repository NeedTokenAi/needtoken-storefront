<template>
  <nav class="tool-navigation" :aria-label="t('needtoken.radar.tools')">
    <RouterLink v-for="item in links" :key="item.path" :to="item.path" :aria-current="route.path === item.path ? 'page' : undefined"><component :is="item.icon" :size="16" :stroke-width="1.5" class="tool-navigation-icon" aria-hidden="true" /><span>{{ t(item.label) }}</span></RouterLink>
  </nav>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { Network, Radar, ScanLine } from 'lucide-vue-next'
const { t } = useI18n()
const route = useRoute()
const links = [
  { path: '/tools/radar', label: 'needtoken.toolsPolish.nav.radar', icon: Radar },
  { path: '/tools/environment', label: 'needtoken.toolsPolish.nav.network', icon: Network },
  { path: '/tools/codex-account', label: 'needtoken.toolsPolish.nav.account', icon: ScanLine },
]
</script>

<style scoped>
.tool-navigation { display: flex; width: fit-content; max-width: 100%; align-items: stretch; gap: 4px; margin-bottom: 32px; padding: 4px; border: 1px solid var(--ui-border); border-radius: 12px; background: var(--ui-bg-elevated); color: var(--ui-text-muted); font-size: 13px; line-height: 20px; }
a { display: inline-flex; min-width: 0; min-height: 40px; justify-content: center; align-items: center; gap: 8px; padding: 8px 12px; border-radius: 8px; white-space: nowrap; transition: color var(--nt-motion-fast, 120ms) var(--nt-motion-ease, ease-out), background-color var(--nt-motion-fast, 120ms) var(--nt-motion-ease, ease-out), transform var(--nt-motion-fast, 120ms) var(--nt-motion-ease, ease-out); }
a[aria-current] { color: var(--ui-text-primary); background: var(--ui-bg-soft); }
a[aria-current] .tool-navigation-icon { color: var(--ui-accent); }
a:hover { color: var(--ui-text-primary); background: var(--ui-bg-soft); }
a:active { transform: translateY(1px); }
.tool-navigation-icon { flex: none; color: var(--ui-text-muted); transition: color var(--nt-motion-fast, 120ms) var(--nt-motion-ease, ease-out), transform var(--nt-motion-fast, 120ms) var(--nt-motion-ease, ease-out); }
a:hover .tool-navigation-icon, a:focus-visible .tool-navigation-icon { color: var(--ui-accent); transform: translateY(-1px); }
a:focus-visible { outline: 2px solid var(--ui-accent); outline-offset: 3px; }
@media (max-width: 640px) { .tool-navigation { margin-bottom: 24px; font-size: 12px; } a { min-height: 44px; padding-inline: 10px; } }
@media (max-width: 480px) { .tool-navigation { width: 100%; display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 0; } a { gap: 6px; padding-inline: 4px; white-space: normal; text-align: center; } .tool-navigation-icon { width: 14px; height: 14px; } }
@media (prefers-reduced-motion: reduce) { a, .tool-navigation-icon { transition: none; } a:active, a:hover .tool-navigation-icon, a:focus-visible .tool-navigation-icon { transform: none; } }
</style>
