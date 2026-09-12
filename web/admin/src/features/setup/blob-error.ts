import type { AlertCopy } from './db-error'

const BLOB_ERROR_PATTERNS: Array<{ pattern: RegExp; key: string }> = [
  {
    pattern: /InvalidAccessKeyId|SignatureDoesNotMatch|AccessDenied/i,
    key: 'setup.errBlobKey',
  },
  {
    pattern: /NoSuchBucket|NoSuchKey/i,
    key: 'setup.errBlobBucket',
  },
  {
    pattern: /connection refused|i\/o timeout|timeout|no such host|unreachable/i,
    key: 'setup.errBlobNet',
  },
  {
    pattern: /empty (endpoint|region|bucket)|missing/i,
    key: 'setup.errBlobIncomplete',
  },
]

export function blobErrorCopy(err: unknown): AlertCopy {
  const detail = err instanceof Error ? err.message : String(err)
  const matched = BLOB_ERROR_PATTERNS.find(({ pattern }) => pattern.test(detail))
  return {
    key: matched?.key ?? 'setup.errBlobGeneric',
    detail,
  }
}
