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

        <el-button size="small" plain type="success" @click="showLocation(scope.row)">
          地图查看
        </el-button>
      	</template>
      </el-table-column>
    </el-table>
    <section v-if="positionList.length" class="family-record-cards" aria-label="位置和安全记录">
      <article v-for="item in positionList" :key="item.oldId" class="family-record-card">
        <h2>{{ item.name || '位置记录' }}</h2>
        <dl>
          <dt>安全状态</dt>
          <dd :class="item.status == 1 ? 'status-danger' : item.status == 0 ? 'status-normal' : ''">{{ item.status == 1 ? '摔倒告警，请及时确认' : item.status == 0 ? '未发现摔倒告警' : '状态未知' }}</dd>
          <dt>记录时间</dt><dd>{{ item.beTime || '暂无时间信息' }}</dd>
          <dt>位置坐标</dt><dd>{{ item.lng }}, {{ item.lat }}</dd>
        </dl>
        <el-button type="primary" @click="showLocation(item)">在地图中查看位置</el-button>
      </article>
    </section>
    <div v-else-if="!isLoading" class="family-record-empty">{{ loadError || '暂时没有位置和安全记录。' }}</div>
    <position-map :selection="selectedLocation" />
  </div>

</template>

<script>

import axios from 'axios'
import PositionMap from '@/components/PositionMap.vue'

export default {
  components: { PositionMap },
  data () {
    return {
      oldId: '',

      positionList: [],
      isLoading: false,
      loadError: '',
      selectedLocation: null,
    }
  },
  created () {

		this.oldId=localStorage.getItem('oldIdfamily')
    this.searchposition()

  },
  methods: {

    searchposition () {
      var _this = this
      var oldId = _this.oldId
      _this.isLoading = true
      _this.loadError = ''

      axios.post('api/v1/position/selectFamily',
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
    showLocation (curRole) {
      this.selectedLocation = { ...curRole }
    },

    }
}

</script>

<style scoped>
  .position-empty, .unknown-state { color: #53645c; font-size: 16px; }

</style>
