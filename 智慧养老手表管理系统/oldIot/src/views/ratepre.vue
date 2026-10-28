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

    <el-dialog :title="alertDialog.title" v-model="alertDialog.visible" width="620px">
      <div v-if="alertDialog.loading" style="text-align:center;padding:20px;">正在读取 ...</div>
      <div v-else-if="alertDialog.hasAlert === 0" style="text-align:center;padding:20px;color:#606266;">
        <i class="el-icon-success" style="color:#52BAB5;font-size:32px;"></i>
        <div style="margin-top:8px;">{{ alertDialog.message }}</div>
      </div>
      <div v-else>
        <el-alert :title="'问题：' + alertDialog.reason" type="error" :closable="false" show-icon></el-alert>
        <table class="alert-table">
          <tbody>
            <tr><th>老人</th><td>{{ alertDialog.oldPersonName }}</td></tr>
            <tr><th>本次测量</th><td>心率 {{ alertDialog.hr }} 次/分　血压 {{ alertDialog.bp }} mmHg</td></tr>
            <tr><th>正常范围</th><td>心率 55-110 次/分；血压 90-140 mmHg</td></tr>
            <tr><th>发生时间</th><td>{{ alertDialog.occurTime }}</td></tr>
            <tr><th>判断依据</th><td>{{ alertDialog.detail }}</td></tr>
            <tr><th>怎么处理</th><td>{{ alertDialog.advice }}</td></tr>
          </tbody>
        </table>
        <div style="margin-top:14px;">
          <el-button v-if="alertDialog.rateBad === 1" type="primary" :loading="alertDialog.resolving"
                     @click="resolveAlert('rate')">标记心率已处理</el-button>
          <el-button v-if="alertDialog.preBad === 1" type="primary" :loading="alertDialog.resolving"
                     @click="resolveAlert('pre')">标记血压已处理</el-button>
          <el-button plain @click="$router.push({ path: '/device' })">去设备管理看该手表</el-button>
        </div>
        <div class="tip-small">
          处理后这条记录会从「异常」变成「已处理」，红点消失，历史记录与曲线仍然保留。
          如果老人基础心率/血压本身偏离默认范围，请到「基础管理-手表信息」调整阈值，避免反复报警。
        </div>
      </div>
      <template #footer><span  class="dialog-footer">
        <el-button @click="alertDialog.visible = false">关 闭</el-button>
      </span></template>
    </el-dialog>

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
    loadAlertCount () {
      var _this = this
      axios.post('api/v1/ratepre/alertCount', {}, {
        headers: { 'content-type': 'application/json' },
        withCredentials: true
      }).then(function (response) {
        _this.rateCount = response.data.rateTotal
        _this.preCount = response.data.preTotal
      }).catch(function (error) {
        console.log(error)
      })
    },
    // 点表格里的"查看原因"
    showAlertRow (row) {
      this.loadAlertDetail(null, row.name, row.oldId)
    },
    // 点顶部的"心率不合格/血压不合格"：找该类型最新一条未处理异常
    showAlertByKind (kind) {
      var _this = this
      var target = null
      this.oldsList.forEach(function (row) {
        var bad = kind === 'rate' ? row.rateState == 1 : row.preState == 1
        if (bad && !target) {
          target = row
        }
      })
      if (!target) {
        this.$message({ type: 'success', message: '当前没有未处理的' + (kind === 'rate' ? '心率' : '血压') + '异常' })
        return
      }
      this.loadAlertDetail(null, target.name, target.oldId, kind)
    },
    // 读取异常详情：优先用 recordId，其次按老人找最新未处理异常
    loadAlertDetail (recordId, name, oldId, kind) {
      var _this = this
      this.alertDialog = {
        visible: true, loading: true, resolving: false, hasAlert: 1,
        title: '异常原因与处理', recordId: recordId, oldPersonName: name || '',
        hr: '', bp: '', rateBad: 0, preBad: 0, reason: '', detail: '',
        occurTime: '', advice: '', message: ''
      }
      var params = recordId ? { recordId: recordId } : { oldId: oldId, kind: kind || '' }
      axios.post('api/v1/ratepre/alertDetail', params, {
        headers: { 'content-type': 'application/json' },
        withCredentials: true
      }).then(function (response) {
        var d = response.data
        _this.alertDialog.loading = false
        if (d.returnCode !== '1111') {
          _this.alertDialog.hasAlert = 0
          _this.alertDialog.message = d.message || '没有找到待处理的异常'
          return
        }
        _this.alertDialog.hasAlert = d.hasAlert === undefined ? 1 : d.hasAlert
        _this.alertDialog.message = d.message || ''
        _this.alertDialog.recordId = d.recordId === undefined ? null : d.recordId
        _this.alertDialog.oldPersonName = d.oldPersonName || _this.alertDialog.oldPersonName
        _this.alertDialog.hr = d.hr
        _this.alertDialog.bp = d.bp
        _this.alertDialog.rateBad = d.rateBad || 0
        _this.alertDialog.preBad = d.preBad || 0
        _this.alertDialog.reason = d.reason || ''
        _this.alertDialog.detail = d.detail || ''
        _this.alertDialog.occurTime = d.occurTime || ''
        _this.alertDialog.advice = d.advice || ''
      }).catch(function (error) {
        console.log(error)
        _this.alertDialog.loading = false
        _this.alertDialog.hasAlert = 0
        _this.alertDialog.message = '读取失败，请稍后重试'
      })
    },
    // 处置：消除红点
    resolveAlert (kind) {
      var _this = this
      this.alertDialog.resolving = true
      axios.post('api/v1/ratepre/resolveAlert',
        { recordId: this.alertDialog.recordId, kind: kind },
        {
          headers: { 'content-type': 'application/json' },
          withCredentials: true
        }).then(function (response) {
          _this.alertDialog.resolving = false
          if (response.data.returnCode === '1111') {
            _this.$message({ type: 'success', message: '已处理，红点已消除' })
            _this.alertDialog.visible = false
            _this.searchOlds()
            _this.getCount()
          } else {
            _this.$message({ type: 'error', message: response.data.message || '处理失败' })
          }
        }).catch(function (error) {
          console.log(error)
          _this.alertDialog.resolving = false
          _this.$message({ type: 'error', message: '处理失败' })
        })
    },
    // 表格里直接点"标记已处理"
    resolveRow (row) {
      var _this = this
      var tasks = []
      if (row.rateState == 1) {
        tasks.push(axios.post('api/v1/ratepre/resolveAlert', { recordId: row.recordId, kind: 'rate' }, _this.postConfig()))
      }
      if (row.preState == 1) {
        tasks.push(axios.post('api/v1/ratepre/resolveAlert', { recordId: row.recordId, kind: 'pre' }, _this.postConfig()))
      }
      Promise.all(tasks).then(function () {
        _this.$message({ type: 'success', message: '已处理，红点已消除' })
        _this.searchOlds()
        _this.getCount()
      }).catch(function (error) {
        console.log(error)
        _this.$message({ type: 'error', message: '处理失败' })
      })
    },
    postConfig () {
      return {
        headers: { 'content-type': 'application/json' },
        withCredentials: true
      }
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
