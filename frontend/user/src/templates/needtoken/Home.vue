<template>
  <div>
    <!-- Hero：徽章条 + 主标题 + 副标题 + 主按钮 -->
    <section class="nt-grid-bg border-b border-border">
      <div class="mx-auto w-full max-w-[1200px] px-6 pb-20 pt-16 text-center">
        <div class="flex flex-wrap justify-center gap-2">
          <span v-for="key in badgeKeys" :key="key" class="nt-mono rounded-full border border-border bg-card px-3 py-1 text-[12px] text-muted-foreground">{{ t(`needtoken.home.badges.${key}`) }}</span>
        </div>
        <h1 class="mx-auto mt-8 max-w-[18ch] text-[40px] font-bold leading-[1.1] tracking-tight text-foreground sm:text-[56px]">
          {{ t('needtoken.home.title') }}
        </h1>
        <p class="mx-auto mt-5 max-w-[56ch] text-[16px] text-muted-foreground sm:text-[18px]">{{ t('needtoken.home.subtitle') }}</p>
        <div class="mt-8 flex flex-wrap justify-center gap-3">
          <a :href="CONSOLE_REGISTER_URL" class="rounded-md bg-primary px-5 py-2.5 text-[15px] font-semibold text-primary-foreground hover:bg-primary/90" rel="noopener">{{ t('needtoken.home.ctaPrimary') }}</a>
          <RouterLink to="/models" class="rounded-md border border-border px-5 py-2.5 text-[15px] font-semibold text-foreground hover:bg-secondary">{{ t('needtoken.home.ctaSecondary') }}</RouterLink>
        </div>
        <div class="nt-mono mx-auto mt-8 inline-flex items-center gap-2 rounded-md border border-border bg-card px-4 py-2 text-[13px] text-muted-foreground">
          <span class="text-primary">baseURL</span>
          <span class="text-foreground">{{ API_BASE_URL }}</span>
        </div>
      </div>
    </section>

    <!-- 亮点区：只写可核对的事实 -->
    <section class="mx-auto grid w-full max-w-[1200px] grid-cols-1 gap-px overflow-hidden border-x border-b border-border bg-border sm:grid-cols-3">
      <div v-for="item in highlights" :key="item.key" class="bg-background px-6 py-8">
        <div class="nt-mono text-[32px] font-bold text-primary">{{ item.value }}</div>
        <div class="mt-1 text-[14px] text-muted-foreground">{{ t(`needtoken.home.highlights.${item.key}`) }}</div>
      </div>
    </section>

    <!-- 默认模型 -->
    <section class="mx-auto w-full max-w-[1200px] px-6 pt-20">
      <h2 class="text-[28px] font-bold text-foreground">{{ t('needtoken.home.models.title') }}</h2>
      <p class="mt-2 text-[15px] text-muted-foreground">{{ t('needtoken.home.models.subtitle') }}</p>
      <div class="mt-8 grid grid-cols-2 gap-3 md:grid-cols-4">
        <RouterLink
          v-for="(model, idx) in DEFAULT_MODELS"
          :key="model"
          :to="{ path: '/models', query: { q: model } }"
          class="nt-mono flex items-center justify-between rounded-lg border border-border bg-card px-4 py-3 text-[14px] text-foreground transition-colors hover:border-primary"
        >
          <span>{{ model }}</span>
          <span v-if="idx === 0" class="rounded bg-primary/15 px-1.5 py-0.5 text-[11px] text-primary">{{ t('needtoken.home.models.preferred') }}</span>
        </RouterLink>
      </div>
    </section>

    <!-- 编号功能卡片 01–04 -->
    <section class="mx-auto w-full max-w-[1200px] px-6 pt-20">
      <h2 class="text-[28px] font-bold text-foreground">{{ t('needtoken.home.features.title') }}</h2>
      <div class="mt-8 grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-4">
        <RouterLink v-for="(card, idx) in featureCards" :key="card.key" :to="card.to" class="group rounded-xl border border-border bg-card p-6 transition-colors hover:border-primary">
          <div class="nt-mono text-[13px] text-primary">{{ String(idx + 1).padStart(2, '0') }}</div>
          <h3 class="mt-3 text-[18px] font-semibold text-foreground">{{ t(`needtoken.home.features.${card.key}.title`) }}</h3>
          <ul class="mt-4 space-y-2 text-[14px] text-muted-foreground">
            <li v-for="n in 3" :key="n" class="flex gap-2"><span class="text-primary">›</span>{{ t(`needtoken.home.features.${card.key}.b${n}`) }}</li>
          </ul>
        </RouterLink>
      </div>
    </section>

    <!-- 对比表：只写可验证项 -->
    <section class="mx-auto w-full max-w-[1200px] px-6 pt-20">
      <h2 class="text-[28px] font-bold text-foreground">{{ t('needtoken.home.compare.title') }}</h2>
      <div class="mt-8 overflow-x-auto rounded-xl border border-border">
        <table class="w-full min-w-[560px] text-left text-[14px]">
          <thead class="bg-card text-muted-foreground">
            <tr>
              <th class="px-5 py-3 font-medium">{{ t('needtoken.home.compare.item') }}</th>
              <th class="px-5 py-3 font-medium text-primary">{{ brandName }}</th>
              <th class="px-5 py-3 font-medium">{{ t('needtoken.home.compare.diy') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="row in compareRows" :key="row" class="border-t border-border">
              <td class="px-5 py-3 text-foreground">{{ t(`needtoken.home.compare.rows.${row}.item`) }}</td>
              <td class="px-5 py-3 text-foreground">{{ t(`needtoken.home.compare.rows.${row}.us`) }}</td>
              <td class="px-5 py-3 text-muted-foreground">{{ t(`needtoken.home.compare.rows.${row}.diy`) }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>

    <!-- 三步快速上手 -->
    <section class="mx-auto w-full max-w-[1200px] px-6 pt-20">
      <h2 class="text-[28px] font-bold text-foreground">{{ t('needtoken.home.steps.title') }}</h2>
      <div class="mt-8 grid grid-cols-1 gap-4 lg:grid-cols-3">
        <div v-for="(step, idx) in steps" :key="step.key" class="flex flex-col gap-4 rounded-xl border border-border bg-card p-5">
          <div>
            <div class="nt-mono text-[13px] text-primary">STEP {{ idx + 1 }}</div>
            <h3 class="mt-2 text-[17px] font-semibold text-foreground">{{ t(`needtoken.home.steps.${step.key}.title`) }}</h3>
            <p class="mt-1 text-[14px] text-muted-foreground">{{ t(`needtoken.home.steps.${step.key}.desc`) }}</p>
          </div>
          <CodeBlock :code="step.code" :title="step.title" class="mt-auto" />
        </div>
      </div>
    </section>

    <!-- 定价摘要：只读 new-api 定价表，未配价不写 -->
    <section class="mx-auto w-full max-w-[1200px] px-6 pt-20">
      <div class="flex flex-wrap items-end justify-between gap-3">
        <div>
          <h2 class="text-[28px] font-bold text-foreground">{{ t('needtoken.home.pricing.title') }}</h2>
          <p class="mt-2 text-[15px] text-muted-foreground">{{ t('needtoken.home.pricing.subtitle') }}</p>
        </div>
        <RouterLink to="/models" class="text-[14px] font-semibold text-primary hover:underline">{{ t('needtoken.home.pricing.more') }}</RouterLink>
      </div>
      <div class="mt-8 overflow-x-auto rounded-xl border border-border">
        <div v-if="pricing.isLoading.value" class="px-5 py-8 text-center text-[14px] text-muted-foreground">{{ t('needtoken.common.loading') }}</div>
        <div v-else-if="pricingRows.length === 0" class="px-5 py-8 text-center text-[14px] text-muted-foreground">{{ t('needtoken.common.pricingPending') }}</div>
        <table v-else class="w-full min-w-[560px] text-left text-[14px]">
          <thead class="bg-card text-muted-foreground">
            <tr>
              <th class="px-5 py-3 font-medium">{{ t('needtoken.models.columns.model') }}</th>
              <th class="px-5 py-3 font-medium">{{ t('needtoken.models.columns.supply') }}</th>
              <th class="px-5 py-3 font-medium">{{ t('needtoken.models.columns.input') }}</th>
              <th class="px-5 py-3 font-medium">{{ t('needtoken.models.columns.output') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="row in pricingRows" :key="row.key" class="border-t border-border">
              <td class="nt-mono px-5 py-3 text-foreground">{{ row.modelId }}</td>
              <td class="px-5 py-3 text-muted-foreground">{{ supplyLabel(row) }}</td>
              <td class="nt-mono px-5 py-3 text-foreground">{{ formatUsd(row.inputPrice) }}</td>
              <td class="nt-mono px-5 py-3 text-foreground">{{ formatUsd(row.outputPrice) }}</td>
            </tr>
          </tbody>
        </table>
      </div>
      <p class="mt-3 text-[12px] text-muted-foreground">{{ t('needtoken.common.priceUnitNote') }}</p>
    </section>

    <!-- FAQ -->
    <section class="mx-auto w-full max-w-[900px] px-6 pt-20">
      <h2 class="text-center text-[28px] font-bold text-foreground">{{ t('needtoken.home.faq.title') }}</h2>
      <div class="mt-8 divide-y divide-border rounded-xl border border-border">
        <details v-for="key in faqKeys" :key="key" class="group px-5 py-4">
          <summary class="flex cursor-pointer list-none items-center justify-between gap-4 text-[15px] font-medium text-foreground">
            {{ t(`needtoken.home.faq.${key}.q`) }}
            <span class="nt-mono text-muted-foreground transition-transform group-open:rotate-45">+</span>
          </summary>
          <p class="mt-3 text-[14px] leading-relaxed text-muted-foreground">{{ t(`needtoken.home.faq.${key}.a`) }}</p>
        </details>
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '../../stores/app'
import { usePageSeo } from '../../composables/usePageSeo'
import CodeBlock from './components/CodeBlock.vue'
import { API_BASE_URL, CATEGORY_SLUGS, CONSOLE_REGISTER_URL, CONSOLE_URL, DEFAULT_MODELS } from './config'
import { formatUsd, sortOffers, type ModelOffer } from './lib/pricing'
import { usePricing } from './lib/usePricing'
import { curlSnippet, exportKeySnippet } from './lib/snippets'

const { t } = useI18n()
const appStore = useAppStore()
const pricing = usePricing()

const brandName = computed(() => String(appStore.config?.brand?.site_name || '').trim() || 'needtoken')

usePageSeo({
  title: () => t('needtoken.home.seoTitle'),
  description: () => t('needtoken.home.subtitle'),
})

const badgeKeys = ['compatible', 'oneKey', 'guestOrder', 'usage']

const highlights = [
  { key: 'models', value: String(DEFAULT_MODELS.length) },
  { key: 'entry', value: '1' },
  { key: 'protocol', value: '/v1' },
]

const featureCards = [
  { key: 'recharge', to: `/categories/${CATEGORY_SLUGS.recharge}` },
  { key: 'rental', to: `/categories/${CATEGORY_SLUGS.rental}` },
  { key: 'plans', to: '/plans' },
  { key: 'models', to: '/models' },
]

const compareRows = ['entry', 'keys', 'billing', 'order', 'models']

const steps = computed(() => [
  { key: 'register', title: 'browser', code: `open ${CONSOLE_REGISTER_URL}` },
  { key: 'key', title: 'shell', code: `# ${CONSOLE_URL}\n${exportKeySnippet()}` },
  { key: 'call', title: 'shell', code: curlSnippet() },
])

const faqKeys = ['account', 'delivery', 'key', 'unpriced', 'aftersale']

// 定价摘要只列默认模型，且只列 new-api 已配价的
const pricingRows = computed<ModelOffer[]>(() =>
  sortOffers(pricing.offers.value.filter(o => o.isDefault), 'default', DEFAULT_MODELS).slice(0, 8),
)

const supplyLabel = (offer: ModelOffer) =>
  offer.supplyLine ? t(`needtoken.supply.${offer.supplyLine}`) : offer.group

onMounted(() => { void pricing.ensureLoaded() })
</script>
