import axios from 'axios'
import { invoke, isTauri } from '@tauri-apps/api/core'
import { getCurrentWindow } from '@tauri-apps/api/window'

export async function createAdministrator (account) {
  if (!isTauri()) return axios.post('/api/v1/setup', account)
  const result = await invoke('setup_admin', { account })
  if (result.status >= 400) {
    const error = new Error(result.data?.message || '暂时无法创建账号')
    error.response = result
    throw error
  }
  return result
}

function hasOpenModalOrOverlay () {
  return Array.from(document.querySelectorAll('.el-overlay, [role="dialog"]')).some(element => {
    const style = window.getComputedStyle(element)
    return element.getClientRects().length > 0 &&
      element.getAttribute('aria-hidden') !== 'true' &&
      style.display !== 'none' && style.visibility !== 'hidden'
  })
}

export function installDesktopKeys () {
  if (!isTauri()) return () => {}

  const onKeyDown = async event => {
    if (event.repeat || !['F11', 'Escape'].includes(event.key)) return
    // Escape belongs to the active dialog rather than the full-screen window.
    if (event.key === 'Escape' && hasOpenModalOrOverlay()) return

    const desktop = getCurrentWindow()
    try {
      const fullscreen = await desktop.isFullscreen()
      if (event.key === 'F11' || fullscreen) {
        event.preventDefault()
        await desktop.setFullscreen(event.key === 'F11' ? !fullscreen : false)
      }
    } catch (_) { /* Fullscreen permissions must not break page interaction. */ }
  }

  window.addEventListener('keydown', onKeyDown)
  return () => window.removeEventListener('keydown', onKeyDown)
}
