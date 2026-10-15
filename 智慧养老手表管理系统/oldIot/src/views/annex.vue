<template>
  <section class="annex-page">
    <div class="page-heading">
      <div><p class="eyebrow">FILE CENTER</p><h1>附件管理</h1><p class="page-subtitle">集中查看和管理照护相关文件。</p></div>
    </div>
    <div class="unavailable-card" role="status">
      <span class="unavailable-icon"><i class="el-icon-document"></i></span>
      <h2>当前版本暂不支持附件管理</h2>
      <p>此版本没有可用的附件上传与列表接口。公告图片可在公告内容中查看。</p>
      <el-button plain @click="$router.push('/main')">返回运营总览</el-button>
    </div>
  </section>
</template>

<script>
export default { name: 'Annex' }
</script>

<style scoped>
.page-heading { margin-bottom: 20px; }
.eyebrow { margin: 0 0 8px; color: #64756c; font-size: 12px; font-weight: 600; letter-spacing: .16em; }
h1 { margin: 0; color: #263632; font-size: 26px; line-height: 1.25; font-weight: 600; }
.page-subtitle { margin: 9px 0 0; color: #718079; font-size: 13px; }
.unavailable-card { display: flex; flex-direction: column; align-items: center; padding: 64px 24px; border: 1px solid #e5ebe7; border-radius: 12px; background: #fff; text-align: center; }
.unavailable-icon { display: grid; place-items: center; width: 56px; height: 56px; border-radius: 16px; background: #f1f5f2; color: #718079; font-size: 25px; }
.unavailable-card h2 { margin: 20px 0 8px; color: #33453c; font-size: 17px; font-weight: 600; }
.unavailable-card p { max-width: 440px; margin: 0 0 20px; color: #718079; font-size: 13px; line-height: 1.7; }
.unavailable-card :deep(.el-button) { border-radius: 8px; }
@media (max-width: 600px) { .unavailable-card { padding: 44px 18px; } h1 { font-size: 23px; } }
</style>
