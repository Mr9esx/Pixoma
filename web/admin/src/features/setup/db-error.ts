export type AlertCopy = {
  key: string
  detail: string
}

const DB_ERROR_PATTERNS: Array<{ pattern: RegExp; key: string }> = [
  {
    pattern: /connection refused/i,
    key: 'setup.errDbRefused',
  },
  {
    pattern:
      /access denied|authentication failed|password authentication failed|pg_hba|1045|28000/i,
    key: 'setup.errDbAuth',
  },
  {
    pattern: /unknown database|1049|3d000/i,
    key: 'setup.errDbMissing',
  },
  {
    pattern: /role .* does not exist|user .* does not exist/i,
    key: 'setup.errDbUserMissing',
  },
  {
    pattern:
      /relation .* does not exist|schema .* does not exist|no such table|table already exists|42p01|42p07|1146|1050/i,
    key: 'setup.errDbTable',
  },
  {
    pattern: /does not exist/i,
    key: 'setup.errDbMissing',
  },
  {
    pattern: /timeout|i\/o timeout|deadline exceeded|timed out/i,
    key: 'setup.errDbTimeout',
  },
  {
    pattern:
      /no such host|no route to host|network is unreachable|could not translate host name|lookup .* no such host/i,
    key: 'setup.errDbHost',
  },
  {
    pattern:
      /connection reset|broken pipe|connection aborted|closed the connection unexpectedly/i,
    key: 'setup.errDbReset',
  },
  {
    pattern:
      /too many connections|too many clients|remaining connection slots|1040|53300/i,
    key: 'setup.errDbBusy',
  },
  {
    pattern: /database is locked|table is locked/i,
    key: 'setup.errDbLocked',
  },
  {
    pattern:
      /disk i\/o error|unable to open database file|no space left on device|disk full/i,
    key: 'setup.errDbDisk',
  },
  {
    pattern: /deadlock detected|lock wait timeout|40p01|55p03|1205|1213/i,
    key: 'setup.errDbDeadlock',
  },
  {
    pattern: /tls:|server does not support ssl|handshake|ssl error/i,
    key: 'setup.errDbSsl',
  },
  {
    pattern: /unknown db driver|unknown driver/i,
    key: 'setup.errDbDriver',
  },
  {
    pattern: /dsn required|empty database dsn/i,
    key: 'setup.errDbDsn',
  },
  {
    pattern: /invalid credentials/i,
    key: 'setup.errCredentials',
  },
  {
    pattern:
      /password must be at least 8 characters|password too short|weak password/i,
    key: 'setup.errPasswordShort',
  },
  {
    pattern: /password already set/i,
    key: 'setup.errPasswordSet',
  },
  {
    pattern: /configure database first/i,
    key: 'setup.errConfigureDb',
  },
  {
    pattern: /finalize setup first/i,
    key: 'setup.errFinalize',
  },
  {
    pattern: /save settings first/i,
    key: 'setup.errSaveFirst',
  },
  {
    pattern: /database ping failed|db: open:|db: empty dsn|ping failed/i,
    key: 'setup.errDbPing',
  },
  {
    pattern:
      /remote deployment cannot use blob\.driver=localfs|remote requires blob\.driver=s3 or tos/i,
    key: 'setup.errRemoteStorage',
  },
  {
    pattern: /localfs requires blob root/i,
    key: 'setup.errLocalRoot',
  },
  {
    pattern: /unknown placement|unknown blob\.driver|unknown proxy kind/i,
    key: 'setup.errInvalidValue',
  },
  {
    pattern: /proxy host required/i,
    key: 'setup.errProxyHost',
  },
  {
    pattern: /invalid proxy port/i,
    key: 'setup.errProxyPort',
  },
  {
    pattern: /unauthorized|not_initialized|restart_required/i,
    key: 'setup.errSession',
  },
]

export function setupErrorCopy(err: unknown): AlertCopy {
  const detail = err instanceof Error ? err.message : String(err)
  const matched = DB_ERROR_PATTERNS.find(({ pattern }) => pattern.test(detail))
  return {
    key: matched?.key ?? 'setup.errGeneric',
    detail,
  }
}
