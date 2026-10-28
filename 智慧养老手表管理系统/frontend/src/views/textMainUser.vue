<template>
  <div class="admin-notice">
    <el-button plain icon="el-icon-arrow-left" @click="$router.push('/main')">返回运营总览</el-button>
    <article v-loading="loading" class="notice-panel">
      <div v-if="error" role="alert" class="notice-feedback"><p>{{ error }}</p><el-button plain @click="loadNotice">重新加载公告</el-button></div>
      <template v-else-if="notice">
        <header><p class="notice-label">平台公告</p><h2>{{ notice.title || '未命名公告' }}</h2><p class="notice-time">发布时间 · {{ notice.time || '暂未提供' }}</p></header>
        <div class="notice-body">{{ notice.textMain || '本公告暂无正文。' }}</div>
        <section v-if="pictures.length" class="notice-pictures" aria-label="公告图片"><img v-for="(pic, index) in pictures" :key="pic.picName" :src="imageUrl(pic.picName)" :alt="'公告附图 ' + (index + 1)"></section>
      </template>
    </article>
    <section class="notice-panel" v-loading="commentsLoading" aria-label="家属留言">
      <div class="comments-heading"><h2>家属留言 <span>{{ comments.length }}</span></h2><el-button plain @click="$router.push('/saidOperation')">管理评论</el-button></div>
      <div v-if="commentsError" role="alert" class="notice-feedback"><p>{{ commentsError }}</p><el-button plain @click="loadComments">重新加载留言</el-button></div>
      <p v-else-if="!commentsLoading && !comments.length" class="notice-time">暂时没有留言。</p>
      <ul v-else class="comment-list"><li v-for="(comment, index) in comments" :key="comment.saidId || index"><strong>{{ comment.personName || '家属' }}</strong><p>{{ comment.said }}</p><p v-if="comment.back" class="comment-reply">平台回复：{{ comment.back }}</p><small>赞同 · {{ comment.niceCount || 0 }}</small></li></ul>
    </section>
  </div>
</template>

<script>
import axios from 'axios'
export default {
  data () { return { notice: null, pictures: [], comments: [], loading: false, commentsLoading: false, error: '', commentsError: '' } },
  created () { this.loadNotice(); this.loadComments() },
  watch: { '$route.query.textId' () { this.loadNotice(); this.loadComments() } },
  methods: {
    imageUrl (name) { return '/api/v1/image/download?name=' + encodeURIComponent(name || '') },
    loadNotice () {
      this.loading = true
      this.error = ''
      this.notice = null
      this.pictures = []
      axios.post('api/v1/text/selectDetail', { textId: this.$route.query.textId }, { withCredentials: true }).then((response) => {
        if (!response.data.text) throw new Error('missing notice')
        this.notice = response.data.text
        this.pictures = response.data.pics || []
      }).catch(() => { this.error = '公告暂时无法加载，请重试。' }).then(() => { this.loading = false })
    },
    loadComments () {
      this.commentsLoading = true
      this.commentsError = ''
      this.comments = []
      axios.post('api/v1/said/select', { textId: this.$route.query.textId, familyPhone: '' }, { withCredentials: true }).then((response) => {
        this.comments = response.data.said2s || []
      }).catch(() => { this.commentsError = '留言暂时无法加载，请重试。' }).then(() => { this.commentsLoading = false })
    }
  }
}
</script>

<style scoped>
.admin-notice { max-width: 1040px; margin: 0 auto; color: #263632; }
.notice-panel { min-width: 0; margin-top: 20px; padding: 32px; border: 1px solid #dfe5d8; border-radius: 16px; background: #fff; }
.notice-label { margin: 0 0 12px; color: #235442; font-size: 13px; font-weight: 600; letter-spacing: .12em; }
.notice-panel h2 { margin: 0; font-size: 24px; line-height: 1.5; overflow-wrap: anywhere; }
.notice-time, .notice-feedback { color: #53645c; font-size: 15px; line-height: 1.8; }
.notice-body { margin-top: 28px; font-size: 17px; line-height: 2; white-space: pre-wrap; overflow-wrap: anywhere; }
.notice-pictures { display: grid; gap: 16px; margin-top: 24px; }
.notice-pictures img { display: block; max-width: 100%; height: auto; border-radius: 10px; }
.comments-heading { display: flex; justify-content: space-between; align-items: center; gap: 16px; flex-wrap: wrap; }
.comments-heading h2 { font-size: 20px; }
.comments-heading h2 span { margin-left: 8px; font-size: 15px; font-weight: 400; color: #53645c; }
.comment-list { list-style: none; margin: 24px 0 0; padding: 0; }
.comment-list li { padding: 20px 0; border-top: 1px solid #e8ece4; font-size: 16px; line-height: 1.8; overflow-wrap: anywhere; }
.comment-list p { margin: 8px 0; white-space: pre-wrap; }
.comment-reply { padding: 12px 16px; background: #f1f4eb; border-radius: 8px; }
.comment-list small { color: #53645c; font-size: 14px; }
@media (max-width: 600px) { .notice-panel { padding: 22px 18px; } .notice-panel h2 { font-size: 22px; } }
</style>
