import { useRef } from 'react'
import { Logo } from '@/assets/logo'
import FaultyTerminal from '@/components/FaultyTerminal'

const DOT_PATTERN = {
  backgroundImage:
    'radial-gradient(circle, color-mix(in oklch, var(--muted-foreground) 25%, transparent) 1px, transparent 1px)',
  backgroundSize: '16px 16px',
}

const DOT_MASK_GRADIENT = {
  ...DOT_PATTERN,
  backgroundImage:
    'radial-gradient(circle, color-mix(in oklch, var(--foreground) 100%, transparent) 1px, transparent 1px)',
  maskImage:
    'radial-gradient(80px circle at 0px 0px, black 0%, transparent 100%)',
  WebkitMaskImage:
    'radial-gradient(80px circle at 0px 0px, black 0%, transparent 100%)',
}

const TERMINAL_GRID_MUL: [number, number] = [2, 1]

export function AuthShell({ children }: { children: React.ReactNode }) {
  const spotRef = useRef<HTMLDivElement>(null)

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
      <div
        aria-hidden
        className='pointer-events-none absolute inset-0'
        style={DOT_PATTERN}
      />
      <div
        aria-hidden
        ref={spotRef}
        className='pointer-events-none absolute inset-0'
        style={DOT_MASK_GRADIENT}
      />
      <div className='relative hidden h-full flex-col p-3 lg:flex'>
        <div className='relative flex min-h-0 flex-1 flex-col overflow-hidden rounded-xl'>
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
          <div className='relative z-20 flex min-h-0 flex-1 flex-col p-10 text-white'>
            <div className='relative z-20 flex items-center gap-2 text-lg font-medium'>
              <Logo className='me-1 size-6 rounded-full bg-white object-contain' />
              Pixoma
            </div>
            <div className='relative z-20 m-auto flex w-full max-w-xl flex-col justify-center gap-10'>
              <div className='flex flex-col gap-6'>
                <h2 className='text-4xl leading-tight font-semibold tracking-tight md:text-5xl'>
                  开始使用 Pixoma
                </h2>
                <p className='text-lg leading-7 text-white/80'>
                  几步配置完，登录后就能接上自己的 ComfyUI。
                </p>
              </div>
              <div className='flex gap-2 overflow-x-auto pb-1'>
                {[
                  { n: '1', label: '设管理员密码' },
                  { n: '2', label: '配置数据库与存储' },
                  { n: '3', label: '连接 ComfyUI' },
                ].map((step, index) => (
                  <div
                    key={step.n}
                    className={
                      index === 0
                        ? 'flex min-w-[150px] flex-col justify-center gap-4 rounded-xl bg-white p-5'
                        : 'flex min-w-[150px] flex-col justify-center gap-4 rounded-xl bg-white/12 p-5 backdrop-blur-md'
                    }
                  >
                    <div
                      className={
                        index === 0
                          ? 'flex size-6 items-center justify-center rounded-full bg-black text-sm font-medium text-white'
                          : 'flex size-6 items-center justify-center rounded-full bg-white/16 text-sm font-medium text-white'
                      }
                    >
                      {step.n}
                    </div>
                    <p
                      className={
                        index === 0
                          ? 'text-base font-medium text-black'
                          : 'text-base text-white/80'
                      }
                    >
                      {step.label}
                    </p>
                  </div>
                ))}
              </div>
            </div>
            <blockquote className='relative z-20 mt-auto flex flex-col gap-2'>
              <p className='text-lg'>
                Pixoma 让你随时随地使用自己的 ComfyUI 进行艺术创作。
              </p>
            </blockquote>
          </div>
        </div>
      </div>
      <div className='relative h-full overflow-y-auto lg:p-8'>
        <div className='relative mx-auto flex min-h-full w-full flex-col items-center justify-center gap-4 p-6'>
          <div className='flex items-center justify-center lg:hidden'>
            <Logo className='me-2 rounded-md' />
            <h1 className='text-xl font-medium'>Pixoma</h1>
          </div>
          {children}
        </div>
      </div>
    </div>
  )
}
