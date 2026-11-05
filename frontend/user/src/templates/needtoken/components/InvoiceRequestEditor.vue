<template>
  <section class="nt-invoice-editor" :aria-label="t('needtoken.invoice.title')">
    <div class="nt-invoice-choice">
      <label><input type="checkbox" :disabled="props.disabled" :checked="!!store.request || store.editorOpen" @change="toggleRequest" /><ReceiptText :size="15" aria-hidden="true" />{{ t('needtoken.invoice.request') }}</label>
      <button v-if="store.request" :disabled="props.disabled" type="button" @click="store.openEditor()">{{ t('needtoken.invoice.edit') }}</button>
    </div>
    <p v-if="store.request" class="nt-invoice-saved" :title="store.request.title">{{ t(`needtoken.invoice.${store.request.type}`) }} · {{ store.request.title }}</p>
    <p class="nt-invoice-hint">{{ t('needtoken.invoice.hint') }}</p>
    <DialogRoot :open="store.editorOpen" @update:open="store.editorOpen = $event">
      <DialogPortal>
        <DialogOverlay class="nt-invoice-overlay" />
        <DialogContent class="nt-invoice-dialog" @close-auto-focus="restoreFocus">
          <div class="nt-invoice-dialog-head"><DialogTitle>{{ t('needtoken.invoice.fillTitle') }}</DialogTitle><DialogClose :aria-label="t('needtoken.invoice.close')"><X :size="18" /></DialogClose></div>
          <DialogDescription class="nt-invoice-description">{{ t('needtoken.invoice.formHint') }}</DialogDescription>
          <form @submit.prevent="save">
            <fieldset class="nt-invoice-types"><legend class="sr-only">{{ t('needtoken.invoice.type') }}</legend><label v-for="type in types" :key="type"><input v-model="form.type" type="radio" name="nt-invoice-type" :value="type" @change="setType(type)" />{{ t(`needtoken.invoice.${type}`) }}</label></fieldset>
            <fieldset v-if="form.type === 'ordinary'" class="nt-invoice-types"><legend class="sr-only">{{ t('needtoken.invoice.titleType') }}</legend><label v-for="type in titleTypes" :key="type"><input v-model="form.title_type" type="radio" name="nt-invoice-title-type" :value="type" />{{ t(`needtoken.invoice.${type}`) }}</label></fieldset>
            <div class="nt-invoice-fields">
              <label>{{ t(form.title_type === 'company' ? 'needtoken.invoice.companyTitle' : 'needtoken.invoice.personalTitle') }}<input v-model="form.title" required maxlength="120" autocomplete="organization" /></label>
              <label v-if="form.title_type === 'company'">{{ t('needtoken.invoice.taxId') }}<input v-model="form.tax_id" required maxlength="20" autocapitalize="characters" autocomplete="off" /></label>
              <label>{{ t('needtoken.invoice.email') }}<input v-model="form.email" required type="email" maxlength="254" autocomplete="email" /></label>
            </div>
            <details v-if="form.type === 'special_vat'" class="nt-invoice-extra"><summary>{{ t('needtoken.invoice.extra') }}</summary><div class="nt-invoice-fields"><label v-for="field in extraFields" :key="field">{{ t(`needtoken.invoice.${field}`) }}<input v-model="form[field]" :maxlength="extraLimits[field]" autocomplete="off" /></label></div></details>
            <p v-if="error" class="nt-invoice-error" role="alert">{{ t(`needtoken.invoice.errors.${error}`) }}</p>
            <div class="nt-invoice-actions"><DialogClose>{{ t('common.cancel') }}</DialogClose><button type="submit">{{ t('needtoken.invoice.save') }}</button></div>
          </form>
        </DialogContent>
      </DialogPortal>
    </DialogRoot>
  </section>
</template>

<script setup lang="ts">
import { reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { DialogRoot, DialogPortal, DialogOverlay, DialogContent, DialogTitle, DialogDescription, DialogClose } from 'reka-ui'
import { ReceiptText, X } from 'lucide-vue-next'
import { useInvoiceRequestStore } from '../../../stores/invoiceRequest'
import { useUserAuthStore } from '../../../stores/userAuth'
import { invoiceContactEmail, invoiceRequestError, type InvoiceRequest, type InvoiceType } from '../../../utils/invoiceRequest'
const props = defineProps<{ email?: string; disabled?: boolean }>()
const { t } = useI18n()
const store = useInvoiceRequestStore(), auth = useUserAuthStore()
const types = ['ordinary', 'special_vat'] as const
const titleTypes = ['personal', 'company'] as const
const extraFields = ['company_address', 'company_phone', 'bank_name', 'bank_account'] as const
const extraLimits = { company_address: 300, company_phone: 32, bank_name: 120, bank_account: 40 }
const error = ref('')
const form = reactive<InvoiceRequest>({ type: 'ordinary', title_type: 'personal', title: '', email: '' })
let previousFocus: HTMLElement | null = null
watch(() => store.editorOpen, open => {
  if (!open) {
    Object.assign(form, { type: 'ordinary', title_type: 'personal', title: '', email: '', tax_id: '', company_address: '', company_phone: '', bank_name: '', bank_account: '' })
    error.value = ''
    return
  }
  previousFocus = document.activeElement instanceof HTMLElement ? document.activeElement : null
  Object.assign(form, { type: store.editorType, title_type: store.editorType === 'special_vat' ? 'company' : 'personal', title: '', email: invoiceContactEmail(props.email) || invoiceContactEmail(auth.user?.email), tax_id: '', company_address: '', company_phone: '', bank_name: '', bank_account: '' }, store.request || {})
  setType(store.editorType)
  error.value = ''
}, { immediate: true })
function setType(type: InvoiceType) { form.type = type; if (type === 'special_vat') form.title_type = 'company' }
function toggleRequest(event: Event) { if ((event.target as HTMLInputElement).checked) store.openEditor(); else store.clear() }
function save() { error.value = invoiceRequestError(form) || ''; if (!error.value) store.save(form) }
function restoreFocus(event: Event) { if (previousFocus?.isConnected) { event.preventDefault(); previousFocus.focus() } }
</script>

<style>
.nt-invoice-editor { border-top: 1px solid var(--ui-border, #29313a); margin-top: 12px; padding-top: 12px; font-size: 12px; }
.nt-invoice-choice { display: flex; align-items: center; justify-content: space-between; gap: 8px; }.nt-invoice-choice label { display: flex; align-items: center; gap: 7px; cursor: pointer; }.nt-invoice-choice input, .nt-invoice-types input { accent-color: #2dd4a7; width: 16px; height: 16px; }.nt-invoice-choice button { color: #2dd4a7; padding: 4px; }.nt-invoice-saved { margin-top: 7px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; color: #d9e5e1; }.nt-invoice-hint { margin-top: 6px; color: #96a3ae; font-size: 11px; line-height: 1.65; }
.nt-invoice-overlay { position: fixed; inset: 0; z-index: 120; background: #03060ac9; }.nt-invoice-dialog { position: fixed; z-index: 121; left: 50%; top: 50%; transform: translate(-50%, -50%); width: min(480px, calc(100vw - 32px)); max-height: min(90dvh, 760px); overflow-y: auto; padding: 23px; border: 1px solid #ffffff26; border-radius: 20px; color: #e7ecef; background: linear-gradient(140deg, #17221f, #0f151d 65%); box-shadow: 0 25px 90px #0009; font: inherit; }.nt-invoice-dialog-head { display: flex; align-items: center; justify-content: space-between; gap: 16px; font-size: 18px; font-weight: 650; }.nt-invoice-dialog-head button { display: grid; place-items: center; min-width: 36px; min-height: 36px; border-radius: 9px; background: #ffffff0b; }.nt-invoice-description { margin: 8px 0 16px; color: #a5b1bc; font-size: 12px; line-height: 1.75; }.nt-invoice-types { display: flex; flex-wrap: wrap; gap: 13px; padding: 10px 0; font-size: 13px; }.nt-invoice-types label { display: inline-flex; align-items: center; gap: 7px; cursor: pointer; }.nt-invoice-fields { display: grid; gap: 13px; margin-top: 12px; }.nt-invoice-fields label { display: grid; gap: 7px; font-size: 12px; color: #cbd4dd; }.nt-invoice-fields input { width: 100%; min-width: 0; height: 42px; padding: 9px 11px; border: 1px solid #ffffff26; border-radius: 9px; background: #080d12; color: #e7ecef; font-size: 14px; }.nt-invoice-extra { margin-top: 17px; font-size: 12px; color: #a5b1bc; }.nt-invoice-extra summary { cursor: pointer; padding: 5px 0; }.nt-invoice-error { color: #fec97c; margin-top: 14px; font-size: 12px; }.nt-invoice-actions { display: flex; justify-content: flex-end; gap: 10px; margin-top: 21px; }.nt-invoice-actions button { min-height: 40px; border: 1px solid #ffffff26; border-radius: 9px; padding: 9px 15px; font-size: 13px; }.nt-invoice-actions button[type=submit] { color: #061510; background: #2dd4a7; border-color: #2dd4a7; font-weight: 650; }.nt-invoice-dialog :is(button, input, summary):focus-visible { outline: 2px solid #2dd4a7; outline-offset: 3px; }
@media(pointer:coarse) { .nt-invoice-choice label, .nt-invoice-dialog-head button, .nt-invoice-types label, .nt-invoice-actions button { min-height: 44px; } }
</style>
