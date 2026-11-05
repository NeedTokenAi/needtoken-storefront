import { computed, ref, watch } from 'vue'
import { defineStore } from 'pinia'
import { useUserAuthStore } from './userAuth'
import { invoiceRequestError, normalizeInvoiceRequest, type InvoiceRequest, type InvoiceType } from '../utils/invoiceRequest'

/** A single order's billing draft. Tax/bank details are never put in browser storage. */
export const useInvoiceRequestStore = defineStore('invoice-request', () => {
  const auth = useUserAuthStore()
  const request = ref<InvoiceRequest | null>(null)
  const editorOpen = ref(false)
  const editorType = ref<InvoiceType>('ordinary')
  const ready = computed(() => !editorOpen.value && (!request.value || !invoiceRequestError(request.value)))
  function openEditor(type: InvoiceType = request.value?.type || 'ordinary') {
    editorType.value = type
    editorOpen.value = true
  }
  function save(value: InvoiceRequest) {
    if (invoiceRequestError(value)) return false
    request.value = normalizeInvoiceRequest(value)
    editorOpen.value = false
    return true
  }
  function clear() {
    request.value = null
    editorOpen.value = false
    editorType.value = 'ordinary'
  }
  watch(() => auth.isAuthenticated ? String(auth.user?.id || 'member') : 'guest', clear, { flush: 'sync' })
  return { request, editorOpen, editorType, ready, openEditor, save, clear }
})
