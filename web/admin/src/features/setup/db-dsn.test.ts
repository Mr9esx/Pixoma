import { describe, expect, it } from 'vitest'
import {
  buildMySQLDSN,
  buildPostgresDSN,
  buildSqliteDSN,
} from './db-dsn'

describe('buildSqliteDSN', () => {
  it('keeps the sqlite path trimmed', () => {
    expect(buildSqliteDSN('  data/app.db  ')).toBe('data/app.db')
  })
})

describe('buildMySQLDSN', () => {
  it('composes a gorm-compatible mysql DSN', () => {
    const dsn = buildMySQLDSN({
      host: 'db.internal',
      port: '3307',
      user: 'pixoma',
      password: 's3cret',
      database: 'pixoma',
    })
    expect(dsn).toBe(
      'pixoma:s3cret@tcp(db.internal:3307)/pixoma?charset=utf8mb4&parseTime=True&loc=Local',
    )
  })

  it('defaults host and port when empty', () => {
    const dsn = buildMySQLDSN({
      host: '',
      port: '',
      user: 'root',
      password: '',
      database: 'pixoma',
    })
    expect(dsn.startsWith('root:@tcp(127.0.0.1:3306)/pixoma?')).toBe(true)
  })

  it('appends extra params after the built-in query string', () => {
    const dsn = buildMySQLDSN({
      host: 'db.internal',
      port: '3306',
      user: 'pixoma',
      password: '',
      database: 'pixoma',
      params: 'timeout=5s&readTimeout=10s',
    })
    expect(dsn).toBe(
      'pixoma:@tcp(db.internal:3306)/pixoma?charset=utf8mb4&parseTime=True&loc=Local&timeout=5s&readTimeout=10s',
    )
  })

  it('strips a leading ampersand from extra params', () => {
    const dsn = buildMySQLDSN({
      host: 'db.internal',
      port: '3306',
      user: 'pixoma',
      password: '',
      database: 'pixoma',
      params: '&timeout=5s',
    })
    expect(dsn.endsWith('loc=Local&timeout=5s')).toBe(true)
  })
})

describe('buildPostgresDSN', () => {
  it('composes a postgres key=value DSN with sslmode', () => {
    const dsn = buildPostgresDSN({
      host: 'pg.internal',
      port: '5433',
      user: 'pixoma',
      password: 's3cret',
      database: 'pixoma',
      sslmode: 'require',
    })
    expect(dsn).toBe(
      'host=pg.internal port=5433 user=pixoma password=s3cret dbname=pixoma sslmode=require',
    )
  })

  it('defaults port and sslmode', () => {
    const dsn = buildPostgresDSN({
      host: '',
      port: '',
      user: 'pixoma',
      password: '',
      database: 'pixoma',
      sslmode: '',
    })
    expect(dsn).toBe(
      'host=127.0.0.1 port=5432 user=pixoma password= dbname=pixoma sslmode=disable',
    )
  })

  it('appends extra params as key=value pairs', () => {
    const dsn = buildPostgresDSN({
      host: 'pg.internal',
      port: '5432',
      user: 'pixoma',
      password: '',
      database: 'pixoma',
      sslmode: 'require',
      params: 'connect_timeout=10 application_name=pixoma',
    })
    expect(dsn).toBe(
      'host=pg.internal port=5432 user=pixoma password= dbname=pixoma sslmode=require connect_timeout=10 application_name=pixoma',
    )
  })
})
