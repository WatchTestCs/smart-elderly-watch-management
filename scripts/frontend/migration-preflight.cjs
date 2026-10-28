// Isolated dependency preflight. No force, legacy-peer-deps, overrides, or edits
// to the project's package/lock files. A successful probe run may record a
// blocked migration: read dependencyResolutionAccepted, not just script exit.
const fs = require('node:fs')
const path = require('node:path')
const crypto = require('node:crypto')
const { createRequire } = require('node:module')
const { spawnSync } = require('node:child_process')
const [npmCLI, scratch, evidence] = process.argv.slice(2)
if (!npmCLI || !scratch || !evidence) throw new Error('usage: node migration-preflight.cjs NPM_CLI NEW_SCRATCH EVIDENCE')
const semver = createRequire(path.resolve(npmCLI))('semver')
const root = path.resolve(__dirname, '../..')
const ui = path.join(root, '智慧养老手表管理系统/oldIot')
const target = '3.6.0-rc.13'
const digest = bytes => crypto.createHash('sha256').update(bytes).digest('hex')
const source = Object.fromEntries(['package.json', 'package-lock.json'].map(file => [file, fs.readFileSync(path.join(ui, file))]))
const metadata = JSON.parse(fs.readFileSync(path.join(evidence, 'registry-metadata.json'), 'utf8').replace(/^\uFEFF/, ''))
const compatibility = metadata.packages.filter(p => p.name !== 'vue').map(p => ({ name: p.name, version: p.version,
  vuePeer: p.peerDependencies?.vue || null, targetVueAccepted: p.peerDependencies?.vue ? semver.satisfies(target, p.peerDependencies.vue) : null,
  nodeEngine: p.engines?.node || null, nodeAccepted: p.engines?.node ? semver.satisfies(process.version, p.engines.node) : null }))
const core = ['vue', '@vue/shared', '@vue/runtime-core', '@vue/runtime-dom', '@vue/reactivity', '@vue/compiler-core', '@vue/compiler-dom', '@vue/compiler-sfc', '@vue/server-renderer']
const installedCore = core.map(name => ({ name, version: JSON.parse(fs.readFileSync(path.join(ui, 'node_modules', name, 'package.json'), 'utf8')).version }))
// Refuse to reuse a directory that might contain someone else's experiment.
fs.mkdirSync(scratch)
const trials = []
for (let repeat = 1; repeat <= 3; repeat++) {
  const cwd = path.join(scratch, 'trial-' + repeat)
  fs.mkdirSync(cwd)
  const pkg = JSON.parse(source['package.json'])
  pkg.dependencies.vue = target
  fs.writeFileSync(path.join(cwd, 'package.json'), JSON.stringify(pkg, null, 2) + '\n')
  fs.writeFileSync(path.join(cwd, 'package-lock.json'), source['package-lock.json'])
  const args = [path.resolve(npmCLI), 'install', '--package-lock-only', '--ignore-scripts', '--no-audit', '--no-fund']
  const started = Date.now()
  const result = spawnSync(process.execPath, args, { cwd, encoding: 'utf8', timeout: 180000,
    env: { ...process.env, PATH: path.dirname(process.execPath) + path.delimiter + process.env.PATH, npm_config_registry: 'https://registry.npmjs.org/' } })
  fs.writeFileSync(path.join(evidence, `vue36-resolution-${repeat}.txt`), (result.stdout || '') + (result.stderr || '') + (result.error ? '\n' + result.error.stack : ''))
  trials.push({ repeat, cwd, command: [process.execPath, ...args], exit: result.status, seconds: (Date.now() - started) / 1000,
    errorCode: /\bERESOLVE\b/.test(result.stderr || '') ? 'ERESOLVE' : result.error?.code || null,
    resultingLockSHA256: digest(fs.readFileSync(path.join(cwd, 'package-lock.json'))) })
  console.log(JSON.stringify(trials.at(-1)))
}
const originalFilesUnchanged = Object.entries(source).every(([file, bytes]) => digest(bytes) === digest(fs.readFileSync(path.join(ui, file))))
const report = { target, prerelease: true, compatibility, installedCore, trials, originalFilesUnchanged,
  dependencyResolutionAccepted: trials.every(t => t.exit === 0) && compatibility.every(p => p.targetVueAccepted !== false),
  note: 'Baseline stays on Vue 3.5.44. No runtime/native/performance migration acceptance inferred from this probe.' }
fs.writeFileSync(path.join(evidence, 'vue36-preflight.json'), JSON.stringify(report, null, 2) + '\n')
if (!originalFilesUnchanged) throw new Error('Project package files unexpectedly changed')
console.log(JSON.stringify({ dependencyResolutionAccepted: report.dependencyResolutionAccepted, compatibility }, null, 2))
