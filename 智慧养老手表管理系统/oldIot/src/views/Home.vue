<template>
  <div class="care-app">
    <a class="skip-link" href="#admin-main" @click.prevent="$refs.workspaceMain.focus()">跳到主要内容</a>
    <nav-top></nav-top>
    <div class="workspace-layout">
      <slider class="workspace-sidebar"></slider>
      <main ref="workspaceMain" id="admin-main" class="workspace-main" tabindex="-1">
        <div class="workspace-tabs" v-if="openTab.length">
          <el-tabs v-model="activeIndex" @tab-click="tabClick" @tab-remove="tabRemove">
            <el-tab-pane v-for="item in openTab" :key="item.route" :label="item.name" :name="item.route" :closable="item.route !== '/main'"></el-tab-pane>
          </el-tabs>
        </div>
        <section class="workspace-content">
          <header v-if="showPageIntro" class="view-intro">
            <p class="view-breadcrumb">工作台 <i class="el-icon-arrow-right"></i> {{ $route.meta.title }}</p>
            <h1>{{ $route.meta.title }}</h1>
            <p class="view-description">{{ pageDescription }}</p>
          </header>
          <router-view v-slot="{ Component }"><transition name="care-view" mode="out-in"><component :is="Component" :key="$route.path" /></transition></router-view>
        </section>
      </main>
    </div>
  </div>
</template>

<script>
import Slider from '@/components/Slider.vue'
import navTop from '@/components/Header.vue'

export default {
  name: 'Home',
  components: { Slider, navTop },
  computed: {
    openTab () { return this.$store.state.openTab },
    showPageIntro () {
      return ['/main', '/device', '/inviteCode', '/annex'].indexOf(this.$route.path) === -1
    },
    pageDescription () {
      var descriptions = {
        '/ratepre': '查看近期心率与单项血压记录，异常状态以设备上报为准。',
        '/step': '查看手表上报的睡眠与每日步数记录。',
        '/position': '查看设备最近一次上报的位置与安全告警。',
        '/life': '查看手表上报的生活信息。',
        '/watch': '维护监测阈值档案；设备注册与在线状态请前往设备管理。',
        '/person': '管理老人基础档案与关联设备。',
        '/customer': '查看监护人资料与联系电话。',
        '/family': '查看家属账号与关联信息。',
        '/text': '编辑公告内容并预览发布信息。',
        '/textTwo': '创建面向家属的服务公告。',
        '/textMainUser': '查阅公告详情与图片内容。',
        '/textOpt': '查看和维护已发布公告。',
        '/saidOperation': '查看公告评论并处理互动内容。'
      }
      return descriptions[this.$route.path] || '在这里查看和管理照护信息。'
    },
    activeIndex: {
      get () { return this.$store.state.activeIndex },
      set (value) { this.$store.activate(value) }
    }
  },
  methods: {
    tabClick () { this.$router.push({ path: this.activeIndex }) },
    tabRemove (route) {
      if (route === '/main') return
      this.$store.removeTab(route)
      if (this.activeIndex === route) {
        var next = this.openTab[this.openTab.length - 1]
        this.$router.push({ path: next ? next.route : '/main' })
      }
    },
    addCurrentTab (route) {
      if (!route.path || route.path === '/home') return
      if (!this.openTab.some(function (tab) { return tab.route === route.path })) {
        this.$store.addTab({ route: route.path, name: route.meta.title || '运营总览' })
      }
      this.$store.activate(route.path)
    }
  },
  mounted () { this.addCurrentTab(this.$route) },
  watch: {
    '$route' (to) { this.addCurrentTab(to) }
  }
}
</script>

<style scoped>
.skip-link { position: fixed; top: -80px; left: 16px; z-index: 3000; padding: 14px 20px; border-radius: 8px; background: #fff; color: #183e33; }
.skip-link:focus { top: 12px; }
.care-app { min-height: 100vh; min-height: 100dvh; background: var(--care-canvas); }
.workspace-layout { display: grid; grid-template-columns: 232px minmax(0, 1fr); min-height: calc(100vh - 72px); }
.workspace-sidebar { min-width: 0; }
.workspace-main { min-width: 0; padding: 26px 32px 40px; }
.workspace-tabs { max-width: 1380px; margin: 0 auto 28px; border-bottom: 1px solid #dce4db; padding: 0; }
.workspace-tabs :deep(.el-tabs__header) { margin: 0; border: 0; }
.workspace-tabs :deep(.el-tabs__nav-wrap::after) { display: none; }
.workspace-tabs :deep(.el-tabs__item) { height: 48px; line-height: 48px; color: var(--care-muted); }
.workspace-tabs :deep(.el-tabs__item.is-active) { color: var(--care-primary); font-weight: 500; }
.workspace-tabs :deep(.el-tabs__active-bar) { background: var(--care-primary); }
.workspace-content :deep(.admin-record-dialog) { display: flex; flex-direction: column; max-width: 640px; max-height: calc(100vh - 48px); margin: 24px auto !important; }
.workspace-content :deep(.admin-record-dialog .el-dialog__header) , .workspace-content :deep(.admin-record-dialog .el-dialog__footer) { flex-shrink: 0; }
.workspace-content :deep(.admin-record-dialog .el-dialog__header) { padding-right: 76px; }
.workspace-content :deep(.admin-record-dialog .el-dialog__body) { min-height: 0; overflow-y: auto; }
.workspace-content :deep(.admin-record-dialog .el-dialog__headerbtn) { top: 8px; right: 12px; display: grid; place-items: center; width: 48px; height: 48px; font-size: 20px; }
.workspace-content { min-width: 0; max-width: 1380px; margin-right: auto; margin-left: auto; }
.view-intro { margin: 0 0 20px; }
.view-breadcrumb { margin: 0 0 10px; color: #53645c; font-size: 13px; }
.view-breadcrumb i { margin: 0 5px; font-size: 9px; }
.view-intro h1 { margin: 0; color: #263632; font-size: 28px; line-height: 1.25; font-weight: 500; letter-spacing: -.02em; }
.view-description { margin: 8px 0 0; color: #64756c; font-size: 14px; line-height: 1.65; }
@media (max-width: 900px) {
  .workspace-layout { display: flex; flex-direction: column; min-height: calc(100vh - 64px); }
  .workspace-main { padding: 20px 20px 30px; }
}
@media (max-width: 600px) {
  .workspace-main { padding: 16px 14px 26px; }
  .view-intro { margin-bottom: 14px; }
  .view-intro h1 { font-size: 24px; }
  .view-description { font-size: 14px; }
  .workspace-tabs { margin-bottom: 20px; padding: 0; }
}
</style>
