import { toast } from 'sonner'

export function showSubmittedData(
  data: unknown,
  title: string = 'You submitted the following values:'
) {
  toast.message(title, {
    description: (
      <pre className='mt-2 w-full overflow-x-auto rounded-md bg-muted p-4'>
        <code className='text-foreground'>{JSON.stringify(data, null, 2)}</code>
      </pre>
    ),
  })
}
