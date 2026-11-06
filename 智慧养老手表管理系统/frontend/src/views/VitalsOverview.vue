<template>
  <div class="roles">

    <div class="sys-research">
      <form class="row-search" role="search" @submit.prevent="searchOlds">
        <div class="search-field">
          
          <label class="table-search-label" for="search-vitals">长者姓名</label>
          <el-input id="search-vitals"
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

    <el-table v-if="!mobileLayout" class="adaptive-record-table" :data="oldsList"
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
    <record-cards v-if="mobileLayout" :records="oldsList" :fields='[{"key":"preState","label":"血压状态","state":"health","handled":"preHandled"},{"key":"rateState","label":"心率状态","state":"health","handled":"rateHandled"},{"key":"oldId","label":"长者编号"},{"key":"sex","label":"性别"}]' label="心率与血压记录" title-key="name" key-field="oldId" time-key="createTime">
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
      <pager-bar
        @size-change="handleSizeChange"
        @current-change="changePage"
        v-model:current-page="currentPage"
        :page-size="pageSize"
        :total="total" />
    </div>

    <HealthAlertDialog :dialog="alertDialog" @resolve="resolveAlert" />

    <el-dialog title="健康记录详情" v-model="detailsVisible" @opened="resizeCharts" @closed="disposeCharts">
      <el-alert v-if="chartError" :title="chartError" type="warning" :closable="false" show-icon></el-alert>
      <div v-loading="chartLoading" class="record-chart-content"><div id="myChart" ref="mychart" class="record-chart"></div><div id="myChartTwo" ref="mychartTwo" class="record-chart"></div></div>
      <template #footer><div  class="dialog-footer"><el-button type="primary" @click="detailsVisible = false">关闭详情</el-button></div></template>
    </el-dialog>
  </div>

</template>

<script>
import { pagedRecords } from '@/composables/pagedRecords'
import { api as requestClient } from '../api/client'
import { useCollectionLayout } from '@/composables/useCollectionLayout'
import { apiErrorMessage } from '../api/httpContract'

import RecordCards from '@/components/RecordCards.vue'
import HealthAlertDialog from '@/components/records/HealthAlertDialog.vue'
import { healthAlertMethods } from './vitals.alert-methods'

import { useRecordCharts } from '@/composables/useRecordCharts'

export default {
  setup () { return { mobileLayout: useCollectionLayout(900), ...useRecordCharts() } },
  components: { RecordCards, HealthAlertDialog },
  mixins: [pagedRecords({ path: "api/v1/vitals/latest", responseKey: "vitalRecords", recordsKey: "oldsList", searchKey: "name", load: "searchOlds", filters: {"timeFirst":"","timeLast":""} })],
  data () {
    return {
      searchText: '',

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

  methods: {
    ...healthAlertMethods,
    getCount () {
      requestClient.post('api/v1/vitals/alerts/device-counts', {})
        .then(({ data }) => {
          this.rateCount = data.heartRate
          this.preCount = data.bloodPressure
        })
        .catch(error => this.$message.error(apiErrorMessage(error, '请求未完成，请重试')))
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



    showRoleDialog (row) {
      return this.showRecordDetails('vitals', row.oldId, row.name)
    }
  }
}
</script>
