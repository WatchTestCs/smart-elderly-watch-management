export interface NoticeRecord {
  textId: string | number
  picId?: string | number
  title?: string
  time?: string
  textMain?: string
}

export interface NoticePicture {
  picName: string
}

export interface NoticeComment {
  saidId: string | number
  personName?: string
  saidTime?: string
  said?: string
  back?: string
  good?: number | string
  niceCount?: number
}

export interface NoticeDetailResponse {
  text?: NoticeRecord
  pics?: NoticePicture[]
}

export interface NoticeCommentsResponse {
  comments?: NoticeComment[]
}

export function noticeImageUrl(name: string): string {
  return '/api/v1/image/download?name=' + encodeURIComponent(name)
}
