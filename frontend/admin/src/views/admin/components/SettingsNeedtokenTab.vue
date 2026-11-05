<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { Textarea } from '@/components/ui/textarea'
import MediaPicker from '@/components/admin/MediaPicker.vue'
import { notifyError, notifySuccess } from '@/utils/notify'
import { mergeSiteSettingPatch, needtokenLanguages, normalizeNeedtokenResources, normalizeNeedtokenSupport, type NeedtokenLanguage, type NeedtokenPublicResource } from '@/utils/needtokenSettings'

const { t } = useI18n()
const busy = ref(false)
const loaded = ref(false)
const language = ref<NeedtokenLanguage>('zh-CN')
const support = reactive(normalizeNeedtokenSupport(undefined))
const resources = ref<NeedtokenPublicResource[]>([])

async function load() {
  busy.value = true
  try {
    const response = await adminAPI.getSettings({ key: 'site_config' })
    const value = response.data.data as Record<string, unknown>
    Object.assign(support, normalizeNeedtokenSupport(value?.needtoken_support))
    resources.value = normalizeNeedtokenResources(value?.needtoken_resources)
    loaded.value = true
  } catch (error) {
    if (!(error as { __notified?: boolean })?.__notified) notifyError(t('needtokenSettings.loadFailed'))
  } finally {
    busy.value = false
  }
}
function addResource() {
  if (resources.value.length >= 40) return
  resources.value.push({
    id: `resource-${crypto.randomUUID()}`,
    title: { 'zh-CN': '', 'zh-TW': '', 'en-US': '' },
    description: { 'zh-CN': '', 'zh-TW': '', 'en-US': '' },
    region: '', url: '', enabled: false, updated_at: '',
  })
}
async function save() {
  if (!loaded.value || busy.value) return
  busy.value = true
  try {
    const latest = await adminAPI.getSettings({ key: 'site_config' })
    const oldResources = normalizeNeedtokenResources((latest.data.data as Record<string, unknown>)?.needtoken_resources)
    const timestamp = new Date().toISOString()
    const entries = resources.value.map((entry) => {
      const old = oldResources.find((item) => item.id === entry.id)
      const content = (item: NeedtokenPublicResource) => JSON.stringify({ ...item, updated_at: '' })
      return { ...entry, updated_at: !old || content(old) !== content(entry) ? timestamp : old.updated_at }
    })
    const payload = mergeSiteSettingPatch(latest.data.data, { needtoken_support: { ...support }, needtoken_resources: { apple_ids: entries } })
    const response = await adminAPI.updateSettings({ key: 'site_config', value: payload })
    const saved = response.data.data as Record<string, unknown>
    Object.assign(support, normalizeNeedtokenSupport(saved.needtoken_support))
    resources.value = normalizeNeedtokenResources(saved.needtoken_resources)
    notifySuccess(t('needtokenSettings.saved'))
  } catch (error) {
    if (!(error as { __notified?: boolean })?.__notified) notifyError((error as Error)?.message || t('needtokenSettings.saveFailed'))
  } finally {
    busy.value = false
  }
}
onMounted(load)
defineExpose({ save })
</script>

<template>
  <div class="space-y-6" :aria-busy="busy">
    <div class="rounded-xl border border-border bg-card">
      <div class="border-b border-border bg-muted/40 px-6 py-4">
        <h2 class="text-lg font-semibold">{{ t('needtokenSettings.supportTitle') }}</h2>
        <p class="mt-1 text-xs text-muted-foreground">{{ t('needtokenSettings.supportHint') }}</p>
      </div>
      <fieldset :disabled="busy || !loaded" class="space-y-6 p-6">
        <div class="flex items-center gap-3">
          <Switch id="needtoken-support-enabled" v-model="support.enabled" />
          <Label for="needtoken-support-enabled">{{ t('needtokenSettings.enabled') }}</Label>
        </div>
        <div class="grid gap-5 md:grid-cols-2">
          <div class="space-y-2">
            <Label for="needtoken-business-email">{{ t('needtokenSettings.businessEmail') }}</Label>
            <Input id="needtoken-business-email" v-model="support.business_email" type="email" maxlength="254" autocomplete="off" />
          </div>
          <div class="space-y-2">
            <Label for="needtoken-service-hours">{{ t('needtokenSettings.serviceHours') }}</Label>
            <Input id="needtoken-service-hours" v-model="support.service_hours" maxlength="120" :placeholder="t('needtokenSettings.serviceHoursPlaceholder')" />
          </div>
          <div class="space-y-2 md:col-span-2">
            <Label for="needtoken-contact-url">{{ t('needtokenSettings.contactURL') }}</Label>
            <Input id="needtoken-contact-url" v-model="support.contact_url" type="url" maxlength="2000" placeholder="https://" />
            <p class="text-xs text-muted-foreground">{{ t('needtokenSettings.contactHint') }}</p>
          </div>
          <div class="space-y-2">
            <Label>{{ t('needtokenSettings.wecomQR') }}</Label>
            <MediaPicker v-model="support.wecom_qr" scene="common" />
            <Input v-model="support.wecom_qr" :aria-label="t('needtokenSettings.wecomQR')" maxlength="2000" :placeholder="t('needtokenSettings.imageURL')" />
          </div>
          <div class="space-y-2">
            <Label>{{ t('needtokenSettings.qqQR') }}</Label>
            <MediaPicker v-model="support.qq_qr" scene="common" />
            <Input v-model="support.qq_qr" :aria-label="t('needtokenSettings.qqQR')" maxlength="2000" :placeholder="t('needtokenSettings.imageURL')" />
          </div>
        </div>
      </fieldset>
    </div>
    <div class="rounded-xl border border-border bg-card">
      <div class="flex flex-wrap items-start justify-between gap-3 border-b border-border bg-muted/40 px-6 py-4">
        <div>
          <h2 class="text-lg font-semibold">{{ t('needtokenSettings.resourcesTitle') }}</h2>
          <p class="mt-1 max-w-3xl text-xs leading-relaxed text-muted-foreground">{{ t('needtokenSettings.resourcesHint') }}</p>
        </div>
        <Button type="button" variant="outline" :disabled="busy || !loaded || resources.length >= 40" @click="addResource">{{ t('needtokenSettings.addResource') }}</Button>
      </div>
      <fieldset :disabled="busy || !loaded" class="space-y-5 p-6">
        <div class="flex flex-wrap gap-2" :aria-label="t('needtokenSettings.language')">
          <Button v-for="lang in needtokenLanguages" :key="lang" type="button" size="sm" :variant="language === lang ? 'default' : 'outline'" :aria-pressed="language === lang" @click="language = lang">{{ t(`admin.common.lang.${lang === 'zh-CN' ? 'zhCN' : lang === 'zh-TW' ? 'zhTW' : 'enUS'}`) }}</Button>
        </div>
        <p class="text-sm text-muted-foreground">{{ t('needtokenSettings.publicWarning') }}</p>
        <div v-if="!resources.length" class="rounded-xl border border-dashed border-border px-4 py-8 text-center text-sm text-muted-foreground">{{ t('needtokenSettings.empty') }}</div>
        <div v-for="(entry, index) in resources" :key="entry.id" class="space-y-4 rounded-xl border border-border p-4">
          <div class="flex flex-wrap items-center justify-between gap-3">
            <div class="flex items-center gap-3">
              <Switch :id="`resource-enabled-${entry.id}`" v-model="entry.enabled" />
              <Label :for="`resource-enabled-${entry.id}`">{{ t('needtokenSettings.publishResource') }}</Label>
            </div>
            <Button type="button" size="sm" variant="destructive" @click="resources.splice(index, 1)">{{ t('admin.common.delete') }}</Button>
          </div>
          <div class="grid gap-4 md:grid-cols-2">
            <div class="space-y-2">
              <Label :for="`resource-title-${entry.id}`">{{ t('needtokenSettings.resourceTitle', { language }) }}</Label>
              <Input :id="`resource-title-${entry.id}`" v-model="entry.title[language]" maxlength="160" />
            </div>
            <div class="space-y-2">
              <Label :for="`resource-region-${entry.id}`">{{ t('needtokenSettings.region') }}</Label>
              <Input :id="`resource-region-${entry.id}`" v-model="entry.region" maxlength="80" />
            </div>
            <div class="space-y-2 md:col-span-2">
              <Label :for="`resource-url-${entry.id}`">{{ t('needtokenSettings.resourceURL') }}</Label>
              <Input :id="`resource-url-${entry.id}`" v-model="entry.url" type="url" maxlength="2000" placeholder="https://" />
            </div>
            <div class="space-y-2 md:col-span-2">
              <Label :for="`resource-description-${entry.id}`">{{ t('needtokenSettings.description', { language }) }}</Label>
              <Textarea :id="`resource-description-${entry.id}`" v-model="entry.description[language]" maxlength="600" rows="3" />
            </div>
          </div>
          <p class="break-all text-xs text-muted-foreground">{{ t('needtokenSettings.publicationNote') }}</p>
        </div>
      </fieldset>
    </div>
    <div v-if="!loaded && !busy" class="flex items-center gap-3"><p class="text-sm text-destructive">{{ t('needtokenSettings.loadFailed') }}</p><Button type="button" variant="outline" @click="load">{{ t('needtokenSettings.retry') }}</Button></div>
  </div>
</template>
