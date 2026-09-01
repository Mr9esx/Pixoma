import { useState, type FormEvent } from 'react'
import { useNavigate, Link } from '@tanstack/react-router'
import { toast } from 'sonner'
import { registerAccount } from '@/lib/api/setup'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Field, FieldGroup, FieldLabel } from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import { SecretInput } from '@/components/secret-input'
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
          <CardDescription>
            创建一个只读账号来访问 Pixoma 后台。
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={onSubmit} data-testid='register-form'>
            <FieldGroup className='gap-3'>
              <Field>
                <FieldLabel htmlFor='register-username'>账号名</FieldLabel>
                <Input
                  id='register-username'
                  value={username}
                  onChange={(e) => setUsername(e.target.value)}
                  autoComplete='off'
                  autoFocus
                />
              </Field>
              <Field>
                <FieldLabel htmlFor='register-email'>邮箱</FieldLabel>
                <Input
                  id='register-email'
                  type='email'
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  autoComplete='off'
                />
              </Field>
              <Field>
                <FieldLabel htmlFor='register-nickname'>昵称</FieldLabel>
                <Input
                  id='register-nickname'
                  value={nickname}
                  onChange={(e) => setNickname(e.target.value)}
                  autoComplete='off'
                />
              </Field>
              <Field>
                <FieldLabel htmlFor='register-password'>
                  密码（至少 8 位）
                </FieldLabel>
                <SecretInput
                  id='register-password'
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  autoComplete='new-password'
                />
              </Field>
              <Field>
                <FieldLabel htmlFor='register-confirm'>再输一遍</FieldLabel>
                <SecretInput
                  id='register-confirm'
                  value={confirm}
                  onChange={(e) => setConfirm(e.target.value)}
                  autoComplete='new-password'
                />
              </Field>
            </FieldGroup>
            {error ? <p className='text-sm text-destructive'>{error}</p> : null}
            <Button
              type='submit'
              disabled={pending}
              data-testid='register-submit'
            >
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
