<template>
  <aside class="care-sidebar" aria-label="家人照护导航">
    <div class="sidebar-label">家人照护</div>
    <el-menu :default-active="$route.path" class="care-menu" router>
      <el-menu-item index="/familyMain" aria-label="照护总览"><i class="el-icon-menu" aria-hidden="true"></i><span class="nav-label-full">照护总览</span><span class="nav-label-short">总览</span></el-menu-item>
      <el-menu-item index="/rateprefamily" aria-label="心率与血压"><i class="el-icon-document" aria-hidden="true"></i><span class="nav-label-full">心率与血压</span><span class="nav-label-short">健康</span></el-menu-item>
      <el-menu-item index="/stepfamily" aria-label="睡眠与步数"><i class="el-icon-time" aria-hidden="true"></i><span class="nav-label-full">睡眠与步数</span><span class="nav-label-short">活动</span></el-menu-item>
      <el-menu-item index="/positionfamily" aria-label="位置与安全"><i class="el-icon-location-outline" aria-hidden="true"></i><span class="nav-label-full">位置与安全</span><span class="nav-label-short">位置</span></el-menu-item>
      <el-menu-item index="/me" aria-label="个人中心"><span class="nav-profile-icon" aria-hidden="true"><svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="12" cy="7" r="3.5"/><path d="M5 21v-2a7 7 0 0 1 14 0v2Z"/></svg></span><span class="nav-label-full">个人中心</span><span class="nav-label-short">我的</span></el-menu-item>
    </el-menu>
    <div class="sidebar-moment" aria-hidden="true"><img src="/static/img/care-botanical.webp" alt="" width="440" height="660"><p>让关爱<br>成为日常</p></div>
    <div class="sidebar-note"><i class="el-icon-time" aria-hidden="true"></i><span>数据以最近一次<br>设备上报为准</span></div>
  </aside>
</template>
<script>
export default { name: 'FamilySlider' }
</script>
<style scoped>
.care-sidebar { position: relative; display: flex; flex-direction: column; min-height: 100%; background: #183e33; color: #f2f4e9; }
.sidebar-label { padding: 28px 26px 16px; color: #c4d4bd; font-size: 16px; font-weight: 500; letter-spacing: .08em; }
.care-menu { padding: 0 14px; border-right: 0; background: transparent; }
.care-menu .el-menu-item { position: relative; display: flex; align-items: center; height: 58px; margin: 6px 0; padding: 0 18px !important; border-radius: 9px; color: #ecf1e6; transition: background-color var(--care-motion-fast) var(--care-ease), color var(--care-motion-fast) var(--care-ease); }
.care-menu .el-menu-item i { margin-right: 14px; color: #c2d1bb; font-size: 22px; }
.care-menu .el-menu-item:hover,
.care-menu .el-menu-item:focus { background: #2b5142; color: #fff; }
.care-menu .el-menu-item.is-active { background: #e5eddc; color: #183e33; font-weight: 700; }
.care-menu .el-menu-item.is-active i { color: #235442; }
.care-menu .el-menu-item:focus-visible { outline: 3px solid #e3cf8e; outline-offset: 2px; }
.nav-profile-icon { display: grid; place-items: center; width: 24px; height: 24px; margin-right: 14px; }
.nav-label-short { display: none; }
.sidebar-moment { position: relative; min-height: 210px; flex: 1; overflow: hidden; }
.sidebar-moment img { position: absolute; width: 180px; height: 270px; bottom: -66px; left: -18px; object-fit: contain; opacity: .3; }
.sidebar-moment p { position: absolute; left: 28px; bottom: 16px; margin: 0; color: #e0e6ce; font-size: 20px; line-height: 1.9; letter-spacing: .12em; }
.sidebar-note { display: flex; gap: 10px; align-items: flex-start; margin: 12px 24px 24px; padding-top: 18px; border-top: 1px solid #41604e; color: #d0dcc7; font-size: 14px; line-height: 1.7; }
.sidebar-note i { margin-top: 5px; font-size: 16px; }
@media (max-width: 900px) {
  .care-sidebar { position: static; min-height: 0; border-top: 1px solid #dce4d6; background: #fff; padding-bottom: env(safe-area-inset-bottom); }
  .sidebar-label, .sidebar-note, .sidebar-moment, .nav-label-full { display: none; }
  .nav-label-short { display: inline; }
  .care-menu { display: grid; grid-template-columns: repeat(5, minmax(0, 1fr)); padding: 6px 6px; background: #fff; }
  .care-menu::before, .care-menu::after { display: none; }
  .nav-profile-icon { margin-right: 0; }
  .care-menu .el-menu-item { display: flex; flex-direction: column; justify-content: center; gap: 4px; height: auto; min-height: 64px; margin: 0 2px; padding: 6px 0 !important; color: #536450; line-height: 1.35; text-align: center; font-size: 16px; }
  .care-menu .el-menu-item i { margin: 0; color: #536450; font-size: 23px; }
  .care-menu .el-menu-item:hover, .care-menu .el-menu-item:focus { background: #f0f4ea; color: #183e33; }
  .care-menu .el-menu-item.is-active { background: #e9f0e2; }
  .care-menu .el-menu-item.is-active::after { content: ''; position: absolute; top: 0; left: 30%; right: 30%; height: 2px; border-radius: 2px; background: #235442; }
  .care-menu .el-menu-item:focus-visible { outline-color: #235442; }
}
</style>
