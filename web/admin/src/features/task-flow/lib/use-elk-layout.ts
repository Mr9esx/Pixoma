import { useCallback, useEffect, useRef, useState, type RefObject } from 'react'
import { useNodesInitialized, useReactFlow } from '@xyflow/react'
import { computeFitViewport, elkLayout } from './elk-layout'

/**
 * ELK 自动布局编排：
 * - 节点首次测量完成后布局一次并 fitView；
 * - 之后结构/尺寸变化由调用方 bump()，防抖后重排（打字过程中不跳动）；
 * - 拖动期间不重排，松手后由调用方通过规则重排触发。
 */
export function useElkLayout(containerRef: RefObject<HTMLDivElement | null>) {
  const { getNodes, getEdges, setNodes, setViewport } = useReactFlow()
  const nodesInitialized = useNodesInitialized()

  const [tick, setTick] = useState(0)
  const draggingRef = useRef(false)
  const timerRef = useRef<ReturnType<typeof setTimeout> | undefined>(undefined)
  const requestSeqRef = useRef(0)
  const fitNextRef = useRef(false)
  const initialDoneRef = useRef(false)

  const runLayout = useCallback(async () => {
    if (draggingRef.current) return
    const seq = ++requestSeqRef.current
    const nextNodes = await elkLayout(getNodes(), getEdges())
    if (seq !== requestSeqRef.current) return // 已被更新的布局请求取代
    setNodes(nextNodes)
    if (fitNextRef.current) {
      fitNextRef.current = false
      const el = containerRef.current
      if (el) {
        const rect = el.getBoundingClientRect()
        const vp = computeFitViewport(nextNodes, rect.width, rect.height)
        void setViewport(vp, { duration: 200 })
      }
    }
  }, [getNodes, getEdges, setNodes, setViewport, containerRef])

  const requestLayout = useCallback(
    (delay = 250) => {
      window.clearTimeout(timerRef.current)
      timerRef.current = setTimeout(() => {
        timerRef.current = undefined
        void runLayout()
      }, delay)
    },
    [runLayout],
  )

  const bump = useCallback(() => setTick((t) => t + 1), [])

  /** 下一次布局完成后 fitView（初始加载、新增分支时使用）。 */
  const fitAfterNextLayout = useCallback(() => {
    fitNextRef.current = true
  }, [])

  // 首次测量完成后立即布局并适配视图。
  useEffect(() => {
    if (nodesInitialized && !initialDoneRef.current) {
      initialDoneRef.current = true
      fitAfterNextLayout()
      requestLayout(0)
    }
  }, [nodesInitialized, fitAfterNextLayout, requestLayout])

  // 图结构/内容变化后防抖重排。
  useEffect(() => {
    if (tick > 0 && nodesInitialized) requestLayout(300)
  }, [tick, nodesInitialized, requestLayout])

  useEffect(() => () => window.clearTimeout(timerRef.current), [])

  return { bump, fitAfterNextLayout, draggingRef }
}
