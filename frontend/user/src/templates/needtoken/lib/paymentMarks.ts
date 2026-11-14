export type PaymentMark = 'alipay' | 'wechat' | 'unionpay' | 'usdt' | 'crypto' | 'card' | 'paypal'
export type PaymentNetwork = 'bep20' | 'trc20'
export interface PublicPaymentMethod {
  key: string
  mark: PaymentMark | null
  label?: string
}
export type PaymentDisplayState = 'loading' | 'ready' | 'wallet' | 'empty' | 'unavailable'

const order: PaymentMark[] = ['alipay', 'wechat', 'unionpay', 'usdt', 'crypto', 'card', 'paypal']
const types: Record<string, PaymentMark> = {
  alipay: 'alipay', wechat: 'wechat', wxpay: 'wechat',
  unionpay: 'unionpay', union_pay: 'unionpay',
  usdt: 'usdt', 'usdt-trc20': 'usdt', 'usdt-bep20': 'usdt',
  'usdc-trc20': 'crypto', trx: 'crypto', bepusdt: 'crypto', epusdt: 'crypto',
  dujiaopay: 'crypto', tokenpay: 'crypto',
  stripe: 'card', card: 'card', credit_card: 'card', paypal: 'paypal',
  // DujiaoPay transaction channels expose token_id as channel_type. These
  // explicit IDs come from the gateway's supportedTokenChains contract.
  'tron-usdt': 'usdt', 'bsc-usdt': 'usdt', 'ethereum-usdt': 'usdt',
  'solana-usdt': 'usdt', 'aptos-usdt': 'usdt',
  'tron-trx': 'crypto', 'ethereum-eth': 'crypto', 'ethereum-usdc': 'crypto',
  'bsc-bnb': 'crypto', 'bsc-usdc': 'crypto', 'polygon-usdc': 'crypto',
  'polygon-usdt0': 'crypto', 'base-usdc': 'crypto', 'arbitrum-usdc': 'crypto',
  'arbitrum-usdt0': 'crypto', 'plasma-usdt0': 'crypto', 'x-layer-usdt0': 'crypto',
  'solana-usdc': 'crypto', 'aptos-usdc': 'crypto',
}

const channelType = (row: Record<string, unknown>): string =>
  typeof row.channel_type === 'string' ? row.channel_type.trim().toLowerCase() : ''

const publicChannels = (channels: unknown): Record<string, unknown>[] =>
  Array.isArray(channels) ? channels.filter((row): row is Record<string, unknown> =>
    !!row && typeof row === 'object' && !Array.isArray(row) &&
    (row.is_active === undefined || row.is_active === true)) : []

/** Public channel metadata only. A logo does not make a channel available to an order. */
export function paymentMarks(channels: unknown): PaymentMark[] {
  const found = new Set<PaymentMark>()
  for (const row of publicChannels(channels)) {
    const mark = types[channelType(row)]
    if (mark) found.add(mark)
  }
  return order.filter(mark => found.has(mark))
}

/** Preserve an unfamiliar but valid public channel by its actual display name. */
export function publicPaymentMethods(channels: unknown): PublicPaymentMethod[] {
  const methods: PublicPaymentMethod[] = paymentMarks(channels).map(mark => ({ key: mark, mark }))
  const labels = new Set<string>()
  for (const row of publicChannels(channels)) {
    const type = channelType(row)
    if (!type || type === 'balance' || types[type] || !Number.isSafeInteger(row.id) || Number(row.id) <= 0 || typeof row.name !== 'string') continue
    const name = row.name.replace(/[\u0000-\u001f\u007f]/g, '').trim().slice(0, 80)
    if (!name || labels.has(name)) continue
    labels.add(name)
    methods.push({ key: `channel-${row.id}`, mark: null, label: name })
  }
  return methods
}

export function paymentDisplayState(config: unknown, loading = false): PaymentDisplayState {
  if (loading) return 'loading'
  if (!config || typeof config !== 'object' || Array.isArray(config)) return 'unavailable'
  const value = config as Record<string, unknown>
  if (value.wallet_only_payment === true) return 'wallet'
  if (!Array.isArray(value.payment_channels)) return 'unavailable'
  return publicPaymentMethods(value.payment_channels).length ? 'ready' : 'empty'
}

/** Gateway names (including BEpusdt) do not identify a token or network. */
export function paymentNetworks(channels: unknown): PaymentNetwork[] {
  const found = new Set<PaymentNetwork>()
  for (const row of publicChannels(channels)) {
    const type = channelType(row)
    if (type === 'usdt-bep20' || type === 'bsc-usdt' || type === 'bsc-usdc') found.add('bep20')
    if (type === 'usdt-trc20' || type === 'usdc-trc20' || type === 'tron-usdt') found.add('trc20')
  }
  return (['bep20', 'trc20'] as const).filter(network => found.has(network))
}
