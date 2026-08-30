# Admin 状态显示规范

适用：`web/admin` 的资源列表、详情页头部，以及后续同类页面。

目标：**状态能力一致**。用户在列表、详情和健康告警之间切换时，不能看到互相矛盾的健康状态。

## 原则（必须遵守）

1. **状态不只看启停**
   资源的展示状态 MUST 综合启停、配置依赖和运行依赖。仅看 `enabled` 会把“已启用但当前不可用”的资源错报成正常。

2. **只有绿色和黄色两档**
   没有任何问题时用绿色；存在一个或多个问题时统一用黄色。禁止为“多个问题”再引入红色档位。

3. **列表与详情共用同一套健康结果**
   任务队列、工作流等资源的列表圆点 MUST 使用与详情 `LinkHealth` / 健康区块相同的断点结果，避免列表绿色但详情报“当前不可用”。

4. **健康数据未就绪时不得降级为绿色**
   健康检查仍在加载、失败或缺少数据时，MUST 视为存在待确认问题；禁止因为暂时没有数据而回退成绿色。

5. **统一使用 `StatusDot`**
   新页面 MUST 复用 `@/components/status-dot`，不要复制圆点样式或重新发明状态徽标。圆点 MUST 提供 `aria-label` 和 `title`。

6. **状态文案与动作文案分开**
   状态值统一使用「已启用 / 已停用」「正常 / 存在问题」「当前不可用」。启停动作按钮仍可使用「启用 / 停用」。

7. **只用语义令牌**
   状态色 MUST 使用 `bg-success` / `bg-warning` 等语义令牌；禁止裸 hex、手写 `dark:` 或一次性 emerald / amber 类名。

## 参考实现

- 组件：`web/admin/src/components/status-dot.tsx`
- 计算节点列表：`web/admin/src/features/edges/list-health.ts`
- 任务队列健康：`web/admin/src/features/link-health/lib/references.ts` 中的 `topicReferences`
- 工作流健康：`web/admin/src/features/link-health/lib/references.ts` 中的 `caseReferences`
