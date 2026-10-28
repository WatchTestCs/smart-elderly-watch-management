<template><notice-editor ref="editor" editing /></template>
<script>
import NoticeEditor from '@/components/NoticeEditor.vue'
export default {
  components: { NoticeEditor },
  beforeRouteLeave (to, from, next) { this.$refs.editor.confirmLeave(next) },
  beforeRouteUpdate (to, from, next) { this.$refs.editor.confirmLeave(next) }
}
</script>
