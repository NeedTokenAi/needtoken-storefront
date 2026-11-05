<template>
  <section v-if="order?.invoice_request" class="nt-invoice-notice" :aria-label="t('needtoken.invoice.title')">
    <ReceiptText :size="19" aria-hidden="true" />
    <div><h2>{{ t(completed ? 'needtoken.invoice.readyTitle' : 'needtoken.invoice.savedTitle') }}</h2><p>{{ t(`needtoken.invoice.${order.invoice_request.type}`) }} · {{ order.invoice_request.title }}</p><p>{{ t(completed ? 'needtoken.invoice.collectHint' : 'needtoken.invoice.pendingHint') }}</p><div v-if="completed" class="nt-invoice-contact"><a v-if="support.contactUrl" :href="support.contactUrl" target="_blank" rel="noopener noreferrer">{{ t('needtoken.support.openContact') }}</a><a v-if="mailHref" :href="mailHref">{{ support.businessEmail }}</a></div></div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { ReceiptText } from 'lucide-vue-next'
import { useAppStore } from '../../../stores/app'
import { normalizeNeedtokenSupport } from '../lib/supportConfig'
import type { InvoiceRequest } from '../../../utils/invoiceRequest'
const props = defineProps<{ order: { order_no?: string; status?: string; invoice_request?: InvoiceRequest | null } | null }>()
const { t } = useI18n(), app = useAppStore()
const completed = computed(() => ['delivered', 'completed'].includes(props.order?.status || ''))
const support = computed(() => normalizeNeedtokenSupport(app.isResellerTenant ? undefined : app.config?.needtoken_support, !app.isResellerTenant))
const mailHref = computed(() => support.value.businessEmail ? `mailto:${encodeURIComponent(support.value.businessEmail)}?subject=${encodeURIComponent(t('needtoken.invoice.mailSubject', { order: props.order?.order_no || '' }))}` : '')
</script>

<style scoped>
.nt-invoice-notice { display: flex; gap: 12px; align-items: flex-start; padding: 19px; border: 1px solid #2dd4a744; background: #12352b66; border-radius: 14px; color: #dce9e4; }.nt-invoice-notice > svg { flex: none; color: #2dd4a7; margin-top: 2px; }.nt-invoice-notice > div { min-width: 0; }.nt-invoice-notice h2 { font-size: 14px; font-weight: 650; }.nt-invoice-notice p { margin-top: 7px; font-size: 12px; line-height: 1.8; color: #b2c3bd; overflow-wrap: anywhere; }.nt-invoice-contact { display: flex; flex-wrap: wrap; gap: 12px; margin-top: 11px; font-size: 12px; }.nt-invoice-contact a { color: #53dfbd; overflow-wrap: anywhere; }
</style>
