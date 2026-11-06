import { createApp } from 'vue'
import App from './App.vue'
import DatabaseNotice from './components/DatabaseNotice.vue'
import PagerBar from './components/PagerBar.vue'
import { installDesktopKeys } from './desktop'
import router from './router'
import installElement from './element'
import { installHttpSecurity } from './auth'
import 'element-plus/dist/index.css'
import '@/assets/css/fonts.css'
import '@/assets/css/index.css'
import '@/assets/css/tokens.css'
import '@/assets/css/element-theme.css'
import '@/assets/css/accessibility.css'
import '@/assets/css/commercial.css'

const app = createApp(App)
app.component('database-notice', DatabaseNotice)
app.component('pager-bar', PagerBar)
installElement(app)
installHttpSecurity(router)
app.use(router)
app.mount('#app')
installDesktopKeys()
