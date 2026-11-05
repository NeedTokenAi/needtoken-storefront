#!/usr/bin/env node
// NeedToken.AI local diagnostics. Read this source before running it.
// Fixed public endpoint only. No account credentials, custom URLs, or environment
// values are printed. curl inherits your proxy environment and verifies TLS.
import { spawn } from 'node:child_process'
import { Resolver } from 'node:dns/promises'
import os from 'node:os'

const host = 'api.anthropic.com'
const present = (...names) => names.some(name => Boolean(process.env[name]))
let timezone = ''
try { timezone = Intl.DateTimeFormat().resolvedOptions().timeZone || '' } catch { /* Keep other checks available. */ }
const report = {
  schema: 'needtoken.claude-network.v1',
  measured_at: new Date().toISOString(),
  scope: 'curl_inherits_proxy_env',
  runtime: { node_major: Number(process.versions.node.split('.')[0]), timezone },
  proxy: {
    http: present('HTTP_PROXY', 'http_proxy'), https: present('HTTPS_PROXY', 'https_proxy'),
    all: present('ALL_PROXY', 'all_proxy'), no_proxy: present('NO_PROXY', 'no_proxy'),
    custom_base_url: present('ANTHROPIC_BASE_URL'),
  },
  dns: { status: 'error', address_count: 0 },
  curl: { status: 'unsupported', exit_code: -1 },
}

async function checkDns() {
  let resolver
  let timer
  try {
    resolver = new Resolver({ timeout: 2000, tries: 1 })
    timer = setTimeout(() => resolver.cancel(), 2500)
    const results = await Promise.allSettled([resolver.resolve4(host), resolver.resolve6(host)])
    const count = results.reduce((total, result) => total + (result.status === 'fulfilled' ? result.value.length : 0), 0)
    report.dns = { status: count ? 'ok' : 'error', address_count: count }
  } catch { /* DNS restrictions do not prevent the curl result. */ }
  finally { clearTimeout(timer); resolver?.cancel() }
}

async function checkCurl() {
  await new Promise(resolve => {
    const writeOut = '{"http_status":%{http_code},"ssl_verify_result":%{ssl_verify_result},"time_total":%{time_total},"peer_ip":"%{remote_ip}"}'
    // -q must be first: local .curlrc cannot silently disable certificate checks.
    const child = spawn('curl', ['-q', '--proto', '=https', '--head', '--silent', '--show-error', '--connect-timeout', '4', '--max-time', '7', '--output', os.devNull, '--write-out', writeOut, `https://${host}/`], {
      shell: false, windowsHide: true, stdio: ['ignore', 'pipe', 'ignore'],
    })
    let output = ''
    let settled = false
    const timer = setTimeout(() => {
      if (settled) return
      settled = true
      child.kill('SIGKILL')
      report.curl = { status: 'error', exit_code: 28 }
      resolve()
    }, 9000)
    child.stdout.on('data', chunk => { if (output.length < 8192) output += chunk.toString() })
    child.once('error', () => { if (settled) return; settled = true; clearTimeout(timer); report.curl = { status: 'unsupported', exit_code: -1 }; resolve() })
    child.once('close', code => {
      if (settled) return
      settled = true
      clearTimeout(timer)
      let data = {}
      try { data = JSON.parse(output) } catch { /* Never print curl stderr or an unparsed response. */ }
      const http = Number(data.http_status), ssl = Number(data.ssl_verify_result), time = Number(data.time_total)
      const metrics = {
        ...(Number.isInteger(http) && http >= 100 && http <= 599 ? { http_status: http } : {}),
        ...(Number.isInteger(ssl) && ssl >= 0 ? { ssl_verify_result: ssl } : {}),
        ...(Number.isFinite(time) && time >= 0 ? { time_ms: Math.round(time * 1000) } : {}),
        ...(typeof data.peer_ip === 'string' && /^[a-fA-F0-9:.]{2,64}$/.test(data.peer_ip) ? { peer_ip: data.peer_ip } : {}),
      }
      report.curl = { status: code === 0 && http >= 100 && ssl === 0 ? 'ok' : 'error', exit_code: Number.isInteger(code) ? code : -1, ...metrics }
      resolve()
    })
  })
}

await Promise.all([checkDns(), checkCurl()])
process.stdout.write(`${JSON.stringify(report, null, 2)}\n`)
