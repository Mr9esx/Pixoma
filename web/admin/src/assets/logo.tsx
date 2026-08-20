import { type ImgHTMLAttributes } from 'react'
import { cn } from '@/lib/utils'

export function Logo({
  className,
  alt = 'Pixoma',
  ...props
}: ImgHTMLAttributes<HTMLImageElement>) {
  return (
    <img
      id='pixoma-admin-logo'
      src='/images/logo.png'
      alt={alt}
      className={cn('size-6 shrink-0 object-cover', className)}
      {...props}
    />
  )
}
