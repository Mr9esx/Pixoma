export type AlertCopy = {
  title: string
  detail: string
}

const DB_ERROR_PATTERNS: Array<{ pattern: RegExp; title: string }> = [
  {
    pattern: /connection refused/i,
    title: '无法连接到数据库，请检查地址和端口',
  },
  {
    pattern: /access denied|authentication failed|password authentication failed|1045|28000/i,
    title: '数据库用户名或密码错误',
  },
  {
    pattern: /unknown database|does not exist|1049|3d000/i,
    title: '数据库不存在，请检查数据库名',
  },
  {
    pattern: /timeout|i\/o timeout|deadline exceeded/i,
    title: '连接数据库超时，请检查网络与防火墙',
  },
  {
    pattern: /unknown db driver|unknown driver/i,
    title: '不支持的数据库类型',
  },
  {
    pattern: /dsn required|empty database dsn/i,
    title: '请填写数据库连接信息',
  },
]

/**
 * Maps a setup/database error into an Alert-friendly copy:
 * a friendly Chinese title for common failures, plus the raw detail below.
 */
export function setupErrorCopy(err: unknown): AlertCopy {
  const detail = err instanceof Error ? err.message : String(err)
  const matched = DB_ERROR_PATTERNS.find(({ pattern }) => pattern.test(detail))
  return {
    title: matched?.title ?? '操作失败，请重试',
    detail,
  }
}
