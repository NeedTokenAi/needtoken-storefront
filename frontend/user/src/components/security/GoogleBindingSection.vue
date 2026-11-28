<template>
  <div class="px-4 py-4 sm:px-5">
    <div v-if="loadingGoogleBinding" class="flex items-center gap-3 py-1 text-sm text-muted-foreground">
      <span class="flex h-10 w-10 shrink-0 items-center justify-center rounded-full border bg-muted/40 text-sm font-bold text-foreground">
        G
      </span>
      <div>
        <p class="text-sm font-semibold text-foreground">{{ t('personalCenter.security.googleTitle') }}</p>
        <p class="text-xs text-muted-foreground">{{ t('personalCenter.security.googleLoading') }}</p>
      </div>
    </div>

    <div v-else-if="googleBound" class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
      <div class="flex min-w-0 items-center gap-3">
        <img
          v-if="avatarUrl"
          :src="avatarUrl"
          :alt="t('personalCenter.security.googleAvatarAlt')"
          class="h-10 w-10 shrink-0 rounded-full border object-cover"
        />
        <div v-else class="flex h-10 w-10 shrink-0 items-center justify-center rounded-full border bg-muted/40 text-sm font-bold text-foreground">
          G
        </div>
        <div class="min-w-0 space-y-0.5">
          <div class="flex flex-wrap items-center gap-2">
            <span class="truncate text-sm font-semibold text-foreground">{{ googleDisplayName }}</span>
            <Badge variant="success" size="sm">{{ t('personalCenter.security.googleBound') }}</Badge>
          </div>
          <p class="truncate text-xs text-muted-foreground">
            <template v-if="email">{{ email }} · </template>{{ t('personalCenter.security.googleBindTime', { time: formattedAuthAt }) }}
          </p>
        </div>
      </div>

      <div class="flex shrink-0 flex-col items-start gap-1.5 sm:items-end">
        <Button
          type="button"
          variant="outline"
          size="sm"
          :disabled="unbindingGoogle || !canUnbindGoogle"
          @click="$emit('unbind')"
        >
          {{ unbindingGoogle ? t('personalCenter.security.googleUnbinding') : t('personalCenter.security.googleUnbind') }}
        </Button>
        <p v-if="!canUnbindGoogle" class="text-[11px] text-muted-foreground sm:text-right">
          {{ t('personalCenter.security.googleUnbindDisabledTip') }}
        </p>
      </div>
    </div>

    <div v-else class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
      <div class="flex min-w-0 items-center gap-3">
        <span class="flex h-10 w-10 shrink-0 items-center justify-center rounded-full border bg-muted/40">
          <svg class="h-4 w-4" viewBox="0 0 24 24" aria-hidden="true">
            <path fill="#4285F4" d="M23.745 12.27c0-.7-.06-1.4-.19-2.07H12v4.51h6.6c-.29 1.52-1.14 2.82-2.4 3.68v3.05h3.88c2.27-2.09 3.665-5.17 3.665-9.17Z" />
            <path fill="#34A853" d="M12 24c3.24 0 5.95-1.08 7.93-2.91l-3.88-3.05c-1.08.72-2.45 1.16-4.05 1.16-3.12 0-5.77-2.11-6.72-4.96H1.29v3.14C3.26 21.3 7.31 24 12 24Z" />
            <path fill="#FBBC05" d="M5.28 14.24c-.24-.72-.38-1.49-.38-2.24s.14-1.52.38-2.24V6.62H1.29C.47 8.24 0 10.06 0 12s.47 3.76 1.29 5.38l3.99-3.14Z" />
            <path fill="#EA4335" d="M12 4.75c1.77 0 3.35.61 4.6 1.8l3.42-3.42C17.95 1.19 15.24 0 12 0 7.31 0 3.26 2.7 1.29 6.62l3.99 3.14c.95-2.85 3.6-4.96 6.72-4.96Z" />
          </svg>
        </span>
        <div class="min-w-0 space-y-0.5">
          <div class="flex flex-wrap items-center gap-2">
            <span class="text-sm font-semibold text-foreground">{{ t('personalCenter.security.googleTitle') }}</span>
            <Badge variant="outline" size="sm">{{ t('personalCenter.security.googleUnbound') }}</Badge>
          </div>
          <p class="text-xs text-muted-foreground">
            {{
              !googleEnabled
                ? t('personalCenter.security.googleDisabledTip')
                : isTelegramMiniApp
                  ? t('personalCenter.security.googleMiniAppUnavailable')
                  : t('personalCenter.security.googleSubtitle')
            }}
          </p>
        </div>
      </div>

      <div v-if="googleEnabled && !isTelegramMiniApp" class="flex shrink-0 flex-col items-start gap-1 sm:items-end">
        <div class="flex justify-start sm:justify-end">
          <GoogleIdentityButton
            :client-id="clientId"
            :locale="locale"
            text="continue_with"
            :ux-mode="googleUxMode"
            :login-uri="googleLoginUri"
            :prepare-redirect="prepareRedirect"
            :disabled="bindingGoogle"
            :loading-label="t('personalCenter.security.googleButtonLoading')"
            @credential="$emit('credential', $event)"
            @error="$emit('scriptError')"
          />
        </div>
        <p v-if="bindingGoogle" class="text-[11px] text-muted-foreground sm:text-right" role="status">
          {{ t('personalCenter.security.googleBinding') }}
        </p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import GoogleIdentityButton from '../auth/GoogleIdentityButton.vue'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import type { GoogleRedirectPreparedIntent } from '../../utils/googleRedirect'

const { t } = useI18n()

defineProps<{
  googleEnabled: boolean
  googleBound: boolean
  loadingGoogleBinding: boolean
  clientId: string
  locale: string
  avatarUrl: string
  googleDisplayName: string
  email: string
  providerUserId: string
  formattedAuthAt: string
  bindingGoogle: boolean
  unbindingGoogle: boolean
  canUnbindGoogle: boolean
  isTelegramMiniApp: boolean
  googleUxMode: 'popup' | 'redirect'
  googleLoginUri: string
  prepareRedirect: () => Promise<GoogleRedirectPreparedIntent>
}>()

defineEmits<{
  credential: [credential: string]
  scriptError: []
  unbind: []
}>()
</script>
