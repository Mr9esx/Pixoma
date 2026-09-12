import * as React from 'react'
import { Eye, EyeOff } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { cn } from '@/lib/utils'
import {
  InputGroup,
  InputGroupAddon,
  InputGroupButton,
  InputGroupInput,
} from './ui/input-group'

type SecretInputProps = Omit<
  React.InputHTMLAttributes<HTMLInputElement>,
  'type'
> & {
  ref?: React.Ref<HTMLInputElement>
  endAddon?: React.ReactNode
}

export function SecretInput({
  className,
  disabled,
  ref,
  endAddon,
  ...props
}: SecretInputProps) {
  const { t } = useTranslation()
  const [showSecret, setShowSecret] = React.useState(false)

  return (
    <InputGroup className={cn(className)}>
      <InputGroupInput
        type={showSecret ? 'text' : 'password'}
        ref={ref}
        disabled={disabled}
        {...props}
      />
      <InputGroupAddon align='inline-end'>
        <InputGroupButton
          type='button'
          variant='ghost'
          size='icon-xs'
          disabled={disabled}
          onClick={() => setShowSecret((prev) => !prev)}
        >
          {showSecret ? <Eye /> : <EyeOff />}
          <span className='sr-only'>
            {showSecret ? t('common.hideSecret') : t('common.showSecret')}
          </span>
        </InputGroupButton>
        {endAddon}
      </InputGroupAddon>
    </InputGroup>
  )
}
