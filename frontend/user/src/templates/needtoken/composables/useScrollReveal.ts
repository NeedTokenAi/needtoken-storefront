import { onMounted, onUnmounted, type Ref } from 'vue'

/**
 * Progressive fallback for `.nt-scroll-reveal` sections when CSS Scroll-Driven
 * Animations (`animation-timeline: view()` + `animation-range: entry`) are not
 * supported. Elements remain visible by default; only below-the-fold items are
 * gently revealed on intersection so content is never stuck hidden.
 */
export function useScrollReveal(rootRef: Ref<HTMLElement | null>) {
  let observer: IntersectionObserver | null = null

  onMounted(() => {
    if (typeof window === 'undefined' || !rootRef.value) return
    if (window.matchMedia?.('(prefers-reduced-motion: reduce)').matches) return
    if (typeof CSS !== 'undefined' && CSS.supports?.('(animation-timeline: view()) and (animation-range: entry)')) {
      return
    }
    if (typeof IntersectionObserver === 'undefined') return

    const targets = [...rootRef.value.querySelectorAll<HTMLElement>('.nt-scroll-reveal')]
    if (!targets.length) return

    const vh = window.innerHeight || 800
    const pending: HTMLElement[] = []
    for (const el of targets) {
      const top = el.getBoundingClientRect().top
      if (top > vh * 0.92) {
        el.classList.add('nt-scroll-reveal--pending')
        pending.push(el)
      }
    }
    if (!pending.length) return

    observer = new IntersectionObserver(
      (entries) => {
        for (const entry of entries) {
          if (!entry.isIntersecting) continue
          const target = entry.target as HTMLElement
          target.classList.remove('nt-scroll-reveal--pending')
          target.classList.add('nt-scroll-reveal--shown')
          observer?.unobserve(target)
        }
      },
      { rootMargin: '0px 0px -8% 0px', threshold: 0.06 },
    )

    for (const el of pending) observer.observe(el)
  })

  onUnmounted(() => {
    observer?.disconnect()
    observer = null
  })
}
