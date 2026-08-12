## 1. 引导启动与一体控制面

- [ ] 1.1 新增/调整 `pixoma` 入口：零配置启动、引导态存储、日志输出后台 URL 与默认管理员凭证
- [ ] 1.2 未初始化门闩：仅放行登录与向导 API；初始化后启用管理员会话鉴权
- [ ] 1.3 一体托管管理 HTTP（同进程或明确子服务）；健康检查可用
- [ ] 1.4 首次登录强制改密；改密后启动日志不再打印明文密码

## 2. Agent 拉取派发（替换默认 Redis 跨进程队列）

- [ ] 2.1 Task/调度模型：可领取态、lease、claimed_by、心跳字段与迁移
- [ ] 2.2 控制面 Agent API：长轮询 claim、续约/心跳、status 上报（幂等接入 applyStatus）
- [ ] 2.3 Orchestrator：prep `job_ref` 后改为「可领取」而非默认 Publish Redis dispatch
- [ ] 2.4 租约过期回收与无在线 Edge 时保持 pending 的行为与测试
- [ ] 2.5 默认路径去掉对 Redis 的运行时依赖（旧适配器可残留但非默认）

## 3. Edge 二进制与本机/远程

- [ ] 3.1 `pixoma-edge-agent`：拉取循环 + Comfy mock/真机 + blob 读写 + status/heartbeat
- [ ] 3.2 本机 localfs 共用目录主路径（控制面 + Edge）跑通
- [ ] 3.3 远程 s3/tos 配置校验；拒绝远程 localfs
- [ ] 3.4 本机自动拉起 Edge（可配置关闭）；远程向导文案与节点登记

## 4. 向导与配置落库

- [ ] 4.1 settings 持久化（业务库）与引导态字段；启动时装配读取顺序（引导态 → settings → env 紧急覆盖）
- [ ] 4.2 向导 API：库连通、本机/远程、存储、节点/Comfy、TG Token 等步骤与校验
- [ ] 4.3 `web/admin`：未初始化向导流、登录页、完成后进入业务壳
- [ ] 4.4 「保存并重启生效」的产品行为与提示

## 5. 文档与验收

- [ ] 5.1 更新 README 部署故事与架构 `runtime`/`overview`（取消 allinone/split 用户矩阵为默认叙事）
- [ ] 5.2 验收：空目录 `pixoma` → 日志账密 → 向导本机 mock → Edge 领取 → 任务成功
- [ ] 5.3 验收：远程配置矩阵（禁 localfs）与 Edge 出站拉取说明可执行
- [ ] 5.4 Comfy mock 开关在新拓扑下仍可端到端成功
