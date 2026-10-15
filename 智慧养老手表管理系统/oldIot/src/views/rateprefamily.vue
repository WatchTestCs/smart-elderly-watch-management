<template>
  <div class="roles">
    <div class="record-heading"><h2>最近上报记录</h2><el-button link :loading="isLoading" @click="searchOlds">刷新记录 <i class="el-icon-refresh" aria-hidden="true"></i></el-button></div>
    <el-alert v-if="loadError" :title="loadError" type="error" :closable="false" show-icon></el-alert>

    <el-table v-loading="isLoading" class="desktop-records" :data="oldsList"
    style="width: 100%" border>
    <template #empty><div  class="care-empty-state"><strong>暂无健康记录</strong><span>老人的最新心率与血压状态会显示在这里。</span></div></template>

			<el-table-column prop="name" label="长者" width="180" align="center"></el-table-column>
			<el-table-column prop="preState" label="血压状态" width="170" align="center">
				<template #default="scope" >
                  <span :class="stateClass(scope.row.preState)">{{ stateLabel(scope.row.preState) }}</span>
				</template>
			</el-table-column>
			<el-table-column prop="rateState" label="心率状态" width="170" align="center">
				<template #default="scope" >
					<span :class="stateClass(scope.row.rateState)">{{ stateLabel(scope.row.rateState) }}</span>
				</template>
			</el-table-column>
			<el-table-column prop="createTime" label="记录时间" width="210" align="center"></el-table-column>
      <el-table-column label="查看" width="190" align="center">
        <template #default="scope" >

          <el-button size="small" plain type="success" icon="el-icon-check" @click="showDetails(scope.row)">
            详细情况
          </el-button>
        </template>
      </el-table-column>
    </el-table>
    <section v-if="oldsList.length" class="family-record-cards" aria-label="心率与血压记录">
      <article v-for="(item, index) in oldsList" :key="index" class="family-record-card">
        <h2>{{ item.name || '健康记录' }}</h2>
        <dl>
          <dt>血压状态</dt><dd :class="stateClass(item.preState)">{{ stateLabel(item.preState) }}{{ item.preState == 1 ? '，请查看详情' : '' }}</dd>
          <dt>心率状态</dt><dd :class="stateClass(item.rateState)">{{ stateLabel(item.rateState) }}{{ item.rateState == 1 ? '，请查看详情' : '' }}</dd>
          <dt>记录时间</dt><dd>{{ item.createTime || '暂无时间信息' }}</dd>
        </dl>
        <el-button type="primary" @click="showDetails(item)">查看心率和血压详情</el-button>
      </article>
    </section>
    <div v-else class="family-record-empty">{{ isLoading ? '正在加载健康记录…' : loadError || '暂时没有心率或血压记录。' }}</div>

    <el-dialog :title="name ? name + '的健康详情' : '心率与血压详情'" width="92%" v-model="detailsVisible" @opened="resizeCharts" @closed="disposeCharts" >
      <el-alert v-if="chartError" :title="chartError" type="warning" :closable="false" show-icon></el-alert><el-button v-if="chartError" link @click="showDetails({ name: name })">重新加载详情</el-button><div v-loading="chartLoading" class="health-chart-grid">

	  <div v-show="!chartError" id="myChart" ref="mychart" class="rate-chart"></div>

		<div v-show="!chartError" id="myChartTwo" ref="mychartTwo" class="rate-chart"></div>

      </div>
      <template #footer><div  class="dialog-footer">
        <el-button type="primary" @click="detailsVisible = false">关闭详情</el-button>
      </div></template>
    </el-dialog>
  </div>

</template>

<script>

import axios from 'axios'
import { recordCharts } from '@/charts'

export default {
  mixins: [recordCharts],
  data () {
    return {

      oldsList: [],
      isLoading: false,
      chartLoading: false,
      chartError: '',
      loadError: '',
		oldId: '',

      detailsVisible: false,

			hrs: [],
			bps: [],
			createTimes: [],
			time: '',
			name: '',

			}
  },
  created () {

		this.oldId=localStorage.getItem('oldIdfamily')

		this.searchOlds()

  },

  beforeUnmount () { this.detailsVisible = false },
  methods: {
    stateLabel (value) {
      if (value === 1 || value === '1') return '异常'
      if (value === 0 || value === '0') return '正常'
      return '暂无状态'
    },
    stateClass (value) {
      if (value === 1 || value === '1') return 'status-danger'
      if (value === 0 || value === '0') return 'status-normal'
      return 'unknown-state'
    },

    searchOlds () {

      var _this = this
      var oldId = _this.oldId

      _this.isLoading = true
      _this.loadError = ''
      axios.post('restful/ratepre/selectRatepreFamily',
        {
					"oldId":oldId
				},
        {
          headers: {
            'content-type': 'application/json'
          },
          withCredentials: true
        }).then(function (response) {

        _this.oldsList = response.data && Array.isArray(response.data.ratepreDtoList) ? response.data.ratepreDtoList : []
      })
        .catch(function (error) {

          _this.loadError = '记录暂时无法加载，请点击刷新重试。'
          _this.oldsList = []
        }).then(function () { _this.isLoading = false })
    },

    showDetails (curOld) {
			var _this=this
			var oldId=_this.oldId

      _this.detailsVisible = true
      _this.chartLoading = true
      _this.chartError = ''
      _this.name = curOld.name || ''
			axios.post('restful/ratepre/selectRatepreDetail',{
          "oldId":oldId
        },
				{
					headers: {
						'content-type': 'application/json'
					},
					withCredentials: true
				}).then(function (response) {

				_this.bps = Array.isArray(response.data.bps) ? response.data.bps : []
				_this.hrs = Array.isArray(response.data.hrs) ? response.data.hrs : []
				_this.time=response.data.time
				_this.createTimes = Array.isArray(response.data.createTimes) ? response.data.createTimes : []
				if (!_this.hrs.length && !_this.bps.length) { _this.chartError = '暂时没有可展示的健康详情。'; return }
        if (!_this.detailsVisible) return
        _this.drawLine()
				_this.drawLineTwo()

			})
				.catch(function (error) {

          _this.chartError = '详情暂时无法加载，请重试。'
				}).then(function () { _this.chartLoading = false })

    },

	drawLine () {
    this.$nextTick(() => {
      if (!this.detailsVisible) return
      this.renderRecordChart(this.$refs.mychart, { title: '心率记录', subtitle: this.name + ' · ' + (this.time || ''), labels: this.createTimes, values: this.hrs, unit: '次/分' })
    })
  },
  drawLineTwo () {
    this.$nextTick(() => {
      if (!this.detailsVisible) return
      this.renderRecordChart(this.$refs.mychartTwo, { title: '血压记录', subtitle: this.name + ' · ' + (this.time || ''), labels: this.createTimes, values: this.bps, unit: 'mmHg' })
    })
  }
  }
}

</script>

<style scoped>
.health-chart-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 20px; }
.rate-chart { width: 100%; min-width: 0; height: 320px; border: 1px solid #e4eadd; border-radius: 12px; background: #fbfcf8; }
@media (max-width: 760px) { .health-chart-grid { grid-template-columns: 1fr; gap: 16px; } .rate-chart { height: 280px; } }
</style>
