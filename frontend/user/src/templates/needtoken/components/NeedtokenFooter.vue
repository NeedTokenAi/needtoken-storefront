<template>
  <footer class="nt-footer-next">
    <div class="nt-footer-card">
      <div class="nt-footer-top" :class="{ 'has-contact': hasContactPanel }">
        <div class="nt-footer-brand-block">
          <RouterLink to="/" class="nt-footer-brand" :aria-label="brandName">
            <img v-if="resellerLogo" :src="resellerLogo" alt="" width="36" height="36" />
            <BrandMark v-else-if="!appStore.isResellerTenant" class="h-9 w-9" />
            <span>{{ brandName }}</span>
          </RouterLink>
          <p class="nt-footer-description">{{ t('needtoken.footer.tagline') }}</p>
        </div>

        <nav class="nt-footer-nav" :aria-label="t('needtoken.footerPreview.navigation')">
          <div v-for="column in columns" :key="column.key" class="nt-footer-column">
            <h2>{{ column.title }}</h2>
            <RouterLink v-for="link in column.links" :key="link.path" :to="link.path">{{ link.label }}</RouterLink>
            <a v-if="column.key === 'help' && appStore.isResellerTenant" :href="CONSOLE_URL" rel="noopener">{{ t('needtoken.footer.console') }}<ArrowUpRight :size="12" aria-hidden="true" /></a>
            <RouterLink v-else-if="column.key === 'help'" to="/me">{{ t('needtoken.footer.console') }}<ArrowUpRight :size="12" aria-hidden="true" /></RouterLink>
          </div>
        </nav>

        <aside v-if="hasContactPanel" class="nt-footer-contact-panel" :aria-label="t('needtoken.footer.contact')">
          <h2>{{ t('needtoken.footer.contact') }}</h2>
          <div class="nt-footer-contact-body">
            <figure v-if="showWecomQr" class="nt-footer-wecom">
              <div class="nt-footer-qr-image">
                <img
                  :src="getImageUrl(support.wecomQr)"
                  :alt="t('needtoken.support.wecom')"
                  width="88"
                  height="88"
                  loading="lazy"
                  decoding="async"
                  referrerpolicy="no-referrer"
                  @error="wecomQrFailed = true"
                />
              </div>
              <figcaption>{{ t('needtoken.support.wecom') }}</figcaption>
            </figure>

            <div class="nt-footer-contact-links">
              <a
                v-if="support.telegramUrl"
                :href="support.telegramUrl"
                target="_blank"
                rel="noopener noreferrer"
                class="nt-footer-tg-btn"
              >
                <NavAnimatedIcon name="telegram" />
                <span>{{ t('needtoken.support.telegramContact') }}</span>
                <ArrowUpRight :size="14" aria-hidden="true" />
              </a>
              <div v-if="support.businessEmail" class="nt-footer-business">
                <span>{{ t('needtoken.support.business') }}</span>
                <a :href="emailHref">
                  <Mail :size="14" aria-hidden="true" />
                  <span>{{ support.businessEmail }}</span>
                  <ArrowUpRight :size="12" aria-hidden="true" />
                </a>
              </div>
              <p v-if="support.serviceHours" class="nt-footer-hours">
                <Clock3 :size="12" aria-hidden="true" />
                <span>{{ support.serviceHours }}</span>
              </p>
            </div>
          </div>
        </aside>
      </div>

      <div class="nt-footer-legal">
        <span>© {{ year }} {{ brandName }}</span>
        <div class="nt-footer-legal-actions">
          <ProviderStatusStrip />
          <div class="nt-footer-legal-links">
            <RouterLink to="/privacy">{{ t('footer.privacy') }}</RouterLink>
            <RouterLink to="/terms">{{ t('footer.terms') }}</RouterLink>
          </div>
        </div>
      </div>
      <FooterShapeWordmark :word="brandName" />
    </div>
  </footer>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { ArrowUpRight, Clock3, Mail } from 'lucide-vue-next'
import { useAppStore } from '../../../stores/app'
import { getImageUrl } from '../../../utils/image'
import { API_SERVICES_PATH, CONSOLE_URL } from '../config'
import { supportEmailHref, type NeedtokenSupport } from '../lib/supportConfig'
import BrandMark from './BrandMark.vue'
import FooterShapeWordmark from './FooterShapeWordmark.vue'
import NavAnimatedIcon from './NavAnimatedIcon.vue'
import ProviderStatusStrip from './ProviderStatusStrip.vue'

const props = defineProps<{ brandName: string; resellerLogo?: string; support: NeedtokenSupport }>()
const { t } = useI18n()
const appStore = useAppStore()
const year = new Date().getFullYear()
const wecomQrFailed = ref(false)
watch(() => props.support.wecomQr, () => { wecomQrFailed.value = false })

const emailHref = computed(() => supportEmailHref(props.support.businessEmail))
const showWecomQr = computed(() => Boolean(props.support.wecomQr) && !wecomQrFailed.value)
const hasContactPanel = computed(() => Boolean(showWecomQr.value || props.support.telegramUrl || props.support.businessEmail))

const columns = computed(() => [
  { key: 'products', title: t('needtoken.footer.products'), links: [
    { path: '/categories/recharge', label: t('needtoken.nav.recharge') },
    { path: '/categories/rental', label: t('needtoken.nav.rental') },
    { path: API_SERVICES_PATH, label: t('needtoken.nav.gateway') },
    { path: '/models', label: t('needtoken.nav.models') },
  ] },
  { key: 'help', title: t('needtoken.nav.help'), links: [
    { path: '/blog', label: t('needtoken.nav.blog') },
    { path: '/docs', label: t('needtoken.nav.docs') },
    { path: '/faq', label: t('needtoken.nav.faq') },
  ] },
  { key: 'support', title: t('needtoken.footer.support'), links: [
    { path: '/tools/radar', label: t('needtoken.nav.radar') },
    { path: '/tools/codex-account', label: t('needtoken.checkCenter.tabs.gpt') },
    { path: '/tools/environment', label: t('needtoken.checkCenter.tabs.claude') },
    { path: '/free-apple-id', label: t('needtoken.nav.appleIds') },
    { path: '/guest/orders', label: t('navbar.guestOrders') },
  ] },
])
</script>

<style scoped>
.nt-footer-next { width: 100%; margin-top: clamp(64px, 7vw, 112px); padding: 0 24px 24px; color: var(--ui-text-muted); }
.nt-footer-card { max-width: var(--nt-content-max-width, 1200px); margin: 0 auto; overflow: visible; isolation: isolate; border: 1px solid var(--nt-card-border); border-radius: 22px; background: var(--nt-surface-footer); box-shadow: var(--nt-card-shadow); }
.nt-footer-top { display: grid; grid-template-columns: minmax(0, 0.95fr) minmax(0, 1.7fr); gap: 40px; padding: 40px 40px 32px; }
.nt-footer-top.has-contact { grid-template-columns: minmax(190px, 0.9fr) minmax(0, 1.55fr) minmax(240px, 1.05fr); }
.nt-footer-brand { display: inline-flex; align-items: center; gap: 11px; max-width: 100%; color: var(--ui-text-primary); font-size: 20px; font-weight: 700; letter-spacing: -.04em; }
.nt-footer-brand > img { flex-shrink: 0; border-radius: 9px; object-fit: contain; }
.nt-footer-brand > span { min-width: 0; overflow-wrap: anywhere; }
.nt-footer-description { max-width: 28ch; margin-top: 14px; font-size: 13px; line-height: 1.85; }
.nt-footer-nav { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 20px; padding-top: 4px; }
.nt-footer-column { display: flex; flex-direction: column; align-items: flex-start; gap: 12px; min-width: 0; font-size: 13px; line-height: 1.6; }
.nt-footer-column h2, .nt-footer-contact-panel h2 { margin-bottom: 4px; color: var(--ui-text-primary); font-size: 13px; font-weight: 600; }
.nt-footer-column a { display: inline-flex; align-items: center; gap: 5px; max-width: 100%; overflow-wrap: anywhere; }
.nt-footer-card a { transition: color 180ms ease; }
.nt-footer-card a:hover { color: var(--ui-accent); }

.nt-footer-contact-panel { display: flex; flex-direction: column; gap: 14px; min-width: 0; padding-left: 24px; border-left: 1px solid var(--ui-border); padding-top: 4px; }
.nt-footer-contact-body { display: flex; align-items: flex-start; gap: 16px; }
.nt-footer-wecom { flex: none; text-align: center; }
.nt-footer-qr-image { width: 88px; padding: 6px; border: 1px solid var(--ui-border); border-radius: 12px; background: #fff; box-shadow: var(--nt-card-shadow); }
.nt-footer-qr-image img { display: block; width: 100%; height: auto; aspect-ratio: 1; object-fit: contain; }
.nt-footer-wecom figcaption { margin-top: 6px; color: var(--ui-text-secondary); font-size: 11px; }
.nt-footer-contact-links { display: flex; flex-direction: column; gap: 12px; min-width: 0; flex: 1; }
.nt-footer-tg-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 7px;
  width: fit-content;
  max-width: 100%;
  min-height: 36px;
  padding: 7px 14px;
  border: 1px solid var(--nt-accent-border);
  border-radius: 999px;
  background: var(--ui-accent-soft);
  color: var(--ui-accent);
  font-size: 12px;
  font-weight: 600;
  transition: background-color 160ms var(--nt-motion-ease), border-color 160ms var(--nt-motion-ease), color 160ms var(--nt-motion-ease), transform var(--nt-duration-rebound, 480ms) var(--nt-ease-spring);
}
.nt-footer-tg-btn:hover {
  background: var(--ui-accent);
  border-color: var(--ui-accent);
  color: var(--ui-text-on-accent) !important;
  transform: translateY(-1px);
}
.nt-footer-tg-btn:active { transform: scale(0.96); transition-duration: var(--nt-duration-press, 80ms); }
.nt-footer-business { font-size: 11px; }
.nt-footer-business > span { display: block; color: var(--ui-text-muted); }
.nt-footer-business a { display: inline-flex; align-items: center; gap: 6px; max-width: 100%; margin-top: 4px; color: var(--ui-text-secondary); font-size: 12px; }
.nt-footer-business a > span { overflow-wrap: anywhere; }
.nt-footer-business a > svg { flex-shrink: 0; color: var(--ui-accent); }
.nt-footer-hours { display: flex; align-items: center; gap: 6px; color: var(--ui-text-muted); font-size: 11px; line-height: 1.5; }
.nt-footer-hours > svg { flex-shrink: 0; }

.nt-footer-legal { display: flex; flex-wrap: wrap; justify-content: space-between; align-items: center; gap: 12px; padding: 20px 40px 24px; border-top: 1px solid var(--ui-border); font-size: 11px; line-height: 1.7; }
.nt-footer-legal-actions { display: flex; flex-wrap: wrap; align-items: center; justify-content: flex-end; gap: 12px 18px; margin-left: auto; }
.nt-footer-legal-links { display: flex; flex-wrap: wrap; gap: 16px; }
.nt-footer-legal + :deep(.nt-footer-wordmark) { margin-top: 12px; overflow: hidden; border-radius: 0 0 22px 22px; }
.nt-footer-card a:focus-visible { outline: 2px solid var(--ui-accent); outline-offset: 4px; border-radius: 4px; }

@media (max-width: 1080px) {
  .nt-footer-top, .nt-footer-top.has-contact { grid-template-columns: 1fr; gap: 28px; padding: 32px 28px 28px; }
  .nt-footer-description { max-width: 48ch; }
  .nt-footer-contact-panel { padding-left: 0; padding-top: 24px; border-left: 0; border-top: 1px solid var(--ui-border); }
  .nt-footer-legal { padding-inline: 28px; }
}
@media (max-width: 640px) {
  .nt-footer-next { padding: 0 14px 14px; }
  .nt-footer-top, .nt-footer-top.has-contact { padding: 26px 20px 24px; }
  .nt-footer-nav { gap: 14px; }
  .nt-footer-column { font-size: 12px; gap: 10px; }
  .nt-footer-brand { font-size: 19px; }
  .nt-footer-legal { flex-direction: column-reverse; align-items: flex-start; padding: 18px 20px 20px; gap: 18px; font-size: 10px; }
  .nt-footer-legal-actions { width: 100%; margin-left: 0; justify-content: flex-start; gap: 16px; }
}
@media (prefers-reduced-motion: reduce) {
  .nt-footer-card a, .nt-footer-tg-btn { transition: none; }
  .nt-footer-tg-btn:hover, .nt-footer-tg-btn:active { transform: none; }
}
</style>
