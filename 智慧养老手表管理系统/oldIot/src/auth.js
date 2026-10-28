import { reactive } from 'vue'
import axios from 'axios'

export const authState = reactive({ account: null, csrfToken: '', csrfHeader: 'X-CSRF-TOKEN', setupRequired: false, databaseReady: null, configError: '' })
let loaded = false
let pending = null
const profileKeys = ['token', 'user_name', 'oldIdfamily', 'familyPhone', 'familyTouPic']

function clearProfile () { profileKeys.forEach(key => localStorage.removeItem(key)) }

export function loadSession (refresh) {
  if (!refresh && loaded) return Promise.resolve(authState.account)
  if (pending) return pending
  pending = axios.get('/api/v1/auth/session', { skipSession: true, timeout: 5000 }).then(response => {
    const data = response.data
    if (!data || !data.csrfToken) throw new Error('Session response missing')
    authState.setupRequired = Boolean(data.setupRequired)
    authState.account = data.authenticated ? data.account : null
    authState.csrfToken = data.csrfToken
    authState.csrfHeader = data.csrfHeader || 'X-CSRF-TOKEN'
    loaded = true
    clearProfile()
    if (authState.account) {
      localStorage.setItem('user_name', authState.account.displayName || '用户')
      if (authState.account.role === 'FAMILY') {
        localStorage.setItem('oldIdfamily', authState.account.oldId)
        localStorage.setItem('familyPhone', authState.account.phone)
      }
    }
    return authState.account
  }).then(account => { pending = null; return account }, error => { pending = null; throw error })
  return pending
}

export function endSession () {
  return axios.post('/api/v1/auth/logout', {}, { skipSession: true, headers: { [authState.csrfHeader]: authState.csrfToken } }).then(() => {
    loaded = false
    authState.account = null
    authState.csrfToken = ''
    clearProfile()
  })
}

export function installHttpSecurity (router, app) {
  axios.defaults.withCredentials = true
  Object.defineProperty(app.config.globalProperties, '$uploadHeaders', { get () {
    return { [authState.csrfHeader]: authState.csrfToken }
  } })
  axios.interceptors.request.use(config => {
    const url = new URL(config.url, window.location.href)
    if (config.skipSession || url.origin !== window.location.origin || !(url.pathname.startsWith('/restful/') || url.pathname.startsWith('/api/v1/'))) return config
    if (['get', 'head', 'options'].includes((config.method || 'get').toLowerCase())) return config
    return loadSession().then(() => {
      config.headers[authState.csrfHeader] = authState.csrfToken
      return config
    })
  })
  axios.interceptors.response.use(response => response, error => {
    if (error.response?.data?.code === 'DATABASE_UNAVAILABLE') {
      loaded = false
      authState.databaseReady = false
      if (!error.config?.skipSession) app.config.globalProperties.$message.warning({ message: authState.configError
        ? '服务端配置有误，依赖数据库的功能不可用，请联系管理员修正后重启。'
        : '此功能需要连接数据库，正在自动连接。恢复后请重试。', grouping: true })
    }
    if (error.response && error.response.status === 401) {
      loaded = false
      authState.account = null
      authState.csrfToken = ''
      clearProfile()
      if (!['/login', '/register'].includes(router.currentRoute.value.path)) router.replace('/login')
    }
    if (error.response && error.response.status === 403 && error.response.data && error.response.data.code === 'CSRF_REJECTED') {
      loaded = false
      authState.csrfToken = ''
      // 只刷新令牌，不自动重放可能产生写入的请求。
      loadSession(true).catch(() => {})
    }
    if (error.response?.status === 403 && error.response.data?.code === 'ACCESS_DENIED' && authState.account?.role === 'ADMIN') {
      // A supervisor may change a duty while a page is open. Refresh the
      // session, but never replay the denied mutation automatically.
      loadSession(true).then(account => {
        if (account?.role === 'ADMIN' && ['caregiver', 'auditor'].includes(account.duty) && !['/main', '/home'].includes(router.currentRoute.value.path)) {
          router.replace('/main')
        }
      }).catch(() => {})
    }
    return Promise.reject(error)
  })
}


// One background probe across all pages; reconnecting never repeats business requests.
export function monitorDatabase (router) {
  let active = true
  let checking = false
  let timer
  async function check () {
    if (!active || checking) return
    clearTimeout(timer)
    checking = true
    const previous = authState.databaseReady
    try {
      const { data } = await axios.get('/restful/service/status', { skipSession: true, timeout: 5000 })
      if (!active) return
      if (typeof data.databaseReady !== 'boolean') throw new Error('Invalid database status')
      authState.databaseReady = data.databaseReady
      authState.configError = typeof data.configError === 'string' ? data.configError : ''
      if (!data.databaseReady) loaded = false
      if (data.databaseReady && previous === false) {
        const account = await loadSession(true)
        if (!active) return
        const route = router.currentRoute.value.path
        if (authState.setupRequired && route === '/login') await router.replace('/setup')
        else if (account && route === '/login') await router.replace(account.role === 'FAMILY' ? '/familyMain' : '/main')
        else if (!account && !['/login', '/register', '/setup'].includes(route)) await router.replace('/login')
      }
    } catch (_) {
      if (active) { authState.databaseReady = false; loaded = false }
    } finally {
      checking = false
      if (active) timer = setTimeout(check, 3000)
    }
  }
  window.addEventListener('online', check)
  check()
  return () => { active = false; clearTimeout(timer); window.removeEventListener('online', check) }
}
