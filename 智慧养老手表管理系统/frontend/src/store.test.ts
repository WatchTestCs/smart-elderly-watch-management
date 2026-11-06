import { beforeEach, expect, it } from 'vitest'
import store from './store'

beforeEach(() => { store.clearTabs() })

it('persists workspace tabs and the selected resident without persisting transient alert signals', () => {
  store.addTab({ route: '/operations/overview', name: '运营总览' })
  store.addTab({ route: '/operations/overview', name: '重复' })
  store.activate('/operations/overview')
  store.selectResident(1001)
  expect(store.state.openTab).toHaveLength(1)
  const persisted = JSON.parse(sessionStorage.getItem('suining-workspace-v1') || '{}')
  expect(persisted.selectedResidentId).toBe('1001')
  expect(persisted.openTab).toEqual([{ route: '/operations/overview', name: '运营总览' }])
  const initial = store.state.alertRevision
  store.notifyAlertsChanged()
  expect(store.state.alertRevision).toBe(initial + 1)
  expect(JSON.parse(sessionStorage.getItem('suining-workspace-v1') || '{}')).not.toHaveProperty('alertRevision')
  store.clearTabs()
  expect(sessionStorage.getItem('suining-workspace-v1')).toBeNull()
})
