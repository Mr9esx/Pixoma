import { useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  createColumnHelper,
  getCoreRowModel,
  getSortedRowModel,
  useReactTable,
} from '@tanstack/react-table'
import { Copy, Eye, MoreHorizontal } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import {
  createMCPUser,
  deleteMCPUser,
  listMCPUsers,
  type MCPUser,
} from '@/lib/api/channels'
import { queryKeys } from '@/lib/api/query-keys'
import { fetchCurrentUser, fetchPlatformSettings } from '@/lib/api/setup'
import { getMCPToken, rotateMCPToken } from '@/lib/api/users'
import { formatDateTime } from '@/lib/format'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent } from '@/components/ui/dialog'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { Field, FieldGroup, FieldLabel } from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import { InputGroupButton } from '@/components/ui/input-group'
import { ConfirmDialog } from '@/components/confirm-dialog'
import { DataTableColumnHeader } from '@/components/data-table/column-header'
import { DataTable } from '@/components/data-table/data-table'
import { ErrorBanner } from '@/components/feedback/error-banner'
import { LoadingSkeleton } from '@/components/feedback/loading-skeleton'
import { SecretInput } from '@/components/secret-input'
import { SectionHead } from '@/components/section-head'
import {
  ResourceDetailLayout,
  type IdentifierItem,
} from '@/features/operations/detail-layout'
import { lifecycleBadgeClass } from '@/features/operations/identity'
import { resourceDetailDialogClassName } from '@/features/resource-modal'

function errorMessage(err: unknown): string | undefined {
  return err instanceof Error ? err.message : undefined
}

async function copyText(text: string) {
  await navigator.clipboard.writeText(text)
}

function mcpClientConfigJSON(url: string, token: string): string {
  return JSON.stringify(
    {
      mcpServers: {
        pixoma: {
          url,
          headers: { Authorization: `Bearer ${token}` },
        },
      },
    },
    null,
    2
  )
}

function isLoopbackURL(raw: string): boolean {
  try {
    const { hostname } = new URL(raw)
    return (
      hostname === 'localhost' ||
      hostname === '127.0.0.1' ||
      hostname === '::1' ||
      hostname === '0.0.0.0'
    )
  } catch {
    return false
  }
}

function accessLabel(access: string, t: (key: string) => string): string {
  if (access === 'always_allowed') return t('users.accessAlwaysAllowed')
  if (access === 'paid') return t('users.accessPaid')
  return t('users.accessDenied')
}

export function McpUsersSection({ channelId }: { channelId: string }) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [name, setName] = useState('')
  const [rotateUser, setRotateUser] = useState<MCPUser | null>(null)
  const [deleteUser, setDeleteUser] = useState<MCPUser | null>(null)
  const [detailUser, setDetailUser] = useState<MCPUser | null>(null)
  const [issued, setIssued] = useState<Record<string, string>>({})
  const [copyingId, setCopyingId] = useState<string | null>(null)

  const meQuery = useQuery({
    queryKey: ['current-user'],
    queryFn: fetchCurrentUser,
  })
  const canReveal =
    meQuery.data?.role === 'admin' || meQuery.data?.role === 'operator'

  const settingsQuery = useQuery({
    queryKey: queryKeys.settings.all,
    queryFn: fetchPlatformSettings,
  })
  const mcpURL = useMemo(() => {
    const publicURL = settingsQuery.data?.public_url
    const origin =
      publicURL && !isLoopbackURL(publicURL)
        ? publicURL.replace(/\/$/, '')
        : typeof window === 'undefined'
          ? ''
          : window.location.origin
    return origin ? `${origin}/mcp` : '/mcp'
  }, [settingsQuery.data?.public_url])

  const usersQuery = useQuery({
    queryKey: queryKeys.channels.mcpUsers(channelId),
    queryFn: () => listMCPUsers(channelId),
  })
  const users = useMemo(() => usersQuery.data ?? [], [usersQuery.data])

  const tokensQuery = useQuery({
    queryKey: [...queryKeys.channels.mcpUsers(channelId), 'tokens'],
    enabled: canReveal && usersQuery.isSuccess,
    queryFn: async () => {
      const entries = await Promise.all(
        (usersQuery.data ?? []).map(async (u) => {
          const res = await getMCPToken(u.id)
          return [u.id, res.token ?? ''] as const
        })
      )
      return Object.fromEntries(entries) as Record<string, string>
    },
  })
  const tokens = useMemo(
    () => ({ ...(tokensQuery.data ?? {}), ...issued }),
    [issued, tokensQuery.data]
  )

  const createMutation = useMutation({
    meta: { handledError: true },
    mutationFn: () => createMCPUser(channelId, name.trim()),
    onSuccess: (res) => {
      setName('')
      setIssued((prev) => ({ ...prev, [res.user.id]: res.token }))
      void queryClient.invalidateQueries({
        queryKey: queryKeys.channels.mcpUsers(channelId),
      })
    },
    onError: (err) => {
      toast.error(errorMessage(err) ?? t('common.errorGeneric'))
    },
  })

  const rotateMutation = useMutation({
    meta: { handledError: true },
    mutationFn: (userId: string) => rotateMCPToken(userId),
    onSuccess: (res, userId) => {
      setIssued((prev) => ({ ...prev, [userId]: res.token }))
      setRotateUser(null)
    },
    onError: (err) => {
      toast.error(errorMessage(err) ?? t('common.errorGeneric'))
    },
  })

  const deleteMutation = useMutation({
    meta: { handledError: true },
    mutationFn: (user: MCPUser) => deleteMCPUser(channelId, user.id),
    onSuccess: (_res, user) => {
      setIssued((prev) => {
        const next = { ...prev }
        delete next[user.id]
        return next
      })
      setDeleteUser(null)
      setDetailUser((current) => (current?.id === user.id ? null : current))
      void queryClient.invalidateQueries({
        queryKey: queryKeys.channels.mcpUsers(channelId),
      })
    },
    onError: (err) => {
      toast.error(errorMessage(err) ?? t('common.errorGeneric'))
    },
  })

  async function copyToken(userId: string) {
    const token = await resolveToken(userId)
    if (!token) return
    await copyText(token)
    toast.success(t('common.copied'))
  }

  async function copyClientConfig(userId: string) {
    const token = await resolveToken(userId)
    if (!token) return
    await copyText(mcpClientConfigJSON(mcpURL, token))
    toast.success(t('common.copied'))
  }

  async function resolveToken(userId: string): Promise<string> {
    const cached = tokens[userId]
    if (cached) return cached
    setCopyingId(userId)
    try {
      const res = await getMCPToken(userId)
      const token = res.token ?? ''
      if (!token) {
        toast.error(t('common.errorGeneric'))
        return ''
      }
      setIssued((prev) => ({ ...prev, [userId]: token }))
      return token
    } catch (err) {
      toast.error(errorMessage(err) ?? t('common.errorGeneric'))
      return ''
    } finally {
      setCopyingId(null)
    }
  }

  return (
    <section id='channel-mcp-users' className='flex flex-col gap-4'>
      <SectionHead title={t('channels.mcpUsers')} />
      <FieldGroup className='gap-4'>
        <Field>
          <FieldLabel htmlFor='mcp-url'>{t('channels.mcpURL')}</FieldLabel>
          <div className='flex flex-wrap gap-2'>
            <Input
              id='mcp-url'
              value={mcpURL}
              readOnly
              className='min-w-0 flex-1'
            />
            <Button
              type='button'
              variant='outline'
              onClick={() => {
                void copyText(mcpURL).then(() => {
                  toast.success(t('common.copied'))
                })
              }}
            >
              {t('common.copy')}
            </Button>
          </div>
        </Field>
        {canReveal ? (
          <Field>
            <FieldLabel htmlFor='mcp-user-name'>
              {t('channels.mcpUserName')}
            </FieldLabel>
            <div className='flex flex-wrap gap-2'>
              <Input
                id='mcp-user-name'
                value={name}
                onChange={(e) => setName(e.target.value)}
                autoComplete='off'
                className='min-w-0 flex-1'
              />
              <Button
                type='button'
                disabled={!name.trim() || createMutation.isPending}
                onClick={() => createMutation.mutate()}
              >
                {t('channels.mcpGenerateUser')}
              </Button>
            </div>
          </Field>
        ) : null}
      </FieldGroup>

      {usersQuery.isLoading ? <LoadingSkeleton rows={3} /> : null}
      {usersQuery.isError ? (
        <ErrorBanner
          message={errorMessage(usersQuery.error) ?? t('common.errorGeneric')}
          onRetry={() => void usersQuery.refetch()}
        />
      ) : null}
      {canReveal && tokensQuery.isError ? (
        <ErrorBanner
          message={errorMessage(tokensQuery.error) ?? t('common.errorGeneric')}
          onRetry={() => void tokensQuery.refetch()}
        />
      ) : null}
      {usersQuery.isSuccess ? (
        <McpUsersTable
          users={users}
          tokens={tokens}
          canReveal={canReveal}
          copyingId={copyingId}
          actionPending={rotateMutation.isPending || deleteMutation.isPending}
          onCopy={copyToken}
          onCopyClient={copyClientConfig}
          onDetail={setDetailUser}
          onRotate={setRotateUser}
          onDelete={setDeleteUser}
        />
      ) : null}

      <ConfirmDialog
        open={rotateUser != null}
        onOpenChange={(open) => {
          if (!open) setRotateUser(null)
        }}
        destructive
        isLoading={rotateMutation.isPending}
        title={t('channels.mcpRotateTitle')}
        desc={t('channels.mcpRotateDesc')}
        confirmText={t('channels.mcpRotate')}
        cancelBtnText={t('common.cancel')}
        handleConfirm={() => {
          if (rotateUser) rotateMutation.mutate(rotateUser.id)
        }}
      />

      <ConfirmDialog
        open={deleteUser != null}
        onOpenChange={(open) => {
          if (!open) setDeleteUser(null)
        }}
        destructive
        isLoading={deleteMutation.isPending}
        title={t('channels.mcpDeleteTitle', {
          name: deleteUser?.username || deleteUser?.id || '',
        })}
        desc={t('channels.mcpDeleteDesc')}
        confirmText={t('common.delete')}
        cancelBtnText={t('common.cancel')}
        handleConfirm={() => {
          if (deleteUser) deleteMutation.mutate(deleteUser)
        }}
      />

      <Dialog
        open={detailUser != null}
        onOpenChange={(open) => {
          if (!open) setDetailUser(null)
        }}
      >
        <DialogContent className={resourceDetailDialogClassName}>
          {detailUser ? <McpUserDetail user={detailUser} /> : null}
        </DialogContent>
      </Dialog>
    </section>
  )
}

function McpUserDetail({ user }: { user: MCPUser }) {
  const { t } = useTranslation()
  const title = user.username || user.id
  const accessClass =
    user.access === 'denied'
      ? 'border-destructive/25 bg-destructive/10 text-destructive'
      : lifecycleBadgeClass('succeeded')
  const identifiers: IdentifierItem[] = [
    {
      label: t('users.fieldExternalUserId'),
      value: user.external_user_id,
      copy: true,
    },
    {
      label: t('users.fieldCreatedAt'),
      value: formatDateTime(user.created_at),
    },
    {
      label: t('users.fieldUpdatedAt'),
      value: formatDateTime(user.updated_at),
    },
  ]

  return (
    <ResourceDetailLayout
      testId='mcp-user-detail'
      title={title}
      status={accessLabel(user.access, t)}
      statusClassName={accessClass}
      recordId={user.id}
      identifiers={identifiers}
    />
  )
}

function McpUsersTable({
  users,
  tokens,
  canReveal,
  copyingId,
  actionPending,
  onCopy,
  onCopyClient,
  onDetail,
  onRotate,
  onDelete,
}: {
  users: MCPUser[]
  tokens: Record<string, string>
  canReveal: boolean
  copyingId: string | null
  actionPending: boolean
  onCopy: (userId: string) => void
  onCopyClient: (userId: string) => void
  onDetail: (user: MCPUser) => void
  onRotate: (user: MCPUser) => void
  onDelete: (user: MCPUser) => void
}) {
  const { t } = useTranslation()
  const columnHelper = useMemo(() => createColumnHelper<MCPUser>(), [])
  const columns = useMemo(
    () => [
      columnHelper.accessor('username', {
        id: 'username',
        header: ({ column }) => (
          <DataTableColumnHeader
            column={column}
            title={t('channels.mcpUserName')}
          />
        ),
        cell: ({ getValue }) => (
          <span className='truncate text-sm'>{getValue() || '—'}</span>
        ),
      }),
      columnHelper.accessor('access', {
        id: 'access',
        header: ({ column }) => (
          <DataTableColumnHeader
            column={column}
            title={t('users.fieldAccess')}
          />
        ),
        cell: ({ getValue }) => {
          const access = getValue()
          return (
            <Badge
              variant={
                access === 'always_allowed'
                  ? 'default'
                  : access === 'paid'
                    ? 'secondary'
                    : 'destructive'
              }
            >
              {accessLabel(access, t)}
            </Badge>
          )
        },
      }),
      ...(canReveal
        ? [
            columnHelper.display({
              id: 'token',
              enableSorting: false,
              header: ({ column }) => (
                <DataTableColumnHeader
                  column={column}
                  title={t('channels.mcpToken')}
                />
              ),
              cell: ({ row }) => {
                const token = tokens[row.original.id] ?? ''
                return (
                  <SecretInput
                    value={token}
                    readOnly
                    autoComplete='off'
                    className='min-w-56'
                    endAddon={
                      <InputGroupButton
                        type='button'
                        variant='ghost'
                        size='icon-xs'
                        disabled={!token || copyingId === row.original.id}
                        onClick={() => onCopy(row.original.id)}
                      >
                        <Copy />
                        <span className='sr-only'>{t('common.copy')}</span>
                      </InputGroupButton>
                    }
                  />
                )
              },
            }),
          ]
        : []),
      columnHelper.accessor('created_at', {
        id: 'created_at',
        header: ({ column }) => (
          <DataTableColumnHeader
            column={column}
            title={t('users.fieldCreatedAt')}
          />
        ),
        cell: ({ getValue }) => (
          <span className='whitespace-nowrap text-muted-foreground tabular-nums'>
            {formatDateTime(getValue())}
          </span>
        ),
      }),
      ...(canReveal
        ? [
            columnHelper.display({
              id: 'actions',
              enableSorting: false,
              header: ({ column }) => (
                <DataTableColumnHeader
                  column={column}
                  title={t('common.actions')}
                />
              ),
              cell: ({ row }) => (
                <div className='flex items-center justify-end gap-1'>
                  <Button
                    type='button'
                    variant='outline'
                    size='sm'
                    onClick={() => onDetail(row.original)}
                  >
                    <Eye className='size-4' />
                    {t('common.detail')}
                  </Button>
                  <DropdownMenu>
                    <DropdownMenuTrigger asChild>
                      <Button
                        type='button'
                        variant='outline'
                        size='sm'
                        disabled={actionPending}
                        aria-label={t('common.moreActions')}
                      >
                        <MoreHorizontal className='size-4' />
                      </Button>
                    </DropdownMenuTrigger>
                    <DropdownMenuContent align='end' className='w-40'>
                      <DropdownMenuItem
                        onSelect={() => onCopyClient(row.original.id)}
                      >
                        {t('channels.mcpCopyClient')}
                      </DropdownMenuItem>
                      <DropdownMenuItem
                        onSelect={() => onRotate(row.original)}
                      >
                        {t('channels.mcpRotate')}
                      </DropdownMenuItem>
                      <DropdownMenuSeparator />
                      <DropdownMenuItem
                        className='text-destructive focus:text-destructive'
                        onSelect={() => onDelete(row.original)}
                      >
                        {t('common.delete')}
                      </DropdownMenuItem>
                    </DropdownMenuContent>
                  </DropdownMenu>
                </div>
              ),
            }),
          ]
        : []),
    ],
    [
      actionPending,
      canReveal,
      columnHelper,
      copyingId,
      onCopy,
      onCopyClient,
      onDelete,
      onDetail,
      onRotate,
      t,
      tokens,
    ]
  )

  const table = useReactTable({
    data: users,
    columns,
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
  })

  return (
    <DataTable
      table={table}
      hidePagination
      emptyState={
        <p className='text-sm text-muted-foreground'>
          {t('channels.mcpNoUsers')}
        </p>
      }
    />
  )
}
