<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import type { AdminPaymentChannel } from '@/api/types'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { Switch } from '@/components/ui/switch'
import { Label } from '@/components/ui/label'
import { notifyError, notifySuccess } from '@/utils/notify'
import ComplianceGuardWrapper from '@/components/ComplianceGuardWrapper.vue'

const { t } = useI18n()

interface RechargeBonusTierForm {
  min_amount: number
  bonus_type: 'fixed' | 'percent'
  bonus_value: number
}

const form = reactive({
  recharge_channel_ids: [] as number[],
  wallet_only_payment: false,
  recharge_bonus: {
    enabled: false,
    first_recharge_bonus_type: 'fixed' as 'fixed' | 'percent',
    first_recharge_bonus_value: 0,
    first_recharge_min_amount: 0,
    max_bonus_per_order: 0,
    cost_rate_percent: 100,
    tiers: [] as RechargeBonusTierForm[],
  },
})
const channels = ref<AdminPaymentChannel[]>([])
const saving = ref(false)

const applyDefaultBonus = () => {
  form.recharge_bonus = {
    enabled: false,
    first_recharge_bonus_type: 'fixed',
    first_recharge_bonus_value: 0,
    first_recharge_min_amount: 0,
    max_bonus_per_order: 0,
    cost_rate_percent: 100,
    tiers: [],
  }
}

const loadConfig = async () => {
  try {
    const res = await adminAPI.getSettings({ key: 'wallet_config' })
    const data = res.data?.data
    if (data && Array.isArray(data.recharge_channel_ids)) {
      form.recharge_channel_ids = data.recharge_channel_ids.filter((id: unknown) => typeof id === 'number' && id > 0)
    } else {
      form.recharge_channel_ids = []
    }
    form.wallet_only_payment = !!data?.wallet_only_payment
    const rb = data?.recharge_bonus
    if (rb && typeof rb === 'object') {
      form.recharge_bonus = {
        enabled: !!rb.enabled,
        first_recharge_bonus_type: rb.first_recharge_bonus_type === 'percent' ? 'percent' : 'fixed',
        first_recharge_bonus_value: Number(rb.first_recharge_bonus_value) || 0,
        first_recharge_min_amount: Number(rb.first_recharge_min_amount) || 0,
        max_bonus_per_order: Number(rb.max_bonus_per_order) || 0,
        cost_rate_percent: rb.cost_rate_percent !== undefined ? Number(rb.cost_rate_percent) : 100,
        tiers: Array.isArray(rb.tiers)
          ? rb.tiers.map((tier: any) => ({
              min_amount: Number(tier?.min_amount) || 0,
              bonus_type: tier?.bonus_type === 'percent' ? 'percent' : 'fixed',
              bonus_value: Number(tier?.bonus_value) || 0,
            }))
          : [],
      }
    } else {
      applyDefaultBonus()
    }
  } catch {
    form.recharge_channel_ids = []
    form.wallet_only_payment = false
    applyDefaultBonus()
  }
}

const addBonusTier = () => {
  form.recharge_bonus.tiers.push({
    min_amount: 100,
    bonus_type: 'fixed',
    bonus_value: 10,
  })
}

const removeBonusTier = (idx: number) => {
  form.recharge_bonus.tiers.splice(idx, 1)
}

const loadChannels = async () => {
  try {
    const res = await adminAPI.getPaymentChannels({ page: 1, page_size: 200 })
    channels.value = (res.data?.data ?? []).filter((ch: AdminPaymentChannel) => ch.is_active)
  } catch {
    channels.value = []
  }
}

const toggleChannel = (channelId: number) => {
  const idx = form.recharge_channel_ids.indexOf(channelId)
  if (idx >= 0) {
    form.recharge_channel_ids.splice(idx, 1)
  } else {
    form.recharge_channel_ids.push(channelId)
  }
}

const save = async () => {
  saving.value = true
  try {
    await adminAPI.updateSettings({
      key: 'wallet_config',
      value: {
        recharge_channel_ids: form.recharge_channel_ids,
        wallet_only_payment: form.wallet_only_payment,
        recharge_bonus: {
          enabled: form.recharge_bonus.enabled,
          first_recharge_bonus_type: form.recharge_bonus.first_recharge_bonus_type,
          first_recharge_bonus_value: Number(form.recharge_bonus.first_recharge_bonus_value) || 0,
          first_recharge_min_amount: Number(form.recharge_bonus.first_recharge_min_amount) || 0,
          max_bonus_per_order: Number(form.recharge_bonus.max_bonus_per_order) || 0,
          cost_rate_percent: Number(form.recharge_bonus.cost_rate_percent) || 0,
          tiers: form.recharge_bonus.tiers.map((tier) => ({
            min_amount: Number(tier.min_amount) || 0,
            bonus_type: tier.bonus_type,
            bonus_value: Number(tier.bonus_value) || 0,
          })),
        },
      },
    } as any)
    notifySuccess(t('admin.settings.saved'))
  } catch (err: any) {
    notifyError(err?.message || t('admin.settings.saveFailed'))
  } finally {
    saving.value = false
  }
}

onMounted(() => {
  loadConfig()
  loadChannels()
})
</script>

<template>
  <ComplianceGuardWrapper>
  <div class="space-y-6">
    <div class="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
      <div>
        <h1 class="text-2xl font-semibold">{{ t('admin.settings.wallet.title') }}</h1>
        <p class="mt-1 text-sm text-muted-foreground">{{ t('admin.settings.wallet.subtitle') }}</p>
      </div>
    </div>

    <div class="rounded-xl border border-border bg-card">
      <div class="space-y-4 p-6">
        <div class="flex items-center justify-between">
          <div>
            <Label for="wallet-only-payment" class="text-sm font-medium">{{ t('admin.settings.wallet.walletOnlyPayment') }}</Label>
            <p class="text-xs text-muted-foreground mt-0.5">{{ t('admin.settings.wallet.walletOnlyPaymentTip') }}</p>
          </div>
          <Switch id="wallet-only-payment" v-model="form.wallet_only_payment" />
        </div>
        <div class="border-t border-border pt-4">
          <Label class="block text-xs font-medium text-muted-foreground mb-2">{{ t('admin.settings.wallet.rechargeChannels') }}</Label>
          <div v-if="channels.length > 0" class="flex flex-wrap gap-2">
            <Label v-for="ch in channels" :key="ch.id" class="inline-flex items-center gap-1.5 rounded-lg border border-border px-3 py-1.5 text-xs cursor-pointer select-none" :class="form.recharge_channel_ids.includes(ch.id) ? 'bg-primary/10 border-primary text-primary' : 'text-muted-foreground hover:border-primary/40'">
              <Checkbox :model-value="form.recharge_channel_ids.includes(ch.id)" @update:model-value="() => toggleChannel(ch.id)" />
              {{ ch.name }}
            </Label>
          </div>
          <p v-else class="text-xs text-muted-foreground">{{ t('admin.settings.wallet.noChannels') }}</p>
          <p class="mt-1 text-xs text-muted-foreground">{{ t('admin.settings.wallet.rechargeChannelsTip') }}</p>
        </div>

        <div class="space-y-4 border-t border-border pt-4">
          <div class="flex items-center justify-between">
            <div>
              <Label for="recharge-bonus-enabled" class="text-sm font-medium">{{ t('admin.settings.wallet.bonusEnabled') }}</Label>
              <p class="mt-0.5 text-xs text-muted-foreground">{{ t('admin.settings.wallet.bonusEnabledTip') }}</p>
            </div>
            <Switch id="recharge-bonus-enabled" v-model="form.recharge_bonus.enabled" />
          </div>

          <div class="grid gap-4 md:grid-cols-4">
            <div>
              <Label class="mb-1.5 block text-xs text-muted-foreground">{{ t('admin.settings.wallet.firstBonusType') }}</Label>
              <select v-model="form.recharge_bonus.first_recharge_bonus_type" class="h-9 w-full rounded-md border border-input bg-background px-2.5 text-sm">
                <option value="fixed">{{ t('admin.settings.wallet.bonusTypeFixed') }}</option>
                <option value="percent">{{ t('admin.settings.wallet.bonusTypePercent') }}</option>
              </select>
            </div>
            <div>
              <Label class="mb-1.5 block text-xs text-muted-foreground">{{ t('admin.settings.wallet.firstBonusValue') }}</Label>
              <input v-model.number="form.recharge_bonus.first_recharge_bonus_value" type="number" step="0.01" min="0" class="h-9 w-full rounded-md border border-input bg-background px-3 text-sm font-mono" />
            </div>
            <div>
              <Label class="mb-1.5 block text-xs text-muted-foreground">{{ t('admin.settings.wallet.firstBonusMin') }}</Label>
              <input v-model.number="form.recharge_bonus.first_recharge_min_amount" type="number" step="0.01" min="0" class="h-9 w-full rounded-md border border-input bg-background px-3 text-sm font-mono" />
            </div>
            <div>
              <Label class="mb-1.5 block text-xs text-muted-foreground">{{ t('admin.settings.wallet.maxBonusPerOrder') }}</Label>
              <input v-model.number="form.recharge_bonus.max_bonus_per_order" type="number" step="0.01" min="0" class="h-9 w-full rounded-md border border-input bg-background px-3 text-sm font-mono" />
            </div>
          </div>

          <div class="grid gap-4 md:grid-cols-2">
            <div>
              <Label class="mb-1.5 block text-xs text-muted-foreground">{{ t('admin.settings.wallet.bonusCostRate') }}</Label>
              <input v-model.number="form.recharge_bonus.cost_rate_percent" type="number" step="1" min="0" max="100" class="h-9 w-full rounded-md border border-input bg-background px-3 text-sm font-mono" />
              <p class="mt-1 text-xs text-muted-foreground">{{ t('admin.settings.wallet.bonusCostRateTip') }}</p>
            </div>
          </div>

          <div class="space-y-2">
            <div class="flex items-center justify-between">
              <Label class="text-xs font-medium text-muted-foreground">{{ t('admin.settings.wallet.bonusTiers') }}</Label>
              <Button type="button" variant="outline" size="sm" @click="addBonusTier">
                {{ t('admin.settings.wallet.addBonusTier') }}
              </Button>
            </div>
            <div v-if="form.recharge_bonus.tiers.length === 0" class="rounded-lg border border-dashed px-3 py-3 text-xs text-muted-foreground">
              {{ t('admin.settings.wallet.noBonusTiers') }}
            </div>
            <div v-else class="space-y-2">
              <div v-for="(tier, idx) in form.recharge_bonus.tiers" :key="idx" class="grid grid-cols-1 items-center gap-2 rounded-lg border p-2.5 sm:grid-cols-4">
                <input v-model.number="tier.min_amount" type="number" step="0.01" min="0" class="h-9 rounded-md border border-input bg-background px-3 text-sm font-mono" :placeholder="t('admin.settings.wallet.tierMinAmount')" />
                <select v-model="tier.bonus_type" class="h-9 rounded-md border border-input bg-background px-2.5 text-sm">
                  <option value="fixed">{{ t('admin.settings.wallet.bonusTypeFixed') }}</option>
                  <option value="percent">{{ t('admin.settings.wallet.bonusTypePercent') }}</option>
                </select>
                <input v-model.number="tier.bonus_value" type="number" step="0.01" min="0" class="h-9 rounded-md border border-input bg-background px-3 text-sm font-mono" :placeholder="t('admin.settings.wallet.tierBonusValue')" />
                <Button type="button" variant="outline" size="sm" @click="removeBonusTier(idx)">
                  {{ t('admin.settings.wallet.removeBonusTier') }}
                </Button>
              </div>
            </div>
          </div>
        </div>

        <div class="flex justify-end border-t border-border pt-4">
          <Button :disabled="saving" @click="save">
            {{ saving ? t('admin.settings.actions.saving') : t('admin.settings.actions.save') }}
          </Button>
        </div>
      </div>
    </div>
  </div>
  </ComplianceGuardWrapper>
</template>
