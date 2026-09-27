import type { ReactNode } from 'react'
import { json } from '@codemirror/lang-json'
import { markdown } from '@codemirror/lang-markdown'
import { HighlightStyle, syntaxHighlighting } from '@codemirror/language'
import { EditorView } from '@codemirror/view'
import { tags } from '@lezer/highlight'
import CodeMirror, { type ReactCodeMirrorProps } from '@uiw/react-codemirror'
import { cn } from '@/lib/utils'
import {
  Card,
  CardAction,
  CardContent,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'

const jsonHighlight = HighlightStyle.define([
  { tag: tags.propertyName, color: 'var(--muted-foreground)' },
  { tag: tags.string, color: 'var(--chart-2)' },
  { tag: tags.number, color: 'var(--chart-5)' },
  { tag: [tags.bool, tags.null], color: 'var(--chart-3)' },
  { tag: tags.punctuation, color: 'var(--muted-foreground)' },
  { tag: tags.invalid, color: 'var(--destructive)' },
])

const ideTheme = EditorView.theme({
  '&': {
    backgroundColor: 'var(--card)',
    color: 'var(--card-foreground)',
    fontSize: '12px',
  },
  '&.cm-focused': {
    outline: 'none',
  },
  '.cm-content': {
    fontFamily:
      'ui-monospace, SFMono-Regular, Menlo, Consolas, "Liberation Mono", monospace',
    padding: '10px 12px',
    caretColor: 'var(--foreground)',
  },
  '.cm-scroller': {
    fontFamily: 'inherit',
    overflow: 'auto',
  },
  '.cm-placeholder': {
    color: 'var(--muted-foreground)',
  },
  '.cm-gutters': {
    backgroundColor: 'var(--card)',
    borderRight: '1px solid var(--color-border)',
    color: 'var(--muted-foreground)',
  },
  '.cm-selectionBackground, &.cm-focused .cm-selectionBackground': {
    backgroundColor: 'color-mix(in oklab, var(--primary) 28%, transparent)',
  },
})

const fillTheme = EditorView.theme({
  '&': { backgroundColor: 'var(--background)' },
  '.cm-gutters': {
    backgroundColor: 'var(--background)',
    borderRight: 'none',
  },
})

const jsonExtensions = [json(), syntaxHighlighting(jsonHighlight), ideTheme]
const markdownExtensions = [markdown(), ideTheme]

type Props = Omit<
  ReactCodeMirrorProps,
  'onChange' | 'maxHeight' | 'minHeight'
> & {
  title?: ReactNode
  action?: ReactNode
  maxHeight?: number | string
  minHeight?: number | string
  language?: 'json' | 'markdown' | 'text'
  fill?: boolean
  onChange?: (value: string) => void
}

export function CodeEditor({
  title,
  action,
  className,
  maxHeight = 250,
  minHeight,
  language = 'json',
  fill = false,
  extensions = [],
  onChange,
  ...props
}: Props) {
  return (
    <Card
      className={cn(
        'overflow-hidden rounded-md py-0 text-card-foreground',
        fill && 'h-full min-h-0 flex-1 gap-0 bg-background text-foreground',
        className
      )}
    >
      {title != null ? (
        <CardHeader className='flex h-9 flex-row items-center justify-between gap-2 border-b border-border bg-muted/30 px-3 py-0 pb-0'>
          <CardTitle className='text-xs font-medium tracking-wide text-muted-foreground'>
            {title}
          </CardTitle>
          {action ? <CardAction className='gap-2'>{action}</CardAction> : null}
        </CardHeader>
      ) : null}
      <CardContent
        className={cn(
          'p-0',
          fill && 'flex min-h-0 flex-1 flex-col overflow-hidden'
        )}
      >
        <CodeMirror
          className={
            fill
              ? 'h-full min-h-0 flex-1 overflow-hidden [&_.cm-editor]:min-h-0 [&_.cm-scroller]:min-h-0'
              : undefined
          }
          extensions={[
            ...(language === 'json'
              ? jsonExtensions
              : language === 'markdown'
                ? markdownExtensions
                : [ideTheme]),
            ...(fill ? [fillTheme] : []),
            ...extensions,
          ]}
          theme='none'
          height={fill ? '100%' : undefined}
          maxHeight={
            fill
              ? undefined
              : typeof maxHeight === 'number'
                ? `${maxHeight}px`
                : maxHeight
          }
          minHeight={
            fill
              ? undefined
              : typeof minHeight === 'number'
                ? `${minHeight}px`
                : minHeight
          }
          basicSetup={{
            lineNumbers: false,
            foldGutter: false,
            highlightActiveLine: false,
            highlightActiveLineGutter: false,
            bracketMatching: true,
            closeBrackets: true,
            autocompletion: false,
          }}
          onChange={onChange ? (value) => onChange(value) : undefined}
          {...props}
        />
      </CardContent>
    </Card>
  )
}
