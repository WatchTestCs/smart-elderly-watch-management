<template>
  <div class="roles">
    
    <div class="sys-research">
      <form class="row-search" role="search" @submit.prevent="searchposition">
		  
        <div class="search-field">
          
          <label class="table-search-label" for="search-position">长者姓名</label>
          <el-input id="search-position"
            placeholder="按长者姓名查找"
            v-model="searchText"
            class="input-with-select">
            <template #append><el-button aria-label="查询长者姓名"
              
              icon="el-icon-search"
              native-type="submit"></el-button></template>
          </el-input>
        </div>
        
				<el-badge :value="this.num" class="item">
				  <el-button size="small">紧急告警</el-button>
				</el-badge>
				
      </form>
    </div>

    <el-alert v-if="loadError" :title="loadError" type="error" :closable="false" show-icon></el-alert>
    <el-table v-if="!mobileLayout" class="adaptive-record-table" v-loading="isLoading" :data="positionList"
    style="width: 100%" border>
    <template #empty><div  class="position-empty">{{ loadError ? '位置记录暂时无法加载' : '暂无位置记录' }}</div></template>

      <el-table-column  width="100" type="index" align="center"></el-table-column>
      <el-table-column prop="name" label="姓名" width="140" align="center"></el-table-column>
      <el-table-column prop="sex" label="性别" width="100" align="center"></el-table-column>
	  <el-table-column prop="status" label="告警状态" width="150" align="center">
		  <template #default="scope" >
		    <el-tag v-if="scope.row.status == 1" type="danger" size="small">紧急告警</el-tag>
		    <el-tag v-else-if="scope.row.status == 0" type="success" size="small">未告警/已处理</el-tag>
        <span v-else class="unknown-state">状态未知</span>
		  </template>
	  </el-table-column>
	  <el-table-column prop="lng" label="经度" width="160" align="center"></el-table-column>
	  <el-table-column prop="lat" label="纬度" width="160" align="center"></el-table-column>
      <el-table-column label="定位来源与采样" width="240" align="left"><template #default="scope"><span class="position-context">{{ describePositionContext(scope.row) }}</span></template></el-table-column>
	  <el-table-column prop="beTime" label="上报时间" width="auto" align="center"></el-table-column>
      <el-table-column label="操作" align="center">
      	<template #default="scope" >
      		<el-button v-if="scope.row.status==1" size="small" plain type="danger" icon="el-icon-warning"
                       @click="showFallReason(scope.row)">告警记录与处理</el-button>
      		<el-button v-if="scope.row.status==1" size="small" plain type="primary" icon="el-icon-check"
      		           @click="handleFall(scope.row)">标记已处理</el-button>
      		<el-button size="small" plain type="success" icon="el-icon-check" @click="showRoleDialog(scope.row)">
      			详细情况
      		</el-button>
      	</template>
      </el-table-column>
    </el-table>
    <record-cards v-if="mobileLayout" :records="positionList" :fields='[{"key":"sex","label":"性别"},{"key":"status","label":"告警状态","state":"fall"},{"key":"lng","label":"经度"},{"key":"lat","label":"纬度"}]' label="位置与告警记录" title-key="name" key-field="oldId" time-key="beTime">
      <template #actions="scope"  ><span class="position-context">{{ describePositionContext(scope.row) }}</span><el-button v-if="scope.row.status==1" size="small" plain type="danger" icon="el-icon-warning"
                       @click="showFallReason(scope.row)">告警记录与处理</el-button>
<el-button v-if="scope.row.status==1" size="small" plain type="primary" icon="el-icon-check"
      		           @click="handleFall(scope.row)">标记已处理</el-button>
<el-button size="small" plain type="success" icon="el-icon-check" @click="showRoleDialog(scope.row)">
      			详细情况
      		</el-button></template>
    </record-cards>
    
    <div style="margin-top: 10px;text-align: center">
      <pager-bar
        @size-change="handleSizeChange"
        @current-change="changePage"
        v-model:current-page="currentPage"
        :page-size="pageSize"
        :total="total" />
    </div>
    <position-map :selection="selectedLocation" />
  </div>
	
</template>

<script>
import { pagedRecords } from '@/composables/pagedRecords'
import { api as requestClient } from '../api/client'
import { useCollectionLayout } from '@/composables/useCollectionLayout'

import PositionMap from '@/components/PositionMap.vue'
import RecordCards from '@/components/RecordCards.vue'
import { describePositionContext } from '../utils/positionContext'

export default {
  computed: {
    num () { return this.positionList.filter(row => row.status === 1 || row.status === "1").length },
    alertRevision () { return this.$store.state.alertRevision }
  },
  watch: {
    alertRevision () { this.searchposition(true) },
    positionList (rows) {
      const id = this.$store.state.selectedResidentId
      const selected = id == null ? null : rows.find(row => String(row.oldId) === id)
      if (selected) this.selectedLocation = { ...selected }
    }
  },
  mixins: [pagedRecords({ path: "api/v1/position/select", responseKey: "positionDtos", recordsKey: "positionList", searchKey: "name", load: "searchposition", failure: "位置数据暂时无法加载，请检查服务连接后重试。" })],
  setup () { return { mobileLayout: useCollectionLayout(900) } },
  components: { RecordCards, PositionMap },
  data () {
    return {
      searchText: '',

			selectedLocation: null,
    }
  },
  created () {

    this.searchposition()

  },
  methods: {
    describePositionContext,
    // 告警记录与处理办法（红点点进来就是这个）
    showFallReason (row) {
      var alarmPosition = row.alarmLng != null && row.alarmLat != null && row.alarmLng !== '' && row.alarmLat !== ''
        ? '经度 ' + row.alarmLng + '，纬度 ' + row.alarmLat
        : '未提供告警坐标（最新定位仅供参考）'
      this.$alert(
        '老人：' + row.name + '\n'
        + '告警时间：' + (row.alarmTime || '未提供告警时间') + '\n'
        + '告警附带坐标：' + alarmPosition + '\n'
        + '事件类型：' + ({ sos: 'SOS 求助', fall: '设备上报摔倒', unknown: '未知' }[row.alarmKind] || '未知') + '\n'
        + describePositionContext(row, true) + '\n\n'
        + '怎么处理：\n'
        + '① 立即电话联系老人或家属确认情况；\n'
        + '② 结合告警附带坐标核实位置，必要时请求现场协助；\n'
        + '③ 确认处理后点「标记已处理」，右上角告警红点数量会减少。',
        '紧急告警 · 原因与处理',
        { confirmButtonText: '知道了', callback: function () {} }
      )
    },
    // 处置：消除告警红点
    handleFall (row) {
      var _this = this
      this.$confirm('确认该紧急告警已经处理（联系过老人/家属）？处理后会从红点统计里移除。', '提示', {
        confirmButtonText: '确定已处理',
        cancelButtonText: '取消',
        type: 'warning'
      }).then(function () {
        requestClient.post('api/v1/locations/alerts/resolve-latest', {'oldId': row.oldId}).then(function (response) {
          if (response.data.returnCode === '1111') {
            _this.$message({type: 'success', message: '已处理，告警红点已消除'})
            _this.$store.notifyAlertsChanged()
          } else {
            _this.$message({type: 'error', message: response.data.message || '处理失败'})
          }
        }).catch(function (error) {
          _this.$message({type: 'error', message: '处理失败'})
        })
      }).catch(function () {})
    },

    showRoleDialog (curRole) {
      this.selectedLocation = { ...curRole }
      this.$store.selectResident(curRole.oldId)
    }

  }
}

</script>

<style scoped>
  .position-context { display: block; width: 100%; line-height: 1.6; font-size: 12px; }
  .position-empty, .unknown-state { color: #71808c; }

</style>
