import type { SessionAccount } from '../auth'

// Mirrors the restricted administration UI policy; the server still authorizes each request.
export function isRestrictedAdmin(account: SessionAccount | null): boolean {
  return account?.role === 'ADMIN' &&
    (account.duty === 'caregiver' || account.duty === 'auditor')
}
