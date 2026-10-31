import { createRouter, createWebHashHistory } from 'vue-router'
import { loadSession, authState } from './auth'
const Login = () => import('./views/Login.vue')
const Register = () => import('./views/Register.vue')
const Home = () => import('./views/Home.vue')
const familyHome = () => import('./views/familyHome.vue')
const main = () => import('./views/main.vue')
const familyMain = () => import('./views/familyMain.vue')
const ratepre = () => import('./views/ratepre.vue')
const rateprefamily = () => import('./views/rateprefamily.vue')
const annex = () => import('./views/annex.vue')
const text = () => import('./views/text.vue')
const textTwo = () => import('./views/textTwo.vue')
const textMain = () => import('./views/textMain.vue')
const textMainUser = () => import('./views/textMainUser.vue')
const step = () => import('./views/step.vue')
const stepfamily = () => import('./views/stepfamily.vue')
const position = () => import('./views/position.vue')
const positionfamily = () => import('./views/positionfamily.vue')
const life = () => import('./views/life.vue')
const watch = () => import('./views/watch.vue')
const person = () => import('./views/person.vue')
const customer = () => import('./views/customer.vue')
const textOpt = () => import('./views/textOpt.vue')
const family = () => import('./views/family.vue')
const me = () => import('./views/me.vue')
const saidOperation = () => import('./views/saidOperation.vue')
const inviteCode = () => import('./views/inviteCode.vue')
const device = () => import('./views/device.vue')


const router = createRouter({
  history: createWebHashHistory(),
  routes: [
    { path: '/connection', redirect: '/login' },
    { path: '/setup', component: () => import('./views/Setup.vue') },
	  {path: '/',
	    redirect: '/login'
	  },
    {path: '/login',
      component: Login
    },
    // 家属凭邀请码自助注册（无需登录）
    {path: '/register',
      component: Register
    },
	{path: '/familyHome',
	  component: familyHome,
	  redirect: '/familyMain',
	  children: [{
				  path: '/familyMain',
				  meta: { title: '家属首页' }, name: 'familyMain',
				  component: familyMain
				},
				{
					path: '/rateprefamily',
					meta: { title: '心率血压' }, name: 'rateprefamily',
					component: rateprefamily
				},
				{
					path: '/stepfamily',
					meta: { title: '睡眠步数' }, name: 'stepfamily',
					component: stepfamily
				},
				{
					path: '/positionfamily',
					meta: { title: '位置信息' }, name: 'positionfamily',
					component: positionfamily
				},
				{
					path: '/me',
					meta: { title: '个人中心' }, name: 'me',
					component: me
				},
				{
					path: '/textMain',
					meta: { title: '公告内容' }, name: 'textMain',
					component: textMain
				}]
	},
    { path: '/home',
      component: Home,
      redirect: '/main',
      children: [
				{
				  path: '/main',
				  meta: { title: '运营总览' }, name: 'main',
				  component: main
				},
				{
					path: '/ratepre',
					meta: { title: '心率与血压' }, name: 'ratepre',
					component: ratepre
				},
				{
					path: '/step',
					meta: { title: '睡眠与步数' }, name: 'step',
					component: step
				},
				{
					path: '/position',
					meta: { title: '位置与告警' }, name: 'position',
					component: position
				},
				{
					path: '/life',
					meta: { title: '生活信息' }, name: 'life',
					component: life
				},
				{
					path: '/watch',
					meta: { title: '监测阈值档案' }, name: 'watch',
					component: watch
				},
				{
					path: '/person',
					meta: { title: '老人档案' }, name: 'person',
					component: person
				},
				{
					path: '/customer',
					meta: { title: '监护人信息' }, name: 'customer',
					component: customer
				},
				{
					path: '/text',
					meta: { title: '编辑公告' }, name: 'text',
					component: text
				},
				{
					path: '/textTwo',
					meta: { title: '发布公告' }, name: 'textTwo',
					component: textTwo
				},
				{
					path: '/textMainUser',
					meta: { title: '公告内容' }, name: 'textMainUser',
					component: textMainUser
				},
				{
					path: '/textOpt',
					meta: { title: '公告管理' }, name: 'textOpt',
					component: textOpt
				},
				{
				  path: '/annex',
				  meta: { title: '附件管理' }, name: 'annex',
				  component: annex
				},
				{
				  path: '/family',
				  meta: { title: '家属账号' }, name: 'family',
				  component: family
				},
				{
				  path: '/saidOperation',
				  meta: { title: '评论管理' }, name: 'saidOperation',
				  component: saidOperation
				},
				{
				  path: '/inviteCode',
				  meta: { title: '家属注册邀请码' }, name: 'inviteCode',
				  component: inviteCode
				},
				{
				  path: '/device',
				  meta: { title: '设备管理' }, name: 'device',
				  component: device
				}
      ]
    }
  ]
})

router.beforeEach((to, from, next) => {
  const publicPage = to.path === '/login' || to.path === '/register'
  loadSession().then(account => {
    if (authState.setupRequired) return next(to.path === '/setup' ? undefined : '/setup')
    if (to.path === '/setup') return next('/login')
    if (publicPage) return next()
    if (!account) return next('/login')
    const familyPage = to.matched.some(route => route.path === '/familyHome')
    if ((account.role === 'FAMILY') !== familyPage) return next(account.role === 'FAMILY' ? '/familyMain' : '/main')
    if (account.role === 'ADMIN' && ['caregiver', 'auditor'].includes(account.duty) && !['/main', '/home'].includes(to.path)) return next('/main')
    next()
  }).catch(error => {
    if (error.response?.data?.code === 'DATABASE_UNAVAILABLE' && authState.account) {
      const familyPage = to.matched.some(route => route.path === '/familyHome')
      if (!publicPage && to.path !== '/setup' && (authState.account.role === 'FAMILY') !== familyPage) return next(authState.account.role === 'FAMILY' ? '/familyMain' : '/main')
      return next(to.path === '/setup' ? '/login' : undefined)
    }
    next(publicPage || (to.path === '/setup' && error.response?.data?.code === 'DATABASE_UNAVAILABLE') ? undefined : '/login')
  })
})

export default router
