import {
  useState,
  type FormEvent,
  type ReactNode,
} from 'react'
import {
  useMutation,
  useQuery,
  useQueryClient,
} from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { queryKeys } from '@/lib/api/query-keys'
import { cn } from '@/lib/utils'
import {
  changeAdminPassword,
  fetchCurrentUser,
  fetchPlatformSettings,
  saveAdminProfile,
  savePlatformSettings,
  waitForSetupReady,
  type SetupDraft,
} from '@/lib/api/setup'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
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
import { Switch } from '@/components/ui/switch'
import { AdminUsersPanel } from '@/features/admin-users/admin-users-panel'
import { TextTemplatesEditor } from '@/features/text-templates/text-templates-editor'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { ErrorBanner } from '@/components/feedback/error-banner'
import { LoadingSkeleton } from '@/components/feedback/loading-skeleton'

function errorMessage(err: unknown): string | undefined {
  return err instanceof Error ? err.message : undefined
}

function secretPayload(value: string): string {
  if (!value || value === '********') return ''
  return value
}

function emptyIfMasked(value: string | undefined): string {
  if (!value || value === '********') return ''
  return value
}

export function SettingsPage({ initialTab }: { initialTab?: string }) {
  const { t } = useTranslation()
  const q = useQuery({
    queryKey: queryKeys.settings.all,
    queryFn: fetchPlatformSettings,
  })
  const tab =
    initialTab === 'storage' ||
    initialTab === 'network' ||
    initialTab === 'users' ||
    initialTab === 'text'
      ? initialTab
      : 'account'

  return (
    <div
      className='flex min-h-0 flex-1 flex-col gap-4 overflow-y-auto'
      data-testid='settings-page'
    >
      <div className='shrink-0'>
        <h1 className='text-2xl font-bold tracking-tight'>
          {t('settings.title')}
        </h1>
      </div>
      {q.isLoading ? <LoadingSkeleton rows={8} /> : null}
      {q.isError ? (
        <ErrorBanner
          message={errorMessage(q.error)}
          onRetry={() => void q.refetch()}
        />
      ) : null}
      {q.data?.configured && q.data.settings ? (
        <SettingsEditor
          key={q.dataUpdatedAt}
          initial={q.data.settings}
          initialTab={tab}
        />
      ) : null}
    </div>
  )
}

function SettingsEditor({
  initial,
  initialTab,
}: {
  initial: SetupDraft
  initialTab?: string
}) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [placement, setPlacement] = useState<'local' | 'remote'>(
    initial.placement === 'remote' ? 'remote' : 'local'
  )
  const [blobDriver, setBlobDriver] = useState(initial.blob_driver || 'localfs')
  const [blobRoot, setBlobRoot] = useState(initial.blob_root ?? '')
  const [blobEndpoint, setBlobEndpoint] = useState(initial.blob_endpoint ?? '')
  const [blobRegion, setBlobRegion] = useState(initial.blob_region ?? '')
  const [blobBucket, setBlobBucket] = useState(initial.blob_bucket ?? '')
  const [blobAccessKey, setBlobAccessKey] = useState(
    emptyIfMasked(initial.blob_access_key)
  )
  const [blobSecretKey, setBlobSecretKey] = useState(
    emptyIfMasked(initial.blob_secret_key)
  )
  const [proxyKind, setProxyKind] = useState(initial.proxy_kind || '')
  const [proxyHost, setProxyHost] = useState(initial.proxy_host ?? '')
  const [proxyPort, setProxyPort] = useState(
    initial.proxy_port && initial.proxy_port > 0
      ? String(initial.proxy_port)
      : ''
  )
  const [allowSelfRegistration, setAllowSelfRegistration] = useState(
    Boolean(initial.allow_self_registration)
  )
  const [pending, setPending] = useState(false)
  const [reloading, setReloading] = useState(false)

  function onPlacementChange(next: string) {
    const value = next === 'remote' ? 'remote' : 'local'
    setPlacement(value)
    if (value === 'remote' && blobDriver === 'localfs') {
      setBlobDriver('s3')
    }
  }

  async function savePlatform(overrides?: { allowSelfRegistration?: boolean }) {
    setPending(true)
    try {
      await savePlatformSettings({
        placement,
        db_driver: initial.db_driver,
        db_dsn: initial.db_dsn,
        blob_driver: blobDriver,
        blob_root: blobRoot,
        blob_endpoint: blobEndpoint,
        blob_region: blobRegion,
        blob_bucket: blobBucket,
        blob_access_key: secretPayload(blobAccessKey),
        blob_secret_key: secretPayload(blobSecretKey),
        comfyui_base_url: initial.comfyui_base_url,
        proxy_kind: proxyKind,
        proxy_host: proxyHost,
        proxy_port: Number(proxyPort) || 0,
        allow_self_registration:
          overrides?.allowSelfRegistration ?? allowSelfRegistration,
      })
      setReloading(true)
      await waitForSetupReady()
      await queryClient.invalidateQueries({ queryKey: queryKeys.settings.all })
      toast.success(t('common.successSaved'))
    } catch (err) {
      toast.error(errorMessage(err) ?? t('common.errorGeneric'))
    } finally {
      setPending(false)
      setReloading(false)
    }
  }

  if (reloading) {
    return (
      <p className='text-sm text-muted-foreground'>
        {t('settings.reloadingWait')}
      </p>
    )
  }

  return (
    <Tabs
      defaultValue={initialTab ?? 'account'}
      className='max-w-2xl gap-4'
    >
      <TabsList>
        <TabsTrigger value='account'>{t('settings.tabAccount')}</TabsTrigger>
        <TabsTrigger value='storage'>{t('settings.tabStorage')}</TabsTrigger>
        <TabsTrigger value='network'>{t('settings.tabNetwork')}</TabsTrigger>
        <TabsTrigger value='users'>{t('settings.tabUsers')}</TabsTrigger>
        <TabsTrigger value='text'>{t('settings.tabText')}</TabsTrigger>
    </TabsList>

      <TabsContent value='account' className='flex flex-col gap-4'>
        <ProfileForm />
        <PasswordForm />
      </TabsContent>

      <TabsContent value='storage' className='flex flex-col gap-4'>
        <SettingsCard
          title={t('settings.databaseTitle')}
          desc={t('settings.databaseDesc')}
        >
          <SettingRow label={t('settings.fieldDbDriver')}>
            <p className='text-sm'>{initial.db_driver}</p>
          </SettingRow>
          <SettingRow label={t('settings.fieldDbDsn')}>
            <p className='text-sm break-all'>{initial.db_dsn}</p>
          </SettingRow>
        </SettingsCard>
        <form
          className='flex flex-col'
          onSubmit={(e) => {
            e.preventDefault()
            void savePlatform()
          }}
        >
          <SettingsCard
            title={t('settings.fieldBlobDriver')}
            desc={t('settings.storageDesc')}
            footer={
              <div className='flex w-full flex-col gap-2 sm:flex-row sm:items-center sm:justify-end'>
                <Button type='submit' disabled={pending}>
                  {pending ? t('common.loading') : t('common.save')}
                </Button>
              </div>
            }
          >
            <SettingRow label={t('settings.fieldPlacement')} htmlFor='placement'>
              <Select value={placement} onValueChange={onPlacementChange}>
                <SelectTrigger id='placement' className='w-full'>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectGroup>
                    <SelectItem value='local'>
                      {t('settings.placementLocal')}
                    </SelectItem>
                    <SelectItem value='remote'>
                      {t('settings.placementRemote')}
                    </SelectItem>
                  </SelectGroup>
                </SelectContent>
              </Select>
            </SettingRow>
            <SettingRow
              label={t('settings.fieldBlobDriver')}
              htmlFor='blob-driver'
            >
              <Select value={blobDriver} onValueChange={setBlobDriver}>
                <SelectTrigger id='blob-driver' className='w-full'>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectGroup>
                    {placement === 'local' ? (
                      <SelectItem value='localfs'>
                        {t('settings.blobLocalfs')}
                      </SelectItem>
                    ) : null}
                    <SelectItem value='s3'>{t('settings.blobS3')}</SelectItem>
                    <SelectItem value='tos'>{t('settings.blobTos')}</SelectItem>
                  </SelectGroup>
                </SelectContent>
              </Select>
            </SettingRow>
            {blobDriver === 'localfs' ? (
              <SettingRow
                label={t('settings.fieldBlobRoot')}
                htmlFor='blob-root'
              >
                <Input
                  id='blob-root'
                  value={blobRoot}
                  onChange={(e) => setBlobRoot(e.target.value)}
                  disabled={pending}
                />
              </SettingRow>
            ) : (
              <>
                <SettingRow
                  label={t('settings.fieldEndpoint')}
                  htmlFor='blob-endpoint'
                >
                  <Input
                    id='blob-endpoint'
                    value={blobEndpoint}
                    onChange={(e) => setBlobEndpoint(e.target.value)}
                    disabled={pending}
                  />
                </SettingRow>
                <SettingRow
                  label={t('settings.fieldRegion')}
                  htmlFor='blob-region'
                >
                  <Input
                    id='blob-region'
                    value={blobRegion}
                    onChange={(e) => setBlobRegion(e.target.value)}
                    disabled={pending}
                  />
                </SettingRow>
                <SettingRow
                  label={t('settings.fieldBucket')}
                  htmlFor='blob-bucket'
                >
                  <Input
                    id='blob-bucket'
                    value={blobBucket}
                    onChange={(e) => setBlobBucket(e.target.value)}
                    disabled={pending}
                  />
                </SettingRow>
                <SettingRow
                  label={t('settings.fieldAccessKey')}
                  htmlFor='blob-access'
                  hint={t('settings.secretHint')}
                >
                  <Input
                    id='blob-access'
                    value={blobAccessKey}
                    onChange={(e) => setBlobAccessKey(e.target.value)}
                    disabled={pending}
                    autoComplete='off'
                  />
                </SettingRow>
                <SettingRow
                  label={t('settings.fieldSecretKey')}
                  htmlFor='blob-secret'
                >
                  <Input
                    id='blob-secret'
                    type='password'
                    value={blobSecretKey}
                    onChange={(e) => setBlobSecretKey(e.target.value)}
                    disabled={pending}
                    autoComplete='off'
                  />
                </SettingRow>
              </>
            )}
          </SettingsCard>
        </form>
      </TabsContent>

      <TabsContent value='network'>
        <form
          className='flex flex-col'
          onSubmit={(e) => {
            e.preventDefault()
            void savePlatform()
          }}
        >
          <SettingsCard
            title={t('settings.networkTitle')}
            desc={t('settings.networkDesc')}
            footer={
              <div className='flex w-full flex-col gap-2 sm:flex-row sm:items-center sm:justify-end'>
                <Button type='submit' disabled={pending}>
                  {pending ? t('common.loading') : t('common.save')}
                </Button>
              </div>
            }
          >
            <SettingRow
              label={t('settings.fieldProxyKind')}
              htmlFor='proxy-kind'
            >
              <Select
                value={proxyKind || 'off'}
                onValueChange={(value) =>
                  setProxyKind(value === 'off' ? '' : value)
                }
              >
                <SelectTrigger id='proxy-kind' className='w-full'>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectGroup>
                    <SelectItem value='off'>{t('settings.proxyOff')}</SelectItem>
                    <SelectItem value='http'>
                      {t('settings.proxyHTTP')}
                    </SelectItem>
                    <SelectItem value='socks5'>
                      {t('settings.proxySOCKS')}
                    </SelectItem>
                  </SelectGroup>
                </SelectContent>
              </Select>
            </SettingRow>
            <SettingRow label={t('settings.fieldProxyHost')} htmlFor='proxy-host'>
              <Input
                id='proxy-host'
                value={proxyHost}
                onChange={(e) => setProxyHost(e.target.value)}
                disabled={pending || !proxyKind}
                autoComplete='off'
              />
            </SettingRow>
            <SettingRow label={t('settings.fieldProxyPort')} htmlFor='proxy-port'>
              <Input
                id='proxy-port'
                inputMode='numeric'
                value={proxyPort}
                onChange={(e) => setProxyPort(e.target.value)}
                disabled={pending || !proxyKind}
                autoComplete='off'
              />
            </SettingRow>
          </SettingsCard>
        </form>
      </TabsContent>

      <TabsContent value='users' className='flex flex-col gap-4'>
        <SettingsCard
          title={t('settings.registrationTitle')}
          desc={t('settings.registrationHint')}
        >
          <div className='flex items-center justify-between gap-4 py-3'>
            <p className='text-sm'>{t('settings.registrationNote')}</p>
            <Switch
              data-testid='allow-self-registration'
              aria-label={t('settings.registrationTitle')}
              checked={allowSelfRegistration}
              onCheckedChange={(v) => {
                setAllowSelfRegistration(Boolean(v))
                void savePlatform({ allowSelfRegistration: Boolean(v) })
              }}
            />
          </div>
        </SettingsCard>
        <AdminUsersPanel />
      </TabsContent>

      <TabsContent value='text' className='flex flex-col gap-4'>
        <TextTemplatesEditor />
      </TabsContent>
    </Tabs>
  )
}

function ProfileForm() {
  const { t } = useTranslation()
  const { data: profile } = useQuery({
    queryKey: ['current-user'],
    queryFn: fetchCurrentUser,
  })

  if (!profile) {
    return (
      <SettingsCard title={t('settings.profileTitle')}>
        <p className='py-3 text-sm text-muted-foreground'>
          {t('common.loading')}
        </p>
      </SettingsCard>
    )
  }

  return <ProfileEditor key={profile.username} profile={profile} />
}

function ProfileEditor({
  profile,
}: {
  profile: NonNullable<Awaited<ReturnType<typeof fetchCurrentUser>>>
}) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [nickname, setNickname] = useState(profile.nickname ?? '')
  const [email, setEmail] = useState(profile.email ?? '')
  const saveProfile = useMutation({
    mutationFn: () =>
      saveAdminProfile({
        nickname: nickname.trim(),
        email: email.trim(),
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['current-user'] })
      toast.success(t('common.successSaved'))
    },
  })

  return (
      <form
        className='flex flex-col'
        onSubmit={(e) => {
          e.preventDefault()
          saveProfile.mutate()
      }}
    >
      <SettingsCard
        title={t('settings.profileTitle')}
        desc={t('settings.profileDesc')}
        footer={
          <div className='flex w-full flex-col gap-2 sm:flex-row sm:items-center sm:justify-end'>
            <Button
              type='submit'
              disabled={saveProfile.isPending}
            >
              {saveProfile.isPending
                ? t('common.loading')
                : t('settings.saveProfile')}
            </Button>
          </div>
        }
      >
        <SettingRow
          label={t('settings.fieldNickname')}
          htmlFor='profile-nickname'
        >
          <Input
            id='profile-nickname'
            value={nickname}
            onChange={(e) => setNickname(e.target.value)}
            disabled={saveProfile.isPending}
            autoComplete='off'
          />
        </SettingRow>
        <SettingRow label={t('settings.fieldEmail')} htmlFor='profile-email'>
          <Input
            id='profile-email'
            type='email'
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            disabled={saveProfile.isPending}
            autoComplete='email'
          />
        </SettingRow>
      </SettingsCard>
    </form>
  )
}

function PasswordForm() {
  const { t } = useTranslation()
  const [oldPassword, setOldPassword] = useState('')
  const [newPassword, setNewPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [pending, setPending] = useState(false)

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    if (newPassword.length < 8) {
      setError(t('settings.passwordTooShort'))
      return
    }
    if (newPassword !== confirmPassword) {
      setError(t('settings.passwordMismatch'))
      return
    }
    setPending(true)
    setError(null)
    try {
      await changeAdminPassword({ oldPassword, newPassword })
      setOldPassword('')
      setNewPassword('')
      setConfirmPassword('')
      toast.success(t('settings.passwordSuccess'))
    } catch (err) {
      toast.error(errorMessage(err) ?? t('common.errorGeneric'))
    } finally {
      setPending(false)
    }
  }

  return (
    <form className='flex flex-col' onSubmit={onSubmit}>
      <SettingsCard
        title={t('settings.passwordTitle')}
        desc={t('settings.passwordDesc')}
        footer={
          <div className='flex w-full flex-col gap-2 sm:flex-row sm:items-center sm:justify-end'>
            {error ? (
              <p className='text-sm text-destructive sm:me-auto'>{error}</p>
            ) : null}
            <Button type='submit' disabled={pending}>
              {pending ? t('common.loading') : t('settings.changePassword')}
            </Button>
          </div>
        }
      >
        <SettingRow
          label={t('settings.fieldCurrentPassword')}
          htmlFor='old-password'
        >
          <Input
            id='old-password'
            type='password'
            autoComplete='current-password'
            value={oldPassword}
            onChange={(e) => setOldPassword(e.target.value)}
            disabled={pending}
            required
          />
        </SettingRow>
        <SettingRow
          label={t('settings.fieldNewPassword')}
          htmlFor='new-password'
        >
          <Input
            id='new-password'
            type='password'
            autoComplete='new-password'
            value={newPassword}
            onChange={(e) => setNewPassword(e.target.value)}
            disabled={pending}
            required
          />
        </SettingRow>
        <SettingRow
          label={t('settings.fieldConfirmPassword')}
          htmlFor='confirm-password'
        >
          <Input
            id='confirm-password'
            type='password'
            autoComplete='new-password'
            value={confirmPassword}
            onChange={(e) => setConfirmPassword(e.target.value)}
            disabled={pending}
            required
          />
        </SettingRow>
      </SettingsCard>
    </form>
  )
}

function SettingsCard({
  title,
  desc,
  footer,
  className,
  children,
}: {
  title: string
  desc?: string
  footer?: ReactNode
  className?: string
  children?: ReactNode
}) {
  return (
    <Card className={cn('gap-0 py-0', className)}>
      <CardHeader className='px-5 pb-2 pt-4'>
        <CardTitle className='text-sm font-semibold'>{title}</CardTitle>
        {desc ? (
          <CardDescription className='text-xs'>{desc}</CardDescription>
        ) : null}
      </CardHeader>
      {children ? (
        <CardContent className='px-5'>
          <div className='divide-y divide-border'>{children}</div>
        </CardContent>
      ) : null}
      {footer ? (
        <CardFooter className='border-t px-5 py-3.5'>{footer}</CardFooter>
      ) : null}
    </Card>
  )
}

function SettingRow({
  label,
  htmlFor,
  hint,
  children,
}: {
  label: string
  htmlFor?: string
  hint?: string
  children: ReactNode
}) {
  return (
    <div className='flex flex-col gap-1.5 py-3'>
      <Label htmlFor={htmlFor} className='text-sm font-medium'>
        {label}
      </Label>
      <div className='flex min-w-0 flex-col gap-1'>
        {children}
        {hint ? <p className='text-xs text-muted-foreground'>{hint}</p> : null}
      </div>
    </div>
  )
}
