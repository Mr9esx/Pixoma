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
      title='退出登录'
      desc='退出后回到登录页，重新登录才能继续使用后台。'
      confirmText='退出登录'
      cancelBtnText='取消'
      destructive
      handleConfirm={handleConfirm}
      className='sm:max-w-sm'
    />
  )
}
