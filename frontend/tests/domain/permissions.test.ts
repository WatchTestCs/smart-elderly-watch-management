import { expect, it } from 'vitest'
import { isRestrictedAdmin } from '@/domain/permissions'

it('restricts caregiver and auditor administration views only', () => {
  expect(isRestrictedAdmin(null)).toBe(false)
  expect(isRestrictedAdmin({ role: 'FAMILY', duty: 'caregiver' })).toBe(false)
  expect(isRestrictedAdmin({ role: 'ADMIN', duty: 'supervisor' })).toBe(false)
  expect(isRestrictedAdmin({ role: 'ADMIN', duty: 'caregiver' })).toBe(true)
  expect(isRestrictedAdmin({ role: 'ADMIN', duty: 'auditor' })).toBe(true)
})
