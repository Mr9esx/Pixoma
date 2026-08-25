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
    pattern: /access denied|authentication failed|password authentication failed|pg_hba|1045|28000/i,
    title: '数据库用户名或密码错误',
  },
  {
    pattern: /unknown database|1049|3d000/i,
    title: '数据库不存在，请检查数据库名',
  },
  {
    pattern: /role .* does not exist|user .* does not exist/i,
    title: '数据库用户不存在，请检查用户名',
  },
  {
    pattern: /relation .* does not exist|schema .* does not exist|no such table|table already exists|42p01|42p07|1146|1050/i,
    title: '数据库表不存在，请检查库表结构',
  },
  {
    pattern: /does not exist/i,
    title: '数据库不存在，请检查数据库名',
  },
  {
    pattern: /timeout|i\/o timeout|deadline exceeded|timed out/i,
    title: '连接数据库超时，请检查网络和防火墙',
  },
  {
    pattern: /no such host|no route to host|network is unreachable|could not translate host name|lookup .* no such host/i,
    title: '无法解析数据库地址，请检查 Host',
  },
  {
    pattern: /connection reset|broken pipe|connection aborted|closed the connection unexpectedly/i,
    title: '数据库连接中断，请重试',
  },
  {
    pattern: /too many connections|too many clients|remaining connection slots|1040|53300/i,
    title: '数据库连接数已满，请稍后重试',
  },
  {
    pattern: /database is locked|table is locked/i,
    title: '数据库文件被占用，请稍后重试',
  },
  {
    pattern: /disk i\/o error|unable to open database file|no space left on device|disk full/i,
    title: '数据库文件异常，请检查磁盘和文件权限',
  },
  {
    pattern: /deadlock detected|lock wait timeout|40p01|55p03|1205|1213/i,
    title: '数据库锁冲突，请稍后重试',
  },
  {
    pattern: /tls:|server does not support ssl|handshake|ssl error/i,
    title: 'SSL 连接失败，请检查 SSL 模式',
  },
  {
    pattern: /unknown db driver|unknown driver/i,
    title: '不支持的数据库类型',
  },
  {
    pattern: /dsn required|empty database dsn/i,
    title: '请填写数据库连接信息',
  },
  {
    pattern: /invalid credentials/i,
    title: '账号或密码错误',
  },
  {
    pattern: /password must be at least 8 characters|password too short|weak password/i,
    title: '密码至少 8 位',
  },
  {
    pattern: /password already set/i,
    title: '密码已设置，请勿重复设置',
  },
  {
    pattern: /configure database first/i,
    title: '请先配置数据库',
  },
  {
    pattern: /finalize setup first/i,
    title: '请先完成向导初始化',
  },
  {
    pattern: /save settings first/i,
    title: '请先保存设置',
  },
  {
    pattern: /database ping failed|db: open:|db: empty dsn|ping failed/i,
    title: '数据库连接失败，请检查配置',
  },
  {
    pattern: /remote deployment cannot use blob\.driver=localfs|remote requires blob\.driver=s3 or tos/i,
    title: '远程部署必须使用 S3 或 TOS 存储',
  },
  {
    pattern: /localfs requires blob root/i,
    title: '本机存储缺少目录，请填写 blob root',
  },
  {
    pattern: /unknown placement|unknown blob\.driver|unknown proxy kind/i,
    title: '配置值不合法，请检查选项',
  },
  {
    pattern: /proxy host required/i,
    title: '代理缺少主机地址',
  },
  {
    pattern: /invalid proxy port/i,
    title: '代理端口不合法',
  },
  {
    pattern: /unauthorized|not_initialized|restart_required/i,
    title: '登录已失效，请重新登录',
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
