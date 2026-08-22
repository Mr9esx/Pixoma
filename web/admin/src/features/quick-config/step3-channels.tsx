import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { toast } from 'sonner'
import { useTranslation } from 'react-i18next'
import { listChannels } from '@/lib/api/channels'
import { getCaseMenuPlacements, type MenuPlacement } from '@/lib/api/channel-menu'
import { queryKeys } from '@/lib/api/query-keys'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { LoadingSkeleton } from '@/components/feedback/loading-skeleton'
import type { WorkflowMenuMode } from './lib/menu-payload'
import { WizardChrome } from './wizard-chrome'
import type { StepActions, WizardShared } from './types'

type Props = StepActions & { shared: WizardShared }

/** Step 3 投放：展示已有投放，为已启用渠道添加 open_workflow 菜单入口。 */
export function Step3Channels({ shared, next, back }: Props) {
  const caseId = shared.caseId
  const { t } = useTranslation()
  const [drafts, setDrafts] = useState<
    Record<string, { label: string; mode: WorkflowMenuMode }>
  >({})

  const placementsQuery = useQuery({
    queryKey: queryKeys.cases.menuPlacements(caseId ?? -1),
    queryFn: () => getCaseMenuPlacements(caseId as number),
    enabled: caseId != null,
  })
  const channelsQuery = useQuery({
    queryKey: queryKeys.channels.all,
    queryFn: listChannels,
  })

  const placements: MenuPlacement[] = placementsQuery.data ?? []

  function queueEntry(channelId: string, label: string, mode: WorkflowMenuMode) {
    shared.updatePendingEntries([
      ...shared.pendingEntries,
      { channelId, label: label.trim(), mode },
    ])
    setDrafts((prev) => ({ ...prev, [channelId]: { label: '', mode: 'direct' } }))
    toast.success('已加入待提交列表，完成页统一保存')
  }

  function handleNext() {
    shared.updatePlacements(placements)
    next({})
  }

  return (
    <WizardChrome
      step={3}
      onBack={() => back({})}
      onNext={handleNext}
      nextLabel={t('quickConfig.finishChecklist')}
    >
      <div className='space-y-4'>
        <section>
          <h3 className='text-sm font-semibold'>{t('quickConfig.currentPlacements')}</h3>
          {placementsQuery.isLoading ? (
            <LoadingSkeleton rows={2} />
          ) : placements.length === 0 ? (
            <p className='mt-1 text-xs text-muted-foreground'>
              {t('quickConfig.noPlacements')}
            </p>
          ) : (
            <ul className='mt-1 space-y-1'>
              {placements.map((placement) => (
                <li
                  key={`${placement.channel_id}-${placement.item_id}`}
                  className='flex items-center gap-2 rounded-md border border-border px-3 py-2 text-sm'
                >
                  <span className='font-medium'>
                    {placement.channel_name ?? placement.channel_id}
                  </span>
                  <span className='text-muted-foreground'>
                    › {placement.path.map((step) => step.label).join(' › ')}
                  </span>
                </li>
              ))}
            </ul>
          )}
        </section>

        <section>
          <h3 className='text-sm font-semibold'>{t('quickConfig.addChannelEntry')}</h3>
          <div className='mt-2 space-y-2'>
            {channelsQuery.isLoading ? (
              <LoadingSkeleton rows={2} />
            ) : (
              (channelsQuery.data ?? []).map((channel) => {
                const draft = drafts[channel.id] ?? {
                  label: '',
                  mode: 'direct' as const,
                }
                return (
                  <div
                    key={channel.id}
                    className={`rounded-lg border border-border p-3 ${
                      channel.enabled ? '' : 'opacity-60'
                    }`}
                  >
                    <div className='mb-2 flex items-center justify-between'>
                      <span className='text-sm font-medium'>
                        {channel.name}
                      </span>
                      <span className='text-xs text-muted-foreground'>
                        {channel.platform} ·{' '}
                        {channel.enabled ? t('quickConfig.enabled') : t('quickConfig.disabled')}
                      </span>
                    </div>
                    {channel.enabled ? (
                      <div className='grid grid-cols-[1fr_180px_auto] gap-2'>
                        <div>
                          <Label className='text-xs'>按钮标签</Label>
                          <Input
                            value={draft.label}
                            placeholder='例如：开始生成'
                            onChange={(e) =>
                              setDrafts((prev) => ({
                                ...prev,
                                [channel.id]: {
                                  ...draft,
                                  label: e.target.value,
                                },
                              }))
                            }
                          />
                        </div>
                        <div>
                          <Label className='text-xs'>打开方式</Label>
                          <Select
                            value={draft.mode}
                            onValueChange={(mode) =>
                              setDrafts((prev) => ({
                                ...prev,
                                [channel.id]: {
                                  ...draft,
                                  mode: mode as WorkflowMenuMode,
                                },
                              }))
                            }
                          >
                            <SelectTrigger className='w-full'>
                              <SelectValue />
                            </SelectTrigger>
                            <SelectContent>
                              <SelectItem value='direct'>
                                direct · 直接打开
                              </SelectItem>
                              <SelectItem value='list'>
                                list · 列出工作流
                              </SelectItem>
                            </SelectContent>
                          </Select>
                        </div>
                        <div className='flex items-end'>
                          <Button
                            type='button'
                            size='sm'
                            disabled={
                              !draft.label.trim() ||
                              shared.pendingEntries.some(
                                (entry) =>
                                  entry.channelId === channel.id &&
                                  entry.label === draft.label.trim()
                              )
                            }
                            onClick={() =>
                              queueEntry(
                                channel.id,
                                draft.label,
                                draft.mode
                              )
                            }
                          >
                            添加按钮
                          </Button>
                        </div>
                      </div>
                    ) : (
                      <p className='text-xs text-muted-foreground'>
                        {t('quickConfig.enableHint')}
                      </p>
                    )}
                  </div>
                )
              })
            )}
          </div>
        </section>
      </div>
    </WizardChrome>
  )
}
