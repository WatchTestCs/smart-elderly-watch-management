// Deterministic mechanism reproduction only. It does not change the scheduler,
// pretend a hang is fixed, or send any business mutation.
import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath, pathToFileURL } from 'node:url'
const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../..')
const { createLatestRead } = await import(pathToFileURL(path.join(root, '智慧养老手表管理系统/oldIot/src/composables/latestRead.ts')))
const rows = []
for (let repeat = 1; repeat <= 3; repeat++) {
  let finish
  const held = new Promise(resolve => { finish = resolve })
  const calls = [], accepted = [], rejected = [], busy = []
  const queue = createLatestRead(query => { calls.push(query); return query === 'old' ? held : Promise.resolve(query) },
    value => accepted.push(value), error => rejected.push(String(error)), value => busy.push(value))
  const pending = queue.run('old')
  await Promise.resolve()
  queue.run('new')
  // Drain only deterministic microtasks: there is no timer or actual network.
  for (let i = 0; i < 10; i++) await Promise.resolve()
  assert.deepEqual(calls, ['old'])
  assert.deepEqual(busy, [true])
  queue.dispose()
  let settled = false
  pending.then(() => { settled = true })
  for (let i = 0; i < 10; i++) await Promise.resolve()
  assert.equal(settled, false)
  rows.push({ repeat, beforeRelease: { calls: [...calls], busy: [...busy], disposedPromiseSettled: settled },
    observed: 'unresolved active read blocks newest read; dispose prevents writes but does not end active promise' })
  finish('late')
  await pending
  assert.deepEqual(accepted, [])
  assert.deepEqual(rejected, [])
  assert.deepEqual(calls, ['old'])
  const normal = []
  const healthy = createLatestRead(async value => value, value => normal.push(value), error => { throw error }, () => {})
  await healthy.run('ok')
  assert.deepEqual(normal, ['ok'])
  rows.at(-1).neighborNormalRead = normal
}
const result = { scope: 'Current-source risk reproduction, not a fix or production incident claim', rows }
fs.writeFileSync(process.argv[2], JSON.stringify(result, null, 2) + '\n')
console.log(JSON.stringify(result, null, 2))
