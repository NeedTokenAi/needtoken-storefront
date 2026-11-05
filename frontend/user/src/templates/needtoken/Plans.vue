<template>
  <div class="api-plans mx-auto w-full max-w-[1200px] px-4 pb-12 pt-12 sm:px-6 sm:pt-16">
    <header class="api-plans-hero" aria-labelledby="api-plans-title">
      <span class="api-plans-eyebrow"><span aria-hidden="true" />{{ t('needtoken.nav.plans') }}</span>
      <h1 id="api-plans-title"><span translate="no">{{ t('needtoken.apiServices.keyLabel') }}</span>{{ t('needtoken.apiServices.titleRest') }}</h1>
      <p class="api-plans-intro">{{ t('needtoken.plans.subtitle') }}</p>
      <ul class="api-plans-benefits" :aria-label="t('needtoken.apiServices.benefitsLabel')">
        <li v-for="key in ['models', 'quota', 'usage']" :key="key"><Check :size="14" aria-hidden="true" />{{ t(`needtoken.apiServices.benefits.${key}`) }}</li>
      </ul>
      <div class="mt-7 flex flex-wrap justify-center gap-3">
        <RouterLink :to="`/categories/${CATEGORY_SLUGS.plans}`" class="rounded-md bg-primary px-5 py-2.5 text-[14px] font-semibold text-primary-foreground hover:bg-primary/90">{{ t('needtoken.plans.buy') }}</RouterLink>
        <a :href="CONSOLE_URL" class="rounded-md border border-border px-5 py-2.5 text-[14px] font-semibold text-foreground hover:bg-secondary" rel="noopener">{{ t('needtoken.plans.console') }}</a>
      </div>
      <p class="api-plans-quota-note">{{ t('needtoken.apiServices.quotaNote') }}</p>
    </header>

    <section class="mt-10 grid gap-4 md:grid-cols-2" :aria-label="t('needtoken.apiServices.options')">
      <article v-for="key in ['personal', 'enterprise']" :key="key" class="api-service-option rounded-2xl border border-border bg-card p-6 sm:p-7">
        <component :is="key === 'personal' ? KeyRound : Building2" class="h-6 w-6 text-primary" aria-hidden="true" />
        <h2 class="mt-5 text-[21px] font-semibold text-foreground">{{ t(`needtoken.apiServices.${key}.title`) }}</h2>
        <p class="mt-3 text-[14px] leading-relaxed text-muted-foreground">{{ t(`needtoken.apiServices.${key}.desc`) }}</p>
        <ul class="mt-5 space-y-2.5 text-[13px] text-muted-foreground">
          <li v-for="n in 3" :key="n" class="flex items-start gap-2"><Check class="mt-0.5 h-4 w-4 shrink-0 text-primary" aria-hidden="true" />{{ t(`needtoken.apiServices.${key}.b${n}`) }}</li>
        </ul>
      </article>
    </section>

    <section class="mt-10 rounded-2xl border border-border bg-card p-6 sm:p-7">
      <div class="mb-5 flex flex-wrap items-center justify-between gap-3">
        <div>
          <h2 class="text-[20px] font-semibold text-foreground">{{ t('needtoken.apiServices.connectTitle') }}</h2>
          <p class="mt-2 text-[14px] text-muted-foreground">{{ t('needtoken.apiServices.connectDesc') }}</p>
        </div>
        <RouterLink to="/docs" class="text-[13px] font-semibold text-primary hover:underline">{{ t('needtoken.nav.docs') }} <span aria-hidden="true">↗</span></RouterLink>
      </div>
      <CodeBlock :code="API_BASE_URL" title="baseURL" />
    </section>

    <!-- 套餐由三个要素组成 -->
    <section class="mt-12">
      <h2 class="text-[22px] font-semibold text-foreground">{{ t('needtoken.plans.what.title') }}</h2>
      <div class="mt-5 grid grid-cols-1 gap-4 md:grid-cols-3">
        <div v-for="(key, idx) in planParts" :key="key" class="rounded-xl border border-border bg-card p-5">
          <div class="nt-mono text-[13px] text-primary">{{ String(idx + 1).padStart(2, '0') }}</div>
          <h3 class="mt-2 text-[16px] font-semibold text-foreground">{{ t(`needtoken.plans.what.${key}.title`) }}</h3>
          <p class="mt-2 text-[14px] text-muted-foreground">{{ t(`needtoken.plans.what.${key}.desc`) }}</p>
        </div>
      </div>
    </section>

    <!-- Tokens 计量 -->
    <section class="mt-12">
      <h2 class="text-[22px] font-semibold text-foreground">{{ t('needtoken.plans.tokens.title') }}</h2>
      <ul class="mt-4 space-y-2 text-[14px] text-muted-foreground">
        <li v-for="n in 4" :key="n" class="flex gap-2"><span class="text-primary">›</span>{{ t(`needtoken.plans.tokens.b${n}`) }}</li>
      </ul>
      <RouterLink to="/models" class="mt-4 inline-block text-[14px] font-semibold text-primary hover:underline">{{ t('needtoken.plans.tokens.link') }}</RouterLink>
    </section>

    <!-- 购买到使用 -->
    <section class="mt-12">
      <h2 class="text-[22px] font-semibold text-foreground">{{ t('needtoken.plans.flow.title') }}</h2>
      <ol class="mt-5 space-y-3">
        <li v-for="n in 4" :key="n" class="flex gap-4 rounded-xl border border-border bg-card p-4">
          <span class="nt-mono grid h-7 w-7 flex-none place-items-center rounded-md bg-primary/15 text-[13px] text-primary">{{ n }}</span>
          <div>
            <div class="text-[15px] font-medium text-foreground">{{ t(`needtoken.plans.flow.s${n}.title`) }}</div>
            <div class="mt-1 text-[14px] text-muted-foreground">{{ t(`needtoken.plans.flow.s${n}.desc`) }}</div>
          </div>
        </li>
      </ol>
      <p class="mt-4 text-[13px] text-muted-foreground">{{ t('needtoken.plans.note') }}</p>
    </section>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { Building2, Check, KeyRound } from 'lucide-vue-next'
import { usePageSeo } from '../../composables/usePageSeo'
import { API_BASE_URL, CATEGORY_SLUGS, CONSOLE_URL } from './config'
import CodeBlock from './components/CodeBlock.vue'

const { t } = useI18n()

usePageSeo({
  title: () => t('needtoken.plans.title'),
  description: () => t('needtoken.plans.subtitle'),
})

const planParts = ['group', 'quota', 'validity']
</script>

<style scoped>
.api-plans-hero { position: relative; isolation: isolate; overflow: hidden; padding: 44px 28px 34px; border: 1px solid var(--ui-border); border-radius: 24px; text-align: center; background: radial-gradient(ellipse at 50% 0, rgba(45, 212, 167, .11), transparent 70%), var(--ui-bg-elevated); }
.api-plans-hero h1 { margin-top: 22px; font-size: clamp(32px, 4.1vw, 52px); line-height: 1.25; letter-spacing: -.045em; font-weight: 750; text-wrap: balance; }
.api-plans-hero h1 > span { color: var(--ui-accent); }
.api-plans-intro { max-width: 660px; margin: 20px auto 0; font-size: 15px; line-height: 1.85; color: var(--ui-text-secondary); text-wrap: pretty; }
.api-plans-eyebrow { display: inline-flex; align-items: center; gap: .5rem; border: 1px solid var(--ui-border); border-radius: 999px; padding: .4rem .75rem; color: var(--ui-accent); font-size: .75rem; }
.api-plans-eyebrow span { height: .375rem; width: .375rem; border-radius: 50%; background: currentColor; }
.api-plans-benefits { display: flex; align-items: center; justify-content: center; flex-wrap: wrap; gap: 12px 25px; margin-top: 24px; font-size: 12px; color: var(--ui-text-secondary); }
.api-plans-benefits li { display: flex; align-items: center; gap: 6px; }
.api-plans-benefits svg { color: var(--ui-accent); }
.api-plans-quota-note { margin-top: 17px; color: var(--ui-text-muted); font-size: 11px; line-height: 1.7; }
.api-service-option { background: linear-gradient(145deg, rgba(45, 212, 167, .035), transparent 65%), var(--ui-bg-elevated); }
@media (max-width: 640px) { .api-plans-hero { padding: 30px 18px 25px; border-radius: 18px; } .api-plans-intro { font-size: 13px; margin-top: 15px; } .api-plans-benefits { gap: 10px 16px; font-size: 11px; } }
</style>
