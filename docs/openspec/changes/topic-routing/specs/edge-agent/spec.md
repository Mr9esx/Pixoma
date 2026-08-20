## ADDED Requirements

### Requirement: Edge 订阅 Topic 配置
Edge-Agent MUST 支持配置订阅 Topic 列表（如 `subscribe_topics`）；未配置时 MUST 默认订阅 `default` Topic。进程启动后 MUST 按实际订阅集合向控制面申报并长轮询领取。Edge MUST NOT 解析投放表达式（规则求值只发生在控制面）。

#### Scenario: 未配置订阅默认
- **WHEN** Edge 配置未声明 subscribe_topics
- **THEN** Edge 以 `default` 为订阅集合领取任务

#### Scenario: 多 Topic 订阅
- **WHEN** Edge 声明订阅 `["default","fast-gpu"]`
- **THEN** Edge 可领取这两个 Topic 下投递的任务，不领取其它 Topic 的任务
