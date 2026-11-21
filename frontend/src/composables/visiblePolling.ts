// Periodic reads run only while the document is visible. Callers retain
// their own request deadlines, cancellation and busy-state policies.
export function startVisiblePolling(refresh: () => void, intervalMs: number): () => void {
  let timer: ReturnType<typeof setInterval> | undefined

  function clearTimer() {
    if (timer !== undefined) clearInterval(timer)
    timer = undefined
  }

  function tick() {
    if (!document.hidden) refresh()
  }

  function onVisibilityChange() {
    clearTimer()
    if (document.hidden) return
    timer = setInterval(tick, intervalMs)
    refresh()
  }

  if (!document.hidden) timer = setInterval(tick, intervalMs)
  document.addEventListener('visibilitychange', onVisibilityChange)
  return () => {
    clearTimer()
    document.removeEventListener('visibilitychange', onVisibilityChange)
  }
}
