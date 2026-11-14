<template>
  <section class="api-hero" aria-labelledby="api-services-title">
    <div class="api-hero-copy">
      <p class="api-hero-label">{{ t('needtoken.apiConnect.eyebrow') }}</p>
      <h1 id="api-services-title">{{ t('needtoken.apiConnect.heroTitle') }}<span>{{ t('needtoken.apiConnect.heroAccent') }}</span></h1>
      <p class="api-hero-intro">{{ t('needtoken.apiConnect.heroIntro') }}</p>
      <div class="api-hero-actions">
        <RouterLink to="/plans" class="primary">{{ t('needtoken.apiUse.buy') }}<ArrowRight :size="16" aria-hidden="true" /></RouterLink>
        <a href="#api-connect">{{ t('needtoken.apiConnect.start') }}<ArrowDown :size="16" aria-hidden="true" /></a>
      </div>
      <p class="api-hero-footnote">{{ t('needtoken.apiConnect.heroNote') }}</p>
    </div>

    <div class="api-preview" :aria-label="t('needtoken.apiConnect.previewLabel')">
      <div class="preview-header"><span><Braces :size="16" aria-hidden="true" />{{ t('needtoken.apiConnect.previewTitle') }}</span><span>{{ t('needtoken.apiConnect.example') }}</span></div>
      <ol class="api-route" :aria-label="t('needtoken.apiConnect.routeLabel')">
        <li><Terminal :size="18" aria-hidden="true" /><span>{{ t('needtoken.apiConnect.client') }}<small>{{ t('needtoken.apiConnect.clientDetail') }}</small></span></li>
        <li class="route-arrow" aria-hidden="true"><ArrowRight :size="16" /></li>
        <li class="route-gateway"><img src="/needtoken-icon.svg" alt="" /><span>NeedToken<small>{{ t('needtoken.apiConnect.gatewayDetail') }}</small></span></li>
        <li class="route-arrow" aria-hidden="true"><ArrowRight :size="16" /></li>
        <li><Cpu :size="18" aria-hidden="true" /><span>{{ t('needtoken.apiConnect.model') }}<small>{{ t('needtoken.apiConnect.modelDetail') }}</small></span></li>
      </ol>
      <div class="preview-endpoint"><span>POST</span><code>/v1/chat/completions</code></div>
      <div class="preview-code-head">
        <div role="group" :aria-label="t('needtoken.apiConnect.previewLanguage')"><button v-for="kind in languages" :key="kind" type="button" :aria-pressed="language === kind" @click="language = kind">{{ kind === 'curl' ? 'cURL' : 'Python' }}</button></div>
        <span>{{ t('needtoken.apiConnect.requestExample') }}</span>
      </div>
      <pre :key="language" class="preview-code" tabindex="0" :aria-label="t('needtoken.apiConnect.requestExample')"><code>{{ code }}</code></pre>
      <div class="preview-footer"><KeyRound :size="14" aria-hidden="true" /><code>NEEDTOKEN_API_KEY</code><span>{{ t('needtoken.apiConnect.environmentKey') }}</span></div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { ArrowDown, ArrowRight, Braces, Cpu, KeyRound, Terminal } from 'lucide-vue-next'
import { API_BASE_URL, DEFAULT_MODELS } from '../config'
import { apiPreviewCode, type ApiPreview } from '../lib/apiIntegrations'

const { t } = useI18n()
const languages = ['curl', 'python'] as const
const language = ref<ApiPreview>('curl')
const code = computed(() => apiPreviewCode(language.value, API_BASE_URL, DEFAULT_MODELS[0]))
</script>

<style scoped>
.api-hero { display: grid; grid-template-columns: minmax(0, .9fr) minmax(0, 1.1fr); align-items: center; gap: 64px; min-height: 560px; padding-block: 64px; animation: api-enter 400ms cubic-bezier(.22,1,.36,1) both; }
.api-hero-copy { min-width: 0; }
.api-hero-label { color: var(--ui-accent); font-size: 13px; font-weight: 500; }
.api-hero h1 { margin-top: 20px; font-size: clamp(36px, 4.2vw, 56px); line-height: 1.16; letter-spacing: -.03em; font-weight: 500; text-wrap: balance; }
.api-hero h1 span { display: block; margin-top: 8px; color: var(--ui-accent); }
.api-hero-intro { margin-top: 24px; max-width: 34em; color: var(--ui-text-secondary); font-size: 16px; line-height: 1.85; text-wrap: pretty; }
.api-hero-actions { display: flex; flex-wrap: wrap; align-items: center; gap: 12px; margin-top: 32px; }
.api-hero-actions a { display: inline-flex; align-items: center; justify-content: center; gap: 12px; min-height: 48px; padding: 12px 20px; border: 1px solid var(--ui-border-strong); border-radius: 10px; color: var(--ui-text-secondary); font-size: 14px; font-weight: 500; transition: color 160ms, background 160ms, border-color 160ms; }
.api-hero-actions a:hover { color: var(--ui-text-primary); background: var(--ui-bg-soft); }
.api-hero-actions a.primary { color: var(--ui-text-on-accent); border-color: var(--ui-accent); background: var(--ui-accent); }
.api-hero-actions a.primary:hover { background: var(--ui-accent-hover); }
.api-hero-footnote { margin-top: 20px; color: var(--ui-text-muted); font-size: 12px; line-height: 1.75; }
.api-preview { min-width: 0; overflow: hidden; border: 1px solid var(--ui-border-strong); border-radius: 20px; background: var(--ui-bg-elevated); }
.preview-header, .preview-footer { display: flex; align-items: center; justify-content: space-between; gap: 12px; padding: 16px 20px; color: var(--ui-text-muted); font-size: 12px; }
.preview-header { border-bottom: 1px solid var(--ui-border); }
.preview-header > span:first-child { display: inline-flex; align-items: center; gap: 8px; color: var(--ui-text-secondary); }
.api-route { display: flex; align-items: center; justify-content: space-between; gap: 12px; margin: 20px; }
.api-route li:not(.route-arrow) { display: flex; align-items: center; gap: 8px; min-width: 0; color: var(--ui-text-secondary); font-size: 13px; }
.api-route li > svg { flex: none; color: var(--ui-text-muted); }
.api-route li > span { min-width: 0; }
.api-route small { display: block; margin-top: 4px; color: var(--ui-text-muted); font-size: 11px; line-height: 1.5; }
.api-route img { width: 28px; height: 28px; flex: none; }
.api-route .route-gateway > span { color: var(--ui-accent); }
.route-arrow { flex: none; color: var(--ui-text-muted); }
.preview-endpoint { display: flex; align-items: center; gap: 12px; padding: 12px 20px; border-block: 1px solid var(--ui-border); background: var(--ui-bg-soft); font-size: 12px; }
.preview-endpoint span { color: var(--ui-accent); font-family: var(--nt-font-mono); }
.preview-endpoint code { min-width: 0; overflow-wrap: anywhere; color: var(--ui-text-secondary); }
.preview-code-head { display: flex; align-items: center; justify-content: space-between; gap: 8px; padding: 12px 20px 0; color: var(--ui-text-muted); font-size: 11px; }
.preview-code-head > div { display: flex; gap: 4px; }
.preview-code-head button { padding: 6px 8px; border-radius: 6px; color: var(--ui-text-muted); font-size: 12px; transition: color 160ms, background 160ms; }
.preview-code-head button[aria-pressed=true] { color: var(--ui-text-primary); background: var(--ui-bg-soft); }
.preview-code { height: 176px; overflow: auto; padding: 16px 20px; color: var(--ui-text-secondary); font-size: 12px; line-height: 1.85; tab-size: 2; }
.preview-footer { justify-content: flex-start; flex-wrap: wrap; gap: 8px; border-top: 1px solid var(--ui-border); background: var(--ui-bg-soft); font-size: 11px; }
.preview-footer > span { margin-left: auto; }
.api-hero :is(a,button,pre):focus-visible { outline: 2px solid var(--ui-accent); outline-offset: 3px; }
@keyframes api-enter { from { opacity: 0; transform: translateY(8px); } to { opacity: 1; transform: none; } }
@media (max-width: 1100px) { .api-hero { gap: 32px; } .api-route { gap: 8px; } .api-route li:not(.route-arrow) { gap: 6px; font-size: 12px; } .api-route small { font-size: 10px; } }
@media (max-width: 850px) { .api-hero { grid-template-columns: minmax(0, 1fr); min-height: auto; gap: 40px; padding-block: 48px; } .api-hero-copy { max-width: 640px; } .api-hero h1 { font-size: clamp(36px, 6vw, 52px); } .api-preview { width: 100%; max-width: 680px; } .api-route { justify-content: space-around; } }
@media (max-width: 480px) { .api-hero { padding-block: 40px; gap: 32px; } .api-hero h1 { font-size: 36px; } .api-hero-intro { font-size: 14px; margin-top: 20px; } .api-hero-actions { margin-top: 24px; } .api-hero-actions a { min-height: 44px; padding: 10px 16px; } .api-preview { border-radius: 16px; } .preview-header, .preview-footer { padding: 12px 16px; } .api-route { margin: 20px 12px; gap: 6px; } .api-route li:not(.route-arrow) { flex-direction: row; align-items: center; gap: 6px; font-size: 12px; } .api-route small { display: none; } .api-route .route-gateway { flex-direction: row; align-items: center; } .api-route img { width: 24px; height: 24px; } .preview-endpoint { padding-inline: 16px; font-size: 11px; gap: 8px; } .preview-code-head { padding-inline: 12px; } .preview-code { padding: 16px; font-size: 11px; height: 144px; } .preview-footer > span { margin-left: 0; } }
@media (prefers-reduced-motion: reduce) { .api-hero { animation: none; } .api-hero :is(a,button) { transition: none; } }
</style>
