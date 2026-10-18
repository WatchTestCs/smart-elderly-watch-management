<template>
  <section class="care-inbox" aria-labelledby="care-inbox-title">
    <div class="care-inbox-heading">
      <div><h2 id="care-inbox-title">需要接手的照护任务</h2><p>求助已接收不等于通知送达；只有人工接单及复核才会结束任务。</p></div>
      <el-button plain :loading="loading" @click="load">刷新任务</el-button>
    </div>
    <p v-if="error" role="alert" class="care-error">{{ error }}</p>
    <p v-else-if="!loading && !active.length" class="care-empty">当前没有查询到待处理任务。请继续检查设备与服务状态。</p>
    <div v-if="active.length" class="care-task-list">
      <article v-for="task in active" :key="task.id" class="care-task" :class="{ overdue: isOverdue(task) }">
        <div class="care-task-main">
          <strong>求助 #{{ task.id }} · {{ task.oldPersonName || '长者编号 ' + task.oldId }}</strong>
          <span>设备 {{ task.deviceId }} · {{ stateLabel(task.state) }}</span>
          <small>接收于 {{ localTime(task.createdAt) }} · {{ isOverdue(task) ? '已超出当前处理期限' : '处理期限：' + localTime(task.dueAt) }}</small>
          <small>责任人：{{ task.acknowledgedBy || '尚未接单' }} · 通知状态：{{ deliveryLabel(task.notificationState) }}</small>
        </div>
        <div class="care-task-actions">
          <el-button plain @click="viewHistory(task)">处置记录</el-button>
          <el-button v-if="['received','notifying','notified','escalated'].includes(task.state)" type="primary" @click="begin(task,'acknowledge')">接单</el-button>
          <el-button v-if="task.state==='acknowledged'" type="primary" @click="begin(task,'start')">开始处置</el-button>
          <el-button v-if="task.state==='in_progress'" type="primary" @click="begin(task,'resolve')">登记处置</el-button>
          <el-button v-if="['resolved','review_overdue'].includes(task.state)" type="primary" @click="begin(task,'verify')">复核关闭</el-button>
          <el-button v-if="['resolved','review_overdue'].includes(task.state)" plain @click="begin(task,'reopen')">重新开启</el-button>
        </div>
      </article>
    </div>
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
    </el-dialog>
    <el-dialog v-model="dialog.visible" :title="actionLabels[dialog.action] || '记录照护措施'" width="min(500px, 94vw)" append-to-body>
      <label for="care-action-note">处置或交班记录</label>
      <el-input id="care-action-note" v-model="dialog.note" type="textarea" :rows="4" maxlength="2000" show-word-limit placeholder="说明现场情况、处理措施及后续跟进" />
      <template #footer>
        <el-button @click="dialog.visible=false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="submit">确认记录</el-button>
      </template>
    </el-dialog>
  </section>
</template>
<script setup>
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import axios from 'axios'

const tasks = ref([])
const error = ref('')
const loading = ref(false)
const saving = ref(false)
const dialog = ref({visible:false,id:null,action:'',note:''})
const history = ref({visible:false,loading:false,error:'',events:[]})
const active = computed(() => tasks.value.filter(t => t.state !== 'verified'))
const actionLabels = {acknowledge:'确认接单',start:'开始处置',resolve:'登记处置措施',verify:'复核关闭',reopen:'重新开启'}
const deliveryLabel = state => ({pending:"待发送",sending:"发送中",failed:"发送失败待重试",sent:"渠道已接收",delivered:"已确认送达",no_recipient:"未找到通知对象"})[state] || "未确认"
const stateLabel = state => ({received:'等待通知与接单',notifying:'通知中',notified:'通知已送达',acknowledged:'已接单',in_progress:'处理中',resolved:'待复核',escalated:'超时升级',review_overdue:'复核超时'})[state] || state
const localTime = value => value ? new Date(value).toLocaleString('zh-CN') : '未知'
const isOverdue = task => Boolean(task.dueAt && Date.now() > new Date(task.dueAt).getTime())

let timer
async function load () {
  if (loading.value) return
  loading.value = true
  try {
    const { data } = await axios.post('restful/care/incidents', {page:1,pageSize:50}, {withCredentials:true})
    if (!Array.isArray(data.items)) throw new Error('invalid care response')
    tasks.value = data.items
    error.value = ''
  } catch (_) {
    error.value = '照护任务暂时无法加载，请检查后端连接和登录状态。'
  } finally { loading.value = false }
}
async function viewHistory (task) {
  history.value = {visible:true,loading:true,error:'',events:[]}
  try {
    const {data} = await axios.post('restful/care/events', {id:String(task.id)}, {withCredentials:true})
    if (!Array.isArray(data.events)) throw new Error('invalid history')
    history.value.events = data.events
  } catch (_) {
    history.value.error = '审计记录暂时无法加载'
  } finally {
    history.value.loading = false
  }
}
function begin (task, action) {
  dialog.value = {visible:true,id:task.id,action,note:''}
}
async function submit () {
  if (saving.value) return
  if (['resolve','verify','reopen'].includes(dialog.value.action) && !dialog.value.note.trim()) {
    error.value = '登记处置、复核或重新开启时必须填写记录'
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
  } catch (_) {
    error.value = '操作未确认成功，任务状态未刷新。请重新加载核实。'
  } finally {saving.value = false}
}
onMounted(() => {load();timer=setInterval(() => {if (!document.hidden) load()},15000)})
onBeforeUnmount(() => clearInterval(timer))
</script>
<style scoped>
.care-inbox{margin-bottom:24px;padding:22px;border:1px solid #d8e2da;border-radius:14px;background:#fff}
.care-inbox-heading{display:flex;justify-content:space-between;align-items:center;gap:16px;margin-bottom:15px}
.care-inbox-heading h2{font-size:20px;font-weight:600;margin:0;color:#235442}
.care-inbox-heading p{margin:6px 0 0;color:#53645c;font-size:14px}
.care-task-list{display:grid;gap:10px}
.care-task{display:flex;align-items:center;justify-content:space-between;gap:16px;padding:15px 18px;background:#f7f9f5;border:1px solid #dfe5df;border-radius:10px}
.care-task.overdue{border-left:4px solid #a4262c}
.care-task-main{display:grid;gap:4px;min-width:0}
.care-task-main strong{color:#263632}
.care-task-main span,.care-task-main small{color:#53645c}
.care-task-actions{display:flex;gap:8px;flex-wrap:wrap}
.care-error{color:#a4262c}
.care-history{list-style:decimal;padding-left:24px}
.care-history li{padding:10px 0;border-bottom:1px solid #e5ebe7}
.care-history small{display:block;color:#64756c}
.care-history p{margin:4px 0;color:#263632}
.care-empty{color:#53645c}
@media(max-width:700px){.care-inbox-heading,.care-task{align-items:stretch;flex-direction:column}}
</style>
