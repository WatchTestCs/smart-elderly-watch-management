<template>
  <div class="roles">
    
    <div class="sys-research">
      <el-row class="row-search">
        <el-col :span="8">
          
          <label class="table-search-label" for="search-customer">监护人姓名</label>
          <el-input id="search-customer"
            placeholder="按监护人姓名查找"
            v-model="searchText"
            class="input-with-select">
            <template #append><el-button aria-label="查询监护人姓名"
              
              icon="el-icon-search"
              @click="searchCustomers"></el-button></template>
          </el-input>
        </el-col>
        <el-col :span="4">
          <el-button plain @click="customerDialogVisible = true">添加监护人</el-button>
        </el-col>
		
      </el-row>
    </div>
    
    <el-dialog
      title="监护人添加"
      v-model="customerDialogVisible"
      @close="closeRoleAddDialog"
      width="30%">
      <el-form label-width="80px" :model="customerForm" :rules="rules"  ref="addcustomerFormRef">
        <el-form-item label="姓名" prop="customerName">
          <el-input v-model="customerForm.customerName"></el-input>
        </el-form-item>
        <el-form-item label="年龄" prop="customerAge">
          <el-input v-model="customerForm.customerAge"></el-input>
        </el-form-item>
		<el-form-item label="性别" prop="customerSex">
		<el-radio-group v-model="customerForm.customerSex">
		  <el-radio value="男">男</el-radio>
		  <el-radio value="女">女</el-radio>
		</el-radio-group>
		</el-form-item>
		<el-form-item label="电话" prop="customerPhone">
		<el-input v-model="customerForm.customerPhone"></el-input>
		</el-form-item>
		<el-form-item label="地址" prop="customerAddress">
		<el-input v-model="customerForm.customerAddress"></el-input>
		</el-form-item>
		
      </el-form>
      <template #footer><span  class="dialog-footer">
      <el-button @click="customerDialogVisible = false">取 消</el-button>
      <el-button type="primary" @click="addRoles('addcustomerFormRef')">确 定</el-button>
    </span></template>
    </el-dialog>
    
    
    <el-dialog
      title="监护人编辑"
      v-model="oldEditDialogVisible"
      width="30%">
      <el-form label-width="80px" :rules="rules" :model="oldEditForm">
        <el-form-item label="姓名" prop="customerName">
        <el-input v-model="oldEditForm.customerName"></el-input>
        </el-form-item>
				<el-form-item label="姓名" prop="customerId" style="display: none;">
				<el-input v-model="oldEditForm.customerId"></el-input>
				</el-form-item>
        <el-form-item label="年龄" prop="customerAge">
        <el-input v-model="oldEditForm.customerAge"></el-input>
        </el-form-item>
        <el-form-item label="性别" prop="customerSex">
        <el-radio-group v-model="oldEditForm.customerSex">
        <el-radio value="男">男</el-radio>
        <el-radio value="女">女</el-radio>
        </el-radio-group>
        </el-form-item>
        <el-form-item label="电话" prop="customerPhone">
        <el-input v-model="oldEditForm.customerPhone"></el-input>
        </el-form-item>
        <el-form-item label="地址" prop="customerAddress">
        <el-input v-model="oldEditForm.customerAddress"></el-input>
        </el-form-item>
      </el-form>
      <template #footer><span  class="dialog-footer">
      <el-button @click="oldEditDialogVisible = false">取 消</el-button>
      <el-button type="primary" @click="editRole">确 定</el-button>
    </span></template>
    </el-dialog>
    
    
    <el-table class="adaptive-record-table" :data="customerList.slice((currentPage-1)*pageSize,currentPage*pageSize)"
    style="width: 100%" border>
    <template #empty><div  class="care-empty-state"><strong>暂无监护人信息</strong><span>添加监护人或调整搜索条件后，信息会显示在这里。</span></div></template>
    
      
      <el-table-column type="index" width="100" align="center"></el-table-column>
      <el-table-column prop="customerName" label="姓名" width="170" align="center"></el-table-column>
      <el-table-column prop="customerAge" label="年龄" width="170" align="center"></el-table-column>
	  <el-table-column prop="customerSex" label="性别" width="170" align="center"></el-table-column>
	  <el-table-column prop="customerPhone" label="电话" width="170" align="center"></el-table-column>
	  <el-table-column prop="customerAddress" label="地址" width="170" align="center"></el-table-column>
      <el-table-column label="操作" align="center">
        <template #default="scope" >
          <el-button size="small" plain type="primary" icon="el-icon-edit"
                     @click="handleEditRole(scope.$index, scope.row)">编辑
          </el-button>
          <el-button size="small" plain type="danger" icon="el-icon-delete" @click="deleteRole(scope.row.customerId)">删除
          </el-button>
          
        </template>
      </el-table-column>
    </el-table>
    <record-cards :records="customerList.slice((currentPage-1)*pageSize,currentPage*pageSize)" :fields='[{"key":"customerAge","label":"年龄"},{"key":"customerSex","label":"性别"},{"key":"customerPhone","label":"电话"},{"key":"customerAddress","label":"地址"}]' label="监护人资料" title-key="customerName" key-field="customerId">
      <template #actions="scope"  ><el-button size="small" plain type="primary" icon="el-icon-edit"
                     @click="handleEditRole(scope.$index, scope.row)">编辑
          </el-button>
<el-button size="small" plain type="danger" icon="el-icon-delete" @click="deleteRole(scope.row.customerId)">删除
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
       var checkAge = (rule, value, callback) => {
        if (!value) {
          return callback(new Error('年龄不能为空'));
        }
        setTimeout(() => {
          if (!Number.isInteger(+value)) {
            callback(new Error('请输入数字值'));
          } else {
            if (+value < 0) {
              callback(new Error('必须大于零'));
            } else if(+value > 150){
							callback(new Error('必须小于150'));
						} else{
              callback();
            }
          }
        }, 1000);
      }
    return {
      searchText: '', // 角色查询内容
      customerDialogVisible: false,
      oldEditDialogVisible: false,
      customerForm: {
        customerName: '',
        customerAge: '',
				customerSex: '',
				customerPhone: '',
				customerAddress: ''
      },
      oldEditForm: {
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
      customerList: [],
      // 控制分配权限对话框的展示和隐藏
      
      // 树形控件的数据
      

      // Tree树形控件，默认使用数据中的 label 属性作为节点名称，使用 children 属性指定该接待你的子节点
      // 但是，如果我们自己的数据结构与默认的结构不同，可以通过 defaultProps 来修改默认的两个属性
      
      // 当前被分配权限的角色id
      
			rules: {
						customerPhone: [
						  { validator: checkPhone, trigger: 'blur'}
						],
						customerAge: [
            { validator: checkAge, trigger: 'blur' }
          ]
					  }
    }
  },
  created () {
    this.searchCustomers()
  },
  methods: {
    // 查询
    searchCustomers () {
				var _this = this
				var customerName = _this.searchText
       
       axios.post('restful/customer/select',
         {
           'customerName': customerName === '' ? '' : _this.searchText
         },
         {
           headers: {
             'content-type': 'application/json'
           },
           withCredentials: true
         }).then(function (response) {
         _this.customerList = response.data.customers
         _this.total = response.data.customers.length
     console.log(_this.customerList)
       })
         .catch(function (error) {
           console.log(error)
         })
      
    },
    // 添加角色
    addRoles () {
      var _this = this
      axios.post('restful/customer/add',
        _this.customerForm,
        {
          headers: {
            'content-type': 'application/json'
          },
          withCredentials: true
        }).then(function (response) {
        console.log(response)
        if (response.data.returnCode === '1111') {
          _this.customerDialogVisible = false
          _this.searchCustomers()
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
      this.$refs.addcustomerFormRef.resetFields()
    },
    // 编辑
    editRole () {
      var _this = this
      axios.post('restful/customer/update', _this.oldEditForm,
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
          _this.searchCustomers()
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
    async deleteRole (customerId) {
      console.log(customerId)

      var _this = this
      try {
        await _this.$confirm('此操作将永久删除该文件, 是否继续?', '提示', {
          confirmButtonText: '确定',
          cancelButtonText: '取消',
          type: 'warning'
        })
        console.log(customerId)
        axios.post('restful/customer/delete', {'customerId': customerId},
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
          _this.searchCustomers()
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
    handleEditRole (index, row) {
      this.oldEditDialogVisible = true
      this.oldEditForm = Object.assign({}, row) // 解决修改弹出框数据时，表格数据会发生同步改变
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
