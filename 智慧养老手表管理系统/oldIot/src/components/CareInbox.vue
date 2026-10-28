<template>
  <section class="care-inbox" aria-labelledby="care-inbox-title" :aria-busy="String(loading)">
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
      <article v-for="task in tasks" :key="task.id" class="care-task" :class="{ overdue: task.state !== 'verified' && isOverdue(task) }">
        <div class="care-task-main">
          <strong>求助 #{{ task.id }} · {{ task.oldPersonName || '长者编号 ' + task.oldId }}</strong>
          <span>设备 {{ task.deviceId }} · {{ stateLabel(task.state) }}</span>
          <small>接收于 {{ localTime(task.createdAt) }} · {{ task.state === 'verified' ? '复核于：' + localTime(task.verifiedAt) : isOverdue(task) ? '已超出当前处理期限' : '处理期限：' + localTime(task.dueAt) }}</small>
          <small>责任人：{{ task.acknowledgedBy || '尚未接单' }} · 通知状态：{{ deliveryLabel(task.notificationState) }}</small>
          <small class="care-next-step">下一步：{{ nextStep(task) }}</small>
        </div>
        <div class="care-task-actions">
          <el-button plain @click="viewHistory(task)">处置记录</el-button>
          <el-button v-if="canAct('acknowledge') && ['received','notifying','notified','escalated'].includes(task.state)" type="primary" @click="begin(task,'acknowledge')">接单</el-button>
          <el-button v-if="canAct('start') && task.state==='acknowledged'" type="primary" @click="begin(task,'start')">开始处置</el-button>
          <el-button v-if="canAct('resolve') && task.state==='in_progress'" type="primary" @click="begin(task,'resolve')">登记处置</el-button>
          <el-button v-if="canAct('verify') && String(task.resolvedBy) !== String(actorId) && ['resolved','review_overdue'].includes(task.state)" type="primary" @click="begin(task,'verify')">复核关闭</el-button>
          <el-button v-if="canAct('reopen') && ['resolved','review_overdue'].includes(task.state)" plain @click="begin(task,'reopen')">重新开启</el-button>
        </div>
      </article>
    </div>
    <el-pagination v-if="!error && total > pageSize" :current-page="page" :page-size="pageSize" :total="total" :disabled="saving" layout="total, prev, pager, next" @current-change="changePage" />
    <el-dialog v-model="history.visible" title="照护事件履历" width="min(600px, 94vw)" append-to-body>
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
<script setup>
import { ref, watch, onMounted, onBeforeUnmount } from 'vue'
import axios from 'axios'
import { createLatestRead, awaitAbortable } from '../composables/latestRead'
import CareOperationsPanel from './CareOperationsPanel.vue'

const emit = defineEmits(['verified'])
const tasks = ref([])
const error = ref('')
const loading = ref(false)
const saving = ref(false)
const duty = ref('')
const actorId = ref(null)
const scope = ref('active')
const page = ref(1)
const total = ref(0)
const pageSize = 20
const dialog = ref({visible:false,id:null,action:'',note:'',error:''})
const history = ref({visible:false,loading:false,error:'',events:[],id:null,page:1,total:0})
const canAct = action => !error.value && (duty.value === 'supervisor' || (duty.value === 'caregiver' && ['acknowledge','start','resolve'].includes(action)))
const actionLabels = {acknowledge:'确认接单',start:'开始处置',resolve:'登记处置措施',verify:'复核关闭',reopen:'重新开启'}
const deliveryLabel = state => ({pending:"待发送",sending:"发送中",failed:"发送失败待重试",sent:"渠道已接收",delivered:"已确认送达",no_recipient:"未找到通知对象"})[state] || "未确认"
const stateLabel = state => ({received:'等待通知与接单',notifying:'通知中',notified:'通知已送达',acknowledged:'已接单',in_progress:'处理中',resolved:'待复核',escalated:'超时升级',review_overdue:'复核超时',verified:'已复核关闭'})[state] || state
const localTime = value => value ? new Date(value).toLocaleString('zh-CN') : '未知'
const isOverdue = task => Boolean(task.dueAt && Date.now() > new Date(task.dueAt).getTime())
const nextStep = task => ({received:'值班人员立即接单并核实长者状况',notifying:'值班人员立即接单，不等待渠道回执',notified:'照护人员接单并到场核实',acknowledged:'开始现场处置并记录措施',in_progress:'完成现场处置并登记结果',resolved:'由另一名主管独立复核',escalated:'升级联系值班主管，优先接单',review_overdue:'主管立即复核或重新开启',verified:'可查看完整处置履历'})[task.state] || '核对任务状态'

let timer
const reads = createLatestRead(
  (query, signal) => axios.post('restful/care/incidents', query, {withCredentials:true, signal}),
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
  () => {
    tasks.value = []; total.value = 0; duty.value = ''; actorId.value = null
    error.value = '照护任务暂时无法加载，请检查后端连接和登录状态。'
  },
  value => { loading.value = value }
)
function load () {
  return reads.run({page:page.value,pageSize,state:scope.value})
}
function changePage (value) { page.value = value; load() }
function changeScope () { page.value = 1; load() }
let historyRequest = 0
let historyAbort
watch(() => history.value.visible, visible => {
  if (!visible) {
    historyRequest++
    historyAbort?.abort()
    historyAbort = undefined
    history.value.loading = false
  }
})
async function viewHistory (task) {
  history.value = {visible:true,loading:false,error:'',events:[],id:task.id,page:1,total:0}
  await loadHistory(1)
}
async function loadHistory (value) {
  const request = ++historyRequest
  historyAbort?.abort()
  const controller = new AbortController()
  historyAbort = controller
  const timeout = setTimeout(() => controller.abort(), 10_000)
  history.value.loading = true
  history.value.error = ''
  history.value.page = value
  try {
    const {data} = await awaitAbortable(
      axios.post('restful/care/events', {id:String(history.value.id),page:value,pageSize}, {withCredentials:true, signal:controller.signal}),
      controller.signal
    )
    if (request !== historyRequest) return
    if (!Array.isArray(data.events) || !Number.isInteger(data.total)) throw new Error('invalid history')
    history.value.events = data.events
    history.value.total = data.total
  } catch (_) {
    if (request === historyRequest) history.value.error = controller.signal.aborted
      ? '审计记录读取超时，请重试' : '审计记录暂时无法加载'
  } finally {
    clearTimeout(timeout)
    if (request === historyRequest) {
      history.value.loading = false
      historyAbort = undefined
    }
  }
}
function begin (task, action) {
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
    const {data} = await axios.post('restful/care/transition', {
      id:String(dialog.value.id),action:dialog.value.action,note:dialog.value.note.trim()
    }, {withCredentials:true})
    if (data.returnCode !== '1111') throw new Error('transition rejected')
    dialog.value.visible = false
    error.value = ''
    await load()
    if (dialog.value.action === 'verify') emit('verified')
  } catch (_) {
    dialog.value.error = '操作未确认成功，请关闭窗口并重新加载任务核实。'
  } finally {saving.value = false}
}
onMounted(() => {load();timer=setInterval(() => {if (!document.hidden && !loading.value && !saving.value && !dialog.value.visible && !history.value.visible) load()},15000)})
onBeforeUnmount(() => { clearInterval(timer); reads.dispose(); historyRequest++; historyAbort?.abort() })
</script>
<style scoped>
.care-inbox{margin-bottom:24px;padding:22px;border:1px solid #d8e2da;border-radius:14px;background:#fff}
.care-inbox-heading{display:flex;justify-content:space-between;align-items:center;gap:16px;margin-bottom:15px}
.care-inbox-heading h2{font-size:20px;font-weight:600;margin:0;color:#235442}
.care-inbox-heading p{margin:6px 0 0;color:#53645c;font-size:14px}
.care-inbox-tools{display:flex;align-items:center;gap:12px;flex-wrap:wrap}
.care-inbox-tools .el-select{width:160px}
.care-inbox :deep(.el-pagination){margin-top:18px;max-width:100%;overflow-x:auto}
.care-task-list{display:grid;gap:10px}
.care-task{display:flex;align-items:center;justify-content:space-between;gap:16px;padding:15px 18px;background:#f7f9f5;border:1px solid #dfe5df;border-radius:10px}
.care-task.overdue{border-left:4px solid #a4262c}
.care-task-main{display:grid;gap:4px;min-width:0}
.care-task-main strong{color:#263632}
.care-next-step{font-weight:600;color:#235442!important}
.care-task-main span,.care-task-main small{color:#53645c}
.care-task-actions{display:flex;gap:8px;flex-wrap:wrap}
.care-loading{margin:10px 0;color:#53645c;font-size:14px}
.care-error{color:#a4262c}
.care-history{list-style:decimal;padding-left:24px}
.care-history li{padding:10px 0;border-bottom:1px solid #e5ebe7}
.care-history small{display:block;color:#64756c}
.care-history p{margin:4px 0;color:#263632}
.care-empty{color:#53645c}
@media(max-width:700px){.care-inbox-heading,.care-task{align-items:stretch;flex-direction:column}}
</style>
