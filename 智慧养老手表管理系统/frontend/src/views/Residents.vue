<template>
  <div class="roles" v-loading="isLoading" element-loading-text="正在加载老人档案…" :aria-busy="String(isLoading)">
    
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
    
    <el-dialog :close-on-click-modal="false"
      title="老人添加"
      v-model="oldDialogVisible"
      @open="addCustomerList"
      @close="closeRoleAddDialog">
      <ResidentForm :draft="oldForm" :rules="rules" ref="addoldFormRef" :options="options" :customer-loading="customerLoading" :customer-error="customerError" @retry-guardians="addCustomerList" />
      <template #footer><span  class="dialog-footer">
      <el-button @click="oldDialogVisible = false">取 消</el-button>
      <el-button type="primary" @click="addOldPersons('addoldFormRef')">确 定</el-button>
    </span></template>
    </el-dialog>

    <el-dialog :close-on-click-modal="false"
      title="老人编辑"
      v-model="oldEditDialogVisible"
      @open="addCustomerList">
      <ResidentForm editing :draft="oldEditForm" :rules="rules" ref="editoldFormRef" :options="options" :customer-loading="customerLoading" :customer-error="customerError" @retry-guardians="addCustomerList" />
      <template #footer><span  class="dialog-footer">
      <el-button @click="oldEditDialogVisible = false">取 消</el-button>
      <el-button type="primary" @click="editOldPerson">确 定</el-button>
    </span></template>
    </el-dialog>

    <div v-if="loadError" class="collection-feedback" role="alert"><span>{{ loadError }}</span><el-button plain :loading="isLoading" @click="searchOldPersons(true)">重新加载</el-button></div>
    <el-table v-if="!mobileLayout && !loadError" class="adaptive-record-table" :data="personDetailList"
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

      <el-table-column label="操作" width="200" align="center" fixed="right">
        <template #default="scope" >
          <el-button size="small" plain type="primary" icon="el-icon-edit"
                     @click="handleeditOldPerson(scope.$index, scope.row)">编辑
          </el-button>
          <el-button size="small" plain type="danger" icon="el-icon-delete" @click="deleteOldPerson(scope.row.oldId)">删除
          </el-button>
          
        </template>
      </el-table-column>
    </el-table>
    <record-cards v-if="mobileLayout && !loadError" :records="personDetailList" :fields='[{"key":"bothdate","label":"出生日期"},{"key":"sex","label":"性别"},{"key":"phone","label":"电话"},{"key":"room","label":"房间"},{"key":"beTime","label":"入院时间"},{"key":"familyName","label":"家属姓名"},{"key":"familyRelation","label":"关系"},{"key":"familyPhone","label":"家属电话"},{"key":"familyAddress","label":"家属地址"},{"key":"familySex","label":"家属性别"},{"key":"customerName","label":"监护人"},{"key":"deviceId","label":"手表编号"},{"key":"illness","label":"病史"},{"key":"warn","label":"过敏"},{"key":"foodLike","label":"喜好食物"},{"key":"foodHate","label":"忌口食物"},{"key":"drag","label":"常用药物"}]' label="老人档案" empty-title="暂无老人档案" title-key="name" key-field="oldId">
      <template #actions="scope"  ><el-button size="small" plain type="primary" icon="el-icon-edit"
                     @click="handleeditOldPerson(scope.$index, scope.row)">编辑
          </el-button>
<el-button size="small" plain type="danger" icon="el-icon-delete" @click="deleteOldPerson(scope.row.oldId)">删除
          </el-button></template>
    </record-cards>
    
    <div v-if="!loadError" style="margin-top: 10px;text-align: center">
      <pager-bar
        @size-change="handleSizeChange"
        @current-change="changePage"
        v-model:current-page="currentPage"
        :page-size="pageSize"
        :total="total" />
    </div>
  </div>

</template>

<script>
import { pagedRecords } from '@/composables/pagedRecords'
import { api as requestClient } from '../api/client'
import { useCollectionLayout } from '@/composables/useCollectionLayout'
import { apiErrorMessage } from '../api/httpContract'

import { residentCommandMethods } from './person.command-methods'
import RecordCards from '@/components/RecordCards.vue'
import ResidentForm from '@/components/records/ResidentForm.vue'

export default {
  mixins: [pagedRecords({ path: "api/v1/residents/details", responseKey: "oldPersonDetails", recordsKey: "personDetailList", searchKey: "name", load: "searchOldPersons", failure: "老人档案暂时无法加载，请检查服务连接后重试。" })],
  setup () { return { mobileLayout: useCollectionLayout(900) } },
  components: { RecordCards, ResidentForm },
  data () {
    var checkPhone = (rule, value, callback) => {
      const phoneReg = /^1[3|4|5|7|8][0-9]{9}$/
      if (!value) {
        return callback(new Error('电话号码不能为空'))
      }
      if (!Number.isInteger(+value)) {
        callback(new Error('请输入数字值'))
      } else {
        if (phoneReg.test(value)) {
          callback()
        } else {
          callback(new Error('电话号码格式不正确'))
        }
      }
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
    drag: ''
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
      customerLoading: false,
      customerError: '',


    rules: {
          phone: [
            { validator: checkPhone, trigger: 'blur'}
          ]
          }
    }
  },
  created () {
    this.searchOldPersons()
  },
  methods: {
    ...residentCommandMethods,
    async addCustomerList () {
      if (this.customerLoading) return
      this.customerLoading = true
      this.customerError = ''
      try {
        var options = [], page = 1, total = 0
        do {
          var response = await requestClient.post('api/v1/guardians/list', { customerName: '', page, pageSize: 100 })
          var items = response.data.customers
          if (!Array.isArray(items)) throw new Error('invalid guardians')
          total = response.data.total ?? items.length
          if (!Number.isInteger(total) || total < 0 || (!items.length && options.length < total)) throw new Error('incomplete guardians')
          options.push(...items)
          page += 1
        } while (options.length < total)
        this.options = options
      } catch (error) {
        this.options = []
        this.customerError = apiErrorMessage(error, '监护人选项暂时无法加载，请重试。')
      } finally { this.customerLoading = false }
    },
    handleeditOldPerson (index, row) {
      this.oldEditDialogVisible = true
      this.oldEditForm = Object.assign({}, row) // 解决修改弹出框数据时，表格数据会发生同步改变

    },



  }
}
</script>
