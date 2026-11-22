import test from 'node:test'
import assert from 'node:assert/strict'
import { API_INTEGRATIONS, apiPreviewCode, integrationEndpoint, integrationModel, integrationTemplate } from '../src/templates/needtoken/lib/apiIntegrations.ts'

const endpoint = 'https://api.needtoken.example/gateway/v1'
const model = 'vendor/model-v2'

test('OpenCode uses the current provider schema and references an environment credential', () => {
  const config = JSON.parse(integrationTemplate('opencode', endpoint, model)!)
  assert.equal(config.model, `needtoken/${model}`)
  assert.equal(config.providers.needtoken.settings.baseURL, endpoint)
  assert.equal(config.providers.needtoken.package, '@opencode/ai/providers/openai-compatible')
  assert.deepEqual(config.providers.needtoken.env, ['NEEDTOKEN_API_KEY'])
  assert.equal(config.providers.needtoken.models[model].name, model)
  assert.equal(config.provider, undefined)
  assert.equal(config.providers.needtoken.apiKey, undefined)
})

test('OpenClaw template merges an explicit provider without embedding an API key', () => {
  const config = JSON.parse(integrationTemplate('openclaw', endpoint, model)!)
  assert.equal(config.models.mode, 'merge')
  assert.deepEqual(config.models.providers.needtoken, { baseUrl: endpoint, apiKey: '${NEEDTOKEN_API_KEY}', api: 'openai-completions', models: [{ id: model, name: model }] })
  assert.equal(config.agents.defaults.model.primary, `needtoken/${model}`)
})

test('gateway prefixes avoid duplicated /v1 and preserve deployment path prefixes', () => {
  for (const id of ['cloudflare', 'sub2api'] as const) assert.equal(integrationEndpoint(id, endpoint), 'https://api.needtoken.example/gateway')
  assert.equal(integrationEndpoint('opencode', endpoint), endpoint)
  assert.equal(integrationEndpoint('openclaw', endpoint), endpoint)
  const config = JSON.parse(integrationTemplate('cloudflare', endpoint, '')!)
  assert.deepEqual(config, { name: 'NeedToken', slug: 'needtoken', base_url: 'https://api.needtoken.example/gateway' })
  assert.equal(integrationEndpoint('cloudflare', 'https://api.example/gateway/v1/'), 'https://api.example/gateway')
})

test('CC Switch OpenAI-compatible setup preserves the SDK /v1 endpoint', () => {
  assert.equal(integrationEndpoint('ccswitch', endpoint), endpoint)
  const template = integrationTemplate('ccswitch', endpoint, model)!
  assert.ok(template.includes(`Base URL: ${endpoint}\n`))
  assert.ok(template.includes('API Format: OpenAI Chat Completions'))
  assert.ok(template.includes('API Key: <YOUR_NEEDTOKEN_API_KEY>'))
})

test('model input cannot produce malformed config or shell fragments', () => {
  assert.equal(integrationModel(' vendor/model-v2 '), model)
  for (const invalid of ['', 'a\nb', 'a"b', "a'b", '$(whoami)', '../bad', 'x'.repeat(201)]) {
    assert.equal(integrationModel(invalid), null)
    assert.equal(integrationTemplate('opencode', endpoint, invalid), null)
    assert.equal(integrationTemplate('openclaw', endpoint, invalid), null)
  }
})

test('preview code quotes request data and source links never contain credentials', () => {
  const curl = apiPreviewCode('curl', endpoint, "model'with-quote")
  assert.ok(curl.includes('"Authorization: Bearer $NEEDTOKEN_API_KEY"'))
  assert.ok(curl.includes(`model'"'"'with-quote`))
  assert.ok(!curl.includes('\n+'))
  assert.ok(apiPreviewCode('python', endpoint, model).includes('os.environ["NEEDTOKEN_API_KEY"]'))
  for (const tool of API_INTEGRATIONS) for (const link of [tool.docs, tool.get].filter(Boolean)) {
    const url = new URL(link!)
    assert.equal(url.protocol, 'https:')
    assert.equal(url.username + url.password, '')
    assert.ok(!/[?&](?:key|api_key|token)=/i.test(link!))
  }
  assert.equal(API_INTEGRATIONS.filter(tool => tool.name === 'CC Switch').length, 1)
  assert.ok(!API_INTEGRATIONS.some(tool => tool.name === 'CC Proxy'))
})

test('newly added coding & desktop integrations strip /v1 where required and emit valid templates', () => {
  assert.equal(integrationEndpoint('claudecode', endpoint), 'https://api.needtoken.example/gateway')
  assert.equal(integrationEndpoint('geminicli', endpoint), 'https://api.needtoken.example/gateway')
  assert.equal(integrationEndpoint('codex', endpoint), endpoint)
  assert.equal(integrationEndpoint('cursor', endpoint), endpoint)
  assert.equal(integrationEndpoint('cherrystudio', endpoint), endpoint)

  const claudeCfg = JSON.parse(integrationTemplate('claudecode', endpoint, model)!)
  assert.equal(claudeCfg.env.ANTHROPIC_BASE_URL, 'https://api.needtoken.example/gateway')
  assert.equal(claudeCfg.env.ANTHROPIC_MODEL, model)

  const codexCfg = integrationTemplate('codex', endpoint, model)!
  assert.ok(codexCfg.includes(`base_url = "${endpoint}"`))
  assert.ok(codexCfg.includes('wire_api = "responses"'))

  const geminiCfg = integrationTemplate('geminicli', endpoint, model)!
  assert.ok(geminiCfg.includes('GOOGLE_GEMINI_BASE_URL="https://api.needtoken.example/gateway"'))
})
