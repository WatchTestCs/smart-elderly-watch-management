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
    listReplies[1]({ data: { devices: [{ deviceId: 2, deviceNo: 'current' }], total: 21 } })
    await flushPromises()
    expect(devices.deviceList.value).toEqual([{ deviceId: 2, deviceNo: 'current' }])
    expect(devices.currentPage.value).toBe(2)
  } finally {
    wrapper.unmount()
    vi.restoreAllMocks()
  }
})

it('limits workbench alert reads to six in flight and preserves device order', async () => {
  const devices = Array.from({ length: 12 }, (_, index) => ({ deviceId: index + 1, deviceNo: `watch-${index + 1}` }))
  const finish: Array<() => void> = []
  let active = 0
  let peak = 0
  vi.spyOn(axios, 'post').mockImplementation((url) => {
    if (url === '/api/v1/device/list') return Promise.resolve({ data: { devices, total: devices.length } })
    if (url === '/api/v1/device/alertCounts') {
      return Promise.resolve({ data: {
        items: devices.map(device => ({ deviceId: device.deviceId, personCount: 1, rateCount: 0, preCount: 0 })),
        total: devices.length
      } })
    }
    if (url === '/api/v1/device/alertDetail') {
      active += 1
      peak = Math.max(peak, active)
      return new Promise(resolve => finish.push(() => {
        active -= 1
        resolve({ data: { hasAlert: 1 } })
      }))
    }
    if (url === '/api/v1/inviteCode/oldPersonList') return Promise.resolve({ data: { oldPeople: [] } })
    throw new Error(`unexpected request: ${url}`)
  })
  let state!: ReturnType<typeof useDevices>
  const wrapper = mount(defineComponent({ setup() { state = useDevices(); return () => null } }))
  try {
    await flushPromises()
    const pending = state.openWorkbench()
    await flushPromises()
    expect(finish).toHaveLength(6)
    expect(peak).toBe(6)
    for (let index = 0; index < devices.length; index++) {
      finish[index]()
      await flushPromises()
    }
    await pending
    expect(peak).toBe(6)
    expect(state.workbench.value.list.map(row => row.deviceId)).toEqual(devices.map(row => row.deviceId))
  } finally {
    wrapper.unmount()
    vi.restoreAllMocks()
  }
})

it('keeps healthy workbench alerts visible when another detail request fails', async () => {
  const devices = [
    { deviceId: 1, deviceNo: 'watch-1' },
    { deviceId: 2, deviceNo: 'watch-2' },
    { deviceId: 3, deviceNo: 'watch-3' }
  ]
  vi.spyOn(axios, 'post').mockImplementation((url, body) => {
    if (url === '/api/v1/device/list') return Promise.resolve({ data: { devices, total: 3 } })
    if (url === '/api/v1/device/alertCounts') return Promise.resolve({ data: {
      items: devices.map(device => ({ deviceId: device.deviceId, personCount: 1, rateCount: 0, preCount: 0 })),
      total: 3
    } })
    if (url === '/api/v1/device/alertDetail') {
      const id = (body as { deviceId: number }).deviceId
      return id === 2 ? Promise.reject(new Error('temporary failure')) :
        Promise.resolve({ data: { hasAlert: 1 } })
    }
    if (url === '/api/v1/inviteCode/oldPersonList') return Promise.resolve({ data: { oldPeople: [] } })
    throw new Error(`unexpected request: ${url}`)
  })
  let state!: ReturnType<typeof useDevices>
  const wrapper = mount(defineComponent({ setup() { state = useDevices(); return () => null } }))
  try {
    await flushPromises()
    await state.openWorkbench()
    expect(state.workbench.value.list.map(row => row.deviceId)).toEqual([1, 3])
  } finally {
    wrapper.unmount()
    vi.restoreAllMocks()
  }
})

it('clears previous workbench results while refreshing alert details', async () => {
  const pendingDetails: Array<(value: unknown) => void> = []
  vi.spyOn(axios, 'post').mockImplementation(url => {
    if (url === '/api/v1/device/list') {
      return Promise.resolve({ data: { devices: [{ deviceId: 1, deviceNo: 'watch-1' }], total: 1 } })
    }
    if (url === '/api/v1/device/alertCounts') {
      return Promise.resolve({ data: { items: [{ deviceId: 1, personCount: 1, rateCount: 0, preCount: 0 }], total: 1 } })
    }
    if (url === '/api/v1/device/alertDetail') {
      return new Promise(resolve => pendingDetails.push(resolve))
    }
    if (url === '/api/v1/inviteCode/oldPersonList') return Promise.resolve({ data: { oldPeople: [] } })
    throw new Error(`unexpected request: ${url}`)
  })
  let state!: ReturnType<typeof useDevices>
  const wrapper = mount(defineComponent({ setup() { state = useDevices(); return () => null } }))
  try {
    await flushPromises()
    const first = state.openWorkbench()
    await flushPromises()
    pendingDetails[0]({ data: { hasAlert: 1 } })
    await first
    expect(state.workbench.value.list).toHaveLength(1)

    const refresh = state.openWorkbench()
    expect(state.workbench.value.list).toEqual([])
    await flushPromises()
    pendingDetails[1]({ data: { hasAlert: 1 } })
    await refresh
    expect(state.workbench.value.list).toHaveLength(1)
  } finally {
    wrapper.unmount()
    vi.restoreAllMocks()
  }
})

it('reloads page one after deleting the final device on a later page', async () => {
  const queries: number[] = []
  vi.spyOn(axios, 'post').mockImplementation((url, body) => {
    if (url === '/api/v1/device/list') {
      const page = (body as { page: number }).page
      queries.push(page)
      return Promise.resolve({ data: { devices: page === 1 ? [{ deviceId: 1, deviceNo: 'remaining' }] : [], total: 1 } })
    }
    if (url === '/api/v1/device/alertCounts') return Promise.resolve({ data: { items: [], total: 0 } })
    if (url === '/api/v1/inviteCode/oldPersonList') return Promise.resolve({ data: { oldPeople: [] } })
    throw new Error(`unexpected request: ${url}`)
  })
  let state!: ReturnType<typeof useDevices>
  const wrapper = mount(defineComponent({ setup() { state = useDevices(); return () => null } }))
  try {
    await flushPromises()
    state.changeDevicePage(2)
    await flushPromises()
    expect(queries).toEqual([1, 2, 1])
    expect(state.currentPage.value).toBe(1)
    expect(state.deviceList.value).toEqual([{ deviceId: 1, deviceNo: 'remaining' }])
  } finally {
    wrapper.unmount()
    vi.restoreAllMocks()
  }
})
