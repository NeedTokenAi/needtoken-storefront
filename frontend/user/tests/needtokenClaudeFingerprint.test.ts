import assert from 'node:assert/strict'
import test from 'node:test'
import {
  SIGNAL_WEIGHTS,
  classifyAnthropicRegion,
  evaluateChinaFingerprint,
  scoreCnBrowser,
  scoreCnDevice,
  scoreLanguagesCn,
  scoreTimezoneCn,
} from '../src/templates/needtoken/lib/claudeFingerprint.ts'

test('10 China fingerprint signal weights sum to 100', () => {
  const sum = Object.values(SIGNAL_WEIGHTS).reduce((acc, value) => acc + value, 0)
  assert.equal(sum, 100)
  assert.equal(Object.keys(SIGNAL_WEIGHTS).length, 10)
})

test('Taiwan timezone and zh-TW language add zero China risk while CN/HK/MO score accurately', () => {
  assert.equal(scoreTimezoneCn('Asia/Taipei'), 0)
  assert.equal(scoreTimezoneCn('America/Los_Angeles'), 0)
  assert.equal(scoreTimezoneCn('Asia/Shanghai'), 1)
  assert.equal(scoreTimezoneCn('Asia/Hong_Kong'), 0.6)
  assert.equal(scoreLanguagesCn(['zh-TW', 'zh', 'en-US']), 0)
  assert.equal(scoreLanguagesCn(['zh-CN', 'en-US']), 1)
  assert.equal(scoreLanguagesCn(['en-US', 'zh-CN']), 0.7)
  assert.equal(scoreLanguagesCn(['zh-HK', 'en']), 0.5)
  assert.equal(scoreLanguagesCn(['en-US', 'ja']), 0)
})

test('clean overseas environment scores low risk while typical CN environment scores high risk', () => {
  const clean = evaluateChinaFingerprint({
    timezone: 'America/Los_Angeles',
    timezoneOffsetMinutes: 480,
    languages: ['en-US', 'en'],
    intlLocale: 'en-US',
    userAgent: 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) Chrome/135.0.0.0 Safari/537.36',
    scFontHits: [],
    tcFontHits: [],
    vendorFontHits: [],
    webrtcLeakDetail: 'sameExit',
  })
  assert.equal(clean.totalScore, 0)
  assert.equal(clean.band, 'low')
  assert.equal(clean.triggeredCount, 0)

  const cn = evaluateChinaFingerprint({
    timezone: 'Asia/Shanghai',
    timezoneOffsetMinutes: -480,
    languages: ['zh-CN', 'zh', 'en'],
    intlLocale: 'zh-CN',
    userAgent: 'Mozilla/5.0 (Linux; Android 14; HarmonyOS; MicroMessenger/8.0)',
    scFontHits: ['PingFang SC', 'Microsoft YaHei', 'SimSun', 'SimHei'],
    tcFontHits: [],
    vendorFontHits: ['HarmonyOS Sans', 'MiSans'],
    webrtcLeakDetail: 'differentExit',
    webrtcPublicIps: ['114.114.114.114'],
  })
  assert.equal(cn.band, 'high')
  assert.ok(cn.totalScore >= 85)
  assert.equal(cn.signals.find(s => s.id === 'timezone')?.claudeUsed, true)
  assert.equal(scoreCnBrowser('MicroMessenger/8.0').name, 'WeChat')
  assert.equal(scoreCnDevice('HarmonyOS Huawei').name, 'HarmonyOS')
})

test('Anthropic region classification separates supported and restricted jurisdictions', () => {
  assert.equal(classifyAnthropicRegion('US'), 'supported')
  assert.equal(classifyAnthropicRegion('TW'), 'supported')
  assert.equal(classifyAnthropicRegion('JP'), 'supported')
  assert.equal(classifyAnthropicRegion('CN'), 'restricted')
  assert.equal(classifyAnthropicRegion('HK'), 'restricted')
  assert.equal(classifyAnthropicRegion('MO'), 'restricted')
  assert.equal(classifyAnthropicRegion(''), 'unknown')
})
