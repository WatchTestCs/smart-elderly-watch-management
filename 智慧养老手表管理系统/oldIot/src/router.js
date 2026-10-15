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
				  meta: { title: '家属首页' }, name: 'page-2973',
				  component: familyMain
				},
				{
					path: '/rateprefamily',
					meta: { title: '心率血压' }, name: 'page-3062',
					component: rateprefamily
				},
				{
					path: '/stepfamily',
					meta: { title: '睡眠步数' }, name: 'page-3150',
					component: stepfamily
				},
				{
					path: '/positionfamily',
					meta: { title: '位置信息' }, name: 'page-3239',
					component: positionfamily
				},
				{
					path: '/me',
					meta: { title: '个人中心' }, name: 'page-3320',
					component: me
				},
				{
					path: '/textMain',
					meta: { title: '公告内容' }, name: 'page-3395',
					component: textMain
				}]
	},
    { path: '/home',
      component: Home,
      redirect: '/main',
      children: [
				{
				  path: '/main',
				  meta: { title: '首页' }, name: 'page-3565',
				  component: main
				},
				{
					path: '/ratepre',
					meta: { title: '心率血压' }, name: 'page-3640',
					component: ratepre
				},
				{
					path: '/step',
					meta: { title: '睡眠步数' }, name: 'page-3716',
					component: step
				},
				{
					path: '/position',
					meta: { title: '位置信息' }, name: 'page-3793',
					component: position
				},
				{
					path: '/life',
					meta: { title: '生活信息' }, name: 'page-3870',
					component: life
				},
				{
					path: '/watch',
					meta: { title: '手表信息' }, name: 'page-3944',
					component: watch
				},
				{
					path: '/person',
					meta: { title: '人员信息' }, name: 'page-4020',
					component: person
				},
				{
					path: '/customer',
					meta: { title: '监护人信息' }, name: 'page-4099',
					component: customer
				},
				{
					path: '/text',
					meta: { title: '编辑公告' }, name: 'page-4177',
					component: text
				},
				{
					path: '/textTwo',
					meta: { title: '发布公告' }, name: 'page-4253',
					component: textTwo
				},
				{
					path: '/textMainUser',
					meta: { title: '公告内容' }, name: 'page-4337',
					component: textMainUser
				},
				{
					path: '/textOpt',
					meta: { title: '公告管理' }, name: 'page-4421',
					component: textOpt
				},
				{
				  path: '/annex',
				  meta: { title: '附件管理' }, name: 'page-4500',
				  component: annex
				},
				{
				  path: '/family',
				  meta: { title: '家属管理' }, name: 'page-4579',
				  component: family
				},
				{
				  path: '/saidOperation',
				  meta: { title: '评论管理' }, name: 'page-4666',
				  component: saidOperation
				},
				{
				  path: '/inviteCode',
				  meta: { title: '邀请码管理' }, name: 'page-4757',
				  component: inviteCode
				},
				{
				  path: '/device',
				  meta: { title: '设备管理' }, name: 'page-4842',
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
    next()
  }).catch(() => { next(publicPage ? undefined : '/login') })
})

export default router
