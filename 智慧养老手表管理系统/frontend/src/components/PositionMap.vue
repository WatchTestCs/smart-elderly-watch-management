<template>
  <section class="position-map" aria-label="位置地图">
    <div class="position-map-heading">
      <div><h2>位置地图</h2><p>{{ valid ? `${selection.name || '长者'} · ${selection.lng}，${selection.lat}` : '选择一条位置记录，在此查看。' }}</p></div>
      <el-button v-if="valid && !online" @click="enableTiles">加载在线地图</el-button>
      <el-button v-if="online" link @click="disableTiles">关闭在线地图</el-button>
    </div>
    <div ref="canvas" class="position-map-canvas" :class="{ 'has-location': valid }" role="region" aria-label="可缩放的位置地图"></div>
    <p v-if="valid" class="position-map-note">{{ describePositionContext(selection) }}</p>
    <p class="position-map-note" role="status">{{ message }}</p>
  </section>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import L from 'leaflet'
import 'leaflet/dist/leaflet.css'
import { describePositionContext } from '../utils/positionContext'

const props = defineProps({ selection: { type: Object, default: null } })
const canvas = ref(null)
const online = ref(false)
const failed = ref(false)
const valid = computed(() => {
  const value = props.selection
  return value && value.lng != null && value.lat != null && String(value.lng).trim() !== '' && String(value.lat).trim() !== '' &&
    Number.isFinite(Number(value.lng)) && Number.isFinite(Number(value.lat)) && Math.abs(Number(value.lng)) <= 180 && Math.abs(Number(value.lat)) <= 90
})
const message = computed(() => !valid.value ? '暂无可用坐标。' : failed.value ? '在线地图加载失败；位置标记与原始坐标仍可查看。' :
  online.value ? '坐标以设备上报为准，地图仅作参考。' : '所选上报坐标已标记。在线底图按需加载，需要网络连接。')
let map, marker, tiles, observer
function updateLocation () {
  if (!map) return
  if (marker) { marker.remove(); marker = null }
  if (!valid.value) { disableTiles(); return }
  const point = [Number(props.selection.lat), Number(props.selection.lng)]
  map.setView(point, 15, { animate: false })
  marker = L.circleMarker(point, { radius: 9, color: '#fff', weight: 3, fillColor: '#215c4a', fillOpacity: 1 }).addTo(map)
  const label = document.createElement('span')
  label.textContent = props.selection.name || '最近上报位置'
  marker.bindTooltip(label, { direction: 'top', permanent: true, offset: [0, -12] })
}
function enableTiles () {
  if (!map || !valid.value || tiles) return
  online.value = true
  failed.value = false
  tiles = L.tileLayer('https://tile.openstreetmap.org/{z}/{x}/{y}.png', {
    maxZoom: 19, minZoom: 2, updateWhenIdle: true, keepBuffer: 1,
    attribution: '© OpenStreetMap contributors'
  })
  tiles.on('tileerror', () => { failed.value = true })
  tiles.addTo(map)
}
function disableTiles () {
  if (tiles) { tiles.off(); tiles.remove(); tiles = null }
  online.value = false
  failed.value = false
}
onMounted(() => {
  map = L.map(canvas.value, { scrollWheelZoom: false, attributionControl: true, zoomControl: false }).setView([30, 110], 5)
  L.control.zoom({ zoomInTitle: '放大地图', zoomOutTitle: '缩小地图' }).addTo(map)
  // Links are disabled in the desktop shell; retain readable attribution locally.
  map.attributionControl.setPrefix(false)
  observer = new ResizeObserver(() => map?.invalidateSize({ animate: false }))
  observer.observe(canvas.value)
  updateLocation()
})
watch(() => props.selection, updateLocation)
onBeforeUnmount(() => { observer?.disconnect(); disableTiles(); map?.remove(); map = null })
</script>

<style scoped>
.position-map { margin-top: 24px; padding: clamp(16px, 3vw, 28px); border-top: 1px solid #dce3d8; }
.position-map-heading { display: flex; flex-wrap: wrap; align-items: center; justify-content: space-between; gap: 16px; margin-bottom: 18px; }
.position-map-heading h2 { margin: 0; font-size: 22px; font-weight: 500; }
.position-map-heading p, .position-map-note { color: #53645c; line-height: 1.7; overflow-wrap: anywhere; }
.position-map-heading p { margin: 6px 0 0; }
.position-map-canvas { height: clamp(240px, 46vh, 440px); z-index: 0; border-radius: 16px; background-color: #eef1e9; background-image: linear-gradient(#dce3d833 1px, transparent 1px), linear-gradient(90deg, #dce3d833 1px, transparent 1px); background-size: 32px 32px; }
.position-map-canvas:not(.has-location) { opacity: .55; pointer-events: none; }
.position-map-note { margin: 12px 0 0; font-size: 15px; }
.position-map :deep(.leaflet-container) { font-family: inherit; }
.position-map :deep(.leaflet-control-zoom a) { width: 44px; height: 44px; line-height: 44px; }
.position-map :deep(.leaflet-control-attribution) { max-width: 100%; font-size: 12px; }
</style>
