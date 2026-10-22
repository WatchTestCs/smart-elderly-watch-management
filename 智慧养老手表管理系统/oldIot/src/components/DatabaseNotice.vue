<template>
  <p v-if="authState.databaseReady === false && authState.configError" class="database-notice database-notice--error" role="alert">服务端配置有误，依赖数据库的功能不可用，自动重连不会恢复：{{ authState.configError }}</p>
  <p v-else-if="authState.databaseReady === false" class="database-notice" role="status">数据服务暂未连接；依赖数据库的功能暂不可用，正在自动连接。</p>
  <slot v-else />
</template>

<script setup>
import { authState } from '../auth'
</script>

<style scoped>
.database-notice { margin: 12px 0; padding: 10px 14px; border-left: 2px solid #9b7a40; background: #f8f5ec; color: #705628; font-size: 14px; line-height: 1.7; overflow-wrap: anywhere; }
.database-notice--error { border-left-color: #a33; background: #fbf2f2; color: #7a2222; }
</style>
