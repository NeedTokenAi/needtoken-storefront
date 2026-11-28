<template>
  <div class="px-4 py-4 sm:px-5">
    <div v-if="loadingTelegramBinding" class="flex items-center justify-between gap-3 py-1 text-sm text-muted-foreground">
      <div class="flex items-center gap-3">
        <span class="flex h-10 w-10 shrink-0 items-center justify-center rounded-full border bg-muted/40 text-sky-500">
          <svg class="h-5 w-5" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
            <path d="M9.78 18.65l.28-4.23 7.68-6.92c.34-.31-.07-.46-.52-.19L7.74 13.3 3.64 12c-.88-.25-.89-.86.2-1.3l15.97-6.16c.73-.33 1.43.18 1.15 1.3l-2.72 12.81c-.19.91-.74 1.13-1.5.71L12.6 16.3l-1.99 1.93c-.23.23-.42.42-.83.42z" />
          </svg>
        </span>
        <div>
          <p class="text-sm font-semibold text-foreground">{{ t('personalCenter.security.telegramTitle') }}</p>
          <p class="text-xs text-muted-foreground">{{ t('personalCenter.security.telegramLoading') }}</p>
        </div>
      </div>
    </div>

    <div v-else-if="telegramBound" class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
      <div class="flex min-w-0 items-center gap-3">
        <img
          v-if="avatarUrl"
          :src="avatarUrl"
          alt="Telegram Avatar"
          class="h-10 w-10 shrink-0 rounded-full border object-cover"
        />
        <span v-else class="flex h-10 w-10 shrink-0 items-center justify-center rounded-full border bg-sky-500/10 text-sky-500">
          <svg class="h-5 w-5" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
            <path d="M9.78 18.65l.28-4.23 7.68-6.92c.34-.31-.07-.46-.52-.19L7.74 13.3 3.64 12c-.88-.25-.89-.86.2-1.3l15.97-6.16c.73-.33 1.43.18 1.15 1.3l-2.72 12.81c-.19.91-.74 1.13-1.5.71L12.6 16.3l-1.99 1.93c-.23.23-.42.42-.83.42z" />
          </svg>
        </span>
        <div class="min-w-0 space-y-0.5">
          <div class="flex flex-wrap items-center gap-2">
            <span class="text-sm font-semibold text-foreground">{{ telegramDisplayName }}</span>
            <Badge variant="success" size="sm">{{ t('personalCenter.security.telegramBound') }}</Badge>
          </div>
          <p class="truncate text-xs text-muted-foreground">
            {{ t('personalCenter.security.telegramBindID', { id: providerUserId }) }} · {{ t('personalCenter.security.telegramBindTime', { time: formattedAuthAt }) }}
          </p>
        </div>
      </div>

      <div class="flex shrink-0 flex-col items-start gap-1.5 sm:items-end">
        <div class="flex flex-wrap items-center gap-2">
          <Button
            v-if="showTelegramMiniAppEntry"
            type="button"
            variant="outline"
            size="sm"
            @click="$emit('openMiniAppEntry')"
          >
            {{ t('personalCenter.security.telegramMiniAppEntryAction') }}
          </Button>
          <Button
            type="button"
            variant="outline"
            size="sm"
            :disabled="unbindingTelegram || !canUnbindTelegram"
            @click="$emit('unbind')"
          >
            {{ unbindingTelegram ? t('personalCenter.security.telegramUnbinding') : t('personalCenter.security.telegramUnbind') }}
          </Button>
        </div>
        <p v-if="!canUnbindTelegram" class="text-[11px] text-muted-foreground sm:text-right">
          {{ t('personalCenter.security.telegramUnbindDisabledTip') }}
        </p>
      </div>
    </div>

    <div v-else class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
      <div class="flex min-w-0 items-center gap-3">
        <span class="flex h-10 w-10 shrink-0 items-center justify-center rounded-full border bg-muted/40 text-sky-500">
          <svg class="h-5 w-5" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
            <path d="M9.78 18.65l.28-4.23 7.68-6.92c.34-.31-.07-.46-.52-.19L7.74 13.3 3.64 12c-.88-.25-.89-.86.2-1.3l15.97-6.16c.73-.33 1.43.18 1.15 1.3l-2.72 12.81c-.19.91-.74 1.13-1.5.71L12.6 16.3l-1.99 1.93c-.23.23-.42.42-.83.42z" />
          </svg>
        </span>
        <div class="min-w-0 space-y-0.5">
          <div class="flex flex-wrap items-center gap-2">
            <span class="text-sm font-semibold text-foreground">{{ t('personalCenter.security.telegramTitle') }}</span>
            <Badge variant="outline" size="sm">{{ t('personalCenter.security.telegramUnbound') }}</Badge>
          </div>
          <p class="text-xs text-muted-foreground">
            {{
              !telegramEnabled
                ? t('personalCenter.security.telegramDisabledTip')
                : showMiniAppBindAction
                  ? t('personalCenter.security.telegramMiniAppBindHint')
                  : t('personalCenter.security.telegramSubtitle')
            }}
          </p>
        </div>
      </div>

      <div v-if="telegramEnabled" class="flex shrink-0 flex-col items-start gap-1.5 sm:items-end">
        <Button
          v-if="showMiniAppBindAction"
          type="button"
          size="sm"
          :disabled="bindingTelegram || miniAppInitData === ''"
          @click="$emit('miniAppBind')"
        >
          {{
            bindingTelegram
              ? t('personalCenter.security.telegramMiniAppBinding')
              : t('personalCenter.security.telegramMiniAppBindAction')
          }}
        </Button>
        <div v-else-if="showTelegramWidget" ref="telegramWidgetRef" class="flex justify-start sm:justify-end"></div>
        <Button
          v-else-if="showTelegramOidcBind"
          type="button"
          variant="outline"
          size="sm"
          :disabled="bindingTelegram"
          @click="$emit('oidcBind')"
        >
          {{ t('personalCenter.security.telegramOidcBindButton') }}
        </Button>
        <Button
          v-if="showTelegramMiniAppEntry"
          type="button"
          variant="ghost"
          size="sm"
          class="h-7 px-2 text-xs text-muted-foreground"
          @click="$emit('openMiniAppEntry')"
        >
          {{ t('personalCenter.security.telegramMiniAppEntryAction') }}
        </Button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'

const { t } = useI18n()

defineProps<{
  telegramEnabled: boolean
  telegramBound: boolean
  loadingTelegramBinding: boolean
  avatarUrl: string
  telegramDisplayName: string
  providerUserId: string
  formattedAuthAt: string
  unbindingTelegram: boolean
  canUnbindTelegram: boolean
  showTelegramMiniAppEntry: boolean
  showMiniAppBindAction: boolean
  showTelegramWidget: boolean
  showTelegramOidcBind?: boolean
  bindingTelegram: boolean
  miniAppInitData: string
}>()

defineEmits<{
  unbind: []
  miniAppBind: []
  openMiniAppEntry: []
  oidcBind: []
}>()

const telegramWidgetRef = ref<HTMLDivElement | null>(null)

defineExpose({ telegramWidgetRef })
</script>
