import { readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const root = join(dirname(fileURLToPath(import.meta.url)), '../../..')

function read(rel: string) {
  return readFileSync(join(root, rel), 'utf8')
}

describe('login and setup pages', () => {
  it('share the split sign-in shell', () => {
    expect(read('src/features/setup/login-page.tsx')).toMatch(/AuthShell/)
    expect(read('src/features/setup/setup-wizard.tsx')).toMatch(/AuthShell/)
    expect(read('src/features/setup/auth-shell.tsx')).toMatch(/lg:grid-cols-2/)
  })

  it('keeps login as a single form without step back', () => {
    const login = read('src/features/setup/login-page.tsx')
    expect(login).not.toMatch(/上一步/)
    expect(login).not.toMatch(/setupStepsFor/)
    expect(login).not.toMatch(/previousSetupStep/)
  })

  it('routes the setup status into the login page so first-run hint can render', () => {
    const route = read('src/routes/login.tsx')
    const page = read('src/features/setup/login-page.tsx')
    expect(route).toMatch(/return \{ status \}/)
    expect(route).toMatch(/Route\.useRouteContext/)
    expect(page).toMatch(/status: SetupStatus/)
  })

  it('shows the default-password hint only on first-run login', () => {
    const page = read('src/features/setup/login-page.tsx')
    expect(page).toMatch(/showFirstRunHint/)
    expect(page).toMatch(/!status\.initialized && status\.must_change_password/)
    expect(page).toMatch(/首次启动系统会生成默认密码/)
    expect(page).toMatch(/在启动日志中搜索/)
    expect(page).toMatch(/Admin password/)
  })

  it('renders the first-run hint below the login card, not inside it', () => {
    const page = read('src/features/setup/login-page.tsx')
    const cardEnd = page.indexOf('</Card>')
    const alertStart = page.indexOf('<Alert')
    expect(cardEnd).toBeGreaterThan(0)
    expect(alertStart).toBeGreaterThan(cardEnd)
  })

  it('uses the auth-shell blockquote as a real elevator pitch, not an onboarding summary', () => {
    const shell = read('src/features/setup/auth-shell.tsx')
    // New pitch text: the product's core value prop
    expect(shell).toMatch(/Pixoma 让你随时随地使用自己的 ComfyUI 进行艺术创作/)
    // Old onboarding-summary phrases must not return
    expect(shell).not.toMatch(/先登录后台/)
    expect(shell).not.toMatch(/还没配过/)
    expect(shell).not.toMatch(/选库/)
    expect(shell).not.toMatch(/这台机器还是远程/)
    // Fake-testimonial framing is gone
    expect(shell).not.toMatch(/「先登录/)
    expect(shell).not.toMatch(/<footer[^>]*>Pixoma<\/footer>/)
  })

  it('uses multi-step only on the setup wizard and allows going back', () => {
    const wizard = read('src/features/setup/setup-wizard.tsx')
    expect(wizard).toMatch(/上一步/)
    expect(wizard).toMatch(/previousSetupStep/)
    expect(wizard).toMatch(/steps\.length/)
  })

  it('keeps the password step description short and action-direct', () => {
    const steps = read('src/features/setup/setup-steps.ts')
    // New: short, single-sentence, action-direct
    expect(steps).toMatch(/设个新的管理员密码/)
    // Old rambling patterns must not return
    expect(steps).not.toMatch(/登录已经验证过/)
    expect(steps).not.toMatch(/启动密码/)
    expect(steps).not.toMatch(/这里设一个/)
    expect(steps).not.toMatch(/你自己记得住的/)
    // Redundant with the form label '再输一遍' must not creep back into the desc
    expect(steps).not.toMatch(/输两遍确认/)
  })

  it('keeps the database step copy to config, not pedagogy', () => {
    const steps = read('src/features/setup/setup-steps.ts')
    const wizard = read('src/features/setup/setup-wizard.tsx')
    // Title is the generic config noun; the step is self-explanatory, so desc is empty
    expect(steps).toMatch(/title: '数据库配置'/)
    expect(steps).toMatch(/database: \{[\s\S]*?title: '数据库配置',\s*desc: '',/)
    expect(steps).toMatch(/submit: '继续'/)
    // Database step uses a structured per-driver form, not a raw one-line DSN
    expect(wizard).toMatch(/htmlFor='db-driver'/)
    expect(wizard).toMatch(/htmlFor='db-sqlite-path'/)
    expect(wizard).toMatch(/htmlFor='db-host'/)
    expect(wizard).toMatch(/htmlFor='db-password'/)
    // Old patterns must not return: any desc for this step, empty deixis,
    // redundant '记录', the deictic question phrasing, and the padded
    // parenthetical on SQLite
    expect(steps).not.toMatch(/数据库配置',[\s\S]*?选库并填连接信息/)
    expect(steps).not.toMatch(/这些记录/)
    expect(steps).not.toMatch(/存在哪/)
    expect(steps).not.toMatch(/业务数据存哪/)
    expect(steps).not.toMatch(/本机先用 SQLite/)
    expect(steps).not.toMatch(/默认 SQLite 文件即可/)
    expect(wizard).not.toMatch(/SQLite（本机文件，适合先跑通）/)
  })

  it('does not expose mock or a ComfyUI node step', () => {
    const wizard = read('src/features/setup/setup-wizard.tsx')
    const steps = read('src/features/setup/setup-steps.ts')
    expect(wizard).not.toMatch(/Comfy Mock/)
    expect(wizard).not.toMatch(/htmlFor='comfy-url'/)
    expect(wizard).not.toMatch(/ComfyUI 节点/)
    expect(steps).not.toMatch(/ComfyUI 节点/)
    expect(wizard).toMatch(/comfyui_base_url: ''/)
    expect(wizard).toMatch(/暂时跳过/)
  })

  it('asks for the new password twice and no longer collects a Telegram token', () => {
    const wizard = read('src/features/setup/setup-wizard.tsx')
    expect(wizard).not.toMatch(/htmlFor='old-password'/)
    expect(wizard).toMatch(/htmlFor='confirm-password'/)
    expect(wizard).not.toMatch(/telegram_bot_token/)
    expect(wizard).not.toMatch(/tg-token/)
  })

  it('offers per-driver connection fields and extra-param examples', () => {
    const wizard = read('src/features/setup/setup-wizard.tsx')
    expect(wizard).toMatch(/SelectItem value='sqlite'/)
    expect(wizard).toMatch(/SelectItem value='mysql'/)
    expect(wizard).toMatch(/SelectItem value='postgres'/)
    expect(wizard).toMatch(/placeholder='data\/app\.db'/)
    expect(wizard).toMatch(/placeholder='timeout=5s&readTimeout=10s'/)
    expect(wizard).toMatch(/placeholder='connect_timeout=10 application_name=pixoma'/)
  })

  it('switches structured fields and port defaults when driver changes', () => {
    const wizard = read('src/features/setup/setup-wizard.tsx')
    expect(wizard).toMatch(/onValueChange=\{onDriverChange\}/)
    expect(wizard).toMatch(/function onDriverChange\(next: string\)/)
    expect(wizard).toMatch(/setDbPort\('3306'\)/)
    expect(wizard).toMatch(/setDbPort\('5432'\)/)
    expect(wizard).toMatch(/buildMySQLDSN/)
    expect(wizard).toMatch(/buildPostgresDSN/)
  })

  it('lets users append extra connection parameters without a raw DSN editor', () => {
    const wizard = read('src/features/setup/setup-wizard.tsx')
    expect(wizard).toMatch(/label='附加参数' htmlFor='db-extra-params'/)
    expect(wizard).toMatch(/setDbExtraParams/)
    expect(wizard).toMatch(/params: dbExtraParams/)
    expect(wizard).not.toMatch(/高级：直接输入 DSN/)
    expect(wizard).not.toMatch(/db-dsn-raw/)
  })

  it('shows errors as destructive alerts with friendly copy and raw detail', () => {
    const wizard = read('src/features/setup/setup-wizard.tsx')
    expect(wizard).toMatch(/Alert variant='destructive'/)
    expect(wizard).toMatch(/AlertTitle>/)
    expect(wizard).toMatch(/AlertDescription>/)
    expect(wizard).toMatch(/setupErrorCopy/)
    expect(wizard).not.toMatch(/<p className='text-sm text-destructive'>/)
  })

  it('splits connectivity test and continue into separate buttons', () => {
    const wizard = read('src/features/setup/setup-wizard.tsx')
    expect(wizard).toMatch(/连通性测试/)
    expect(wizard).toMatch(/testDatabase\(driver, dsn\)/)
    expect(wizard).toMatch(/setDbTested\(true\)/)
    expect(wizard).not.toMatch(/测连通并继续/)
    expect(wizard).not.toMatch(/submitDisabled/)
    expect(wizard).toMatch(/setStep\('storage'\)/)
    expect(wizard).toMatch(/Alert variant='success'/)
    expect(wizard).toMatch(/AlertTitle>连接正常<\/AlertTitle>/)
    expect(wizard).toMatch(/CircleCheck/)
    expect(wizard).not.toMatch(/text-emerald-600/)
    expect(wizard).toMatch(
      /onSubmit=\{[\s\S]*?await testDatabase\(driver, dsn\)[\s\S]*?setStep\('storage'\)/
    )
  })

  it('removes the deployment placement step', () => {
    const steps = read('src/features/setup/setup-steps.ts')
    const wizard = read('src/features/setup/setup-wizard.tsx')
    expect(steps).not.toMatch(/placement/)
    expect(wizard).not.toMatch(/出图机器在哪/)
    expect(wizard).not.toMatch(/RadioGroup/)
    expect(wizard).toMatch(
      /placement: blobDriver === 'localfs' \? 'local' : 'remote'/
    )
  })

  it('tests blob connectivity and offers bucket creation', () => {
    const wizard = read('src/features/setup/setup-wizard.tsx')
    expect(wizard).toMatch(/testBlob\(/)
    expect(wizard).toMatch(/bucket_not_found/)
    expect(wizard).toMatch(/帮你创建/)
    expect(wizard).toMatch(/auto_create_bucket/)
    expect(wizard).toMatch(/Alert variant='info'/)
    expect(wizard).not.toMatch(/variant='ghost'[\s\S]*?取消/)
    expect(wizard).toMatch(/variant='outline'[\s\S]*?取消/)
    expect(wizard).not.toMatch(/variant='outline'[\s\S]*?创建/)
  })

  it('uses storage step copy and warns for localfs', () => {
    const steps = read('src/features/setup/setup-steps.ts')
    const wizard = read('src/features/setup/setup-wizard.tsx')
    expect(steps).toMatch(/title: '文件存储配置'/)
    expect(steps).toMatch(/desc: '决定了生成的图和视频存放的位置。'/)
    expect(wizard).toMatch(/AlertTitle>注意！<\/AlertTitle>/)
    expect(wizard).toMatch(/AlertDescription>/)
    expect(wizard).toMatch(
      /这个配置只适合 ComfyUI 和后台在同一台机器上使用，无法使用远程节点。/
    )
    expect(wizard).not.toMatch(/<br \/>/)
    expect(wizard).toMatch(/blobDriver === 'localfs'/)
    expect(wizard).toMatch(/Alert variant='warn'/)
    expect(wizard).toMatch(/CircleAlert/)
  })

  it('labels the s3 driver plainly', () => {
    const wizard = read('src/features/setup/setup-wizard.tsx')
    expect(wizard).toMatch(/SelectItem value='s3'>S3<\/SelectItem>/)
    expect(wizard).not.toMatch(/SelectItem value='s3'>S3 兼容/)
  })

  it('prefills tos defaults when the driver is selected', () => {
    const wizard = read('src/features/setup/setup-wizard.tsx')
    expect(wizard).toMatch(/tos-cn-beijing\.volces\.com/)
    expect(wizard).toMatch(/cn-beijing/)
    expect(wizard).toMatch(/TOS_DEFAULTS/)
    expect(wizard).toMatch(/onBlobDriverChange/)
  })

  it('offers shared directory (SMB/NFS) with mount guidance', () => {
    const wizard = read('src/features/setup/setup-wizard.tsx')
    const alert = read('src/components/ui/alert.tsx')
    expect(wizard).toMatch(/共享目录（SMB \/ NFS）/)
    expect(wizard).toMatch(/SelectItem value='sharedfs'/)
    expect(wizard).toMatch(/mount -t nfs/)
    expect(wizard).toMatch(/mount -t cifs/)
    expect(wizard).toMatch(/<server-ip>/)
    expect(wizard).not.toMatch(/192\.168\./)
    expect(wizard).toMatch(/blobDriver === 'sharedfs'/)
    expect(wizard).toMatch(/MinIO/)
    expect(wizard).toMatch(/overflow-x-auto/)
    expect(alert).toMatch(/data-slot='alert-description'[\s\S]*?min-w-0/)
    expect(wizard).toMatch(
      /placement: blobDriver === 'localfs' \? 'local' : 'remote'/
    )
  })
})
