export interface PlaygroundChatMessage {
  role: 'system' | 'user' | 'assistant' | string
  content: string
}

export interface BuildPlaygroundChatRequestInput {
  model: string
  systemPrompt?: string
  historyMessages?: ReadonlyArray<PlaygroundChatMessage>
  userPrompt?: string
  temperature?: number
  maxTokens?: number | null
}

export interface PlaygroundChatRequestPayload {
  model: string
  messages: Array<{ role: string; content: string }>
  temperature: number
  stream: boolean
  stream_options: { include_usage: boolean }
  max_tokens?: number
}

export function buildPlaygroundChatRequest(input: BuildPlaygroundChatRequestInput): PlaygroundChatRequestPayload {
  const messages: Array<{ role: string; content: string }> = []
  const sys = String(input.systemPrompt || '').trim()
  if (sys) {
    messages.push({ role: 'system', content: sys })
  }

  let hasHistoryUser = false
  for (const item of input.historyMessages || []) {
    const role = String(item?.role || '').trim()
    const content = String(item?.content || '')
    if (!role) continue
    // Drop blank in-flight assistant placeholder bubbles while keeping valid turns verbatim.
    if (role === 'assistant' && !content) continue
    if (role === 'user') hasHistoryUser = true
    messages.push({ role, content })
  }

  const prompt = String(input.userPrompt || '').trim()
  if (prompt) {
    messages.push({ role: 'user', content: prompt })
  } else if (!hasHistoryUser) {
    messages.push({ role: 'user', content: 'Hello' })
  }

  const temp = typeof input.temperature === 'number' && Number.isFinite(input.temperature)
    ? input.temperature
    : 0.7

  const payload: PlaygroundChatRequestPayload = {
    model: String(input.model || '').trim(),
    messages,
    temperature: temp,
    stream: true,
    stream_options: { include_usage: true },
  }

  if (typeof input.maxTokens === 'number' && Number.isFinite(input.maxTokens) && input.maxTokens > 0) {
    payload.max_tokens = Math.round(input.maxTokens)
  }

  return payload
}

/**
 * Escapes an arbitrary string as a single POSIX shell argument wrapped in single quotes.
 */
export function shellSingleQuote(value: string): string {
  return `'${String(value).replace(/'/g, "'\\''")}'`
}

export function buildPlaygroundCurlSnippet(options: {
  apiBaseURL: string
  apiKey: string
  payload: PlaygroundChatRequestPayload
}): string {
  const endpoint = `${String(options.apiBaseURL || '').replace(/\/+$/, '')}/chat/completions`
  return [
    `curl -N ${shellSingleQuote(endpoint)} \\`,
    `  -H ${shellSingleQuote('Content-Type: application/json')} \\`,
    `  -H ${shellSingleQuote(`Authorization: Bearer ${String(options.apiKey || '')}`)} \\`,
    `  -d ${shellSingleQuote(JSON.stringify(options.payload))}`,
  ].join('\n')
}
