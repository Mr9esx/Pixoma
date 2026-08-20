## ADDED Requirements

### Requirement: 节点订阅 Topic 绑定
节点记录 MUST 支持声明订阅 Topic 列表；未声明时 MUST 按默认 Topic（`default`）处理。节点管理 API MUST 支持读取与更新该绑定；更新后 MUST 影响后续按 Topic 领取（进行中任务按既有租约/对账路径处理）。

#### Scenario: 节点绑定多个 Topic
- **WHEN** 节点 `gpu-1` 更新订阅为 `["default","fast-gpu"]`
- **THEN** 该节点后续可领取这两个 Topic 的任务

#### Scenario: 未绑定按默认
- **WHEN** 查询未设置订阅的节点
- **THEN** 其有效订阅集合等于 `["default"]`
