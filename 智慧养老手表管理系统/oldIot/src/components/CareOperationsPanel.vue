<template>
  <section v-if="!forbidden" class="care-operations" aria-labelledby="care-operations-title">
    <header class="care-operations-head">
      <div><h3 id="care-operations-title">值守与服务诊断</h3><p>以下为服务端任务统计，不代表人员到岗或通知送达。</p></div>
      <el-button plain :loading="loading" @click="refresh">检查服务</el-button>
    </header>
    <p v-if="error" class="care-error" role="alert">{{ error }}</p>
    <div v-if="diagnostics" class="care-operation-metrics">
      <span>待发送通知 <strong>{{ diagnostics.pendingNotifications }}</strong></span>
      <span>发送失败 <strong>{{ diagnostics.failedNotifications }}</strong></span>
      <span>发送中断 <strong>{{ diagnostics.staleSending }}</strong></span>
      <span>超时任务 <strong>{{ diagnostics.overdueIncidents }}</strong></span>
    </div>
    <div v-if="authorized" class="care-staff-settings">
      <div class="care-operations-head"><h4>岗位权限</h4><el-button plain @click="loadStaff">刷新人员</el-button></div>
      <p>主管可配置护理员或审计员。护理员可接单及记录处置；只有主管可复核关闭。</p>
      <div v-for="person in staff" :key="person.id" class="care-staff-row">
        <span>{{ person.name || '管理员' }}（编号 {{ person.id }}）</span>
        <el-select :model-value="person.duty" :aria-label="'岗位：'+person.id" @change="duty => assign(person,duty)">
          <el-option label="主管" value="supervisor" />
          <el-option label="护理员" value="caregiver" />
          <el-option label="审计员" value="auditor" />
        </el-select>
      </div>
      <el-button plain @click="showAudit = !showAudit; ifAudit()">查看敏感数据访问审计</el-button>
      <ol v-if="showAudit" class="care-access-list">
        <li v-for="(entry,index) in accesses" :key="index">
          {{ entry.actorId }} · {{ entry.resource }} · {{ new Date(entry.at).toLocaleString('zh-CN') }}
        </li>
      </ol>
    </div>
  </section>
</template>
<script setup>
import {ref,onMounted} from 'vue'
import axios from 'axios'
const diagnostics=ref(null)
const staff=ref([])
const accesses=ref([])
const authorized=ref(false)
const forbidden=ref(false)
const showAudit=ref(false)
const loading=ref(false)
const error=ref('')
const post=(url,body={})=>axios.post('restful/care/'+url,body,{withCredentials:true})
async function refresh(){
  loading.value=true
  try{
    const {data}=await post('diagnostics')
    diagnostics.value=data
    authorized.value=true
    forbidden.value=false
    error.value=''
    await loadStaff()
  }catch(e){
    authorized.value=false
    diagnostics.value=null
    if(e?.response?.status===403){forbidden.value=true;error.value=''}
    else{error.value='服务诊断不可用，请检查后端和数据库。'}
  }finally{loading.value=false}
}
async function loadStaff(){
  try{const {data}=await post('staff');staff.value=data.staff||[]}
  catch(_){error.value='人员权限列表无法加载。'}
}
async function assign(person,duty){
  if(person.duty===duty)return
  try{await post('assignDuty',{adminId:String(person.id),duty});await loadStaff();error.value=''}
  catch(_){error.value='岗位更改失败，请检查当前权限。';await loadStaff()}
}
async function ifAudit(){
  if(!showAudit.value)return
  try{const {data}=await post('accessAudit');accesses.value=data.accesses||[]}
  catch(_){error.value='审计记录暂不可用。'}
}
onMounted(refresh)
</script>
<style scoped>
.care-operations{margin:16px 0;padding:18px;border:1px solid var(--care-border,#dce4da);border-radius:12px;background:#fff}
.care-operations-head{display:flex;justify-content:space-between;gap:12px;align-items:center;flex-wrap:wrap}
.care-operations-head h3,.care-operations-head h4{margin:0;color:var(--care-heading,#235442)}
.care-operations-head p,.care-staff-settings p{margin:4px 0;color:var(--care-text-muted,#65756e)}
.care-operation-metrics{display:flex;gap:16px;flex-wrap:wrap;margin:16px 0}
.care-operation-metrics span{padding:12px;border-radius:8px;background:#f6f8f5}
.care-staff-row{display:flex;justify-content:space-between;align-items:center;padding:10px 0;border-top:1px solid #e4e9e2;gap:12px}
.care-staff-row .el-select{max-width:150px}
.care-access-list{max-height:200px;overflow:auto;font-size:13px;color:#47584f}
</style>
