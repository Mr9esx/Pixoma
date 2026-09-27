import type { ReactNode } from 'react'
import {
  Background,
  ReactFlow,
  type Edge,
  type Node,
  type ReactFlowProps,
} from '@xyflow/react'
import '@xyflow/react/dist/style.css'

type CanvasProps<NodeType extends Node, EdgeType extends Edge> = ReactFlowProps<
  NodeType,
  EdgeType
> & {
  children?: ReactNode
  background?: boolean
}

const deleteKeyCode = ['Backspace', 'Delete']

export function CanvasBackground() {
  return (
    <Background
      bgColor='var(--card)'
      color='color-mix(in oklch, var(--muted-foreground) 55%, var(--card))'
    />
  )
}

export const Canvas = <
  NodeType extends Node = Node,
  EdgeType extends Edge = Edge,
>({
  children,
  background = true,
  style,
  ...props
}: CanvasProps<NodeType, EdgeType>) => (
  <ReactFlow<NodeType, EdgeType>
    deleteKeyCode={deleteKeyCode}
    fitView
    panOnDrag={false}
    panOnScroll
    selectionOnDrag={true}
    zoomOnDoubleClick={false}
    style={background ? style : { ...style, backgroundColor: 'transparent' }}
    {...props}
  >
    {background ? <CanvasBackground /> : null}
    {children}
  </ReactFlow>
)
