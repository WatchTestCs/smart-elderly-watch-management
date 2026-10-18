import { createApp } from 'vue'
import App from './App.vue'
import DatabaseNotice from './components/DatabaseNotice.vue'
import { installDesktopKeys } from './desktop'
import router from './router'
import store from './store'
import installElement from './element'
import axios from 'axios'
import { installHttpSecurity } from './auth'
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
app.config.globalProperties.$axios = axios
app.config.globalProperties.$store = store
installElement(app)
installHttpSecurity(router, app)
app.use(router)
app.mount('#app')
installDesktopKeys()
