import { computed, ref } from 'vue'
import { BLOCKED_MODELS, DEFAULT_MODELS, PRICING_URL, SUPPLY_LINE_BY_GROUP } from '../config'
import { buildOffers, type ModelOffer, type PricingResponse } from './pricing'

type LoadState = 'idle' | 'loading' | 'ready' | 'error'

// 页面间共享一份：首页定价摘要与模型广场不重复请求
const offers = ref<ModelOffer[]>([])
const state = ref<LoadState>('idle')
const loadedAt = ref<Date | null>(null)
const sourceVersion = ref<string | null>(null)
const response = ref<PricingResponse | null>(null)
let inflight: Promise<void> | null = null

const load = async (): Promise<void> => {
    state.value = 'loading'
    try {
        const res = await fetch(PRICING_URL, { headers: { Accept: 'application/json' }, credentials: 'omit' })
        if (!res.ok) throw new Error(`pricing http ${res.status}`)
        const body = (await res.json()) as PricingResponse
        if (body.success === false) throw new Error('pricing not available')
        response.value = body
        offers.value = buildOffers(body, {
            defaultModels: DEFAULT_MODELS,
            blockedModels: BLOCKED_MODELS,
            supplyLineByGroup: SUPPLY_LINE_BY_GROUP,
        })
        loadedAt.value = new Date()
        sourceVersion.value = body.pricing_version || null
        state.value = 'ready'
    } catch {
        // 接口不可用时保留上次已知数据（若有），界面按 error 提示“以控制台为准”
        state.value = 'error'
    }
}

export const usePricing = () => {
    const ensureLoaded = () => {
        if (state.value === 'ready' || inflight) return inflight ?? Promise.resolve()
        inflight = load().finally(() => { inflight = null })
        return inflight
    }
    return {
        offers,
        state,
        loadedAt,
        sourceVersion,
        response,
        isLoading: computed(() => state.value === 'loading' || state.value === 'idle'),
        ensureLoaded,
        reload: () => { if (!inflight) inflight = load().finally(() => { inflight = null }); return inflight },
    }
}
