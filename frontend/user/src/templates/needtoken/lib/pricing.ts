/**
 * 模型广场的价格换算与筛选（纯函数，不依赖 Vue / 网络，便于单测）。
 *
 * 数据源是 new-api 的 GET /api/pricing（经店面同源缓存 /api/v1/public/api-pricing 代理）。
 * 换算公式与 new-api 控制台一致（web/src/features/pricing/lib/price.ts）：
 *   输入价（美元 / 百万 tokens）= model_ratio × 2 × 分组倍率
 *   输出价 = 输入价 × completion_ratio
 *   缓存命中价 = 输入价 × cache_ratio（若配置了非负 cache_ratio）
 *   按次计费（quota_type = 1）= model_price × 分组倍率（美元 / 次）
 */

export interface PricingModel {
    model_name: string
    quota_type: number
    model_ratio: number
    model_price: number
    completion_ratio: number
    vendor_id?: number
    owner_by?: string
    enable_groups?: string[]
    tags?: string
    description?: string
    billing_mode?: string
    billing_expr?: string
    billing_plugin_variants?: { plugin_key: string; plugin_name?: string; billing_mode?: string; billing_expr?: string }[]
    billing_usage_schema?: Record<string, { type?: string; unit?: string }>
    supported_endpoint_types?: string[]
    pricing_version?: string
    cache_ratio?: number | null
    create_cache_ratio?: number | null
    image_ratio?: number | null
    audio_ratio?: number | null
    audio_completion_ratio?: number | null
    official_model_ratio?: number | null
    official_completion_ratio?: number | null
    official_model_price?: number | null
}

export interface OfficialPriceOverride {
    input_usd?: number | null
    output_usd?: number | null
    cache_usd?: number | null
    context_length?: string
}

export interface PublicPlanOffer {
    id: number
    title: string
    subtitle?: string
    price_usd: number
    total_usd: number
    multiplier: number
    duration_unit?: string
    duration_value?: number
    allowed_groups?: string[]
    upgrade_group?: string
    allow_wallet_overflow?: boolean
}

export interface NeedtokenPricingMeta {
    featured_models?: string[]
    official_overrides?: Record<string, OfficialPriceOverride>
    active_plans?: PublicPlanOffer[]
}

export interface PricingVendor {
    id: number
    name: string
    icon?: string
    description?: string
}

export interface PricingEndpointMeta {
    path?: string
    method?: string
}

export interface PricingResponse {
    success?: boolean
    data?: PricingModel[]
    vendors?: PricingVendor[]
    group_ratio?: Record<string, number>
    usable_group?: Record<string, string>
    supported_endpoint?: Record<string, PricingEndpointMeta>
    pricing_version?: string
    needtoken_meta?: NeedtokenPricingMeta
}

/**
 * new-api 对未配置倍率的模型返回 37.5（GetModelRatio 的兜底值，自用模式下照样放行）。
 * 接口本身分不出“配了 37.5”和“没配”，这里一律按未配价处理，宁可不展示也不写假价。
 */
export const UNCONFIGURED_MODEL_RATIO = 37.5

export type PriceUnit = 'per_million_tokens' | 'per_call' | 'conditional'
export type PricingCondition = 'expression' | 'plugin' | 'task' | 'multimodal' | 'unknown'

export interface EndpointBadge {
    key: string
    label: string
    path: string
}

export interface PricingGroupOption {
    name: string
    ratio: number
    description: string
    supplyLine: string | null
}

export interface ModelOffer {
    /** 唯一键：模型 + 分组 */
    key: string
    modelId: string
    vendor: string
    group: string
    groupRatio: number
    groupDescription: string
    /** 供应线：已登记的分组用登记值，否则为 null（界面上显示分组名） */
    supplyLine: string | null
    quotaType: number
    unit: PriceUnit
    /** 美元；per_call 时 output 为 null */
    inputPrice: number | null
    outputPrice: number | null
    /** 美元 / 百万 tokens；仅当 quota_type=0 且配置了非负 cache_ratio 时有值 */
    cachePrice: number | null
    /** 官方基准参考价（美元 / 百万 tokens，或 per_call）；无参考价时为 null */
    officialInputPrice: number | null
    officialOutputPrice: number | null
    officialCachePrice: number | null
    /** 相比官方基准价的折扣百分比（如 70 表示 70% off / 官方 3 折）；无折扣或无官方价时为 null */
    discountPercent: number | null
    /** 上下文长度展示（优先取 Ops 配置，其次按常见模型族推断） */
    contextLength: string | null
    isDefault: boolean
    condition: PricingCondition | null
    /** Backend cache/schema marker; never treated as an immutable checkout quote. */
    sourceVersion: string | null
    usageUnits: string[]
    endpoints: EndpointBadge[]
    tags: string[]
    description: string
}

export interface BuildOffersOptions {
    defaultModels: readonly string[]
    blockedModels: readonly string[]
    supplyLineByGroup: Record<string, string>
}

const KNOWN_OWNER_BRANDS: Record<string, string> = {
    openai: 'OpenAI',
    anthropic: 'Anthropic',
    claude: 'Anthropic',
    google: 'Google',
    gemini: 'Google',
    vertex: 'Google',
    'vertex-ai': 'Google',
    xai: 'xAI',
    deepseek: 'DeepSeek',
    qwen: 'Qwen',
    alibaba: 'Qwen',
    aliyun: 'Qwen',
    zhipu: 'Zhipu',
    glm: 'Zhipu',
    moonshot: 'Moonshot',
    kimi: 'Moonshot',
    meta: 'Meta',
    mistral: 'Mistral',
    cohere: 'Cohere',
    perplexity: 'Perplexity',
}

const MODEL_PREFIX_VENDORS: readonly [RegExp, string][] = [
    [/^(?:gpt-|o1|o3|o4|chatgpt|codex|text-embedding|dall-e|whisper|tts-)/i, 'OpenAI'],
    [/^claude/i, 'Anthropic'],
    [/^(?:gemini|gemma|imagen|veo)/i, 'Google'],
    [/^grok/i, 'xAI'],
    [/^deepseek/i, 'DeepSeek'],
    [/^(?:qwen|qwq)/i, 'Qwen'],
    [/^glm/i, 'Zhipu'],
    [/^(?:kimi|moonshot)/i, 'Moonshot'],
    [/^llama/i, 'Meta'],
    [/^(?:mistral|codestral|pixtral)/i, 'Mistral'],
]

const ENDPOINT_LABELS: Record<string, { label: string; defaultPath: string }> = {
    openai: { label: 'Chat', defaultPath: '/v1/chat/completions' },
    'openai-response': { label: 'Responses', defaultPath: '/v1/responses' },
    'openai-response-compact': { label: 'Responses', defaultPath: '/v1/responses/compact' },
    anthropic: { label: 'Messages', defaultPath: '/v1/messages' },
    gemini: { label: 'Gemini', defaultPath: '/v1beta/models' },
    embeddings: { label: 'Embeddings', defaultPath: '/v1/embeddings' },
    'image-generation': { label: 'Image', defaultPath: '/v1/images/generations' },
    'openai-video': { label: 'Video', defaultPath: '/v1/videos' },
    rerank: { label: 'Rerank', defaultPath: '/v1/rerank' },
}

export const pricingCondition = (model: PricingModel): PricingCondition | null => {
    if (model.billing_plugin_variants?.length) return 'plugin'
    if (String(model.billing_expr || '').trim() || (model.billing_mode && model.billing_mode !== 'ratio')) return 'expression'
    if (Object.keys(model.billing_usage_schema || {}).length || model.supported_endpoint_types?.some(type => ['openai-video', 'image-generation'].includes(type))) return 'task'
    if ([model.image_ratio, model.audio_ratio, model.audio_completion_ratio].some(value => value !== undefined && value !== null)) return 'multimodal'
    if (model.quota_type !== 0 && model.quota_type !== 1) return 'unknown'
    return null
}

export const isModelPriced = (model: PricingModel): boolean => {
    // Never interpret expression/plugin/task prices through the legacy token formula.
    if (pricingCondition(model)) return true
    if (model.quota_type === 1) {
        return Number.isFinite(model.model_price) && model.model_price > 0
    }
    return (
        Number.isFinite(model.model_ratio)
        && model.model_ratio > 0
        && model.model_ratio !== UNCONFIGURED_MODEL_RATIO
        && Number.isFinite(model.completion_ratio) && model.completion_ratio >= 0
    )
}

export const inferVendorFromModelName = (modelName: string): string | null => {
    const trimmed = String(modelName || '').trim()
    if (!trimmed) return null
    const slashIdx = trimmed.indexOf('/')
    if (slashIdx > 0) {
        const prefix = trimmed.slice(0, slashIdx).toLowerCase()
        if (KNOWN_OWNER_BRANDS[prefix]) return KNOWN_OWNER_BRANDS[prefix]!
    }
    const baseName = slashIdx >= 0 ? trimmed.slice(slashIdx + 1) : trimmed
    for (const [pattern, brand] of MODEL_PREFIX_VENDORS) {
        if (pattern.test(baseName)) return brand
    }
    return null
}

const resolveVendor = (model: PricingModel, vendors: Map<number, string>): string => {
    if (model.vendor_id && vendors.has(model.vendor_id)) {
        const mapped = vendors.get(model.vendor_id)
        if (mapped) return mapped
    }
    const owner = String(model.owner_by || '').trim()
    if (owner) {
        const canonical = KNOWN_OWNER_BRANDS[owner.toLowerCase()]
        return canonical || owner
    }
    return inferVendorFromModelName(model.model_name) || 'other'
}

export const parseModelTags = (raw?: string | null): string[] => {
    if (!raw || typeof raw !== 'string') return []
    const seen = new Set<string>()
    const out: string[] = []
    for (const part of raw.split(/[,，;|]/)) {
        const trimmed = part.trim()
        if (!trimmed || seen.has(trimmed)) continue
        seen.add(trimmed)
        out.push(trimmed)
    }
    return out
}

export const resolveEndpointBadges = (
    endpointTypes?: string[] | null,
    supportedEndpointMap?: Record<string, PricingEndpointMeta> | null,
): EndpointBadge[] => {
    if (!Array.isArray(endpointTypes) || endpointTypes.length === 0) return []
    const seenLabels = new Set<string>()
    const badges: EndpointBadge[] = []
    for (const rawKey of endpointTypes) {
        const key = String(rawKey || '').trim()
        if (!key) continue
        const preset = ENDPOINT_LABELS[key]
        const label = preset?.label || key
        if (seenLabels.has(label)) continue
        seenLabels.add(label)
        const remotePath = supportedEndpointMap?.[key]?.path
        badges.push({
            key,
            label,
            path: typeof remotePath === 'string' && remotePath.trim() ? remotePath.trim() : (preset?.defaultPath || ''),
        })
    }
    return badges
}

/** 提取有效计费分组列表（含 usable_group 说明与 group_ratio 倍率） */
export const buildPricingGroups = (
    response: PricingResponse,
    supplyLineByGroup: Record<string, string> = {},
): PricingGroupOption[] => {
    const groupRatio = response.group_ratio || {}
    const usableGroup = response.usable_group || {}
    const names = new Set<string>([
        ...Object.keys(groupRatio),
        ...Object.keys(usableGroup),
    ])
    const out: PricingGroupOption[] = []
    for (const name of names) {
        if (!name || name === 'auto') {
            // Keep explicit numeric groups; include auto only if it has a finite nonnegative ratio.
            if (!(name === 'auto' && Number.isFinite(groupRatio[name]) && groupRatio[name]! >= 0)) continue
        }
        const ratio = groupRatio[name]
        if (typeof ratio !== 'number' || !Number.isFinite(ratio) || ratio < 0) continue
        out.push({
            name,
            ratio,
            description: typeof usableGroup[name] === 'string' ? usableGroup[name]!.trim() : '',
            supplyLine: supplyLineByGroup[name] ?? null,
        })
    }
    return out.sort((a, b) => {
        if (a.name === 'default' && b.name !== 'default') return -1
        if (b.name === 'default' && a.name !== 'default') return 1
        return a.ratio - b.ratio || a.name.localeCompare(b.name)
    })
}

/**
 * 把 new-api 定价表展开成“模型 × 分组”的报价列表。
 * 同一模型在不同分组（供应线）价格不同，分开列出、不混算。
 */
const inferContextLength = (modelName: string): string | null => {
    const n = String(modelName || '').toLowerCase()
    if (!n) return null
    if (/gemini-(?:2\.|3\.|1\.5|pro|flash)/.test(n)) return '1M'
    if (/claude-(?:opus-4|sonnet-4|haiku-4|3\.7|3\.5)/.test(n)) return '200K'
    if (/gpt-(?:5|4\.1|4\.5)/.test(n)) return '1M'
    if (/gpt-4o|o1|o3|o4|deepseek|grok/.test(n)) return '128K'
    return null
}

export const effectiveDiscountPercent = (officialPrice: number | null, actualPrice: number | null): number | null => {
    if (
        officialPrice === null || actualPrice === null
        || !Number.isFinite(officialPrice) || !Number.isFinite(actualPrice)
        || officialPrice <= 0 || actualPrice < 0 || actualPrice >= officialPrice
    ) {
        return null
    }
    const pct = Math.round((1 - actualPrice / officialPrice) * 100)
    return pct >= 1 && pct <= 99 ? pct : null
}

export const effectivePlanOfferPrice = (price: number | null, multiplier: number): number | null => {
    if (price === null || !Number.isFinite(price) || price < 0) return null
    if (!Number.isFinite(multiplier) || multiplier <= 1) return price
    return price / multiplier
}

export const buildOffers = (response: PricingResponse, options: BuildOffersOptions): ModelOffer[] => {
    const models = Array.isArray(response.data) ? response.data : []
    const groupRatio = response.group_ratio || {}
    const usableGroup = response.usable_group || {}
    const officialOverrides = response.needtoken_meta?.official_overrides || {}
    const vendors = new Map<number, string>()
    for (const v of response.vendors || []) {
        if (v && typeof v.id === 'number') {
            const name = String(v.name || '').trim()
            if (name) vendors.set(v.id, name)
        }
    }
    const blocked = new Set(options.blockedModels)
    const defaults = new Set(options.defaultModels)

    const offers: ModelOffer[] = []
    for (const model of models) {
        if (!model || !model.model_name || blocked.has(model.model_name)) continue
        if (!isModelPriced(model)) continue
        const condition = pricingCondition(model)
        const enabled = model.enable_groups?.includes('all') ? Object.keys(groupRatio) : (model.enable_groups || [])
        const groups = [...new Set(enabled)].filter(g => Number.isFinite(groupRatio[g]) && groupRatio[g]! >= 0)
        const endpoints = resolveEndpointBadges(model.supported_endpoint_types, response.supported_endpoint)
        const tags = parseModelTags(model.tags)
        const description = typeof model.description === 'string' ? model.description.trim() : ''
        const vendor = resolveVendor(model, vendors)
        const override = officialOverrides[model.model_name]
        const contextLength = (override?.context_length && override.context_length.trim()) || inferContextLength(model.model_name)

        for (const group of groups) {
            const ratio = groupRatio[group] as number
            const perCall = model.quota_type === 1
            const input = condition ? null : perCall ? model.model_price * ratio : model.model_ratio * 2 * ratio
            const output = condition || perCall || input === null ? null : input * model.completion_ratio
            const valid = input !== null && Number.isFinite(input) && (output === null || Number.isFinite(output))
            const rawCacheRatio = model.cache_ratio
            const cachePrice = (
                valid
                && !perCall
                && input !== null
                && typeof rawCacheRatio === 'number'
                && Number.isFinite(rawCacheRatio)
                && rawCacheRatio >= 0
                && Number.isFinite(input * rawCacheRatio)
            )
                ? input * rawCacheRatio
                : null

            // Official reference price resolution priority:
            // 1. Explicit Ops override (official_overrides[model_name])
            // 2. Built-in official_* ratio/price emitted by new-api /api/pricing
            // 3. Fallback to model's own 1.0x baseline when group_ratio < 1 (discount group)
            let officialInput: number | null = null
            let officialOutput: number | null = null
            let officialCache: number | null = null
            if (!condition && valid) {
                if (typeof override?.input_usd === 'number' && Number.isFinite(override.input_usd) && override.input_usd > 0) {
                    officialInput = override.input_usd
                } else if (perCall && typeof model.official_model_price === 'number' && Number.isFinite(model.official_model_price) && model.official_model_price > 0) {
                    officialInput = model.official_model_price
                } else if (!perCall && typeof model.official_model_ratio === 'number' && Number.isFinite(model.official_model_ratio) && model.official_model_ratio > 0) {
                    officialInput = model.official_model_ratio * 2
                } else if (ratio > 0 && ratio < 1) {
                    officialInput = perCall ? model.model_price : model.model_ratio * 2
                }

                if (!perCall) {
                    if (typeof override?.output_usd === 'number' && Number.isFinite(override.output_usd) && override.output_usd > 0) {
                        officialOutput = override.output_usd
                    } else if (officialInput !== null) {
                        const offComp = (typeof model.official_completion_ratio === 'number' && Number.isFinite(model.official_completion_ratio) && model.official_completion_ratio > 0)
                            ? model.official_completion_ratio
                            : model.completion_ratio
                        officialOutput = officialInput * offComp
                    }
                    if (typeof override?.cache_usd === 'number' && Number.isFinite(override.cache_usd) && override.cache_usd >= 0) {
                        officialCache = override.cache_usd
                    } else if (officialInput !== null && typeof rawCacheRatio === 'number' && Number.isFinite(rawCacheRatio) && rawCacheRatio >= 0) {
                        officialCache = officialInput * rawCacheRatio
                    }
                }
            }
            const discountPercent = effectiveDiscountPercent(officialInput, valid ? input : null)

            offers.push({
                key: `${model.model_name}@${group}`,
                modelId: model.model_name,
                vendor,
                group,
                groupRatio: ratio,
                groupDescription: typeof usableGroup[group] === 'string' ? usableGroup[group]!.trim() : '',
                supplyLine: options.supplyLineByGroup[group] ?? null,
                quotaType: model.quota_type,
                unit: valid ? perCall ? 'per_call' : 'per_million_tokens' : 'conditional',
                inputPrice: valid ? input : null,
                outputPrice: valid ? output : null,
                cachePrice,
                officialInputPrice: officialInput,
                officialOutputPrice: officialOutput,
                officialCachePrice: officialCache,
                discountPercent,
                contextLength,
                isDefault: defaults.has(model.model_name),
                condition: condition ?? (valid ? null : 'unknown'),
                sourceVersion: model.pricing_version || response.pricing_version || null,
                usageUnits: [...new Set(Object.entries(model.billing_usage_schema || {}).map(([key, field]) => `${key}${field.unit ? ` (${field.unit})` : ''}`))],
                endpoints,
                tags,
                description,
            })
        }
    }
    return offers
}

/**
 * 当用户选择“全部分组”视图时，同一模型保留一个首选分组报价（优先 default，其次低倍率），
 * 避免同一模型在多分组下重复渲染；当选中特定分组时，返回该分组的全部报价。
 */
export const selectOffersByGroup = (offers: ModelOffer[], selectedGroup: string): ModelOffer[] => {
    const trimmedGroup = String(selectedGroup || '').trim()
    if (trimmedGroup && trimmedGroup !== '__all_offers__') {
        return offers.filter(o => o.group === trimmedGroup)
    }
    if (trimmedGroup === '__all_offers__') {
        return offers
    }
    const bestByModel = new Map<string, ModelOffer>()
    for (const offer of offers) {
        const current = bestByModel.get(offer.modelId)
        if (!current) {
            bestByModel.set(offer.modelId, offer)
            continue
        }
        const currentIsDefault = current.group === 'default'
        const candidateIsDefault = offer.group === 'default'
        if (candidateIsDefault && !currentIsDefault) {
            bestByModel.set(offer.modelId, offer)
            continue
        }
        if (!currentIsDefault && offer.groupRatio < current.groupRatio) {
            bestByModel.set(offer.modelId, offer)
        }
    }
    return [...bestByModel.values()]
}

export interface OfferFilter {
    vendor?: string
    /** 供应线值；未登记分组用 `group:<分组名>` 表示 */
    supply?: string
    group?: string
    billingType?: '' | 'token' | 'per_call'
    keyword?: string
}

export const supplyKeyOf = (offer: ModelOffer): string => offer.supplyLine ?? `group:${offer.group}`

export const filterOffers = (offers: ModelOffer[], filter: OfferFilter): ModelOffer[] => {
    const keyword = String(filter.keyword || '').trim().toLowerCase()
    return offers.filter(o => {
        if (filter.vendor && o.vendor !== filter.vendor) return false
        if (filter.supply && supplyKeyOf(o) !== filter.supply) return false
        if (filter.group && o.group !== filter.group) return false
        if (filter.billingType === 'token' && o.unit !== 'per_million_tokens') return false
        if (filter.billingType === 'per_call' && o.unit !== 'per_call') return false
        if (!keyword) return true
        return (
            o.modelId.toLowerCase().includes(keyword)
            || o.vendor.toLowerCase().includes(keyword)
            || o.description.toLowerCase().includes(keyword)
            || o.tags.some(tag => tag.toLowerCase().includes(keyword))
        )
    })
}

export type OfferSort = 'default' | 'price_asc' | 'price_desc'

/** 默认排序：默认模型按白名单顺序在前，其余按模型名；价格排序只比较同计价单位，按次计费排在最后 */
export const sortOffers = (offers: ModelOffer[], sort: OfferSort, defaultModels: readonly string[]): ModelOffer[] => {
    const list = [...offers]
    if (sort === 'default') {
        const rank = (o: ModelOffer) => {
            const i = defaultModels.indexOf(o.modelId)
            return i === -1 ? defaultModels.length : i
        }
        return list.sort((a, b) => rank(a) - rank(b) || a.modelId.localeCompare(b.modelId) || a.group.localeCompare(b.group))
    }
    const dir = sort === 'price_asc' ? 1 : -1
    return list.sort((a, b) => {
        const units: Record<PriceUnit, number> = { per_million_tokens: 0, per_call: 1, conditional: 2 }
        if (a.unit !== b.unit) return units[a.unit] - units[b.unit]
        if (a.inputPrice === null || b.inputPrice === null) return a.modelId.localeCompare(b.modelId)
        return (a.inputPrice - b.inputPrice) * dir || a.modelId.localeCompare(b.modelId)
    })
}

export const MAX_COMPARE = 3

/** 比价选择：已选则取消；未选且未满则加入；满了不变 */
export const toggleCompare = (selected: string[], key: string): string[] => {
    if (selected.includes(key)) return selected.filter(k => k !== key)
    if (selected.length >= MAX_COMPARE) return selected
    return [...selected, key]
}

/** 美元价格显示：去掉多余的 0，最多 6 位小数 */
export const formatUsd = (value: number | null): string => {
    if (value === null || !Number.isFinite(value) || value < 0) return '-'
    if (value > 0 && value < 0.000001) return '<$0.000001'
    const fixed = value.toFixed(6).replace(/\.?0+$/, '')
    return `$${fixed}`
}
