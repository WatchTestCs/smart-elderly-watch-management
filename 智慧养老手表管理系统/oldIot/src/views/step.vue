<template>
  <div class="roles">
    
    <div class="sys-research">
      <form class="row-search" role="search" @submit.prevent="searchStep">
        <div class="search-field">
          
          <label class="table-search-label" for="search-step">长者姓名</label>
          <el-input id="search-step"
            placeholder="按长者姓名查找"
            v-model="searchText"
            class="input-with-select">
            <template #append><el-button aria-label="查询长者姓名"
              
              icon="el-icon-search"
              native-type="submit"></el-button></template>
          </el-input>
        </div>

		<span class="record-summary" role="status">睡眠异常 <strong>{{ stepCount === '' ? '—' : stepCount }}</strong></span>
      </form>
    </div>

    <el-table class="adaptive-record-table" :data="stepList"
    style="width: 100%" border>
    <template #empty><div  class="care-empty-state"><strong>暂无活动数据</strong><span>手表上报的步数与睡眠记录会显示在这里。</span></div></template>
    
      <el-table-column type="index" width="100" align="center"></el-table-column>
      <el-table-column prop="name" label="姓名" width="180" align="center"></el-table-column>
      <el-table-column prop="sex" label="性别" width="100" align="center"></el-table-column>
			<el-table-column prop="step" label="今日步数" width="150" align="center"></el-table-column>
			<el-table-column prop="sleep" label="睡眠时间" width="150" align="center"></el-table-column>
			<el-table-column prop="sleepSate" label="睡眠情况" width="150" align="center">
				<template #default="scope" >
				  <span v-if="scope.row.sleepSate==1" class="status-danger">异常</span>
				  <span v-else-if="scope.row.sleepSate === 0 || scope.row.sleepSate === '0'" class="status-normal">正常</span><span v-else class="status-unknown">暂无状态</span>
				</template>
			</el-table-column>
			<el-table-column prop="createTime" label="时间" width="180" align="center"></el-table-column>
      <el-table-column label="操作" align="center">
        <template #default="scope" >
          
          <el-button size="small" plain type="success" icon="el-icon-check" aria-label="查看睡眠详情" @click="showRoleDialog(scope.row)">
            查看睡眠详情
          </el-button>
        </template>
      </el-table-column>
    </el-table>
    <record-cards :records="stepList" :fields='[{"key":"sex","label":"性别"},{"key":"step","label":"今日步数"},{"key":"sleep","label":"睡眠时间"},{"key":"sleepSate","label":"睡眠情况","state":"health"}]' label="睡眠与步数记录" title-key="name" key-field="oldId" time-key="createTime">
      <template #actions="scope"  ><el-button size="small" plain type="success" icon="el-icon-check" aria-label="查看睡眠详情" @click="showRoleDialog(scope.row)">
            查看睡眠详情
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

    <el-dialog title="睡眠记录详情" v-model="isShowRoleDialog" @opened="resizeCharts" @closed="disposeCharts">
      <el-alert v-if="chartError" :title="chartError" type="warning" :closable="false" show-icon></el-alert>
      <div v-loading="chartLoading" class="record-chart-content"><div id="myChart" ref="mychart" class="record-chart"></div></div>
      <template #footer><div  class="dialog-footer"><el-button type="primary" @click="isShowRoleDialog = false">关闭详情</el-button></div></template>
    </el-dialog>
  </div>

</template>

<script>
import axios from 'axios'
import RecordCards from '@/components/RecordCards.vue'

import { recordCharts } from '@/charts'

export default {
  components: { RecordCards },
  mixins: [recordCharts],
  data () {
    return {
      searchText: '',
      chartLoading: false, chartError: '',

      total: 0,
      pageSize: 10,
      currentPage: 1,
      stepList: [],

		sleep: [],
		name: '',

		createTime: [],
      isShowRoleDialog: false,

			stepCount:''
    }
  },
  created () {

    this.searchStep()
		this.getCount()

  },

  beforeUnmount () { this.isShowRoleDialog = false },
  methods: {
    searchStep (keepPage = false) {
      if (keepPage !== true) this.currentPage = 1
      
      var _this = this
      var name = _this.searchText

      axios.post('api/v1/step/selectStep',
        {
          page: _this.currentPage, pageSize: _this.pageSize,
          'name': name === '' ? '' : _this.searchText,
          'timeFirst': '',
          'timeLast': ''
        },
        {
          headers: {
            'content-type': 'application/json'
          },
          withCredentials: true
        }).then(function (response) {
        _this.stepList = response.data.stepDtos
        _this.total = response.data.total ?? response.data.stepDtos.length
      })
        .catch(function (error) {
          console.log(error)
        })
    },
		getCount(){

			var _this=this
			axios.post('api/v1/step/selectStepCount',
				{
					
				},
				{
					headers: {
						'content-type': 'application/json'
					},
					withCredentials: true
				}).then(function (response) {
				_this.stepCount=response.data.deviceIdList.length
			})
				.catch(function (error) {
					console.log(error)
				})
		},

    handleSizeChange (val) {

      this.pageSize = val
      this.currentPage = 1
      this.searchStep(true)
    },

    changePage (val) {

      this.currentPage = val
      this.searchStep(true)
    },
    showRoleDialog (curOld) {
      this.isShowRoleDialog = true
      this.chartLoading = true
      this.chartError = ''
      this.name = curOld.name || ''
      return axios.post('api/v1/step/selectStepDetail', { oldId: curOld.oldId }, { withCredentials: true }).then(response => {
        if (!this.isShowRoleDialog) return
        this.sleep = Array.isArray(response.data.sleep) ? response.data.sleep : []; this.createTime = Array.isArray(response.data.createTime) ? response.data.createTime : [];
        if (!this.sleep.length) { this.chartError = '暂无可展示的睡眠详情。'; return }
        this.drawLine()
      }).catch(() => { this.chartError = '详情暂时无法加载，请关闭后重试。' }).then(() => { this.chartLoading = false })
    },

	drawLine () {
    this.$nextTick(() => {
      if (!this.isShowRoleDialog) return
      this.renderRecordChart(this.$refs.mychart, { title: '睡眠时长', subtitle: this.name, labels: this.createTime, values: this.sleep, unit: '小时', type: 'bar' })
    })
  }
  }
}

</script>
