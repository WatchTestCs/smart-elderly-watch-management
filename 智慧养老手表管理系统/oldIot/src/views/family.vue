<template>
  <div class="roles">
    
    <div class="sys-research">
      <form class="row-search" role="search" @submit.prevent="searchfamily">
        <div class="search-field">
          
          <label class="table-search-label" for="search-family">家属姓名</label>
          <el-input id="search-family"
            placeholder="按家属姓名查找"
            v-model="searchText"
            class="input-with-select">
            <template #append><el-button aria-label="查询家属姓名"
              
              icon="el-icon-search"
              native-type="submit"></el-button></template>
          </el-input>
        </div>
        <div class="toolbar-actions">
          <el-button plain @click="customerDialogVisible = true">添加家属</el-button>
        </div>
		
      </form>
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

    <el-table class="adaptive-record-table" :data="familyList"
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
    <record-cards :records="familyList" :fields='[{"key":"name","label":"姓名"},{"key":"oldId","label":"老人编号"},{"key":"familyRelation","label":"关系"},{"key":"familySex","label":"性别"},{"key":"familyPhone","label":"电话"},{"key":"familyAddress","label":"地址"}]' label="家属账号资料" title-key="familyName" key-field="familyId">
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
      familyEditForm: {},
      
      total: 0,
      pageSize: 10,
      currentPage: 1,
      familyList: [],

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
    searchfamily (keepPage = false) {
      if (keepPage !== true) this.currentPage = 1
     var _this = this
       var familyName = _this.searchText
       
       axios.post('restful/family/select',
         { page: this.currentPage, pageSize: this.pageSize,
           'familyName': familyName === '' ? '' : _this.searchText
         },
         {
           headers: {
             'content-type': 'application/json'
           },
           withCredentials: true
         }).then(function (response) {
         _this.familyList = response.data.familyDtos
         _this.total = response.data.total ?? response.data.familyDtos.length

       })
         .catch(function (error) {
           console.log(error)
         })
      
    },
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
    closeRoleAddDialog () {
      this.$refs.addfamilyFormRef.resetFields()
    },
    editFamily () {
      var _this = this
      axios.post('restful/family/updateFamily', _this.familyEditForm,
        {
          headers: {
            'content-type': 'application/json'
          },
          withCredentials: true
        }).then(function (response) {

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
    async deleteFamily (familyId) {

      var _this = this
      try {
        await _this.$confirm('此操作将永久删除该文件, 是否继续?', '提示', {
          confirmButtonText: '确定',
          cancelButtonText: '取消',
          type: 'warning'
        })

        axios.post('restful/family/deleteFamily', {'familyId': familyId},
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
          _this.searchfamily()
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
    handleeditFamily (index, row) {
      this.oldEditDialogVisible = true
      this.familyEditForm = Object.assign({}, row) // 解决修改弹出框数据时，表格数据会发生同步改变

    },
    handleSizeChange (val) { this.pageSize = val; this.currentPage = 1; this.searchfamily(true)
    },

    changePage (val) { this.currentPage = val; this.searchfamily(true)
    },


  }
}

</script>
