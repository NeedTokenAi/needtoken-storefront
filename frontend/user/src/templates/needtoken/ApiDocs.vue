<template>
  <div class="mx-auto w-full max-w-[1000px] px-6 pb-10 pt-12">
    <header>
      <h1 class="text-[32px] font-bold text-foreground">{{ t('needtoken.docs.title') }}</h1>
      <p class="mt-2 max-w-[70ch] text-[15px] text-muted-foreground">{{ t('needtoken.docs.subtitle') }}</p>
    </header>

    <section id="base-url" class="mt-10">
      <h2 class="text-[22px] font-semibold text-foreground">{{ t('needtoken.docs.baseUrl.title') }}</h2>
      <p class="mt-2 text-[14px] text-muted-foreground">{{ t('needtoken.docs.baseUrl.desc') }}</p>
      <CodeBlock class="mt-4" title="baseURL" :code="API_BASE_URL" />
    </section>

    <section id="models" class="mt-10">
      <h2 class="text-[22px] font-semibold text-foreground">{{ t('needtoken.docs.models.title') }}</h2>
      <p class="mt-2 text-[14px] text-muted-foreground">{{ t('needtoken.docs.models.desc') }}</p>
      <div class="mt-4 flex flex-wrap gap-2">
        <button
          v-for="m in DEFAULT_MODELS"
          :key="m"
          type="button"
          class="nt-mono rounded-md border px-3 py-1.5 text-[13px]"
          :class="m === model ? 'border-primary text-primary' : 'border-border text-foreground hover:bg-secondary'"
          @click="model = m"
        >{{ m }}</button>
      </div>
    </section>

    <section id="auth" class="mt-10">
      <h2 class="text-[22px] font-semibold text-foreground">{{ t('needtoken.docs.auth.title') }}</h2>
      <p class="mt-2 text-[14px] text-muted-foreground">{{ t('needtoken.docs.auth.desc') }}</p>
      <CodeBlock class="mt-4" title="shell" :code="exportKeySnippet()" />
    </section>

    <section id="curl" class="mt-10">
      <h2 class="text-[22px] font-semibold text-foreground">curl</h2>
      <CodeBlock class="mt-4" title="shell" :code="curlSnippet(model)" />
    </section>

    <section id="python" class="mt-10">
      <h2 class="text-[22px] font-semibold text-foreground">Python (openai SDK)</h2>
      <CodeBlock class="mt-4" title="main.py" :code="pythonSnippet(model)" />
    </section>

    <section id="opencode" class="mt-10">
      <h2 class="text-[22px] font-semibold text-foreground">OpenCode</h2>
      <p class="mt-2 text-[14px] text-muted-foreground">{{ t('needtoken.docs.opencode.desc') }}</p>
      <CodeBlock class="mt-4" title="opencode.json" :code="opencodeSnippet(model)" />
      <ul class="mt-4 space-y-2 text-[14px] text-muted-foreground">
        <li v-for="n in 4" :key="n" class="flex gap-2"><span class="text-primary">›</span>{{ t(`needtoken.docs.opencode.b${n}`, { model: `needtoken/${model}` }) }}</li>
      </ul>
    </section>

    <section id="errors" class="mt-10">
      <h2 class="text-[22px] font-semibold text-foreground">{{ t('needtoken.docs.errors.title') }}</h2>
      <ul class="mt-4 space-y-2 text-[14px] text-muted-foreground">
        <li v-for="n in 3" :key="n" class="flex gap-2"><span class="text-primary">›</span>{{ t(`needtoken.docs.errors.b${n}`) }}</li>
      </ul>
    </section>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import { usePageSeo } from '../../composables/usePageSeo'
import CodeBlock from './components/CodeBlock.vue'
import { API_BASE_URL, DEFAULT_MODELS } from './config'
import { curlSnippet, exportKeySnippet, opencodeSnippet, pythonSnippet } from './lib/snippets'

const { t } = useI18n()
const route = useRoute()

usePageSeo({
  title: () => t('needtoken.docs.title'),
  description: () => t('needtoken.docs.subtitle'),
})

const fromQuery = typeof route.query.model === 'string' ? route.query.model : ''
const model = ref<string>(fromQuery || DEFAULT_MODELS[0])
</script>
