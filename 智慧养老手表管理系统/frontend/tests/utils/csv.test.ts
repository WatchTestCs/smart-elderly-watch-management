import { describe, expect, it } from 'vitest'
import { toCsv } from '@/utils/csv'

describe('toCsv', () => {
  const columns = [{ key: 'name', label: '姓名' }, { key: 'note', label: '备注' }]

  it('writes a BOM, header and escaped cells', () => {
    const csv = toCsv(columns, [{ name: '张三', note: 'a,"b"\nc' }])
    expect(csv).toBe('﻿姓名,备注\r\n张三,"a,""b""\nc"')
  })

  it('neutralises formulas preceded by whitespace', () => {
    expect(toCsv(columns, [{ name: '  =1+1', note: 'ok' }])).toContain("\r\n'  =1+1,ok")
  })

  it('neutralises spreadsheet formulas and tolerates missing values', () => {
    const csv = toCsv(columns, [{ name: '=HYPERLINK("x")', note: null }, { name: '-1' }])
    expect(csv.split('\r\n').slice(1)).toEqual(['"\'=HYPERLINK(""x"")",', "'-1,"])
  })
})
