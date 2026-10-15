<template>
  <div class="roles">
    <div class="record-heading"><h2>最近上报记录</h2><el-button link :loading="isLoading" @click="searchposition">刷新记录 <i class="el-icon-refresh" aria-hidden="true"></i></el-button></div>

    <el-alert v-if="loadError" :title="loadError" type="error" :closable="false" show-icon></el-alert>
    <el-table class="desktop-records" v-loading="isLoading" :data="positionList"
    style="width: 100%" border>
    <template #empty><div  class="position-empty">{{ loadError ? '位置记录暂时无法加载' : '暂无位置记录' }}</div></template>

	  <el-table-column prop="name" label="长者" width="150" align="center"></el-table-column>
	  <el-table-column prop="status" label="安全状态" width="200" align="center">
		  <template #default="scope" >
		    <el-tag v-if="scope.row.status == 1" type="danger" size="small">摔倒告警</el-tag>
		    <el-tag v-else-if="scope.row.status == 0" type="success" size="small">未发现摔倒告警</el-tag>
        <span v-else class="unknown-state">状态未知</span>
		  </template>
	  </el-table-column>
	  <el-table-column label="位置坐标" width="220" align="center"><template #default="scope" >{{ scope.row.lng }}，{{ scope.row.lat }}</template></el-table-column>
	  <el-table-column prop="beTime" label="记录时间" width="190" align="center"></el-table-column>
      <el-table-column label="查看" width="170" align="center">
      	<template #default="scope" >

        <el-button v-if="mapAvailable" size="small" plain type="success" @click="showLocation(scope.row)">
          地图查看
        </el-button><span v-else class="unknown-state">地图暂不可用</span>
      	</template>
      </el-table-column>
    </el-table>
    <section v-if="positionList.length" class="family-record-cards" aria-label="位置和安全记录">
      <article v-for="(item, index) in positionList" :key="index" class="family-record-card">
        <h2>{{ item.name || '位置记录' }}</h2>
        <dl>
          <dt>安全状态</dt>
          <dd :class="item.status == 1 ? 'status-danger' : item.status == 0 ? 'status-normal' : ''">{{ item.status == 1 ? '摔倒告警，请及时确认' : item.status == 0 ? '未发现摔倒告警' : '状态未知' }}</dd>
          <dt>记录时间</dt><dd>{{ item.beTime || '暂无时间信息' }}</dd>
          <dt>位置坐标</dt><dd>{{ item.lng }}, {{ item.lat }}</dd>
        </dl>
        <el-button v-if="mapAvailable" type="primary" @click="showLocation(item)">在地图中查看位置</el-button>
        <p v-else class="map-card-note">地图暂不可用；上方仍可查看位置坐标。</p>
      </article>
    </section>
    <div v-else-if="!isLoading" class="family-record-empty">{{ loadError || '暂时没有位置和安全记录。' }}</div>

	<section class="map-panel">
    <div class="map-heading"><div><h2>位置地图</h2><p>地图可用时，选择记录中的“地图查看”定位对应坐标。</p></div></div>
    <div v-if="mapAvailable" id="allmap"></div>
    <div v-else class="map-unavailable"><i class="el-icon-location-outline"></i><strong>地图服务尚未配置</strong><span>上方仍显示手表最近上传的时间和位置坐标；如需地图，请联系平台管理员配置地图服务。</span></div>
	</section>
  </div>

</template>

<script>

import axios from 'axios'

export default {
  data () {
    return {
      oldId: '',

      positionList: [],
      isLoading: false,
      loadError: '',
      mapAvailable: false,
      mapInstance: null,

			lng: '',
			lat: ''
    }
  },
  created () {

		this.oldId=localStorage.getItem('oldIdfamily')
    this.searchposition()

  },

  mounted(){
	  this.ready()

  },
  methods: {

    searchposition () {
      var _this = this
      var oldId = _this.oldId
      _this.isLoading = true
      _this.loadError = ''

      axios.post('restful/position/selectFamily',
        {
          "oldId":oldId
        },
        {
          headers: {
            'content-type': 'application/json'
          },
          withCredentials: true
        }).then(function (response) {

        _this.positionList = response.data && response.data.positionDtos ? response.data.positionDtos : []
      })
        .catch(function (error) {

          _this.positionList = []
          _this.loadError = '位置记录暂时无法加载，请检查服务连接后重试。'
        }).then(function () {
          _this.isLoading = false
        })
    },

	ready(){
		this.mapAvailable = typeof window.BMap !== 'undefined'
	},
	theLocation(){
		var BMap = window.BMap
		var longitude = Number(this.lng)
		var latitude = Number(this.lat)
		if (!this.mapAvailable || !BMap || !isFinite(longitude) || !isFinite(latitude) || longitude === 0 || latitude === 0) {
			this.$message({ type: 'warning', message: '当前记录没有可用坐标或地图服务未就绪' })
			return
		}
		var map = this.mapInstance
		if (!map) {
			map = new BMap.Map('allmap')
			this.mapInstance = map
			map.enableScrollWheelZoom(true)
		}
		map.centerAndZoom(new BMap.Point(longitude, latitude), 16)
		map.clearOverlays()
		var new_point = new BMap.Point(longitude, latitude)
		var marker = new BMap.Marker(new_point)
		map.addOverlay(marker)
		map.panTo(new_point)
	},

    showLocation (curRole) {

			var _this=this

      _this.lng=curRole.lng
			_this.lat=curRole.lat

			this.theLocation()
    },

    }
}

</script>

<style lang="less" scoped>
  .map-panel { margin-top: 24px; padding: 24px; background: #fff; border: 1px solid #e7ecef; border-radius: 12px; }
  .map-heading { margin-bottom: 16px; }
  .map-heading h2 { margin: 0; font-size: 18px; color: #243746; }
  .map-heading p { margin: 8px 0 0; color: #53645c; font-size: 16px; line-height: 1.7; }
  .map-unavailable { min-height: 200px; display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 10px; color: #53645c; text-align: center; font-size: 16px; line-height: 1.7; }
  .map-unavailable i { font-size: 28px; color: #91a4ad; }
  .map-unavailable strong { color: #344955; }
  .map-unavailable span { max-width: 520px; line-height: 1.6; }
  .position-empty, .unknown-state { color: #53645c; font-size: 16px; }

  #allmap { height: 420px; width: 100%; border-radius: 8px; }

</style>
