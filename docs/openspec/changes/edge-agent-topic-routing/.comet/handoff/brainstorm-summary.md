# Brainstorm Summary

- Change: edge-agent-topic-routing
- Date: 2026-08-10

## Confirmed Technical Approach

用户已确认 Design 提案：

- 本期不做 Topic 分流
- allinone：单进程；Memory；localfs；方案 A
- split：Edge + Redis Streams + S3；方案 A
- dispatch 按实例 Topic；执行面只认 `job_ref`

Design Doc: `docs/superpowers/specs/2026-08-10-edge-agent-dual-mode-design.md`

## Key Trade-offs and Risks

- Open change 名含 topic-routing，本期延后 Topic；以 Design Doc / Spec Patch 为准
- ALLINONE 也写 job：路径统一
- 模式与驱动误配须启动失败

## Testing Strategy

- prep/job 单测；allinone+mock E2E；Redis/S3 适配；split 冒烟；误配负向

## Spec Patches

- 已写回：topic-* 延后；edge/blob/queue/job/orchestrator 对齐双模式 + 方案 A；tasks 重排
