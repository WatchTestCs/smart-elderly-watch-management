// One active read and one replaceable pending read. A superseded result or
// error cannot update the view. This scheduler must never wrap business writes.
export function createLatestRead<P, T>(
  read: (params: P) => Promise<T>,
  accept: (value: T, params: P) => void,
  reject: (error: unknown) => void,
  busy: (value: boolean) => void
) {
  let revision = 0
  let pending: { params: P; revision: number } | undefined
  let active: Promise<void> | undefined
  let disposed = false

  async function drain() {
    try {
      while (pending && !disposed) {
        const job = pending
        pending = undefined
        try {
          const value = await read(job.params)
          if (!disposed && job.revision === revision) accept(value, job.params)
        } catch (error) {
          if (!disposed && job.revision === revision) reject(error)
        }
      }
    } finally {
      active = undefined
      if (!disposed) busy(false)
    }
  }

  function run(params: P): Promise<void> {
    if (disposed) return Promise.resolve()
    pending = { params, revision: ++revision }
    if (!active) {
      busy(true)
      // Assign active before drain can invoke callbacks or synchronous readers.
      active = Promise.resolve().then(drain)
    }
    return active
  }

  function dispose() {
    disposed = true
    pending = undefined
    revision++
  }

  return { run, dispose }
}
