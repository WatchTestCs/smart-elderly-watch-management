<template>
  <div class="announcement-admin">
    <section class="announcement-list-panel">
      <header class="announcement-toolbar">
        <div><h2>公告动态</h2><p>查看、检索并维护已发布的公告。</p></div>
        <div class="announcement-actions">
          <form class="announcement-query" role="search" @submit.prevent="searchOlds"><label class="table-search-label" for="manage-notice-search">按标题查找</label><el-input id="manage-notice-search" v-model="searchText" placeholder="搜索公告标题" clearable @clear="searchOlds">
            <template #append><el-button  icon="el-icon-search" aria-label="搜索公告" native-type="submit"></el-button></template>
          </el-input></form>
          <el-button type="primary" icon="el-icon-plus" @click="addtext">添加公告</el-button>
        </div>
      </header>
      <div v-if="loadError" class="collection-feedback" role="alert">{{ loadError }}<el-button plain @click="searchOlds">重新加载公告</el-button></div>
      <el-table class="managed-notices-desktop" v-loading="isLoading" :data="tableData" @row-dblclick="getDetail" row-class-name="clickable-row">
        <template #empty><div  class="care-empty-state"><strong>暂无已发布公告</strong><span>发布后的公告会显示在这里。</span></div></template>
        <el-table-column type="index" label="序号" width="80" align="center"></el-table-column>
        <el-table-column prop="title" label="公告标题" min-width="260" show-overflow-tooltip></el-table-column>
        <el-table-column prop="time" label="发布时间" width="190" align="center"></el-table-column>
        <el-table-column label="操作" width="180" align="center">
          <template #default="scope" >
            <el-button size="small" @click="handleEdit(scope.$index, scope.row)">编辑</el-button>
            <el-button size="small" type="danger" :disabled="deletingId !== null" @click="handleDelete(scope.$index, scope.row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
      <div class="managed-notices-mobile" v-loading="isLoading"><article v-for="item in tableData" :key="item.textId" class="managed-notice"><h3>{{ item.title || '未命名公告' }}</h3><p>{{ item.time || '暂无发布时间' }}</p><div><el-button plain @click="getDetail(item)">阅读</el-button><el-button type="primary" plain @click="handleEdit(0, item)">编辑</el-button><el-button type="danger" plain :disabled="deletingId !== null" @click="handleDelete(0, item)">删除</el-button></div></article><div v-if="!tableData.length" class="collection-empty"><strong>{{ isLoading ? '正在加载公告…' : loadError ? '公告暂时无法加载' : '暂无匹配的公告' }}</strong><p>您可以调整关键词，或发布一条新的公告。</p></div></div>
      <div v-if="total > pageSize" class="pagination-wrap">
        <el-pagination @size-change="handleSizeChange" @current-change="changePage" v-model:current-page="currentPage" :page-size="pageSize" layout="total, prev, pager, next" :pager-count="5" :total="total"></el-pagination>
      </div>
    </section>
    <el-dialog title="展示图片预览" v-model="previewVisible"><img class="gallery-preview" :src="previewUrl" alt="选中的展示图片"></el-dialog>

    <details class="upload-panel"><summary class="image-maintenance-summary">更多展示图片设置<span>公告附图请在公告编辑中添加</span></summary><div class="image-maintenance-body">
      <div class="upload-heading"><div><h2>公告图片</h2><p>维护独立的展示图片，JPG 格式，单张不超过8MB。</p></div></div>
      <el-upload ref="galleryUpload" class="upload-demo" :disabled="gallerySaving" :on-change="galleryChanged" :on-error="galleryError" action="restful/image/saveImage" :with-credentials="true" :headers="$uploadHeaders" :on-preview="handlePreview" :on-remove="handleRemove" :on-success="handleAvatarSuccess" :before-upload="beforeAvatarUpload" multiple :limit="10" :file-list="fileList2" list-type="picture">
        <el-button size="small" type="primary" icon="el-icon-upload2">选择图片</el-button>
        <template #tip><div  class="el-upload__tip">最多上传 10 张图片。</div></template>
      </el-upload>
      <div class="upload-footer"><el-button type="primary" :loading="gallerySaving" :disabled="!fileList2.length || fileList2.some(file => file.status !== 'success')" @click="picShow">提交图片</el-button></div>
    </div></details>
  </div>
</template>
<style scoped>
.announcement-admin { display: grid; grid-template-columns: minmax(0, 1fr); gap: 20px; min-width: 0; color: #263632; }
.announcement-list-panel, .upload-panel { min-width: 0; max-width: 100%; box-sizing: border-box; padding: 22px; border: 1px solid #e5ebe7; border-radius: 14px; background: #fff; box-shadow: 0 4px 18px rgba(29,63,51,.04); }
.announcement-toolbar { display: flex; align-items: center; justify-content: space-between; gap: 20px; min-width: 0; margin-bottom: 18px; }
.announcement-toolbar h2, .upload-heading h2 { margin: 0; font-size: 18px; font-weight: 600; }
.announcement-toolbar p, .upload-heading p { margin: 6px 0 0; color: #64756c; font-size: 13px; line-height: 1.6; }
.announcement-query { flex: 1; min-width: 0; }
.announcement-actions { display: flex; align-items: flex-end; gap: 16px; width: min(100%, 540px); min-width: 0; }
.announcement-actions .el-input { min-width: 0; }
.announcement-actions > .el-button { min-height: var(--care-control-height); }
.announcement-list-panel .el-table { width: 100% !important; }
.pagination-wrap { display: flex; justify-content: flex-end; padding-top: 18px; }
.upload-panel { padding-bottom: 18px; }
.upload-footer { display: flex; justify-content: flex-end; margin-top: 18px; }
@media (max-width: 760px) { .announcement-list-panel, .upload-panel { padding: 18px 14px; } .announcement-toolbar { align-items: stretch; flex-direction: column; } .announcement-actions { width: 100%; } .pagination-wrap { justify-content: flex-start; overflow-x: auto; } }
@media (max-width: 480px) { .announcement-actions { align-items: stretch; flex-direction: column; } .announcement-actions .el-input { width: 100%; } }
</style>
<script>
	
	import axios from 'axios'
	
export default {
	created () {

		this.searchOlds()

	},

	methods: {
      handleEdit(index, row) {

				var title = row.title
				var textMain = row.textMain
				var textId = row.textId
				var picId = row.picId
				this.$router.push({
					path: '/text',
					query: {
						title: title,
						textMain: textMain,
						textId: textId,
						picId: picId
					}
				})
      },
      picShow () {
        if (this.gallerySaving || !this.fileList2.length || this.fileList2.some(file => file.status !== 'success')) return
        this.gallerySaving = true
        axios.post('restful/picshow/add', { fileListName: this.fileList2.map(file => file.name) }, { withCredentials: true }).then((response) => {
          if (response.data.returnCode !== '1111') throw new Error('gallery rejected')
          this.$message.success('展示图片已保存')
          this.$refs.galleryUpload.clearFiles()
          this.fileList2 = []
        }).catch(() => { this.$message.error('展示图片未能保存，选择的图片已保留，请重试') }).then(() => { this.gallerySaving = false })
      },
	  handleSizeChange (val) { this.pageSize = val; this.currentPage = 1; this.searchOlds(true) },

	  changePage (val) { this.currentPage = val; this.searchOlds(true) },
      galleryChanged (file, files) { this.fileList2 = files },
      handleRemove (file, files) { this.fileList2 = files },
      handlePreview (file) { this.previewUrl = file.url; this.previewVisible = true },
      handleAvatarSuccess (res, file, files) {
        if (!res || res.returnCode !== '1111' || !res.filename) { file.status = 'fail'; this.$message.error('图片未能上传，请删除后重试') }
        else { file.name = res.filename }
        this.fileList2 = files
      },
      galleryError (error, file, files) { this.fileList2 = files; this.$message.error('图片暂时无法上传，请重试') },
      beforeAvatarUpload (file) {
        if (file.type !== 'image/jpeg') { this.$message.error('请选择 JPG 图片'); return false }
        if (file.size > 8 * 1024 * 1024) { this.$message.error('图片大小不能超过8MB'); return false }
        return true
      },
	  getDetail(row){

	  	var textId = row.textId
	  	var picId = row.picId

	  	this.$router.push({
	  	path: '/textMainUser',
	  	query: {
	  		textId: textId,
	  		picId: picId
	  	}
	  	})
	  },
      handleDelete (index, row) {
        if (this.deletingId !== null) return
        this.deletingId = row.textId
        this.$confirm('删除后将无法继续阅读这条公告，是否删除？', '删除公告', { confirmButtonText: '删除公告', cancelButtonText: '保留公告', type: 'warning' }).then(() => {
          return axios.post('restful/text/delete', { textId: row.textId, picId: row.picId }, { withCredentials: true })
        }).then((response) => {
          if (response.data.returnCode !== '1111') throw new Error('delete rejected')
          this.$message.success('公告已删除')
          this.searchOlds()
        }).catch((error) => { if (error !== 'cancel' && error !== 'close') this.$message.error('公告暂时无法删除，请重试') }).then(() => { this.deletingId = null })
      },
			addtext(){
				this.$router.push('/textTwo')
			},
      searchOlds (keepPage = false) {
      if (keepPage !== true) this.currentPage = 1
        this.isLoading = true
        this.loadError = ''
        return axios.post('restful/text/select', { page: this.currentPage, pageSize: this.pageSize,  title: this.searchText }, { withCredentials: true }).then((response) => {
          if (!Array.isArray(response.data.texts)) throw new Error('missing notices')
          this.tableData = response.data.texts
          this.total = response.data.total ?? this.tableData.length
        }).catch(() => { this.loadError = '公告暂时无法更新，请重试。' }).then(() => { this.isLoading = false })
      }

    },
  data() {
    return {
	  tableData: [],
        isLoading: false, loadError: '', deletingId: null,
		searchText: '',
		gallerySaving: false, previewVisible: false, previewUrl: '',
		fileList2: [],
		total: 0,
		pageSize: 10,
		currentPage: 1
    };
  }
}
</script>
