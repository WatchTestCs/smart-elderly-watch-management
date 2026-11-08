<template>
  <div class="roles" v-loading="isLoading" element-loading-text="正在加载监护人信息…" :aria-busy="isLoading">
    
    <div class="sys-research">
      <form class="row-search" role="search" @submit.prevent="searchGuardians()">
        <div class="search-field">
          
          <label class="table-search-label" for="search-guardian">监护人姓名</label>
          <el-input id="search-guardian"
            placeholder="按监护人姓名查找"
            v-model="searchText"
            class="input-with-select">
            <template #append><el-button aria-label="查询监护人姓名"
              
              :icon="Search"
              native-type="submit"></el-button></template>
          </el-input>
        </div>
        <div class="toolbar-actions">
          <el-button plain @click="exportCurrentPage">导出本页</el-button>
          <el-button plain @click="guardianDialogVisible = true">添加监护人</el-button>
        </div>
		
      </form>
    </div>
    
    <el-dialog :close-on-click-modal="false"
      title="监护人添加"
      v-model="guardianDialogVisible"
      @close="closeRoleAddDialog">
      <el-form label-width="80px" :model="guardianForm" :rules="rules"  ref="addGuardianFormRef">
        <el-form-item label="姓名" prop="guardianName">
          <el-input v-model="guardianForm.guardianName"></el-input>
        </el-form-item>
        <el-form-item label="年龄" prop="guardianAge">
          <el-input v-model="guardianForm.guardianAge"></el-input>
        </el-form-item>
		<el-form-item label="性别" prop="guardianSex">
		<el-radio-group v-model="guardianForm.guardianSex">
		  <el-radio value="男">男</el-radio>
		  <el-radio value="女">女</el-radio>
		</el-radio-group>
		</el-form-item>
		<el-form-item label="电话" prop="guardianPhone">
		<el-input v-model="guardianForm.guardianPhone"></el-input>
		</el-form-item>
		<el-form-item label="地址" prop="guardianAddress">
		<el-input v-model="guardianForm.guardianAddress"></el-input>
		</el-form-item>
		
      </el-form>
      <template #footer><span  class="dialog-footer">
      <el-button @click="guardianDialogVisible = false">取 消</el-button>
      <el-button type="primary" @click="addRoles()" :loading="saving">确 定</el-button>
    </span></template>
    </el-dialog>

    <el-dialog :close-on-click-modal="false"
      title="监护人编辑"
      v-model="oldEditDialogVisible">
      <el-form ref="editGuardianFormRef" label-width="80px" :rules="rules" :model="oldEditForm">
        <el-form-item label="姓名" prop="guardianName">
        <el-input v-model="oldEditForm.guardianName"></el-input>
        </el-form-item>
				<el-form-item label="姓名" prop="guardianId" style="display: none;">
				<el-input v-model="oldEditForm.guardianId"></el-input>
				</el-form-item>
        <el-form-item label="年龄" prop="guardianAge">
        <el-input v-model="oldEditForm.guardianAge"></el-input>
        </el-form-item>
        <el-form-item label="性别" prop="guardianSex">
        <el-radio-group v-model="oldEditForm.guardianSex">
        <el-radio value="男">男</el-radio>
        <el-radio value="女">女</el-radio>
        </el-radio-group>
        </el-form-item>
        <el-form-item label="电话" prop="guardianPhone">
        <el-input v-model="oldEditForm.guardianPhone"></el-input>
        </el-form-item>
        <el-form-item label="地址" prop="guardianAddress">
        <el-input v-model="oldEditForm.guardianAddress"></el-input>
        </el-form-item>
      </el-form>
      <template #footer><span  class="dialog-footer">
      <el-button @click="oldEditDialogVisible = false">取 消</el-button>
      <el-button type="primary" @click="editRole()" :loading="saving">确 定</el-button>
    </span></template>
    </el-dialog>

    <el-alert v-if="mutationError" :title="mutationError" type="error" :closable="false" show-icon />
    <div v-if="loadError" class="collection-feedback" role="alert"><span>{{ loadError }}</span><el-button plain :loading="isLoading" @click="searchGuardians(true)">重新加载</el-button></div>
    <el-table v-if="!mobileLayout && !loadError" class="adaptive-record-table" :data="guardianList"
    style="width: 100%" border>
    <template #empty><div  class="care-empty-state"><strong>暂无监护人信息</strong><span>添加监护人或调整搜索条件后，信息会显示在这里。</span></div></template>

      <el-table-column type="index" width="100" align="center"></el-table-column>
      <el-table-column prop="guardianName" label="姓名" width="170" align="center"></el-table-column>
      <el-table-column prop="guardianAge" label="年龄" width="170" align="center"></el-table-column>
	  <el-table-column prop="guardianSex" label="性别" width="170" align="center"></el-table-column>
	  <el-table-column prop="guardianPhone" label="电话" width="170" align="center"></el-table-column>
	  <el-table-column prop="guardianAddress" label="地址" width="170" align="center"></el-table-column>
      <el-table-column label="操作" align="center">
        <template #default="scope" >
          <el-button size="small" plain type="primary" :icon="Edit"
                     @click="handleEditRole(scope.$index, scope.row)">编辑
          </el-button>
          <el-button size="small" plain type="danger" :icon="Delete" @click="deleteRole(scope.row.guardianId)">删除
          </el-button>
          
        </template>
      </el-table-column>
    </el-table>
    <record-cards v-if="mobileLayout && !loadError" :records="guardianList" :fields='[{"key":"guardianAge","label":"年龄"},{"key":"guardianSex","label":"性别"},{"key":"guardianPhone","label":"电话"},{"key":"guardianAddress","label":"地址"}]' label="监护人资料" title-key="guardianName" key-field="guardianId">
      <template #actions="scope"  ><el-button size="small" plain type="primary" :icon="Edit"
                     @click="handleEditRole(scope.$index, scope.row)">编辑
          </el-button>
<el-button size="small" plain type="danger" :icon="Delete" @click="deleteRole(scope.row.guardianId)">删除
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

<script setup lang="ts">
import { Delete, Edit, Search } from '@element-plus/icons-vue'
import { useCollectionLayout } from '@/composables/useCollectionLayout'
import { onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox, type FormInstance } from 'element-plus'
import RecordCards from '@/components/RecordCards.vue'
import { api } from '../api/client'
import { downloadCsv } from '../utils/csv'
import { useGuardians, validAge, validPhone } from '../composables/useGuardians'
const mobileLayout = useCollectionLayout(900)
const state = useGuardians(api)
const { searchText, guardianDialogVisible, oldEditDialogVisible, guardianForm, oldEditForm, isLoading, saving, loadError, mutationError, total, pageSize, currentPage, guardianList, searchGuardians, handleEditRole, closeRoleAddDialog, handleSizeChange, changePage } = state
const addGuardianFormRef = ref<FormInstance>()
const editGuardianFormRef = ref<FormInstance>()
const rules = {
 guardianPhone: [{ validator: (_rule: unknown, value: unknown, done: (err?: Error) => void) => done(validPhone(value) ? undefined : new Error('请输入有效的 11 位手机号')), trigger: 'blur' }],
 guardianAge: [{ validator: (_rule: unknown, value: unknown, done: (err?: Error) => void) => done(validAge(value) ? undefined : new Error('请输入 0–150 岁的整数年龄')), trigger: 'blur' }]
}
async function addRoles() {
 if (!await addGuardianFormRef.value?.validate().catch(() => false)) return
 if (await state.save()) ElMessage.success('添加监护人成功')
}
async function editRole() {
 if (!await editGuardianFormRef.value?.validate().catch(() => false)) return
 if (await state.save(true)) ElMessage.success('编辑监护人信息成功')
}
async function deleteRole(id: number | string | undefined) {
 if (id === undefined) return
 try { await ElMessageBox.confirm('确认删除该监护人？', '删除监护人', { type: 'warning' }) } catch { return }
 if (await state.remove(id)) ElMessage.success('删除监护人成功')
}
function exportCurrentPage () {
  if (!guardianList.value.length) { ElMessage.warning('当前没有可导出的数据'); return }
  downloadCsv(`监护人-${new Date().toISOString().slice(0, 10)}.csv`, [{ key: 'guardianName', label: '姓名' }, { key: 'guardianAge', label: '年龄' }, { key: 'guardianSex', label: '性别' }, { key: 'guardianPhone', label: '电话' }, { key: 'guardianAddress', label: '地址' }], guardianList.value)
}
onMounted(() => searchGuardians())
</script>
