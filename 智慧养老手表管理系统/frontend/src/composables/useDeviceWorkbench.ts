import { ElMessage } from 'element-plus'
import { readErrorMessage } from '../api/httpContract'
import type { AlertDetailResponse, DeviceId, DeviceRecord, DeviceState } from './deviceModels'

type AlertKind = 'person' | 'rate' | 'pre'
interface AlertRequest { device: DeviceRecord; kind: AlertKind }

const detailTimeoutMs = 3000

// Owns the bounded detail queue and prevents obsolete dialog reads from publishing.
export function useDeviceWorkbench(
  state: DeviceState,
  readDetail: (deviceId: DeviceId, kind: AlertKind, signal: AbortSignal) => Promise<AlertDetailResponse>
) {
  let revision = 0
  let disposed = false
  const activeReads = new Set<AbortController>()

  function abortReads(): void {
    for (const controller of activeReads) controller.abort()
    activeReads.clear()
  }

  async function openWorkbench(): Promise<void> {
    abortReads()
    const request = ++revision
    const current = () => !disposed && request === revision
    state.workbench.visible = true
    state.workbench.list = []

    const pending: AlertRequest[] = []
    for (const device of state.deviceList) {
      const counts = state.alertCounts[device.deviceId]
      if (!counts) continue
      if (counts.personCount > 0) pending.push({ device, kind: 'person' })
      if (counts.rateCount > 0) pending.push({ device, kind: 'rate' })
      if (counts.preCount > 0) pending.push({ device, kind: 'pre' })
    }

    const results: (DeviceRecord | null)[] = Array(pending.length).fill(null)
    let nextIndex = 0
    let failed = false
    let firstError: unknown = null

    async function worker(): Promise<void> {
      while (nextIndex < pending.length && current()) {
        const index = nextIndex++
        const { device, kind } = pending[index]
        const controller = new AbortController()
        activeReads.add(controller)
        const timeout = setTimeout(() => controller.abort(), detailTimeoutMs)
        try {
          const detail = await readDetail(device.deviceId, kind, controller.signal)
          if (!current()) return
          if (detail.hasAlert !== 1) continue
          results[index] = {
            deviceId: device.deviceId, deviceNo: detail.deviceNo || device.deviceNo,
            alias: detail.alias ?? device.alias ?? '',
            oldPersonName: detail.oldPersonName ?? device.oldPersonName ?? '',
            kind, kindText: detail.kindText, reason: detail.reason, recordId: detail.recordId
          }
        } catch (error) {
          if (!current()) return
          failed = true
          if (firstError === null) firstError = error
        } finally {
          clearTimeout(timeout)
          activeReads.delete(controller)
        }
      }
    }

    await Promise.all(Array.from({ length: Math.min(6, pending.length) }, () => worker()))
    if (!current()) return
    state.workbench.list = results.filter((row): row is DeviceRecord => row !== null)
    if (failed) ElMessage({ type: 'warning', message: readErrorMessage(firstError, '部分异常详情加载失败') })
  }

  function disposeWorkbench(): void {
    disposed = true
    revision++
    abortReads()
  }

  return { openWorkbench, disposeWorkbench }
}
