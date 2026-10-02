import { useEffect, useImperativeHandle, useMemo, useRef, useState, type Ref } from 'react'
import {
  EditorContent,
  Extension,
  Node,
  NodeViewWrapper,
  ReactNodeViewRenderer,
  useEditor,
  type Editor,
  type ReactNodeViewProps,
} from '@tiptap/react'
import StarterKit from '@tiptap/starter-kit'
import Suggestion, { exitSuggestion, type SuggestionProps } from '@tiptap/suggestion'
import { NodeSelection, PluginKey, Selection, TextSelection } from '@tiptap/pm/state'
import { Boxes, Sparkles, Workflow } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { STUDIO_REFERENCE_BEFORE_CARET, STUDIO_REFERENCE_CARET, serializeStudioComposer, studioComposerContent, type StudioComposerValue } from './studio-composer-content'
import { StudioReferenceBadge } from './studio-reference-badge'

export type StudioReference = {
  kind: 'skill' | 'asset' | 'workflow'
  id: string
  label: string
  versionId?: string
}

type SuggestionKind = StudioReference['kind']
const suggestionTriggers = [
  { char: '/', kind: 'skill', key: new PluginKey('studioSkillMenu') },
  { char: '@', kind: 'asset', key: new PluginKey('studioAssetMenu') },
  { char: '!', kind: 'workflow', key: new PluginKey('studioWorkflowMenu') },
] as const
const suggestionLabels = { skill: '技能', asset: '资产', workflow: '工作流' }

export type StudioComposerHandle = {
  insertReference: (reference: StudioReference) => void
  serialize: () => StudioComposerValue
  clear: () => void
  setText: (text: string) => void
  setValue: (value: StudioComposerValue) => void
}

function previousGraphemeSize(text: string) {
  return new Intl.Segmenter(undefined, { granularity: 'grapheme' })
    .segment(text).containing(text.length - 1)!.segment.length
}

function insertStudioReference(editor: Editor, reference: StudioReference, range: { from: number; to: number }) {
  editor.chain().focus().insertContentAt(range, [
    { type: 'text', text: STUDIO_REFERENCE_BEFORE_CARET },
    { type: 'studioReference', attrs: reference },
    { type: 'text', text: STUDIO_REFERENCE_CARET },
  ]).run()
}

function StudioReferenceNodeView({ node }: ReactNodeViewProps) {
  const { kind, label } = node.attrs as StudioReference
  return (
    <NodeViewWrapper as='span' className='inline' contentEditable={false}>
      <StudioReferenceBadge kind={kind} label={label} />
    </NodeViewWrapper>
  )
}

const StudioReferenceNode = Node.create({
  name: 'studioReference',
  group: 'inline',
  inline: true,
  atom: true,
  selectable: true,
  addAttributes() {
    return {
      kind: { default: null },
      id: { default: null },
      label: { default: null },
      versionId: { default: null },
    }
  },
  parseHTML() {
    return [{
      tag: 'span[data-studio-reference]',
      getAttrs: (element) => {
        if (!(element instanceof HTMLElement)) return false
        return {
          kind: element.dataset.studioReference,
          id: element.dataset.id,
          label: element.dataset.label,
          versionId: element.dataset.versionId,
        }
      },
    }]
  },
  renderHTML({ node }) {
    return ['span', {
      'data-studio-reference': node.attrs.kind,
      'data-id': node.attrs.id,
      'data-label': node.attrs.label,
      'data-version-id': node.attrs.versionId,
    }, node.attrs.label]
  },
  addNodeView() {
    return ReactNodeViewRenderer(StudioReferenceNodeView, { className: 'select-none' })
  },
})

const textExtensions = [
  StarterKit.configure({
    blockquote: false,
    bold: false,
    bulletList: false,
    code: false,
    codeBlock: false,
    heading: false,
    horizontalRule: false,
    italic: false,
    link: false,
    listItem: false,
    orderedList: false,
    strike: false,
    underline: false,
  }),
  StudioReferenceNode,
]

export function StudioComposer({
  ref,
  placeholder,
  disabled = false,
  referenceItems = [],
  onSuggestionChange,
  suggestionLoading = false,
  suggestionError = false,
  onValueChange,
  onSubmit,
}: {
  ref?: Ref<StudioComposerHandle>
  placeholder: string
  disabled?: boolean
  referenceItems?: StudioReference[]
  onSuggestionChange?: (kind: SuggestionKind | null, query: string) => void
  suggestionLoading?: boolean
  suggestionError?: boolean
  onValueChange?: (value: StudioComposerValue) => void
  onSubmit?: () => void
}) {
  const [value, setValue] = useState<StudioComposerValue>({
    text: '', parts: [], selectedSkillIds: [], selectedAssets: [],
  })
  const submitRef = useRef(onSubmit)
  const changeRef = useRef(onValueChange)
  const referenceItemsRef = useRef(referenceItems)
  const suggestionChangeRef = useRef(onSuggestionChange)
  const [suggestionMenu, setSuggestionMenu] = useState<{ kind: SuggestionKind; props: SuggestionProps<StudioReference, StudioReference> } | null>(null)
  const suggestionMenuRef = useRef(suggestionMenu)
  const [activeItem, setActiveItem] = useState(0)
  const activeItemRef = useRef(activeItem)
  const visibleItems = useMemo(() => suggestionMenu
    ? referenceItems.filter((item) => item.kind === suggestionMenu.kind && item.label.toLocaleLowerCase().includes(suggestionMenu.props.query.toLocaleLowerCase()))
    : [], [referenceItems, suggestionMenu])
  const visibleItemsRef = useRef(visibleItems)
  useEffect(() => {
    submitRef.current = onSubmit
    changeRef.current = onValueChange
    referenceItemsRef.current = referenceItems
    suggestionChangeRef.current = onSuggestionChange
  }, [onSubmit, onValueChange, referenceItems, onSuggestionChange])
  useEffect(() => { visibleItemsRef.current = visibleItems }, [visibleItems])
  // eslint-disable-next-line react-hooks/refs
  const [suggestionExtension] = useState(() => Extension.create({
    name: 'studioReferenceMenu',
    addProseMirrorPlugins() {
      return suggestionTriggers.map(({ char, kind, key }) => Suggestion<StudioReference, StudioReference>({
        pluginKey: key,
        editor: this.editor,
        char,
        shouldResetDismissed: ({ transaction }) => transaction.docChanged,
        shouldShow: ({ query }) => kind !== 'workflow' || !query.startsWith('['),
        items: ({ query }) => referenceItemsRef.current.filter((item) =>
          item.kind === kind && item.label.toLocaleLowerCase().includes(query.toLocaleLowerCase())
        ),
        command: ({ editor: current, range, props: reference }) => {
          insertStudioReference(current, reference, range)
        },
        render: () => ({
          onStart: (props) => {
            activeItemRef.current = 0
            suggestionMenuRef.current = { kind, props }
            setActiveItem(0)
            setSuggestionMenu({ kind, props })
            suggestionChangeRef.current?.(kind, props.query)
          },
          onUpdate: (props) => {
            activeItemRef.current = 0
            suggestionMenuRef.current = { kind, props }
            setActiveItem(0)
            setSuggestionMenu({ kind, props })
            suggestionChangeRef.current?.(kind, props.query)
          },
          onExit: () => {
            if (suggestionMenuRef.current?.kind !== kind) return
            suggestionMenuRef.current = null
            setSuggestionMenu(null)
            suggestionChangeRef.current?.(null, '')
          },
        }),
      }))
    },
  }))
  const editor = useEditor({
    extensions: [...textExtensions, suggestionExtension],
    editable: !disabled,
    content: '',
    editorProps: {
      attributes: { role: 'textbox', 'aria-label': '输入消息', class: 'min-h-12 w-full outline-none' },
      clipboardTextSerializer: (slice) => slice.content.textBetween(
        0,
        slice.content.size,
        '\n\n',
        (node) => node.type.name === 'studioReference' ? node.attrs.label : ''
      ).replaceAll(STUDIO_REFERENCE_CARET, '').replaceAll(STUDIO_REFERENCE_BEFORE_CARET, ''),
      handleTextInput: (view, _from, _to, text) => {
        const selection = view.state.selection
        if (!(selection instanceof NodeSelection) || selection.node.type.name !== 'studioReference') return false
        const before = view.state.doc.resolve(selection.from).nodeBefore
        const beforeSize = before?.text?.endsWith(STUDIO_REFERENCE_BEFORE_CARET) ? 1 : 0
        const after = view.state.doc.nodeAt(selection.to)
        const spacerSize = after?.text?.startsWith(STUDIO_REFERENCE_CARET) ? 1 : 0
        const start = selection.from - beforeSize
        const transaction = view.state.tr.insertText(text, start, selection.to + spacerSize)
        view.dispatch(transaction.setSelection(TextSelection.create(transaction.doc, start + text.length)))
        return true
      },
      handleKeyDown: (view, event) => {
        const selection = view.state.selection
        if (selection instanceof NodeSelection && selection.node.type.name === 'studioReference') {
          const before = view.state.doc.resolve(selection.from).nodeBefore
          const beforeSize = before?.text?.endsWith(STUDIO_REFERENCE_BEFORE_CARET) ? 1 : 0
          const after = view.state.doc.nodeAt(selection.to)
          const spacerSize = after?.text?.startsWith(STUDIO_REFERENCE_CARET) ? 1 : 0
          if (event.key === 'Backspace' || event.key === 'Delete') {
            view.dispatch(view.state.tr.delete(selection.from - beforeSize, selection.to + spacerSize))
            return true
          }
          if (event.key === 'ArrowRight') {
            view.dispatch(view.state.tr.setSelection(TextSelection.create(view.state.doc, selection.to + spacerSize)))
            return true
          }
        }
        if (selection.empty) {
          const { $from } = view.state.selection
          if ($from.nodeBefore?.text?.endsWith(STUDIO_REFERENCE_CARET)
            && $from.parent.childBefore($from.parentOffset - 1).node?.type.name === 'studioReference') {
            if (event.key === 'ArrowLeft') {
              view.dispatch(view.state.tr.setSelection(TextSelection.create(view.state.doc, $from.pos - 2)))
              return true
            }
            if (event.key === 'Backspace') {
              const before = view.state.doc.resolve($from.pos - 2).nodeBefore
              const beforeSize = before?.text?.endsWith(STUDIO_REFERENCE_BEFORE_CARET) ? 1 : 0
              view.dispatch(view.state.tr.delete($from.pos - 2 - beforeSize, $from.pos))
              return true
            }
          }
          if ($from.nodeAfter?.type.name === 'studioReference') {
            const beforeSize = $from.nodeBefore?.text?.endsWith(STUDIO_REFERENCE_BEFORE_CARET) ? 1 : 0
            const after = view.state.doc.nodeAt($from.pos + 1)
            const spacerSize = after?.text?.startsWith(STUDIO_REFERENCE_CARET) ? 1 : 0
            if (event.key === 'ArrowLeft' && beforeSize) {
              const text = $from.nodeBefore?.text?.slice(0, -1) ?? ''
              if (!text) {
                view.dispatch(view.state.tr.setSelection(Selection.near(view.state.doc.resolve($from.pos - 2), -1)))
                return true
              }
              view.dispatch(view.state.tr.setSelection(TextSelection.create(view.state.doc, $from.pos - 1 - previousGraphemeSize(text))))
              return true
            }
            if (event.key === 'ArrowRight') {
              view.dispatch(view.state.tr.setSelection(TextSelection.create(view.state.doc, $from.pos + 1 + spacerSize)))
              return true
            }
            if (event.key === 'Delete') {
              view.dispatch(view.state.tr.delete($from.pos - beforeSize, $from.pos + 1 + spacerSize))
              return true
            }
          }
          if (event.key === 'ArrowRight' && $from.nodeAfter?.text?.startsWith(STUDIO_REFERENCE_BEFORE_CARET)
            && view.state.doc.nodeAt($from.pos + 1)?.type.name === 'studioReference') {
            const after = view.state.doc.nodeAt($from.pos + 2)
            const spacerSize = after?.text?.startsWith(STUDIO_REFERENCE_CARET) ? 1 : 0
            view.dispatch(view.state.tr.setSelection(TextSelection.create(view.state.doc, $from.pos + 2 + spacerSize)))
            return true
          }
          if (event.key === 'ArrowLeft' && $from.nodeBefore?.text
            && $from.nodeAfter?.text?.endsWith(STUDIO_REFERENCE_BEFORE_CARET)
            && view.state.doc.nodeAt($from.pos + $from.nodeAfter.nodeSize)?.type.name === 'studioReference') {
            view.dispatch(view.state.tr.setSelection(TextSelection.create(view.state.doc, $from.pos - previousGraphemeSize($from.nodeBefore.text))))
            return true
          }
        }
        const menu = suggestionMenuRef.current
        if (menu && event.key === 'Escape') {
          exitSuggestion(view, suggestionTriggers.find((trigger) => trigger.kind === menu.kind)!.key)
          suggestionMenuRef.current = null
          setSuggestionMenu(null)
          suggestionChangeRef.current?.(null, '')
          return true
        }
        const items = visibleItemsRef.current
        if (menu && items.length && (event.key === 'ArrowDown' || event.key === 'ArrowUp')) {
          const change = event.key === 'ArrowDown' ? 1 : -1
          const next = (activeItemRef.current + change + items.length) % items.length
          activeItemRef.current = next
          setActiveItem(next)
          return true
        }
        if (menu && items.length && event.key === 'Enter' && !event.isComposing) {
          menu.props.command(items[activeItemRef.current])
          return true
        }
        if (event.key === 'Enter' && !event.shiftKey && !event.isComposing) {
          event.preventDefault()
          submitRef.current?.()
          return true
        }
        return false
      },
    },
    onUpdate: ({ editor: current }) => {
      const next = serializeStudioComposer(current.getJSON())
      setValue(next)
      changeRef.current?.(next)
    },
  })
  useEffect(() => {
    if (!editor) return
    const document = editor.view.dom.ownerDocument
    const moveCaretOutOfReference = () => {
      const selection = document.getSelection()
      if (!selection?.isCollapsed || !selection.anchorNode) return
      const element = selection.anchorNode instanceof Element
        ? selection.anchorNode
        : selection.anchorNode.parentElement
      const nodeView = element?.closest('.node-studioReference')
      if (!nodeView || !editor.view.dom.contains(nodeView)) return
      editor.commands.setTextSelection(editor.view.posAtDOM(nodeView, nodeView.childNodes.length))
    }
    document.addEventListener('selectionchange', moveCaretOutOfReference)
    return () => document.removeEventListener('selectionchange', moveCaretOutOfReference)
  }, [editor])
  useEffect(() => {
    editor?.setEditable(!disabled)
  }, [editor, disabled])
  useImperativeHandle(ref, () => ({
    insertReference: (reference) => {
      if (editor) insertStudioReference(editor, reference, editor.state.selection)
    },
    serialize: () => editor ? serializeStudioComposer(editor.getJSON()) : value,
    clear: () => editor?.commands.clearContent(),
    setText: (text) => editor?.commands.setContent(text),
    setValue: (next) => {
      editor?.commands.setContent(studioComposerContent(next.parts))
      editor?.commands.focus('end')
    },
  }), [editor, value])

  return (
    <div className='relative w-full min-w-48 flex-1'>
      {!value.text ? (
        <span className='pointer-events-none absolute left-3 top-2 text-sm leading-6 text-muted-foreground'>
          {placeholder}
        </span>
      ) : null}
      <EditorContent
        editor={editor}
        className='max-h-48 min-h-16 w-full overflow-y-auto px-3 py-2 text-sm leading-6'
      />
      {suggestionMenu ? (
        <div className='absolute bottom-full left-0 z-50 mb-2 max-h-56 min-w-52 overflow-y-auto rounded-md border bg-popover p-1 shadow-md'>
          {visibleItems.map((item, index) => (
            <Button
              key={`${item.kind}:${item.id}`}
              type='button'
              variant='ghost'
              className={`flex w-full justify-start gap-2 ${index === activeItem ? 'bg-accent' : ''}`}
              aria-label={`插入${suggestionLabels[item.kind]}：${item.label}`}
              onMouseDown={(event) => event.preventDefault()}
              onClick={() => suggestionMenu.props.command(item)}
            >
              {item.kind === 'skill' ? <Sparkles /> : item.kind === 'workflow' ? <Workflow /> : <Boxes />}
              {item.label}
            </Button>
          ))}
          {visibleItems.length === 0 ? <p className='px-2 py-1.5 text-sm text-muted-foreground'>
            {suggestionLoading ? `正在读取${suggestionLabels[suggestionMenu.kind]}…` : suggestionError ? `${suggestionLabels[suggestionMenu.kind]}读取失败` : `没有匹配的${suggestionLabels[suggestionMenu.kind]}`}
          </p> : null}
        </div>
      ) : null}
      <input name='message' type='hidden' value={value.text} readOnly />
    </div>
  )
}
