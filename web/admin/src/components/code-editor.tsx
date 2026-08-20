import type { ReactNode } from 'react'
import { json } from '@codemirror/lang-json'
import { HighlightStyle, syntaxHighlighting } from '@codemirror/language'
import { EditorView } from '@codemirror/view'
import { tags } from '@lezer/highlight'
import CodeMirror, { type ReactCodeMirrorProps } from '@uiw/react-codemirror'
import { cn } from '@/lib/utils'

const jsonHighlight = HighlightStyle.define([
  { tag: tags.propertyName, color: '#7ee787' },
  { tag: tags.string, color: '#a5d6ff' },
  { tag: tags.number, color: '#79c0ff' },
  { tag: [tags.bool, tags.null], color: '#d2a8ff' },
  { tag: tags.punctuation, color: '#8b949e' },
  { tag: tags.invalid, color: '#f85149' },
])

const ideTheme = EditorView.theme({
  '&': {
    backgroundColor: '#0d1117',
    color: '#e6edf3',
    fontSize: '12px',
  },
  '&.cm-focused': {
    outline: 'none',
  },
  '.cm-content': {
    fontFamily:
      'ui-monospace, SFMono-Regular, Menlo, Consolas, "Liberation Mono", monospace',
    padding: '10px 12px',
    caretColor: '#e6edf3',
  },
  '.cm-scroller': {
    fontFamily: 'inherit',
    overflow: 'auto',
  },
  '.cm-placeholder': {
    color: '#8b949e',
  },
  '.cm-gutters': {
    backgroundColor: '#0d1117',
    borderRight: '1px solid #30363d',
    color: '#8b949e',
  },
  '.cm-selectionBackground, &.cm-focused .cm-selectionBackground': {
    backgroundColor: 'rgba(56, 139, 253, 0.28)',
  },
})

const extensions = [json(), syntaxHighlighting(jsonHighlight), ideTheme]

type Props = Omit<
  ReactCodeMirrorProps,
  'onChange' | 'extensions' | 'maxHeight'
> & {
  title?: ReactNode
  action?: ReactNode
  maxHeight?: number | string
  onChange?: (value: string) => void
}

export function CodeEditor({
  title,
  action,
  className,
  maxHeight = 250,
  onChange,
  ...props
}: Props) {
  return (
    <div
      className={cn(
        'overflow-hidden rounded-md border border-[#30363d] bg-[#0d1117]',
        className
      )}
    >
      {title != null ? (
        <header className='flex h-9 items-center justify-between gap-2 border-b border-[#30363d] bg-[#161b22] px-3'>
          <h4 className='text-xs font-medium tracking-wide text-[lab(75.0771%_-60.7313_19.4147)]'>
            {title}
          </h4>
          {action}
        </header>
      ) : null}
      <CodeMirror
        extensions={extensions}
        theme='none'
        maxHeight={typeof maxHeight === 'number' ? `${maxHeight}px` : maxHeight}
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
    </div>
  )
}
