export interface ThemeTransitionOrigin {
  clientX: number
  clientY: number
}

interface ViewTransitionLike {
  finished: Promise<void>
}

type StartViewTransition = (updateCallback: () => void | Promise<void>) => ViewTransitionLike

/**
 * Progressive enhancement for theme toggles using the View Transitions API.
 * Expands a circular reveal from the pointer/button center and cleans up all
 * inline custom properties once the transition finishes.
 */
export function runThemeTransition(
  apply: () => void,
  origin?: ThemeTransitionOrigin | null,
): void {
  if (typeof window === 'undefined' || typeof document === 'undefined') {
    apply()
    return
  }

  const reducedMotion = window.matchMedia?.('(prefers-reduced-motion: reduce)').matches ?? false
  const startViewTransition = (document as Document & { startViewTransition?: StartViewTransition }).startViewTransition

  if (reducedMotion || typeof startViewTransition !== 'function') {
    apply()
    return
  }

  const root = document.documentElement
  const x = origin ? origin.clientX : window.innerWidth / 2
  const y = origin ? origin.clientY : 32
  const radius = Math.hypot(
    Math.max(x, window.innerWidth - x),
    Math.max(y, window.innerHeight - y),
  )

  root.style.setProperty('--nt-theme-x', `${x}px`)
  root.style.setProperty('--nt-theme-y', `${y}px`)
  root.style.setProperty('--nt-theme-r', `${Math.ceil(radius)}px`)
  root.classList.add('nt-theme-transitioning')

  let transition: ViewTransitionLike
  try {
    transition = startViewTransition.call(document, () => {
      apply()
    })
  } catch {
    root.classList.remove('nt-theme-transitioning')
    root.style.removeProperty('--nt-theme-x')
    root.style.removeProperty('--nt-theme-y')
    root.style.removeProperty('--nt-theme-r')
    apply()
    return
  }

  void transition.finished.finally(() => {
    root.classList.remove('nt-theme-transitioning')
    root.style.removeProperty('--nt-theme-x')
    root.style.removeProperty('--nt-theme-y')
    root.style.removeProperty('--nt-theme-r')
  })
}
