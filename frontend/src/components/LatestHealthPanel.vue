<template>
  <section class="latest-health" aria-labelledby="latest-health-heading" :aria-busy="loading">
    <div class="latest-heading">
      <div><h2 id="latest-health-heading">最近健康采样</h2><p>各来源分别展示，按采样开始时间选取；历史记录保留原时间。</p></div>
      <el-button link :loading="loading" @click="load">刷新采样</el-button>
    </div>
    <p v-if="error" role="alert">{{ error }}</p>
    <p v-else-if="loading" role="status">正在读取最近采样…</p>
    <p v-else-if="!items.length" class="latest-empty">暂无健康采样；日常上报记录仍可在心率、睡眠与步数页面查看。</p>
    <ul v-else class="latest-list">
      <li v-for="item in items" :key="item.sampleId">
        <h3>{{ metricLabels[item.sample.metric] || item.sample.metric }}</h3>
        <p class="latest-value">{{ item.sample.value == null ? '暂无数值' : item.sample.value }} <span>{{ unitLabels[item.sample.unit] || item.sample.unit }}</span></p>
        <p>{{ semanticsLabels[item.sample.semantics] || item.sample.semantics }}<span v-if="item.sample.stage"> · {{ stageLabels[item.sample.stage] || item.sample.stage }}</span></p>
        <p v-if="item.sample.timeBasis === 'legacy-unzoned'">原始时间：{{ item.sample.start }}<span v-if="item.sample.semantics === 'interval'"> — {{ item.sample.end }}</span>（时区未确认）</p>
        <p v-else-if="item.sample.semantics === 'interval'">{{ item.sample.metadata?.timestampBasis === 'app-received' ? '应用观测区间' : '采样区间' }}：{{ localTime(item.sample.start) }} — {{ localTime(item.sample.end) }}</p>
        <p v-else>{{ item.sample.metadata?.timestampBasis === 'app-received' ? '应用接收时间' : '采样时间' }}：{{ localTime(item.sample.start) }}</p>
        <p class="latest-source">来源：{{ item.sample.source === 'suining-harmonyos' ? '鸿蒙手表（应用上报）' : item.sample.source }} · {{ item.sample.sourceDevice }}<span v-if="item.deviceId != null"> · 绑定版本 {{ item.bindingVersion }}</span></p>
      </li>
    </ul>
  </section>
</template>

<script setup lang="ts">
import { apiErrorMessage } from '../api/httpContract'
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { readLatestHealth, type LatestHealthRecord } from '../api/health'
import { createLatestRead } from '../composables/latestRead'

const items = ref<LatestHealthRecord[]>([])
const loading = ref(false)
const error = ref('')
const reads = createLatestRead<void, LatestHealthRecord[]>(
  (_, signal) => readLatestHealth(signal),
  records => { items.value = records; error.value = '' },
  cause => { error.value = apiErrorMessage(cause, '健康采样暂时无法加载，请点击刷新采样重试。') },
  busy => { loading.value = busy }
)
const load = () => reads.run()
onMounted(load)
onBeforeUnmount(reads.dispose)

const metricLabels: Record<string, string> = {
  heart_rate: '心率', blood_pressure_systolic: '收缩压', blood_pressure_diastolic: '舒张压',
  legacy_blood_pressure: '血压（旧协议）', oxygen_saturation: '血氧饱和度', body_temperature: '体温',
  body_mass: '体重', steps: '步数', sleep_duration: '睡眠时长', sleep_stage: '睡眠阶段'
}
const unitLabels: Record<string, string> = { 'count/min': '次/分', count: '步', min: '分钟', degC: '℃', kg: '千克', mmHg: 'mmHg', '%': '%' }
const semanticsLabels: Record<string, string> = { instant: '单次采样', daily_total: '日累计快照', interval: '区间采样' }
const stageLabels: Record<string, string> = { asleep: '睡眠', in_bed: '卧床', awake: '清醒', core: '浅睡', deep: '深睡', rem: '快速眼动' }
const localTime = (value: string) => new Date(value).toLocaleString('zh-CN', { timeZoneName: 'short' })
</script>

<style scoped>
.latest-health { margin: 0 0 28px; padding: 24px 28px; border: 1px solid #dfe6dc; border-radius: 12px; background: #fff; color: var(--care-heading); }
.latest-heading { display: flex; justify-content: space-between; align-items: start; gap: 16px; }
.latest-heading h2 { margin: 0; font-size: 23px; font-weight: 500; }
.latest-heading p, .latest-empty { color: var(--care-text-muted); line-height: 1.8; }
.latest-list { display: grid; grid-template-columns: repeat(auto-fit, minmax(min(100%, 260px), 1fr)); gap: 20px; margin: 16px 0 0; padding: 0; list-style: none; }
.latest-list li { padding-top: 16px; border-top: 1px solid #dfe6dc; min-width: 0; }
.latest-list h3 { margin: 0; font-size: 18px; font-weight: 500; }
.latest-list p { margin: 8px 0; line-height: 1.7; overflow-wrap: anywhere; }
.latest-list .latest-value { font-size: 26px; color: var(--care-primary); }
.latest-value span, .latest-source { font-size: 14px; color: var(--care-text-muted); }
@media (max-width: 700px) { .latest-health { padding: 20px 18px; } }
</style>
