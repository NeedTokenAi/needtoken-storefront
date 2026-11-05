<template>
  <div ref="rootEl" class="nt-footer-wordmark" :class="{ 'nt-footer-wordmark--ripple': pointerVisible && canvasReady }" :data-motion="pointerVisible && canvasReady ? 'running' : 'static'" aria-hidden="true" @pointermove="onPointerMove" @pointerleave="hidePointer">
    <svg viewBox="0 0 1400 274" preserveAspectRatio="xMidYMax meet" fill="none" focusable="false">
      <defs>
        <pattern :id="`${id}-squares`" width="8" height="8" patternUnits="userSpaceOnUse"><rect width="6" height="6" fill="#789e94" shape-rendering="crispEdges" /></pattern>
        <linearGradient :id="`${id}-fade`" x1="0" y1="0" x2="0" y2="274" gradientUnits="userSpaceOnUse"><stop stop-color="white" stop-opacity="0" /><stop offset=".21" stop-color="white" stop-opacity=".16" /><stop offset=".52" stop-color="white" stop-opacity=".72" /><stop offset="1" stop-color="white" /></linearGradient>
        <mask :id="`${id}-word`" maskUnits="userSpaceOnUse" x="0" y="0" width="1400" height="274"><rect width="1400" height="274" fill="white" /><text x="700" y="258" text-anchor="middle" textLength="1300" lengthAdjust="spacingAndGlyphs" fill="black" font-family="Inter, ui-sans-serif, system-ui, sans-serif" font-size="224" font-weight="800" letter-spacing="-10">{{ word }}</text></mask>
        <mask :id="`${id}-topfade`" maskUnits="userSpaceOnUse"><rect width="1400" height="274" :fill="`url(#${id}-fade)`" /></mask>
      </defs>
      <g :mask="`url(#${id}-topfade)`"><rect width="1400" height="274" :mask="`url(#${id}-word)`" :fill="`url(#${id}-squares)`" /></g>
    </svg>
    <canvas ref="canvasEl" />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, useId, watch } from 'vue'
const props = defineProps<{ word: string }>()
const id = `nt-footer-art-${useId().replace(/[^a-zA-Z0-9_-]/g, '')}`
const rootEl = ref<HTMLElement | null>(null), canvasEl = ref<HTMLCanvasElement | null>(null)
const visible = ref(false), pageVisible = ref(false), motionAllowed = ref(false), pointerAllowed = ref(false), pointerVisible = ref(false), canvasReady = ref(false)
const canAnimate = computed(() => visible.value && pageVisible.value && motionAllowed.value && pointerAllowed.value)
let observer: IntersectionObserver | undefined, resizeObserver: ResizeObserver | undefined, motionQuery: MediaQueryList | undefined, pointerQuery: MediaQueryList | undefined
interface ConnectionHint extends EventTarget { saveData?: boolean }
let connection: ConnectionHint | undefined
let frame = 0, lastFrame = 0, disposed = false, canvasFailed = false, performanceLimited = false, slowFrames = 0
let context: CanvasRenderingContext2D | null = null, grid: HTMLCanvasElement | null = null, wordMask: HTMLCanvasElement | null = null
let pixelRatio = 1, scale = 1, originX = 0, originY = 0, pointerX = 0, pointerY = 0
const CELL = 8, SQUARE = 6, COLOR = '#789e94', FONT = 'Inter, ui-sans-serif, system-ui, sans-serif'

function opacityAt(y: number) {
  const position = y / 274
  if (position < .21) return position / .21 * .16
  if (position < .52) return .16 + (position - .21) / .31 * .56
  return .72 + (position - .52) / .48 * .28
}
function prepareCanvas() {
  if (!rootEl.value || !canvasEl.value || !canAnimate.value || canvasFailed) return false
  try {
    const bounds = rootEl.value.getBoundingClientRect()
    if (!bounds.width || !bounds.height) return false
    context = canvasEl.value.getContext('2d')
    if (!context) { canvasFailed = true; return false }
    pixelRatio = Math.min(devicePixelRatio || 1, 1.5)
    const width = Math.round(bounds.width * pixelRatio), height = Math.round(bounds.height * pixelRatio)
    canvasEl.value.width = width; canvasEl.value.height = height
    grid = document.createElement('canvas'); wordMask = document.createElement('canvas')
    grid.width = wordMask.width = width; grid.height = wordMask.height = height
    const gridContext = grid.getContext('2d'), maskContext = wordMask.getContext('2d')
    if (!gridContext || !maskContext) { grid = wordMask = null; canvasFailed = true; return false }
    scale = Math.min(width / 1400, height / 274); originX = (width - 1400 * scale) / 2; originY = height - 274 * scale
    // Snap to device pixels. At rest the static SVG needs no render loop.
    gridContext.fillStyle = COLOR
    const side = Math.max(1, Math.round(SQUARE * scale))
    for (let y = 0; y < 274; y += CELL) {
      gridContext.globalAlpha = opacityAt(y)
      for (let x = 0; x < 1400; x += CELL) gridContext.fillRect(Math.round(originX + x * scale), Math.round(originY + y * scale), side, side)
    }
    maskContext.fillStyle = '#fff'; maskContext.fillRect(0, 0, width, height)
    maskContext.globalCompositeOperation = 'destination-out'; maskContext.font = `800 ${224 * scale}px ${FONT}`
    maskContext.textAlign = 'center'; maskContext.textBaseline = 'alphabetic'; maskContext.letterSpacing = `${-10 * scale}px`
    const measured = maskContext.measureText(props.word).width
    maskContext.save(); maskContext.translate(width / 2, originY + 258 * scale)
    maskContext.scale(measured > 0 ? 1300 * scale / measured : 1, 1); maskContext.fillText(props.word, 0, 0); maskContext.restore()
    canvasReady.value = true
    return true
  } catch { canvasFailed = true; context = null; grid = wordMask = null; return false }
}
function renderRipple(now: number) {
  frame = 0
  if (!canAnimate.value || !pointerVisible.value || !context || !canvasEl.value || !grid || !wordMask) return
  // Only the pointer's neighborhood changes, at at most 24 fps.
  if (lastFrame && now - lastFrame < 1000 / 24) { frame = requestAnimationFrame(renderRipple); return }
  lastFrame = now
  const started = performance.now()
  const ctx = context, width = canvasEl.value.width, height = canvasEl.value.height
  ctx.globalAlpha = 1; ctx.globalCompositeOperation = 'source-over'; ctx.clearRect(0, 0, width, height); ctx.drawImage(grid, 0, 0)
  const radius = Math.min(170 * pixelRatio, width * .22), amplitude = 9 * pixelRatio, padding = amplitude + CELL * scale
  const left = Math.max(0, Math.floor(pointerX - radius - padding)), top = Math.max(0, Math.floor(pointerY - radius - padding))
  const right = Math.min(width, Math.ceil(pointerX + radius + padding)), bottom = Math.min(height, Math.ceil(pointerY + radius + padding))
  ctx.save(); ctx.beginPath(); ctx.rect(left, top, right - left, bottom - top); ctx.clip()
  ctx.clearRect(left, top, right - left, bottom - top)
  const firstColumn = Math.max(0, Math.floor((left - originX) / scale / CELL) - 1), lastColumn = Math.min(Math.ceil(1400 / CELL), Math.ceil((right - originX) / scale / CELL) + 1)
  const firstRow = Math.max(0, Math.floor((top - originY) / scale / CELL) - 1), lastRow = Math.min(Math.ceil(274 / CELL), Math.ceil((bottom - originY) / scale / CELL) + 1)
  const side = Math.max(1, Math.round(SQUARE * scale)), phase = now / 1000 * 4.5
  ctx.fillStyle = COLOR
  for (let row = firstRow; row < lastRow; row++) {
    const y = originY + row * CELL * scale
    for (let column = firstColumn; column < lastColumn; column++) {
      const x = originX + column * CELL * scale, dx = x - pointerX, dy = y - pointerY, distance = Math.hypot(dx, dy)
      const envelope = Math.max(0, 1 - distance / radius) ** 2, wave = distance / (11 * pixelRatio) - phase
      const displacement = Math.sin(wave) * amplitude * envelope, inverseDistance = distance > 0 ? 1 / distance : 0
      // Refract square positions and open transparent troughs. No glow/ring
      // overlay: the actual artwork bends and reveals the footer beneath it.
      ctx.globalAlpha = opacityAt(row * CELL) * (1 - envelope * (.2 + .6 * Math.max(0, Math.cos(wave)) ** 4))
      ctx.fillRect(Math.round(x + dx * inverseDistance * displacement), Math.round(y + dy * inverseDistance * displacement), side, side)
    }
  }
  ctx.restore(); ctx.globalAlpha = 1; ctx.globalCompositeOperation = 'destination-in'; ctx.drawImage(wordMask, 0, 0); ctx.globalCompositeOperation = 'source-over'
  slowFrames = performance.now() - started > 20 ? slowFrames + 1 : Math.max(0, slowFrames - 1)
  if (slowFrames >= 8) { performanceLimited = true; pointerAllowed.value = false; hidePointer(); return }
  frame = requestAnimationFrame(renderRipple)
}
function hidePointer() { pointerVisible.value = false; if (frame) cancelAnimationFrame(frame); frame = 0; lastFrame = 0 }
function syncCapability() {
  const hardware = navigator as Navigator & { deviceMemory?: number }
  motionAllowed.value = !motionQuery?.matches && !connection?.saveData && !(hardware.deviceMemory && hardware.deviceMemory <= 4) && !(hardware.hardwareConcurrency && hardware.hardwareConcurrency <= 4)
  pointerAllowed.value = Boolean(pointerQuery?.matches && motionAllowed.value && !performanceLimited)
  if (!canAnimate.value) hidePointer()
}
function syncVisibility() { pageVisible.value = !document.hidden; if (document.hidden) hidePointer() }
function onPointerMove(event: PointerEvent) {
  if (!canAnimate.value || event.pointerType === 'touch' || !rootEl.value) return
  const bounds = rootEl.value.getBoundingClientRect()
  if (!bounds.width || !bounds.height || (!canvasReady.value && !prepareCanvas())) return
  pointerX = (event.clientX - bounds.left) * pixelRatio; pointerY = (event.clientY - bounds.top) * pixelRatio
  pointerVisible.value = true
  if (!frame) frame = requestAnimationFrame(renderRipple)
}
function onResize() { hidePointer(); canvasReady.value = false; grid = wordMask = null }
function observeMedia(query: MediaQueryList, attach: boolean) {
  if (typeof query.addEventListener === 'function') { if (attach) query.addEventListener('change', syncCapability); else query.removeEventListener('change', syncCapability) }
  else if (attach) query.addListener(syncCapability)
  else query.removeListener(syncCapability)
}
watch(() => props.word, onResize)
onMounted(() => {
  motionQuery = matchMedia('(prefers-reduced-motion: reduce)'); pointerQuery = matchMedia('(hover: hover) and (pointer: fine)')
  connection = (navigator as Navigator & { connection?: ConnectionHint }).connection
  observeMedia(motionQuery, true); observeMedia(pointerQuery, true); connection?.addEventListener('change', syncCapability)
  document.addEventListener('visibilitychange', syncVisibility); syncCapability(); syncVisibility()
  if ('IntersectionObserver' in window && rootEl.value) { observer = new IntersectionObserver(([entry]) => { visible.value = !!entry?.isIntersecting; if (!visible.value) hidePointer() }); observer.observe(rootEl.value) } else visible.value = true
  if ('ResizeObserver' in window && rootEl.value) { resizeObserver = new ResizeObserver(onResize); resizeObserver.observe(rootEl.value) }
  void document.fonts?.ready.then(() => { if (!disposed) onResize() })
})
onUnmounted(() => {
  disposed = true; observer?.disconnect(); resizeObserver?.disconnect()
  if (motionQuery) observeMedia(motionQuery, false)
  if (pointerQuery) observeMedia(pointerQuery, false)
  connection?.removeEventListener('change', syncCapability); document.removeEventListener('visibilitychange', syncVisibility)
  hidePointer(); grid = wordMask = null; context = null
})
</script>

<style scoped>
.nt-footer-wordmark { position: relative; width: 100%; height: clamp(136px, 21vw, 286px); overflow: hidden; contain: layout paint; }
.nt-footer-wordmark svg, .nt-footer-wordmark canvas { display: block; width: 100%; height: 100%; }
.nt-footer-wordmark svg { opacity: 1; }
.nt-footer-wordmark canvas { position: absolute; inset: 0; opacity: 0; pointer-events: none; }
.nt-footer-wordmark--ripple svg { opacity: 0; }
.nt-footer-wordmark--ripple canvas { opacity: 1; }
@media (max-width: 640px) { .nt-footer-wordmark { height: 125px; } }
</style>
