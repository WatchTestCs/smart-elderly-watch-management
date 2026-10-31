import { mount } from '@vue/test-utils'
import { expect, it, vi } from 'vitest'
import Position from './position.vue'
import RecordCards from '../components/RecordCards.vue'

type AlertView = { methods: { showFallReason(this: { $alert: ReturnType<typeof vi.fn> }, row: Record<string, unknown>): void } }
const showReason = (Position as unknown as AlertView).methods.showFallReason

it('shows the incident snapshot rather than the later location in the alarm dialog', () => {
  const alert = vi.fn()
  showReason.call({ $alert: alert }, { name: '长者', beTime: '2026-10-20 11:00:00', lng: '121', lat: '31', alarmTime: '2026-10-20 10:00:00', alarmLng: '120', alarmLat: '30' })
  const [message, title] = alert.mock.calls[0]
  expect(message).toContain('2026-10-20 10:00:00')
  expect(message).toContain('经度 120，纬度 30')
  expect(message).not.toContain('2026-10-20 11:00:00')
  expect(message).not.toContain('经度 121')
  expect(title).toBe('紧急告警 · 原因与处理')
})

it('keeps missing incident coordinates unknown and accepts an explicit zero point', () => {
  const alert = vi.fn()
  showReason.call({ $alert: alert }, { name: '长者', lng: '121', lat: '31', alarmTime: null, alarmLng: null, alarmLat: null })
  expect(alert.mock.calls[0][0]).toContain('未提供告警坐标')
  expect(alert.mock.calls[0][0]).not.toContain('经度 121')
  showReason.call({ $alert: alert }, { name: '长者', alarmLng: '0', alarmLat: '0' })
  expect(alert.mock.calls[1][0]).toContain('经度 0，纬度 0')
})

it('does not classify the shared emergency flag as a confirmed fall', async () => {
  const view = mount(RecordCards, { props: { records: [{ oldId: 1, name: '长者', status: '1' }], fields: [{ key: 'status', label: '告警状态', state: 'fall' }], label: '位置记录' } })
  expect(view.text()).toContain('紧急告警')
  expect(view.text()).not.toContain('摔倒')
  await view.setProps({ records: [{ oldId: 1, name: '长者', status: '0' }] })
  expect(view.text()).toContain('未告警/已处理')
  view.unmount()
})
