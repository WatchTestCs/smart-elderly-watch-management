<template>
<el-form ref="form" label-width="80px" :model="draft" :rules="rules" :class="{ 'demo-ruleForm': !editing }" >
        <el-form-item label="姓名" prop="familyName">
          <el-input v-model="draft.familyName"></el-input>
        </el-form-item>
<el-form-item v-if="editing" label="姓名" prop="familyId" style="display: none;">
    <el-input v-model="draft.familyId"></el-input>
    </el-form-item>
        <el-form-item label="关系" prop="familyRelation">
          <el-input v-model="draft.familyRelation"></el-input>
        </el-form-item>
    <el-form-item label="性别" prop="familySex">
    <el-radio-group v-model="draft.familySex">
      <el-radio value="男">男</el-radio>
      <el-radio value="女">女</el-radio>
    </el-radio-group>
    </el-form-item>
    <el-form-item label="电话" prop="familyPhone">
    <el-input v-model="draft.familyPhone"></el-input>
    </el-form-item>
    <el-form-item label="地址" prop="familyAddress">
    <el-input v-model="draft.familyAddress"></el-input>
    </el-form-item>
    <el-form-item v-if="!editing" label="老人电话" prop="oldPhone">
    <el-input v-model="draft.oldPhone"></el-input>
    </el-form-item>
      </el-form>
</template>

<script>
export default {
  props: { draft: { type: Object, required: true }, rules: Object, editing: Boolean },
  methods: {
    validate (...args) { return this.$refs.form.validate(...args) },
    resetFields (...args) { return this.$refs.form.resetFields(...args) }
  }
}
</script>
