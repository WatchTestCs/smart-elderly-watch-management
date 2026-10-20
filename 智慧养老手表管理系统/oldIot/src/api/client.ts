import axios from 'axios'

export interface Envelope { returnCode?: string; message?: string }
export interface PostClient { post<T>(path: string, body: unknown): Promise<{ data: T }> }
export const api: PostClient = {
  post: <T>(path: string, body: unknown) => axios.post<T>(path, body, { withCredentials: true })
}
export function accepted(data: Envelope): void {
  if (data.returnCode !== '1111') throw new Error(data.message || '服务端未确认操作成功')
}
