export interface SlidingIndicatorRect {
  x: number
  y: number
  width: number
  height: number
  visible: boolean
}

export interface ElementBox {
  left: number
  top: number
  width: number
  height: number
}

/**
 * Computes the indicator pill transform relative to its positioning container.
 * Uses getBoundingClientRect deltas so nested wrappers or scroll containers
 * stay aligned accurately across responsive reflows.
 */
export function computeSlidingIndicator(
  container: ElementBox | null | undefined,
  active: ElementBox | null | undefined,
  scrollLeft = 0,
  scrollTop = 0,
): SlidingIndicatorRect {
  if (!container || !active || active.width <= 0 || active.height <= 0) {
    return { x: 0, y: 0, width: 0, height: 0, visible: false }
  }
  const x = Math.round((active.left - container.left + scrollLeft) * 100) / 100
  const y = Math.round((active.top - container.top + scrollTop) * 100) / 100
  const width = Math.round(active.width * 100) / 100
  const height = Math.round(active.height * 100) / 100
  return { x, y, width, height, visible: true }
}
