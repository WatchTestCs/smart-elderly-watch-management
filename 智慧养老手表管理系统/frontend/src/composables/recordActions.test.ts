import { beforeEach, expect, it, vi } from 'vitest'

const post = vi.hoisted(() => vi.fn())
vi.mock('../api/client', () => ({ api: { post }, accepted: () => undefined }))

import { writeRecord } from './recordActions'

beforeEach(() => post.mockReset())

it('ignores a duplicate write while the same request is in flight', async () => {
  let release: (v: unknown) => void = () => undefined
  post.mockReturnValue(new Promise(resolve => { release = resolve }))
  const view = { $message: vi.fn() }
  const refresh = vi.fn()
  const first = writeRecord(view, 'api/v1/x', { a: 1 }, 'ok', refresh)
  await writeRecord(view, 'api/v1/x', { a: 1 }, 'ok', refresh)
  expect(post).toHaveBeenCalledTimes(1)
  release({ data: {} })
  await first
  expect(refresh).toHaveBeenCalledTimes(1)
  post.mockResolvedValue({ data: {} })
  await writeRecord(view, 'api/v1/x', { a: 1 }, 'ok', refresh)
  expect(post).toHaveBeenCalledTimes(2)
})
