import { useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import {
  createStudioProject,
  renameStudioProject,
  type StudioProject,
} from '@/lib/api/studio'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

export function StudioProjectDialog({
  open,
  project,
  onOpenChange,
  onSaved,
}: {
  open: boolean
  project?: StudioProject
  onOpenChange: (open: boolean) => void
  onSaved?: (project: StudioProject) => void
}) {
  const queryClient = useQueryClient()
  const [name, setName] = useState(project?.name ?? '')
  const save = useMutation({
    mutationFn: () =>
      project
        ? renameStudioProject(project.id, name.trim())
        : createStudioProject(name.trim()),
    onSuccess: async (saved) => {
      await queryClient.invalidateQueries({ queryKey: ['studio', 'projects'] })
      onOpenChange(false)
      onSaved?.(saved)
    },
  })

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <form
          onSubmit={(event) => {
            event.preventDefault()
            if (name.trim()) save.mutate()
          }}
        >
          <DialogHeader>
            <DialogTitle>{project ? '重命名项目' : '新建项目'}</DialogTitle>
            <DialogDescription className='sr-only'>项目名称</DialogDescription>
          </DialogHeader>
          <div className='mt-5 space-y-2'>
            <Label htmlFor='studio-project-name'>项目名称</Label>
            <Input
              id='studio-project-name'
              autoFocus
              maxLength={80}
              required
              value={name}
              onChange={(event) => setName(event.target.value)}
            />
          </div>
          {save.isError ? (
            <p role='alert' className='mt-2 text-sm text-destructive'>
              {save.error.message}
            </p>
          ) : null}
          <DialogFooter className='mt-5'>
            <Button
              type='button'
              variant='outline'
              onClick={() => onOpenChange(false)}
            >
              取消
            </Button>
            <Button type='submit' disabled={!name.trim() || save.isPending}>
              {save.isPending ? '保存中…' : '确认'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
