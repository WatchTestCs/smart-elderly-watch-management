import { defineComponent, h, nextTick } from 'vue'
import { mount } from '@vue/test-utils'
import { expect, it } from 'vitest'
import ErrorBoundary from '@/components/ErrorBoundary.vue'

it('contains a child render failure and remounts it on retry', async () => {
  let broken = true
  const Child = defineComponent(() => () => {
    if (broken) throw new Error('render failure')
    return h('p', '已恢复')
  })
  const wrapper = mount(ErrorBoundary, { slots: { default: () => h(Child) } })
  await nextTick()
  expect(wrapper.get('[role="alert"]').text()).toContain('当前页面暂时无法显示')
  broken = false
  await wrapper.get('button').trigger('click')
  expect(wrapper.text()).toBe('已恢复')
  wrapper.unmount()
})

it('keeps shell navigation mounted and clears a failure when the route key changes', async () => {
  const Broken = defineComponent(() => () => { throw new Error('bad child') })
  const Shell = defineComponent({
    props: { path: { type: String, required: true } },
    setup: props => () => h('main', [h('nav', '菜单'), h(ErrorBoundary, { key: props.path }, {
      default: () => props.path === '/broken' ? h(Broken) : h('p', '其他页面')
    })])
  })
  const wrapper = mount(Shell, { props: { path: '/broken' } })
  await nextTick()
  expect(wrapper.get('nav').text()).toBe('菜单')
  expect(wrapper.find('[role="alert"]').exists()).toBe(true)
  await wrapper.setProps({ path: '/healthy' })
  expect(wrapper.find('[role="alert"]').exists()).toBe(false)
  expect(wrapper.text()).toContain('其他页面')
  wrapper.unmount()
})
