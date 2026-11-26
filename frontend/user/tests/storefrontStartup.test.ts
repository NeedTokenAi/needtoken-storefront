import assert from 'node:assert/strict'
import { existsSync, readFileSync } from 'node:fs'
import test from 'node:test'
import {
  isLegacyDefaultSiteIcon,
  NEEDTOKEN_MAIN_BRAND_NAME,
  NEEDTOKEN_MAIN_FALLBACK_ICON,
  NEEDTOKEN_MAIN_SITE_ICON,
  NEEDTOKEN_MAIN_TOUCH_ICON,
  resolveFallbackSiteName,
  resolveStorefrontHeadBrand,
  shouldBrandNeedtokenStartup,
  storefrontStartupPhase,
} from '../src/utils/storefrontStartup.ts'

test('boot skeleton hands over as soon as configuration and route are ready', () => {
  assert.equal(storefrontStartupPhase({ hasConfig: false, routeReady: false, routeFailed: false }), 'pending')
  assert.equal(storefrontStartupPhase({ hasConfig: true, routeReady: false, routeFailed: false }), 'pending')
  assert.equal(storefrontStartupPhase({ hasConfig: true, routeReady: true, routeFailed: false }), 'ready')
})
test('failed startup is retryable instead of an indefinite loading state', () => {
  assert.equal(storefrontStartupPhase({ hasConfig: false, routeReady: true, routeFailed: false }), 'error')
  assert.equal(storefrontStartupPhase({ hasConfig: true, routeReady: false, routeFailed: true }), 'error')
})
test('unknown and reseller domains do not inherit NeedToken boot branding from template alone', () => {
  assert.equal(shouldBrandNeedtokenStartup(null), false)
  assert.equal(shouldBrandNeedtokenStartup('needtoken'), false)
  assert.equal(shouldBrandNeedtokenStartup('classic', true), false)
  assert.equal(shouldBrandNeedtokenStartup('vault', true), false)
  assert.equal(shouldBrandNeedtokenStartup('needtoken', true), true)
})
test('first visits to the exact public main hosts show the NeedToken logo without cached configuration', () => {
  assert.equal(shouldBrandNeedtokenStartup(null, false, 'needtoken.ai'), true)
  assert.equal(shouldBrandNeedtokenStartup(null, false, 'www.needtoken.ai'), true)
})
test('host branding does not extend to subdomains, suffix matches, or reseller hosts', () => {
  for (const hostname of ['ops.needtoken.ai', 'api.needtoken.ai', 'store.needtoken.ai', 'needtoken.ai.example.com', 'example.needtoken.ai', 'needtoken.app', 'needtoken.io', 'reseller.example.com', 'localhost', '']) {
    assert.equal(shouldBrandNeedtokenStartup('needtoken', false, hostname), false, hostname)
  }
})

test('legacy upstream default favicon paths are filtered out', () => {
  assert.equal(isLegacyDefaultSiteIcon('/dj.svg'), true)
  assert.equal(isLegacyDefaultSiteIcon('dj.svg'), true)
  assert.equal(isLegacyDefaultSiteIcon('/dj.svg?v=123'), true)
  assert.equal(isLegacyDefaultSiteIcon('https://cdn.example.com/assets/DJ.SVG'), true)
  assert.equal(isLegacyDefaultSiteIcon('/needtoken-icon.svg'), false)
  assert.equal(isLegacyDefaultSiteIcon('/uploads/reseller/favicon.png'), false)
  assert.equal(isLegacyDefaultSiteIcon(''), false)
})

test('main hosts resolve NeedToken favicon immediately before config or layout mount', () => {
  const beforeConfig = resolveStorefrontHeadBrand({
    config: null,
    brandOverride: null,
    templateHint: null,
    verifiedMainHint: false,
    hostname: 'needtoken.ai',
  })
  assert.deepEqual(beforeConfig, {
    siteIcon: NEEDTOKEN_MAIN_SITE_ICON,
    fallbackIcon: NEEDTOKEN_MAIN_FALLBACK_ICON,
    touchIcon: NEEDTOKEN_MAIN_TOUCH_ICON,
    brandTitle: NEEDTOKEN_MAIN_BRAND_NAME,
  })

  const afterConfigBeforeLayout = resolveStorefrontHeadBrand({
    config: {
      storefront_template: 'needtoken',
      tenant: { mode: 'main' },
      brand: { site_icon: '' },
    },
    brandOverride: null,
    hostname: 'localhost',
  })
  assert.deepEqual(afterConfigBeforeLayout, {
    siteIcon: NEEDTOKEN_MAIN_SITE_ICON,
    fallbackIcon: NEEDTOKEN_MAIN_FALLBACK_ICON,
    touchIcon: NEEDTOKEN_MAIN_TOUCH_ICON,
    brandTitle: NEEDTOKEN_MAIN_BRAND_NAME,
  })
})

test('reseller and unknown hosts stay white-label and never flash upstream or NeedToken favicons', () => {
  const unknownBeforeConfig = resolveStorefrontHeadBrand({
    config: null,
    brandOverride: null,
    templateHint: 'needtoken',
    verifiedMainHint: false,
    hostname: 'shop.reseller.example',
  })
  assert.deepEqual(unknownBeforeConfig, { siteIcon: '' })

  const resellerWithLegacyIcon = resolveStorefrontHeadBrand({
    config: {
      storefront_template: 'needtoken',
      tenant: { mode: 'reseller' },
      brand: { site_icon: '/dj.svg' },
    },
    brandOverride: null,
    hostname: 'shop.reseller.example',
    resolveImageUrl: raw => `https://cdn.example${raw}`,
  })
  assert.deepEqual(resellerWithLegacyIcon, { siteIcon: '' })

  const resellerWithCustomIcon = resolveStorefrontHeadBrand({
    config: {
      storefront_template: 'needtoken',
      tenant: { mode: 'reseller' },
      brand: { site_icon: '/uploads/reseller/favicon.png' },
    },
    brandOverride: null,
    hostname: 'shop.reseller.example',
    resolveImageUrl: raw => `https://cdn.example${raw}`,
  })
  assert.deepEqual(resellerWithCustomIcon, { siteIcon: 'https://cdn.example/uploads/reseller/favicon.png' })
})

test('fallback site name avoids upstream brand and respects reseller isolation', () => {
  assert.equal(resolveFallbackSiteName('', { hostname: 'needtoken.ai' }), NEEDTOKEN_MAIN_BRAND_NAME)
  assert.equal(resolveFallbackSiteName('', { hostname: 'shop.example.com', isReseller: true, fallbackLabel: 'Digital Store' }), 'Digital Store')
  assert.equal(resolveFallbackSiteName('Reseller Shop', { hostname: 'shop.example.com', isReseller: true }), 'Reseller Shop')
  assert.equal(resolveFallbackSiteName('', { brandOverrideName: NEEDTOKEN_MAIN_BRAND_NAME }), NEEDTOKEN_MAIN_BRAND_NAME)
})

test('storefront HTML injects main-host favicon in head and ships without public dj.svg', () => {
  const indexHtml = readFileSync(new URL('../index.html', import.meta.url), 'utf8')
  const headSection = indexHtml.slice(0, indexHtml.indexOf('</head>'))
  assert.match(headSection, /needtoken-icon\.svg/)
  assert.match(headSection, /data-nt-boot-head|ntBootHead/)
  assert.match(indexHtml, /bootLayout\s*=\s*'console'/)
  assert.match(indexHtml, /class="nt-preboot-console"/)
  assert.equal(existsSync(new URL('../public/dj.svg', import.meta.url)), false)
})
