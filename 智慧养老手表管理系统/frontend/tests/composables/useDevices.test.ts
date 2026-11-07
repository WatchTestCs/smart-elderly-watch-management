import { flushPromises, mount } from '@vue/test-utils'
import { defineComponent } from 'vue'
import { expect, it, vi } from 'vitest'
import axios from 'axios'
import { useDevices } from '@/composables/useDevices'

vi.mock('vue-router', () => ({ useRouter: () => ({ push: vi.fn() }) }))

it('keeps binding codes scoped to the currently selected resident', async () => {
  const replies = new Map<string, (value: unknown) => void>()
  vi.spyOn(axios, 'post').mockImplementation((url, body) => {
    if (url === '/api/v1/device/bindCodeList') {
      return new Promise(resolve => { replies.set(String((body as { oldId?: string | number }).oldId), resolve) })
    }
    if (url === '/api/v1/device/list') return Promise.resolve({ data: { devices: [], total: 0 } })
    if (url === '/api/v1/device/alertCounts') return Promise.resolve({ data: { items: [], total: 0 } })
    if (url === '/api/v1/inviteCode/oldPersonList') return Promise.resolve({ data: { oldPeople: [] } })
    throw new Error(`unexpected request: ${url}`)
  })
  let devices!: ReturnType<typeof useDevices>
  const wrapper = mount(defineComponent({
    setup() { devices = useDevices(); return () => null }
  }))
  try {
    await flushPromises()
    devices.openBindCodeDialog({ deviceId: 1, deviceNo: 'watch-1', oldId: 1 })
    await flushPromises()
    devices.bindOldId.value = 2
    await flushPromises()
    replies.get('2')?.({ data: { bindCodes: [{ id: 2, code: 'new', expireTime: 'tomorrow', used: 0 }] } })
    await flushPromises()
    replies.get('1')?.({ data: { bindCodes: [{ id: 1, code: 'old', expireTime: 'yesterday', used: 0 }] } })
    await flushPromises()
    expect(devices.bindCodeList.value).toEqual([{ id: 2, code: 'new', expireTime: 'tomorrow', used: 0 }])
    devices.bindCodeDialogVisible.value = false
    await flushPromises()
    expect(devices.bindCodeList.value).toEqual([{ id: 2, code: 'new', expireTime: 'tomorrow', used: 0 }])
  } finally {
    wrapper.unmount()
    vi.restoreAllMocks()
  }
})

it('does not attach a created binding code to a newly selected resident', async () => {
  const replies = new Map<string, (value: unknown) => void>()
  let finishCreate!: (value: unknown) => void
  vi.spyOn(axios, 'post').mockImplementation((url, body) => {
    if (url === '/api/v1/device/bindCodeList') {
      return new Promise(resolve => { replies.set(String((body as { oldId?: string | number }).oldId), resolve) })
    }
    if (url === '/api/v1/device/createBindCode') return new Promise(resolve => { finishCreate = resolve })
    if (url === '/api/v1/device/list') return Promise.resolve({ data: { devices: [], total: 0 } })
    if (url === '/api/v1/device/alertCounts') return Promise.resolve({ data: { items: [], total: 0 } })
    if (url === '/api/v1/inviteCode/oldPersonList') return Promise.resolve({ data: { oldPeople: [] } })
    throw new Error(`unexpected request: ${url}`)
  })
  let devices!: ReturnType<typeof useDevices>
  const wrapper = mount(defineComponent({
    setup() { devices = useDevices(); return () => null }
  }))
  try {
    await flushPromises()
    devices.openBindCodeDialog({ deviceId: 1, deviceNo: 'watch-1', oldId: 1 })
    await flushPromises()
    const creating = devices.createBindCode()
    await flushPromises()
    devices.bindOldId.value = 2
    await flushPromises()
    replies.get('2')?.({ data: { bindCodes: [{ id: 2, code: 'new', expireTime: 'tomorrow', used: 0 }] } })
    await flushPromises()
    finishCreate({ data: { returnCode: '1111', code: 'old', expireTime: 'today' } })
    await creating
    expect(devices.newBindCode.value).toBeNull()
    expect(devices.bindCodeList.value).toEqual([{ id: 2, code: 'new', expireTime: 'tomorrow', used: 0 }])
  } finally {
    wrapper.unmount()
    vi.restoreAllMocks()
  }
})

it('does not publish device rows from a query superseded during an active read', async () => {
  const listReplies: ((value: unknown) => void)[] = []
  vi.spyOn(axios, 'post').mockImplementation(url => {
    if (url === '/api/v1/device/list') return new Promise(resolve => { listReplies.push(resolve) })
    if (url === '/api/v1/device/alertCounts') return Promise.resolve({ data: { items: [], total: 0 } })
    if (url === '/api/v1/inviteCode/oldPersonList') return Promise.resolve({ data: { oldPeople: [] } })
    throw new Error(`unexpected request: ${url}`)
  })
  let devices!: ReturnType<typeof useDevices>
  const wrapper = mount(defineComponent({
    setup() { devices = useDevices(); return () => null }
  }))
  try {
    await flushPromises()
    expect(listReplies).toHaveLength(1)
    devices.changeDevicePage(2)
    listReplies[0]({ data: { devices: [{ deviceId: 1, deviceNo: 'stale' }], total: 1 } })
    await flushPromises()
    expect(devices.deviceList.value).toEqual([])
    expect(listReplies).toHaveLength(2)
    listReplies[1]({ data: { devices: [{ deviceId: 2, deviceNo: 'current' }], total: 1 } })
    await flushPromises()
    expect(devices.deviceList.value).toEqual([{ deviceId: 2, deviceNo: 'current' }])
    expect(devices.currentPage.value).toBe(2)
  } finally {
    wrapper.unmount()
    vi.restoreAllMocks()
  }
})
