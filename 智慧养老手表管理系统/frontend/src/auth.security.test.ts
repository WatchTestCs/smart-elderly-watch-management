import { expect, it, vi } from 'vitest'
const mock = vi.hoisted(() => ({ request: null as any, responseError: null as any, get: vi.fn() }))
vi.mock('axios', () => ({ default: {
  defaults: {}, get: mock.get,
  interceptors: { request: { use: (fn: unknown) => { mock.request = fn } }, response: { use: (_success: unknown, error: unknown) => { mock.responseError = error } } }
} }))
import { authState, installHttpSecurity } from './auth'

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

it('refreshes a downgraded ADMIN duty on forbidden response without replaying the write', async () => {
  const router = { currentRoute: { value: { path: '/customer' } }, replace: vi.fn() }
  const app = { config: { globalProperties: {} } }
  authState.account = { role: 'ADMIN', duty: 'supervisor' } as any
  mock.get.mockResolvedValue({
    data: { csrfToken: 'rotated', authenticated: true, account: { role: 'ADMIN', duty: 'caregiver' } }
  })
  installHttpSecurity(router, app)
  const denied = { response: { status: 403, data: { code: 'ACCESS_DENIED' } } }
  await expect(mock.responseError(denied)).rejects.toBe(denied)
  await vi.waitFor(() => expect(router.replace).toHaveBeenCalledWith('/main'))
  expect(authState.account?.duty).toBe('caregiver')
  expect(mock.get).toHaveBeenCalledWith('/api/v1/auth/session', expect.objectContaining({ skipSession: true }))
})
