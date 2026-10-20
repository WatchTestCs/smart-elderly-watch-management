import { expect, it, vi } from 'vitest'
const mock = vi.hoisted(() => ({ request: null as any, get: vi.fn() }))
vi.mock('axios', () => ({ default: {
  defaults: {}, get: mock.get,
  interceptors: { request: { use: (fn: unknown) => { mock.request = fn } }, response: { use: vi.fn() } }
} }))
import { installHttpSecurity } from './auth'

it('adds CSRF to versioned mutations without leaking it to another origin', async () => {
  mock.get.mockResolvedValue({ data: { csrfToken: 'session-csrf', authenticated: false } })
  installHttpSecurity({ currentRoute: { value: { path: '/login' } } }, { config: { globalProperties: {} } })
  const request = { url: '/api/v1/security/factor', method: 'post', headers: {} as Record<string, string> }
  await mock.request(request)
  expect(request.headers['X-CSRF-TOKEN']).toBe('session-csrf')
  const external = { url: 'https://other.example/api/v1/security/factor', method: 'post', headers: {} }
  await mock.request(external)
  expect(external.headers).toEqual({})
})
