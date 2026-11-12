export const CARE_ACTION_LABELS = {
  acknowledge: '确认接单',
  start: '开始处置',
  resolve: '登记处置措施',
  escalate: '升级跟进',
  verify: '复核关闭',
  reopen: '重新开启'
} as const

export type CareAction = keyof typeof CARE_ACTION_LABELS

export const DELIVERY_STATE_LABELS = {
  pending: '待发送',
  sending: '发送中',
  failed: '发送失败待重试',
  sent: '渠道已接收',
  delivered: '已确认送达',
  no_recipient: '未找到通知对象'
} as const

export type DeliveryState = keyof typeof DELIVERY_STATE_LABELS

export const CARE_TASK_STATE_LABELS = {
  received: '等待通知与接单',
  notifying: '通知中',
  notified: '通知已送达',
  acknowledged: '已接单',
  in_progress: '处理中',
  resolved: '待复核',
  escalated: '超时升级',
  review_overdue: '复核超时',
  verified: '已复核关闭'
} as const

export type CareTaskState = keyof typeof CARE_TASK_STATE_LABELS

export interface CareTask {
  id: string | number
  oldPersonName?: string
  oldId?: string | number
  deviceId?: string | number
  kind?: string
  severity?: string
  state: string
  createdAt?: string
  verifiedAt?: string
  dueAt?: string
  acknowledgedBy?: string
  notificationState?: string
  resolvedBy?: string | number
}

export interface CareEvent {
  action: string
  actor?: string
  note?: string
  createdAt?: string
}

export interface HistoryState {
  visible: boolean
  loading: boolean
  error: string
  events: CareEvent[]
  id: string | number | null
  page: number
  total: number
}

export interface CareListResponse {
  items: CareTask[]
  total: number
  duty: string
  actorId: number
}

export const actionLabels: Readonly<Record<string, string>> = CARE_ACTION_LABELS

export function deliveryLabel(state: string | undefined): string {
  return (state && DELIVERY_STATE_LABELS[state as DeliveryState]) || '未确认'
}

export function stateLabel(state: string): string {
  return CARE_TASK_STATE_LABELS[state as CareTaskState] ?? state
}

export function priorityLabel(task: CareTask): string {
  if (task.kind === 'sos' || task.kind === 'fall' || task.severity === 'critical') return 'P0 紧急'
  if (task.severity === 'high' || task.severity === 'warning') return 'P1 重点'
  if (task.severity === 'low' || task.severity === 'normal') return 'P2 常规'
  return '待分级'
}

export function priorityClass(task: CareTask): string {
  const priority = priorityLabel(task)
  if (priority.startsWith('P0')) return 'priority-p0'
  if (priority.startsWith('P1')) return 'priority-p1'
  return 'priority-normal'
}

export const localTime = (value?: string): string =>
  value ? new Date(value).toLocaleString('zh-CN') : '未知'

export const isOverdue = (task: CareTask): boolean =>
  Boolean(task.dueAt && Date.now() > new Date(task.dueAt).getTime())

export const CARE_NEXT_STEPS = {
  received: '值班人员立即接单并核实长者状况',
  notifying: '值班人员立即接单，不等待渠道回执',
  notified: '照护人员接单并到场核实',
  acknowledged: '开始现场处置并记录措施',
  in_progress: '完成现场处置并登记结果',
  resolved: '由另一名主管独立复核',
  escalated: '升级联系值班主管，优先接单',
  review_overdue: '主管立即复核或重新开启',
  verified: '可查看完整处置履历'
} as const satisfies Record<CareTaskState, string>

export function nextStep(task: CareTask): string {
  return CARE_NEXT_STEPS[task.state as CareTaskState] ?? '核对任务状态'
}
