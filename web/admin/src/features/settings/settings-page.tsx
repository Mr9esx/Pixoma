import { useState, type FormEvent, type ReactNode } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { ErrorBanner } from '@/components/feedback/error-banner'
import { LoadingSkeleton } from '@/components/feedback/loading-skeleton'
import { Button } from '@/components/ui/button'
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
import { Separator } from '@/components/ui/separator'
import { Switch } from '@/components/ui/switch'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { queryKeys } from '@/lib/api/query-keys'
import {
  changeAdminPassword,
  fetchPlatformSettings,
  savePlatformSettings,
  waitForSetupReady,
  type SetupDraft,
} from '@/lib/api/setup'

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

export function SettingsPage() {
  const { t } = useTranslation()
  const q = useQuery({
    queryKey: queryKeys.settings.all,
    queryFn: fetchPlatformSettings,
  })

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
        <SettingsEditor key={q.dataUpdatedAt} initial={q.data.settings} />
      ) : null}
    </div>
  )
}

function SettingsEditor({ initial }: { initial: SetupDraft }) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [placement, setPlacement] = useState<'local' | 'remote'>(
    initial.placement === 'remote' ? 'remote' : 'local',
  )
  const [blobDriver, setBlobDriver] = useState(initial.blob_driver || 'localfs')
  const [blobRoot, setBlobRoot] = useState(initial.blob_root ?? '')
  const [blobEndpoint, setBlobEndpoint] = useState(initial.blob_endpoint ?? '')
  const [blobRegion, setBlobRegion] = useState(initial.blob_region ?? '')
  const [blobBucket, setBlobBucket] = useState(initial.blob_bucket ?? '')
  const [blobAccessKey, setBlobAccessKey] = useState(
    emptyIfMasked(initial.blob_access_key),
  )
  const [blobSecretKey, setBlobSecretKey] = useState(
    emptyIfMasked(initial.blob_secret_key),
  )
  const [autoSpawn, setAutoSpawn] = useState(Boolean(initial.auto_spawn_edge))
  const [proxyKind, setProxyKind] = useState(initial.proxy_kind || '')
  const [proxyHost, setProxyHost] = useState(initial.proxy_host ?? '')
  const [proxyPort, setProxyPort] = useState(
    initial.proxy_port && initial.proxy_port > 0 ? String(initial.proxy_port) : '',
  )
  const [error, setError] = useState<string | null>(null)
  const [pending, setPending] = useState(false)
  const [reloading, setReloading] = useState(false)

  function onPlacementChange(next: string) {
    const value = next === 'remote' ? 'remote' : 'local'
    setPlacement(value)
    if (value === 'remote' && blobDriver === 'localfs') {
      setBlobDriver('s3')
    }
    if (value === 'remote') {
      setAutoSpawn(false)
    }
  }

  async function savePlatform() {
    setPending(true)
    setError(null)
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
        comfy_mock: initial.comfy_mock,
        comfyui_base_url: initial.comfyui_base_url,
        default_edge_id: initial.default_edge_id,
        auto_spawn_edge: placement === 'local' ? autoSpawn : false,
        proxy_kind: proxyKind,
        proxy_host: proxyHost,
        proxy_port: Number(proxyPort) || 0,
      })
      setReloading(true)
      await waitForSetupReady()
      await queryClient.invalidateQueries({ queryKey: queryKeys.settings.all })
      toast.success(t('common.successSaved'))
    } catch (err) {
      setError(errorMessage(err) ?? t('common.errorGeneric'))
    } finally {
      setPending(false)
      setReloading(false)
    }
  }

  if (reloading) {
    return (
      <p className='text-sm text-muted-foreground'>
        {error ?? t('settings.reloadingWait')}
      </p>
    )
  }

  return (
    <Tabs defaultValue='account' className='max-w-2xl gap-4'>
      <TabsList>
        <TabsTrigger value='account'>{t('settings.tabAccount')}</TabsTrigger>
        <TabsTrigger value='storage'>{t('settings.tabStorage')}</TabsTrigger>
        <TabsTrigger value='network'>{t('settings.tabNetwork')}</TabsTrigger>
      </TabsList>

      <TabsContent value='account' className='flex flex-col gap-6'>
        <PasswordForm />
        <section className='flex flex-col'>
          <h2 className='text-sm font-medium'>{t('settings.databaseTitle')}</h2>
          <SettingRow label={t('settings.fieldDbDriver')}>
            <p className='text-sm'>{initial.db_driver}</p>
          </SettingRow>
          <Separator />
          <SettingRow label={t('settings.fieldDbDsn')}>
            <p className='break-all text-sm'>{initial.db_dsn}</p>
          </SettingRow>
        </section>
      </TabsContent>

      <TabsContent value='storage'>
        <form
          className='flex flex-col'
          onSubmit={(e) => {
            e.preventDefault()
            void savePlatform()
          }}
        >
          <SettingRow label={t('settings.fieldPlacement')} htmlFor='placement'>
            <Select value={placement} onValueChange={onPlacementChange}>
              <SelectTrigger id='placement' className='w-full sm:max-w-xs'>
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
          <Separator />
          <SettingRow
            label={t('settings.fieldBlobDriver')}
            htmlFor='blob-driver'
          >
            <Select value={blobDriver} onValueChange={setBlobDriver}>
              <SelectTrigger id='blob-driver' className='w-full sm:max-w-xs'>
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
          <Separator />
          {blobDriver === 'localfs' ? (
            <SettingRow label={t('settings.fieldBlobRoot')} htmlFor='blob-root'>
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
              <Separator />
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
              <Separator />
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
              <Separator />
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
              <Separator />
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
          {placement === 'local' ? (
            <>
              <Separator />
              <SettingRow
                label={t('settings.fieldAutoSpawn')}
                htmlFor='auto-spawn'
              >
                <Switch
                  id='auto-spawn'
                  checked={autoSpawn}
                  onCheckedChange={setAutoSpawn}
                  disabled={pending}
                />
              </SettingRow>
            </>
          ) : null}
          {error ? (
            <p className='pt-3 text-sm text-destructive'>{error}</p>
          ) : null}
          <div className='pt-4'>
            <Button type='submit' size='sm' disabled={pending}>
              {pending ? t('common.loading') : t('common.save')}
            </Button>
          </div>
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
          <SettingRow label={t('settings.fieldProxyKind')} htmlFor='proxy-kind'>
            <Select
              value={proxyKind || 'off'}
              onValueChange={(value) =>
                setProxyKind(value === 'off' ? '' : value)
              }
            >
              <SelectTrigger id='proxy-kind' className='w-full sm:max-w-xs'>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
                  <SelectItem value='off'>{t('settings.proxyOff')}</SelectItem>
                  <SelectItem value='http'>{t('settings.proxyHTTP')}</SelectItem>
                  <SelectItem value='socks5'>
                    {t('settings.proxySOCKS')}
                  </SelectItem>
                </SelectGroup>
              </SelectContent>
            </Select>
          </SettingRow>
          <Separator />
          <SettingRow label={t('settings.fieldProxyHost')} htmlFor='proxy-host'>
            <Input
              id='proxy-host'
              value={proxyHost}
              onChange={(e) => setProxyHost(e.target.value)}
              disabled={pending || !proxyKind}
              autoComplete='off'
            />
          </SettingRow>
          <Separator />
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
          {error ? (
            <p className='pt-3 text-sm text-destructive'>{error}</p>
          ) : null}
          <div className='pt-4'>
            <Button type='submit' size='sm' disabled={pending}>
              {pending ? t('common.loading') : t('common.save')}
            </Button>
          </div>
        </form>
      </TabsContent>

    </Tabs>
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
      setError(errorMessage(err) ?? t('common.errorGeneric'))
    } finally {
      setPending(false)
    }
  }

  return (
    <form className='flex flex-col' onSubmit={onSubmit}>
      <h2 className='text-sm font-medium'>{t('settings.passwordTitle')}</h2>
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
      <Separator />
      <SettingRow label={t('settings.fieldNewPassword')} htmlFor='new-password'>
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
      <Separator />
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
      {error ? <p className='pt-3 text-sm text-destructive'>{error}</p> : null}
      <div className='pt-4'>
        <Button type='submit' size='sm' disabled={pending}>
          {pending ? t('common.loading') : t('settings.changePassword')}
        </Button>
      </div>
    </form>
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
    <div className='grid gap-2 py-3 sm:grid-cols-[9rem_minmax(0,1fr)] sm:items-center'>
      <Label htmlFor={htmlFor}>{label}</Label>
      <div className='flex min-w-0 flex-col gap-1'>
        {children}
        {hint ? (
          <p className='text-xs text-muted-foreground'>{hint}</p>
        ) : null}
      </div>
    </div>
  )
}
