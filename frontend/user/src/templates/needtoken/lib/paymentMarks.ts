export type PaymentMark = 'alipay' | 'wechat' | 'unionpay' | 'usdt' | 'crypto' | 'card' | 'paypal'
export type PaymentNetwork = 'bep20' | 'trc20'

const order: PaymentMark[] = ['alipay', 'wechat', 'unionpay', 'usdt', 'crypto', 'card', 'paypal']
const types: Record<string, PaymentMark> = {
  alipay: 'alipay', wechat: 'wechat', wxpay: 'wechat',
  unionpay: 'unionpay', union_pay: 'unionpay',
  usdt: 'usdt', 'usdt-trc20': 'usdt', 'usdt-bep20': 'usdt',
  'usdc-trc20': 'crypto', trx: 'crypto', bepusdt: 'crypto', epusdt: 'crypto',
  dujiaopay: 'crypto', tokenpay: 'crypto',
  stripe: 'card', card: 'card', credit_card: 'card', paypal: 'paypal',
}

const channelType = (row: Record<string, unknown>): string =>
  typeof row.channel_type === 'string' ? row.channel_type.trim().toLowerCase() : ''

const publicChannels = (channels: unknown): Record<string, unknown>[] =>
  Array.isArray(channels) ? channels.filter((row): row is Record<string, unknown> =>
    !!row && typeof row === 'object' && !Array.isArray(row) && row.is_active !== false) : []

/** Public channel metadata only. A logo does not make a channel available to an order. */
export function paymentMarks(channels: unknown): PaymentMark[] {
  const found = new Set<PaymentMark>()
  for (const row of publicChannels(channels)) {
    const mark = types[channelType(row)]
    if (mark) found.add(mark)
  }
  return order.filter(mark => found.has(mark))
}

/** Gateway names (including BEpusdt) do not identify a token or network. */
export function paymentNetworks(channels: unknown): PaymentNetwork[] {
  const found = new Set<PaymentNetwork>()
  for (const row of publicChannels(channels)) {
    const type = channelType(row)
    if (type === 'usdt-bep20') found.add('bep20')
    if (type === 'usdt-trc20' || type === 'usdc-trc20') found.add('trc20')
  }
  return (['bep20', 'trc20'] as const).filter(network => found.has(network))
}
