<template>
  <span class="product-brand-mark" :data-brand="brand" aria-hidden="true">
    <svg v-if="brand !== 'unknown'" class="product-brand-symbol" :viewBox="productBrandMarks[brand].viewBox" fill="currentColor" :fill-rule="brand === 'grok' ? 'evenodd' : 'nonzero'" focusable="false"><path v-for="(path, index) in productBrandMarks[brand].paths" :key="index" :d="path" /></svg>
    <Package v-else :size="20" :stroke-width="1.5" />
  </span>
</template>

<script setup lang="ts">
import { Package } from 'lucide-vue-next'
import type { ProductBrand } from '../lib/homeCardPresentation'
import { productBrandMarks } from '../lib/productBrandMarks'
defineProps<{ brand: ProductBrand }>()
</script>

<style scoped>
.product-brand-mark { display: inline-grid; width: 24px; height: 24px; flex: none; place-items: center; color: var(--product-brand-ink, #263c32); }
.product-brand-mark[data-brand="claude"] { color: var(--product-brand-ink, #a9553b); }
.product-brand-mark[data-brand="meta"] { color: var(--product-brand-ink, #0866ff); }
.product-brand-mark[data-brand="gemini"] { color: var(--product-brand-ink, #4489f7); }
.product-brand-mark[data-brand="unknown"] { color: var(--product-brand-ink, #65766d); }
.product-brand-symbol { display: block; width: 22px; height: 22px; }
</style>
