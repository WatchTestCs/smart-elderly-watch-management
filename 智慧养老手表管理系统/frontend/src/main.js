import { createApp } from 'vue'
import App from './App.vue'
import DatabaseNotice from './components/DatabaseNotice.vue'
import PagerBar from './components/PagerBar.vue'
import { installDesktopKeys } from './desktop'
import router from './router'
import store from './store'
import installElement from './element'
import { installHttpSecurity } from './auth'
import { recoverFromChunkError } from './appGuards'
import 'element-plus/dist/index.css'
import './assets/css/icons.css'
import '@/assets/css/fonts.css'
import '@/assets/css/index.css'
import '@/assets/css/tokens.css'
import '@/assets/css/element-theme.css'
import '@/assets/css/accessibility.css'
import '@/assets/css/commercial.css'

const app = createApp(App)
app.component('database-notice', DatabaseNotice)
app.component('pager-bar', PagerBar)
app.config.globalProperties.$store = store
installElement(app)
installHttpSecurity(router, app)
// 兜底：组件渲染/生命周期里未捕获的错误不再静默，页面脚本加载失败时自动刷新到新版本。
app.config.errorHandler = (error, _instance, info) => {
  console.error('[app]', info, error)
  if (recoverFromChunkError(error)) return
  app.config.globalProperties.$message.error({ message: '页面出现异常，请刷新后重试；若持续出现请联系管理员。', grouping: true })
}
app.use(router)
app.mount('#app')
installDesktopKeys()
