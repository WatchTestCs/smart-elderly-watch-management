import { accepted, api } from '../api/client'
import { apiErrorMessage } from '../api/httpContract'

// 同一请求（路径+内容）未完成时忽略重复触发，避免双击确定键造成重复创建。
const inFlight = new Set()

export async function writeRecord(view, path, body, success, refresh, failure = '请求未完成，请重试') {
  const key = path + JSON.stringify(body)
  if (inFlight.has(key)) return
  inFlight.add(key)
  try {
    const { data } = await api.post(path, body)
    accepted(data)
    view.$message({ type: 'success', message: success })
    refresh()
  } catch (error) {
    view.$message({ type: 'error', message: apiErrorMessage(error, failure) })
  } finally {
    inFlight.delete(key)
  }
}

export async function deleteRecord(view, path, body, prompt, refresh) {
  try {
    await view.$confirm(prompt, '提示', { confirmButtonText: '确定', cancelButtonText: '取消', type: 'warning' })
  } catch {
    view.$message({ type: 'info', message: '已取消删除' })
    return
  }
  return writeRecord(view, path, body, '删除信息成功', refresh, '删除失败，请稍后重试')
}
