<template>
  <div ref="container" class="nt-fiber-background" :class="[`nt-fiber--${variant}`, { 'nt-fiber--ready': ready }]" :data-renderer="ready ? 'webgl2' : 'poster'" :data-motion="moving ? 'running' : 'static'" aria-hidden="true">
    <div class="nt-fiber-haze" />
    <img v-if="!posterFailed" class="nt-fiber-poster" src="/needtoken-prism.webp" alt="" loading="lazy" decoding="async" @error="posterFailed = true" />
    <div v-for="(ribbon, layer) in (posterFailed ? ribbons : [])" :key="layer" class="nt-fiber-layer" :class="`nt-fiber-layer--${layer}`">
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
    <canvas ref="canvas" class="nt-fiber-canvas" @webglcontextlost="onContextLost" />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, useId } from 'vue'
import { ghostFibersPolicy } from '../lib/ghostFibersPolicy'
import type { GhostFibersController } from '../lib/ghostFibersRenderer.js'

const props = withDefaults(defineProps<{ variant?: 'hero' | 'subtle' }>(), { variant: 'subtle' })
const id = `nt-fibers-${useId().replace(/[^a-zA-Z0-9_-]/g, '')}`
const container = ref<HTMLElement | null>(null)
const canvas = ref<HTMLCanvasElement | null>(null)
const ready = ref(false)
const posterFailed = ref(false)
const inView = ref(false)
const pageVisible = ref(false)
const motionAllowed = ref(false)
const heldStatic = ref(false)
const moving = computed(() => ready.value && inView.value && pageVisible.value && motionAllowed.value && !heldStatic.value)
let renderer: GhostFibersController | null = null
let intersection: IntersectionObserver | undefined
let motionQuery: MediaQueryList | undefined
let pointerQuery: MediaQueryList | undefined
interface ConnectionHint extends EventTarget { saveData?: boolean }
let connection: ConnectionHint | undefined
let disposed = false
let failed = false
let initializing = false
let idle: number | undefined
let idleUsesTimeout = false
let policy = ghostFibersPolicy({ subtle: props.variant === 'subtle' })

// Static vectors remain available if the exported WebP cannot load.
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

function fallback() {
  ready.value = false
  failed = true
  renderer?.destroy()
  renderer = null
}
function onContextLost(event: Event) {
  event.preventDefault()
  if (!disposed) fallback()
}
async function initialize() {
  idle = undefined
  if (disposed || failed || initializing || renderer || !inView.value || !pageVisible.value || !policy.animate) return
  initializing = true
  try {
    // GPU code is a separate chunk and only loads for visible, capable devices.
    const { createGhostFibers } = await import('../lib/ghostFibersRenderer.js')
    if (disposed || !inView.value || !pageVisible.value || !policy.animate || !container.value || !canvas.value) return
    renderer = createGhostFibers(container.value, canvas.value, {
      active: true, maxDimension: policy.maxDimension, maxFps: policy.maxFps, dpr: policy.dpr,
      onReady: () => { if (!disposed) ready.value = true },
      onStatic: () => { heldStatic.value = true },
      onFallback: fallback,
    })
    if (!renderer) fallback()
  } catch { fallback() } finally { initializing = false }
}
function sync() {
  if (disposed) return
  const hardware = navigator as Navigator & { deviceMemory?: number }
  policy = ghostFibersPolicy({
    reducedMotion: motionQuery?.matches, saveData: connection?.saveData,
    hardwareConcurrency: hardware.hardwareConcurrency, deviceMemory: hardware.deviceMemory,
    coarsePointer: pointerQuery?.matches, subtle: props.variant === 'subtle',
  })
  motionAllowed.value = policy.animate
  pageVisible.value = !document.hidden
  const active = inView.value && pageVisible.value
  if (renderer) {
    renderer.setReducedMotion(!policy.animate || heldStatic.value)
    renderer.setActive(active)
  } else if (active && policy.animate && !failed && !initializing && idle === undefined) {
    if (typeof window.requestIdleCallback === 'function') idle = window.requestIdleCallback(() => { void initialize() }, { timeout: 700 })
    else { idleUsesTimeout = true; idle = window.setTimeout(() => { void initialize() }, 80) }
  }
}
function observeMedia(query: MediaQueryList, observe: boolean) {
  if (typeof query.addEventListener === 'function') {
    if (observe) query.addEventListener('change', sync)
    else query.removeEventListener('change', sync)
  } else if (observe) query.addListener(sync)
  else query.removeListener(sync)
}
onMounted(() => {
  motionQuery = window.matchMedia('(prefers-reduced-motion: reduce)')
  pointerQuery = window.matchMedia('(pointer: coarse)')
  connection = (navigator as Navigator & { connection?: ConnectionHint }).connection
  observeMedia(motionQuery, true)
  observeMedia(pointerQuery, true)
  connection?.addEventListener('change', sync)
  document.addEventListener('visibilitychange', sync)
  if ('IntersectionObserver' in window && container.value) {
    intersection = new IntersectionObserver(([entry]) => { inView.value = Boolean(entry?.isIntersecting); sync() })
    intersection.observe(container.value)
  } else inView.value = true
  sync()
})
onUnmounted(() => {
  disposed = true
  if (idle !== undefined) {
    if (idleUsesTimeout) window.clearTimeout(idle)
    else window.cancelIdleCallback(idle)
  }
  intersection?.disconnect()
  if (motionQuery) observeMedia(motionQuery, false)
  if (pointerQuery) observeMedia(pointerQuery, false)
  connection?.removeEventListener('change', sync)
  document.removeEventListener('visibilitychange', sync)
  renderer?.destroy()
  renderer = null
})
</script>

<style scoped>
.nt-fiber-background { position: absolute; inset: 0; z-index: 0; overflow: hidden; pointer-events: none; contain: paint; background: #07090d; }
.nt-fiber-haze { position: absolute; inset: 0; background: radial-gradient(ellipse at 17% 88%, #8067d31a, transparent 45%), radial-gradient(ellipse at 89% 6%, #27aea11c, transparent 43%); }
.nt-fiber-poster, .nt-fiber-canvas { position: absolute; inset: 0; display: block; width: 100%; height: 100%; }
.nt-fiber-poster { object-fit: cover; opacity: .9; transition: opacity 800ms ease; }
.nt-fiber-canvas { opacity: 0; transition: opacity 1100ms ease; }
.nt-fiber--ready .nt-fiber-canvas { opacity: 1; }
.nt-fiber--ready .nt-fiber-poster, .nt-fiber--ready .nt-fiber-layer { opacity: 0; }
.nt-fiber-layer { position: absolute; inset: -7%; opacity: .65; }
.nt-fiber-layer svg { display: block; width: 100%; height: 100%; }
.nt-fiber-layer--1 { opacity: .4; }
.nt-fiber-layer--2 { opacity: .24; }
.nt-fiber-background::after { position: absolute; inset: 0; content: ''; background: linear-gradient(180deg, #07090d08, #07090d18 32%, #07090d08 68%, #07090d); }
.nt-fiber--subtle { max-height: 1100px; opacity: .36; }
@media (prefers-reduced-motion: reduce) { .nt-fiber-canvas, .nt-fiber-poster { transition: none; } }
</style>
