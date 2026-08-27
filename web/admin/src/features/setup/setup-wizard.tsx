import { useEffect, useMemo, useRef, useState } from 'react'
import { useNavigate } from '@tanstack/react-router'
import { ArrowLeft, CircleAlert, CircleCheck, Info } from 'lucide-react'
import {
  changeAdminPassword,
  finalizeSetup,
  saveAdminProfile,
  saveSetupDraft,
  testDatabase,
  testBlob,
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
import {
  Alert,
  AlertDescription,
  AlertTitle,
} from '@/components/ui/alert'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
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
  buildMySQLDSN,
  buildPostgresDSN,
  buildSqliteDSN,
} from './db-dsn'
import { setupErrorCopy, type AlertCopy } from './db-error'
import { blobErrorCopy } from './blob-error'
import {
  initialSetupStep,
  previousSetupStep,
  SETUP_STEP_COPY,
  setupStepIndex,
  setupStepsFor,
  type SetupStep,
} from './setup-steps'

const TOS_DEFAULTS = {
  endpoint: 'https://tos-cn-beijing.volces.com',
  region: 'cn-beijing',
  bucket: 'pixoma',
}

export function SetupWizard({ status }: { status: SetupStatus }) {
  const navigate = useNavigate()
  const steps = useMemo(
    () => setupStepsFor(status.must_change_password),
    [status.must_change_password]
  )
  const [step, setStep] = useState<SetupStep>(() => initialSetupStep(status))
  const [error, setError] = useState<AlertCopy | null>(null)
  const [pending, setPending] = useState(false)
  const [reloading, setReloading] = useState(Boolean(status.restart_required))

  const [newPassword, setNewPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [nickname, setNickname] = useState('')
  const [email, setEmail] = useState('')
  const [avatarUrl, setAvatarUrl] = useState('')
  const [driver, setDriver] = useState('sqlite')
  const [sqlitePath, setSqlitePath] = useState('data/app.db')
  const [dbHost, setDbHost] = useState('127.0.0.1')
  const [dbPort, setDbPort] = useState('3306')
  const [dbUser, setDbUser] = useState('pixoma')
  const [dbPassword, setDbPassword] = useState('')
  const [dbName, setDbName] = useState('pixoma')
  const [pgSslMode, setPgSslMode] = useState('disable')
  const [dbExtraParams, setDbExtraParams] = useState('')
  const [dbTested, setDbTested] = useState(false)
  const [blobTested, setBlobTested] = useState(false)
  const [blobPromptCreate, setBlobPromptCreate] = useState(false)
  const [blobMissingBucket, setBlobMissingBucket] = useState('')
  const [blobDriver, setBlobDriver] = useState('localfs')
  const [blobRoot, setBlobRoot] = useState('data/blob')
  const [blobEndpoint, setBlobEndpoint] = useState('')
  const [blobRegion, setBlobRegion] = useState('')
  const [blobBucket, setBlobBucket] = useState('')
  const [blobAccessKey, setBlobAccessKey] = useState('')
  const [blobSecretKey, setBlobSecretKey] = useState('')
  const backStep = previousSetupStep(steps, step)
  const copy = SETUP_STEP_COPY[step]

  const dsn = useMemo(() => {
    if (driver === 'mysql') {
      return buildMySQLDSN({
        host: dbHost,
        port: dbPort,
        user: dbUser,
        password: dbPassword,
        database: dbName,
        params: dbExtraParams,
      })
    }
    if (driver === 'postgres') {
      return buildPostgresDSN({
        host: dbHost,
        port: dbPort,
        user: dbUser,
        password: dbPassword,
        database: dbName,
        sslmode: pgSslMode,
        params: dbExtraParams,
      })
    }
    return buildSqliteDSN(sqlitePath)
  }, [
    driver,
    dbHost,
    dbPort,
    dbUser,
    dbPassword,
    dbName,
    pgSslMode,
    dbExtraParams,
    sqlitePath,
  ])

  const lastTestedDsn = useRef(dsn)
  useEffect(() => {
    if (lastTestedDsn.current !== dsn) {
      lastTestedDsn.current = dsn
      setDbTested(false)
    }
  }, [dsn])

  const blobKey = useMemo(
    () =>
      JSON.stringify({
        driver: blobDriver,
        root: blobRoot,
        endpoint: blobEndpoint,
        region: blobRegion,
        bucket: blobBucket,
        ak: blobAccessKey,
        sk: blobSecretKey,
      }),
    [
      blobDriver,
      blobRoot,
      blobEndpoint,
      blobRegion,
      blobBucket,
      blobAccessKey,
      blobSecretKey,
    ]
  )
  const lastTestedBlob = useRef(blobKey)
  useEffect(() => {
    if (lastTestedBlob.current !== blobKey) {
      lastTestedBlob.current = blobKey
      setBlobTested(false)
      setBlobPromptCreate(false)
    }
  }, [blobKey])

  function blobConfig() {
    return {
      blob_driver: blobDriver,
      blob_root: blobRoot,
      blob_endpoint: blobEndpoint,
      blob_region: blobRegion,
      blob_bucket: blobBucket,
      blob_access_key: blobAccessKey,
      blob_secret_key: blobSecretKey,
    }
  }

  async function runBlobTest(autoCreateBucket: boolean) {
    setPending(true)
    setError(null)
    try {
      const res = await testBlob({ ...blobConfig(), auto_create_bucket: autoCreateBucket })
      if (!res.ok && res.code === 'bucket_not_found' && !autoCreateBucket) {
        setBlobMissingBucket(res.bucket ?? blobBucket)
        setBlobPromptCreate(true)
        setBlobTested(false)
        return
      }
      setBlobPromptCreate(false)
      lastTestedBlob.current = blobKey
      setBlobTested(true)
    } catch (err) {
      setError(blobErrorCopy(err))
    } finally {
      setPending(false)
    }
  }

  function onDriverChange(next: string) {
    if (next === 'mysql' && dbPort === '5432') setDbPort('3306')
    if (next === 'postgres' && dbPort === '3306') setDbPort('5432')
    setDriver(next)
  }

  function onBlobDriverChange(next: string) {
    setBlobDriver(next)
    if (next === 'tos') {
      if (!blobEndpoint) setBlobEndpoint(TOS_DEFAULTS.endpoint)
      if (!blobRegion) setBlobRegion(TOS_DEFAULTS.region)
      if (!blobBucket) setBlobBucket(TOS_DEFAULTS.bucket)
    }
  }

  function draft(overrides?: Partial<SetupDraft>): SetupDraft {
    return {
      placement: blobDriver === 'localfs' ? 'local' : 'remote',
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
      ...overrides,
    }
  }

  async function run(fn: () => Promise<void>) {
    setPending(true)
    setError(null)
    try {
      await fn()
    } catch (err) {
      setError(setupErrorCopy(err))
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
          setError(setupErrorCopy(err))
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
              <Alert variant='destructive'>
                <AlertTitle>{error.title}</AlertTitle>
                <AlertDescription>{error.detail}</AlertDescription>
              </Alert>
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
              setStep('profile')
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

      {step === 'profile' ? (
        <form
          className='flex flex-col gap-4'
          onSubmit={(e) => {
            e.preventDefault()
            void run(async () => {
              await saveAdminProfile({
                nickname: nickname.trim(),
                email: email.trim(),
                avatarUrl: avatarUrl.trim(),
              })
              setStep('database')
            })
          }}
        >
          <Field label='昵称（怎么称呼你）' htmlFor='profile-nickname'>
            <Input
              id='profile-nickname'
              value={nickname}
              onChange={(e) => setNickname(e.target.value)}
              autoComplete='off'
              placeholder='例如：小 P'
            />
          </Field>
          <Field label='邮箱' htmlFor='profile-email'>
            <Input
              id='profile-email'
              type='email'
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              autoComplete='off'
              placeholder='admin@example.com'
            />
          </Field>
          <Field label='头像地址（可选）' htmlFor='profile-avatar'>
            <Input
              id='profile-avatar'
              value={avatarUrl}
              onChange={(e) => setAvatarUrl(e.target.value)}
              autoComplete='off'
              placeholder='https://…'
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

      {step === 'database' ? (
        <form
          className='flex flex-col gap-4'
          onSubmit={(e) => {
            e.preventDefault()
            void run(async () => {
              await testDatabase(driver, dsn)
              lastTestedDsn.current = dsn
              setDbTested(true)
              setStep('storage')
            })
          }}
        >
          <Field label='业务库' htmlFor='db-driver'>
            <Select value={driver} onValueChange={onDriverChange}>
              <SelectTrigger id='db-driver' className='w-full'>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
                  <SelectItem value='sqlite'>SQLite</SelectItem>
                  <SelectItem value='mysql'>MySQL</SelectItem>
                  <SelectItem value='postgres'>Postgres</SelectItem>
                </SelectGroup>
              </SelectContent>
            </Select>
          </Field>
          {driver === 'sqlite' ? (
            <Field label='数据库文件' htmlFor='db-sqlite-path'>
              <Input
                id='db-sqlite-path'
                value={sqlitePath}
                onChange={(e) => setSqlitePath(e.target.value)}
                placeholder='data/app.db'
              />
            </Field>
          ) : driver === 'mysql' ? (
            <>
              <div className='grid grid-cols-2 gap-3'>
                <Field label='Host' htmlFor='db-host'>
                  <Input
                    id='db-host'
                    value={dbHost}
                    onChange={(e) => setDbHost(e.target.value)}
                    placeholder='127.0.0.1'
                  />
                </Field>
                <Field label='端口' htmlFor='db-port'>
                  <Input
                    id='db-port'
                    value={dbPort}
                    onChange={(e) => setDbPort(e.target.value)}
                    placeholder='3306'
                  />
                </Field>
              </div>
              <Field label='用户' htmlFor='db-user'>
                <Input
                  id='db-user'
                  value={dbUser}
                  onChange={(e) => setDbUser(e.target.value)}
                  placeholder='pixoma'
                />
              </Field>
              <Field label='密码' htmlFor='db-password'>
                <Input
                  id='db-password'
                  type='password'
                  value={dbPassword}
                  onChange={(e) => setDbPassword(e.target.value)}
                  autoComplete='off'
                  placeholder='留空表示无密码'
                />
              </Field>
              <Field label='数据库' htmlFor='db-name'>
                <Input
                  id='db-name'
                  value={dbName}
                  onChange={(e) => setDbName(e.target.value)}
                  placeholder='pixoma'
                />
              </Field>
              <Field label='附加参数' htmlFor='db-extra-params'>
                <Input
                  id='db-extra-params'
                  value={dbExtraParams}
                  onChange={(e) => setDbExtraParams(e.target.value)}
                  placeholder='timeout=5s&readTimeout=10s'
                />
              </Field>
            </>
          ) : (
            <>
              <div className='grid grid-cols-2 gap-3'>
                <Field label='Host' htmlFor='db-host'>
                  <Input
                    id='db-host'
                    value={dbHost}
                    onChange={(e) => setDbHost(e.target.value)}
                    placeholder='127.0.0.1'
                  />
                </Field>
                <Field label='端口' htmlFor='db-port'>
                  <Input
                    id='db-port'
                    value={dbPort}
                    onChange={(e) => setDbPort(e.target.value)}
                    placeholder='5432'
                  />
                </Field>
              </div>
              <Field label='用户' htmlFor='db-user'>
                <Input
                  id='db-user'
                  value={dbUser}
                  onChange={(e) => setDbUser(e.target.value)}
                  placeholder='pixoma'
                />
              </Field>
              <Field label='密码' htmlFor='db-password'>
                <Input
                  id='db-password'
                  type='password'
                  value={dbPassword}
                  onChange={(e) => setDbPassword(e.target.value)}
                  autoComplete='off'
                  placeholder='留空表示无密码'
                />
              </Field>
              <Field label='数据库' htmlFor='db-name'>
                <Input
                  id='db-name'
                  value={dbName}
                  onChange={(e) => setDbName(e.target.value)}
                  placeholder='pixoma'
                />
              </Field>
              <Field label='SSL 模式' htmlFor='db-ssl-mode'>
                <Select value={pgSslMode} onValueChange={setPgSslMode}>
                  <SelectTrigger id='db-ssl-mode' className='w-full'>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectGroup>
                      <SelectItem value='disable'>disable</SelectItem>
                      <SelectItem value='require'>require</SelectItem>
                      <SelectItem value='prefer'>prefer</SelectItem>
                    </SelectGroup>
                  </SelectContent>
                </Select>
              </Field>
              <Field label='附加参数' htmlFor='db-extra-params'>
                <Input
                  id='db-extra-params'
                  value={dbExtraParams}
                  onChange={(e) => setDbExtraParams(e.target.value)}
                  placeholder='connect_timeout=10 application_name=pixoma'
                />
              </Field>
            </>
          )}
          <StepActions
            error={error}
            pending={pending}
            submit={copy.submit}
            onBack={backStep ? goBack : undefined}
            onTest={() => {
              void run(async () => {
                await testDatabase(driver, dsn)
                lastTestedDsn.current = dsn
                setDbTested(true)
              })
            }}
            testPassed={dbTested}
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
            <Select value={blobDriver} onValueChange={onBlobDriverChange}>
              <SelectTrigger id='blob-driver' className='w-full'>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
                  <SelectItem value='localfs'>本机目录</SelectItem>
                  <SelectItem value='sharedfs'>共享目录（SMB / NFS）</SelectItem>
                  <SelectItem value='s3'>S3</SelectItem>
                  <SelectItem value='tos'>火山 TOS</SelectItem>
                </SelectGroup>
              </SelectContent>
            </Select>
          </Field>
          {blobDriver === 'localfs' || blobDriver === 'sharedfs' ? (
            <Field
              label={blobDriver === 'sharedfs' ? '挂载目录' : '目录'}
              htmlFor='blob-root'
            >
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
              {blobDriver === 's3' ? (
                <p className='text-sm text-muted-foreground'>
                  局域网可用 MinIO 等 S3 兼容服务，如{' '}
                  <code>http://&lt;server-ip&gt;:9000</code>
                </p>
              ) : null}
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
          {blobPromptCreate ? (
            <Alert variant='info'>
              <AlertTitle>
                bucket `{blobMissingBucket}` 不存在，帮你创建？
              </AlertTitle>
              <div className='flex gap-2 pt-2'>
                <Button
                  type='button'
                  disabled={pending}
                  onClick={() => {
                    setBlobPromptCreate(false)
                    void runBlobTest(true)
                  }}
                >
                  创建
                </Button>
                <Button
                  type='button'
                  variant='outline'
                  disabled={pending}
                  onClick={() => setBlobPromptCreate(false)}
                >
                  取消
                </Button>
              </div>
            </Alert>
          ) : null}
          {blobDriver === 'localfs' ? (
            <Alert variant='warn'>
              <CircleAlert aria-hidden='true' />
              <AlertTitle>注意！</AlertTitle>
              <AlertDescription>
                这个配置只适合 ComfyUI 和后台在同一台机器上使用，无法使用远程节点。
              </AlertDescription>
            </Alert>
          ) : null}
          {blobDriver === 'sharedfs' ? (
            <Alert variant='info'>
              <Info aria-hidden='true' />
              <AlertTitle>注意！</AlertTitle>
              <AlertDescription>
                需要先在所有机器上挂载同一共享目录（SMB / NFS）。
                <pre className='mt-2 w-full overflow-x-auto rounded-md border bg-card p-2 text-xs leading-relaxed'>
                  {`# Linux NFS（<server-ip> 换成共享服务器地址，<share-path> 换成导出目录）
mount -t nfs <server-ip>:/<share-path> /mnt/pixoma-shared

# Linux / macOS CIFS (SMB)（<server-ip> / <share-name> / <mount-point> 按实际替换）
mount -t cifs //<server-ip>/<share-name> <mount-point> -o username=<user>`}
                </pre>
              </AlertDescription>
            </Alert>
          ) : null}
          <StepActions
            error={error}
            pending={pending}
            submit={copy.submit}
            onBack={backStep ? goBack : undefined}
            onTest={() => void runBlobTest(false)}
            testPassed={blobTested}
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
  onTest,
  testPassed,
}: {
  error: AlertCopy | null
  pending: boolean
  submit: string
  onBack?: () => void
  onSkip?: () => void
  onTest?: () => void
  testPassed?: boolean
}) {
  return (
    <div className='flex flex-col gap-2'>
      {error ? (
        <Alert variant='destructive'>
          <AlertTitle>{error.title}</AlertTitle>
          <AlertDescription>{error.detail}</AlertDescription>
        </Alert>
      ) : null}
      {testPassed ? (
        <Alert variant='success'>
          <CircleCheck aria-hidden='true' />
          <AlertTitle>连接正常</AlertTitle>
        </Alert>
      ) : null}
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
        {onTest ? (
          <Button
            type='button'
            variant='outline'
            disabled={pending}
            onClick={onTest}
          >
            连通性测试
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
