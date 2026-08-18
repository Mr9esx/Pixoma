import { useEffect, useMemo, useState } from 'react'
import { useNavigate } from '@tanstack/react-router'
import { ArrowLeft } from 'lucide-react'
import {
  changeAdminPassword,
  finalizeSetup,
  saveSetupDraft,
  testDatabase,
  waitForSetupReady,
  type SetupDraft,
  type SetupStatus,
} from '@/lib/api/setup'
import { cn } from '@/lib/utils'
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
import { RadioGroup, RadioGroupItem } from '@/components/ui/radio-group'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { AuthShell } from './auth-shell'
import {
  initialSetupStep,
  previousSetupStep,
  SETUP_STEP_COPY,
  setupStepIndex,
  setupStepsFor,
  type SetupStep,
} from './setup-steps'

export function SetupWizard({ status }: { status: SetupStatus }) {
  const navigate = useNavigate()
  const steps = useMemo(
    () => setupStepsFor(status.must_change_password),
    [status.must_change_password]
  )
  const [step, setStep] = useState<SetupStep>(() => initialSetupStep(status))
  const [error, setError] = useState<string | null>(null)
  const [pending, setPending] = useState(false)
  const [reloading, setReloading] = useState(Boolean(status.restart_required))

  const [newPassword, setNewPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [driver, setDriver] = useState('sqlite')
  const [dsn, setDsn] = useState('data/app.db')
  const [placement, setPlacement] = useState<'local' | 'remote'>('local')
  const [blobDriver, setBlobDriver] = useState('localfs')
  const [blobRoot, setBlobRoot] = useState('data/blob')
  const [blobEndpoint, setBlobEndpoint] = useState('')
  const [blobRegion, setBlobRegion] = useState('')
  const [blobBucket, setBlobBucket] = useState('')
  const [blobAccessKey, setBlobAccessKey] = useState('')
  const [blobSecretKey, setBlobSecretKey] = useState('')
  const backStep = previousSetupStep(steps, step)
  const copy = SETUP_STEP_COPY[step]

  function draft(overrides?: Partial<SetupDraft>): SetupDraft {
    return {
      placement,
      db_driver: driver,
      db_dsn: dsn,
      blob_driver: blobDriver,
      blob_root: blobRoot,
      blob_endpoint: blobEndpoint,
      blob_region: blobRegion,
      blob_bucket: blobBucket,
      blob_access_key: blobAccessKey,
      blob_secret_key: blobSecretKey,
      comfy_mock: false,
      comfyui_base_url: '',
      default_edge_id: 'local',
      auto_spawn_edge: placement === 'local',
      ...overrides,
    }
  }

  async function run(fn: () => Promise<void>) {
    setPending(true)
    setError(null)
    try {
      await fn()
    } catch (err) {
      setError(err instanceof Error ? err.message : '操作失败')
    } finally {
      setPending(false)
    }
  }

  function goBack() {
    if (!backStep) return
    setError(null)
    setStep(backStep)
  }

  useEffect(() => {
    if (!reloading) return
    let cancelled = false
    void (async () => {
      try {
        await waitForSetupReady()
        if (!cancelled) await navigate({ to: '/' })
      } catch (err) {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : '等待重启失败')
        }
      }
    })()
    return () => {
      cancelled = true
    }
  }, [reloading, navigate])

  if (reloading) {
    return (
      <AuthShell>
        <Card className='w-full max-w-md'>
          <CardHeader>
            <CardTitle>正在重启</CardTitle>
            <CardDescription>
              配置已经写好。等 pixoma 重新起来就会进后台，不用自己杀进程。
            </CardDescription>
          </CardHeader>
          <CardContent className='flex flex-col gap-3 text-sm'>
            {error ? (
              <p className='text-destructive'>{error}</p>
            ) : (
              <p className='text-muted-foreground'>正在等待服务回来…</p>
            )}
          </CardContent>
        </Card>
      </AuthShell>
    )
  }

  return (
    <WizardCard steps={steps} step={step} title={copy.title} desc={copy.desc}>
      {step === 'password' ? (
        <form
          className='flex flex-col gap-4'
          onSubmit={(e) => {
            e.preventDefault()
            void run(async () => {
              if (newPassword !== confirmPassword) {
                throw new Error('两次输入不一致')
              }
              await changeAdminPassword({ newPassword })
              setStep('database')
            })
          }}
        >
          <Field label='新密码（至少 8 位）' htmlFor='new-password'>
            <Input
              id='new-password'
              type='password'
              value={newPassword}
              onChange={(e) => setNewPassword(e.target.value)}
              autoComplete='new-password'
            />
          </Field>
          <Field label='再输一遍' htmlFor='confirm-password'>
            <Input
              id='confirm-password'
              type='password'
              value={confirmPassword}
              onChange={(e) => setConfirmPassword(e.target.value)}
              autoComplete='new-password'
            />
          </Field>
          <StepActions error={error} pending={pending} submit={copy.submit} />
        </form>
      ) : null}

      {step === 'database' ? (
        <form
          className='flex flex-col gap-4'
          onSubmit={(e) => {
            e.preventDefault()
            void run(async () => {
              await testDatabase(driver, dsn)
              setStep('placement')
            })
          }}
        >
          <Field label='业务库' htmlFor='db-driver'>
            <Select value={driver} onValueChange={setDriver}>
              <SelectTrigger id='db-driver' className='w-full'>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
                  <SelectItem value='sqlite'>
                    SQLite（本机文件，适合先跑通）
                  </SelectItem>
                  <SelectItem value='mysql'>MySQL</SelectItem>
                  <SelectItem value='postgres'>Postgres</SelectItem>
                </SelectGroup>
              </SelectContent>
            </Select>
          </Field>
          <Field label='DSN / 文件路径' htmlFor='db-dsn'>
            <Input
              id='db-dsn'
              value={dsn}
              onChange={(e) => setDsn(e.target.value)}
            />
          </Field>
          <StepActions
            error={error}
            pending={pending}
            submit={copy.submit}
            onBack={backStep ? goBack : undefined}
          />
        </form>
      ) : null}

      {step === 'placement' ? (
        <form
          className='flex flex-col gap-4'
          onSubmit={(e) => {
            e.preventDefault()
            if (placement === 'local') setBlobDriver('localfs')
            else if (blobDriver === 'localfs') setBlobDriver('s3')
            setStep('storage')
          }}
        >
          <RadioGroup
            value={placement}
            onValueChange={(v) => setPlacement(v as 'local' | 'remote')}
          >
            <label className='flex items-start gap-3 text-sm'>
              <RadioGroupItem value='local' />
              <span>
                <strong>本机</strong>
                <span className='mt-1 block text-muted-foreground'>
                  Comfy 和后台在同一台机器。文件用本地目录，pixoma
                  会自动拉起本机 Edge。
                </span>
              </span>
            </label>
            <label className='flex items-start gap-3 text-sm'>
              <RadioGroupItem value='remote' />
              <span>
                <strong>远程</strong>
                <span className='mt-1 block text-muted-foreground'>
                  GPU 在别的机器。必须用对象存储（S3 / TOS），不能用本机目录。
                </span>
              </span>
            </label>
          </RadioGroup>
          <StepActions
            error={error}
            pending={pending}
            submit={copy.submit}
            onBack={backStep ? goBack : undefined}
          />
        </form>
      ) : null}

      {step === 'storage' ? (
        <form
          className='flex flex-col gap-4'
          onSubmit={(e) => {
            e.preventDefault()
            void run(async () => {
              await saveSetupDraft(draft())
              await finalizeSetup()
              setReloading(true)
            })
          }}
        >
          <Field label='对象存储' htmlFor='blob-driver'>
            <Select value={blobDriver} onValueChange={setBlobDriver}>
              <SelectTrigger id='blob-driver' className='w-full'>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
                  {placement === 'local' ? (
                    <SelectItem value='localfs'>本机目录</SelectItem>
                  ) : null}
                  <SelectItem value='s3'>S3 兼容</SelectItem>
                  <SelectItem value='tos'>火山 TOS</SelectItem>
                </SelectGroup>
              </SelectContent>
            </Select>
          </Field>
          {blobDriver === 'localfs' ? (
            <Field label='目录' htmlFor='blob-root'>
              <Input
                id='blob-root'
                value={blobRoot}
                onChange={(e) => setBlobRoot(e.target.value)}
              />
            </Field>
          ) : (
            <>
              <Field label='Endpoint' htmlFor='blob-endpoint'>
                <Input
                  id='blob-endpoint'
                  value={blobEndpoint}
                  onChange={(e) => setBlobEndpoint(e.target.value)}
                />
              </Field>
              <Field label='Region' htmlFor='blob-region'>
                <Input
                  id='blob-region'
                  value={blobRegion}
                  onChange={(e) => setBlobRegion(e.target.value)}
                />
              </Field>
              <Field label='Bucket' htmlFor='blob-bucket'>
                <Input
                  id='blob-bucket'
                  value={blobBucket}
                  onChange={(e) => setBlobBucket(e.target.value)}
                />
              </Field>
              <Field label='Access Key' htmlFor='blob-access'>
                <Input
                  id='blob-access'
                  value={blobAccessKey}
                  onChange={(e) => setBlobAccessKey(e.target.value)}
                />
              </Field>
              <Field label='Secret Key' htmlFor='blob-secret'>
                <Input
                  id='blob-secret'
                  type='password'
                  value={blobSecretKey}
                  onChange={(e) => setBlobSecretKey(e.target.value)}
                />
              </Field>
            </>
          )}
          <StepActions
            error={error}
            pending={pending}
            submit={copy.submit}
            onBack={backStep ? goBack : undefined}
          />
        </form>
      ) : null}

    </WizardCard>
  )
}

function WizardCard({
  steps,
  step,
  title,
  desc,
  children,
}: {
  steps: SetupStep[]
  step: SetupStep
  title: string
  desc: string
  children: React.ReactNode
}) {
  const index = setupStepIndex(steps, step)
  return (
    <AuthShell>
      <Card className='w-full max-w-md'>
        <CardHeader>
          <div className='flex gap-1' aria-hidden>
            {steps.map((item, i) => (
              <div
                key={item}
                className={cn(
                  'h-1 flex-1 rounded-full',
                  i <= index ? 'bg-primary' : 'bg-muted'
                )}
              />
            ))}
          </div>
          <p className='text-sm text-muted-foreground'>
            第 {index + 1} / {steps.length} 步
          </p>
          <CardTitle>{title}</CardTitle>
          {desc ? <CardDescription>{desc}</CardDescription> : null}
        </CardHeader>
        <CardContent className='flex flex-col gap-4'>{children}</CardContent>
      </Card>
    </AuthShell>
  )
}

function Field({
  label,
  htmlFor,
  children,
}: {
  label: string
  htmlFor: string
  children: React.ReactNode
}) {
  return (
    <div className='flex flex-col gap-2'>
      <Label htmlFor={htmlFor}>{label}</Label>
      {children}
    </div>
  )
}

function StepActions({
  error,
  pending,
  submit,
  onBack,
  onSkip,
}: {
  error: string | null
  pending: boolean
  submit: string
  onBack?: () => void
  onSkip?: () => void
}) {
  return (
    <div className='flex flex-col gap-2'>
      {error ? <p className='text-sm text-destructive'>{error}</p> : null}
      <div className='flex gap-2'>
        {onBack ? (
          <Button
            type='button'
            variant='outline'
            disabled={pending}
            onClick={onBack}
          >
            <ArrowLeft />
            上一步
          </Button>
        ) : null}
        {onSkip ? (
          <Button
            type='button'
            variant='ghost'
            disabled={pending}
            onClick={onSkip}
          >
            暂时跳过
          </Button>
        ) : null}
        <Button type='submit' className='flex-1' disabled={pending}>
          {pending ? '处理中…' : submit}
        </Button>
      </div>
    </div>
  )
}
