<template>
  <div class="roles" v-loading="isLoading" element-loading-text="正在加载评论…" :aria-busy="String(isLoading)">

    <el-dialog
      title="评论回复"
      v-model="watchEditDialogVisible"
      width="30%">
      <el-form label-width="80px" :model="saidEditForm">
        <el-form-item label="" prop="saidId" style="display: none; ">
        <el-input v-model="saidEditForm.saidId"></el-input>
        </el-form-item>
        <el-form-item label="评论" prop="said" >
        <el-input v-model="saidEditForm.said" disabled="disabled"></el-input>
        </el-form-item>
        <el-form-item label="回复" prop="back">
        <el-input v-model="saidEditForm.back"></el-input>
        </el-form-item>
      </el-form>
      <template #footer><span  class="dialog-footer">
      <el-button @click="watchEditDialogVisible = false">取 消</el-button>
      <el-button type="primary" @click="editSaid">确 定</el-button>
    </span></template>
    </el-dialog>

    <div v-if="loadError" class="collection-feedback" role="alert"><span>{{ loadError }}</span><el-button plain :loading="isLoading" @click="searchSaid(true)">重新加载</el-button></div>
    <el-table v-if="!loadError" class="adaptive-record-table" :data="saidList"
    style="width: 100%" >
    <template #empty><div  class="care-empty-state"><strong>暂无评论</strong><span>收到的评论和回复会显示在这里。</span></div></template>

      <el-table-column prop="personName" label="作者" width="100" align="center"></el-table-column>
	  
      <el-table-column prop="said" label="评论" width="160" align="center"></el-table-column>
			<el-table-column prop="saidTime" label="评论时间" width="180" align="center"></el-table-column>
			<el-table-column prop="back" label="回复" width="160" align="center"></el-table-column>
			<el-table-column prop="backTime" label="回复时间" width="180" align="center"></el-table-column>
			<el-table-column prop="title" label="文章标题" width="160" align="center"></el-table-column>
	  
      <el-table-column label="操作" align="center">
        <template #default="scope" >
          <el-button v-if="scope.row.back==null" size="small" plain type="primary" icon="el-icon-edit"
                     @click="handleeditSaid(scope.$index, scope.row)">回复
          </el-button>
          <el-button size="small" plain type="danger" icon="el-icon-delete" @click="deleteSaid(scope.row.saidId)">删除
          </el-button>
          
        </template>
      </el-table-column>
    </el-table>
    <record-cards v-if="!loadError" :records="saidList" :fields='[{"key":"saidId","label":"作者"},{"key":"said","label":"评论"},{"key":"back","label":"回复"},{"key":"backTime","label":"回复时间"},{"key":"title","label":"文章标题"}]' label="公告评论" title-key="personName" key-field="saidId" time-key="saidTime">
      <template #actions="scope"  ><el-button v-if="scope.row.back==null" size="small" plain type="primary" icon="el-icon-edit"
                     @click="handleeditSaid(scope.$index, scope.row)">回复
          </el-button>
<el-button size="small" plain type="danger" icon="el-icon-delete" @click="deleteSaid(scope.row.saidId)">删除
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
import RecordCards from '@/components/RecordCards.vue'

export default {
  components: { RecordCards },
  data () {
    return {
      
      watchEditDialogVisible: false,
      
      saidEditForm: {
        saidId: '',
        said: '',
        back: ''
      },
      
      isLoading: false,
      loadError: '',
      total: 0,
      pageSize: 10,
      currentPage: 1,
      saidList: [],

    }
  },
  created () {
    this.searchSaid()
  },
  methods: {
    searchSaid (keepPage = false) {
      if (this.isLoading) return
      this.isLoading = true
      this.loadError = ''
      if (keepPage !== true) this.currentPage = 1
      var _this = this
        
        return axios.post('restful/said/selectByUser',
          { page: this.currentPage, pageSize: this.pageSize,
          },
          {
            headers: {
              'content-type': 'application/json'
            },
            withCredentials: true
          }).then(function (response) {
          if (!Array.isArray(response.data.saidDtos)) throw new Error('invalid records')
          var tableData2 = response.data.saidDtos
          _this.total = response.data.total ?? response.data.saidDtos.length
					for(var i=0;i<tableData2.length;i++){

						tableData2[i].touPic = '/restful/image/download?name='+tableData2[i].touPic
					}
					_this.saidList=tableData2

        })
          .catch(function () {
            _this.loadError = '评论暂时无法加载，请检查服务连接后重试。'
            _this.saidList = []
            _this.total = 0
          }).finally(function () { _this.isLoading = false })
      
    },
    
    
    editSaid () {
      var _this = this
      axios.post('restful/said/update',_this.saidEditForm,
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
            message: '回复评论成功'
          })
          _this.searchSaid()
        } else {
          _this.$message({
            type: 'error',
            message: '回复评论失败'
          })
        }
      })
        .catch(function (error) {
          console.log(error)
        })
    },
    async deleteSaid (saidId) {

      var _this = this
      try {
        await _this.$confirm('此操作将永久删除该文件, 是否继续?', '提示', {
          confirmButtonText: '确定',
          cancelButtonText: '取消',
          type: 'warning'
        })
        axios.post('restful/said/delete', {'saidId': saidId},
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
          _this.searchSaid()
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
    handleeditSaid (index, row) {
      this.watchEditDialogVisible = true
      this.saidEditForm = Object.assign({}, row) // 解决修改弹出框数据时，表格数据会发生同步改变

    },
    handleSizeChange (val) {

      this.pageSize = val
      this.currentPage = 1
      this.searchSaid(true)
    },

    changePage (val) {

      this.currentPage = val
      this.searchSaid(true)
    },


  }
}

</script>
