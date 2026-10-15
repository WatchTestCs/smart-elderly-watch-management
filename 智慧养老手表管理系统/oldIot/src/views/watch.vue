<template>
  <div class="roles">
    
    <div class="sys-research">
      <el-row class="row-search">
        <el-col :span="8">
          
          <label class="table-search-label" for="search-watch">长者姓名</label>
          <el-input id="search-watch"
            placeholder="按长者姓名查找"
            v-model="searchText"
            class="input-with-select">
            <template #append><el-button aria-label="查询长者姓名"
              
              icon="el-icon-search"
              @click="searchWatchs"></el-button></template>
          </el-input>
        </el-col>
        <el-col :span="4">
          <el-button plain @click="watchDialogVisible = true">添加手表</el-button>
        </el-col>
      </el-row>
    </div>
    
    <el-dialog
      title="设备添加"
      v-model="watchDialogVisible"
      @close="closeRoleAddDialog"
      width="30%">
      <el-form label-width="80px" :model="watchForm"  ref="addwatchFormRef">
        
        <el-form-item label="手表Id" prop="deviceId">
          <el-input v-model="watchForm.deviceId"></el-input>
        </el-form-item>
		<el-form-item label="血压高值" prop="bpf">
		<el-input v-model="watchForm.bpf"></el-input>
		</el-form-item>
		<el-form-item label="血压低值" prop="bpl">
		<el-input v-model="watchForm.bpl"></el-input>
		</el-form-item>
		<el-form-item label="心率高值" prop="hrf">
		<el-input v-model="watchForm.hrf"></el-input>
		</el-form-item>
		<el-form-item label="心率低值" prop="hrl">
		<el-input v-model="watchForm.hrl"></el-input>
		</el-form-item>
      </el-form>
      <template #footer><span  class="dialog-footer">
      <el-button @click="watchDialogVisible = false">取 消</el-button>
      <el-button type="primary" @click="addWatchs('addwatchFormRef')">确 定</el-button>
    </span></template>
    </el-dialog>
    
    
    <el-dialog
      title="设备编辑"
      v-model="watchEditDialogVisible"
      width="30%">
      <el-form label-width="80px" :model="watchEditForm">
        
        <el-form-item label="手表Id" prop="deviceId">
        <el-input v-model="watchEditForm.deviceId" disabled="disabled"></el-input>
        </el-form-item>
        <el-form-item label="血压高值" prop="bpf">
        <el-input v-model="watchEditForm.bpf"></el-input>
        </el-form-item>
        <el-form-item label="血压低值" prop="bpl">
        <el-input v-model="watchEditForm.bpl"></el-input>
        </el-form-item>
        <el-form-item label="心率高值" prop="hrf">
        <el-input v-model="watchEditForm.hrf"></el-input>
        </el-form-item>
        <el-form-item label="心率低值" prop="hrl">
        <el-input v-model="watchEditForm.hrl"></el-input>
        </el-form-item>
      </el-form>
      <template #footer><span  class="dialog-footer">
      <el-button @click="watchEditDialogVisible = false">取 消</el-button>
      <el-button type="primary" @click="editWatch">确 定</el-button>
    </span></template>
    </el-dialog>
    
    
    <el-table class="adaptive-record-table" :data="watchsList.slice((currentPage-1)*pageSize,currentPage*pageSize)"
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
    <record-cards :records="watchsList.slice((currentPage-1)*pageSize,currentPage*pageSize)" :fields='[{"key":"deviceId","label":"手表ID"},{"key":"bpf","label":"血压高值"},{"key":"bpl","label":"血压低值"},{"key":"hrf","label":"心率高值"},{"key":"hrl","label":"心率低值"}]' label="监测阈值档案" title-key="name" key-field="deviceId" time-key="opTime">
      <template #actions="scope"  ><el-button size="small" plain type="primary" icon="el-icon-edit"
                     @click="handleEditwatch(scope.$index, scope.row)">编辑
          </el-button>
<el-button size="small" plain type="danger" icon="el-icon-delete" @click="deleteRole(scope.row.deviceId)">删除
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
  </div>

</template>

<script>
// 引入
import axios from 'axios'
import RecordCards from '@/components/RecordCards.vue'

export default {
  components: { RecordCards },
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
      
      total: 0,
      pageSize: 10,
      currentPage: 1,
      watchsList: [],
      // 控制分配权限对话框的展示和隐藏
      

      // 树形控件的数据
      

      // Tree树形控件，默认使用数据中的 label 属性作为节点名称，使用 children 属性指定该接待你的子节点
      // 但是，如果我们自己的数据结构与默认的结构不同，可以通过 defaultProps 来修改默认的两个属性
      
      // 当前被分配权限的角色id
      
    }
  },
  created () {
    this.searchWatchs()
  },
  methods: {
    // 查询
    searchWatchs () {
      var _this = this
        var oldName = _this.searchText
        
        axios.post('restful/watchInfo/select',
          {
            'oldName': oldName === '' ? '' : _this.searchText
          },
          {
            headers: {
              'content-type': 'application/json'
            },
            withCredentials: true
          }).then(function (response) {
          _this.watchsList = response.data.watchDtos
          _this.total = response.data.watchDtos.length
      console.log(_this.watchsList)
        })
          .catch(function (error) {
            console.log(error)
          })
      
    },
    // 添加角色
    addWatchs () {
      var _this = this
      axios.post('restful/watchInfo/add',
        _this.watchForm,
        {
          headers: {
            'content-type': 'application/json'
          },
          withCredentials: true
        }).then(function (response) {
        console.log(response)
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
    },
    // 关闭添加角色对话框
    closeRoleAddDialog () {
      // 重置表单
      this.$refs.addwatchFormRef.resetFields()
    },
    // 编辑
    editWatch () {
      var _this = this
      axios.post('restful/watchInfo/update', _this.watchEditForm,
        {
          headers: {
            'content-type': 'application/json'
          },
          withCredentials: true
        }).then(function (response) {
        console.log(response)
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
    },
    // 删除
    async deleteRole (deviceId) {
      console.log(deviceId)

      var _this = this
      try {
        await _this.$confirm('此操作将永久删除该文件, 是否继续?', '提示', {
          confirmButtonText: '确定',
          cancelButtonText: '取消',
          type: 'warning'
        })
        console.log(deviceId)
        axios.post('restful/watchInfo/delete', {'deviceId': deviceId},
          {
            headers: {
              'content-type': 'application/json'
            },
            withCredentials: true
          }).then(function (response) {
          console.log(response)
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
        // 取消删除
        this.$message({
          type: 'info',
          message: '已取消删除'
        })
      }
    },
    // 编辑信息
    handleEditwatch (index, row) {
      this.watchEditDialogVisible = true
      this.watchEditForm = Object.assign({}, row) // 解决修改弹出框数据时，表格数据会发生同步改变
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
    

    // 给角色分配权限
    
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
