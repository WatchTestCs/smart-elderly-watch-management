<template>
  <el-dialog v-model="visible" title="常规设置" width="min(520px, 94vw)" append-to-body>
    <section class="settings-group">
      <h3>显示与操作</h3>
      <div class="setting"><label for="setting-large-text">大字模式</label><el-switch id="setting-large-text" :model-value="largeText" @change="setLargeText" /></div>
      <div class="setting"><label for="setting-reduced-motion">减少界面动画</label><el-switch id="setting-reduced-motion" v-model="mapPreferences.reducedMotion" @change="saveMapPreferences" /></div>
    </section>
    <section class="settings-group">
      <h3>位置地图</h3>
      <label for="setting-map-provider">默认地图服务</label>
      <el-select id="setting-map-provider" v-model="mapPreferences.provider" @change="saveMapPreferences">
        <el-option value="amap" label="高德地图 · 内嵌，无需 Key" />
        <el-option value="tencent" label="腾讯地图 · 在浏览器打开" />
        <el-option value="google" label="Google 地图 · 官方免 Key 链接" />
        <el-option value="offline" label="仅显示原始坐标（不联网）" />
      </el-select>
      <label for="setting-coordinate-system">设备上报坐标系</label>
      <el-select id="setting-coordinate-system" v-model="mapPreferences.coordinateSystem" @change="saveMapPreferences">
        <el-option value="wgs84" label="WGS84 · 原始 GPS（默认）" />
        <el-option value="gcj02" label="GCJ-02 · 已转换的中国地图坐标" />
      </el-select>
      <template v-if="mapPreferences.provider === 'amap'">
        <label for="setting-map-style">高德地图类型</label>
        <el-select id="setting-map-style" v-model="mapPreferences.style" @change="saveMapPreferences">
          <el-option value="roadmap" label="标准地图" />
          <el-option value="satellite" label="卫星地图" />
        </el-select>
        <div class="setting"><label for="setting-map-auto">选中位置后自动加载地图</label><el-switch id="setting-map-auto" v-model="mapPreferences.loadOnlineAutomatically" @change="saveMapPreferences" /></div>
      </template>
      <p class="hint">默认按需加载。加载在线地图时，所选坐标会发送给地图服务商；不会发送老人姓名。腾讯地图与 Google 地图使用官方位置链接，经确认后由浏览器打开；Google 地图在中国大陆网络可能不可用。坐标系必须与设备真实上报格式一致，错误设置会产生位置偏移。需要自定义轨迹、多点图层和 Key 认证时，应通过独立 SDK 接口实现。</p>
    </section>
    <template #footer><el-button @click="resetMapPreferences">恢复地图及动画默认设置</el-button><el-button type="primary" @click="visible=false">完成</el-button></template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { mapPreferences, saveMapPreferences, resetMapPreferences } from '../utils/mapProviders'

const emit = defineEmits<{ textSizeChanged: [value: boolean] }>()
const visible = ref(false)
const largeText = ref(false)
function open() {
  largeText.value = document.documentElement.classList.contains('large-text')
  visible.value = true
}
function setLargeText(value: string | number | boolean) {
  largeText.value = Boolean(value)
  document.documentElement.classList.toggle('large-text', largeText.value)
  emit('textSizeChanged', largeText.value)
  try { localStorage.setItem('large_text', String(largeText.value)) } catch { /* Session-only preference. */ }
}
defineExpose({ open })
</script>

<style scoped>
.settings-group { display: grid; gap: 12px; padding: 8px 0 22px; }
.settings-group + .settings-group { border-top: 1px solid var(--care-border); padding-top: 20px; }
.settings-group h3 { margin: 0; font-size: 18px; }
.setting { display: flex; justify-content: space-between; align-items: center; gap: 16px; min-height: 42px; }
.hint { margin: 0; color: var(--care-text-muted); font-size: 13px; line-height: 1.7; }
</style>
