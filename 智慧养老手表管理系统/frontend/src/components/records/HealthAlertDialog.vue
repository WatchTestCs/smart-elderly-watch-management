<template>
    <el-dialog :title="dialog.title" v-model="dialog.visible" width="620px">
      <div v-if="dialog.loading" style="text-align:center;padding:20px;">正在读取 ...</div>
      <div v-else-if="dialog.hasAlert === 0" style="text-align:center;padding:20px;color:#606266;">
        <i class="el-icon-success" style="color:#52BAB5;font-size:32px;"></i>
        <div style="margin-top:8px;">{{ dialog.message }}</div>
      </div>
      <div v-else>
        <el-alert :title="'问题：' + dialog.reason" type="error" :closable="false" show-icon></el-alert>
        <table class="alert-table">
          <tbody>
            <tr><th>老人</th><td>{{ dialog.oldPersonName }}</td></tr>
            <tr><th>本次测量</th><td>心率 {{ dialog.hr }} 次/分　血压 {{ dialog.bp }} mmHg</td></tr>
            <tr><th>正常范围</th><td>心率 55-110 次/分；血压 90-140 mmHg</td></tr>
            <tr><th>发生时间</th><td>{{ dialog.occurTime }}</td></tr>
            <tr><th>判断依据</th><td>{{ dialog.detail }}</td></tr>
            <tr><th>怎么处理</th><td>{{ dialog.advice }}</td></tr>
          </tbody>
        </table>
        <div style="margin-top:14px;">
          <el-button v-if="dialog.rateBad === 1" type="primary" :loading="dialog.resolving"
                     @click="$emit('resolve', 'rate')">标记心率已处理</el-button>
          <el-button v-if="dialog.preBad === 1" type="primary" :loading="dialog.resolving"
                     @click="$emit('resolve', 'pre')">标记血压已处理</el-button>
          <el-button plain @click="$router.push({ path: '/device' })">去设备管理看该手表</el-button>
        </div>
        <div class="tip-small">
          处理后这条记录会从「异常」变成「已处理」，红点消失，历史记录与曲线仍然保留。
          如果老人基础心率/血压本身偏离默认范围，请到「基础管理-手表信息」调整阈值，避免反复报警。
        </div>
      </div>
      <template #footer><span  class="dialog-footer">
        <el-button @click="dialog.visible = false">关 闭</el-button>
      </span></template>
    </el-dialog>
</template>

<script>
export default {
  props: { dialog: { type: Object, required: true } },
  emits: ['resolve']
}
</script>
