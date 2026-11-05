export interface GhostFibersDeviceHints {
  reducedMotion?: boolean
  saveData?: boolean
  hardwareConcurrency?: number
  deviceMemory?: number
  coarsePointer?: boolean
  subtle?: boolean
}

/** Keep the Prism artwork; adapt the rendering budget to this device only. */
export function ghostFibersPolicy(hints: GhostFibersDeviceHints) {
  const limitedCPU = typeof hints.hardwareConcurrency === 'number' && hints.hardwareConcurrency > 0 && hints.hardwareConcurrency <= 4
  const limitedMemory = typeof hints.deviceMemory === 'number' && hints.deviceMemory > 0 && hints.deviceMemory <= 4
  const staticOnly = Boolean(hints.reducedMotion || hints.saveData || limitedCPU || limitedMemory)
  const modestDevice = Boolean(hints.coarsePointer || (hints.deviceMemory && hints.deviceMemory <= 8) || (hints.hardwareConcurrency && hints.hardwareConcurrency <= 8))
  return {
    animate: !staticOnly,
    maxDimension: hints.subtle ? 1100 : modestDevice ? 1200 : 1600,
    maxFps: hints.subtle ? 18 : modestDevice ? 24 : 30,
    dpr: modestDevice || hints.subtle ? 1 : 1.25,
  }
}
