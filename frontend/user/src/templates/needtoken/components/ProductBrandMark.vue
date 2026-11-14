<template>
  <span class="product-brand-mark" :data-brand="brand" aria-hidden="true">
    <span v-if="brand !== 'unknown'" class="product-brand-symbol" :style="{ maskImage: `url(/brands/${brand === 'gpt' ? 'openai' : 'claude'}.svg)` }" />
    <Package v-else :size="20" :stroke-width="1.5" />
  </span>
</template>

<script setup lang="ts">
import { Package } from 'lucide-vue-next'
import type { ProductBrand } from '../lib/homeCardPresentation'
defineProps<{ brand: ProductBrand }>()
</script>

<style scoped>
.product-brand-mark { display: inline-grid; width: 24px; height: 24px; flex: none; place-items: center; color: var(--product-brand-ink, #263c32); }
.product-brand-mark[data-brand="claude"] { color: var(--product-brand-ink, #a9553b); }
.product-brand-mark[data-brand="unknown"] { color: var(--product-brand-ink, #65766d); }
.product-brand-symbol { display: block; width: 22px; height: 22px; background: currentColor; mask-repeat: no-repeat; mask-position: center; mask-size: contain; }
</style>
