import type { AlertCopy } from './db-error'

const BLOB_ERROR_PATTERNS: Array<{ pattern: RegExp; title: string }> = [
  {
    pattern: /InvalidAccessKeyId|SignatureDoesNotMatch|AccessDenied/i,
    title: '对象存储密钥错误，请检查 Access Key / Secret Key',
  },
  {
    pattern: /NoSuchBucket|NoSuchKey/i,
    title: 'bucket 或对象不存在，请检查 bucket 名称',
  },
  {
    pattern: /connection refused|i\/o timeout|timeout|no such host|unreachable/i,
    title: '连不上对象存储，请检查 endpoint 和网络',
  },
  {
    pattern: /empty (endpoint|region|bucket)|missing/i,
    title: '对象存储配置不完整，请填全必填项',
  },
]

/**
 * Maps an object-storage check error into an Alert-friendly copy:
 * a friendly Chinese title for common failures, plus the raw detail below.
 */
export function blobErrorCopy(err: unknown): AlertCopy {
  const detail = err instanceof Error ? err.message : String(err)
  const matched = BLOB_ERROR_PATTERNS.find(({ pattern }) => pattern.test(detail))
  return {
    title: matched?.title ?? '对象存储配置失败，请重试',
    detail,
  }
}
