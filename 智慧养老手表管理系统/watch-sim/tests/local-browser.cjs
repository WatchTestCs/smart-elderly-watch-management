const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const { spawnSync } = require('node:child_process');
const { pathToFileURL } = require('node:url');
const { chromium } = require('../../frontend/qa/node_modules/playwright');

async function main() {
  const binary = path.resolve(process.argv[2] || 'target/debug/watch-sim.exe');
  const directory = fs.mkdtempSync(path.join(os.tmpdir(), 'watch-local-browser-'));
  let browser;
  try {
    const result = spawnSync(binary, ['--local', directory, '--once', '--scenario', 'sos',
      '--device-no', '</script><img src=x>', '--secret', 'test-only-not-for-export'], { encoding: 'utf8' });
    assert.equal(result.status, 0, result.stderr);
    const run = path.join(directory, fs.readdirSync(directory)[0]);
    const html = fs.readFileSync(path.join(run, 'index.html'), 'utf8');
    assert(!html.includes('test-only-not-for-export'));
    browser = await chromium.launch({ headless: true,
      ...(process.env.CARE_CHROME_PATH ? { executablePath: process.env.CARE_CHROME_PATH } : {}) });
    const context = await browser.newContext({ offline: true, viewport: { width: 1280, height: 960 } });
    const page = await context.newPage();
    const requests = [], errors = [];
    page.on('request', request => { if (/^https?:/.test(request.url())) requests.push(request.url()); });
    page.on('pageerror', error => errors.push(error.message));
    await page.goto(pathToFileURL(path.join(run, 'index.html')).href);
    assert.equal(await page.locator('#scene').innerText(), '一键求救');
    assert((await page.locator('#events').innerText()).includes('模拟求救'));
    assert(Number(await page.locator('#hr').innerText()) > 0);
    assert.equal(await page.locator('img').count(), 0, 'device label was interpreted as HTML');
    await page.waitForTimeout(3200);
    await page.setViewportSize({ width: 390, height: 844 });
    assert.equal(await page.evaluate(() => document.documentElement.scrollWidth > innerWidth), false);
    assert.deepEqual(requests, []);
    assert.deepEqual(errors, []);
    console.log('Offline page, reload, local SOS, escaped labels and mobile layout passed; no network requests.');
  } finally {
    if (browser) await browser.close();
    for (const run of fs.readdirSync(directory)) {
      const runPath = path.join(directory, run);
      for (const file of fs.readdirSync(runPath)) fs.unlinkSync(path.join(runPath, file));
      fs.rmdirSync(runPath);
    }
    fs.rmdirSync(directory);
  }
}
main().catch(error => { console.error(error); process.exitCode = 1; });
