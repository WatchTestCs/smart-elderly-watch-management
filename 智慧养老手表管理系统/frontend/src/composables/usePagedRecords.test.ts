import { defineComponent, ref } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import { expect, it, vi } from 'vitest'

const mock = vi.hoisted(() => ({ post: vi.fn() }))
vi.mock('../api/client', () => ({ api: { post: mock.post } }))
import { usePagedRecords } from './usePagedRecords'

it('uses the server page contract and unwrapped response, then preserves page on navigation', async () => {
  const row = { oldId: 7, name: '测试长者' }
  mock.post.mockResolvedValue({ data: { returnCode: '1111', oldPeople: [row], total: 25 } })
  let state!: ReturnType<typeof usePagedRecords<{ oldId: number; name: string }, { name: string }>>
  const wrapper = mount(defineComponent({
    setup() {
      const searchText = ref('张')
      state = usePagedRecords<{ oldId: number; name: string }, { name: string }>({
        path: 'api/v1/residents/list',
        responseKey: 'oldPeople',
        buildQuery: () => ({ name: searchText.value })
      })
      return () => null
    }
  }))
  await flushPromises()
  expect(mock.post).toHaveBeenCalledWith('api/v1/residents/list',
    { name: '张', page: 1, pageSize: 10 }, { signal: expect.any(AbortSignal) })
  expect(state.records.value).toEqual([row])
  expect(state.total.value).toBe(25)

  await state.changePage(2)
  expect(mock.post).toHaveBeenLastCalledWith('api/v1/residents/list',
    { name: '张', page: 2, pageSize: 10 }, { signal: expect.any(AbortSignal) })
  expect(state.currentPage.value).toBe(2)
  wrapper.unmount()
})
