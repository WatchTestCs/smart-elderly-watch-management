import { reactive } from 'vue'

export interface Tab { route: string; name: string }
interface WorkspaceState {
  openTab: Tab[]
  activeIndex: string
  selectedResidentId: string | null
  alertRevision: number
}
const storageKey = 'suining-workspace-v1'
const home = '/operations/overview'

function restoreWorkspace(): Pick<WorkspaceState, 'openTab' | 'activeIndex' | 'selectedResidentId'> {
  try {
    const raw = sessionStorage.getItem(storageKey)
    if (raw) {
      const saved = JSON.parse(raw) as Partial<WorkspaceState>
      return {
        openTab: Array.isArray(saved.openTab) ? saved.openTab.filter(tab =>
          tab && typeof tab.route === 'string' && tab.route.startsWith('/operations/') &&
          typeof tab.name === 'string').slice(0, 20) : [],
        activeIndex: typeof saved.activeIndex === 'string' && saved.activeIndex.startsWith('/operations/')
          ? saved.activeIndex : home,
        selectedResidentId: typeof saved.selectedResidentId === 'string' ? saved.selectedResidentId : null
      }
    }
  } catch { /* Storage-disabled sessions keep the workspace in memory. */ }
  return { openTab: [], activeIndex: home, selectedResidentId: null }
}
const state = reactive<WorkspaceState>({ ...restoreWorkspace(), alertRevision: 0 })
function persistWorkspace() {
  try {
    sessionStorage.setItem(storageKey, JSON.stringify({
      openTab: state.openTab, activeIndex: state.activeIndex, selectedResidentId: state.selectedResidentId
    }))
  } catch { /* Storage availability must not break navigation. */ }
}
const alertsChannel = typeof window !== 'undefined' && typeof window.BroadcastChannel === 'function'
  ? new window.BroadcastChannel('suining-alerts-v1') : null
if (alertsChannel) alertsChannel.onmessage = event => {
  if (event.data?.type === 'alerts-changed') state.alertRevision++
}

const store = {
  state,
  addTab(tab: Tab) {
    if (!state.openTab.some(item => item.route === tab.route)) {
      state.openTab.push(tab)
      persistWorkspace()
    }
  },
  removeTab(route: string) {
    state.openTab = state.openTab.filter(tab => tab.route !== route)
    persistWorkspace()
  },
  clearTabs() {
    state.openTab = []
    state.activeIndex = home
    state.selectedResidentId = null
    try { sessionStorage.removeItem(storageKey) } catch { /* Storage may be unavailable. */ }
  },
  activate(route: string) {
    state.activeIndex = route
    persistWorkspace()
  },
  selectResident(id: string | number | null) {
    state.selectedResidentId = id == null ? null : String(id)
    persistWorkspace()
  },
  notifyAlertsChanged() {
    state.alertRevision++
    alertsChannel?.postMessage({ type: 'alerts-changed' })
  }
}
export default store
