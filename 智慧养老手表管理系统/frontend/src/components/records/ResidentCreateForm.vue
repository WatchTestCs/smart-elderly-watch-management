<template>
<el-form ref="form" label-width="80px" :model="draft" :rules="rules" >
        <el-form-item label="姓名" prop="name">
          <el-input v-model="draft.name"></el-input>
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
    <el-form-item label="常用药物" prop="drag">
    <el-input v-model="draft.drag"></el-input>
    </el-form-item>
    <el-form-item label="出生日期" prop="bothdate">

      <el-date-picker type="date" placeholder="选择日期" value-format="YYYY-MM-DD" v-model="draft.bothdate" style="width: 100%;"></el-date-picker>

    </el-form-item>
    <el-form-item label="监护人" prop="customerId">

        <el-select class="guardian-select" v-model="draft.customerId" placeholder="请选择监护人" :loading="customerLoading" :aria-busy="String(customerLoading)">
        <el-option v-for="item in options"
      :key="item.customerId"
      :label="item.customerName"
      :value="item.customerId">
      </el-option>
      </el-select>
          <p v-if="customerError" role="alert">{{ customerError }} <el-button plain :loading="customerLoading" @click="$emit('retry-guardians')">重新加载监护人</el-button></p>


    </el-form-item>
        <p>建档后可在设备管理中关联手表。</p>
      </el-form>
</template>

<script>
// The page owns the draft and requests; this component owns form validation.
export default {
  props: { draft: { type: Object, required: true }, rules: Object, options: Array, customerLoading: Boolean, customerError: String },
  emits: ['retry-guardians'],
  methods: {
    validate (...args) { return this.$refs.form.validate(...args) },
    resetFields (...args) { return this.$refs.form.resetFields(...args) }
  }
}
</script>
