import { useState, type FormEvent, type ReactNode } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { queryKeys } from '@/lib/api/query-keys'
import {
  changeAdminPassword,
  fetchCurrentUser,
  fetchPlatformSettings,
  saveAdminProfile,
  savePlatformSettings,
  waitForSetupReady,
  type SetupDraft,
} from '@/lib/api/setup'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Field, FieldDescription, FieldLabel } from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { AvatarUpload } from '@/components/avatar-upload'
import { ErrorBanner } from '@/components/feedback/error-banner'
import { LoadingSkeleton } from '@/components/feedback/loading-skeleton'
import { SecretInput } from '@/components/secret-input'
import { AdminUsersPanel } from '@/features/admin-users/admin-users-panel'
import { TextTemplatesEditor } from '@/features/text-templates/text-templates-editor'

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
  const [mediaMax, setMediaMax] = useState<string>(
    initial.media_max_bytes && initial.media_max_bytes > 0
      ? String(initial.media_max_bytes)
      : ''
  )
  const [mediaUnit, setMediaUnit] = useState<'B' | 'KB' | 'MB' | 'GB'>(
    pickMediaUnit(initial.media_max_bytes)
  )
  const [allowSelfRegistration, setAllowSelfRegistration] = useState(
    Boolean(initial.allow_self_registration)
  )
  const [defaultUserAccess, setDefaultUserAccess] = useState(
    normalizeDefaultUserAccess(initial.default_user_access)
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

  async function savePlatform(overrides?: {
    allowSelfRegistration?: boolean
    defaultUserAccess?: DefaultUserAccess
  }) {
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
        media_max_bytes: computeMediaMaxBytes(mediaMax, mediaUnit),
        allow_self_registration:
          overrides?.allowSelfRegistration ?? allowSelfRegistration,
        default_user_access:
          overrides?.defaultUserAccess ?? defaultUserAccess,
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
    <Tabs defaultValue={initialTab ?? 'account'} className='max-w-2xl gap-4'>
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
            <SettingRow
              label={t('settings.fieldPlacement')}
              htmlFor='placement'
            >
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
                  <SecretInput
                    id='blob-secret'
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
        <MediaUploadCard
          value={mediaMax}
          unit={mediaUnit}
          onValueChange={setMediaMax}
          onUnitChange={setMediaUnit}
          onReset={() => {
            setMediaMax('')
            setMediaUnit('MB')
          }}
          onSave={() => void savePlatform()}
          pending={pending}
        />
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
                    <SelectItem value='off'>
                      {t('settings.proxyOff')}
                    </SelectItem>
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
            <SettingRow
              label={t('settings.fieldProxyHost')}
              htmlFor='proxy-host'
            >
              <Input
                id='proxy-host'
                value={proxyHost}
                onChange={(e) => setProxyHost(e.target.value)}
                disabled={pending || !proxyKind}
                autoComplete='off'
              />
            </SettingRow>
            <SettingRow
              label={t('settings.fieldProxyPort')}
              htmlFor='proxy-port'
            >
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
        <SettingsCard title={t('settings.defaultUserAccess')}>
          <SettingRow
            label={t('settings.defaultUserAccess')}
            htmlFor='default-user-access'
          >
            <Select
              value={defaultUserAccess}
              onValueChange={(next) => {
                const value = normalizeDefaultUserAccess(next)
                setDefaultUserAccess(value)
                void savePlatform({ defaultUserAccess: value })
              }}
            >
              <SelectTrigger
                id='default-user-access'
                className='w-full'
                disabled={pending}
              >
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
                  <SelectItem value='always_allowed'>
                    {t('users.accessAlwaysAllowed')}
                  </SelectItem>
                  <SelectItem value='paid'>
                    {t('users.accessPaid')}
                  </SelectItem>
                  <SelectItem value='denied'>
                    {t('users.accessDenied')}
                  </SelectItem>
                </SelectGroup>
              </SelectContent>
            </Select>
          </SettingRow>
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
  const [avatarUrl, setAvatarUrl] = useState(profile.avatar_url ?? '')
  const saveProfile = useMutation({
    mutationFn: () =>
      saveAdminProfile({
        nickname: nickname.trim(),
        email: email.trim(),
        avatarUrl: avatarUrl.trim(),
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
            <Button type='submit' disabled={saveProfile.isPending}>
              {saveProfile.isPending
                ? t('common.loading')
                : t('settings.saveProfile')}
            </Button>
          </div>
        }
      >
        <div className='grid grid-cols-[auto_minmax(0,1fr)] items-stretch gap-3 py-3'>
          <div className='flex h-full items-center justify-center'>
            <FieldLabel htmlFor='profile-avatar' className='sr-only'>
              {t('settings.fieldAvatar')}
            </FieldLabel>
            <AvatarUpload
              id='profile-avatar'
              value={avatarUrl}
              onChange={(next) => setAvatarUrl(next ?? '')}
              disabled={saveProfile.isPending}
            />
          </div>
          <div className='flex min-w-0 flex-col gap-4'>
            <Field className='gap-1.5'>
              <FieldLabel
                htmlFor='profile-nickname'
                className='text-sm font-medium'
              >
                {t('settings.fieldNickname')}
              </FieldLabel>
              <Input
                id='profile-nickname'
                value={nickname}
                onChange={(e) => setNickname(e.target.value)}
                disabled={saveProfile.isPending}
                autoComplete='off'
              />
            </Field>
            <Field className='gap-1.5'>
              <FieldLabel
                htmlFor='profile-email'
                className='text-sm font-medium'
              >
                {t('settings.fieldEmail')}
              </FieldLabel>
              <Input
                id='profile-email'
                type='email'
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                disabled={saveProfile.isPending}
                autoComplete='email'
              />
            </Field>
          </div>
        </div>
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
          <SecretInput
            id='old-password'
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
          <SecretInput
            id='new-password'
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
          <SecretInput
            id='confirm-password'
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
      <CardHeader className='px-5 pt-4 pb-2'>
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
    <Field className='gap-1.5 py-3'>
      <FieldLabel htmlFor={htmlFor} className='text-sm font-medium'>
        {label}
      </FieldLabel>
      {children}
      {hint ? (
        <FieldDescription className='text-xs'>{hint}</FieldDescription>
      ) : null}
    </Field>
  )
}

type MediaUnit = 'B' | 'KB' | 'MB' | 'GB'
type DefaultUserAccess = 'always_allowed' | 'paid' | 'denied'

function normalizeDefaultUserAccess(
  value: string | undefined
): DefaultUserAccess {
  if (value === 'always_allowed' || value === 'paid') return value
  return 'denied'
}

const MEDIA_UNIT_FACTORS: Record<MediaUnit, number> = {
  B: 1,
  KB: 1024,
  MB: 1024 * 1024,
  GB: 1024 * 1024 * 1024,
}

/** 选择最贴合给定字节数的展示单位。 */
function pickMediaUnit(bytes: number | undefined): MediaUnit {
  if (!bytes || bytes <= 0) return 'MB'
  if (bytes >= MEDIA_UNIT_FACTORS.GB && bytes % MEDIA_UNIT_FACTORS.GB === 0)
    return 'GB'
  if (bytes >= MEDIA_UNIT_FACTORS.MB && bytes % MEDIA_UNIT_FACTORS.MB === 0)
    return 'MB'
  if (bytes >= MEDIA_UNIT_FACTORS.KB && bytes % MEDIA_UNIT_FACTORS.KB === 0)
    return 'KB'
  return 'B'
}

/** 把「数值 + 单位」合成字节数；空串 / 非数字 / 非法单位一律回退 0。 */
function computeMediaMaxBytes(value: string, unit: MediaUnit): number {
  if (!value.trim()) return 0
  const n = Number(value)
  if (!Number.isFinite(n) || n < 0) return 0
  return Math.round(n * MEDIA_UNIT_FACTORS[unit])
}

function MediaUploadCard({
  value,
  unit,
  onValueChange,
  onUnitChange,
  onReset,
  onSave,
  pending,
}: {
  value: string
  unit: MediaUnit
  onValueChange: (next: string) => void
  onUnitChange: (next: MediaUnit) => void
  onReset: () => void
  onSave: () => void
  pending: boolean
}) {
  const { t } = useTranslation()
  const bytes = computeMediaMaxBytes(value, unit)
  const invalid = value.trim() !== '' && bytes <= 0
  return (
    <SettingsCard
      title={t('settings.mediaUploadTitle')}
      desc={t('settings.mediaUploadDesc')}
      footer={
        <div className='flex w-full flex-col gap-2 sm:flex-row sm:items-center sm:justify-end'>
          <Button
            type='button'
            variant='ghost'
            onClick={onReset}
            disabled={pending}
          >
            {t('settings.mediaUploadReset')}
          </Button>
          <Button type='button' onClick={onSave} disabled={pending || invalid}>
            {pending ? t('common.loading') : t('common.save')}
          </Button>
        </div>
      }
    >
      <SettingRow
        label={t('settings.mediaUploadMaxLabel')}
        hint={t('settings.mediaUploadMaxHint')}
        htmlFor='media-max'
      >
        <div className='flex items-center gap-2'>
          <Input
            id='media-max'
            inputMode='numeric'
            value={value}
            onChange={(e) => onValueChange(e.target.value)}
            disabled={pending}
            placeholder='25'
            aria-invalid={invalid}
            className='w-28 tabular-nums'
          />
          <Select
            value={unit}
            onValueChange={(next) => onUnitChange(next as MediaUnit)}
          >
            <SelectTrigger
              aria-label={t('settings.mediaUploadMaxLabel')}
              className='w-24 tabular-nums'
            >
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectGroup>
                <SelectItem value='B'>
                  {t('settings.mediaUploadUnitBytes')}
                </SelectItem>
                <SelectItem value='KB'>
                  {t('settings.mediaUploadUnitKB')}
                </SelectItem>
                <SelectItem value='MB'>
                  {t('settings.mediaUploadUnitMB')}
                </SelectItem>
                <SelectItem value='GB'>
                  {t('settings.mediaUploadUnitGB')}
                </SelectItem>
              </SelectGroup>
            </SelectContent>
          </Select>
        </div>
      </SettingRow>
    </SettingsCard>
  )
}
