<template>
  <nav
    ref="railEl"
    class="tool-navigation"
    :aria-label="t('needtoken.checkCenter.navLabel')"
  >
    <span
      class="nt-sliding-thumb tool-nav-thumb"
      :class="{ 'is-ready': indicatorReady }"
      :style="indicatorStyle"
      aria-hidden="true"
    />
    <RouterLink
      v-for="item in links"
      :key="item.path"
      :to="item.path"
      class="tool-nav-link"
      :data-tool-active="isCurrent(item.path) ? 'true' : undefined"
      :aria-current="isCurrent(item.path) ? 'page' : undefined"
    >
      <ProductBrandMark :brand="item.brand" class="tool-nav-brand" />
      <span>{{ t(item.label) }}</span>
    </RouterLink>
  </nav>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import ProductBrandMark from './ProductBrandMark.vue'
import { useSlidingIndicator } from '../composables/useSlidingIndicator'

const { t, locale } = useI18n()
const route = useRoute()
const railEl = ref<HTMLElement | null>(null)

const links = [
  { path: '/tools/codex-account', label: 'needtoken.checkCenter.tabs.gpt', brand: 'gpt' as const },
  { path: '/tools/environment', label: 'needtoken.checkCenter.tabs.claude', brand: 'claude' as const },
]

const isCurrent = (path: string) => route.path === path || route.path.startsWith(`${path}/`)

const { indicatorStyle, indicatorReady } = useSlidingIndicator(
  railEl,
  '[data-tool-active="true"]',
  [() => route.path, () => locale.value],
)
</script>

<style scoped>
.tool-navigation {
  position: relative;
  display: inline-flex;
  align-items: center;
  gap: 4px;
  max-width: 100%;
  margin-bottom: 24px;
  padding: 4px;
  border: 1px solid var(--ui-border);
  border-radius: 14px;
  background: var(--nt-nav-rail-bg, var(--ui-bg-elevated));
  overflow-x: auto;
  scrollbar-width: none;
  scroll-snap-type: inline proximity;
}

.tool-navigation::-webkit-scrollbar {
  display: none;
}

.tool-nav-thumb {
  border-radius: 10px;
  background: var(--nt-nav-thumb-bg, var(--ui-bg-elevated));
  box-shadow: var(--nt-nav-thumb-shadow);
}

.tool-nav-link {
  position: relative;
  z-index: 1;
  flex: none;
  scroll-snap-align: start;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  min-height: 38px;
  padding: 7px 16px;
  border: 1px solid transparent;
  border-radius: 10px;
  color: var(--ui-text-muted);
  font-size: 13.5px;
  font-weight: 500;
  line-height: 1.35;
  white-space: nowrap;
  transition: color 160ms var(--nt-motion-ease, ease-out);
}

.tool-nav-link:hover {
  color: var(--ui-text-primary);
}

.tool-nav-link[aria-current="page"] {
  color: var(--ui-text-primary);
}

/* Non-color active affordance (WCAG 1.4.1 & forced-colors safe) */
.tool-nav-link[aria-current="page"]::after {
  content: '';
  position: absolute;
  inset: auto 16px 2px;
  border-block-end: 2px solid var(--ui-accent);
  border-radius: 999px;
}

.tool-nav-brand {
  width: 16px;
  height: 16px;
  opacity: 0.72;
  transition: opacity 160ms ease, transform 360ms var(--nt-ease-spring, ease-out);
}

.tool-nav-brand :deep(.product-brand-symbol) {
  width: 16px;
  height: 16px;
}

.tool-nav-link:hover .tool-nav-brand,
.tool-nav-link[aria-current="page"] .tool-nav-brand {
  opacity: 1;
}

@media (prefers-reduced-motion: no-preference) {
  .tool-nav-link:hover .tool-nav-brand {
    transform: scale(1.1);
  }
  .tool-nav-link:active {
    transform: scale(0.97);
    transition-duration: var(--nt-duration-press, 80ms);
  }
}

.tool-nav-link:focus-visible {
  outline: 2px solid var(--ui-accent);
  outline-offset: 2px;
}

@media (max-width: 640px) {
  .tool-navigation {
    margin-bottom: 20px;
  }
  .tool-nav-link {
    min-height: 40px;
    padding: 7px 14px;
    font-size: 13px;
  }
}
</style>
