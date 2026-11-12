<template>
  <div class="api-rebuild">
    <header class="api-rebuild-head">
      <h1>{{ t('needtoken.nav.plans') }}</h1>
      <p>{{ t('needtoken.apiUse.intro') }}</p>
      <div class="api-rebuild-actions">
        <RouterLink to="/plans">{{ t('needtoken.apiUse.buy') }}<ArrowUpRight :size="15" aria-hidden="true" /></RouterLink>
        <a :href="CONSOLE_URL" rel="noopener">{{ t('needtoken.apiUse.console') }}<ArrowUpRight :size="15" aria-hidden="true" /></a>
      </div>
    </header>

    <section class="api-module" aria-labelledby="api-tools-title">
      <h2 id="api-tools-title">{{ t('needtoken.apiUse.toolsTitle') }}</h2>
      <article v-for="tool in tools" :key="tool.key" class="api-tool">
        <div>
          <h3>{{ t(`needtoken.apiUse.tools.${tool.key}.title`) }}</h3>
          <p>{{ t(`needtoken.apiUse.tools.${tool.key}.body`) }}</p>
        </div>
        <CodeBlock :code="tool.code" :title="tool.title" />
      </article>
      <p class="api-tool-note">{{ t('needtoken.apiUse.compatible') }}</p>
    </section>

    <section class="api-module" aria-labelledby="api-usage-title">
      <h2 id="api-usage-title">{{ t('needtoken.apiUse.calculatorTitle') }}</h2>
      <p class="api-tool-note">{{ t('needtoken.apiUse.calculatorIntro') }}</p>
      <ApiCostCalculator />
    </section>
  </div>
</template>

<script setup lang="ts">
import { defineAsyncComponent } from 'vue'
import { useI18n } from 'vue-i18n'
import { ArrowUpRight } from 'lucide-vue-next'
import { usePageSeo } from '../../composables/usePageSeo'
import { API_BASE_URL, CONSOLE_URL, DEFAULT_MODELS } from './config'
import CodeBlock from './components/CodeBlock.vue'
import { curlSnippet, opencodeSnippet, pythonSnippet } from './lib/snippets'

const ApiCostCalculator = defineAsyncComponent(() => import('./components/ApiCostCalculator.vue'))
const { t } = useI18n()
const model = DEFAULT_MODELS[0]
const tools = [
  { key: 'console', title: 'baseURL', code: API_BASE_URL },
  { key: 'opencode', title: 'opencode.json', code: opencodeSnippet(model) },
  { key: 'curl', title: 'shell', code: curlSnippet(model) },
  { key: 'python', title: 'main.py', code: pythonSnippet(model) },
]
usePageSeo({ title: () => t('needtoken.nav.plans'), description: () => t('needtoken.apiUse.intro'), canonicalPath: () => '/api-services' })
</script>

<style scoped>
.api-rebuild { width: min(1100px, calc(100% - 32px)); margin: 28px auto 8px; }
.api-rebuild-head h1 { font-size: clamp(30px, 4vw, 44px); font-weight: 750; letter-spacing: -.045em; }
.api-rebuild-head p { max-width: 68ch; margin-top: 12px; color: var(--ui-text-secondary); font-size: 14px; line-height: 1.8; }
.api-rebuild-actions { display: flex; flex-wrap: wrap; gap: 10px; margin-top: 18px; }
.api-rebuild-actions a { display: inline-flex; align-items: center; gap: 6px; min-height: 40px; padding: 8px 14px; border-radius: 10px; background: var(--ui-accent); color: var(--ui-text-on-accent); font-size: 13px; font-weight: 650; }
.api-rebuild-actions a:last-child { background: transparent; border: 1px solid var(--ui-border); color: var(--ui-text-secondary); }
.api-module { margin-top: 28px; }
.api-module > h2 { font-size: 22px; font-weight: 700; letter-spacing: -.03em; }
.api-tool { display: grid; grid-template-columns: minmax(0, .8fr) minmax(0, 1.2fr); gap: 18px; margin-top: 16px; padding: 16px; border: 1px solid var(--ui-border); border-radius: 16px; background: rgba(255,255,255,.025); }
.api-tool h3 { font-size: 16px; font-weight: 700; }
.api-tool p, .api-tool-note { margin-top: 8px; color: var(--ui-text-muted); font-size: 13px; line-height: 1.75; }
.api-tool :deep(.nt-code) { min-width: 0; }
@media (max-width: 800px) { .api-tool { grid-template-columns: 1fr; } .api-rebuild { width: min(100% - 28px, 1100px); } }
</style>
