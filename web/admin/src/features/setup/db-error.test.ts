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
    expect(copy.title).toBe('连接数据库超时，请检查网络和防火墙')
  })

  it('maps unknown drivers to a driver title', () => {
    const copy = setupErrorCopy(new Error('db: unknown driver "oracle"'))
    expect(copy.title).toBe('不支持的数据库类型')
  })

  it('maps dns resolution failures to a host title', () => {
    const copy = setupErrorCopy(
      new Error('database unreachable: dial tcp: lookup db.internal on 8.8.8.8: no such host'),
    )
    expect(copy.title).toBe('无法解析数据库地址，请检查 Host')
  })

  it('maps too many connections to a busy title', () => {
    const copy = setupErrorCopy(
      new Error('database unreachable: Error 1040: Too many connections'),
    )
    expect(copy.title).toBe('数据库连接数已满，请稍后重试')
  })

  it('maps sqlite locks to a retry title', () => {
    const copy = setupErrorCopy(new Error('database is locked'))
    expect(copy.title).toBe('数据库文件被占用，请稍后重试')
  })

  it('maps deadlocks to a lock-conflict title', () => {
    const copy = setupErrorCopy(
      new Error('database unreachable: ERROR: deadlock detected (SQLSTATE 40P01)'),
    )
    expect(copy.title).toBe('数据库锁冲突，请稍后重试')
  })

  it('maps missing roles to a username title', () => {
    const copy = setupErrorCopy(
      new Error('database unreachable: role "pixoma" does not exist'),
    )
    expect(copy.title).toBe('数据库用户不存在，请检查用户名')
  })

  it('maps missing relations to a table title', () => {
    const copy = setupErrorCopy(
      new Error('database unreachable: relation "tasks" does not exist'),
    )
    expect(copy.title).toBe('数据库表不存在，请检查库表结构')
  })

  it('maps tls failures to an ssl title', () => {
    const copy = setupErrorCopy(
      new Error('database unreachable: tls: first record does not look like a TLS handshake'),
    )
    expect(copy.title).toBe('SSL 连接失败，请检查 SSL 模式')
  })

  it('maps connection resets to an interrupted title', () => {
    const copy = setupErrorCopy(
      new Error('database unreachable: read tcp 127.0.0.1:5432: connection reset by peer'),
    )
    expect(copy.title).toBe('数据库连接中断，请重试')
  })

  it('maps login failures to a credentials title', () => {
    const copy = setupErrorCopy(new ApiError(401, 'invalid credentials'))
    expect(copy.title).toBe('账号或密码错误')
  })

  it('maps weak passwords to a length title', () => {
    const copy = setupErrorCopy(
      new ApiError(400, 'password must be at least 8 characters'),
    )
    expect(copy.title).toBe('密码至少 8 位')
  })

  it('maps missing database config to a step title', () => {
    const copy = setupErrorCopy(new ApiError(400, 'configure database first'))
    expect(copy.title).toBe('请先配置数据库')
  })

  it('maps remote localfs to a storage title', () => {
    const copy = setupErrorCopy(
      new Error('settings: remote deployment cannot use blob.driver=localfs'),
    )
    expect(copy.title).toBe('远程部署必须使用 S3 或 TOS 存储')
  })

  it('maps auth expiry to a login title', () => {
    const copy = setupErrorCopy(new ApiError(401, 'unauthorized'))
    expect(copy.title).toBe('登录已失效，请重新登录')
  })

  it('falls back to a generic title and keeps the raw message', () => {
    const copy = setupErrorCopy(new Error('boom'))
    expect(copy.title).toBe('操作失败，请重试')
    expect(copy.detail).toBe('boom')
  })
})
