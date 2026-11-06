import { flushPromises } from '@vue/test-utils'
import { expect, it, vi } from 'vitest'
import { editableResident, residentFields } from '../composables/residentFields'
import { deviceAdminMethods } from './device.admin-methods'

it('keeps only editable resident fields and the selected guardian in writes', () => {
  const fields = { name: '长者', bothdate: '1950-01-01', sex: '女', phone: '13800000001',
    room: 'A', illness: '', warn: '', foodLike: '', foodHate: '', drag: '', customerId: 2 }
  const draft = { ...fields, oldId: 7, beTime: 'old', familyName: 'display', deviceId: 3 }
  expect(Object.keys(editableResident(draft))).toEqual([...residentFields])
  expect(editableResident(draft)).toEqual(fields)
})

it.each(['resetSecret', 'unbind', 'deleteDevice'])('%s displays the rejected write once', async method => {
  const post = vi.fn().mockRejectedValue({ response: { status: 409, data: { returnCode: '0000', code: 'CONFLICT', message: '设备归属已变化' } } })
  const error = vi.fn(), loadDevices = vi.fn()
  const page = { post, loadDevices, $confirm: vi.fn().mockResolvedValue('confirm'), $message: { error } }
  deviceAdminMethods[method as 'resetSecret' | 'unbind' | 'deleteDevice'].call(page, { deviceId: 9 })
  await flushPromises()
  expect(post).toHaveBeenCalledTimes(1)
  expect(error).toHaveBeenCalledExactlyOnceWith('设备归属已变化')
  expect(loadDevices).not.toHaveBeenCalled()
})
