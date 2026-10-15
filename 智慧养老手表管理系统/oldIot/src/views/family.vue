<template>
  <div class="roles">
    
    <div class="sys-research">
      <el-row class="row-search">
        <el-col :span="8">
          
          <label class="table-search-label" for="search-family">家属姓名</label>
          <el-input id="search-family"
            placeholder="按家属姓名查找"
            v-model="searchText"
            class="input-with-select">
            <template #append><el-button aria-label="查询家属姓名"
              
              icon="el-icon-search"
              @click="searchfamily"></el-button></template>
          </el-input>
        </el-col>
        <el-col :span="4">
          <el-button plain @click="customerDialogVisible = true">添加家属</el-button>
        </el-col>
		
      </el-row>
    </div>
    
    <el-dialog
      title="家属添加"
      v-model="customerDialogVisible"
      @close="closeRoleAddDialog"
      width="92%" custom-class="admin-record-dialog">
      <el-form label-width="80px" :model="familyForm" :rules="rules" class="demo-ruleForm"  ref="addfamilyFormRef">
        <el-form-item label="姓名" prop="familyName">
          <el-input v-model="familyForm.familyName"></el-input>
        </el-form-item>
        <el-form-item label="关系" prop="familyRelation">
          <el-input v-model="familyForm.familyRelation"></el-input>
        </el-form-item>
		<el-form-item label="性别" prop="familySex">
		<el-radio-group v-model="familyForm.familySex">
		  <el-radio value="男">男</el-radio>
		  <el-radio value="女">女</el-radio>
		</el-radio-group>
		</el-form-item>
		<el-form-item label="电话" prop="familyPhone">
		<el-input v-model="familyForm.familyPhone"></el-input>
		</el-form-item>
		<el-form-item label="地址" prop="familyAddress">
		<el-input v-model="familyForm.familyAddress"></el-input>
		</el-form-item>
		<el-form-item label="老人电话" prop="oldPhone">
		<el-input v-model="familyForm.oldPhone"></el-input>
		</el-form-item>
      </el-form>
      <template #footer><span  class="dialog-footer">
      <el-button @click="customerDialogVisible = false">取 消</el-button>
      <el-button type="primary" @click="addFamily('addfamilyFormRef')">确 定</el-button>
    </span></template>
    </el-dialog>
    
    
    <el-dialog
      title="家属编辑"
      v-model="oldEditDialogVisible"
      width="92%" custom-class="admin-record-dialog">
      <el-form label-width="80px" :rules="rules" :model="familyEditForm">
        <el-form-item label="姓名" prop="familyName">
        <el-input v-model="familyEditForm.familyName"></el-input>
        </el-form-item>
		<el-form-item label="姓名" prop="familyId" style="display: none;">
		<el-input v-model="familyEditForm.familyId"></el-input>
		</el-form-item>		
        <el-form-item label="关系" prop="familyRelation">
        <el-input v-model="familyEditForm.familyRelation"></el-input>
        </el-form-item>
        <el-form-item label="性别" prop="familySex">
        <el-radio-group v-model="familyEditForm.familySex">
        <el-radio value="男">男</el-radio>
        <el-radio value="女">女</el-radio>
        </el-radio-group>
        </el-form-item>
        <el-form-item label="电话" prop="familyPhone">
        <el-input v-model="familyEditForm.familyPhone"></el-input>
        </el-form-item>
        <el-form-item label="地址" prop="familyAddress">
        <el-input v-model="familyEditForm.familyAddress"></el-input>
        </el-form-item>
      </el-form>
      <template #footer><span  class="dialog-footer">
      <el-button @click="oldEditDialogVisible = false">取 消</el-button>
      <el-button type="primary" @click="editFamily">确 定</el-button>
    </span></template>
    </el-dialog>
    
    
    <el-table class="adaptive-record-table" :data="familyList.slice((currentPage-1)*pageSize,currentPage*pageSize)"
    style="width: 100%" border>
    <template #empty><div  class="care-empty-state"><strong>暂无家属账号</strong><span>添加家属或调整搜索条件后，账号会显示在这里。</span></div></template>
    
      
      <el-table-column type="index" width="100" align="center"></el-table-column>
	  <el-table-column prop="name" label="姓名" width="140" align="center"></el-table-column>
	  <el-table-column prop="oldId" label="老人编号" width="140" align="center"></el-table-column>
      <el-table-column prop="familyName" label="家属姓名" width="140" align="center"></el-table-column>
      <el-table-column prop="familyRelation" label="关系" width="140" align="center"></el-table-column>
	  <el-table-column prop="familySex" label="性别" width="140" align="center"></el-table-column>
	  <el-table-column prop="familyPhone" label="电话" width="140" align="center"></el-table-column>
	  <el-table-column prop="familyAddress" label="地址" width="auto" align="center"></el-table-column>
      <el-table-column label="操作" align="center">
        <template #default="scope" >
          <el-button size="small" plain type="primary" icon="el-icon-edit"
                     @click="handleeditFamily(scope.$index, scope.row)">编辑
          </el-button>
          <el-button size="small" plain type="danger" icon="el-icon-delete" @click="deleteFamily(scope.row.familyId)">删除
          </el-button>
          
        </template>
      </el-table-column>
    </el-table>
    <record-cards :records="familyList.slice((currentPage-1)*pageSize,currentPage*pageSize)" :fields='[{"key":"name","label":"姓名"},{"key":"oldId","label":"老人编号"},{"key":"familyRelation","label":"关系"},{"key":"familySex","label":"性别"},{"key":"familyPhone","label":"电话"},{"key":"familyAddress","label":"地址"}]' label="家属账号资料" title-key="familyName" key-field="familyId">
      <template #actions="scope"  ><el-button size="small" plain type="primary" icon="el-icon-edit"
                     @click="handleeditFamily(scope.$index, scope.row)">编辑
          </el-button>
<el-button size="small" plain type="danger" icon="el-icon-delete" @click="deleteFamily(scope.row.familyId)">删除
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
	  var checkPhone = (rule, value, callback) => {
    const phoneReg = /^1[3|4|5|7|8][0-9]{9}$/
    if (!value) {
      return callback(new Error('电话号码不能为空'))
    }
    setTimeout(() => {
      // Number.isInteger是es6验证数字是否为整数的方法,但是我实际用的时候输入的数字总是识别成字符串
      // 所以我就在前面加了一个+实现隐式转换

      if (!Number.isInteger(+value)) {
        callback(new Error('请输入数字值'))
      } else {
        if (phoneReg.test(value)) {
          callback()
        } else {
          callback(new Error('电话号码格式不正确'))
        }
      }
    }, 100)
  }
    return {
      searchText: '', // 角色查询内容
      customerDialogVisible: false,
      oldEditDialogVisible: false,
      familyForm: {
        familyName: '',
        familyRelation: '',
		familySex: '',
		familyPhone: '',
		familyAddress: '',
		oldPhone: ''
      },
      familyEditForm: {
        customerName: '',
        customerAge: '',
        customerSex: '',
        customerPhone: '',
        customerAddress: '',
		customerId: ''
      },
      
      total: 0,
      pageSize: 10,
      currentPage: 1,
      familyList: [],
      // 控制分配权限对话框的展示和隐藏
      
      // 树形控件的数据
      

      // Tree树形控件，默认使用数据中的 label 属性作为节点名称，使用 children 属性指定该接待你的子节点
      // 但是，如果我们自己的数据结构与默认的结构不同，可以通过 defaultProps 来修改默认的两个属性
      
      // 当前被分配权限的角色id
      
	  rules: {
			familyPhone: [
			  { validator: checkPhone, trigger: 'blur'}
			],
			oldPhone: [
			  { validator: checkPhone, trigger: 'blur'}
			]
		  }
    }
  },
  created () {
    this.searchfamily()
  },
  methods: {
    // 查询
    searchfamily () {
     var _this = this
       var familyName = _this.searchText
       
       axios.post('restful/family/select',
         {
           'familyName': familyName === '' ? '' : _this.searchText
         },
         {
           headers: {
             'content-type': 'application/json'
           },
           withCredentials: true
         }).then(function (response) {
         _this.familyList = response.data.familyDtos
         _this.total = response.data.familyDtos.length
     console.log(_this.familyList)
       })
         .catch(function (error) {
           console.log(error)
         })
      
    },
    // 添加角色
    addFamily () {
      var _this = this
      axios.post('restful/family/add',
        _this.familyForm,
        {
          headers: {
            'content-type': 'application/json'
          },
          withCredentials: true
        }).then(function (response) {
        console.log(response)
        if (response.data.returnCode === '1111') {
          _this.customerDialogVisible = false
          _this.searchfamily()
          _this.$message({
            type: 'success',
            message: '添加监护人成功'
          })
        } else {
          _this.$message({
            type: 'error',
            message: '添加监护人失败'
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
      this.$refs.addfamilyFormRef.resetFields()
    },
    // 编辑
    editFamily () {
      var _this = this
      axios.post('restful/family/updateFamily', _this.familyEditForm,
        {
          headers: {
            'content-type': 'application/json'
          },
          withCredentials: true
        }).then(function (response) {
        console.log(response)
        if (response.data.returnCode === '1111') {
          _this.oldEditDialogVisible = false
          _this.$message({
            type: 'success',
            message: '编辑监护人信息成功'
          })
          _this.searchfamily()
        } else {
          _this.$message({
            type: 'error',
            message: '编辑监护人信息失败'
          })
        }
      })
        .catch(function (error) {
          console.log(error)
        })
    },
    // 删除
    async deleteFamily (familyId) {
      console.log(familyId)

      var _this = this
      try {
        await _this.$confirm('此操作将永久删除该文件, 是否继续?', '提示', {
          confirmButtonText: '确定',
          cancelButtonText: '取消',
          type: 'warning'
        })
        console.log(familyId)
        axios.post('restful/family/deleteFamily', {'familyId': familyId},
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
          _this.searchfamily()
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
    handleeditFamily (index, row) {
      this.oldEditDialogVisible = true
      this.familyEditForm = Object.assign({}, row) // 解决修改弹出框数据时，表格数据会发生同步改变
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
