## ADDED Requirements

### Requirement: 启动初始化默认 Topic
控制面启动 MUST 在业务库就绪后幂等确保默认 Topic（`default`）存在；已存在则跳过。该初始化 MUST NOT 阻塞未完成初始化向导的路径。

#### Scenario: 空库启动创建默认 Topic
- **WHEN** 业务库中无任何 Topic 记录且控制面启动
- **THEN** 创建 `default` Topic；再次启动不重复创建
