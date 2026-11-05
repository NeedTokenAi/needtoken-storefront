/** Auth continuations stay on this storefront and cannot loop back into auth. */
export function authReturnPath(value: unknown, fallback = '/me/orders'): string {
  if (typeof value !== 'string' || !value.startsWith('/') || value.startsWith('//') || /[\\\u0000-\u0020\u007f]/.test(value)) return fallback
  try {
    const url = new URL(value, 'https://storefront.invalid')
    const pathname = decodeURIComponent(url.pathname)
    // Dot segments can turn a single-slash input into a protocol-relative
    // output. Check the normalized/decoded path before exposing it as a link.
    if (url.origin !== 'https://storefront.invalid' || pathname.startsWith('//') || /[\\\u0000-\u0020\u007f]/.test(pathname) || /^\/auth(?:\/|$)/i.test(pathname)) return fallback
    return `${url.pathname}${url.search}${url.hash}`
  } catch { return fallback }
}
