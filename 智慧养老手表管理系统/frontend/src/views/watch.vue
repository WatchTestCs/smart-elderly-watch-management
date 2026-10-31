<template>
  <div class="roles" v-loading="isLoading" element-loading-text="正在加载监测阈值档案…" :aria-busy="String(isLoading)">
    
    <div class="sys-research">
      <form class="row-search" role="search" @submit.prevent="searchWatchs">
        <div class="search-field">
          
          <label class="table-search-label" for="search-watch">长者姓名</label>
          <el-input id="search-watch"
            placeholder="按长者姓名查找"
            v-model="searchText"
            class="input-with-select">
            <template #append><el-button aria-label="查询长者姓名"
              
              icon="el-icon-search"
              native-type="submit"></el-button></template>
          </el-input>
        </div>
        <div class="toolbar-actions">
          <el-button plain @click="watchDialogVisible = true">添加手表</el-button>
        </div>
      </form>
    </div>
    
    <el-dialog
      title="设备添加"
      v-model="watchDialogVisible"
      @close="closeRoleAddDialog"
      width="30%">
      <ThresholdForm :draft="watchForm" :rules="rules" ref="addwatchFormRef" />
      <template #footer><span  class="dialog-footer">
      <el-button @click="watchDialogVisible = false">取 消</el-button>
      <el-button type="primary" @click="addWatchs('addwatchFormRef')">确 定</el-button>
    </span></template>
    </el-dialog>

    <el-dialog
      title="设备编辑"
      v-model="watchEditDialogVisible"
      width="30%">
      <ThresholdForm editing :draft="watchEditForm" :rules="rules" ref="editwatchFormRef" />
      <template #footer><span  class="dialog-footer">
      <el-button @click="watchEditDialogVisible = false">取 消</el-button>
      <el-button type="primary" @click="editWatch">确 定</el-button>
    </span></template>
    </el-dialog>

    <div v-if="loadError" class="collection-feedback" role="alert"><span>{{ loadError }}</span><el-button plain :loading="isLoading" @click="searchWatchs(true)">重新加载</el-button></div>
    <el-table v-if="!loadError" class="adaptive-record-table" :data="watchsList"
    style="width: 100%" border>
    <template #empty><div  class="care-empty-state"><strong>暂无监测阈值档案</strong><span>添加手表或调整搜索条件后，档案会显示在这里。</span></div></template>

      <el-table-column prop="deviceId" label="手表ID" width="100" align="center"></el-table-column>
      <el-table-column prop="name" label="佩戴老人" width="160" align="center"></el-table-column>
      <el-table-column prop="bpf" label="血压高值" width="160" align="center"></el-table-column>
			<el-table-column prop="bpl" label="血压低值" width="160" align="center"></el-table-column>
			<el-table-column prop="hrf" label="心率高值" width="160" align="center"></el-table-column>
			<el-table-column prop="hrl" label="心率低值" width="160" align="center"></el-table-column>
			<el-table-column prop="opTime" label="启用时间" width="160" align="center"></el-table-column>
	  
      <el-table-column label="操作" align="center">
        <template #default="scope" >
          <el-button size="small" plain type="primary" icon="el-icon-edit"
                     @click="handleEditwatch(scope.$index, scope.row)">编辑
          </el-button>
          <el-button size="small" plain type="danger" icon="el-icon-delete" @click="deleteRole(scope.row.deviceId)">删除
          </el-button>
          
        </template>
      </el-table-column>
    </el-table>
    <record-cards v-if="!loadError" :records="watchsList" :fields='[{"key":"deviceId","label":"手表ID"},{"key":"bpf","label":"血压高值"},{"key":"bpl","label":"血压低值"},{"key":"hrf","label":"心率高值"},{"key":"hrl","label":"心率低值"}]' label="监测阈值档案" title-key="name" key-field="deviceId" time-key="opTime">
      <template #actions="scope"  ><el-button size="small" plain type="primary" icon="el-icon-edit"
                     @click="handleEditwatch(scope.$index, scope.row)">编辑
          </el-button>
<el-button size="small" plain type="danger" icon="el-icon-delete" @click="deleteRole(scope.row.deviceId)">删除
          </el-button></template>
    </record-cards>
    
    <div v-if="!loadError" style="margin-top: 10px;text-align: center">
      <el-pagination
        @size-change="handleSizeChange"
        @current-change="changePage"
        v-model:current-page="currentPage"
        :page-size="pageSize"
        layout="total, sizes, prev, pager, next, jumper"
        :total="total">
      </el-pagination>
    </div>
  </div>

</template>

<script>
import axios from 'axios'
import { checkThreshold } from './watch.validation'
import RecordCards from '@/components/RecordCards.vue'
import ThresholdForm from '@/components/records/ThresholdForm.vue'

export default {
  components: { RecordCards, ThresholdForm },
  data () {
    return {
      searchText: '', // 角色查询内容
      watchDialogVisible: false,
      watchEditDialogVisible: false,
      watchForm: {
        deviceId: '',
				bpf: '',
				bpl: '',
				hrf: '',
				hrl: ''
      },
      watchEditForm: {
        deviceId: '',
        bpf: '',
        bpl: '',
        hrf: '',
        hrl: ''
      },
      rules: {
        deviceId: [
          { required: true, message: '请输入手表Id', trigger: 'blur' }
        ],
        bpf: [{ validator: checkThreshold, trigger: 'blur' }],
        bpl: [{ validator: checkThreshold, trigger: 'blur' }],
        hrf: [{ validator: checkThreshold, trigger: 'blur' }],
        hrl: [{ validator: checkThreshold, trigger: 'blur' }]
      },

      isLoading: false,
      loadError: '',
      total: 0,
      pageSize: 10,
      currentPage: 1,
      watchsList: [],

    }
  },
  created () {
    this.searchWatchs()
  },
  methods: {
    searchWatchs (keepPage = false) {
      if (this.isLoading) return
      this.isLoading = true
      this.loadError = ''
      if (keepPage !== true) this.currentPage = 1
      var _this = this
        var oldName = _this.searchText
        
        return axios.post('api/v1/watchInfo/select',
          { page: this.currentPage, pageSize: this.pageSize,
            'oldName': oldName === '' ? '' : _this.searchText
          },
          {
            headers: {
              'content-type': 'application/json'
            },
            withCredentials: true
          }).then(function (response) {
          if (!Array.isArray(response.data.watchDtos)) throw new Error('invalid records')
          _this.watchsList = response.data.watchDtos
          _this.total = response.data.total ?? response.data.watchDtos.length

        })
          .catch(function () {
            _this.loadError = '监测阈值档案暂时无法加载，请检查服务连接后重试。'
            _this.watchsList = []
            _this.total = 0
          }).finally(function () { _this.isLoading = false })
      
    },
    addWatchs () {
      var _this = this
      this.$refs.addwatchFormRef.validate(function (valid) {
        if (!valid) return
        axios.post('api/v1/watchInfo/add',
          _this.watchForm,
          {
            headers: {
              'content-type': 'application/json'
            },
            withCredentials: true
          }).then(function (response) {

          if (response.data.returnCode === '1111') {
            _this.watchDialogVisible = false
            _this.searchWatchs()
            _this.$message({
              type: 'success',
              message: '添加手表成功'
            })
          } else {
            _this.$message({
              type: 'error',
              message: '添加手表失败——可能原因手表ID重复'
            })
          }
        })
          .catch(function (error) {
            console.log(error)
          })
      })
    },
    closeRoleAddDialog () {
      this.$refs.addwatchFormRef.resetFields()
    },
    editWatch () {
      var _this = this
      this.$refs.editwatchFormRef.validate(function (valid) {
        if (!valid) return
        axios.post('api/v1/watchInfo/update', _this.watchEditForm,
          {
            headers: {
              'content-type': 'application/json'
            },
            withCredentials: true
          }).then(function (response) {

          if (response.data.returnCode === '1111') {
            _this.watchEditDialogVisible = false
            _this.$message({
              type: 'success',
              message: '编辑手表信息成功'
            })
            _this.searchWatchs()
          } else {
            _this.$message({
              type: 'error',
              message: '编辑手表信息失败'
            })
          }
        })
          .catch(function (error) {
            console.log(error)
          })
      })
    },
    async deleteRole (deviceId) {

      var _this = this
      try {
        await _this.$confirm('此操作将永久删除该文件, 是否继续?', '提示', {
          confirmButtonText: '确定',
          cancelButtonText: '取消',
          type: 'warning'
        })

        axios.post('api/v1/watchInfo/delete', {'deviceId': deviceId},
          {
            headers: {
              'content-type': 'application/json'
            },
            withCredentials: true
          }).then(function (response) {

          if (response.data.returnCode === '1111') {
            _this.$message({
              type: 'success',
              message: '删除信息成功'
            })
          }
          _this.searchWatchs()
        })
          .catch(function (error) {
            console.log(error)
          })
      } catch (err) {
        this.$message({
          type: 'info',
          message: '已取消删除'
        })
      }
    },
    handleEditwatch (index, row) {
      this.watchEditDialogVisible = true
      this.watchEditForm = Object.assign({}, row) // 解决修改弹出框数据时，表格数据会发生同步改变

    },
    handleSizeChange (val) { this.pageSize = val; this.currentPage = 1; this.searchWatchs(true)
    },

    changePage (val) { this.currentPage = val; this.searchWatchs(true)
    },


  }
}

</script>
