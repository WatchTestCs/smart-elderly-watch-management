import { AxiosError, isCancel, type AxiosResponse } from 'axios'
import statusErrors from './status-errors.json'

export interface ApiFailure {
  returnCode: '0000'
  code: string
  message: string
  reference?: string
}

export function isSuccessEnvelope(value: unknown): value is { returnCode: '1111' } {
  return value !== null && typeof value === 'object' && !Array.isArray(value) &&
    (value as { returnCode?: unknown }).returnCode === '1111'
}

export function isApplicationAPI(url: string | undefined): boolean {
  if (!url) return false
  try {
    const target = new URL(url, window.location.href)
    return target.origin === window.location.origin && target.pathname.startsWith('/api/v1/')
  } catch {
    return false
  }
}

export function apiErrorMessage(error: unknown, fallback = '请求未完成'): string {
  if (isCancel(error)) return '请求已取消'
  const failure = error as { response?: { data?: Partial<ApiFailure>; status?: number }; name?: string; message?: string; status?: number; cause?: Partial<ApiFailure> }
  let data = failure?.response?.data || (failure?.cause?.returnCode === '0000' ? failure.cause : undefined)
  if (!data && failure?.name === 'UploadAjaxError' && failure.message) {
    try { data = JSON.parse(failure.message) as Partial<ApiFailure> } catch { /* Plain upload transport error. */ }
  }
  if (data?.returnCode === '0000' && typeof data.code === 'string' && typeof data.message === 'string' && data.message.trim()) {
    return data.message + (typeof data.reference === 'string' && data.reference ? `（编号：${data.reference}）` : '')
  }
  const status = failure?.response?.status || failure?.status
  const definition = statusErrors[String(status) as keyof typeof statusErrors]
  return definition?.message || fallback
}

export function assertApiResponse(response: AxiosResponse): AxiosResponse {
  if (!isApplicationAPI(response.config.url) || ['blob', 'arraybuffer'].includes(response.config.responseType || '')) return response
  if (isSuccessEnvelope(response.data)) return response
  const message = response.data?.returnCode === '0000'
    ? apiErrorMessage({ response }, '请求被拒绝')
    : '服务响应格式错误'
  throw new AxiosError(message, 'ERR_BAD_RESPONSE', response.config, response.request, response)
}
