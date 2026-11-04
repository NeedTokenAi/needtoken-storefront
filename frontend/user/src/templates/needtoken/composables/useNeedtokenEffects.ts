import { onMounted, onUnmounted, ref, type Ref } from 'vue'

const SOUND_PREFERENCE_KEY = 'needtoken-ui-sound'
const ACTION_SELECTOR = 'button, a[href], summary, [role="button"], [role="tab"], input[type="button"], input[type="submit"], input[type="reset"]'

/** One delegated listener per storefront, rather than a listener/audio file per button. */
export function useNeedtokenEffects(scope: Ref<HTMLElement | null>) {
  const soundEnabled = ref(true)
  let audio: AudioContext | undefined
  let mounted = false
  let lastTap = -Infinity
  let activeControl: HTMLElement | null = null
  let hoverControl: HTMLElement | null = null
  let focusControl: HTMLElement | null = null
  let pointerFrame = 0
  let pointerX = 0
  let pointerY = 0
  let listenerRoot: Document | null = null

  const actionFor = (target: EventTarget | null): HTMLElement | null => {
    const element = target instanceof Element ? target.closest<HTMLElement>(ACTION_SELECTOR) : null
    if (!element || !(scope.value?.contains(element) || document.body.classList.contains('nt-tokens'))) return null
    if (element.matches(':disabled') || element.closest('[inert], [aria-disabled="true"], [data-nt-effect="off"]')) return null
    return element
  }

  const removeHighlight = () => {
    activeControl?.classList.remove('nt-specular', 'nt-specular-position')
    activeControl?.style.removeProperty('--nt-pointer-x')
    activeControl?.style.removeProperty('--nt-pointer-y')
    activeControl = null
  }

  const highlight = (control: HTMLElement | null) => {
    if (control === activeControl) return
    removeHighlight()
    if (!control || control.matches('input')) return
    const style = getComputedStyle(control)
    // Plain editorial links keep their text treatment; padded link buttons get the same sheen.
    if (control.matches('a') && parseFloat(style.paddingTop) === 0 && parseFloat(style.paddingLeft) === 0) return
    // Do not replace a native widget's existing pseudo element.
    const existingContent = getComputedStyle(control, '::after').content
    if (existingContent && !['none', 'normal'].includes(existingContent)) return
    activeControl = control
    control.classList.add('nt-specular')
    if (style.position === 'static') control.classList.add('nt-specular-position')
    control.style.setProperty('--nt-pointer-x', '50%')
    control.style.setProperty('--nt-pointer-y', '0%')
  }

  const updatePointer = () => {
    pointerFrame = 0
    if (!activeControl || activeControl !== hoverControl) return
    const bounds = activeControl.getBoundingClientRect()
    activeControl.style.setProperty('--nt-pointer-x', `${Math.max(0, Math.min(bounds.width, pointerX - bounds.left))}px`)
    activeControl.style.setProperty('--nt-pointer-y', `${Math.max(0, Math.min(bounds.height, pointerY - bounds.top))}px`)
  }

  const onPointerMove = (event: PointerEvent) => {
    if (event.pointerType === 'touch') return
    pointerX = event.clientX
    pointerY = event.clientY
    hoverControl = actionFor(event.target)
    highlight(hoverControl || focusControl)
    if (activeControl === hoverControl && hoverControl && !pointerFrame) pointerFrame = window.requestAnimationFrame(updatePointer)
  }
  const onPointerLeave = () => {
    hoverControl = null
    highlight(focusControl)
  }
  const onFocusIn = (event: FocusEvent) => {
    focusControl = actionFor(event.target)
    highlight(hoverControl || focusControl)
  }
  const onFocusOut = (event: FocusEvent) => {
    focusControl = actionFor(event.relatedTarget)
    highlight(hoverControl || focusControl)
  }

  const playTap = () => {
    if (!mounted || !soundEnabled.value || document.hidden) return
    const now = performance.now()
    if (now - lastTap < 55) return
    lastTap = now
    try {
      // Created only inside a trusted click. No autoplay, downloads, microphone access, or audio library.
      const AudioContextClass = window.AudioContext || (window as typeof window & { webkitAudioContext?: typeof AudioContext }).webkitAudioContext
      if (!AudioContextClass) return
      audio ||= new AudioContextClass()
      const context = audio
      const ring = () => {
        if (!mounted || !soundEnabled.value || document.hidden || context.state !== 'running') return
        const start = context.currentTime
        const master = context.createGain()
        master.gain.value = 0.35
        master.connect(context.destination)
        let remaining = 2
        ;[2200, 3360].forEach((frequency, index) => {
          const oscillator = context.createOscillator()
          const envelope = context.createGain()
          oscillator.type = 'sine'
          oscillator.frequency.setValueAtTime(frequency, start)
          oscillator.frequency.exponentialRampToValueAtTime(frequency * 0.82, start + 0.08)
          envelope.gain.setValueAtTime(0.0001, start)
          envelope.gain.exponentialRampToValueAtTime(index ? 0.025 : 0.045, start + 0.002)
          envelope.gain.exponentialRampToValueAtTime(0.0001, start + 0.085)
          oscillator.connect(envelope)
          envelope.connect(master)
          oscillator.onended = () => {
            oscillator.disconnect()
            envelope.disconnect()
            if (--remaining === 0) master.disconnect()
          }
          oscillator.start(start)
          oscillator.stop(start + 0.09)
        })
      }
      if (context.state === 'suspended') void context.resume().then(ring).catch(() => {})
      else ring()
    } catch {
      // Restricted browsers and unavailable audio devices retain fully functional buttons.
    }
  }
  const onClick = (event: MouseEvent) => {
    const control = actionFor(event.target)
    if (event.isTrusted && control && !control.hasAttribute('data-nt-sound-toggle')) playTap()
  }
  const onVisibility = () => {
    if (document.hidden && audio?.state === 'running') void audio.suspend().catch(() => {})
    if (document.hidden) {
      hoverControl = null
      focusControl = null
      removeHighlight()
    }
  }
  const toggleSound = () => {
    soundEnabled.value = !soundEnabled.value
    try { localStorage.setItem(SOUND_PREFERENCE_KEY, soundEnabled.value ? 'on' : 'off') } catch { /* Session-only preference is sufficient in private browsing. */ }
    if (!soundEnabled.value && audio?.state === 'running') void audio.suspend().catch(() => {})
  }
  const onStorage = (event: StorageEvent) => {
    if (event.key === SOUND_PREFERENCE_KEY) soundEnabled.value = event.newValue !== 'off'
  }

  onMounted(() => {
    mounted = true
    try { soundEnabled.value = localStorage.getItem(SOUND_PREFERENCE_KEY) !== 'off' } catch { /* Keep the session default. */ }
    // The layout owns body.nt-tokens, including native dialogs teleported to body.
    listenerRoot = document
    listenerRoot?.addEventListener('click', onClick, { capture: true })
    listenerRoot?.addEventListener('pointermove', onPointerMove, { passive: true })
    listenerRoot?.addEventListener('pointerleave', onPointerLeave)
    listenerRoot?.addEventListener('focusin', onFocusIn)
    listenerRoot?.addEventListener('focusout', onFocusOut)
    document.addEventListener('visibilitychange', onVisibility)
    window.addEventListener('storage', onStorage)
  })
  onUnmounted(() => {
    mounted = false
    listenerRoot?.removeEventListener('click', onClick, true)
    listenerRoot?.removeEventListener('pointermove', onPointerMove)
    listenerRoot?.removeEventListener('pointerleave', onPointerLeave)
    listenerRoot?.removeEventListener('focusin', onFocusIn)
    listenerRoot?.removeEventListener('focusout', onFocusOut)
    document.removeEventListener('visibilitychange', onVisibility)
    window.removeEventListener('storage', onStorage)
    if (pointerFrame) window.cancelAnimationFrame(pointerFrame)
    removeHighlight()
    if (audio && audio.state !== 'closed') void audio.close().catch(() => {})
  })

  return { soundEnabled, toggleSound }
}
