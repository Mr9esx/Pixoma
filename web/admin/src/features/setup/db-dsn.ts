export type MySQLFields = {
  host: string
  port: string
  user: string
  password: string
  database: string
  params?: string
}

export type PostgresFields = {
  host: string
  port: string
  user: string
  password: string
  database: string
  sslmode: string
  params?: string
}

export function buildSqliteDSN(path: string): string {
  return path.trim()
}

export function buildMySQLDSN(f: MySQLFields): string {
  const host = f.host.trim() || '127.0.0.1'
  const port = f.port.trim() || '3306'
  const user = f.user.trim()
  const database = f.database.trim()
  const base = `${user}:${f.password}@tcp(${host}:${port})/${database}?charset=utf8mb4&parseTime=True&loc=Local`
  const params = f.params?.trim().replace(/^&+/, '')
  return params ? `${base}&${params}` : base
}

export function buildPostgresDSN(f: PostgresFields): string {
  const host = f.host.trim() || '127.0.0.1'
  const port = f.port.trim() || '5432'
  const user = f.user.trim()
  const database = f.database.trim()
  const sslmode = f.sslmode.trim() || 'disable'
  const base = `host=${host} port=${port} user=${user} password=${f.password} dbname=${database} sslmode=${sslmode}`
  const params = f.params?.trim()
  return params ? `${base} ${params}` : base
}
