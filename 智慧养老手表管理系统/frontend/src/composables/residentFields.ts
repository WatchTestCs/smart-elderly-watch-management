export const residentFields = [
  'name', 'birthdate', 'sex', 'phone', 'room',
  'illness', 'warn', 'foodLike', 'foodHate', 'drug', 'guardianId'
] as const

export function editableResident(draft: Record<string, unknown>): Record<string, unknown> {
  return Object.fromEntries(residentFields.map(field => [field, draft[field]]))
}
