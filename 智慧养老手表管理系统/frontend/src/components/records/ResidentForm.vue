<template>
<el-form ref="form" label-width="80px" :model="draft" :rules="rules" >
        <el-form-item label="姓名" prop="name">
          <el-input v-model="draft.name"></el-input>
        </el-form-item>
<el-form-item v-if="editing" label="年龄" prop="residentId" style="display: none;">
              <el-input v-model="draft.residentId"></el-input>
            </el-form-item>

    <el-form-item label="性别" prop="sex">
    <el-radio-group v-model="draft.sex">
      <el-radio value="男">男</el-radio>
      <el-radio value="女">女</el-radio>
    </el-radio-group>
    </el-form-item>
    <el-form-item label="电话" prop="phone">
    <el-input v-model="draft.phone"></el-input>
    </el-form-item>
    <el-form-item label="房间" prop="room">
    <el-input v-model="draft.room"></el-input>
    </el-form-item>
    <el-form-item label="病史" prop="illness">
    <el-input v-model="draft.illness"></el-input>
    </el-form-item>
    <el-form-item label="过敏药物" prop="warn">
    <el-input v-model="draft.warn"></el-input>
    </el-form-item>
    <el-form-item label="喜好食物" prop="foodLike">
    <el-input v-model="draft.foodLike"></el-input>
    </el-form-item>
    <el-form-item label="忌口食物" prop="foodHate">
    <el-input v-model="draft.foodHate"></el-input>
    </el-form-item>
    <el-form-item label="常用药物" prop="drug">
    <el-input v-model="draft.drug"></el-input>
    </el-form-item>
    <el-form-item label="出生日期" prop="birthdate">

      <el-date-picker type="date" placeholder="选择日期" value-format="YYYY-MM-DD" v-model="draft.birthdate" style="width: 100%;"></el-date-picker>

    </el-form-item>
    <el-form-item label="监护人" prop="guardianId">

        <el-select class="guardian-select" v-model="draft.guardianId" placeholder="请选择监护人" :loading="guardianLoading" :aria-busy="String(guardianLoading)">
        <el-option v-for="item in options"
      :key="item.guardianId"
      :label="item.guardianName"
      :value="item.guardianId">
      </el-option>
      </el-select>
          <p v-if="guardianError" role="alert">{{ guardianError }} <el-button plain :loading="guardianLoading" @click="emit('retry-guardians')">重新加载监护人</el-button></p>


    </el-form-item>
        <p v-if="!editing">建档后可在设备管理中关联手表。</p>
<el-form-item v-if="editing" label="手表编号" prop="deviceId">
          <el-input :model-value="draft.deviceId || '尚未关联'" readonly></el-input>
          <span>设备关联请在设备管理中修改。</span>
        </el-form-item>
      </el-form>
</template>

<script setup>
import { ref } from 'vue'
defineProps({
  draft: { type: Object, required: true },
  rules: Object, editing: Boolean, options: Array, guardianLoading: Boolean, guardianError: String
})
const emit = defineEmits(['retry-guardians'])
const form = ref(null)
defineExpose({
  validate: (...args) => form.value?.validate(...args),
  resetFields: (...args) => form.value?.resetFields(...args)
})
</script>
