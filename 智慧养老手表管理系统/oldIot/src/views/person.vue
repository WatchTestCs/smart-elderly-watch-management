<template>
  <div class="roles">
    
    <div class="sys-research">
      <form class="row-search" role="search" @submit.prevent="searchOldPersons">
        <div class="search-field">
          
          <label class="table-search-label" for="search-person">长者姓名</label>
          <el-input id="search-person"
            placeholder="按长者姓名查找"
            v-model="searchText"
            class="input-with-select">
            <template #append><el-button aria-label="查询长者姓名"
              
              icon="el-icon-search"
              native-type="submit"></el-button></template>
          </el-input>
        </div>
        <div class="toolbar-actions">
          <el-button plain @click="oldDialogVisible = true">添加老人</el-button>
        </div>
		
      </form>
    </div>
    
    <el-dialog
      title="老人添加"
      v-model="oldDialogVisible"
      @close="closeRoleAddDialog"
      width="30%">
      <el-form label-width="80px" :model="oldForm" :rules="rules"  ref="addoldFormRef">
        <el-form-item label="姓名" prop="name">
          <el-input v-model="oldForm.name"></el-input>
        </el-form-item>
        
		<el-form-item label="性别" prop="sex">
		<el-radio-group v-model="oldForm.sex">
		  <el-radio value="男">男</el-radio>
		  <el-radio value="女">女</el-radio>
		</el-radio-group>
		</el-form-item>
		<el-form-item label="电话" prop="phone">
		<el-input v-model="oldForm.phone"></el-input>
		</el-form-item>
		<el-form-item label="房间" prop="room">
		<el-input v-model="oldForm.room"></el-input>
		</el-form-item>
		<el-form-item label="病史" prop="illness">
		<el-input v-model="oldForm.illness"></el-input>
		</el-form-item>
		<el-form-item label="过敏药物" prop="warn">
		<el-input v-model="oldForm.warn"></el-input>
		</el-form-item>
		<el-form-item label="喜好食物" prop="foodLike">
		<el-input v-model="oldForm.foodLike"></el-input>
		</el-form-item>
		<el-form-item label="忌口食物" prop="foodHate">
		<el-input v-model="oldForm.foodHate"></el-input>
		</el-form-item>
		<el-form-item label="常用药物" prop="drag">
		<el-input v-model="oldForm.drag"></el-input>
		</el-form-item>
		<el-form-item label="出生日期" prop="bothdate">

		  <el-date-picker type="date" placeholder="选择日期" value-format="YYYY-MM-DD" v-model="oldForm.bothdate" style="width: 100%;"></el-date-picker>

		</el-form-item>
		<el-form-item label="监护人" prop="customerId">

				<el-select v-model="oldForm.customerId" placeholder="请选择监护人">
				<el-option v-for="item in options"
      :key="item.customerId"
      :label="item.customerName"
      :value="item.customerId">
			</el-option>
			</el-select>			

		
		</el-form-item>
			<el-form-item label="手表编号" prop="deviceId">
			<el-input v-model="oldForm.deviceId"></el-input>
			</el-form-item>
      </el-form>
      <template #footer><span  class="dialog-footer">
      <el-button @click="oldDialogVisible = false">取 消</el-button>
      <el-button type="primary" @click="addOldPersons('addoldFormRef')">确 定</el-button>
    </span></template>
    </el-dialog>

    <el-dialog
      title="角色编辑"
      v-model="oldEditDialogVisible"
      width="30%">
      <el-form label-width="80px" :rules="rules" :model="oldEditForm">
        		<el-form-item label="姓名" prop="name">
        			<el-input v-model="oldEditForm.name"></el-input>
        		</el-form-item>
        		<el-form-item label="年龄" prop="oldId" style="display: none;">
        			<el-input v-model="oldEditForm.oldId"></el-input>
        		</el-form-item>
        <el-form-item label="性别" prop="sex">
        <el-radio-group v-model="oldEditForm.sex">
        	<el-radio value="男">男</el-radio>
        	<el-radio value="女">女</el-radio>
        </el-radio-group>
        </el-form-item>
        <el-form-item label="电话" prop="phone">
        <el-input v-model="oldEditForm.phone"></el-input>
        </el-form-item>
        <el-form-item label="房间" prop="room">
        <el-input v-model="oldEditForm.room"></el-input>
        </el-form-item>
        <el-form-item label="病史" prop="illness">
        <el-input v-model="oldEditForm.illness"></el-input>
        </el-form-item>
        <el-form-item label="过敏药物" prop="warn">
        <el-input v-model="oldEditForm.warn"></el-input>
        </el-form-item>
        <el-form-item label="喜好食物" prop="foodLike">
        <el-input v-model="oldEditForm.foodLike"></el-input>
        </el-form-item>
        <el-form-item label="忌口食物" prop="foodHate">
        <el-input v-model="oldEditForm.foodHate"></el-input>
        </el-form-item>
        <el-form-item label="常用药物" prop="drag">
        <el-input v-model="oldEditForm.drag"></el-input>
        </el-form-item>
        <el-form-item label="出生日期" prop="bothdate">

        	<el-date-picker type="date" placeholder="选择日期" value-format="YYYY-MM-DD" v-model="oldEditForm.bothdate" style="width: 100%;"></el-date-picker>

        </el-form-item>
        <el-form-item label="监护人" prop="customerId">

        		<el-select v-model="oldEditForm.customerName" placeholder="请选择监护人">
        		<el-option v-for="item in options"
      :key="item.customerId"
      :label="item.customerName"
      :value="item.customerId">
			</el-option>
        		
        	</el-select>			

        
        </el-form-item>
				<el-form-item label="手表编号" prop="deviceId">
				<el-input v-model="oldEditForm.deviceId"></el-input>
				</el-form-item>
      </el-form>
      <template #footer><span  class="dialog-footer">
      <el-button @click="oldEditDialogVisible = false">取 消</el-button>
      <el-button type="primary" @click="editOldPerson">确 定</el-button>
    </span></template>
    </el-dialog>

    <el-table class="adaptive-record-table" :data="personDetailList"
    style="width: 100%" border>
    <template #empty><div  class="care-empty-state"><strong>暂无老人档案</strong><span>添加老人或调整搜索条件后，档案会显示在这里。</span></div></template>

	<el-table-column type="expand">
			  <template #default="prop" >
				<p>家属姓名:{{prop.row.familyName}}</p><p>家属性别:{{prop.row.familySex}}</p>
				<p>关系:{{prop.row.familyRelation}}</p><p>家属电话:{{prop.row.familyPhone}}</p>
				<p>家属地址:{{prop.row.familyAddress}}</p><p>过敏:{{prop.row.warn}}</p>
				<p>喜好食物:{{prop.row.foodLike}}</p><p>忌口食物:{{prop.row.foodHate}}</p>
				<p>常用药物:{{prop.row.drag}}</p><p>病史:{{prop.row.illness}}</p>
				
			  </template>
	  
	</el-table-column>
      
      <el-table-column type="index" width="100" align="center"></el-table-column>
      <el-table-column prop="name" label="姓名" width="100" align="center"></el-table-column>
      <el-table-column prop="bothdate" label="出生日期" width="120" align="center"></el-table-column>
	  <el-table-column prop="sex" label="性别" width="100" align="center"></el-table-column>
	  <el-table-column prop="phone" label="电话" width="120" align="center"></el-table-column>
	  <el-table-column prop="room" label="房间" width="100" align="center"></el-table-column>
	  <el-table-column prop="beTime" label="入院时间" width="100" align="center"></el-table-column>
	  
		<el-table-column prop="customerName" label="监护人" width="100" align="center"></el-table-column>
		<el-table-column prop="deviceId" label="手表编号" width="100" align="center"></el-table-column>

      <el-table-column label="操作" align="center">
        <template #default="scope" >
          <el-button size="small" plain type="primary" icon="el-icon-edit"
                     @click="handleeditOldPerson(scope.$index, scope.row)">编辑
          </el-button>
          <el-button size="small" plain type="danger" icon="el-icon-delete" @click="deleteOldPerson(scope.row.oldId)">删除
          </el-button>
          
        </template>
      </el-table-column>
    </el-table>
    <record-cards :records="personDetailList" :fields='[{"key":"bothdate","label":"出生日期"},{"key":"sex","label":"性别"},{"key":"phone","label":"电话"},{"key":"room","label":"房间"},{"key":"beTime","label":"入院时间"},{"key":"familyName","label":"家属姓名"},{"key":"familyRelation","label":"关系"},{"key":"familyPhone","label":"家属电话"},{"key":"familySex","label":"家属性别"},{"key":"customerName","label":"监护人"},{"key":"deviceId","label":"手表编号"},{"key":"illness","label":"病史"},{"key":"warn","label":"过敏"},{"key":"foodLike","label":"喜好食物"},{"key":"foodHate","label":"忌口食物"},{"key":"drag","label":"常用药物"}]' label="长者档案" title-key="name" key-field="oldId">
      <template #actions="scope"  ><el-button size="small" plain type="primary" icon="el-icon-edit"
                     @click="handleeditOldPerson(scope.$index, scope.row)">编辑
          </el-button>
<el-button size="small" plain type="danger" icon="el-icon-delete" @click="deleteOldPerson(scope.row.oldId)">删除
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
      oldDialogVisible: false,
      oldEditDialogVisible: false,
      oldForm: {
        name: '',
        bothdate: '',
		sex: '',
		phone: '',
		room: '',

		beTime: '',
		customerId: '',
		illness: '',
		warn: '',
		foodLike: '',
		foodHate: '',
		drag: '',
		deviceId: ''
      },
      oldEditForm: {
       		name: '',
       		bothdate: '',
			oldId: '',
       sex: '',
       phone: '',
       room: '',

       beTime: '',
       customerId: '',
			 customerName: '',
			 illness: '',
			 warn: '',
			 foodLike: '',
			 foodHate: '',
			 drag: '',
			 deviceId: ''
      },
			options:[],
      
      total: 0,
      pageSize: 10,
      currentPage: 1,
      personDetailList: [],

	  rules: {
	  			phone: [
	  			  { validator: checkPhone, trigger: 'blur'}
	  			]
	  		  }
    }
  },
  created () {
    this.searchOldPersons()
		this.addCustomerList()
  },
  methods: {
    searchOldPersons (keepPage = false) {
      if (keepPage !== true) this.currentPage = 1
      var _this = this
      var name = _this.searchText
      
      axios.post('restful/oldPerson/selectDetail',
        { page: this.currentPage, pageSize: this.pageSize,
          'name': name === '' ? '' : _this.searchText
        },
        {
          headers: {
            'content-type': 'application/json'
          },
          withCredentials: true
        }).then(function (response) {
        _this.personDetailList = response.data.oldPersonDetails
        _this.total = response.data.total ?? response.data.oldPersonDetails.length

      })
        .catch(function (error) {
          console.log(error)
        })
      
    },
    addOldPersons () {
      var _this = this
	  var param = _this.oldForm
      axios.post('restful/oldPerson/add',
        param,
        {
          headers: {
            'content-type': 'application/json'
          },
          withCredentials: true
        }).then(function (response) {

        if (response.data.returnCode === '1111') {
          _this.oldDialogVisible = false
          _this.searchOldPersons()
          _this.$message({
            type: 'success',
            message: '添加老人成功'
          })
        } else {
          _this.$message({
            type: 'error',
            message: '添加老人失败'
          })
        }
      })
        .catch(function (error) {
          console.log(error)
        })
    },
    closeRoleAddDialog () {
      this.$refs.addoldFormRef.resetFields()
    },
    editOldPerson () {
			this.addCustomerList()
      var _this = this
      axios.post('restful/oldPerson/update', _this.oldEditForm,
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
            message: '编辑老人信息成功'
          })
          _this.searchOldPersons()
        } else {
          _this.$message({
            type: 'error',
            message: '编辑老人信息失败，手表ID重复或没有手表设备'
          })
        }
      })
        .catch(function (error) {
          console.log(error)
        })
    },
		addCustomerList(){
			var _this = this
			  
			  axios.post('restful/customer/select',
			    {
			      'customerName': ""
			    },
			    {
			      headers: {
			        'content-type': 'application/json'
			      },
			      withCredentials: true
			    }).then(function (response) {
			    _this.options = response.data.customers
			  })
			    .catch(function (error) {
			      console.log(error)
			    })
			 
		},
    async deleteOldPerson (oldId) {

      var _this = this
      try {
        await _this.$confirm('此操作将永久删除该文件, 是否继续?', '提示', {
          confirmButtonText: '确定',
          cancelButtonText: '取消',
          type: 'warning'
        })

        axios.post('restful/oldPerson/delete', {'oldId': oldId},
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
          _this.searchOldPersons()
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
    handleeditOldPerson (index, row) {
      this.oldEditDialogVisible = true
      this.oldEditForm = Object.assign({}, row) // 解决修改弹出框数据时，表格数据会发生同步改变

    },
    handleSizeChange (val) { this.pageSize = val; this.currentPage = 1; this.searchOldPersons(true)
    },

    changePage (val) { this.currentPage = val; this.searchOldPersons(true)
    },


  }
}

</script>
