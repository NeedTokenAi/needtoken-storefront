<template>
  <div class="nt-code">
    <div class="nt-code-bar">
      <i /><i /><i />
      <span class="ml-2 truncate">{{ title }}</span>
      <button type="button" class="ml-auto rounded px-2 py-0.5 text-[12px] text-muted-foreground hover:bg-secondary hover:text-foreground" @click="copy">
        {{ copied ? t('needtoken.common.copied') : t('needtoken.common.copy') }}
      </button>
    </div>
    <pre><code>{{ code }}</code></pre>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'

const props = defineProps<{ code: string; title?: string }>()
const { t } = useI18n()
const copied = ref(false)

const copy = async () => {
  try {
    await navigator.clipboard.writeText(props.code)
    copied.value = true
    window.setTimeout(() => { copied.value = false }, 1500)
  } catch {
    /* 剪贴板不可用时忽略，用户可手动选择复制 */
  }
}
</script>
