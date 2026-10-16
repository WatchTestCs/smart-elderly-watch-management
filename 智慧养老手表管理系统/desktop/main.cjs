const { app, BrowserWindow, dialog, session, Menu } = require('electron')
const { spawn } = require('node:child_process')
const { randomBytes, createHash } = require('node:crypto')
const fs = require('node:fs')
const path = require('node:path')
const http = require('node:http')
const { appendLog, removeDiscovery } = require('./runtime-files.cjs')

app.setName('岁宁')
app.setAppUserModelId('org.watchtestcs.suining')
if (process.env.SUINING_USER_DATA) app.setPath('userData', path.resolve(process.env.SUINING_USER_DATA))
let window, backend, origin, closing = false, stopped = false
const token = randomBytes(32).toString('hex')
const sleep = ms => new Promise(resolve => setTimeout(resolve, ms))
if (!app.requestSingleInstanceLock()) app.quit()
else {
  app.on('second-instance', () => { if (window) { if (window.isMinimized()) window.restore(); window.focus() } })
  app.whenReady().then(start).catch(error => {
    appendLog(path.join(app.getPath('userData'), 'startup-error.log'), error.stack + '\n')
    if (!process.env.SUINING_QA_HEADLESS) dialog.showErrorBox('岁宁暂时无法启动', '请重新打开软件。如仍未恢复，请保留用户目录中的日志以便排查。')
    app.quit()
  })
}

async function start () {
  Menu.setApplicationMenu(null)
  const manifest = JSON.parse(fs.readFileSync(path.join(__dirname,'integrity.json'),'utf8'))
  const buffer = Buffer.allocUnsafe(1024*1024)
  for (const [file, expected] of Object.entries(manifest)) {
    const descriptor = fs.openSync(path.join(process.resourcesPath,file),'r')
    const hash = createHash('sha256')
    try {
      let count
      while ((count=fs.readSync(descriptor,buffer,0,buffer.length,null))>0) hash.update(buffer.subarray(0,count))
    } finally { fs.closeSync(descriptor) }
    if (hash.digest('hex')!==expected) throw new Error('Bundled component integrity mismatch: '+file)
  }
  const data = app.getPath('userData')
  fs.mkdirSync(path.join(data, 'database'), { recursive: true })
  fs.mkdirSync(path.join(data, 'images'), { recursive: true })
  const ready = path.join(data, 'runtime', `ready-${process.pid}.json`)
  fs.mkdirSync(path.dirname(ready), { recursive: true })
  const java = path.join(process.resourcesPath, 'runtime', 'bin', 'java.exe')
  const jar = path.join(process.resourcesPath, 'server', 'suining-server.jar')
  const bindAddress = process.env.SUINING_BIND_ADDRESS === '0.0.0.0' ? '0.0.0.0' : '127.0.0.1'
  const requestedPort = Number(process.env.SUINING_PORT || '0')
  const port = Number.isInteger(requestedPort) && requestedPort >= 0 && requestedPort <= 65535 ? requestedPort : 0
  const logFile = path.join(data, 'server-console.log')
  backend = spawn(java, ['-Xms32m', '-Xmx256m', '-XX:+UseG1GC', '-XX:+ExitOnOutOfMemoryError', '-Dfile.encoding=UTF-8', '-jar', jar], {
    cwd: data, windowsHide: true, stdio: ['ignore', 'pipe', 'pipe'],
    env: { ...process.env, SUINING_DATA_DIR: data.replaceAll('\\', '/'), SUINING_PORT: String(port), SUINING_BIND_ADDRESS: bindAddress,
      SUINING_READY_FILE: ready, SUINING_DESKTOP_TOKEN: token, SUINING_PARENT_PID: String(process.pid) }
  })
  backend.stdout.on('data', chunk => appendLog(logFile, chunk, 5 * 1024 * 1024))
  backend.stderr.on('data', chunk => appendLog(logFile, chunk, 5 * 1024 * 1024))
  backend.on('error', error => { appendLog(path.join(data, 'startup-error.log'), error.stack + '\n') })
  backend.on('exit', (code, signal) => {
    removeDiscovery(path.join(data, 'runtime', 'server.json'), backend.pid)
    if (!closing) appendLog(path.join(data, 'startup-error.log'), `Backend exit: code=${code}, signal=${signal}\n`)
    stopped = true
    if (!closing) {
      if (!process.env.SUINING_QA_HEADLESS) dialog.showErrorBox('照护服务已停止', '请重新打开岁宁。您的本机记录会保留。')
      app.quit()
    }
  })
  for (let attempt = 0; attempt < 600; attempt++) {
    if (stopped) throw new Error('Bundled server exited before readiness')
    if (fs.existsSync(ready)) {
      const address = JSON.parse(fs.readFileSync(ready, 'utf8'))
      if (address.pid !== backend.pid || !Number.isInteger(address.port) || address.port < 1 || address.port > 65535) throw new Error('Invalid server readiness')
      origin = `http://127.0.0.1:${address.port}`
      const discovery = path.join(data, 'runtime', 'server.json')
      fs.writeFileSync(ready, JSON.stringify({ ...address, desktopPid: process.pid }))
      fs.renameSync(ready, discovery)
      break
    }
    await sleep(100)
  }
  if (!origin) throw new Error('Bundled server did not become ready')
  session.defaultSession.setPermissionRequestHandler((contents, permission, callback) => callback(false))
  session.defaultSession.setPermissionCheckHandler(() => false)
  session.defaultSession.webRequest.onBeforeSendHeaders({ urls: [origin + '/*'] }, (details, callback) => {
    // Do not expose the bootstrap secret to page JavaScript or localStorage.
    details.requestHeaders['X-Suining-Desktop'] = token
    callback({ requestHeaders: details.requestHeaders })
  })
  window = new BrowserWindow({
    width: 1360, height: 900, minWidth: 360, minHeight: 560, backgroundColor: '#f8f8f2', title: '岁宁',
    show: false, autoHideMenuBar: true, icon: path.join(process.resourcesPath, 'suining.ico'),
    webPreferences: { nodeIntegration: false, contextIsolation: true, sandbox: true, webSecurity: true, spellcheck: false, devTools: Boolean(process.env.SUINING_QA_HEADLESS), offscreen: Boolean(process.env.SUINING_QA_HEADLESS) }
  })
  window.webContents.setWindowOpenHandler(() => ({ action: 'deny' }))
  window.webContents.on('before-input-event', (event, input) => {
    if (input.type !== 'keyDown' || input.isAutoRepeat) return
    if (input.key === 'F11') {
      event.preventDefault()
      window.setFullScreen(!window.isFullScreen())
    } else if (input.key === 'Escape' && window.isFullScreen()) {
      event.preventDefault()
      window.setFullScreen(false)
    }
  })
  if (process.env.SUINING_QA_HEADLESS) window.on('show', () => window.hide())
  window.webContents.on('will-navigate', (event, url) => { if (new URL(url).origin !== origin) event.preventDefault() })
  window.webContents.on('will-attach-webview', event => event.preventDefault())
  window.webContents.session.on('will-download', event => event.preventDefault())
  window.once('ready-to-show', () => {
    window.maximize()
    if (!process.env.SUINING_QA_HEADLESS) window.show()
  })
  await window.loadURL(origin + '/index.html')
  window.on('closed', () => { window = null })
}

app.on('window-all-closed', () => app.quit())
app.on('before-quit', event => {
  if (closing || stopped || !backend) return
  event.preventDefault()
  closing = true
  const finish = async () => {
    if (origin) {
      const request = http.request(origin + '/restful/desktop/exit', { method: 'POST', headers: { 'X-Suining-Desktop': token }, timeout: 2000 }, response => response.resume())
      request.on('error', () => {})
      request.on('timeout', () => request.destroy())
      request.end()
    }
    for (let attempt = 0; attempt < 120 && !stopped; attempt++) await sleep(100)
    if (!stopped) backend.kill()
    app.quit()
  }
  finish()
})
