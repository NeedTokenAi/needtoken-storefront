export type PaymentMark = 'alipay' | 'wechat' | 'unionpay' | 'crypto' | 'card' | 'paypal'

const order: PaymentMark[] = ['alipay', 'wechat', 'unionpay', 'crypto', 'card', 'paypal']
const types: Record<string, PaymentMark> = {
  alipay: 'alipay', wechat: 'wechat', wxpay: 'wechat',
  unionpay: 'unionpay', union_pay: 'unionpay',
  usdt: 'crypto', 'usdt-trc20': 'crypto', 'usdc-trc20': 'crypto', trx: 'crypto',
  stripe: 'card', card: 'card', credit_card: 'card', paypal: 'paypal',
}

/** Public channel metadata only. A logo does not make a channel available to an order. */
export function paymentMarks(channels: unknown): PaymentMark[] {
  if (!Array.isArray(channels)) return []
  const found = new Set<PaymentMark>()
  for (const row of channels) {
    if (!row || typeof row !== 'object' || row.is_active === false) continue
    const type = typeof row.channel_type === 'string' ? row.channel_type.trim().toLowerCase() : ''
    const mark = types[type]
    if (mark) found.add(mark)
  }
  return order.filter(mark => found.has(mark))
}
