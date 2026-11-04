<template>
  <div ref="container" class="nt-fiber-background" :class="`nt-fiber--${variant}`" :data-motion="canAnimate ? 'running' : 'static'" aria-hidden="true">
    <div class="nt-fiber-haze" />
    <div v-for="(ribbon, layer) in ribbons" :key="layer" class="nt-fiber-layer" :class="`nt-fiber-layer--${layer}`">
      <svg viewBox="0 0 1600 950" preserveAspectRatio="xMidYMid slice" fill="none" focusable="false">
        <defs>
          <linearGradient :id="`${id}-ribbon-${layer}`" x1="70" y1="850" x2="1510" y2="80" gradientUnits="userSpaceOnUse">
            <stop stop-color="#8b6ce8" stop-opacity="0.12" />
            <stop offset="0.23" stop-color="#a5b6ec" stop-opacity="0.7" />
            <stop offset="0.37" stop-color="#dcecf1" />
            <stop offset="0.52" stop-color="#34d8bb" stop-opacity="0.45" />
            <stop offset="0.7" stop-color="#9c82df" stop-opacity="0.6" />
            <stop offset="0.87" stop-color="#d8ecef" stop-opacity="0.9" />
            <stop offset="1" stop-color="#2dd4bf" stop-opacity="0.05" />
          </linearGradient>
        </defs>
        <path v-for="(fiber, index) in ribbon" :key="index" :d="fiber.path" :stroke="`url(#${id}-ribbon-${layer})`" :stroke-width="fiber.width" :opacity="fiber.opacity" :stroke-dasharray="fiber.dashed ? '1 5' : undefined" stroke-linecap="round" vector-effect="non-scaling-stroke" />
      </svg>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, useId } from 'vue'

withDefaults(defineProps<{ variant?: 'hero' | 'subtle' }>(), { variant: 'subtle' })
const id = `nt-fibers-${useId().replace(/[^a-zA-Z0-9_-]/g, '')}`
const container = ref<HTMLElement | null>(null)
const inView = ref(false)
const pageVisible = ref(false)
const motionAllowed = ref(false)
const canAnimate = computed(() => inView.value && pageVisible.value && motionAllowed.value)
let observer: IntersectionObserver | undefined
let motionQuery: MediaQueryList | undefined
interface ConnectionHint extends EventTarget { saveData?: boolean }
let connection: ConnectionHint | undefined

// Original static vector artwork. Only three composited layers move: no WebGL, render loop,
// shader, image download, SVG filter, or per-frame path recalculation is needed.
// The slow drift uses discrete CSS steps (at most 15 changes/second), not a 60fps scene.
const ribbons = [26, 20, 18].map((count, layer) => Array.from({ length: count }, (_, index) => {
  const offset = index - (count - 1) / 2
  const points = Array.from({ length: 53 }, (_, sample) => {
    const t = sample / 52
    const x = -130 + t * 1860
    const twist = Math.sin(t * 7.3 + layer * 1.6)
    const y = 870 - t * 830 + Math.sin(t * 9 + layer * 1.9) * (120 + layer * 16)
      + offset * (6 + 8 * twist) + layer * 95
    return `${sample ? 'L' : 'M'}${x.toFixed(1)} ${y.toFixed(1)}`
  })
  return {
    path: points.join(' '),
    width: index % 7 === 0 ? 1.2 : 0.7,
    opacity: index % 7 === 0 ? 0.74 : 0.22 + (index % 4) * 0.08,
    dashed: index % 3 === 0,
  }
}))

const syncMotion = () => {
  const hardware = navigator as Navigator & { deviceMemory?: number }
  const lowMemory = typeof hardware.deviceMemory === 'number' && hardware.deviceMemory <= 4
  const lowCPU = typeof hardware.hardwareConcurrency === 'number' && hardware.hardwareConcurrency <= 4
  motionAllowed.value = !motionQuery?.matches && !connection?.saveData && !lowMemory && !lowCPU
}
const syncVisibility = () => { pageVisible.value = !document.hidden }
onMounted(() => {
  motionQuery = window.matchMedia('(prefers-reduced-motion: reduce)')
  connection = (navigator as Navigator & { connection?: ConnectionHint }).connection
  motionQuery.addEventListener('change', syncMotion)
  connection?.addEventListener('change', syncMotion)
  document.addEventListener('visibilitychange', syncVisibility)
  syncMotion()
  syncVisibility()
  if ('IntersectionObserver' in window && container.value) {
    observer = new IntersectionObserver(([entry]) => { inView.value = Boolean(entry?.isIntersecting) }, { threshold: 0 })
    observer.observe(container.value)
  } else inView.value = true
})
onUnmounted(() => {
  observer?.disconnect()
  motionQuery?.removeEventListener('change', syncMotion)
  connection?.removeEventListener('change', syncMotion)
  document.removeEventListener('visibilitychange', syncVisibility)
})
</script>

<style scoped>
.nt-fiber-background { position: absolute; inset: 0; z-index: 0; overflow: hidden; pointer-events: none; contain: paint; background: #07090d; }
.nt-fiber-haze { position: absolute; inset: 0; background: radial-gradient(ellipse at 17% 88%, rgba(130, 103, 211, .1), transparent 41%), radial-gradient(ellipse at 89% 6%, rgba(39, 174, 161, .11), transparent 43%); }
.nt-fiber-layer { position: absolute; inset: -7%; opacity: .66; transform-origin: center; animation: fiber-drift 28s steps(420, end) infinite alternate; animation-play-state: paused; }
.nt-fiber-layer svg { display: block; width: 100%; height: 100%; }
.nt-fiber-layer--1 { opacity: .4; animation-name: fiber-drift-reverse; animation-duration: 33s; }
.nt-fiber-layer--2 { opacity: .24; animation-duration: 39s; animation-direction: alternate-reverse; }
.nt-fiber-background[data-motion="running"] .nt-fiber-layer { animation-play-state: running; }
.nt-fiber-background::after { position: absolute; inset: 0; content: ''; background: linear-gradient(180deg, rgba(7, 9, 13, .15), rgba(7, 9, 13, .2) 35%, rgba(7, 9, 13, .04) 64%, #07090d); }
.nt-fiber--subtle { max-height: 1100px; opacity: .4; }
.nt-fiber--subtle .nt-fiber-layer { opacity: .35; }
.nt-fiber--subtle .nt-fiber-layer--1 { opacity: .2; }
.nt-fiber--subtle .nt-fiber-layer--2 { display: none; }
@keyframes fiber-drift { from { transform: translate3d(-1%, 0, 0) rotate(-2deg) scale(1); } to { transform: translate3d(1.5%, -2%, 0) rotate(2deg) scale(1.035); } }
@keyframes fiber-drift-reverse { from { transform: translate3d(1.5%, -1%, 0) rotate(1.5deg) scale(1.02); } to { transform: translate3d(-1%, 2%, 0) rotate(-1.5deg) scale(1); } }
@media (max-width: 640px) { .nt-fiber-layer { inset: -3% -45%; } .nt-fiber--hero .nt-fiber-layer { opacity: .48; } }
@media (prefers-reduced-motion: reduce) { .nt-fiber-layer { animation: none; transform: none; } }
</style>
