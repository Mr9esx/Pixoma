export function retainStudioComposerFocus(event: Event) {
  if (document.activeElement?.matches('.tiptap[role="textbox"][aria-label="输入消息"]')) {
    event.preventDefault()
  }
}
