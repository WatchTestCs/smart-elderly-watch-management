<template>
  <section class="care-inbox" aria-labelledby="care-inbox-title" :aria-busy="loading">
    <div class="care-inbox-heading">
      <div><h2 id="care-inbox-title">{{ scope === 'verified' ? '已复核的照护任务' : '需要接手的照护任务' }}</h2><p>求助已接收不等于通知送达；只有人工接单及复核才会结束任务。</p></div>
      <div class="care-inbox-tools"><label for="care-task-scope">任务范围</label><el-select id="care-task-scope" v-model="scope" :disabled="saving" @change="changeScope"><el-option label="待复核任务" value="active"/><el-option label="已复核任务" value="verified"/></el-select><el-button plain :disabled="saving" @click="load">刷新任务</el-button></div>
    </div>
    <CareOperationsPanel />
    <p v-if="loading" role="status" class="care-loading">正在读取照护任务…（可继续刷新或切换分页）</p>
    <p v-if="error" role="alert" class="care-error">{{ error }}</p>
    <p v-else-if="!loading && !tasks.length" class="care-empty">{{ scope === 'verified' ? '当前没有已复核任务。' : '当前没有查询到待处理任务。请继续检查设备与服务状态。' }}</p>
    <p v-if="duty === 'auditor'" class="care-empty">当前为审计员岗位，可查看任务与履历。</p>
    <div v-if="tasks.length" class="care-task-list">
      <article v-for="task in tasks" :key="task.id" class="care-task" :class="{ overdue: task.state !== 'verified' && isOverdue(task), urgent: priorityLabel(task) === 'P0 紧急' }">
        <div class="care-task-main">
          <strong>求助 #{{ task.id }} · {{ task.oldPersonName || '长者编号 ' + task.oldId }}</strong>
          <span><strong class="care-priority" :class="priorityClass(task)">{{ priorityLabel(task) }}</strong> · 设备 {{ task.deviceId }} · {{ stateLabel(task.state) }}</span>
          <small>接收于 {{ localTime(task.createdAt) }} · {{ task.state === 'verified' ? '复核于：' + localTime(task.verifiedAt) : isOverdue(task) ? '已超出当前处理期限' : '处理期限：' + localTime(task.dueAt) }}</small>
          <small>责任人：{{ task.acknowledgedBy || '尚未接单' }} · 通知状态：{{ deliveryLabel(task.notificationState) }}</small>
          <small class="care-next-step">下一步：{{ nextStep(task) }}</small>
        </div>
        <div class="care-task-actions">
          <el-button plain @click="viewHistory(task)">处置记录</el-button>
          <el-button v-if="canAct('acknowledge') && ['received','notifying','notified','escalated'].includes(task.state)" type="primary" @click="begin(task,'acknowledge')">接单</el-button>
          <el-button v-if="canAct('start') && canContinue(task) && task.state==='acknowledged'" type="primary" @click="begin(task,'start')">开始处置</el-button>
          <el-button v-if="canAct('resolve') && canContinue(task) && task.state==='in_progress'" type="primary" @click="begin(task,'resolve')">登记处置</el-button>
          <el-button v-if="canAct('escalate') && ['received','notifying','notified','acknowledged','in_progress'].includes(task.state)" plain @click="begin(task,'escalate')">升级跟进</el-button>
          <el-button v-if="canAct('verify') && String(task.resolvedBy) !== String(actorId) && ['resolved','review_overdue'].includes(task.state)" type="primary" @click="begin(task,'verify')">复核关闭</el-button>
          <el-button v-if="canAct('reopen') && ['resolved','review_overdue'].includes(task.state)" plain @click="begin(task,'reopen')">重新开启</el-button>
        </div>
      </article>
    </div>
    <el-pagination v-if="!error && total > pageSize" :current-page="page" :page-size="pageSize" :total="total" :disabled="saving" layout="total, prev, pager, next" @current-change="changePage" />
    <el-dialog v-model="history.visible" title="照护事件履历" width="min(600px, 94vw)" append-to-body @close="historyReads.cancel">
      <p v-if="history.loading">正在读取记录…</p>
      <p v-if="history.error" role="alert">{{ history.error }}</p>
      <ol v-if="!history.loading && !history.error" class="care-history">
        <li v-for="(event, index) in history.events" :key="index">
          <strong>{{ event.action }}</strong> · {{ localTime(event.createdAt) }}
          <small>操作者：{{ event.actor }}</small>
          <p>{{ event.note || '无补充说明' }}</p>
        </li>
      </ol>
      <el-pagination v-if="!history.loading && !history.error && history.total > pageSize" :current-page="history.page" :page-size="pageSize" :total="history.total" layout="total, prev, pager, next" @current-change="loadHistory" />
    </el-dialog>
    <el-dialog v-model="dialog.visible" :title="actionLabels[dialog.action] || '记录照护措施'" width="min(500px, 94vw)" append-to-body :close-on-click-modal="!saving" :close-on-press-escape="!saving" :show-close="!saving">
      <p v-if="dialog.error" role="alert" class="care-error">{{ dialog.error }}</p>
      <label for="care-action-note">处置或交班记录</label>
      <el-input id="care-action-note" v-model="dialog.note" type="textarea" :rows="4" maxlength="2000" show-word-limit placeholder="说明现场情况、处理措施及后续跟进" />
      <template #footer>
        <el-button :disabled="saving" @click="dialog.visible=false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="submit">确认记录</el-button>
      </template>
    </el-dialog>
  </section>
</template>
<script setup lang="ts">
import { api as requestClient, type Envelope } from '../api/client'
import { apiErrorMessage, readErrorMessage } from '../api/httpContract'
import { ref, shallowRef, watch, onMounted, onBeforeUnmount } from 'vue'
import { createLatestRead } from '../composables/latestRead'
import CareOperationsPanel from './CareOperationsPanel.vue'
import store from '../store'
import {
  actionLabels, deliveryLabel, stateLabel, priorityLabel, priorityClass,
  localTime, isOverdue, nextStep
} from '../domain/care'
import type { CareTask, CareEvent, CareListResponse, HistoryState } from '../domain/care'

const emit = defineEmits(['verified'])
const tasks = shallowRef<CareTask[]>([])
const error = ref('')
const loading = ref(false)
const saving = ref(false)
const duty = ref('')
const actorId = ref<number | null>(null)
const scope = ref('active')
const page = ref(1)
const total = ref(0)
const pageSize = 20
const dialog = ref<{ visible: boolean; id: string | number | null; action: string; note: string; error: string }>({visible:false,id:null,action:'',note:'',error:''})
const history = ref<HistoryState>({visible:false,loading:false,error:'',events:[],id:null,page:1,total:0})
const canAct = (action: string) => !error.value && (duty.value === 'supervisor' || (duty.value === 'caregiver' && ['acknowledge','start','resolve'].includes(action)))
const canContinue = (task: CareTask) =>
  duty.value === 'supervisor' || String(task.acknowledgedBy ?? '') === String(actorId.value)

let timer: ReturnType<typeof setInterval> | undefined
const reads = createLatestRead(
  (query: { page: number; pageSize: number; state: string }, signal: AbortSignal) => requestClient.post<CareListResponse>('api/v1/care/incidents', query, { signal }),
  ({data}, query) => {
    if (!Array.isArray(data.items) || !Number.isInteger(data.total) || data.total < 0 || !['supervisor','caregiver','auditor'].includes(data.duty) || !Number.isInteger(data.actorId)) throw new Error('invalid care response')
    const lastPage = Math.max(1, Math.ceil(data.total / pageSize))
    if (query.page > lastPage) { page.value = lastPage; load(); return }
    tasks.value = data.items
    total.value = data.total
    duty.value = data.duty
    actorId.value = data.actorId
    error.value = ''
  },
  cause => {
    tasks.value = []; total.value = 0; duty.value = ''; actorId.value = null
    error.value = apiErrorMessage(cause, '照护任务暂时无法加载')
  },
  value => { loading.value = value }
)
function load () {
  return reads.run({page:page.value,pageSize,state:scope.value})
}
function changePage (value: number) { page.value = value; load() }
function changeScope () { page.value = 1; load() }
const historyReads = createLatestRead(
  (query: { id: string; page: number; pageSize: number }, signal: AbortSignal) => requestClient.post<{ events: CareEvent[]; total: number }>('api/v1/care/events', query, { signal }),
  ({data}) => {
    if (!Array.isArray(data.events) || !Number.isInteger(data.total)) throw new Error('invalid history')
    history.value.events = data.events
    history.value.total = data.total
  },
  error => { history.value.error = readErrorMessage(error, '审计记录暂时无法加载') },
  value => { history.value.loading = value }
)
async function viewHistory (task: CareTask) {
  history.value = {visible:true,loading:false,error:'',events:[],id:task.id,page:1,total:0}
  await loadHistory(1)
}
function loadHistory (value: number) {
  history.value.error = ''
  history.value.page = value
  return historyReads.run({id:String(history.value.id),page:value,pageSize})
}
function begin (task: CareTask, action: string) {
  dialog.value = {visible:true,id:task.id,action,note:'',error:''}
}
async function submit () {
  if (saving.value) return
  if (['resolve','verify','reopen'].includes(dialog.value.action) && !dialog.value.note.trim()) {
    dialog.value.error = '登记处置、复核或重新开启时必须填写记录'
    return
  }
  saving.value = true
  try {
    const {data} = await requestClient.post<Envelope>('api/v1/care/transition', {
      id:String(dialog.value.id),action:dialog.value.action,note:dialog.value.note.trim()
    })
    if (data.returnCode !== '1111') throw new Error('transition rejected')
    dialog.value.visible = false
    error.value = ''
    store.notifyAlertsChanged()
    await load()
    if (dialog.value.action === 'verify') emit('verified')
  } catch (cause) {
    dialog.value.error = apiErrorMessage(cause, '任务操作未完成')
  } finally {saving.value = false}
}
function refreshWhenVisible() {
  if (!document.hidden && !saving.value && !dialog.value.visible && !history.value.visible) void load()
}
watch(() => store.state.alertRevision, refreshWhenVisible)
onMounted(() => {
  void load()
  timer = setInterval(() => {
    if (!loading.value) refreshWhenVisible()
  }, 15000)
  document.addEventListener('visibilitychange', refreshWhenVisible)
})
onBeforeUnmount(() => {
  if (timer !== undefined) clearInterval(timer)
  document.removeEventListener('visibilitychange', refreshWhenVisible)
  reads.dispose()
  historyReads.dispose()
})
</script>
<style scoped>
.care-inbox{margin-bottom:24px;padding:22px;border:1px solid #d8e2da;border-radius:14px;background:#fff}
.care-inbox-heading{display:flex;justify-content:space-between;align-items:center;gap:16px;margin-bottom:15px}
.care-inbox-heading h2{font-size:20px;font-weight:600;margin:0;color:var(--care-primary)}
.care-inbox-heading p{margin:6px 0 0;color:var(--care-text-muted);font-size:14px}
.care-inbox-tools{display:flex;align-items:center;gap:12px;flex-wrap:wrap}
.care-inbox-tools .el-select{width:160px}
.care-inbox :deep(.el-pagination){margin-top:18px;max-width:100%;overflow-x:auto}
.care-task-list{display:grid;gap:10px}
.care-task{display:flex;align-items:center;justify-content:space-between;gap:16px;padding:15px 18px;background:#f7f9f5;border:1px solid #dfe5df;border-radius:10px}
.care-task.overdue,.care-task.urgent{border-left:4px solid var(--care-danger)}
.care-priority{font-weight:650;font-variant-numeric:tabular-nums}
.care-priority.priority-p0{color:var(--care-danger)}
.care-priority.priority-p1{color:#91651c}
.care-priority.priority-normal{color:var(--care-text-muted)}
.care-task-main{display:grid;gap:4px;min-width:0}
.care-task-main strong{color:var(--care-text)}
.care-next-step{font-weight:600;color:var(--care-primary)!important}
.care-task-main span,.care-task-main small{color:var(--care-text-muted)}
.care-task-actions{display:flex;gap:8px;flex-wrap:wrap}
.care-loading{margin:10px 0;color:var(--care-text-muted);font-size:14px}
.care-error{color:var(--care-danger)}
.care-history{list-style:decimal;padding-left:24px}
.care-history li{padding:10px 0;border-bottom:1px solid #e5ebe7}
.care-history small{display:block;color:#64756c}
.care-history p{margin:4px 0;color:var(--care-text)}
.care-empty{color:var(--care-text-muted)}
@media(max-width:700px){.care-inbox-heading,.care-task{align-items:stretch;flex-direction:column}}
</style>
