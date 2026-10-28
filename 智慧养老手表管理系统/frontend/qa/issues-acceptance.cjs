const fs = require('node:fs')
const path = require('node:path')
const assert = require('node:assert/strict')
const run = require('./browser-harness.cjs')

run(async ({ page, context, base, report, setRole, setSignedIn }) => {
  const checks = []
  const check = (name, pass) => { checks.push({ name, pass: Boolean(pass) }); assert.ok(pass, name) }
  const visit = async (route, role = 'ADMIN') => {
    setRole(role); setSignedIn(true)
    await page.goto(base + '/?case=' + Date.now() + '#' + route)
    await page.locator('.care-app').waitFor()
  }
  let tileRequests = [], failTiles = false
  await context.route('https://tile.openstreetmap.org/**', async route => {
    tileRequests.push(route.request().headers())
    if (failTiles) return route.fulfill({ status: 503, body: '' })
    await route.fulfill({ contentType: 'image/png', body: Buffer.from('iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+aD1sAAAAASUVORK5CYII=', 'base64') })
  })
  let coordinates = { lng: '0', lat: '0', name: '<img src=x onerror=alert(1)>', oldId: 1, status: '0' }
  for (const endpoint of ['select', 'selectFamily']) await context.route('**/api/v1/position/' + endpoint, route => route.fulfill({
    contentType: 'application/json', body: JSON.stringify({ positionDtos: [coordinates], total: 1 })
  }))
  for (const [route, role] of [['/position', 'ADMIN'], ['/positionfamily', 'FAMILY']]) {
    await visit(route, role)
    await page.getByRole('button', { name: role === 'ADMIN' ? '详细情况' : '地图查看', exact: true }).first().click()
    await page.locator('.leaflet-tooltip').waitFor()
    check(route + ' accepts zero coordinates', (await page.locator('.position-map-heading p').textContent()).includes('0，0'))
    check(route + ' escapes marker labels', await page.locator('.leaflet-tooltip img').count() === 0)
    check(route + ' makes no tile request without consent', tileRequests.length === 0)
    await page.getByRole('button', { name: '加载在线地图', exact: true }).click()
    await page.waitForFunction(() => document.querySelectorAll('.leaflet-tile-loaded').length > 0)
    check(route + ' renders local Leaflet with keyless tiles', tileRequests.length > 0)
    check(route + ' keeps provider attribution visible', (await page.locator('.leaflet-control-attribution').textContent()).includes('OpenStreetMap'))
    check(route + ' sends an origin Referer', tileRequests.every(headers => headers.referer?.startsWith(base)))
    await page.getByRole('button', { name: '关闭在线地图', exact: true }).click()
    check(route + ' removes online tiles', await page.locator('.leaflet-tile-pane img').count() === 0)
    tileRequests = []
  }
  failTiles = true
  coordinates = { ...coordinates, lng: '12', lat: '50' }
  await visit('/positionfamily', 'FAMILY')
  await page.getByRole('button', { name: '地图查看', exact: true }).first().click()
  await page.getByRole('button', { name: '加载在线地图', exact: true }).click()
  await page.getByRole('status').filter({ hasText: '在线地图加载失败' }).waitFor()
  check('tile failures retain the marker and coordinates', await page.locator('.leaflet-tooltip').count() === 1)
  await visit('/familyMain', 'FAMILY')
  await page.setViewportSize({ width: 390, height: 844 })
  check('map cleanup survives navigation and resize', report.pageErrors.length === 0)

  // Exercise actual paging controls and verify the request window.
  for (const [route, endpoint, key, button] of [
    ['/step', 'step/selectStep', 'stepDtos', '查询长者姓名'],
    ['/ratepre', 'ratepre/selectRatepre', 'ratepreDtoList', '查询长者姓名'],
    ['/position', 'position/select', 'positionDtos', '查询长者姓名'],
    ['/device', 'device/list', 'devices', null],
    ['/life', 'oldPerson/select', 'oldPeople', '查询长者姓名'],
    ['/person', 'oldPerson/selectDetail', 'oldPersonDetails', '查询长者姓名'],
    ['/watch', 'watchInfo/select', 'watchDtos', '查询长者姓名'],
    ['/customer', 'customer/select', 'customers', '查询监护人姓名'],
    ['/family', 'family/select', 'familyDtos', '查询家属姓名'],
    ['/main', 'text/select', 'texts', '搜索公告'],
    ['/textOpt', 'text/select', 'texts', '搜索公告'],
    ['/inviteCode', 'inviteCode/list', 'inviteCodes', null],
    ['/saidOperation', 'said/selectByUser', 'saidDtos', null],
    ['/familyMain', 'text/select', 'texts', null]
  ]) {
    await context.route('**/api/v1/' + endpoint, r => {
      const input = r.request().postDataJSON()
      return r.fulfill({ contentType: 'application/json', body: JSON.stringify({
        [key]: [{ oldId: input.page || 1, deviceId: input.page || 1, deviceNo: 'PAGE', name: '分页长者', alias: '分页设备', status: 1 }],
        total: 45, onlineCount: 0
      }) })
    })
    await visit(route, route === '/familyMain' ? 'FAMILY' : 'ADMIN')
    await page.locator('.el-pagination .btn-next:not([disabled])').waitFor()
    const request = page.waitForRequest(r => r.url().endsWith('/api/v1/' + endpoint) && r.postDataJSON().page === 2)
    await page.locator('.el-pagination .btn-next').click()
    const input = (await request).postDataJSON()
    check(route + ' sends page 2 to the server', input.page === 2 && input.pageSize <= 100)
    if (button) {
      const search = page.waitForRequest(r => r.url().endsWith('/api/v1/' + endpoint) && r.postDataJSON().page === 1)
      await page.getByRole('button', { name: button, exact: true }).click()
      check(route + ' resets page on search', (await search).postDataJSON().page === 1)
    }
  }
  check('no runtime errors in new interactions', report.pageErrors.length === 0)
  const files = fs.readdirSync(path.join(__dirname, '../dist/assets')).filter(name => name.endsWith('.js'))
  check('production JavaScript strips console.log', files.every(file => !/console\.log\s*\(/.test(fs.readFileSync(path.join(__dirname, '../dist/assets', file), 'utf8'))))
  fs.writeFileSync(path.join(__dirname, 'artifacts/issues-acceptance.json'), JSON.stringify({ checks }, null, 2))
  console.log(JSON.stringify({ checks: checks.length, failed: checks.filter(c => !c.pass) }))
}).catch(error => { console.error(error); process.exitCode = 1 })
