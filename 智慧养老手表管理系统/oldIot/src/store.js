import { reactive } from 'vue'
const state = reactive({ openTab: [], activeIndex: '/main' })
export default {
  state,
  addTab (tab) { state.openTab.push(tab) },
  removeTab (route) { state.openTab = state.openTab.filter(tab => tab.route !== route) },
  clearTabs () { state.openTab = []; state.activeIndex = '/main' },
  activate (route) { state.activeIndex = route }
}
