import { useRef } from 'react'
import { useTranslation } from 'react-i18next'
import { Logo } from '@/assets/logo'
import { useTheme } from '@/context/theme-provider'
import Dither from '@/components/Dither'
import FaultyTerminal from '@/components/FaultyTerminal'
import { DotPatternBackground } from '@/components/feedback/dot-pattern-background'

export type AuthAmbient = 'terminal' | 'dither'

type AuthShellProps = {
  children: React.ReactNode
  ambient?: AuthAmbient
  logoVariant?: 'classic' | 'brand'
}

const DOT_PATTERN_GRADIENT = {
  backgroundImage:
    'radial-gradient(circle, color-mix(in oklch, var(--muted-foreground) 25%, transparent) 1px, transparent 1px)',
  backgroundSize: '16px 16px',
}

const DOT_MASK_GRADIENT = {
  ...DOT_PATTERN_GRADIENT,
  backgroundImage:
    'radial-gradient(circle, color-mix(in oklch, var(--foreground) 100%, transparent) 1px, transparent 1px)',
  maskImage:
    'radial-gradient(80px circle at 0px 0px, black 0%, transparent 100%)',
  WebkitMaskImage:
    'radial-gradient(80px circle at 0px 0px, black 0%, transparent 100%)',
}

const TERMINAL_GRID_MUL: [number, number] = [2, 1]

export function AuthShell({
  children,
  ambient = 'terminal',
  logoVariant = 'classic',
}: AuthShellProps) {
  const { t } = useTranslation()
  const { theme } = useTheme()
  const spotRef = useRef<HTMLDivElement>(null)
  const brandMobileLogoSrc =
    theme === 'dark'
      ? '/images/login-logo-dark.svg'
      : '/images/login-logo-light.svg'

  function handleMouseMove(e: React.MouseEvent<HTMLDivElement>) {
    const rect = e.currentTarget.getBoundingClientRect()
    const x = e.clientX - rect.left
    const y = e.clientY - rect.top
    if (spotRef.current) {
      const gradient = `radial-gradient(80px circle at ${x}px ${y}px, black 0%, transparent 100%)`
      spotRef.current.style.maskImage = gradient
      spotRef.current.style.webkitMaskImage = gradient
    }
  }

  return (
    <div
      className='relative container grid h-svh flex-col items-center justify-center bg-background lg:max-w-none lg:grid-cols-2 lg:px-0'
      onMouseMove={handleMouseMove}
    >
      <DotPatternBackground />
      <div
        aria-hidden
        ref={spotRef}
        className='pointer-events-none absolute inset-0'
        style={DOT_MASK_GRADIENT}
      />
      <div className='relative hidden h-full flex-col p-3 lg:flex'>
        <div className='relative flex min-h-0 flex-1 flex-col overflow-hidden rounded-xl'>
          {ambient === 'dither' ? (
            <div aria-hidden className='absolute inset-0'>
              <Dither
                waveColor={[0.5, 0.5, 0.5]}
                colorNum={4}
                pixelSize={2}
                waveAmplitude={0.3}
                waveFrequency={3}
                waveSpeed={0.05}
                disableAnimation={false}
                enableMouseInteraction
                mouseRadius={0.3}
              />
            </div>
          ) : (
            <FaultyTerminal
              style={{
                position: 'absolute',
                inset: 0,
                pointerEvents: 'none',
              }}
              scale={1.5}
              gridMul={TERMINAL_GRID_MUL}
              digitSize={1.2}
              timeScale={0.5}
              pause={false}
              scanlineIntensity={0.5}
              glitchAmount={1}
              flickerAmount={1}
              noiseAmp={1}
              chromaticAberration={0}
              dither={0}
              curvature={0.1}
              tint='#A7EF9E'
              mouseReact
              mouseStrength={0.5}
              pageLoadAnimation
              brightness={0.6}
            />
          )}
          <div className='relative z-20 flex min-h-0 flex-1 flex-col p-10 text-white'>
            {logoVariant === 'brand' ? (
              <div className='relative z-20 flex items-center'>
                <img
                  alt='Pixoma'
                  className='h-7 w-auto'
                  src='/images/login-logo-dark.svg'
                />
              </div>
            ) : (
              <div className='relative z-20 flex items-center gap-2 text-lg font-medium'>
                <Logo className='me-1 size-6 rounded-full bg-white object-contain' />
                Pixoma
              </div>
            )}
            <blockquote className='relative z-20 mt-auto flex flex-col gap-2'>
              <p className='text-lg'>{t('auth.quote')}</p>
            </blockquote>
          </div>
        </div>
      </div>
      <div className='relative h-full overflow-y-auto lg:p-8'>
        <div className='relative mx-auto flex min-h-full w-full flex-col items-center justify-center gap-4 p-6'>
          {logoVariant === 'brand' ? (
            <div className='flex items-center justify-center lg:hidden'>
              <img
                alt='Pixoma'
                className='h-6 w-auto'
                src={brandMobileLogoSrc}
              />
            </div>
          ) : (
            <div className='flex items-center justify-center lg:hidden'>
              <Logo className='me-2 rounded-md' />
              <h1 className='text-xl font-medium'>Pixoma</h1>
            </div>
          )}
          {children}
        </div>
      </div>
    </div>
  )
}
