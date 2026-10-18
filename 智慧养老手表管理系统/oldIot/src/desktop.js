import axios from 'axios'
import { invoke, isTauri } from '@tauri-apps/api/core'
import { getCurrentWindow } from '@tauri-apps/api/window'

export async function createAdministrator (account) {
  if (!isTauri()) return axios.post('/restful/setup', account)
  const result = await invoke('setup_admin', { account })
  if (result.status >= 400) {
    const error = new Error(result.data?.message || '暂时无法创建账号')
    error.response = result
    throw error
  }
  return result
}

export function installDesktopKeys () {
  if (!isTauri()) return
  window.addEventListener('keydown', async event => {
    if (event.repeat || !['F11', 'Escape'].includes(event.key)) return
    if (event.key === 'F11') event.preventDefault()
    const desktop = getCurrentWindow()
    try {
      const fullscreen = await desktop.isFullscreen()
      if (event.key === 'F11' || fullscreen) {
        event.preventDefault()
        await desktop.setFullscreen(event.key === 'F11' ? !fullscreen : false)
      }
    } catch (_) { /* Fullscreen permissions do not affect ordinary page interaction. */ }
  })
}
