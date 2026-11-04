<template>
  <div class="nt-free-id mx-auto w-full max-w-[1200px] px-4 sm:px-6">
    <header class="nt-id-heading">
      <span><Apple :size="17" aria-hidden="true" />{{ t('needtoken.freeAppleId.eyebrow') }}</span>
      <h1>{{ t('needtoken.freeAppleId.title') }}</h1>
      <p>{{ t('needtoken.freeAppleId.subtitle') }}</p>
    </header>

    <div class="nt-id-note"><ShieldCheck :size="23" aria-hidden="true" /><div><h2>{{ t('needtoken.freeAppleId.noticeTitle') }}</h2><p>{{ t('needtoken.freeAppleId.notice') }}</p></div></div>

    <section :aria-label="t('needtoken.freeAppleId.resources')">
      <div class="nt-id-toolbar">
        <div><h2>{{ t('needtoken.freeAppleId.resources') }}</h2><span>{{ t('needtoken.freeAppleId.count', { count: resources.length }) }}</span></div>
        <RouterLink to="/faq"><CircleHelp :size="15" aria-hidden="true" />{{ t('needtoken.freeAppleId.help') }}<ArrowUpRight :size="14" aria-hidden="true" /></RouterLink>
      </div>

      <div v-if="appStore.loading || reloading" class="nt-id-state" role="status"><LoaderCircle class="animate-spin" :size="26" aria-hidden="true" /><h3>{{ t('needtoken.common.loading') }}</h3></div>
      <div v-else-if="!appStore.config" class="nt-id-state" role="alert"><CircleHelp :size="30" aria-hidden="true" /><h3>{{ t('needtoken.faq.loadError') }}</h3><button type="button" @click="reload">{{ t('needtoken.common.retry') }}</button></div>
      <div v-else-if="!resources.length" class="nt-id-state" role="status"><PackageOpen :size="34" aria-hidden="true" /><h3>{{ t('needtoken.freeAppleId.empty') }}</h3><p>{{ t('needtoken.freeAppleId.emptyDescription') }}</p><RouterLink to="/blog">{{ t('needtoken.freeAppleId.guides') }}<ArrowRight :size="15" aria-hidden="true" /></RouterLink></div>
      <div v-else class="nt-id-grid">
        <article v-for="resource in resources" :key="resource.id" class="nt-id-card">
          <div class="nt-id-card-top"><div class="nt-id-icon"><Apple :size="26" aria-hidden="true" /></div><span v-if="resource.region"><Globe2 :size="13" aria-hidden="true" />{{ resource.region }}</span></div>
          <h3>{{ localized(resource.title) || t('needtoken.freeAppleId.resourceFallback') }}</h3>
          <p>{{ localized(resource.description) }}</p>
          <div class="nt-id-meta"><span>{{ t('needtoken.freeAppleId.manualRecord') }}</span><time v-if="formatDate(resource.updated_at)" :datetime="resource.updated_at">{{ t('needtoken.freeAppleId.updated', { date: formatDate(resource.updated_at) }) }}</time></div>
          <a :href="resource.url" target="_blank" rel="noopener noreferrer" referrerpolicy="no-referrer">{{ t('needtoken.freeAppleId.openResource') }}<ArrowUpRight :size="17" aria-hidden="true" /></a>
        </article>
      </div>
    </section>

    <section class="nt-id-guide" :aria-label="t('needtoken.freeAppleId.beforeUse')">
      <h2>{{ t('needtoken.freeAppleId.beforeUse') }}</h2>
      <div><article v-for="(icon, index) in [ExternalLink, LockKeyhole, CircleHelp]" :key="index"><span>{{ String(index + 1).padStart(2, '0') }}</span><component :is="icon" :size="19" aria-hidden="true" /><h3>{{ t(`needtoken.freeAppleId.steps.${index}.title`) }}</h3><p>{{ t(`needtoken.freeAppleId.steps.${index}.description`) }}</p></article></div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { Apple, ArrowRight, ArrowUpRight, CircleHelp, ExternalLink, Globe2, LoaderCircle, LockKeyhole, PackageOpen, ShieldCheck } from 'lucide-vue-next'
import { usePageSeo } from '../../composables/usePageSeo'
import { useAppStore } from '../../stores/app'
import { getLocalizedText } from '../../utils/resellerSiteConfig'
import { publicAppleResources } from './lib/helpContent'

const { t } = useI18n()
const appStore = useAppStore()
const reloading = ref(false)
async function reload() { reloading.value = true; try { await appStore.loadConfig(true) } finally { reloading.value = false } }
const localized = (value: Record<string, unknown>) => getLocalizedText(value, appStore.locale)
const resources = computed(() => appStore.isResellerTenant ? [] : publicAppleResources(appStore.config?.needtoken_resources?.apple_ids))
const formatDate = (value: string) => {
  if (!value) return ''
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? '' : date.toLocaleDateString(appStore.locale, { year: 'numeric', month: 'short', day: 'numeric' })
}
usePageSeo({ title: () => t('needtoken.freeAppleId.title'), description: () => t('needtoken.freeAppleId.subtitle'), canonicalPath: () => '/free-apple-id' })
</script>

<style scoped>
.nt-free-id { padding-top: 58px; padding-bottom: 76px; }
.nt-id-heading { max-width: 760px; margin: 0 auto 38px; text-align: center; }
.nt-id-heading > span { display: inline-flex; align-items: center; gap: 8px; color: var(--ui-accent); font-size: 13px; }
.nt-id-heading h1 { margin: 16px 0; font-size: clamp(32px, 4.5vw, 52px); font-weight: 750; letter-spacing: -.04em; }
.nt-id-heading > p { color: var(--ui-text-muted); font-size: 15px; line-height: 1.9; }
.nt-id-note { display: flex; gap: 14px; padding: 20px 24px; border: 1px solid color-mix(in srgb, var(--ui-accent) 20%, transparent); border-radius: 15px; background: color-mix(in srgb, var(--ui-accent) 4.5%, transparent); }
.nt-id-note > svg { flex-shrink: 0; color: var(--ui-accent); margin-top: 2px; }
.nt-id-note h2 { margin-bottom: 5px; font-size: 14px; font-weight: 650; }
.nt-id-note p { font-size: 13px; color: var(--ui-text-muted); line-height: 1.9; }
.nt-id-toolbar { display: flex; flex-wrap: wrap; align-items: center; justify-content: space-between; gap: 15px; margin: 42px 0 21px; }
.nt-id-toolbar > div { display: flex; align-items: baseline; gap: 12px; }
.nt-id-toolbar h2 { font-size: 20px; font-weight: 650; }
.nt-id-toolbar span { color: var(--ui-text-muted); font-size: 12px; }
.nt-id-toolbar > a { display: inline-flex; align-items: center; gap: 6px; color: var(--ui-accent); font-size: 13px; }
.nt-id-grid { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 20px; }
.nt-id-card { display: flex; flex-direction: column; min-width: 0; padding: 26px; border: 1px solid var(--ui-border); border-radius: 18px; background: linear-gradient(145deg, color-mix(in srgb, var(--ui-accent) 4%, transparent), color-mix(in srgb, var(--ui-bg-elevated) 85%, transparent) 65%); }
.nt-id-card-top { display: flex; justify-content: space-between; align-items: center; gap: 12px; margin-bottom: 24px; }
.nt-id-icon { display: grid; place-items: center; width: 54px; height: 54px; color: var(--ui-accent); background: color-mix(in srgb, var(--ui-accent) 8%, transparent); border: 1px solid color-mix(in srgb, var(--ui-accent) 18%, transparent); border-radius: 14px; }
.nt-id-card-top > span { display: inline-flex; align-items: center; gap: 5px; padding: 5px 10px; border: 1px solid var(--ui-border); border-radius: 100px; color: var(--ui-text-muted); font-size: 12px; }
.nt-id-card h3 { font-size: 20px; font-weight: 650; overflow-wrap: anywhere; }
.nt-id-card > p { margin: 12px 0 22px; color: var(--ui-text-muted); font-size: 13px; line-height: 1.9; white-space: pre-line; overflow-wrap: anywhere; }
.nt-id-meta { display: flex; flex-wrap: wrap; gap: 7px 10px; align-items: center; margin-top: auto; padding-bottom: 16px; color: var(--ui-text-muted); font-size: 11px; }
.nt-id-meta > span { color: var(--ui-accent); }
.nt-id-card > a { display: flex; align-items: center; justify-content: space-between; gap: 10px; padding: 12px 15px; border: 1px solid color-mix(in srgb, var(--ui-accent) 28%, transparent); border-radius: 10px; background: color-mix(in srgb, var(--ui-accent) 8%, transparent); color: var(--ui-accent); font-size: 13px; font-weight: 600; }
.nt-id-state { display: grid; justify-items: center; align-content: center; gap: 16px; min-height: 285px; padding: 30px 22px; border: 1px dashed var(--ui-border); border-radius: 17px; text-align: center; }
.nt-id-state > svg { color: var(--ui-accent); }
.nt-id-state h3 { font-weight: 600; }
.nt-id-state p { max-width: 50ch; font-size: 14px; line-height: 1.9; color: var(--ui-text-muted); }
.nt-id-state a, .nt-id-state button { display: inline-flex; align-items: center; gap: 7px; color: var(--ui-accent); font-size: 13px; }
.nt-id-guide { margin-top: 54px; }
.nt-id-guide > h2 { margin-bottom: 21px; font-size: 20px; font-weight: 650; }
.nt-id-guide > div { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); border: 1px solid var(--ui-border); border-radius: 17px; background: color-mix(in srgb, var(--ui-bg-elevated) 65%, transparent); overflow: hidden; }
.nt-id-guide article { display: grid; grid-template-columns: 1fr auto; gap: 12px; padding: 26px; }
.nt-id-guide article + article { border-left: 1px solid var(--ui-border); }
.nt-id-guide article > span { color: var(--ui-accent); font-size: 12px; font-family: monospace; }
.nt-id-guide article > svg { color: var(--ui-text-muted); }
.nt-id-guide article h3 { grid-column: 1 / -1; font-size: 14px; font-weight: 600; }
.nt-id-guide article p { grid-column: 1 / -1; color: var(--ui-text-muted); font-size: 13px; line-height: 1.9; }
@media (max-width: 950px) { .nt-id-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); } }
@media (max-width: 640px) { .nt-free-id { padding-top: 36px; padding-bottom: 52px; } .nt-id-grid, .nt-id-guide > div { grid-template-columns: 1fr; } .nt-id-guide article + article { border-left: 0; border-top: 1px solid var(--ui-border); } .nt-id-note { padding: 18px; gap: 11px; } .nt-id-note > svg { width: 20px; } .nt-id-card { padding: 22px; } }
</style>
