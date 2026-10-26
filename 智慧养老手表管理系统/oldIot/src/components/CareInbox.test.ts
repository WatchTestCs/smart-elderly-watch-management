import { mount, flushPromises } from '@vue/test-utils'
import { defineComponent } from 'vue'
import { beforeEach, describe, it, expect, vi } from 'vitest'
import CareInbox from './CareInbox.vue'

const { post } = vi.hoisted(() => ({ post: vi.fn() }))
vi.mock('axios', () => ({ default: { post } }))
const Pagination = defineComponent({ name: 'Pagination', emits: ['current-change'], template: '<nav />' })
const fixture = (name: string, total = 100) => ({ data: { items: [{ id: name, oldPersonName: name, state: 'received' }], total, duty: 'supervisor', actorId: 7 } })
function mountInbox() {
  return mount(CareInbox, { global: {
    stubs: { 'el-pagination': Pagination, 'el-select': true, 'el-option': true, 'el-button': true, 'el-dialog': true, 'el-input': true, CareOperationsPanel: true },
    directives: { loading() {} }
  } })
}

describe('care task read scheduling', () => {
  beforeEach(() => { post.mockReset() })

  it('keeps delivered notifications visibly unclaimed until a person acknowledges', async () => {
    post.mockResolvedValueOnce({ data: { items: [{ id: 'delivery-only', oldPersonName: '送达待接单', state: 'notified', notificationState: 'delivered', acknowledgedBy: null }], total: 1, duty: 'supervisor', actorId: 7 } })
    const view = mountInbox()
    await flushPromises()
    expect(view.text()).toContain('已确认送达')
    expect(view.text()).toContain('尚未接单')
    expect(view.text()).not.toContain('已复核关闭')
    view.unmount()
  })

  it('discards a superseded page and loads the newest requested page', async () => {
    let second!: (value: unknown) => void
    let third!: (value: unknown) => void
    post.mockResolvedValueOnce(fixture('初始任务'))
      .mockReturnValueOnce(new Promise(resolve => { second = resolve }))
      .mockReturnValueOnce(new Promise(resolve => { third = resolve }))
    const view = mountInbox()
    await flushPromises()
    view.findComponent(Pagination).vm.$emit('current-change', 2)
    await flushPromises()
    view.findComponent(Pagination).vm.$emit('current-change', 3)
    second(fixture('过期任务'))
    await flushPromises()
    expect(view.text()).not.toContain('过期任务')
    expect(post.mock.calls.map(([, body]) => body.page)).toEqual([1, 2, 3])
    third(fixture('最新任务'))
    await flushPromises()
    expect(view.text()).toContain('最新任务')
    expect(view.get('.care-inbox').attributes('aria-busy')).toBe('false')
    view.unmount()
  })

  it('queues the last available page when records have been removed', async () => {
    post.mockResolvedValueOnce(fixture('初始任务'))
      .mockResolvedValueOnce({ ...fixture('越界响应', 21), data: { ...fixture('越界响应', 21).data, items: [] } })
      .mockResolvedValueOnce(fixture('尾页任务', 21))
    const view = mountInbox()
    await flushPromises()
    view.findComponent(Pagination).vm.$emit('current-change', 5)
    await flushPromises()
    expect(post.mock.calls.map(([, body]) => body.page)).toEqual([1, 5, 2])
    expect(view.text()).toContain('尾页任务')
    expect(view.get('.care-inbox').attributes('aria-busy')).toBe('false')
    view.unmount()
  })

  it('does not start a queued page after the component is unmounted', async () => {
    let second!: (value: unknown) => void
    post.mockResolvedValueOnce(fixture('初始任务')).mockReturnValueOnce(new Promise(resolve => { second = resolve }))
    const view = mountInbox()
    await flushPromises()
    view.findComponent(Pagination).vm.$emit('current-change', 2)
    await flushPromises()
    view.findComponent(Pagination).vm.$emit('current-change', 3)
    view.unmount()
    second(fixture('迟到任务'))
    await flushPromises()
    expect(post).toHaveBeenCalledTimes(2)
  })
})
