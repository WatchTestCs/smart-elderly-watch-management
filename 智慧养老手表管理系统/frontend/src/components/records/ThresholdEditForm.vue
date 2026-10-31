<template>
<el-form ref="form" label-width="80px" :model="draft" :rules="rules">

        <el-form-item label="手表Id" prop="deviceId">
        <el-input v-model="draft.deviceId" disabled="disabled"></el-input>
        </el-form-item>
        <el-form-item label="血压高值" prop="bpf">
        <el-input v-model="draft.bpf"></el-input>
        </el-form-item>
        <el-form-item label="血压低值" prop="bpl">
        <el-input v-model="draft.bpl"></el-input>
        </el-form-item>
        <el-form-item label="心率高值" prop="hrf">
        <el-input v-model="draft.hrf"></el-input>
        </el-form-item>
        <el-form-item label="心率低值" prop="hrl">
        <el-input v-model="draft.hrl"></el-input>
        </el-form-item>
      </el-form>
</template>

<script>
// The page owns the draft and requests; this component owns form validation.
export default {
  props: { draft: { type: Object, required: true }, rules: Object },
  methods: {
    validate (...args) { return this.$refs.form.validate(...args) },
    resetFields (...args) { return this.$refs.form.resetFields(...args) }
  }
}
</script>
