<template>
  <div class="nt-home">
    <section class="home-hero" aria-labelledby="home-title">
      <GhostFibersBackground variant="hero" />
      <div class="home-container hero-content">
        <div class="library-head">
          <div class="hero-eyebrow">
            {{ t('needtoken.homeLibrary.eyebrow') }}
          </div>
          <div class="library-slogan">
            <h1 id="home-title">
              <span v-for="(part, index) in sloganParts" :key="part" class="slogan-word" :style="{ animationDelay: `${index * 60}ms` }">{{ part }}</span>
            </h1>
          </div>
        </div>
        <div class="library-toolbar">
          <nav class="library-tabs" :aria-label="t('needtoken.homeLibrary.tabs')">
            <button type="button" :aria-pressed="kind === 'recharge'" :class="{ active: kind === 'recharge' }" @click="setKind('recharge')"><Zap :size="16" aria-hidden="true" />{{ t('needtoken.nav.recharge') }}</button>
            <button type="button" :aria-pressed="kind === 'rental'" :class="{ active: kind === 'rental' }" @click="setKind('rental')"><PackageCheck :size="16" aria-hidden="true" />{{ t('needtoken.nav.rental') }}</button>
            <a href="https://needtoken.ai/api-services"><KeyRound :size="16" aria-hidden="true" />{{ t('needtoken.nav.plans') }}<ArrowUpRight :size="13" aria-hidden="true" /></a>
          </nav>
          <div class="library-payment-controls">
            <PaymentMethodStrip compact mobile-inline />
            <DisplayCurrencySelector compact hide-label mobile-code />
          </div>
        </div>
        <div v-if="categoriesLoading" class="library-filters library-filters--loading" aria-hidden="true"><i /><i /></div>
        <div v-else-if="scope?.root && childCategories.length" class="library-filters" role="group" :aria-label="t('needtoken.homeLibrary.categories')">
          <button v-for="category in childCategories" :key="category.id" type="button" :class="{ active: selectedId === category.id }" :aria-pressed="selectedId === category.id" @click="selectCategory(category.id)" @pointerenter="prefetchCategory(category.id)" @focus="prefetchCategory(category.id)">{{ getLocalizedText(category.name) }}</button>
        </div>
        <div class="price-rail" :class="{ 'is-ready': !loading && !error && products.length }" :aria-busy="loading">
          <template v-if="loading">
            <NeedtokenSkeleton kind="prices" :count="4" />
          </template>
          <p v-else-if="error" class="price-rail-state" role="alert">{{ t('needtoken.homeLibrary.loadError') }} <button type="button" @click="reload">{{ t('needtoken.common.retry') }}</button></p>
          <p v-else-if="!products.length" class="price-rail-state" role="status">{{ t('needtoken.homeLibrary.empty') }}</p>
          <template v-else>
            <HomePriceCard v-for="product in products" :key="serviceCardKey(product)" :product="product" :kind="kind" />
          </template>
        </div>

        <div class="hero-benefits">
          <div v-for="benefit in benefits" :key="benefit.key" class="hero-benefit">
            <component :is="benefit.icon" :size="16" :stroke-width="1.5" aria-hidden="true" />
            <span>{{ t(`needtoken.homeServices.benefits.${benefit.key}`) }}</span>
          </div>
          <div class="hero-benefit hero-benefit--rate"><StandardExchangeRate /></div>
        </div>
      </div>
    </section>

    <section class="home-container purchase-section" aria-labelledby="purchase-title" @mouseenter="hoverPaused = true" @mouseleave="hoverPaused = false" @focusin="focusPaused = true" @focusout="onFlowFocusOut">
      <div class="section-heading">
        <p class="section-eyebrow">{{ t('needtoken.homeServices.steps.eyebrow') }}</p>
        <div class="flow-controls">
          <div ref="flowTabs" class="flow-tabs" role="tablist" :aria-label="t('needtoken.homeServices.flowControls.label')">
            <button v-for="(flow, index) in flowKeys" :id="`flow-tab-${flow}`" :key="flow" type="button" role="tab" :aria-selected="activeFlow === flow" aria-controls="purchase-flow-panel" :tabindex="activeFlow === flow ? 0 : -1" :class="{ active: activeFlow === flow }" @click="selectFlow(index)" @keydown="onFlowKeydown($event, index)">{{ t(`needtoken.homeServices.flows.${flow}.label`) }}</button>
          </div>
          <button v-if="!reducedMotion" type="button" class="flow-playback" :aria-label="t(userPaused ? 'needtoken.homeServices.flowControls.resume' : 'needtoken.homeServices.flowControls.pause')" :title="t(userPaused ? 'needtoken.homeServices.flowControls.resume' : 'needtoken.homeServices.flowControls.pause')" :aria-pressed="userPaused" @click="userPaused = !userPaused">
            <Play v-if="userPaused" :size="13" aria-hidden="true" /><Pause v-else :size="13" aria-hidden="true" />
          </button>
        </div>
      </div>
      <div id="purchase-flow-panel" role="tabpanel" :aria-labelledby="`flow-tab-${activeFlow}`" :data-flow="activeFlow">
        <h2 id="purchase-title" class="flow-title"><span :key="activeFlow" class="flow-word" :translate="activeFlow === 'api' ? 'no' : undefined">{{ t(`needtoken.homeServices.flows.${activeFlow}.label`) }}</span>{{ t('needtoken.homeServices.steps.suffix') }}</h2>
        <ol class="purchase-steps" :key="activeFlow">
          <li v-for="index in 3" :key="index" class="purchase-step">
            <span class="step-index" aria-hidden="true">{{ String(index).padStart(2, '0') }}</span>
            <div>
              <h3>{{ t(`needtoken.homeServices.flows.${activeFlow}.s${index}.title`) }}</h3>
              <p>{{ t(`needtoken.homeServices.flows.${activeFlow}.s${index}.description`) }}</p>
            </div>
            <ArrowRight v-if="index < 3" class="step-arrow" :size="17" aria-hidden="true" />
          </li>
        </ol>
      </div>
      <div class="order-help">
        <div class="order-help-copy">
          <span class="order-help-icon"><ClipboardList :size="22" :stroke-width="1.6" aria-hidden="true" /></span>
          <div>
            <h3>{{ t('needtoken.homeServices.orders.title') }}</h3>
            <p>{{ t('needtoken.homeServices.orders.description') }}</p>
          </div>
        </div>
        <RouterLink to="/guest/orders" class="text-action">
          {{ t('needtoken.homeServices.orders.action') }}<ArrowRight :size="16" aria-hidden="true" />
        </RouterLink>
      </div>
    </section>

    <section class="home-container information-section" aria-labelledby="questions-title">
      <div class="questions-intro">
        <p class="section-eyebrow">{{ t('needtoken.homeServices.faq.eyebrow') }}</p>
        <h2 id="questions-title">{{ t('needtoken.homeServices.faq.title') }}</h2>
        <p class="section-description">{{ t('needtoken.homeServices.faq.description') }}</p>
        <RouterLink to="/notice" class="text-action">
          {{ t('needtoken.homeServices.faq.notices') }}<ArrowUpRight :size="16" aria-hidden="true" />
        </RouterLink>
      </div>
      <div class="questions-list">
        <details v-for="key in faqKeys" :key="key" class="question">
          <summary>
            {{ t(`needtoken.homeServices.faq.${key}.question`) }}
            <Plus :size="18" :stroke-width="1.5" class="question-toggle" aria-hidden="true" />
          </summary>
          <p>{{ t(`needtoken.homeServices.faq.${key}.answer`) }}</p>
        </details>
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { ArrowRight, ArrowUpRight, BadgeCheck, ClipboardList, Clock3, Pause, Play, Plus, ReceiptText, ShieldCheck, KeyRound, PackageCheck, Zap } from 'lucide-vue-next'
import { useLocalized } from '../../composables/useProduct'
import { usePageSeo } from '../../composables/usePageSeo'
import HomePriceCard from './components/HomePriceCard.vue'
import NeedtokenSkeleton from './components/NeedtokenSkeleton.vue'
import PaymentMethodStrip from './components/PaymentMethodStrip.vue'
import { useHomeCatalog } from './composables/useHomeCatalog'
import GhostFibersBackground from './components/GhostFibersBackground.vue'
import DisplayCurrencySelector from './components/DisplayCurrencySelector.vue'
import StandardExchangeRate from './components/StandardExchangeRate.vue'
import { serviceCardKey } from './lib/serviceCatalog'

const { t } = useI18n()
const { getLocalizedText } = useLocalized()
const { kind, scope, childCategories, selectedId, products, categoriesLoading, loading, error, setKind, selectCategory, prefetchCategory, reload } = useHomeCatalog()
const sloganParts = computed(() => t('needtoken.homeLibrary.title').split(/[、,]/).map(part => part.trim()).filter(Boolean))
usePageSeo({
  title: () => t('needtoken.homeServices.seoTitle'),
  description: () => t('needtoken.homeServices.subtitle'),
})
const benefits = [
  { key: 'selfService', icon: Clock3 },
  { key: 'channels', icon: ShieldCheck },
  { key: 'afterSales', icon: BadgeCheck },
  { key: 'business', icon: ReceiptText },
]
const faqKeys = ['service', 'recharge', 'delivery', 'business']

const flowKeys = ['recharge', 'products', 'api'] as const
const activeFlowIndex = ref(0)
const activeFlow = computed(() => flowKeys[activeFlowIndex.value]!)
const userPaused = ref(false)
const hoverPaused = ref(false)
const focusPaused = ref(false)
const reducedMotion = ref(false)
const pageHidden = ref(false)
const mounted = ref(false)
const flowTabs = ref<HTMLElement | null>(null)
let rotationTimer: number | undefined
let motionQuery: MediaQueryList | undefined

const canRotate = computed(() => mounted.value && !userPaused.value && !hoverPaused.value && !focusPaused.value && !reducedMotion.value && !pageHidden.value)
const clearRotation = () => {
  if (rotationTimer !== undefined) window.clearInterval(rotationTimer)
  rotationTimer = undefined
}
const syncRotation = () => {
  clearRotation()
  if (canRotate.value) rotationTimer = window.setInterval(() => {
    activeFlowIndex.value = (activeFlowIndex.value + 1) % flowKeys.length
  }, 6000)
}
watch(canRotate, syncRotation)

const selectFlow = (index: number, focus = false) => {
  activeFlowIndex.value = (index + flowKeys.length) % flowKeys.length
  if (focus) flowTabs.value?.querySelectorAll<HTMLButtonElement>('[role="tab"]')[activeFlowIndex.value]?.focus()
  syncRotation()
}
const onFlowKeydown = (event: KeyboardEvent, index: number) => {
  const target = event.key === 'ArrowRight' ? index + 1
    : event.key === 'ArrowLeft' ? index - 1
    : event.key === 'Home' ? 0
    : event.key === 'End' ? flowKeys.length - 1
    : null
  if (target === null) return
  event.preventDefault()
  selectFlow(target, true)
}
const onFlowFocusOut = (event: FocusEvent) => {
  focusPaused.value = !!event.relatedTarget && (event.currentTarget as HTMLElement).contains(event.relatedTarget as Node)
}
const onMotionChange = () => { reducedMotion.value = motionQuery?.matches ?? false }
const onVisibilityChange = () => { pageHidden.value = document.hidden }
onMounted(() => {
  motionQuery = window.matchMedia('(prefers-reduced-motion: reduce)')
  onMotionChange()
  onVisibilityChange()
  motionQuery.addEventListener('change', onMotionChange)
  document.addEventListener('visibilitychange', onVisibilityChange)
  mounted.value = true
})
onUnmounted(() => {
  mounted.value = false
  clearRotation()
  motionQuery?.removeEventListener('change', onMotionChange)
  document.removeEventListener('visibilitychange', onVisibilityChange)
})
</script>

<style scoped>
.nt-home { --home-secondary: var(--ui-text-secondary, #b4bfcc); --home-muted: var(--ui-text-muted, #7d8a99); }
.home-container { width: 100%; max-width: var(--nt-content-max-width, 1200px); margin-inline: auto; padding-inline: 24px; }
.home-hero { position: relative; isolation: isolate; }
.home-hero :deep(.nt-fiber-background) { opacity: .45; }
.hero-content { position: relative; z-index: 1; padding-block: 48px 24px; }
.library-head { display: flex; min-height: 56px; align-items: center; justify-content: space-between; gap: 32px; }
.hero-eyebrow { display: inline-flex; align-items: center; gap: 8px; color: var(--ui-text-primary); font-size: 24px; font-weight: 600; letter-spacing: 0; }
.library-slogan { min-width: 0; margin-left: auto; text-align: right; }
.library-slogan h1 { display: flex; flex-wrap: wrap; justify-content: flex-end; gap: .5em; color: var(--ui-accent); font-size: clamp(28px, 3.2vw, 42px); font-weight: 600; line-height: 1.3; letter-spacing: -.02em; }
.slogan-word { display: inline-block; animation: slogan-rise var(--nt-motion-enter, 180ms) var(--nt-motion-ease, ease-out) both; }
.library-toolbar { display: grid; grid-template-columns: auto minmax(0, 1fr); align-items: center; gap: 24px; margin-top: 24px; }
.library-payment-controls { display: flex; align-items: center; justify-content: flex-end; gap: 16px; min-width: 0; }
.library-payment-controls :deep(.payment-method-strip) { min-width: 0; }
.library-payment-controls :deep(.nt-display-currency) { flex-shrink: 0; }
.library-tabs { display: inline-flex; flex-wrap: wrap; gap: 4px; padding: 4px; border: 1px solid var(--ui-border); border-radius: 14px; background: var(--ui-bg-elevated); }
.library-tabs button, .library-tabs a { display: inline-flex; justify-content: center; align-items: center; gap: 8px; min-height: 40px; padding: 8px 16px; border: 0; border-radius: 10px; color: var(--home-secondary); font-size: 14px; font-weight: 500; transition: color var(--nt-motion-fast, 120ms), background-color var(--nt-motion-fast, 120ms), transform var(--nt-motion-fast, 120ms) var(--nt-motion-ease, ease-out); }
.library-tabs button.active { color: #04110d; background: var(--ui-accent); }
.library-tabs :is(button, a):not(.active):hover { color: var(--ui-text-primary); background: var(--ui-bg-soft); }
.library-filters { display: flex; flex-wrap: wrap; align-items: center; gap: 8px; margin-top: 20px; }
.library-filters--loading i { display: block; width: 104px; height: 36px; border-radius: 18px; background: var(--ui-bg-elevated); }
.library-filters button { min-height: 36px; padding: 8px 16px; border: 1px solid transparent; border-radius: 999px; color: var(--home-secondary); font-size: 13px; transition: color var(--nt-motion-fast, 120ms), background-color var(--nt-motion-fast, 120ms), border-color var(--nt-motion-fast, 120ms), transform var(--nt-motion-fast, 120ms) var(--nt-motion-ease, ease-out); }
.library-filters button:hover { color: var(--ui-text-primary); background: var(--ui-bg-soft); }
.library-tabs :is(button, a):active, .library-filters button:active { transform: translateY(1px); }
.library-filters button.active { color: var(--ui-accent); border-color: rgba(45, 212, 167, .22); background: var(--ui-accent-soft); }
.price-rail { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); min-height: 132px; gap: 16px; margin-top: 24px; padding-bottom: 8px; }
.price-rail.is-ready { animation: catalogue-ready var(--nt-motion-enter, 180ms) var(--nt-motion-ease, ease-out) both; }
@keyframes catalogue-ready { from { opacity: .55; } to { opacity: 1; } }
.price-rail-state { grid-column: 1 / -1; display: flex; flex-wrap: wrap; align-items: center; justify-content: center; gap: 12px; min-height: 180px; padding: 24px; border: 1px dashed var(--ui-border-strong); border-radius: 20px; color: var(--home-muted); font-size: 14px; text-align: center; }
.price-rail-state button { min-height: 40px; padding: 8px 16px; border: 1px solid var(--ui-border); border-radius: 8px; color: var(--ui-accent); }
.price-rail > :deep(.nt-page-skeleton--prices) { grid-column: 1 / -1; padding: 0; }
.hero-benefits { display: grid; grid-template-columns: repeat(5, minmax(0, 1fr)); align-items: stretch; gap: 12px; margin-top: 24px; }
.hero-benefit { display: flex; align-items: center; justify-content: center; min-width: 0; min-height: 44px; padding: 12px; gap: 8px; border: 1px solid #f6d96a; border-radius: 12px; background: #f3c94d; color: #2a220d; font-size: 12px; font-weight: 500; line-height: 1.5; text-align: center; }
.hero-benefit svg { flex: none; color: #574516; }
.hero-benefit--rate { position: relative; z-index: 3; }
@keyframes slogan-rise { from { opacity: 0; transform: translateY(6px); } to { opacity: 1; transform: none; } }
.purchase-section { padding-top: 64px; }
.section-heading { display: flex; align-items: center; justify-content: space-between; gap: 24px; }
.section-eyebrow { margin-bottom: 12px; color: var(--ui-accent); font-size: 12px; font-weight: 500; letter-spacing: .01em; }
.section-heading .section-eyebrow { margin-bottom: 0; }
.flow-title, .questions-intro h2 { font-size: 28px; line-height: 1.4; font-weight: 600; letter-spacing: 0; text-wrap: balance; }
.flow-title { margin-top: 12px; }
.flow-word { display: inline-block; color: var(--ui-accent); margin-right: .3em; animation: flow-reveal var(--nt-motion-enter, 180ms) var(--nt-motion-ease, ease-out); }
.flow-controls { display: flex; align-items: center; gap: 8px; }
.flow-tabs { display: flex; align-items: center; gap: 4px; }
.flow-tabs button { min-height: 36px; padding: 8px 12px; border: 1px solid transparent; border-radius: 10px; color: var(--home-muted); font-size: 12px; line-height: 1.5; }
.flow-tabs button.active { color: var(--ui-accent); border-color: rgba(45, 212, 167, .2); background: rgba(45, 212, 167, .07); }
.flow-tabs button:hover { color: var(--ui-text-primary); background: var(--ui-bg-soft); }
.flow-tabs button, .flow-playback, .text-action { transition: color var(--nt-motion-fast, 120ms), background-color var(--nt-motion-fast, 120ms), border-color var(--nt-motion-fast, 120ms), transform var(--nt-motion-fast, 120ms) var(--nt-motion-ease, ease-out); }
.flow-tabs button:active, .flow-playback:active, .text-action:active { transform: translateY(1px); }
.flow-playback { display: grid; place-items: center; width: 32px; height: 32px; flex: none; border: 1px solid var(--ui-border); border-radius: 50%; color: var(--home-muted); }
.flow-playback:hover { color: var(--ui-accent); border-color: var(--ui-accent); }
.section-description { max-width: 365px; color: var(--home-muted); font-size: 13px; line-height: 1.85; text-wrap: pretty; }
.purchase-steps { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); margin-top: 28px; border: 1px solid var(--ui-border); border-radius: 16px; background: rgba(14, 18, 24, 0.45); list-style: none; animation: flow-reveal var(--nt-motion-enter, 180ms) var(--nt-motion-ease, ease-out); }
@keyframes flow-reveal { from { opacity: .35; transform: translateY(6px); } to { opacity: 1; transform: translateY(0); } }
.purchase-step { display: flex; align-items: start; gap: 16px; position: relative; padding: 24px; }
.purchase-step + .purchase-step { border-left: 1px solid var(--ui-border); }
.step-index { flex: none; padding-top: 4px; color: var(--ui-accent); font-family: var(--nt-font-mono); font-size: 12px; font-variant-numeric: tabular-nums; }
.purchase-step h3 { font-size: 15px; line-height: 1.5; font-weight: 600; }
.purchase-step p { margin-top: 8px; color: var(--home-muted); font-size: 12px; line-height: 1.8; }
.step-arrow { flex: none; position: absolute; right: -8px; top: 28px; z-index: 1; color: var(--home-muted); background: var(--ui-bg-page); }
.order-help { display: flex; align-items: center; justify-content: space-between; gap: 24px; padding: 24px 0; border-bottom: 1px solid var(--ui-border); }
.order-help-copy { display: flex; align-items: center; gap: 16px; }
.order-help-icon { display: grid; width: 44px; height: 44px; flex: none; place-items: center; border: 1px solid var(--ui-border); border-radius: 12px; color: var(--home-secondary); background: rgba(148, 163, 184, 0.035); }
.order-help h3 { font-size: 14px; font-weight: 600; line-height: 1.5; }
.order-help p { margin-top: 4px; color: var(--home-muted); font-size: 12px; line-height: 1.8; }
.text-action { display: inline-flex; flex: none; align-items: center; gap: 8px; color: var(--ui-accent); font-size: 13px; font-weight: 500; line-height: 1.8; }
.text-action:hover { text-decoration: underline; text-underline-offset: 4px; }
.information-section { display: grid; grid-template-columns: minmax(0, 0.85fr) minmax(0, 1.5fr); gap: 80px; padding-top: 64px; padding-bottom: 4px; }
.questions-intro .section-description { margin-top: 12px; }
.questions-intro .text-action { margin-top: 24px; }
.questions-list { border-top: 1px solid var(--ui-border); }
.question { border-bottom: 1px solid var(--ui-border); }
.question summary { display: flex; align-items: center; justify-content: space-between; gap: 20px; padding: 20px 0; color: var(--ui-text-primary); cursor: pointer; font-size: 14px; font-weight: 500; line-height: 1.6; list-style: none; }
.question summary::-webkit-details-marker { display: none; }
.question-toggle { flex: none; color: var(--home-muted); transition: color var(--nt-motion-fast, 120ms), transform var(--nt-motion-enter, 180ms) var(--nt-motion-ease, ease-out); }
.question summary:hover .question-toggle { color: var(--ui-text-primary); }
.question[open] .question-toggle { transform: rotate(45deg); color: var(--ui-accent); }
.question > p { padding: 0 32px 20px 0; color: var(--home-muted); font-size: 13px; line-height: 1.9; }
.nt-home a:focus-visible, .nt-home button:focus-visible, .question summary:focus-visible { outline: 2px solid var(--ui-accent); outline-offset: 4px; }
@media (min-width: 1440px) { .price-rail { grid-template-columns: repeat(5, minmax(0, 1fr)); } }
@media (max-width: 1100px) { .price-rail { grid-template-columns: repeat(2, minmax(0, 1fr)); } }
@media (max-width: 1100px) { .library-toolbar { grid-template-columns: 1fr; gap: 16px; } .library-payment-controls { justify-content: flex-end; } }
@media (max-width: 900px) {
  .purchase-step { padding: 20px 16px; gap: 12px; }
  .information-section { gap: 40px; }
}
@media (max-width: 640px) {
  .home-container { padding-inline: 20px; }
  .hero-content { padding-block: 32px 24px; }
  .library-head { align-items: center; gap: 16px; }
  .library-slogan { margin-left: auto; }
  .library-slogan h1 { font-size: clamp(20px, 5.5vw, 30px); gap: .3em; letter-spacing: 0; }
  .hero-eyebrow { flex: none; font-size: 20px; }
  .library-toolbar { align-items: stretch; gap: 16px; margin-top: 20px; }
  .library-tabs { display: flex; width: 100%; gap: 2px; }
  .library-tabs button, .library-tabs a { flex: 1; gap: 6px; padding-inline: 8px; font-size: 12px; }
  .library-tabs a > svg:first-child { display: none; }
  .library-payment-controls { justify-content: flex-end; flex-wrap: nowrap; gap: 12px; }
  .library-payment-controls :deep(.payment-methods) { justify-content: flex-end; }
  .library-filters { gap: 4px; margin-top: 16px; }
  .library-filters button { padding-inline: 12px; font-size: 12px; }
  .price-rail { grid-template-columns: 1fr; min-height: 184px; gap: 16px; margin-top: 20px; }
  .hero-benefits { grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 12px; margin-top: 20px; }
  .hero-benefit { gap: 8px; padding: 12px 8px; font-size: 11px; }
  .hero-benefit--rate { grid-column: 1 / -1; }
  .purchase-section { padding-top: 40px; }
  .section-heading { align-items: start; flex-direction: column; gap: 12px; }
  .flow-title, .questions-intro h2 { font-size: 24px; }
  .flow-controls { width: 100%; justify-content: space-between; gap: 8px; }
  .flow-tabs { gap: 4px; }
  .flow-tabs button { padding-inline: 8px; font-size: 11px; }
  .purchase-steps { grid-template-columns: 1fr; margin-top: 24px; }
  .purchase-step { padding: 20px; }
  .purchase-step + .purchase-step { border-left: none; border-top: 1px solid var(--ui-border); }
  .step-arrow { display: none; }
  .order-help { align-items: start; flex-direction: column; gap: 16px; }
  .order-help .text-action { margin-left: 60px; }
  .information-section { grid-template-columns: 1fr; gap: 24px; padding-top: 40px; }
  .questions-intro .text-action { margin-top: 16px; }
}
@media (prefers-reduced-motion: reduce) {
  .library-tabs :is(button, a), .library-filters button, .question-toggle { transition: none; }
  .flow-word, .purchase-steps, .slogan-word, .price-rail.is-ready { animation: none; }
  .flow-tabs button, .flow-playback, .text-action { transition: none; }
  .library-tabs :is(button, a):active, .library-filters button:active, .flow-tabs button:active, .flow-playback:active, .text-action:active { transform: none; }
}
</style>
