import { useState } from 'react'
import { Link, useNavigate } from '@tanstack/react-router'
import { KeyRound } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { ApiError } from '@/lib/api/client'
import { loginAdmin, type SetupStatus } from '@/lib/api/setup'
import { useTheme } from '@/context/theme-provider'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Checkbox } from '@/components/ui/checkbox'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { ThemeSwitcher } from '@/components/kibo-ui/theme-switcher'
import { AuthShell, type AuthAmbient } from './auth-shell'
import { LocaleSwitcher } from './locale-switcher'

export function LoginPage({
  status,
  registrationOpen = false,
}: {
  status: SetupStatus
  registrationOpen?: boolean
}) {
  const navigate = useNavigate()
  const expired =
    typeof window !== 'undefined' &&
    new URLSearchParams(window.location.search).get('expired') === '1'
  const [username, setUsername] = useState('admin')
  const [password, setPassword] = useState('')
  const [remember, setRemember] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [pending, setPending] = useState(false)
  const [ambient] = useState<AuthAmbient>(() =>
    Math.random() < 0.5 ? 'dither' : 'terminal'
  )
  const { theme, setTheme } = useTheme()
  const { t } = useTranslation()
  const showFirstRunHint = !status.initialized && status.must_change_password
  const showLiveDemoHint = status.live_demo === true

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
      setError(err instanceof ApiError ? err.message : t('auth.loginFailed'))
    } finally {
      setPending(false)
    }
  }

  return (
    <AuthShell ambient={ambient} logoVariant='brand'>
      {expired ? (
        <Alert
          variant='destructive'
          className='w-full max-w-sm'
          data-testid='session-expired'
        >
          <AlertTitle>{t('auth.sessionExpiredTitle')}</AlertTitle>
          <AlertDescription>
            {t('auth.sessionExpiredDescription')}
          </AlertDescription>
        </Alert>
      ) : null}
      {showLiveDemoHint ? (
        <Alert className='w-full max-w-sm' data-testid='live-demo-credentials'>
          <KeyRound />
          <AlertTitle>Live Demo</AlertTitle>
          <AlertDescription>
            Live Demo 为只读模式。演示账号：admin / 123456
          </AlertDescription>
        </Alert>
      ) : null}
      <Card className='w-full max-w-sm'>
        <CardHeader>
          <CardTitle>{t('auth.title')}</CardTitle>
        </CardHeader>
        <CardContent>
          <form className='flex flex-col gap-4' onSubmit={onSubmit}>
            <div className='flex flex-col gap-2'>
              <Label htmlFor='username'>{t('auth.username')}</Label>
              <Input
                id='username'
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                autoComplete='username'
              />
            </div>
            <div className='flex flex-col gap-2'>
              <Label htmlFor='password'>{t('auth.password')}</Label>
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
              {t('auth.remember')}
            </label>
            {error ? <p className='text-sm text-destructive'>{error}</p> : null}
            <Button type='submit' className='w-full' disabled={pending}>
              {pending ? t('auth.submitting') : t('auth.submit')}
            </Button>
            {registrationOpen ? (
              <p className='text-center text-sm text-muted-foreground'>
                {t('auth.registrationPrompt')}{' '}
                <Link to='/register' className='text-primary hover:underline'>
                  {t('auth.createAccount')}
                </Link>
              </p>
            ) : null}
          </form>
        </CardContent>
      </Card>
      {showFirstRunHint ? (
        <Alert
          variant='warn'
          className='w-full max-w-sm'
          style={{ marginTop: '16px' }}
          data-testid='first-run-hint'
        >
          <KeyRound />
          <AlertDescription>{t('auth.firstRunHint')}</AlertDescription>
        </Alert>
      ) : null}
      <div
        className='fixed right-4 bottom-4 z-30 flex items-center gap-2'
        data-testid='login-theme-switcher'
      >
        <LocaleSwitcher />
        <ThemeSwitcher value={theme} onChange={setTheme} />
      </div>
    </AuthShell>
  )
}
