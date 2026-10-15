<template>
  <div class="notice-page">
    <el-button class="notice-back" link icon="el-icon-back" @click="$router.push('/familyMain')">返回照护总览</el-button>
    <article v-loading="isLoading" class="notice-article" :aria-busy="String(isLoading)">
      <div v-if="loadError" class="notice-empty" role="alert"><h1>公告暂时无法加载</h1><p>{{ loadError }}</p><el-button type="primary" @click="searchDetail">重新加载</el-button></div>
      <template v-else><p class="notice-kicker">平台公告</p><h1>{{ title || (isLoading ? '正在加载公告…' : '公告内容') }}</h1><p class="notice-time">{{ time || '发布时间未提供' }}</p><div class="notice-body">{{ textMain || (isLoading ? '' : '暂时没有正文内容。') }}</div><figure v-for="(img, index) in imgList" :key="index" class="notice-attachment"><img v-if="!failedImages[index]" :src="imageSource(img.picName)" :alt="'公告附件 ' + (index + 1)" loading="lazy" @error="failedImages[index] = true"><p v-else>附件图片暂时无法显示。</p></figure></template>
    </article>
    <section v-if="!isLoading && !loadError" class="notice-comments" aria-labelledby="notice-comment-title">
      <h2 id="notice-comment-title">留言交流</h2>
      <el-form :model="formSaid" @submit.prevent="onSubmit"><el-form-item label="您的留言（最多30字）"><el-input aria-label="您的留言（最多30字）" type="textarea" :rows="3" maxlength="30" v-model="formSaid.said" placeholder="写下您想咨询或分享的内容"></el-input></el-form-item><p v-if="submitError" class="notice-error" role="alert">{{ submitError }}</p><el-button type="primary" :loading="submitting" :disabled="!formSaid.said.trim()" @click="onSubmit">发表留言</el-button><p v-if="submitSuccess" class="notice-success" role="status">{{ submitSuccess }}</p></el-form>
      <div class="comment-list" v-loading="commentsLoading"><p v-if="commentsError" role="alert">{{ commentsError }} <el-button link @click="searchSaid">重新加载</el-button></p><p v-else-if="!tableData.length" class="comment-empty">还没有留言，欢迎分享您的想法。</p><article v-for="(item, index) in tableData" :key="item.saidId || index" class="comment-item"><div class="comment-heading"><strong>{{ item.personName || '家属' }}</strong><span>{{ item.saidTime || '' }}</span></div><p>{{ item.said }}</p><div v-if="item.back" class="comment-reply"><strong>平台回复</strong><p>{{ item.back }}</p></div><el-button link :aria-pressed="String(item.good == 1)" :loading="likingId === item.saidId" @click="getview(item)">{{ item.good == 1 ? '取消赞同' : '赞同' }} · {{ item.niceCount || 0 }}</el-button></article></div>
    </section>
  </div>
</template>
<script>
import axios from 'axios'
export default {
  data () {
    return { title: '', time: '', textMain: '', imgList: [], failedImages: {}, tableData: [], formSaid: { said: '' }, isLoading: false, loadError: '', commentsLoading: false, commentsError: '', submitting: false, submitError: '', submitSuccess: '', likingId: null }
  },
  created () { this.searchDetail(); this.searchSaid() },
  methods: {
    imageSource (name) { return 'restful/image/download?name=' + encodeURIComponent(name || '') },
    searchDetail () {
      if (!this.$route.query.textId) { this.loadError = '缺少公告信息，请返回总览重新选择。'; return }
      this.isLoading = true
      this.loadError = ''
      this.failedImages = {}
      axios.post('restful/text/selectDetail', { textId: this.$route.query.textId }, { withCredentials: true }).then(response => {
        var text = response.data && response.data.text
        if (!text) throw new Error('missing notice')
        this.title = text.title || '未命名公告'
        this.time = text.time || ''
        this.textMain = text.textMain || ''
        this.imgList = Array.isArray(response.data.pics) ? response.data.pics : []
      }).catch(() => { this.loadError = '请检查服务连接后重试。' }).then(() => { this.isLoading = false })
    },
    searchSaid () {
      if (!this.$route.query.textId) return
      this.commentsLoading = true
      this.commentsError = ''
      return axios.post('restful/said/select', { textId: this.$route.query.textId, familyPhone: localStorage.getItem('familyPhone') || '' }, { withCredentials: true }).then(response => {
        this.tableData = response.data && Array.isArray(response.data.said2s) ? response.data.said2s : []
      }).catch(() => { this.commentsError = '留言暂时无法加载。' }).then(() => { this.commentsLoading = false })
    },
    onSubmit () {
      if (!this.formSaid.said.trim() || this.submitting) return
      this.submitting = true
      this.submitError = ''
      this.submitSuccess = ''
      axios.post('restful/said/insert', { phone: localStorage.getItem('familyPhone') || '', said: this.formSaid.said.trim(), textId: this.$route.query.textId, personName: localStorage.getItem('user_name') || '', touPic: localStorage.getItem('familyTouPic') || '' }, { withCredentials: true }).then(response => {
        if (!response.data || response.data.returnCode !== '1111') throw new Error('comment rejected')
        this.formSaid.said = ''
        this.submitSuccess = '留言已发表。'
        return this.searchSaid()
      }).catch(() => { this.submitError = '留言未能发表，内容已保留，请重试。' }).then(() => { this.submitting = false })
    },
    getview (item) {
      if (this.likingId !== null) return
      this.likingId = item.saidId
      this.submitError = ''
      axios.post('restful/said/goodCount', { saidId: item.saidId, phone: localStorage.getItem('familyPhone') || '' }, { withCredentials: true }).then(response => {
        if (!response.data || response.data.returnCode !== '1111') throw new Error('like rejected')
        return this.searchSaid()
      }).catch(() => { this.submitError = '赞同操作未完成，请重试。' }).then(() => { this.likingId = null })
    }
  }
}
</script>
<style scoped>
.notice-back { margin-bottom: 16px; }
.notice-article, .notice-comments { padding: 32px 40px; border: 1px solid #dfe5d8; border-radius: 16px; background: #fff; }
.notice-article { min-height: 260px; }
.notice-kicker { margin: 0 0 12px; color: #496151; font-size: 17px; }
.notice-article h1 { margin: 0; color: #183e33; font-size: 32px; line-height: 1.6; overflow-wrap: anywhere; }
.notice-time { color: #59665f; font-size: 17px; }
.notice-body { margin-top: 28px; padding-top: 24px; border-top: 1px solid #e8ece4; color: #33453c; white-space: pre-wrap; overflow-wrap: anywhere; font-size: 19px; line-height: 1.95; }
.notice-attachment { margin: 24px 0 0; }
.notice-attachment img { display: block; width: auto; max-width: 100%; height: auto; border-radius: 10px; }
.notice-comments { margin-top: 22px; }
.notice-comments h2 { margin: 0 0 24px; font-size: 24px; color: #213c32; }
.notice-comments :deep(.el-form-item__label) { float: none; display: block; text-align: left; }
.notice-comments :deep(.el-form-item__content) { margin-left: 0 !important; }
.comment-list { margin-top: 28px; min-height: 80px; }
.comment-item { padding: 22px 0; border-top: 1px solid #e8ece4; }
.comment-item > p, .comment-reply p { margin: 10px 0; white-space: pre-wrap; overflow-wrap: anywhere; line-height: 1.8; }
.comment-heading { display: flex; flex-wrap: wrap; justify-content: space-between; gap: 8px; }
.comment-heading span, .comment-empty { color: #59665f; font-size: 16px; }
.comment-reply { padding: 14px 18px; border-left: 3px solid #b3c6a8; background: #f4f7ef; border-radius: 0 8px 8px 0; }
.notice-error { color: #a4262c; } .notice-success { color: #236b3d; }
.notice-empty { text-align: center; }
@media (max-width: 600px) { .notice-article, .notice-comments { padding: 24px; } .notice-article h1 { font-size: 26px; } }
html.large-text .care-family .notice-body { font-size: 22px; }
</style>
