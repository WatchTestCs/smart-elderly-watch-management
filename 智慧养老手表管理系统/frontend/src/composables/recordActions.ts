import type { ElMessage, ElMessageBox } from 'element-plus'
import { accepted, api, type Envelope } from '../api/client'
import { apiErrorMessage } from '../api/httpContract'

type MessageHost = { $message: typeof ElMessage }
type ConfirmationHost = MessageHost & { $confirm: typeof ElMessageBox.confirm }

export async function writeRecord(view: MessageHost, path: string, body: unknown, success: string, refresh: () => unknown, failure = '请求未完成，请重试') {
  try {
    const { data } = await api.post<Envelope>(path, body)
    accepted(data)
    view.$message({ type: 'success', message: success })
    refresh()
  } catch (error) {
    view.$message({ type: 'error', message: apiErrorMessage(error, failure) })
  }
}

export async function deleteRecord(view: ConfirmationHost, path: string, body: unknown, prompt: string, refresh: () => unknown) {
  try {
    await view.$confirm(prompt, '提示', { confirmButtonText: '确定', cancelButtonText: '取消', type: 'warning' })
  } catch {
    view.$message({ type: 'info', message: '已取消删除' })
    return
  }
  return writeRecord(view, path, body, '删除信息成功', refresh, '删除失败，请稍后重试')
}
