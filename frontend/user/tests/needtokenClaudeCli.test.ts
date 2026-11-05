import assert from 'node:assert/strict'
import { execFile, spawnSync } from 'node:child_process'
import { mkdtemp, rm, writeFile } from 'node:fs/promises'
import os from 'node:os'
import path from 'node:path'
import { promisify } from 'node:util'
import { fileURLToPath } from 'node:url'
import test from 'node:test'

const execute = promisify(execFile)
const script = fileURLToPath(new URL('../public/needtoken-claude-network-check.mjs', import.meta.url))

async function runLocalReport(missingCurl = false) {
  const directory = await mkdtemp(path.join(os.tmpdir(), 'nt-claude-cli-'))
  const preload = path.join(directory, 'local-dns.mjs')
  // DNS is mocked only in this test; the shipped script uses the real resolver.
  await writeFile(preload, "import dns from 'node:dns/promises';import {syncBuiltinESMExports} from 'node:module';dns.Resolver=class{resolve4(){return Promise.reject(new Error('blocked'))}resolve6(){return Promise.reject(new Error('blocked'))}cancel(){}};syncBuiltinESMExports();")
  try {
    const proxy = 'http://test-user:test-secret@127.0.0.1:1'
    const result = await execute(process.execPath, ['--import', preload, script], { timeout: 11000, maxBuffer: 65536, env: { ...process.env, HTTPS_PROXY: proxy, https_proxy: proxy, HTTP_PROXY: proxy, http_proxy: proxy, ALL_PROXY: proxy, all_proxy: proxy, NO_PROXY: '', no_proxy: '', ANTHROPIC_BASE_URL: 'https://secret-custom-endpoint.invalid', ...(missingCurl ? { PATH: directory } : {}) } })
    assert.equal(result.stderr, '')
    assert.equal(result.stdout.includes('test-secret'), false)
    assert.equal(result.stdout.includes('test-user'), false)
    assert.equal(result.stdout.includes('secret-custom-endpoint'), false)
    return JSON.parse(result.stdout)
  } finally { await rm(directory, { recursive: true, force: true }) }
}

test('local CLI script uses actual curl and returns a bounded failure for an unreachable local proxy', async context => {
  if (spawnSync('curl', ['--version'], { stdio: 'ignore' }).status !== 0) { context.skip('curl is not installed'); return }
  const report = await runLocalReport()
  assert.equal(report.schema, 'needtoken.claude-network.v1')
  assert.equal(report.scope, 'curl_inherits_proxy_env')
  assert.equal(report.proxy.https, true)
  assert.equal(report.proxy.custom_base_url, true)
  assert.equal(report.dns.status, 'error')
  assert.equal(report.curl.status, 'error')
  assert.notEqual(report.curl.exit_code, 0)
  assert.equal(report.curl.http_status, undefined)
})

test('local CLI script reports missing curl without crashing or printing environment values', async () => {
  const report = await runLocalReport(true)
  assert.equal(report.curl.status, 'unsupported')
  assert.equal(report.curl.exit_code, -1)
})
