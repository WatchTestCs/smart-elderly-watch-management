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

it('falls back to the new last page when the current page becomes empty', async () => {
  const post = mock.post
  let state!: ReturnType<typeof usePagedRecords<{ id: number }>>
  const Probe = defineComponent({
    setup() {
      state = usePagedRecords<{ id: number }>({ path: 'api/v1/x', responseKey: 'items', failure: '失败' })
      return () => null
    }
  })

  post.mockReset()
  post.mockResolvedValue({ data: { items: [], total: 0 } })
  // 先吞掉 onMounted 的自加载（total=0 → lastPage=1，1<1 不成立，不回退）
  mount(Probe)
  await flushPromises()

  post.mockReset()
  post
    .mockResolvedValueOnce({ data: { items: [], total: 10 } }) // page 3 被删空 → lastPage=1，1<3 成立，触发回退
    .mockResolvedValueOnce({ data: { items: [{ id: 10 }], total: 10 } })
  state.currentPage.value = 3
  await state.load(true) // 必须传 true，否则 currentPage 被打回 1
  await flushPromises()
  expect(post.mock.calls.map(call => call[1].page)).toEqual([3, 1])
  expect(state.currentPage.value).toBe(1)
  expect(state.records.value).toEqual([{ id: 10 }])
})

it('does not loop when the server returns an empty first page', async () => {
  const post = mock.post
  let state!: ReturnType<typeof usePagedRecords<{ id: number }>>
  const Probe = defineComponent({
    setup() {
      state = usePagedRecords<{ id: number }>({ path: 'api/v1/x', responseKey: 'items', failure: '失败' })
      return () => null
    }
  })

  post.mockReset()
  post.mockResolvedValue({ data: { items: [], total: 0 } })
  const wrapper = mount(Probe)
  await flushPromises()
  expect(state.currentPage.value).toBe(1)
  expect(post).toHaveBeenCalledTimes(1) // 这一次就是自加载，没有二次回退请求
  wrapper.unmount()
})
