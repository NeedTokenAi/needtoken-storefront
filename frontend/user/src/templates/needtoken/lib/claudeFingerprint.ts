/**
 * Pure local China-environment fingerprint scoring for Claude usage readiness,
 * inspired by LinXiaoTao/FuckClaude's 10 weighted local signals (total 100 pts).
 * Kept strictly separate from exit-IP geo/reputation checks.
 */

export type FingerprintSignalId =
  | 'timezone'
  | 'language'
  | 'fonts'
  | 'vendorFonts'
  | 'webrtcLeak'
  | 'cnBrowser'
  | 'deviceVendor'
  | 'intlLocale'
  | 'timezoneOffset'
  | 'emoji'

export type RiskBand = 'low' | 'medium' | 'high'

export interface FingerprintSignalResult {
  id: FingerprintSignalId
  weight: number
  /** Normalized signal hit ratio in [0, 1]. */
  ratio: number
  /** Weighted contribution in [0, weight]. */
  points: number
  /** Human-readable observed value (never sent to any server). */
  raw: string
  /** True for signals known to be directly read by Claude Code runtime checks. */
  claudeUsed?: boolean
  status: 'safe' | 'warn' | 'risk' | 'unknown'
}

export interface FingerprintReport {
  totalScore: number
  band: RiskBand
  signals: FingerprintSignalResult[]
  triggeredCount: number
}

export const SIGNAL_WEIGHTS: Record<FingerprintSignalId, number> = {
  timezone: 24,
  language: 18,
  fonts: 14,
  vendorFonts: 10,
  webrtcLeak: 10,
  cnBrowser: 8,
  deviceVendor: 6,
  intlLocale: 4,
  timezoneOffset: 3,
  emoji: 3,
}

export const CN_TIMEZONES = [
  'Asia/Shanghai',
  'Asia/Urumqi',
  'Asia/Chongqing',
  'Asia/Chungking',
  'Asia/Harbin',
  'Asia/Kashgar',
] as const

export const GREATER_CN_RESTRICTED_TIMEZONES = [
  'Asia/Hong_Kong',
  'Asia/Macau',
] as const

export const FONTS_SC = [
  'Microsoft YaHei',
  'Microsoft YaHei UI',
  'SimSun',
  'NSimSun',
  'SimHei',
  'KaiTi',
  'FangSong',
  'DengXian',
  'PingFang SC',
  'Hiragino Sans GB',
  'STHeiti',
  'STSong',
  'Songti SC',
  'Source Han Sans CN',
  'Source Han Sans SC',
  'Noto Sans CJK SC',
  'Noto Serif CJK SC',
  'WenQuanYi Micro Hei',
  'WenQuanYi Zen Hei',
] as const

export const FONTS_TC = [
  'Microsoft JhengHei',
  'PMingLiU',
  'MingLiU',
  'DFKai-SB',
  'PingFang TC',
  'PingFang HK',
  'Source Han Sans TW',
  'Noto Sans CJK TC',
] as const

export const FONTS_CN_VENDOR = [
  'MiSans',
  'MIUI',
  'HarmonyOS Sans SC',
  'HarmonyOS Sans',
  'HONOR Sans',
  'OPPO Sans',
  'vivo Sans',
  'Alibaba PuHuiTi',
  'Alibaba Sans',
  'DingTalk JinBuTi',
  'Douyin Sans',
  'HYQiHei',
  'FZShuSong-Z01S',
  'FZKai-Z03S',
  'FZHei-B01S',
  'FZFangSong-Z02S',
] as const

export const CN_BROWSER_PATTERNS: ReadonlyArray<[RegExp, string]> = [
  [/micromessenger|wxwork/i, 'WeChat'],
  [/mqqbrowser|qqbrowser|\bqq\//i, 'QQ Browser'],
  [/quark/i, 'Quark'],
  [/ucbrowser|ucweb/i, 'UC Browser'],
  [/baiduboxapp|bidubrowser|baidubrowser/i, 'Baidu'],
  [/miuibrowser|xiaomi\/|mibrowser/i, 'Mi Browser'],
  [/huaweibrowser/i, 'Huawei Browser'],
  [/heytapbrowser|oppobrowser/i, 'HeyTap (OPPO)'],
  [/vivobrowser/i, 'vivo Browser'],
  [/sogoumobilebrowser|\bmetasr\b|\bse 2\.x\b/i, 'Sogou'],
  [/maxthon/i, 'Maxthon'],
  [/360se|360ee|qihoobrowser|\bqhbrowser\b/i, '360 Browser'],
  [/2345explorer|2345browser/i, '2345'],
  [/lbbrowser/i, 'Liebao'],
  [/theworld/i, 'TheWorld'],
  [/aweme|bytedancewebview|newsarticle|toutiaomicroapp/i, 'Douyin / Toutiao'],
  [/alipayclient/i, 'Alipay'],
  [/dingtalk/i, 'DingTalk'],
  [/weibo/i, 'Weibo'],
  [/xiaohongshu|xhsminiapp/i, 'Xiaohongshu'],
  [/\bbilibili\b/i, 'Bilibili'],
]

export const CN_DEVICE_PATTERNS: ReadonlyArray<[RegExp, string, number]> = [
  [/harmonyos|openharmony/i, 'HarmonyOS', 1],
  [/huawei|\bhonor\b/i, 'Huawei / Honor', 0.8],
  [/meizu/i, 'Meizu', 0.8],
  [/nubia|\bzte\b/i, 'ZTE / nubia', 0.7],
  [/xiaomi|redmi|\bpoco\b|\bm2\d{3}[a-z0-9]+\b/i, 'Xiaomi', 0.6],
  [/oppo|\bpd[a-z]m\d{2}\b/i, 'OPPO', 0.6],
  [/vivo|\bv2\d{3}[a-z]{1,2}\b/i, 'vivo', 0.6],
  [/realme|\brmx\d{4}\b/i, 'realme', 0.6],
  [/oneplus/i, 'OnePlus', 0.6],
  [/\blenovo\b|\bzuk\b/i, 'Lenovo', 0.5],
]

/** Regions widely restricted by Anthropic consumer/API terms (CN/HK/MO + embargoed territories). */
const ANTHROPIC_RESTRICTED_COUNTRIES = new Set([
  'CN', 'HK', 'MO', 'RU', 'BY', 'IR', 'KP', 'SY', 'CU', 'VE',
])

export type AnthropicRegionSupport = 'supported' | 'restricted' | 'unknown'

export function classifyAnthropicRegion(countryCode?: string): AnthropicRegionSupport {
  const code = (countryCode || '').trim().toUpperCase()
  if (!/^[A-Z]{2}$/.test(code)) return 'unknown'
  return ANTHROPIC_RESTRICTED_COUNTRIES.has(code) ? 'restricted' : 'supported'
}

export function scoreTimezoneCn(tz: string): number {
  const trimmed = tz.trim()
  if ((CN_TIMEZONES as readonly string[]).includes(trimmed)) return 1
  if ((GREATER_CN_RESTRICTED_TIMEZONES as readonly string[]).includes(trimmed)) return 0.6
  return 0
}

/**
 * Taiwan (zh-TW) is supported by Anthropic and adds 0 risk, including the bare
 * "zh" fallback appended after zh-TW. HK/MO retain partial risk.
 */
export function scoreLanguagesCn(langs: readonly string[]): number {
  const list = langs.map(item => (item || '').trim().toLowerCase()).filter(Boolean)
  const isTW = (l: string) => l.startsWith('zh-tw') || (l.includes('hant') && l.includes('tw'))
  const isHKMO = (l: string) => l.startsWith('zh-hk') || l.startsWith('zh-mo')
  const isTrad = (l: string) => isTW(l) || isHKMO(l) || l.includes('hant')
  const firstTrad = list.findIndex(isTrad)
  const isHansCN = (l: string, index: number) =>
    l.startsWith('zh-cn')
    || l.includes('hans')
    || (l === 'zh' && (firstTrad === -1 || index < firstTrad))

  const kept = list
    .map((l, i) => ({ l, i }))
    .filter(({ l, i }) => !isTW(l) && !(l === 'zh' && firstTrad !== -1 && i > firstTrad))

  const primary = kept[0]
  if (!primary) return 0
  if (isHansCN(primary.l, primary.i)) return 1
  if (isHKMO(primary.l) || primary.l.includes('hant')) return 0.5
  if (kept.some(({ l, i }) => isHansCN(l, i))) return 0.7
  if (kept.some(({ l }) => l.startsWith('zh'))) return 0.4
  return 0
}

export function scoreIntlLocaleCn(locale: string): number {
  const l = locale.trim().toLowerCase()
  if (!l) return 0
  if (l.startsWith('zh-cn') || l.includes('hans') || l === 'zh') return 1
  if (l.startsWith('zh') && !l.startsWith('zh-tw')) return 0.5
  return 0
}

export function scoreTimezoneOffsetCn(offsetMinutesWestOfUtc: number): number {
  return offsetMinutesWestOfUtc === -480 ? 0.7 : 0
}

export function scoreCnBrowser(probe: string): { name: string | null; ratio: number } {
  for (const [pattern, name] of CN_BROWSER_PATTERNS) {
    if (pattern.test(probe)) return { name, ratio: 1 }
  }
  return { name: null, ratio: 0 }
}

export function scoreCnDevice(probe: string): { name: string | null; ratio: number } {
  for (const [pattern, name, ratio] of CN_DEVICE_PATTERNS) {
    if (pattern.test(probe)) return { name, ratio }
  }
  return { name: null, ratio: 0 }
}

export function scoreChineseFonts(scHits: readonly string[], tcHits: readonly string[]): { raw: string; ratio: number } {
  let ratio = 0
  if (scHits.length >= 1) ratio = Math.min(1, 0.75 + 0.08 * scHits.length)
  else if (tcHits.length >= 1) ratio = 0.2
  const merged = [...scHits, ...tcHits]
  const raw = merged.length ? merged.slice(0, 4).join(', ') + (merged.length > 4 ? '…' : '') : ''
  return { raw, ratio }
}

export function scoreVendorFonts(hits: readonly string[]): { raw: string; ratio: number } {
  const ratio = hits.length >= 2 ? 1 : hits.length === 1 ? 0.8 : 0
  const raw = hits.length ? hits.slice(0, 3).join(', ') + (hits.length > 3 ? '…' : '') : ''
  return { raw, ratio }
}

export function scoreEmojiStyle(probe: string): { vendor: string; ratio: number } {
  const p = probe.toLowerCase()
  let vendor = 'Unknown'
  if (/iphone|ipad|ipod|mac/.test(p)) vendor = 'Apple'
  else if (/android/.test(p)) vendor = 'Google / Android'
  else if (/win/.test(p)) vendor = 'Microsoft'
  else if (/cros/.test(p)) vendor = 'Google'
  else if (/linux/.test(p)) vendor = 'Linux'

  const cnEmojiVendor = /huawei|honor|harmony|xiaomi|miui|oppo|vivo|meizu/.test(p)
  return { vendor, ratio: cnEmojiVendor ? 1 : 0 }
}

export function riskBandForScore(score: number): RiskBand {
  if (score <= 30) return 'low'
  if (score <= 60) return 'medium'
  return 'high'
}

export interface FingerprintInput {
  timezone: string
  timezoneOffsetMinutes: number
  languages: readonly string[]
  intlLocale: string
  userAgent: string
  modelHint?: string
  scFontHits: readonly string[]
  tcFontHits: readonly string[]
  vendorFontHits: readonly string[]
  webrtcLeakDetail?: 'differentExit' | 'sameExit' | 'publicOnly' | 'localOnly' | 'noCandidates' | 'unsupported' | 'notRequested'
  webrtcPublicIps?: readonly string[]
}

const signalStatus = (ratio: number, unknown = false): FingerprintSignalResult['status'] => {
  if (unknown) return 'unknown'
  if (ratio >= 0.7) return 'risk'
  if (ratio > 0) return 'warn'
  return 'safe'
}

export function evaluateChinaFingerprint(input: FingerprintInput): FingerprintReport {
  const tzRatio = scoreTimezoneCn(input.timezone)
  const langRatio = scoreLanguagesCn(input.languages)
  const fontRes = scoreChineseFonts(input.scFontHits, input.tcFontHits)
  const vendorFontRes = scoreVendorFonts(input.vendorFontHits)
  const probeStr = `${input.userAgent} ${input.modelHint || ''}`.trim()
  const browserRes = scoreCnBrowser(probeStr)
  const deviceRes = scoreCnDevice(probeStr)
  const intlRatio = scoreIntlLocaleCn(input.intlLocale)
  const offsetRatio = scoreTimezoneOffsetCn(input.timezoneOffsetMinutes)
  const emojiRes = scoreEmojiStyle(probeStr)

  let rtcRatio = 0
  let rtcRaw = ''
  let rtcUnknown = false
  switch (input.webrtcLeakDetail) {
    case 'differentExit':
      rtcRatio = 1
      rtcRaw = (input.webrtcPublicIps || []).slice(0, 2).join(', ')
      break
    case 'sameExit':
    case 'localOnly':
    case 'noCandidates':
      rtcRatio = 0
      rtcRaw = input.webrtcLeakDetail
      break
    case 'publicOnly':
      rtcRatio = 0.4
      rtcRaw = (input.webrtcPublicIps || []).slice(0, 2).join(', ')
      break
    default:
      rtcRatio = 0
      rtcUnknown = true
      rtcRaw = ''
      break
  }

  const offsetHours = -(input.timezoneOffsetMinutes / 60)
  const offsetLabel = `UTC${offsetHours >= 0 ? '+' : ''}${offsetHours}`

  const build = (
    id: FingerprintSignalId,
    ratio: number,
    raw: string,
    options: { claudeUsed?: boolean; unknown?: boolean } = {},
  ): FingerprintSignalResult => {
    const weight = SIGNAL_WEIGHTS[id]
    const clamped = Math.max(0, Math.min(1, ratio))
    const points = Math.round(weight * clamped)
    return {
      id,
      weight,
      ratio: clamped,
      points,
      raw,
      ...(options.claudeUsed ? { claudeUsed: true } : {}),
      status: signalStatus(clamped, options.unknown),
    }
  }

  const signals: FingerprintSignalResult[] = [
    build('timezone', tzRatio, input.timezone || '', { claudeUsed: true, unknown: !input.timezone }),
    build('language', langRatio, input.languages.slice(0, 4).join(', '), { unknown: !input.languages.length }),
    build('fonts', fontRes.ratio, fontRes.raw),
    build('vendorFonts', vendorFontRes.ratio, vendorFontRes.raw),
    build('webrtcLeak', rtcRatio, rtcRaw, { unknown: rtcUnknown }),
    build('cnBrowser', browserRes.ratio, browserRes.name || ''),
    build('deviceVendor', deviceRes.ratio, deviceRes.name || ''),
    build('intlLocale', intlRatio, input.intlLocale || '', { unknown: !input.intlLocale }),
    build('timezoneOffset', offsetRatio, offsetLabel),
    build('emoji', emojiRes.ratio, emojiRes.vendor),
  ]

  const totalScore = Math.min(100, signals.reduce((sum, s) => sum + s.points, 0))
  const triggeredCount = signals.filter(s => s.points > 0).length
  return {
    totalScore,
    band: riskBandForScore(totalScore),
    signals,
    triggeredCount,
  }
}

/** Browser-only Canvas font probe (never sends data off-device). */
function isFontAvailable(font: string, testString: string, ctx: CanvasRenderingContext2D): boolean {
  const size = '72px'
  const bases = ['monospace', 'sans-serif', 'serif']
  return bases.some(base => {
    ctx.font = `${size} ${base}`
    const baseWidth = ctx.measureText(testString).width
    ctx.font = `${size} "${font}", ${base}`
    const testWidth = ctx.measureText(testString).width
    return Math.abs(testWidth - baseWidth) > 0.5
  })
}

export async function collectBrowserFingerprintInput(
  webrtc?: { detail: FingerprintInput['webrtcLeakDetail']; publicIps?: readonly string[] },
): Promise<FingerprintInput> {
  let timezone = ''
  let intlLocale = ''
  try {
    const resolved = Intl.DateTimeFormat().resolvedOptions()
    timezone = resolved.timeZone || ''
    intlLocale = resolved.locale || ''
  } catch { /* Restricted Intl */ }

  let timezoneOffsetMinutes = 0
  try {
    timezoneOffsetMinutes = new Date().getTimezoneOffset()
  } catch { /* Restricted Date */ }

  const languages = typeof navigator !== 'undefined'
    ? (navigator.languages?.length ? [...navigator.languages] : navigator.language ? [navigator.language] : [])
    : []

  const userAgent = typeof navigator !== 'undefined' ? navigator.userAgent || '' : ''
  let modelHint = ''
  try {
    const uaData = (navigator as Navigator & {
      userAgentData?: {
        brands?: Array<{ brand: string }>
        getHighEntropyValues?: (hints: string[]) => Promise<Record<string, unknown>>
      }
    }).userAgentData
    const brands = uaData?.brands?.map(b => b.brand).join(' ') || ''
    const high = uaData?.getHighEntropyValues ? await uaData.getHighEntropyValues(['model', 'platform']) : {}
    modelHint = [brands, typeof high.model === 'string' ? high.model : '', typeof high.platform === 'string' ? high.platform : '']
      .filter(Boolean)
      .join(' ')
  } catch { /* High entropy hints optional */ }

  let scFontHits: string[] = []
  let tcFontHits: string[] = []
  let vendorFontHits: string[] = []
  if (typeof document !== 'undefined') {
    try {
      const canvas = document.createElement('canvas')
      const ctx = canvas.getContext('2d')
      if (ctx) {
        const probe = '中文字体检测ABCabc012'
        scFontHits = FONTS_SC.filter(f => isFontAvailable(f, probe, ctx))
        tcFontHits = FONTS_TC.filter(f => isFontAvailable(f, probe, ctx))
        vendorFontHits = FONTS_CN_VENDOR.filter(f => isFontAvailable(f, probe, ctx))
      }
    } catch { /* Canvas probe blocked */ }
  }

  return {
    timezone,
    timezoneOffsetMinutes,
    languages,
    intlLocale,
    userAgent,
    modelHint,
    scFontHits,
    tcFontHits,
    vendorFontHits,
    webrtcLeakDetail: webrtc?.detail,
    webrtcPublicIps: webrtc?.publicIps,
  }
}
