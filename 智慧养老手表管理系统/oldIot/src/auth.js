import { reactive } from 'vue'
import axios from 'axios'

export const authState = reactive({ account: null, csrfToken: '', csrfHeader: 'X-CSRF-TOKEN', setupRequired: false })
let loaded = false
let pending = null
const profileKeys = ['token', 'user_name', 'oldIdfamily', 'familyPhone', 'familyTouPic']

function clearProfile () { profileKeys.forEach(key => localStorage.removeItem(key)) }

export function loadSession (refresh) {
  if (!refresh && loaded) return Promise.resolve(authState.account)
  if (pending) return pending
  pending = axios.get('/restful/userLogin/session', { skipSession: true, timeout: 5000 }).then(response => {
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
  return axios.post('/restful/userLogin/logout', {}).then(() => {
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
    if (config.skipSession || url.origin !== window.location.origin || !url.pathname.startsWith('/restful/')) return config
    if (['get', 'head', 'options'].includes((config.method || 'get').toLowerCase())) return config
    return loadSession().then(() => {
      config.headers[authState.csrfHeader] = authState.csrfToken
      return config
    })
  })
  axios.interceptors.response.use(response => response, error => {
    if (error.response?.data?.code === 'DATABASE_UNAVAILABLE') {
      loaded = false
      if (router.currentRoute.value.matched.length && router.currentRoute.value.path !== '/connection') router.replace('/connection')
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
    return Promise.reject(error)
  })
}
