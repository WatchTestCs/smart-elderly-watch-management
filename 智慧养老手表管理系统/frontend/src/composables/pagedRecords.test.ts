import { flushPromises, mount } from '@vue/test-utils'
import { defineComponent, h } from 'vue'
import { expect, it, vi } from 'vitest'

const post = vi.hoisted(() => vi.fn())
vi.mock('../api/client', () => ({ api: { post } }))

import { pagedRecords } from './pagedRecords'

const Probe = defineComponent({
  mixins: [pagedRecords({ path: 'api/v1/x', responseKey: 'items', recordsKey: 'rows', searchKey: '', load: 'reload', failure: '失败', trim: false, filters: {}, exportColumns: null, exportName: '' })],
  render() { return h('div') }
})

it('falls back to the new last page when the current page becomes empty', async () => {
  post.mockReset()
  post
    .mockResolvedValueOnce({ data: { items: [], total: 10 } }) // page 3 emptied by a delete
    .mockResolvedValueOnce({ data: { items: [{ id: 10 }], total: 10 } })
  const view = mount(Probe)
  const vm = view.vm as unknown as { currentPage: number; rows: unknown[]; reload: (keep?: boolean) => Promise<void> }
  vm.currentPage = 3
  await vm.reload(true)
  await flushPromises()
  expect(post.mock.calls.map(call => call[1].page)).toEqual([3, 1])
  expect(vm.currentPage).toBe(1)
  expect(vm.rows).toEqual([{ id: 10 }])
})

it('does not loop when the server returns an empty first page', async () => {
  post.mockReset()
  post.mockResolvedValue({ data: { items: [], total: 0 } })
  const vm = mount(Probe).vm as unknown as { reload: (keep?: boolean) => Promise<void> }
  await vm.reload()
  await flushPromises()
  expect(post).toHaveBeenCalledTimes(1)
})
