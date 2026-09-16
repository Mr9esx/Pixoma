---
comet_change: wecom-aibot-channel
role: technical-design
canonical_spec: openspec
scope: narrowed
---

# 跨平台渠道运行时优化设计

## 范围

本设计记录 `wecom-aibot-channel` 的最终收敛范围。企业微信与个人微信均不接入；保留已完成的共享会话核心、Telegram 迁移、飞书工厂隔离和适配器安全重建。

## 架构

`internal/channels/conversation` 承担平台无关的会话状态、能力调用、菜单/卡片动作、导航、结果渲染决策和异步通知。Telegram 与飞书适配器只负责编解码平台事件、转存媒体和投递消息。

渠道工厂按平台显式分发。飞书不会创建 Telegram Bot；通知处理器仅在飞书适配器成功启动后注册，并在成功停止后注销。

Assembler 替换适配器时先等待旧实例停止。停止失败会写入运行时错误状态，且本轮不创建替代实例。

## 已撤回内容

企业微信智能机器人所需的凭证、HTTP 字段、健康探测、协议客户端、媒体桥和 Go 模块依赖全部移除。`wecom` 不再是有效的平台类型。

## 验证

渠道相关测试覆盖共享控制器、Telegram、飞书工厂、适配器停止失败和企业微信平台拒绝。归档前运行渠道 Go 包测试与全局 Go 测试。
