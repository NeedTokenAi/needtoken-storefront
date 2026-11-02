<template>
  <div class="mx-auto w-full max-w-[1000px] px-6 pb-10 pt-12">
    <header>
      <h1 class="text-[32px] font-bold text-foreground">{{ t('needtoken.plans.title') }}</h1>
      <p class="mt-2 max-w-[70ch] text-[15px] text-muted-foreground">{{ t('needtoken.plans.subtitle') }}</p>
      <div class="mt-6 flex flex-wrap gap-3">
        <RouterLink :to="`/categories/${CATEGORY_SLUGS.plans}`" class="rounded-md bg-primary px-5 py-2.5 text-[14px] font-semibold text-primary-foreground hover:bg-primary/90">{{ t('needtoken.plans.buy') }}</RouterLink>
        <a :href="CONSOLE_URL" class="rounded-md border border-border px-5 py-2.5 text-[14px] font-semibold text-foreground hover:bg-secondary" rel="noopener">{{ t('needtoken.plans.console') }}</a>
      </div>
    </header>

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
      <p class="mt-4 text-[13px] text-muted-foreground">{{ t('needtoken.plans.flow.note') }}</p>
    </section>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { usePageSeo } from '../../composables/usePageSeo'
import { CATEGORY_SLUGS, CONSOLE_URL } from './config'

const { t } = useI18n()

usePageSeo({
  title: () => t('needtoken.plans.title'),
  description: () => t('needtoken.plans.subtitle'),
})

const planParts = ['group', 'quota', 'validity']
</script>
