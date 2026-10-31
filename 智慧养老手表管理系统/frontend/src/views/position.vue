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
    <el-table class="adaptive-record-table" v-loading="isLoading" :data="positionList"
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
	  <el-table-column prop="beTime" label="时间" width="auto" align="center"></el-table-column>
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
    <record-cards :records="positionList" :fields='[{"key":"sex","label":"性别"},{"key":"status","label":"告警状态","state":"fall"},{"key":"lng","label":"经度"},{"key":"lat","label":"纬度"}]' label="位置与告警记录" title-key="name" key-field="oldId" time-key="beTime">
      <template #actions="scope"  ><el-button v-if="scope.row.status==1" size="small" plain type="danger" icon="el-icon-warning"
                       @click="showFallReason(scope.row)">告警记录与处理</el-button>
<el-button v-if="scope.row.status==1" size="small" plain type="primary" icon="el-icon-check"
      		           @click="handleFall(scope.row)">标记已处理</el-button>
<el-button size="small" plain type="success" icon="el-icon-check" @click="showRoleDialog(scope.row)">
      			详细情况
      		</el-button></template>
    </record-cards>
    
    <div style="margin-top: 10px;text-align: center">
      <el-pagination
        @size-change="handleSizeChange"
        @current-change="changePage"
        v-model:current-page="currentPage"
        :page-size="pageSize"
        layout="total, sizes, prev, pager, next, jumper"
        :total="total">
      </el-pagination>
    </div>
    <position-map :selection="selectedLocation" />
  </div>
	
</template>

<script>
import axios from 'axios'
import PositionMap from '@/components/PositionMap.vue'
import RecordCards from '@/components/RecordCards.vue'

export default {
  components: { RecordCards, PositionMap },
  data () {
    return {
      searchText: '',

      total: 0,
      pageSize: 10,
      currentPage: 1,
			positionList: [],
			isLoading: false,
			loadError: '',
			selectedLocation: null,
			num:''
    }
  },
  created () {

    this.searchposition()

  },
  methods: {
    searchposition (keepPage = false) {
      if (keepPage !== true) this.currentPage = 1
      var _this = this
      _this.isLoading = true
      _this.loadError = ''
      var name = _this.searchText

      axios.post('api/v1/position/select',
        {
          page: _this.currentPage, pageSize: _this.pageSize,
          'name': name === '' ? '' : _this.searchText
        },
        {
          headers: {
            'content-type': 'application/json'
          },
          withCredentials: true
        }).then(function (response) {

				_this.positionList = response.data && response.data.positionDtos ? response.data.positionDtos : []
				var count=0;
        var len=_this.positionList.length
				for(var i=0;i<len;i++){
					if(_this.positionList[i].status=='1'){

						count++;
					}
				}

				_this.num=count
        _this.total = response.data.total ?? _this.positionList.length
      })
        .catch(function (error) {
          console.log(error)
          _this.positionList = []
          _this.total = 0
          _this.num = ''
          _this.loadError = '请检查服务连接后重试'
        }).then(function () {
          _this.isLoading = false
        })
    },
    // 告警记录与处理办法（红点点进来就是这个）
    showFallReason (row) {
      var alarmPosition = row.alarmLng != null && row.alarmLat != null && row.alarmLng !== '' && row.alarmLat !== ''
        ? '经度 ' + row.alarmLng + '，纬度 ' + row.alarmLat
        : '未提供告警坐标（最新定位仅供参考）'
      this.$alert(
        '老人：' + row.name + '\n'
        + '告警时间：' + (row.alarmTime || '未提供告警时间') + '\n'
        + '告警附带坐标：' + alarmPosition + '\n\n'
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
        axios.post('api/v1/device/handleAlarm', {'oldId': row.oldId},
          {
            headers: {'content-type': 'application/json'},
            withCredentials: true
          }).then(function (response) {
          if (response.data.returnCode === '1111') {
            _this.$message({type: 'success', message: '已处理，告警红点已消除'})
          } else {
            _this.$message({type: 'error', message: response.data.message || '处理失败'})
          }
          _this.searchposition()
        }).catch(function (error) {
          console.log(error)
          _this.$message({type: 'error', message: '处理失败'})
        })
      }).catch(function () {})
    },
    handleSizeChange (val) {

      this.pageSize = val
      this.currentPage = 1
      this.searchposition(true)
    },

    changePage (val) {

      this.currentPage = val
      this.searchposition(true)
    },
    showRoleDialog (curRole) {
      this.selectedLocation = { ...curRole }
    }

  }
}

</script>

<style scoped>
  .position-empty, .unknown-state { color: #71808c; }

</style>
