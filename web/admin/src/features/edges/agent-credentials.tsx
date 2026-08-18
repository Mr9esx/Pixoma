import { useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { rotateEdgeToken } from '@/lib/api/edges'
import { queryKeys } from '@/lib/api/query-keys'
import type { ComfyEdge } from '@/lib/api/types'
import { Button } from '@/components/ui/button'
import { ConfirmDialog } from '@/components/confirm-dialog'
import { DeployCredentials } from './deploy-credentials'

function errorMessage(err: unknown): string | undefined {
  return err instanceof Error ? err.message : undefined
}

type Props = {
  edge: ComfyEdge
  onEdgeChange?: (edge: ComfyEdge) => void
}

export function AgentCredentials({ edge, onEdgeChange }: Props) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [confirmOpen, setConfirmOpen] = useState(false)

  const rotateMutation = useMutation({
    mutationFn: () => rotateEdgeToken(edge.id),
    onSuccess: async (updated) => {
      setConfirmOpen(false)
      await queryClient.invalidateQueries({ queryKey: queryKeys.edges.all })
      await queryClient.invalidateQueries({
        queryKey: queryKeys.edges.detail(edge.id),
      })
      queryClient.setQueryData(queryKeys.edges.detail(edge.id), updated)
      onEdgeChange?.(updated)
      toast.success(t('common.successSaved'))
    },
    onError: (err) => {
      toast.error(errorMessage(err) ?? t('common.errorGeneric'))
    },
  })

  return (
    <div className='flex flex-col gap-6' data-testid='agent-credentials'>
      <DeployCredentials
        edge={edge}
        tokenActions={
          <Button
            type='button'
            variant='destructive'
            disabled={rotateMutation.isPending}
            onClick={() => setConfirmOpen(true)}
          >
            {t('edges.regenerate')}
          </Button>
        }
      />

      <ConfirmDialog
        open={confirmOpen}
        onOpenChange={setConfirmOpen}
        title={t('edges.regenerateConfirmTitle')}
        desc={t('edges.regenerateConfirmDesc')}
        confirmText={t('edges.regenerate')}
        cancelBtnText={t('common.cancel')}
        destructive
        isLoading={rotateMutation.isPending}
        handleConfirm={() => rotateMutation.mutate()}
      />
    </div>
  )
}
