import { useState, type FormEvent } from 'react'
import { useNavigate, Link } from '@tanstack/react-router'
import { toast } from 'sonner'
import { registerAccount } from '@/lib/api/setup'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { AuthShell } from '@/features/setup/auth-shell'

function errorMessage(err: unknown): string | undefined {
  return err instanceof Error ? err.message : undefined
}

export function RegisterForm() {
  const navigate = useNavigate()
  const [username, setUsername] = useState('')
  const [email, setEmail] = useState('')
  const [nickname, setNickname] = useState('')
  const [password, setPassword] = useState('')
  const [confirm, setConfirm] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [pending, setPending] = useState(false)

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    if (username.trim().length < 1) {
      setError('账号名不能为空')
      return
    }
    if (password.length < 8) {
      setError('密码至少 8 位')
      return
    }
    if (password !== confirm) {
      setError('两次密码不一致')
      return
    }
    setPending(true)
    setError(null)
    try {
      await registerAccount({
        username: username.trim(),
        email: email.trim(),
        nickname: nickname.trim(),
        password,
      })
      toast.success('注册成功，已登录')
      await navigate({ to: '/' })
    } catch (err) {
      setError(errorMessage(err) ?? '注册失败')
    } finally {
      setPending(false)
    }
  }

  return (
    <AuthShell>
      <Card className='w-full max-w-md'>
        <CardHeader>
          <CardTitle>注册账号</CardTitle>
          <CardDescription>创建一个只读账号来访问 Pixoma 后台。</CardDescription>
        </CardHeader>
        <CardContent>
          <form className='flex flex-col gap-3' onSubmit={onSubmit} data-testid='register-form'>
            <div className='space-y-1.5'>
              <Label htmlFor='register-username'>账号名</Label>
              <Input
                id='register-username'
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                autoComplete='off'
                autoFocus
              />
            </div>
            <div className='space-y-1.5'>
              <Label htmlFor='register-email'>邮箱</Label>
              <Input
                id='register-email'
                type='email'
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                autoComplete='off'
              />
            </div>
            <div className='space-y-1.5'>
              <Label htmlFor='register-nickname'>昵称</Label>
              <Input
                id='register-nickname'
                value={nickname}
                onChange={(e) => setNickname(e.target.value)}
                autoComplete='off'
              />
            </div>
            <div className='space-y-1.5'>
              <Label htmlFor='register-password'>密码（至少 8 位）</Label>
              <Input
                id='register-password'
                type='password'
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                autoComplete='new-password'
              />
            </div>
            <div className='space-y-1.5'>
              <Label htmlFor='register-confirm'>再输一遍</Label>
              <Input
                id='register-confirm'
                type='password'
                value={confirm}
                onChange={(e) => setConfirm(e.target.value)}
                autoComplete='new-password'
              />
            </div>
            {error ? <p className='text-sm text-destructive'>{error}</p> : null}
            <Button type='submit' disabled={pending} data-testid='register-submit'>
              {pending ? '注册中…' : '注册'}
            </Button>
            <p className='text-center text-sm text-muted-foreground'>
              已有账号？{' '}
              <Link to='/login' className='text-primary hover:underline'>
                去登录
              </Link>
            </p>
          </form>
        </CardContent>
      </Card>
    </AuthShell>
  )
}
