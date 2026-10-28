// Production output inventory. No source rewriting and no performance claim.
const fs = require('node:fs')
const path = require('node:path')
const zlib = require('node:zlib')
const crypto = require('node:crypto')
const [dist, output] = process.argv.slice(2)
if (!dist || !output) throw new Error('usage: node resource-sizes.cjs DIST OUTPUT.json')
function walk(dir) {
  return fs.readdirSync(dir, { withFileTypes: true }).flatMap(item => {
    const full = path.join(dir, item.name)
    return item.isDirectory() ? walk(full) : [full]
  })
}
const files = walk(dist).filter(file => /\.(?:js|css|html)$/.test(file)).sort().map(file => {
  const bytes = fs.readFileSync(file)
  return { file: path.relative(dist, file).replaceAll('\\', '/'), raw: bytes.length,
    gzip: zlib.gzipSync(bytes, { level: 9, mtime: 0 }).length,
    brotli: zlib.brotliCompressSync(bytes, { params: { [zlib.constants.BROTLI_PARAM_QUALITY]: 11 } }).length,
    sha256: crypto.createHash('sha256').update(bytes).digest('hex') }
})
const totals = files.reduce((sum, file) => {
  for (const key of ['raw', 'gzip', 'brotli']) sum[key] += file[key]
  return sum
}, { raw: 0, gzip: 0, brotli: 0 })
fs.writeFileSync(output, JSON.stringify({ compression: { gzipLevel: 9, brotliQuality: 11 },
  scope: 'All emitted JS/CSS/HTML; route cumulative requests measured separately in browser baseline', files, totals }, null, 2) + '\n')
console.log(JSON.stringify({ files: files.length, totals }))
