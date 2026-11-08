import { api, type Envelope } from './client'

export interface InvitationResult extends Envelope {
  residentName?: string
  remainDays?: number
  codes?: string[]
  inviteCodes?: unknown[]
  residents?: unknown[]
  total?: number
}

// Keep the Go invitation routes and cookie policy in one place.
export const invitations = {
  checkCode: (body: { code: string }) => api.post<InvitationResult>('api/v1/inviteCode/checkCode', body),
  join: (body: Record<string, unknown>) => api.post<InvitationResult>('api/v1/inviteCode/join', body),
  list: (body: { page: number; pageSize: number; state: string }) => api.post<InvitationResult>('api/v1/inviteCode/list', body),
  residentList: () => api.post<InvitationResult>('api/v1/inviteCode/residentList', {}),
  create: (body: Record<string, unknown>) => api.post<InvitationResult>('api/v1/inviteCode/create', body),
  delete: (body: { id: number }) => api.post<InvitationResult>('api/v1/inviteCode/delete', body)
}
