<template>
<el-dialog :close-on-click-modal="false" title="设备绑定码" :model-value="visible" @update:model-value="onVisibleChange" width="560px">
      <HarmonyWatchSetup v-if="currentDevice" :device-no="currentDevice.deviceNo" />
      <div v-if="currentDevice" class="tip">
        设备：{{ currentDevice.deviceNo }}（{{ currentDevice.alias }}）；
        当前绑定老人：{{ currentDevice.oldPersonName }}
      </div>
      <el-form :inline="true" label-width="80px">
        <el-form-item label="绑定老人">
          <el-select v-model="selectedResident" placeholder="请选择老人" style="width: 160px;">
            <el-option v-for="item in oldPeople" :key="item.oldId"
                       :label="item.name + '（编号 ' + item.oldId + '）'" :value="item.oldId"></el-option>
          </el-select>
        </el-form-item>
        <el-form-item label="有效天数">
          <el-input v-model="expiryDays" style="width: 90px;"></el-input>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="emit('create')">生成绑定码</el-button>
        </el-form-item>
      </el-form>

      <div v-if="newBindCode" class="new-code">
        <div>生成成功，请在手表上输入：</div>
        <div class="code-text">{{ newBindCode.code }}
          <el-button size="small" plain type="primary" @click="emit('copy', newBindCode.code)">复制</el-button>
        </div>
        <div class="tip">有效期至 {{ newBindCode.expireTime }}，一个绑定码只能绑定一台手表</div>
      </div>

      <el-table :data="bindCodeList" size="small" border style="width: 100%;margin-top: 10px;">
        <el-table-column prop="code" label="绑定码" width="110" align="center"></el-table-column>
        <el-table-column prop="expireTime" label="过期时间" width="160" align="center"></el-table-column>
        <el-table-column label="状态" width="90" align="center">
          <template #default="scope" >
            <el-tag v-if="scope.row.used === 1" type="info" size="small">已使用</el-tag>
            <el-tag v-else type="success" size="small">未使用</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" align="center">
          <template #default="scope" >
            <el-button size="small" plain type="danger" :disabled="scope.row.used === 1"
                       @click="emit('delete', scope.row)">作废</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-dialog>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import HarmonyWatchSetup from './HarmonyWatchSetup.vue'
import type { DeviceId, DeviceRecord, ResidentRecord, DeviceBindCode, NewBindCodeResponse } from '../composables/deviceModels'

const props = defineProps<{
  visible: boolean
  currentDevice: DeviceRecord | null
  oldPeople: ResidentRecord[]
  bindOldId: DeviceId | ''
  bindExpireDays: string
  newBindCode: NewBindCodeResponse | null
  bindCodeList: DeviceBindCode[]
}>()
const emit = defineEmits<{
  close: []
  create: []
  delete: [code: DeviceBindCode]
  copy: [text: string]
  'update:bindOldId': [value: DeviceId | '']
  'update:bindExpireDays': [value: string]
}>()
const selectedResident = computed({
  get: () => props.bindOldId,
  set: value => emit('update:bindOldId', value)
})
const expiryDays = computed({
  get: () => props.bindExpireDays,
  set: value => emit('update:bindExpireDays', value)
})
function onVisibleChange(visible: boolean) {
  if (!visible) emit('close')
}
</script>

<style scoped>
.tip { color: #909399; font-size: 12px; line-height: 24px; padding-top: 6px; }
.new-code { padding: 10px; background: #f0f9f8; border-radius: 6px; }
.code-text { font-size: 26px; letter-spacing: 4px; color: #52bab5; font-weight: bold; padding: 4px 0; }
</style>
