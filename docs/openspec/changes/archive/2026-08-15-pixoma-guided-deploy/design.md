## Context

当前控制面依赖 YAML/`runtime_mode`/`queue`/`blob` 矩阵，跨进程默认走 Redis Streams，同机「allinone」则把执行面塞进 Bot 进程。用户已确认：**永远控制面 + Edge**；**默认无 Redis、无用户可见 queue**；派活改为 **DB + Edge 长轮询领取**；配置经后台向导落库；提供零配置 `pixoma` 与 `pixoma-edge-agent`。

本 change **不拆分**（用户选项 2）：一次性交付引导启动、拉取运行时、向导与落库，避免半成品拓扑长期并存。

## Goals / Non-Goals

**Goals:**

- `pixoma` 零配置可起：日志打印后台 URL + 默认管理员账号密码；未初始化仅放行登录与向导
- 引导态（bootstrap）解决「配置进库 vs 库未就绪」；业务 settings 进业务库
- 向导：库 → 本机/远程 → 存储 → 节点/Comfy 指引 → 消息平台（含 TG Token）；远程禁 localfs
- Edge 经 HTTP 长轮询 claim/lease/heartbeat/status；任务可调度态在 DB
- 同机 localfs 共用目录可跑通主路径；远程 OSS（s3|tos）
- 本机可自动拉起 Edge；文档亦支持手起
- 架构/README 改为新部署故事

**Non-Goals:**

- 保留用户必选的 allinone/split、queue.driver 作为一等配置
- 默认路径继续依赖 Redis / 跨进程 memory Bus
- NATS、预签名 CDN、running 任务 Comfy interrupt
- 一次删光所有旧 YAML 兼容（过渡期可 env/YAML 紧急覆盖）

## Decisions

1. **拓扑：始终双进程语义**  
   控制面不内嵌生产执行面；执行只在 Edge。同机也起 Edge（可自动拉起）。  
   - 备选：保留 allinone 同进程执行 → 否决（与「简化心智」冲突）。

2. **跨进程总线：DB + Agent HTTP，不用 Redis**  
   调度把 Task 置为可领取；Edge `GET` 长轮询 claim（带 lease）；`POST` status/heartbeat。  
   - 备选：WebSocket 推送 → 二期；短轮询 → 可作实现细节但契约按长轮询写。  
   - 备选：继续 Redis → 否决为默认。

3. **控制面内部事件**  
   `task.created` 等同进程信号可用函数调用或内存通道，**不**暴露为用户 queue 配置。

4. **两段存储**  
   - Bootstrap：本机小库/文件（initialized、管理员哈希、业务 DSN、向导步）。  
   - App DB：向导所选库；settings + 业务表。

5. **本机 vs 远程（取代 mode 矩阵）**  
   - 本机：localfs + Agent 拉活。  
   - 远程：s3|tos + Agent 出站连控制面；UI 禁止 localfs。

6. **二进制**  
   - `pixoma`：Bot 调度 + 管理 API +（可内嵌）静态后台。  
   - `pixoma-edge-agent`：现 edge-agent 协议改为拉取；最小配置：控制面 URL、instance_id、凭证、blob。

7. **安全**  
   默认监听本机；默认密码仅未初始化时打日志；首次登录强制改密；密钥进库须加密或至少与引导密钥派生。本期管理 API 从「完全无鉴权」升级为「初始化后需管理员会话」（向导/登录）。

8. **生效方式**  
   一期向导「保存并重启生效」；不承诺热切换 Redis/OSS 中途无感。

## Risks / Trade-offs

- [单 change 过大] → 任务按里程碑勾选；Verify 分主路径（本机 mock）与远程（可手工/标签）  
- [claim 双领] → lease + 实例维度唯一领取；单测覆盖  
- [Edge 失联] → lease 过期回可领取；心跳更新在线  
- [密钥进库] → 加密与备份引导态风险写入文档与威胁说明  
- [旧 split+Redis 部署] → 过渡兼容或迁移说明；默认文档只讲新故事  
- [admin 无鉴权 → 有登录] → **BREAKING** 对现有联调脚本；提供默认账密与改密流

## Migration Plan

1. 新部署只走 `pixoma` + 向导 + Edge 拉取。  
2. 旧 YAML/`RUNTIME_MODE`：过渡可读，新 UI 不展示 queue/mode。  
3. 回滚：切回旧二进制/配置（文档标明版本边界）。  
4. 数据：Task 增加 lease/claimed_by 等字段需迁移。

## Open Questions

- Agent 鉴权：共享 token vs mTLS（一期共享 token 即可）  
- 长轮询默认 wait（建议 25s）与 lease（建议 60–120s）具体值 Build 时定  
- 静态后台内嵌进 `pixoma` 还是仍独立 `web/admin` 开发、生产由一体托管（倾向一体托管生产构建产物）
