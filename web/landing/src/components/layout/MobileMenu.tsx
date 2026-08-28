import { AnimatePresence, motion } from 'motion/react'
import { useTranslation } from 'react-i18next'
import { LangSwitch } from './LangSwitch'

type MobileMenuProps = {
  open: boolean
  onClose: () => void
}

export function MobileMenu({ open, onClose }: MobileMenuProps) {
  const { t } = useTranslation()

  return (
    <AnimatePresence>
      {open && (
        <motion.div
          initial={{ opacity: 0, height: 0 }}
          animate={{ opacity: 1, height: 'auto' }}
          exit={{ opacity: 0, height: 0 }}
          transition={{ duration: 0.2 }}
          className="overflow-hidden border-t border-border bg-background md:hidden"
        >
          <nav className="flex flex-col gap-4 px-4 py-5 text-sm text-muted-foreground">
            <a href="#features" onClick={onClose} className="hover:text-foreground">
              {t('nav.features')}
            </a>
            <a href="#scenarios" onClick={onClose} className="hover:text-foreground">
              {t('nav.scenarios')}
            </a>
            <a href="#download" onClick={onClose} className="hover:text-foreground">
              {t('nav.download')}
            </a>
            <div className="pt-2">
              <LangSwitch />
            </div>
          </nav>
        </motion.div>
      )}
    </AnimatePresence>
  )
}
