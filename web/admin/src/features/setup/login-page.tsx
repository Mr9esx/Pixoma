import { useState } from 'react'
import { useNavigate } from '@tanstack/react-router'
import { KeyRound } from 'lucide-react'
import { ApiError } from '@/lib/api/client'
import { loginAdmin, type SetupStatus } from '@/lib/api/setup'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Checkbox } from '@/components/ui/checkbox'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { AuthShell } from './auth-shell'

export function LoginPage({ status }: { status: SetupStatus }) {
  const navigate = useNavigate()
  const [username, setUsername] = useState('admin')
  const [password, setPassword] = useState('')
  const [remember, setRemember] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [pending, setPending] = useState(false)
  const showFirstRunHint = !status.initialized && status.must_change_password

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault()
    setPending(true)
    setError(null)
    try {
      const res = await loginAdmin(username, password, remember)
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
            <label className='flex cursor-pointer items-center gap-2 text-sm text-muted-foreground'>
              <Checkbox
                checked={remember}
                onCheckedChange={(v) => setRemember(v === true)}
              />
              记住我，30 天内免登录
            </label>
            {error ? <p className='text-sm text-destructive'>{error}</p> : null}
            <Button type='submit' className='w-full' disabled={pending}>
              {pending ? '登录中…' : '登录'}
            </Button>
          </form>
        </CardContent>
      </Card>
      {showFirstRunHint ? (
        <Alert className='w-full max-w-sm border-amber-500/40 bg-amber-500/10 text-amber-900 dark:text-amber-200 [&>svg]:text-amber-600 dark:[&>svg]:text-amber-400'>
          <KeyRound />
          <AlertDescription>
            首次启动系统会生成默认密码，在启动日志中搜索 "Admin password" 即可。
          </AlertDescription>
        </Alert>
      ) : null}
    </AuthShell>
  )
}
