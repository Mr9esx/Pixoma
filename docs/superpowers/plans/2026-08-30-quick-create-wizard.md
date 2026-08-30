---
change: quick-create-wizard
design-doc: docs/superpowers/specs/2026-08-30-quick-create-wizard-design.md
base-ref: 82bf8222d7bcb13c5fc70e66b7a31237efbd8117
---

# 快速新建向导 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 把快速配置收成「快速新建」四屏：工作流（无处理流程表）→ 显式队列 → 节点 → 启用后还差一步。

**Architecture:** 继续 Formity。提交链 createTopic? → createCase（一条 always 规则）→ 按需 patchEdge → enableCase。工作流页不改四段结构，仅增加 hideProcessing。

**Tech Stack:** React, Formity, TanStack Query, vitest 契约测试。

## Global Constraints

- 文案：字段/按钮/校验；还差一步短教程落到按钮。
- 不改调度引擎。路由仍 `/quick-config`。
- TDD：先失败测试再实现。本 change 不自动 git commit（除非用户要求）。

## File map

- `lib/session.ts` v3
- `lib/queue-binding.ts` 新
- `lib/commit.ts` 新
- `step2-queue.tsx` 新
- `step4-next.tsx` 新
- 改：flow、page、chrome、step1、step2-node、types、menu、i18n、workflow-editor
- 删：step3-channels、done-screen 引用

---

## Task 1: 会话 v3

- [ ] 改 session 测试：v2 清空；v3 含 topicKey/selectedEdgeId
- [ ] 实现 SESSION_SCHEMA_VERSION=3
- [ ] 跑 session.test.ts

## Task 2: 队列绑定纯函数

- [ ] 写 queue-binding.test.ts
- [ ] 实现 queueHasSubscribers / nodeStepCanAdvance

## Task 3: hideProcessing

- [ ] 契约或组件测试：hideProcessing 时无 processing section
- [ ] WorkflowEditor 实现 + stepRail 量 outputs

## Task 4: 向导四屏与提交

- [ ] 改 contract 测试为新四屏与 commit 链
- [ ] Step2Queue、commit、Step4Next、flow、page、i18n、chrome
- [ ] 跑 vitest quick-config + workflow-editor 相关

## Task 5: 收尾

- [ ] 勾选 openspec tasks.md
- [ ] 删除无引用的投放/完成页文件（若无他用）
