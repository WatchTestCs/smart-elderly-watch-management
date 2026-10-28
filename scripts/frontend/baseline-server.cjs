// Minimal, fixed synthetic dataset for measurement. API fixtures are served by
// HTTP rather than Playwright routing, which disables the browser HTTP cache.
const fs = require('node:fs')
const http = require('node:http')
const path = require('node:path')
const zlib = require('node:zlib')
const root = path.resolve(__dirname, '../..')
const { chromium } = require(path.join(root, '智慧养老手表管理系统/oldIot/qa/node_modules/playwright'))
const dist = path.join(root, '智慧养老手表管理系统/oldIot/dist')
const mime = { '.html': 'text/html', '.js': 'application/javascript', '.css': 'text/css', '.webp': 'image/webp', '.png': 'image/png', '.svg': 'image/svg+xml', '.woff2': 'font/woff2', '.woff': 'font/woff', '.ttf': 'font/ttf' }
module.exports = async callback => {
  let signedIn = true
  const report = { pageErrors: [], missingAssets: [], unexpectedApi: [], requests: [] }
  const fixtures = {
    '/restful/service/status': { databaseReady: true },
    '/restful/text/select': { texts: [{ textId: 1, picId: 0, title: '设备使用与照护提醒', time: '2026-10-13 10:20' }] },
    '/restful/ratepre/selectRatepreFamily': { ratepreDtoList: [{ name: '李奶奶', preState: 0, rateState: 1, createTime: '2026-10-13 10:20' }] },
    '/restful/ratepre/selectRatepreDetail': { hrs: [72, 76, 73], bps: [120, 125, 122], createTimes: ['08:00', '09:00', '10:00'], time: '2026-10-13' },
    '/restful/position/selectFamily': { positionDtos: [{ name: '李奶奶', status: 1, lng: 121.4737, lat: 31.2304, beTime: '2026-10-13 10:20' }] }
  }
  const server = http.createServer((req, res) => {
    const pathname = decodeURIComponent(new URL(req.url, 'http://localhost').pathname)
    report.requests.push({ path: pathname, method: req.method })
    let data = fixtures[pathname]
    if (pathname === '/restful/userLogin/session') data = { authenticated: signedIn, account: signedIn ? { role: 'FAMILY', id: 1, oldId: 1, phone: '13000000000', displayName: '小明' } : null, csrfToken: 'baseline-csrf', csrfHeader: 'X-CSRF-TOKEN' }
    if (pathname === '/restful/userLogin/login') { signedIn = true; data = { returnCode: '1111', family: { familyName: '小明', oldId: 1, familyPhone: '13000000000' } } }
    if (pathname.startsWith('/restful/')) {
      req.resume()
      res.setHeader('Cache-Control', 'no-store')
      if (!data) { report.unexpectedApi.push(pathname); res.writeHead(404); res.end('{}'); return }
      res.setHeader('Content-Type', 'application/json')
      res.end(JSON.stringify(data)); return
    }
    const file = path.resolve(dist, '.' + (pathname === '/' ? '/index.html' : pathname))
    if (!file.startsWith(dist + path.sep)) { res.writeHead(403); res.end(); return }
    fs.readFile(file, (error, bytes) => {
      if (error) { res.writeHead(404); res.end(); return }
      res.setHeader('Content-Type', mime[path.extname(file)] || 'application/octet-stream')
      res.setHeader('Content-Security-Policy', "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data: blob: https://tile.openstreetmap.org; font-src 'self'; object-src 'none'")
      // A controlled test policy, not a statement about deployed Go/nginx headers.
      res.setHeader('Cache-Control', path.extname(file) === '.html' ? 'no-store' : 'public, max-age=3600')
      if (/\.(?:js|css|html)$/.test(file)) { res.setHeader('Content-Encoding', 'gzip'); bytes = zlib.gzipSync(bytes) }
      res.end(bytes)
    })
  })
  await new Promise(resolve => server.listen(0, '127.0.0.1', resolve))
  let browser
  try {
    browser = await chromium.launch({ headless: true, ...(process.env.CARE_CHROME_PATH ? { executablePath: process.env.CARE_CHROME_PATH } : {}) })
    const context = await browser.newContext({ viewport: { width: 1440, height: 1000 } })
    const page = await context.newPage()
    page.on('pageerror', error => report.pageErrors.push(error.message))
    page.on('response', response => { if (response.status() >= 400 && !response.url().includes('/restful/')) report.missingAssets.push(response.url()) })
    await callback({ page, context, browser, base: 'http://127.0.0.1:' + server.address().port, report,
      setRole: role => { if (role !== 'FAMILY') throw new Error('Fixed baseline dataset only supports FAMILY') }, setSignedIn: value => { signedIn = value } })
  } finally {
    if (browser) await browser.close()
    await new Promise(resolve => server.close(resolve))
  }
}
