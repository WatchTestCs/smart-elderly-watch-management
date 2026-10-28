const fs = require('node:fs')
const path = require('node:path')
const { createRequire } = require('node:module')
const root = path.resolve(__dirname, '../..')
const ui = path.join(root, '智慧养老手表管理系统/oldIot')
const requireUI = createRequire(path.join(ui, 'package.json'))
const postcss = createRequire(requireUI.resolve('@vue/compiler-sfc'))('postcss')
const evidence = process.argv[2]
if (!evidence) throw new Error('usage: node inventory.cjs EVIDENCE')
const read = file => fs.readFileSync(file, 'utf8').replace(/^\uFEFF/, '')
const write = (file, data) => fs.writeFileSync(path.join(evidence, file), JSON.stringify(data, null, 2) + '\n')
const cssFiles = ['src/assets/css/element-theme.css', 'src/assets/css/commercial.css']
const declarations = []
const cssCounts = cssFiles.map(file => {
  let important = 0, rules = 0
  postcss.parse(read(path.join(ui, file)), { from: file }).walkRules(rule => {
    rules++
    const media = []
    for (let parent = rule.parent; parent; parent = parent.parent) if (parent.type === 'atrule') media.push('@' + parent.name + ' ' + parent.params)
    rule.walkDecls(decl => {
      if (decl.important) important++
      for (const selector of rule.selectors) declarations.push({ file, line: decl.source.start.line, selector, property: decl.prop, value: decl.value, important: Boolean(decl.important), media })
    })
  })
  return { file, rules, important }
})
const overlaps = []
for (const first of declarations.filter(d => d.file === cssFiles[0])) {
  for (const second of declarations.filter(d => d.file === cssFiles[1] && d.selector === first.selector && d.property === first.property)) overlaps.push({ first, second })
}
write('css-inventory.json', { note: 'Same-selector/property overlaps across files; responsive overrides can be intentional. This is not proof of a UI defect.', counts: cssCounts, overlaps })
const spec = JSON.parse(read(path.join(root, '智慧养老手表管理系统/backend/internal/apidoc/openapi.json')))
const groups = { session: ['/restful/userLogin/'], devices: ['/restful/device/'], health: ['/restful/ratepre/', '/restful/step/', '/restful/position/', '/restful/health/'], customers: ['/restful/customer/'] }
const contracts = Object.fromEntries(Object.entries(groups).map(([name, prefixes]) => [name, Object.entries(spec.paths).filter(([url]) => prefixes.some(p => url.startsWith(p))).flatMap(([url, methods]) => Object.entries(methods).map(([method, operation]) => {
  const response = operation.responses?.['200']?.content?.['application/json']?.schema
  return { url, method, response, request: operation.requestBody?.content?.['application/json']?.schema, explicit200Properties: Object.keys(response?.properties || {}), security: operation.security }
}))]))
write('api-contract-inventory.json', { pathsTotal: Object.keys(spec.paths).length, note: 'Route/authorization inventory exists. Open legacy response schemas cannot generate precise DTO/nullability contracts.', contracts })
const qaDir = fs.existsSync(path.join(evidence, 'qa')) ? path.join(evidence, 'qa') : path.join(ui, 'qa/artifacts')
const qa = fs.readdirSync(qaDir).filter(f => f.endsWith('.json')).sort().map(file => {
  const report = JSON.parse(read(path.join(qaDir, file)))
  const checks = Array.isArray(report) ? report : report.checks
  if (!Array.isArray(checks)) throw new Error('Unknown QA artifact format: ' + file)
  return { file, checks: checks.length, failed: checks.filter(c => c.pass !== true), scope: file.startsWith('real-backend-') ? 'actual Go/MySQL' : 'API mocks' }
})
write('qa-summary.json', qa)
const baseline = JSON.parse(read(path.join(evidence, 'browser-baseline.json')))
const resources = JSON.parse(read(path.join(evidence, 'resource-sizes.json')))
const totals = files => files.reduce((sum, file) => { for (const key of ['raw', 'gzip', 'brotli']) sum[key] += file[key]; return sum }, { raw: 0, gzip: 0, brotli: 0 })
const routes = ['/login', '/rateprefamily', '/positionfamily'].map(route => {
  const samples = baseline.samples.filter(s => s.route === route)
  const names = new Set(samples.flatMap(s => s.resources.map(r => r.name.replace(/^\//, ''))).filter(n => /\.(?:js|css)$/.test(n)))
  const files = resources.files.filter(f => names.has(f.file))
  return { route, scope: 'Cumulative emitted JS/CSS requested up to list readiness, before detail action; excludes online map tiles and font/image binary files', files, totals: totals(files),
    samples: samples.map(s => ({ cache: s.cache, sample: s.sample, jsCssTransferBytes: s.resources.filter(r => /\.(?:js|css)$/.test(r.name)).reduce((sum, r) => sum + r.transferSize, 0), cachedJsCssEntries: s.resources.filter(r => /\.(?:js|css)$/.test(r.name) && r.transferSize === 0 && r.decodedBodySize > 0).length })) }
})
write('route-resources.json', routes)
console.log(JSON.stringify({ qa: qa.map(q => ({ ...q, failed: q.failed.length })), cssCounts, cssOverlaps: overlaps.length, routes: routes.map(r => ({ route: r.route, totals: r.totals })), contractOperations: Object.fromEntries(Object.entries(contracts).map(([k, v]) => [k, v.length])) }, null, 2))
