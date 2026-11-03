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
    response.statusCode = 401;
    response.end(JSON.stringify({ returnCode: '0000', code: 'DEVICE_AUTH_FAILED', message: '验收设备未注册' }))
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


const crypto = require('node:crypto')
const { createState, buildPayload, postJson, submitPendingSOS, acknowledgeFall } = require('./watch-sim.js')

test('demo visits fall and SOS, with exactly one pending SOS per phase', () => {
  const config = { startPoint: { lng: 113.0625, lat: 28.1975 }, scenario: 'demo', deviceNo: 'WATCH001', secret: 'a'.repeat(32) }
  const state = createState(config)
  const phases = []
  let incident
  for (let i = 0; i < 5; i++) {
    state.phaseLeft = 1
    const payload = buildPayload(config, state)
    phases.push(state.phase)
    assert.equal(payload.positionSource, 'simulated')
    if (state.phase === 'fall') {
      assert.equal(payload.alarm, 'fall')
      const id = payload.requestId
      state.phaseLeft = 3
      const retry = buildPayload(config, state)
      assert.equal(retry.alarm, 'fall', 'unacknowledged fall must be retried')
      assert.equal(retry.requestId, id, 'retry must preserve the idempotency key')
      assert.equal(state.pendingFalls.length, 1, 'a single fall cannot create duplicate incident IDs')
      assert.equal(acknowledgeFall(state, retry, { returnCode: '1111' }), true)
      const later = buildPayload(config, state)
      assert.equal(later.alarm, undefined, 'acknowledged fall must not be reported again')
    }
    if (state.phase === 'sos') {
      assert.ok(state.pendingSOS)
      incident = state.pendingSOS.requestId
      assert.match(incident, /^[a-f0-9]{32}$/)
      buildPayload(config, state)
      assert.equal(state.pendingSOS.requestId, incident, 'same phase must not create another incident')
    }
  }
  assert.deepEqual(phases, ['highHr', 'recovery', 'fall', 'sos', 'normal'])
  assert.equal(state.pendingSOS.requestId, incident, 'unacknowledged SOS survives phase change')
})

test('fall alarm survives rejection, lost acknowledgement, and phase changes without changing its ID', () => {
  const config = { startPoint: { lng: 113.0625, lat: 28.1975 }, scenario: 'demo', deviceNo: 'WATCH001', secret: 'a'.repeat(32) }
  const state = createState(config)
  state.phase = 'fall'
  state.phaseLeft = 4
  const first = buildPayload(config, state)
  assert.equal(first.alarm, 'fall')
  assert.match(first.requestId, /^[0-9a-f]{32}$/)
  assert.equal(acknowledgeFall(state, first, { returnCode: '0000' }), false)
  const retry = buildPayload(config, state)
  assert.equal(retry.requestId, first.requestId)
  assert.equal(retry.lng, first.lng)
  assert.equal(retry.lat, first.lat)

  state.phase = 'normal'
  state.phaseLeft = 4
  const late = buildPayload(config, state)
  assert.equal(late.alarm, 'fall', 'outstanding alarm must survive a return to normal phase')
  assert.equal(late.requestId, first.requestId)

  state.phase = 'fall'
  state.phaseLeft = 4
  const nextFall = buildPayload(config, state)
  assert.equal(nextFall.requestId, first.requestId, 'FIFO retries the oldest alarm first')
  assert.equal(state.pendingFalls.length, 2, 'separate falls must not be conflated')
  assert.notEqual(state.pendingFalls[1].requestId, first.requestId)
  assert.equal(acknowledgeFall(state, { ...first, requestId: 'wrong-id' }, { returnCode: '1111' }), false)
  assert.equal(acknowledgeFall(state, nextFall, { returnCode: '1111' }), true)
  const second = buildPayload(config, state)
  assert.equal(second.requestId, state.pendingFalls[0].requestId)
  assert.notEqual(second.requestId, first.requestId)
  assert.equal(acknowledgeFall(state, second, { returnCode: '1111' }), true)
  assert.equal(state.pendingFalls.length, 0)
  assert.equal(buildPayload(config, state).alarm, undefined)
})

test('SOS retries with the same requestId until the backend acknowledges it', async () => {
  const config = { server: 'http://127.0.0.1:1', deviceNo: 'WATCH001', secret: 'a'.repeat(32) }
  const state = createState({ startPoint: { lng: 113, lat: 28 } })
  state.pendingSOS = { requestId: 'stable-test-request', lng: 113, lat: 28 }
  const calls = []
  const fakePost = async (server, path, payload) => {
    calls.push({ server, path, payload })
    if (calls.length === 1) throw new Error('lost acknowledgement')
    return { returnCode: '1111', incidentId: 42 }
  }
  await assert.rejects(submitPendingSOS(config, state, 1, fakePost), /lost acknowledgement/)
  assert.equal(state.pendingSOS.requestId, 'stable-test-request')
  const ack = await submitPendingSOS(config, state, 1, fakePost)
  assert.equal(ack.incidentId, 42)
  assert.equal(state.pendingSOS, null)
  assert.equal(calls.length, 2)
  assert.deepEqual(calls.map(c => c.payload.requestId), ['stable-test-request', 'stable-test-request'])
  assert.ok(calls.every(c => c.payload.bindingVersion === 1 && c.payload.positionSource === 'simulated'))
})

test('device report uses the backend HMAC-SHA256 signature contract', async t => {
  const secret = 'a'.repeat(32)
  let received
  const server = http.createServer((req, res) => {
    const chunks = []
    req.on('data', chunk => chunks.push(chunk))
    req.on('end', () => {
      received = { headers: req.headers, bytes: Buffer.concat(chunks) }
      res.setHeader('Content-Type', 'application/json')
      res.end(JSON.stringify({ returnCode: '1111' }))
    })
  })
  await new Promise(resolve => server.listen(0, '127.0.0.1', resolve))
  t.after(() => new Promise(resolve => server.close(resolve)))
  const response = await postJson('http://127.0.0.1:' + server.address().port, 'api/v1/device/report', {
    deviceNo: 'WATCH001', secret, hr: 75
  })
  assert.equal(response.returnCode, '1111')
  assert.ok(received)
  const timestamp = received.headers['x-device-timestamp']
  const nonce = received.headers['x-device-nonce']
  assert.match(timestamp, /^\d{10}$/)
  assert.match(nonce, /^[0-9a-f]{32}$/)
  const digest = crypto.createHash('sha256').update(received.bytes).digest('hex')
  const expected = crypto.createHmac('sha256', secret).update('WATCH001:' + timestamp + ':' + nonce + ':' + digest).digest('hex')
  assert.equal(received.headers['x-device-signature'], expected)
})
