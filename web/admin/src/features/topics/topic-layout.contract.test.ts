import { readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const here = dirname(fileURLToPath(import.meta.url))
const read = (p: string) => readFileSync(join(here, p), 'utf8')

const LIST_PANEL = read('topic-list-panel.tsx')
const DETAIL_PANEL = read('topic-detail-panel.tsx')
const ROUTE = read('../../routes/_app/topics/route.tsx')
const NEW = read('create-topic-form.tsx')
const STATS = read('topic-stats-panel.tsx')

describe('topics admin page contract', () => {
  it('列表：搜索 + key/name + 启用状态 + 默认标记', () => {
    expect(LIST_PANEL).toContain("data-testid='topics-list-panel'")
    expect(LIST_PANEL).toContain('topics.listSearch')
    expect(LIST_PANEL).toContain('topic.key')
    expect(LIST_PANEL).toContain('topics.enabled')
    expect(LIST_PANEL).toContain('<StatusDot')
    expect(LIST_PANEL).toContain("topic.key === 'default'")
    expect(LIST_PANEL).toContain("to='/topics/$key'")
  })

  it('详情：编辑名称 / 启停 / 删除，默认 Topic 禁止停用删除', () => {
    expect(DETAIL_PANEL).toContain("data-testid='topic-detail-panel'")
    expect(DETAIL_PANEL).toContain('updateTopic')
    expect(DETAIL_PANEL).toContain('deleteTopic')
    expect(DETAIL_PANEL).toContain('isDefault')
    expect(DETAIL_PANEL).toContain(
      'disabled={isDefault || enableMutation.isPending}'
    )
    expect(DETAIL_PANEL).toContain(
      'disabled={isDefault || deleteMutation.isPending}'
    )
    expect(DETAIL_PANEL).toContain('topics.defaultHint')
    expect(DETAIL_PANEL).toContain('listEdges')
    expect(DETAIL_PANEL).toContain('topics.deleteWillRemoveRules')
    expect(DETAIL_PANEL).toContain('topics.deleteWillUnbindNodes')
    expect(DETAIL_PANEL).toContain('topics.deleteWillFailQueued')
    expect(DETAIL_PANEL).toContain('topicDeleteErrorMessage')
    expect(DETAIL_PANEL).toContain('deleteTopic(topicKey, ackImpact)')
    expect(DETAIL_PANEL).toContain('TopicStatsPanel')
    expect(DETAIL_PANEL).toContain('topics.statsTitle')
  })

  it('统计面板：吞吐曲线 + 状态分布 + 错误码 + 耗时', () => {
    expect(STATS).toContain('getTopicStats')
    expect(STATS).toContain('TimeRangeControl')
    expect(STATS).toContain('dashboard.range7d')
    expect(STATS).toContain('AreaChart')
    expect(STATS).toContain('throughput')
    expect(STATS).toMatch(/formatThroughputLabel\(payload\)/)
    expect(STATS).toContain('stats.status')
    expect(STATS).toContain('error_codes')
    expect(STATS).toContain('runtime_ms')
    expect(STATS).toContain("data-testid='topic-stats-panel'")
  })

  it('路由：280px master-detail + 自动选中第一个 topic + 新建为 modal', () => {
    expect(ROUTE).toContain('MasterDetailShell')
    expect(ROUTE).toContain('md:grid-cols-[280px_1fr]')
    expect(ROUTE).toContain('TopicDetailPanel')
    expect(ROUTE).toContain('items[0]?.key')
    expect(ROUTE).toContain('replace: true')
    expect(ROUTE).toContain("key === 'new'")
    expect(ROUTE).toContain('CreateTopicForm')
    expect(ROUTE).toContain('<Dialog')
    expect(ROUTE).toMatch(
      /hasSelection=\{Boolean\(selectedKey\) \|\| key === 'new'\}/
    )
  })

  it('新建：key 模式校验 + name 必填', () => {
    expect(NEW).toContain('KEY_PATTERN')
    expect(NEW).toContain('createTopic')
    expect(NEW).toContain('keyValid')
    expect(NEW).toContain('canCreate')
  })

  it('新建表单：按钮在 DialogFooter', () => {
    expect(NEW).toMatch(/DialogFooter/)
    expect(NEW).toMatch(/flex flex-1 flex-col gap-4/)
    expect(NEW).not.toMatch(/max-w-xl/)
  })
})
