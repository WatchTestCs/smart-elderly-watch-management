const fs = require('node:fs')
const path = require('node:path')
const crypto = require('node:crypto')
const root = path.resolve(__dirname, '..')
const metadata = JSON.parse(fs.readFileSync(path.join(__dirname, 'version.json'), 'utf8'))
const version = process.env.SUINING_VERSION || metadata.version
const channel = process.env.SUINING_CHANNEL || metadata.channel
const output = path.resolve(process.env.SUINING_OUTPUT_DIR || path.join(root, `outputs/v${version}-${channel}`))
const tools = path.join(output, 'tools')
const buildTools = fs.existsSync(path.join(__dirname,'build-tools/node_modules')) ? path.join(__dirname,'build-tools') : path.join(output,'build-tools')
const requireBuild = require('node:module').createRequire(path.join(buildTools,'package.json'))
const asar = requireBuild('@electron/asar')
const { flipFuses, FuseVersion, FuseV1Options: F } = requireBuild('@electron/fuses')
const PE = requireBuild('pe-library')
const RE = requireBuild('resedit')

async function build () {
  const target = path.join(output, 'unsigned/Suining')
  fs.rmSync(target, { recursive: true, force: true })
  fs.mkdirSync(target, { recursive: true })
  fs.cpSync(path.join(tools, 'electron'), target, { recursive: true })
  const exe = path.join(target, 'Suining.exe')
  fs.renameSync(path.join(target, 'electron.exe'), exe)
  const resources = path.join(target, 'resources')
  fs.rmSync(path.join(resources, 'default_app.asar'), { force: true })

  const staging = path.join(output, 'app-staging')
  fs.rmSync(staging, { recursive: true, force: true })
  fs.mkdirSync(staging, { recursive: true })
  for (const file of ['package.json', 'main.cjs']) fs.copyFileSync(path.join(__dirname, file), path.join(staging, file))

  const jre = fs.readdirSync(path.join(tools, 'jre')).find(name => name.endsWith('-jre'))
  if (!jre) throw new Error('Verified JRE directory not found')
  fs.cpSync(path.join(tools, 'jre', jre), path.join(resources, 'runtime'), { recursive: true })
  fs.mkdirSync(path.join(resources, 'server'), { recursive: true })

  const backendJar = path.join(root, `careold/target/careold-${version}.jar`)
  if (!fs.existsSync(backendJar)) throw new Error(`Backend artifact not found: ${backendJar}`)
  fs.copyFileSync(backendJar, path.join(resources, 'server/suining-server.jar'))
  fs.copyFileSync(path.join(__dirname, 'suining.ico'), path.join(resources, 'suining.ico'))

  const protectedFiles = ['server/suining-server.jar','runtime/bin/java.exe','runtime/bin/server/jvm.dll','runtime/lib/modules']
  const manifest = Object.fromEntries(protectedFiles.map(file => [file,crypto.createHash('sha256').update(fs.readFileSync(path.join(resources,file))).digest('hex')]))
  fs.writeFileSync(path.join(staging,'integrity.json'),JSON.stringify(manifest))
  await asar.createPackage(staging, path.join(resources, 'app.asar'))

  const executable = PE.NtExecutable.from(fs.readFileSync(exe), { ignoreCert: true })
  const resource = PE.NtExecutableResource.from(executable)
  const icon = RE.Data.IconFile.from(fs.readFileSync(path.join(__dirname, 'suining.ico')))
  const groups = RE.Resource.IconGroupEntry.fromEntries(resource.entries)
  for (const group of groups) RE.Resource.IconGroupEntry.replaceIconsForResource(resource.entries, group.id, group.lang, icon.icons.map(item => item.data))

  const numericVersion = version.split('.').map(Number)
  while (numericVersion.length < 4) numericVersion.push(0)
  if (numericVersion.slice(0,4).some(number => !Number.isInteger(number) || number < 0)) throw new Error(`Invalid numeric product version: ${version}`)

  const vi = RE.Resource.VersionInfo.fromEntries(resource.entries)[0]
  vi.setFileVersion(...numericVersion.slice(0,4),1033)
  vi.setProductVersion(...numericVersion.slice(0,4),1033)
  vi.setStringValues({ lang:1033, codepage:1200 }, {
    CompanyName:'WatchTestCs',
    FileDescription:'岁宁 · 智慧养老与家人照护',
    ProductName:'岁宁',
    OriginalFilename:'Suining.exe',
    InternalName:'Suining',
    ProductVersion:version,
    FileVersion:version,
    LegalCopyright:'Copyright WatchTestCs. Third-party licenses included.'
  })
  vi.outputToResourceEntries(resource.entries)

  const header = asar.getRawHeader(path.join(resources, 'app.asar')).headerString
  const integrity = crypto.createHash('sha256').update(header).digest('hex')
  resource.entries.push({
    type:'INTEGRITY', id:'ELECTRONASAR', lang:1033, codepage:1200,
    bin:new TextEncoder().encode(JSON.stringify([{ file:'resources\\app.asar', alg:'SHA256', value:integrity }])).buffer
  })
  resource.outputResource(executable)
  fs.writeFileSync(exe, Buffer.from(executable.generate()))

  await flipFuses(exe, { version:FuseVersion.V1,
    [F.RunAsNode]:false, [F.EnableCookieEncryption]:true,
    [F.EnableNodeOptionsEnvironmentVariable]:false, [F.EnableNodeCliInspectArguments]:false,
    [F.EnableEmbeddedAsarIntegrityValidation]:true, [F.OnlyLoadAppFromAsar]:true,
    [F.GrantFileProtocolExtraPrivileges]:false })

  console.log(JSON.stringify({ version, channel, output, executable:exe, asarHeaderHash:integrity }))
}

build().catch(error => { console.error(error); process.exitCode=1 })
