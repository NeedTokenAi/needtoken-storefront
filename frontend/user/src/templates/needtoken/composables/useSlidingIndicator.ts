import { computed, nextTick, onMounted, onUnmounted, ref, watch, type Ref, type WatchSource } from 'vue'
import { computeSlidingIndicator, type SlidingIndicatorRect } from '../lib/slidingIndicatorGeometry'

/**
 * Tracks the active item inside a segmented rail/tab container and exposes
 * compositor-friendly transform styles for a shared sliding pill thumb.
 * First measurement snaps immediately (no entrance fly-in from 0,0);
 * subsequent updates animate smoothly with a spring curve.
 */
export function useSlidingIndicator(
  containerRef: Ref<HTMLElement | null>,
  activeSelector: string,
  watchSources: WatchSource<unknown>[] = [],
) {
  const rect = ref<SlidingIndicatorRect>({ x: 0, y: 0, width: 0, height: 0, visible: false })
  const ready = ref(false)
  let observer: ResizeObserver | null = null
  let rafId = 0

  const measure = () => {
    const container = containerRef.value
    if (!container) {
      rect.value = { x: 0, y: 0, width: 0, height: 0, visible: false }
      return
    }
    const active = container.querySelector<HTMLElement>(activeSelector)
    if (!active) {
      rect.value = { x: 0, y: 0, width: 0, height: 0, visible: false }
      return
    }
    const next = computeSlidingIndicator(
      container.getBoundingClientRect(),
      active.getBoundingClientRect(),
      container.scrollLeft,
      container.scrollTop,
    )
    const wasVisible = rect.value.visible
    rect.value = next
    if (next.visible && !wasVisible) {
      if (rafId) window.cancelAnimationFrame(rafId)
      rafId = window.requestAnimationFrame(() => {
        rafId = 0
        ready.value = true
      })
    } else if (!next.visible) {
      ready.value = false
    }
  }

  const scheduleMeasure = () => {
    void nextTick(() => {
      measure()
    })
  }

  if (watchSources.length) {
    watch(watchSources, scheduleMeasure, { flush: 'post' })
  }

  onMounted(() => {
    scheduleMeasure()
    if (typeof ResizeObserver !== 'undefined' && containerRef.value) {
      observer = new ResizeObserver(() => measure())
      observer.observe(containerRef.value)
    }
    window.addEventListener('resize', measure, { passive: true })
    if (typeof document !== 'undefined' && 'fonts' in document) {
      void (document as Document & { fonts?: { ready?: Promise<unknown> } }).fonts?.ready?.then(measure)
    }
  })

  onUnmounted(() => {
    observer?.disconnect()
    observer = null
    if (rafId) window.cancelAnimationFrame(rafId)
    window.removeEventListener('resize', measure)
  })

  const indicatorStyle = computed(() => ({
    width: `${rect.value.width}px`,
    height: `${rect.value.height}px`,
    transform: `translate3d(${rect.value.x}px, ${rect.value.y}px, 0)`,
    opacity: rect.value.visible ? '1' : '0',
  }))

  return {
    indicatorStyle,
    indicatorReady: ready,
    indicatorVisible: computed(() => rect.value.visible),
    measure,
  }
}
