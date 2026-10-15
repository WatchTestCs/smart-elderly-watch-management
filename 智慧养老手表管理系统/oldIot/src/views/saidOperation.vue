<template>
  <div class="roles">
    
    
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
    
    
    <el-table class="adaptive-record-table" :data="saidList.slice((currentPage-1)*pageSize,currentPage*pageSize)"
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
    <record-cards :records="saidList.slice((currentPage-1)*pageSize,currentPage*pageSize)" :fields='[{"key":"saidId","label":"作者"},{"key":"said","label":"评论"},{"key":"back","label":"回复"},{"key":"backTime","label":"回复时间"},{"key":"title","label":"文章标题"}]' label="公告评论" title-key="personName" key-field="saidId" time-key="saidTime">
      <template #actions="scope"  ><el-button v-if="scope.row.back==null" size="small" plain type="primary" icon="el-icon-edit"
                     @click="handleeditSaid(scope.$index, scope.row)">回复
          </el-button>
<el-button size="small" plain type="danger" icon="el-icon-delete" @click="deleteSaid(scope.row.saidId)">删除
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
       // 角色查询内容
      
      watchEditDialogVisible: false,
      
      saidEditForm: {
        saidId: '',
        said: '',
        back: ''
      },
      
      total: 0,
      pageSize: 10,
      currentPage: 1,
      saidList: [],
      // 控制分配权限对话框的展示和隐藏
      

      // 树形控件的数据
      

      // Tree树形控件，默认使用数据中的 label 属性作为节点名称，使用 children 属性指定该接待你的子节点
      // 但是，如果我们自己的数据结构与默认的结构不同，可以通过 defaultProps 来修改默认的两个属性
      
      // 当前被分配权限的角色id
      
    }
  },
  created () {
    this.searchSaid()
  },
  methods: {
    // 查询
    searchSaid () {
      var _this = this
        
        axios.post('restful/said/selectByUser',
          {
            // 'oldName': oldName === '' ? '' : _this.searchText
          },
          {
            headers: {
              'content-type': 'application/json'
            },
            withCredentials: true
          }).then(function (response) {
          var tableData2 = response.data.saidDtos
          _this.total = response.data.saidDtos.length
					for(var i=0;i<tableData2.length;i++){
						console.log(tableData2[i].touPic)
						tableData2[i].touPic = '/restful/image/download?name='+tableData2[i].touPic
					}
					_this.saidList=tableData2
      console.log(_this.saidList)
        })
          .catch(function (error) {
            console.log(error)
          })
      
    },
    // 添加角色
    
    // 关闭添加角色对话框
    
    // 编辑
    editSaid () {
      var _this = this
      axios.post('restful/said/update',_this.saidEditForm,
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
    // 删除
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
          console.log(response)
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
        // 取消删除
        this.$message({
          type: 'info',
          message: '已取消删除'
        })
      }
    },
    // 编辑信息
    handleeditSaid (index, row) {
      this.watchEditDialogVisible = true
      this.saidEditForm = Object.assign({}, row) // 解决修改弹出框数据时，表格数据会发生同步改变
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
