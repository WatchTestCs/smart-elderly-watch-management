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
    
    <el-dialog :close-on-click-modal="false"
      title="设备添加"
      v-model="watchDialogVisible"
      @close="closeRoleAddDialog">
      <ThresholdForm :draft="watchForm" :rules="rules" ref="addwatchFormRef" />
      <template #footer><span  class="dialog-footer">
      <el-button @click="watchDialogVisible = false">取 消</el-button>
      <el-button type="primary" @click="addWatchs('addwatchFormRef')">确 定</el-button>
    </span></template>
    </el-dialog>

    <el-dialog :close-on-click-modal="false"
      title="设备编辑"
      v-model="watchEditDialogVisible">
      <ThresholdForm editing :draft="watchEditForm" :rules="rules" ref="editwatchFormRef" />
      <template #footer><span  class="dialog-footer">
      <el-button @click="watchEditDialogVisible = false">取 消</el-button>
      <el-button type="primary" @click="editWatch">确 定</el-button>
    </span></template>
    </el-dialog>

    <div v-if="loadError" class="collection-feedback" role="alert"><span>{{ loadError }}</span><el-button plain :loading="isLoading" @click="searchWatchs(true)">重新加载</el-button></div>
    <el-table v-if="!mobileLayout && !loadError" class="adaptive-record-table" :data="watchsList"
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
    <record-cards v-if="mobileLayout && !loadError" :records="watchsList" :fields='[{"key":"deviceId","label":"手表ID"},{"key":"bpf","label":"血压高值"},{"key":"bpl","label":"血压低值"},{"key":"hrf","label":"心率高值"},{"key":"hrl","label":"心率低值"}]' label="监测阈值档案" title-key="name" key-field="deviceId" time-key="opTime">
      <template #actions="scope"  ><el-button size="small" plain type="primary" icon="el-icon-edit"
                     @click="handleEditwatch(scope.$index, scope.row)">编辑
          </el-button>
<el-button size="small" plain type="danger" icon="el-icon-delete" @click="deleteRole(scope.row.deviceId)">删除
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

import { checkThreshold } from './watch.validation'
import RecordCards from '@/components/RecordCards.vue'
import ThresholdForm from '@/components/records/ThresholdForm.vue'

export default {
  mixins: [pagedRecords({ path: "api/v1/monitoring-profiles/list", responseKey: "watchDtos", recordsKey: "watchsList", searchKey: "oldName", load: "searchWatchs", failure: "监测阈值档案暂时无法加载，请检查服务连接后重试。" })],
  setup () { return { mobileLayout: useCollectionLayout(900) } },
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


    }
  },
  created () {
    this.searchWatchs()
  },
  methods: {
    addWatchs () { return this.saveProfile(false) },
    closeRoleAddDialog () {
      this.$refs.addwatchFormRef.resetFields()
    },
    editWatch () { return this.saveProfile(true) },
    saveProfile (editing) {
      const form = this.$refs[editing ? 'editwatchFormRef' : 'addwatchFormRef']
      form.validate(valid => {
        if (!valid) return
        const path = editing ? 'api/v1/monitoring-profiles/update' : 'api/v1/monitoring-profiles/create'
        const body = editing ? this.watchEditForm : this.watchForm
        return writeRecord(this, path, body, '监测档案已保存', () => {
          if (editing) this.watchEditDialogVisible = false
          else this.watchDialogVisible = false
          this.searchWatchs()
        })
      })
    },
    deleteRole (deviceId) {
      return deleteRecord(this, 'api/v1/monitoring-profiles/delete', { deviceId }, '确认删除该监测档案？', () => this.searchWatchs())
    },
    handleEditwatch (index, row) {
      this.watchEditDialogVisible = true
      this.watchEditForm = Object.fromEntries(['deviceId', 'bpf', 'bpl', 'hrf', 'hrl'].map(key => [key, row[key]])) // 解决修改弹出框数据时，表格数据会发生同步改变

    },



  }
}

</script>
