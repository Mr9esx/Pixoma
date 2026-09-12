import { useState, type FormEvent } from 'react'
import { useNavigate, Link } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
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
  const { t } = useTranslation()
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
      setError(t('auth.usernameRequired'))
      return
    }
    if (password.length < 8) {
      setError(t('settings.passwordTooShort'))
      return
    }
    if (password !== confirm) {
      setError(t('settings.passwordMismatch'))
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
      toast.success(t('auth.registerSuccess'))
      await navigate({ to: '/' })
    } catch (err) {
      setError(errorMessage(err) ?? t('auth.registerFailed'))
    } finally {
      setPending(false)
    }
  }

  return (
    <AuthShell>
      <Card className='w-full max-w-md'>
        <CardHeader>
          <CardTitle>{t('auth.registerTitle')}</CardTitle>
          <CardDescription>{t('auth.registerDesc')}</CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={onSubmit} data-testid='register-form'>
            <FieldGroup className='gap-3'>
              <Field>
                <FieldLabel htmlFor='register-username'>
                  {t('auth.registerUsername')}
                </FieldLabel>
                <Input
                  id='register-username'
                  value={username}
                  onChange={(e) => setUsername(e.target.value)}
                  autoComplete='off'
                  autoFocus
                />
              </Field>
              <Field>
                <FieldLabel htmlFor='register-email'>
                  {t('setup.email')}
                </FieldLabel>
                <Input
                  id='register-email'
                  type='email'
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  autoComplete='off'
                />
              </Field>
              <Field>
                <FieldLabel htmlFor='register-nickname'>
                  {t('setup.nickname')}
                </FieldLabel>
                <Input
                  id='register-nickname'
                  value={nickname}
                  onChange={(e) => setNickname(e.target.value)}
                  autoComplete='off'
                />
              </Field>
              <Field>
                <FieldLabel htmlFor='register-password'>
                  {t('auth.registerPassword')}
                </FieldLabel>
                <SecretInput
                  id='register-password'
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  autoComplete='new-password'
                />
              </Field>
              <Field>
                <FieldLabel htmlFor='register-confirm'>
                  {t('auth.registerConfirm')}
                </FieldLabel>
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
              {pending ? t('auth.registerSubmitting') : t('auth.registerSubmit')}
            </Button>
            <p className='text-center text-sm text-muted-foreground'>
              {t('auth.hasAccount')}{' '}
              <Link to='/login' className='text-primary hover:underline'>
                {t('auth.goLogin')}
              </Link>
            </p>
          </form>
        </CardContent>
      </Card>
    </AuthShell>
  )
}
