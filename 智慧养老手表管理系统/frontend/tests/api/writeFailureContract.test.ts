import { afterEach, describe, expect, it, vi } from 'vitest'
import axios from 'axios'
import fs from 'node:fs'
import path from 'node:path'
import { flushPromises } from '@vue/test-utils'
import {
  apiErrorMessage, readErrorMessage, readTimeoutError,
  READ_TIMEOUT_MESSAGE, transportOutcome, UNCONFIRMED_MESSAGE
} from '@/api/httpContract'
import { createLatestRead } from '@/composables/latestRead'
import { noticeEditTarget } from '@/domain/notices'
import { authState, endSession } from '@/auth'

describe('classifying a write whose response was lost', () => {
  it('treats a raw-XHR upload with no HTTP response as unconfirmed', () => {
    const lost = { name: 'UploadAjaxError', status: 0, message: 'upload failed' }
    expect(transportOutcome(lost)).toBe('network')
    expect(apiErrorMessage(lost, '图片暂时无法上传，请重试')).toBe(UNCONFIRMED_MESSAGE)
  })

  it('keeps an upload business rejection on the server path', () => {
    expect(apiErrorMessage({
      name: 'UploadAjaxError', status: 400,
      message: JSON.stringify({ returnCode: '0000', code: 'INVALID_REQUEST', message: '图片格式不支持' })
    })).toBe('图片格式不支持')
  })

  it('never describes a read timeout as a possibly-applied write', () => {
    const error = readTimeoutError()
    expect(transportOutcome(error)).toBe('application')
    expect(readErrorMessage(error)).toBe(READ_TIMEOUT_MESSAGE)
    expect(apiErrorMessage(error, '数据暂时无法加载')).toBe('数据暂时无法加载')
  })

  it('carries the structured timeout through the read scheduler', async () => {
    vi.useFakeTimers()
    try {
      const read = () => new Promise<number>(() => {})
      const fail = vi.fn()
      const queue = createLatestRead(read, vi.fn(), fail, vi.fn(), { timeoutMs: 10 })
      const pending = queue.run(1)
      await vi.advanceTimersByTimeAsync(10)
      await pending
      expect(fail).toHaveBeenCalledTimes(1)
      expect(readErrorMessage(fail.mock.calls[0][0])).toBe(READ_TIMEOUT_MESSAGE)
    } finally {
      vi.useRealTimers()
      await flushPromises()
    }
  })
})

describe('session state after an unconfirmed logout', () => {
  afterEach(() => {
    vi.restoreAllMocks()
    axios.interceptors.request.clear()
    axios.interceptors.response.clear()
  })

  it('drops local credentials when the logout response is lost', async () => {
    authState.account = { role: 'ADMIN', id: 1 }
    authState.csrfToken = 'stale-token'
    vi.spyOn(axios, 'post').mockRejectedValue(Object.assign(new Error('timeout'), { code: 'ECONNABORTED' }))
    await expect(endSession()).rejects.toThrow('timeout')
    expect(authState.account).toBeNull()
    expect(authState.csrfToken).toBe('')
  })

  it('drops local credentials on an acknowledged logout too', async () => {
    authState.account = { role: 'FAMILY', id: 2 }
    authState.csrfToken = 'stale-token'
    vi.spyOn(axios, 'post').mockResolvedValue({ data: { returnCode: '1111' } })
    await endSession()
    expect(authState.account).toBeNull()
    expect(authState.csrfToken).toBe('')
  })
})

describe('binding a save to the record that was loaded', () => {
  it('refuses to save unless the displayed record still matches the route', () => {
    expect(noticeEditTarget(null, '17')).toBeNull()
    expect(noticeEditTarget('16', '17')).toBeNull()
    expect(noticeEditTarget('17', '17')).toBe('17')
  })
})

// The source tree is the only source of truth: a new catch block that tells the
// user to retry a write without going through the canonical error exit fails
// here, with no file list, no allowlist and no count to keep in sync.
function findSourceRoot(): string {
  let directory = process.cwd()
  for (let depth = 0; depth < 5; depth++) {
    const candidate = path.join(directory, 'src')
    if (fs.existsSync(path.join(candidate, 'api', 'client.ts'))) return candidate
    directory = path.dirname(directory)
  }
  throw new Error('frontend src directory not found')
}
const SRC = findSourceRoot()

function sourceFiles(directory: string): string[] {
  return fs.readdirSync(directory, { withFileTypes: true }).flatMap(entry => {
    const full = path.join(directory, entry.name)
    if (entry.isDirectory()) return sourceFiles(full)
    return /\.(?:ts|vue)$/.test(entry.name) ? [full] : []
  })
}

function catchBlocks(source: string): { index: number; text: string }[] {
  const blocks: { index: number; text: string }[] = []
  const pattern = /\bcatch\s*(?:\([^)]*\))?\s*\{/g
  let match: RegExpExecArray | null
  while ((match = pattern.exec(source))) {
    let depth = 1
    let end = pattern.lastIndex
    for (; end < source.length && depth > 0; end++) {
      const char = source[end]
      if (char === '{') depth++
      else if (char === '}') depth--
      else if (char === '"' || char === "'" || char === '`') {
        const quote = char
        end++
        while (end < source.length && source[end] !== quote) {
          if (source[end] === '\\') end++
          end++
        }
      }
    }
    blocks.push({ index: match.index, text: source.slice(match.index, end) })
    pattern.lastIndex = end
  }
  return blocks
}

describe('write failure exit guard', () => {
  it('routes every retry-inducing catch message through the canonical error exit', () => {
    const files = sourceFiles(SRC)
    expect(files.length).toBeGreaterThan(0)
    const canonical = /apiErrorMessage|readErrorMessage/
    const retryCopy = /(['"`])[^'"`\n]*(?:重试|稍后|检查网络|检查服务连接|刷新后)[^'"`\n]*\1/
    const violations: string[] = []
    for (const file of files) {
      const source = fs.readFileSync(file, 'utf8')
      for (const block of catchBlocks(source)) {
        if (canonical.test(block.text) || !retryCopy.test(block.text)) continue
        const line = source.slice(0, block.index).split('\n').length
        violations.push(`${path.relative(SRC, file)}:${line}`)
      }
    }
    expect(violations).toEqual([])
  })

  it('keeps one-shot setup failures on the staged classifier', () => {
    const consumers: string[] = []
    const classifier = /from ['"][^'"]*domain\/setup['"]/
    const violations: string[] = []
    for (const file of sourceFiles(SRC)) {
      const source = fs.readFileSync(file, 'utf8')
      if (!classifier.test(source)) continue
      consumers.push(path.relative(SRC, file))
      for (const block of catchBlocks(source)) {
        if (!/setupFailureMessage\s*\(/.test(block.text)) violations.push(path.relative(SRC, file))
      }
    }
    // Derived from imports, not a file list: dropping the classifier entirely
    // also fails this guard.
    expect(consumers.length).toBeGreaterThan(0)
    expect(violations).toEqual([])
  })
})
