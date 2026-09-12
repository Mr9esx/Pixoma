import { useTranslation } from 'react-i18next'
import { ConfirmDialog } from '@/components/confirm-dialog'

interface SignOutDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onSignOut?: () => void | Promise<void>
}

export function SignOutDialog({
  open,
  onOpenChange,
  onSignOut,
}: SignOutDialogProps) {
  const { t } = useTranslation()
  const handleConfirm = async () => {
    try {
      await onSignOut?.()
    } finally {
      onOpenChange(false)
    }
  }

  return (
    <ConfirmDialog
      open={open}
      onOpenChange={onOpenChange}
      title={t('common.signOut')}
      desc={t('common.signOutDesc')}
      confirmText={t('common.signOut')}
      cancelBtnText={t('common.cancel')}
      destructive
      handleConfirm={handleConfirm}
      className='sm:max-w-sm'
    />
  )
}
