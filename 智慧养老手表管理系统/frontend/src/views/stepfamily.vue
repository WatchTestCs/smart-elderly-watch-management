<template>
  <div class="roles">
    <div class="record-heading"><h2>最近上报记录</h2><el-button link :loading="isLoading" @click="searchStep">刷新记录 <i class="el-icon-refresh" aria-hidden="true"></i></el-button></div>
    <el-alert v-if="loadError" :title="loadError" type="error" :closable="false" show-icon></el-alert>

    <el-table v-loading="isLoading" class="desktop-records" :data="stepList"
    style="width: 100%" border>
    <template #empty><div  class="care-empty-state"><strong>暂无活动数据</strong><span>家人手表上报的步数与睡眠记录会显示在这里。</span></div></template>

			<el-table-column prop="name" label="长者" width="160" align="center"></el-table-column>
			<el-table-column prop="step" label="今日步数" width="145" align="center"></el-table-column>
			<el-table-column prop="sleep" label="睡眠时长" width="155" align="center"></el-table-column>
			<el-table-column prop="sleepSate" label="睡眠情况" width="165" align="center">
				<template #default="scope" >
				  <span :class="stateClass(scope.row.sleepSate)">{{ stateLabel(scope.row.sleepSate) }}</span>
				</template>
			</el-table-column>
			<el-table-column prop="createTime" label="记录时间" width="190" align="center"></el-table-column>
      <el-table-column label="查看" width="180" align="center">
        <template #default="scope" >

          <el-button size="small" plain type="success" icon="el-icon-check" @click="showDetails(scope.row)">
            详细情况
          </el-button>
        </template>
      </el-table-column>
    </el-table>
    <section v-if="stepList.length" class="family-record-cards" aria-label="活动和睡眠记录">
      <article v-for="item in stepList" :key="item.oldId" class="family-record-card">
        <h2>{{ item.name || '活动记录' }}</h2>
        <dl>
          <dt>今日步数</dt><dd>{{ item.step == null ? '暂无数据' : item.step + ' 步' }}</dd>
          <dt>睡眠时长</dt><dd>{{ item.sleep || '暂无数据' }}</dd>
          <dt>睡眠情况</dt><dd :class="stateClass(item.sleepSate)">{{ stateLabel(item.sleepSate) }}{{ item.sleepSate == 1 ? '，请查看详情' : '' }}</dd>
          <dt>记录时间</dt><dd>{{ item.createTime || '暂无时间信息' }}</dd>
        </dl>
        <el-button type="primary" @click="showDetails(item)">查看睡眠详情</el-button>
      </article>
    </section>
    <div v-else class="family-record-empty">{{ isLoading ? '正在加载活动记录…' : loadError || '暂时没有活动或睡眠记录。' }}</div>

    <el-dialog :title="name ? name + '的睡眠详情' : '睡眠详情'" width="92%" v-model="detailsVisible" @opened="resizeCharts" @closed="disposeCharts">
      <el-alert v-if="chartError" :title="chartError" type="warning" :closable="false" show-icon></el-alert><el-button v-if="chartError" link @click="showDetails({ name: name })">重新加载详情</el-button><div v-loading="chartLoading">

	  <div v-show="!chartError" id="myChart" ref="mychart" class="sleep-chart"></div>

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
      oldId: '',

      stepList: [],
      isLoading: false,
      chartLoading: false,
      chartError: '',
      loadError: '',
		sleep: [],
		name: '',

		createTime: [],

      detailsVisible: false,

      }
  },
  created () {

		this.oldId=localStorage.getItem('oldIdfamily')

    this.searchStep()

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

    searchStep () {
      var _this=this

      			var oldId=_this.oldId
      _this.isLoading = true
      _this.loadError = ''
      axios.post('api/v1/step/selectStepFamily',
        {
          "oldId":oldId
        },
        {
          headers: {
            'content-type': 'application/json'
          },
          withCredentials: true
        }).then(function (response) {

        _this.stepList = response.data && Array.isArray(response.data.stepDtos) ? response.data.stepDtos : []
      })
        .catch(function (error) {

          _this.loadError = '记录暂时无法加载，请点击刷新重试。'
          _this.stepList = []
        }).then(function () { _this.isLoading = false })
    },

    showDetails (curOld) {
      var _this=this
      var oldId=_this.oldId

      _this.detailsVisible = true
      _this.chartLoading = true
      _this.chartError = ''
      _this.name = curOld.name || ''
      			axios.post('api/v1/step/selectStepDetail',{
								"oldId":oldId
							},
      				{
      					headers: {
      						'content-type': 'application/json'
      					},
      					withCredentials: true
      				}).then(function (response) {

        _this.sleep = Array.isArray(response.data.sleep) ? response.data.sleep : []
        _this.createTime = Array.isArray(response.data.createTime) ? response.data.createTime : []
        if (!_this.sleep.length) { _this.chartError = '暂时没有可展示的睡眠详情。'; return }
        if (!_this.detailsVisible) return
        _this.drawLine()

      			})
      				.catch(function (error) {

          _this.chartError = '详情暂时无法加载，请重试。'
        }).then(function () { _this.chartLoading = false })

    },

	drawLine () {
    this.$nextTick(() => {
      if (!this.detailsVisible) return
      this.renderRecordChart(this.$refs.mychart, { title: '睡眠时长', subtitle: this.name, labels: this.createTime, values: this.sleep, unit: '小时', type: 'bar' })
    })
  }
  }
}

</script>

<style scoped>

  .sleep-chart { width: 100%; max-width: 520px; height: 300px; margin: 0 auto; }

</style>
