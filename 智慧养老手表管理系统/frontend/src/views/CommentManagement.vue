<template>
  <div class="roles" v-loading="isLoading" element-loading-text="正在加载评论…" :aria-busy="isLoading">

    <el-dialog :close-on-click-modal="false"
      title="评论回复"
      v-model="watchEditDialogVisible">
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

    <div v-if="loadError" class="collection-feedback" role="alert"><span>{{ loadError }}</span><el-button plain :loading="isLoading" @click="loadComments(true)">重新加载</el-button></div>
    <el-table v-if="!mobileLayout && !loadError" class="adaptive-record-table" :data="comments"
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
          <el-button v-if="scope.row.back==null" size="small" plain type="primary" :icon="Edit"
                     @click="handleeditSaid(scope.$index, scope.row)">回复
          </el-button>
          <el-button size="small" plain type="danger" :icon="Delete" @click="deleteSaid(scope.row.saidId)">删除
          </el-button>
          
        </template>
      </el-table-column>
    </el-table>
    <record-cards v-if="mobileLayout && !loadError" :records="comments" :fields='[{"key":"saidId","label":"作者"},{"key":"said","label":"评论"},{"key":"back","label":"回复"},{"key":"backTime","label":"回复时间"},{"key":"title","label":"文章标题"}]' label="公告评论" title-key="personName" key-field="saidId" time-key="saidTime">
      <template #actions="scope"  ><el-button v-if="scope.row.back==null" size="small" plain type="primary" :icon="Edit"
                     @click="handleeditSaid(scope.$index, scope.row)">回复
          </el-button>
<el-button size="small" plain type="danger" :icon="Delete" @click="deleteSaid(scope.row.saidId)">删除
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
import { Delete, Edit } from '@element-plus/icons-vue'
import { ref } from 'vue'
import { usePagedRecords } from '@/composables/usePagedRecords'
import { writeRecord, deleteRecord } from '@/composables/recordActions'
import { useCollectionLayout } from '@/composables/useCollectionLayout'
import RecordCards from '@/components/RecordCards.vue'

type CommentRow = Record<string, any>
const mobileLayout = useCollectionLayout(900)
const {
  records: comments, isLoading, loadError, total,
  pageSize, currentPage, load: loadComments,
  handleSizeChange, changePage
} = usePagedRecords<CommentRow>({
  path: 'api/v1/notice-comments/search',
  responseKey: 'moderatedComments',
  failure: '评论暂时无法加载，请检查服务连接后重试。'
})
const watchEditDialogVisible = ref(false)
const saidEditForm = ref<CommentRow>({ saidId: '', said: '', back: '' })
function editSaid() {
  return writeRecord('api/v1/notice-comments/reply', saidEditForm.value, '回复已保存', async () => {
    watchEditDialogVisible.value = false
    await loadComments(true)
  })
}
function deleteSaid(saidId: number | string) {
  return deleteRecord('api/v1/notice-comments/delete', { saidId },
    '确认删除该评论？', () => loadComments(true))
}
function handleeditSaid(_index: number, row: CommentRow) {
  saidEditForm.value = { ...row }
  watchEditDialogVisible.value = true
}
</script>
