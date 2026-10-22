<template>
  <aside class="care-sidebar" aria-label="管理端导航">
    <button class="navigation-toggle" type="button" :aria-expanded="expanded ? 'true' : 'false'" aria-controls="admin-navigation" @click="expanded = !expanded"><span><i class="el-icon-menu" aria-hidden="true"></i> 功能导航</span><span>{{ expanded ? '收起' : '展开' }} <i :class="expanded ? 'el-icon-arrow-up' : 'el-icon-arrow-down'" aria-hidden="true"></i></span></button>
    <div class="sidebar-label">工作空间</div>
    <el-menu :key="$route.path" :default-active="$route.path" :default-openeds="openedGroups" unique-opened id="admin-navigation" :class="['care-menu', { 'is-expanded': expanded }]" router>
      <el-menu-item index="/main"><i class="el-icon-menu" aria-hidden="true"></i><template #title><span >运营总览</span></template></el-menu-item>
      <el-menu-item v-if="!restrictedDuty" index="/device"><i class="el-icon-time" aria-hidden="true"></i><template #title><span >设备管理</span></template></el-menu-item>
      <el-sub-menu v-if="!restrictedDuty" index="health">
        <template #title ><i class="el-icon-document" aria-hidden="true"></i><span>健康与安全</span></template>
        <el-menu-item index="/ratepre">心率与血压</el-menu-item>
        <el-menu-item index="/step">睡眠与步数</el-menu-item>
        <el-menu-item index="/position">位置与告警</el-menu-item>
        <el-menu-item index="/life">生活信息</el-menu-item>
      </el-sub-menu>
      <el-sub-menu v-if="!restrictedDuty" index="records">
        <template #title ><i class="el-icon-service" aria-hidden="true"></i><span>档案管理</span></template>
        <el-menu-item index="/person">老人档案</el-menu-item>
        <el-menu-item index="/customer">监护人信息</el-menu-item>
        <el-menu-item index="/watch">监测阈值档案</el-menu-item>
        <el-menu-item index="/family">家属账号</el-menu-item>
        <el-menu-item index="/inviteCode">家属注册邀请码</el-menu-item>
      </el-sub-menu>
      <el-sub-menu v-if="!restrictedDuty" index="service">
        <template #title ><i class="el-icon-message" aria-hidden="true"></i><span>公告与互动</span></template>
        <el-menu-item index="/textTwo">发布公告</el-menu-item>
        <el-menu-item index="/textOpt">公告管理</el-menu-item>
        <el-menu-item index="/saidOperation">评论管理</el-menu-item>
      </el-sub-menu>
    </el-menu>
    <div class="sidebar-note"><span class="note-dot"></span><span>照护信息请以设备上报为准</span></div>
  </aside>
</template>

<script>
import { authState } from '@/auth'
export default {
  name: 'Slider',
  data () { return { expanded: false } },
  computed: {
    restrictedDuty () { return authState.account?.role === 'ADMIN' && ['caregiver', 'auditor'].includes(authState.account.duty) },
    openedGroups () {
      var groups = {
        health: ['/ratepre', '/step', '/position', '/life'],
        records: ['/person', '/customer', '/watch', '/family', '/inviteCode'],
        service: ['/textTwo', '/textOpt', '/saidOperation', '/text', '/textMainUser', '/annex']
      }
      return Object.keys(groups).filter(group => groups[group].includes(this.$route.path))
    }
  },
  watch: { '$route' () { this.expanded = false } }
}
</script>

<style scoped>
.care-sidebar { display: flex; flex-direction: column; min-height: 100%; background: #183e33; }
.navigation-toggle { display: none; }
.sidebar-label { padding: 24px 22px 12px; color: #c3d5c7; font-size: 13px; font-weight: 600; letter-spacing: .1em; }
.care-menu { border-right: 0; padding: 0 10px; background: transparent; }
.care-menu :deep(.el-menu) { background: transparent; }
.care-menu .el-menu-item, .care-menu :deep(.el-sub-menu__title) { height: 48px; line-height: 48px; margin: 3px 0; border-radius: 8px; color: #e0ebe0; font-size: 15px; }
.care-menu .el-menu-item i, .care-menu :deep(.el-sub-menu__title i) { color: #c3d5c7; font-size: 17px; }
.care-menu .el-menu-item:hover, .care-menu :deep(.el-sub-menu__title:hover) { background: #2b5143; color: #fff; }
.care-menu .el-menu-item.is-active { background: #e9efdf; color: #183e33; font-weight: 600; }
.care-menu .el-menu-item.is-active i { color: #183e33; }
.care-menu .el-menu-item:focus-visible, .care-menu :deep(.el-sub-menu__title:focus-visible) { outline: 3px solid #e9efdf; outline-offset: -3px; }
.care-menu :deep(.el-menu-item-group__title) { display: none; }
.care-menu :deep(.el-sub-menu .el-menu-item) { min-width: 0; padding-left: 48px !important; }
.sidebar-note { display: flex; gap: 8px; align-items: flex-start; margin: 28px 16px 20px; padding: 12px; border: 1px solid #587365; border-radius: 9px; color: #d7e3d8; font-size: 13px; line-height: 1.65; }
.note-dot { width: 7px; height: 7px; margin-top: 6px; flex: 0 0 7px; border-radius: 50%; background: #c1d899; }
@media (max-width: 900px) {
  .care-sidebar { border-bottom: 1px solid #587365; }
  .sidebar-label, .sidebar-note { display: none; }
  .navigation-toggle { display: flex; justify-content: space-between; align-items: center; width: 100%; min-height: 56px; padding: 0 20px; background: #183e33; border: 0; color: #e0ebe0; font: inherit; font-size: 15px; cursor: pointer; }
  .navigation-toggle:focus-visible { outline: 3px solid #e9efdf; outline-offset: -3px; }
  .navigation-toggle i { margin: 0 5px; }
  .care-menu { display: none; padding-bottom: 12px; }
  .care-menu.is-expanded { display: block; }
}
</style>
