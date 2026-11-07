<template>
  <section v-if="failed" class="view-error" role="alert">
    <h2>当前页面暂时无法显示</h2>
    <p>可以重试，或通过菜单打开其他页面。</p>
    <button type="button" @click="retry">重新加载当前页面</button>
  </section>
  <slot v-else :key="attempt" />
</template>

<script setup lang="ts">
import { onErrorCaptured, ref } from 'vue'

const failed = ref(false)
const attempt = ref(0)
onErrorCaptured(() => {
  failed.value = true
  return false
})
function retry() {
  attempt.value++
  failed.value = false
}
</script>

<style scoped>
.view-error { padding: 24px; color: var(--care-text); }
.view-error button { padding: 12px 18px; cursor: pointer; }
</style>
