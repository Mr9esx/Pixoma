import { useState } from 'react'
import { Bold, Code, Italic, Link2, List, ListOrdered, SquareCode } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Textarea } from '@/components/ui/textarea'

type Props = {
  id?: string
  value: string
  onChange: (next: string) => void
  placeholder?: string
  disabled?: boolean
}

/** 轻量 Markdown 输入编辑框：固定 100px 高，带常用语法快捷插入。 */
export function MarkdownTextField({
  id,
  value,
  onChange,
  placeholder,
  disabled,
}: Props) {
  const { t } = useTranslation()
  const [sel, setSel] = useState<{ start: number; end: number }>({
    start: value.length,
    end: value.length,
  })

  function syncSel(ta: HTMLTextAreaElement) {
    setSel({ start: ta.selectionStart, end: ta.selectionEnd })
  }

  function wrap(before: string, after = before, fallback = '') {
    const start = sel.start
    const end = sel.end
    const picked = value.slice(start, end) || fallback
    onChange(
      value.slice(0, start) + before + picked + after + value.slice(end)
    )
    setSel({ start: start + before.length, end: start + before.length + picked.length })
  }

  function prepend(prefix: string) {
    const start = sel.start
    const lineStart = value.lastIndexOf('\n', start - 1) + 1
    const lineEnd = value.indexOf('\n', start)
    const selEnd = lineEnd === -1 ? value.length : lineEnd
    const line = value.slice(lineStart, selEnd)
    onChange(value.slice(0, lineStart) + prefix + line + value.slice(selEnd))
    setSel({ start: start + prefix.length, end: start + prefix.length })
  }

  const tools: Array<{ title: string; icon: typeof Bold; run: () => void }> = [
    { title: t('cases.mdBold'), icon: Bold, run: () => wrap('**', '**', '加粗') },
    { title: t('cases.mdItalic'), icon: Italic, run: () => wrap('*', '*', '斜体') },
    { title: t('cases.mdLink'), icon: Link2, run: () => wrap('[', '](链接)', '文字') },
    { title: t('cases.mdInlineCode'), icon: Code, run: () => wrap('`', '`', 'code') },
    { title: t('cases.mdCodeBlock'), icon: SquareCode, run: () => wrap('\n```\n', '\n```\n', 'code') },
    { title: t('cases.mdBullet'), icon: List, run: () => prepend('- ') },
    { title: t('cases.mdOrdered'), icon: ListOrdered, run: () => prepend('1. ') },
  ]

  return (
    <div
      data-testid='markdown-text-field'
      className='overflow-hidden rounded-md border border-input'
    >
      <div className='flex items-center gap-0.5 border-b bg-muted/40 px-1.5 py-1'>
        {tools.map((tool) => (
          <button
            key={tool.title}
            type='button'
            title={tool.title}
            aria-label={tool.title}
            disabled={disabled}
            onClick={tool.run}
            className='inline-flex size-7 items-center justify-center rounded-md text-muted-foreground hover:bg-accent hover:text-foreground disabled:opacity-50'
          >
            <tool.icon className='size-3.5' />
          </button>
        ))}
      </div>
      <Textarea
        id={id}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder={placeholder ?? t('cases.mdPlaceholder')}
        disabled={disabled}
        className='h-[100px] resize-y rounded-none border-0 px-3 py-2 focus-visible:border-0 focus-visible:ring-0'
        onSelect={(e) => syncSel(e.currentTarget)}
        onClick={(e) => syncSel(e.currentTarget)}
        onFocus={(e) => syncSel(e.currentTarget)}
        onKeyUp={(e) => syncSel(e.currentTarget)}
      />
    </div>
  )
}
