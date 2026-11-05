export interface GhostFibersRendererOptions {
  active?: boolean
  reducedMotion?: boolean
  intro?: boolean
  maxDimension?: number
  maxFps?: number
  dpr?: number
  onReady?: () => void
  onStatic?: () => void
  onFallback?: () => void
}
export interface GhostFibersController {
  setActive(active: boolean): void
  setReducedMotion(reduced: boolean): void
  resize(): void
  destroy(): void
}
export const GHOST_FIBERS_PRISM: Readonly<Record<string, string | number | boolean>>
export function createGhostFibers(container: HTMLElement, canvas: HTMLCanvasElement, options?: GhostFibersRendererOptions): GhostFibersController | null
