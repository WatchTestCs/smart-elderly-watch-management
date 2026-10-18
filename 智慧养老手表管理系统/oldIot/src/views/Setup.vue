<template>
  <main class="setup-page">
    <section class="setup-story" aria-label="岁宁">
      <img src="/static/brand/suining-logo-512.webp" width="280" height="280" alt="岁宁山水标志">
      <p>岁月有伴，日常安宁</p>
      <h1>从此刻，<br>安放每一份牵挂。</h1>
    </section>
    <section class="setup-panel">
      <span class="setup-kicker">初次相见 · 岁宁</span>
      <h2>开启照护日常</h2>
      <database-notice><p class="setup-intro">创建管理员账号。业务记录保存在独立数据库中，关闭软件不会清除记录。</p></database-notice>
      <el-form ref="form" :model="account" label-position="top" :rules="rules" @submit.prevent="submit">
        <el-form-item label="怎么称呼您" prop="name"><el-input v-model="account.name" maxlength="10" autocomplete="name" /></el-form-item>
        <el-form-item label="登录手机号" prop="phone"><el-input v-model="account.phone" maxlength="11" inputmode="tel" autocomplete="username" /></el-form-item>
        <el-form-item label="设置密码" prop="password"><el-input v-model="account.password" type="password" show-password autocomplete="new-password" /></el-form-item>
        <p class="setup-hint">12–20 位，包含英文字母与数字。请妥善保管，软件不设通用密码。</p>
        <p v-if="error" role="alert" class="setup-error">{{ error }}</p>
        <el-button type="primary" native-type="submit" :loading="busy">创建账号，开始使用 <i class="el-icon-arrow-right" aria-hidden="true" /></el-button>
      </el-form>
      <p class="setup-footnote">您的健康与照护记录，始终由您掌握。</p>
    </section>
  </main>
</template>

<script>
import axios from 'axios'
import { loadSession } from '../auth'
import { createAdministrator } from '../desktop'
export default {
  data () {
    return {
      account: { name: '', phone: '', password: '' }, busy: false, error: '',
      rules: {
        name: [{ required: true, message: '请输入称呼', trigger: 'blur' }],
        phone: [{ pattern: /^1\d{10}$/, required: true, message: '请输入有效的 11 位手机号', trigger: 'blur' }],
        password: [{ pattern: /^(?=.*[A-Za-z])(?=.*\d).{12,20}$/, required: true, message: '请设置 12–20 位含字母和数字的密码', trigger: 'blur' }]
      }
    }
  },
  methods: {
    async submit () {
      if (this.busy || !await this.$refs.form.validate().catch(() => false)) return
      this.busy = true
      this.error = ''
      try {
        await createAdministrator({ ...this.account })
        await axios.post('/restful/userLogin/login', { phone: this.account.phone, password: this.account.password })
        this.account.password = ''
        const account = await loadSession(true)
        if (!account) throw new Error('Session missing')
        this.$router.replace('/main')
      } catch (e) {
        if (e.response?.data?.code === 'DATABASE_UNAVAILABLE') {
          this.error = '创建账号需要连接数据库，正在自动连接。恢复后请重试，填写内容已保留。'
          return
        }
        this.error = e.response?.status === 409 ? '管理员已创建，请返回登录页面。' : '暂时无法创建，请检查填写的信息后重试。'
      } finally { this.busy = false }
    }
  }
}
</script>

<style scoped>
.setup-page { min-height:100vh; display:grid; grid-template-columns:1fr 1fr; background:#f8f8f2; color:#183e33; }
.setup-story { display:flex; flex-direction:column; justify-content:center; align-items:center; padding:60px 40px; background:#edece1; text-align:center; }
.setup-story img { max-width:70%; height:auto; mix-blend-mode:multiply; }
.setup-story p { margin-top:36px; font-size:16px; letter-spacing:.15em; }
.setup-story h1 { margin:14px 0 0; font-size:38px; line-height:1.8; font-weight:500; }
.setup-panel { width:100%; max-width:550px; align-self:center; justify-self:center; padding:56px; }
.setup-kicker { font-size:14px; letter-spacing:.14em; color:#53645c; }
.setup-panel h2 { margin:18px 0; font-size:32px; font-weight:500; }
.setup-intro { color:#53645c; line-height:1.9; margin-bottom:30px; }
.setup-hint,.setup-footnote { color:#53645c; font-size:14px; line-height:1.8; }
.setup-footnote { margin:28px 0 0; }
.setup-error { color:#a4262c; line-height:1.7; }
.setup-panel :deep(.el-input__wrapper) { min-height:50px; padding:0 14px; }
.setup-panel :deep(.el-form-item__label) { font-size:16px; }
.setup-panel :deep(.el-button) { width:100%; min-height:54px; margin-top:18px; font-size:17px; }
.setup-panel :deep(.el-button i) { margin-left:16px; }
@media(max-width:760px) { .setup-page { grid-template-columns:1fr; }.setup-story { padding:26px 20px; }.setup-story img { width:90px; }.setup-story p { margin:12px 0 0; }.setup-story h1 { display:none; }.setup-panel { padding:32px 24px; } }
</style>
