import { computed, ref } from 'vue'
import { BLOCKED_MODELS, DEFAULT_MODELS, PRICING_URL, SUPPLY_LINE_BY_GROUP } from '../config'
import {
    buildOffers,
    buildPricingGroups,
    type ModelOffer,
    type NeedtokenPricingMeta,
    type PricingGroupOption,
    type PricingResponse,
    type PublicPlanOffer,
} from './pricing'

type LoadState = 'idle' | 'loading' | 'ready' | 'error'

// 页面间共享一份：首页定价摘要、API 服务页与模型广场不重复请求
const offers = ref<ModelOffer[]>([])
const groups = ref<PricingGroupOption[]>([])
const state = ref<LoadState>('idle')
const loadedAt = ref<Date | null>(null)
const sourceVersion = ref<string | null>(null)
const isStale = ref(false)
const response = ref<PricingResponse | null>(null)
const meta = ref<NeedtokenPricingMeta | null>(null)
let inflight: Promise<void> | null = null

const load = async (): Promise<void> => {
    state.value = 'loading'
    try {
        const res = await fetch(PRICING_URL, { headers: { Accept: 'application/json' }, credentials: 'omit' })
        if (!res.ok) throw new Error(`pricing http ${res.status}`)
        const staleHeader = res.headers.get('X-Storefront-Stale') === '1'
        const body = (await res.json()) as PricingResponse
        if (body.success === false) throw new Error('pricing not available')
        response.value = body
        meta.value = body.needtoken_meta || null
        offers.value = buildOffers(body, {
            defaultModels: DEFAULT_MODELS,
            blockedModels: BLOCKED_MODELS,
            supplyLineByGroup: SUPPLY_LINE_BY_GROUP,
        })
        groups.value = buildPricingGroups(body, SUPPLY_LINE_BY_GROUP)
        loadedAt.value = new Date()
        sourceVersion.value = body.pricing_version || null
        isStale.value = staleHeader
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
    const featuredModels = computed<string[]>(() => {
        const configured = meta.value?.featured_models?.filter(m => typeof m === 'string' && m.trim()) || []
        return configured.length > 0 ? configured : [...DEFAULT_MODELS]
    })
    const activePlans = computed<PublicPlanOffer[]>(() => {
        const list = meta.value?.active_plans
        return Array.isArray(list) ? list : []
    })
    return {
        offers,
        groups,
        state,
        loadedAt,
        sourceVersion,
        isStale,
        response,
        meta,
        featuredModels,
        activePlans,
        isLoading: computed(() => state.value === 'loading' || state.value === 'idle'),
        ensureLoaded,
        reload: () => { if (!inflight) inflight = load().finally(() => { inflight = null }); return inflight },
    }
}
