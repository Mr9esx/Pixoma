import { describe, expect, it } from 'vitest'
import { ApiError } from '../../lib/api/client'
import { setupErrorCopy } from './db-error'

describe('setupErrorCopy', () => {
  it('maps connection refused to a friendly title with raw detail', () => {
    const copy = setupErrorCopy(
      new Error(
        'database unreachable: dial tcp 127.0.0.1:3306: connect: connection refused'
      )
    )
    expect(copy.key).toBe('setup.errDbRefused')
    expect(copy.detail).toContain('connection refused')
  })

  it('maps auth failures to a credentials title', () => {
    const copy = setupErrorCopy(
      new ApiError(400, 'database unreachable: Error 1045 (28000): Access denied')
    )
    expect(copy.key).toBe('setup.errDbAuth')
  })

  it('maps unknown database to a database-name title', () => {
    const copy = setupErrorCopy(
      new Error('database unreachable: Error 1049: Unknown database pixoma')
    )
    expect(copy.key).toBe('setup.errDbMissing')
  })

  it('maps timeouts to a network title', () => {
    const copy = setupErrorCopy(
      new Error('database unreachable: dial tcp 10.0.0.1:5432: i/o timeout')
    )
    expect(copy.key).toBe('setup.errDbTimeout')
  })

  it('maps unknown drivers to a driver title', () => {
    const copy = setupErrorCopy(new Error('db: unknown driver "oracle"'))
    expect(copy.key).toBe('setup.errDbDriver')
  })

  it('maps dns resolution failures to a host title', () => {
    const copy = setupErrorCopy(
      new Error(
        'database unreachable: dial tcp: lookup db.internal on 8.8.8.8: no such host'
      )
    )
    expect(copy.key).toBe('setup.errDbHost')
  })

  it('maps too many connections to a busy title', () => {
    const copy = setupErrorCopy(
      new Error('database unreachable: Error 1040: Too many connections')
    )
    expect(copy.key).toBe('setup.errDbBusy')
  })

  it('maps sqlite locks to a retry title', () => {
    const copy = setupErrorCopy(new Error('database is locked'))
    expect(copy.key).toBe('setup.errDbLocked')
  })

  it('maps deadlocks to a lock-conflict title', () => {
    const copy = setupErrorCopy(
      new Error(
        'database unreachable: ERROR: deadlock detected (SQLSTATE 40P01)'
      )
    )
    expect(copy.key).toBe('setup.errDbDeadlock')
  })

  it('maps missing roles to a username title', () => {
    const copy = setupErrorCopy(
      new Error('database unreachable: role "pixoma" does not exist')
    )
    expect(copy.key).toBe('setup.errDbUserMissing')
  })

  it('maps missing relations to a table title', () => {
    const copy = setupErrorCopy(
      new Error('database unreachable: relation "tasks" does not exist')
    )
    expect(copy.key).toBe('setup.errDbTable')
  })

  it('maps tls failures to an ssl title', () => {
    const copy = setupErrorCopy(
      new Error(
        'database unreachable: tls: first record does not look like a TLS handshake'
      )
    )
    expect(copy.key).toBe('setup.errDbSsl')
  })

  it('maps connection resets to an interrupted title', () => {
    const copy = setupErrorCopy(
      new Error(
        'database unreachable: read tcp 127.0.0.1:5432: connection reset by peer'
      )
    )
    expect(copy.key).toBe('setup.errDbReset')
  })

  it('maps login failures to a credentials title', () => {
    const copy = setupErrorCopy(new ApiError(401, 'invalid credentials'))
    expect(copy.key).toBe('setup.errCredentials')
  })

  it('maps weak passwords to a length title', () => {
    const copy = setupErrorCopy(
      new ApiError(400, 'password must be at least 8 characters')
    )
    expect(copy.key).toBe('setup.errPasswordShort')
  })

  it('maps missing database config to a step title', () => {
    const copy = setupErrorCopy(new ApiError(400, 'configure database first'))
    expect(copy.key).toBe('setup.errConfigureDb')
  })

  it('maps remote localfs to a storage title', () => {
    const copy = setupErrorCopy(
      new Error('settings: remote deployment cannot use blob.driver=localfs')
    )
    expect(copy.key).toBe('setup.errRemoteStorage')
  })

  it('maps auth expiry to a login title', () => {
    const copy = setupErrorCopy(new ApiError(401, 'unauthorized'))
    expect(copy.key).toBe('setup.errSession')
  })

  it('falls back to a generic title and keeps the raw message', () => {
    const copy = setupErrorCopy(new Error('boom'))
    expect(copy.key).toBe('setup.errGeneric')
    expect(copy.detail).toBe('boom')
  })
})
