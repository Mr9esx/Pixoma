import { Logo } from '@/assets/logo'

export function AuthShell({ children }: { children: React.ReactNode }) {
  return (
    <div className='relative container grid h-svh flex-col items-center justify-center lg:max-w-none lg:grid-cols-2 lg:px-0'>
      <div className='relative hidden h-full flex-col bg-muted p-10 text-background lg:flex dark:border-e'>
        <div className='absolute inset-0 bg-foreground' />
        <div className='relative z-20 flex items-center text-lg font-medium'>
          <Logo className='me-2' />
          Pixoma
        </div>
        <Logo className='relative m-auto size-96' />
        <blockquote className='relative z-20 mt-auto flex flex-col gap-2'>
          <p className='text-lg'>
            「先登录后台。还没配过的话，接着改密码、选库、选这台机器还是远程
            GPU。」
          </p>
          <footer className='text-sm'>Pixoma</footer>
        </blockquote>
      </div>
      <div className='h-full overflow-y-auto lg:p-8'>
        <div className='relative mx-auto flex min-h-full w-full flex-col items-center justify-center gap-4 p-6'>
          <div className='flex items-center justify-center lg:hidden'>
            <Logo className='me-2' />
            <h1 className='text-xl font-medium'>Pixoma</h1>
          </div>
          {children}
        </div>
      </div>
    </div>
  )
}
