<template>
  <section class="admin-record-list" :aria-label="label">
    <div v-if="!records.length" class="care-empty-state" role="status"><strong>{{ emptyTitle }}</strong><span>记录会显示在这里，可调整搜索条件后重试。</span></div>
    <article v-for="(row, index) in records" :key="recordKey(row, index)" class="admin-record">
      <header><h2>{{ valueAt(row, titleKey) || '记录' }}</h2><span v-if="timeKey && valueAt(row, timeKey)">{{ valueAt(row, timeKey) }}</span></header>
      <dl><template v-for="field in primaryFields" :key="field.key"><dt>{{ field.label }}</dt><dd :class="stateClass(row, field)">{{ valueOf(row, field) }}</dd></template></dl>
      <details v-if="extraFields.length" class="record-more"><summary>更多资料</summary><dl><template v-for="field in extraFields" :key="field.key"><dt>{{ field.label }}</dt><dd :class="stateClass(row, field)">{{ valueOf(row, field) }}</dd></template></dl></details>
      <div v-if="$slots.actions" class="record-actions"><slot name="actions" v-bind="{ row: row, $index: index }"></slot></div>
    </article>
  </section>
</template>

<script setup lang="ts" generic="Row extends object">
import { computed } from 'vue'

interface RecordField {
  key: string
  label: string
  state?: string
  handled?: string
}

const props = withDefaults(defineProps<{
  records: Row[]
  fields: RecordField[]
  label: string
  emptyTitle?: string
  titleKey?: string
  keyField?: string
  timeKey?: string
}>(), { emptyTitle: '暂无记录', titleKey: 'name', keyField: 'oldId', timeKey: '' })

defineSlots<{ actions(props: { row: Row; $index: number }): unknown }>()
const primaryFields = computed(() => props.fields.slice(0, 4))
const extraFields = computed(() => props.fields.slice(4))

// The column names are supplied by each page; keep their values unknown until
// a display or state branch explicitly handles them, without casting the row.
function valueAt(row: Row, key: string): unknown { return Reflect.get(row, key) }
function recordKey(row: Row, index: number): PropertyKey {
  const value = valueAt(row, props.keyField)
  return typeof value === 'string' || typeof value === 'number' || typeof value === 'symbol' ? value : index
}
function valueOf(row: Row, field: RecordField): unknown {
  const value = valueAt(row, field.key)
  if (field.state) {
    if (value === 1 || value === '1') return field.state === 'fall' ? '紧急告警' : '异常'
    const handled = field.handled ? valueAt(row, field.handled) : undefined
    if (handled === 1 || handled === '1') return '已处理'
    if (value === 0 || value === '0') return field.state === 'fall' ? '未告警/已处理' : '正常'
    return '暂无状态'
  }
  return value == null || value === '' ? '暂未提供' : value
}
function stateClass(row: Row, field: RecordField): string {
  if (!field.state) return ''
  const value = valueAt(row, field.key)
  if (value === 1 || value === '1') return 'status-danger'
  const handled = field.handled ? valueAt(row, field.handled) : undefined
  if (handled === 1 || handled === '1') return 'status-handled'
  return value === 0 || value === '0' ? 'status-normal' : 'status-unknown'
}
</script>

<style scoped>
@media (max-width: 900px) {
  .admin-record-list { display: grid; gap: 20px; }
  .admin-record { min-width: 0; padding: 20px 0; border-top: 1px solid #dce4da; }
  .admin-record:first-child { padding-top: 4px; border-top: 0; }
  .admin-record header { margin-bottom: 20px; }
  .admin-record h2 { margin: 0; color: var(--care-brand-dark); font-size: 20px; font-weight: 500; line-height: 1.6; overflow-wrap: anywhere; }
  .admin-record header > span { display: block; margin-top: 6px; color: var(--care-text-muted); font-size: 14px; }
  .admin-record dl { display: grid; grid-template-columns: minmax(0, 100px) minmax(0, 1fr); gap: 14px 16px; margin: 0; line-height: 1.8; }
  .admin-record dt { color: var(--care-text-muted); font-size: 14px; }
  .admin-record dd { margin: 0; color: var(--care-text); font-size: 16px; overflow-wrap: anywhere; }
  .record-more { margin-top: 20px; }
  .record-more summary { min-height: 44px; padding: 8px 0; color: var(--care-primary); cursor: pointer; }
  .record-more dl { padding: 12px 0; }
  .record-actions { display: flex; gap: 10px; flex-wrap: wrap; padding-top: 22px; }
  .record-actions :deep(.el-button) { margin: 0; }
  .admin-record .status-danger { color: var(--care-danger); }
  .admin-record .status-normal { color: var(--care-primary); }
  .admin-record .status-handled { color: var(--care-warning-strong); }
  .admin-record .status-unknown { color: var(--care-text-muted); }
}
</style>
