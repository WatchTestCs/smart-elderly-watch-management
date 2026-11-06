import { ElMessage, ElMessageBox } from 'element-plus'
import { accepted, api, type Envelope } from '../api/client'
import { apiErrorMessage } from '../api/httpContract'

// Writes and prompts use Vue 3 services directly instead of receiving a component instance.
export async function writeRecord(
  path: string,
  body: unknown,
  success: string,
  refresh: () => unknown | Promise<unknown>,
  failure = '请求未完成，请重试'
): Promise<boolean> {
  try {
    const { data } = await api.post<Envelope>(path, body)
    accepted(data)
  } catch (error) {
    ElMessage({ type: 'error', message: apiErrorMessage(error, failure) })
    return false
  }
  ElMessage({ type: 'success', message: success })
  try {
    await refresh()
  } catch {
    ElMessage({ type: 'warning', message: '操作已成功，请手动刷新列表。' })
  }
  return true
}

export async function deleteRecord(
  path: string,
  body: unknown,
  prompt: string,
  refresh: () => unknown | Promise<unknown>
): Promise<boolean> {
  try {
    await ElMessageBox.confirm(prompt, '提示', {
      confirmButtonText: '确定', cancelButtonText: '取消', type: 'warning'
    })
  } catch {
    return false
  }
  return writeRecord(path, body, '删除信息成功', refresh, '删除失败，请稍后重试')
}
