import { useState } from 'react'
import { useNavigate } from '@tanstack/react-router'
import { ApiError } from '@/lib/api/client'
import { loginAdmin } from '@/lib/api/setup'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { AuthShell } from './auth-shell'

export function LoginPage() {
  const navigate = useNavigate()
  const [username, setUsername] = useState('admin')
  const [password, setPassword] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [pending, setPending] = useState(false)

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault()
    setPending(true)
    setError(null)
    try {
      const res = await loginAdmin(username, password)
      if (!res.initialized) {
        await navigate({ to: '/setup' })
        return
      }
      await navigate({ to: '/' })
    } catch (err) {
      setError(err instanceof ApiError ? err.message : '登录失败')
    } finally {
      setPending(false)
    }
  }

  return (
    <AuthShell>
      <Card className='w-full max-w-sm'>
        <CardHeader>
          <CardTitle>登录</CardTitle>
          <CardDescription>
            用启动日志里的管理员账号。用户名一般是 admin。
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form className='flex flex-col gap-4' onSubmit={onSubmit}>
            <div className='flex flex-col gap-2'>
              <Label htmlFor='username'>用户名</Label>
              <Input
                id='username'
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                autoComplete='username'
              />
            </div>
            <div className='flex flex-col gap-2'>
              <Label htmlFor='password'>密码</Label>
              <Input
                id='password'
                type='password'
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                autoComplete='current-password'
              />
            </div>
            {error ? <p className='text-sm text-destructive'>{error}</p> : null}
            <Button type='submit' className='w-full' disabled={pending}>
              {pending ? '登录中…' : '登录'}
            </Button>
          </form>
        </CardContent>
      </Card>
    </AuthShell>
  )
}
