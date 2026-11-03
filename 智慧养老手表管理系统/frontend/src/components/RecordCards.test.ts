import { h } from 'vue'
import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import RecordCards from './RecordCards.vue'

describe('record cards', () => {
  it('preserves handled alerts, zero measurements, extra fields and the action row', async () => {
    const records = [{ oldId: 8, name: '测试长者', alarm: '0', handled: '1', steps: 0, extra: '补充资料' }]
    const select = vi.fn()
    const wrapper = mount(RecordCards, {
      props: { records, label: '测试记录', fields: [
        { key: 'alarm', label: '告警', state: 'fall', handled: 'handled' },
        { key: 'steps', label: '步数' }, { key: 'missing', label: '缺失数据' },
        { key: 'name', label: '姓名' }, { key: 'extra', label: '补充' }
      ] },
      slots: { actions: ({ row, $index }: { row: object; $index: number }) =>
        h('button', { onClick: () => select(row, $index) }, '查看') }
    })
    expect(wrapper.attributes('aria-label')).toBe('测试记录')
    expect(wrapper.find('.status-handled').text()).toBe('已处理')
    expect(wrapper.findAll('dd').map(value => value.text())).toContain('0')
    expect(wrapper.text()).toContain('暂未提供')
    expect(wrapper.find('details').text()).toContain('补充资料')
    await wrapper.find('button').trigger('click')
    expect(select).toHaveBeenCalledWith(records[0], 0)
    wrapper.unmount()
  })

  it('announces an empty collection and keeps active alarms distinct from handled history', () => {
    const empty = mount(RecordCards, { props: { records: [], fields: [], label: '测试记录' } })
    expect(empty.find('[role="status"]').text()).toContain('暂无记录')
    empty.unmount()
    const active = mount(RecordCards, { props: {
      records: [{ oldId: 9, name: '测试长者', alarm: '1', handled: 1 }], label: '测试记录',
      fields: [{ key: 'alarm', label: '告警', state: 'fall', handled: 'handled' }]
    } })
    expect(active.find('.status-danger').text()).toBe('紧急告警')
    active.unmount()
  })
})
