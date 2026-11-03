import path from 'node:path'
import ts from 'typescript'
import { exportDocument, generateTypes, generatedPath } from './api-contract.mjs'

const fixturePath = path.join(path.dirname(generatedPath), '__contract_drift_probe__.ts')
const fixture = `
import { postContract, type JsonReply } from './contractClient'
postContract('/api/v1/health/latest', { oldId: 1 })
declare const reply: JsonReply<'/api/v1/health/latest'>
const metric: string = reply.items[0].sample.metric
const device: number | null = reply.items[0].deviceId
const timestamp: string | undefined = reply.items[0].sample.metadata?.timestampBasis
`

function compile(generated) {
  const options = {
    strict: true, noEmit: true, skipLibCheck: true, types: [],
    target: ts.ScriptTarget.ES2022, module: ts.ModuleKind.ESNext,
    moduleResolution: ts.ModuleResolutionKind.Bundler, esModuleInterop: true
  }
  const host = ts.createCompilerHost(options)
  const originalSource = host.getSourceFile.bind(host)
  const originalExists = host.fileExists.bind(host)
  host.fileExists = file => path.resolve(file) === fixturePath || originalExists(file)
  host.getSourceFile = (file, languageVersion, onError, shouldCreate) => {
    const resolved = path.resolve(file)
    if (resolved === fixturePath || resolved === generatedPath) {
      return ts.createSourceFile(file, resolved === fixturePath ? fixture : generated, languageVersion, true)
    }
    return originalSource(file, languageVersion, onError, shouldCreate)
  }
  const program = ts.createProgram([fixturePath], options, host)
  if (program.getSyntacticDiagnostics().length) throw new Error('Contract probe contains syntax errors')
  return ts.getPreEmitDiagnostics(program)
}

const original = exportDocument()
const baseline = compile(await generateTypes(original))
if (baseline.length) {
  console.error(ts.formatDiagnosticsWithColorAndContext(baseline, {
    getCurrentDirectory: () => process.cwd(), getCanonicalFileName: file => file, getNewLine: () => '\n'
  }))
  throw new Error('Current generated contract must compile before testing drift')
}
console.log('Current Go contract and typed consumer compile')
const latest = document => document.paths['/api/v1/health/latest'].post
for (const [name, mutate] of [
  ['removed route', document => {
    document.paths['/api/v1/health/latest-renamed'] = document.paths['/api/v1/health/latest']
    delete document.paths['/api/v1/health/latest']
  }],
  ['changed request field type', document => {
    latest(document).requestBody.content['application/json'].schema.properties.oldId.type = 'string'
  }],
  ['renamed required response field', document => {
    const sample = latest(document).responses['200'].content['application/json'].schema.properties.items.items.properties.sample
    sample.properties.metricName = sample.properties.metric
    delete sample.properties.metric
    sample.required = sample.required.map(name => name === 'metric' ? 'metricName' : name)
  }],
  ['changed metadata value type', document => {
    const sample = latest(document).responses['200'].content['application/json'].schema.properties.items.items.properties.sample
    sample.properties.metadata.additionalProperties.type = 'integer'
  }]
]) {
  const changed = structuredClone(original)
  mutate(changed)
  const diagnostics = compile(await generateTypes(changed))
  if (!diagnostics.some(item => path.resolve(item.file?.fileName || '') === fixturePath)) {
    throw new Error(`Typed consumer did not reject API drift: ${name}`)
  }
  console.log(`Expected compile rejection: ${name}`)
}
