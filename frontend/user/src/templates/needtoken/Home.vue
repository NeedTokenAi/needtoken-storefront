<template>
  <div class="nt-home">
    <section class="home-hero" aria-labelledby="home-title">
      <div class="home-container hero-content">
        <div class="hero-eyebrow">
          <Sparkles :size="13" aria-hidden="true" />
          {{ t('needtoken.homeServices.eyebrow') }}
        </div>
        <h1 id="home-title" class="hero-title">
          {{ t('needtoken.homeServices.title') }}
          <span>{{ t('needtoken.homeServices.titleAccent') }}</span>
        </h1>
        <p class="hero-description">{{ t('needtoken.homeServices.subtitle') }}</p>

        <div class="service-grid">
          <article v-for="service in services" :key="service.key" class="service-card" :class="`service--${service.key}`">
            <div class="service-topline">
              <span class="service-icon"><component :is="service.icon" :size="23" :stroke-width="1.6" aria-hidden="true" /></span>
              <span class="service-tag">{{ t(`needtoken.homeServices.services.${service.key}.tag`) }}</span>
            </div>
            <h2>{{ t(`needtoken.homeServices.services.${service.key}.title`) }}</h2>
            <p class="service-description">{{ t(`needtoken.homeServices.services.${service.key}.description`) }}</p>
            <div class="service-footnote">
              <Check :size="14" aria-hidden="true" />
              {{ t(`needtoken.homeServices.services.${service.key}.note`) }}
            </div>
            <RouterLink :to="service.to" class="service-action">
              <span>{{ t(`needtoken.homeServices.services.${service.key}.action`) }}</span>
              <ArrowUpRight :size="18" aria-hidden="true" />
            </RouterLink>
          </article>
        </div>

        <div class="hero-benefits">
          <div v-for="benefit in benefits" :key="benefit.key" class="hero-benefit">
            <component :is="benefit.icon" :size="15" :stroke-width="1.7" aria-hidden="true" />
            <span>{{ t(`needtoken.homeServices.benefits.${benefit.key}`) }}</span>
          </div>
        </div>
      </div>
    </section>

    <section class="home-container purchase-section" aria-labelledby="purchase-title">
      <div class="section-heading">
        <div>
          <p class="section-eyebrow">{{ t('needtoken.homeServices.steps.eyebrow') }}</p>
          <h2 id="purchase-title">{{ t('needtoken.homeServices.steps.title') }}</h2>
        </div>
        <p class="section-description">{{ t('needtoken.homeServices.steps.description') }}</p>
      </div>
      <ol class="purchase-steps">
        <li v-for="(step, index) in steps" :key="step" class="purchase-step">
          <span class="step-index" aria-hidden="true">{{ String(index + 1).padStart(2, '0') }}</span>
          <div>
            <h3>{{ t(`needtoken.homeServices.steps.${step}.title`) }}</h3>
            <p>{{ t(`needtoken.homeServices.steps.${step}.description`) }}</p>
          </div>
          <ArrowRight v-if="index < steps.length - 1" class="step-arrow" :size="17" aria-hidden="true" />
        </li>
      </ol>
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
import { useI18n } from 'vue-i18n'
import { ArrowRight, ArrowUpRight, Check, ClipboardList, Clock3, KeyRound, PackageCheck, Plus, ReceiptText, ShieldCheck, Sparkles, Zap } from 'lucide-vue-next'
import { usePageSeo } from '../../composables/usePageSeo'
import { CATEGORY_SLUGS } from './config'

const { t } = useI18n()
usePageSeo({
  title: () => t('needtoken.homeServices.seoTitle'),
  description: () => t('needtoken.homeServices.subtitle'),
})
const services = [
  { key: 'recharge', icon: Zap, to: `/categories/${CATEGORY_SLUGS.recharge}` },
  { key: 'products', icon: PackageCheck, to: `/categories/${CATEGORY_SLUGS.rental}` },
  { key: 'api', icon: KeyRound, to: '/plans' },
]
const benefits = [
  { key: 'selfService', icon: Clock3 },
  { key: 'orderTracking', icon: ClipboardList },
  { key: 'afterSales', icon: ShieldCheck },
  { key: 'business', icon: ReceiptText },
]
const steps = ['choose', 'pay', 'receive']
const faqKeys = ['service', 'recharge', 'delivery', 'business']
</script>

<style scoped>
.nt-home { --home-secondary: var(--ui-text-secondary, #b4bfcc); --home-muted: var(--ui-text-muted, #7d8a99); }
.home-container { width: 100%; max-width: 1200px; margin-inline: auto; padding-inline: 24px; }
.home-hero { position: relative; isolation: isolate; overflow: hidden; border-bottom: 1px solid var(--ui-border); background: radial-gradient(ellipse at 50% -16%, rgba(45, 212, 167, 0.12), transparent 56%); }
.home-hero::before { content: ''; position: absolute; z-index: -1; inset: 0; opacity: 0.42; background-image: radial-gradient(rgba(148, 163, 184, 0.26) 0.7px, transparent 0.7px); background-size: 25px 25px; mask-image: linear-gradient(#000, transparent 72%); pointer-events: none; }
.hero-content { padding-block: 38px 26px; }
.hero-eyebrow { display: flex; align-items: center; justify-content: center; gap: 8px; font-size: 12px; font-weight: 500; letter-spacing: 0.08em; color: var(--home-secondary); }
.hero-eyebrow svg { color: var(--ui-accent); }
.hero-title { margin: 17px 0 0; text-align: center; font-size: clamp(35px, 4.1vw, 56px); font-weight: 750; line-height: 1.17; letter-spacing: -0.045em; text-wrap: balance; }
.hero-title span { color: var(--ui-accent); }
.hero-description { max-width: 720px; margin: 15px auto 0; color: var(--home-muted); font-size: 15px; line-height: 1.8; text-align: center; text-wrap: balance; }
.service-grid { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 16px; margin-top: 32px; }
.service-card { --service-color: #2dd4a7; --service-rgb: 45, 212, 167; display: flex; flex-direction: column; position: relative; overflow: hidden; padding: 25px; border: 1px solid rgba(148, 163, 184, 0.2); border-radius: 20px; background: linear-gradient(145deg, rgba(var(--service-rgb), 0.065), rgba(14, 18, 24, 0.9) 56%); backdrop-filter: blur(18px); transition: border-color 180ms ease, transform 180ms ease; }
.service-card:hover { border-color: rgba(var(--service-rgb), 0.48); transform: translateY(-3px); }
.service--products { --service-color: #b1baff; --service-rgb: 177, 186, 255; }
.service--api { --service-color: #efcc88; --service-rgb: 239, 204, 136; }
.service-topline { display: flex; align-items: center; justify-content: space-between; gap: 12px; }
.service-icon { display: grid; width: 44px; height: 44px; place-items: center; color: var(--service-color); border: 1px solid rgba(var(--service-rgb), 0.18); border-radius: 13px; background: rgba(var(--service-rgb), 0.075); }
.service-tag { padding: 4px 9px; border: 1px solid rgba(var(--service-rgb), 0.16); border-radius: 999px; color: var(--service-color); background: rgba(var(--service-rgb), 0.045); font-size: 11px; line-height: 1.5; }
.service-card h2 { margin-top: 22px; font-size: 27px; line-height: 1.25; font-weight: 700; letter-spacing: -0.035em; }
.service-description { flex: 1; margin-top: 11px; min-height: 48px; color: var(--home-secondary); font-size: 13px; line-height: 1.8; text-wrap: pretty; }
.service-footnote { display: flex; align-items: center; gap: 6px; margin-top: 15px; color: var(--home-muted); font-size: 11px; }
.service-footnote svg { flex: none; color: var(--service-color); }
.service-action { display: flex; align-items: center; justify-content: space-between; gap: 12px; margin-top: 23px; padding: 11px 15px; border: 1px solid rgba(var(--service-rgb), 0.3); border-radius: 10px; color: var(--service-color); background: rgba(var(--service-rgb), 0.085); font-size: 14px; font-weight: 650; transition: background 180ms ease; }
.service-action:hover { background: rgba(var(--service-rgb), 0.16); }
.service--recharge .service-action { color: #07120f; border-color: transparent; background: #2dd4a7; }
.service--recharge .service-action:hover { background: #5eead4; }
.hero-benefits { display: flex; flex-wrap: wrap; justify-content: center; gap: 14px 35px; margin-top: 25px; }
.hero-benefit { display: flex; align-items: center; gap: 7px; color: var(--home-muted); font-size: 12px; }
.hero-benefit svg { color: var(--home-secondary); }
.purchase-section { padding-top: 67px; }
.section-heading { display: flex; align-items: end; justify-content: space-between; gap: 22px; }
.section-eyebrow { margin-bottom: 11px; color: var(--ui-accent); font-size: 11px; font-weight: 650; letter-spacing: 0.09em; }
.section-heading h2, .questions-intro h2 { font-size: 27px; line-height: 1.4; font-weight: 650; letter-spacing: -0.035em; text-wrap: balance; }
.section-description { max-width: 365px; color: var(--home-muted); font-size: 13px; line-height: 1.85; text-wrap: pretty; }
.purchase-steps { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); margin-top: 28px; border: 1px solid var(--ui-border); border-radius: 16px; background: rgba(14, 18, 24, 0.45); list-style: none; }
.purchase-step { display: flex; align-items: start; gap: 14px; position: relative; padding: 25px; }
.purchase-step + .purchase-step { border-left: 1px solid var(--ui-border); }
.step-index { flex: none; padding-top: 3px; color: var(--ui-accent); font-family: var(--nt-font-mono); font-size: 12px; }
.purchase-step h3 { font-size: 15px; line-height: 1.5; font-weight: 600; }
.purchase-step p { margin-top: 8px; color: var(--home-muted); font-size: 12px; line-height: 1.8; }
.step-arrow { flex: none; position: absolute; right: -9px; top: 29px; z-index: 1; color: var(--home-muted); background: var(--ui-bg-page); }
.order-help { display: flex; align-items: center; justify-content: space-between; gap: 22px; padding: 24px 0; border-bottom: 1px solid var(--ui-border); }
.order-help-copy { display: flex; align-items: center; gap: 14px; }
.order-help-icon { display: grid; width: 43px; height: 43px; flex: none; place-items: center; border: 1px solid var(--ui-border); border-radius: 12px; color: var(--home-secondary); background: rgba(148, 163, 184, 0.035); }
.order-help h3 { font-size: 14px; font-weight: 600; line-height: 1.5; }
.order-help p { margin-top: 4px; color: var(--home-muted); font-size: 12px; line-height: 1.8; }
.text-action { display: inline-flex; flex: none; align-items: center; gap: 7px; color: var(--ui-accent); font-size: 13px; font-weight: 550; line-height: 1.8; }
.text-action:hover { text-decoration: underline; text-underline-offset: 4px; }
.information-section { display: grid; grid-template-columns: minmax(0, 0.85fr) minmax(0, 1.5fr); gap: 80px; padding-top: 67px; padding-bottom: 5px; }
.questions-intro .section-description { margin-top: 13px; }
.questions-intro .text-action { margin-top: 22px; }
.questions-list { border-top: 1px solid var(--ui-border); }
.question { border-bottom: 1px solid var(--ui-border); }
.question summary { display: flex; align-items: center; justify-content: space-between; gap: 20px; padding: 20px 0; color: var(--ui-text-primary); cursor: pointer; font-size: 14px; font-weight: 500; line-height: 1.6; list-style: none; }
.question summary::-webkit-details-marker { display: none; }
.question-toggle { flex: none; color: var(--home-muted); transition: transform 180ms ease; }
.question[open] .question-toggle { transform: rotate(45deg); color: var(--ui-accent); }
.question > p { padding: 0 30px 20px 0; color: var(--home-muted); font-size: 13px; line-height: 1.9; }
.nt-home a:focus-visible, .question summary:focus-visible { outline: 2px solid var(--ui-accent); outline-offset: 5px; }
@media (max-width: 900px) {
  .service-card { padding: 21px; }
  .service-card h2 { font-size: 23px; }
  .service-description { min-height: 72px; }
  .service-tag, .service-footnote { font-size: 10px; }
  .purchase-step { padding: 21px 17px; gap: 11px; }
  .information-section { gap: 40px; }
}
@media (max-width: 640px) {
  .home-container { padding-inline: 20px; }
  .hero-content { padding-block: 24px 22px; }
  .hero-eyebrow { font-size: 11px; }
  .hero-title { margin-top: 12px; font-size: 31px; }
  .hero-title span { display: block; }
  .hero-description { display: none; }
  .service-grid { grid-template-columns: 1fr; gap: 10px; margin-top: 23px; }
  .service-card { display: grid; grid-template-columns: minmax(0, 1fr) 84px; gap: 5px 12px; min-height: 118px; padding: 16px; border-radius: 15px; }
  .service-topline { position: absolute; top: 16px; left: 16px; }
  .service-icon { width: 30px; height: 30px; border-radius: 9px; }
  .service-icon svg { width: 18px; height: 18px; }
  .service-card h2 { display: flex; align-items: center; grid-column: 1; grid-row: 1; min-height: 30px; margin: 0; padding-left: 40px; font-size: 20px; }
  .service-description { grid-column: 1; grid-row: 2; min-height: 0; margin: 3px 0 0; font-size: 11px; line-height: 1.65; }
  .service-tag, .service-footnote { display: none; }
  .service-action { align-self: center; justify-content: center; grid-column: 2; grid-row: 1 / span 2; min-height: 44px; margin: 0; padding: 9px 8px; border-radius: 10px; font-size: 12px; text-align: center; }
  .service-action svg { display: none; }
  .hero-benefits { display: grid; grid-template-columns: repeat(2, max-content); gap: 10px 23px; margin-top: 20px; }
  .hero-benefit { font-size: 11px; }
  .purchase-section { padding-top: 43px; }
  .section-heading { align-items: start; flex-direction: column; gap: 10px; }
  .section-heading h2, .questions-intro h2 { font-size: 24px; }
  .purchase-steps { grid-template-columns: 1fr; margin-top: 22px; }
  .purchase-step { padding: 22px; }
  .purchase-step + .purchase-step { border-left: none; border-top: 1px solid var(--ui-border); }
  .step-arrow { display: none; }
  .order-help { align-items: start; flex-direction: column; gap: 15px; }
  .order-help .text-action { margin-left: 57px; }
  .information-section { grid-template-columns: 1fr; gap: 26px; padding-top: 44px; }
  .questions-intro .text-action { margin-top: 15px; }
}
@media (prefers-reduced-motion: reduce) {
  .service-card, .service-action, .question-toggle { transition: none; }
  .service-card:hover { transform: none; }
}
</style>
