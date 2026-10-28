import { mount, flushPromises } from '@vue/test-utils'
import { defineComponent } from 'vue'
import { afterEach, beforeEach, describe, it, expect, vi } from 'vitest'
import CareInbox from './CareInbox.vue'

const { post } = vi.hoisted(() => ({ post: vi.fn() }))
vi.mock('axios', () => ({ default: { post } }))
const Pagination = defineComponent({ name: 'Pagination', emits: ['current-change'], template: '<nav />' })
const Button = defineComponent({ emits: ['click'], template: '<button @click="$emit(\'click\')"><slot /></button>' })
const Dialog = defineComponent({ props: ['modelValue'], emits: ['close', 'update:modelValue'], template: '<section v-if="modelValue" class="test-dialog"><slot /><slot name="footer" /><button class="close-dialog" @click="$emit(\'update:modelValue\', false); $emit(\'close\')">关闭</button></section>' })
const fixture = (name: string, total = 100) => ({ data: { items: [{ id: name, oldPersonName: name, state: 'received' }], total, duty: 'supervisor', actorId: 7 } })
function mountInbox() {
  return mount(CareInbox, { global: {
    stubs: { 'el-pagination': Pagination, 'el-select': true, 'el-option': true, 'el-button': Button, 'el-dialog': Dialog, 'el-input': true, CareOperationsPanel: true },
    directives: { loading() {} }
  } })
}

describe('care task read scheduling', () => {
  beforeEach(() => { post.mockReset() })
  afterEach(() => { vi.useRealTimers() })

  it('aborts a hung task page on timeout and can poll successfully afterwards', async () => {
    vi.useFakeTimers()
    post.mockReturnValueOnce(new Promise(() => {})).mockResolvedValueOnce(fixture('恢复任务'))
    const view = mountInbox()
    await vi.advanceTimersByTimeAsync(10000)
    expect(view.get('.care-inbox').attributes('aria-busy')).toBe('false')
    expect(view.text()).toContain('照护任务暂时无法加载')
    expect(post.mock.calls[0][2].signal.aborted).toBe(true)
    await vi.advanceTimersByTimeAsync(5000)
    expect(view.text()).toContain('恢复任务')
    expect(view.text()).not.toContain('照护任务暂时无法加载')
    expect(post.mock.calls.every(([path]) => path === 'restful/care/incidents')).toBe(true)
    view.unmount()
    expect(vi.getTimerCount()).toBe(0)
  })

  it('aborts the active task request immediately when unmounted', async () => {
    post.mockReturnValue(new Promise(() => {}))
    const view = mountInbox()
    await flushPromises()
    const signal = post.mock.calls[0][2].signal
    expect(signal.aborted).toBe(false)
    view.unmount()
    expect(signal.aborted).toBe(true)
    await flushPromises()
    expect(post).toHaveBeenCalledTimes(1)
  })

  it('replaces a hung history read and cancels another read when its dialog closes', async () => {
    let finishOld!: (value: unknown) => void
    post.mockResolvedValueOnce({ data: { ...fixture('first').data, items: [
      { id: 'first', oldPersonName: '第一位', state: 'received' },
      { id: 'second', oldPersonName: '第二位', state: 'received' }
    ] } })
      .mockReturnValueOnce(new Promise(resolve => { finishOld = resolve }))
      .mockResolvedValueOnce({ data: { events: [{ action: '最新履历' }], total: 1 } })
      .mockReturnValueOnce(new Promise(() => {}))
    const view = mountInbox()
    await flushPromises()
    const buttons = view.findAll('.care-task-actions button').filter(button => button.text() === '处置记录')
    await buttons[0].trigger('click')
    await flushPromises()
    await buttons[1].trigger('click')
    await flushPromises()
    expect(post.mock.calls[1][2].signal.aborted).toBe(true)
    expect(post.mock.calls[2][1]).toMatchObject({ id: 'second', page: 1 })
    expect(view.text()).toContain('最新履历')
    finishOld({ data: { events: [{ action: '过期履历' }], total: 99 } })
    await flushPromises()
    expect(view.text()).not.toContain('过期履历')
    await buttons[0].trigger('click')
    await flushPromises()
    const signal = post.mock.calls[3][2].signal
    await view.get('.close-dialog').trigger('click')
    expect(signal.aborted).toBe(true)
    await flushPromises()
    expect(view.find('.test-dialog').exists()).toBe(false)
    view.unmount()
  })

  it('times out history and aborts a later history request on unmount', async () => {
    vi.useFakeTimers()
    post.mockResolvedValueOnce(fixture('历史任务')).mockReturnValue(new Promise(() => {}))
    const view = mountInbox()
    await vi.advanceTimersByTimeAsync(0)
    const button = view.findAll('.care-task-actions button').find(button => button.text() === '处置记录')!
    await button.trigger('click')
    await vi.advanceTimersByTimeAsync(10000)
    expect(view.text()).toContain('审计记录读取超时')
    expect(view.text()).not.toContain('正在读取记录')
    expect(post.mock.calls[1][2].signal.aborted).toBe(true)
    await button.trigger('click')
    await vi.advanceTimersByTimeAsync(0)
    const signal = post.mock.calls[2][2].signal
    view.unmount()
    expect(signal.aborted).toBe(true)
    await vi.advanceTimersByTimeAsync(0)
    expect(vi.getTimerCount()).toBe(0)
  })

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

  it('does not leave loading stuck when a superseded page never resolves', async () => {
    post.mockResolvedValueOnce(fixture('初始任务'))
      .mockReturnValueOnce(new Promise(() => {}))
      .mockResolvedValueOnce(fixture('恢复成功'))
    const view = mountInbox()
    await flushPromises()
    view.findComponent(Pagination).vm.$emit('current-change', 2)
    await flushPromises()
    view.findComponent(Pagination).vm.$emit('current-change', 3)
    await flushPromises()
    expect(post.mock.calls.map(([, body]) => body.page)).toEqual([1, 2, 3])
    expect(view.text()).toContain('恢复成功')
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
