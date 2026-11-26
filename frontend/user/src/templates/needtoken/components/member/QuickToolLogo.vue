<template>
  <div class="nt-quick-tool-logo" :style="containerStyle">
    <img
      v-if="logoUrl && !imgBroken"
      :src="logoUrl"
      :alt="name"
      class="h-7 w-7 object-contain"
      @error="imgBroken = true"
    />

    <!-- OpenAI / Codex CLI -->
    <svg
      v-else-if="resolvedBrand === 'codex'"
      :viewBox="productBrandMarks.gpt.viewBox"
      class="h-6 w-6"
      fill="currentColor"
      aria-hidden="true"
    >
      <path v-for="(d, idx) in productBrandMarks.gpt.paths" :key="idx" :d="d" />
    </svg>

    <!-- Anthropic / Claude Code -->
    <svg
      v-else-if="resolvedBrand === 'claude'"
      :viewBox="productBrandMarks.claude.viewBox"
      class="h-6 w-6"
      fill="currentColor"
      aria-hidden="true"
    >
      <path v-for="(d, idx) in productBrandMarks.claude.paths" :key="idx" :d="d" />
    </svg>

    <!-- Google Gemini CLI -->
    <svg
      v-else-if="resolvedBrand === 'gemini'"
      :viewBox="productBrandMarks.gemini.viewBox"
      class="h-6 w-6"
      fill="currentColor"
      aria-hidden="true"
    >
      <path v-for="(d, idx) in productBrandMarks.gemini.paths" :key="idx" :d="d" />
    </svg>

    <!-- Grok -->
    <svg
      v-else-if="resolvedBrand === 'grok'"
      :viewBox="productBrandMarks.grok.viewBox"
      class="h-6 w-6"
      fill="currentColor"
      fill-rule="evenodd"
      aria-hidden="true"
    >
      <path v-for="(d, idx) in productBrandMarks.grok.paths" :key="idx" :d="d" />
    </svg>

    <!-- CC Switch: 双轨切换矢量徽标 -->
    <svg v-else-if="resolvedBrand === 'ccswitch'" viewBox="0 0 24 24" class="h-6 w-6" fill="none" aria-hidden="true">
      <rect x="2.5" y="4.5" width="19" height="6.5" rx="3.25" stroke="currentColor" stroke-width="1.8" />
      <circle cx="17.25" cy="7.75" r="2" fill="currentColor" />
      <rect x="2.5" y="13" width="19" height="6.5" rx="3.25" stroke="currentColor" stroke-width="1.8" opacity="0.72" />
      <circle cx="6.75" cy="16.25" r="2" fill="currentColor" />
    </svg>

    <!-- OpenCode: 几何终端内核徽标 -->
    <svg v-else-if="resolvedBrand === 'opencode'" viewBox="0 0 24 24" class="h-6 w-6" fill="none" aria-hidden="true">
      <rect x="3" y="3" width="18" height="18" rx="4.5" stroke="currentColor" stroke-width="1.9" />
      <path d="M8.5 9.25L11.5 12L8.5 14.75" stroke="currentColor" stroke-width="1.9" stroke-linecap="round" stroke-linejoin="round" />
      <path d="M13 14.75H16" stroke="currentColor" stroke-width="1.9" stroke-linecap="round" />
    </svg>

    <!-- Cursor: 立体等轴测立方光标徽标 -->
    <svg v-else-if="resolvedBrand === 'cursor'" viewBox="0 0 24 24" class="h-6 w-6" fill="none" aria-hidden="true">
      <path d="M12 2.2L20.5 7.1V16.9L12 21.8L3.5 16.9V7.1L12 2.2Z" stroke="currentColor" stroke-width="1.6" stroke-linejoin="round" />
      <path d="M12 2.2L20.5 7.1L12 12L3.5 7.1L12 2.2Z" fill="currentColor" opacity="0.22" />
      <path d="M12 12V21.8L20.5 16.9V7.1L12 12Z" fill="currentColor" opacity="0.72" />
      <path d="M9.5 8.5L15.8 11.4L12.3 12.8L10.9 16.2L9.5 8.5Z" fill="currentColor" />
    </svg>

    <!-- Cline / Roo Code: 自主编码机器人矢量徽标 -->
    <svg v-else-if="resolvedBrand === 'cline'" viewBox="0 0 24 24" class="h-6 w-6" fill="none" aria-hidden="true">
      <rect x="4" y="6" width="16" height="12.5" rx="4" stroke="currentColor" stroke-width="1.8" />
      <path d="M12 3V6" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" />
      <circle cx="9.2" cy="11.8" r="1.4" fill="currentColor" />
      <circle cx="14.8" cy="11.8" r="1.4" fill="currentColor" />
      <path d="M9.2 15.2C10.1 15.9 11 16.2 12 16.2C13 16.2 13.9 15.9 14.8 15.2" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" />
    </svg>

    <!-- T3 Chat / Cherry Studio: 多模型对话星标矢量徽标 -->
    <svg v-else-if="resolvedBrand === 't3'" viewBox="0 0 24 24" class="h-6 w-6" fill="none" aria-hidden="true">
      <path d="M5 6.5C5 4.84 6.34 3.5 8 3.5H16C17.66 3.5 19 4.84 19 6.5V13.5C19 15.16 17.66 16.5 16 16.5H10.5L6.5 20V16.5H8C6.34 16.5 5 15.16 5 13.5V6.5Z" stroke="currentColor" stroke-width="1.8" stroke-linejoin="round" />
      <path d="M12 7L12.9 9.1L15 10L12.9 10.9L12 13L11.1 10.9L9 10L11.1 9.1L12 7Z" fill="currentColor" />
    </svg>

    <!-- 通用终端 -->
    <svg v-else-if="resolvedBrand === 'terminal'" viewBox="0 0 24 24" class="h-6 w-6" fill="none" aria-hidden="true">
      <rect x="3" y="4.5" width="18" height="15" rx="3.5" stroke="currentColor" stroke-width="1.8" />
      <path d="M7.5 10L10.5 12.5L7.5 15" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" />
      <path d="M12.5 15H16.5" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" />
    </svg>

    <!-- 兜底：未知自定义工具且未填 Logo URL 时展示字母标识 -->
    <span v-else class="text-sm font-black tracking-tight">{{ fallbackGlyph }}</span>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { productBrandMarks } from '../../lib/productBrandMarks'

const props = defineProps<{
  name: string
  slug?: string
  iconKey?: string
  logoUrl?: string
}>()

const imgBroken = ref(false)
watch(() => props.logoUrl, () => {
  imgBroken.value = false
})

type BuiltinToolBrand = 'codex' | 'claude' | 'gemini' | 'grok' | 'ccswitch' | 'opencode' | 'cursor' | 'cline' | 't3' | 'terminal' | 'fallback'

const resolvedBrand = computed<BuiltinToolBrand>(() => {
  const hint = `${props.iconKey || ''} ${props.slug || ''} ${props.name || ''}`.toLowerCase()
  if (hint.includes('ccswitch') || hint.includes('cc-switch') || hint.includes('cc switch')) return 'ccswitch'
  if (hint.includes('codex') || hint.includes('openai') || hint.includes('chatgpt')) return 'codex'
  if (hint.includes('claude') || hint.includes('anthropic')) return 'claude'
  if (hint.includes('gemini') || hint.includes('google')) return 'gemini'
  if (hint.includes('grok') || hint.includes('xai')) return 'grok'
  if (hint.includes('opencode')) return 'opencode'
  if (hint.includes('cursor')) return 'cursor'
  if (hint.includes('cline') || hint.includes('roo')) return 'cline'
  if (hint.includes('t3') || hint.includes('cherry')) return 't3'
  if (hint.includes('terminal') || hint.includes('cli')) return 'terminal'
  return 'fallback'
})

const containerStyle = computed<Record<string, string>>(() => {
  switch (resolvedBrand.value) {
    case 'codex':
      return { background: 'rgba(13, 148, 115, 0.12)', color: 'var(--nt-brand-openai, #173026)' }
    case 'claude':
      return { background: 'rgba(194, 93, 63, 0.12)', color: 'var(--nt-brand-claude, #c25d3f)' }
    case 'gemini':
      return { background: 'rgba(59, 126, 240, 0.12)', color: 'var(--nt-brand-gemini, #3b7ef0)' }
    case 'grok':
      return { background: 'rgba(24, 35, 31, 0.1)', color: 'var(--nt-brand-grok, #18231f)' }
    case 'ccswitch':
      return { background: 'rgba(13, 148, 115, 0.14)', color: 'var(--ui-accent, #0d9473)' }
    case 'opencode':
      return { background: 'rgba(124, 77, 147, 0.12)', color: '#7c4d93' }
    case 'cursor':
      return { background: 'rgba(71, 91, 116, 0.12)', color: '#334155' }
    case 'cline':
      return { background: 'rgba(14, 116, 144, 0.12)', color: '#0e7490' }
    case 't3':
      return { background: 'rgba(217, 70, 119, 0.12)', color: '#be185d' }
    default:
      return { background: 'var(--ui-accent-soft)', color: 'var(--ui-accent)' }
  }
})

const fallbackGlyph = computed(() => {
  const clean = (props.name || 'AI').trim()
  return clean.slice(0, 2).toUpperCase()
})
</script>

<style scoped>
.nt-quick-tool-logo {
  display: grid;
  place-items: center;
  width: 48px;
  height: 48px;
  border-radius: 14px;
  border: 1px solid var(--ui-border);
  flex: none;
}
</style>
