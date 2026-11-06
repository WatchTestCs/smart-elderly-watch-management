<template>
  <div class="roles" v-loading="isLoading" element-loading-text="正在加载家属账号…" :aria-busy="String(isLoading)">
    
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
          <el-button plain @click="exportCurrentPage">导出本页</el-button>
          <el-button plain @click="customerDialogVisible = true">添加家属</el-button>
        </div>
		
      </form>
    </div>
    
    <el-dialog :close-on-click-modal="false"
      title="家属添加"
      v-model="customerDialogVisible"
      @close="closeRoleAddDialog"
      width="92%" custom-class="admin-record-dialog">
      <FamilyForm :draft="familyForm" :rules="rules" ref="addfamilyFormRef" />
      <template #footer><span  class="dialog-footer">
      <el-button @click="customerDialogVisible = false">取 消</el-button>
      <el-button type="primary" @click="addFamily('addfamilyFormRef')">确 定</el-button>
    </span></template>
    </el-dialog>

    <el-dialog :close-on-click-modal="false"
      title="家属编辑"
      v-model="oldEditDialogVisible"
      width="92%" custom-class="admin-record-dialog">
      <FamilyForm editing :draft="familyEditForm" :rules="rules" />
      <template #footer><span  class="dialog-footer">
      <el-button @click="oldEditDialogVisible = false">取 消</el-button>
      <el-button type="primary" @click="editFamily">确 定</el-button>
    </span></template>
    </el-dialog>

    <div v-if="loadError" class="collection-feedback" role="alert"><span>{{ loadError }}</span><el-button plain :loading="isLoading" @click="searchfamily(true)">重新加载</el-button></div>
    <el-table v-if="!mobileLayout && !loadError" class="adaptive-record-table" :data="familyList"
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
    <record-cards v-if="mobileLayout && !loadError" :records="familyList" :fields='[{"key":"name","label":"姓名"},{"key":"oldId","label":"老人编号"},{"key":"familyRelation","label":"关系"},{"key":"familySex","label":"性别"},{"key":"familyPhone","label":"电话"},{"key":"familyAddress","label":"地址"}]' label="家属账号资料" title-key="familyName" key-field="familyId">
      <template #actions="scope"  ><el-button size="small" plain type="primary" icon="el-icon-edit"
                     @click="handleeditFamily(scope.$index, scope.row)">编辑
          </el-button>
<el-button size="small" plain type="danger" icon="el-icon-delete" @click="deleteFamily(scope.row.familyId)">删除
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
import { deleteRecord, writeRecord } from '@/composables/recordActions'
import { pagedRecords } from '@/composables/pagedRecords'
import { useCollectionLayout } from '@/composables/useCollectionLayout'

import RecordCards from '@/components/RecordCards.vue'
import FamilyForm from '@/components/records/FamilyForm.vue'

export default {
  mixins: [pagedRecords({ path: "api/v1/family/select", responseKey: "familyDtos", recordsKey: "familyList", searchKey: "familyName", load: "searchfamily", failure: "家属账号暂时无法加载，请检查服务连接后重试。" , exportName: "家属账号", exportColumns: [{key:"name",label:"老人姓名"},{key:"oldId",label:"老人编号"},{key:"familyName",label:"家属姓名"},{key:"familyRelation",label:"关系"},{key:"familySex",label:"性别"},{key:"familyPhone",label:"电话"},{key:"familyAddress",label:"地址"}] })],
  setup () { return { mobileLayout: useCollectionLayout(900) } },
  components: { RecordCards, FamilyForm },
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
    addFamily () {
      return writeRecord(this, 'api/v1/family/add', this.familyForm, '家属已添加，请生成该老人的邀请码，由家属注册并设置登录密码。', () => {
        this.customerDialogVisible = false
        this.searchfamily()
      }, '添加家属失败，请检查服务连接后重试。')
    },
    closeRoleAddDialog () {
      this.$refs.addfamilyFormRef.resetFields()
    },
    editFamily () {
      const fields = ['familyId', 'familyName', 'familyRelation', 'familySex', 'familyPhone', 'familyAddress']
      const body = Object.fromEntries(fields.map(key => [key, this.familyEditForm[key]]))
      return writeRecord(this, 'api/v1/family/updateFamily', body, '家属信息已保存', () => {
        this.oldEditDialogVisible = false
        this.searchfamily(true)
      })
    },
    deleteFamily (familyId) {
      return deleteRecord(this, 'api/v1/family/deleteFamily', { familyId }, '确认删除该家属账号？', () => this.searchfamily(true))
    },
    handleeditFamily (index, row) {
      this.oldEditDialogVisible = true
      this.familyEditForm = Object.assign({}, row) // 解决修改弹出框数据时，表格数据会发生同步改变

    },



  }
}
</script>
