<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { userOrderAPI, guestOrderAPI } from '../api/order'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Textarea } from '@/components/ui/textarea'
import { Alert, AlertDescription } from '@/components/ui/alert'

export interface OrderAfterSalesSupplement {
  author: string
  message: string
  created_at: string
}

export interface OrderAfterSalesTicket {
  id: string
  order_no: string
  category: string
  priority: string
  status: string
  summary: string
  resolution_note?: string
  supplements?: OrderAfterSalesSupplement[]
  created_at: string
  updated_at: string
  resolved_at?: string
}

const props = defineProps<{
  order: { order_no: string; status?: string }
  guestAuth?: { email: string; order_password: string }
}>()

const { t } = useI18n()

const loading = ref(false)
const submitting = ref(false)
const supplementingTicketId = ref('')
const showForm = ref(false)
const unavailable = ref(false)
const errorMessage = ref('')
const successMessage = ref('')
const tickets = ref<OrderAfterSalesTicket[]>([])
const category = ref('fulfillment_failed')
const summary = ref('')
const supplementDrafts = ref<Record<string, string>>({})
const createOperationKey = ref('')

const categories = [
  'fulfillment_failed',
  'amount_mismatch',
  'refund_dispute',
  'customer_inquiry',
  'other',
]

const isGuest = computed(() => Boolean(props.guestAuth?.email && props.guestAuth?.order_password))
const canOpenTicket = computed(() => {
  const status = String(props.order?.status || '').toLowerCase()
  return status !== 'pending_payment'
})

const categoryLabel = (value: string) => {
  const key = `orderAfterSales.categories.${value}`
  const translated = t(key)
  return translated === key ? value : translated
}

const statusLabel = (value: string) => {
  const key = `orderAfterSales.statuses.${value}`
  const translated = t(key)
  return translated === key ? value : translated
}

const statusVariant = (value: string): 'default' | 'secondary' | 'destructive' | 'outline' | 'success' | 'warning' | 'neutral' => {
  switch (value) {
    case 'resolved':
      return 'success'
    case 'in_progress':
      return 'warning'
    case 'closed':
      return 'neutral'
    default:
      return 'default'
  }
}

const loadTickets = async () => {
  if (!props.order?.order_no) return
  loading.value = true
  unavailable.value = false
  try {
    const res = isGuest.value && props.guestAuth
      ? await guestOrderAPI.afterSalesList(props.order.order_no, props.guestAuth)
      : await userOrderAPI.afterSalesList(props.order.order_no)
    const data = res?.data?.data
    tickets.value = Array.isArray(data?.items) ? data.items : []
  } catch (err: any) {
    const bizStatus = Number(err?.response?.data?.status_code || 0)
    const httpStatus = Number(err?.response?.status || 0)
    if (bizStatus >= 500 || httpStatus >= 500) {
      unavailable.value = true
    }
    tickets.value = []
  } finally {
    loading.value = false
  }
}

watch(
  () => [props.order?.order_no, props.guestAuth?.email, props.guestAuth?.order_password],
  () => {
    errorMessage.value = ''
    successMessage.value = ''
    showForm.value = false
    summary.value = ''
    createOperationKey.value = ''
    void loadTickets()
  },
  { immediate: true },
)

const ensureOperationKey = () => {
  if (!createOperationKey.value) {
    createOperationKey.value = typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function'
      ? `as_${crypto.randomUUID().replace(/-/g, '')}`
      : `as_${Date.now()}_${Math.random().toString(36).slice(2, 10)}`
  }
  return createOperationKey.value
}

const submitTicket = async () => {
  const text = summary.value.trim()
  if (!text || submitting.value) return
  submitting.value = true
  errorMessage.value = ''
  successMessage.value = ''
  try {
    const payload = {
      category: category.value,
      summary: text,
      idempotency_key: ensureOperationKey(),
    }
    const res = isGuest.value && props.guestAuth
      ? await guestOrderAPI.afterSalesCreate(props.order.order_no, props.guestAuth, payload)
      : await userOrderAPI.afterSalesCreate(props.order.order_no, payload)
    const created = res?.data?.data as OrderAfterSalesTicket | undefined
    if (created?.id) {
      summary.value = ''
      createOperationKey.value = ''
      showForm.value = false
      successMessage.value = t('orderAfterSales.createSuccess')
      await loadTickets()
    }
  } catch (err: any) {
    errorMessage.value = err?.response?.data?.msg || err?.message || t('orderAfterSales.submitFailed')
  } finally {
    submitting.value = false
  }
}

const submitSupplement = async (ticketId: string) => {
  const msg = String(supplementDrafts.value[ticketId] || '').trim()
  if (!msg || supplementingTicketId.value) return
  supplementingTicketId.value = ticketId
  errorMessage.value = ''
  successMessage.value = ''
  try {
    const res = isGuest.value && props.guestAuth
      ? await guestOrderAPI.afterSalesSupplement(props.order.order_no, props.guestAuth, ticketId, { message: msg })
      : await userOrderAPI.afterSalesSupplement(props.order.order_no, ticketId, { message: msg })
    if (res?.data?.data?.id) {
      supplementDrafts.value = { ...supplementDrafts.value, [ticketId]: '' }
      successMessage.value = t('orderAfterSales.supplementSuccess')
      await loadTickets()
    }
  } catch (err: any) {
    errorMessage.value = err?.response?.data?.msg || err?.message || t('orderAfterSales.submitFailed')
  } finally {
    supplementingTicketId.value = ''
  }
}
</script>

<template>
  <section class="theme-float rounded-[22px] border bg-card p-5 md:p-6">
    <div class="flex flex-wrap items-start justify-between gap-3">
      <div>
        <h2 class="text-[17px] font-bold text-foreground">{{ t('orderAfterSales.title') }}</h2>
        <p class="mt-1 text-xs text-muted-foreground">{{ t('orderAfterSales.subtitle') }}</p>
      </div>
      <div class="flex items-center gap-2">
        <Button type="button" variant="outline" size="sm" :disabled="loading" @click="loadTickets">
          {{ t('orderAfterSales.refresh') }}
        </Button>
        <Button
          v-if="canOpenTicket && !unavailable"
          type="button"
          size="sm"
          @click="showForm = !showForm"
        >
          {{ showForm ? t('orderAfterSales.cancelForm') : t('orderAfterSales.openForm') }}
        </Button>
      </div>
    </div>

    <Alert v-if="errorMessage" variant="destructive" class="mt-4">
      <AlertDescription>{{ errorMessage }}</AlertDescription>
    </Alert>
    <Alert v-if="successMessage" class="mt-4">
      <AlertDescription>{{ successMessage }}</AlertDescription>
    </Alert>

    <div v-if="showForm" class="mt-4 space-y-3 rounded-xl border bg-secondary/40 p-4">
      <div>
        <label class="mb-1.5 block text-xs font-medium text-muted-foreground">
          {{ t('orderAfterSales.categoryLabel') }}
        </label>
        <select
          v-model="category"
          class="h-10 w-full rounded-lg border bg-background px-3 text-sm text-foreground"
        >
          <option v-for="item in categories" :key="item" :value="item">
            {{ categoryLabel(item) }}
          </option>
        </select>
      </div>
      <div>
        <label class="mb-1.5 block text-xs font-medium text-muted-foreground">
          {{ t('orderAfterSales.summaryLabel') }}
        </label>
        <Textarea
          v-model="summary"
          rows="3"
          maxlength="2000"
          :placeholder="t('orderAfterSales.summaryPlaceholder')"
        />
      </div>
      <div class="flex justify-end">
        <Button type="button" size="sm" :disabled="submitting || !summary.trim()" @click="submitTicket">
          {{ submitting ? t('orderAfterSales.submitting') : t('orderAfterSales.submitButton') }}
        </Button>
      </div>
    </div>

    <div v-if="loading && tickets.length === 0" class="mt-4 text-xs text-muted-foreground">
      {{ t('orderAfterSales.loading') }}
    </div>
    <div v-else-if="unavailable" class="mt-4 text-xs text-muted-foreground">
      {{ t('orderAfterSales.unavailableHint') }}
    </div>
    <div v-else-if="tickets.length === 0 && !showForm" class="mt-4 text-xs text-muted-foreground">
      {{ t('orderAfterSales.empty') }}
    </div>

    <div v-else-if="tickets.length > 0" class="mt-4 space-y-3">
      <div
        v-for="ticket in tickets"
        :key="ticket.id"
        class="rounded-xl border bg-secondary/30 p-4 text-sm"
      >
        <div class="flex flex-wrap items-center justify-between gap-2">
          <div class="flex flex-wrap items-center gap-2">
            <Badge :variant="statusVariant(ticket.status)" size="sm">
              {{ statusLabel(ticket.status) }}
            </Badge>
            <span class="text-xs font-medium text-muted-foreground">
              {{ categoryLabel(ticket.category) }}
            </span>
            <span class="font-mono text-[11px] text-muted-foreground">#{{ ticket.id }}</span>
          </div>
          <span class="font-mono text-xs text-muted-foreground">{{ ticket.updated_at }}</span>
        </div>

        <p class="mt-2 whitespace-pre-wrap text-sm text-foreground">{{ ticket.summary }}</p>

        <div v-if="ticket.resolution_note" class="mt-3 rounded-lg border bg-card p-3 text-xs">
          <div class="font-semibold text-foreground">{{ t('orderAfterSales.resolutionLabel') }}</div>
          <p class="mt-1 whitespace-pre-wrap text-muted-foreground">{{ ticket.resolution_note }}</p>
        </div>

        <div v-if="ticket.supplements && ticket.supplements.length" class="mt-3 space-y-2">
          <div class="text-xs font-semibold text-muted-foreground">
            {{ t('orderAfterSales.supplementsTitle') }}
          </div>
          <div
            v-for="(sup, idx) in ticket.supplements"
            :key="idx"
            class="rounded-lg bg-background/80 px-3 py-2 text-xs text-muted-foreground"
          >
            <div class="flex items-center justify-between gap-2">
              <span class="font-medium text-foreground">{{ t('orderAfterSales.customerAuthor') }}</span>
              <span class="font-mono text-[11px]">{{ sup.created_at }}</span>
            </div>
            <p class="mt-1 whitespace-pre-wrap">{{ sup.message }}</p>
          </div>
        </div>

        <div v-if="ticket.status !== 'closed'" class="mt-3 flex flex-col gap-2 sm:flex-row sm:items-end">
          <Textarea
            v-model="supplementDrafts[ticket.id]"
            rows="2"
            maxlength="2000"
            class="flex-1"
            :placeholder="t('orderAfterSales.supplementPlaceholder')"
          />
          <Button
            type="button"
            variant="outline"
            size="sm"
            :disabled="supplementingTicketId === ticket.id || !(supplementDrafts[ticket.id] || '').trim()"
            @click="submitSupplement(ticket.id)"
          >
            {{ supplementingTicketId === ticket.id ? t('orderAfterSales.submitting') : t('orderAfterSales.supplementButton') }}
          </Button>
        </div>
      </div>
    </div>
  </section>
</template>
