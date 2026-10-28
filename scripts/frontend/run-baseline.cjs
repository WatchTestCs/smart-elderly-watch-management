// Cross-platform baseline entry point. Dependencies are locked by the project;
// choose Node/NPM/browser externally. Never reuse an evidence directory.
const fs = require('node:fs')
const path = require('node:path')
const crypto = require('node:crypto')
const { spawnSync, execFileSync } = require('node:child_process')
const [npmCLI, output] = process.argv.slice(2)
if (!npmCLI || !output) throw new Error('usage: node scripts/frontend/run-baseline.cjs NPM_CLI NEW_EVIDENCE_DIRECTORY (optional CARE_CHROME_PATH)')
const root = path.resolve(__dirname, '../..')
const ui = path.join(root, '智慧养老手表管理系统/oldIot')
const out = path.resolve(output)
const npm = path.resolve(npmCLI)
if (!fs.existsSync(npm)) throw new Error('NPM_CLI must identify the actual npm-cli.js')
fs.mkdirSync(out)
const env = { ...process.env, PATH: path.dirname(process.execPath) + path.delimiter + process.env.PATH,
  npm_config_registry: 'https://registry.npmjs.org/', FRONTEND_EVIDENCE: out }
const hash = file => crypto.createHash('sha256').update(fs.readFileSync(file)).digest('hex')
const git = (...args) => execFileSync('git', args, { cwd: root, encoding: 'utf8' }).trim()
const metadata = { sourceSHA: git('rev-parse', 'HEAD'), sourceStatus: git('status', '--short'),
  lockSHA256Before: hash(path.join(ui, 'package-lock.json')), node: process.version,
  npm: execFileSync(process.execPath, [npm, '--version'], { encoding: 'utf8' }).trim(),
  platform: process.platform, arch: process.arch, timeUTC: new Date().toISOString() }
const records = []
const write = (name, data) => fs.writeFileSync(path.join(out, name), JSON.stringify(data, null, 2) + '\n')
function run(name, args, cwd = ui) {
  const log = fs.openSync(path.join(out, name + '.txt'), 'w')
  const started = Date.now()
  let result
  try { result = spawnSync(process.execPath, args, { cwd, env, stdio: ['ignore', log, log], timeout: 600000 }) }
  finally { fs.closeSync(log) }
  records.push({ name, command: [process.execPath, ...args], cwd, exit: result.status, seconds: (Date.now() - started) / 1000,
    ...(result.error ? { error: String(result.error) } : {}) })
  write('commands.json', records)
  console.log(JSON.stringify(records.at(-1)))
  if (result.status !== 0 || result.error) throw new Error('Baseline command failed: ' + name)
}
try {
  write('environment.json', metadata)
  for (const [name, args] of [['npm-ci', ['ci']], ['npm-ls', ['ls', '--all', '--json']], ['unit', ['test']],
    ['typecheck', ['run', 'typecheck']], ['build-1', ['run', 'build']]]) run(name, [npm, ...args])
  run('resource-sizes-1', [path.join(__dirname, 'resource-sizes.cjs'), path.join(ui, 'dist'), path.join(out, 'resource-sizes-1.json')])
  run('build-2', [npm, 'run', 'build'])
  run('resource-sizes', [path.join(__dirname, 'resource-sizes.cjs'), path.join(ui, 'dist'), path.join(out, 'resource-sizes.json')])
  metadata.buildFilesIdentical = hash(path.join(out, 'resource-sizes-1.json')) === hash(path.join(out, 'resource-sizes.json'))
  if (!metadata.buildFilesIdentical) throw new Error('Repeated builds differ; investigate before baseline acceptance')
  run('audit', [npm, 'audit', '--audit-level=high', '--json'])
  run('qa-npm-ci', [npm, 'ci'], path.join(ui, 'qa'))
  run('browser-baseline', [path.join(__dirname, 'browser-baseline.cjs')], root)
  run('qa-run', [path.join(ui, 'qa/run.cjs')], root)
  fs.mkdirSync(path.join(out, 'qa'))
  for (const name of ['browser-acceptance', 'controls-acceptance', 'forms-acceptance', 'lifecycle-acceptance', 'screen-acceptance',
    'setup-browser-acceptance', 'issues-acceptance', 'elder-delete', 'database-availability', 'ui-refinement', 'architecture-acceptance']) {
    fs.copyFileSync(path.join(ui, 'qa/artifacts', name + '.json'), path.join(out, 'qa', name + '.json'))
  }
  run('inventory', [path.join(__dirname, 'inventory.cjs'), out], root)
} catch (error) {
  metadata.failure = error.stack
  console.error(error)
  process.exitCode = 1
} finally {
  metadata.lockSHA256After = hash(path.join(ui, 'package-lock.json'))
  metadata.lockUnchanged = metadata.lockSHA256Before === metadata.lockSHA256After
  write('environment.json', metadata)
  if (!metadata.lockUnchanged) process.exitCode = 1
}
