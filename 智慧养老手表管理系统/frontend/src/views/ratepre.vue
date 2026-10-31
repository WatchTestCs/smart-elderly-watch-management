<template>
  <div class="roles">

    <div class="sys-research">
      <form class="row-search" role="search" @submit.prevent="searchOlds">
        <div class="search-field">
          
          <label class="table-search-label" for="search-ratepre">长者姓名</label>
          <el-input id="search-ratepre"
            placeholder="按长者姓名查找"
            v-model="searchText"
            class="input-with-select">
            <template #append><el-button aria-label="查询长者姓名"
              
              icon="el-icon-search"
              native-type="submit"></el-button></template>
          </el-input>
        </div>

		<el-badge :value="this.rateCount" :hidden="this.rateCount === 0" class="item">
		  <el-button size="small" :type="Number(rateCount) > 0 ? 'danger' : 'default'" plain @click="showAlertByKind('rate')">心率异常</el-button>
		</el-badge>
		<el-badge :value="this.preCount" :hidden="this.preCount === 0" class="item">
			<el-button size="small" :type="Number(preCount) > 0 ? 'danger' : 'default'" plain @click="showAlertByKind('pre')">血压异常</el-button>
		</el-badge>
		<el-button size="small" plain type="primary" @click="openWorkbench">只看待处理异常</el-button>
      </form>
    </div>

    <el-table class="adaptive-record-table" :data="oldsList"
    style="width: 100%" border>
    <template #empty><div  class="care-empty-state"><strong>暂无健康记录</strong><span>最新上报的心率与血压数据会显示在这里。</span></div></template>

      <el-table-column type="index" width="150" align="center"></el-table-column>
      <el-table-column prop="name" label="姓名" width="180" align="center"></el-table-column>
			
      <el-table-column prop="sex" label="性别" width="150" align="center"></el-table-column>
			<el-table-column prop="preState" label="血压状态" width="150" align="center">
				<template #default="scope" >
                  <span v-if="scope.row.preState==1" class="status-danger">异常</span>
                  <span v-else-if="scope.row.preHandled==1" class="status-handled">已处理</span>
                  <span v-else-if="scope.row.preState === 0 || scope.row.preState === '0'" class="status-normal">正常</span><span v-else class="status-unknown">暂无状态</span>
				</template>
			</el-table-column>
			<el-table-column prop="rateState" label="心率状态" width="150" align="center">
				<template #default="scope" >
					<span v-if="scope.row.rateState==1" class="status-danger">异常</span>
					<span v-else-if="scope.row.rateHandled==1" class="status-handled">已处理</span>
					<span v-else-if="scope.row.rateState === 0 || scope.row.rateState === '0'" class="status-normal">正常</span><span v-else class="status-unknown">暂无状态</span>
				</template>
			</el-table-column>
			<el-table-column prop="createTime" label="时间" width="170" align="center"></el-table-column>
      <el-table-column label="操作" align="center">
        <template #default="scope" >
          <el-button v-if="scope.row.rateState==1 || scope.row.preState==1"
                     size="small" plain type="danger" icon="el-icon-warning"
                     @click="showAlertRow(scope.row)">查看原因</el-button>
          <el-button v-if="scope.row.rateState==1 || scope.row.preState==1"
                     size="small" plain type="primary" icon="el-icon-check"
                     @click="resolveRow(scope.row)">标记已处理</el-button>
          <el-button size="small" plain type="success" icon="el-icon-check" aria-label="查看健康详情" @click="showRoleDialog(scope.row)">
            查看健康详情
          </el-button>
        </template>
      </el-table-column>
    </el-table>
    <record-cards :records="oldsList" :fields='[{"key":"preState","label":"血压状态","state":"health","handled":"preHandled"},{"key":"rateState","label":"心率状态","state":"health","handled":"rateHandled"},{"key":"oldId","label":"长者编号"},{"key":"sex","label":"性别"}]' label="心率与血压记录" title-key="name" key-field="oldId" time-key="createTime">
      <template #actions="scope"  ><el-button v-if="scope.row.rateState==1 || scope.row.preState==1"
                     size="small" plain type="danger" icon="el-icon-warning"
                     @click="showAlertRow(scope.row)">查看原因</el-button>
<el-button v-if="scope.row.rateState==1 || scope.row.preState==1"
                     size="small" plain type="primary" icon="el-icon-check"
                     @click="resolveRow(scope.row)">标记已处理</el-button>
<el-button size="small" plain type="success" icon="el-icon-check" aria-label="查看健康详情" @click="showRoleDialog(scope.row)">
            查看健康详情
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

    <HealthAlertDialog :dialog="alertDialog" @resolve="resolveAlert" />

    <el-dialog title="健康记录详情" v-model="isShowRoleDialog" @opened="resizeCharts" @closed="disposeCharts">
      <el-alert v-if="chartError" :title="chartError" type="warning" :closable="false" show-icon></el-alert>
      <div v-loading="chartLoading" class="record-chart-content"><div id="myChart" ref="mychart" class="record-chart"></div><div id="myChartTwo" ref="mychartTwo" class="record-chart"></div></div>
      <template #footer><div  class="dialog-footer"><el-button type="primary" @click="isShowRoleDialog = false">关闭详情</el-button></div></template>
    </el-dialog>
  </div>

</template>

<script>
import axios from 'axios'
import RecordCards from '@/components/RecordCards.vue'
import HealthAlertDialog from '@/components/records/HealthAlertDialog.vue'
import { healthAlertMethods } from './ratepre.alert-methods'

import { recordCharts } from '@/charts'

export default {
  components: { RecordCards, HealthAlertDialog },
  mixins: [recordCharts],
  data () {
    return {
      searchText: '',
      chartLoading: false, chartError: '',

      total: 0,
      pageSize: 10,
      currentPage: 1,
      oldsList: [],
      isShowRoleDialog: false,

      hrs: [],
      bps: [],
      createTimes: [],
      time: '',
      name: '',
      rateCount: '',
      preCount: '',
      // 异常（红点）处置
      alertDialog: {
        visible: false,
        loading: false,
        resolving: false,
        hasAlert: 1,
        title: '异常原因与处理',
        recordId: null,
        oldPersonName: '',
        hr: '',
        bp: '',
        rateBad: 0,
        preBad: 0,
        reason: '',
        detail: '',
        occurTime: '',
        advice: '',
        message: ''
      }
    }
  },
  created () {

    this.searchOlds()
    this.getCount()

  },

  beforeUnmount () { this.isShowRoleDialog = false },
  methods: {
    ...healthAlertMethods,
    searchOlds (keepPage = false) {
      if (keepPage !== true) this.currentPage = 1

      var _this = this
      var name = _this.searchText

      axios.post('api/v1/ratepre/selectRatepre',
        {
          page: _this.currentPage, pageSize: _this.pageSize,
          'name': name === '' ? '' : _this.searchText,
          'timeFirst':'',
          'timeLast': ''
        },
        {
          headers: {
            'content-type': 'application/json'
          },
          withCredentials: true
        }).then(function (response) {
        _this.oldsList = response.data.ratepreDtoList
        _this.total = response.data.total ?? response.data.ratepreDtoList.length
      })
        .catch(function (error) {
          console.log(error)
        })
    },
    getCount(){
      var _this=this
      axios.post('api/v1/ratepre/selectRatepreCount',
        {

        },
        {
          headers: {
            'content-type': 'application/json'
          },
          withCredentials: true
        }).then(function (response) {
        _this.rateCount=response.data.rates.length
        _this.preCount=response.data.pres.length
      })
        .catch(function (error) {
          console.log(error)
        })
      // 红点数字用"未处理的异常条数"，而不是历史异常设备数
      this.loadAlertCount()
    },
    // 只看待处理异常（把有异常的老人筛出来）
    openWorkbench () {
      var list = []
      this.oldsList.forEach(function (row) {
        if (row.rateState == 1 || row.preState == 1) {
          list.push(row)
        }
      })
      if (list.length === 0) {
        this.$message({ type: 'success', message: '当前没有待处理的异常' })
        return
      }
      this.oldsList = list
      this.total = list.length
      this.$message({ type: 'info', message: '已筛选出 ' + list.length + ' 条待处理异常，点「查看原因」了解处理办法' })
    },

    handleSizeChange (val) {

      this.pageSize = val
      this.currentPage = 1
      this.searchOlds(true)
    },

    changePage (val) {

      this.currentPage = val
      this.searchOlds(true)
    },
    showRoleDialog (curOld) {
      this.isShowRoleDialog = true
      this.chartLoading = true
      this.chartError = ''
      this.name = curOld.name || ''
      return axios.post('api/v1/ratepre/selectRatepreDetail', { oldId: curOld.oldId }, { withCredentials: true }).then(response => {
        if (!this.isShowRoleDialog) return
        this.bps = Array.isArray(response.data.bps) ? response.data.bps : []; this.hrs = Array.isArray(response.data.hrs) ? response.data.hrs : []; this.createTimes = Array.isArray(response.data.createTimes) ? response.data.createTimes : []; this.time = response.data.time;
        if (!this.hrs.length && !this.bps.length) { this.chartError = '暂无可展示的健康详情。'; return }
        this.drawLine(); this.drawLineTwo()
      }).catch(() => { this.chartError = '详情暂时无法加载，请关闭后重试。' }).then(() => { this.chartLoading = false })
    },

  drawLine () {
    this.$nextTick(() => {
      if (!this.isShowRoleDialog) return
      this.renderRecordChart(this.$refs.mychart, { title: '心率记录', subtitle: this.name + ' · ' + (this.time || ''), labels: this.createTimes, values: this.hrs, unit: '次/分' })
    })
  },
  drawLineTwo () {
    this.$nextTick(() => {
      if (!this.isShowRoleDialog) return
      this.renderRecordChart(this.$refs.mychartTwo, { title: '血压记录', subtitle: this.name + ' · ' + (this.time || ''), labels: this.createTimes, values: this.bps, unit: 'mmHg' })
    })
  }
  }
}
</script>
