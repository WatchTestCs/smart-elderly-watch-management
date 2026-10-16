const fs = require('node:fs')
const path = require('node:path')

function appendLog (file, message, maxBytes = 1024 * 1024) {
  fs.mkdirSync(path.dirname(file), { recursive: true })
  const bytes = Buffer.isBuffer(message) ? message : Buffer.from(String(message))
  if (fs.existsSync(file) && fs.statSync(file).size + bytes.length > maxBytes) {
    for (let index = 2; index >= 1; index--) {
      const previous = file + '.' + index
      if (fs.existsSync(previous)) {
        if (index === 2) fs.unlinkSync(previous)
        else fs.renameSync(previous, file + '.2')
      }
    }
    fs.renameSync(file, file + '.1')
  }
  fs.appendFileSync(file, bytes.subarray(Math.max(0, bytes.length - maxBytes)))
}
function removeDiscovery (file, pid) {
  try {
    if (JSON.parse(fs.readFileSync(file, 'utf8')).pid === pid) fs.unlinkSync(file)
  } catch (error) { if (error.code !== 'ENOENT') throw error }
}
module.exports = { appendLog, removeDiscovery }
