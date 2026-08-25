import { describe, expect, it } from 'vitest'
import { ApiError } from '../../lib/api/client'
import { setupErrorCopy } from './db-error'

describe('setupErrorCopy', () => {
  it('maps connection refused to a friendly title with raw detail', () => {
    const copy = setupErrorCopy(
      new Error(
        'database unreachable: dial tcp 127.0.0.1:3306: connect: connection refused',
      ),
    )
    expect(copy.title).toBe('无法连接到数据库，请检查地址和端口')
    expect(copy.detail).toContain('connection refused')
  })

  it('maps auth failures to a credentials title', () => {
    const copy = setupErrorCopy(
      new ApiError(400, 'database unreachable: Error 1045 (28000): Access denied'),
    )
    expect(copy.title).toBe('数据库用户名或密码错误')
  })

  it('maps unknown database to a database-name title', () => {
    const copy = setupErrorCopy(
      new Error('database unreachable: Error 1049: Unknown database pixoma'),
    )
    expect(copy.title).toBe('数据库不存在，请检查数据库名')
  })

  it('maps timeouts to a network title', () => {
    const copy = setupErrorCopy(
      new Error('database unreachable: dial tcp 10.0.0.1:5432: i/o timeout'),
    )
    expect(copy.title).toBe('连接数据库超时，请检查网络与防火墙')
  })

  it('maps unknown drivers to a driver title', () => {
    const copy = setupErrorCopy(new Error('db: unknown driver "oracle"'))
    expect(copy.title).toBe('不支持的数据库类型')
  })

  it('falls back to a generic title and keeps the raw message', () => {
    const copy = setupErrorCopy(new Error('boom'))
    expect(copy.title).toBe('操作失败，请重试')
    expect(copy.detail).toBe('boom')
  })
})
