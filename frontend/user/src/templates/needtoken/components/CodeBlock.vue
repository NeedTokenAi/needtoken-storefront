<template>
  <div class="nt-code">
    <div class="nt-code-bar">
      <i /><i /><i />
      <span class="ml-2 truncate">{{ title }}</span>
      <button type="button" class="code-copy ml-auto" :disabled="copying" :aria-busy="copying" @click="copy">
        <Check v-if="copyState === 'copied'" :size="14" aria-hidden="true" /><Copy v-else :size="14" aria-hidden="true" />
        {{ t(copyState === 'copied' ? 'needtoken.common.copied' : 'needtoken.common.copy') }}
      </button>
    </div>
    <pre><code>{{ code }}</code></pre>
    <p v-if="copyState === 'failed'" class="code-copy-feedback" role="status">{{ t('needtoken.common.copyFailed') }}</p>
  </div>
</template>

<script setup lang="ts">
import { onUnmounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { Check, Copy } from 'lucide-vue-next'

const props = defineProps<{ code: string; title?: string }>()
const { t } = useI18n()
const copyState = ref<'idle' | 'copied' | 'failed'>('idle')
const copying = ref(false)
let feedbackTimer: ReturnType<typeof setTimeout> | undefined
let generation = 0
let disposed = false
function clearFeedback() {
  generation++
  if (feedbackTimer !== undefined) clearTimeout(feedbackTimer)
  feedbackTimer = undefined
  copyState.value = 'idle'
  copying.value = false
}
watch(() => props.code, clearFeedback)
onUnmounted(() => { disposed = true; clearFeedback() })

const copy = async () => {
  if (copying.value) return
  clearFeedback()
  const request = ++generation
  copying.value = true
  try {
    await navigator.clipboard.writeText(props.code)
    if (disposed || request !== generation) return
    copyState.value = 'copied'
    feedbackTimer = setTimeout(() => { copyState.value = 'idle'; feedbackTimer = undefined }, 1800)
  } catch {
    if (!disposed && request === generation) copyState.value = 'failed'
  } finally {
    if (!disposed && request === generation) copying.value = false
  }
}
</script>

<style scoped>
.code-copy { display: inline-flex; align-items: center; justify-content: center; gap: 8px; min-height: 32px; padding: 4px 8px; border: 1px solid transparent; border-radius: 8px; color: var(--ui-text-secondary); font-size: 12px; white-space: nowrap; transition: color var(--nt-motion-fast) var(--nt-motion-ease), background-color var(--nt-motion-fast) var(--nt-motion-ease), transform var(--nt-motion-fast) var(--nt-motion-ease); }
.code-copy:hover:not(:disabled), .code-copy:focus-visible { background: var(--ui-bg-soft); color: var(--ui-text-primary); }
.code-copy:active:not(:disabled) { transform: translateY(1px); }
.code-copy:disabled { cursor: wait; opacity: .65; }
.code-copy-feedback { padding: 12px 16px; border-top: 1px solid var(--ui-border); color: var(--nt-warning-ink); font-size: 12px; line-height: 1.7; }
@media (prefers-reduced-motion: reduce) { .code-copy { transition: none; }.code-copy:active:not(:disabled) { transform: none; } }
</style>
