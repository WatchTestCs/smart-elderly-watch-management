import { describe, it, expect, vi } from 'vitest'
import { useCustomers, validAge, validPhone } from './useCustomers'
import type { PostClient } from '../api/client'
import { flushPromises } from '@vue/test-utils'

describe('guardian data flow', () => {
  it('loads the latest page and search after changes during an active read', async () => {
    let first!: (value: unknown) => void
    let last!: (value: unknown) => void
    const post = vi.fn().mockReturnValueOnce(new Promise(resolve => { first = resolve })).mockReturnValueOnce(new Promise(resolve => { last = resolve }))
    const state = useCustomers({ post } as PostClient)
    const pending = state.searchCustomers()
    await Promise.resolve()
    state.changePage(2)
    state.changePage(3)
    state.searchText.value = '新查询'
    state.searchCustomers()
    first({ data: { customers: [{ customerName: '旧数据' }], total: 99 } })
    await flushPromises()
    expect(state.customerList.value).toEqual([])
    expect(post.mock.calls.map(([, body]) => body)).toEqual([
      { page: 1, pageSize: 10, customerName: '' },
      { page: 1, pageSize: 10, customerName: '新查询' }
    ])
    last({ data: { customers: [{ customerName: '当前数据' }], total: 1 } })
    await pending
    expect(state.customerList.value[0]?.customerName).toBe('当前数据')
    expect(state.total.value).toBe(1)
    expect(state.isLoading.value).toBe(false)
  })

  it('can refresh after an earlier customer query hangs indefinitely', async () => {
    const post = vi.fn()
      .mockReturnValueOnce(new Promise(() => {}))
      .mockResolvedValueOnce({ data: { customers: [{ customerName: '新数据' }], total: 1 } })
    const state = useCustomers({ post } as PostClient)
    const old = state.searchCustomers()
    await flushPromises()
    const current = state.changePage(2)
    await Promise.all([old, current])
    expect(post.mock.calls.map(([, body]) => body.page)).toEqual([1, 2])
    expect(state.customerList.value[0]?.customerName).toBe('新数据')
    expect(state.isLoading.value).toBe(false)
  })
  it('waits for a read started after a confirmed write before save completes', async () => {
    let first!: (value: unknown) => void
    const post = vi.fn().mockReturnValueOnce(new Promise(resolve => { first = resolve })).mockResolvedValueOnce({ data: { returnCode: '1111' } }).mockResolvedValueOnce({ data: { customers: [], total: 0 } })
    const state = useCustomers({ post } as PostClient)
    state.searchCustomers()
    await Promise.resolve()
    Object.assign(state.customerForm, { customerName: '姓名', customerPhone: '19912345678', customerAge: 70 })
    const saved = state.save()
    await flushPromises()
    expect(state.saving.value).toBe(true)
    first({ data: { customers: [{ customerName: '过期数据' }], total: 1 } })
    expect(await saved).toBe(true)
    expect(post.mock.calls.map(([path]) => path)).toEqual(['restful/customer/select', 'restful/customer/add', 'restful/customer/select'])
    expect(state.customerList.value).toEqual([])
    expect(state.saving.value).toBe(false)
  })
  it('accepts modern phone prefixes and valid integer age boundaries', () => {
    expect(validPhone('19912345678')).toBe(true)
    expect(validPhone('1|912345678')).toBe(false)
    for (const age of [0, '0', 150]) expect(validAge(age)).toBe(true)
    for (const age of ['', null, -1, 151, 1.2]) expect(validAge(age)).toBe(false)
  })
  it('clears failed list state and can retry the same page', async () => {
    const post = vi.fn().mockRejectedValueOnce(new Error('offline')).mockResolvedValueOnce({ data: { customers: [], total: 0 } })
    const state = useCustomers({ post } as PostClient)
    state.currentPage.value = 3
    await state.searchCustomers(true)
    expect(state.loadError.value).not.toBe('')
    expect(state.isLoading.value).toBe(false)
    await state.searchCustomers(true)
    expect(post.mock.calls[1]?.[1]).toMatchObject({ page: 3 })
    expect(state.loadError.value).toBe('')
  })
  it('keeps a rejected edit open and never mutates the selected table row', async () => {
    const post = vi.fn().mockResolvedValue({ data: { returnCode: '0000', message: 'denied' } })
    const state = useCustomers({ post } as PostClient)
    const row = { customerId: 1, customerName: '原名', customerAge: 70, customerPhone: '19912345678', customerSex: '女', customerAddress: '' }
    state.handleEditRole(0, row); state.oldEditForm.customerName = '编辑中'
    expect(row.customerName).toBe('原名')
    expect(await state.save(true)).toBe(false)
    expect(state.oldEditDialogVisible.value).toBe(true)
    expect(state.mutationError.value).toBe('denied')
  })
  it('does not pretend a rejected deletion succeeded', async () => {
    const post = vi.fn().mockRejectedValue(new Error('network'))
    const state = useCustomers({ post } as PostClient)
    expect(await state.remove(1)).toBe(false)
    expect(post).toHaveBeenCalledTimes(1)
    expect(state.mutationError.value).not.toBe('')
  })
})
