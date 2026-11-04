<script setup lang="ts">
import { useId } from 'vue'
import { useI18n } from 'vue-i18n'
import type { AdminSKUDisplayMeta } from '@/api/types'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'

const props = defineProps<{ language: string; siteCurrency: string }>()
const model = defineModel<AdminSKUDisplayMeta>({ required: true })
const { t } = useI18n()
const id = useId()
</script>

<template>
  <details class="rounded-lg border border-border bg-muted/15">
    <summary class="cursor-pointer rounded-lg px-4 py-3 text-xs font-semibold text-foreground">{{ t('skuDisplay.title') }}</summary>
    <div class="space-y-4 border-t border-border p-4">
      <p class="text-xs leading-6 text-muted-foreground">{{ t('skuDisplay.hint', { currency: siteCurrency }) }}</p>
      <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
        <div>
          <label :for="`${id}-label`" class="mb-1.5 block text-xs font-medium text-muted-foreground">{{ t('skuDisplay.label', { language: props.language }) }}</label>
          <Input :id="`${id}-label`" v-model="model.label[props.language]" maxlength="80" :placeholder="t('skuDisplay.labelPlaceholder')" />
        </div>
        <div>
          <label :for="`${id}-title`" class="mb-1.5 block text-xs font-medium text-muted-foreground">{{ t('skuDisplay.displayTitle', { language: props.language }) }}</label>
          <Input :id="`${id}-title`" v-model="model.title[props.language]" maxlength="160" />
        </div>
        <div class="md:col-span-2">
          <label :for="`${id}-description`" class="mb-1.5 block text-xs font-medium text-muted-foreground">{{ t('skuDisplay.description', { language: props.language }) }}</label>
          <Textarea :id="`${id}-description`" v-model="model.description[props.language]" maxlength="1000" rows="2" />
        </div>
        <div class="md:col-span-2">
          <label :for="`${id}-features`" class="mb-1.5 block text-xs font-medium text-muted-foreground">{{ t('skuDisplay.features', { language: props.language }) }}</label>
          <Textarea :id="`${id}-features`" v-model="model.feature_lines[props.language]" maxlength="1500" rows="3" :placeholder="t('skuDisplay.featuresPlaceholder')" />
          <p class="mt-1.5 text-xs text-muted-foreground">{{ t('skuDisplay.featuresHint') }}</p>
        </div>
        <div>
          <label :for="`${id}-suffix`" class="mb-1.5 block text-xs font-medium text-muted-foreground">{{ t('skuDisplay.suffix', { language: props.language }) }}</label>
          <Input :id="`${id}-suffix`" v-model="model.price_suffix[props.language]" maxlength="32" :placeholder="t('skuDisplay.suffixPlaceholder')" />
        </div>
        <div>
          <label :for="`${id}-official`" class="mb-1.5 block text-xs font-medium text-muted-foreground">{{ t('skuDisplay.officialPrice') }}</label>
          <Input :id="`${id}-official`" v-model="model.official_price_usd" type="text" inputmode="decimal" maxlength="11" placeholder="USD" />
          <p class="mt-1.5 text-xs leading-6 text-muted-foreground">{{ t('skuDisplay.officialHint') }}</p>
        </div>
      </div>
    </div>
  </details>
</template>
