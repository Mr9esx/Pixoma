import { describe, expect, it } from 'vitest'
import { formatDate, formatSize } from './studio-library'

describe('资产详情中的文件信息', () => {
  it('缺少或无效日期时保持空值显示', () => {
    expect(formatDate(null)).toBe('—')
    expect(formatDate('invalid')).toBe('—')
  })

  it('按字节数显示文件大小', () => {
    expect(formatSize(512)).toBe('512 B')
    expect(formatSize(1024)).toBe('1.00 KB')
    expect(formatSize(1024 ** 2)).toBe('1.00 MB')
    expect(formatSize(1024 ** 3)).toBe('1.00 GB')
  })
})
