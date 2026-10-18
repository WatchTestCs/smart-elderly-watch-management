<template>
  <main class="connection-page">
    <section class="connection-panel" aria-labelledby="connection-title">
      <img src="/static/brand/suining-logo-512.webp" width="112" height="112" alt="岁宁">
      <p class="connection-eyebrow">岁月有伴，日常安宁</p>
      <h1 id="connection-title">岁宁已打开，<br>等待数据连接。</h1>
      <p class="connection-description">{{ message }}</p>
      <p class="connection-status" role="status" aria-live="polite"><span aria-hidden="true"></span>{{ checking ? '正在检查连接' : '后台将自动重新连接' }}</p>
      <button type="button" :disabled="checking" @click="checkConnection">{{ checking ? '正在检查…' : '重新检查连接' }}</button>
      <p class="connection-note">数据库连接后将自动继续。暂时无法登录或查看业务记录，现有数据不会因此被清除。</p>
    </section>
  </main>
</template>

<script>
import axios from 'axios'
import { loadSession } from '../auth'
export default {
  data () {
    return { checking: false, message: '请启动数据库服务，或请管理员检查数据库连接配置。', timer: null, active: true }
  },
  mounted () { this.checkConnection() },
  beforeUnmount () { this.active = false; clearTimeout(this.timer) },
  methods: {
    async checkConnection () {
      if (this.checking || !this.active) return
      clearTimeout(this.timer)
      this.checking = true
      try {
        const { data } = await axios.get('/restful/service/status', { skipSession: true, timeout: 5000 })
        if (!this.active) return
        if (data.databaseReady) {
          const account = await loadSession(true)
          if (!this.active) return
          await this.$router.replace(account ? (account.role === 'FAMILY' ? '/familyMain' : '/main') : '/login')
        } else {
          this.message = '请启动数据库服务，或请管理员检查数据库连接配置。'
        }
      } catch (_) {
        if (this.active) this.message = '连接暂未恢复，请检查数据库服务与网络。岁宁会继续尝试连接。'
      } finally {
        this.checking = false
        if (this.active) this.timer = setTimeout(() => this.checkConnection(), 3000)
      }
    }
  }
}
</script>

<style scoped>
.connection-page { min-height: 100dvh; display: grid; place-items: center; padding: 32px 24px; background: var(--care-canvas, #f7f8f5); color: var(--care-ink, #263632); }
.connection-panel { width: 100%; max-width: 520px; padding: 40px 0; }
.connection-panel img { display: block; width: 112px; height: 112px; object-fit: contain; mix-blend-mode: multiply; }
.connection-eyebrow { margin: 24px 0 14px; font-size: 15px; color: var(--care-muted, #59665f); letter-spacing: .12em; }
h1 { margin: 0; font: inherit; font-size: clamp(30px, 5vw, 44px); line-height: 1.5; font-weight: 500; }
.connection-description { margin: 24px 0; max-width: 32em; font-size: 18px; line-height: 1.9; }
.connection-status { display: flex; align-items: center; gap: 10px; font-size: 15px; color: var(--care-muted, #59665f); }
.connection-status span { width: 8px; height: 8px; border-radius: 50%; background: #9b7a40; }
button { min-height: 52px; margin-top: 12px; padding: 12px 26px; border: 1px solid #235442; border-radius: 8px; background: #235442; color: #fff; cursor: pointer; }
button:disabled { opacity: .65; cursor: wait; }
button:hover:not(:disabled) { background: #183e33; }
.connection-note { margin: 28px 0 0; padding-top: 20px; border-top: 1px solid var(--care-line, #e5ebe7); font-size: 15px; color: var(--care-muted, #59665f); line-height: 1.8; }
@media (max-width: 480px) { .connection-panel { padding: 16px 0; } }
</style>
