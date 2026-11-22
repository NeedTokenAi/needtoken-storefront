export type IntegrationScenario = 'coding' | 'local' | 'cloud' | 'agent'
export type IntegrationId =
  | 'claudecode'
  | 'codex'
  | 'geminicli'
  | 'opencode'
  | 'cursor'
  | 'ccswitch'
  | 'sub2api'
  | 'cloudflare'
  | 'openclaw'
  | 'cherrystudio'

export interface ApiIntegration {
  id: IntegrationId
  name: string
  scenario: IntegrationScenario
  badge?: string
  docs?: string
  get?: string
  filename?: string
}

export const CC_SWITCH_REFERENCE_URL = 'https://ccswitch.io/'
export const CC_SWITCH_IMPORT_GUIDE_URL = 'https://github.com/farion1231/cc-switch/blob/main/docs/guides/deeplink-site-integration-guide-en.md'

// Public project-owned sources checked against their current documentation.
export const API_INTEGRATIONS: ApiIntegration[] = [
  { id: 'claudecode', name: 'Claude Code', scenario: 'coding', badge: 'Anthropic', docs: 'https://docs.anthropic.com/en/docs/claude-code/overview', get: 'https://docs.anthropic.com/en/docs/claude-code/setup', filename: 'claude-code-settings.json' },
  { id: 'codex', name: 'Codex CLI', scenario: 'coding', badge: 'Responses', docs: 'https://github.com/openai/codex', get: 'https://github.com/openai/codex#quickstart', filename: 'codex-config.toml' },
  { id: 'geminicli', name: 'Gemini CLI', scenario: 'coding', badge: 'Gemini', docs: 'https://github.com/google-gemini/gemini-cli', get: 'https://github.com/google-gemini/gemini-cli#quickstart' },
  { id: 'opencode', name: 'OpenCode', scenario: 'coding', badge: 'OpenAI', docs: 'https://opencode.ai/docs/providers/', get: 'https://opencode.ai/product', filename: 'opencode.json' },
  { id: 'cursor', name: 'Cursor / Cline', scenario: 'coding', badge: 'IDE', docs: 'https://docs.cursor.com/', get: 'https://cursor.com/' },
  { id: 'ccswitch', name: 'CC Switch', scenario: 'local', badge: '推荐路由', docs: CC_SWITCH_IMPORT_GUIDE_URL, get: 'https://github.com/farion1231/cc-switch/releases/latest' },
  { id: 'sub2api', name: 'Sub2API', scenario: 'local', badge: '本地聚合', docs: 'https://github.com/Wei-Shaw/sub2api', get: 'https://github.com/Wei-Shaw/sub2api#deployment' },
  { id: 'cloudflare', name: 'Cloudflare AI Gateway', scenario: 'cloud', badge: '边缘代理', docs: 'https://developers.cloudflare.com/ai-gateway/configuration/custom-providers/', get: 'https://dash.cloudflare.com/', filename: 'needtoken-cloudflare-provider.json' },
  { id: 'openclaw', name: 'OpenClaw', scenario: 'agent', badge: 'Agent', docs: 'https://docs.openclaw.ai/gateway/config-tools/custom-providers', get: 'https://openclaw.ai/', filename: 'needtoken-openclaw.template.json' },
  { id: 'cherrystudio', name: 'Cherry Studio', scenario: 'agent', badge: 'Desktop', docs: 'https://docs.cherry-ai.com/', get: 'https://cherry-ai.com/' },
]

export const integrationModel = (value: string): string | null => {
  const model = value.trim()
  return /^[a-zA-Z0-9][a-zA-Z0-9._:/@-]{0,199}$/.test(model) ? model : null
}

const rootEndpoint = (baseUrl: string) => baseUrl.replace(/\/+$/, '').replace(/\/v1$/, '')
export const integrationEndpoint = (id: IntegrationId, baseUrl: string) =>
  id === 'cloudflare' || id === 'sub2api' || id === 'claudecode' || id === 'geminicli'
    ? rootEndpoint(baseUrl)
    : baseUrl

export function integrationTemplate(id: IntegrationId, baseUrl: string, model: string): string | null {
  // Provider creation is independent of the model used by the client.
  if (id === 'cloudflare') return JSON.stringify({ name: 'NeedToken', slug: 'needtoken', base_url: rootEndpoint(baseUrl) }, null, 2)
  const modelId = integrationModel(model)
  if (!modelId) return null
  if (id === 'claudecode') return JSON.stringify({
    env: {
      ANTHROPIC_BASE_URL: rootEndpoint(baseUrl),
      ANTHROPIC_AUTH_TOKEN: '<YOUR_NEEDTOKEN_API_KEY>',
      ANTHROPIC_MODEL: modelId,
    },
  }, null, 2)
  if (id === 'codex') return `model_provider = "needtoken"\nmodel = "${modelId}"\n\n[model_providers.needtoken]\nname = "NeedToken"\nbase_url = "${baseUrl}"\nwire_api = "responses"\nenv_key = "NEEDTOKEN_API_KEY"`
  if (id === 'geminicli') return `export GOOGLE_GEMINI_BASE_URL="${rootEndpoint(baseUrl)}"\nexport GEMINI_API_KEY="<YOUR_NEEDTOKEN_API_KEY>"\nexport GEMINI_MODEL="${modelId}"`
  if (id === 'cursor' || id === 'cherrystudio') return `Provider: OpenAI Compatible\nBase URL: ${baseUrl}\nAPI Key: <YOUR_NEEDTOKEN_API_KEY>\nModel: ${modelId}`
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
  return `Name: NeedToken\nBase URL: ${baseUrl}\nAPI Format: OpenAI Chat Completions\nAPI Key: <YOUR_NEEDTOKEN_API_KEY>\nModel: ${modelId}`
}

export type ApiPreview = 'curl' | 'python' | 'opencode'

const shellQuote = (value: string) => `'${value.replace(/'/g, `'"'"'`)}'`

export function apiPreviewCode(kind: ApiPreview, baseUrl: string, model: string): string {
  if (kind === 'opencode') return integrationTemplate('opencode', baseUrl, model) || ''
  if (kind === 'python') return `from openai import OpenAI\nimport os\n\nclient = OpenAI(\n    base_url=${JSON.stringify(baseUrl)},\n    api_key=os.environ["NEEDTOKEN_API_KEY"],\n)\n\nresponse = client.chat.completions.create(\n    model=${JSON.stringify(model)},\n    messages=[{"role": "user", "content": "Hello"}],\n)`
  const body = JSON.stringify({ model, messages: [{ role: 'user', content: 'Hello' }] }, null, 2)
  return `curl ${shellQuote(`${baseUrl}/chat/completions`)} \\\n  -H "Authorization: Bearer $NEEDTOKEN_API_KEY" \\\n  -H "Content-Type: application/json" \\\n  -d ${shellQuote(body)}`
}
