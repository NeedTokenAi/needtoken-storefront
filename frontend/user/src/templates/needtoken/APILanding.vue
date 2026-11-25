<template>
  <div class="api-landing">
    <!-- 1. Hero 区域（含智能路由可视化架构图与实时指标卡） -->
    <ApiGatewayHero>
      <template #before-metrics>
        <ApiSmartRoutingShowcase />
      </template>
    </ApiGatewayHero>

    <!-- 2. 三步使用步骤 -->
    <section class="api-steps-section" aria-labelledby="api-steps-title">
      <h2 id="api-steps-title" class="sr-only">{{ t('needtoken.apiServices.steps.title') }}</h2>
      <ol class="steps-grid">
        <li v-for="(step, idx) in steps" :key="step.key" class="step-card">
          <div class="step-icon-wrap" aria-hidden="true">
            <component :is="step.icon" :size="24" :stroke-width="1.6" />
            <span class="step-badge">{{ idx + 1 }}</span>
          </div>
          <h3>{{ t(`needtoken.apiServices.steps.${step.key}.title`) }}</h3>
          <p>{{ t(`needtoken.apiServices.steps.${step.key}.desc`) }}</p>
        </li>
      </ol>
    </section>

    <!-- 3. Need 网关套餐与优势直购区 -->
    <section id="gateway-plans" class="api-plans-section" aria-labelledby="gateway-plans-title">
      <header class="section-head">
        <div>
          <span class="section-eyebrow">{{ t('needtoken.apiServices.plansSection.eyebrow') }}</span>
          <h2 id="gateway-plans-title">{{ t('needtoken.apiServices.plansSection.title') }}</h2>
          <p>{{ t('needtoken.apiServices.plansSection.subtitle') }}</p>
        </div>
        <RouterLink to="/plans" class="view-all-link">
          <span>{{ t('needtoken.apiServices.plansSection.viewAll') }}</span>
          <ArrowRight :size="15" aria-hidden="true" />
        </RouterLink>
      </header>

      <!-- 核心套餐矩阵亮点：官方定价基准 · 多模型混合 · 动态套餐额度 · 多专享卡叠加 · 超额自动扣余额 -->
      <div class="plan-advantage-bar">
        <span v-for="adv in planAdvantages" :key="adv.key" class="adv-pill">
          <component :is="adv.icon" :size="14" aria-hidden="true" />
          <span>{{ adv.label }}</span>
        </span>
      </div>

      <div class="plan-tracks-grid">
        <article v-for="track in planTracks" :key="track.key" class="plan-track-card">
          <div class="plan-track-top">
            <strong>{{ t(`needtoken.apiServices.plansSection.tracks.${track.key}.title`) }}</strong>
            <span class="track-badge">{{ track.badge }}</span>
          </div>
          <p>{{ t(`needtoken.apiServices.plansSection.tracks.${track.key}.desc`) }}</p>
        </article>
      </div>

      <!-- 实时在售 Need 网关套餐商品卡片直购 -->
      <div class="live-plans-wrap">
        <div v-if="planCatalog.categoriesLoading.value || planCatalog.loading.value" class="live-plans-grid" role="status" aria-busy="true">
          <div v-for="n in 3" :key="n" class="plan-skeleton" />
        </div>
        <div v-else-if="planCatalog.products.value.length > 0" class="live-plans-grid">
          <ServiceProductCard
            v-for="product in planCatalog.products.value.slice(0, 6)"
            :key="product.id"
            :product="product"
            kind="recharge"
            member-only
            :category-label="t('needtoken.nav.plans')"
            :cta-label="t(userAuth.isAuthenticated ? 'needtoken.apiCatalog.choose' : 'needtoken.apiCatalog.loginToChoose')"
            :delivery-text="t('needtoken.apiCatalog.memberDelivery')"
            :description-fallback="t('needtoken.apiCatalog.productFallback')"
          />
        </div>
        <div v-else class="live-plans-empty">
          <p>{{ t('needtoken.apiServices.plansSection.fallbackHint') }}</p>
          <RouterLink to="/plans" class="view-all-link">
            <span>{{ t('needtoken.apiServices.plansSection.viewAll') }}</span>
            <ArrowRight :size="15" aria-hidden="true" />
          </RouterLink>
        </div>
      </div>
    </section>

    <!-- 4. 动态模型精选折扣卡片与全模型实时比价表 -->
    <ApiPricingShowcase @quick-start="onQuickStartModel" />

    <!-- 6. CC Switch 双栏场景配置工作台（图 5） -->
    <ApiIntegrationGuide :initial-model="selectedGuideModel" />

    <!-- 7. 费用计算器 -->
    <section class="api-usage" :aria-label="t('needtoken.apiUse.calculatorTitle')">
      <ApiCostCalculator />
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  ArrowRight,
  BarChart3,
  Layers,
  Repeat,
  Settings,
  ShieldCheck,
  Sparkles,
  Wallet,
  Zap,
} from 'lucide-vue-next'
import { usePageSeo } from '../../composables/usePageSeo'
import { useUserAuthStore } from '../../stores/userAuth'
import { useApiPlanCatalog } from './composables/useApiPlanCatalog'
import { usePricing } from './lib/usePricing'
import type { PublicPlanOffer } from './lib/pricing'
import ApiGatewayHero from './components/ApiGatewayHero.vue'
import ApiSmartRoutingShowcase from './components/ApiSmartRoutingShowcase.vue'
import ApiPricingShowcase from './components/ApiPricingShowcase.vue'
import ApiIntegrationGuide from './components/ApiIntegrationGuide.vue'
import ApiCostCalculator from './components/ApiCostCalculator.vue'
import ServiceProductCard from './components/ServiceProductCard.vue'

const { t } = useI18n()
const userAuth = useUserAuthStore()
const planCatalog = useApiPlanCatalog({ pageSize: 6, syncRoute: false })
const pricing = usePricing()
const selectedGuideModel = ref('')

const steps = [
  { key: 'configure', icon: Settings },
  { key: 'connect', icon: Zap },
  { key: 'monitor', icon: BarChart3 },
] as const

const formatMultiplierBadge = (mult: number): string => {
  const rounded = Math.round(mult * 10) / 10
  return Number.isInteger(rounded) ? `${rounded}x` : `${rounded.toFixed(1)}x`
}

const maxActiveMultiplier = computed(() => {
  let maxMult = 0
  for (const plan of pricing.activePlans.value) {
    if (Number.isFinite(plan.multiplier) && plan.multiplier > maxMult) {
      maxMult = plan.multiplier
    }
  }
  return maxMult
})

const planAdvantages = computed(() => [
  { key: 'officialBase', icon: ShieldCheck, label: t('needtoken.apiServices.plansSection.advantages.officialBase') },
  { key: 'mixedModels', icon: Layers, label: t('needtoken.apiServices.plansSection.advantages.mixedModels') },
  {
    key: 'multipliers',
    icon: Sparkles,
    label: maxActiveMultiplier.value > 1.05
      ? t('needtoken.apiServices.plansSection.advantages.multipliersDynamic', { mult: formatMultiplierBadge(maxActiveMultiplier.value) })
      : t('needtoken.apiServices.plansSection.advantages.multipliers'),
  },
  { key: 'stackable', icon: Repeat, label: t('needtoken.apiServices.plansSection.advantages.stackable') },
  { key: 'overflow', icon: Wallet, label: t('needtoken.apiServices.plansSection.advantages.overflow') },
])

const TRACK_KEYWORDS = {
  openai: ['openai', 'codex', 'gpt', 'o3', 'o4'],
  claude: ['claude', 'anthropic', 'opus', 'sonnet'],
  gemini: ['gemini', 'google'],
  xai: ['xai', 'grok'],
  image: ['image', 'img', 'midjourney', 'flux', 'dalle', '图片', '生图', '繪圖', '圖片'],
  video: ['video', 'sora', 'veo', 'kling', 'runway', '视频', '影片'],
} as const

type TrackKey = keyof typeof TRACK_KEYWORDS

const matchPlanTrack = (plan: PublicPlanOffer, key: TrackKey): boolean => {
  const haystack = [
    plan.title || '',
    plan.subtitle || '',
    plan.upgrade_group || '',
    ...(plan.allowed_groups || []),
  ].join(' ').toLowerCase()
  return TRACK_KEYWORDS[key].some(keyword => haystack.includes(keyword))
}

const planTrackKeys: readonly TrackKey[] = ['openai', 'claude', 'gemini', 'xai', 'image', 'video']

const planTracks = computed(() => {
  const exclusiveLabel = t('needtoken.apiServices.plansSection.exclusiveBadge')
  return planTrackKeys.map((key) => {
    let bestMult = 0
    for (const plan of pricing.activePlans.value) {
      if (matchPlanTrack(plan, key) && Number.isFinite(plan.multiplier) && plan.multiplier > bestMult) {
        bestMult = plan.multiplier
      }
    }
    return {
      key,
      badge: bestMult > 1.05 ? formatMultiplierBadge(bestMult) : exclusiveLabel,
    }
  })
})

onMounted(() => {
  void pricing.ensureLoaded()
})

const onQuickStartModel = (modelId: string) => {
  selectedGuideModel.value = modelId
}

usePageSeo({
  title: () => t('needtoken.nav.plans'),
  description: () => t('needtoken.apiConnect.heroIntro'),
  canonicalPath: () => '/api-services',
})
</script>

<style scoped>
.api-landing { width: min(1200px, calc(100% - 48px)); margin-inline: auto; padding-bottom: 44px; }

/* 3 步使用步骤区（参考图 6） */
.api-steps-section { padding-block: 36px 44px; border-top: 1px solid var(--ui-border); }
.steps-grid { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 24px; list-style: none; padding: 0; margin: 0; }
.step-card { display: flex; flex-direction: column; align-items: center; text-align: center; padding: 14px 16px; }
.step-icon-wrap {
  position: relative;
  display: grid;
  place-items: center;
  width: 62px;
  height: 62px;
  border-radius: 18px;
  border: 1px solid var(--nt-accent-border, var(--ui-border));
  background: var(--nt-accent-subtle, var(--ui-bg-soft));
  color: var(--ui-accent);
}
.step-badge {
  position: absolute;
  top: -7px;
  right: -7px;
  display: grid;
  place-items: center;
  width: 24px;
  height: 24px;
  border-radius: 999px;
  background: #111827;
  color: #fff;
  font: 700 12px var(--nt-font-mono);
}
.dark .step-badge { background: var(--ui-accent); color: var(--ui-text-on-accent); }
.step-card h3 { margin-top: 16px; font-size: 16px; font-weight: 700; color: var(--ui-text-primary); }
.step-card p { margin-top: 8px; max-width: 30ch; font-size: 13px; line-height: 1.7; color: var(--ui-text-secondary); }

/* Need 网关套餐与优势直购区 */
.api-plans-section { padding-block: 48px; border-top: 1px solid var(--ui-border); scroll-margin-top: 86px; }
.section-head { display: flex; align-items: flex-end; justify-content: space-between; flex-wrap: wrap; gap: 16px; }
.section-eyebrow { font-size: 11.5px; font-weight: 700; letter-spacing: .12em; text-transform: uppercase; color: var(--ui-accent); }
.section-head h2 { margin-top: 8px; font-size: clamp(24px, 2.8vw, 32px); font-weight: 700; letter-spacing: -.025em; color: var(--ui-text-primary); }
.section-head p { margin-top: 8px; max-width: 68ch; font-size: 14px; line-height: 1.75; color: var(--ui-text-secondary); }
.view-all-link { display: inline-flex; align-items: center; gap: 6px; min-height: 38px; padding: 0 14px; border: 1px solid var(--ui-border-strong); border-radius: 10px; font-size: 13px; font-weight: 600; color: var(--ui-accent); background: var(--ui-bg-elevated); }
.view-all-link:hover { border-color: var(--ui-accent); background: var(--ui-bg-soft); }

.plan-advantage-bar { display: flex; flex-wrap: wrap; gap: 8px; margin-top: 20px; }
.adv-pill {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  border-radius: 999px;
  border: 1px solid var(--nt-accent-border, var(--ui-border));
  background: var(--nt-accent-subtle, var(--ui-bg-soft));
  color: var(--ui-text-primary);
  font-size: 12.5px;
  font-weight: 600;
}
.adv-pill svg { color: var(--ui-accent); flex: none; }

.plan-tracks-grid {
  display: grid;
  grid-template-columns: repeat(6, minmax(0, 1fr));
  gap: 10px;
  margin-top: 16px;
}
.plan-track-card {
  padding: 13px 14px;
  border: 1px solid var(--ui-border);
  border-radius: 14px;
  background: var(--ui-bg-elevated);
}
.plan-track-top { display: flex; align-items: center; justify-content: space-between; gap: 6px; }
.plan-track-top strong { font-size: 13px; font-weight: 700; color: var(--ui-text-primary); }
.track-badge {
  padding: 1px 7px;
  border-radius: 999px;
  background: var(--ui-accent);
  color: var(--ui-text-on-accent);
  font: 700 10.5px var(--nt-font-mono);
}
.plan-track-card p { margin-top: 6px; font-size: 11.5px; line-height: 1.55; color: var(--ui-text-muted); }

.live-plans-wrap { margin-top: 22px; }
.live-plans-grid { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 16px; }
.plan-skeleton { min-height: 290px; border-radius: 18px; border: 1px solid var(--ui-border); background: var(--ui-bg-soft); }
.live-plans-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  padding: 28px 20px;
  border: 1px dashed var(--ui-border-strong);
  border-radius: 16px;
  background: var(--ui-bg-elevated);
  text-align: center;
  color: var(--ui-text-secondary);
  font-size: 13.5px;
}

.api-usage { padding-top: 48px; border-top: 1px solid var(--ui-border); }
.api-usage :deep(.nt-api-cost) { margin-top: 0; }

@media (max-width: 1024px) {
  .plan-tracks-grid { grid-template-columns: repeat(3, minmax(0, 1fr)); }
  .live-plans-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); }
}
@media (max-width: 760px) {
  .api-landing { width: calc(100% - 32px); padding-bottom: 24px; }
  .steps-grid { grid-template-columns: minmax(0, 1fr); gap: 16px; }
  .plan-tracks-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .live-plans-grid { grid-template-columns: minmax(0, 1fr); }
  .api-plans-section, .api-usage { padding-block: 32px; }
}
</style>
