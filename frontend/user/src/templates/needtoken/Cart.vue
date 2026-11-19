<template>
  <div class="nt-cart-redirect" aria-hidden="true" />
</template>

<script setup lang="ts">
import { onMounted } from 'vue'
import { useRouter } from 'vue-router'

// NeedToken storefront uses an inline single-step checkout flow without a standalone cart page.
const router = useRouter()
onMounted(() => {
  void router.replace('/')
})
</script>
