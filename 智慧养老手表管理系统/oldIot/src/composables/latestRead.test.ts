import { describe, it, expect, vi } from 'vitest'
import { flushPromises } from '@vue/test-utils'
import { createLatestRead } from './latestRead'

function deferred<T>() {
  let resolve!: (value: T) => void
  let reject!: (error: unknown) => void
  const promise = new Promise<T>((yes, no) => { resolve = yes; reject = no })
  return { promise, resolve, reject }
}

describe('bounded latest read scheduler', () => {
  it('keeps one active read and coalesces a burst into its latest successor', async () => {
    const first = deferred<number>(), last = deferred<number>()
    const read = vi.fn<(query: number) => Promise<number>>().mockReturnValueOnce(first.promise).mockReturnValueOnce(last.promise)
    const accept = vi.fn(), fail = vi.fn(), busy = vi.fn()
    const queue = createLatestRead(read, accept, fail, busy)
    const pending = [queue.run(1)]
    await Promise.resolve()
    for (let query = 2; query <= 1000; query++) pending.push(queue.run(query))
    expect(read).toHaveBeenCalledTimes(1)
    first.resolve(1)
    await flushPromises()
    expect(accept).not.toHaveBeenCalled()
    expect(read.mock.calls).toEqual([[1], [1000]])
    last.resolve(1000)
    await Promise.all(pending)
    expect(accept.mock.calls).toEqual([[1000, 1000]])
    expect(fail).not.toHaveBeenCalled()
    expect(busy.mock.calls).toEqual([[true], [false]])
  })

  it('ignores stale failures and surfaces a current failure before retry', async () => {
    const first = deferred<number>()
    const read = vi.fn<(query: number) => Promise<number>>().mockReturnValueOnce(first.promise).mockRejectedValueOnce(new Error('current')).mockResolvedValueOnce(3)
    const accept = vi.fn(), fail = vi.fn()
    const queue = createLatestRead(read, accept, fail, vi.fn())
    const pending = queue.run(1)
    await Promise.resolve()
    queue.run(2)
    first.reject(new Error('stale'))
    await pending
    expect(fail.mock.calls.map(([error]) => error.message)).toEqual(['current'])
    await queue.run(3)
    expect(accept.mock.calls).toEqual([[3, 3]])
  })

  it('allows an accept callback to queue a corrected page without deadlock', async () => {
    const read = vi.fn(async (page: number) => page)
    const accepted: number[] = []
    const queue = createLatestRead(read, page => {
      if (page === 9) queue.run(2)
      else accepted.push(page)
    }, vi.fn(), vi.fn())
    await queue.run(9)
    expect(read.mock.calls).toEqual([[9], [2]])
    expect(accepted).toEqual([2])
  })

  it('handles a synchronous reader failure and can start again', async () => {
    const read = vi.fn<(query: number) => Promise<number>>().mockImplementationOnce(() => { throw new Error('synchronous') }).mockResolvedValueOnce(2)
    const accept = vi.fn(), fail = vi.fn(), busy = vi.fn()
    const queue = createLatestRead(read, accept, fail, busy)
    await queue.run(1)
    await queue.run(2)
    expect(fail).toHaveBeenCalledTimes(1)
    expect(accept.mock.calls).toEqual([[2, 2]])
    expect(busy.mock.calls).toEqual([[true], [false], [true], [false]])
  })

  it('discards pending reads and late callbacks after disposal', async () => {
    const first = deferred<number>()
    const read = vi.fn(() => first.promise), accept = vi.fn(), fail = vi.fn(), busy = vi.fn()
    const queue = createLatestRead<number, number>(read, accept, fail, busy)
    const pending = queue.run(1)
    await Promise.resolve()
    queue.run(2)
    queue.dispose()
    first.reject(new Error('late failure'))
    await pending
    await queue.run(3)
    expect(read).toHaveBeenCalledTimes(1)
    expect(accept).not.toHaveBeenCalled()
    expect(fail).not.toHaveBeenCalled()
  })
})
