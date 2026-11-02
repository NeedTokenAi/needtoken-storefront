/** 示例代码片段：只用环境变量引用 key，不含任何真实密钥 */
import { API_BASE_URL, DEFAULT_MODELS } from '../config'

export const exportKeySnippet = () => `export NEEDTOKEN_API_KEY="<YOUR_NEEDTOKEN_API_KEY>"`

export const curlSnippet = (model: string = DEFAULT_MODELS[0]) => `curl ${API_BASE_URL}/chat/completions \\
  -H "Authorization: Bearer $NEEDTOKEN_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "${model}",
    "messages": [{"role": "user", "content": "Hello"}]
  }'`

export const opencodeSnippet = (model: string = DEFAULT_MODELS[0]) => {
    const models = Object.fromEntries(DEFAULT_MODELS.map(id => [id, { name: id }]))
    return JSON.stringify({
        $schema: 'https://opencode.ai/config.json',
        provider: {
            needtoken: {
                npm: '@ai-sdk/openai-compatible',
                name: 'needtoken',
                options: {
                    baseURL: API_BASE_URL,
                    apiKey: '{env:NEEDTOKEN_API_KEY}',
                },
                models,
            },
        },
        model: `needtoken/${model}`,
    }, null, 2)
}

export const pythonSnippet = (model: string = DEFAULT_MODELS[0]) => `from openai import OpenAI
import os

client = OpenAI(
    base_url="${API_BASE_URL}",
    api_key=os.environ["NEEDTOKEN_API_KEY"],
)
resp = client.chat.completions.create(
    model="${model}",
    messages=[{"role": "user", "content": "Hello"}],
)
print(resp.choices[0].message.content)`
