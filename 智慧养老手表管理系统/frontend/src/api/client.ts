import axios from 'axios'
import { isSuccessEnvelope } from './httpContract'

export interface Envelope { returnCode: '1111' | '0000'; code?: string; message?: string; reference?: string }
export interface PostClient { post<T>(path: string, body: unknown, options?: { signal?: AbortSignal }): Promise<{ data: T }> }
export const api: PostClient = {
  post: <T>(path: string, body: unknown, options?: { signal?: AbortSignal }) => axios.post<T>(path, body, { withCredentials: true, signal: options?.signal })
}
export function accepted(data: Envelope): void {
  if (!isSuccessEnvelope(data)) throw new Error(data?.message || '请求被拒绝', { cause: data })
}
