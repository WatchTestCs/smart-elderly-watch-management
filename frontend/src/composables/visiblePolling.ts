// Periodic reads run only while the document is visible. Callers retain
// their own request deadlines, cancellation and busy-state policies.
export interface VisiblePollingOptions {
  /** Minimum elapsed milliseconds before a visibility-return refresh. */
  minGapMs?: number
}

export function startVisiblePolling(
  refresh: () => void,
  intervalMs: number,
  options: VisiblePollingOptions = {}
): () => void {
  const minGapMs = Math.max(0, options.minGapMs ?? Math.min(intervalMs, 1500))
  let timer: ReturnType<typeof setInterval> | undefined
  let lastRunAt = Date.now()

  function clearTimer() {
    if (timer !== undefined) clearInterval(timer)
    timer = undefined
  }

  function runRefresh() {
    lastRunAt = Date.now()
    refresh()
  }

  function tick() {
    if (!document.hidden) runRefresh()
  }

  function onVisibilityChange() {
    clearTimer()
    if (document.hidden) return
    timer = setInterval(tick, intervalMs)
    if (Date.now() - lastRunAt >= minGapMs) runRefresh()
  }

  if (!document.hidden) timer = setInterval(tick, intervalMs)
  document.addEventListener('visibilitychange', onVisibilityChange)
  return () => {
    clearTimer()
    document.removeEventListener('visibilitychange', onVisibilityChange)
  }
}
