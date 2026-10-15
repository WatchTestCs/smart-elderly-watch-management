<template>
  <section class="admin-record-list" :aria-label="label">
    <div v-if="!records.length" class="care-empty-state" role="status"><strong>{{ emptyTitle }}</strong><span>记录会显示在这里，可调整搜索条件后重试。</span></div>
    <article v-for="(row, index) in records" :key="row[keyField] == null ? index : row[keyField]" class="admin-record">
      <header><h2>{{ row[titleKey] || '记录' }}</h2><span v-if="timeKey && row[timeKey]">{{ row[timeKey] }}</span></header>
      <dl><template v-for="field in fields.slice(0, 4)" :key="field.key"><dt>{{ field.label }}</dt><dd :class="stateClass(row, field)">{{ valueOf(row, field) }}</dd></template></dl>
      <details v-if="fields.length > 4" class="record-more"><summary>更多资料</summary><dl><template v-for="field in fields.slice(4)" :key="field.key"><dt>{{ field.label }}</dt><dd :class="stateClass(row, field)">{{ valueOf(row, field) }}</dd></template></dl></details>
      <div v-if="$slots.actions" class="record-actions"><slot name="actions" v-bind="{ row: row, $index: index }"></slot></div>
    </article>
  </section>
</template>

<script>
export default {
  name: 'RecordCards',
  props: {
    records: { type: Array, required: true }, fields: { type: Array, required: true },
    label: { type: String, required: true }, emptyTitle: { type: String, default: '暂无记录' },
    titleKey: { type: String, default: 'name' }, keyField: { type: String, default: 'oldId' },
    timeKey: { type: String, default: '' }
  },
  methods: {
    valueOf (row, field) {
      var value = row[field.key]
      if (field.state) {
        if (value === 1 || value === '1') return field.state === 'fall' ? '摔倒告警' : '异常'
        if (field.handled && (row[field.handled] === 1 || row[field.handled] === '1')) return '已处理'
        if (value === 0 || value === '0') return field.state === 'fall' ? '未发现摔倒告警' : '正常'
        return '暂无状态'
      }
      return value == null || value === '' ? '暂未提供' : value
    },
    stateClass (row, field) {
      if (!field.state) return ''
      if (row[field.key] === 1 || row[field.key] === '1') return 'status-danger'
      if (field.handled && (row[field.handled] === 1 || row[field.handled] === '1')) return 'status-handled'
      return row[field.key] === 0 || row[field.key] === '0' ? 'status-normal' : 'status-unknown'
    }
  }
}
</script>

<style scoped>
.admin-record-list { display: none; }
@media (max-width: 900px) {
  .admin-record-list { display: grid; gap: 20px; }
  .admin-record { min-width: 0; padding: 20px 0; border-top: 1px solid #dce4da; }
  .admin-record:first-child { padding-top: 4px; border-top: 0; }
  .admin-record header { margin-bottom: 20px; }
  .admin-record h2 { margin: 0; color: #183e33; font-size: 20px; font-weight: 500; line-height: 1.6; overflow-wrap: anywhere; }
  .admin-record header > span { display: block; margin-top: 6px; color: #53645c; font-size: 14px; }
  .admin-record dl { display: grid; grid-template-columns: minmax(0, 100px) minmax(0, 1fr); gap: 14px 16px; margin: 0; line-height: 1.8; }
  .admin-record dt { color: #53645c; font-size: 14px; }
  .admin-record dd { margin: 0; color: #263632; font-size: 16px; overflow-wrap: anywhere; }
  .record-more { margin-top: 20px; }
  .record-more summary { min-height: 44px; padding: 8px 0; color: #235442; cursor: pointer; }
  .record-more dl { padding: 12px 0; }
  .record-actions { display: flex; gap: 10px; flex-wrap: wrap; padding-top: 22px; }
  .record-actions :deep(.el-button) { margin: 0; }
  .admin-record .status-danger { color: #a4262c; }
  .admin-record .status-normal { color: #235442; }
  .admin-record .status-handled { color: #70531f; }
  .admin-record .status-unknown { color: #53645c; }
}
</style>
