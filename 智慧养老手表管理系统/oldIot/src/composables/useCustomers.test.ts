import { describe, it, expect, vi } from 'vitest'
import { useCustomers, validAge, validPhone } from './useCustomers'
import type { PostClient } from '../api/client'

describe('guardian data flow', () => {
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
