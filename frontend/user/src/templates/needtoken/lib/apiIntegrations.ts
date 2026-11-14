export type IntegrationScenario = 'coding' | 'local' | 'cloud' | 'agent'
export type IntegrationId = 'opencode' | 'ccswitch' | 'sub2api' | 'cloudflare' | 'openclaw'

export interface ApiIntegration {
  id: IntegrationId
  name: string
  scenario: IntegrationScenario
  docs?: string
  get?: string
  filename?: string
}

// Public project-owned sources checked against their current documentation.
export const API_INTEGRATIONS: ApiIntegration[] = [
  { id: 'opencode', name: 'OpenCode', scenario: 'coding', docs: 'https://opencode.ai/docs/providers/', get: 'https://opencode.ai/product', filename: 'opencode.json' },
  { id: 'ccswitch', name: 'CC Switch', scenario: 'local', docs: 'https://ccswitch.io/zh/docs?item=service&section=proxy', get: 'https://github.com/farion1231/cc-switch/releases/latest' },
  { id: 'sub2api', name: 'Sub2API', scenario: 'local', docs: 'https://github.com/Wei-Shaw/sub2api', get: 'https://github.com/Wei-Shaw/sub2api#deployment' },
  { id: 'cloudflare', name: 'Cloudflare AI Gateway', scenario: 'cloud', docs: 'https://developers.cloudflare.com/ai-gateway/configuration/custom-providers/', get: 'https://dash.cloudflare.com/', filename: 'needtoken-cloudflare-provider.json' },
  { id: 'openclaw', name: 'OpenClaw', scenario: 'agent', docs: 'https://docs.openclaw.ai/gateway/config-tools/custom-providers', get: 'https://openclaw.ai/', filename: 'needtoken-openclaw.template.json' },
]

export const integrationModel = (value: string): string | null => {
  const model = value.trim()
  return /^[a-zA-Z0-9][a-zA-Z0-9._:/@-]{0,199}$/.test(model) ? model : null
}

const rootEndpoint = (baseUrl: string) => baseUrl.replace(/\/+$/, '').replace(/\/v1$/, '')
export const integrationEndpoint = (id: IntegrationId, baseUrl: string) => id === 'cloudflare' || id === 'sub2api' || id === 'ccswitch' ? rootEndpoint(baseUrl) : baseUrl

export function integrationTemplate(id: IntegrationId, baseUrl: string, model: string): string | null {
  // Provider creation is independent of the model used by the client.
  if (id === 'cloudflare') return JSON.stringify({ name: 'NeedToken', slug: 'needtoken', base_url: rootEndpoint(baseUrl) }, null, 2)
  const modelId = integrationModel(model)
  if (!modelId) return null
  if (id === 'opencode') return JSON.stringify({
    $schema: 'https://opencode.ai/config.json',
    model: `needtoken/${modelId}`,
    providers: {
      needtoken: {
        name: 'NeedToken',
        env: ['NEEDTOKEN_API_KEY'],
        package: '@opencode/ai/providers/openai-compatible',
        settings: { baseURL: baseUrl },
        models: { [modelId]: { name: modelId } },
      },
    },
  }, null, 2)
  if (id === 'openclaw') return JSON.stringify({
    models: {
      mode: 'merge',
      providers: {
        needtoken: {
          baseUrl,
          apiKey: '${NEEDTOKEN_API_KEY}',
          api: 'openai-completions',
          models: [{ id: modelId, name: modelId }],
        },
      },
    },
    agents: { defaults: { model: { primary: `needtoken/${modelId}` } } },
  }, null, 2)
  // Cloudflare appends the complete provider path after custom-needtoken/.
  // Keeping /v1 here and in that path would send /v1/v1 upstream.
  if (id === 'sub2api') return `Base URL: ${rootEndpoint(baseUrl)}\nAPI Key: <YOUR_NEEDTOKEN_API_KEY>\nModel: ${modelId}`
  return `Name: NeedToken\nBase URL: ${rootEndpoint(baseUrl)}\nAPI Key: <YOUR_NEEDTOKEN_API_KEY>\nModel: ${modelId}`
}

export type ApiPreview = 'curl' | 'python' | 'opencode'

const shellQuote = (value: string) => `'${value.replace(/'/g, `'"'"'`)}'`

export function apiPreviewCode(kind: ApiPreview, baseUrl: string, model: string): string {
  if (kind === 'opencode') return integrationTemplate('opencode', baseUrl, model) || ''
  if (kind === 'python') return `from openai import OpenAI\nimport os\n\nclient = OpenAI(\n    base_url=${JSON.stringify(baseUrl)},\n    api_key=os.environ["NEEDTOKEN_API_KEY"],\n)\n\nresponse = client.chat.completions.create(\n    model=${JSON.stringify(model)},\n    messages=[{"role": "user", "content": "Hello"}],\n)`
  const body = JSON.stringify({ model, messages: [{ role: 'user', content: 'Hello' }] }, null, 2)
  return `curl ${shellQuote(`${baseUrl}/chat/completions`)} \\\n  -H "Authorization: Bearer $NEEDTOKEN_API_KEY" \\\n  -H "Content-Type: application/json" \\\n  -d ${shellQuote(body)}`
}
