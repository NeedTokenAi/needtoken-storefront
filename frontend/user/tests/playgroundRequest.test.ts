import test from 'node:test'
import assert from 'node:assert/strict'
import http from 'node:http'
import { execFile, execFileSync } from 'node:child_process'
import { promisify } from 'node:util'

const execFileAsync = promisify(execFile)
import {
  buildPlaygroundChatRequest,
  buildPlaygroundCurlSnippet,
  shellSingleQuote,
} from '../src/utils/playgroundRequest.ts'

test('R18: shellSingleQuote and buildPlaygroundCurlSnippet produce valid Bash for quotes, backslashes, newlines, shell metacharacters, and CJK', async () => {
  const syntheticKey = 'sk-synthetic-local-test-key'
  const complexPrompt = `What's 2 + 2? "double" \\ backslash\nline2 $HOME $(whoami) 你好世界`

  const payload = buildPlaygroundChatRequest({
    model: 'gpt-4o-mini',
    systemPrompt: `You're concise & helpful.`,
    historyMessages: [
      { role: 'user', content: `Hi 'there'` },
      { role: 'assistant', content: `Hello! How can I help with "it"?` },
    ],
    userPrompt: complexPrompt,
    temperature: 0.3,
    maxTokens: 512,
  })

  assert.equal(payload.model, 'gpt-4o-mini')
  assert.equal(payload.temperature, 0.3)
  assert.equal(payload.max_tokens, 512)
  assert.equal(payload.stream, true)
  assert.deepEqual(payload.stream_options, { include_usage: true })
  assert.deepEqual(payload.messages, [
    { role: 'system', content: `You're concise & helpful.` },
    { role: 'user', content: `Hi 'there'` },
    { role: 'assistant', content: `Hello! How can I help with "it"?` },
    { role: 'user', content: complexPrompt },
  ])

  const placeholderSnippet = buildPlaygroundCurlSnippet({
    apiBaseURL: 'http://127.0.0.1:9999/v1',
    apiKey: syntheticKey,
    payload,
  })
  // Validate strict Bash syntax first (`What's` previously broke bash -n!).
  execFileSync('bash', ['-n', '-c', placeholderSnippet], { encoding: 'utf8' })

  // Spin up local HTTP capture server and actually execute the copied cURL via bash.
  let capturedAuth = ''
  let capturedBody: unknown = null
  const server = http.createServer((req, res) => {
    const chunks: Buffer[] = []
    req.on('data', chunk => chunks.push(chunk))
    req.on('end', () => {
      capturedAuth = String(req.headers.authorization || '')
      capturedBody = JSON.parse(Buffer.concat(chunks).toString('utf8'))
      res.writeHead(200, { 'Content-Type': 'application/json', Connection: 'close' })
      res.end('data: [DONE]\n\n')
    })
  })
  await new Promise<void>(resolve => server.listen(0, '127.0.0.1', () => resolve()))
  try {
    const address = server.address() as { port: number }
    const liveSnippet = buildPlaygroundCurlSnippet({
      apiBaseURL: `http://127.0.0.1:${address.port}/v1`,
      apiKey: syntheticKey,
      payload,
    })
    await execFileAsync('bash', ['-c', liveSnippet], {
      encoding: 'utf8',
      timeout: 5000,
      env: { ...process.env, http_proxy: '', https_proxy: '', all_proxy: '', HTTP_PROXY: '', HTTPS_PROXY: '', ALL_PROXY: '', no_proxy: '*', NO_PROXY: '*' },
    })
    assert.equal(capturedAuth, `Bearer ${syntheticKey}`)
    assert.deepEqual(capturedBody, payload)
  } finally {
    server.closeAllConnections?.()
    await new Promise<void>(resolve => server.close(() => resolve()))
  }

  assert.equal(shellSingleQuote("a'b"), `'a'\\''b'`)
})
