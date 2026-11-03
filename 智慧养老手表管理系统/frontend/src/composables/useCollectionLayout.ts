import { onBeforeUnmount, onMounted, ref } from 'vue'

// Keep this breakpoint aligned with the corresponding CSS layouts.
export function useCollectionLayout (breakpoint = 1100) {
  const query = window.matchMedia(`(max-width: ${breakpoint}px)`)
  const mobileLayout = ref(query.matches)
  const update = (event: MediaQueryListEvent) => { mobileLayout.value = event.matches }
  onMounted(() => {
    query.addEventListener('change', update)
    mobileLayout.value = query.matches
  })
  onBeforeUnmount(() => query.removeEventListener('change', update))
  return mobileLayout
}
