const fs = require('node:fs')
const path = require('node:path')
const crypto = require('node:crypto')
const { createRequire } = require('node:module')
const { spawnSync } = require('node:child_process')

function stage (name, details = '') {
  const suffix = details ? ` — ${details}` : ''
  process.stdout.write(`[desktop-package] ${name}${suffix}\n`)
}

async function main () {
  const root = path.resolve(__dirname, '..')
  const metadata = JSON.parse(fs.readFileSync(path.join(__dirname, 'version.json'), 'utf8'))
  const version = process.env.SUINING_VERSION || metadata.version
  const channel = process.env.SUINING_CHANNEL || metadata.channel
  const output = path.resolve(process.env.SUINING_OUTPUT_DIR || path.join(root, `outputs/v${version}-${channel}`))
  const tools = path.join(output, 'tools')
  const buildTools = fs.existsSync(path.join(__dirname, 'build-tools/node_modules'))
    ? path.join(__dirname, 'build-tools')
    : path.join(output, 'build-tools')
  const requireBuild = createRequire(path.join(buildTools, 'package.json'))

  stage('load build modules', buildTools)
  const asar = requireBuild('@electron/asar')
  const { flipFuses, FuseVersion, FuseV1Options: F } = requireBuild('@electron/fuses')
  const RE = requireBuild('resedit')

  const target = path.join(output, 'unsigned/Suining')
  const electronSource = path.join(tools, 'electron')
  stage('reset unsigned Electron directory', target)
  fs.rmSync(target, { recursive: true, force: true })
  fs.mkdirSync(target, { recursive: true })
  stage('copy verified Electron runtime', target)
  if (process.platform === 'win32') {
    // Windows CI has intermittently terminated during Node's recursive copy
    // without producing a JS exception. Robocopy provides bounded retries,
    // distinct exit codes and diagnostics; 0..7 mean successful copy.
    const copy = spawnSync('robocopy', [
      electronSource, target, '/E', '/XJ', '/COPY:DAT', '/R:2', '/W:1',
      '/NJH', '/NJS', '/NDL', '/NFL', '/NP'
    ], { windowsHide: true, encoding: 'utf8', timeout: 120000 });
    if (copy.error || copy.status === null || copy.status >= 8) {
      throw new Error('Verified Electron runtime copy failed: ' +
        JSON.stringify({ status: copy.status, signal: copy.signal, error: copy.error?.message,
          stdout: copy.stdout?.slice(-2500), stderr: copy.stderr?.slice(-2500) }));
    }
    stage('verified Electron runtime copied', 'robocopy exit ' + copy.status)
  } else {
    fs.cpSync(electronSource, target, { recursive: true })
  }

  const exe = path.join(target, 'Suining.exe')
  fs.renameSync(path.join(target, 'electron.exe'), exe)
  const resources = path.join(target, 'resources')
  fs.rmSync(path.join(resources, 'default_app.asar'), { force: true })

  const staging = path.join(output, 'app-staging')
  stage('prepare ASAR staging', staging)
  fs.rmSync(staging, { recursive: true, force: true })
  fs.mkdirSync(staging, { recursive: true })
  for (const file of ['package.json', 'main.cjs', 'runtime-files.cjs']) {
    fs.copyFileSync(path.join(__dirname, file), path.join(staging, file))
  }

  const jreRoot = path.join(tools, 'jre')
  const jre = fs.readdirSync(jreRoot).find(name => name.endsWith('-jre'))
  if (!jre) throw new Error('Verified JRE directory not found')
  stage('copy verified Java runtime', jre)
  fs.cpSync(path.join(jreRoot, jre), path.join(resources, 'runtime'), { recursive: true })
  fs.mkdirSync(path.join(resources, 'server'), { recursive: true })

  const backendJar = path.join(root, `careold/target/careold-${version}.jar`)
  if (!fs.existsSync(backendJar)) throw new Error(`Backend artifact not found: ${backendJar}`)
  fs.copyFileSync(backendJar, path.join(resources, 'server/suining-server.jar'))
  fs.copyFileSync(path.join(__dirname, 'suining.ico'), path.join(resources, 'suining.ico'))

  const protectedFiles = [
    'server/suining-server.jar',
    'runtime/bin/java.exe',
    'runtime/bin/server/jvm.dll',
    'runtime/lib/modules'
  ]
  const manifest = Object.fromEntries(protectedFiles.map(file => [
    file,
    crypto.createHash('sha256').update(fs.readFileSync(path.join(resources, file))).digest('hex')
  ]))
  fs.writeFileSync(path.join(staging, 'integrity.json'), JSON.stringify(manifest))

  stage('create app.asar')
  const asarPath = path.join(resources, 'app.asar')
  await asar.createPackage(staging, asarPath)

  stage('parse Windows executable resources')
  const executable = RE.NtExecutable.from(fs.readFileSync(exe), { ignoreCert: true })
  const resource = RE.NtExecutableResource.from(executable)

  stage('replace application icon')
  const icon = RE.Data.IconFile.from(fs.readFileSync(path.join(__dirname, 'suining.ico')))
  const groups = RE.Resource.IconGroupEntry.fromEntries(resource.entries)
  if (groups.length === 0) throw new Error('Electron executable has no icon group')
  for (const group of groups) {
    RE.Resource.IconGroupEntry.replaceIconsForResource(
      resource.entries,
      group.id,
      group.lang,
      icon.icons.map(item => item.data)
    )
  }

  stage('update version information')
  const numericVersion = version.split('.').map(Number)
  while (numericVersion.length < 4) numericVersion.push(0)
  if (numericVersion.length > 4 || numericVersion.some(number => !Number.isInteger(number) || number < 0)) {
    throw new Error(`Invalid numeric product version: ${version}`)
  }

  const versionInfo = RE.Resource.VersionInfo.fromEntries(resource.entries)
  if (versionInfo.length !== 1) {
    throw new Error(`Expected one version-info resource, found ${versionInfo.length}`)
  }
  const vi = versionInfo[0]
  const languageInfo = vi.getAllLanguagesForStringValues()
  if (languageInfo.length !== 1) {
    throw new Error(`Expected one version-info language, found ${languageInfo.length}`)
  }
  vi.setFileVersion(...numericVersion)
  vi.setProductVersion(...numericVersion)
  vi.setStringValues(languageInfo[0], {
    CompanyName: 'WatchTestCs',
    FileDescription: '岁宁 · 智慧养老与家人照护',
    ProductName: '岁宁',
    OriginalFilename: 'Suining.exe',
    InternalName: 'Suining',
    ProductVersion: version,
    FileVersion: version,
    LegalCopyright: 'Copyright WatchTestCs. Third-party licenses included.'
  })
  vi.outputToResourceEntries(resource.entries)

  stage('embed ASAR header integrity')
  const header = asar.getRawHeader(asarPath).headerString
  const integrity = crypto.createHash('sha256').update(header).digest('hex')
  const integrityBuffer = Buffer.from(JSON.stringify([{
    file: 'resources\\app.asar',
    alg: 'sha256',
    value: integrity
  }]), 'utf8')
  resource.entries.push({
    type: 'INTEGRITY',
    id: 'ELECTRONASAR',
    bin: integrityBuffer.buffer.slice(
      integrityBuffer.byteOffset,
      integrityBuffer.byteOffset + integrityBuffer.length
    ),
    lang: languageInfo[0].lang,
    codepage: languageInfo[0].codepage
  })

  stage('write modified executable resources')
  resource.outputResource(executable)
  fs.writeFileSync(exe, Buffer.from(executable.generate()))

  stage('flip Electron security fuses')
  await flipFuses(exe, {
    version: FuseVersion.V1,
    [F.RunAsNode]: false,
    [F.EnableCookieEncryption]: true,
    [F.EnableNodeOptionsEnvironmentVariable]: false,
    [F.EnableNodeCliInspectArguments]: false,
    [F.EnableEmbeddedAsarIntegrityValidation]: true,
    [F.OnlyLoadAppFromAsar]: true,
    [F.GrantFileProtocolExtraPrivileges]: false
  })

  stage('desktop package complete', exe)
  console.log(JSON.stringify({ version, channel, output, executable: exe, asarHeaderHash: integrity }))
}

main().catch(error => {
  console.error('[desktop-package] fatal:', error && error.stack ? error.stack : error)
  process.exitCode = 1
})
