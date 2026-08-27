import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
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
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
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
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from '@/components/ui/alert-dialog'
import { LoadingSkeleton } from '@/components/feedback/loading-skeleton'
import { ErrorBanner } from '@/components/feedback/error-banner'

function errorMessage(err: unknown): string | undefined {
  return err instanceof Error ? err.message : undefined
}

function roleLabel(role: AdminUser['role']): string {
  if (role === 'admin') return '管理员'
  if (role === 'operator') return '操作员'
  return '只读'
}

export function AdminUsersPanel() {
  const queryClient = useQueryClient()
  const [query, setQuery] = useState('')

  const q = useQuery({
    queryKey: queryKeys.adminUsers.all,
    queryFn: () => listAdminUsers({ q: query.trim() || undefined }),
  })

  const invalidate = () =>
    queryClient.invalidateQueries({ queryKey: queryKeys.adminUsers.all })

  const enableMutation = useMutation({
    mutationFn: (u: AdminUser) =>
      updateAdminUser(u.id, { enabled: !u.enabled }),
    onSuccess: () => {
      void invalidate()
      toast.success('已更新账号状态')
    },
    onError: (err) => toast.error(errorMessage(err) ?? '操作失败'),
  })

  const deleteMutation = useMutation({
    mutationFn: (id: string) => deleteAdminUser(id),
    onSuccess: () => {
      void invalidate()
      toast.success('已删除账号')
    },
    onError: (err) => toast.error(errorMessage(err) ?? '删除失败'),
  })

  const users = q.data ?? []

  return (
    <section className='flex flex-col gap-4' data-testid='admin-users-panel'>
      <div className='flex flex-wrap items-center gap-2'>
        <Input
          data-testid='admin-users-search'
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder='搜索账号名或邮箱'
          className='max-w-xs'
          aria-label='搜索用户'
        />
        <CreateUserDialog onCreated={() => void invalidate()} />
      </div>

      {q.isLoading ? <LoadingSkeleton rows={4} /> : null}
      {q.isError ? <ErrorBanner message={errorMessage(q.error)} /> : null}
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>账号</TableHead>
            <TableHead>昵称</TableHead>
            <TableHead>邮箱</TableHead>
            <TableHead>角色</TableHead>
            <TableHead>状态</TableHead>
            <TableHead className='text-right'>操作</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {users.map((u) => (
            <TableRow key={u.id}>
              <TableCell className='font-medium'>{u.username}</TableCell>
              <TableCell>{u.nickname || '—'}</TableCell>
              <TableCell>{u.email || '—'}</TableCell>
              <TableCell>{roleLabel(u.role)}</TableCell>
              <TableCell>
                <Badge variant={u.enabled ? 'default' : 'outline'}>
                  {u.enabled ? '启用' : '禁用'}
                </Badge>
              </TableCell>
              <TableCell className='text-right'>
                <div className='flex flex-wrap justify-end gap-1'>
                  <Button
                    size='sm'
                    variant='outline'
                    onClick={() => enableMutation.mutate(u)}
                    data-testid={`toggle-${u.username}`}
                  >
                    {u.enabled ? '禁用' : '启用'}
                  </Button>
                  <ResetPasswordDialog user={u} onDone={() => void invalidate()} />
                  <AlertDialog>
                    <AlertDialogTrigger asChild>
                      <Button
                        size='sm'
                        variant='destructive'
                        data-testid={`delete-${u.username}`}
                      >
                        删除
                      </Button>
                    </AlertDialogTrigger>
                    <AlertDialogContent>
                      <AlertDialogHeader>
                        <AlertDialogTitle>删除账号 {u.username}？</AlertDialogTitle>
                        <AlertDialogDescription>
                          删除后不可恢复。若这是唯一的管理员，后端会拒绝删除。
                        </AlertDialogDescription>
                      </AlertDialogHeader>
                      <AlertDialogFooter>
                        <AlertDialogCancel>取消</AlertDialogCancel>
                        <AlertDialogAction
                          onClick={() => deleteMutation.mutate(u.id)}
                        >
                          确认删除
                        </AlertDialogAction>
                      </AlertDialogFooter>
                    </AlertDialogContent>
                  </AlertDialog>
                </div>
              </TableCell>
            </TableRow>
          ))}
          {!q.isLoading && !q.isError && users.length === 0 ? (
            <TableRow>
              <TableCell colSpan={6} className='text-center text-muted-foreground'>
                暂无用户
              </TableCell>
            </TableRow>
          ) : null}
        </TableBody>
      </Table>
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
        <div className='flex flex-col gap-3'>
          <div className='space-y-1.5'>
            <Label htmlFor='admin-user-username'>账号名</Label>
            <Input
              id='admin-user-username'
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              autoComplete='off'
            />
          </div>
          <div className='space-y-1.5'>
            <Label htmlFor='admin-user-email'>邮箱</Label>
            <Input
              id='admin-user-email'
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              type='email'
              autoComplete='off'
            />
          </div>
          <div className='space-y-1.5'>
            <Label htmlFor='admin-user-nickname'>昵称</Label>
            <Input
              id='admin-user-nickname'
              value={nickname}
              onChange={(e) => setNickname(e.target.value)}
              autoComplete='off'
            />
          </div>
          <div className='space-y-1.5'>
            <Label htmlFor='admin-user-password'>密码（至少 8 位）</Label>
            <Input
              id='admin-user-password'
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              type='password'
              autoComplete='new-password'
            />
          </div>
        </div>
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
  onDone,
}: {
  user: AdminUser
  onDone: () => void
}) {
  const [open, setOpen] = useState(false)
  const [password, setPassword] = useState('')

  const reset = useMutation({
    mutationFn: () => updateAdminUser(user.id, { password }),
    onSuccess: () => {
      toast.success('已重置密码并强制该用户下次改密')
      setOpen(false)
      setPassword('')
      onDone()
    },
    onError: (err) => toast.error(errorMessage(err) ?? '重置失败'),
  })

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button
          size='sm'
          variant='outline'
          data-testid={`reset-${user.username}`}
        >
          重置密码
        </Button>
      </DialogTrigger>
      <DialogContent className='sm:max-w-sm'>
        <DialogHeader>
          <DialogTitle>重置 {user.username} 的密码</DialogTitle>
          <DialogDescription>
            设置新密码后，该用户下次登录将被要求改密。
          </DialogDescription>
        </DialogHeader>
        <div className='space-y-1.5'>
          <Label htmlFor='admin-reset-password'>新密码（至少 8 位）</Label>
          <Input
            id='admin-reset-password'
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            type='password'
            autoComplete='new-password'
          />
        </div>
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
