<template>
  <footer class="nt-footer-next">
    <div class="nt-footer-card">
      <div class="nt-footer-top">
        <div class="nt-footer-brand-block">
          <RouterLink to="/" class="nt-footer-brand" :aria-label="brandName"><img v-if="resellerLogo" :src="resellerLogo" alt="" width="36" height="36" /><BrandMark v-else-if="!appStore.isResellerTenant" class="h-9 w-9" /><span>{{ brandName }}</span></RouterLink>
          <p class="nt-footer-description">{{ t('needtoken.footer.tagline') }}</p>
          <div v-if="support.businessEmail" class="nt-footer-business"><span>{{ t('needtoken.support.business') }}</span><a :href="emailHref"><Mail :size="15" aria-hidden="true" /><span>{{ support.businessEmail }}</span><ArrowUpRight :size="13" aria-hidden="true" /></a></div>
        </div>
        <nav class="nt-footer-nav" :aria-label="t('needtoken.footerPreview.navigation')">
          <div v-for="column in columns" :key="column.key" class="nt-footer-column">
            <h2>{{ column.title }}</h2><RouterLink v-for="link in column.links" :key="link.path" :to="link.path">{{ link.label }}</RouterLink>
            <a v-if="column.key === 'help'" :href="CONSOLE_URL" rel="noopener">{{ t('needtoken.footer.console') }}<ArrowUpRight :size="12" aria-hidden="true" /></a>
          </div>
        </nav>
      </div>
      <section v-if="qrCards.length" class="nt-footer-contact" :aria-label="t('needtoken.support.scanTitle')">
        <div class="nt-footer-contact-copy">
          <span class="nt-footer-section-label"><ScanLine :size="15" aria-hidden="true" />{{ t('needtoken.footerPreview.contactLabel', { brand: brandName }) }}</span>
          <h2>{{ t('needtoken.footerPreview.scanHeading') }}</h2><p>{{ t('needtoken.footerPreview.scanDescription') }}</p>
          <p v-if="support.serviceHours" class="nt-footer-hours"><Clock3 :size="13" aria-hidden="true" />{{ t('needtoken.support.hours', { hours: support.serviceHours }) }}</p>
          <a v-if="support.contactUrl" :href="support.contactUrl" target="_blank" rel="noopener noreferrer" class="nt-footer-contact-link">{{ t('needtoken.support.openContact') }}<ArrowUpRight :size="14" aria-hidden="true" /></a>
        </div>
        <div class="nt-footer-qr-list"><figure v-for="card in qrCards" :key="card.key"><div class="nt-footer-qr-image"><img :src="getImageUrl(card.image)" :alt="card.label" width="112" height="112" loading="lazy" decoding="async" referrerpolicy="no-referrer" /></div><figcaption><span>{{ card.label }}</span><small>{{ t('needtoken.footerPreview.scanHint') }}</small></figcaption></figure></div>
      </section>
      <div class="nt-footer-legal"><span>© {{ year }} {{ brandName }}</span><div><RouterLink to="/terms">{{ t('footer.terms') }}</RouterLink><RouterLink to="/privacy">{{ t('footer.privacy') }}</RouterLink></div></div>
      <FooterShapeWordmark :word="brandName" />
    </div>
  </footer>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { ArrowUpRight, Clock3, Mail, ScanLine } from 'lucide-vue-next'
import { useAppStore } from '../../../stores/app'
import { getImageUrl } from '../../../utils/image'
import { CONSOLE_URL } from '../config'
import { supportEmailHref, type NeedtokenSupport } from '../lib/supportConfig'
import BrandMark from './BrandMark.vue'
import FooterShapeWordmark from './FooterShapeWordmark.vue'
const props = defineProps<{ brandName: string; resellerLogo?: string; support: NeedtokenSupport }>()
const { t } = useI18n(), appStore = useAppStore()
const year = new Date().getFullYear()
const emailHref = computed(() => supportEmailHref(props.support.businessEmail))
const qrCards = computed(() => [{ key: 'wecom', label: t('needtoken.support.wecom'), image: props.support.wecomQr }, { key: 'qq', label: t('needtoken.support.qq'), image: props.support.qqQr }].filter(card => !!card.image))
const columns = computed(() => [
  { key: 'products', title: t('needtoken.footer.products'), links: [{ path: '/categories/recharge', label: t('needtoken.nav.recharge') }, { path: '/categories/rental', label: t('needtoken.nav.rental') }, { path: '/plans', label: t('needtoken.nav.plans') }, { path: '/models', label: t('needtoken.nav.models') }] },
  { key: 'help', title: t('needtoken.nav.help'), links: [{ path: '/blog', label: t('needtoken.nav.blog') }, { path: '/docs', label: t('needtoken.nav.docs') }, { path: '/faq', label: t('needtoken.nav.faq') }] },
  { key: 'support', title: t('needtoken.footer.support'), links: [{ path: '/tools/environment', label: t('needtoken.nav.environment') }, { path: '/free-apple-id', label: t('needtoken.nav.appleIds') }, { path: '/guest/orders', label: t('navbar.guestOrders') }] },
])
</script>

<style scoped>
.nt-footer-next { width: 100%; margin-top: clamp(64px, 7vw, 112px); padding: 0 24px 24px; color: var(--ui-text-muted); }
.nt-footer-card { max-width: 1320px; margin: 0 auto; overflow: hidden; isolation: isolate; border: 1px solid rgba(160,190,185,.13); border-radius: 28px; background: radial-gradient(ellipse at 0 0, #2dd4a70b, transparent 50%), radial-gradient(ellipse at 100% 100%, #8b5cf609, transparent 65%), #0e1218; box-shadow: inset 0 1px 0 #ffffff05; }
.nt-footer-top { display: grid; grid-template-columns: minmax(0,1fr) minmax(0,1.7fr); gap: 56px; padding: 42px 42px 36px; }
.nt-footer-brand { display: inline-flex; align-items: center; gap: 11px; max-width: 100%; color: var(--ui-text-primary); font-size: 21px; font-weight: 700; letter-spacing: -.04em; }
.nt-footer-brand > img { flex-shrink: 0; border-radius: 9px; object-fit: contain; }.nt-footer-brand > span { min-width: 0; overflow-wrap: anywhere; }
.nt-footer-description { max-width: 31ch; margin-top: 17px; font-size: 13px; line-height: 1.9; }
.nt-footer-business { margin-top: 26px; font-size: 11px; }.nt-footer-business a { display: flex; align-items: center; width: fit-content; max-width: 100%; gap: 8px; margin-top: 8px; color: var(--ui-text-secondary); font-size: 13px; }.nt-footer-business a > span { overflow-wrap: anywhere; }.nt-footer-business a > svg { flex-shrink: 0; color: var(--ui-accent); }
.nt-footer-nav { display: grid; grid-template-columns: repeat(3,minmax(0,1fr)); gap: 30px; padding-top: 8px; }
.nt-footer-column { display: flex; flex-direction: column; align-items: flex-start; gap: 13px; min-width: 0; font-size: 13px; line-height: 1.65; }.nt-footer-column h2 { margin-bottom: 6px; color: var(--ui-text-primary); font-size: 13px; font-weight: 600; }.nt-footer-column a { display: inline-flex; align-items: center; gap: 5px; max-width: 100%; overflow-wrap: anywhere; }
.nt-footer-card a { transition: color 180ms ease; }.nt-footer-card a:hover { color: var(--ui-accent); }
.nt-footer-contact { display: flex; align-items: center; justify-content: space-between; gap: 32px; margin: 0 42px; padding: 28px 0; border-top: 1px solid var(--ui-border); border-bottom: 1px solid var(--ui-border); }.nt-footer-contact-copy { min-width: 0; max-width: 54ch; }
.nt-footer-section-label { display: inline-flex; align-items: center; gap: 7px; color: var(--ui-accent); font-size: 11px; }.nt-footer-contact-copy h2 { margin-top: 9px; color: var(--ui-text-primary); font-size: 23px; line-height: 1.35; font-weight: 600; letter-spacing: -.035em; }.nt-footer-contact-copy > p { margin-top: 9px; font-size: 12px; line-height: 1.85; }.nt-footer-contact-copy > p.nt-footer-hours { display: flex; align-items: center; gap: 6px; margin-top: 11px; font-size: 11px; }.nt-footer-hours > svg { flex-shrink: 0; }.nt-footer-contact-link { display: inline-flex; align-items: center; gap: 5px; margin-top: 13px; color: var(--ui-text-secondary); font-size: 12px; }
.nt-footer-qr-list { display: flex; flex-shrink: 0; gap: 18px; }.nt-footer-qr-list figure { min-width: 0; text-align: center; }.nt-footer-qr-image { padding: 8px; width: 128px; border: 1px solid rgba(255,255,255,.15); border-radius: 14px; background: #fff; }.nt-footer-qr-image img { display: block; width: 100%; height: auto; aspect-ratio: 1; object-fit: contain; }.nt-footer-qr-list figcaption { display: flex; flex-direction: column; gap: 4px; margin-top: 10px; color: var(--ui-text-secondary); font-size: 12px; }.nt-footer-qr-list figcaption small { color: var(--ui-text-muted); font-size: 10px; }
.nt-footer-legal { display: flex; flex-wrap: wrap; justify-content: space-between; align-items: center; gap: 12px; padding: 24px 42px 0; font-size: 11px; line-height: 1.7; }.nt-footer-legal > div { display: flex; flex-wrap: wrap; gap: 20px; }.nt-footer-legal + :deep(.nt-footer-wordmark) { margin-top: 14px; }
@media (max-width:900px) { .nt-footer-top { grid-template-columns:1fr; gap:32px; padding:34px 30px 30px; }.nt-footer-description { max-width:45ch; }.nt-footer-business { margin-top:18px; }.nt-footer-nav { gap:24px; }.nt-footer-contact { margin-inline:30px; }.nt-footer-legal { padding-inline:30px; } }
@media (max-width:640px) { .nt-footer-next { padding:0 14px 14px; }.nt-footer-card { border-radius:22px; }.nt-footer-top { padding:28px 22px 26px; }.nt-footer-nav { gap:15px; }.nt-footer-column { font-size:12px; gap:11px; }.nt-footer-column h2 { font-size:12px; }.nt-footer-brand { font-size:20px; }.nt-footer-contact { flex-direction:column; align-items:flex-start; gap:24px; margin-inline:22px; padding:24px 0; }.nt-footer-contact-copy h2 { font-size:22px; }.nt-footer-qr-list { align-self:stretch; justify-content:flex-start; gap:16px; }.nt-footer-qr-image { width:118px; padding:8px; }.nt-footer-legal { padding:22px 22px 0; font-size:10px; } }
@media (max-width:359px) { .nt-footer-qr-list { gap:12px; }.nt-footer-qr-image { width:108px; } }
@media (prefers-reduced-motion:reduce) { .nt-footer-card a { transition:none; } }
</style>
