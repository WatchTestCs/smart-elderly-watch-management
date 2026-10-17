<template>
  <div class="dashboard">
    <div class="page-heading">
      <div><h1>运营总览</h1><p class="page-subtitle">查看设备异常与平台公告，及时跟进照护事项。</p></div>
      <el-button type="primary" icon="el-icon-plus" @click="addtext">发布公告</el-button>
    </div>
    <section class="overview-grid" aria-label="设备告警摘要">
      <article class="overview-card alert-card" v-loading="countLoading" :aria-busy="String(countLoading)">
        <div class="metric-icon amber"><i class="el-icon-warning"></i></div>
        <div class="metric-copy"><span>设备异常类别计数</span><strong v-if="!countError">{{ allCount == null ? '—' : allCount }}</strong><strong v-else class="metric-error" role="status">{{ countError }}</strong><small>同一设备可能对应多类异常</small></div>
        <el-button v-if="countError" link @click="getCount">重新加载设备统计</el-button><el-button v-else link @click="$router.push('/device')">查看设备 <i class="el-icon-arrow-right"></i></el-button>
      </article>
      <article class="overview-card fall-card" v-loading="positionLoading" :aria-busy="String(positionLoading)">
        <div class="metric-icon rose"><i class="el-icon-warning"></i></div>
        <div class="metric-copy"><span>摔倒告警</span><strong v-if="!positionError">{{ num == null ? '—' : num }}</strong><strong v-else class="metric-error" role="status">{{ positionError }}</strong><small>根据最近上报的位置记录统计</small></div>
        <el-button v-if="positionError" link @click="searchposition">重新加载摔倒统计</el-button><el-button v-else link @click="$router.push('/position')">查看位置 <i class="el-icon-arrow-right"></i></el-button>
      </article>
      <article class="overview-note">
        <span class="note-icon"><i class="el-icon-info"></i></span>
        <div><strong>数据说明</strong><p>健康与位置数据取决于手表最近一次上报。页面不会将未上报的数据标记为正常。</p></div>
      </article>
    </section>
    <section class="announcement-panel">
      <div class="panel-heading">
        <div><h2>平台公告 <span v-if="!isLoading && !loadError" class="notice-count" :aria-label="'共 ' + total + ' 条公告'">{{ total }}</span></h2><p>面向家属发布的服务与照护信息</p></div>
        <form class="search-box" role="search" @submit.prevent="searchOlds"><label class="search-label" for="admin-notice-search">按标题查找公告</label><el-input id="admin-notice-search" v-model="searchText" placeholder="搜索公告标题" :disabled="isLoading" clearable @clear="searchOlds"><template #append><el-button  icon="el-icon-search" aria-label="搜索公告" :loading="isLoading" native-type="submit"></el-button></template></el-input></form>
      </div>
      <el-table class="desktop-announcements" v-loading="isLoading" :data="tableData" @row-click="getDetail" row-class-name="clickable-row">
        <el-table-column type="index" label="序号" width="64" align="center"></el-table-column>
        <el-table-column prop="title" label="公告标题" min-width="220" show-overflow-tooltip>
          <template #default="scope" ><span class="announcement-title">{{ scope.row.title || '未命名公告' }}</span></template>
        </el-table-column>
        <el-table-column prop="time" label="发布时间" width="175"></el-table-column>
        <el-table-column label="查看" width="110" align="right"><template #default="scope" ><el-button link @click.stop="getDetail(scope.row)">查看详情 <i class="el-icon-arrow-right"></i></el-button></template></el-table-column>
        <template #empty><div  class="table-empty" role="status"><img v-if="!loadError && !appliedSearch" src="/static/img/care-announcement-empty.webp" alt="" width="160" height="160"><strong>{{ emptyTitle }}</strong><span>{{ emptyDescription }}</span><el-button v-if="loadError" plain @click="searchOlds">重新加载</el-button><el-button v-else-if="appliedSearch" plain @click="clearSearch">清除搜索条件</el-button></div></template>
      </el-table>
      <div class="mobile-announcements" v-loading="isLoading">
        <div v-if="!tableData.length" class="table-empty" role="status"><img v-if="!loadError && !appliedSearch" src="/static/img/care-announcement-empty.webp" alt="" width="160" height="160"><strong>{{ emptyTitle }}</strong><span>{{ emptyDescription }}</span><el-button v-if="loadError" plain @click="searchOlds">重新加载</el-button><el-button v-else-if="appliedSearch" plain @click="clearSearch">清除搜索条件</el-button></div>
        <button v-for="notice in tableData" :key="notice.textId" type="button" class="mobile-notice" :aria-label="'查看详情：' + (notice.title || '未命名公告')" @click="getDetail(notice)"><span><strong>{{ notice.title || '未命名公告' }}</strong><time>{{ notice.time || '暂无发布时间' }}</time></span><i class="el-icon-arrow-right" aria-hidden="true"></i></button>
      </div>
      <div class="pagination-wrap" v-if="total > pageSize"><el-pagination @size-change="handleSizeChange" @current-change="changePage" v-model:current-page="currentPage" :page-size="pageSize" layout="total, sizes, prev, pager, next" :total="total"></el-pagination></div>
    </section>
  </div>
</template>
<style scoped>
.mobile-announcements { display: none; }
.mobile-notice { display: flex; width: 100%; min-height: 88px; justify-content: space-between; align-items: center; gap: 16px; padding: 20px 0; background: transparent; border: 0; border-top: 1px solid #e8ece4; font: inherit; text-align: left; color: #263632; cursor: pointer; }
.mobile-notice span { min-width: 0; }
.mobile-notice strong { display: block; font-size: 16px; line-height: 1.7; overflow-wrap: anywhere; }
.mobile-notice time { display: block; margin-top: 6px; font-size: 14px; color: #53645c; }
.mobile-notice:focus-visible { outline: 3px solid #235442; outline-offset: 2px; }
.dashboard { color: #263632; }
.page-heading { display: flex; align-items: flex-end; justify-content: space-between; gap: 20px; margin-bottom: 24px; }
.page-heading h1 { margin: 0; font-size: 28px; line-height: 1.4; font-weight: 500; letter-spacing: 0; }
.page-subtitle { margin: 9px 0 0; color: #64756c; font-size: 14px; line-height: 1.6; }
.page-heading :deep(.el-button) { height: 48px; padding: 0 20px; border-radius: 8px; }
.overview-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 0; margin-bottom: 28px; border: 1px solid #dfe5df; border-radius: 12px; background: #fff; overflow: hidden; }
.overview-card { display: flex; flex-wrap: wrap; align-items: center; gap: 16px; min-height: 148px; padding: 28px; background: #fff; }
.alert-card { border-right: 1px solid #e2e9e1; }
.metric-icon { display: grid; place-items: center; width: 44px; height: 44px; flex: 0 0 44px; border-radius: 12px; font-size: 20px; }
.metric-icon.amber { background: #fff5e5; color: #b7791f; }
.metric-icon.rose { background: #fff0ed; color: #b5473c; }
.metric-copy { display: flex; flex: 1; flex-direction: column; gap: 3px; min-width: 0; }
.metric-copy span { color: #53645c; font-size: 14px; }
.metric-copy .metric-error { color: #a4262c; font-size: 16px; line-height: 1.7; }
.metric-copy strong { color: #263632; font-size: 30px; line-height: 1.15; font-weight: 600; }
.metric-copy small { color: #53645c; font-size: 14px; line-height: 1.5; }
.overview-card :deep(.el-button.is-link) { min-height: 48px; flex: 0 0 auto; color: #28776a; font-size: 14px; }
.overview-note { grid-column: 1 / -1; display: flex; gap: 12px; align-items: flex-start; padding: 16px 28px; border-top: 1px solid #e2e9e1; background: #f3f6f1; }
.note-icon { color: #28776a; font-size: 17px; }
.overview-note strong { color: #43534b; font-size: 14px; }
.overview-note p { margin: 4px 0 0; color: #64756c; font-size: 14px; line-height: 1.65; }
.announcement-panel { padding: 21px; border: 1px solid #e5ebe7; border-radius: 12px; background: #fff; box-shadow: none; }
.panel-heading { display: flex; align-items: center; justify-content: space-between; gap: 18px; margin-bottom: 16px; }
.panel-heading h2 { margin: 0; font-size: 18px; font-weight: 500; }
.panel-heading p { margin: 6px 0 0; color: #53645c; font-size: 14px; }
.search-label { display: block; margin-bottom: 8px; font-size: 14px; color: #53645c; }
.notice-count { display: inline-block; min-width: 28px; margin-left: 6px; padding: 1px 8px; border-radius: 6px; background: #edf2e6; color: #235442; font-size: 14px; text-align: center; vertical-align: middle; }
.search-box { width: min(100%, 280px); }
.announcement-title { color: #33453c; font-weight: 500; }
.table-empty { display: flex; flex-direction: column; align-items: center; gap: 8px; padding: 28px 10px; color: #53645c; }
.table-empty img { display: block; object-fit: contain; }
.table-empty strong { color: #263632; font-size: 17px; line-height: 1.6; }
.table-empty span { max-width: 420px; font-size: 15px; line-height: 1.8; white-space: normal; }
.pagination-wrap { display: flex; justify-content: flex-end; padding-top: 18px; }
@media (min-width: 901px) and (max-width: 1180px) { .overview-grid { grid-template-columns: 1fr; } .alert-card { border-right: 0; border-bottom: 1px solid #e2e9e1; } }
@media (max-width: 760px) { .overview-card { padding: 22px 18px; } .alert-card { border-right: 0; border-bottom: 1px solid #e2e9e1; } .overview-note { padding: 18px; } .desktop-announcements { display: none; } .mobile-announcements { display: block; } .page-heading { align-items: flex-start; flex-direction: column; } .page-heading h1 { font-size: 25px; } .overview-grid { grid-template-columns: 1fr; } .overview-note { grid-column: auto; } .panel-heading { align-items: flex-start; flex-direction: column; } .search-box { width: 100%; } .announcement-panel { padding: 18px 14px; } .pagination-wrap { justify-content: center; overflow-x: auto; } }
</style>
<script>
import axios from 'axios'

export default {
  data () {
    return {
      tableData: [], searchText: '', appliedSearch: '', total: 0,
      isLoading: false, loadError: '', pageSize: 10, currentPage: 1,
      allCount: null, num: null, countLoading: false, positionLoading: false,
      countError: '', positionError: ''
    }
  },
  computed: {
    emptyTitle () {
      if (this.loadError) return '公告暂时无法加载'
      return this.appliedSearch ? '没有找到相关公告' : '暂时没有公告'
    },
    emptyDescription () {
      if (this.loadError) return this.loadError
      return this.appliedSearch ? '试试其他标题关键词，或清除搜索条件查看全部公告。' : '发布后的服务信息和照护提醒会显示在这里。'
    }
  },
  created () { this.searchOlds(); this.getCount(); this.searchposition() },
  methods: {
    getCount () {
      if (this.countLoading) return
      this.countLoading = true
      this.countError = ''
      this.allCount = null
      return axios.post('restful/step/getAllCount', {}, { withCredentials: true }).then((response) => {
        var value = response.data.allCount
        if ((typeof value !== 'number' && typeof value !== 'string') || String(value).trim() === '' || !Number.isInteger(Number(value)) || Number(value) < 0) throw new Error('missing count')
        this.allCount = Number(value)
      }).catch(() => { this.countError = '设备统计暂时无法加载' }).then(() => { this.countLoading = false })
    },
    searchposition () {
      if (this.positionLoading) return
      this.positionLoading = true
      this.positionError = ''
      this.num = null
      return axios.post('restful/position/select', { name: '' }, { withCredentials: true }).then((response) => {
        if (!Array.isArray(response.data.positionDtos)) throw new Error('missing positions')
        this.num = response.data.positionDtos.filter(item => item.status === 1 || item.status === '1').length
      }).catch(() => { this.positionError = '摔倒统计暂时无法加载' }).then(() => { this.positionLoading = false })
    },
    handleSizeChange (val) { this.pageSize = val; this.currentPage = 1; this.searchOlds(true) },
    changePage (val) { this.currentPage = val; this.searchOlds(true) },
    getDetail (row) { this.$router.push({ path: '/textMainUser', query: { textId: row.textId, picId: row.picId } }) },
    addtext () { this.$router.push('/textTwo') },
    clearSearch () { this.searchText = ''; this.searchOlds() },
    searchOlds (keepPage = false) {
      if (keepPage !== true) this.currentPage = 1
      if (this.isLoading) return
      this.appliedSearch = this.searchText.trim()
      this.isLoading = true
      this.loadError = ''
      return axios.post('restful/text/select', { page: this.currentPage, pageSize: this.pageSize,  title: this.appliedSearch }, { withCredentials: true }).then((response) => {
        if (!Array.isArray(response.data.texts)) throw new Error('missing notices')
        this.tableData = response.data.texts
        this.total = response.data.total ?? this.tableData.length
      }).catch(() => {
        this.loadError = '请检查服务连接后重试'
        this.tableData = []
        this.total = 0
      }).then(() => { this.isLoading = false })
    }
  }
}
</script>
