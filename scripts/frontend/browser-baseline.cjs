// Exploratory production baseline, not an A/B benefit test. Uses the existing
// synthetic HTTP fixture server; the report states this boundary explicitly.
const assert = require('node:assert/strict')
const fs = require('node:fs')
const path = require('node:path')
const { performance } = require('node:perf_hooks')
const root = path.resolve(__dirname, '../..')
const run = require('./baseline-server.cjs')
const output = process.env.FRONTEND_EVIDENCE || path.join(root, 'evidence/frontend-20261028')
const result = { scope: 'Production bundle, browser, fixed synthetic HTTP API fixtures; no actual backend/WebView2 or performance improvement claim',
  cacheMethod: 'No Playwright routing. Cold: CDP clearBrowserCache. Warm: normal full navigation without clearing. Unique HTML queries force document navigation; test server HTML/API no-store, other assets public max-age=3600, JS/CSS/HTML gzip; this controlled policy is not a claim about deployment headers. Record actual transferSize.',
  viewport: { width: 1440, height: 1000 }, samples: [], chartCycles: [], errors: [] }
const frames = page => page.evaluate(() => new Promise(resolve => requestAnimationFrame(() => requestAnimationFrame(resolve))))
function summary(values) {
  const sorted = [...values].sort((a, b) => a - b)
  return { n: sorted.length, median: sorted.length % 2 ? sorted[(sorted.length - 1) / 2] : (sorted[sorted.length / 2 - 1] + sorted[sorted.length / 2]) / 2,
    min: sorted[0], max: sorted.at(-1) }
}
run(async ({ page, context, browser, base, report, setRole, setSignedIn }) => {
  result.browser = browser.version()
  const cdp = await context.newCDPSession(page)
  await cdp.send('Network.enable')
  await cdp.send('Performance.enable')
  async function captureMemory(point) {
    await cdp.send('HeapProfiler.collectGarbage')
    await frames(page)
    const metrics = Object.fromEntries((await cdp.send('Performance.getMetrics')).metrics.map(m => [m.name, m.value]))
    return { point, metrics: Object.fromEntries(['JSHeapUsedSize', 'JSHeapTotalSize', 'Nodes', 'Documents', 'JSEventListeners'].map(key => [key, metrics[key]])),
      dom: await cdp.send('Memory.getDOMCounters'),
      chart: await page.evaluate(() => ({ canvases: document.querySelectorAll('.el-dialog canvas').length,
        instanceMarkers: [...document.querySelectorAll('[_echarts_instance_]')].filter(el => el.getAttribute('_echarts_instance_')).length })) }
  }
  for (const route of ['/login', '/rateprefamily', '/positionfamily']) {
    setRole('FAMILY')
    for (let sample = 1; sample <= 5; sample++) {
      await cdp.send('Network.clearBrowserCache')
      for (const cache of ['cold', 'warm']) {
        const started = performance.now()
        setSignedIn(route !== '/login')
        await page.goto(base + '/?baseline=' + encodeURIComponent(route) + '-' + sample + '-' + cache + '#' + route)
        if (route === '/login') await page.getByRole('heading', { name: '欢迎回来' }).waitFor()
        else await page.getByRole('button', { name: route === '/rateprefamily' ? '详细情况' : '地图查看', exact: true }).first().waitFor()
        await page.evaluate(() => document.fonts.ready)
        await frames(page)
        const readyMs = performance.now() - started
        const data = await page.evaluate(() => ({ navigation: performance.getEntriesByType('navigation').map(e => e.toJSON()),
          paint: performance.getEntriesByType('paint').map(e => e.toJSON()),
          resources: performance.getEntriesByType('resource').map(e => ({ name: new URL(e.name).pathname, initiatorType: e.initiatorType,
            duration: e.duration, transferSize: e.transferSize, encodedBodySize: e.encodedBodySize, decodedBodySize: e.decodedBodySize })) }))
        assert.equal(data.navigation.length, 1)
        assert.equal(data.navigation[0].name, page.url(), 'Require a real document navigation for every sample')
        const actionStart = performance.now()
        let action
        if (route === '/login') {
          await page.getByLabel('手机号', { exact: true }).fill('13000000000')
          await page.getByLabel('密码', { exact: true }).fill('baseline-fixture-password')
          const clicked = performance.now()
          await page.getByRole('button', { name: '登录', exact: true }).click()
          await page.waitForURL(url => url.hash === '#/familyMain')
          await frames(page)
          action = { name: 'login-click-to-family-route-two-frames', ms: performance.now() - clicked }
        } else if (route === '/rateprefamily') {
          await page.getByRole('button', { name: '详细情况', exact: true }).first().click()
          await page.waitForFunction(() => document.querySelectorAll('.el-dialog canvas').length === 2)
          await frames(page)
          action = { name: 'chart-click-to-two-canvases-two-frames', ms: performance.now() - actionStart }
          await page.getByRole('button', { name: '关闭详情', exact: true }).click()
          await page.waitForFunction(() => document.querySelectorAll('.el-dialog canvas').length === 0)
        } else {
          await page.getByRole('button', { name: '地图查看', exact: true }).first().click()
          await page.locator('.leaflet-tooltip').waitFor()
          await frames(page)
          action = { name: 'map-click-to-local-marker-two-frames-no-online-tiles', ms: performance.now() - actionStart }
        }
        result.samples.push({ route, cache, sample, readyMs, action, ...data })
      }
    }
  }
  await page.goto(base + '/?baseline=chart-cycles#/rateprefamily')
  await page.getByRole('button', { name: '详细情况', exact: true }).first().waitFor()
  result.chartStart = await captureMemory('before first open')
  for (let cycle = 1; cycle <= 20; cycle++) {
    const started = performance.now()
    await page.getByRole('button', { name: '详细情况', exact: true }).first().click()
    await page.waitForFunction(() => document.querySelectorAll('.el-dialog canvas').length === 2)
    await frames(page)
    const openReadyMs = performance.now() - started
    // Allow normal chart/dialog animations to finish before a fixed memory point.
    await page.waitForTimeout(400)
    const opened = await captureMemory('opened')
    assert.equal(opened.chart.instanceMarkers, 2)
    await page.getByRole('button', { name: '关闭详情', exact: true }).click()
    await page.waitForFunction(() => document.querySelectorAll('.el-dialog canvas').length === 0)
    await frames(page)
    const closed = await captureMemory('closed')
    assert.equal(closed.chart.instanceMarkers, 0)
    result.chartCycles.push({ cycle, openReadyMs, opened, closed })
  }
  result.errors = report.pageErrors
  result.missingAssets = report.missingAssets
  result.unexpectedApi = report.unexpectedApi
  assert.equal(result.errors.length, 0, 'Browser errors')
  assert.equal(result.missingAssets.length, 0, 'Missing assets')
  assert.equal(result.unexpectedApi.length, 0, 'Unexpected API calls')
  result.summary = Object.fromEntries(['/login', '/rateprefamily', '/positionfamily'].flatMap(route => ['cold', 'warm'].map(cache => {
    const samples = result.samples.filter(s => s.route === route && s.cache === cache)
    return [route + ':' + cache, { readyMs: summary(samples.map(s => s.readyMs)), actionMs: summary(samples.map(s => s.action.ms)) }]
  })))
  result.chartSummary = { openReadyMs: summary(result.chartCycles.map(c => c.openReadyMs)),
    closedHeapBytes: summary(result.chartCycles.map(c => c.closed.metrics.JSHeapUsedSize)),
    closedListeners: summary(result.chartCycles.map(c => c.closed.metrics.JSEventListeners)) }
  await page.screenshot({ path: path.join(output, 'health-baseline.png'), fullPage: true })
  console.log(JSON.stringify({ browser: result.browser, summary: result.summary, chart: result.chartSummary }, null, 2))
}).catch(error => { result.failure = error.stack; console.error(error); process.exitCode = 1 })
  .finally(() => fs.writeFileSync(path.join(output, 'browser-baseline.json'), JSON.stringify(result, null, 2) + '\n'))
