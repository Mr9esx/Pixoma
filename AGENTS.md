# Pixoma 项目约定

## Comet build 默认配置（用户偏好，永久生效）

运行 Comet build 阶段时，默认采用以下配置，除非用户在当次任务中明确要求其他选择：

- 工作区隔离：`current`（当前分支直接工作）
- 执行方式：`executing-plans`（主会话按计划顺序执行）
- TDD 模式：`tdd`
- 代码审查模式：`standard`
