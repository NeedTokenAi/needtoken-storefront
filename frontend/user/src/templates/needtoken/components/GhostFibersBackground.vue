<template>
  <div
    ref="container"
    class="nt-fiber-background"
    :class="[`nt-fiber--${variant}`, { 'nt-fiber--animated': animated, 'nt-fiber--lit': lit }]"
    data-renderer="css-field"
    :data-motion="animated ? 'running' : 'static'"
    aria-hidden="true"
  >
    <div class="nt-hero-aurora nt-hero-aurora--primary" />
    <div class="nt-hero-aurora nt-hero-aurora--secondary" />
    <div class="nt-hero-grid" />
    <div class="nt-hero-dots" />
    <div v-if="variant === 'hero'" class="nt-hero-dots-lit" />
    <div v-if="variant === 'hero'" class="nt-hero-spotlight" />
    <div v-if="variant === 'hero'" class="nt-hero-beam" />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { ghostFibersPolicy } from '../lib/ghostFibersPolicy'

const props = withDefaults(defineProps<{ variant?: 'hero' | 'subtle' }>(), { variant: 'subtle' })
const container = ref<HTMLElement | null>(null)
const inView = ref(true)
const pageVisible = ref(true)
const motionAllowed = ref(true)
const finePointer = ref(false)
const lit = ref(false)
const animated = computed(() => props.variant === 'hero' && inView.value && pageVisible.value && motionAllowed.value)

let intersection: IntersectionObserver | undefined
let motionQuery: MediaQueryList | undefined
let pointerQuery: MediaQueryList | undefined
interface ConnectionHint extends EventTarget { saveData?: boolean }
let connection: ConnectionHint | undefined
let rafId = 0
let pointerX = 0
let pointerY = 0
let disposed = false

function syncPolicy() {
  if (disposed) return
  const hardware = navigator as Navigator & { deviceMemory?: number }
  const policy = ghostFibersPolicy({
    reducedMotion: motionQuery?.matches,
    saveData: connection?.saveData,
    hardwareConcurrency: hardware.hardwareConcurrency,
    deviceMemory: hardware.deviceMemory,
    coarsePointer: pointerQuery?.matches === false,
    subtle: props.variant === 'subtle',
  })
  motionAllowed.value = policy.animate
  finePointer.value = pointerQuery?.matches ?? false
  pageVisible.value = !document.hidden
  if (!animated.value || !finePointer.value) lit.value = false
}

function flushPointer() {
  rafId = 0
  const el = container.value
  if (!el || !animated.value || !finePointer.value) return
  const rect = el.getBoundingClientRect()
  if (
    pointerX < rect.left || pointerX > rect.right ||
    pointerY < rect.top || pointerY > rect.bottom
  ) {
    lit.value = false
    return
  }
  el.style.setProperty('--nt-hero-x', `${Math.round(pointerX - rect.left)}px`)
  el.style.setProperty('--nt-hero-y', `${Math.round(pointerY - rect.top)}px`)
  lit.value = true
}

function onPointerMove(event: PointerEvent) {
  if (event.pointerType === 'touch' || !animated.value || !finePointer.value) return
  pointerX = event.clientX
  pointerY = event.clientY
  if (!rafId) rafId = window.requestAnimationFrame(flushPointer)
}

function onPointerLeave() {
  lit.value = false
}

onMounted(() => {
  motionQuery = window.matchMedia('(prefers-reduced-motion: reduce)')
  pointerQuery = window.matchMedia('(hover: hover) and (pointer: fine)')
  connection = (navigator as Navigator & { connection?: ConnectionHint }).connection
  motionQuery.addEventListener?.('change', syncPolicy)
  pointerQuery.addEventListener?.('change', syncPolicy)
  connection?.addEventListener?.('change', syncPolicy)
  document.addEventListener('visibilitychange', syncPolicy)

  if ('IntersectionObserver' in window && container.value) {
    intersection = new IntersectionObserver(([entry]) => {
      inView.value = Boolean(entry?.isIntersecting)
      syncPolicy()
    })
    intersection.observe(container.value)
  }
  if (props.variant === 'hero') {
    window.addEventListener('pointermove', onPointerMove, { passive: true })
    document.addEventListener('pointerleave', onPointerLeave)
  }
  syncPolicy()
})

onUnmounted(() => {
  disposed = true
  if (rafId) window.cancelAnimationFrame(rafId)
  intersection?.disconnect()
  motionQuery?.removeEventListener?.('change', syncPolicy)
  pointerQuery?.removeEventListener?.('change', syncPolicy)
  connection?.removeEventListener?.('change', syncPolicy)
  document.removeEventListener('visibilitychange', syncPolicy)
  window.removeEventListener('pointermove', onPointerMove)
  document.removeEventListener('pointerleave', onPointerLeave)
})
</script>

<style>
.nt-fiber-background {
  position: absolute;
  inset: 0;
  z-index: 0;
  overflow: hidden;
  pointer-events: none;
  contain: paint;
  background: var(--ui-bg-page);
  --nt-hero-x: 50%;
  --nt-hero-y: 22%;
  --nt-field-dot: rgba(15, 23, 20, 0.13);
  --nt-field-dot-lit: rgba(13, 148, 115, 0.56);
  --nt-field-grid: rgba(15, 23, 20, 0.048);
  --nt-field-aurora-a: radial-gradient(ellipse 56% 52% at 18% 12%, rgba(13, 148, 115, 0.14), transparent 72%);
  --nt-field-aurora-b: radial-gradient(ellipse 52% 48% at 84% 10%, rgba(14, 165, 233, 0.11), transparent 72%);
  --nt-field-spot: radial-gradient(340px circle at var(--nt-hero-x) var(--nt-hero-y), rgba(13, 148, 115, 0.11), transparent 72%);
  --nt-field-beam: linear-gradient(90deg, transparent 8%, rgba(13, 148, 115, 0.28) 50%, transparent 92%);
}

html.dark .nt-fiber-background {
  --nt-field-dot: rgba(148, 163, 184, 0.17);
  --nt-field-dot-lit: rgba(45, 212, 167, 0.68);
  --nt-field-grid: rgba(148, 163, 184, 0.055);
  --nt-field-aurora-a: radial-gradient(ellipse 58% 54% at 18% 14%, rgba(45, 212, 167, 0.17), transparent 70%);
  --nt-field-aurora-b: radial-gradient(ellipse 54% 48% at 84% 8%, rgba(56, 189, 248, 0.12), transparent 70%);
  --nt-field-spot: radial-gradient(360px circle at var(--nt-hero-x) var(--nt-hero-y), rgba(45, 212, 167, 0.14), transparent 72%);
  --nt-field-beam: linear-gradient(90deg, transparent 8%, rgba(94, 234, 212, 0.34) 50%, transparent 92%);
}

.nt-hero-aurora {
  position: absolute;
  inset: -12% -8% -8%;
  will-change: transform;
  mask-image: radial-gradient(ellipse 84% 82% at 50% 22%, #000 38%, transparent 100%);
  -webkit-mask-image: radial-gradient(ellipse 84% 82% at 50% 22%, #000 38%, transparent 100%);
}
.nt-hero-aurora--primary {
  background: var(--nt-field-aurora-a);
}
.nt-hero-aurora--secondary {
  background: var(--nt-field-aurora-b);
}

.nt-hero-grid {
  position: absolute;
  inset: 0;
  background-image:
    linear-gradient(var(--nt-field-grid) 1px, transparent 1px),
    linear-gradient(90deg, var(--nt-field-grid) 1px, transparent 1px);
  background-size: 112px 112px;
  mask-image: radial-gradient(ellipse 84% 78% at 50% 18%, #000 22%, transparent 92%);
  -webkit-mask-image: radial-gradient(ellipse 84% 78% at 50% 18%, #000 22%, transparent 92%);
}

.nt-hero-dots,
.nt-hero-dots-lit {
  position: absolute;
  inset: 0;
  background-size: 28px 28px;
}

.nt-hero-dots {
  background-image: radial-gradient(circle at 1px 1px, var(--nt-field-dot) 1.05px, transparent 1.15px);
  mask-image: radial-gradient(ellipse 82% 76% at 50% 20%, #000 28%, transparent 94%);
  -webkit-mask-image: radial-gradient(ellipse 82% 76% at 50% 20%, #000 28%, transparent 94%);
}

.nt-hero-dots-lit {
  background-image: radial-gradient(circle at 1px 1px, var(--nt-field-dot-lit) 1.25px, transparent 1.35px);
  mask-image: radial-gradient(240px circle at var(--nt-hero-x) var(--nt-hero-y), #000 0%, transparent 78%);
  -webkit-mask-image: radial-gradient(240px circle at var(--nt-hero-x) var(--nt-hero-y), #000 0%, transparent 78%);
  opacity: 0;
  transition: opacity 280ms cubic-bezier(0.22, 1, 0.36, 1);
}

.nt-hero-spotlight {
  position: absolute;
  inset: 0;
  background: var(--nt-field-spot);
  opacity: 0;
  transition: opacity 320ms cubic-bezier(0.22, 1, 0.36, 1);
}

.nt-fiber--lit .nt-hero-dots-lit,
.nt-fiber--lit .nt-hero-spotlight {
  opacity: 1;
}

.nt-hero-beam {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 1px;
  background: var(--nt-field-beam);
  opacity: 0.72;
}

.nt-fiber-background::after {
  position: absolute;
  inset: 0;
  content: '';
  background: linear-gradient(180deg, transparent 0%, transparent 64%, var(--ui-bg-page) 100%);
}

.nt-fiber--subtle {
  max-height: 760px;
  opacity: 0.62;
}

@media (prefers-reduced-motion: no-preference) {
  .nt-fiber--animated .nt-hero-aurora--primary {
    animation: nt-aurora-drift-a 18s ease-in-out infinite alternate;
  }
  .nt-fiber--animated .nt-hero-aurora--secondary {
    animation: nt-aurora-drift-b 22s ease-in-out infinite alternate;
  }
}

@keyframes nt-aurora-drift-a {
  from { transform: translate3d(-1.8%, -1%, 0) scale(1); }
  to { transform: translate3d(2.4%, 1.8%, 0) scale(1.04); }
}

@keyframes nt-aurora-drift-b {
  from { transform: translate3d(1.6%, 1.2%, 0) scale(1.03); }
  to { transform: translate3d(-2.2%, -1.4%, 0) scale(0.98); }
}

@media (prefers-reduced-motion: reduce) {
  .nt-hero-dots-lit,
  .nt-hero-spotlight {
    display: none;
  }
}
</style>
