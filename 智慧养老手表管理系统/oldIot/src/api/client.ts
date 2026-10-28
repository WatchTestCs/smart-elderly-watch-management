import axios from 'axios'

export interface Envelope { returnCode?: string; message?: string }
export interface PostClient { post<T>(path: string, body: unknown, options?: { signal?: AbortSignal }): Promise<{ data: T }> }
export const api: PostClient = {
  post: <T>(path: string, body: unknown, options?: { signal?: AbortSignal }) => axios.post<T>(path, body, { withCredentials: true, signal: options?.signal })
}
export function accepted(data: Envelope): void {
  if (data.returnCode !== '1111') throw new Error(data.message || '服务端未确认操作成功')
}
