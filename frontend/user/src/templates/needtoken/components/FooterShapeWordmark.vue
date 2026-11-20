<template>
  <div
    ref="rootEl"
    class="nt-footer-wordmark"
    :class="{ 'nt-footer-wordmark--canvas': canvasReady }"
    :data-motion="animating ? 'running' : 'static'"
    aria-hidden="true"
    @pointermove="onPointerMove"
    @pointerdown="onPointerDown"
    @pointerleave="onPointerLeave"
  >
    <!-- 静态兜底：与 Canvas 同源的正向点阵字形（非白字抠洞），仅在 Canvas 未就绪或不可用时显示 -->
    <svg class="nt-footer-wordmark__fallback" viewBox="0 0 1400 274" preserveAspectRatio="xMidYMax meet" fill="none" focusable="false">
      <defs>
        <pattern :id="`${id}-bg`" width="8" height="8" patternUnits="userSpaceOnUse">
          <rect x="1" y="1" width="5" height="5" rx="1" class="nt-wordmark-bg-dot" />
        </pattern>
        <pattern :id="`${id}-fg`" width="8" height="8" patternUnits="userSpaceOnUse">
          <rect x="0.6" y="0.6" width="6.4" height="6.4" rx="1.2" class="nt-wordmark-fg-dot" />
        </pattern>
        <linearGradient :id="`${id}-fade`" x1="0" y1="0" x2="0" y2="274" gradientUnits="userSpaceOnUse">
          <stop stop-color="white" stop-opacity="0" />
          <stop offset="0.22" stop-color="white" stop-opacity="0.18" />
          <stop offset="0.56" stop-color="white" stop-opacity="0.78" />
          <stop offset="1" stop-color="white" stop-opacity="1" />
        </linearGradient>
        <mask :id="`${id}-topfade`" maskUnits="userSpaceOnUse">
          <rect width="1400" height="274" :fill="`url(#${id}-fade)`" />
        </mask>
        <mask :id="`${id}-glyph`" maskUnits="userSpaceOnUse" x="0" y="0" width="1400" height="274">
          <rect width="1400" height="274" fill="black" />
          <text
            x="700"
            y="248"
            text-anchor="middle"
            textLength="1260"
            lengthAdjust="spacingAndGlyphs"
            fill="white"
            font-family="Inter, ui-sans-serif, system-ui, sans-serif"
            font-size="220"
            font-weight="800"
            letter-spacing="-8"
          >{{ word }}</text>
        </mask>
      </defs>
      <g :mask="`url(#${id}-topfade)`">
        <rect width="1400" height="274" :fill="`url(#${id}-bg)`" />
      </g>
      <rect width="1400" height="274" :mask="`url(#${id}-glyph)`" :fill="`url(#${id}-fg)`" />
    </svg>

    <canvas ref="canvasEl" class="nt-footer-wordmark__canvas" />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, useId, watch } from 'vue'

const props = defineProps<{ word: string }>()
const id = `nt-footer-art-${useId().replace(/[^a-zA-Z0-9_-]/g, '')}`

const rootEl = ref<HTMLElement | null>(null)
const canvasEl = ref<HTMLCanvasElement | null>(null)
const rendered = ref(true)
const pageVisible = ref(true)
const motionAllowed = ref(false)
const pointerAllowed = ref(false)
const canvasReady = ref(false)
const animating = ref(false)
const canAnimate = computed(() => rendered.value && pageVisible.value && motionAllowed.value && pointerAllowed.value)

interface ConnectionHint extends EventTarget { saveData?: boolean }
interface RipplePulse { x: number; y: number; bornAt: number; strength: number }
interface ThemePalette {
  bg: [number, number, number]
  fg: [number, number, number]
  glow: [number, number, number]
  bgAlpha: number
  fgBaseAlpha: number
}

const VIRTUAL_W = 1400
const VIRTUAL_H = 274
const CELL = 8
const COLS = Math.floor(VIRTUAL_W / CELL)
const ROWS = Math.floor(VIRTUAL_H / CELL)
const TOTAL_CELLS = COLS * ROWS
const FONT = 'Inter, ui-sans-serif, system-ui, sans-serif'

const LIGHT_PALETTE: ThemePalette = {
  bg: [134, 168, 159],
  fg: [20, 118, 95],
  glow: [13, 174, 134],
  bgAlpha: 0.34,
  fgBaseAlpha: 0.78,
}

const DARK_PALETTE: ThemePalette = {
  bg: [108, 148, 140],
  fg: [52, 198, 160],
  glow: [110, 242, 208],
  bgAlpha: 0.28,
  fgBaseAlpha: 0.76,
}

let observer: IntersectionObserver | undefined
let resizeObserver: ResizeObserver | undefined
let themeObserver: MutationObserver | undefined
let motionQuery: MediaQueryList | undefined
let pointerQuery: MediaQueryList | undefined
let connection: ConnectionHint | undefined

let frame = 0
let lastFrame = 0
let disposed = false
let canvasFailed = false
let performanceLimited = false
let slowFrames = 0

let ctx: CanvasRenderingContext2D | null = null
let dpr = 1
let width = 0
let height = 0
let scale = 1
let originX = 0
let originY = 0
let isDark = false

// 扁平化统一像素矩阵：背景格与文字格同属一个阵列，交互时完全同频共振
const cellX = new Float32Array(TOTAL_CELLS)
const cellY = new Float32Array(TOTAL_CELLS)
const cellFade = new Float32Array(TOTAL_CELLS)
const cellGlyph = new Float32Array(TOTAL_CELLS)
let fieldBuilt = false

let pointerInside = false
let pointerX = 0
let pointerY = 0
let smoothX = 0
let smoothY = 0
let pointerEnergy = 0
let lastPulseX = -9999
let lastPulseY = -9999
let lastPulseAt = 0
const ripples: RipplePulse[] = []

function fadeAt(y: number) {
  const position = y / VIRTUAL_H
  if (position < 0.2) return (position / 0.2) * 0.16
  if (position < 0.56) return 0.16 + ((position - 0.2) / 0.36) * 0.6
  return 0.76 + ((position - 0.56) / 0.44) * 0.24
}

function readTheme(): boolean {
  return typeof document !== 'undefined' && document.documentElement.classList.contains('dark')
}

function sampleWordField() {
  const sampleW = VIRTUAL_W / 2
  const sampleH = VIRTUAL_H / 2
  const offscreen = document.createElement('canvas')
  offscreen.width = sampleW
  offscreen.height = sampleH
  const octx = offscreen.getContext('2d', { willReadFrequently: true })
  if (!octx) {
    canvasFailed = true
    return false
  }
  octx.clearRect(0, 0, sampleW, sampleH)
  octx.fillStyle = '#ffffff'
  octx.font = `800 ${110}px ${FONT}`
  octx.textAlign = 'center'
  octx.textBaseline = 'alphabetic'
  octx.letterSpacing = '-4px'
  const text = props.word.trim() || 'NeedToken.AI'
  const measured = octx.measureText(text).width
  const targetW = 630
  octx.save()
  octx.translate(sampleW / 2, 124)
  octx.scale(measured > 0 ? Math.min(1.06, targetW / measured) : 1, 1)
  octx.fillText(text, 0, 0)
  octx.restore()

  const image = octx.getImageData(0, 0, sampleW, sampleH).data
  const readAlpha = (sx: number, sy: number) => {
    const ix = Math.max(0, Math.min(sampleW - 1, sx))
    const iy = Math.max(0, Math.min(sampleH - 1, sy))
    return image[(iy * sampleW + ix) * 4 + 3]! / 255
  }

  let idx = 0
  for (let row = 0; row < ROWS; row++) {
    const vy = row * CELL + CELL * 0.5
    const fade = fadeAt(vy)
    const sy0 = Math.floor((row * CELL + 2) / 2)
    const sy1 = Math.floor((row * CELL + 5) / 2)
    for (let col = 0; col < COLS; col++) {
      const vx = col * CELL + CELL * 0.5
      const sx0 = Math.floor((col * CELL + 2) / 2)
      const sx1 = Math.floor((col * CELL + 5) / 2)
      // 2x2 超采样字形权重，使点阵字母边缘细腻平滑且与背景方格严格同格对齐
      const coverage = (
        readAlpha(sx0, sy0) +
        readAlpha(sx1, sy0) +
        readAlpha(sx0, sy1) +
        readAlpha(sx1, sy1)
      ) * 0.25
      cellX[idx] = vx
      cellY[idx] = vy
      cellFade[idx] = fade
      cellGlyph[idx] = coverage > 0.08 ? coverage : 0
      idx++
    }
  }
  fieldBuilt = true
  return true
}

function prepareCanvas() {
  if (!rootEl.value || !canvasEl.value || canvasFailed) return false
  try {
    const bounds = rootEl.value.getBoundingClientRect()
    if (!bounds.width || !bounds.height) return false
    ctx = canvasEl.value.getContext('2d')
    if (!ctx) {
      canvasFailed = true
      return false
    }
    dpr = Math.min(window.devicePixelRatio || 1, 1.75)
    width = Math.round(bounds.width * dpr)
    height = Math.round(bounds.height * dpr)
    canvasEl.value.width = width
    canvasEl.value.height = height
    scale = Math.min(width / VIRTUAL_W, height / VIRTUAL_H)
    originX = (width - VIRTUAL_W * scale) * 0.5
    originY = height - VIRTUAL_H * scale
    isDark = readTheme()
    if (!fieldBuilt && !sampleWordField()) return false
    canvasReady.value = true
    return true
  } catch {
    canvasFailed = true
    ctx = null
    return false
  }
}

function drawFrame(now: number, interactive: boolean) {
  if (!ctx || !canvasReady.value) return
  const palette = isDark ? DARK_PALETTE : LIGHT_PALETTE
  ctx.clearRect(0, 0, width, height)

  const phase = now * 0.0042
  const radius = Math.min(205 * dpr, width * 0.23)
  const invRadius = radius > 0 ? 1 / radius : 0
  const maxAmp = 7.5 * dpr
  const baseUnit = CELL * scale
  const rippleSpeed = 240 * dpr
  const rippleLife = 1350
  const rippleBand = 44 * dpr

  // 清理已衰减完毕的扩散涟漪
  for (let i = ripples.length - 1; i >= 0; i--) {
    if (now - ripples[i]!.bornAt > rippleLife) ripples.splice(i, 1)
  }

  const hasField = interactive && (pointerEnergy > 0.004 || ripples.length > 0)
  const [bgR, bgG, bgB] = palette.bg
  const [fgR, fgG, fgB] = palette.fg
  const [glR, glG, glB] = palette.glow

  for (let i = 0; i < TOTAL_CELLS; i++) {
    const glyph = cellGlyph[i]!
    const fade = cellFade[i]!
    const cx = originX + cellX[i]! * scale
    const cy = originY + cellY[i]! * scale

    let dispX = 0
    let dispY = 0
    let crest = 0
    let glow = 0

    if (hasField) {
      if (pointerEnergy > 0.004) {
        const dx = cx - smoothX
        const dy = cy - smoothY
        const dist = Math.hypot(dx, dy)
        if (dist < radius) {
          const norm = 1 - dist * invRadius
          const env = norm * norm * pointerEnergy
          const wave = Math.sin(dist / (13 * dpr) - phase)
          const push = wave * maxAmp * env
          if (dist > 0.001) {
            const invDist = 1 / dist
            dispX += dx * invDist * push
            dispY += dy * invDist * push
          }
          crest += wave * env
          glow += env * (0.45 + 0.55 * Math.max(0, wave))
        }
      }

      for (let r = 0; r < ripples.length; r++) {
        const pulse = ripples[r]!
        const age = now - pulse.bornAt
        if (age < 0 || age > rippleLife) continue
        const progress = age / rippleLife
        const ringRadius = (age / 1000) * rippleSpeed
        const dx = cx - pulse.x
        const dy = cy - pulse.y
        const dist = Math.hypot(dx, dy)
        const delta = dist - ringRadius
        if (Math.abs(delta) < rippleBand) {
          const ringEnv = Math.cos((delta / rippleBand) * Math.PI * 0.5)
          const decay = (1 - progress) * (1 - progress) * pulse.strength * ringEnv * ringEnv
          const ringWave = Math.sin((delta / rippleBand) * Math.PI)
          if (dist > 0.001) {
            const invDist = 1 / dist
            const push = ringWave * maxAmp * 0.85 * decay
            dispX += dx * invDist * push
            dispY += dy * invDist * push
          }
          crest += ringEnv * decay * 0.9
          glow += decay * 0.8
        }
      }
    }

    const glowClamped = Math.min(1, glow)
    // 统一由字形权重驱动：同一网格上的方块平滑地从背景微格过渡为字形主格
    const baseSideRatio = 0.64 + glyph * 0.24
    const dynamicScale = 1 + crest * 0.2 + glowClamped * 0.14
    const side = Math.max(1.2, baseUnit * baseSideRatio * dynamicScale)

    const baseAlpha = glyph > 0
      ? Math.max(fade * palette.bgAlpha, (palette.fgBaseAlpha + fade * 0.2) * glyph)
      : fade * palette.bgAlpha
    const alpha = Math.min(1, baseAlpha + glowClamped * (glyph > 0 ? 0.2 : 0.42))
    if (alpha < 0.015) continue

    const baseR = bgR + (fgR - bgR) * glyph
    const baseG = bgG + (fgG - bgG) * glyph
    const baseB = bgB + (fgB - bgB) * glyph
    const mix = Math.min(1, glowClamped * (glyph > 0 ? 0.72 : 0.82))
    const red = Math.round(baseR + (glR - baseR) * mix)
    const green = Math.round(baseG + (glG - baseG) * mix)
    const blue = Math.round(baseB + (glB - baseB) * mix)

    ctx.fillStyle = `rgba(${red},${green},${blue},${alpha.toFixed(3)})`
    const x = cx + dispX - side * 0.5
    const y = cy + dispY - side * 0.5
    ctx.fillRect(x, y, side, side)
  }
}

function drawStatic() {
  if (!canvasReady.value && !prepareCanvas()) return
  drawFrame(performance.now(), false)
}

function stopLoop() {
  if (frame) {
    window.cancelAnimationFrame(frame)
    frame = 0
  }
  lastFrame = 0
  animating.value = false
}

function tick(now: number) {
  frame = 0
  if (disposed || !canAnimate.value || !canvasReady.value) {
    stopLoop()
    return
  }
  // 限帧至最高 40fps，在流畅水波与极低 CPU 占用之间保持平衡
  if (lastFrame && now - lastFrame < 1000 / 40) {
    frame = window.requestAnimationFrame(tick)
    return
  }
  lastFrame = now
  const started = performance.now()

  const targetEnergy = pointerInside ? 1 : 0
  pointerEnergy += (targetEnergy - pointerEnergy) * (pointerInside ? 0.24 : 0.11)
  smoothX += (pointerX - smoothX) * 0.26
  smoothY += (pointerY - smoothY) * 0.26

  drawFrame(now, true)

  const elapsed = performance.now() - started
  slowFrames = elapsed > 22 ? slowFrames + 1 : Math.max(0, slowFrames - 1)
  if (slowFrames >= 10) {
    performanceLimited = true
    pointerAllowed.value = false
    pointerInside = false
    pointerEnergy = 0
    ripples.length = 0
    stopLoop()
    drawStatic()
    return
  }

  if (pointerInside || pointerEnergy > 0.005 || ripples.length > 0) {
    animating.value = true
    frame = window.requestAnimationFrame(tick)
  } else {
    pointerEnergy = 0
    drawFrame(now, false)
    stopLoop()
  }
}

function ensureRunning() {
  if (!canAnimate.value) return
  if (!canvasReady.value && !prepareCanvas()) return
  if (!frame) {
    animating.value = true
    frame = window.requestAnimationFrame(tick)
  }
}

function pushRipple(x: number, y: number, strength: number, now: number) {
  if (ripples.length >= 5) ripples.shift()
  ripples.push({ x, y, bornAt: now, strength })
}

function onPointerMove(event: PointerEvent) {
  if (!canAnimate.value || event.pointerType === 'touch' || !rootEl.value) return
  const bounds = rootEl.value.getBoundingClientRect()
  if (!bounds.width || !bounds.height) return
  if (!canvasReady.value && !prepareCanvas()) return
  const x = (event.clientX - bounds.left) * dpr
  const y = (event.clientY - bounds.top) * dpr
  if (!pointerInside) {
    smoothX = x
    smoothY = y
  }
  pointerX = x
  pointerY = y
  pointerInside = true

  const now = performance.now()
  if (Math.hypot(x - lastPulseX, y - lastPulseY) > 54 * dpr && now - lastPulseAt > 140) {
    lastPulseX = x
    lastPulseY = y
    lastPulseAt = now
    pushRipple(x, y, 0.72, now)
  }
  ensureRunning()
}

function onPointerDown(event: PointerEvent) {
  if (!canAnimate.value || event.pointerType === 'touch' || !rootEl.value) return
  const bounds = rootEl.value.getBoundingClientRect()
  if (!bounds.width || !bounds.height) return
  if (!canvasReady.value && !prepareCanvas()) return
  const x = (event.clientX - bounds.left) * dpr
  const y = (event.clientY - bounds.top) * dpr
  pointerX = x
  pointerY = y
  smoothX = x
  smoothY = y
  pointerInside = true
  pushRipple(x, y, 1, performance.now())
  ensureRunning()
}

function onPointerLeave() {
  pointerInside = false
  lastPulseX = -9999
  lastPulseY = -9999
}

function syncCapability() {
  const hardware = navigator as Navigator & { deviceMemory?: number }
  motionAllowed.value = !motionQuery?.matches
    && !connection?.saveData
    && !(hardware.deviceMemory && hardware.deviceMemory <= 4)
    && !(hardware.hardwareConcurrency && hardware.hardwareConcurrency <= 4)
  pointerAllowed.value = Boolean(pointerQuery?.matches && motionAllowed.value && !performanceLimited)
  if (!canAnimate.value) {
    pointerInside = false
    pointerEnergy = 0
    ripples.length = 0
    stopLoop()
    drawStatic()
  }
}

function syncVisibility() {
  pageVisible.value = !document.hidden
  if (!pageVisible.value) {
    pointerInside = false
    stopLoop()
  }
}

function onContentVisibilityChange(event: Event) {
  const skipped = Boolean((event as Event & { skipped?: boolean }).skipped)
  rendered.value = !skipped
  if (skipped) {
    pointerInside = false
    stopLoop()
  } else {
    drawStatic()
  }
}

function rebuild(resampleWord = false) {
  stopLoop()
  if (resampleWord) fieldBuilt = false
  canvasReady.value = false
  if (rendered.value && !disposed) drawStatic()
}

function observeMedia(query: MediaQueryList, attach: boolean) {
  if (typeof query.addEventListener === 'function') {
    if (attach) query.addEventListener('change', syncCapability)
    else query.removeEventListener('change', syncCapability)
  } else if (attach) {
    query.addListener(syncCapability)
  } else {
    query.removeListener(syncCapability)
  }
}

watch(() => props.word, () => rebuild(true))

onMounted(() => {
  motionQuery = window.matchMedia('(prefers-reduced-motion: reduce)')
  pointerQuery = window.matchMedia('(hover: hover) and (pointer: fine)')
  connection = (navigator as Navigator & { connection?: ConnectionHint }).connection
  observeMedia(motionQuery, true)
  observeMedia(pointerQuery, true)
  connection?.addEventListener('change', syncCapability)
  document.addEventListener('visibilitychange', syncVisibility)
  syncCapability()
  syncVisibility()

  const el = rootEl.value
  const supportsContentVisibility = 'contentVisibility' in document.documentElement.style
  if (el && supportsContentVisibility) {
    el.addEventListener('contentvisibilityautostatechange', onContentVisibilityChange)
  } else if (el && 'IntersectionObserver' in window) {
    observer = new IntersectionObserver(([entry]) => {
      rendered.value = Boolean(entry?.isIntersecting)
      if (!rendered.value) {
        pointerInside = false
        stopLoop()
      } else if (!canvasReady.value) {
        drawStatic()
      }
    }, { rootMargin: '200px' })
    observer.observe(el)
  }

  if (el && 'ResizeObserver' in window) {
    resizeObserver = new ResizeObserver(() => rebuild(false))
    resizeObserver.observe(el)
  }

  themeObserver = new MutationObserver(() => {
    const nextDark = readTheme()
    if (nextDark !== isDark) {
      isDark = nextDark
      if (canvasReady.value) drawFrame(performance.now(), animating.value)
    }
  })
  themeObserver.observe(document.documentElement, { attributes: true, attributeFilter: ['class'] })

  drawStatic()
  void document.fonts?.ready.then(() => {
    if (!disposed) rebuild(true)
  })
})

onUnmounted(() => {
  disposed = true
  stopLoop()
  rootEl.value?.removeEventListener('contentvisibilityautostatechange', onContentVisibilityChange)
  observer?.disconnect()
  resizeObserver?.disconnect()
  themeObserver?.disconnect()
  if (motionQuery) observeMedia(motionQuery, false)
  if (pointerQuery) observeMedia(pointerQuery, false)
  connection?.removeEventListener('change', syncCapability)
  document.removeEventListener('visibilitychange', syncVisibility)
  ctx = null
})
</script>

<style scoped>
.nt-footer-wordmark {
  position: relative;
  width: 100%;
  height: clamp(136px, 21vw, 286px);
  overflow: clip;
  contain: layout paint;
  content-visibility: auto;
  contain-intrinsic-size: auto none auto 220px;
}

.nt-footer-wordmark__fallback,
.nt-footer-wordmark__canvas {
  display: block;
  width: 100%;
  height: 100%;
}

.nt-footer-wordmark__canvas {
  position: absolute;
  inset: 0;
  opacity: 0;
  pointer-events: none;
}

.nt-footer-wordmark--canvas .nt-footer-wordmark__fallback {
  opacity: 0;
}

.nt-footer-wordmark--canvas .nt-footer-wordmark__canvas {
  opacity: 1;
}

.nt-wordmark-bg-dot {
  fill: rgba(134, 168, 159, 0.34);
}

.nt-wordmark-fg-dot {
  fill: rgba(20, 118, 95, 0.88);
}

:global(.dark) .nt-wordmark-bg-dot {
  fill: rgba(108, 148, 140, 0.28);
}

:global(.dark) .nt-wordmark-fg-dot {
  fill: rgba(52, 198, 160, 0.86);
}

@media (max-width: 640px) {
  .nt-footer-wordmark {
    height: 125px;
    contain-intrinsic-size: auto none auto 125px;
  }
}
</style>
