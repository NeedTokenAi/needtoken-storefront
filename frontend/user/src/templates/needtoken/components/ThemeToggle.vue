<template>
  <div
    class="nt-theme-toggle"
    :class="{ 'nt-theme-toggle--menu': variant === 'menu' }"
    role="group"
    :aria-label="t('needtoken.theme.label')"
    data-nt-effect="off"
  >
    <span class="nt-theme-toggle__thumb" :class="{ 'is-dark': theme === 'dark' }" aria-hidden="true" />
    <button
      type="button"
      class="nt-theme-toggle__btn"
      :class="{ 'is-active': theme === 'light' }"
      :aria-pressed="theme === 'light'"
      :aria-label="t('needtoken.theme.light')"
      :title="t('needtoken.theme.light')"
      @click="choose('light', $event)"
    >
      <Sun class="nt-theme-toggle__icon" :size="15" :stroke-width="1.8" aria-hidden="true" />
      <span v-if="variant === 'menu'" class="nt-theme-toggle__text">{{ t('needtoken.theme.lightShort') }}</span>
    </button>
    <button
      type="button"
      class="nt-theme-toggle__btn"
      :class="{ 'is-active': theme === 'dark' }"
      :aria-pressed="theme === 'dark'"
      :aria-label="t('needtoken.theme.dark')"
      :title="t('needtoken.theme.dark')"
      @click="choose('dark', $event)"
    >
      <Moon class="nt-theme-toggle__icon" :size="15" :stroke-width="1.8" aria-hidden="true" />
      <span v-if="variant === 'menu'" class="nt-theme-toggle__text">{{ t('needtoken.theme.darkShort') }}</span>
    </button>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { Moon, Sun } from 'lucide-vue-next'
import { useTheme, type ResolvedTheme } from '../../../utils/theme'
import { runThemeTransition } from '../lib/themeTransition'

withDefaults(defineProps<{ variant?: 'pill' | 'menu' }>(), { variant: 'pill' })

const { t } = useI18n()
const { theme, setTheme } = useTheme()

function choose(target: ResolvedTheme, event: MouseEvent) {
  if (theme.value === target) return
  const el = event.currentTarget instanceof HTMLElement ? event.currentTarget : null
  const rect = el?.getBoundingClientRect()
  const origin = event.clientX || event.clientY
    ? { clientX: event.clientX, clientY: event.clientY }
    : rect
      ? { clientX: rect.left + rect.width / 2, clientY: rect.top + rect.height / 2 }
      : null
  runThemeTransition(() => {
    setTheme(target)
  }, origin)
}
</script>

<style scoped>
.nt-theme-toggle {
  position: relative;
  display: inline-grid;
  grid-template-columns: repeat(2, 1fr);
  align-items: center;
  padding: 3px;
  border: 1px solid var(--ui-border);
  border-radius: 999px;
  background: var(--ui-bg-soft);
  box-shadow: var(--nt-inner-highlight);
}

.nt-theme-toggle--menu {
  width: 100%;
  border-radius: 10px;
}

.nt-theme-toggle__thumb {
  position: absolute;
  top: 3px;
  bottom: 3px;
  left: 3px;
  width: calc(50% - 3px);
  border-radius: 999px;
  background: var(--ui-bg-elevated);
  border: 1px solid var(--ui-border);
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.14);
  transform: translateX(0);
  pointer-events: none;
  transition: transform 360ms var(--nt-ease-spring, cubic-bezier(0.34, 1.4, 0.64, 1)), background-color 160ms ease, border-color 160ms ease;
}

.nt-theme-toggle--menu .nt-theme-toggle__thumb {
  border-radius: 7px;
}

.nt-theme-toggle__thumb.is-dark {
  transform: translateX(100%);
}

.nt-theme-toggle__btn {
  position: relative;
  z-index: 1;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  width: 30px;
  height: 28px;
  border-radius: 999px;
  color: var(--ui-text-secondary);
  opacity: 0.76;
  cursor: pointer;
  transition: color 160ms ease, opacity 160ms ease;
}

.nt-theme-toggle--menu .nt-theme-toggle__btn {
  width: auto;
  height: 34px;
  padding-inline: 10px;
  border-radius: 7px;
  font-size: 12px;
  font-weight: 500;
}

.nt-theme-toggle__btn:hover {
  color: var(--ui-text-primary);
  opacity: 1;
}

.nt-theme-toggle__btn.is-active {
  color: var(--ui-text-primary);
  opacity: 1;
}

.nt-theme-toggle__icon {
  transition: transform 480ms var(--nt-ease-spring, cubic-bezier(0.34, 1.4, 0.64, 1));
}

.nt-theme-toggle__btn.is-active .nt-theme-toggle__icon {
  transform: rotate(0deg) scale(1.04);
  color: var(--ui-accent);
}

.nt-theme-toggle__btn:not(.is-active) .nt-theme-toggle__icon {
  transform: rotate(-16deg) scale(0.92);
}

.nt-theme-toggle__text {
  white-space: nowrap;
}

@media (prefers-reduced-motion: reduce) {
  .nt-theme-toggle__thumb,
  .nt-theme-toggle__icon {
    transition: none;
  }
  .nt-theme-toggle__btn:not(.is-active) .nt-theme-toggle__icon,
  .nt-theme-toggle__btn.is-active .nt-theme-toggle__icon {
    transform: none;
  }
}
</style>
