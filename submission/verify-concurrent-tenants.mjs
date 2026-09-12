import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import { createRequire } from 'node:module'
import { randomUUID } from 'node:crypto'
import { execFileSync } from 'node:child_process'
import path from 'node:path'

const [checkout, accountFile, base = 'http://127.0.0.1:18081/api/v1',
  origin = 'http://127.0.0.1:15173'] = process.argv.slice(2)
assert.ok(checkout && accountFile,
  'Usage: node verify-concurrent-tenants.mjs <checkout> <private-accounts.json> [api-base] [origin]')
const require = createRequire(path.resolve(checkout, 'frontend/package.json'))
const WebSocket = require('ws')
const accounts = JSON.parse(await readFile(accountFile, 'utf8'))
const sourceCommit = execFileSync('git', ['-C', checkout, 'rev-parse', 'HEAD'], { encoding: 'utf8' }).trim()
const report = { source_commit: sourceCommit, started_at: new Date().toISOString(), status: 'running', cases: [], cleanup: [] }
const owned = []
const sockets = []
async function api(account, method, route, body, expected = 200) {
  const response = await fetch(base + route, {
    method,
    headers: {
      'Content-Type': 'application/json', Origin: origin,
      ...(account.token ? { Authorization: `Bearer ${account.token}`,
        'X-Tenant-ID': String(account.tenant.id) } : {}),
    },
    body: body ? JSON.stringify(body) : undefined,
    signal: AbortSignal.timeout(60000),
  })
  assert.equal(response.status, expected, `${method} /[test-session]: HTTP ${response.status}`)
  if (expected >= 400) return
  if (route.includes('/download')) return response.text()
  return response.json()
}
class Console {
  constructor(socket) {
    this.socket = socket
    this.frames = []
    this.output = ''
    socket.on('message', (bytes, binary) => {
      if (binary) this.output += bytes.toString()
      else this.frames.push(JSON.parse(bytes.toString()))
    })
    socket.on('error', () => { this.failed = true })
  }
  send(frame) { this.socket.send(JSON.stringify(frame)) }
  async until(predicate) {
    const deadline = Date.now() + 60000
    while (Date.now() < deadline) {
      const result = predicate()
      if (result) return result
      assert.ok(!this.failed && this.socket.readyState !== WebSocket.CLOSED,
        'Console closed before expected response')
      const error = this.frames.find(frame => frame.type === 'error')
      assert.ok(!error, `Console error: ${error?.code}`)
      await new Promise(resolve => setTimeout(resolve, 25))
    }
    throw new Error('Console response timeout')
  }
  frame(type) { return this.until(() => this.frames.find(frame => frame.type === type)) }
  async close() {
    if (this.socket.readyState === WebSocket.CLOSED) return
    await new Promise(resolve => {
      const timer = setTimeout(() => { this.socket.terminate(); resolve() }, 3000)
      this.socket.once('close', () => { clearTimeout(timer); resolve() })
      this.socket.close()
    })
  }
}
async function attach(account, session) {
  const ticket = (await api(account, 'POST', `/sessions/${session}/sandbox/command-ticket`, {})).data.ticket
  const url = new URL(base + '/sandbox-terminal')
  url.protocol = url.protocol === 'https:' ? 'wss:' : 'ws:'
  const socket = new WebSocket(url, { origin, handshakeTimeout: 10000 })
  const terminal = new Console(socket)
  sockets.push(terminal)
  await new Promise((resolve, reject) => {
    socket.once('open', resolve)
    socket.once('error', reject)
  })
  terminal.send({ type: 'auth', ticket })
  await terminal.frame('ready')
  return terminal
}
const quote = value => "'" + value.replaceAll("'", "'\\''") + "'"
function command(marker) {
  return "stty -echo; python3 -u -c " + quote(`import json,os,sys
marker=${JSON.stringify(marker)}
open("/workspace/output/"+marker+".txt","w").write(marker)
print("READY",flush=True)
for line in sys.stdin:
 request=json.loads(line)
 if request.get("stop"): break
 own=False
 foreign=False
 for entry in os.listdir("/proc"):
  if not entry.isdigit(): continue
  try:
   data=open("/proc/"+entry+"/cmdline","rb").read()
   own=own or marker.encode() in data
   foreign=foreign or request["other"].encode() in data
  except OSError: pass
 print("INSPECT:"+json.dumps({"own":own,"foreign":foreign,"namespace":os.readlink("/proc/self/ns/pid")}),flush=True)
print("DONE",flush=True)`)
}
try {
  for (const name of ['alice', 'bob']) {
    const account = accounts[name]
    assert.ok(account?.email && account.password, `${name}: test credentials required`)
    const login = await api(account, 'POST', '/auth/login',
      { email: account.email, password: account.password })
    assert.ok(login.success && login.token)
    account.token = login.token
    account.tenant = login.active_tenant
    account.user = login.user
  }
  assert.notEqual(accounts.alice.tenant.id, accounts.bob.tenant.id, 'Distinct tenants')
  for (const backend of ['docker', 'e2b']) {
    const users = []
    for (const label of ['alice', 'bob']) {
      const account = accounts[label]
      assert.ok(account[backend + '_config_id'], `${backend} test configuration required`)
      const session = (await api(account, 'POST', '/sessions',
        { title: `Acceptance - Concurrent ${backend} ${label}` }, 201)).data.id
      owned.push({ account, session, backend, label })
      await api(account, 'POST', `/sessions/${session}/sandbox/workbench`,
        { config_id: account[backend + '_config_id'] })
      users.push({ account, session, marker: `wb-${randomUUID()}`, terminal: await attach(account, session) })
    }
    for (const user of users) user.terminal.send({ type: 'command', command: command(user.marker) })
    await Promise.all(users.map(user => user.terminal.until(() => user.terminal.output.includes('READY'))))
    assert.ok(users.every(user => !user.terminal.frames.some(frame => frame.type === 'exit')),
      'Both command processes must be running simultaneously')
    const inspections = []
    for (let i = 0; i < 2; i++) {
      const user = users[i], other = users[1 - i]
      user.terminal.send({ type: 'stdin',
        data: JSON.stringify({ other: other.marker }) + '\n' })
      await user.terminal.until(() => /INSPECT:\{[^\r\n]+\}/.test(user.terminal.output))
      const value = JSON.parse(user.terminal.output.match(/INSPECT:(\{[^\r\n]+\})/)[1])
      assert.equal(value.own, true)
      assert.equal(value.foreign, false)
      inspections.push(value)
      const own = await api(user.account, 'GET',
        `/sessions/${user.session}/sandbox/files/download?path=${user.marker}.txt`)
      assert.equal(own, user.marker)
      await api(user.account, 'GET',
        `/sessions/${user.session}/sandbox/files/download?path=${other.marker}.txt`, undefined, 404)
      await api(user.account, 'GET',
        `/sessions/${other.session}/sandbox/workbench`, undefined, 404)
      await api(user.account, 'GET',
        `/sessions/${other.session}/sandbox/audit`, undefined, 404)
      for (const value of ['../etc/passwd', '/etc/passwd']) {
        await api(user.account, 'GET',
          `/sessions/${user.session}/sandbox/files/download?path=${encodeURIComponent(value)}`, undefined, 400)
      }
    }
    assert.notEqual(inspections[0].namespace, inspections[1].namespace, 'Separate PID namespaces')
    for (const user of users) user.terminal.send({ type: 'stdin', data: '{"stop":true}\n' })
    for (const user of users) {
      assert.equal((await user.terminal.frame('exit')).exit_code, 0)
      const execution = (await user.terminal.frame('started')).execution_id
      const entries = (await api(user.account, 'GET', `/sessions/${user.session}/sandbox/audit`)).data
      const events = entries.filter(entry => entry.details?.execution_id === execution)
      assert.ok(events.some(entry => entry.outcome === 'accepted'))
      assert.ok(events.some(entry => entry.details?.exit_code === 0))
      assert.ok(events.every(entry => String(entry.tenant_id) === String(user.account.tenant.id)
        && entry.actor_user_id === user.account.user.id))
      await user.terminal.close()
    }
    report.cases.push({
      backend, status: 'passed', simultaneous_terminals: 2, distinct_pid_namespaces: true,
      own_process_visible: true, foreign_process_invisible: true,
      own_file_readable: true, foreign_file_invisible: true,
      cross_tenant_http_denied: true, outside_output_paths_denied: true,
      accepted_and_completion_audit_for_each_command: true,
    })
  }
  report.status = 'passed'
} catch (error) {
  report.status = 'failed'
  report.error = error.message
  process.exitCode = 1
} finally {
  for (const terminal of sockets) await terminal.close()
  for (const { account, session, backend, label } of owned) {
    try {
      await api(account, 'DELETE', `/sessions/${session}`)
      report.cleanup.push({ backend, label, session_delete: 'passed' })
    } catch {
      report.cleanup.push({ backend, label, session_delete: 'failed' })
      report.status = 'failed'
      process.exitCode = 1
    }
  }
  report.completed_at = new Date().toISOString()
  console.log(JSON.stringify(report, null, 2))
}
