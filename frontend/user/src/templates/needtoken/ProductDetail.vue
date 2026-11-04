<template>
  <ServicePurchasePage :key="String(route.params.slug || '')" />
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router'
import ServicePurchasePage from './components/ServicePurchasePage.vue'

// A different product must get fresh native purchase state, including its SKU
// selection and quantity. Query-only SKU changes are handled by the page.
const route = useRoute()
</script>
