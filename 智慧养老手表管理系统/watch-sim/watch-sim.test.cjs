const { test } = require('node:test')
const assert = require('node:assert/strict')
const fs = require('node:fs')
const os = require('node:os')
const path = require('node:path')
const http = require('node:http')
const { spawn } = require('node:child_process')

async function fixture (t) {
  const data = fs.mkdtempSync(path.join(os.tmpdir(), 'suining-simulator-'))
  fs.mkdirSync(path.join(data, 'runtime'))
  let requests = 0
  const server = http.createServer((request, response) => {
    requests++; request.resume()
    response.setHeader('Content-Type', 'application/json')
    response.end(JSON.stringify({ returnCode: '0004', message: '验收设备未注册' }))
  })
  await new Promise(resolve => server.listen(0, '127.0.0.1', resolve))
  const origin = 'http://127.0.0.1:' + server.address().port
  const record = path.join(data, 'runtime', 'server.json')
  t.after(async () => {
    await new Promise(resolve => server.close(resolve))
    if (fs.existsSync(record)) fs.unlinkSync(record)
    fs.rmdirSync(path.dirname(record)); fs.rmdirSync(data)
  })
  return { data, record, origin, requests: () => requests }
}
function run (data, args = []) {
  return new Promise((resolve, reject) => {
    const child = spawn(process.execPath, [path.join(__dirname, 'watch-sim.js'), '--once', '--deviceNo', 'TEST', '--secret', 'a'.repeat(32), ...args], {
      windowsHide: true, env: { ...process.env, SUINING_USER_DATA: data }
    })
    let output = ''
    child.stdout.on('data', chunk => { output += chunk })
    child.stderr.on('data', chunk => { output += chunk })
    const timer = setTimeout(() => { child.kill(); reject(new Error('Simulator did not terminate')) }, 10000)
    child.on('error', reject)
    child.on('exit', code => { clearTimeout(timer); resolve({ code, output }) })
  })
}
test('simulator discovers the live desktop port without a private token', async t => {
  const f = await fixture(t)
  fs.writeFileSync(f.record, JSON.stringify({ port: Number(new URL(f.origin).port), pid: process.pid, desktopPid: process.pid }))
  const result = await run(f.data)
  assert.equal(f.requests(), 1)
  assert.ok(result.output.includes(f.origin))
  assert.ok(result.output.includes('验收设备未注册'))
})
test('explicit server overrides a stale discovery record', async t => {
  const f = await fixture(t)
  fs.writeFileSync(f.record, JSON.stringify({ port: 1, pid: 2147483647, desktopPid: 2147483647 }))
  const result = await run(f.data, ['--server', f.origin])
  assert.equal(f.requests(), 1)
  assert.ok(result.output.includes(f.origin))
})
test('stale discovery fails clearly instead of contacting an unrelated port', async t => {
  const f = await fixture(t)
  fs.writeFileSync(f.record, JSON.stringify({ port: Number(new URL(f.origin).port), pid: 2147483647, desktopPid: 2147483647 }))
  const result = await run(f.data)
  assert.equal(result.code, 1)
  assert.equal(f.requests(), 0)
  assert.ok(result.output.includes('端口记录已失效'))
})
