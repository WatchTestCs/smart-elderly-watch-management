import { computed, onBeforeUnmount, onMounted, reactive, toRefs, watch } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { accepted, api, type Envelope } from '../api/client'
import { apiErrorMessage } from '../api/httpContract'
import { copyText } from '../utils/clipboard'

type DeviceId = number | string

export interface DeviceRecord {
  deviceId: DeviceId
  deviceNo: string
  oldId?: DeviceId | null
  alias?: string
  online?: number | string
  battery?: number | string
  model?: string
  oldPersonName?: string
  lastTime?: string
  lastLng?: number | string | null
  lastLat?: number | string | null
  kind?: string
  kindText?: string
  reason?: string
  recordId?: DeviceId
}
interface ResidentRecord { oldId: DeviceId; name: string }
interface DeviceBindCode { id: DeviceId; code: string; expireTime: string; used: number }
interface DeviceAlertCounts { personCount: number; rateCount: number; preCount: number }
interface DeviceAlertCountRecord extends DeviceAlertCounts { deviceId: DeviceId }
interface AlertCountsResponse {
  items: DeviceAlertCountRecord[]
  total: number
  personTotal?: number
  rateTotal?: number
  preTotal?: number
}
interface DeviceListResponse { devices: DeviceRecord[]; total?: number; onlineCount?: number }
interface AlertDetailResponse {
  hasAlert?: number
  message?: string
  oldPersonName?: string
  reason?: string
  detail?: string
  value?: string | number | null
  threshold?: string
  occurTime?: string
  advice?: string
  recordId?: DeviceId
  deviceNo?: string
  alias?: string
  kindText?: string
}
interface CreatedDeviceResponse { deviceNo: string; secret: string }
interface BindCodeListResponse { bindCodes?: DeviceBindCode[] }
interface NewBindCodeResponse { code: string; expireTime: string }
interface DeviceMutationResponse { secret?: string }
interface DeviceAlert {
  visible: boolean
  loading: boolean
  resolving: boolean
  hasAlert: number
  title: string
  kind: string
  deviceId: DeviceId | null
  deviceNo: string
  alias: string
  oldPersonName: string
  reason: string
  detail: string
  value: string | number
  threshold: string
  occurTime: string
  advice: string
  recordId: DeviceId | null
  message: string
}
interface DeviceState {
  deviceList: DeviceRecord[]
  deviceTotal: number
  serverOnlineCount: number
  currentPage: number
  pageSize: number
  reloadPending: boolean
  deviceDetailVisible: boolean
  detailDevice: DeviceRecord | null
  deviceQuery: string
  deviceState: string
  deviceLoading: boolean
  deviceLoadError: string
  alertLoadError: string
  alertsReady: boolean
  creating: boolean
  oldPeople: ResidentRecord[]
  autoRefresh: boolean
  createDialogVisible: boolean
  bindCodeDialogVisible: boolean
  createdDevice: CreatedDeviceResponse | null
  currentDevice: DeviceRecord | null
  bindCodeList: DeviceBindCode[]
  newBindCode: NewBindCodeResponse | null
  bindOldId: DeviceId | ''
  bindExpireDays: string
  createForm: { alias: string; model: string; oldId: DeviceId | '' }
  alertCounts: Record<string, DeviceAlertCounts>
  alertTotals: { personTotal: number; rateTotal: number; preTotal: number; total: number }
  alertDialog: DeviceAlert
  workbench: { visible: boolean; list: DeviceRecord[] }
}


const post = <T extends object = Record<string, never>>(path: string, params: Record<string, unknown> = {}) => api.post<Envelope & T>('/api/v1/device/' + path, params)

function emptyAlert(): DeviceAlert {
  return {
    visible: false, loading: false, resolving: false, hasAlert: 1,
    title: '异常原因', kind: 'person', deviceId: null, deviceNo: '',
    alias: '', oldPersonName: '', reason: '', detail: '', value: '',
    threshold: '', occurTime: '', advice: '', recordId: null, message: ''
  }
}

export function useDevices() {
  const router = useRouter()
  const state = reactive<DeviceState>({
    deviceList: [], deviceTotal: 0, serverOnlineCount: 0,
    currentPage: 1, pageSize: 20, reloadPending: false,
    deviceDetailVisible: false, detailDevice: null,
    deviceQuery: '', deviceState: 'all', deviceLoading: false,
    deviceLoadError: '', alertLoadError: '', alertsReady: false,
    creating: false, oldPeople: [], autoRefresh: true,
    createDialogVisible: false, bindCodeDialogVisible: false,
    createdDevice: null, currentDevice: null, bindCodeList: [], newBindCode: null,
    bindOldId: '', bindExpireDays: '1',
    createForm: { alias: '', model: 'HarmonyOS-Wear', oldId: '' },
    alertCounts: {},
    alertTotals: { personTotal: 0, rateTotal: 0, preTotal: 0, total: 0 },
    alertDialog: emptyAlert(),
    workbench: { visible: false, list: [] }
  })
  const visibleDevices = computed(() => state.deviceList)
  const onlineCount = computed(() => state.serverOnlineCount)
  let filterTimer: ReturnType<typeof setTimeout> | undefined
  let refreshTimer: ReturnType<typeof setTimeout> | undefined
  let disposed = false
  // Only the newest read may publish state, even if an older request ignores cancellation.
  let alertRequest = 0
  let alertCountsRequest = 0
  let workbenchRequest = 0
  let bindCodeRequest = 0
  let bindDialogRevision = 0

  async function loadAlertCounts() {
    const revision = ++alertCountsRequest
    if (disposed) return
    state.alertLoadError = ''
    try {
      const { data } = await post<AlertCountsResponse>('alertCounts')
      if (disposed || revision !== alertCountsRequest) return
      if (!Array.isArray(data.items) || typeof data.total !== 'number' || !Number.isSafeInteger(data.total)) throw new Error('missing alerts')
      state.alertCounts = Object.fromEntries(data.items.map(item => [item.deviceId, item]))
      state.alertTotals = {
        personTotal: data.personTotal ?? 0, rateTotal: data.rateTotal ?? 0,
        preTotal: data.preTotal ?? 0, total: data.total
      }
      state.alertsReady = true
    } catch (error) {
      if (disposed || revision !== alertCountsRequest) return
      state.alertsReady = false
      state.alertLoadError = apiErrorMessage(error, '异常统计暂时无法更新，请重新加载。')
    }
  }

  async function loadDevices() {
    if (disposed) return
    if (state.deviceLoading) {
      state.reloadPending = true
      return
    }
    const query = {
      page: state.currentPage, pageSize: state.pageSize,
      query: state.deviceQuery, state: state.deviceState
    }
    const isCurrentQuery = () =>
      query.page === state.currentPage && query.pageSize === state.pageSize &&
      query.query === state.deviceQuery && query.state === state.deviceState
    state.deviceLoading = true
    state.deviceLoadError = ''
    try {
      await Promise.all([
        post<DeviceListResponse>('list', query).then(({ data }) => {
          if (disposed) return
          if (!isCurrentQuery()) {
            state.reloadPending = true
            return
          }
          if (!Array.isArray(data.devices)) throw new Error('missing devices')
          state.deviceList = data.devices
          state.deviceTotal = typeof data.total === 'number' ? data.total : data.devices.length
          state.serverOnlineCount = typeof data.onlineCount === 'number' ? data.onlineCount :
            data.devices.filter(item => item.online === 1 || item.online === '1').length
        }).catch(error => {
          if (disposed) return
          if (!isCurrentQuery()) {
            state.reloadPending = true
            return
          }
          state.deviceLoadError = apiErrorMessage(error, '设备列表暂时无法更新，请重新加载。')
        }),
        loadAlertCounts()
      ])
    } finally {
      if (disposed) return
      state.deviceLoading = false
      if (state.reloadPending) {
        state.reloadPending = false
        void loadDevices()
      }
    }
  }
  function scheduleFilter() {
    clearTimeout(filterTimer)
    filterTimer = setTimeout(() => {
      state.currentPage = 1
      void loadDevices()
    }, 250)
  }
  function changeDevicePage(page: number) {
    state.currentPage = page
    void loadDevices()
  }

  function handleDeviceCommand(command: string, row: DeviceRecord) {
    if (command === 'details') {
      state.detailDevice = row
      state.deviceDetailVisible = true
    } else if (command === 'reset') void resetSecret(row)
    else if (command === 'unbind') void unbind(row)
    else if (command === 'delete') void deleteDevice(row)
  }
  function alertOf(deviceId: DeviceId): DeviceAlertCounts {
    return state.alertCounts[deviceId] || { personCount: 0, rateCount: 0, preCount: 0 }
  }
  async function showAlert(row: DeviceRecord, kind: string) {
    const revision = ++alertRequest
    state.alertDialog = {
      ...emptyAlert(), visible: true, loading: true,
      title: kind === 'rate' ? '心率不合格 · 原因与处理' :
        kind === 'pre' ? '血压不合格 · 原因与处理' : '人员不合格 · 原因与处理',
      kind, deviceId: row.deviceId, deviceNo: row.deviceNo, alias: row.alias || ''
    }
    try {
      const { data } = await post<AlertDetailResponse>('alertDetail', { deviceId: row.deviceId, kind })
      if (revision !== alertRequest || disposed) return
      Object.assign(state.alertDialog, {
        loading: false, hasAlert: data.hasAlert === undefined ? 1 : data.hasAlert,
        message: data.message || '', oldPersonName: data.oldPersonName || '',
        reason: data.reason || '', detail: data.detail || '',
        value: data.value == null ? '' : data.value, threshold: data.threshold || '',
        occurTime: data.occurTime || '', advice: data.advice || '',
        recordId: data.recordId === undefined ? null : data.recordId
      })
    } catch (error) {
      if (revision === alertRequest && !disposed) {
        state.alertDialog.loading = false
        ElMessage({ type: 'error', message: apiErrorMessage(error, '读取异常详情失败') })
      }
    }
  }
  async function resolveAlert() {
    if (state.alertDialog.resolving) return
    state.alertDialog.resolving = true
    try {
      const { data } = await post('resolveAlert', {
        deviceId: state.alertDialog.deviceId, kind: state.alertDialog.kind,
        recordId: state.alertDialog.recordId == null ? '' : state.alertDialog.recordId
      })
      accepted(data)
      ElMessage({ type: 'success', message: '已处理，红点已消除' })
      state.alertDialog.visible = false
      await loadDevices()
    } catch (error) {
      ElMessage({ type: 'error', message: apiErrorMessage(error, '处理失败') })
    } finally { state.alertDialog.resolving = false }
  }
  async function resolveAlertRow(row: DeviceRecord) {
    try {
      const { data } = await post('resolveAlert', {
        deviceId: row.deviceId, kind: row.kind, recordId: row.recordId || ''
      })
      accepted(data)
      ElMessage({ type: 'success', message: '已处理，红点已消除' })
      await loadDevices()
      await openWorkbench()
    } catch (error) {
      ElMessage({ type: 'error', message: apiErrorMessage(error, '处理失败') })
    }
  }
  async function openWorkbench() {
    const revision = ++workbenchRequest
    state.workbench.visible = true
    const kinds = ['person', 'rate', 'pre']
    const requests: Promise<DeviceRecord | null>[] = []
    for (const device of state.deviceList) {
      for (const kind of kinds) {
        const counts = alertOf(device.deviceId)
        const count = kind === 'person' ? counts.personCount :
          kind === 'rate' ? counts.rateCount : counts.preCount
        if (count < 1) continue
        requests.push(post<AlertDetailResponse>('alertDetail', { deviceId: device.deviceId, kind }).then(({ data }) =>
          data.hasAlert === 1 ? {
            deviceId: device.deviceId, deviceNo: data.deviceNo || device.deviceNo,
            alias: data.alias ?? device.alias ?? '', oldPersonName: data.oldPersonName ?? device.oldPersonName ?? '',
            kind, kindText: data.kindText, reason: data.reason, recordId: data.recordId
          } : null))
      }
    }
    try {
      const rows = (await Promise.all(requests)).filter((row): row is DeviceRecord => row !== null)
      if (!disposed && revision === workbenchRequest) state.workbench.list = rows
    } catch (error) {
      if (disposed || revision !== workbenchRequest) return
      ElMessage({ type: 'error', message: apiErrorMessage(error, '读取异常工作台失败') })
    }
  }
  function gotoPage(kind: string) {
    void router.push(kind === 'person' ? '/operations/monitoring-profiles' : '/operations/vitals')
    state.alertDialog.visible = false
  }

  async function loadOldPeople() {
    try {
      const { data } = await api.post<{ oldPeople: ResidentRecord[] }>('/api/v1/inviteCode/oldPersonList', {})
      if (!disposed) state.oldPeople = Array.isArray(data.oldPeople) ? data.oldPeople : []
    } catch (error) {
      if (!disposed) ElMessage({ type: 'error', message: apiErrorMessage(error, '读取老人列表失败') })
    }
  }
  function stopTimer() {
    clearTimeout(refreshTimer)
    refreshTimer = undefined
  }
  function startTimer() {
    stopTimer()
    if (!state.autoRefresh || document.hidden || disposed) return
    refreshTimer = setTimeout(async () => {
      refreshTimer = undefined
      try { await loadDevices() } finally { startTimer() }
    }, 5000)
  }
  async function handleVisibilityChange() {
    stopTimer()
    if (!document.hidden && state.autoRefresh) {
      await loadDevices()
      startTimer()
    }
  }
  function closeCreateDialog() {
    state.createDialogVisible = false
    state.createdDevice = null
    state.createForm = { alias: '', model: 'HarmonyOS-Wear', oldId: '' }
  }
  async function createDevice() {
    if (state.creating) return
    state.creating = true
    try {
      const { data } = await post<CreatedDeviceResponse>('create', {
        alias: state.createForm.alias || '新手表',
        model: state.createForm.model || 'HarmonyOS-Wear',
        oldId: state.createForm.oldId === '' ? '' : state.createForm.oldId
      })
      accepted(data)
      if (typeof data.deviceNo !== 'string' || typeof data.secret !== 'string') throw new Error('invalid device registration response')
      state.createdDevice = { deviceNo: data.deviceNo, secret: data.secret }
      ElMessage({ type: 'success', message: '设备登记成功' })
      await loadDevices()
    } catch (error) {
      ElMessage({ type: 'error', message: apiErrorMessage(error, '登记失败') })
    } finally { state.creating = false }
  }
  function openBindCodeDialog(row: DeviceRecord) {
    const wasVisible = state.bindCodeDialogVisible
    const oldId = row.oldId ?? ''
    const residentChanged = String(state.bindOldId) !== String(oldId)
    bindDialogRevision++
    state.currentDevice = row
    state.bindOldId = oldId
    state.bindExpireDays = '1'
    state.newBindCode = null
    state.bindCodeList = []
    state.bindCodeDialogVisible = true
    if (!wasVisible || !residentChanged) void loadBindCodes(oldId)
  }
  async function loadBindCodes(oldId: DeviceId | '') {
    const revision = ++bindCodeRequest
    if (disposed) return
    if (!oldId) {
      state.bindCodeList = []
      return
    }
    try {
      const { data } = await post<BindCodeListResponse>('bindCodeList', { oldId })
      // The dialog may now belong to another resident.
      if (disposed || revision !== bindCodeRequest || !state.bindCodeDialogVisible || String(state.bindOldId) !== String(oldId)) return
      state.bindCodeList = Array.isArray(data.bindCodes) ? data.bindCodes : []
    } catch (error) {
      if (!disposed && revision === bindCodeRequest && state.bindCodeDialogVisible) {
        ElMessage({ type: 'error', message: apiErrorMessage(error, '请求未完成，请重试') })
      }
    }
  }
  async function createBindCode() {
    const oldId = state.bindOldId
    if (!oldId) {
      ElMessage({ type: 'warning', message: '请先选择要绑定的老人' })
      return
    }
    if (!/^\d+$/.test(state.bindExpireDays)) {
      ElMessage({ type: 'warning', message: '有效天数只能填数字' })
      return
    }
    const revision = bindDialogRevision
    try {
      const { data } = await post<NewBindCodeResponse>('createBindCode', { oldId, expireDays: state.bindExpireDays })
      accepted(data)
      if (disposed || revision !== bindDialogRevision || !state.bindCodeDialogVisible || String(state.bindOldId) !== String(oldId)) return
      if (typeof data.code !== 'string' || typeof data.expireTime !== 'string') throw new Error('invalid binding code response')
      state.newBindCode = { code: data.code, expireTime: data.expireTime }
      await loadBindCodes(oldId)
    } catch (error) {
      if (!disposed && revision === bindDialogRevision && state.bindCodeDialogVisible && String(state.bindOldId) === String(oldId)) {
        ElMessage({ type: 'error', message: apiErrorMessage(error, '生成失败') })
      }
    }
  }
  async function deleteBindCode(row: { id: DeviceId }) {
    try {
      const { data } = await post('deleteBindCode', { id: row.id })
      accepted(data)
      ElMessage({ type: 'success', message: '已作废' })
      await loadBindCodes(state.bindOldId)
    } catch (error) {
      ElMessage({ type: 'error', message: apiErrorMessage(error, '请求未完成，请重试') })
    }
  }

  async function destructiveOperation(row: DeviceRecord, prompt: string, path: string, success: string, onSuccess?: (data: Envelope & DeviceMutationResponse) => unknown | Promise<unknown>) {
    try {
      await ElMessageBox.confirm(prompt, '提示', {
        confirmButtonText: '确定', cancelButtonText: '取消', type: 'warning'
      })
    } catch { return }
    try {
      const { data } = await post<DeviceMutationResponse>(path, { deviceId: row.deviceId })
      accepted(data)
      if (onSuccess) await onSuccess(data)
      else ElMessage({ type: 'success', message: success })
      await loadDevices()
    } catch (error) {
      ElMessage({ type: 'error', message: apiErrorMessage(error, '设备操作未完成，请重试') })
    }
  }
  function resetSecret(row: DeviceRecord) {
    return destructiveOperation(
      row, '重置后手表端需要用新密钥重新登录，是否继续？',
      'resetSecret', '密钥已重置',
      data => ElMessageBox.alert('新密钥：' + String(data.secret ?? ''), '密钥已重置', { confirmButtonText: '知道了' })
    )
  }
  function unbind(row: DeviceRecord) {
    return destructiveOperation(
      row, '解绑后该手表将不再上报到这位老人，是否继续？', 'unbind', '已解绑'
    )
  }
  function deleteDevice(row: DeviceRecord) {
    return destructiveOperation(
      row, '删除后需要重新登记设备，是否继续？', 'delete', '已删除'
    )
  }
  async function copy(value: string) {
    try {
      await copyText(value)
      ElMessage({ type: 'success', message: '已复制' })
    } catch {
      ElMessage({ type: 'warning', message: '复制失败，请手动记录。' })
    }
  }

  watch(() => state.bindOldId, oldId => {
    if (!state.bindCodeDialogVisible) return
    bindDialogRevision++
    state.bindCodeList = []
    state.newBindCode = null
    void loadBindCodes(oldId)
  }, { flush: 'sync' })
  watch(() => state.bindCodeDialogVisible, visible => {
    if (!visible) {
      bindCodeRequest++
      bindDialogRevision++
    }
  })
  watch(() => state.deviceQuery, scheduleFilter)
  watch(() => state.deviceState, scheduleFilter)
  watch(() => state.autoRefresh, enabled => enabled ? startTimer() : stopTimer())
  onMounted(() => {
    void loadDevices()
    void loadOldPeople()
    document.addEventListener('visibilitychange', handleVisibilityChange)
    startTimer()
  })
  onBeforeUnmount(() => {
    disposed = true
    alertRequest++
    alertCountsRequest++
    workbenchRequest++
    bindCodeRequest++
    bindDialogRevision++
    clearTimeout(filterTimer)
    document.removeEventListener('visibilitychange', handleVisibilityChange)
    stopTimer()
  })
  return {
    ...toRefs(state), visibleDevices, onlineCount,
    loadDevices, loadAlertCounts, scheduleFilter, changeDevicePage, handleDeviceCommand,
    alertOf, showAlert, resolveAlert, resolveAlertRow, openWorkbench, gotoPage,
    loadOldPeople, startTimer, stopTimer, handleVisibilityChange,
    closeCreateDialog, createDevice, openBindCodeDialog, loadBindCodes,
    createBindCode, deleteBindCode, resetSecret, unbind, deleteDevice, copy
  }
}
