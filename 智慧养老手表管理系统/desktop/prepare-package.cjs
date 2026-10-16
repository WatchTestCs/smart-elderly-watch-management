const fs = require('node:fs')
const path = require('node:path')
const crypto = require('node:crypto')
const root = path.resolve(__dirname, '..')
const output = path.join(root, 'outputs/desktop-release-20261015')
const tools = path.join(output, 'tools')
const buildTools = fs.existsSync(path.join(__dirname,'build-tools/node_modules')) ? path.join(__dirname,'build-tools') : path.join(output,'build-tools')
const requireBuild = require('node:module').createRequire(path.join(buildTools,'package.json'))
const asar = requireBuild('@electron/asar')
const { flipFuses, FuseVersion, FuseV1Options: F } = requireBuild('@electron/fuses')
const PE = requireBuild('pe-library')
const RE = requireBuild('resedit')

async function build () {
  const target = path.join(output, 'release/Suining')
  fs.mkdirSync(target, { recursive: true })
  fs.cpSync(path.join(tools, 'electron'), target, { recursive: true })
  const exe = path.join(target, 'Suining.exe')
  fs.renameSync(path.join(target, 'electron.exe'), exe)
  const resources = path.join(target, 'resources')
  fs.rmSync(path.join(resources, 'default_app.asar'), { force: true })
  const staging = path.join(output, 'app-staging')
  fs.mkdirSync(staging, { recursive: true })
  for (const file of ['package.json', 'main.cjs', 'runtime-files.cjs']) fs.copyFileSync(path.join(__dirname, file), path.join(staging, file))
  const jre = fs.readdirSync(path.join(tools, 'jre')).find(name => name.endsWith('-jre'))
  fs.cpSync(path.join(tools, 'jre', jre), path.join(resources, 'runtime'), { recursive: true })
  fs.mkdirSync(path.join(resources, 'server'), { recursive: true })
  fs.copyFileSync(path.join(root, 'careold/target/careold-2.0.0.jar'), path.join(resources, 'server/suining-server.jar'))
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
  const vi = RE.Resource.VersionInfo.fromEntries(resource.entries)[0]
  vi.setFileVersion(2,0,0,0,1033)
  vi.setProductVersion(2,0,0,0,1033)
  vi.setStringValues({ lang:1033, codepage:1200 }, { CompanyName:'WatchTestCs', FileDescription:'岁宁 · 智慧养老与家人照护', ProductName:'岁宁', OriginalFilename:'Suining.exe', InternalName:'Suining', LegalCopyright:'Copyright WatchTestCs. Third-party licenses included.' })
  vi.outputToResourceEntries(resource.entries)
  const header = asar.getRawHeader(path.join(resources, 'app.asar')).headerString
  const integrity = crypto.createHash('sha256').update(header).digest('hex')
  resource.entries.push({ type:'INTEGRITY', id:'ELECTRONASAR', lang:1033, codepage:1200,
    bin:new TextEncoder().encode(JSON.stringify([{ file:'resources\\app.asar', alg:'SHA256', value:integrity }])).buffer })
  resource.outputResource(executable)
  fs.writeFileSync(exe, Buffer.from(executable.generate()))
  await flipFuses(exe, { version:FuseVersion.V1,
    [F.RunAsNode]:false, [F.EnableCookieEncryption]:true,
    [F.EnableNodeOptionsEnvironmentVariable]:false, [F.EnableNodeCliInspectArguments]:false,
    [F.EnableEmbeddedAsarIntegrityValidation]:true, [F.OnlyLoadAppFromAsar]:true,
    [F.GrantFileProtocolExtraPrivileges]:false })
  console.log(JSON.stringify({ executable:exe, asarHeaderHash:integrity }))
}
build().catch(error => { console.error(error); process.exitCode=1 })
