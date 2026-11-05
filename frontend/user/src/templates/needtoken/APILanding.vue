<template>
  <div class="nt-api-landing mx-auto w-full max-w-[1200px] px-4 sm:px-6">
    <section class="nt-api-hero" aria-labelledby="nt-api-title">
      <div class="nt-api-hero-copy">
        <p class="nt-api-eyebrow"><KeyRound :size="15" aria-hidden="true" />{{ t('needtoken.apiLanding.eyebrow') }}</p>
        <h1 id="nt-api-title">{{ t('needtoken.apiLanding.title') }}<span>{{ t('needtoken.apiLanding.titleAccent') }}</span></h1>
        <p class="nt-api-intro">{{ t('needtoken.apiLanding.intro') }}</p>
        <div class="nt-api-actions"><RouterLink :to="API_PLANS_PATH" class="nt-api-button">{{ t('needtoken.apiLanding.order') }}<ArrowUpRight :size="17" aria-hidden="true" /></RouterLink><a :href="CONSOLE_URL" class="nt-api-button nt-api-button--secondary" rel="noopener">{{ t('needtoken.apiLanding.console') }}<ArrowUpRight :size="17" aria-hidden="true" /></a></div>
        <p class="nt-api-browse-note">{{ t('needtoken.apiLanding.publicBrowse') }}</p>
        <ul class="nt-api-hero-notes"><li v-for="feature in features" :key="feature"><Check :size="14" aria-hidden="true" />{{ t(`needtoken.apiLanding.features.${feature}`) }}</li></ul>
      </div>
      <div class="nt-api-example">
        <div class="nt-api-example-heading"><span><Braces :size="17" aria-hidden="true" />{{ t('needtoken.apiLanding.exampleTitle') }}</span><span class="nt-api-protocol" translate="no">OpenAI SDK</span></div>
        <CodeBlock :code="exampleCode" title="JavaScript" />
        <div class="nt-api-example-links"><span>{{ t('needtoken.apiLanding.compatible') }}</span><RouterLink to="/docs">{{ t('needtoken.apiLanding.quickStart') }}<ArrowUpRight :size="14" aria-hidden="true" /></RouterLink></div>
      </div>
    </section>

    <NeedtokenDeferredRegion :estimated-height="560" :mobile-estimated-height="860"><ApiCostCalculator /></NeedtokenDeferredRegion>

    <section class="nt-api-compare" aria-labelledby="nt-api-choice-title">
      <div class="nt-api-section-heading"><p class="nt-api-eyebrow">{{ t('needtoken.apiLanding.choiceEyebrow') }}</p><h2 id="nt-api-choice-title">{{ t('needtoken.apiLanding.choiceTitle') }}</h2><p>{{ t('needtoken.apiLanding.choiceIntro') }}</p></div>
      <div class="nt-api-choice-grid"><article v-for="service in services" :key="service.key" class="nt-api-choice" :class="{ 'nt-api-choice--api': service.key === 'api' }"><component :is="service.icon" :size="23" :stroke-width="1.6" aria-hidden="true" /><h3>{{ t(`needtoken.apiLanding.choices.${service.key}.title`) }}</h3><p>{{ t(`needtoken.apiLanding.choices.${service.key}.description`) }}</p><RouterLink :to="service.path">{{ t(`needtoken.apiLanding.choices.${service.key}.action`) }}<ArrowUpRight :size="15" aria-hidden="true" /></RouterLink></article></div>
    </section>

    <section class="nt-api-terms" aria-labelledby="nt-api-terms-title"><div class="nt-api-section-heading"><h2 id="nt-api-terms-title">{{ t('needtoken.plans.what.title') }}</h2><p>{{ t('needtoken.apiLanding.termsIntro') }}</p></div><div class="nt-api-terms-grid"><article v-for="part in parts" :key="part.key"><component :is="part.icon" :size="19" aria-hidden="true" /><h3>{{ t(`needtoken.plans.what.${part.key}.title`) }}</h3><p>{{ t(`needtoken.plans.what.${part.key}.desc`) }}</p></article></div><p class="nt-api-account-note"><ShieldCheck :size="17" aria-hidden="true" />{{ t('needtoken.apiLanding.accountNote') }}</p></section>

    <section class="nt-api-next" aria-labelledby="nt-api-next-title"><div><h2 id="nt-api-next-title">{{ t('needtoken.apiLanding.nextTitle') }}</h2><p>{{ t('needtoken.apiLanding.nextDescription') }}</p></div><div class="nt-api-next-links"><RouterLink :to="{ path: API_PLANS_PATH, query: { kind: 'personal' } }">{{ t('needtoken.apiCatalog.personal') }}<ArrowUpRight :size="15" aria-hidden="true" /></RouterLink><RouterLink :to="{ path: API_PLANS_PATH, query: { kind: 'enterprise' } }">{{ t('needtoken.apiCatalog.enterprise') }}<ArrowUpRight :size="15" aria-hidden="true" /></RouterLink><RouterLink to="/models">{{ t('needtoken.nav.models') }}<ArrowUpRight :size="15" aria-hidden="true" /></RouterLink></div></section>
  </div>
</template>

<script setup lang="ts">
import { defineAsyncComponent } from 'vue'
import { useI18n } from 'vue-i18n'
import { ArrowUpRight, Braces, Check, Clock3, Gauge, KeyRound, Layers3, PackageCheck, ShieldCheck, Zap } from 'lucide-vue-next'
import { usePageSeo } from '../../composables/usePageSeo'
import { API_BASE_URL, API_PLANS_PATH, CONSOLE_URL } from './config'
import CodeBlock from './components/CodeBlock.vue'
import NeedtokenDeferredRegion from './components/NeedtokenDeferredRegion.vue'

const ApiCostCalculator = defineAsyncComponent(() => import('./components/ApiCostCalculator.vue'))
const { t } = useI18n()
const features = ['entry', 'usage', 'terms']
const services = [{ key: 'recharge', icon: Zap, path: '/categories/recharge' }, { key: 'ready', icon: PackageCheck, path: '/categories/rental' }, { key: 'api', icon: KeyRound, path: API_PLANS_PATH }]
const parts = [{ key: 'group', icon: Layers3 }, { key: 'quota', icon: Gauge }, { key: 'validity', icon: Clock3 }]
const exampleCode = `import OpenAI from 'openai';\n\nconst client = new OpenAI({\n  apiKey: process.env.NEEDTOKEN_API_KEY,\n  baseURL: '${API_BASE_URL}'\n});`
usePageSeo({ title: () => t('needtoken.apiLanding.seoTitle'), description: () => t('needtoken.apiLanding.intro'), canonicalPath: () => '/api-services' })
</script>

<style scoped>
.nt-api-landing { padding-top: 46px; padding-bottom: 24px; }
.nt-api-hero { display: grid; grid-template-columns: minmax(0,1.05fr) minmax(0,1fr); align-items: center; gap: 48px; margin-bottom: 52px; }
.nt-api-hero-copy { min-width: 0; }
.nt-api-eyebrow { display: flex; align-items: center; gap: 8px; color: var(--ui-accent); font-size: 12px; font-weight: 600; letter-spacing: .025em; }
.nt-api-hero h1 { margin-top: 17px; font-size: clamp(34px,3.6vw,49px); line-height: 1.22; font-weight: 750; letter-spacing: -.045em; text-wrap: balance; }
.nt-api-hero h1 > span { display: block; margin-top: 5px; color: var(--ui-accent); }
.nt-api-intro { max-width: 45ch; margin-top: 20px; font-size: 14px; line-height: 1.9; color: var(--ui-text-secondary); }
.nt-api-actions { display: flex; flex-wrap: wrap; gap: 11px; margin-top: 25px; }
.nt-api-button { display: inline-flex; align-items: center; justify-content: center; gap: 12px; min-height: 44px; padding: 11px 19px; border: 1px solid transparent; border-radius: 12px; background: var(--ui-accent); color: var(--ui-text-on-accent); font-size: 13px; font-weight: 650; }
.nt-api-button--secondary { border-color: var(--ui-border); background: var(--ui-bg-elevated); color: var(--ui-text-secondary); }
.nt-api-browse-note { margin-top: 13px; font-size: 12px; line-height: 1.75; color: var(--ui-text-muted); }
.nt-api-hero-notes { display: flex; flex-wrap: wrap; gap: 10px 16px; margin-top: 23px; font-size: 12px; color: var(--ui-text-secondary); }
.nt-api-hero-notes li { display: inline-flex; align-items: center; gap: 7px; }
.nt-api-hero-notes svg { flex: none; color: var(--ui-accent); }
.nt-api-example { min-width: 0; padding: 20px; border: 1px solid #2dd4a72b; border-radius: 22px; background: linear-gradient(140deg,#2dd4a70b,#ffffff03); box-shadow: inset 0 1px 0 #ffffff05; }
.nt-api-example-heading { display: flex; align-items: center; justify-content: space-between; flex-wrap: wrap; gap: 12px; margin-bottom: 16px; font-size: 13px; color: var(--ui-text-secondary); }
.nt-api-example-heading > span:first-child { display: inline-flex; align-items: center; gap: 8px; }
.nt-api-example-heading svg { color: var(--ui-accent); }
.nt-api-protocol { padding: 5px 8px; border: 1px solid var(--ui-border); border-radius: 8px; font-size: 11px; color: var(--ui-text-muted); }
.nt-api-example-links { display: flex; align-items: center; justify-content: space-between; flex-wrap: wrap; gap: 10px; margin-top: 14px; font-size: 12px; color: var(--ui-text-muted); }
.nt-api-example-links a { display: inline-flex; align-items: center; gap: 6px; color: var(--ui-accent); }
.nt-api-section-heading h2, .nt-api-next h2 { font-size: clamp(23px,2.3vw,29px); font-weight: 650; letter-spacing: -.035em; line-height: 1.35; text-wrap: balance; }
.nt-api-section-heading > .nt-api-eyebrow + h2 { margin-top: 9px; }
.nt-api-section-heading > p:last-child { margin-top: 10px; font-size: 13px; line-height: 1.85; color: var(--ui-text-muted); }
.nt-api-compare, .nt-api-terms { margin-top: 48px; }
.nt-api-choice-grid, .nt-api-terms-grid { display: grid; grid-template-columns: repeat(3,minmax(0,1fr)); gap: 16px; margin-top: 22px; }
.nt-api-choice { display: flex; flex-direction: column; min-width: 0; padding: 24px; border: 1px solid var(--ui-border); border-radius: 19px; background: var(--ui-bg-elevated); }
.nt-api-choice > svg { color: var(--ui-text-muted); }
.nt-api-choice--api { border-color: #2dd4a74d; background: linear-gradient(140deg,#2dd4a70e,var(--ui-bg-elevated)); }
.nt-api-choice--api > svg { color: var(--ui-accent); }
.nt-api-choice h3, .nt-api-terms-grid h3 { margin-top: 16px; font-size: 17px; line-height: 1.4; font-weight: 650; }
.nt-api-choice p, .nt-api-terms-grid p { margin-top: 10px; font-size: 13px; line-height: 1.85; color: var(--ui-text-secondary); overflow-wrap: anywhere; }
.nt-api-choice a { display: inline-flex; align-items: center; gap: 7px; width: fit-content; min-height: 40px; margin-top: auto; padding-top: 16px; color: var(--ui-accent); font-size: 13px; }
.nt-api-terms-grid article { min-width: 0; padding: 22px; border: 1px solid var(--ui-border); border-radius: 17px; background: #ffffff03; }
.nt-api-terms-grid article > svg { color: var(--ui-accent); }
.nt-api-terms-grid h3 { margin-top: 12px; }
.nt-api-account-note { display: flex; align-items: flex-start; gap: 9px; padding: 16px 18px; margin-top: 17px; border: 1px solid #2dd4a72b; border-radius: 12px; background: #2dd4a707; color: var(--ui-text-secondary); font-size: 12px; line-height: 1.85; }
.nt-api-account-note svg { flex: none; margin-top: 2px; color: var(--ui-accent); }
.nt-api-next { display: flex; align-items: center; justify-content: space-between; gap: 26px; margin-top: 42px; padding: 26px; border: 1px solid var(--ui-border); border-radius: 19px; background: var(--ui-bg-elevated); }
.nt-api-next > div:first-child { min-width: 0; }
.nt-api-next p { margin-top: 10px; font-size: 13px; line-height: 1.85; color: var(--ui-text-muted); }
.nt-api-next-links { display: flex; flex-wrap: wrap; justify-content: flex-end; gap: 9px; flex: none; }
.nt-api-next-links a { display: inline-flex; align-items: center; gap: 8px; min-height: 42px; padding: 9px 12px; border: 1px solid #2dd4a739; border-radius: 10px; color: var(--ui-accent); background: #2dd4a708; font-size: 12px; }
@media (max-width: 1023px) { .nt-api-hero { gap: 25px; }.nt-api-next { align-items: flex-start; flex-direction: column; }.nt-api-next-links { justify-content: flex-start; } }
@media (max-width: 760px) { .nt-api-landing { padding-top: 30px; }.nt-api-hero { grid-template-columns: 1fr; gap: 28px; margin-bottom: 36px; }.nt-api-intro { max-width: none; }.nt-api-hero-notes { gap: 10px 14px; margin-top: 19px; }.nt-api-example { padding: 17px; }.nt-api-choice-grid, .nt-api-terms-grid { grid-template-columns: 1fr; gap: 12px; }.nt-api-choice { padding: 21px; }.nt-api-compare, .nt-api-terms { margin-top: 36px; }.nt-api-next { padding: 22px; }.nt-api-next-links { flex: initial; } }
</style>
