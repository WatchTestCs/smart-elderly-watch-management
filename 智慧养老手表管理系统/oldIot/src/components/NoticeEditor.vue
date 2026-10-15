<template>
  <div class="notice-editor" :aria-busy="String(loading || saving)">
    <div v-if="loadError" class="collection-feedback" role="alert"><span>{{ loadError }}</span><el-button plain @click="loadNotice">重新加载公告</el-button></div>
    <section class="editor-main collection-panel" v-loading="loading">
      <div class="editor-section-heading"><div><h2>公告内容</h2><p>清晰的标题与简短的正文，让家属更容易了解重点。</p></div></div>
      <el-form ref="editorForm" :model="form" :rules="rules" label-position="top">
        <el-form-item label="公告标题" for="notice-title" prop="title"><el-input id="notice-title" v-model="form.title" maxlength="20" placeholder="用一句话说明本次提醒" :disabled="saving || loading || !!loadError"></el-input><p class="editor-counter">{{ form.title.length }} / 20 字</p></el-form-item>
        <el-form-item label="公告正文" for="notice-body" prop="textMain"><el-input id="notice-body" type="textarea" :rows="4" v-model="form.textMain" maxlength="50" placeholder="写下需要家属知悉的简要提醒" :disabled="saving || loading || !!loadError"></el-input><p class="editor-counter">{{ form.textMain.length }} / 50 字</p></el-form-item>
      </el-form>
      <section class="editor-attachments" aria-labelledby="attachment-heading"><div class="editor-section-heading"><div><h2 id="attachment-heading">补充图片</h2><p>图片可选，请上传与本次公告直接相关的内容。</p></div></div><el-upload action="restful/image/saveImage" :headers="$uploadHeaders" :with-credentials="true" :file-list="attachments" :disabled="saving || loading || !!loadError" :limit="10" accept="image/jpeg,image/png" list-type="picture" :before-upload="beforeUpload" :on-change="filesChanged" :on-success="uploadSuccess" :on-error="uploadError" :on-remove="filesRemoved" :on-exceed="uploadExceeded"><el-button plain icon="el-icon-plus" :disabled="saving || loading || !!loadError">添加公告图片</el-button><template #tip><p  class="collection-caption">最多10张 JPG / PNG 图片，每张不超过500KB。{{ uploadsPending ? '图片仍在上传或未成功，请处理后再发布。' : '' }}</p></template></el-upload></section>
      <p v-if="saveError" class="editor-error" role="alert">{{ saveError }}</p>
      <footer class="editor-actions"><span>确认预览内容后再{{ editing ? '保存' : '发布' }}。</span><el-button type="primary" :loading="saving" :disabled="loading || !!loadError || uploadsPending" @click="submit">{{ editing ? '保存公告' : '发布公告' }}</el-button></footer>
    </section>
    <aside class="editor-side"><section class="notice-preview collection-panel" aria-label="公告实时预览"><p class="collection-eyebrow">家属阅读预览</p><h2>{{ form.title || '这里显示公告标题' }}</h2><p class="preview-time">{{ editing ? '保存后更新公告内容' : '发布后显示发布时间' }}</p><div class="preview-body">{{ form.textMain || '正文会在这里显示，方便您确认语气与重点。' }}</div><img v-for="file in successfulAttachments" :key="file.uid || file.name" :src="file.url" alt="公告附图预览"></section><section class="editor-guidance"><img src="/static/img/care-courtyard.webp" alt="" width="960" height="411"><div><h3>简洁一点，关怀更清楚</h3><p>先说明事项，再交代家属需要留意的内容。避免模糊时间和未经确认的健康结论。</p></div></section></aside>
  </div>
</template>
<script>
import axios from 'axios'
export default {
  name: 'NoticeEditor',
  props: { editing: { type: Boolean, default: false } },
  data () { return { form: { title: '', textMain: '' }, attachments: [], picId: '', loading: false, saving: false, loadError: '', saveError: '', baseline: JSON.stringify({ title: '', textMain: '', names: [] }), rules: { title: [{ required: true, whitespace: true, message: '请输入公告标题', trigger: 'blur' }], textMain: [{ required: true, whitespace: true, message: '请输入公告正文', trigger: 'blur' }] } } },
  computed: {
    signature () { return JSON.stringify({ title: this.form.title, textMain: this.form.textMain, names: this.attachments.map(file => file.name) }) },
    dirty () { return this.signature !== this.baseline },
    successfulAttachments () { return this.attachments.filter(file => file.status === 'success') },
    uploadsPending () { return this.attachments.some(file => file.status !== 'success') }
  },
  created () { if (this.editing) this.loadNotice() },
  watch: { '$route.query.textId' () { if (this.editing) this.loadNotice() } },
  methods: {
    confirmLeave (next) {
      if (!this.dirty) { next(); return }
      if (this.saving || this.attachments.some(file => file.status === 'ready' || file.status === 'uploading')) { this.$message.warning('正在保存或处理图片，请稍候再离开'); next(false); return }
      this.$confirm('尚有未保存的内容，离开后将丢失。', '离开编辑', { confirmButtonText: '离开编辑', cancelButtonText: '继续编辑', type: 'warning' }).then(() => next()).catch(() => next(false))
    },
    loadNotice () {
      this.loading = true
      this.loadError = ''
      this.attachments = []
      return axios.post('restful/text/selectDetail', { textId: this.$route.query.textId }, { withCredentials: true }).then((response) => {
        if (!response.data.text || response.data.text.picId == null) throw new Error('missing notice')
        this.form = { title: response.data.text.title || '', textMain: response.data.text.textMain || '' }
        this.picId = response.data.text.picId
        this.attachments = (response.data.pics || []).map((pic, index) => ({ uid: 'existing-' + index, name: pic.picName, url: '/restful/image/download?name=' + encodeURIComponent(pic.picName), status: 'success' }))
        this.baseline = this.signature
      }).catch(() => { this.loadError = '公告暂时无法加载，请重试。' }).then(() => { this.loading = false })
    },
    beforeUpload (file) {
      if (!['image/jpeg', 'image/png'].includes(file.type)) { this.$message.error('请选择 JPG 或 PNG 图片'); return false }
      if (file.size > 500 * 1024) { this.$message.error('每张图片不能超过500KB'); return false }
      if (this.attachments.filter(item => item.name === file.name).length > 1) { this.$message.error('已有同名图片，请更换文件名后上传'); return false }
      return true
    },
    filesChanged (file, files) { this.attachments = files },
    filesRemoved (file, files) { this.attachments = files },
    uploadSuccess (response, file, files) {
      if (!response || response.returnCode !== '1111' || !response.filename) { file.status = 'fail'; this.$message.error('图片未能上传，请删除后重试') }
      else { file.name = response.filename }
      this.attachments = files
    },
    uploadError (error, file, files) { this.attachments = files; this.$message.error('图片暂时无法上传，请删除后重试') },
    uploadExceeded () { this.$message.warning('最多添加10张公告图片') },
    submit () {
      if (this.saving || this.loading || this.loadError || this.uploadsPending) return
      this.$refs.editorForm.validate((valid) => {
        if (!valid || this.saving) return
        this.saving = true
        this.saveError = ''
        var params = { title: this.form.title.trim(), textMain: this.form.textMain.trim(), fileListName: this.successfulAttachments.map(file => file.name) }
        if (this.editing) { params.textId = this.$route.query.textId; params.picId = this.picId }
        axios.post('restful/text/' + (this.editing ? 'update' : 'add'), params, { withCredentials: true }).then((response) => {
          if (response.data.returnCode !== '1111') throw new Error('save rejected')
          this.$message.success(this.editing ? '公告已保存' : '公告已发布')
          this.baseline = this.signature
          this.$router.push('/textOpt')
        }).catch(() => { this.saveError = '公告暂时未能保存，内容已保留，请稍后重试。' }).then(() => { this.saving = false })
      })
    }
  }
}
</script>
<style scoped>
.notice-editor { display: grid; grid-template-columns: minmax(0, 1.45fr) minmax(260px, 1fr); gap: 24px; align-items: start; }
.editor-main { padding: 28px; }
.editor-section-heading { display: flex; align-items: flex-start; gap: 14px; margin-bottom: 24px; }
.editor-section-heading h2 { margin: 0; color: #213c32; font-size: 20px; line-height: 1.6; }
.editor-section-heading p, .editor-counter { color: #53645c; font-size: 14px; line-height: 1.8; margin: 4px 0 0; }
.editor-counter { text-align: right; }
.editor-attachments { margin-top: 28px; padding-top: 24px; border-top: 1px solid #e4eadd; }
.editor-actions { display: flex; flex-wrap: wrap; gap: 16px; align-items: center; justify-content: space-between; margin-top: 28px; padding-top: 24px; border-top: 1px solid #e4eadd; }
.editor-actions span { color: #53645c; font-size: 14px; }
.editor-error { color: #a4262c; line-height: 1.8; }
.editor-side { min-width: 0; }
.notice-preview { padding: 28px; }
.notice-preview h2 { margin: 16px 0 10px; color: #213c32; font-size: 24px; line-height: 1.65; overflow-wrap: anywhere; }
.preview-time { color: #53645c; font-size: 13px; }
.preview-body { min-height: 100px; margin-top: 24px; color: #33453c; font-size: 17px; line-height: 2; white-space: pre-wrap; overflow-wrap: anywhere; }
.notice-preview img { display: block; width: 100%; height: auto; margin-top: 16px; border-radius: 8px; }
.editor-guidance { margin-top: 20px; overflow: hidden; border: 1px solid #dfe5d8; border-radius: 16px; background: #edf1e5; }
.editor-guidance > img { display: block; width: 100%; height: 130px; object-fit: cover; }
.editor-guidance > div { padding: 20px 24px; }
.editor-guidance h3 { margin: 0 0 8px; font-size: 17px; color: #213c32; }
.editor-guidance p { margin: 0; color: #53645c; font-size: 14px; line-height: 1.9; }
@media (max-width: 1100px) { .notice-editor { grid-template-columns: 1fr; } .editor-side { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 20px; } .editor-guidance { margin-top: 0; } }
@media (max-width: 600px) { .editor-main, .notice-preview { padding: 22px 18px; } .editor-side { grid-template-columns: 1fr; } .editor-actions .el-button { width: 100%; } }
</style>
