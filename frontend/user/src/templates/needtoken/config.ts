/**
 * needtoken 模板的站点常量。
 *
 * 只放公开地址与展示规则，不放任何密钥。地址可用 VITE_ 环境变量在构建期覆盖。
 */

const env = import.meta.env as Record<string, string | undefined>

const trimSlash = (value: string) => value.replace(/\/+$/, '')

/** NeedToken 主站品牌。分销站点继续使用各自的品牌配置。 */
export const NEEDTOKEN_BRAND_NAME = 'NeedToken.AI'
export const NEEDTOKEN_BRAND_ICON = '/needtoken-icon.svg'
export const NEEDTOKEN_TOUCH_ICON = '/needtoken-touch.png'

/** new-api 控制台（会员注册 / 登录 / key / 用量） */
export const CONSOLE_URL = trimSlash(env.VITE_NEEDTOKEN_CONSOLE_URL || 'https://api.needtoken.ai')

/** 对外唯一 API 入口 */
export const API_BASE_URL = `${CONSOLE_URL}/v1`

/** API 服务介绍与实际套餐目录分开；/api 是后端保留路径。 */
export const API_SERVICES_PATH = '/api-services'
export const API_PLANS_PATH = '/plans'

/** 店面同源公开定价缓存接口（可通过 VITE_NEEDTOKEN_PRICING_URL 覆盖） */
export const PRICING_URL = trimSlash(env.VITE_NEEDTOKEN_PRICING_URL || '/api/v1/public/api-pricing')

/** 店面同源公开号池状态缓存接口（可通过 VITE_NEEDTOKEN_STATUS_URL 覆盖） */
export const STATUS_URL = trimSlash(env.VITE_NEEDTOKEN_STATUS_URL || '/api/v1/public/pool-status')

export const CONSOLE_LOGIN_URL = `${CONSOLE_URL}/sign-in`
export const CONSOLE_REGISTER_URL = `${CONSOLE_URL}/sign-up`

/** 独角商品分类 slug：在独角后台建同名分类即可对应菜单 */
export const CATEGORY_SLUGS = {
    recharge: 'recharge',
    rental: 'rental',
    plans: 'plans',
} as const

/** 默认主流推荐模型，首个为示例与排序首选 */
export const DEFAULT_MODELS = [
    'gpt-5.4',
    'claude-sonnet-4-6',
    'claude-opus-4-6',
    'gemini-2.5-pro',
    'deepseek-v3',
    'deepseek-r1',
    'grok-4',
    'gpt-4.1',
] as const

/** 上游已拒绝、不再使用的模型，即使 new-api 返回也不展示 */
export const BLOCKED_MODELS = ['gpt-5.2', 'gpt-5.3-codex-spark'] as const

/**
 * new-api 分组 → 供应线。分组名以 new-api 后台为准，未列出的分组按分组名原样展示。
 * 号池渠道所在分组请在这里登记，模型广场据此打“号池”标签。
 */
export const SUPPLY_LINE_BY_GROUP: Record<string, SupplyLine> = {}

export type SupplyLine = 'pool' | 'aws' | 'official' | 'open_source'

export const SUPPLY_LINES: SupplyLine[] = ['pool', 'aws', 'official', 'open_source']
