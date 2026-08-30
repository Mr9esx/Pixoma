# 显式路由兜底设计

## 目标

路由必须显式指定目标 Topic。无规则或全部规则未命中时，任务不回退 `default`。新建工作流默认有一条「无条件 → 默认」规则，用户可以编辑、删除、替换或新增规则。

## 现状

- `routing.Resolve` 在 `routing == nil`、规则为空或全部未命中时返回 `default`。
- 前端空规则可保存，表格文案宣称未命中会回退默认 Topic。
- 新建 `CaseRecord` 初始没有 `routing`。

## 协议

`RoutingRule.when` 新增无条件分支：

```json
{
  "rules": [
    {
      "when": { "always": true },
      "topic": "default"
    }
  ]
}
```

- `always` 只能独立使用；不能与 `field` / `op` / `value` / `and` / `or` 混用。
- `{"always": true}` 恒命中。
- `{"always": false}` 无效。

## 后端

1. 条件引擎支持 `always`：解析合法，校验合法，评估恒真。
2. `routing.Resolve`：
   - 无路由或无规则：返回 `ErrNoMatch`。
   - 全部规则未命中：返回 `ErrNoMatch`。
   - 条件评估或提供方瞬时错误：保留现有 pending 重试语义。
3. 调度器收到 `ErrNoMatch` 时，把 pending 任务标记为 `failed`，错误码 `routing_no_match`，错误文案「未命中路由规则」。该错误不重试。
4. `ValidateRouting` 要求 `routing != nil` 且至少一条规则。每条规则的 Topic 仍必须存在、启用，条件仍必须通过属性目录校验。
5. 不做存量 Case 自动迁移：`routing` 缺失或规则为空的 Case 会被 API 校验拒绝；运行时全部规则未命中时任务失败。

## 前端

1. 新建 `CaseRecord` 初始 `routing` 为「无条件 → default」。
2. `TaskFlowTable`：
   - 无条件规则显示「无条件」。
   - 条件编辑器提供「无条件」开关；关闭后显示常规条件编辑器，开启后写入 `{ "always": true }`。
   - 底部文案改为「按顺序匹配；未命中任务失败」。
   - 规则为空时显示校验失败，阻止保存。
3. 常规规则可继续配置 `user` / `case` / `input` 条件和任意启用 Topic。
4. 就绪检查只把规则中显式命中的 Topic 作为使用中 Topic；不再额外追加 `default`。空规则是 gap。
5. 快速配置完成页同样按显式规则订阅 Topic，不再为空规则补订 `default`。

## 文案

- 中英文同步更新。
- 不使用「请」「您」「温馨提示」「进行」「完成」等禁用词。
- 面向操作者说明结果：未命中不兜底，任务失败。

## 验收

- 新建工作流表单内有一条可编辑、可删除的「无条件 → 默认」规则。
- 删除全部规则后，前端校验失败，API 校验也失败。
- 有条件规则但任务上下文未命中时，任务不进入 `default`，任务变为失败并记录 `routing_no_match`。
- 存量无规则 Case 不被自动修改；API 校验拒绝其继续使用。
- Go 与前端相关测试通过。
