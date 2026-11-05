<template>
  <div ref="rootEl" class="nt-footer-wordmark" :data-motion="canAnimate ? 'running' : 'static'" aria-hidden="true" @pointermove="onPointerMove" @pointerleave="hidePointer">
    <svg viewBox="0 0 1400 274" preserveAspectRatio="xMidYMax meet" fill="none" focusable="false">
      <defs>
        <symbol :id="`${id}-cell`" viewBox="0 0 42 42">
          <circle cx="7" cy="7" r="2.7" opacity=".85" /><path d="M21 3.6 24.2 9.5H17.8Z" opacity=".68" /><rect x="32.5" y="4.5" width="5" height="5" opacity=".75" />
          <rect x="4.5" y="18.5" width="5" height="5" opacity=".65" /><circle cx="21" cy="21" r="2.7" opacity=".9" /><path d="M35 17.6 38.2 23.5H31.8Z" opacity=".72" />
          <path d="M7 31.6 10.2 37.5H3.8Z" opacity=".77" /><rect x="18.5" y="32.5" width="5" height="5" opacity=".85" /><circle cx="35" cy="35" r="2.7" opacity=".67" />
        </symbol>
        <pattern :id="`${id}-base`" width="42" height="42" patternUnits="userSpaceOnUse"><use :href="`#${id}-cell`" width="42" height="42" fill="#54756f" /></pattern>
        <pattern :id="`${id}-light`" width="42" height="42" patternUnits="userSpaceOnUse"><use :href="`#${id}-cell`" width="42" height="42" fill="#8cebd2" /></pattern>
        <linearGradient :id="`${id}-fade`" x1="0" y1="0" x2="0" y2="274" gradientUnits="userSpaceOnUse"><stop stop-color="white" stop-opacity="0" /><stop offset=".21" stop-color="white" stop-opacity=".16" /><stop offset=".52" stop-color="white" stop-opacity=".72" /><stop offset="1" stop-color="white" /></linearGradient>
        <radialGradient :id="`${id}-spot`"><stop stop-color="white" /><stop offset="1" stop-color="white" stop-opacity="0" /></radialGradient>
        <mask :id="`${id}-word`" maskUnits="userSpaceOnUse" x="0" y="0" width="1400" height="300">
          <rect width="1400" height="300" fill="white" />
          <text x="700" y="258" text-anchor="middle" textLength="1300" lengthAdjust="spacingAndGlyphs" fill="black" font-family="Inter, ui-sans-serif, system-ui, sans-serif" font-size="224" font-weight="800" letter-spacing="-10">{{ word }}</text>
        </mask>
        <mask :id="`${id}-topfade`" maskUnits="userSpaceOnUse"><rect width="1400" height="274" :fill="`url(#${id}-fade)`" /></mask>
        <mask :id="`${id}-pointer`" maskUnits="userSpaceOnUse"><circle :cx="pointer.x" :cy="pointer.y" r="220" :fill="`url(#${id}-spot)`" /></mask>
      </defs>
      <g :mask="`url(#${id}-topfade)`"><g :mask="`url(#${id}-word)`">
        <rect class="nt-footer-shape-field" x="-14" y="-14" width="1428" height="302" :fill="`url(#${id}-base)`" />
        <g :mask="`url(#${id}-pointer)`" class="nt-footer-shape-highlight" :style="{ opacity: pointerVisible ? .9 : 0 }"><rect class="nt-footer-shape-field" x="-14" y="-14" width="1428" height="302" :fill="`url(#${id}-light)`" /></g>
      </g></g>
    </svg>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive, ref, useId } from 'vue'
defineProps<{ word: string }>()
const id = `nt-footer-art-${useId().replace(/[^a-zA-Z0-9_-]/g, '')}`
const rootEl = ref<HTMLElement | null>(null)
const visible = ref(false), pageVisible = ref(false), motionAllowed = ref(false), pointerAllowed = ref(false), pointerVisible = ref(false)
const pointer = reactive({ x: 700, y: 180 })
const canAnimate = computed(() => visible.value && pageVisible.value && motionAllowed.value)
let observer: IntersectionObserver | undefined, motionQuery: MediaQueryList | undefined, pointerQuery: MediaQueryList | undefined
let frame = 0, nextX = 700, nextY = 180
interface ConnectionHint extends EventTarget { saveData?: boolean }
let connection: ConnectionHint | undefined
function syncCapability() {
  const hardware = navigator as Navigator & { deviceMemory?: number }
  motionAllowed.value = !motionQuery?.matches && !connection?.saveData && !(hardware.deviceMemory && hardware.deviceMemory <= 4) && !(hardware.hardwareConcurrency && hardware.hardwareConcurrency <= 4)
  pointerAllowed.value = Boolean(pointerQuery?.matches && motionAllowed.value)
  if (!pointerAllowed.value) hidePointer()
}
function hidePointer() { pointerVisible.value = false; if (frame) cancelAnimationFrame(frame); frame = 0 }
function syncVisibility() { pageVisible.value = !document.hidden; if (document.hidden) hidePointer() }
function onPointerMove(event: PointerEvent) {
  if (!canAnimate.value || !pointerAllowed.value || event.pointerType === 'touch' || !rootEl.value) return
  const bounds = rootEl.value.getBoundingClientRect()
  if (!bounds.width || !bounds.height) return
  nextX = (event.clientX - bounds.left) * 1400 / bounds.width; nextY = (event.clientY - bounds.top) * 274 / bounds.height
  if (!frame) frame = requestAnimationFrame(() => { frame = 0; if (!canAnimate.value || !pointerAllowed.value) return; pointer.x = nextX; pointer.y = nextY; pointerVisible.value = true })
}
onMounted(() => {
  motionQuery = matchMedia('(prefers-reduced-motion: reduce)'); pointerQuery = matchMedia('(hover: hover) and (pointer: fine)')
  connection = (navigator as Navigator & { connection?: ConnectionHint }).connection
  motionQuery.addEventListener('change', syncCapability); pointerQuery.addEventListener('change', syncCapability); connection?.addEventListener('change', syncCapability)
  document.addEventListener('visibilitychange', syncVisibility); syncCapability(); syncVisibility()
  if ('IntersectionObserver' in window && rootEl.value) { observer = new IntersectionObserver(([entry]) => { visible.value = !!entry?.isIntersecting; if (!visible.value) hidePointer() }); observer.observe(rootEl.value) } else visible.value = true
})
onUnmounted(() => {
  observer?.disconnect(); motionQuery?.removeEventListener('change', syncCapability); pointerQuery?.removeEventListener('change', syncCapability); connection?.removeEventListener('change', syncCapability)
  document.removeEventListener('visibilitychange', syncVisibility); if (frame) cancelAnimationFrame(frame)
})
</script>

<style scoped>
.nt-footer-wordmark { position: relative; width: 100%; height: clamp(136px, 21vw, 286px); overflow: hidden; contain: layout paint; opacity: .7; }
.nt-footer-wordmark svg { display: block; width: 100%; height: 100%; }
.nt-footer-shape-field { transform-origin: center; animation: nt-footer-field-drift 26s steps(390) infinite alternate; animation-play-state: paused; }
.nt-footer-wordmark[data-motion="running"] .nt-footer-shape-field { animation-play-state: running; }
.nt-footer-shape-highlight { transition: opacity 240ms ease; }
@keyframes nt-footer-field-drift { from { transform: translate3d(-3px, 0, 0); } to { transform: translate3d(3px, -7px, 0); } }
@media (max-width: 640px) { .nt-footer-wordmark { height: 125px; } }
@media (prefers-reduced-motion: reduce) { .nt-footer-shape-field { animation: none; } .nt-footer-shape-highlight { transition: none; } }
</style>
