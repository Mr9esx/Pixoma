import {
  createContext,
  useContext,
  useState,
  type HTMLAttributes,
  type ReactNode,
} from 'react'
import { ChevronRight, FileText, Folder, FolderOpen } from 'lucide-react'
import { cn } from '@/lib/utils'
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from '@/components/ui/collapsible'

type FileTreeContextValue = {
  expanded: Set<string>
  toggle: (path: string) => void
  selectedPath?: string
  onSelect?: (path: string) => void
}

const FileTreeContext = createContext<FileTreeContextValue | null>(null)

function useFileTree() {
  const value = useContext(FileTreeContext)
  if (!value) throw new Error('FileTree components require FileTree')
  return value
}

type FileTreeProps = Omit<HTMLAttributes<HTMLDivElement>, 'onSelect'> & {
  expanded?: Set<string>
  defaultExpanded?: Set<string>
  selectedPath?: string
  onSelect?: (path: string) => void
  onExpandedChange?: (expanded: Set<string>) => void
}

export function FileTree({
  expanded: controlledExpanded,
  defaultExpanded,
  selectedPath,
  onSelect,
  onExpandedChange,
  className,
  children,
  ...props
}: FileTreeProps) {
  const [internalExpanded, setInternalExpanded] = useState(
    () => defaultExpanded ?? new Set<string>()
  )
  const expanded = controlledExpanded ?? internalExpanded
  const toggle = (path: string) => {
    const next = new Set(expanded)
    if (next.has(path)) next.delete(path)
    else next.add(path)
    setInternalExpanded(next)
    onExpandedChange?.(next)
  }
  return (
    <FileTreeContext.Provider
      value={{ expanded, toggle, selectedPath, onSelect }}
    >
      <div
        role='tree'
        className={cn('rounded-lg border bg-background p-2 text-sm', className)}
        {...props}
      >
        {children}
      </div>
    </FileTreeContext.Provider>
  )
}

type FileTreeFolderProps = HTMLAttributes<HTMLDivElement> & {
  path: string
  name: string
}

export function FileTreeFolder({
  path,
  name,
  className,
  children,
  ...props
}: FileTreeFolderProps) {
  const { expanded, toggle } = useFileTree()
  const open = expanded.has(path)
  return (
    <Collapsible open={open} onOpenChange={() => toggle(path)}>
      <div className={className} {...props}>
        <CollapsibleTrigger asChild>
          <button
            type='button'
            role='treeitem'
            aria-expanded={open}
            className='flex min-h-8 w-full items-center gap-1 rounded-md px-2 text-left text-xs transition-colors hover:bg-accent hover:text-accent-foreground'
          >
            <ChevronRight
              className={cn(
                'size-3.5 shrink-0 text-muted-foreground transition-transform',
                open && 'rotate-90'
              )}
            />
            {open ? (
              <FolderOpen className='size-3.5 shrink-0' />
            ) : (
              <Folder className='size-3.5 shrink-0' />
            )}
            <span className='truncate'>{name}</span>
          </button>
        </CollapsibleTrigger>
        <CollapsibleContent role='group' className='ml-3 border-l pl-2'>
          {children}
        </CollapsibleContent>
      </div>
    </Collapsible>
  )
}

type FileTreeFileProps = HTMLAttributes<HTMLButtonElement> & {
  path: string
  name: string
  icon?: ReactNode
}

export function FileTreeFile({
  path,
  name,
  icon,
  className,
  ...props
}: FileTreeFileProps) {
  const { selectedPath, onSelect } = useFileTree()
  return (
    <button
      type='button'
      role='treeitem'
      aria-selected={selectedPath === path}
      className={cn(
        'flex min-h-8 w-full items-center gap-2 rounded-md px-2 text-left text-xs transition-colors hover:bg-accent hover:text-accent-foreground',
        selectedPath === path && 'bg-accent text-accent-foreground',
        className
      )}
      onClick={() => onSelect?.(path)}
      {...props}
    >
      {icon ?? <FileText className='size-3.5 shrink-0' />}
      <span className='truncate'>{name}</span>
    </button>
  )
}
