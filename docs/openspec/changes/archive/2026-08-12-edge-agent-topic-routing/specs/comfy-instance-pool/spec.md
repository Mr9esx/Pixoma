## ADDED Requirements

### Requirement: 实例或 Edge 与 Topic 关联
系统 MUST 能表达执行端（Comfy 实例元数据与/或 Edge 注册信息）与其订阅/服务的逻辑 Topic 之间的关联，供调度判断 Topic 是否可投递，并供运维查看。云侧 MUST NOT 将「对家里 Comfy 的 HTTP 探活成功」作为家里部署场景下唯一的健康信号。

#### Scenario: 记录 Edge 订阅的 Topic
- **WHEN** 某 Edge 配置订阅 Topic `vip` 并成功上报在线
- **THEN** 控制面能查询到 `vip` 存在可用消费者

#### Scenario: 家里场景不以云直连 Comfy 为唯一健康条件
- **WHEN** 实例位于不可被云直连的网络，但对应 Topic 的 Edge 在线
- **THEN** 该 Topic 仍可作为可投递目标（在其他条件满足时）
