<template>
  <div class="roles">
    
    <div class="sys-research">
      <el-row class="row-search">
        <el-col :span="8">
          
          <label class="table-search-label" for="search-step">长者姓名</label>
          <el-input id="search-step"
            placeholder="按长者姓名查找"
            v-model="searchText"
            class="input-with-select">
            <template #append><el-button aria-label="查询长者姓名"
              
              icon="el-icon-search"
              @click="searchStep"></el-button></template>
          </el-input>
        </el-col>
        <el-col :span="8">
        <div class="block">
        <el-date-picker
        v-model="value6"
				value-format="YYYY-MM-DD"
        type="daterange"
        range-separator="至"
        start-placeholder="开始日期"
        end-placeholder="结束日期"
		style="display: none;">
        </el-date-picker>
        </div>
        </el-col>
		<span class="record-summary" role="status">睡眠异常 <strong>{{ stepCount === '' ? '—' : stepCount }}</strong></span>
      </el-row>
    </div>
    
    <el-dialog
      title="角色添加"
      v-model="roleDialogVisible"
      @close="closeRoleAddDialog"
      width="30%">
      <el-form label-width="80px" :model="roleForm"  ref="addRoleFormRef">
        <el-form-item label="角色名称" prop="roleName">
          <el-input v-model="roleForm.roleName"></el-input>
        </el-form-item>
        <el-form-item label="角色描述" prop="roleDesc">
          <el-input v-model="roleForm.roleDesc"></el-input>
        </el-form-item>
      </el-form>
      <template #footer><span  class="dialog-footer">
      <el-button @click="roleDialogVisible = false">取 消</el-button>
      <el-button type="primary" @click="addRoles('addRoleFormRef')">确 定</el-button>
    </span></template>
    </el-dialog>
    
    
    <el-dialog
      title="角色编辑"
      v-model="roleEditDialogVisible"
      width="30%">
      <el-form label-width="80px" :model="roleEditForm">
        <el-form-item label="角色名称" prop="roleName">
          <el-input v-model="roleEditForm.roleName"></el-input>
        </el-form-item>
        <el-form-item label="角色描述" prop="roleDesc">
          <el-input v-model="roleEditForm.roleDesc"></el-input>
        </el-form-item>
      </el-form>
      <template #footer><span  class="dialog-footer">
      <el-button @click="roleEditDialogVisible = false">取 消</el-button>
      <el-button type="primary" @click="editRole">确 定</el-button>
    </span></template>
    </el-dialog>
    
    
    <el-table class="adaptive-record-table" :data="stepList.slice((currentPage-1)*pageSize,currentPage*pageSize)"
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
    <record-cards :records="stepList.slice((currentPage-1)*pageSize,currentPage*pageSize)" :fields='[{"key":"sex","label":"性别"},{"key":"step","label":"今日步数"},{"key":"sleep","label":"睡眠时间"},{"key":"sleepSate","label":"睡眠情况","state":"health"}]' label="睡眠与步数记录" title-key="name" key-field="oldId" time-key="createTime">
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
// 引入
import axios from 'axios'
import RecordCards from '@/components/RecordCards.vue'

import { recordCharts } from '@/charts'

export default {
  components: { RecordCards },
  mixins: [recordCharts],
  data () {
    return {
      searchText: '',
      chartLoading: false, chartError: '', // 角色查询内容
      roleDialogVisible: false,
      roleEditDialogVisible: false,
      roleForm: {
        roleName: '',
        roleDesc: ''
      },
			value6: '',
      roleEditForm: {
        roleId: '',
        roleName: '',
        roleDesc: ''
      },
      popoverRole: true,
      total: 0,
      pageSize: 10,
      currentPage: 1,
      stepList: [],
		sleep: '',
		sleep: [],
		name: '',
		time: '',
		createTime: [],
      // 控制分配权限对话框的展示和隐藏
      isShowRoleDialog: false,

      // 树形控件的数据
      rightsTree: [],

      // Tree树形控件，默认使用数据中的 label 属性作为节点名称，使用 children 属性指定该接待你的子节点
      // 但是，如果我们自己的数据结构与默认的结构不同，可以通过 defaultProps 来修改默认的两个属性
      defaultProps: {
        // children: 'children',
        label: 'authName'
      },
      // 当前被分配权限的角色id
      roleId: -1,
			stepCount:''
    }
  },
  created () {
	  // let myChart = this.$echarts.init(document.getElementById('myChart'))
    this.searchStep()
		this.getCount()
	// this.drawLine()
  },
  updated() {
	  // this.drawLine()
     // this.getAttendanceData();
  },
  mounted(){
    // this.drawLine()
  },
  methods: {
    // 查询
    searchStep () {
      
      var _this = this
      var name = _this.searchText
      if(this.value6.length==0){
      	this.value6=['','']
      }
      console.log('00'+this.value6[0])
      axios.post('restful/step/selectStep',
        {
          'name': name === '' ? '' : _this.searchText,
      		'timeFirst':_this.value6[0] =='' ? '' : _this.value6[0],
      		'timeLast': _this.value6[1] =='' ? '' : _this.value6[1]
        },
        {
          headers: {
            'content-type': 'application/json'
          },
          withCredentials: true
        }).then(function (response) {
        _this.stepList = response.data.stepDtos
        _this.total = response.data.stepDtos.length
      })
        .catch(function (error) {
          console.log(error)
        })
    },
		getCount(){
			console.log('kaishe')
			var _this=this
			axios.post('restful/step/selectStepCount',
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
    // 添加角色
    addRoles () {
      var _this = this
      axios.post('restful/roles/save',
        {
          'roleName': _this.roleForm.roleName
        },
        {
          headers: {
            'content-type': 'application/json'
          },
          withCredentials: true
        }).then(function (response) {
        console.log(response)
        if (response.data.returnCode === '1111') {
          _this.roleDialogVisible = false
          _this.searchStep()
          _this.$message({
            type: 'success',
            message: '添加角色成功'
          })
        } else {
          _this.$message({
            type: 'error',
            message: '添加角色失败'
          })
        }
      })
        .catch(function (error) {
          console.log(error)
        })
    },
    // 关闭添加角色对话框
    closeRoleAddDialog () {
      // 重置表单
      this.$refs.addRoleFormRef.resetFields()
    },
    // 编辑
    editRole () {

    },
    // 删除
    async deleteRole (roleName) {

    },
    // 编辑信息
    handleEditRole (index, row) {
      this.roleEditDialogVisible = true
      this.roleEditForm = Object.assign({}, row) // 解决修改弹出框数据时，表格数据会发生同步改变
      // console.log(row)
    },
    handleSizeChange (val) {
      console.log(`每页 ${val} 条`)
      this.pageSize = val
    },
    handleCurrentChange (val) {
      console.log(`当前页: ${val}`)
      this.currentPage = val
    },
    changePage (val) {
      console.log(`当前页: ${val}`)
      this.currentPage = val
    },
    // 展示分配权限对话框
    showRoleDialog (curOld) {
      this.isShowRoleDialog = true
      this.chartLoading = true
      this.chartError = ''
      this.name = curOld.name || ''
      return axios.post('restful/step/selectStepDetail', { oldId: curOld.oldId }, { withCredentials: true }).then(response => {
        if (!this.isShowRoleDialog || this._isDestroyed) return
        this.sleep = Array.isArray(response.data.sleep) ? response.data.sleep : []; this.createTime = Array.isArray(response.data.createTime) ? response.data.createTime : [];
        if (!this.sleep.length) { this.chartError = '暂无可展示的睡眠详情。'; return }
        this.drawLine()
      }).catch(() => { this.chartError = '详情暂时无法加载，请关闭后重试。' }).then(() => { this.chartLoading = false })
    },

	
	//  echarts 
	drawLine () {
    this.$nextTick(() => {
      if (!this.isShowRoleDialog) return
      this.renderRecordChart(this.$refs.mychart, { title: '睡眠时长', subtitle: this.name, labels: this.createTime, values: this.sleep, unit: '小时', type: 'bar' })
    })
  }
  }
}

</script>

<style lang="less" scoped>
  .sys-research {
    height: 60px;
    padding: 20px 20px 0;
  }

  .level1 {
    border-bottom: 1px dotted #ccc;
    padding: 10px 0;

    &:last-child {
      border: 0;
    }
  }

  .level2 {
    padding: 5px 0;
  }

  .level3 {
    margin: 3px;
  }

</style>
