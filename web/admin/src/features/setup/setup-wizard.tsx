import { useMemo, useState } from 'react'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { RadioGroup, RadioGroupItem } from '@/components/ui/radio-group'
import { ApiError } from '@/lib/api/client'
import {
  changeAdminPassword,
  finalizeSetup,
  saveSetupDraft,
  testDatabase,
  type SetupStatus,
} from '@/lib/api/setup'

const steps = ['password', 'database', 'placement', 'storage', 'edge', 'channel'] as const
type Step = (typeof steps)[number]

export function SetupWizard({ status }: { status: SetupStatus }) {
  const initial = useMemo<Step>(() => {
    if (status.must_change_password) return 'password'
    const s = status.wizard_step as Step | undefined
    if (s && steps.includes(s)) return s
    return 'database'
  }, [status])
  const [step, setStep] = useState<Step>(initial)
  const [error, setError] = useState<string | null>(null)
  const [pending, setPending] = useState(false)
  const [restartMessage, setRestartMessage] = useState<string | null>(null)

  const [oldPassword, setOldPassword] = useState('')
  const [newPassword, setNewPassword] = useState('')
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
  const [comfyMock, setComfyMock] = useState(true)
  const [comfyURL, setComfyURL] = useState('http://127.0.0.1:8188')
  const [instanceId, setInstanceId] = useState('local')
  const [tgToken, setTgToken] = useState('')

  async function run(fn: () => Promise<void>) {
    setPending(true)
    setError(null)
    try {
      await fn()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : '操作失败')
    } finally {
      setPending(false)
    }
  }

  async function saveDraftAnd(next: Step) {
    await saveSetupDraft({
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
      comfy_mock: comfyMock,
      comfyui_base_url: comfyURL,
      default_instance_id: instanceId,
      auto_spawn_edge: placement === 'local',
      telegram_bot_token: tgToken,
    })
    setStep(next)
  }

  if (restartMessage) {
    return (
      <Shell title='保存成功' desc='配置已写入。重启 pixoma 后才会按新设置装配。'>
        <p className='text-sm'>{restartMessage}</p>
        <p className='text-muted-foreground text-sm'>
          停掉当前进程，再执行同一条启动命令即可。改过的管理员密码不会再出现在启动日志里。
        </p>
      </Shell>
    )
  }

  return (
    <Shell title='初始化向导' desc='先改密码，再选库、部署位置和存储。完成后重启生效。'>
      {step === 'password' ? (
        <form
          className='space-y-4'
          onSubmit={(e) => {
            e.preventDefault()
            void run(async () => {
              await changeAdminPassword(oldPassword, newPassword)
              setStep('database')
            })
          }}
        >
          <Field label='当前密码'>
            <Input type='password' value={oldPassword} onChange={(e) => setOldPassword(e.target.value)} />
          </Field>
          <Field label='新密码（至少 8 位）'>
            <Input type='password' value={newPassword} onChange={(e) => setNewPassword(e.target.value)} />
          </Field>
          <Actions error={error} pending={pending} submit='保存密码' />
        </form>
      ) : null}

      {step === 'database' ? (
        <form
          className='space-y-4'
          onSubmit={(e) => {
            e.preventDefault()
            void run(async () => {
              await testDatabase(driver, dsn)
              setStep('placement')
            })
          }}
        >
          <Field label='业务库'>
            <select
              className='border-input h-9 w-full rounded-md border bg-transparent px-3 text-sm'
              value={driver}
              onChange={(e) => setDriver(e.target.value)}
            >
              <option value='sqlite'>SQLite（本机文件，适合先跑通）</option>
              <option value='mysql'>MySQL</option>
              <option value='postgres'>Postgres</option>
            </select>
          </Field>
          <Field label='DSN / 文件路径'>
            <Input value={dsn} onChange={(e) => setDsn(e.target.value)} />
          </Field>
          <Actions error={error} pending={pending} submit='测连通并保存' />
        </form>
      ) : null}

      {step === 'placement' ? (
        <div className='space-y-4'>
          <RadioGroup value={placement} onValueChange={(v) => setPlacement(v as 'local' | 'remote')}>
            <label className='flex items-start gap-3 text-sm'>
              <RadioGroupItem value='local' />
              <span>
                <strong>本机</strong>
                <span className='text-muted-foreground mt-1 block'>
                  Comfy 和后台在同一台机器。文件用本地目录，pixoma 会自动拉起本机 Edge。
                </span>
              </span>
            </label>
            <label className='flex items-start gap-3 text-sm'>
              <RadioGroupItem value='remote' />
              <span>
                <strong>远程</strong>
                <span className='text-muted-foreground mt-1 block'>
                  GPU 在别的机器。必须用对象存储（S3 / TOS），不能用本机目录。
                </span>
              </span>
            </label>
          </RadioGroup>
          <Button
            disabled={pending}
            onClick={() => {
              if (placement === 'local') setBlobDriver('localfs')
              else if (blobDriver === 'localfs') setBlobDriver('s3')
              setStep('storage')
            }}
          >
            下一步
          </Button>
        </div>
      ) : null}

      {step === 'storage' ? (
        <form
          className='space-y-4'
          onSubmit={(e) => {
            e.preventDefault()
            void run(async () => {
              await saveDraftAnd('edge')
            })
          }}
        >
          <Field label='对象存储'>
            <select
              className='border-input h-9 w-full rounded-md border bg-transparent px-3 text-sm'
              value={blobDriver}
              onChange={(e) => setBlobDriver(e.target.value)}
            >
              {placement === 'local' ? <option value='localfs'>本机目录</option> : null}
              <option value='s3'>S3 兼容</option>
              <option value='tos'>火山 TOS</option>
            </select>
          </Field>
          {blobDriver === 'localfs' ? (
            <Field label='目录'>
              <Input value={blobRoot} onChange={(e) => setBlobRoot(e.target.value)} />
            </Field>
          ) : (
            <>
              <Field label='Endpoint'>
                <Input value={blobEndpoint} onChange={(e) => setBlobEndpoint(e.target.value)} />
              </Field>
              <Field label='Region'>
                <Input value={blobRegion} onChange={(e) => setBlobRegion(e.target.value)} />
              </Field>
              <Field label='Bucket'>
                <Input value={blobBucket} onChange={(e) => setBlobBucket(e.target.value)} />
              </Field>
              <Field label='Access Key'>
                <Input value={blobAccessKey} onChange={(e) => setBlobAccessKey(e.target.value)} />
              </Field>
              <Field label='Secret Key'>
                <Input type='password' value={blobSecretKey} onChange={(e) => setBlobSecretKey(e.target.value)} />
              </Field>
            </>
          )}
          <Actions error={error} pending={pending} submit='保存存储' />
        </form>
      ) : null}

      {step === 'edge' ? (
        <div className='space-y-4'>
          <Field label='节点 ID'>
            <Input value={instanceId} onChange={(e) => setInstanceId(e.target.value)} />
          </Field>
          <Field label='Comfy 地址'>
            <Input value={comfyURL} onChange={(e) => setComfyURL(e.target.value)} />
          </Field>
          <label className='flex items-center gap-2 text-sm'>
            <input type='checkbox' checked={comfyMock} onChange={(e) => setComfyMock(e.target.checked)} />
            使用 Comfy Mock（没有真机时勾上，仍能走完出图）
          </label>
          {placement === 'remote' ? (
            <pre className='bg-muted overflow-x-auto rounded-md p-3 text-xs'>
{`# 在 GPU 机器上启动执行面（出站连控制面，不要用本机目录）
export CONTROL_PLANE_URL=<控制面地址>
export AGENT_TOKEN=<data/agent.token 里的内容>
export INSTANCE_ID=${instanceId}
export BLOB_DRIVER=${blobDriver === 'localfs' ? 's3' : blobDriver}
export COMFY_MOCK=${comfyMock ? 'true' : 'false'}
pixoma-edge-agent`}
            </pre>
          ) : (
            <p className='text-muted-foreground text-sm'>
              本机部署会由 pixoma 自动拉起 Edge。也可以设 EDGE_AUTO_SPAWN=0 后手动启动 pixoma-edge-agent。
            </p>
          )}
          <Button
            disabled={pending}
            onClick={() => {
              void run(async () => {
                await saveDraftAnd('channel')
              })
            }}
          >
            下一步
          </Button>
          {error ? <p className='text-sm text-destructive'>{error}</p> : null}
        </div>
      ) : null}

      {step === 'channel' ? (
        <form
          className='space-y-4'
          onSubmit={(e) => {
            e.preventDefault()
            void run(async () => {
              await saveSetupDraft({
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
                comfy_mock: comfyMock,
                comfyui_base_url: comfyURL,
                default_instance_id: instanceId,
                auto_spawn_edge: placement === 'local',
                telegram_bot_token: tgToken,
              })
              const res = await finalizeSetup()
              setRestartMessage(res.message)
            })
          }}
        >
          <Field label='Telegram Bot Token（可先留空，之后用环境变量补）'>
            <Input type='password' value={tgToken} onChange={(e) => setTgToken(e.target.value)} />
          </Field>
          <Actions error={error} pending={pending} submit='完成并提示重启' />
        </form>
      ) : null}
    </Shell>
  )
}

function Shell({
  title,
  desc,
  children,
}: {
  title: string
  desc: string
  children: React.ReactNode
}) {
  return (
    <div className='flex min-h-svh items-center justify-center p-6'>
      <Card className='w-full max-w-xl'>
        <CardHeader>
          <CardTitle>{title}</CardTitle>
          <CardDescription>{desc}</CardDescription>
        </CardHeader>
        <CardContent className='space-y-4'>{children}</CardContent>
      </Card>
    </div>
  )
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className='space-y-2'>
      <Label>{label}</Label>
      {children}
    </div>
  )
}

function Actions({
  error,
  pending,
  submit,
}: {
  error: string | null
  pending: boolean
  submit: string
}) {
  return (
    <div className='space-y-2'>
      {error ? <p className='text-sm text-destructive'>{error}</p> : null}
      <Button type='submit' disabled={pending}>
        {pending ? '处理中…' : submit}
      </Button>
    </div>
  )
}
