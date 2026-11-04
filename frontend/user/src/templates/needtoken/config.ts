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

/** new-api 公开定价接口（模型广场数据源） */
export const PRICING_URL = trimSlash(env.VITE_NEEDTOKEN_PRICING_URL || `${CONSOLE_URL}/api/pricing`)

/** 胶水公开的号池状态接口；不配置时模型广场显示“状态待接入” */
export const STATUS_URL = trimSlash(env.VITE_NEEDTOKEN_STATUS_URL || '')

export const CONSOLE_LOGIN_URL = `${CONSOLE_URL}/sign-in`
export const CONSOLE_REGISTER_URL = `${CONSOLE_URL}/sign-up`

/** 独角商品分类 slug：在独角后台建同名分类即可对应菜单 */
export const CATEGORY_SLUGS = {
    recharge: 'recharge',
    rental: 'rental',
    plans: 'plans',
} as const

/** 默认模型（账号 #2 桌面端 Ultra 档 2026-11-01 实测可用），首个为首选 */
export const DEFAULT_MODELS = [
    'gpt-6.1-sol',
    'gpt-6-astra',
    'gpt-6.1-terra',
    'gpt-6-sol',
    'gpt-6-luna',
    'gpt-5.6-sol',
    'gpt-5.6-terra',
    'gpt-5.6-luna',
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
