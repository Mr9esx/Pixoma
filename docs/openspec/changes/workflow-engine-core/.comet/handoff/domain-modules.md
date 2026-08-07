# 核心领域模块（组合优先，部署后置）

> 目标：模块边界清晰，可同进程组装为 all-in-one，也可按模块拆成独立进程——**设计按模块，不按二进制个数。**

## 模块地图

```text
┌─ adapter/tg ─────────────────────────────────────┐
│  TG 入出：Update→用例；DTO→消息/按钮；执行 notify │
└───────────────────────┬──────────────────────────┘
                        │
┌─ app（应用用例）──────▼──────────────────────────┐
│  Menu / CaseQuery / Session / ConfirmRun / …     │
└───────┬─────────────────────────────┬────────────┘
        │                             │
┌───────▼────────┐           ┌────────▼────────────┐
│ domain/protocol│           │ domain/session      │
│ domain/case    │           │ domain/task         │
│ (注册·校验)    │           │ (锁·任务状态机)     │
└────────────────┘           └────────┬────────────┘
                                      │
        ┌─────────────────────────────┼─────────────────────┐
        ▼                             ▼                     ▼
┌─ orchestrator ──────┐    ┌─ actuator ──────────┐   ┌─ ports ─────┐
│ 选实例·投递·收敛    │    │ 执行 Comfy·报 status│   │ queue       │
│ 取消策略·驱动 notify│    │ 写读 Blob           │   │ blob        │
└─────────────────────┘    └──────────┬──────────┘   │ notify      │
                                      │               │ instance    │
                                      ▼               └─────────────┘
                               ┌─ comfyui client ─┐
                               └──────────────────┘
```

## 模块职责（领域语言）

| 模块 | 领域职责 | 不负责 |
|------|----------|--------|
| **protocol** | Case 契约、JSON Schema 校验、绑定模型 | 存储、TG、调度 |
| **case（registry）** | Case 持久化/查询/上下架 | 会话、执行 |
| **session** | 填表锁、草稿、Start/Exit/Confirm 前状态 | 执行、通知渠道细节 |
| **task** | Task 聚合与状态迁移规则 | 选实例、调 Comfy |
| **app** | 用例编排（对外应用 API） | 基础设施实现 |
| **adapter/tg** | 渠道适配 | 领域规则 |
| **orchestrator** | 任务控制面：路由投递、status 收敛、对账、温和取消、**发出 notify 意图** | 发 TG、跑 workflow 图 |
| **actuator** | 执行面：对本实例 ComfyUI 执行并上报 | TG、全局选实例 |
| **comfyui** | HTTP 客户端与注入/拉历史 | 业务状态机 |
| **queue / blob / notify / instance** | 端口（可替换适配器） | 业务决策 |

## 组合方式（同一套模块）

```text
all-in-one 进程 = tg + app + domain* + orchestrator + actuator + memory queue + local blob

拆开时（示例，非现在必做）：
  进程 A = tg + app + domain（写）+ queue.publish
  进程 B = orchestrator + domain（读写真相）+ queue
  进程 C = actuator + comfyui + blob + queue（订自己的 dispatch）
```

拆的是 **进程组装**；模块代码与依赖方向不变。

## 依赖规则（保证可拆）

1. `domain/*` 不依赖 adapter、orchestrator、actuator、具体 MQ/OSS SDK  
2. `orchestrator` / `actuator` 只依赖 `domain/task` + **ports**  
3. `adapter/tg` → `app` → `domain`  
4. 通知：`orchestrator` → `notify` 端口；**只有 tg adapter（或将来别的渠道 adapter）实现「发到用户」**  
5. 换 Memory→NATS、LocalFS→S3 = 只加 port 适配器，不改模块边界  

## 和「几个二进制」的关系

- **现在不讨论** 部署几个二进制  
- **现在要钉死** 的是上表模块 + 端口 + 依赖方向  
- all-in-one / 拆分 = main 里 `wire` 哪些模块启动，属于组装层（`cmd/`）
