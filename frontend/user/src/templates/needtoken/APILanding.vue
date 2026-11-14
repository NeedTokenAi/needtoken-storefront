<template>
  <div class="api-landing">
    <ApiGatewayHero />
    <ApiIntegrationGuide />
    <section class="api-usage" :aria-label="t('needtoken.apiUse.calculatorTitle')">
      <ApiCostCalculator />
    </section>
  </div>
</template>

<script setup lang="ts">
import { defineAsyncComponent } from 'vue'
import { useI18n } from 'vue-i18n'
import { usePageSeo } from '../../composables/usePageSeo'
import ApiGatewayHero from './components/ApiGatewayHero.vue'
import ApiIntegrationGuide from './components/ApiIntegrationGuide.vue'

const ApiCostCalculator = defineAsyncComponent(() => import('./components/ApiCostCalculator.vue'))
const { t } = useI18n()
usePageSeo({ title: () => t('needtoken.nav.plans'), description: () => t('needtoken.apiConnect.heroIntro'), canonicalPath: () => '/api-services' })
</script>

<style scoped>
.api-landing { width: min(1200px, calc(100% - 48px)); margin-inline: auto; padding-bottom: 32px; }
.api-usage { padding-top: 48px; border-top: 1px solid var(--ui-border); }
.api-usage :deep(.nt-api-cost) { margin-top: 0; }
@media (max-width: 760px) { .api-landing { width: calc(100% - 32px); padding-bottom: 24px; } .api-usage { padding-top: 32px; } }
</style>
