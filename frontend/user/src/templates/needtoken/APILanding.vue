<template>
  <div class="api-landing">
    <ApiGatewayHero />

    <ApiAccountLinkBanner class="mb-10" next-path="/api-services" />

    <!-- 核心协议与企业级特性矩阵 -->
    <section class="api-protocols" aria-labelledby="api-protocols-title">
      <header class="section-head">
        <div>
          <span class="section-eyebrow">{{ t('needtoken.apiServices.protocolsEyebrow') }}</span>
          <h2 id="api-protocols-title">{{ t('needtoken.apiServices.protocolsTitle') }}</h2>
          <p>{{ t('needtoken.apiServices.protocolsIntro') }}</p>
        </div>
      </header>

      <div class="protocol-grid">
        <article v-for="card in protocolCards" :key="card.key" class="protocol-card">
          <div class="protocol-card-top">
            <span class="protocol-icon" aria-hidden="true"><component :is="card.icon" :size="19" /></span>
            <code class="protocol-endpoint">{{ card.endpoint }}</code>
          </div>
          <h3>{{ t(`needtoken.apiServices.protocolCards.${card.key}.title`) }}</h3>
          <p>{{ t(`needtoken.apiServices.protocolCards.${card.key}.desc`) }}</p>
          <div class="protocol-tags">
            <span v-for="client in card.clients" :key="client">{{ client }}</span>
          </div>
        </article>
      </div>
    </section>

    <!-- 热门模型实时价格速览表 -->
    <section class="api-hot-models" aria-labelledby="api-hot-models-title">
      <header class="section-head">
        <div>
          <span class="section-eyebrow">{{ t('needtoken.apiServices.hotModelsEyebrow') }}</span>
          <h2 id="api-hot-models-title">{{ t('needtoken.apiServices.hotModelsTitle') }}</h2>
          <p>{{ t('needtoken.apiServices.hotModelsIntro') }}</p>
        </div>
        <RouterLink to="/models" class="view-all-link">
          <span>{{ t('needtoken.apiServices.viewAllModels') }}</span>
          <ArrowRight :size="15" aria-hidden="true" />
        </RouterLink>
      </header>

      <div v-if="featuredOffers.length > 0" class="hot-table-wrap">
        <table class="hot-table">
          <thead>
            <tr>
              <th>{{ t('needtoken.models.columns.model') }}</th>
              <th>{{ t('needtoken.models.vendor') }}</th>
              <th>{{ t('needtoken.apiServices.protocolsCol') }}</th>
              <th>{{ t('needtoken.models.columns.input') }}</th>
              <th>{{ t('needtoken.models.columns.output') }}</th>
              <th>{{ t('needtoken.models.columns.cacheHit') }}</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="item in featuredOffers" :key="item.key">
              <td>
                <div class="model-cell">
                  <code class="model-id">{{ item.modelId }}</code>
                  <button
                    type="button"
                    class="copy-btn"
                    :aria-label="t('needtoken.models.copyModelId', { model: item.modelId })"
                    @click="copyModel(item.modelId)"
                  >
                    <Check v-if="copiedId === item.modelId" :size="13" />
                    <Copy v-else :size="13" />
                  </button>
                </div>
              </td>
              <td class="muted-col">{{ item.vendor }}</td>
              <td>
                <div class="ep-badges">
                  <span v-for="ep in (item.endpoints.length ? item.endpoints.slice(0, 3) : [{ key: 'chat', label: 'Chat', path: '/v1/chat/completions' }])" :key="ep.label">
                    {{ ep.label }}
                  </span>
                </div>
              </td>
              <template v-if="item.unit === 'per_call'">
                <td colspan="3" class="price-col">
                  <strong>{{ formatUsd(item.inputPrice) }}</strong>
                  <small> / {{ t('needtoken.models.perCallShort') }}</small>
                </td>
              </template>
              <template v-else-if="item.unit === 'per_million_tokens'">
                <td class="price-col"><strong>{{ formatUsd(item.inputPrice) }}</strong><small>/1M</small></td>
                <td class="price-col"><strong>{{ formatUsd(item.outputPrice) }}</strong><small>/1M</small></td>
                <td class="price-col cache-col">{{ item.cachePrice !== null ? `${formatUsd(item.cachePrice)}/1M` : '—' }}</td>
              </template>
              <template v-else>
                <td colspan="3" class="muted-col">{{ t('needtoken.models.conditionalPrice') }}</td>
              </template>
              <td class="action-col">
                <RouterLink :to="{ path: '/models', query: { q: item.modelId } }" class="table-action">
                  {{ t('needtoken.apiServices.inspectModel') }}
                </RouterLink>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <div v-else class="hot-fallback">
        <div class="hot-fallback-chips">
          <RouterLink v-for="m in DEFAULT_MODELS.slice(0, 6)" :key="m" :to="{ path: '/models', query: { q: m } }" class="fallback-chip">
            <code>{{ m }}</code>
            <ArrowUpRight :size="13" aria-hidden="true" />
          </RouterLink>
        </div>
        <p>{{ t('needtoken.apiServices.hotModelsFallbackHint') }}</p>
      </div>
    </section>

    <ApiIntegrationGuide />

    <section class="api-usage" :aria-label="t('needtoken.apiUse.calculatorTitle')">
      <ApiCostCalculator />
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, defineAsyncComponent, onMounted, onUnmounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { ArrowRight, ArrowUpRight, Blocks, Braces, Check, Copy, Sparkles, Terminal } from 'lucide-vue-next'
import { usePageSeo } from '../../composables/usePageSeo'
import { DEFAULT_MODELS } from './config'
import { formatUsd, selectOffersByGroup, sortOffers } from './lib/pricing'
import { usePricing } from './lib/usePricing'
import ApiGatewayHero from './components/ApiGatewayHero.vue'
import ApiAccountLinkBanner from './components/ApiAccountLinkBanner.vue'
import ApiIntegrationGuide from './components/ApiIntegrationGuide.vue'

const ApiCostCalculator = defineAsyncComponent(() => import('./components/ApiCostCalculator.vue'))
const { t } = useI18n()
const pricing = usePricing()
const copiedId = ref('')
let copyTimer: ReturnType<typeof setTimeout> | undefined

const protocolCards = [
  {
    key: 'openai',
    endpoint: '/v1/responses · /v1/chat/completions',
    icon: Braces,
    clients: ['Codex CLI', 'Cursor', 'OpenAI SDK'],
  },
  {
    key: 'anthropic',
    endpoint: '/v1/messages',
    icon: Terminal,
    clients: ['Claude Code', 'Anthropic SDK', 'Cline'],
  },
  {
    key: 'gemini',
    endpoint: '/v1beta/models',
    icon: Sparkles,
    clients: ['Gemini CLI', 'Google GenAI SDK'],
  },
  {
    key: 'ecosystem',
    endpoint: 'Zero-Code Drop-In',
    icon: Blocks,
    clients: ['Cherry Studio', 'NextChat', 'OpenCode', 'CC Switch'],
  },
] as const

const featuredOffers = computed(() => {
  const deduped = selectOffersByGroup(pricing.offers.value, '')
  return sortOffers(deduped, 'default', DEFAULT_MODELS).slice(0, 6)
})

const copyModel = async (modelId: string) => {
  if (copyTimer) clearTimeout(copyTimer)
  try {
    await navigator.clipboard.writeText(modelId)
    copiedId.value = modelId
    copyTimer = setTimeout(() => { copiedId.value = '' }, 1500)
  } catch {
    copiedId.value = ''
  }
}

onMounted(() => {
  void pricing.ensureLoaded()
})

onUnmounted(() => {
  if (copyTimer) clearTimeout(copyTimer)
})

usePageSeo({
  title: () => t('needtoken.nav.plans'),
  description: () => t('needtoken.apiConnect.heroIntro'),
  canonicalPath: () => '/api-services',
})
</script>

<style scoped>
.api-landing { width: min(1200px, calc(100% - 48px)); margin-inline: auto; padding-bottom: 40px; }
.api-protocols, .api-hot-models { padding-block: 48px; border-top: 1px solid var(--ui-border); }
.section-head { display: flex; align-items: flex-end; justify-content: space-between; flex-wrap: wrap; gap: 16px; }
.section-eyebrow { font-size: 11.5px; font-weight: 600; letter-spacing: .1em; text-transform: uppercase; color: var(--ui-accent); }
.section-head h2 { margin-top: 8px; font-size: clamp(24px, 2.8vw, 32px); font-weight: 650; letter-spacing: -.025em; color: var(--ui-text-primary); }
.section-head p { margin-top: 8px; max-width: 66ch; font-size: 14px; line-height: 1.75; color: var(--ui-text-secondary); }
.view-all-link { display: inline-flex; align-items: center; gap: 6px; min-height: 38px; padding: 0 14px; border: 1px solid var(--ui-border-strong); border-radius: 10px; font-size: 13px; font-weight: 600; color: var(--ui-accent); background: var(--ui-bg-elevated); }
.view-all-link:hover { border-color: var(--ui-accent); background: var(--ui-bg-soft); }

.protocol-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 16px; margin-top: 24px; }
.protocol-card { display: flex; flex-direction: column; gap: 10px; padding: 22px 24px; border: 1px solid var(--ui-border); border-radius: 18px; background: var(--ui-bg-elevated); transition: border-color 160ms; }
.protocol-card:hover { border-color: var(--nt-accent-border-strong, var(--ui-border-strong)); }
.protocol-card-top { display: flex; align-items: center; justify-content: space-between; gap: 10px; }
.protocol-icon { display: grid; place-items: center; width: 38px; height: 38px; border-radius: 10px; border: 1px solid var(--ui-border); background: var(--ui-bg-soft); color: var(--ui-accent); }
.protocol-endpoint { padding: 3px 9px; border-radius: 999px; background: var(--ui-bg-soft); color: var(--ui-text-secondary); font: 11.5px var(--nt-font-mono); }
.protocol-card h3 { margin-top: 4px; font-size: 17px; font-weight: 650; color: var(--ui-text-primary); }
.protocol-card p { font-size: 13px; line-height: 1.75; color: var(--ui-text-secondary); }
.protocol-tags { display: flex; flex-wrap: wrap; gap: 6px; margin-top: auto; padding-top: 8px; }
.protocol-tags span { padding: 2px 9px; border-radius: 6px; border: 1px solid var(--ui-border); font-size: 11.5px; color: var(--ui-text-muted); background: var(--ui-bg-soft); }

.hot-table-wrap { margin-top: 22px; overflow-x: auto; border: 1px solid var(--ui-border); border-radius: 18px; background: var(--ui-bg-elevated); }
.hot-table { width: 100%; min-width: 680px; border-collapse: collapse; text-align: left; font-size: 13px; }
.hot-table th { padding: 13px 16px; border-bottom: 1px solid var(--ui-border); font-size: 12px; font-weight: 600; color: var(--ui-text-muted); background: var(--ui-bg-soft); }
.hot-table td { padding: 13px 16px; border-bottom: 1px solid var(--ui-border); vertical-align: middle; }
.hot-table tbody tr:last-child td { border-bottom: 0; }
.model-cell { display: inline-flex; align-items: center; gap: 6px; }
.model-id { font: 600 13px var(--nt-font-mono); color: var(--ui-text-primary); }
.copy-btn { display: grid; place-items: center; width: 24px; height: 24px; border-radius: 6px; color: var(--ui-text-muted); }
.copy-btn:hover { color: var(--ui-accent); background: var(--ui-bg-soft); }
.muted-col { color: var(--ui-text-secondary); }
.ep-badges { display: flex; flex-wrap: wrap; gap: 5px; }
.ep-badges span { padding: 2px 7px; border-radius: 5px; border: 1px solid var(--ui-border); font: 11px var(--nt-font-mono); color: var(--ui-text-muted); }
.price-col { font-family: var(--nt-font-mono); color: var(--ui-text-primary); white-space: nowrap; }
.price-col small { color: var(--ui-text-muted); font-size: 11px; margin-left: 2px; }
.cache-col { color: #059669; }
.dark .cache-col { color: #34d399; }
.action-col { text-align: right; white-space: nowrap; }
.table-action { font-size: 12px; font-weight: 600; color: var(--ui-accent); }
.table-action:hover { text-decoration: underline; }

.hot-fallback { margin-top: 22px; padding: 22px; border: 1px solid var(--ui-border); border-radius: 18px; background: var(--ui-bg-elevated); }
.hot-fallback-chips { display: flex; flex-wrap: wrap; gap: 10px; }
.fallback-chip { display: inline-flex; align-items: center; gap: 6px; padding: 7px 12px; border: 1px solid var(--ui-border); border-radius: 999px; font: 12.5px var(--nt-font-mono); color: var(--ui-text-primary); background: var(--ui-bg-soft); }
.fallback-chip:hover { border-color: var(--ui-accent); color: var(--ui-accent); }
.hot-fallback p { margin-top: 12px; font-size: 12.5px; color: var(--ui-text-muted); }

.api-usage { padding-top: 48px; border-top: 1px solid var(--ui-border); }
.api-usage :deep(.nt-api-cost) { margin-top: 0; }
@media (max-width: 760px) {
  .api-landing { width: calc(100% - 32px); padding-bottom: 24px; }
  .api-protocols, .api-hot-models, .api-usage { padding-block: 32px; }
  .protocol-grid { grid-template-columns: minmax(0, 1fr); }
}
</style>
