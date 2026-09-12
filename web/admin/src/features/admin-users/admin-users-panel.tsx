import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { MoreHorizontal } from 'lucide-react'
import { toast } from 'sonner'
import {
  createAdminUser,
  deleteAdminUser,
  listAdminUsers,
  updateAdminUser,
  type AdminUser,
} from '@/lib/api/admin-users'
import { queryKeys } from '@/lib/api/query-keys'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import {
  Field,
  FieldDescription,
  FieldGroup,
  FieldLabel,
} from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { ConfirmDialog } from '@/components/confirm-dialog'
import { ErrorBanner } from '@/components/feedback/error-banner'
import { LoadingSkeleton } from '@/components/feedback/loading-skeleton'
import { SecretInput } from '@/components/secret-input'
import { StatusDot } from '@/components/status-dot'

function errorMessage(err: unknown): string | undefined {
  return err instanceof Error ? err.message : undefined
}

const ROLE_FILTERS: Array<AdminUser['role'] | 'all'> = [
  'all',
  'admin',
  'operator',
  'viewer',
]

function roleBadge(
  role: AdminUser['role']
): 'default' | 'secondary' | 'outline' {
  if (role === 'admin') return 'default'
  if (role === 'operator') return 'secondary'
  return 'outline'
}

function roleLabelKey(
  role: AdminUser['role'] | 'all'
): 'adminUsers.roleAll' | 'adminUsers.roleAdmin' | 'adminUsers.roleOperator' | 'adminUsers.roleViewer' {
  if (role === 'all') return 'adminUsers.roleAll'
  if (role === 'admin') return 'adminUsers.roleAdmin'
  if (role === 'operator') return 'adminUsers.roleOperator'
  return 'adminUsers.roleViewer'
}

export function AdminUsersPanel() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [query, setQuery] = useState('')
  const [roleFilter, setRoleFilter] = useState<AdminUser['role'] | 'all'>('all')
  const [resetTarget, setResetTarget] = useState<AdminUser | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<AdminUser | null>(null)

  const q = useQuery({
    queryKey: [...queryKeys.adminUsers.all, query.trim()],
    queryFn: () => listAdminUsers({ q: query.trim() || undefined }),
  })

  const users = useMemo(
    () =>
      (q.data ?? []).filter(
        (u) => roleFilter === 'all' || u.role === roleFilter
      ),
    [q.data, roleFilter]
  )

  const invalidate = () =>
    queryClient.invalidateQueries({ queryKey: queryKeys.adminUsers.all })

  const enableMutation = useMutation({
    meta: { handledError: true },
    mutationFn: (u: AdminUser) =>
      updateAdminUser(u.id, { enabled: !u.enabled }),
    onSuccess: () => {
      void invalidate()
      toast.success(t('adminUsers.updated'))
    },
    onError: (err) => toast.error(errorMessage(err) ?? t('adminUsers.opFailed')),
  })

  const deleteMutation = useMutation({
    meta: { handledError: true },
    mutationFn: (id: string) => deleteAdminUser(id),
    onSuccess: () => {
      setDeleteTarget(null)
      void invalidate()
      toast.success(t('adminUsers.deleted'))
    },
    onError: (err) => toast.error(errorMessage(err) ?? t('adminUsers.deleteFailed')),
  })

  return (
    <section className='flex flex-col gap-4' data-testid='admin-users-panel'>
      <div className='flex flex-wrap items-center gap-2'>
        <Input
          data-testid='admin-users-search'
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder={t('adminUsers.searchPlaceholder')}
          className='max-w-xs'
          aria-label={t('a11y.searchUsers')}
        />
        <div
          role='group'
          aria-label={t('a11y.filterRole')}
          className='inline-flex items-center rounded-lg bg-muted p-0.5'
        >
          {ROLE_FILTERS.map((value) => (
            <Button
              key={value}
              type='button'
              size='sm'
              variant={roleFilter === value ? 'secondary' : 'ghost'}
              aria-pressed={roleFilter === value}
              className='min-h-8'
              onClick={() => setRoleFilter(value)}
            >
              {t(roleLabelKey(value))}
            </Button>
          ))}
        </div>
        <div className='ms-auto'>
          <CreateUserDialog onCreated={() => void invalidate()} />
        </div>
      </div>

      {q.isLoading ? <LoadingSkeleton rows={4} /> : null}
      {q.isError ? <ErrorBanner message={errorMessage(q.error)} /> : null}

      <Card className='overflow-hidden'>
        <CardContent className='p-0'>
          <Table data-testid='admin-users-table'>
            <TableHeader>
              <TableRow>
                <TableHead>{t('adminUsers.account')}</TableHead>
                <TableHead>{t('adminUsers.nickname')}</TableHead>
                <TableHead>{t('adminUsers.email')}</TableHead>
                <TableHead>{t('adminUsers.role')}</TableHead>
                <TableHead>{t('adminUsers.status')}</TableHead>
                <TableHead className='text-right'>{t('common.actions')}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {users.map((u) => (
                <TableRow key={u.id}>
                  <TableCell className='font-medium'>{u.username}</TableCell>
                  <TableCell>{u.nickname || '—'}</TableCell>
                  <TableCell>{u.email || '—'}</TableCell>
                  <TableCell>
                    <Badge variant={roleBadge(u.role)}>
                      {t(roleLabelKey(u.role))}
                    </Badge>
                  </TableCell>
                  <TableCell>
                    <StatusDot
                      problems={u.enabled ? 0 : 1}
                      label={
                        u.enabled
                          ? t('adminUsers.enabled')
                          : t('adminUsers.disabled')
                      }
                    />
                  </TableCell>
                  <TableCell className='text-right'>
                    <DropdownMenu>
                      <DropdownMenuTrigger asChild>
                        <Button
                          size='sm'
                          variant='ghost'
                          className='size-9'
                          aria-label={t('a11y.userActions', {
                            name: u.username,
                          })}
                        >
                          <MoreHorizontal className='size-4' />
                        </Button>
                      </DropdownMenuTrigger>
                      <DropdownMenuContent align='end' className='w-40'>
                        <DropdownMenuItem
                          onSelect={() => enableMutation.mutate(u)}
                        >
                          {u.enabled
                            ? t('adminUsers.disable')
                            : t('adminUsers.enable')}
                        </DropdownMenuItem>
                        <DropdownMenuItem onSelect={() => setResetTarget(u)}>
                          {t('adminUsers.resetPassword')}
                        </DropdownMenuItem>
                        <DropdownMenuSeparator />
                        <DropdownMenuItem
                          className='text-destructive focus:text-destructive'
                          onSelect={() => setDeleteTarget(u)}
                        >
                          {t('adminUsers.delete')}
                        </DropdownMenuItem>
                      </DropdownMenuContent>
                    </DropdownMenu>
                  </TableCell>
                </TableRow>
              ))}
              {!q.isLoading && !q.isError && users.length === 0 ? (
                <TableRow>
                  <TableCell
                    colSpan={6}
                    className='text-center text-muted-foreground'
                  >
                    {roleFilter === 'all'
                      ? t('adminUsers.empty')
                      : t('adminUsers.emptyRole')}
                  </TableCell>
                </TableRow>
              ) : null}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      <ResetPasswordDialog
        user={resetTarget}
        onClose={() => setResetTarget(null)}
        onDone={() => void invalidate()}
      />

      <ConfirmDialog
        open={deleteTarget !== null}
        onOpenChange={(open) => {
          if (!open) setDeleteTarget(null)
        }}
        destructive
        isLoading={deleteMutation.isPending}
        title={t('adminUsers.deleteTitle', {
          name: deleteTarget?.username ?? '',
        })}
        desc={t('adminUsers.deleteDesc')}
        confirmText={t('adminUsers.confirmDelete')}
        cancelBtnText={t('common.cancel')}
        handleConfirm={() => {
          if (deleteTarget) deleteMutation.mutate(deleteTarget.id)
        }}
      />
    </section>
  )
}

function CreateUserDialog({ onCreated }: { onCreated: () => void }) {
  const [open, setOpen] = useState(false)
  const [username, setUsername] = useState('')
  const [email, setEmail] = useState('')
  const [nickname, setNickname] = useState('')
  const [password, setPassword] = useState('')

  const create = useMutation({
    meta: { handledError: true },
    mutationFn: () =>
      createAdminUser({
        username: username.trim(),
        email: email.trim(),
        nickname: nickname.trim(),
        password,
      }),
    onSuccess: () => {
      toast.success('已新增用户')
      setOpen(false)
      setUsername('')
      setEmail('')
      setNickname('')
      setPassword('')
      onCreated()
    },
    onError: (err) => toast.error(errorMessage(err) ?? '新增失败'),
  })

  const canSubmit =
    username.trim().length > 0 && password.length >= 8 && !create.isPending

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button size='sm' data-testid='admin-users-create'>
          新增用户
        </Button>
      </DialogTrigger>
      <DialogContent className='sm:max-w-md'>
        <DialogHeader>
          <DialogTitle>新增用户</DialogTitle>
          <DialogDescription>
            新用户默认角色为「只读」，稍后可在列表里调整。
          </DialogDescription>
        </DialogHeader>
        <FieldGroup className='gap-4'>
          <Field>
            <FieldLabel htmlFor='admin-user-username'>账号名</FieldLabel>
            <Input
              id='admin-user-username'
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              autoComplete='off'
            />
          </Field>
          <Field>
            <FieldLabel htmlFor='admin-user-email'>邮箱</FieldLabel>
            <Input
              id='admin-user-email'
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              type='email'
              autoComplete='off'
            />
          </Field>
          <Field>
            <FieldLabel htmlFor='admin-user-nickname'>昵称</FieldLabel>
            <Input
              id='admin-user-nickname'
              value={nickname}
              onChange={(e) => setNickname(e.target.value)}
              autoComplete='off'
            />
          </Field>
          <Field>
            <FieldLabel htmlFor='admin-user-password'>
              密码（至少 8 位）
            </FieldLabel>
            <SecretInput
              id='admin-user-password'
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              autoComplete='new-password'
            />
          </Field>
        </FieldGroup>
        <DialogFooter>
          <DialogClose asChild>
            <Button type='button' variant='outline'>
              取消
            </Button>
          </DialogClose>
          <Button onClick={() => create.mutate()} disabled={!canSubmit}>
            {create.isPending ? '提交中…' : '新增'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

function ResetPasswordDialog({
  user,
  onClose,
  onDone,
}: {
  user: AdminUser | null
  onClose: () => void
  onDone: () => void
}) {
  const [password, setPassword] = useState('')

  const reset = useMutation({
    meta: { handledError: true },
    mutationFn: () =>
      user
        ? updateAdminUser(user.id, { password })
        : Promise.reject(new Error('missing user')),
    onSuccess: () => {
      toast.success('已重置密码并强制该用户下次改密')
      setPassword('')
      onClose()
      onDone()
    },
    onError: (err) => toast.error(errorMessage(err) ?? '重置失败'),
  })

  return (
    <Dialog open={user !== null} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className='sm:max-w-sm'>
        <DialogHeader>
          <DialogTitle>重置 {user?.username} 的密码</DialogTitle>
          <DialogDescription>
            设置新密码后，该用户下次登录将被要求改密。
          </DialogDescription>
        </DialogHeader>
        <FieldGroup>
          <Field>
            <FieldLabel htmlFor='admin-reset-password'>
              新密码（至少 8 位）
            </FieldLabel>
            <SecretInput
              id='admin-reset-password'
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              autoComplete='new-password'
            />
            <FieldDescription>
              设置新密码后，该用户下次登录将被要求改密。
            </FieldDescription>
          </Field>
        </FieldGroup>
        <DialogFooter>
          <DialogClose asChild>
            <Button type='button' variant='outline'>
              取消
            </Button>
          </DialogClose>
          <Button
            onClick={() => reset.mutate()}
            disabled={password.length < 8 || reset.isPending}
          >
            {reset.isPending ? '提交中…' : '重置'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
