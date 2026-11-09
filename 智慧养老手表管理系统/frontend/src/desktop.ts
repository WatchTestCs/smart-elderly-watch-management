import { api } from './api/client'
import { invoke, isTauri } from '@tauri-apps/api/core'
import { getCurrentWindow } from '@tauri-apps/api/window'

interface SetupResult {
  status: number
  data?: { message?: string }
}

export async function createAdministrator(account: {
  name: string
  phone: string
  password: string
}) {
  if (!isTauri()) return api.post('/api/v1/setup', account)
  const result = await invoke<SetupResult>('setup_admin', { account })
  if (result.status >= 400) {
    const error = new Error(result.data?.message || '暂时无法创建账号') as Error & {
      response?: SetupResult
    }
    error.response = result
    throw error
  }
  return result
}

function hasOpenModalOrOverlay(): boolean {
  return Array.from(document.querySelectorAll<HTMLElement>('.el-overlay, [role="dialog"]')).some(element => {
    const style = window.getComputedStyle(element)
    return element.getClientRects().length > 0 &&
      element.getAttribute('aria-hidden') !== 'true' &&
      style.display !== 'none' && style.visibility !== 'hidden'
  })
}

export function installDesktopKeys(): () => void {
  if (!isTauri()) return () => {}

  const onKeyDown = async (event: KeyboardEvent) => {
    if (event.repeat || !['F11', 'Escape'].includes(event.key)) return
    if (event.key === 'Escape' && hasOpenModalOrOverlay()) return

    const desktop = getCurrentWindow()
    try {
      const fullscreen = await desktop.isFullscreen()
      if (event.key === 'F11' || fullscreen) {
        event.preventDefault()
        await desktop.setFullscreen(event.key === 'F11' ? !fullscreen : false)
      }
    } catch { /* Fullscreen errors must not interrupt active dialogs. */ }
  }

  window.addEventListener('keydown', onKeyDown)
  return () => window.removeEventListener('keydown', onKeyDown)
}
