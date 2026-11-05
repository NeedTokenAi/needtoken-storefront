<template><div class="nt-deferred-region" :style="{ '--nt-region-size': `${estimatedHeight}px`, '--nt-region-mobile-size': `${mobileEstimatedHeight}px` }"><slot /></div></template>

<script setup lang="ts">
withDefaults(defineProps<{ estimatedHeight?: number; mobileEstimatedHeight?: number }>(), { estimatedHeight: 360, mobileEstimatedHeight: 560 })
</script>

<style scoped>
/* Keep links, text and accessibility available; the browser defers only offscreen rendering. */
.nt-deferred-region { content-visibility: auto; contain-intrinsic-size: auto var(--nt-region-size, 360px); }
@media (max-width: 640px) { .nt-deferred-region { contain-intrinsic-size: auto var(--nt-region-mobile-size, 560px); } }
@supports not (content-visibility: auto) { .nt-deferred-region { contain-intrinsic-size: none; } }
</style>
